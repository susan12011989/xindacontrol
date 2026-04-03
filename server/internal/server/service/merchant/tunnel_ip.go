package merchant

import (
	"server/internal/dbhelper"
)

// AllocateTunnelIPForMerchant 为商户自动分配 TunnelIP（委托给 dbhelper 避免循环引用）
func AllocateTunnelIPForMerchant(merchantId int, serverId int) (string, error) {
	return dbhelper.AllocateTunnelIPForMerchant(merchantId, serverId)
}
