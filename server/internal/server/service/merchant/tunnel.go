package merchant

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"server/internal/server/model"
	deploySvc "server/internal/server/service/deploy"
	"server/internal/server/utils"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"

	"github.com/zeromicro/go-zero/core/logx"
)

// probeE2e 通过 SSH 在系统服务器上 curl 本地隧道端口，验证端到端链路
func probeE2e(client *utils.PooledSSHClient, port int, path string, timeoutSec int) (success bool, message string) {
	cmd := fmt.Sprintf(
		`curl -s -o /dev/null -w '%%{http_code}' --connect-timeout %d --max-time %d http://127.0.0.1:%d%s 2>/dev/null || echo '000'`,
		timeoutSec, timeoutSec+3, port, path,
	)
	output, err := client.ExecuteCommandWithTimeout(cmd, time.Duration(timeoutSec+5)*time.Second)
	code := strings.TrimSpace(output)

	switch {
	case err != nil && code == "":
		return false, fmt.Sprintf("探测失败: %v", err)
	case code == "000" || code == "":
		return false, fmt.Sprintf("隧道不通(端口%d无响应)", port)
	default:
		return true, fmt.Sprintf("OK (HTTP %s)", code)
	}
}

// TunnelCheck 遍历所有系统服务器，执行两层检测：
// 1. 直连探测：系统服务器 → 商户GOST端口（验证端口可达）
// 2. 端到端探测：通过本地隧道端口 → TLS握手 → 商户业务端口（验证完整链路含握手）
func TunnelCheck(req model.TunnelCheckReq) ([]model.TunnelCheckItem, error) {
	// 1) 解析目标 IP 和商户端口
	targetIP := strings.TrimSpace(req.ServerIP)
	merchantPort := 0
	if req.MerchantId > 0 {
		var m entity.Merchants
		has, err := dbs.DBAdmin.Where("id = ?", req.MerchantId).Get(&m)
		if err != nil {
			return nil, fmt.Errorf("查询商户失败: %v", err)
		}
		if !has {
			return nil, fmt.Errorf("商户不存在")
		}
		if targetIP == "" {
			targetIP = strings.TrimSpace(m.ServerIP)
		}
		merchantPort = m.Port
	}
	// 如果只传了 server_ip，尝试通过 IP 反查商户端口
	if merchantPort == 0 && targetIP != "" {
		var m entity.Merchants
		has, _ := dbs.DBAdmin.Where("server_ip = ?", targetIP).Get(&m)
		if has {
			merchantPort = m.Port
		}
	}
	if targetIP == "" {
		return nil, fmt.Errorf("目标IP不能为空")
	}

	// 2) 获取所有系统服务器
	var servers []entity.Servers
	if err := dbs.DBAdmin.Where("server_type = ? AND status = 1", 2).Find(&servers); err != nil {
		return nil, fmt.Errorf("查询系统服务器失败: %v", err)
	}
	if len(servers) == 0 {
		return []model.TunnelCheckItem{}, nil
	}

	// 3) 构造直连检测命令
	port := gostapi.TargetPortTCP
	const timeoutSec = 5
	directCmd := fmt.Sprintf(`IP="%s"; PORT=%d; TIMEOUT=%d
sh -c '
if command -v nc >/dev/null 2>&1; then
  nc -z -w '"$TIMEOUT"' '"$IP"' '"$PORT"' && echo OK || (echo FAIL; exit 1)
elif command -v timeout >/dev/null 2>&1 && command -v bash >/dev/null 2>&1; then
  timeout '"$TIMEOUT"' bash -c "</dev/tcp/$IP/$PORT" && echo OK || (echo FAIL; exit 1)
elif command -v telnet >/dev/null 2>&1; then
  (echo quit | telnet '"$IP"' '"$PORT"') >/dev/null 2>&1 && echo OK || (echo FAIL; exit 1)
elif command -v python3 >/dev/null 2>&1; then
  python3 - <<PY
import socket,sys
s=socket.socket(); s.settimeout('"$TIMEOUT"')
ret = s.connect_ex(("'"$IP"'", '"$PORT"'))
sys.exit(0 if ret==0 else 1)
PY
  [ $? -eq 0 ] && echo OK || (echo FAIL; exit 1)
else
  echo FAIL && exit 1
fi'`, targetIP, port, timeoutSec)

	// 4) 并发检测
	results := make([]model.TunnelCheckItem, 0, len(servers))
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

			item := model.TunnelCheckItem{
				ServerName:  server.Name,
				ServerIP:    server.Host,
				ForwardType: forwardType,
			}

			client, err := deploySvc.GetSSHClient(server.Id)
			if err != nil {
				item.Message = fmt.Sprintf("获取SSH失败: %v", err)
				item.E2eMessage = "跳过（SSH不可用）"
				mu.Lock()
				results = append(results, item)
				mu.Unlock()
				return
			}

			// === 直连探测 ===
			output, execErr := client.ExecuteCommandWithTimeout(directCmd, (timeoutSec+3)*time.Second)
			out := strings.TrimSpace(output)
			if execErr == nil {
				item.Success = true
				item.Message = "OK"
				if out != "" && out != "OK" {
					item.Message = out
				}
			} else {
				if out != "" {
					item.Message = fmt.Sprintf("%s | %v", out, execErr)
				} else {
					item.Message = execErr.Error()
				}
			}

			// === 端到端探测 ===
			if merchantPort > 0 {
				// V2: 统一入口探测 → relay+tls → 商户:10443 → nginx → 业务程序
				e2ePort := merchantPort + gostapi.PortOffsetHTTP
				item.E2eSuccess, item.E2eMessage = probeE2e(client, e2ePort, "/", timeoutSec)

				// MinIO 探测（走统一入口 /s3/ 路径）
				minioPort := merchantPort + gostapi.PortOffsetMinIO
				item.MinioE2eSuccess, item.MinioE2eMessage = probeE2e(client, minioPort, "/minio/health/live", timeoutSec)
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

	// 5) 排序：e2e失败优先 > 直连失败优先 > 按名称
	sort.Slice(results, func(i, j int) bool {
		// 先按 e2e 失败排序
		if results[i].E2eSuccess != results[j].E2eSuccess {
			return !results[i].E2eSuccess && results[j].E2eSuccess
		}
		// 再按直连失败排序
		if results[i].Success != results[j].Success {
			return !results[i].Success && results[j].Success
		}
		return results[i].ServerName < results[j].ServerName
	})

	logx.Infof("tunnel check to %s (port %d) on %d servers finished", targetIP, merchantPort, len(servers))
	return results, nil
}

// GetTunnelStats 获取隧道统计信息
func GetTunnelStats() (*model.TunnelStats, error) {
	stats := &model.TunnelStats{}

	// 1. 统计商户总数（有效商户）
	merchantCount, err := dbs.DBAdmin.Where("status = 1 AND expired_at > ?", time.Now()).Count(&entity.Merchants{})
	if err != nil {
		logx.Errorf("count merchants err: %+v", err)
		return nil, err
	}
	stats.TotalMerchants = int(merchantCount)

	// 2. 统计系统服务器总数（启用状态）
	gostServerCount, err := dbs.DBAdmin.Where("server_type = 2 AND status = 1").Count(&entity.Servers{})
	if err != nil {
		logx.Errorf("count gost servers err: %+v", err)
		return nil, err
	}
	stats.TotalGostServers = int(gostServerCount)

	// 3. 统计商户服务器总数
	merchantServerCount, err := dbs.DBAdmin.Where("server_type = 1 AND status = 1").Count(&entity.Servers{})
	if err != nil {
		logx.Errorf("count merchant servers err: %+v", err)
		return nil, err
	}
	stats.TotalMerchantServers = int(merchantServerCount)

	return stats, nil
}
