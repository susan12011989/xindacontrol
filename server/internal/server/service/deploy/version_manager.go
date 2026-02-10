package deploy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"sync"
	"time"
)

// 版本文件存储目录
const VersionsDir = "/opt/control/versions"

// ListVersions 获取版本列表
func ListVersions(req model.ListVersionsReq) (model.ListVersionsResp, error) {
	var resp model.ListVersionsResp

	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	if req.ServiceName != "" {
		session = session.Where("service_name = ?", req.ServiceName)
	}

	// 分页
	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var versions []entity.ServiceVersions
	total, err := session.Desc("created_at").Limit(pageSize, (page-1)*pageSize).FindAndCount(&versions)
	if err != nil {
		return resp, fmt.Errorf("查询版本列表失败: %v", err)
	}

	resp.Total = total
	resp.List = make([]*model.VersionInfo, 0, len(versions))
	for _, v := range versions {
		resp.List = append(resp.List, &model.VersionInfo{
			Id:          v.Id,
			ServiceName: v.ServiceName,
			Version:     v.Version,
			FileHash:    v.FileHash,
			FileSize:    v.FileSize,
			FilePath:    v.FilePath,
			Changelog:   v.Changelog,
			IsCurrent:   v.IsCurrent == 1,
			UploadedBy:  v.UploadedBy,
			CreatedAt:   v.CreatedAt,
		})
	}

	return resp, nil
}

// UploadVersion 上传新版本
func UploadVersion(serviceName, version, changelog, operator string, fileReader io.Reader) (*model.VersionInfo, error) {
	// 确保版本目录存在
	versionDir := path.Join(VersionsDir, serviceName, version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return nil, fmt.Errorf("创建版本目录失败: %v", err)
	}

	// 获取实际的二进制文件名
	binaryName, ok := model.ServiceBinaryNames[serviceName]
	if !ok {
		binaryName = serviceName
	}

	// 写入文件并计算 hash
	filePath := path.Join(versionDir, binaryName)
	file, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %v", err)
	}

	hasher := sha256.New()
	teeReader := io.TeeReader(fileReader, hasher)
	fileSize, err := io.Copy(file, teeReader)
	file.Close()
	if err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("写入文件失败: %v", err)
	}

	// 设置可执行权限
	os.Chmod(filePath, 0755)

	fileHash := hex.EncodeToString(hasher.Sum(nil))

	// 检查版本是否已存在
	var existing entity.ServiceVersions
	has, _ := dbs.DBAdmin.Where("service_name = ? AND version = ?", serviceName, version).Get(&existing)
	if has {
		// 更新现有记录
		existing.FileHash = fileHash
		existing.FileSize = fileSize
		existing.FilePath = filePath
		existing.Changelog = changelog
		existing.UploadedBy = operator
		dbs.DBAdmin.ID(existing.Id).Cols("file_hash", "file_size", "file_path", "changelog", "uploaded_by").Update(&existing)
		return &model.VersionInfo{
			Id:          existing.Id,
			ServiceName: existing.ServiceName,
			Version:     existing.Version,
			FileHash:    existing.FileHash,
			FileSize:    existing.FileSize,
			FilePath:    existing.FilePath,
			Changelog:   existing.Changelog,
			IsCurrent:   existing.IsCurrent == 1,
			UploadedBy:  existing.UploadedBy,
			CreatedAt:   existing.CreatedAt,
		}, nil
	}

	// 插入新记录
	newVersion := &entity.ServiceVersions{
		ServiceName: serviceName,
		Version:     version,
		FileHash:    fileHash,
		FileSize:    fileSize,
		FilePath:    filePath,
		Changelog:   changelog,
		IsCurrent:   0,
		UploadedBy:  operator,
		CreatedAt:   time.Now(),
	}

	if _, err := dbs.DBAdmin.Insert(newVersion); err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("保存版本记录失败: %v", err)
	}

	return &model.VersionInfo{
		Id:          newVersion.Id,
		ServiceName: newVersion.ServiceName,
		Version:     newVersion.Version,
		FileHash:    newVersion.FileHash,
		FileSize:    newVersion.FileSize,
		FilePath:    newVersion.FilePath,
		Changelog:   newVersion.Changelog,
		IsCurrent:   false,
		UploadedBy:  newVersion.UploadedBy,
		CreatedAt:   newVersion.CreatedAt,
	}, nil
}

// SetCurrentVersion 设置当前版本
func SetCurrentVersion(versionId int) error {
	var version entity.ServiceVersions
	has, err := dbs.DBAdmin.ID(versionId).Get(&version)
	if err != nil {
		return fmt.Errorf("查询版本失败: %v", err)
	}
	if !has {
		return fmt.Errorf("版本不存在")
	}

	// 先取消该服务的所有当前版本标记
	dbs.DBAdmin.Table("service_versions").Where("service_name = ?", version.ServiceName).Update(map[string]interface{}{
		"is_current": 0,
	})

	// 设置新的当前版本
	dbs.DBAdmin.Table("service_versions").ID(versionId).Update(map[string]interface{}{
		"is_current": 1,
	})

	return nil
}

// DeleteVersion 删除版本
func DeleteVersion(versionId int) error {
	var version entity.ServiceVersions
	has, err := dbs.DBAdmin.ID(versionId).Get(&version)
	if err != nil {
		return fmt.Errorf("查询版本失败: %v", err)
	}
	if !has {
		return fmt.Errorf("版本不存在")
	}

	if version.IsCurrent == 1 {
		return fmt.Errorf("无法删除当前版本")
	}

	// 删除文件
	if version.FilePath != "" {
		os.RemoveAll(path.Dir(version.FilePath))
	}

	// 删除记录
	dbs.DBAdmin.ID(versionId).Delete(&entity.ServiceVersions{})

	return nil
}

// DeployVersion 部署版本到服务器
func DeployVersion(req model.DeployVersionReq, operator string) (model.DeployVersionResp, error) {
	var resp model.DeployVersionResp
	resp.StartedAt = time.Now()

	// 获取版本信息
	var version entity.ServiceVersions
	has, err := dbs.DBAdmin.ID(req.VersionId).Get(&version)
	if err != nil {
		return resp, fmt.Errorf("查询版本失败: %v", err)
	}
	if !has {
		return resp, fmt.Errorf("版本不存在")
	}

	// 检查文件是否存在
	if _, err := os.Stat(version.FilePath); os.IsNotExist(err) {
		return resp, fmt.Errorf("版本文件不存在: %s", version.FilePath)
	}

	// 获取目标服务器列表
	var servers []entity.Servers
	if err := dbs.DBAdmin.In("id", req.ServerIds).Find(&servers); err != nil {
		return resp, fmt.Errorf("查询服务器失败: %v", err)
	}

	resp.Total = len(servers)
	resp.Results = make([]*model.DeployResult, 0, len(servers))

	if req.Parallel {
		// 并行部署
		var wg sync.WaitGroup
		var mu sync.Mutex
		sem := make(chan struct{}, maxBatchConcurrent)

		for _, server := range servers {
			wg.Add(1)
			sem <- struct{}{}
			go func(srv entity.Servers) {
				defer wg.Done()
				defer func() { <-sem }()

				result := deployToServer(srv, version, operator)
				mu.Lock()
				resp.Results = append(resp.Results, result)
				if result.Success {
					resp.Success++
				} else {
					resp.Failed++
				}
				mu.Unlock()
			}(server)
		}
		wg.Wait()
	} else {
		// 顺序部署
		for _, server := range servers {
			result := deployToServer(server, version, operator)
			resp.Results = append(resp.Results, result)
			if result.Success {
				resp.Success++
			} else {
				resp.Failed++
			}
		}
	}

	resp.EndedAt = time.Now()
	return resp, nil
}

// deployToServer 部署到单台服务器
func deployToServer(server entity.Servers, version entity.ServiceVersions, operator string) *model.DeployResult {
	result := &model.DeployResult{
		ServerId:   server.Id,
		ServerName: server.Name,
	}
	startTime := time.Now()

	// 获取 SSH 客户端
	client, err := GetSSHClient(server.Id)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("连接服务器失败: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		recordDeployment(server.Id, version, 0, operator, result.Message, entity.DeployStatusFailed)
		return result
	}

	// 获取当前版本信息（用于回滚）
	var currentVersionId int
	remoteUploadPath := model.ServiceUploadPaths[version.ServiceName]
	binaryName := model.ServiceBinaryNames[version.ServiceName]
	remotePath := path.Join(remoteUploadPath, binaryName)

	// 获取当前部署的版本ID
	var lastDeploy entity.DeploymentRecords
	has, _ := dbs.DBAdmin.Where("server_id = ? AND service_name = ? AND status = ?",
		server.Id, version.ServiceName, entity.DeployStatusSuccess).
		Desc("id").Get(&lastDeploy)
	if has {
		currentVersionId = lastDeploy.VersionId
	}

	// 确保远程目录存在
	client.ExecuteCommand(fmt.Sprintf("mkdir -p %s", remoteUploadPath))

	// 备份当前文件
	ts := time.Now().Format("20060102_150405")
	backupPath := fmt.Sprintf("%s.%s.bak", remotePath, ts)
	client.ExecuteCommand(fmt.Sprintf("if [ -f '%s' ]; then cp -f '%s' '%s'; fi", remotePath, remotePath, backupPath))

	// 上传新版本
	localFile, err := os.Open(version.FilePath)
	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("打开版本文件失败: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		recordDeployment(server.Id, version, currentVersionId, operator, result.Message, entity.DeployStatusFailed)
		return result
	}
	defer localFile.Close()

	tmpPath := fmt.Sprintf("%s.%s.tmp", remotePath, ts)
	if err := client.SSHClient.UploadFile(tmpPath, localFile); err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("上传文件失败: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		recordDeployment(server.Id, version, currentVersionId, operator, result.Message, entity.DeployStatusFailed)
		return result
	}

	// 原子替换
	if _, err := client.ExecuteCommand(fmt.Sprintf("mv -f '%s' '%s'", tmpPath, remotePath)); err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("替换文件失败: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		recordDeployment(server.Id, version, currentVersionId, operator, result.Message, entity.DeployStatusFailed)
		return result
	}

	// 设置可执行权限
	client.ExecuteCommand(fmt.Sprintf("chmod +x '%s'", remotePath))

	// 重启服务
	systemdName := serviceSystemdNames[version.ServiceName]
	restartCmd := fmt.Sprintf("systemctl restart %s", systemdName)
	restartOutput, restartErr := client.ExecuteCommand(restartCmd)
	if restartErr != nil {
		result.Success = true
		result.Message = fmt.Sprintf("部署成功，但重启失败: %v (output: %s)", restartErr, restartOutput)
	} else {
		result.Success = true
		result.Message = fmt.Sprintf("部署成功，版本: %s", version.Version)
	}

	result.Duration = time.Since(startTime).Milliseconds()
	recordDeployment(server.Id, version, currentVersionId, operator, result.Message, entity.DeployStatusSuccess)

	return result
}

// recordDeployment 记录部署历史
func recordDeployment(serverId int, version entity.ServiceVersions, previousVersionId int, operator, output string, status int) {
	now := time.Now()
	record := &entity.DeploymentRecords{
		ServerId:          serverId,
		ServiceName:       version.ServiceName,
		VersionId:         version.Id,
		PreviousVersionId: previousVersionId,
		Action:            entity.DeployActionDeploy,
		Status:            status,
		Operator:          operator,
		Output:            output,
		StartedAt:         now,
		CompletedAt:       now,
		CreatedAt:         now,
	}
	dbs.DBAdmin.Insert(record)
}

// RollbackVersion 回滚到上一版本
func RollbackVersion(req model.RollbackReq, operator string) (model.RollbackResp, error) {
	var resp model.RollbackResp

	// 获取上一次成功的部署记录
	var lastDeploy entity.DeploymentRecords
	has, err := dbs.DBAdmin.Where("server_id = ? AND service_name = ? AND status = ?",
		req.ServerId, req.ServiceName, entity.DeployStatusSuccess).
		Desc("id").Get(&lastDeploy)
	if err != nil {
		return resp, fmt.Errorf("查询部署记录失败: %v", err)
	}
	if !has {
		return resp, fmt.Errorf("没有找到部署记录")
	}

	// 获取上一个版本
	if lastDeploy.PreviousVersionId == 0 {
		return resp, fmt.Errorf("没有可回滚的版本")
	}

	var previousVersion entity.ServiceVersions
	has, err = dbs.DBAdmin.ID(lastDeploy.PreviousVersionId).Get(&previousVersion)
	if err != nil || !has {
		return resp, fmt.Errorf("上一版本不存在")
	}

	// 获取当前版本信息
	var currentVersion entity.ServiceVersions
	dbs.DBAdmin.ID(lastDeploy.VersionId).Get(&currentVersion)

	// 执行回滚（实际上是部署上一个版本）
	deployReq := model.DeployVersionReq{
		VersionId: previousVersion.Id,
		ServerIds: []int{req.ServerId},
		Parallel:  false,
	}
	deployResp, err := DeployVersion(deployReq, operator)
	if err != nil {
		return resp, err
	}

	if len(deployResp.Results) > 0 && deployResp.Results[0].Success {
		resp.Success = true
		resp.Message = "回滚成功"
		resp.RolledBackTo = previousVersion.Version
		resp.PreviousVersion = currentVersion.Version

		// 记录回滚操作
		now := time.Now()
		record := &entity.DeploymentRecords{
			ServerId:          req.ServerId,
			ServiceName:       req.ServiceName,
			VersionId:         previousVersion.Id,
			PreviousVersionId: lastDeploy.VersionId,
			Action:            entity.DeployActionRollback,
			Status:            entity.DeployStatusSuccess,
			Operator:          operator,
			Output:            resp.Message,
			StartedAt:         now,
			CompletedAt:       now,
			CreatedAt:         now,
		}
		dbs.DBAdmin.Insert(record)
	} else {
		resp.Success = false
		if len(deployResp.Results) > 0 {
			resp.Message = deployResp.Results[0].Message
		} else {
			resp.Message = "回滚失败"
		}
	}

	return resp, nil
}

// GetDeploymentHistory 获取部署历史
func GetDeploymentHistory(req model.DeploymentHistoryReq) (model.DeploymentHistoryResp, error) {
	var resp model.DeploymentHistoryResp

	session := dbs.DBAdmin.NewSession()
	defer session.Close()

	if req.ServerId > 0 {
		session = session.Where("server_id = ?", req.ServerId)
	}
	if req.ServiceName != "" {
		session = session.Where("service_name = ?", req.ServiceName)
	}

	page := req.Page
	pageSize := req.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	var records []entity.DeploymentRecords
	total, err := session.Desc("id").Limit(pageSize, (page-1)*pageSize).FindAndCount(&records)
	if err != nil {
		return resp, fmt.Errorf("查询部署历史失败: %v", err)
	}

	// 获取服务器名称映射
	serverIds := make([]int, 0)
	versionIds := make([]int, 0)
	for _, r := range records {
		serverIds = append(serverIds, r.ServerId)
		versionIds = append(versionIds, r.VersionId)
		if r.PreviousVersionId > 0 {
			versionIds = append(versionIds, r.PreviousVersionId)
		}
	}

	serverMap := make(map[int]string)
	if len(serverIds) > 0 {
		var servers []entity.Servers
		dbs.DBAdmin.In("id", serverIds).Find(&servers)
		for _, s := range servers {
			serverMap[s.Id] = s.Name
		}
	}

	versionMap := make(map[int]string)
	if len(versionIds) > 0 {
		var versions []entity.ServiceVersions
		dbs.DBAdmin.In("id", versionIds).Find(&versions)
		for _, v := range versions {
			versionMap[v.Id] = v.Version
		}
	}

	resp.Total = total
	resp.List = make([]*model.DeploymentRecord, 0, len(records))

	statusTextMap := map[int]string{
		entity.DeployStatusPending: "进行中",
		entity.DeployStatusSuccess: "成功",
		entity.DeployStatusFailed:  "失败",
	}

	for _, r := range records {
		record := &model.DeploymentRecord{
			Id:                r.Id,
			ServerId:          r.ServerId,
			ServerName:        serverMap[r.ServerId],
			ServiceName:       r.ServiceName,
			VersionId:         r.VersionId,
			Version:           versionMap[r.VersionId],
			PreviousVersionId: r.PreviousVersionId,
			PreviousVersion:   versionMap[r.PreviousVersionId],
			Action:            r.Action,
			Status:            r.Status,
			StatusText:        statusTextMap[r.Status],
			Operator:          r.Operator,
			BackupPath:        r.BackupPath,
			Output:            r.Output,
			StartedAt:         r.StartedAt,
			CompletedAt:       r.CompletedAt,
			CreatedAt:         r.CreatedAt,
		}
		resp.List = append(resp.List, record)
	}

	return resp, nil
}
