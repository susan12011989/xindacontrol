package docker

import (
	"fmt"
	"server/internal/server/model"
	deployService "server/internal/server/service/deploy"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"
	"time"
)

// CheckServerHealth 检查单个服务器的健康状态
func CheckServerHealth(req model.HealthCheckReq) (model.HealthCheckResponse, error) {
	var resp model.HealthCheckResponse
	resp.ServerId = req.ServerId
	resp.CheckTime = time.Now().Format("2006-01-02 15:04:05")

	// 获取服务器信息
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", req.ServerId).Get(&server)
	if err != nil {
		return resp, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return resp, fmt.Errorf("服务器不存在")
	}

	resp.ServerName = server.Name
	resp.ServerHost = server.Host

	// 获取SSH客户端
	client, err := deployService.GetSSHClient(req.ServerId)
	if err != nil {
		resp.Overall = "unhealthy"
		resp.Services = append(resp.Services, model.HealthCheckItem{
			Name:    "SSH连接",
			Status:  "error",
			Message: fmt.Sprintf("SSH连接失败: %v", err),
		})
		return resp, nil
	}

	// 检查基础服务
	resp.Services = checkBaseServices(client)

	// 检查API健康
	resp.APIs = checkAPIHealth(client, server.Host)

	// 计算总体状态
	resp.Overall = calculateOverallStatus(resp.Services, resp.APIs)

	return resp, nil
}

// checkBaseServices 检查基础服务状态
func checkBaseServices(client interface{ ExecuteCommand(string) (string, error) }) []model.HealthCheckItem {
	var results []model.HealthCheckItem

	// 1. 检查 MySQL
	results = append(results, checkMySQL(client))

	// 2. 检查 Redis
	results = append(results, checkRedis(client))

	// 3. 检查 Docker
	results = append(results, checkDocker(client))

	// 4. 检查 Nginx
	results = append(results, checkNginx(client))

	return results
}

// checkMySQL 检查MySQL状态
func checkMySQL(client interface{ ExecuteCommand(string) (string, error) }) model.HealthCheckItem {
	item := model.HealthCheckItem{Name: "MySQL", ContainerName: "mysql"}

	start := time.Now()

	// 首先检查容器是否存在
	containerCmd := "docker ps -a --format '{{.Names}}:{{.Status}}' | grep -i mysql | head -1"
	containerOutput, _ := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", containerCmd))
	containerOutput = strings.TrimSpace(containerOutput)

	// 方法1：尝试docker方式检查mysql
	cmd := "docker exec $(docker ps -qf 'name=mysql' 2>/dev/null | head -1) mysqladmin ping 2>/dev/null || systemctl is-active mysql 2>/dev/null || systemctl is-active mysqld 2>/dev/null"
	output, err := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", cmd))

	item.Latency = time.Since(start).Milliseconds()

	if err != nil {
		// 尝试其他方式检查
		checkCmd := "pgrep -x mysql || pgrep -x mysqld || docker ps --format '{{.Names}}' | grep -i mysql"
		checkOutput, checkErr := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", checkCmd))
		if checkErr != nil || strings.TrimSpace(checkOutput) == "" {
			item.Status = "error"
			// 判断是容器停止还是不存在
			if containerOutput != "" && strings.Contains(containerOutput, "Exited") {
				item.Message = "MySQL容器已停止"
				item.Action = "start"
				item.ActionLabel = "启动容器"
			} else if containerOutput != "" {
				item.Message = "MySQL服务异常"
				item.Action = "restart"
				item.ActionLabel = "重启容器"
			} else {
				item.Message = "MySQL服务未部署"
				item.Action = "deploy"
				item.ActionLabel = "部署服务"
			}
			return item
		}
		item.Status = "ok"
		item.Message = "MySQL运行中"
		return item
	}

	output = strings.TrimSpace(output)
	if strings.Contains(output, "alive") || output == "active" {
		item.Status = "ok"
		item.Message = "MySQL运行正常"
	} else if output != "" {
		item.Status = "ok"
		item.Message = "MySQL运行中"
	} else {
		item.Status = "warning"
		item.Message = "MySQL状态未知"
		item.Action = "restart"
		item.ActionLabel = "重启容器"
	}

	return item
}

// checkRedis 检查Redis状态
func checkRedis(client interface{ ExecuteCommand(string) (string, error) }) model.HealthCheckItem {
	item := model.HealthCheckItem{Name: "Redis", ContainerName: "redis"}

	start := time.Now()

	// 首先检查容器是否存在
	containerCmd := "docker ps -a --format '{{.Names}}:{{.Status}}' | grep -i redis | head -1"
	containerOutput, _ := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", containerCmd))
	containerOutput = strings.TrimSpace(containerOutput)

	// 尝试多种方式检查Redis
	cmd := "redis-cli ping 2>/dev/null || docker exec $(docker ps -qf 'name=redis' 2>/dev/null | head -1) redis-cli ping 2>/dev/null || systemctl is-active redis 2>/dev/null"
	output, err := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", cmd))

	item.Latency = time.Since(start).Milliseconds()

	if err != nil {
		// 尝试检查进程
		checkCmd := "pgrep -x redis-server || docker ps --format '{{.Names}}' | grep -i redis"
		checkOutput, checkErr := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", checkCmd))
		if checkErr != nil || strings.TrimSpace(checkOutput) == "" {
			item.Status = "error"
			// 判断是容器停止还是不存在
			if containerOutput != "" && strings.Contains(containerOutput, "Exited") {
				item.Message = "Redis容器已停止"
				item.Action = "start"
				item.ActionLabel = "启动容器"
			} else if containerOutput != "" {
				item.Message = "Redis服务异常"
				item.Action = "restart"
				item.ActionLabel = "重启容器"
			} else {
				item.Message = "Redis服务未部署"
				item.Action = "deploy"
				item.ActionLabel = "部署服务"
			}
			return item
		}
		item.Status = "ok"
		item.Message = "Redis运行中"
		return item
	}

	output = strings.TrimSpace(output)
	if output == "PONG" || output == "active" {
		item.Status = "ok"
		item.Message = "Redis运行正常"
	} else if output != "" {
		item.Status = "ok"
		item.Message = "Redis运行中"
	} else {
		item.Status = "warning"
		item.Message = "Redis状态未知"
		item.Action = "restart"
		item.ActionLabel = "重启容器"
	}

	return item
}

// checkDocker 检查Docker状态
func checkDocker(client interface{ ExecuteCommand(string) (string, error) }) model.HealthCheckItem {
	item := model.HealthCheckItem{Name: "Docker"}

	start := time.Now()

	cmd := "docker info >/dev/null 2>&1 && echo 'ok' || echo 'error'"
	output, err := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", cmd))

	item.Latency = time.Since(start).Milliseconds()

	if err != nil {
		item.Status = "error"
		item.Message = "Docker服务异常"
		item.Action = "none"
		item.ActionLabel = "需手动处理"
		return item
	}

	output = strings.TrimSpace(output)
	if output == "ok" {
		item.Status = "ok"
		item.Message = "Docker运行正常"
	} else {
		item.Status = "error"
		item.Message = "Docker服务异常"
		item.Action = "none"
		item.ActionLabel = "需手动处理"
	}

	return item
}

// checkNginx 检查Nginx状态
func checkNginx(client interface{ ExecuteCommand(string) (string, error) }) model.HealthCheckItem {
	item := model.HealthCheckItem{Name: "Nginx", ContainerName: "nginx"}

	start := time.Now()

	// 首先检查容器是否存在
	containerCmd := "docker ps -a --format '{{.Names}}:{{.Status}}' | grep -i nginx | head -1"
	containerOutput, _ := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", containerCmd))
	containerOutput = strings.TrimSpace(containerOutput)

	cmd := "systemctl is-active nginx 2>/dev/null || docker ps --format '{{.Names}}' | grep -i nginx"
	output, err := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", cmd))

	item.Latency = time.Since(start).Milliseconds()

	output = strings.TrimSpace(output)
	if err != nil || output == "" {
		item.Status = "warning"
		// 判断是容器停止还是不存在
		if containerOutput != "" && strings.Contains(containerOutput, "Exited") {
			item.Message = "Nginx容器已停止"
			item.Action = "start"
			item.ActionLabel = "启动容器"
		} else if containerOutput != "" {
			item.Message = "Nginx服务异常"
			item.Action = "restart"
			item.ActionLabel = "重启容器"
		} else {
			item.Message = "Nginx未检测到"
			item.Action = "deploy"
			item.ActionLabel = "部署服务"
		}
		return item
	}

	if output == "active" || strings.Contains(output, "nginx") {
		item.Status = "ok"
		item.Message = "Nginx运行正常"
	} else {
		item.Status = "warning"
		item.Message = "Nginx状态未知"
		item.Action = "restart"
		item.ActionLabel = "重启容器"
	}

	return item
}

// checkAPIHealth 检查API健康状态
func checkAPIHealth(client interface{ ExecuteCommand(string) (string, error) }, serverHost string) []model.HealthCheckItem {
	var results []model.HealthCheckItem

	// 定义需要检查的API端点（GET请求）
	apiEndpoints := []struct {
		name          string
		path          string
		port          int
		description   string
		containerName string // 关联的容器名称
		isWebSocket   bool   // 是否是WebSocket端口
	}{
		// 业务程序端口
		{"商户管理后台(GET)", "/v1/health", 8084, "nginx反向代理到后端API", "manager", false},
		{"后端API直连(GET)", "/v1/health", 8090, "TangSengDaoDaoServer直接访问", "tsddserver", false},
		{"WuKongIM", "/health", 5001, "IM通讯核心服务", "wukongim", false},
		{"WebSocket", "/", 5200, "WebSocket连接端口", "wukongim", true},
		// V2 GOST + nginx 路径分发
		{"GOST服务", "/", 9394, "GOST relay代理 (API端口)", "gost", true},
		{"GOST统一入口", "/", 10443, "relay+tls统一入口(V2)", "gost", true},
		{"GOST TCP入口", "/", 10010, "relay+tls TCP长连接(V2)", "gost", true},
		{"nginx路径分发", "/health", 8080, "V2 nginx路径分发(/ws,/api/,/s3/)", "tsdd-nginx", false},
	}

	for _, api := range apiEndpoints {
		item := model.HealthCheckItem{Name: api.name, ContainerName: api.containerName}
		start := time.Now()

		var cmd string
		if api.isWebSocket {
			// WebSocket端口使用TCP连接测试，不使用HTTP请求
			cmd = fmt.Sprintf("(echo > /dev/tcp/127.0.0.1/%d) 2>/dev/null && echo 'ok' || echo 'fail'", api.port)
		} else {
			// 使用curl检查API，获取状态码
			cmd = fmt.Sprintf("curl -s -w '\\n%%{http_code}' --connect-timeout 5 --max-time 10 http://127.0.0.1:%d%s 2>/dev/null | tail -1 || echo '000'", api.port, api.path)
		}
		output, _ := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", cmd))

		item.Latency = time.Since(start).Milliseconds()
		output = strings.TrimSpace(output)

		if api.isWebSocket {
			// WebSocket端口检测结果处理
			if output == "ok" {
				item.Status = "ok"
				item.Message = fmt.Sprintf("%s 端口连通 (%dms)", api.description, item.Latency)
			} else {
				item.Status = "error"
				item.Message = fmt.Sprintf("端口 %d 无法连接 - 服务未启动", api.port)
			}
		} else {
			item.Status, item.Message = parseHTTPStatus(output, api.port, api.description, item.Latency)
		}

		// 如果状态异常，设置操作建议
		if item.Status == "error" {
			// 检查容器状态
			containerCmd := fmt.Sprintf("docker ps -a --format '{{.Names}}:{{.Status}}' | grep -i %s | head -1", api.containerName)
			containerOutput, _ := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", containerCmd))
			containerOutput = strings.TrimSpace(containerOutput)

			if containerOutput != "" && strings.Contains(containerOutput, "Exited") {
				item.Action = "start"
				item.ActionLabel = "启动容器"
			} else if containerOutput != "" {
				item.Action = "restart"
				item.ActionLabel = "重启容器"
			} else {
				item.Action = "deploy"
				item.ActionLabel = "部署服务"
			}
		}

		results = append(results, item)
	}

	// 检查 POST 方法支持（关键！验证 nginx 是否正确转发 POST 请求）
	results = append(results, checkPOSTMethodSupport(client)...)

	// 检查关键业务 API
	results = append(results, checkBusinessAPIs(client)...)

	// 验证nginx代理配置
	results = append(results, checkNginxProxyConfig(client))

	// V2: 检查 nginx 路径分发是否正确转发到各后端
	results = append(results, checkNginxPathRouting(client)...)

	return results
}

// parseHTTPStatus 解析 HTTP 状态码
func parseHTTPStatus(code string, port int, description string, latency int64) (status, message string) {
	switch code {
	case "200", "201", "204":
		return "ok", fmt.Sprintf("%s 响应正常 (%dms)", description, latency)
	case "401", "403":
		return "ok", fmt.Sprintf("%s 需要认证(服务正常)", description)
	case "404":
		return "warning", fmt.Sprintf("%s 路径不存在，但端口可连接", description)
	case "405":
		return "error", fmt.Sprintf("405方法不允许 - nginx未正确配置转发该HTTP方法")
	case "000":
		return "error", fmt.Sprintf("端口 %d 无法连接 - 服务未启动", port)
	case "502":
		return "error", fmt.Sprintf("502网关错误 - nginx无法连接后端服务")
	case "503":
		return "error", fmt.Sprintf("503服务不可用 - 后端过载或维护中")
	case "500":
		return "error", fmt.Sprintf("500内部错误 - 后端服务异常")
	case "504":
		return "error", fmt.Sprintf("504网关超时 - 后端响应过慢")
	default:
		return "warning", fmt.Sprintf("端口 %d 响应码 %s", port, code)
	}
}

// checkPOSTMethodSupport 检查 POST 方法是否被正确转发
func checkPOSTMethodSupport(client interface{ ExecuteCommand(string) (string, error) }) []model.HealthCheckItem {
	var results []model.HealthCheckItem

	// 测试 nginx (8084) 和直连 (8090) 的 POST 请求对比
	postTests := []struct {
		name string
		port int
		path string
	}{
		{"Nginx代理POST", 8084, "/v1/health"},
		{"后端直连POST", 8090, "/v1/health"},
	}

	for _, test := range postTests {
		item := model.HealthCheckItem{Name: test.name}
		start := time.Now()

		// 使用 POST 方法请求
		cmd := fmt.Sprintf(`curl -s -X POST -w '\n%%{http_code}' --connect-timeout 5 --max-time 10 -H "Content-Type: application/json" http://127.0.0.1:%d%s -d '{}' 2>/dev/null | tail -1 || echo '000'`, test.port, test.path)
		output, _ := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", cmd))

		item.Latency = time.Since(start).Milliseconds()
		output = strings.TrimSpace(output)

		switch output {
		case "200", "201", "204", "400", "401", "403", "404":
			// 这些状态码都说明 POST 请求被正确转发了
			item.Status = "ok"
			item.Message = fmt.Sprintf("POST请求正常转发 (状态码:%s, %dms)", output, item.Latency)
		case "405":
			item.Status = "error"
			item.Message = "POST请求返回405 - Nginx未配置转发POST方法"
		case "000":
			item.Status = "error"
			item.Message = fmt.Sprintf("端口 %d 无法连接", test.port)
		default:
			item.Status = "warning"
			item.Message = fmt.Sprintf("POST请求状态码: %s", output)
		}

		results = append(results, item)
	}

	// 如果 nginx POST 失败但直连成功，给出明确诊断
	if len(results) == 2 {
		nginxResult := results[0]
		directResult := results[1]
		if nginxResult.Status == "error" && directResult.Status == "ok" {
			// 添加诊断信息
			diagItem := model.HealthCheckItem{
				Name:    "POST代理诊断",
				Status:  "error",
				Message: "Nginx代理POST失败但后端正常 - 需要检查nginx配置，确保正确转发POST/PUT/DELETE方法",
			}
			results = append(results, diagItem)
		}
	}

	return results
}

// checkBusinessAPIs 检查关键业务 API
func checkBusinessAPIs(client interface{ ExecuteCommand(string) (string, error) }) []model.HealthCheckItem {
	var results []model.HealthCheckItem

	// 关键业务 API 列表（通过 nginx 代理）
	businessAPIs := []struct {
		name   string
		method string
		path   string
		body   string
	}{
		{"钱包API(POST)", "POST", "/v1/manager/wallet/recharge", `{"uid":"test","amount":0}`},
		{"红包API(POST)", "POST", "/v1/redpacket/send", `{}`},
		{"消息API(POST)", "POST", "/v1/message/sync", `{}`},
	}

	for _, api := range businessAPIs {
		item := model.HealthCheckItem{Name: api.name}
		start := time.Now()

		// 通过 nginx 代理测试
		cmd := fmt.Sprintf(`curl -s -X %s -w '\n%%{http_code}' --connect-timeout 5 --max-time 10 -H "Content-Type: application/json" http://127.0.0.1:8084%s -d '%s' 2>/dev/null | tail -1 || echo '000'`, api.method, api.path, api.body)
		output, _ := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", cmd))

		item.Latency = time.Since(start).Milliseconds()
		output = strings.TrimSpace(output)

		switch output {
		case "200", "201", "204":
			item.Status = "ok"
			item.Message = fmt.Sprintf("API正常 (%dms)", item.Latency)
		case "400":
			item.Status = "ok"
			item.Message = "API可访问(参数校验失败-正常)"
		case "401", "403":
			item.Status = "ok"
			item.Message = "API需要认证(转发正常)"
		case "404":
			item.Status = "warning"
			item.Message = "API路径不存在"
		case "405":
			item.Status = "error"
			item.Message = "405方法不允许 - Nginx未转发POST"
		case "000":
			item.Status = "error"
			item.Message = "无法连接"
		case "502":
			item.Status = "error"
			item.Message = "502网关错误"
		default:
			item.Status = "warning"
			item.Message = fmt.Sprintf("状态码: %s", output)
		}

		results = append(results, item)
	}

	return results
}

// checkNginxProxyConfig 检查nginx代理配置是否正确
func checkNginxProxyConfig(client interface{ ExecuteCommand(string) (string, error) }) model.HealthCheckItem {
	item := model.HealthCheckItem{Name: "Nginx代理配置"}
	start := time.Now()

	// 检查manager容器中的nginx配置
	cmd := `docker exec $(docker ps -qf 'name=manager' 2>/dev/null | head -1) cat /etc/nginx/conf.d/default.conf 2>/dev/null | grep -o 'proxy_pass http://[^;]*' | head -1 || echo 'not_found'`
	output, err := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", cmd))

	item.Latency = time.Since(start).Milliseconds()

	if err != nil || strings.TrimSpace(output) == "not_found" || strings.TrimSpace(output) == "" {
		item.Status = "warning"
		item.Message = "无法读取nginx代理配置"
		return item
	}

	output = strings.TrimSpace(output)
	// 检查代理目标端口是否合理
	if strings.Contains(output, ":8090") {
		item.Status = "ok"
		item.Message = fmt.Sprintf("代理配置正确: %s", output)
	} else if strings.Contains(output, ":5002") {
		item.Status = "warning"
		item.Message = fmt.Sprintf("代理配置可能有误: %s (应为8090)", output)
	} else {
		item.Status = "warning"
		item.Message = fmt.Sprintf("代理配置: %s", output)
	}

	return item
}

// calculateOverallStatus 计算总体健康状态
func calculateOverallStatus(services, apis []model.HealthCheckItem) string {
	errorCount := 0
	warningCount := 0
	total := len(services) + len(apis)

	for _, s := range services {
		switch s.Status {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		}
	}

	for _, a := range apis {
		switch a.Status {
		case "error":
			errorCount++
		case "warning":
			warningCount++
		}
	}

	if errorCount == 0 && warningCount == 0 {
		return "healthy"
	}
	if errorCount > total/2 {
		return "unhealthy"
	}
	return "partial"
}

// checkNginxPathRouting V2: 检查 nginx 路径分发是否正确转发到各后端
func checkNginxPathRouting(client interface{ ExecuteCommand(string) (string, error) }) []model.HealthCheckItem {
	var results []model.HealthCheckItem

	// 通过 nginx:8080 检测各路径是否可达后端
	paths := []struct {
		name string
		path string
		want string // 期望的 HTTP 状态码前缀
	}{
		{"nginx→WS(/ws)", "/ws", ""},       // WebSocket 升级请求返回 400 也算通（端口通）
		{"nginx→API(/api/)", "/api/v1/health", "200"},
		{"nginx→S3(/s3/)", "/s3/minio/health/live", "200"},
	}

	for _, p := range paths {
		item := model.HealthCheckItem{Name: p.name, ContainerName: "tsdd-nginx"}
		start := time.Now()

		if p.path == "/ws" {
			// WebSocket: 只检查端口连通性
			cmd := "(echo > /dev/tcp/127.0.0.1/8080) 2>/dev/null && echo 'ok' || echo 'fail'"
			output, _ := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", cmd))
			item.Latency = time.Since(start).Milliseconds()
			if strings.TrimSpace(output) == "ok" {
				item.Status = "ok"
				item.Message = fmt.Sprintf("nginx:8080 WS 路径可达 (%dms)", item.Latency)
			} else {
				item.Status = "error"
				item.Message = "nginx:8080 不可达，V2 路径分发异常"
			}
		} else {
			cmd := fmt.Sprintf("curl -s -w '\\n%%{http_code}' --connect-timeout 5 --max-time 10 http://127.0.0.1:8080%s 2>/dev/null | tail -1 || echo '000'", p.path)
			output, _ := client.ExecuteCommand(fmt.Sprintf("bash -c \"%s\"", cmd))
			item.Latency = time.Since(start).Milliseconds()
			code := strings.TrimSpace(output)

			if p.want != "" && strings.HasPrefix(code, p.want) {
				item.Status = "ok"
				item.Message = fmt.Sprintf("路径分发正常 HTTP %s (%dms)", code, item.Latency)
			} else if code == "000" {
				item.Status = "error"
				item.Message = "nginx:8080 不可达，请检查 tsdd-nginx 容器"
			} else if code == "502" {
				item.Status = "error"
				item.Message = fmt.Sprintf("502 后端不可达 - nginx 无法连接到后端服务")
			} else {
				item.Status = "warning"
				item.Message = fmt.Sprintf("HTTP %s（预期 %s）", code, p.want)
			}
		}

		results = append(results, item)
	}

	return results
}

// BatchCheckServerHealth 批量检查服务器健康状态
func BatchCheckServerHealth(req model.BatchHealthCheckReq) (model.BatchHealthCheckResponse, error) {
	var resp model.BatchHealthCheckResponse
	resp.Summary.Total = len(req.ServerIds)

	for _, serverId := range req.ServerIds {
		healthReq := model.HealthCheckReq{ServerId: serverId}
		result, err := CheckServerHealth(healthReq)
		if err != nil {
			result.Overall = "unhealthy"
			result.ServerId = serverId
		}

		resp.Results = append(resp.Results, result)

		switch result.Overall {
		case "healthy":
			resp.Summary.Healthy++
		case "unhealthy":
			resp.Summary.Unhealthy++
		case "partial":
			resp.Summary.Partial++
		}
	}

	return resp, nil
}
