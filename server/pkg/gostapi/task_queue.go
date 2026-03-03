package gostapi

import (
	"context"
	"encoding/json"
	"fmt"
	"server/internal/server/utils"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/logx"
)

const (
	// Redis 队列 key
	GostTaskQueueKey = "gost:task:queue"
	// Redis 版本号 Hash key（用于任务去重/取消）
	GostTaskVersionKey = "gost:task:version"
	// 最大重试次数
	MaxRetryCount = 5
	// 重试间隔（秒）
	RetryIntervalSec = 10
	// 任务过期时间（小时）- 超过此时间的任务直接丢弃
	TaskExpireHours = 24
)

// TaskType 任务类型
type TaskType string

const (
	// TaskCreateMerchantLocalForwards 创建商户本地转发
	TaskCreateMerchantLocalForwards TaskType = "create_merchant_local_forwards"
	// TaskDeleteMerchantLocalForwards 删除商户本地转发
	TaskDeleteMerchantLocalForwards TaskType = "delete_merchant_local_forwards"
	// TaskCreateMerchantForwards 创建系统服务器转发（加密模式）
	TaskCreateMerchantForwards TaskType = "create_merchant_forwards"
	// TaskDeleteMerchantForwards 删除系统服务器转发（加密模式）
	TaskDeleteMerchantForwards TaskType = "delete_merchant_forwards"
	// TaskUpdateMerchantForwards 更新系统服务器转发（加密模式）
	TaskUpdateMerchantForwards TaskType = "update_merchant_forwards"
	// TaskCreateMerchantDirectForwards 创建系统服务器直连转发
	TaskCreateMerchantDirectForwards TaskType = "create_merchant_direct_forwards"
	// TaskDeleteMerchantDirectForwards 删除系统服务器直连转发
	TaskDeleteMerchantDirectForwards TaskType = "delete_merchant_direct_forwards"
	// TaskUpdateMerchantDirectForwards 更新系统服务器直连转发
	TaskUpdateMerchantDirectForwards TaskType = "update_merchant_direct_forwards"
)

// GostTask GOST 任务结构
type GostTask struct {
	ID             string    `json:"id"`               // 任务唯一 ID
	Type           TaskType  `json:"type"`             // 任务类型
	ServerIP       string    `json:"server_ip"`        // 目标服务器 IP
	BasePort       int       `json:"base_port"`        // 基础端口
	TargetIP       string    `json:"target_ip"`        // 转发目标 IP（系统服务器转发时使用）
	TargetBasePort int       `json:"target_base_port"` // 目标基础端口（自定义时使用，0 表示使用默认值）
	TlsListener    bool      `json:"tls_listener"`     // 监听端是否启用 TLS（客户端加密）
	Version        int64     `json:"version"`          // 任务版本号（用于检测是否被新任务取代）
	RetryCount     int       `json:"retry_count"`      // 已重试次数
	CreatedAt      time.Time `json:"created_at"`       // 创建时间
	LastRetryAt    time.Time `json:"last_retry_at"`    // 上次重试时间
	ErrorMessage   string    `json:"error_message"`    // 最后一次错误信息
}

// TaskQueue 任务队列管理器
type TaskQueue struct {
	rds     *redis.Client
	ctx     context.Context
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	running bool
	mu      sync.Mutex
}

var (
	taskQueue     *TaskQueue
	taskQueueOnce sync.Once
)

// InitTaskQueue 初始化任务队列（需要在程序启动时调用）
func InitTaskQueue(rds *redis.Client) {
	taskQueueOnce.Do(func() {
		ctx, cancel := context.WithCancel(context.Background())
		taskQueue = &TaskQueue{
			rds:    rds,
			ctx:    ctx,
			cancel: cancel,
		}
		// 程序重启时清除所有 GOST 任务
		taskQueue.clearAllTasks()
		taskQueue.Start()
	})
}

// GetTaskQueue 获取任务队列实例
func GetTaskQueue() *TaskQueue {
	return taskQueue
}

// clearAllTasks 清除所有任务（程序重启时调用）
func (tq *TaskQueue) clearAllTasks() {
	// 清除任务队列
	deleted, err := tq.rds.Del(tq.ctx, GostTaskQueueKey).Result()
	if err != nil {
		logx.Errorf("清除 GOST 任务队列失败: %+v", err)
	} else {
		logx.Infof("GOST 任务队列已清除，删除 key 数: %d", deleted)
	}

	// 清除版本号
	deleted, err = tq.rds.Del(tq.ctx, GostTaskVersionKey).Result()
	if err != nil {
		logx.Errorf("清除 GOST 任务版本号失败: %+v", err)
	} else {
		logx.Infof("GOST 任务版本号已清除，删除 key 数: %d", deleted)
	}
}

// Start 启动任务消费者
func (tq *TaskQueue) Start() {
	tq.mu.Lock()
	if tq.running {
		tq.mu.Unlock()
		return
	}
	tq.running = true
	tq.mu.Unlock()

	tq.wg.Add(1)
	go tq.consume()
	logx.Info("GOST task queue consumer started")
}

// Stop 停止任务消费者
func (tq *TaskQueue) Stop() {
	tq.mu.Lock()
	if !tq.running {
		tq.mu.Unlock()
		return
	}
	tq.running = false
	tq.mu.Unlock()

	tq.cancel()
	tq.wg.Wait()
	logx.Info("GOST task queue consumer stopped")
}

// getTaskKey 获取任务的版本控制 key
// 格式：{serverIP}:{basePort} 或 {serverIP}:local（本地转发）
func getTaskKey(task *GostTask) string {
	switch task.Type {
	case TaskCreateMerchantLocalForwards, TaskDeleteMerchantLocalForwards:
		return fmt.Sprintf("%s:local", task.ServerIP)
	default:
		return fmt.Sprintf("%s:%d", task.ServerIP, task.BasePort)
	}
}

// updateVersion 更新任务版本号并返回新版本
func (tq *TaskQueue) updateVersion(taskKey string) (int64, error) {
	version, err := tq.rds.HIncrBy(tq.ctx, GostTaskVersionKey, taskKey, 1).Result()
	if err != nil {
		return 0, fmt.Errorf("更新版本号失败: %w", err)
	}
	return version, nil
}

// getVersion 获取当前版本号
func (tq *TaskQueue) getVersion(taskKey string) (int64, error) {
	version, err := tq.rds.HGet(tq.ctx, GostTaskVersionKey, taskKey).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("获取版本号失败: %w", err)
	}
	return version, nil
}

// Enqueue 添加任务到队列
func (tq *TaskQueue) Enqueue(task *GostTask) error {
	if task.ID == "" {
		task.ID = fmt.Sprintf("%s_%s_%d_%d", task.Type, task.ServerIP, task.BasePort, time.Now().UnixNano())
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}

	// 获取任务 key 并更新版本号
	taskKey := getTaskKey(task)
	version, err := tq.updateVersion(taskKey)
	if err != nil {
		return err
	}
	task.Version = version

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("序列化任务失败: %w", err)
	}

	if err := tq.rds.RPush(tq.ctx, GostTaskQueueKey, data).Err(); err != nil {
		return fmt.Errorf("任务入队失败: %w", err)
	}

	logx.Infof("GOST task enqueued: type=%s, server=%s, port=%d, version=%d",
		task.Type, task.ServerIP, task.BasePort, version)
	return nil
}

// consume 消费任务
func (tq *TaskQueue) consume() {
	defer tq.wg.Done()

	for {
		select {
		case <-tq.ctx.Done():
			return
		default:
			// 使用 BLPOP 阻塞获取任务，超时 5 秒
			result, err := tq.rds.BLPop(tq.ctx, 5*time.Second, GostTaskQueueKey).Result()
			if err != nil {
				if err == redis.Nil || err == context.Canceled {
					continue
				}
				logx.Errorf("从队列获取任务失败: %+v", err)
				time.Sleep(time.Second)
				continue
			}

			if len(result) < 2 {
				continue
			}

			var task GostTask
			if err := json.Unmarshal([]byte(result[1]), &task); err != nil {
				logx.Errorf("解析任务失败: %+v, data=%s", err, result[1])
				continue
			}

			tq.processTask(&task)
		}
	}
}

// processTask 处理单个任务
func (tq *TaskQueue) processTask(task *GostTask) {
	// 检查任务是否过期（超过 24 小时直接丢弃）
	if time.Since(task.CreatedAt) > time.Duration(TaskExpireHours)*time.Hour {
		logx.Errorf("GOST task expired (created %v ago): type=%s, server=%s, port=%d",
			time.Since(task.CreatedAt).Round(time.Minute), task.Type, task.ServerIP, task.BasePort)
		return
	}

	// 检查任务版本号，如果有更新的任务则跳过当前任务
	taskKey := getTaskKey(task)
	currentVersion, err := tq.getVersion(taskKey)
	if err != nil {
		logx.Errorf("获取任务版本失败: %+v", err)
		// 获取版本失败时继续执行任务
	} else if task.Version < currentVersion {
		logx.Infof("GOST task skipped (outdated): type=%s, server=%s, port=%d, task_version=%d, current_version=%d",
			task.Type, task.ServerIP, task.BasePort, task.Version, currentVersion)
		return
	}

	// 检查服务器 IP 是否还存在，不存在则跳过任务
	if !tq.checkServerIPExists(task) {
		logx.Infof("GOST task skipped (server IP not found): type=%s, server=%s, port=%d",
			task.Type, task.ServerIP, task.BasePort)
		return
	}

	logx.Infof("processing GOST task: type=%s, server=%s, port=%d, retry=%d, version=%d",
		task.Type, task.ServerIP, task.BasePort, task.RetryCount, task.Version)

	switch task.Type {
	case TaskCreateMerchantLocalForwards:
		err = CreateMerchantLocalForwards(task.ServerIP)
	case TaskDeleteMerchantLocalForwards:
		err = DeleteMerchantLocalForwards(task.ServerIP)
	case TaskCreateMerchantForwards:
		if task.TlsListener {
			err = CreateMerchantForwardsWithTls(task.ServerIP, task.BasePort, task.TargetIP)
		} else {
			err = CreateMerchantForwards(task.ServerIP, task.BasePort, task.TargetIP)
		}
	case TaskDeleteMerchantForwards:
		err = DeleteMerchantForwards(task.ServerIP, task.BasePort)
	case TaskUpdateMerchantForwards:
		if task.TargetBasePort > 0 {
			if task.TlsListener {
				err = UpdateMerchantForwardsWithTargetPortWithTls(task.ServerIP, task.BasePort, task.TargetIP, task.TargetBasePort)
			} else {
				err = UpdateMerchantForwardsWithTargetPort(task.ServerIP, task.BasePort, task.TargetIP, task.TargetBasePort)
			}
		} else {
			if task.TlsListener {
				err = UpdateMerchantForwardsWithTls(task.ServerIP, task.BasePort, task.TargetIP)
			} else {
				err = UpdateMerchantForwards(task.ServerIP, task.BasePort, task.TargetIP)
			}
		}
	case TaskCreateMerchantDirectForwards:
		if task.TlsListener {
			err = CreateMerchantDirectForwardsWithTls(task.ServerIP, task.BasePort, task.TargetIP)
		} else {
			err = CreateMerchantDirectForwards(task.ServerIP, task.BasePort, task.TargetIP)
		}
	case TaskDeleteMerchantDirectForwards:
		err = DeleteMerchantDirectForwards(task.ServerIP, task.BasePort)
	case TaskUpdateMerchantDirectForwards:
		err = UpdateMerchantDirectForwards(task.ServerIP, task.BasePort, task.TargetIP)
	default:
		logx.Errorf("未知任务类型: %s", task.Type)
		return
	}

	if err != nil {
		task.RetryCount++
		task.LastRetryAt = time.Now()
		task.ErrorMessage = err.Error()

		if task.RetryCount >= MaxRetryCount {
			logx.Errorf("GOST task failed after %d retries: type=%s, server=%s, port=%d, error=%s",
				task.RetryCount, task.Type, task.ServerIP, task.BasePort, err.Error())
			return
		}

		// 重试前再次检查版本号，如果已被取代则不再重试
		currentVersion, _ = tq.getVersion(taskKey)
		if task.Version < currentVersion {
			logx.Infof("GOST task cancelled (superseded by newer task): type=%s, server=%s, port=%d",
				task.Type, task.ServerIP, task.BasePort)
			return
		}

		// 重新入队，延迟执行（保持原版本号，不更新）
		logx.Errorf("GOST task failed, will retry (%d/%d): type=%s, server=%s, port=%d, error=%s",
			task.RetryCount, MaxRetryCount, task.Type, task.ServerIP, task.BasePort, err.Error())

		// 异步延迟重新入队，不阻塞消费者处理其他任务
		retryTask := *task // 复制一份避免闭包问题
		go func() {
			time.Sleep(time.Duration(RetryIntervalSec) * time.Second)
			if err := tq.enqueueWithoutVersionUpdate(&retryTask); err != nil {
				logx.Errorf("任务重新入队失败: %+v", err)
			}
		}()
	} else {
		logx.Infof("GOST task completed: type=%s, server=%s, port=%d", task.Type, task.ServerIP, task.BasePort)
		// 异步健康检查：确认 GOST 热重载后所有端口正常监听
		serverIP := task.ServerIP
		go tq.healthCheckAndRecover(serverIP)
	}
}

// enqueueWithoutVersionUpdate 重新入队（不更新版本号，用于重试）
func (tq *TaskQueue) enqueueWithoutVersionUpdate(task *GostTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("序列化任务失败: %w", err)
	}

	if err := tq.rds.RPush(tq.ctx, GostTaskQueueKey, data).Err(); err != nil {
		return fmt.Errorf("任务入队失败: %w", err)
	}

	return nil
}

// EnqueueCreateMerchantLocalForwards 入队：创建商户本地转发
func EnqueueCreateMerchantLocalForwards(merchantServerIP string) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:     TaskCreateMerchantLocalForwards,
		ServerIP: merchantServerIP,
	})
}

// EnqueueDeleteMerchantLocalForwards 入队：删除商户本地转发
func EnqueueDeleteMerchantLocalForwards(merchantServerIP string) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:     TaskDeleteMerchantLocalForwards,
		ServerIP: merchantServerIP,
	})
}

// EnqueueCreateMerchantForwards 入队：创建系统服务器转发
func EnqueueCreateMerchantForwards(systemServerIP string, basePort int, targetIP string) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:     TaskCreateMerchantForwards,
		ServerIP: systemServerIP,
		BasePort: basePort,
		TargetIP: targetIP,
	})
}

// EnqueueCreateMerchantForwardsWithTls 入队：创建系统服务器转发（监听端启用 TLS）
func EnqueueCreateMerchantForwardsWithTls(systemServerIP string, basePort int, targetIP string) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:        TaskCreateMerchantForwards,
		ServerIP:    systemServerIP,
		BasePort:    basePort,
		TargetIP:    targetIP,
		TlsListener: true,
	})
}

// EnqueueDeleteMerchantForwards 入队：删除系统服务器转发
func EnqueueDeleteMerchantForwards(systemServerIP string, basePort int) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:     TaskDeleteMerchantForwards,
		ServerIP: systemServerIP,
		BasePort: basePort,
	})
}

// EnqueueUpdateMerchantForwards 入队：更新系统服务器转发（使用默认目标端口）
func EnqueueUpdateMerchantForwards(systemServerIP string, basePort int, targetIP string, tlsListener bool) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:        TaskUpdateMerchantForwards,
		ServerIP:    systemServerIP,
		BasePort:    basePort,
		TargetIP:    targetIP,
		TlsListener: tlsListener,
	})
}

// EnqueueUpdateMerchantForwardsWithTargetPort 入队：更新系统服务器转发（自定义目标端口）
func EnqueueUpdateMerchantForwardsWithTargetPort(systemServerIP string, basePort int, targetIP string, targetBasePort int, tlsListener bool) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:           TaskUpdateMerchantForwards,
		ServerIP:       systemServerIP,
		BasePort:       basePort,
		TargetIP:       targetIP,
		TargetBasePort: targetBasePort,
		TlsListener:    tlsListener,
	})
}

// ========== 直连转发任务入队函数 ==========

// EnqueueCreateMerchantDirectForwards 入队：创建系统服务器直连转发
func EnqueueCreateMerchantDirectForwards(systemServerIP string, basePort int, targetIP string) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:     TaskCreateMerchantDirectForwards,
		ServerIP: systemServerIP,
		BasePort: basePort,
		TargetIP: targetIP,
	})
}

// EnqueueCreateMerchantDirectForwardsWithTls 入队：创建系统服务器直连转发（监听端启用 TLS）
func EnqueueCreateMerchantDirectForwardsWithTls(systemServerIP string, basePort int, targetIP string) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:        TaskCreateMerchantDirectForwards,
		ServerIP:    systemServerIP,
		BasePort:    basePort,
		TargetIP:    targetIP,
		TlsListener: true,
	})
}

// EnqueueDeleteMerchantDirectForwards 入队：删除系统服务器直连转发
func EnqueueDeleteMerchantDirectForwards(systemServerIP string, basePort int) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:     TaskDeleteMerchantDirectForwards,
		ServerIP: systemServerIP,
		BasePort: basePort,
	})
}

// EnqueueUpdateMerchantDirectForwards 入队：更新系统服务器直连转发
func EnqueueUpdateMerchantDirectForwards(systemServerIP string, basePort int, targetIP string) error {
	if taskQueue == nil {
		return fmt.Errorf("task queue not initialized")
	}
	return taskQueue.Enqueue(&GostTask{
		Type:     TaskUpdateMerchantDirectForwards,
		ServerIP: systemServerIP,
		BasePort: basePort,
		TargetIP: targetIP,
	})
}

// checkServerIPExists 检查服务器 IP 是否还存在于数据库中
// 商户本地转发任务：检查 merchants 表的 server_ip
// 系统服务器转发任务：检查 servers 表的 host
func (tq *TaskQueue) checkServerIPExists(task *GostTask) bool {
	if dbs.DBAdmin == nil {
		// 数据库未初始化时默认允许执行
		logx.Errorf("checkServerIPExists: database not initialized, allowing task execution")
		return true
	}

	switch task.Type {
	case TaskCreateMerchantLocalForwards, TaskDeleteMerchantLocalForwards:
		// 检查商户服务器 IP 是否存在
		exists, err := dbs.DBAdmin.Where("server_ip = ?", task.ServerIP).Exist(&entity.Merchants{})
		if err != nil {
			logx.Errorf("checkServerIPExists: query merchant by server_ip failed: %+v", err)
			return true // 查询失败时默认允许执行
		}
		return exists
	default:
		// 检查系统服务器 IP 是否存在
		exists, err := dbs.DBAdmin.Where("host = ?", task.ServerIP).Exist(&entity.Servers{})
		if err != nil {
			logx.Errorf("checkServerIPExists: query server by host failed: %+v", err)
			return true // 查询失败时默认允许执行
		}
		return exists
	}
}

// ========== GOST 健康检查与自动恢复 ==========

// healthCheckAndRecover 在 GOST 任务完成后检查所有服务端口是否正常监听
// 如果发现端口异常，自动通过 SSH 执行 systemctl restart gost 恢复
func (tq *TaskQueue) healthCheckAndRecover(serverIP string) {
	// 等待 GOST 热重载生效
	time.Sleep(3 * time.Second)

	// 从 GOST API 获取所有已配置服务的监听端口
	expectedPorts, err := GetExpectedPorts(serverIP)
	if err != nil {
		logx.Errorf("GOST health check: 获取期望端口失败 server=%s: %v", serverIP, err)
		// API 不可达时，尝试直接通过 SSH 检查并重启
		tq.forceRestartGost(serverIP, "GOST API 不可达")
		return
	}
	if len(expectedPorts) == 0 {
		return
	}

	// 查询服务器 SSH 信息
	sshClient, err := tq.getSSHClientByIP(serverIP)
	if err != nil {
		logx.Errorf("GOST health check: 获取SSH连接失败 server=%s: %v", serverIP, err)
		return
	}

	// 检查端口
	missingPorts := checkGostPorts(sshClient, expectedPorts)
	if len(missingPorts) == 0 {
		logx.Infof("GOST health check passed: server=%s, all %d ports healthy", serverIP, len(expectedPorts))
		return
	}

	// 有端口缺失，尝试重启 GOST
	logx.Errorf("GOST health check FAILED: server=%s, missing ports=%v, attempting restart", serverIP, missingPorts)
	restartAndVerify(sshClient, serverIP, expectedPorts)
}

// forceRestartGost 当 GOST API 不可达时，强制通过 SSH 重启
func (tq *TaskQueue) forceRestartGost(serverIP string, reason string) {
	sshClient, err := tq.getSSHClientByIP(serverIP)
	if err != nil {
		logx.Errorf("GOST force restart: SSH连接失败 server=%s: %v", serverIP, err)
		return
	}

	logx.Errorf("GOST force restart: server=%s, reason=%s", serverIP, reason)
	output, err := sshClient.ExecuteCommandWithTimeout("sudo systemctl restart gost", 30*time.Second)
	if err != nil {
		logx.Errorf("GOST force restart failed: server=%s, output=%s, error=%v", serverIP, output, err)
	} else {
		logx.Infof("GOST force restart completed: server=%s", serverIP)
	}
}

// getSSHClientByIP 根据服务器 IP 查询数据库并获取 SSH 连接
func (tq *TaskQueue) getSSHClientByIP(serverIP string) (*utils.PooledSSHClient, error) {
	if dbs.DBAdmin == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}

	var server entity.Servers
	has, err := dbs.DBAdmin.Where("host = ? AND status = 1", serverIP).Get(&server)
	if err != nil {
		return nil, fmt.Errorf("查询服务器失败: %w", err)
	}
	if !has {
		return nil, fmt.Errorf("服务器不存在或已禁用: %s", serverIP)
	}

	pool := utils.GetSSHPool()
	key := fmt.Sprintf("server_%d", server.Id)
	return pool.GetOrCreateConnection(key, server.Host, server.Port, server.Username, server.Password, server.PrivateKey)
}

// checkGostPorts 通过 SSH 检查 GOST 进程实际监听的端口，返回缺失端口列表
func checkGostPorts(sshClient *utils.PooledSSHClient, expectedPorts []int) []int {
	// 使用 || true 确保 grep 无匹配时不报错
	output, _ := sshClient.ExecuteCommandWithTimeout("ss -tlnp 2>/dev/null | grep gost || true", 10*time.Second)

	// 解析实际监听端口
	listeningPorts := make(map[int]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.Contains(line, "gost") {
			continue
		}
		// ss -tlnp 输出格式:
		// LISTEN  0  4096  *:10000  *:*  users:(("gost",pid=...,fd=...))
		// 第4列(index 3)为 Local Address:Port
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			localAddr := fields[3]
			if idx := strings.LastIndex(localAddr, ":"); idx >= 0 {
				var port int
				if _, err := fmt.Sscanf(localAddr[idx+1:], "%d", &port); err == nil && port > 0 {
					listeningPorts[port] = true
				}
			}
		}
	}

	// 找出缺失端口
	var missing []int
	for _, port := range expectedPorts {
		if !listeningPorts[port] {
			missing = append(missing, port)
		}
	}
	return missing
}

// restartAndVerify 重启 GOST 并验证端口恢复
func restartAndVerify(sshClient *utils.PooledSSHClient, serverIP string, expectedPorts []int) {
	output, err := sshClient.ExecuteCommandWithTimeout("sudo systemctl restart gost", 30*time.Second)
	if err != nil {
		logx.Errorf("GOST restart failed: server=%s, output=%s, error=%v", serverIP, output, err)
		return
	}

	// 等待 GOST 重启完成
	time.Sleep(5 * time.Second)

	// 验证端口恢复
	missingPorts := checkGostPorts(sshClient, expectedPorts)
	if len(missingPorts) == 0 {
		logx.Infof("GOST recovery SUCCESS: server=%s, all ports restored after restart", serverIP)
	} else {
		logx.Errorf("GOST recovery FAILED: server=%s, still missing ports=%v after restart", serverIP, missingPorts)
	}
}
