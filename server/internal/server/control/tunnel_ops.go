package control

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
)

// --- 隧道检测相关类型 ---

// TunnelCheckRequest 隧道检测请求
type TunnelCheckRequest struct {
	MerchantId int    `json:"merchant_id"`
	ServerIP   string `json:"server_ip"`
}

// TunnelCheckItem 单个服务器的检测结果
type TunnelCheckItem struct {
	ServerName      string `json:"server_name"`
	ServerIP        string `json:"server_ip"`
	ForwardType     string `json:"forward_type"`
	Success         bool   `json:"success"`
	Message         string `json:"message"`
	E2eSuccess      bool   `json:"e2e_success"`
	E2eMessage      string `json:"e2e_message"`
	MinioE2eSuccess bool   `json:"minio_e2e_success"`
	MinioE2eMessage string `json:"minio_e2e_message"`
}

// ITunnelController 隧道检测接口
type ITunnelController interface {
	// TunnelCheck 执行隧道检测
	TunnelCheck(ctx context.Context, req TunnelCheckRequest) ([]TunnelCheckItem, error)
}

// --- 单机模式隧道控制器 ---

type localTunnelController struct {
	executor *LocalExecutor
}

func newLocalTunnelController(executor *LocalExecutor) *localTunnelController {
	return &localTunnelController{executor: executor}
}

// TunnelCheck 单机模式：在本机直接探测 GOST 端口
func (t *localTunnelController) TunnelCheck(ctx context.Context, req TunnelCheckRequest) ([]TunnelCheckItem, error) {
	targetIP, merchantPort, err := resolveTarget(req)
	if err != nil {
		return nil, err
	}

	item := TunnelCheckItem{
		ServerName:  "localhost",
		ServerIP:    "127.0.0.1",
		ForwardType: "local",
	}

	// 直连探测本地 GOST 端口
	port := gostapi.TargetPortTCP
	cmd := fmt.Sprintf(`nc -z -w 5 %s %d 2>/dev/null && echo OK || echo FAIL`, targetIP, port)
	result := t.executor.Execute(ctx, cmd)
	out := strings.TrimSpace(result.Output)
	if out == "OK" {
		item.Success = true
		item.Message = "OK"
	} else {
		item.Message = fmt.Sprintf("端口 %d 不可达", port)
	}

	// 端到端探测
	if merchantPort > 0 {
		item.E2eSuccess, item.E2eMessage = localProbeE2e(t.executor, ctx, merchantPort+gostapi.PortOffsetHTTP, "/")
		item.MinioE2eSuccess, item.MinioE2eMessage = localProbeE2e(t.executor, ctx, merchantPort+gostapi.PortOffsetMinIO, "/minio/health/live")
	} else {
		item.E2eMessage = "跳过（未找到商户端口）"
		item.MinioE2eMessage = "跳过（未找到商户端口）"
	}

	return []TunnelCheckItem{item}, nil
}

func localProbeE2e(executor *LocalExecutor, ctx context.Context, port int, path string) (bool, string) {
	cmd := fmt.Sprintf(
		`curl -s -o /dev/null -w '%%{http_code}' --connect-timeout 5 --max-time 8 http://127.0.0.1:%d%s 2>/dev/null || echo '000'`,
		port, path,
	)
	result := executor.Execute(ctx, cmd)
	code := strings.TrimSpace(result.Output)
	switch {
	case result.Err != nil && code == "":
		return false, fmt.Sprintf("探测失败: %v", result.Err)
	case code == "000" || code == "":
		return false, fmt.Sprintf("隧道不通(端口%d无响应)", port)
	default:
		return true, fmt.Sprintf("OK (HTTP %s)", code)
	}
}

// --- 多机模式隧道控制器 ---

type clusterTunnelController struct {
	cluster *ClusterController
}

func newClusterTunnelController(cluster *ClusterController) *clusterTunnelController {
	return &clusterTunnelController{cluster: cluster}
}

// TunnelCheck 多机模式：遍历所有系统服务器并发检测
func (t *clusterTunnelController) TunnelCheck(ctx context.Context, req TunnelCheckRequest) ([]TunnelCheckItem, error) {
	targetIP, merchantPort, err := resolveTarget(req)
	if err != nil {
		return nil, err
	}

	// 获取所有系统服务器
	var servers []entity.Servers
	if err := dbs.DBAdmin.Where("server_type = ? AND status = 1", 2).Find(&servers); err != nil {
		return nil, fmt.Errorf("查询系统服务器失败: %v", err)
	}
	if len(servers) == 0 {
		return []TunnelCheckItem{}, nil
	}

	// 构建直连检测命令
	port := gostapi.TargetPortTCP
	const timeoutSec = 5
	directCmd := buildDirectProbeCmd(targetIP, port, timeoutSec)

	// 并发检测
	results := make([]TunnelCheckItem, 0, len(servers))
	var wg sync.WaitGroup
	mu := &sync.Mutex{}

	for _, s := range servers {
		server := s
		wg.Add(1)
		go func() {
			defer wg.Done()

			forwardType := "encrypted"
			if server.ForwardType == entity.ForwardTypeDirect {
				forwardType = "direct"
			}

			item := TunnelCheckItem{
				ServerName:  server.Name,
				ServerIP:    server.Host,
				ForwardType: forwardType,
			}

			executor, err := t.cluster.getExecutor(server.Id)
			if err != nil {
				item.Message = fmt.Sprintf("获取连接失败: %v", err)
				item.E2eMessage = "跳过（连接不可用）"
				item.MinioE2eMessage = "跳过（连接不可用）"
				mu.Lock()
				results = append(results, item)
				mu.Unlock()
				return
			}

			// 直连探测
			result := executor.Execute(ctx, directCmd)
			out := strings.TrimSpace(result.Output)
			if result.Err == nil && (out == "OK" || out == "") {
				item.Success = true
				item.Message = "OK"
			} else {
				if out != "" {
					item.Message = out
				} else if result.Err != nil {
					item.Message = result.Err.Error()
				}
			}

			// 端到端探测
			if merchantPort > 0 {
				item.E2eSuccess, item.E2eMessage = remoteProbeE2e(executor, ctx, merchantPort+gostapi.PortOffsetHTTP, "/", timeoutSec)
				item.MinioE2eSuccess, item.MinioE2eMessage = remoteProbeE2e(executor, ctx, merchantPort+gostapi.PortOffsetMinIO, "/minio/health/live", timeoutSec)
			} else {
				item.E2eMessage = "跳过（未找到商户端口）"
				item.MinioE2eMessage = "跳过（未找到商户端口）"
			}

			mu.Lock()
			results = append(results, item)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// 排序：失败优先
	sort.Slice(results, func(i, j int) bool {
		if results[i].E2eSuccess != results[j].E2eSuccess {
			return !results[i].E2eSuccess && results[j].E2eSuccess
		}
		if results[i].Success != results[j].Success {
			return !results[i].Success && results[j].Success
		}
		return results[i].ServerName < results[j].ServerName
	})

	return results, nil
}

func remoteProbeE2e(executor *RemoteExecutor, ctx context.Context, port int, path string, timeoutSec int) (bool, string) {
	cmd := fmt.Sprintf(
		`curl -s -o /dev/null -w '%%{http_code}' --connect-timeout %d --max-time %d http://127.0.0.1:%d%s 2>/dev/null || echo '000'`,
		timeoutSec, timeoutSec+3, port, path,
	)
	result := executor.Execute(ctx, cmd)
	code := strings.TrimSpace(result.Output)
	switch {
	case result.Err != nil && code == "":
		return false, fmt.Sprintf("探测失败: %v", result.Err)
	case code == "000" || code == "":
		return false, fmt.Sprintf("隧道不通(端口%d无响应)", port)
	default:
		return true, fmt.Sprintf("OK (HTTP %s)", code)
	}
}

// --- 共享工具函数 ---

// resolveTarget 从请求中解析目标 IP 和商户端口
func resolveTarget(req TunnelCheckRequest) (string, int, error) {
	targetIP := strings.TrimSpace(req.ServerIP)
	merchantPort := 0

	if req.MerchantId > 0 {
		var m entity.Merchants
		has, err := dbs.DBAdmin.Where("id = ?", req.MerchantId).Get(&m)
		if err != nil {
			return "", 0, fmt.Errorf("查询商户失败: %v", err)
		}
		if !has {
			return "", 0, fmt.Errorf("商户不存在")
		}
		if targetIP == "" {
			targetIP = strings.TrimSpace(m.ServerIP)
		}
		merchantPort = m.Port
	}

	// 通过 IP 反查商户端口
	if merchantPort == 0 && targetIP != "" {
		var m entity.Merchants
		has, _ := dbs.DBAdmin.Where("server_ip = ?", targetIP).Get(&m)
		if has {
			merchantPort = m.Port
		}
	}

	if targetIP == "" {
		return "", 0, fmt.Errorf("目标IP不能为空")
	}
	return targetIP, merchantPort, nil
}

func buildDirectProbeCmd(targetIP string, port int, timeoutSec int) string {
	return fmt.Sprintf(`IP="%s"; PORT=%d; TIMEOUT=%d
sh -c '
if command -v nc >/dev/null 2>&1; then
  nc -z -w '"$TIMEOUT"' '"$IP"' '"$PORT"' && echo OK || (echo FAIL; exit 1)
elif command -v timeout >/dev/null 2>&1 && command -v bash >/dev/null 2>&1; then
  timeout '"$TIMEOUT"' bash -c "</dev/tcp/$IP/$PORT" && echo OK || (echo FAIL; exit 1)
else
  echo FAIL && exit 1
fi'`, targetIP, port, timeoutSec)
}

// unused import guard
var _ = time.Now
