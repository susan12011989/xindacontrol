package merchant

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"server/internal/dbhelper"
	"server/internal/server/cloud/tencent"
	cloud_aliyun "server/internal/server/service/cloud_aliyun"
	cloud_aws "server/internal/server/service/cloud_aws"
	"server/internal/server/service/utils"
	"server/pkg/dbs"
	"server/pkg/entity"

	"github.com/zeromicro/go-zero/core/logx"
)

// IP嵌入使用的seed（与现有代码一致）
const ipEmbedSeed uint32 = 444013

// SyncGostIPReq 同步 GOST IP 请求
type SyncGostIPReq struct {
	MerchantId  int      `json:"merchant_id" binding:"required"`
	IPs         []string `json:"ips"`          // 指定 IP 列表，留空则自动从 GOST 服务器获取
	FileNames   []string `json:"file_names"`   // 要处理的源文件名，留空则处理全部
	ObjectKey   string   `json:"object_key"`   // 自定义 OSS 对象名，留空使用默认
	EmbedToFile bool     `json:"embed_to_file"` // 是否嵌入到文件（如 icon），否则上传纯文本
}

// SyncGostIPResp 同步结果响应
type SyncGostIPResp struct {
	MerchantId   int                `json:"merchant_id"`
	MerchantName string             `json:"merchant_name"`
	IPs          []string           `json:"ips"`
	Results      []SyncResultItem   `json:"results"`
	Summary      SyncSummary        `json:"summary"`
}

// SyncResultItem 单个同步结果
type SyncResultItem struct {
	OssConfigId   int    `json:"oss_config_id"`
	OssConfigName string `json:"oss_config_name"`
	CloudType     string `json:"cloud_type"`
	Bucket        string `json:"bucket"`
	ObjectKey     string `json:"object_key"`
	ObjectUrl     string `json:"object_url"`
	Success       bool   `json:"success"`
	Error         string `json:"error,omitempty"`
}

// SyncSummary 同步摘要
type SyncSummary struct {
	TotalOss     int    `json:"total_oss"`
	SuccessCount int    `json:"success_count"`
	FailCount    int    `json:"fail_count"`
	Duration     string `json:"duration"`
}

// SyncMerchantGostIP 同步商户的 GOST IP 到所有 OSS
func SyncMerchantGostIP(req SyncGostIPReq) (*SyncGostIPResp, error) {
	startTime := time.Now()

	// 1. 获取商户信息
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.ID(req.MerchantId).Get(&merchant)
	if err != nil {
		return nil, fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("商户不存在: %d", req.MerchantId)
	}

	// 2. 获取 IP 列表
	ips := req.IPs
	if len(ips) == 0 {
		// 自动从 GOST 服务器获取
		ips, err = getMerchantGostIPs(req.MerchantId)
		if err != nil {
			return nil, fmt.Errorf("获取 GOST IP 失败: %v", err)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("没有可用的 GOST IP")
	}

	// 3. 获取商户的所有 OSS 配置
	ossConfigs, err := ListMerchantOssConfigs(req.MerchantId)
	if err != nil {
		return nil, fmt.Errorf("获取 OSS 配置失败: %v", err)
	}
	if len(ossConfigs) == 0 {
		return nil, fmt.Errorf("商户未配置 OSS")
	}

	// 4. 准备上传内容
	var uploadData []byte
	objectKey := req.ObjectKey
	if objectKey == "" {
		objectKey = "ip.txt" // 默认文件名
	}

	if req.EmbedToFile {
		// 嵌入到文件模式（需要源文件）
		// 这里简化处理，实际可以调用 ip_embed 服务
		uploadData = []byte(strings.Join(ips, "\n"))
	} else {
		// 纯文本 IP 列表
		uploadData = []byte(strings.Join(ips, "\n"))
	}

	// 5. 上传到各个 OSS
	results := make([]SyncResultItem, 0, len(ossConfigs))
	successCount, failCount := 0, 0

	for _, ossConfig := range ossConfigs {
		result := SyncResultItem{
			OssConfigId:   ossConfig.Id,
			OssConfigName: ossConfig.Name,
			CloudType:     ossConfig.CloudType,
			Bucket:        ossConfig.Bucket,
			ObjectKey:     objectKey,
		}

		objectUrl, uploadErr := uploadToMerchantOss(ossConfig, objectKey, uploadData)
		if uploadErr != nil {
			result.Success = false
			result.Error = uploadErr.Error()
			failCount++
			logx.Errorf("同步商户%d OSS[%s]失败: %v", req.MerchantId, ossConfig.Name, uploadErr)
		} else {
			result.Success = true
			result.ObjectUrl = objectUrl
			successCount++
			logx.Infof("同步商户%d OSS[%s]成功: %s", req.MerchantId, ossConfig.Name, objectUrl)
		}
		results = append(results, result)
	}

	duration := time.Since(startTime)

	return &SyncGostIPResp{
		MerchantId:   req.MerchantId,
		MerchantName: merchant.Name,
		IPs:          ips,
		Results:      results,
		Summary: SyncSummary{
			TotalOss:     len(ossConfigs),
			SuccessCount: successCount,
			FailCount:    failCount,
			Duration:     duration.String(),
		},
	}, nil
}

// getMerchantGostIPs 获取商户的 GOST 服务器 IP 列表
func getMerchantGostIPs(merchantId int) ([]string, error) {
	relations, servers, err := GetMerchantAllGostServers(merchantId)
	if err != nil {
		return nil, err
	}

	if len(relations) == 0 || len(servers) == 0 {
		return nil, nil
	}

	// 构建服务器 ID 到服务器的映射
	serverMap := make(map[int]entity.Servers)
	for _, s := range servers {
		serverMap[s.Id] = s
	}

	// 按优先级收集 IP
	ips := make([]string, 0)
	for _, r := range relations {
		if server, ok := serverMap[r.ServerId]; ok {
			if server.Host != "" {
				ips = append(ips, server.Host)
			}
			// 如果有辅助 IP，也加入
			if server.AuxiliaryIP != "" {
				ips = append(ips, server.AuxiliaryIP)
			}
		}
	}

	return ips, nil
}

// uploadToMerchantOss 上传到商户的 OSS
func uploadToMerchantOss(ossConfig MerchantOssConfigResp, objectKey string, data []byte) (string, error) {
	reader := bytes.NewReader(data)

	switch ossConfig.CloudType {
	case "aws":
		acc, err := dbhelper.GetCloudAccountByID(ossConfig.CloudAccountId)
		if err != nil {
			return "", fmt.Errorf("获取 AWS 账号失败: %v", err)
		}
		err = cloud_aws.UploadObject(acc, ossConfig.Region, ossConfig.Bucket, objectKey, reader)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", ossConfig.Bucket, ossConfig.Region, objectKey), nil

	case "aliyun":
		err := cloud_aliyun.UploadOssObject(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket, objectKey, reader)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://%s.oss-%s.aliyuncs.com/%s", ossConfig.Bucket, ossConfig.Region, objectKey), nil

	case "tencent":
		err := tencent.UploadObject(0, ossConfig.CloudAccountId, ossConfig.Region, ossConfig.Bucket, objectKey, reader)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", ossConfig.Bucket, ossConfig.Region, objectKey), nil

	default:
		return "", fmt.Errorf("不支持的云类型: %s", ossConfig.CloudType)
	}
}

// SyncMerchantGostIPWithEmbed 同步商户的 GOST IP 到所有 OSS（带文件嵌入）
func SyncMerchantGostIPWithEmbed(merchantId int, ips []string, sourceFilePath string) (*SyncGostIPResp, error) {
	startTime := time.Now()

	// 1. 获取商户信息
	var merchantInfo entity.Merchants
	has, err := dbs.DBAdmin.ID(merchantId).Get(&merchantInfo)
	if err != nil {
		return nil, fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("商户不存在: %d", merchantId)
	}

	// 2. 获取 IP 列表
	if len(ips) == 0 {
		ips, err = getMerchantGostIPs(merchantId)
		if err != nil {
			return nil, fmt.Errorf("获取 GOST IP 失败: %v", err)
		}
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("没有可用的 GOST IP")
	}

	// 3. 读取源文件并嵌入 IP
	srcBytes, err := os.ReadFile(sourceFilePath)
	if err != nil {
		return nil, fmt.Errorf("读取源文件失败: %v", err)
	}
	embeddedData, err := utils.EmbedIPsToBytes(srcBytes, ips, ipEmbedSeed)
	if err != nil {
		return nil, fmt.Errorf("嵌入 IP 失败: %v", err)
	}

	// 4. 获取商户的所有 OSS 配置
	ossConfigs, err := ListMerchantOssConfigs(merchantId)
	if err != nil {
		return nil, fmt.Errorf("获取 OSS 配置失败: %v", err)
	}
	if len(ossConfigs) == 0 {
		return nil, fmt.Errorf("商户未配置 OSS")
	}

	// 5. 上传到各个 OSS
	objectKey := "icon.png" // 默认使用 icon.png
	results := make([]SyncResultItem, 0, len(ossConfigs))
	successCount, failCount := 0, 0

	for _, ossConfig := range ossConfigs {
		resultItem := SyncResultItem{
			OssConfigId:   ossConfig.Id,
			OssConfigName: ossConfig.Name,
			CloudType:     ossConfig.CloudType,
			Bucket:        ossConfig.Bucket,
			ObjectKey:     objectKey,
		}

		objectUrl, uploadErr := uploadToMerchantOss(ossConfig, objectKey, embeddedData)
		if uploadErr != nil {
			resultItem.Success = false
			resultItem.Error = uploadErr.Error()
			failCount++
		} else {
			resultItem.Success = true
			resultItem.ObjectUrl = objectUrl
			successCount++
		}
		results = append(results, resultItem)
	}

	duration := time.Since(startTime)

	return &SyncGostIPResp{
		MerchantId:   merchantId,
		MerchantName: merchantInfo.Name,
		IPs:          ips,
		Results:      results,
		Summary: SyncSummary{
			TotalOss:     len(ossConfigs),
			SuccessCount: successCount,
			FailCount:    failCount,
			Duration:     duration.String(),
		},
	}, nil
}

// BatchSyncMerchantGostIP 批量同步多个商户的 GOST IP
func BatchSyncMerchantGostIP(merchantIds []int) ([]SyncGostIPResp, error) {
	results := make([]SyncGostIPResp, 0, len(merchantIds))

	for _, merchantId := range merchantIds {
		resp, err := SyncMerchantGostIP(SyncGostIPReq{
			MerchantId: merchantId,
		})
		if err != nil {
			logx.Errorf("同步商户%d失败: %v", merchantId, err)
			// 记录失败但继续处理其他商户
			results = append(results, SyncGostIPResp{
				MerchantId: merchantId,
				Summary: SyncSummary{
					FailCount: 1,
				},
			})
			continue
		}
		results = append(results, *resp)
	}

	return results, nil
}

// GetMerchantGostIPSyncStatus 获取商户 GOST IP 同步状态
func GetMerchantGostIPSyncStatus(merchantId int) (*MerchantGostSyncStatusResp, error) {
	// 获取商户信息
	var merchant entity.Merchants
	has, err := dbs.DBAdmin.ID(merchantId).Get(&merchant)
	if err != nil {
		return nil, fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("商户不存在: %d", merchantId)
	}

	// 获取 GOST 服务器
	gostServers, err := ListMerchantGostServers(merchantId)
	if err != nil {
		return nil, fmt.Errorf("获取 GOST 服务器失败: %v", err)
	}

	// 获取 OSS 配置
	ossConfigs, err := ListMerchantOssConfigs(merchantId)
	if err != nil {
		return nil, fmt.Errorf("获取 OSS 配置失败: %v", err)
	}

	// 收集 IP
	ips := make([]string, 0)
	for _, g := range gostServers {
		if g.ServerHost != "" {
			ips = append(ips, g.ServerHost)
		}
	}

	return &MerchantGostSyncStatusResp{
		MerchantId:   merchantId,
		MerchantName: merchant.Name,
		GostServers:  gostServers,
		OssConfigs:   ossConfigs,
		CurrentIPs:   ips,
	}, nil
}

// MerchantGostSyncStatusResp 商户 GOST 同步状态响应
type MerchantGostSyncStatusResp struct {
	MerchantId   int                      `json:"merchant_id"`
	MerchantName string                   `json:"merchant_name"`
	GostServers  []MerchantGostServerResp `json:"gost_servers"`
	OssConfigs   []MerchantOssConfigResp  `json:"oss_configs"`
	CurrentIPs   []string                 `json:"current_ips"`
}
