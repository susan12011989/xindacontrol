package tencent

import (
	"fmt"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common/profile"
	vpc "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/vpc/v20170312"
)

// NewVpcClient 创建腾讯云 VPC 客户端（安全组 API 在 VPC 命名空间下）
func NewVpcClient(accessKey, accessSecret, regionId string) (*vpc.Client, error) {
	credential := common.NewCredential(accessKey, accessSecret)
	cpf := profile.NewClientProfile()
	cpf.HttpProfile.Endpoint = "vpc.tencentcloudapi.com"
	return vpc.NewClient(credential, regionId, cpf)
}

// DescribeSecurityGroups 查询安全组列表
func DescribeSecurityGroups(cred *CloudAccountInfo, regionId string) ([]*vpc.SecurityGroup, error) {
	client, err := NewVpcClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return nil, fmt.Errorf("创建 VPC 客户端失败: %v", err)
	}

	request := vpc.NewDescribeSecurityGroupsRequest()
	request.Limit = common.StringPtr("100")

	response, err := client.DescribeSecurityGroups(request)
	if err != nil {
		return nil, fmt.Errorf("查询安全组失败: %v", err)
	}

	return response.Response.SecurityGroupSet, nil
}

// DescribeSecurityGroupPolicies 查询安全组规则
func DescribeSecurityGroupPolicies(cred *CloudAccountInfo, regionId, securityGroupId string) (*vpc.SecurityGroupPolicySet, error) {
	client, err := NewVpcClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return nil, fmt.Errorf("创建 VPC 客户端失败: %v", err)
	}

	request := vpc.NewDescribeSecurityGroupPoliciesRequest()
	request.SecurityGroupId = common.StringPtr(securityGroupId)

	response, err := client.DescribeSecurityGroupPolicies(request)
	if err != nil {
		return nil, fmt.Errorf("查询安全组规则失败: %v", err)
	}

	return response.Response.SecurityGroupPolicySet, nil
}

// SecurityGroupRuleInput 安全组规则输入
type SecurityGroupRuleInput struct {
	Protocol    string // TCP, UDP, ICMP, ALL
	Port        string // 如 "80", "8000-9000"
	CidrBlock   string // 如 "0.0.0.0/0"
	Action      string // ACCEPT, DROP
	Description string
}

// CreateSecurityGroupIngress 添加入站规则
func CreateSecurityGroupIngress(cred *CloudAccountInfo, regionId, securityGroupId string, rules []SecurityGroupRuleInput) error {
	client, err := NewVpcClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建 VPC 客户端失败: %v", err)
	}

	var policies []*vpc.SecurityGroupPolicy
	for _, r := range rules {
		policies = append(policies, &vpc.SecurityGroupPolicy{
			Protocol:          common.StringPtr(r.Protocol),
			Port:              common.StringPtr(r.Port),
			CidrBlock:         common.StringPtr(r.CidrBlock),
			Action:            common.StringPtr(r.Action),
			PolicyDescription: common.StringPtr(r.Description),
		})
	}

	request := vpc.NewCreateSecurityGroupPoliciesRequest()
	request.SecurityGroupId = common.StringPtr(securityGroupId)
	request.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
		Ingress: policies,
	}

	_, err = client.CreateSecurityGroupPolicies(request)
	if err != nil {
		return fmt.Errorf("添加入站规则失败: %v", err)
	}
	return nil
}

// DeleteSecurityGroupIngress 删除入站规则（按 PolicyIndex）
func DeleteSecurityGroupIngress(cred *CloudAccountInfo, regionId, securityGroupId string, policyIndexes []int64) error {
	client, err := NewVpcClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return fmt.Errorf("创建 VPC 客户端失败: %v", err)
	}

	var policies []*vpc.SecurityGroupPolicy
	for _, idx := range policyIndexes {
		policies = append(policies, &vpc.SecurityGroupPolicy{
			PolicyIndex: common.Int64Ptr(idx),
		})
	}

	request := vpc.NewDeleteSecurityGroupPoliciesRequest()
	request.SecurityGroupId = common.StringPtr(securityGroupId)
	request.SecurityGroupPolicySet = &vpc.SecurityGroupPolicySet{
		Ingress: policies,
	}

	_, err = client.DeleteSecurityGroupPolicies(request)
	if err != nil {
		return fmt.Errorf("删除入站规则失败: %v", err)
	}
	return nil
}

// DescribeVpcs 查询 VPC 列表
func DescribeVpcs(cred *CloudAccountInfo, regionId string) ([]*vpc.Vpc, error) {
	client, err := NewVpcClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return nil, fmt.Errorf("创建 VPC 客户端失败: %v", err)
	}

	request := vpc.NewDescribeVpcsRequest()
	request.Limit = common.StringPtr("100")

	response, err := client.DescribeVpcs(request)
	if err != nil {
		return nil, fmt.Errorf("查询 VPC 失败: %v", err)
	}

	return response.Response.VpcSet, nil
}

// DescribeSubnets 查询子网列表
func DescribeSubnets(cred *CloudAccountInfo, regionId, vpcId string) ([]*vpc.Subnet, error) {
	client, err := NewVpcClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return nil, fmt.Errorf("创建 VPC 客户端失败: %v", err)
	}

	request := vpc.NewDescribeSubnetsRequest()
	request.Limit = common.StringPtr("100")
	if vpcId != "" {
		request.Filters = []*vpc.Filter{
			{
				Name:   common.StringPtr("vpc-id"),
				Values: common.StringPtrs([]string{vpcId}),
			},
		}
	}

	response, err := client.DescribeSubnets(request)
	if err != nil {
		return nil, fmt.Errorf("查询子网失败: %v", err)
	}

	return response.Response.SubnetSet, nil
}

// CreateSecurityGroup 创建安全组
func CreateSecurityGroupNew(cred *CloudAccountInfo, regionId, name, description string) (string, error) {
	client, err := NewVpcClient(cred.AccessKey, cred.AccessSecret, regionId)
	if err != nil {
		return "", fmt.Errorf("创建 VPC 客户端失败: %v", err)
	}

	request := vpc.NewCreateSecurityGroupRequest()
	request.GroupName = common.StringPtr(name)
	request.GroupDescription = common.StringPtr(description)

	response, err := client.CreateSecurityGroup(request)
	if err != nil {
		return "", fmt.Errorf("创建安全组失败: %v", err)
	}

	return *response.Response.SecurityGroup.SecurityGroupId, nil
}
