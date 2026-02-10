package cloud_aws

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/big"
	"strings"
	"time"

	awscloud "server/internal/server/cloud/aws"
	"server/internal/server/model"
	"server/internal/server/utils"
	"server/pkg/consts"
	"server/pkg/entity"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	costexplorertypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/zeromicro/go-zero/core/logx"
)

// 常见规格内存映射（MiB），可按需补充
var presetInstanceTypeMemoryMiB = map[string]int64{
	"t2.nano":     512,
	"t2.micro":    1024,
	"t2.small":    2048,
	"t2.medium":   4096,
	"t2.large":    8192,
	"t3.nano":     512,
	"t3.micro":    1024,
	"t3.small":    2048,
	"t3.medium":   4096,
	"t3.large":    8192,
	"t3.xlarge":   16384,
	"t3.2xlarge":  32768,
	"t4g.nano":    512,
	"t4g.micro":   1024,
	"t4g.small":   2048,
	"t4g.medium":  4096,
	"t4g.large":   8192,
	"t4g.xlarge":  16384,
	"t4g.2xlarge": 32768,
}

func resolveInstanceMemoryMiB(ctx context.Context, cli *ec2.Client, instanceType string) int64 {
	if v, ok := presetInstanceTypeMemoryMiB[instanceType]; ok {
		return v
	}
	in := &ec2.DescribeInstanceTypesInput{InstanceTypes: []ec2types.InstanceType{ec2types.InstanceType(instanceType)}}
	out, err := cli.DescribeInstanceTypes(ctx, in)
	if err != nil || len(out.InstanceTypes) == 0 || out.InstanceTypes[0].MemoryInfo == nil || out.InstanceTypes[0].MemoryInfo.SizeInMiB == nil {
		return 0
	}
	return *out.InstanceTypes[0].MemoryInfo.SizeInMiB
}

// ResolveInstanceTypesMemoryMiB 批量查询规格内存
func ResolveInstanceTypesMemoryMiB(cloudAccountId int64, region string, types map[string]struct{}) (map[string]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2ClientBySystem(ctx, cloudAccountId, region)
	if err != nil {
		return nil, err
	}
	// 先映射内置
	result := make(map[string]int64, len(types))
	var toQuery []ec2types.InstanceType
	for t := range types {
		if v, ok := presetInstanceTypeMemoryMiB[t]; ok {
			result[t] = v
		} else {
			toQuery = append(toQuery, ec2types.InstanceType(t))
		}
	}
	// 分批请求避免超限
	const batch = 100
	for i := 0; i < len(toQuery); i += batch {
		end := i + batch
		if end > len(toQuery) {
			end = len(toQuery)
		}
		in := &ec2.DescribeInstanceTypesInput{InstanceTypes: toQuery[i:end]}
		out, err := cli.DescribeInstanceTypes(ctx, in)
		if err != nil {
			return result, err
		}
		for _, it := range out.InstanceTypes {
			if it.MemoryInfo != nil && it.MemoryInfo.SizeInMiB != nil {
				result[string(it.InstanceType)] = *it.MemoryInfo.SizeInMiB
			}
		}
	}
	return result, nil
}

// ResolveInstanceTypesMemoryMiBWithAccount 使用统一账号信息解析规格内存
func ResolveInstanceTypesMemoryMiBWithAccount(acc *entity.CloudAccounts, region string, types map[string]struct{}) (map[string]int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return nil, err
	}
	// 先映射内置
	result := make(map[string]int64, len(types))
	var toQuery []ec2types.InstanceType
	for t := range types {
		if v, ok := presetInstanceTypeMemoryMiB[t]; ok {
			result[t] = v
		} else {
			toQuery = append(toQuery, ec2types.InstanceType(t))
		}
	}
	// 分批请求避免超限
	const batch = 100
	for i := 0; i < len(toQuery); i += batch {
		end := i + batch
		if end > len(toQuery) {
			end = len(toQuery)
		}
		in := &ec2.DescribeInstanceTypesInput{InstanceTypes: toQuery[i:end]}
		out, err := cli.DescribeInstanceTypes(ctx, in)
		if err != nil {
			return result, err
		}
		for _, it := range out.InstanceTypes {
			if it.MemoryInfo != nil && it.MemoryInfo.SizeInMiB != nil {
				result[string(it.InstanceType)] = *it.MemoryInfo.SizeInMiB
			}
		}
	}
	return result, nil
}

// InstanceTypeInfo 实例类型详细信息
type InstanceTypeInfo struct {
	VCpu      int32
	MemoryMiB int64
}

// ResolveInstanceTypesInfoWithAccount 批量获取实例类型的 vCPU 和内存信息
func ResolveInstanceTypesInfoWithAccount(acc *entity.CloudAccounts, region string, types map[string]struct{}) (map[string]InstanceTypeInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return nil, err
	}

	result := make(map[string]InstanceTypeInfo, len(types))
	var toQuery []ec2types.InstanceType
	for t := range types {
		toQuery = append(toQuery, ec2types.InstanceType(t))
	}

	// 分批请求避免超限
	const batch = 100
	for i := 0; i < len(toQuery); i += batch {
		end := i + batch
		if end > len(toQuery) {
			end = len(toQuery)
		}
		in := &ec2.DescribeInstanceTypesInput{InstanceTypes: toQuery[i:end]}
		out, err := cli.DescribeInstanceTypes(ctx, in)
		if err != nil {
			return result, err
		}
		for _, it := range out.InstanceTypes {
			info := InstanceTypeInfo{}
			// 获取内存
			if it.MemoryInfo != nil && it.MemoryInfo.SizeInMiB != nil {
				info.MemoryMiB = *it.MemoryInfo.SizeInMiB
			}
			// 获取 vCPU
			if it.VCpuInfo != nil && it.VCpuInfo.DefaultVCpus != nil {
				info.VCpu = *it.VCpuInfo.DefaultVCpus
			}
			result[string(it.InstanceType)] = info
		}
	}
	return result, nil
}

// ========== EC2 ==========

func ListEc2Instances(acc *entity.CloudAccounts, region string) ([]ec2types.Instance, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return nil, err
	}
	var out []ec2types.Instance
	p := ec2.NewDescribeInstancesPaginator(cli, &ec2.DescribeInstancesInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		for _, r := range page.Reservations {
			out = append(out, r.Instances...)
		}
	}
	return out, nil
}

func OperateEc2Instance(acc *entity.CloudAccounts, region, instanceId, operation string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return err
	}
	switch operation {
	case "start":
		_, err = cli.StartInstances(ctx, &ec2.StartInstancesInput{InstanceIds: []string{instanceId}})
	case "stop":
		_, err = cli.StopInstances(ctx, &ec2.StopInstancesInput{InstanceIds: []string{instanceId}})
	case "reboot":
		_, err = cli.RebootInstances(ctx, &ec2.RebootInstancesInput{InstanceIds: []string{instanceId}})
	case "terminate":
		_, err = cli.TerminateInstances(ctx, &ec2.TerminateInstancesInput{InstanceIds: []string{instanceId}})
	}
	return err
}

// ========== Security Group ==========

func ListSecurityGroups(acc *entity.CloudAccounts, region string) ([]ec2types.SecurityGroup, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return nil, err
	}
	var out []ec2types.SecurityGroup
	p := ec2.NewDescribeSecurityGroupsPaginator(cli, &ec2.DescribeSecurityGroupsInput{})
	for p.HasMorePages() {
		page, err := p.NextPage(ctx)
		if err != nil {
			return nil, err
		}
		out = append(out, page.SecurityGroups...)
	}
	return out, nil
}

func DescribeSecurityGroup(acc *entity.CloudAccounts, region, groupId string) (*ec2.DescribeSecurityGroupsOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return nil, err
	}
	return cli.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{GroupIds: []string{groupId}})
}

func AuthorizeSecurityGroupIngress(acc *entity.CloudAccounts, req model.AwsAuthorizeSecurityGroupReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, req.RegionId)
	if err != nil {
		return err
	}
	ipRanges := make([]ec2types.IpRange, 0, len(req.CidrBlocks))
	for _, c := range req.CidrBlocks {
		ipRanges = append(ipRanges, ec2types.IpRange{CidrIp: &c})
	}
	_, err = cli.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: &req.GroupId,
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: &req.IpProtocol,
				FromPort:   &req.FromPort,
				ToPort:     &req.ToPort,
				IpRanges:   ipRanges,
			},
		},
	})
	return err
}

func RevokeSecurityGroupIngress(acc *entity.CloudAccounts, req model.AwsAuthorizeSecurityGroupReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, req.RegionId)
	if err != nil {
		return err
	}
	ipRanges := make([]ec2types.IpRange, 0, len(req.CidrBlocks))
	for _, c := range req.CidrBlocks {
		ipRanges = append(ipRanges, ec2types.IpRange{CidrIp: &c})
	}
	_, err = cli.RevokeSecurityGroupIngress(ctx, &ec2.RevokeSecurityGroupIngressInput{
		GroupId: &req.GroupId,
		IpPermissions: []ec2types.IpPermission{
			{
				IpProtocol: &req.IpProtocol,
				FromPort:   &req.FromPort,
				ToPort:     &req.ToPort,
				IpRanges:   ipRanges,
			},
		},
	})
	return err
}

// ========== EIP ==========

func ListAddresses(acc *entity.CloudAccounts, region string) ([]ec2types.Address, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return nil, err
	}
	out, err := cli.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{})
	if err != nil {
		return nil, err
	}
	return out.Addresses, nil
}

func OperateAddress(acc *entity.CloudAccounts, req model.AwsOperateEipReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, req.RegionId)
	if err != nil {
		return err
	}
	switch req.Operation {
	case "associate":
		_, err = cli.AssociateAddress(ctx, &ec2.AssociateAddressInput{
			AllocationId:       &req.AllocationId,
			InstanceId:         &req.InstanceId,
			NetworkInterfaceId: &req.NetworkInterfaceId,
			PrivateIpAddress:   &req.PrivateIpAddress,
		})
	case "disassociate":
		// 需要 AssociationId，简单场景用 AllocationId 先 Describe 查找
		addrs, derr := cli.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{AllocationIds: []string{req.AllocationId}})
		if derr != nil {
			return derr
		}
		if len(addrs.Addresses) > 0 && addrs.Addresses[0].AssociationId != nil {
			_, err = cli.DisassociateAddress(ctx, &ec2.DisassociateAddressInput{AssociationId: addrs.Addresses[0].AssociationId})
		}
	case "release":
		_, err = cli.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{AllocationId: &req.AllocationId})
	}
	return err
}

func AllocateAddress(acc *entity.CloudAccounts, region string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return "", err
	}
	out, err := cli.AllocateAddress(ctx, &ec2.AllocateAddressInput{Domain: ec2types.DomainTypeVpc})
	if err != nil {
		return "", err
	}
	if out.AllocationId == nil {
		return "", nil
	}
	return *out.AllocationId, nil
}

// DescribeInstance 返回单个实例的核心详情（用于 EIP 详情展示）
func DescribeInstance(acc *entity.CloudAccounts, region, instanceId string) (map[string]any, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return nil, err
	}
	out, err := cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceId}})
	if err != nil {
		return nil, err
	}
	if len(out.Reservations) == 0 || len(out.Reservations[0].Instances) == 0 {
		return map[string]any{}, nil
	}
	ins := out.Reservations[0].Instances[0]
	// 选取常用字段
	mem := resolveInstanceMemoryMiB(ctx, cli, string(ins.InstanceType))
	// 计算 vCPU：CoreCount × ThreadsPerCore
	vcpu := int32(0)
	if ins.CpuOptions != nil {
		cores := int32(1)
		threads := int32(1)
		if ins.CpuOptions.CoreCount != nil {
			cores = *ins.CpuOptions.CoreCount
		}
		if ins.CpuOptions.ThreadsPerCore != nil {
			threads = *ins.CpuOptions.ThreadsPerCore
		}
		vcpu = cores * threads
	}
	return map[string]any{
		"instance_id":   deref(ins.InstanceId),
		"instance_type": string(ins.InstanceType),
		"cpu":           vcpu,
		"memory_mib":    mem,
		"tags":          ins.Tags,
	}, nil
}

// ========== S3 ==========

// AwsBucketInfo AWS S3 Bucket 信息（包含区域）
type AwsBucketInfo struct {
	Name     string `json:"name"`
	Location string `json:"location"`
}

func ListBuckets(acc *entity.CloudAccounts, region string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	// ListBuckets 是全局操作，如果没有指定 region 则使用默认值
	if region == "" {
		region = "us-east-1"
	}
	cli, err := awscloud.NewS3Client(ctx, acc, region)
	if err != nil {
		return nil, err
	}
	out, err := cli.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}
	var names []string
	for _, b := range out.Buckets {
		if b.Name != nil {
			names = append(names, *b.Name)
		}
	}
	return names, nil
}

// ListBucketsWithLocation 列出 buckets 并获取每个 bucket 的区域
func ListBucketsWithLocation(acc *entity.CloudAccounts) ([]AwsBucketInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 使用 us-east-1 作为默认 region 来列出 buckets
	cli, err := awscloud.NewS3Client(ctx, acc, "us-east-1")
	if err != nil {
		return nil, err
	}

	out, err := cli.ListBuckets(ctx, &s3.ListBucketsInput{})
	if err != nil {
		return nil, err
	}

	var buckets []AwsBucketInfo
	for _, b := range out.Buckets {
		if b.Name == nil {
			continue
		}
		bucketName := *b.Name
		location := "us-east-1" // 默认值

		// 获取 bucket 的区域
		locOut, locErr := cli.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
			Bucket: &bucketName,
		})
		if locErr == nil && locOut.LocationConstraint != "" {
			location = string(locOut.LocationConstraint)
		}
		// 注意：us-east-1 的 bucket 返回空字符串，所以保持默认值

		buckets = append(buckets, AwsBucketInfo{
			Name:     bucketName,
			Location: location,
		})
	}

	return buckets, nil
}

func ListObjects(acc *entity.CloudAccounts, req model.AwsS3ListObjectsReq) (model.AwsS3ListObjectsResponse, error) {
	var resp model.AwsS3ListObjectsResponse
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewS3Client(ctx, acc, req.RegionId)
	if err != nil {
		return resp, err
	}
	in := &s3.ListObjectsV2Input{Bucket: &req.Bucket}
	if req.Prefix != "" {
		in.Prefix = &req.Prefix
	}
	if req.MaxKeys > 0 {
		in.MaxKeys = &req.MaxKeys
	}
	if req.ContinuationToken != "" {
		in.ContinuationToken = &req.ContinuationToken
	}
	out, err := cli.ListObjectsV2(ctx, in)
	if err != nil {
		return resp, err
	}
	for _, o := range out.Contents {
		size := int64(0)
		if o.Size != nil {
			size = *o.Size
		}
		item := model.AwsS3ObjectItem{Key: *o.Key, Size: size, StorageClass: string(o.StorageClass)}
		if o.ETag != nil {
			item.ETag = *o.ETag
		}
		if !o.LastModified.IsZero() {
			item.LastModified = o.LastModified.Format("2006-01-02 15:04:05")
		}
		resp.List = append(resp.List, item)
	}
	if out.IsTruncated != nil {
		resp.IsTruncated = *out.IsTruncated
	}
	if out.NextContinuationToken != nil {
		resp.NextContinuationToken = *out.NextContinuationToken
	}
	resp.Total = len(resp.List)
	return resp, nil
}

func UploadObject(acc *entity.CloudAccounts, region, bucket, key string, r io.Reader) error {
	// 增加超时时间至10分钟，支持大文件上传
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cli, err := awscloud.NewS3Client(ctx, acc, region)
	if err != nil {
		return err
	}
	_, err = cli.PutObject(ctx, &s3.PutObjectInput{Bucket: &bucket, Key: &key, Body: r})
	return err
}

func SetBucketPublicAccess(acc *entity.CloudAccounts, region, bucket string, public bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewS3Client(ctx, acc, region)
	if err != nil {
		return err
	}

	if public {
		// 设置为公开：允许所有人读取
		policy := `{
	"Version": "2012-10-17",
	"Statement": [
		{
			"Sid": "PublicReadGetObject",
			"Effect": "Allow",
			"Principal": "*",
			"Action": "s3:GetObject",
			"Resource": "arn:aws:s3:::` + bucket + `/*"
		}
	]
}`
		_, err = cli.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{
			Bucket: &bucket,
			Policy: &policy,
		})
		return err
	} else {
		// 设置为私有：删除公开策略
		_, err = cli.DeleteBucketPolicy(ctx, &s3.DeleteBucketPolicyInput{
			Bucket: &bucket,
		})
		// 如果策略不存在，AWS 会返回错误，我们忽略它
		if err != nil && !strings.Contains(err.Error(), "NoSuchBucketPolicy") {
			return err
		}
		return nil
	}
}

func DownloadObject(acc *entity.CloudAccounts, region, bucket, key string) ([]byte, string, string, error) {
	// 增加超时时间至10分钟，支持大文件下载
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	cli, err := awscloud.NewS3Client(ctx, acc, region)
	if err != nil {
		return nil, "", "", err
	}
	out, err := cli.GetObject(ctx, &s3.GetObjectInput{Bucket: &bucket, Key: &key})
	if err != nil {
		return nil, "", "", err
	}
	defer out.Body.Close()
	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, out.Body)
	if err != nil {
		return nil, "", "", err
	}
	contentType := "application/octet-stream"
	if out.ContentType != nil {
		contentType = *out.ContentType
	}
	filename := key
	return buf.Bytes(), contentType, filename, nil
}

// DeleteObject 删除 S3 对象
func DeleteObject(acc *entity.CloudAccounts, region, bucket, key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewS3Client(ctx, acc, region)
	if err != nil {
		return err
	}
	_, err = cli.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &bucket, Key: &key})
	return err
}

// CreateBucket 创建 S3 Bucket
func CreateBucket(acc *entity.CloudAccounts, region, bucket string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewS3Client(ctx, acc, region)
	if err != nil {
		return err
	}
	input := &s3.CreateBucketInput{
		Bucket: &bucket,
	}
	// us-east-1 不需要指定 LocationConstraint
	if region != "us-east-1" {
		input.CreateBucketConfiguration = &s3types.CreateBucketConfiguration{
			LocationConstraint: s3types.BucketLocationConstraint(region),
		}
	}
	_, err = cli.CreateBucket(ctx, input)
	return err
}

// DeleteBucket 删除 S3 Bucket（Bucket必须为空）
func DeleteBucket(acc *entity.CloudAccounts, region, bucket string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := awscloud.NewS3Client(ctx, acc, region)
	if err != nil {
		return err
	}
	_, err = cli.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: &bucket})
	return err
}

// ========== Billing (Cost Explorer) ==========

// GetCostAndUsage 查询账单（聚合，自定义指标/分组）
func GetCostAndUsage(acc *entity.CloudAccounts, region, startDate, endDate, granularity string, metrics []string, groupByKey string, excludeEstimated bool) (*costexplorer.GetCostAndUsageOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	// 前端传入 region，不再写死；若为空则由 NewCeClient 内部回退到 us-east-1
	ce, err := awscloud.NewCeClient(ctx, acc, region)
	if err != nil {
		return nil, err
	}
	if granularity == "" {
		granularity = string(costexplorertypes.GranularityDaily)
	}
	if len(metrics) == 0 {
		metrics = []string{"UnblendedCost"}
	}
	in := &costexplorer.GetCostAndUsageInput{
		TimePeriod:  &costexplorertypes.DateInterval{Start: &startDate, End: &endDate},
		Granularity: costexplorertypes.Granularity(granularity),
		Metrics:     metrics,
	}
	if groupByKey != "" {
		key := groupByKey
		typ := costexplorertypes.GroupDefinitionTypeDimension
		in.GroupBy = []costexplorertypes.GroupDefinition{{Type: typ, Key: &key}}
	}
	out, err := ce.GetCostAndUsage(ctx, in)
	if err != nil || !excludeEstimated {
		return out, err
	}
	var filtered []costexplorertypes.ResultByTime
	for _, r := range out.ResultsByTime {
		if r.Estimated {
			continue
		}
		filtered = append(filtered, r)
	}
	out.ResultsByTime = filtered
	return out, nil
}

// getDefaultAwsAmiId 获取默认的AWS Ubuntu AMI ID
func getDefaultAwsAmiId(region string) string {
	// AWS各地区Ubuntu 22.04 LTS AMI ID (HVM, SSD Volume Type)
	// 这些是官方Ubuntu AMI，需要定期更新
	defaultAmis := map[string]string{
		"us-east-1":      "ami-0c7217cdde317cfec", // 美东
		"us-west-2":      "ami-0efcece6bed30fd98", // 美西(俄勒冈)
		"eu-west-1":      "ami-0905a3c97561e0b69", // 欧洲(爱尔兰)
		"ap-southeast-1": "ami-078c1149d8ad719a7", // 新加坡
		"ap-northeast-1": "ami-0d52744d6551d851e", // 东京
		"ap-east-1":      "ami-0d96ec8a788679eb2", // 香港
	}
	if ami, ok := defaultAmis[region]; ok {
		return ami
	}
	// 默认返回美东的AMI
	return "ami-0c7217cdde317cfec"
}

// CreateEc2Instance 根据镜像创建单台实例
func CreateEc2Instance(acc *entity.CloudAccounts, req model.AwsCreateEc2InstanceReq) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	cli, err := awscloud.NewEc2Client(ctx, acc, req.RegionId)
	if err != nil {
		return "", err
	}

	// 如果未指定镜像ID，使用默认Ubuntu镜像
	if req.ImageId == "" {
		req.ImageId = getDefaultAwsAmiId(req.RegionId)
		logx.Infof("使用默认AMI: %s for region %s", req.ImageId, req.RegionId)
	}

	// 计算根设备名：优先使用镜像 RootDeviceName，回退为 /dev/xvda
	var deviceName *string = strPtr("/dev/xvda")
	if req.ImageId != "" {
		if imgOut, imgErr := cli.DescribeImages(ctx, &ec2.DescribeImagesInput{ImageIds: []string{req.ImageId}}); imgErr == nil {
			if len(imgOut.Images) > 0 && imgOut.Images[0].RootDeviceName != nil && *imgOut.Images[0].RootDeviceName != "" {
				deviceName = imgOut.Images[0].RootDeviceName
			}
		}
	}
	blockDevice := &ec2types.BlockDeviceMapping{
		DeviceName: deviceName,
		Ebs:        &ec2types.EbsBlockDevice{VolumeSize: &req.VolumeSizeGiB},
	}

	runIn := &ec2.RunInstancesInput{
		ImageId:      &req.ImageId,
		InstanceType: ec2types.InstanceType(req.InstanceType),
		MinCount:     int32Ptr(1),
		MaxCount:     int32Ptr(1),
	}
	if req.SubnetId != "" {
		runIn.SubnetId = &req.SubnetId
	}
	if req.KeyName != "" {
		runIn.KeyName = &req.KeyName
	}
	if req.VolumeSizeGiB > 0 {
		runIn.BlockDeviceMappings = []ec2types.BlockDeviceMapping{*blockDevice}
	}
	// 始终由后端生成用于 Debian 的 cloud-init，开启 root 密码登录
	rootPassword := consts.DefaultPassword //generateStrongPassword(16)
	userDataPlain := buildDebianRootPasswordCloudInit(rootPassword)
	userDataEncoded := base64.StdEncoding.EncodeToString([]byte(userDataPlain))
	runIn.UserData = &userDataEncoded

	out, err := cli.RunInstances(ctx, runIn)
	if err != nil {
		return "", err
	}
	if len(out.Instances) == 0 || out.Instances[0].InstanceId == nil {
		return "", nil
	}
	instance := out.Instances[0]
	instanceId := *instance.InstanceId

	// 可选：设置标签作为实例名
	if req.InstanceName != "" {
		_, _ = cli.CreateTags(ctx, &ec2.CreateTagsInput{
			Resources: []string{instanceId},
			Tags:      []ec2types.Tag{{Key: strPtr("Name"), Value: &req.InstanceName}},
		})
	}

	// 自动为实例所属安全组开放必要端口（22, 10543, 10544）
	groupIds := make([]string, 0, len(instance.SecurityGroups))
	for _, group := range instance.SecurityGroups {
		groupIds = append(groupIds, *group.GroupId)
	}
	logx.Errorf("aws实例安全组开放端口: instanceId: %s, groupIds: %v", instanceId, groupIds)
	if len(groupIds) > 0 {
		err = OpenRequiredPortsForSecurityGroups(ctx, cli, groupIds, 0)
		if err != nil {
			logx.Errorf("aws实例安全组开放端口: openRequiredPortsForSecurityGroups failed: %v", err)
		}
	}

	// 如果是 TSDD AMI 部署，异步配置 IP（不阻塞返回）
	if req.ConfigureTSDD {
		logx.Infof("[CreateEc2Instance] 启动 TSDD IP 配置任务: instanceId=%s", instanceId)
		ConfigureTSDDServicesIPAsync(acc, req.RegionId, instanceId)
	}

	return instanceId, nil
}

func int32Ptr(v int32) *int32 { return &v }
func strPtr(v string) *string { return &v }

type PortRange struct {
	fromPort int32
	toPort   int32
}

// OpenRequiredPortsForSecurityGroups 为安全组开放必要的端口
func OpenRequiredPortsForSecurityGroups(ctx context.Context, cli *ec2.Client, securityGroupIds []string, port int32) error {
	// 定义需要开放的端口规则
	requiredPorts := make([]PortRange, 0, 4)
	if port > 0 {
		requiredPorts = append(requiredPorts, PortRange{port, port}) // 指定端口
	} else {
		requiredPorts = append(requiredPorts, PortRange{22, 22})       // SSH
		requiredPorts = append(requiredPorts, PortRange{10010, 10012}) // gost端口
		requiredPorts = append(requiredPorts, PortRange{10000, 10002}) // 业务端口
		requiredPorts = append(requiredPorts, PortRange{8084, 8084})   // 管理后台端口
		requiredPorts = append(requiredPorts, PortRange{54321, 54321}) // 总控后台端口
	}

	protocol := "tcp"
	cidr := "0.0.0.0/0"

	// 为每个安全组添加规则
	for _, groupId := range securityGroupIds {
		for _, port := range requiredPorts {
			input := &ec2.AuthorizeSecurityGroupIngressInput{
				GroupId: &groupId,
				IpPermissions: []ec2types.IpPermission{
					{
						IpProtocol: &protocol,
						FromPort:   &port.fromPort,
						ToPort:     &port.toPort,
						IpRanges: []ec2types.IpRange{
							{CidrIp: &cidr},
						},
					},
				},
			}
			// 忽略规则已存在的错误
			_, _ = cli.AuthorizeSecurityGroupIngress(ctx, input)
		}
	}
	return nil
}

// generateStrongPassword 生成包含大小写、数字和符号的强密码
func generateStrongPassword(length int) string {
	if length < 8 {
		length = 8
	}
	lower := []rune("abcdefghijklmnopqrstuvwxyz")
	upper := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	digits := []rune("0123456789")
	symbols := []rune("!@#-_+=^&*?")
	all := append(append(append([]rune{}, lower...), upper...), digits...)
	all = append(all, symbols...)

	// 确保每类至少一个
	pass := make([]rune, 0, length)
	pass = append(pass, randPick(lower), randPick(upper), randPick(digits), randPick(symbols))
	for len(pass) < length {
		pass = append(pass, randPick(all))
	}
	// 简单洗牌
	for i := range pass {
		j := int(randInt(int64(i + 1)))
		pass[i], pass[j] = pass[j], pass[i]
	}
	return string(pass)
}

func randPick(set []rune) rune {
	idx := randInt(int64(len(set)))
	return set[idx]
}

func randInt(n int64) int64 {
	if n <= 0 {
		return 0
	}
	v, err := rand.Int(rand.Reader, big.NewInt(n))
	if err != nil {
		return 0
	}
	return v.Int64()
}

// buildDebianRootPasswordCloudInit 生成 Debian 系列启用 root 密码登录的 cloud-init 配置
func buildDebianRootPasswordCloudInit(password string) string {
	// 使用 cloud-config 配置 root 密码与 SSH 密码登录
	return "#cloud-config\n" +
		"ssh_pwauth: true\n" +
		"disable_root: false\n" +
		"chpasswd:\n" +
		"  list: |\n" +
		"    root:" + password + "\n" +
		"  expire: false\n" +
		"runcmd:\n" +
		"  - sed -i 's/^#\\?PermitRootLogin.*/PermitRootLogin yes/' /etc/ssh/sshd_config\n" +
		"  - sed -i 's/^#\\?PasswordAuthentication.*/PasswordAuthentication yes/' /etc/ssh/sshd_config\n" +
		"  - passwd -u root || true\n" +
		"  - systemctl reload ssh || systemctl reload sshd\n"
}

// ===== 选择项 =====
func ListImages(req model.AwsListImagesReq) ([]model.AwsImageItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var cli *ec2.Client
	var err error
	if req.CloudAccountId > 0 {
		cli, err = awscloud.NewEc2ClientBySystem(ctx, req.CloudAccountId, req.RegionId)
	} else {
		cli, err = awscloud.NewEc2ClientByMerchant(ctx, req.MerchantId, req.RegionId)
	}
	if err != nil {
		return nil, err
	}

	in := &ec2.DescribeImagesInput{}
	if len(req.Owners) > 0 {
		in.Owners = req.Owners
	}
	if req.Name != "" {
		in.Filters = append(in.Filters, ec2types.Filter{Name: strPtr("name"), Values: []string{"*" + req.Name + "*"}})
	}
	if req.MaxResults > 0 {
		in.MaxResults = int32Ptr(req.MaxResults)
	}
	out, err := cli.DescribeImages(ctx, in)
	if err != nil {
		return nil, err
	}
	items := make([]model.AwsImageItem, 0, len(out.Images))
	for _, img := range out.Images {
		items = append(items, model.AwsImageItem{
			ImageId:      deref(img.ImageId),
			Name:         deref(img.Name),
			Description:  deref(img.Description),
			OwnerId:      deref(img.OwnerId),
			CreationDate: deref(img.CreationDate),
		})
	}
	return items, nil
}

func ListInstanceTypes(req model.AwsListInstanceTypesReq) ([]model.AwsInstanceTypeItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var cli *ec2.Client
	var err error
	if req.CloudAccountId > 0 {
		cli, err = awscloud.NewEc2ClientBySystem(ctx, req.CloudAccountId, req.RegionId)
	} else {
		cli, err = awscloud.NewEc2ClientByMerchant(ctx, req.MerchantId, req.RegionId)
	}
	if err != nil {
		return nil, err
	}

	in := &ec2.DescribeInstanceTypesInput{}
	if req.Prefix != "" {
		in.Filters = append(in.Filters, ec2types.Filter{Name: strPtr("instance-type"), Values: []string{req.Prefix + "*"}})
	}
	if req.MaxResults > 0 {
		in.MaxResults = int32Ptr(req.MaxResults)
	}
	out, err := cli.DescribeInstanceTypes(ctx, in)
	if err != nil {
		return nil, err
	}
	items := make([]model.AwsInstanceTypeItem, 0, len(out.InstanceTypes))
	for _, it := range out.InstanceTypes {
		items = append(items, model.AwsInstanceTypeItem{
			InstanceType: string(it.InstanceType),
			VCpu:         deref(it.VCpuInfo.DefaultVCpus),
			MemoryMiB:    deref(it.MemoryInfo.SizeInMiB),
		})
	}
	return items, nil
}

func ListSubnets(req model.AwsListSubnetsReq) ([]model.AwsSubnetItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var cli *ec2.Client
	var err error
	if req.CloudAccountId > 0 {
		cli, err = awscloud.NewEc2ClientBySystem(ctx, req.CloudAccountId, req.RegionId)
	} else {
		cli, err = awscloud.NewEc2ClientByMerchant(ctx, req.MerchantId, req.RegionId)
	}
	if err != nil {
		return nil, err
	}

	in := &ec2.DescribeSubnetsInput{}
	if req.VpcId != "" {
		in.Filters = append(in.Filters, ec2types.Filter{Name: strPtr("vpc-id"), Values: []string{req.VpcId}})
	}
	out, err := cli.DescribeSubnets(ctx, in)
	if err != nil {
		return nil, err
	}
	items := make([]model.AwsSubnetItem, 0, len(out.Subnets))
	for _, s := range out.Subnets {
		var name string
		for _, t := range s.Tags {
			if deref(t.Key) == "Name" {
				name = deref(t.Value)
				break
			}
		}
		items = append(items, model.AwsSubnetItem{
			SubnetId:         deref(s.SubnetId),
			VpcId:            deref(s.VpcId),
			CidrBlock:        deref(s.CidrBlock),
			AvailabilityZone: deref(s.AvailabilityZone),
			Name:             name,
		})
	}
	return items, nil
}

func ListSecurityGroupOptions(req model.AwsListSecurityGroupsReq) ([]model.AwsSecurityGroupOption, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var cli *ec2.Client
	var err error
	if req.CloudAccountId > 0 {
		cli, err = awscloud.NewEc2ClientBySystem(ctx, req.CloudAccountId, req.RegionId)
	} else {
		cli, err = awscloud.NewEc2ClientByMerchant(ctx, req.MerchantId, req.RegionId)
	}
	if err != nil {
		return nil, err
	}

	in := &ec2.DescribeSecurityGroupsInput{}
	if req.Name != "" {
		in.Filters = append(in.Filters, ec2types.Filter{Name: strPtr("group-name"), Values: []string{"*" + req.Name + "*"}})
	}
	out, err := cli.DescribeSecurityGroups(ctx, in)
	if err != nil {
		return nil, err
	}
	items := make([]model.AwsSecurityGroupOption, 0, len(out.SecurityGroups))
	for _, g := range out.SecurityGroups {
		items = append(items, model.AwsSecurityGroupOption{GroupId: deref(g.GroupId), GroupName: deref(g.GroupName)})
	}
	return items, nil
}

func deref[T ~string | ~int32 | ~int64](p *T) T {
	if p == nil {
		var z T
		return z
	}
	return *p
}

// ModifyEc2Instance 修改实例名称/描述和标签
func ModifyEc2Instance(req model.AwsModifyEc2InstanceReq) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var cli *ec2.Client
	var err error
	if req.CloudAccountId > 0 {
		cli, err = awscloud.NewEc2ClientBySystem(ctx, req.CloudAccountId, req.RegionId)
	} else {
		cli, err = awscloud.NewEc2ClientByMerchant(ctx, req.MerchantId, req.RegionId)
	}
	if err != nil {
		return err
	}

	// 名称/描述通过标签 Name 与 Description 体现
	var tags []ec2types.Tag
	if req.Name != "" {
		tags = append(tags, ec2types.Tag{Key: strPtr("Name"), Value: &req.Name})
	}
	if req.Description != "" {
		tags = append(tags, ec2types.Tag{Key: strPtr("Description"), Value: &req.Description})
	}
	for k, v := range req.Tags {
		kCopy, vCopy := k, v
		tags = append(tags, ec2types.Tag{Key: &kCopy, Value: &vCopy})
	}
	if len(tags) > 0 {
		_, err = cli.CreateTags(ctx, &ec2.CreateTagsInput{Resources: []string{req.InstanceId}, Tags: tags})
		if err != nil {
			return err
		}
	}
	return nil
}

// ListVolumes 列举卷（支持按实例或卷ID过滤），并补充设备名
func ListVolumes(req model.AwsListVolumesReq) ([]model.AwsVolumeItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var cli *ec2.Client
	var err error
	if req.CloudAccountId > 0 {
		cli, err = awscloud.NewEc2ClientBySystem(ctx, req.CloudAccountId, req.RegionId)
	} else {
		cli, err = awscloud.NewEc2ClientByMerchant(ctx, req.MerchantId, req.RegionId)
	}
	if err != nil {
		return nil, err
	}

	in := &ec2.DescribeVolumesInput{}
	if len(req.VolumeIds) > 0 {
		in.VolumeIds = req.VolumeIds
	}
	if req.InstanceId != "" {
		in.Filters = append(in.Filters, ec2types.Filter{Name: strPtr("attachment.instance-id"), Values: []string{req.InstanceId}})
	}
	out, err := cli.DescribeVolumes(ctx, in)
	if err != nil {
		return nil, err
	}
	items := make([]model.AwsVolumeItem, 0, len(out.Volumes))
	for _, v := range out.Volumes {
		var dev string
		if len(v.Attachments) > 0 && v.Attachments[0].Device != nil {
			dev = *v.Attachments[0].Device
		}
		items = append(items, model.AwsVolumeItem{
			VolumeId:   deref(v.VolumeId),
			SizeGiB:    deref(v.Size),
			VolumeType: string(v.VolumeType),
			State:      string(v.State),
			Encrypted:  v.Encrypted != nil && *v.Encrypted,
			DeviceName: dev,
		})
	}
	return items, nil
}

// GetVolumeUsage 通过 SSM 在实例内执行 df，返回挂载点使用情况
func GetVolumeUsage(req model.AwsGetVolumeUsageReq) ([]model.AwsVolumeUsageItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// SSM 客户端
	var ssmcli *ssm.Client
	var err error
	if req.CloudAccountId > 0 {
		acc, erra := awscloud.ResolveAwsAccount(ctx, 0, req.CloudAccountId)
		if erra != nil {
			return nil, erra
		}
		ssmcli, err = awscloud.NewSsmClient(ctx, acc, req.RegionId)
	} else {
		acc, erra := awscloud.ResolveAwsAccount(ctx, req.MerchantId, 0)
		if erra != nil {
			return nil, erra
		}
		ssmcli, err = awscloud.NewSsmClient(ctx, acc, req.RegionId)
	}
	if err != nil {
		return nil, err
	}

	commands := []string{
		"set -euo pipefail",
		// 过滤 tmpfs/devtmpfs/overlay 等内存或容器卷
		"df -B1 --output=source,size,used,avail,pcent,target -x tmpfs -x devtmpfs -x overlay | tail -n +2",
	}
	in := &ssm.SendCommandInput{
		DocumentName:   strPtr("AWS-RunShellScript"),
		InstanceIds:    []string{req.InstanceId},
		Comment:        strPtr("Collect filesystem usage"),
		TimeoutSeconds: int32Ptr(60),
		Parameters:     map[string][]string{"commands": commands},
	}
	out, err := ssmcli.SendCommand(ctx, in)
	if err != nil {
		// Fallback: SSH 采集
		return getVolumeUsageViaSSH(req)
	}
	if out.Command == nil || out.Command.CommandId == nil {
		return nil, errors.New("failed to send ssm command")
	}

	cmdId := deref(out.Command.CommandId)
	// 轮询直到完成或超时
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		time.Sleep(3 * time.Second)
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		inv, err2 := ssmcli.GetCommandInvocation(ctx2, &ssm.GetCommandInvocationInput{CommandId: &cmdId, InstanceId: &req.InstanceId})
		cancel2()
		if err2 != nil {
			continue
		}
		if inv.Status == ssmtypes.CommandInvocationStatusSuccess {
			return parseDfOutput(deref(inv.StandardOutputContent)), nil
		}
		if inv.Status == ssmtypes.CommandInvocationStatusFailed || inv.Status == ssmtypes.CommandInvocationStatusCancelled || inv.Status == ssmtypes.CommandInvocationStatusTimedOut {
			// Fallback: SSH 采集
			return getVolumeUsageViaSSH(req)
		}
	}
	// 超时则回退到 SSH
	return getVolumeUsageViaSSH(req)
}

func parseDfOutput(out string) []model.AwsVolumeUsageItem {
	var items []model.AwsVolumeUsageItem
	lines := strings.Split(out, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 6 {
			continue
		}
		source := fields[0]
		size := parseInt64(fields[1])
		used := parseInt64(fields[2])
		avail := parseInt64(fields[3])
		pcent := strings.TrimSuffix(fields[4], "%")
		percent := int32(parseInt64(pcent))
		mount := strings.Join(fields[5:], " ")
		items = append(items, model.AwsVolumeUsageItem{
			Source:     source,
			Mountpoint: mount,
			SizeBytes:  size,
			UsedBytes:  used,
			AvailBytes: avail,
			Percent:    percent,
		})
	}
	return items
}

func parseInt64(s string) int64 {
	var v int64
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c < '0' || c > '9' {
			continue
		}
		v = v*10 + int64(c-'0')
	}
	return v
}

// 通过 SSH 采集 df 使用情况（根据实例ID查IP并匹配 servers 表）
func getVolumeUsageViaSSH(req model.AwsGetVolumeUsageReq) ([]model.AwsVolumeUsageItem, error) {
	// 1) 根据实例ID查询其 IP
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	acc, err := awscloud.ResolveAwsAccount(ctx, req.MerchantId, req.CloudAccountId)
	if err != nil {
		return nil, err
	}
	ec2cli, err := awscloud.NewEc2Client(ctx, acc, req.RegionId)
	if err != nil {
		return nil, err
	}
	din, err := ec2cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{req.InstanceId}})
	if err != nil {
		return nil, err
	}
	if len(din.Reservations) == 0 || len(din.Reservations[0].Instances) == 0 {
		return nil, errors.New("实例不存在")
	}
	ins := din.Reservations[0].Instances[0]
	ip := deref(ins.PublicIpAddress)
	if ip == "" {
		ip = deref(ins.PrivateIpAddress)
	}
	if ip == "" && len(ins.NetworkInterfaces) > 0 {
		if ins.NetworkInterfaces[0].Association != nil && ins.NetworkInterfaces[0].Association.PublicIp != nil {
			ip = deref(ins.NetworkInterfaces[0].Association.PublicIp)
		} else if ins.NetworkInterfaces[0].PrivateIpAddress != nil {
			ip = deref(ins.NetworkInterfaces[0].PrivateIpAddress)
		}
	}
	if ip == "" {
		return nil, errors.New("无法解析实例IP")
	}

	// 2) 使用 root/DefaultPassword 直接 SSH 采集
	client := &utils.SSHClient{Host: ip, Port: 22, Username: "root", Password: consts.DefaultPassword}
	cmd := "LC_ALL=C LANG=C df -B1 --output=source,size,used,avail,pcent,target -x tmpfs -x devtmpfs -x overlay 2>/dev/null | tail -n +2"
	out, _ := client.ExecuteCommandWithTimeout(cmd, 30*time.Second)
	// 3) 解析输出
	items := parseDfOutput(out)
	return items, nil
}

// ========== AMI 操作 ==========

// CreateAMI 从实例创建 AMI
func CreateAMI(acc *entity.CloudAccounts, req model.AwsCreateAMIReq) (model.AwsCreateAMIResp, error) {
	var resp model.AwsCreateAMIResp
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cli, err := awscloud.NewEc2Client(ctx, acc, req.RegionId)
	if err != nil {
		return resp, err
	}

	input := &ec2.CreateImageInput{
		InstanceId:  &req.InstanceId,
		Name:        &req.Name,
		Description: &req.Description,
		NoReboot:    &req.NoReboot,
	}

	out, err := cli.CreateImage(ctx, input)
	if err != nil {
		return resp, err
	}

	resp.ImageId = deref(out.ImageId)
	resp.Name = req.Name
	resp.State = "pending"

	// 添加标签
	if resp.ImageId != "" {
		_, _ = cli.CreateTags(ctx, &ec2.CreateTagsInput{
			Resources: []string{resp.ImageId},
			Tags: []ec2types.Tag{
				{Key: strPtr("Name"), Value: &req.Name},
				{Key: strPtr("CreatedBy"), Value: strPtr("TSDD-Control")},
			},
		})
	}

	return resp, nil
}

// WaitForAMIAvailable 等待 AMI 变为 available 状态
func WaitForAMIAvailable(acc *entity.CloudAccounts, region, imageId string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := cli.DescribeImages(ctx, &ec2.DescribeImagesInput{ImageIds: []string{imageId}})
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		if len(out.Images) > 0 {
			state := string(out.Images[0].State)
			if state == "available" {
				return nil
			}
			if state == "failed" {
				return errors.New("AMI 创建失败")
			}
		}
		time.Sleep(10 * time.Second)
	}
	return errors.New("等待 AMI 可用超时")
}

// GetInstancePublicIP 获取实例的公网 IP
func GetInstancePublicIP(acc *entity.CloudAccounts, region, instanceId string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return "", err
	}

	out, err := cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceId}})
	if err != nil {
		return "", err
	}

	if len(out.Reservations) == 0 || len(out.Reservations[0].Instances) == 0 {
		return "", errors.New("实例不存在")
	}

	ins := out.Reservations[0].Instances[0]
	ip := deref(ins.PublicIpAddress)
	if ip == "" && len(ins.NetworkInterfaces) > 0 {
		if ins.NetworkInterfaces[0].Association != nil {
			ip = deref(ins.NetworkInterfaces[0].Association.PublicIp)
		}
	}
	return ip, nil
}

// WaitForInstanceRunning 等待实例变为 running 状态
func WaitForInstanceRunning(acc *entity.CloudAccounts, region, instanceId string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cli, err := awscloud.NewEc2Client(ctx, acc, region)
	if err != nil {
		return err
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		out, err := cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceId}})
		if err != nil {
			time.Sleep(5 * time.Second)
			continue
		}
		if len(out.Reservations) > 0 && len(out.Reservations[0].Instances) > 0 {
			state := out.Reservations[0].Instances[0].State
			if state != nil && state.Name == ec2types.InstanceStateNameRunning {
				return nil
			}
		}
		time.Sleep(5 * time.Second)
	}
	return errors.New("等待实例启动超时")
}

// ConfigureTSDDServicesIP 在 EC2 实例创建后配置 TSDD 服务 IP
// 该函数会 SSH 到实例，更新 .env 文件和 Web 前端配置，然后重启服务
func ConfigureTSDDServicesIP(acc *entity.CloudAccounts, region, instanceId string) error {
	// 1. 等待实例运行
	logx.Infof("[ConfigureTSDDServicesIP] 等待实例 %s 运行...", instanceId)
	if err := WaitForInstanceRunning(acc, region, instanceId, 5*time.Minute); err != nil {
		return err
	}

	// 2. 获取公网 IP
	publicIP, err := GetInstancePublicIP(acc, region, instanceId)
	if err != nil {
		return err
	}
	if publicIP == "" {
		return errors.New("实例没有公网 IP")
	}
	logx.Infof("[ConfigureTSDDServicesIP] 实例 %s 公网 IP: %s", instanceId, publicIP)

	// 3. 等待 SSH 可用（cloud-init 可能还在执行）
	time.Sleep(30 * time.Second)

	// 4. SSH 连接并配置
	client := &utils.SSHClient{
		Host:     publicIP,
		Port:     22,
		Username: "root",
		Password: consts.DefaultPassword,
	}

	// 重试 SSH 连接
	var lastErr error
	for i := 0; i < 5; i++ {
		if i > 0 {
			time.Sleep(15 * time.Second)
		}
		lastErr = configureTSDDServicesViaSSH(client, publicIP)
		if lastErr == nil {
			logx.Infof("[ConfigureTSDDServicesIP] 实例 %s TSDD 服务配置完成", instanceId)
			return nil
		}
		logx.Errorf("[ConfigureTSDDServicesIP] SSH 配置失败 (尝试 %d/5): %v", i+1, lastErr)
	}
	return lastErr
}

// configureTSDDServicesViaSSH 通过 SSH 配置 TSDD 服务
func configureTSDDServicesViaSSH(client *utils.SSHClient, publicIP string) error {
	configVersion := time.Now().Unix()

	// 配置脚本：更新 .env，更新 Web 前端配置，重启服务
	script := buildTSDDConfigScript(publicIP, configVersion)

	output, err := client.ExecuteCommandWithTimeout(script, 3*time.Minute)
	if err != nil {
		return err
	}
	logx.Infof("[configureTSDDServicesViaSSH] 输出: %s", output)
	return nil
}

// buildTSDDConfigScript 构建 TSDD 配置脚本
func buildTSDDConfigScript(publicIP string, configVersion int64) string {
	return fmt.Sprintf(`#!/bin/bash
set -e

TSDD_DIR="/opt/tsdd"
NEW_IP="%s"
CONFIG_VERSION="%d"

echo "=== 配置 TSDD 服务 IP: $NEW_IP ==="

# 1. 更新 .env 文件
if [ -f "$TSDD_DIR/.env" ]; then
    if grep -q "^EXTERNAL_IP=" "$TSDD_DIR/.env"; then
        sed -i "s/^EXTERNAL_IP=.*/EXTERNAL_IP=$NEW_IP/" "$TSDD_DIR/.env"
    else
        echo "EXTERNAL_IP=$NEW_IP" >> "$TSDD_DIR/.env"
    fi
    echo "已更新 .env: EXTERNAL_IP=$NEW_IP"
else
    echo "警告: $TSDD_DIR/.env 不存在"
fi

# 2. 等待 Docker 服务就绪
sleep 5

# 3. 更新 Web 前端配置
if docker ps | grep -q tsdd-web; then
    docker exec tsdd-web sh -c "cat > /usr/share/nginx/html/tsdd-config.js << 'CONFIGEOF'
window.TSDD_CONFIG = {
  IP: '$NEW_IP',
  WS_PORT: 5200,
  HTTP_PORT: 8090
};
(function() {
  var CONFIG_VERSION = '$CONFIG_VERSION';
  var savedVersion = localStorage.getItem('ip_config_version');
  if (savedVersion !== CONFIG_VERSION) {
    localStorage.removeItem('ip_config_custom');
    localStorage.removeItem('ip_config_custom_ports');
    localStorage.removeItem('ip_config_ip_list');
    localStorage.removeItem('ip_config_last_ip');
    localStorage.removeItem('ip_config_invite_code');
    localStorage.setItem('ip_config_custom', JSON.stringify({ip:'$NEW_IP',wsPort:'5200',httpPort:'8090',mode:'custom'}));
    localStorage.setItem('ip_config_custom_ports', JSON.stringify({wsPort:5200,httpPort:8090}));
    localStorage.setItem('ip_config_ip_list', JSON.stringify(['$NEW_IP']));
    localStorage.setItem('ip_config_last_ip', '$NEW_IP');
    localStorage.setItem('ip_config_version', CONFIG_VERSION);
  }
})();
CONFIGEOF"

    # 确保 index.html 引用了 tsdd-config.js
    docker exec tsdd-web sh -c 'grep -q "tsdd-config.js" /usr/share/nginx/html/index.html || sed -i "s|<script defer=\"defer\"|<script src=\"./tsdd-config.js\"></script><script defer=\"defer\"|" /usr/share/nginx/html/index.html'
    echo "已更新 Web 前端配置"
else
    echo "警告: tsdd-web 容器未运行"
fi

# 4. 更新管理后台配置
if docker ps | grep -q tsdd-manager; then
    docker exec tsdd-manager sh -c "cat > /usr/share/nginx/html/tsdd-config.js << 'EOF'
var TSDD_CONFIG = {APP_URL: '/api/'};
window.TSDD_CONFIG = TSDD_CONFIG;
EOF"
    echo "已更新管理后台配置"
fi

# 5. 重启需要 IP 配置的服务
cd "$TSDD_DIR"
if [ -f "docker-compose.yml" ] || [ -f "docker-compose.yaml" ]; then
    docker compose down tsddserver wukongim 2>/dev/null || true
    docker compose up -d
    echo "已重启 Docker 服务"
fi

echo "=== TSDD 服务 IP 配置完成 ==="
echo "公网 IP: $NEW_IP"
echo "Web: http://$NEW_IP:82"
echo "Admin: http://$NEW_IP:8084"
echo "API: http://$NEW_IP:8090"
`, publicIP, configVersion)
}

// ConfigureTSDDServicesIPAsync 异步配置 TSDD 服务 IP（用于创建实例后的后台任务）
func ConfigureTSDDServicesIPAsync(acc *entity.CloudAccounts, region, instanceId string) {
	go func() {
		if err := ConfigureTSDDServicesIP(acc, region, instanceId); err != nil {
			logx.Errorf("[ConfigureTSDDServicesIPAsync] 配置失败: %v", err)
		}
	}()
}

// 通过 SSH 执行扩展分区/文件系统（与 SSM 脚本一致，尽量幂等）
func sshExpandFs(req model.AwsResizeVolumeReq) error {
	// 1) 解出实例 IP
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	acc, err := awscloud.ResolveAwsAccount(ctx, req.MerchantId, req.CloudAccountId)
	if err != nil {
		return err
	}
	ec2cli, err := awscloud.NewEc2Client(ctx, acc, req.RegionId)
	if err != nil {
		return err
	}
	din, err := ec2cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{req.InstanceId}})
	if err != nil {
		return err
	}
	if len(din.Reservations) == 0 || len(din.Reservations[0].Instances) == 0 {
		return errors.New("实例不存在")
	}
	ins := din.Reservations[0].Instances[0]
	ip := deref(ins.PublicIpAddress)
	if ip == "" {
		ip = deref(ins.PrivateIpAddress)
	}
	if ip == "" && len(ins.NetworkInterfaces) > 0 {
		if ins.NetworkInterfaces[0].Association != nil && ins.NetworkInterfaces[0].Association.PublicIp != nil {
			ip = deref(ins.NetworkInterfaces[0].Association.PublicIp)
		} else if ins.NetworkInterfaces[0].PrivateIpAddress != nil {
			ip = deref(ins.NetworkInterfaces[0].PrivateIpAddress)
		}
	}
	if ip == "" {
		return errors.New("无法解析实例IP")
	}

	// 2) 直接 SSH（root/DefaultPassword）
	client := &utils.SSHClient{Host: ip, Port: 22, Username: "root", Password: consts.DefaultPassword}
	// 与 SSM 相同的脚本（去掉 sudo）
	script := strings.Join([]string{
		"set -euxo pipefail",
		"ROOT=$(findmnt -n -o SOURCE /)",
		"PARENT=$ROOT; PARTNUM=",
		"if [[ $ROOT =~ ^/dev/nvme[0-9]+n[0-9]+p[0-9]+$ ]]; then PARENT=${ROOT%p*}; PARTNUM=${ROOT##*p}; fi",
		"if [[ $ROOT =~ ^/dev/[a-z]+[0-9]+$ ]]; then PARENT=$(echo $ROOT | sed -E 's/[0-9]+$//'); PARTNUM=$(echo $ROOT | sed -E 's/^.*([0-9]+)$/\\1/'); fi",
		// 安装 growpart
		"if ! command -v growpart >/dev/null 2>&1; then if command -v yum >/dev/null 2>&1; then yum install -y cloud-utils-growpart || true; fi; fi",
		"if ! command -v growpart >/dev/null 2>&1; then if command -v apt-get >/dev/null 2>&1; then apt-get update && apt-get install -y cloud-guest-utils || true; fi; fi",
		// 扩分区
		"if command -v growpart >/dev/null 2>&1 && [ -n \"$PARTNUM\" ]; then growpart $PARENT $PARTNUM || true; fi",
		// 扩文件系统
		"FSTYPE=$(findmnt -n -o FSTYPE /)",
		"if [ \"$FSTYPE\" = \"xfs\" ]; then xfs_growfs /; else resize2fs \"$ROOT\"; fi",
		"echo done",
	}, " && ")
	_, err = client.ExecuteCommandWithTimeout(script, 2*time.Minute)
	return err
}
