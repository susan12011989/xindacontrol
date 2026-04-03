package dbhelper

import (
	"fmt"
	"server/pkg/dbs"
	"server/pkg/entity"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
)

// AllocateTunnelIPForMerchant 为商户自动分配 TunnelIP
// 从系统服务器的可用 IP（Host + AuxiliaryIP）中选取未被其他商户使用的
func AllocateTunnelIPForMerchant(merchantId int, serverId int) (string, error) {
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.ID(merchantId).Get(&merchant)
	if err != nil || !has {
		return "", fmt.Errorf("商户不存在: %d", merchantId)
	}
	if merchant.TunnelIP != "" {
		return merchant.TunnelIP, nil
	}

	var server entity.Servers
	has, err = dbs.DBAdmin.ID(serverId).Get(&server)
	if err != nil || !has {
		return "", fmt.Errorf("服务器不存在: %d", serverId)
	}

	// 收集服务器所有 IP
	allIPs := []string{}
	if server.Host != "" {
		allIPs = append(allIPs, server.Host)
	}
	if server.AuxiliaryIP != "" {
		for _, ip := range strings.Split(server.AuxiliaryIP, ",") {
			ip = strings.TrimSpace(ip)
			if ip != "" {
				allIPs = append(allIPs, ip)
			}
		}
	}
	if len(allIPs) == 0 {
		return "", fmt.Errorf("服务器 %d 无可用 IP", serverId)
	}

	// 查询已使用的 TunnelIP
	var merchants []entity.Merchants
	_ = dbs.DBAdmin.Where("tunnel_ip != '' AND status = 1").Cols("id", "tunnel_ip").Find(&merchants)
	usedIPs := make(map[string]bool, len(merchants))
	for _, m := range merchants {
		usedIPs[m.TunnelIP] = true
	}

	// 选取第一个未使用的
	var allocatedIP string
	for _, ip := range allIPs {
		if !usedIPs[ip] {
			allocatedIP = ip
			break
		}
	}
	if allocatedIP == "" {
		return "", fmt.Errorf("服务器 %d 的所有 IP 已被分配（共 %d 个）", serverId, len(allIPs))
	}

	_, err = dbs.DBAdmin.ID(merchantId).Cols("tunnel_ip").Update(&entity.Merchants{TunnelIP: allocatedIP})
	if err != nil {
		return "", fmt.Errorf("更新商户 TunnelIP 失败: %v", err)
	}

	logx.Infof("为商户 %d 分配 TunnelIP: %s (来自服务器 %d)", merchantId, allocatedIP, serverId)
	return allocatedIP, nil
}
