package model

// AWS 通用列表请求
type AwsListReq struct {
	MerchantId     int      `form:"merchant_id"`
	CloudAccountId int64    `form:"cloud_account_id"`
	RegionId       []string `form:"region_id[]" binding:"required,dive"`
}

// EC2 实例操作
type AwsOperateEc2Req struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	InstanceId     string `json:"instance_id" binding:"required"`
	Operation      string `json:"operation" binding:"required,oneof=start stop reboot terminate"`
}

// 安全组
type AwsSecurityGroupItem struct {
	RegionId      string      `json:"RegionId"`
	SecurityGroup interface{} `json:"SecurityGroup"`
}

type AwsDescribeSecurityGroupReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	GroupId        string `form:"group_id" binding:"required"`
}

type AwsAuthorizeSecurityGroupReq struct {
	MerchantId     int      `json:"merchant_id"`
	CloudAccountId int64    `json:"cloud_account_id"`
	RegionId       string   `json:"region_id" binding:"required"`
	GroupId        string   `json:"group_id" binding:"required"`
	IpProtocol     string   `json:"ip_protocol" binding:"required"` // tcp/udp/icmp/-1
	FromPort       int32    `json:"from_port"`
	ToPort         int32    `json:"to_port"`
	CidrBlocks     []string `json:"cidr_blocks"`
}

// EIP
type AwsOperateEipReq struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	RegionId           string `json:"region_id" binding:"required"`
	AllocationId       string `json:"allocation_id" binding:"required"`
	Operation          string `json:"operation" binding:"required,oneof=associate disassociate release"`
	InstanceId         string `json:"instance_id"`
	NetworkInterfaceId string `json:"network_interface_id"`
	PrivateIpAddress   string `json:"private_ip_address"`
}

// EIP 分配请求
type AwsAllocateEipReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
}

// EC2 实例详情
type AwsDescribeInstanceReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	InstanceId     string `form:"instance_id" binding:"required"`
}

// S3
type AwsS3ListBucketsReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id"` // S3可全局、或指定区域
	Prefix         string `form:"prefix"`
}

type AwsS3ListObjectsReq struct {
	MerchantId        int    `form:"merchant_id"`
	CloudAccountId    int64  `form:"cloud_account_id"`
	RegionId          string `form:"region_id"`
	Bucket            string `form:"bucket" binding:"required"`
	Prefix            string `form:"prefix"`
	ContinuationToken string `form:"continuation_token"`
	MaxKeys           int32  `form:"max_keys"`
}

type AwsS3ObjectItem struct {
	Key          string `json:"key"`
	Size         int64  `json:"size"`
	ETag         string `json:"etag"`
	LastModified string `json:"last_modified"`
	StorageClass string `json:"storage_class"`
}

type AwsS3ListObjectsResponse struct {
	List                  []AwsS3ObjectItem `json:"list"`
	IsTruncated           bool              `json:"is_truncated"`
	NextContinuationToken string            `json:"next_continuation_token"`
	Total                 int               `json:"total"`
}

type AwsS3SetBucketPublicReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id"`
	Bucket         string `json:"bucket" binding:"required"`
	Public         bool   `json:"public"` // true=公开，false=私有
}

// EC2 创建实例请求
type AwsCreateEc2InstanceReq struct {
	MerchantId       int      `json:"merchant_id"`
	CloudAccountId   int64    `json:"cloud_account_id"`
	RegionId         string   `json:"region_id" binding:"required"`
	ImageId          string   `json:"image_id"`                        // 镜像ID，为空使用默认Ubuntu
	InstanceType     string   `json:"instance_type" binding:"required"`
	SubnetId         string   `json:"subnet_id"`
	SecurityGroupIds []string `json:"security_group_ids"`
	KeyName          string   `json:"key_name"`
	VolumeSizeGiB    int32    `json:"volume_size_gib"`
	UserData         string   `json:"user_data"`
	InstanceName     string   `json:"instance_name"`
	ConfigureTSDD    bool     `json:"configure_tsdd"` // 是否为 TSDD AMI 部署，创建后自动配置 IP
}

// ========== 选择项查询 ==========
type AwsListImagesReq struct {
	MerchantId     int      `form:"merchant_id"`
	CloudAccountId int64    `form:"cloud_account_id"`
	RegionId       string   `form:"region_id" binding:"required"`
	Owners         []string `form:"owners[]"`
	Name           string   `form:"name"`
	MaxResults     int32    `form:"max_results"`
}

type AwsImageItem struct {
	ImageId      string `json:"image_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	OwnerId      string `json:"owner_id"`
	CreationDate string `json:"creation_date"`
}

type AwsListInstanceTypesReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	Prefix         string `form:"prefix"`
	MaxResults     int32  `form:"max_results"`
}

type AwsInstanceTypeItem struct {
	InstanceType string `json:"instance_type"`
	VCpu         int32  `json:"vcpu"`
	MemoryMiB    int64  `json:"memory_mib"`
}

type AwsInstanceTypesMemoryReq struct {
	MerchantId     int      `json:"merchant_id"`
	CloudAccountId int64    `json:"cloud_account_id"`
	RegionId       string   `json:"region_id" binding:"required"`
	Types          []string `json:"types" binding:"required,dive,required"`
}

type AwsListSubnetsReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	VpcId          string `form:"vpc_id"`
}

type AwsSubnetItem struct {
	SubnetId         string `json:"subnet_id"`
	VpcId            string `json:"vpc_id"`
	CidrBlock        string `json:"cidr_block"`
	AvailabilityZone string `json:"availability_zone"`
	Name             string `json:"name"`
}

type AwsListSecurityGroupsReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	Name           string `form:"name"`
}

type AwsSecurityGroupOption struct {
	GroupId   string `json:"group_id"`
	GroupName string `json:"group_name"`
}

// 修改 EC2 实例属性/标签
type AwsModifyEc2InstanceReq struct {
	MerchantId       int               `json:"merchant_id"`
	CloudAccountId   int64             `json:"cloud_account_id"`
	RegionId         string            `json:"region_id" binding:"required"`
	InstanceId       string            `json:"instance_id" binding:"required"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Tags             map[string]string `json:"tags"`
	SecurityGroupIds []string          `json:"security_group_ids"`
}

// 扩容（修改）EBS 卷容量并可选在实例内扩展文件系统
type AwsResizeVolumeReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	// 二选一：优先使用 VolumeId；否则通过 InstanceId(+DeviceName 可选)推导根卷或指定卷
	VolumeId   string `json:"volume_id"`
	InstanceId string `json:"instance_id"`
	// 可选：当通过 InstanceId 指定时，如提供 DeviceName（例如 /dev/xvda 或 /dev/nvme0n1p1）将精确匹配对应卷；
	// 若未提供则默认选择 RootDeviceName 对应的卷
	DeviceName  string `json:"device_name"`
	NewSizeGiB  int32  `json:"new_size_gib" binding:"required"`
	ExpandFS    bool   `json:"expand_fs"`    // 是否在实例内扩展分区/文件系统
	AsyncExpand bool   `json:"async_expand"` // 是否异步触发实例内扩展（默认true，避免接口长时间阻塞）
}

// 列举卷（可按实例或卷ID过滤）
type AwsListVolumesReq struct {
	MerchantId     int      `form:"merchant_id"`
	CloudAccountId int64    `form:"cloud_account_id"`
	RegionId       string   `form:"region_id" binding:"required"`
	InstanceId     string   `form:"instance_id"`  // 若提供则按实例过滤
	VolumeIds      []string `form:"volume_ids[]"` // 可选，按卷ID列表过滤
}

type AwsVolumeItem struct {
	VolumeId   string `json:"volume_id"`
	SizeGiB    int32  `json:"size_gib"`
	VolumeType string `json:"volume_type"`
	State      string `json:"state"`
	Encrypted  bool   `json:"encrypted"`
	DeviceName string `json:"device_name"`
}

// 获取卷使用率（通过 SSM 在实例内执行 df）
type AwsGetVolumeUsageReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	InstanceId     string `form:"instance_id" binding:"required"`
}

type AwsVolumeUsageItem struct {
	Source     string `json:"source"`     // 设备名，如 /dev/nvme0n1p1
	Mountpoint string `json:"mountpoint"` // 挂载点
	SizeBytes  int64  `json:"size_bytes"`
	UsedBytes  int64  `json:"used_bytes"`
	AvailBytes int64  `json:"avail_bytes"`
	Percent    int32  `json:"percent"` // 0-100
}

// 实例磁盘使用（来自实例内 df）
type AwsInstanceDiskUsageReq struct {
	MerchantId     int    `form:"merchant_id"`
	CloudAccountId int64  `form:"cloud_account_id"`
	RegionId       string `form:"region_id" binding:"required"`
	InstanceId     string `form:"instance_id" binding:"required"`
}

type AwsDiskUsageItem struct {
	Source      string `json:"source"`
	MountPoint  string `json:"mount_point"`
	FsType      string `json:"fs_type"`
	SizeBytes   int64  `json:"size_bytes"`
	UsedBytes   int64  `json:"used_bytes"`
	AvailBytes  int64  `json:"avail_bytes"`
	UsedPercent int32  `json:"used_percent"`
}

// S3 创建 Bucket 请求
type AwsS3CreateBucketReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
}

// S3 删除 Bucket 请求
type AwsS3DeleteBucketReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
}

// S3 删除对象请求
type AwsS3DeleteObjectReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	Bucket         string `json:"bucket" binding:"required"`
	Key            string `json:"key" binding:"required"`
}

// ========== AMI 相关 ==========

// 创建 AMI 请求
type AwsCreateAMIReq struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id" binding:"required"`
	InstanceId     string `json:"instance_id" binding:"required"`
	Name           string `json:"name" binding:"required"`
	Description    string `json:"description"`
	NoReboot       bool   `json:"no_reboot"` // 是否不重启实例（默认会重启以确保数据一致性）
}

// 创建 AMI 响应
type AwsCreateAMIResp struct {
	ImageId string `json:"image_id"`
	Name    string `json:"name"`
	State   string `json:"state"`
}

// 使用 AMI 部署 TSDD 请求
type DeployTSDDWithAMIReq struct {
	MerchantId     int    `json:"merchant_id" binding:"required"`
	CloudAccountId int64  `json:"cloud_account_id" binding:"required"`
	RegionId       string `json:"region_id" binding:"required"`
	AMIId          string `json:"ami_id"`                            // 可选，不填则使用默认 TSDD AMI
	InstanceType   string `json:"instance_type"`                     // 可选，默认 t3.medium
	SubnetId       string `json:"subnet_id"`                         // 可选
	KeyName        string `json:"key_name"`                          // 可选
	VolumeSizeGiB  int32  `json:"volume_size_gib"`                   // 可选，默认 30
	ServerName     string `json:"server_name"`                       // 服务器名称
	SourceServerId int    `json:"source_server_id"`                  // 可选，从某服务器克隆（会先创建 AMI）
}

// 使用 AMI 部署响应
type DeployTSDDWithAMIResp struct {
	Success    bool   `json:"success"`
	Message    string `json:"message"`
	InstanceId string `json:"instance_id"`
	PublicIP   string `json:"public_ip"`
	ServerId   int    `json:"server_id"` // 注册到系统的服务器ID
	APIUrl     string `json:"api_url"`
	WebUrl     string `json:"web_url"`
	AdminUrl   string `json:"admin_url"`
}
