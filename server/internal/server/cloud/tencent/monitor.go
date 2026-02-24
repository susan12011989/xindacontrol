package tencent

import (
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	monitor "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/monitor/v20180724"
)

// NewMonitorClient 创建腾讯云云监控客户端
func NewMonitorClient(accessKey, accessSecret, regionId string) (*monitor.Client, error) {
	credential := common.NewCredential(accessKey, accessSecret)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "monitor.tencentcloudapi.com"
	return monitor.NewClient(credential, regionId, cpf)
}
