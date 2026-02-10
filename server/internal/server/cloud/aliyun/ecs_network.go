package aliyun

import (
	"errors"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v6/client"
	"github.com/alibabacloud-go/tea/tea"
)

type CreateNetworkInterfaceRequest struct {
	MerchantId           int    `json:"merchant_id"`
	CloudAccountId       int64  `json:"cloud_account_id"`
	RegionId             string `json:"region_id"`
	VSwitchId            string `json:"vswitch_id"`             // 弹性网卡的交换机ID，从ecs实例信息中获取
	SecurityGroupId      string `json:"security_group_id"`      // 安全组
	NetworkInterfaceName string `json:"network_interface_name"` // 弹性网卡名称
}

// 创建弹性网卡
func CreateNetworkInterface(req *CreateNetworkInterfaceRequest) (string, error) {
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
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return "", err
	}
	request := &ecs20140526.CreateNetworkInterfaceRequest{
		RegionId:             tea.String(req.RegionId),
		VSwitchId:            tea.String(req.VSwitchId),
		NetworkInterfaceName: tea.String(req.NetworkInterfaceName),
	}
	if req.SecurityGroupId != "" {
		request.SecurityGroupId = tea.String(req.SecurityGroupId)
	}
	response, err := client.CreateNetworkInterface(request)
	if err != nil {
		return "", err
	}
	if *response.StatusCode != 200 {
		return "", errors.New(response.String())
	}
	return *response.Body.NetworkInterfaceId, nil
}

type DescribeNetworkInterfaceAttributeRequest struct {
	MerchantId         int    `json:"merchant_id"`
	Region             string `json:"region"`
	NetworkInterfaceId string `json:"network_interface_id"`
}

// DescribeNetworkInterfaceAttribute 获取弹性网卡属性
func DescribeNetworkInterfaceAttribute(req *DescribeNetworkInterfaceAttributeRequest) (*ecs20140526.DescribeNetworkInterfaceAttributeResponseBody, error) {
	cloud, err := GetMerchantCloud(req.MerchantId)
	if err != nil {
		return nil, err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.Region)
	if err != nil {
		return nil, err
	}
	request := &ecs20140526.DescribeNetworkInterfaceAttributeRequest{
		RegionId:           tea.String(req.Region),
		NetworkInterfaceId: tea.String(req.NetworkInterfaceId),
	}
	response, err := client.DescribeNetworkInterfaceAttribute(request)
	if err != nil {
		return nil, err
	}
	if *response.StatusCode != 200 {
		return nil, errors.New(response.String())
	}
	return response.Body, nil
}

type DeleteNetworkInterfaceRequest struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	RegionId           string `json:"region_id"`
	NetworkInterfaceId string `json:"network_interface_id"`
}

// DeleteNetworkInterface 删除弹性网卡
func DeleteNetworkInterface(req *DeleteNetworkInterfaceRequest) error {
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
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return err
	}
	request := &ecs20140526.DeleteNetworkInterfaceRequest{
		RegionId:           tea.String(req.RegionId),
		NetworkInterfaceId: tea.String(req.NetworkInterfaceId),
	}
	response, err := client.DeleteNetworkInterface(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type AttachNetworkInterfaceRequest struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	RegionId           string `json:"region_id"`
	NetworkInterfaceId string `json:"network_interface_id"`
	InstanceId         string `json:"instance_id"` // ECS实例ID
}

// AttachNetworkInterface 附加一个弹性网卡到一台专有网络VPC类型ECS实例上
func AttachNetworkInterface(req *AttachNetworkInterfaceRequest) error {
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
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return err
	}
	request := &ecs20140526.AttachNetworkInterfaceRequest{
		RegionId:           tea.String(req.RegionId),
		NetworkInterfaceId: tea.String(req.NetworkInterfaceId),
		InstanceId:         tea.String(req.InstanceId),
	}
	response, err := client.AttachNetworkInterface(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type DetachNetworkInterfaceRequest struct {
	MerchantId         int    `json:"merchant_id"`
	CloudAccountId     int64  `json:"cloud_account_id"`
	RegionId           string `json:"region_id"`
	NetworkInterfaceId string `json:"network_interface_id"`
	InstanceId         string `json:"instance_id"`
}

// DetachNetworkInterface 从ecs实例解绑弹性网卡
func DetachNetworkInterface(req *DetachNetworkInterfaceRequest) error {
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
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return err
	}
	request := &ecs20140526.DetachNetworkInterfaceRequest{
		RegionId:           tea.String(req.RegionId),
		NetworkInterfaceId: tea.String(req.NetworkInterfaceId),
		InstanceId:         tea.String(req.InstanceId),
	}
	response, err := client.DetachNetworkInterface(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type ModifyNetworkInterfaceAttributeRequest struct {
	MerchantId           int       `json:"merchant_id"`
	CloudAccountId       int64     `json:"cloud_account_id"`
	RegionId             string    `json:"region_id"`
	NetworkInterfaceId   string    `json:"network_interface_id"`
	NetworkInterfaceName string    `json:"network_interface_name"`
	Description          string    `json:"description"`
	SecurityGroupId      []*string `json:"security_group_id"`
}

// ModifyNetworkInterfaceAttribute 修改弹性网卡属性
func ModifyNetworkInterfaceAttribute(req *ModifyNetworkInterfaceAttributeRequest) error {
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
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return err
	}
	request := &ecs20140526.ModifyNetworkInterfaceAttributeRequest{
		RegionId:           tea.String(req.RegionId),
		NetworkInterfaceId: tea.String(req.NetworkInterfaceId),
		SecurityGroupId:    req.SecurityGroupId, // 最终加入的安全组,会移除之前的
	}
	if req.NetworkInterfaceName != "" {
		request.NetworkInterfaceName = tea.String(req.NetworkInterfaceName)
	}
	if req.Description != "" {
		request.Description = tea.String(req.Description)
	}
	response, err := client.ModifyNetworkInterfaceAttribute(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

// DescribeNetworkInterfaces 查看弹性网卡列表
func DescribeNetworkInterfaces(cloudAccountId int64, merchantId int, region string) ([]*ecs20140526.DescribeNetworkInterfacesResponseBodyNetworkInterfaceSetsNetworkInterfaceSet, error) {
	var cloud *CloudAccountInfo
	var err error
	if cloudAccountId > 0 {
		cloud, err = GetSystemCloudAccount(cloudAccountId)
	} else {
		cloud, err = GetMerchantCloud(merchantId)
	}
	if err != nil {
		return nil, err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return nil, err
	}
	request := &ecs20140526.DescribeNetworkInterfacesRequest{
		RegionId: tea.String(region),
	}
	response, err := client.DescribeNetworkInterfaces(request)
	if err != nil {
		return nil, err
	}
	return response.Body.NetworkInterfaceSets.NetworkInterfaceSet, nil
}
