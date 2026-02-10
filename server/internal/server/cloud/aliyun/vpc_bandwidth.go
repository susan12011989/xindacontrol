package aliyun

import (
	"errors"
	"fmt"
	"server/internal/dbhelper"

	"github.com/alibabacloud-go/tea/tea"
	vpc20160428 "github.com/alibabacloud-go/vpc-20160428/v6/client"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateCommonBandwidthPackageRequest struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id"`
	Bandwidth      int32  `json:"bandwidth"`
}

// CreateCommonBandwidthPackage 创建共享带宽
func CreateCommonBandwidthPackage(req *CreateCommonBandwidthPackageRequest) (string, error) {
	var cloud *CloudAccountInfo
	var err error

	if req.CloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(req.CloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(req.MerchantId)
	}
	if err != nil {
		return "", err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return "", err
	}
	var name string
	if req.MerchantId > 0 {
		merchant, _ := dbhelper.GetMerchantByID(req.MerchantId)
		name = fmt.Sprintf("%s_%s", merchant.Name, req.RegionId)
	} else {
		name = fmt.Sprintf("system_%s", req.RegionId)
	}
	logx.Infof("创建共享带宽 %d %s", req.Bandwidth, name)
	request := &vpc20160428.CreateCommonBandwidthPackageRequest{
		RegionId:           tea.String(req.RegionId),
		Bandwidth:          tea.Int32(req.Bandwidth),
		Name:               tea.String(name),
		InternetChargeType: tea.String("PayByTraffic"),
	}
	response, err := client.CreateCommonBandwidthPackage(request)
	if err != nil {
		return "", err
	}
	bandwidthPackageId := *response.Body.BandwidthPackageId
	return bandwidthPackageId, nil
}

type AddCommonBandwidthPackageIpsRequest struct {
	MerchantId         int      `json:"merchant_id"`
	CloudAccountId     int64    `json:"cloud_account_id"`
	Region             string   `json:"region"`
	BandwidthPackageId string   `json:"bandwidth_package_id"`
	IpInstanceIds      []string `json:"ip_instance_ids"`
}

// AddCommonBandwidthPackageIps 批量添加eip到共享带宽中
func AddCommonBandwidthPackageIps(req *AddCommonBandwidthPackageIpsRequest) error {
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
	request := &vpc20160428.AddCommonBandwidthPackageIpsRequest{
		RegionId:           tea.String(req.Region),
		BandwidthPackageId: tea.String(req.BandwidthPackageId),
		IpType:             tea.String("EIP"),
		IpInstanceIds:      tea.StringSlice(req.IpInstanceIds),
	}
	response, err := client.AddCommonBandwidthPackageIps(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type RemoveCommonBandwidthPackageIpRequest struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	Region             string `json:"region"`
	BandwidthPackageId string `json:"bandwidth_package_id"`
	IpInstanceId       string `json:"ip_instance_id"`
}

// RemoveCommonBandwidthPackageIp 移除共享带宽中的eip
func RemoveCommonBandwidthPackageIp(req *RemoveCommonBandwidthPackageIpRequest) error {
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
	request := &vpc20160428.RemoveCommonBandwidthPackageIpRequest{
		RegionId:           tea.String(req.Region),
		BandwidthPackageId: tea.String(req.BandwidthPackageId),
		IpInstanceId:       tea.String(req.IpInstanceId),
	}
	response, err := client.RemoveCommonBandwidthPackageIp(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type DeleteCommonBandwidthPackageRequest struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	Region             string `json:"region"`
	BandwidthPackageId string `json:"bandwidth_package_id"`
}

// DeleteCommonBandwidthPackage 删除共享带宽
func DeleteCommonBandwidthPackage(req *DeleteCommonBandwidthPackageRequest) error {
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
	logx.Infof("删除共享带宽,RegionId:%s,BandwidthPackageId:%s", req.Region, req.BandwidthPackageId)
	request := &vpc20160428.DeleteCommonBandwidthPackageRequest{
		RegionId:           tea.String(req.Region),
		BandwidthPackageId: tea.String(req.BandwidthPackageId),
	}
	response, err := client.DeleteCommonBandwidthPackage(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type ModifyCommonBandwidthPackageAttributeRequest struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	Region             string `json:"region"`
	BandwidthPackageId string `json:"bandwidth_package_id"`
	Name               string `json:"name"`
	Description        string `json:"description"`
}

// ModifyCommonBandwidthPackageAttribute 修改共享带宽属性 name, description
func ModifyCommonBandwidthPackageAttribute(req *ModifyCommonBandwidthPackageAttributeRequest) error {
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
	request := &vpc20160428.ModifyCommonBandwidthPackageAttributeRequest{
		RegionId:           tea.String(req.Region),
		BandwidthPackageId: tea.String(req.BandwidthPackageId),
		Name:               tea.String(req.Name),
		Description:        tea.String(req.Description),
	}
	response, err := client.ModifyCommonBandwidthPackageAttribute(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type ModifyCommonBandwidthPackageSpecRequest struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	Region             string `json:"region"`
	BandwidthPackageId string `json:"bandwidth_package_id"`
	Bandwidth          string `json:"bandwidth"`
}

// ModifyCommonBandwidthPackageSpec 修改共享带宽的带宽峰值
func ModifyCommonBandwidthPackageSpec(req *ModifyCommonBandwidthPackageSpecRequest) error {
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
	request := &vpc20160428.ModifyCommonBandwidthPackageSpecRequest{
		RegionId:           tea.String(req.Region),
		BandwidthPackageId: tea.String(req.BandwidthPackageId),
		Bandwidth:          tea.String(req.Bandwidth),
	}
	response, err := client.ModifyCommonBandwidthPackageSpec(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type ModifyCommonBandwidthPackageIpBandwidthRequest struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	Region             string `json:"region"`
	BandwidthPackageId string `json:"bandwidth_package_id"`
	EipId              string `json:"eip_id"`
	Bandwidth          string `json:"bandwidth"`
}

// ModifyCommonBandwidthPackageIpBandwidth 修改共享带宽中的eip的最大可用带宽值
func ModifyCommonBandwidthPackageIpBandwidth(req *ModifyCommonBandwidthPackageIpBandwidthRequest) error {
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
	request := &vpc20160428.ModifyCommonBandwidthPackageIpBandwidthRequest{
		RegionId:           tea.String(req.Region),
		BandwidthPackageId: tea.String(req.BandwidthPackageId),
		EipId:              tea.String(req.EipId),
		Bandwidth:          tea.String(req.Bandwidth),
	}
	response, err := client.ModifyCommonBandwidthPackageIpBandwidth(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type CancelCommonBandwidthPackageIpBandwidthRequest struct {
	MerchantId         int    `json:"merchant_id"`
	Region             string `json:"region"`
	BandwidthPackageId string `json:"bandwidth_package_id"`
	EipId              string `json:"eip_id"`
}

// CancelCommonBandwidthPackageIpBandwidth 取消共享带宽中的eip的最大可用带宽值
func CancelCommonBandwidthPackageIpBandwidth(req *CancelCommonBandwidthPackageIpBandwidthRequest) error {
	cloud, err := GetMerchantCloud(req.MerchantId)
	if err != nil {
		return err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, req.Region)
	if err != nil {
		return err
	}
	request := &vpc20160428.CancelCommonBandwidthPackageIpBandwidthRequest{
		RegionId:           tea.String(req.Region),
		BandwidthPackageId: tea.String(req.BandwidthPackageId),
		EipId:              tea.String(req.EipId),
	}
	response, err := client.CancelCommonBandwidthPackageIpBandwidth(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

// DescribeCommonBandwidthPackages 查看共享带宽列表
func DescribeCommonBandwidthPackages(merchantId int, region string) ([]*vpc20160428.DescribeCommonBandwidthPackagesResponseBodyCommonBandwidthPackagesCommonBandwidthPackage, error) {
	cloud, err := GetMerchantCloud(merchantId)
	if err != nil {
		return nil, err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return nil, err
	}
	request := &vpc20160428.DescribeCommonBandwidthPackagesRequest{
		RegionId:   tea.String(region),
		PageNumber: tea.Int32(1),
		PageSize:   tea.Int32(10),
	}
	response, err := client.DescribeCommonBandwidthPackages(request)
	if err != nil {
		return nil, err
	}
	return response.Body.CommonBandwidthPackages.CommonBandwidthPackage, nil
}

// DescribeCommonBandwidthPackagesByCloudAccount 使用系统云账号查询带宽包列表
func DescribeCommonBandwidthPackagesByCloudAccount(cloudAccountId int64, region string) ([]*vpc20160428.DescribeCommonBandwidthPackagesResponseBodyCommonBandwidthPackagesCommonBandwidthPackage, error) {
	cloud, err := GetSystemCloudAccount(cloudAccountId)
	if err != nil {
		return nil, err
	}
	client, err := NewVpcEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return nil, err
	}
	request := &vpc20160428.DescribeCommonBandwidthPackagesRequest{
		RegionId:   tea.String(region),
		PageNumber: tea.Int32(1),
		PageSize:   tea.Int32(10),
	}
	response, err := client.DescribeCommonBandwidthPackages(request)
	if err != nil {
		return nil, err
	}
	return response.Body.CommonBandwidthPackages.CommonBandwidthPackage, nil
}
