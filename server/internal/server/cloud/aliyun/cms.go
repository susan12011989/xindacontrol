package aliyun

import (
	"fmt"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	cms20190101 "github.com/alibabacloud-go/cms-20190101/v9/client"
	"github.com/alibabacloud-go/tea/tea"
)

// NewCmsClient 创建阿里云云监控 (CMS) 客户端
func NewCmsClient(accessKey, accessSecret, regionId string) (*cms20190101.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(accessKey),
		AccessKeySecret: tea.String(accessSecret),
	}
	config.Endpoint = tea.String(fmt.Sprintf("metrics.%s.aliyuncs.com", regionId))
	return cms20190101.NewClient(config)
}
