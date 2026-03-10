package merchant

import (
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"

	"github.com/zeromicro/go-zero/core/logx"
)

// updateGostServicesOnSystemServers 当商户 IP 变更时，入队更新所有系统服务器上的商户转发服务
// listenPort 为商户端口（对外访问端口），forwardHost 为商户服务器 IP
// tunnelIP 为系统服务器上分配给此商户的隧道 IP（用于多商户端口隔离）
// targetBasePort 为商户服务器上的 GOST 基础监听端口，如果 <= 0 则使用默认值
// 加密模式默认目标端口 10010/10011/10012，直连模式默认目标端口 10000/10001/10002
func updateGostServicesOnSystemServers(merchantId int, listenPort int, forwardHost string, tunnelIP string, targetBasePort int) {
	if listenPort <= 0 {
		return
	}

	var sysServers []entity.Servers
	if err := dbs.DBAdmin.Where("server_type = ?", 2).Find(&sysServers); err != nil {
		logx.Errorf("list system servers err: %+v", err)
		return
	}
	if len(sysServers) == 0 {
		return
	}

	encryptedCount := 0
	directCount := 0
	for _, s := range sysServers {
		var err error
		tlsEnabled := s.TlsEnabled == 1
		if s.ForwardType == entity.ForwardTypeDirect {
			// 直连转发：直接转发到商户业务程序
			err = gostapi.EnqueueUpdateMerchantDirectForwards(s.Host, listenPort, forwardHost, tunnelIP)
			directCount++
		} else {
			// 加密转发（默认）
			if targetBasePort > 0 {
				err = gostapi.EnqueueUpdateMerchantForwardsWithTargetPort(s.Host, listenPort, forwardHost, targetBasePort, tlsEnabled, tunnelIP)
			} else {
				err = gostapi.EnqueueUpdateMerchantForwards(s.Host, listenPort, forwardHost, tlsEnabled, tunnelIP)
			}
			encryptedCount++
		}
		if err != nil {
			logx.Errorf("enqueue update merchant forwards task for server %d (%s, forward_type=%d) failed: %+v",
				s.Id, s.Host, s.ForwardType, err)
		}
	}
	logx.Infof("enqueued update merchant forwards tasks for %d servers (encrypted: %d, direct: %d), port %d, tunnelIP %s",
		len(sysServers), encryptedCount, directCount, listenPort, tunnelIP)
}
