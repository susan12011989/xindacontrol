package aliyun

import (
	"errors"

	"github.com/alibabacloud-go/tea/tea"
	vpc20160428 "github.com/alibabacloud-go/vpc-20160428/v6/client"
	"github.com/zeromicro/go-zero/core/logx"
)

type AllocateEipAddressRequest struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	RegionId           string `json:"region_id"`
	InstanceChargeType string `json:"instance_charge_type"` // PrePaid：包年包月。 PostPaid（默认值）：按量计费
	/*
		InternetChargeType: PayByBandwidth（默认值）：按带宽计费。 PayByTraffic：按流量计费
		当 InstanceChargeType 取值为 PrePaid 时，InternetChargeType 必须取值 PayByBandwidth。
		当 InstanceChargeType 取值为 PostPaid 时，InternetChargeType 可取值 PayByBandwidth 或 PayByTraffic。
	*/
	InternetChargeType string `json:"internet_charge_type"`
	Bandwidth          string `json:"bandwidth"` // 带宽
	Num                int    `json:"num"`       // 数量
}

// 申请EIP,返回实例id，ip地址
func AllocateEipAddress(req *AllocateEipAddressRequest) (string, string, error) {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return "", "", err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return "", "", err
	}
	request := &vpc20160428.AllocateEipAddressRequest{
		RegionId:           tea.String(req.RegionId),
		AutoPay:            tea.Bool(false),
		InstanceChargeType: tea.String(req.InstanceChargeType),
		InternetChargeType: tea.String(req.InternetChargeType),
		Bandwidth:          tea.String(req.Bandwidth),
		ISP:                tea.String("BGP"),
	}
	response, err := client.AllocateEipAddress(request)
	if err != nil {
		return "", "", err
	}
	if *response.StatusCode != 200 {
		return "", "", errors.New(response.String())
	}
	return *response.Body.AllocationId, *response.Body.EipAddress, nil
}

type ReleaseEipAddressRequest struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	Region         string `json:"region"`
	AllocationId   string `json:"allocation_id"`
}

// 释放EIP
func ReleaseEipAddress(req *ReleaseEipAddressRequest) error {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, req.Region)
	if err != nil {
		return err
	}
	logx.Infof("释放EIP,RegionId:%s,AllocationId:%s", req.Region, req.AllocationId)
	request := &vpc20160428.ReleaseEipAddressRequest{
		RegionId:     tea.String(req.Region),
		AllocationId: tea.String(req.AllocationId),
	}
	response, err := client.ReleaseEipAddress(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type ModifyEipAddressAttributeRequest struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	Region         string `json:"region"`
	AllocationId   string `json:"allocation_id"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Bandwidth      string `json:"bandwidth"` // 带宽峰值
}

// 修改EIP属性
func ModifyEipAddressAttribute(req *ModifyEipAddressAttributeRequest) error {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, req.Region)
	if err != nil {
		return err
	}
	request := &vpc20160428.ModifyEipAddressAttributeRequest{
		RegionId:     tea.String(req.Region),
		AllocationId: tea.String(req.AllocationId),
		Name:         tea.String(req.Name),
		Description:  tea.String(req.Description),
		Bandwidth:    tea.String(req.Bandwidth),
	}
	response, err := client.ModifyEipAddressAttribute(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type DescribeEipAddressesRequest struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	Region         string `json:"region"`
}

// DescribeEipAddressByAllocationIdRequest 按AllocationId查询单个EIP请求
type DescribeEipAddressByAllocationIdRequest struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	Region         string `json:"region"`
	AllocationId   string `json:"allocation_id"`
}

// DescribeEipAddressByAllocationId 按AllocationId查询单个EIP详情
func DescribeEipAddressByAllocationId(req *DescribeEipAddressByAllocationIdRequest) (*vpc20160428.DescribeEipAddressesResponseBodyEipAddressesEipAddress, error) {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return nil, err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, req.Region)
	if err != nil {
		return nil, err
	}
	request := &vpc20160428.DescribeEipAddressesRequest{
		RegionId:     tea.String(req.Region),
		AllocationId: tea.String(req.AllocationId),
	}
	response, err := client.DescribeEipAddresses(request)
	if err != nil {
		return nil, err
	}
	if *response.StatusCode != 200 {
		return nil, errors.New(response.String())
	}
	if len(response.Body.EipAddresses.EipAddress) == 0 {
		return nil, errors.New("EIP不存在")
	}
	return response.Body.EipAddresses.EipAddress[0], nil
}

// DescribeEipAddresses 查询EIP列表
func DescribeEipAddresses(req *DescribeEipAddressesRequest) ([]*vpc20160428.DescribeEipAddressesResponseBodyEipAddressesEipAddress, error) {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return nil, err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, req.Region)
	if err != nil {
		return nil, err
	}
	request := &vpc20160428.DescribeEipAddressesRequest{
		RegionId:   tea.String(req.Region),
		PageNumber: tea.Int32(1),
		PageSize:   tea.Int32(100),
	}
	response, err := client.DescribeEipAddresses(request)
	if err != nil {
		return nil, err
	}
	if *response.StatusCode != 200 {
		return nil, errors.New(response.String())
	}
	return response.Body.EipAddresses.EipAddress, nil
}

type DescribeEipAddressesByCloudAccountRequest struct {
	CloudAccountId int64  `json:"cloud_account_id"`
	Region         string `json:"region"`
}

// DescribeEipAddressesByCloudAccount 使用系统云账号查询EIP列表
func DescribeEipAddressesByCloudAccount(req *DescribeEipAddressesByCloudAccountRequest) ([]*vpc20160428.DescribeEipAddressesResponseBodyEipAddressesEipAddress, error) {
	cloud, err := GetSystemCloudAccount(req.CloudAccountId)
	if err != nil {
		return nil, err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, req.Region)
	if err != nil {
		return nil, err
	}
	request := &vpc20160428.DescribeEipAddressesRequest{
		RegionId:   tea.String(req.Region),
		PageNumber: tea.Int32(1),
		PageSize:   tea.Int32(100),
	}
	response, err := client.DescribeEipAddresses(request)
	if err != nil {
		return nil, err
	}
	if *response.StatusCode != 200 {
		return nil, errors.New(response.String())
	}
	return response.Body.EipAddresses.EipAddress, nil
}

type AssociateEipAddressRequest struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	Region         string `json:"region"`
	AllocationId   string `json:"allocation_id"` // EIP的实例id
	InstanceType   string `json:"instance_type"` // ecs或者弹性网卡 EcsInstance, NetworkInterface
	InstanceId     string `json:"instance_id"`   // ecs或者弹性网卡的实例id
}

// AssociateEipAddress 绑定eip到ecs实例或者弹性网卡
func AssociateEipAddress(req *AssociateEipAddressRequest) error {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, req.Region)
	if err != nil {
		return err
	}
	request := &vpc20160428.AssociateEipAddressRequest{
		RegionId:     tea.String(req.Region),
		AllocationId: tea.String(req.AllocationId),
		InstanceType: tea.String(req.InstanceType),
		InstanceId:   tea.String(req.InstanceId),
	}
	response, err := client.AssociateEipAddress(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type UnassociateEipAddressRequest struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	Region         string `json:"region"`
	AllocationId   string `json:"allocation_id"` // EIP的实例id
	InstanceType   string `json:"instance_type"` // ecs或者弹性网卡 EcsInstance, NetworkInterface
	InstanceId     string `json:"instance_id"`   // ecs或者弹性网卡的实例id
}

// UnassociateEipAddress 从实例解绑eip
func UnassociateEipAddress(req *UnassociateEipAddressRequest) error {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, req.Region)
	if err != nil {
		return err
	}
	request := &vpc20160428.UnassociateEipAddressRequest{
		RegionId:     tea.String(req.Region),
		AllocationId: tea.String(req.AllocationId),
		InstanceType: tea.String(req.InstanceType),
		InstanceId:   tea.String(req.InstanceId),
	}
	response, err := client.UnassociateEipAddress(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}
