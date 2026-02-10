package merchant

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"server/internal/server/model"
	deploySvc "server/internal/server/service/deploy"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"

	"github.com/zeromicro/go-zero/core/logx"
)

// TunnelCheck 遍历所有系统服务器，通过 SSH 在远端发起对 targetIP:gostPort 的 TCP 探测
func TunnelCheck(req model.TunnelCheckReq) ([]model.TunnelCheckItem, error) {
	// 1) 解析目标 IP
	targetIP := strings.TrimSpace(req.ServerIP)
	if targetIP == "" && req.MerchantId > 0 {
		var m entity.Merchants
		has, err := dbs.DBAdmin.Where("id = ?", req.MerchantId).Get(&m)
		if err != nil {
			return nil, fmt.Errorf("查询商户失败: %v", err)
		}
		if !has {
			return nil, fmt.Errorf("商户不存在")
		}
		targetIP = strings.TrimSpace(m.ServerIP)
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

	// 3) 构造远端检测命令（带超时与多工具降级）
	// 退出码 0=成功，非0=失败；stdout 会输出 OK/FAIL
	// 商户服务器 GOST 固定监听端口为 10000（TCP）
	port := gostapi.TargetPortTCP
	const timeoutSec = 5
	cmd := fmt.Sprintf(`IP="%s"; PORT=%d; TIMEOUT=%d
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

			item := model.TunnelCheckItem{
				ServerName: server.Name,
				ServerIP:   server.Host,
				Success:    false,
				Message:    "",
			}

			client, err := deploySvc.GetSSHClient(server.Id)
			if err != nil {
				item.Success = false
				item.Message = fmt.Sprintf("获取SSH失败: %v", err)
				mu.Lock()
				results = append(results, item)
				mu.Unlock()
				return
			}

			// 执行命令（再加一层超时保护）
			output, execErr := client.ExecuteCommandWithTimeout(cmd, (timeoutSec+3)*time.Second)
			out := strings.TrimSpace(output)
			if execErr == nil {
				item.Success = true
				if out == "" {
					item.Message = "OK"
				} else {
					item.Message = out
				}
			} else {
				// 远端命令返回非0或其他错误
				item.Success = false
				if out != "" {
					item.Message = fmt.Sprintf("%s | %v", out, execErr)
				} else {
					item.Message = execErr.Error()
				}
			}

			mu.Lock()
			results = append(results, item)
			mu.Unlock()
		}()
	}
	wg.Wait()

	// 5) 统一返回（失败置顶）
	sort.Slice(results, func(i, j int) bool {
		if results[i].Success == results[j].Success {
			return results[i].ServerName < results[j].ServerName
		}
		return !results[i].Success && results[j].Success
	})

	// 失败项将由前端高亮，这里保持结构清晰
	logx.Infof("tunnel check to %s:%d on %d servers finished", targetIP, port, len(servers))
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
