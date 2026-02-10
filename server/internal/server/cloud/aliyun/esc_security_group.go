package aliyun

import (
	"errors"
	"fmt"
	"time"

	ecs20140526 "github.com/alibabacloud-go/ecs-20140526/v6/client"
	"github.com/alibabacloud-go/tea/tea"
)

type CreateSecurityGroupRequest struct {
	MerchantId     int    `json:"merchant_id"`
	CloudAccountId int64  `json:"cloud_account_id"`
	RegionId       string `json:"region_id"`
	Name           string `json:"name"`        // 安全组名
	Description    string `json:"description"` // 描述
}

// CreateSecurityGroup 创建安全组,返回安全组id
func CreateSecurityGroup(req *CreateSecurityGroupRequest) (string, error) {
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
	if req.Name == "" {
		req.Name = fmt.Sprintf("sg-%s", time.Now().Format("20060102150405"))
	}
	request := &ecs20140526.CreateSecurityGroupRequest{
		RegionId:          tea.String(req.RegionId),
		SecurityGroupName: tea.String(req.Name),
	}
	if req.Description != "" {
		request.Description = tea.String(req.Description)
	}
	response, err := client.CreateSecurityGroup(request)
	if err != nil {
		return "", err
	}
	if *response.StatusCode != 200 {
		return "", errors.New(response.String())
	}
	return *response.Body.SecurityGroupId, nil
}

type DescribeSecurityGroupAttributesRequest struct {
	MerchantId      int    `json:"merchant_id"`
	RegionId        string `json:"region_id"`
	SecurityGroupId string `json:"security_group_id"`
}

// DescribeSecurityGroupAttributes 查询安全组属性
func DescribeSecurityGroupAttributes(req *DescribeSecurityGroupAttributesRequest) (*ecs20140526.DescribeSecurityGroupAttributeResponseBody, error) {
	cloud, err := GetMerchantCloud(req.MerchantId)
	if err != nil {
		return nil, err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return nil, err
	}
	request := &ecs20140526.DescribeSecurityGroupAttributeRequest{
		RegionId:        tea.String(req.RegionId),
		SecurityGroupId: tea.String(req.SecurityGroupId),
	}
	response, err := client.DescribeSecurityGroupAttribute(request)
	if err != nil {
		return nil, err
	}
	if *response.StatusCode != 200 {
		return nil, errors.New(response.String())
	}
	return response.Body, nil
}

type DeleteSecurityGroupRequest struct {
	MerchantId      int    `json:"merchant_id"`
	CloudAccountId  int64  `json:"cloud_account_id"`
	RegionId        string `json:"region_id"`
	SecurityGroupId string `json:"security_group_id"`
}

// DeleteSecurityGroup 删除安全组
func DeleteSecurityGroup(req *DeleteSecurityGroupRequest) error {
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
	request := &ecs20140526.DeleteSecurityGroupRequest{
		RegionId:        tea.String(req.RegionId),
		SecurityGroupId: tea.String(req.SecurityGroupId),
	}
	response, err := client.DeleteSecurityGroup(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type AuthorizeSecurityGroupRequest struct {
	MerchantId      int                                                     `json:"merchant_id"`
	CloudAccountId  int64                                                   `json:"cloud_account_id"`
	RegionId        string                                                  `json:"region_id"`
	SecurityGroupId string                                                  `json:"security_group_id"`
	Permissions     []*ecs20140526.AuthorizeSecurityGroupRequestPermissions `json:"permissions"`
}

// AuthorizeSecurityGroup 增加安全组入方向规则
func AuthorizeSecurityGroup(req *AuthorizeSecurityGroupRequest) error {
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
	request := &ecs20140526.AuthorizeSecurityGroupRequest{
		RegionId:        tea.String(req.RegionId),
		SecurityGroupId: tea.String(req.SecurityGroupId),
		Permissions:     req.Permissions,
	}

	response, err := client.AuthorizeSecurityGroup(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type RevokeSecurityGroupRequest struct {
	MerchantId      int                                                  `json:"merchant_id"`
	CloudAccountId  int64                                                `json:"cloud_account_id"`
	RegionId        string                                               `json:"region_id"`
	SecurityGroupId string                                               `json:"security_group_id"`
	Permissions     []*ecs20140526.RevokeSecurityGroupRequestPermissions `json:"permissions"`
}

// RevokeSecurityGroup 删除安全组入方向规则
func RevokeSecurityGroup(req *RevokeSecurityGroupRequest) error {
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
	request := &ecs20140526.RevokeSecurityGroupRequest{
		RegionId:        tea.String(req.RegionId),
		SecurityGroupId: tea.String(req.SecurityGroupId),
		Permissions:     req.Permissions,
	}
	response, err := client.RevokeSecurityGroup(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type JoinSecurityGroupRequest struct {
	MerchantId         int    `json:"merchant_id"`
	RegionId           string `json:"region_id"`
	SecurityGroupId    string `json:"security_group_id"`
	InstanceId         string `json:"instance_id"`          // ecs实例ID
	NetworkInterfaceId string `json:"network_interface_id"` // 弹性网卡ID
}

// JoinSecurityGroup 将实例或弹性网卡加入安全组
func JoinSecurityGroup(req *JoinSecurityGroupRequest) error {
	cloud, err := GetMerchantCloud(req.MerchantId)
	if err != nil {
		return err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return err
	}
	request := &ecs20140526.JoinSecurityGroupRequest{
		RegionId:        tea.String(req.RegionId),
		SecurityGroupId: tea.String(req.SecurityGroupId),
	}
	if req.InstanceId != "" {
		request.InstanceId = tea.String(req.InstanceId)
	}
	if req.NetworkInterfaceId != "" {
		request.NetworkInterfaceId = tea.String(req.NetworkInterfaceId)
	}
	response, err := client.JoinSecurityGroup(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

type LeaveSecurityGroupRequest struct {
	MerchantId         int    `json:"merchant_id"`
	RegionId           string `json:"region_id"`
	SecurityGroupId    string `json:"security_group_id"`
	InstanceId         string `json:"instance_id"`          // ecs实例ID
	NetworkInterfaceId string `json:"network_interface_id"` // 弹性网卡ID
}

// LeaveSecurityGroup 将实例或弹性网卡从安全组移除
func LeaveSecurityGroup(req *LeaveSecurityGroupRequest) error {
	cloud, err := GetMerchantCloud(req.MerchantId)
	if err != nil {
		return err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, req.RegionId)
	if err != nil {
		return err
	}
	request := &ecs20140526.LeaveSecurityGroupRequest{
		RegionId:        tea.String(req.RegionId),
		SecurityGroupId: tea.String(req.SecurityGroupId),
	}
	if req.InstanceId != "" {
		request.InstanceId = tea.String(req.InstanceId)
	}
	if req.NetworkInterfaceId != "" {
		request.NetworkInterfaceId = tea.String(req.NetworkInterfaceId)
	}
	response, err := client.LeaveSecurityGroup(request)
	if err != nil {
		return err
	}
	if *response.StatusCode != 200 {
		return errors.New(response.String())
	}
	return nil
}

// DescribeSecurityGroups 查看安全组列表
func DescribeSecurityGroups(merchantId int, region string) ([]*ecs20140526.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup, error) {
	cloud, err := GetMerchantCloud(merchantId)
	if err != nil {
		return nil, err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return nil, err
	}
	request := &ecs20140526.DescribeSecurityGroupsRequest{
		RegionId: tea.String(region),
	}
	response, err := client.DescribeSecurityGroups(request)
	if err != nil {
		return nil, err
	}
	return response.Body.SecurityGroups.SecurityGroup, nil
}

// DescribeSecurityGroupsByCloudAccount 使用系统云账号查询安全组列表
func DescribeSecurityGroupsByCloudAccount(cloudAccountId int64, region string) ([]*ecs20140526.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup, error) {
	cloud, err := GetSystemCloudAccount(cloudAccountId)
	if err != nil {
		return nil, err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, region)
	if err != nil {
		return nil, err
	}
	request := &ecs20140526.DescribeSecurityGroupsRequest{
		RegionId: tea.String(region),
	}
	response, err := client.DescribeSecurityGroups(request)
	if err != nil {
		return nil, err
	}
	return response.Body.SecurityGroups.SecurityGroup, nil
}

// DescribeSecurityGroupAttribute 查询安全组详情
func DescribeSecurityGroupAttribute(merchantId int, regionId, securityGroupId string) (*ecs20140526.DescribeSecurityGroupAttributeResponseBody, error) {
	cloud, err := GetMerchantCloud(merchantId)
	if err != nil {
		return nil, err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, regionId)
	if err != nil {
		return nil, err
	}
	request := &ecs20140526.DescribeSecurityGroupAttributeRequest{
		RegionId:        tea.String(regionId),
		SecurityGroupId: tea.String(securityGroupId),
	}
	response, err := client.DescribeSecurityGroupAttribute(request)
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}

func DescribeSecurityGroupAttributeByCloudAccount(cloudAccountId int64, regionId, securityGroupId string) (*ecs20140526.DescribeSecurityGroupAttributeResponseBody, error) {
	cloud, err := GetSystemCloudAccount(cloudAccountId)
	if err != nil {
		return nil, err
	}
	client, err := NewEcsClient(cloud.AccessKey, cloud.AccessSecret, regionId)
	if err != nil {
		return nil, err
	}
	request := &ecs20140526.DescribeSecurityGroupAttributeRequest{
		RegionId:        tea.String(regionId),
		SecurityGroupId: tea.String(securityGroupId),
	}
	response, err := client.DescribeSecurityGroupAttribute(request)
	if err != nil {
		return nil, err
	}
	return response.Body, nil
}
