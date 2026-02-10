package worker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"server/internal/buildworker/artifact"
	"server/internal/buildworker/cfg"
	"server/internal/buildworker/queue"
	"server/internal/server/model"
	"server/internal/server/utils"
	"server/pkg/buildqueue"
	"server/pkg/dbs"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

type Worker struct {
	queue     *queue.BuildQueue
	uploader  *artifact.Uploader
	ctx       context.Context
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	semaphore chan struct{} // 并发控制
}

func NewWorker(q *queue.BuildQueue, uploader *artifact.Uploader) *Worker {
	ctx, cancel := context.WithCancel(context.Background())
	return &Worker{
		queue:     q,
		uploader:  uploader,
		ctx:       ctx,
		cancel:    cancel,
		semaphore: make(chan struct{}, cfg.C.Worker.MaxConcurrent),
	}
}

func (w *Worker) Start() {
	logx.Info("Build worker started")

	pollInterval := time.Duration(cfg.C.Worker.PollInterval) * time.Second

	for {
		select {
		case <-w.ctx.Done():
			logx.Info("Worker stopping, waiting for running tasks...")
			w.wg.Wait()
			logx.Info("Build worker stopped")
			return
		default:
			// 尝试获取任务
			task, err := w.queue.Dequeue(pollInterval)
			if err != nil {
				logx.Errorf("Dequeue error: %v", err)
				time.Sleep(time.Second)
				continue
			}

			if task == nil {
				continue // 无任务
			}

			// 检查是否已取消
			if w.queue.IsCancelled(task.ID) {
				logx.Infof("Task %d already cancelled, skipping", task.ID)
				w.queue.RemoveCancelMark(task.ID)
				continue
			}

			// 获取并发槽位
			w.semaphore <- struct{}{}
			w.wg.Add(1)

			go func(t *buildqueue.BuildTaskMessage) {
				defer func() {
					<-w.semaphore
					w.wg.Done()
				}()
				w.processTask(t)
			}(task)
		}
	}
}

func (w *Worker) Stop() {
	w.cancel()
}

func (w *Worker) processTask(task *buildqueue.BuildTaskMessage) {
	taskID := task.ID
	logx.Infof("Processing build task: %d", taskID)

	// 更新状态为构建中
	w.updateTaskStatus(taskID, model.BuildStatusBuilding, 0, "准备构建环境")

	// 1. 获取商户配置
	var merchant model.BuildMerchant
	has, err := dbs.DBAdmin.ID(task.BuildMerchantID).Get(&merchant)
	if err != nil || !has {
		w.failTask(taskID, "获取商户配置失败", task)
		return
	}

	// 2. 选择构建服务器
	server, err := w.selectBuildServer(task)
	if err != nil {
		w.failTask(taskID, fmt.Sprintf("选择构建服务器失败: %v", err), task)
		return
	}
	defer w.releaseBuildServer(server.ID)

	// 3. 连接构建服务器
	w.updateTaskStatus(taskID, model.BuildStatusBuilding, 5, "连接构建服务器")

	sshClient, err := w.connectBuildServer(server)
	if err != nil {
		w.failTask(taskID, fmt.Sprintf("连接构建服务器失败: %v", err), task)
		return
	}
	defer sshClient.Close()

	// 4. 准备构建配置
	w.updateTaskStatus(taskID, model.BuildStatusBuilding, 10, "生成构建配置")

	configJSON := w.generateMerchantConfig(&merchant, task)
	configBase64 := base64.StdEncoding.EncodeToString([]byte(configJSON))

	// 5. 执行构建
	w.updateTaskStatus(taskID, model.BuildStatusBuilding, 15, "开始构建")

	executor := NewExecutor(sshClient, w.queue, taskID)
	buildCmd := fmt.Sprintf("%s/worker.sh %d %s %s",
		cfg.C.Worker.ScriptsDir, taskID, configBase64, task.Platforms)

	output, err := executor.ExecuteWithProgress(buildCmd,
		time.Duration(cfg.C.Worker.BuildTimeout)*time.Second)

	if err != nil {
		// 检查是否是取消
		if w.queue.IsCancelled(taskID) {
			w.updateTaskStatus(taskID, model.BuildStatusCancelled, 0, "已取消")
			w.queue.RemoveCancelMark(taskID)
			return
		}
		w.failTask(taskID, fmt.Sprintf("构建失败: %v", err), task)
		w.saveLog(taskID, output)
		return
	}

	// 6. 上传产物
	w.updateTaskStatus(taskID, model.BuildStatusBuilding, 90, "上传构建产物")

	artifacts, err := w.uploadArtifacts(sshClient, taskID, task.Platforms, &merchant)
	if err != nil {
		logx.Errorf("Upload artifacts warning: %v", err)
		// 不阻止任务完成，产物上传失败只记录警告
	}

	// 7. 保存产物记录
	for i := range artifacts {
		artifacts[i].TaskID = taskID
		artifacts[i].BuildMerchantID = task.BuildMerchantID
		artifacts[i].MerchantName = merchant.Name
		dbs.DBAdmin.Insert(&artifacts[i])
	}

	// 8. 保存日志
	w.saveLog(taskID, output)

	// 9. 完成
	w.completeTask(taskID)
	logx.Infof("Build task %d completed successfully", taskID)
}

func (w *Worker) selectBuildServer(task *buildqueue.BuildTaskMessage) (*model.BuildServer, error) {
	var server model.BuildServer

	// 如果指定了服务器ID
	if task.ServerID > 0 {
		has, err := dbs.DBAdmin.ID(task.ServerID).Get(&server)
		if err != nil || !has {
			return nil, fmt.Errorf("指定的构建服务器不存在")
		}
		// 增加当前任务计数
		dbs.DBAdmin.ID(server.ID).Incr("current_tasks").Update(&model.BuildServer{})
		return &server, nil
	}

	// 根据平台自动选择
	platforms := strings.Split(task.Platforms, ",")
	needMac := false
	for _, p := range platforms {
		p = strings.TrimSpace(p)
		if p == "ios" || p == "macos" {
			needMac = true
			break
		}
	}

	// 查找可用的构建服务器
	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	session.Where("status = 1") // 状态正常
	session.Where("current_tasks < max_concurrent")

	if needMac {
		session.Where("platforms LIKE '%ios%' OR platforms LIKE '%macos%'")
	}

	has, err := session.OrderBy("current_tasks ASC").Get(&server)
	if err != nil || !has {
		return nil, fmt.Errorf("没有可用的构建服务器")
	}

	// 增加当前任务计数
	dbs.DBAdmin.ID(server.ID).Incr("current_tasks").Update(&model.BuildServer{})

	return &server, nil
}

func (w *Worker) releaseBuildServer(serverID int) {
	dbs.DBAdmin.ID(serverID).Decr("current_tasks").Update(&model.BuildServer{})
}

func (w *Worker) connectBuildServer(server *model.BuildServer) (*utils.SSHClient, error) {
	client := &utils.SSHClient{
		Host:       server.Host,
		Port:       server.Port,
		Username:   server.Username,
		Password:   server.Password,
		PrivateKey: server.PrivateKey,
	}

	if err := client.Connect(); err != nil {
		return nil, err
	}

	return client, nil
}

func (w *Worker) generateMerchantConfig(merchant *model.BuildMerchant, task *buildqueue.BuildTaskMessage) string {
	config := map[string]interface{}{
		"name":                 merchant.Name,
		"app_name":             merchant.AppName,
		"short_name":           merchant.ShortName,
		"android_package":      merchant.AndroidPackage,
		"android_version_code": merchant.AndroidVersionCode,
		"android_version_name": merchant.AndroidVersionName,
		"ios_bundle_id":        merchant.IOSBundleID,
		"ios_version":          merchant.IOSVersion,
		"ios_build":            merchant.IOSBuild,
		"windows_app_name":     merchant.WindowsAppName,
		"windows_version":      merchant.WindowsVersion,
		"macos_bundle_id":      merchant.MacOSBundleID,
		"macos_app_name":       merchant.MacOSAppName,
		"macos_version":        merchant.MacOSVersion,
		"server_api_url":       merchant.ServerAPIURL,
		"server_ws_host":       merchant.ServerWSHost,
		"server_ws_port":       merchant.ServerWSPort,
		"enterprise_code":      merchant.EnterpriseCode,
		"icon_url":             merchant.IconURL,
		// Git 源码配置
		"git_repo_url":  merchant.GitRepoURL,
		"git_branch":    merchant.GitBranch,
		"git_tag":       merchant.GitTag,
		"git_username":  merchant.GitUsername,
		"git_token":     merchant.GitToken,
	}

	// 应用版本覆盖
	if task.OverrideAndroidVersionCode != nil {
		config["android_version_code"] = *task.OverrideAndroidVersionCode
	}
	if task.OverrideAndroidVersionName != nil {
		config["android_version_name"] = *task.OverrideAndroidVersionName
	}
	if task.OverrideIOSVersion != nil {
		config["ios_version"] = *task.OverrideIOSVersion
	}
	if task.OverrideIOSBuild != nil {
		config["ios_build"] = *task.OverrideIOSBuild
	}

	data, _ := json.Marshal(config)
	return string(data)
}

func (w *Worker) updateTaskStatus(taskID int, status int, progress int, step string) {
	now := time.Now()

	// 更新数据库
	task := &model.BuildTask{
		Status:      status,
		Progress:    progress,
		CurrentStep: step,
	}

	cols := []string{"status", "progress", "current_step"}

	if status == model.BuildStatusBuilding && progress == 0 {
		task.StartedAt = &now
		cols = append(cols, "started_at")
	}

	dbs.DBAdmin.ID(taskID).Cols(cols...).Update(task)

	// 更新 Redis 进度
	w.queue.UpdateProgress(&buildqueue.BuildProgress{
		TaskID:      taskID,
		Status:      status,
		Progress:    progress,
		CurrentStep: step,
	})
}

func (w *Worker) failTask(taskID int, errMsg string, task *buildqueue.BuildTaskMessage) {
	now := time.Now()

	// 检查是否需要重试
	if task != nil && task.RetryCount < buildqueue.MaxRetryCount {
		task.RetryCount++
		logx.Infof("Task %d failed, retry %d/%d: %s", taskID, task.RetryCount, buildqueue.MaxRetryCount, errMsg)

		// 更新状态为排队中
		w.updateTaskStatus(taskID, model.BuildStatusQueued, 0, fmt.Sprintf("等待重试 (%d/%d)", task.RetryCount, buildqueue.MaxRetryCount))

		// 延迟后重新入队
		time.AfterFunc(time.Duration(buildqueue.RetryIntervalSec)*time.Second, func() {
			w.queue.Enqueue(task)
		})
		return
	}

	// 更新失败状态
	updateTask := &model.BuildTask{
		Status:     model.BuildStatusFailed,
		ErrorMsg:   errMsg,
		FinishedAt: &now,
	}

	dbs.DBAdmin.ID(taskID).Cols("status", "error_msg", "finished_at").Update(updateTask)

	w.queue.UpdateProgress(&buildqueue.BuildProgress{
		TaskID:      taskID,
		Status:      model.BuildStatusFailed,
		Progress:    0,
		CurrentStep: errMsg,
	})

	logx.Errorf("Build task %d failed: %s", taskID, errMsg)
}

func (w *Worker) completeTask(taskID int) {
	now := time.Now()

	// 获取开始时间计算耗时
	var task model.BuildTask
	dbs.DBAdmin.ID(taskID).Get(&task)

	duration := 0
	if task.StartedAt != nil {
		duration = int(now.Sub(*task.StartedAt).Seconds())
	}

	updateTask := &model.BuildTask{
		Status:      model.BuildStatusSuccess,
		Progress:    100,
		CurrentStep: "构建完成",
		FinishedAt:  &now,
		Duration:    duration,
	}

	dbs.DBAdmin.ID(taskID).Cols("status", "progress", "current_step", "finished_at", "duration").Update(updateTask)

	w.queue.UpdateProgress(&buildqueue.BuildProgress{
		TaskID:      taskID,
		Status:      model.BuildStatusSuccess,
		Progress:    100,
		CurrentStep: "构建完成",
	})
}

func (w *Worker) uploadArtifacts(sshClient *utils.SSHClient, taskID int, platforms string, merchant *model.BuildMerchant) ([]model.BuildArtifact, error) {
	var artifacts []model.BuildArtifact

	outputDir := fmt.Sprintf("%s/task_%d", cfg.C.Worker.OutputDir, taskID)
	platformList := strings.Split(platforms, ",")

	for _, platform := range platformList {
		platform = strings.TrimSpace(platform)

		var pattern, version string
		switch platform {
		case "android":
			pattern = "*.apk"
			version = merchant.AndroidVersionName
		case "ios":
			pattern = "*.ipa"
			version = merchant.IOSVersion
		case "windows":
			pattern = "*.exe"
			version = merchant.WindowsVersion
		case "macos":
			pattern = "*.app"
			version = merchant.MacOSVersion
		default:
			continue
		}

		// 查找产物文件
		findCmd := fmt.Sprintf("find %s/%s -name '%s' -type f 2>/dev/null | head -5", outputDir, platform, pattern)
		output, err := sshClient.ExecuteCommand(findCmd)
		if err != nil || output == "" {
			continue // 该平台没有产物
		}

		files := strings.Split(strings.TrimSpace(output), "\n")
		for _, filePath := range files {
			if filePath == "" {
				continue
			}

			// 获取文件信息
			statCmd := fmt.Sprintf("stat -c '%%s' %s 2>/dev/null || stat -f '%%z' %s 2>/dev/null", filePath, filePath)
			sizeStr, _ := sshClient.ExecuteCommand(statCmd)
			fileSize := int64(0)
			fmt.Sscanf(strings.TrimSpace(sizeStr), "%d", &fileSize)

			// 生成文件名
			fileName := fmt.Sprintf("task_%d/%s/%s", taskID, platform,
				strings.TrimPrefix(filePath, outputDir+"/"+platform+"/"))

			// 上传到云存储
			fileURL, err := w.uploader.UploadFromSSH(sshClient, filePath, fileName)
			if err != nil {
				logx.Errorf("Upload artifact failed: %v", err)
				continue
			}

			artifacts = append(artifacts, model.BuildArtifact{
				Platform:  platform,
				FileName:  fileName,
				FileSize:  fileSize,
				FileURL:   fileURL,
				Version:   version,
				ExpiresAt: time.Now().AddDate(0, 0, 30), // 30天后过期
			})
		}
	}

	return artifacts, nil
}

func (w *Worker) saveLog(taskID int, log string) {
	if log == "" {
		return
	}

	// 将日志上传到云存储
	logFileName := fmt.Sprintf("task_%d/build.log", taskID)
	logURL, err := w.uploader.UploadContent([]byte(log), logFileName, "text/plain")
	if err != nil {
		logx.Errorf("Upload log failed: %v", err)
		return
	}

	// 更新任务的日志URL
	dbs.DBAdmin.ID(taskID).Cols("log_url").Update(&model.BuildTask{LogURL: logURL})
}
