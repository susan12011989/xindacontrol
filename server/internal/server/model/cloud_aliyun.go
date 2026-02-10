package model

import (
	"server/internal/server/cloud/aliyun"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v6/client"
)

type ListReq struct {
	MerchantId     int      `form:"merchant_id"`
	CloudAccountId int64    `form:"cloud_account_id"`
	RegionId       []string `form:"region_id[]" binding:"required,dive"`
}

// 弹性IP操作请求模型
type OperateEipReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	AllocationId   string `json:"allocation_id" binding:"required"`
	Operation      string `json:"operation" binding:"required,oneof=modify associate unassociate delete"` // 操作类型：modify, associate, unassociate, delete
	InstanceId     string `json:"instance_id,omitempty"`                                                  // 关联实例时使用
	InstanceType   string `json:"instance_type,omitempty"`                                                // 关联实例时使用：EcsInstance, NetworkInterface
	Name           string `json:"name,omitempty"`                                                         // 修改EIP属性时使用
	Bandwidth      string `json:"bandwidth,omitempty"`                                                    // 修改EIP带宽
}

// 共享带宽操作请求模型
type OperateBandwidthPackageReq struct {
	MerchantId         int      `json:"merchant_id"`
	CloudAccountId     int64    `json:"cloud_account_id"`
	RegionId           string   `json:"region_id" binding:"required"`
	BandwidthPackageId string   `json:"bandwidth_package_id" binding:"required"`
	Operation          string   `json:"operation" binding:"required,oneof=modify spec addEip removeEip delete"` // 操作类型：modify, spec, addEip, removeEip, delete
	Name               string   `json:"name,omitempty"`                                                         // 修改属性用
	Description        string   `json:"description,omitempty"`                                                  // 修改属性用
	Bandwidth          string   `json:"bandwidth,omitempty"`                                                    // 修改规格用
	IpInstanceIds      []string `json:"ip_instance_ids,omitempty"`                                              // 添加EIP用
	IpInstanceId       string   `json:"ip_instance_id,omitempty"`                                               // 移除EIP用
}

type OperateEcsInstanceReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	InstanceId     string `json:"instance_id" binding:"required"`
	Operation      string `json:"operation" binding:"required,oneof=start stop reboot delete restart"`
}

type CreateInstancesReq struct {
	List []*aliyun.CreateInstanceRequest `json:"list" binding:"required"`
}
type NetworkInterfaceData struct {
	RegionId         string                                                                                    `json:"RegionId"`
	NetworkInterface *ecs20140526.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSet `json:"NetworkInterface"`
}

type SecurityGroupData struct {
	RegionId      string                                                                     `json:"RegionId"`
	SecurityGroup *ecs20140526.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup `json:"SecurityGroup"`
}

type DescribeSecurityGroupAttributeReq struct {
	MerchantId      int    `form:"merchant_id"`
	CloudAccountId  int64  `form:"cloud_account_id"`
	RegionId        string `form:"region_id" binding:"required"`
	SecurityGroupId string `form:"security_group_id" binding:"required"`
}

type CreateSecurityGroupReq struct {
	List []*aliyun.CreateSecurityGroupRequest `json:"list" binding:"required"`
}

type CreateEipReq struct {
	List []*aliyun.AllocateEipAddressRequest `json:"list" binding:"required"`
}

// 批量创建辅助网卡的请求模型
type CreateSecondaryNetworkInterfaceReq struct {
	MerchantId     int                  `json:"merchant_id"`
	CloudAccountId int64                `json:"cloud_account_id"`
	Instances      []InstanceRegionPair `json:"instances" binding:"required,dive"`
}

// 实例和区域的组合
type InstanceRegionPair struct {
	InstanceId string `json:"instance_id" binding:"required"`
	RegionId   string `json:"region_id" binding:"required"`
}

// 批量绑定弹性IP的请求模型
type BatchAssociateEipReq struct {
	MerchantId     int             `json:"merchant_id"`
	CloudAccountId int64           `json:"cloud_account_id"`
	EipList        []EipBindConfig `json:"eip_list" binding:"required,dive"`
}

// 弹性IP绑定配置
type EipBindConfig struct {
	RegionId     string `json:"region_id" binding:"required"`     // 区域ID
	AllocationId string `json:"allocation_id" binding:"required"` // EIP的实例ID
}

// 更换弹性IP请求模型
type ReplaceEipReq struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	RegionId           string `json:"region_id" binding:"required"`
	AllocationId       string `json:"allocation_id" binding:"required"`        // 要更换的EIP实例ID
	OldIpAddress       string `json:"old_ip_address,omitempty"`                // 旧EIP的IP地址（可选，传入可跳过查询）
	InstanceId         string `json:"instance_id,omitempty"`                   // 绑定的实例ID（网卡ID）
	InstanceType       string `json:"instance_type,omitempty"`                 // 实例类型：EcsInstance, NetworkInterface
	BandwidthPackageId string `json:"bandwidth_package_id,omitempty"`          // 所属共享带宽ID
	Bandwidth          string `json:"bandwidth,omitempty"`                     // 新EIP带宽（默认继承原带宽）
	InternetChargeType string `json:"internet_charge_type,omitempty"`          // 计费方式：PayByBandwidth, PayByTraffic
	ISP                string `json:"isp,omitempty"`                           // 线路类型：BGP, BGP_PRO
	Name               string `json:"name,omitempty"`                          // 新EIP名称（默认继承原名称）
}

// 批量更换弹性IP请求模型
type BatchReplaceEipReq struct {
	MerchantId     int                      `json:"merchant_id"`
	CloudAccountId int64                    `json:"cloud_account_id"`
	EipList        []BatchReplaceEipConfig `json:"eip_list" binding:"required,dive"`
}

// 批量更换EIP的配置项
type BatchReplaceEipConfig struct {
	RegionId           string `json:"region_id" binding:"required"`           // 区域ID
	AllocationId       string `json:"allocation_id" binding:"required"`       // 要更换的EIP实例ID
	OldIpAddress       string `json:"old_ip_address,omitempty"`               // 旧EIP的IP地址（可选，传入可跳过查询）
	InstanceId         string `json:"instance_id,omitempty"`                  // 绑定的实例ID（网卡ID）
	InstanceType       string `json:"instance_type,omitempty"`                // 实例类型：EcsInstance, NetworkInterface
	BandwidthPackageId string `json:"bandwidth_package_id,omitempty"`         // 所属共享带宽ID
	Bandwidth          string `json:"bandwidth,omitempty"`                    // 新EIP带宽（默认继承原带宽）
	InternetChargeType string `json:"internet_charge_type,omitempty"`         // 计费方式：PayByBandwidth, PayByTraffic
}

// ========== OSS ==========

// OssListObjectsReq 列举 OSS 对象请求
type OssListObjectsReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	Bucket         string `form:"bucket" binding:"required"`
	Endpoint       string `form:"endpoint"` // 已废弃，保留以兼容旧版本
	Prefix         string `form:"prefix"`
	Marker         string `form:"marker"`
	MaxKeys        int    `form:"max_keys"`
}

// OssObjectItem OSS 对象条目
type OssObjectItem struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
	StorageClass string `json:"storage_class"`
}

// OssListObjectsResponse 列表响应
type OssListObjectsResponse struct {
	List        []OssObjectItem `json:"list"`
	IsTruncated bool            `json:"is_truncated"`
	NextMarker  string          `json:"next_marker"`
	Total       int             `json:"total"`
}

// OssUploadForm 上传表单（multipart）
type OssUploadForm struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	Bucket         string `form:"bucket" binding:"required"`
	ObjectKey      string `form:"object_key" binding:"required"`
	Endpoint       string `form:"endpoint"`
}

// OssDownloadReq 下载请求
type OssDownloadReq struct {
	MerchantId     int    `form:"merchant_id" json:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id" json:"cloud_account_id"`
	RegionId       string `form:"region_id" json:"region_id" binding:"required"`
	Bucket         string `form:"bucket" json:"bucket" binding:"required"`
	ObjectKey      string `form:"object_key" json:"object_key" binding:"required"`
	Filename       string `form:"filename" json:"filename"`
	Attachment     int    `form:"attachment" json:"attachment"`
	Endpoint       string `form:"endpoint" json:"endpoint"` // 已废弃，保留以兼容旧版本
}

// 列举 Buckets 请求
type OssListBucketsReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id"` // 可选；为空则使用公共域名
	Prefix         string `form:"prefix"`
	Marker         string `form:"marker"`
	MaxKeys        int    `form:"max_keys"`
}

// Bucket 条目
type OssBucketItem struct {
	Name         string `json:"name"`
	Location     string `json:"location"`
	CreationDate string `json:"creation_date"`
	StorageClass string `json:"storage_class"` // 存储类型：Standard、IA、Archive等
}

// 列举 Buckets 响应
type OssListBucketsResponse struct {
	List        []OssBucketItem `json:"list"`
	IsTruncated bool            `json:"is_truncated"`
	NextMarker  string          `json:"next_marker"`
	Total       int             `json:"total"`
}

// OSS 创建 Bucket 请求
type OssCreateBucketReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
	StorageClass   string `json:"storage_class"` // Standard, IA, Archive 等，默认 Standard
}

// OSS 删除 Bucket 请求
type OssDeleteBucketReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
}

// OSS 设置 Bucket 公开访问请求
type OssSetBucketPublicReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
	Public         bool   `json:"public"` // true=公开读，false=私有
}

// OSS 删除对象请求
type OssDeleteObjectReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
	ObjectKey      string `json:"object_key" binding:"required"`
}
