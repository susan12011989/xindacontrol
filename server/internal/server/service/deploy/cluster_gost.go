package deploy

import (
	"context"
	"fmt"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

// SyncClusterGostForward 同步集群 GOST 转发配置
// 每个中转服务器都指向所有 App 节点，GOST 内置 round-robin 负载均衡 + 故障转移
//
// 端口映射：
//
//	GOST 监听 merchantPort+0/1/2/3 → relay+tls/tcp → AppNode:10010/10011/10012/10013
//	例：商户端口 10000 → GOST 监听 10000/10001/10002/10003 → 转发到 App 节点 10010/10011/10012/10013
//	其中 10013 为 MinIO presigned URL 直传隧道
func SyncClusterGostForward(merchantId int) ([]model.GostSyncResult, error) {
	// 1. 获取商户信息（含端口配置）
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.ID(merchantId).Get(&merchant)
	if err != nil {
		return nil, fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("商户 %d 不存在", merchantId)
	}

	merchantPort := merchant.Port
	if merchantPort == 0 {
		merchantPort = 10000 // 默认商户端口
	}

	// 2. 查商户所有已部署的 App 节点
	var appNodes []entity.ClusterNodes
	err = dbs.DBAdmin.Where("merchant_id = ? AND node_role = 'app' AND status = 'deployed'", merchantId).
		Find(&appNodes)
	if err != nil {
		return nil, fmt.Errorf("查询 App 节点失败: %v", err)
	}
	if len(appNodes) == 0 {
		return nil, fmt.Errorf("商户 %d 没有已部署的 App 节点", merchantId)
	}

	// 获取 App 节点的公网 IP（GOST 通过公网转发）
	appIPs := make([]string, 0, len(appNodes))
	for _, n := range appNodes {
		var server entity.Servers
		has, err := dbs.DBAdmin.Where("id = ?", n.ServerId).Cols("host").Get(&server)
		if err != nil {
			logx.Errorf("[SyncClusterGost] 查询 App 节点 serverId=%d 的服务器信息失败: %v", n.ServerId, err)
			continue
		}
		if has && server.Host != "" {
			appIPs = append(appIPs, server.Host)
		} else {
			logx.Infof("[SyncClusterGost] App 节点 serverId=%d 无公网 IP，跳过", n.ServerId)
		}
	}
	if len(appIPs) == 0 {
		return nil, fmt.Errorf("无法获取 App 节点公网 IP")
	}

	targetIPsStr := strings.Join(appIPs, ",")
	logx.Infof("[SyncClusterGost] 商户 %d 端口 %d 共 %d 个 App 节点: %s", merchantId, merchantPort, len(appIPs), targetIPsStr)

	// 3. 查商户的 GOST 服务器（优先从 merchant_gost_servers 关联表，回退到 servers 表的系统服务器）
	type gostServerRef struct {
		ServerId   int
		ListenPort int // 0 = 使用商户默认端口
	}
	var gostRefs []gostServerRef

	var mgs []entity.MerchantGostServers
	err = dbs.DBAdmin.Where("merchant_id = ? AND status = 1", merchantId).
		OrderBy("priority, id").Find(&mgs)
	if err != nil {
		return nil, fmt.Errorf("查询 GOST 服务器失败: %v", err)
	}
	for _, gs := range mgs {
		gostRefs = append(gostRefs, gostServerRef{ServerId: gs.ServerId, ListenPort: gs.ListenPort})
	}

	// 回退：如果 merchant_gost_servers 无记录，从 servers 表查 server_type=2 的系统服务器
	if len(gostRefs) == 0 {
		var sysServers []entity.Servers
		dbs.DBAdmin.Where("merchant_id = ? AND server_type = 2 AND status = 1", merchantId).
			Cols("id").Find(&sysServers)
		for _, s := range sysServers {
			gostRefs = append(gostRefs, gostServerRef{ServerId: s.Id})
		}
		if len(gostRefs) > 0 {
			logx.Infof("[SyncClusterGost] 从 servers 表找到 %d 个系统服务器", len(gostRefs))
		}
	}

	if len(gostRefs) == 0 {
		return nil, fmt.Errorf("商户 %d 没有关联的 GOST 服务器", merchantId)
	}

	// 4. 每个中转服务器都配置所有 App 节点（多目标负载均衡）
	results := make([]model.GostSyncResult, 0, len(gostRefs))

	for _, gs := range gostRefs {
		// 获取 GOST 服务器信息
		var server entity.Servers
		has, err := dbs.DBAdmin.Where("id = ?", gs.ServerId).Get(&server)
		if err != nil {
			results = append(results, model.GostSyncResult{
				ServerId:   gs.ServerId,
				ServerName: "未知",
				TargetIP:   targetIPsStr,
				Success:    false,
				Error:      fmt.Sprintf("查询服务器信息失败: %v", err),
			})
			continue
		}
		if !has {
			results = append(results, model.GostSyncResult{
				ServerId:   gs.ServerId,
				ServerName: "未知",
				TargetIP:   targetIPsStr,
				Success:    false,
				Error:      "GOST 服务器不存在",
			})
			continue
		}

		// 确定监听基础端口：优先用关联表的 ListenPort，否则用商户端口
		basePort := gs.ListenPort
		if basePort == 0 {
			basePort = merchantPort
		}

		// 构建端口转发详情
		ports := make([]model.GostPortForward, 0, len(gostapi.MerchantPortConfigs))
		for _, cfg := range gostapi.MerchantPortConfigs {
			ports = append(ports, model.GostPortForward{
				Name:       cfg.Name,
				ListenPort: basePort + cfg.Offset,
				TargetPort: cfg.TargetPort,
			})
		}

		// 根据转发类型配置 GOST（多目标）
		tlsListener := server.TlsEnabled == 1
		var forwardErr error
		forwardType := "encrypted"
		if server.ForwardType == entity.ForwardTypeEncrypted {
			forwardErr = syncClusterEncryptedForward(server.Host, basePort, appIPs, tlsListener)
		} else {
			forwardType = "direct"
			forwardErr = syncClusterDirectForward(server.Host, basePort, appIPs)
		}

		r := model.GostSyncResult{
			ServerId:    gs.ServerId,
			ServerName:  server.Name,
			ServerHost:  server.Host,
			TargetIP:    targetIPsStr,
			ForwardType: forwardType,
			Ports:       ports,
			Success:     forwardErr == nil,
		}
		if forwardErr != nil {
			r.Error = forwardErr.Error()
			logx.Errorf("[SyncClusterGost] GOST %s:%d -> [%s] 失败: %v", server.Host, basePort, targetIPsStr, forwardErr)
		} else {
			logx.Infof("[SyncClusterGost] GOST %s:%d -> [%s] 成功 (%d 个 App 节点)", server.Host, basePort, targetIPsStr, len(appIPs))
		}

		results = append(results, r)
	}

	// 同步持久化配置到磁盘（并发执行，但等待全部完成后再返回）
	var wg sync.WaitGroup
	var mu sync.Mutex
	for i, r := range results {
		if r.Success {
			wg.Add(1)
			go func(idx int, sid int, host string) {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				defer cancel()

				done := make(chan error, 1)
				go func() { done <- PersistGostConfig(sid) }()

				select {
				case persistErr := <-done:
					if persistErr != nil {
						mu.Lock()
						results[idx].PersistError = fmt.Sprintf("持久化失败: %v", persistErr)
						mu.Unlock()
						logx.Errorf("[SyncClusterGost] 持久化 GOST 配置失败 %s(id=%d): %v", host, sid, persistErr)
					} else {
						logx.Infof("[SyncClusterGost] 已持久化 GOST 配置 %s(id=%d)", host, sid)
					}
				case <-ctx.Done():
					mu.Lock()
					results[idx].PersistError = "持久化超时(30s)"
					mu.Unlock()
					logx.Errorf("[SyncClusterGost] 持久化 GOST 配置超时 %s(id=%d)", host, sid)
				}
			}(i, r.ServerId, r.ServerHost)
		}
	}
	wg.Wait()

	return results, nil
}

// syncClusterEncryptedForward 同步加密转发 (relay+tls)
// 监听 basePort+0/1/2/3 → relay+tls → appNode:10010/10011/10012/10013
// tlsListener: 是否在监听端启用 TLS（供 App 直接 TLS/HTTPS 连接）
func syncClusterEncryptedForward(gostServerIP string, basePort int, targetIPs []string, tlsListener bool) error {
	// 清除旧规则：兼容多种命名格式
	// 1. 单机 Control 创建的协议命名规则 (tcp-relay-10000, ws-relay-10001, http-relay-10002)
	_ = gostapi.DeleteMerchantForwards(gostServerIP, basePort)
	// 2. 旧版集群同步用 ForwardPorts 创建的规则 (tcp-relay-10010/10011/10012)
	_ = gostapi.DeleteMerchantForwards(gostServerIP, gostapi.ForwardPorts[0])
	// 3. 集群同步创建的统一命名规则 (tcp-relay-{port})
	for _, cfg := range gostapi.MerchantPortConfigs {
		_ = gostapi.DeleteRelayTLSForward(gostServerIP, basePort+cfg.Offset)
		_ = gostapi.DeleteRelayTLSForward(gostServerIP, gostapi.ForwardPorts[0]+cfg.Offset)
	}
	// 4. 直连转发规则 (fwd-{port})（转发类型可能从直连切换到加密）
	clearPorts := make([]int, 0, len(gostapi.MerchantPortConfigs)*2)
	for _, cfg := range gostapi.MerchantPortConfigs {
		clearPorts = append(clearPorts, basePort+cfg.Offset)
		clearPorts = append(clearPorts, gostapi.ForwardPorts[0]+cfg.Offset)
	}
	_ = gostapi.ClearDirectForwardTargetWithPorts(gostServerIP, clearPorts)

	// 创建新规则：监听 basePort+offset → relay+tls → appNode:targetPort
	var createdPorts []int
	for _, cfg := range gostapi.MerchantPortConfigs {
		listenPort := basePort + cfg.Offset
		targetPort := cfg.TargetPort
		var err error
		if tlsListener {
			_, err = gostapi.CreateRelayTLSForwardMultiTargetWithTlsListener(gostServerIP, listenPort, targetIPs, targetPort)
		} else {
			_, err = gostapi.CreateRelayTLSForwardMultiTarget(gostServerIP, listenPort, targetIPs, targetPort)
		}
		if err != nil {
			// 回滚已创建的规则
			for _, p := range createdPorts {
				_ = gostapi.DeleteRelayTLSForward(gostServerIP, p)
			}
			return fmt.Errorf("创建端口 %d→%d relay+tls 转发失败: %w", listenPort, targetPort, err)
		}
		createdPorts = append(createdPorts, listenPort)
	}

	return nil
}

// syncClusterDirectForward 同步直连转发 (TCP)
// 监听 basePort+0/1/2/3 → TCP → appNode:10010/10011/10012/10013
func syncClusterDirectForward(gostServerIP string, basePort int, targetIPs []string) error {
	// 清除旧规则：兼容多种命名格式
	// 1. 单机 Control 创建的协议命名规则
	_ = gostapi.DeleteMerchantForwards(gostServerIP, basePort)
	_ = gostapi.DeleteMerchantForwards(gostServerIP, gostapi.ForwardPorts[0])
	// 2. 直连转发规则 (fwd-{port})
	clearPorts := make([]int, 0, len(gostapi.MerchantPortConfigs)*2)
	for _, cfg := range gostapi.MerchantPortConfigs {
		clearPorts = append(clearPorts, basePort+cfg.Offset)
		clearPorts = append(clearPorts, gostapi.ForwardPorts[0]+cfg.Offset)
	}
	_ = gostapi.ClearDirectForwardTargetWithPorts(gostServerIP, clearPorts)
	// 3. 集群 relay+tls 规则（转发类型可能被修改过）
	for _, cfg := range gostapi.MerchantPortConfigs {
		_ = gostapi.DeleteRelayTLSForward(gostServerIP, basePort+cfg.Offset)
	}

	// 创建新规则：监听 basePort+offset → TCP → appNode:targetPort
	var createdPorts []int
	for _, cfg := range gostapi.MerchantPortConfigs {
		listenPort := basePort + cfg.Offset
		targetPort := cfg.TargetPort
		_, err := gostapi.CreateDirectForwardMultiTarget(gostServerIP, listenPort, targetIPs, targetPort)
		if err != nil {
			// 回滚已创建的规则
			_ = gostapi.ClearDirectForwardTargetWithPorts(gostServerIP, createdPorts)
			return fmt.Errorf("创建端口 %d→%d TCP 直连转发失败: %w", listenPort, targetPort, err)
		}
		createdPorts = append(createdPorts, listenPort)
	}

	return nil
}
