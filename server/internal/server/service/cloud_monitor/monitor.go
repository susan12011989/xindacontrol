package cloud_monitor

import (
	"fmt"

	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
)

// GetCloudMonitorMetrics 统一云监控入口，根据服务器的云类型分发到对应云平台
func GetCloudMonitorMetrics(req model.CloudMonitorMetricsReq) (*model.CloudMonitorMetricsResp, error) {
	// 1. 查询服务器
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", req.ServerId).Get(&server)
	if err != nil {
		return nil, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("服务器不存在")
	}

	// 2. 确定云类型（优先新字段，兼容旧 AWS 字段）
	cloudType := server.CloudType
	instanceId := server.CloudInstanceId
	regionId := server.CloudRegionId

	if cloudType == "" && server.AwsInstanceId != "" {
		cloudType = "aws"
		instanceId = server.AwsInstanceId
		regionId = server.AwsRegionId
	}

	if cloudType == "" || instanceId == "" {
		return nil, fmt.Errorf("该服务器未配置云实例信息")
	}
	if regionId == "" {
		return nil, fmt.Errorf("该服务器未配置云区域")
	}

	// 3. 按云类型分发
	switch cloudType {
	case "aws":
		return getAwsMetrics(server, req.Period)
	case "aliyun":
		return getAliyunMetrics(server, instanceId, regionId, req.Period)
	case "tencent":
		return getTencentMetrics(server, instanceId, regionId, req.Period)
	default:
		return nil, fmt.Errorf("不支持的云类型: %s", cloudType)
	}
}
