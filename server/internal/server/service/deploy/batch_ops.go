package deploy

import (
	"fmt"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"sync"
	"time"
)

const (
	// 批量操作最大并发数
	maxBatchConcurrent = 10
)

// BatchServiceAction 批量执行服务操作
func BatchServiceAction(req model.BatchServiceActionReq, operator string) (model.BatchServiceActionResp, error) {
	var resp model.BatchServiceActionResp
	resp.TotalCount = len(req.ServerIds)

	// 验证服务名
	if _, ok := serviceSystemdNames[req.ServiceName]; !ok {
		return resp, fmt.Errorf("不支持的服务: %s，仅支持 server/wukongim/gost", req.ServiceName)
	}

	// 获取服务器列表
	var servers []entity.Servers
	err := dbs.DBAdmin.In("id", req.ServerIds).Where("status = 1").Find(&servers)
	if err != nil {
		return resp, fmt.Errorf("查询服务器列表失败: %v", err)
	}

	if len(servers) == 0 {
		return resp, fmt.Errorf("未找到有效的服务器")
	}

	// 构建服务器映射
	serverMap := make(map[int]entity.Servers)
	for _, s := range servers {
		serverMap[s.Id] = s
	}

	if req.Parallel {
		resp = executeBatchParallel(servers, req, operator)
	} else {
		resp = executeBatchSequential(servers, req, operator)
	}

	return resp, nil
}

// executeBatchParallel 并行执行批量操作
func executeBatchParallel(servers []entity.Servers, req model.BatchServiceActionReq, operator string) model.BatchServiceActionResp {
	resp := model.BatchServiceActionResp{
		TotalCount: len(servers),
		Results:    make([]model.BatchServiceResult, 0, len(servers)),
	}

	sem := make(chan struct{}, maxBatchConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, s := range servers {
		server := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := executeSingleServiceAction(server, req, operator)

			mu.Lock()
			resp.Results = append(resp.Results, result)
			if result.Success {
				resp.SuccessCount++
			} else {
				resp.FailCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	return resp
}

// executeBatchSequential 顺序执行批量操作
func executeBatchSequential(servers []entity.Servers, req model.BatchServiceActionReq, operator string) model.BatchServiceActionResp {
	resp := model.BatchServiceActionResp{
		TotalCount: len(servers),
		Results:    make([]model.BatchServiceResult, 0, len(servers)),
	}

	for _, server := range servers {
		result := executeSingleServiceAction(server, req, operator)
		resp.Results = append(resp.Results, result)
		if result.Success {
			resp.SuccessCount++
		} else {
			resp.FailCount++
		}
	}

	return resp
}

// executeSingleServiceAction 执行单个服务器的服务操作
func executeSingleServiceAction(server entity.Servers, req model.BatchServiceActionReq, operator string) model.BatchServiceResult {
	result := model.BatchServiceResult{
		ServerId:   server.Id,
		ServerName: server.Name,
		ServerHost: server.Host,
	}

	// 调用现有的 ServiceAction
	actionReq := model.ServiceActionReq{
		ServerId:    server.Id,
		ServiceName: req.ServiceName,
		Action:      req.Action,
	}

	actionResp, err := ServiceAction(actionReq, operator)
	if err != nil {
		result.Success = false
		result.Message = err.Error()
		return result
	}

	result.Success = actionResp.Success
	result.Message = actionResp.Message
	result.Output = actionResp.Output

	return result
}

// BatchHealthCheck 批量健康检查
func BatchHealthCheck(req model.BatchHealthCheckReq) (model.BatchHealthCheckResp, error) {
	var resp model.BatchHealthCheckResp
	resp.TotalCount = len(req.ServerIds)

	// 获取服务器列表
	var servers []entity.Servers
	err := dbs.DBAdmin.In("id", req.ServerIds).Where("status = 1").Find(&servers)
	if err != nil {
		return resp, fmt.Errorf("查询服务器列表失败: %v", err)
	}

	if len(servers) == 0 {
		return resp, fmt.Errorf("未找到有效的服务器")
	}

	// 并行执行健康检查
	sem := make(chan struct{}, maxBatchConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex
	resp.Results = make([]model.ServerHealthResult, 0, len(servers))

	for _, s := range servers {
		server := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := checkSingleServerHealth(server)

			mu.Lock()
			resp.Results = append(resp.Results, result)
			switch result.Status {
			case "healthy":
				resp.HealthyCount++
			case "unhealthy":
				resp.UnhealthyCount++
			case "partial":
				resp.PartialCount++
			default:
				resp.UnhealthyCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	return resp, nil
}

// checkSingleServerHealth 检查单个服务器健康状态
func checkSingleServerHealth(server entity.Servers) model.ServerHealthResult {
	result := model.ServerHealthResult{
		ServerId:   server.Id,
		ServerName: server.Name,
		ServerHost: server.Host,
		CheckTime:  time.Now().Format("2006-01-02 15:04:05"),
	}

	// 尝试获取 SSH 连接
	client, err := GetSSHClient(server.Id)
	if err != nil {
		result.Status = "unhealthy"
		result.Message = fmt.Sprintf("SSH连接失败: %v", err)
		return result
	}

	// 执行简单命令检查连通性
	output, err := client.ExecuteCommandWithTimeout("echo 'health_check'", 5*time.Second)
	if err != nil {
		result.Status = "unhealthy"
		result.Message = fmt.Sprintf("命令执行失败: %v", err)
		return result
	}

	if output == "" || output != "health_check\n" && output != "health_check" {
		result.Status = "partial"
		result.Message = "响应异常"
		return result
	}

	result.Status = "healthy"
	result.Message = "服务器正常"
	return result
}

// BatchCommand 批量执行命令
func BatchCommand(req model.BatchCommandReq, operator string) (model.BatchCommandResp, error) {
	var resp model.BatchCommandResp
	resp.TotalCount = len(req.ServerIds)

	// 获取服务器列表
	var servers []entity.Servers
	err := dbs.DBAdmin.In("id", req.ServerIds).Where("status = 1").Find(&servers)
	if err != nil {
		return resp, fmt.Errorf("查询服务器列表失败: %v", err)
	}

	if len(servers) == 0 {
		return resp, fmt.Errorf("未找到有效的服务器")
	}

	timeout := time.Duration(req.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	if req.Parallel {
		resp = executeBatchCommandParallel(servers, req.Command, timeout)
	} else {
		resp = executeBatchCommandSequential(servers, req.Command, timeout)
	}

	return resp, nil
}

// executeBatchCommandParallel 并行执行命令
func executeBatchCommandParallel(servers []entity.Servers, command string, timeout time.Duration) model.BatchCommandResp {
	resp := model.BatchCommandResp{
		TotalCount: len(servers),
		Results:    make([]model.BatchCommandResult, 0, len(servers)),
	}

	sem := make(chan struct{}, maxBatchConcurrent)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, s := range servers {
		server := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			result := executeSingleCommand(server, command, timeout)

			mu.Lock()
			resp.Results = append(resp.Results, result)
			if result.Success {
				resp.SuccessCount++
			} else {
				resp.FailCount++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()
	return resp
}

// executeBatchCommandSequential 顺序执行命令
func executeBatchCommandSequential(servers []entity.Servers, command string, timeout time.Duration) model.BatchCommandResp {
	resp := model.BatchCommandResp{
		TotalCount: len(servers),
		Results:    make([]model.BatchCommandResult, 0, len(servers)),
	}

	for _, server := range servers {
		result := executeSingleCommand(server, command, timeout)
		resp.Results = append(resp.Results, result)
		if result.Success {
			resp.SuccessCount++
		} else {
			resp.FailCount++
		}
	}

	return resp
}

// executeSingleCommand 在单个服务器上执行命令
func executeSingleCommand(server entity.Servers, command string, timeout time.Duration) model.BatchCommandResult {
	result := model.BatchCommandResult{
		ServerId:   server.Id,
		ServerName: server.Name,
		ServerHost: server.Host,
	}

	startTime := time.Now()

	client, err := GetSSHClient(server.Id)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("SSH连接失败: %v", err)
		result.Duration = time.Since(startTime).Milliseconds()
		return result
	}

	output, err := client.ExecuteCommandWithTimeout(command, timeout)
	result.Duration = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Success = false
		result.Output = output
		result.Error = err.Error()
		return result
	}

	result.Success = true
	result.Output = output
	return result
}

// GetMerchantAppServerIds 获取商户所有 App 节点的 server IDs
func GetMerchantAppServerIds(merchantId int) ([]int, error) {
	var nodes []entity.MerchantServiceNodes
	err := dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).
		In("role", "app", "all", entity.ServiceNodeRoleAPI, entity.ServiceNodeRoleAll).
		Find(&nodes)
	if err != nil {
		return nil, err
	}
	ids := make([]int, 0, len(nodes))
	for _, n := range nodes {
		if n.ServerId > 0 {
			ids = append(ids, n.ServerId)
		}
	}
	return ids, nil
}

// MerchantBatchRestart 重启商户所有 App 节点的指定服务
func MerchantBatchRestart(merchantId int, serviceName string, operator string) (model.BatchServiceActionResp, error) {
	serverIds, err := GetMerchantAppServerIds(merchantId)
	if err != nil {
		return model.BatchServiceActionResp{}, err
	}
	if len(serverIds) == 0 {
		return model.BatchServiceActionResp{}, fmt.Errorf("商户 %d 没有 App 节点", merchantId)
	}
	return BatchServiceAction(model.BatchServiceActionReq{
		ServerIds:   serverIds,
		ServiceName: serviceName,
		Action:      "restart",
		Parallel:    true,
	}, operator)
}

// MerchantBatchCommand 在商户所有 App 节点执行命令
func MerchantBatchCommand(merchantId int, command string, operator string) (model.BatchCommandResp, error) {
	serverIds, err := GetMerchantAppServerIds(merchantId)
	if err != nil {
		return model.BatchCommandResp{}, err
	}
	if len(serverIds) == 0 {
		return model.BatchCommandResp{}, fmt.Errorf("商户 %d 没有 App 节点", merchantId)
	}
	return BatchCommand(model.BatchCommandReq{
		ServerIds: serverIds,
		Command:   command,
		Parallel:  true,
		Timeout:   120,
	}, operator)
}
