package model

// ========== 腾讯云 COS ==========

// CosListBucketsReq 列举 Bucket 请求
type CosListBucketsReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id"` // 可选；为空则列出所有区域
}

// CosBucketItem Bucket 条目
type CosBucketItem struct {
	Name         string `json:"name"`
	Location     string `json:"location"`
	CreationDate string `json:"creation_date"`
}

// CosListBucketsResponse 列举 Bucket 响应
type CosListBucketsResponse struct {
	List  []CosBucketItem `json:"list"`
	Total int             `json:"total"`
}

// CosListObjectsReq 列举对象请求
type CosListObjectsReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	Bucket         string `form:"bucket" binding:"required"`
	Prefix         string `form:"prefix"`
	Marker         string `form:"marker"`
	MaxKeys        int    `form:"max_keys"`
}

// CosObjectItem 对象条目
type CosObjectItem struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
	StorageClass string `json:"storage_class"`
}

// CosListObjectsResponse 列举对象响应
type CosListObjectsResponse struct {
	List        []CosObjectItem `json:"list"`
	IsTruncated bool            `json:"is_truncated"`
	NextMarker  string          `json:"next_marker"`
	Total       int             `json:"total"`
}

// CosUploadForm 上传表单（multipart）
type CosUploadForm struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	Bucket         string `form:"bucket" binding:"required"`
	ObjectKey      string `form:"object_key" binding:"required"`
}

// CosDownloadReq 下载请求
type CosDownloadReq struct {
	MerchantId     int    `form:"merchant_id" json:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id" json:"cloud_account_id"`
	RegionId       string `form:"region_id" json:"region_id" binding:"required"`
	Bucket         string `form:"bucket" json:"bucket" binding:"required"`
	ObjectKey      string `form:"object_key" json:"object_key" binding:"required"`
	Filename       string `form:"filename" json:"filename"`
	Attachment     int    `form:"attachment" json:"attachment"`
}

// COS 创建 Bucket 请求
type CosCreateBucketReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"` // 格式：bucketname-appid
}

// COS 删除 Bucket 请求
type CosDeleteBucketReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
}

// COS 设置 Bucket 公开访问请求
type CosSetBucketPublicReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
	Public         bool   `json:"public"` // true=公开读，false=私有
}

// COS 删除对象请求
type CosDeleteObjectReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
	ObjectKey      string `json:"object_key" binding:"required"`
}

// ========== CVM 实例管理 ==========

// TencentListReq 通用列表请求（支持多 region 并发查询）
type TencentListReq struct {
	MerchantId     int      `form:"merchant_id"`
	CloudAccountId int64    `form:"cloud_account_id"`
	RegionId       []string `form:"region_id[]" binding:"required,dive"`
}

// TencentOperateInstanceReq 单实例操作请求
type TencentOperateInstanceReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	InstanceId     string `json:"instance_id" binding:"required"`
	Operation      string `json:"operation" binding:"required,oneof=start stop restart delete"`
}

// TencentBatchOperateInstanceReq 批量实例操作请求
type TencentBatchOperateInstanceReq struct {
	MerchantId     int      `json:"merchant_id"`
	CloudAccountId int64    `json:"cloud_account_id"`
	RegionId       string   `json:"region_id" binding:"required"`
	InstanceIds    []string `json:"instance_ids" binding:"required"`
	Operation      string   `json:"operation" binding:"required,oneof=start stop restart delete"`
}

// TencentModifyInstanceReq 修改实例属性请求
type TencentModifyInstanceReq struct {
	MerchantId       int      `json:"merchant_id"`
	CloudAccountId   int64    `json:"cloud_account_id"`
	RegionId         string   `json:"region_id" binding:"required"`
	InstanceId       string   `json:"instance_id" binding:"required"`
	InstanceName     string   `json:"instance_name"`
	SecurityGroupIds []string `json:"security_group_ids"`
}

// TencentResetPasswordReq 重置密码请求
type TencentResetPasswordReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	InstanceId     string `json:"instance_id" binding:"required"`
	Password       string `json:"password" binding:"required"`
}

// ========== 安全组管理 ==========

// TencentListSecurityGroupsReq 安全组列表请求
type TencentListSecurityGroupsReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
}

// TencentDescribeSecurityGroupReq 安全组详情请求
type TencentDescribeSecurityGroupReq struct {
	MerchantId      int    `form:"merchant_id"`
	CloudAccountId  int64  `form:"cloud_account_id"`
	RegionId        string `form:"region_id" binding:"required"`
	SecurityGroupId string `form:"security_group_id" binding:"required"`
}

// TencentSecurityGroupPolicyReq 添加安全组规则请求
type TencentSecurityGroupPolicyReq struct {
	MerchantId      int                        `json:"merchant_id"`
	CloudAccountId  int64                      `json:"cloud_account_id"`
	RegionId        string                     `json:"region_id" binding:"required"`
	SecurityGroupId string                     `json:"security_group_id" binding:"required"`
	Policies        []TencentSecurityGroupRule `json:"policies" binding:"required"`
}

// TencentSecurityGroupRule 安全组规则
type TencentSecurityGroupRule struct {
	Protocol    string `json:"protocol"`    // TCP, UDP, ICMP, ALL
	Port        string `json:"port"`        // 如 "80", "8000-9000"
	CidrBlock   string `json:"cidr_block"`  // 如 "0.0.0.0/0"
	Action      string `json:"action"`      // ACCEPT, DROP
	Description string `json:"description"`
}

// TencentDeleteSecurityGroupPoliciesReq 删除安全组规则请求
type TencentDeleteSecurityGroupPoliciesReq struct {
	MerchantId      int     `json:"merchant_id"`
	CloudAccountId  int64   `json:"cloud_account_id"`
	RegionId        string  `json:"region_id" binding:"required"`
	SecurityGroupId string  `json:"security_group_id" binding:"required"`
	PolicyIndexes   []int64 `json:"policy_indexes" binding:"required"`
}

// ========== 创建实例 ==========

// TencentCreateInstanceReq 创建单个实例请求
type TencentCreateInstanceReq struct {
	MerchantId              int      `json:"merchant_id"`
	CloudAccountId          int64    `json:"cloud_account_id"`
	RegionId                string   `json:"region_id" binding:"required"`
	Zone                    string   `json:"zone" binding:"required"`
	ImageId                 string   `json:"image_id" binding:"required"`
	InstanceType            string   `json:"instance_type" binding:"required"`
	InstanceChargeType      string   `json:"instance_charge_type"`
	SystemDiskType          string   `json:"system_disk_type"`
	SystemDiskSize          int64    `json:"system_disk_size"`
	VpcId                   string   `json:"vpc_id"`
	SubnetId                string   `json:"subnet_id"`
	SecurityGroupIds        []string `json:"security_group_ids"`
	InstanceName            string   `json:"instance_name"`
	Password                string   `json:"password"`
	Period                  int64    `json:"period"`
	RenewFlag               string   `json:"renew_flag"`
	InternetMaxBandwidthOut int64    `json:"internet_max_bandwidth_out"`
}

// TencentCreateInstancesReq 批量创建实例请求
type TencentCreateInstancesReq struct {
	List []TencentCreateInstanceReq `json:"list" binding:"required"`
}

// ========== 创建安全组 ==========

// TencentCreateSecurityGroupReq 创建单个安全组请求
type TencentCreateSecurityGroupReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Name           string `json:"name"`
	Description    string `json:"description"`
}

// TencentCreateSecurityGroupsReq 批量创建安全组请求
type TencentCreateSecurityGroupsReq struct {
	List []TencentCreateSecurityGroupReq `json:"list" binding:"required"`
}

// ========== VPC/子网查询 ==========

// TencentVpcReq VPC/子网查询请求
type TencentVpcReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	VpcId          string `form:"vpc_id"`
}
