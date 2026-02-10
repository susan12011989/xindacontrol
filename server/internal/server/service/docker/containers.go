package docker

import (
	"errors"
	"fmt"
	"server/internal/server/model"
	deployService "server/internal/server/service/deploy"
	"server/internal/server/utils"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// executeDockerCommand 执行docker命令的辅助函数
func executeDockerCommand(client *utils.PooledSSHClient, dockerCmd string) (string, error) {
	// 方案1: 尝试直接执行
	cmd := fmt.Sprintf("bash -l -c \"%s\"", dockerCmd)
	output, err := client.ExecuteCommand(cmd)

	// 如果成功，直接返回
	if err == nil {
		return output, nil
	}

	// 方案2: 尝试使用 sudo（如果用户有sudo权限）
	sudoCmd := fmt.Sprintf("bash -l -c \"sudo %s\"", dockerCmd)
	sudoOutput, sudoErr := client.ExecuteCommand(sudoCmd)
	if sudoErr == nil {
		return sudoOutput, nil
	}

	// 方案3: 尝试使用完整路径
	fullPathCmd := fmt.Sprintf("bash -l -c \"/usr/bin/%s\"", dockerCmd)
	fullPathOutput, fullPathErr := client.ExecuteCommand(fullPathCmd)
	if fullPathErr == nil {
		return fullPathOutput, nil
	}

	// 所有方案都失败，返回原始错误和输出
	return output, fmt.Errorf("%v (原始输出: %s)", err, output)
}

// QueryContainers 查询容器列表
func QueryContainers(req model.QueryDockerContainersReq) (model.QueryDockerContainersResponse, error) {
	var resp model.QueryDockerContainersResponse

	// 获取服务器信息
	type ServerWithMerchant struct {
		entity.Servers `xorm:"extends"`
		MerchantName   string `xorm:"merchants.name"`
	}
	var server ServerWithMerchant

	has, err := dbs.DBAdmin.Table("servers").
		Join("LEFT", "merchants", "servers.merchant_id = merchants.id").
		Where("servers.id = ?", req.ServerId).
		Get(&server)
	if err != nil {
		return resp, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return resp, errors.New("服务器不存在")
	}

	// 填充服务器信息
	resp.ServerInfo = model.ServerInfo{
		Id:           server.Id,
		MerchantId:   server.MerchantId,
		Name:         server.Name,
		Host:         server.Host,
		MerchantName: server.MerchantName,
	}

	// 获取SSH客户端（使用连接池，自动管理连接）
	client, err := deployService.GetSSHClient(req.ServerId)
	if err != nil {
		return resp, err
	}

	// 构建docker ps命令
	statusFlag := "-a" // 默认显示所有
	if req.Status == "running" {
		statusFlag = ""
	} else if req.Status == "exited" {
		statusFlag = "-a --filter status=exited"
	}

	// 执行 docker ps 命令
	dockerCmd := fmt.Sprintf("docker ps %s --format '{{.ID}}|{{.Names}}|{{.Image}}|{{.Status}}|{{.State}}|{{.Ports}}|{{.CreatedAt}}'", statusFlag)
	output, err := executeDockerCommand(client, dockerCmd)
	if err != nil {
		return resp, fmt.Errorf("执行docker ps命令失败: %v", err)
	}

	// 解析输出
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 7 {
			continue
		}

		container := model.DockerContainerResp{
			ContainerId: parts[0],
			Name:        parts[1],
			Image:       parts[2],
			Status:      parts[3],
			State:       parts[4],
			Ports:       parts[5],
			CreatedAt:   parts[6],
		}

		// 如果有名称过滤
		if req.Name != "" && !strings.Contains(container.Name, req.Name) {
			continue
		}

		resp.List = append(resp.List, container)
	}

	resp.Total = len(resp.List)
	return resp, nil
}

// GetContainerStats 获取容器资源使用情况
func GetContainerStats(serverId int) ([]model.DockerContainerStatsResp, error) {
	var stats []model.DockerContainerStatsResp

	// 获取SSH客户端（使用连接池，自动管理连接）
	client, err := deployService.GetSSHClient(serverId)
	if err != nil {
		return stats, err
	}

	// 执行 docker stats 命令
	dockerCmd := "docker stats --no-stream --format '{{.Container}}|{{.Name}}|{{.CPUPerc}}|{{.MemUsage}}|{{.MemPerc}}|{{.NetIO}}|{{.BlockIO}}|{{.PIDs}}'"
	output, err := executeDockerCommand(client, dockerCmd)
	if err != nil {
		return stats, fmt.Errorf("执行docker stats命令失败: %v", err)
	}

	// 解析输出
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := strings.Split(line, "|")
		if len(parts) < 8 {
			continue
		}

		stat := model.DockerContainerStatsResp{
			ContainerId: parts[0],
			Name:        parts[1],
			CPUPerc:     parts[2],
			MemUsage:    parts[3],
			MemPerc:     parts[4],
			NetIO:       parts[5],
			BlockIO:     parts[6],
			Pids:        parts[7],
		}

		stats = append(stats, stat)
	}

	return stats, nil
}

// GetContainerLogs 获取容器日志
func GetContainerLogs(req model.GetDockerLogsReq) (model.GetDockerLogsResponse, error) {
	var resp model.GetDockerLogsResponse

	// 获取SSH客户端（使用连接池，自动管理连接）
	client, err := deployService.GetSSHClient(req.ServerId)
	if err != nil {
		return resp, err
	}

	// 构建docker logs命令
	dockerCmd := fmt.Sprintf("docker logs %s", req.ContainerId)

	if req.Lines > 0 {
		dockerCmd += fmt.Sprintf(" --tail %d", req.Lines)
	} else {
		dockerCmd += " --tail 100" // 默认100行
	}

	if req.Since != "" {
		dockerCmd += fmt.Sprintf(" --since '%s'", req.Since)
	}

	if req.Until != "" {
		dockerCmd += fmt.Sprintf(" --until '%s'", req.Until)
	}

	if req.Timestamps {
		dockerCmd += " --timestamps"
	}

	// 执行 docker logs 命令
	output, err := executeDockerCommand(client, dockerCmd)
	if err != nil {
		return resp, fmt.Errorf("读取日志失败: %v", err)
	}

	// 获取容器名称
	dockerNameCmd := fmt.Sprintf("docker inspect --format='{{.Name}}' %s", req.ContainerId)
	containerName, _ := executeDockerCommand(client, dockerNameCmd)
	containerName = strings.TrimSpace(strings.TrimPrefix(containerName, "/"))

	resp.Logs = output
	resp.TotalLines = len(strings.Split(output, "\n"))
	resp.ContainerId = req.ContainerId
	resp.ContainerName = containerName

	return resp, nil
}

// OperateContainer 操作容器
func OperateContainer(req model.DockerContainerOperationReq, operator string) (model.DockerOperationResponse, error) {
	var resp model.DockerOperationResponse

	// 获取服务器信息（需要merchant_id记录历史）
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", req.ServerId).Get(&server)
	if err != nil {
		return resp, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return resp, errors.New("服务器不存在")
	}

	// 获取SSH客户端（使用连接池，自动管理连接）
	client, err := deployService.GetSSHClient(req.ServerId)
	if err != nil {
		return resp, err
	}

	// 获取容器名称
	dockerNameCmd := fmt.Sprintf("docker inspect --format='{{.Name}}' %s", req.ContainerId)
	containerName, _ := executeDockerCommand(client, dockerNameCmd)
	containerName = strings.TrimSpace(strings.TrimPrefix(containerName, "/"))

	// 构建命令
	var dockerCmd string
	switch req.Action {
	case "start":
		dockerCmd = fmt.Sprintf("docker start %s", req.ContainerId)
	case "stop":
		dockerCmd = fmt.Sprintf("docker stop %s", req.ContainerId)
	case "restart":
		dockerCmd = fmt.Sprintf("docker restart %s", req.ContainerId)
	case "remove":
		if req.Force {
			dockerCmd = fmt.Sprintf("docker rm -f %s", req.ContainerId)
		} else {
			dockerCmd = fmt.Sprintf("docker rm %s", req.ContainerId)
		}
	default:
		return resp, fmt.Errorf("不支持的操作: %s", req.Action)
	}

	// 执行命令
	output, err := executeDockerCommand(client, dockerCmd)

	// 记录操作历史
	historyStatus := 1 // 成功
	errMsg := ""
	if err != nil {
		historyStatus = 2 // 失败
		errMsg = err.Error()
	}

	history := entity.DockerOperationHistory{
		ServerId:      req.ServerId,
		MerchantId:    server.MerchantId,
		ContainerId:   req.ContainerId,
		ContainerName: containerName,
		Action:        req.Action,
		Operator:      operator,
		Status:        historyStatus,
		Output:        output,
		ErrorMsg:      errMsg,
	}
	dbs.DBAdmin.Insert(&history)

	// 构造响应
	resp.Success = (err == nil)
	if err != nil {
		resp.Message = fmt.Sprintf("操作失败: %v", err)
	} else {
		resp.Message = fmt.Sprintf("容器 %s 操作成功", containerName)
	}

	return resp, nil
}

// BatchOperateContainers 批量操作容器
func BatchOperateContainers(req model.DockerBatchOperationReq, operator string) (model.DockerOperationResponse, error) {
	var resp model.DockerOperationResponse
	resp.Success = true

	for _, containerId := range req.ContainerIds {
		operateReq := model.DockerContainerOperationReq{
			ServerId:    req.ServerId,
			ContainerId: containerId,
			Action:      req.Action,
			Force:       req.Force,
		}

		result := model.DockerOperationResult{
			ContainerId: containerId,
		}

		operateResp, err := OperateContainer(operateReq, operator)
		if err != nil {
			result.Success = false
			result.Message = err.Error()
			resp.Success = false
		} else {
			result.Success = operateResp.Success
			result.Message = operateResp.Message
			if !operateResp.Success {
				resp.Success = false
			}
		}

		resp.Results = append(resp.Results, result)
	}

	if resp.Success {
		resp.Message = "批量操作全部成功"
	} else {
		resp.Message = "批量操作部分失败，请查看详情"
	}

	return resp, nil
}

// QueryDockerHistory 查询Docker操作历史
func QueryDockerHistory(req model.QueryDockerHistoryReq) (model.QueryDockerHistoryResponse, error) {
	var resp model.QueryDockerHistoryResponse

	session := dbs.DBAdmin.Table("docker_operation_history").
		Join("LEFT", "servers", "docker_operation_history.server_id = servers.id").
		Join("LEFT", "merchants", "docker_operation_history.merchant_id = merchants.id")

	// 条件过滤
	if req.ServerId > 0 {
		session = session.Where("docker_operation_history.server_id = ?", req.ServerId)
	}
	if req.MerchantId > 0 {
		session = session.Where("docker_operation_history.merchant_id = ?", req.MerchantId)
	}
	if req.ContainerId != "" {
		session = session.Where("docker_operation_history.container_id = ?", req.ContainerId)
	}
	if req.Action != "" {
		session = session.Where("docker_operation_history.action = ?", req.Action)
	}

	// 分页查询（使用 FindAndCount 一次完成）
	offset := (req.Page - 1) * req.Size
	type HistoryWithInfo struct {
		entity.DockerOperationHistory `xorm:"extends"`
		ServerName                    string `xorm:"servers.name"`
		MerchantName                  string `xorm:"merchants.name"`
	}
	var histories []HistoryWithInfo
	total, err := session.Desc("docker_operation_history.id").
		Limit(req.Size, offset).
		FindAndCount(&histories)
	if err != nil {
		logx.Errorf("query docker history err: %+v", err)
		return resp, err
	}
	resp.Total = int(total)

	// 转换为响应格式
	for _, h := range histories {
		resp.List = append(resp.List, model.DockerHistoryResp{
			Id:            h.Id,
			ServerName:    h.ServerName,
			MerchantName:  h.MerchantName,
			ContainerId:   h.ContainerId,
			ContainerName: h.ContainerName,
			Action:        h.Action,
			Operator:      h.Operator,
			Status:        h.Status,
			Output:        h.Output,
			ErrorMsg:      h.ErrorMsg,
			CreatedAt:     h.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return resp, nil
}
