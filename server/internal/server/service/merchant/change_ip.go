package merchant

import (
	"context"
	"errors"
	"fmt"
	"time"

	awscloud "server/internal/server/cloud/aws"
	"server/internal/server/model"
	cloudAwsSvc "server/internal/server/service/cloud_aws"
	"server/pkg/dbs"
	"server/pkg/entity"
	"server/pkg/gostapi"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/zeromicro/go-zero/core/logx"
	"xorm.io/xorm"
)

// 获取实例公网IP（兼容 PublicIpAddress 或 ENI 关联的 PublicIp）
func resolveInstancePublicIP(ins ec2types.Instance) string {
	if ins.PublicIpAddress != nil && *ins.PublicIpAddress != "" {
		return *ins.PublicIpAddress
	}
	for _, ni := range ins.NetworkInterfaces {
		if ni.Association != nil && ni.Association.PublicIp != nil && *ni.Association.PublicIp != "" {
			return *ni.Association.PublicIp
		}
	}
	return ""
}

// findInstanceByPublicIP 在给定区域列表中查找绑定指定公网IP的实例
// 返回：region, instanceId, oldAllocationId(EIP时返回)，未找到返回错误
func findInstanceByPublicIP(ctx context.Context, acc *entity.CloudAccounts, regions []string, oldIP string) (string, string, string, error) {
	var targetRegion, instanceId, oldAllocationId string
	for _, region := range regions {
		cli, err := awscloud.NewEc2Client(ctx, acc, region)
		if err != nil {
			logx.Errorf("change-ip: create ec2 client failed, region=%s err=%+v", region, err)
			continue
		}
		// 先按EIP精确匹配
		addrOut, aerr := cli.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{PublicIps: []string{oldIP}})
		if aerr != nil {
			logx.Infof("change-ip: DescribeAddresses err, region=%s ip=%s err=%+v", region, oldIP, aerr)
		} else {
			logx.Infof("change-ip: DescribeAddresses ok, region=%s ip=%s count=%d", region, oldIP, len(addrOut.Addresses))
		}
		if aerr == nil && addrOut != nil && len(addrOut.Addresses) > 0 {
			a := addrOut.Addresses[0]
			targetRegion = region
			if a.AllocationId != nil {
				oldAllocationId = *a.AllocationId
			}
			if a.InstanceId != nil {
				instanceId = *a.InstanceId
			} else if a.NetworkInterfaceId != nil {
				eniOut, eniErr := cli.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{NetworkInterfaceIds: []string{*a.NetworkInterfaceId}})
				if eniErr == nil && len(eniOut.NetworkInterfaces) > 0 && eniOut.NetworkInterfaces[0].Attachment != nil && eniOut.NetworkInterfaces[0].Attachment.InstanceId != nil {
					instanceId = *eniOut.NetworkInterfaces[0].Attachment.InstanceId
				}
				logx.Infof("change-ip: eni lookup by allocation, region=%s eni_found=%t", region, instanceId != "")
			}
		}
		// 若未命中EIP，按实例临时公网IP匹配（尝试两类过滤名）
		if instanceId == "" {
			name1 := "ip-address"
			insOut1, ierr1 := cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: []ec2types.Filter{{Name: &name1, Values: []string{oldIP}}}})
			if ierr1 == nil && insOut1 != nil && len(insOut1.Reservations) > 0 && len(insOut1.Reservations[0].Instances) > 0 {
				targetRegion = region
				ins := insOut1.Reservations[0].Instances[0]
				if ins.InstanceId != nil {
					instanceId = *ins.InstanceId
				}
				logx.Infof("change-ip: DescribeInstances by ip-address matched, region=%s instance=%s", region, instanceId)
			} else {
				name2 := "network-interface.addresses.association.public-ip"
				insOut2, ierr2 := cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{Filters: []ec2types.Filter{{Name: &name2, Values: []string{oldIP}}}})
				if ierr2 == nil && insOut2 != nil && len(insOut2.Reservations) > 0 && len(insOut2.Reservations[0].Instances) > 0 {
					targetRegion = region
					ins := insOut2.Reservations[0].Instances[0]
					if ins.InstanceId != nil {
						instanceId = *ins.InstanceId
					}
					logx.Infof("change-ip: DescribeInstances by network-interface.* matched, region=%s instance=%s", region, instanceId)
				} else {
					// 最后按 ENI 查询
					f1 := "addresses.association.public-ip"
					eniOut, eniErr := cli.DescribeNetworkInterfaces(ctx, &ec2.DescribeNetworkInterfacesInput{Filters: []ec2types.Filter{{Name: &f1, Values: []string{oldIP}}}})
					if eniErr == nil && eniOut != nil && len(eniOut.NetworkInterfaces) > 0 {
						eni := eniOut.NetworkInterfaces[0]
						if eni.Attachment != nil && eni.Attachment.InstanceId != nil {
							targetRegion = region
							instanceId = *eni.Attachment.InstanceId
						}
						logx.Infof("change-ip: ENI matched, region=%s instance=%s", region, instanceId)
					} else if eniErr != nil {
						logx.Infof("change-ip: DescribeNetworkInterfaces err, region=%s ip=%s err=%+v", region, oldIP, eniErr)
					}
					// 兜底：全量遍历该区域实例列表匹配公网IP
					if instanceId == "" {
						pager := ec2.NewDescribeInstancesPaginator(cli, &ec2.DescribeInstancesInput{})
						scanned := 0
						for pager.HasMorePages() {
							page, perr := pager.NextPage(ctx)
							if perr != nil {
								logx.Infof("change-ip: paginator error, region=%s err=%+v", region, perr)
								break
							}
							for _, r := range page.Reservations {
								for _, ins := range r.Instances {
									logx.Infof("change-ip: ins: %+v", ins)
									scanned++
									if resolveInstancePublicIP(ins) == oldIP {
										if ins.InstanceId != nil {
											instanceId = *ins.InstanceId
											targetRegion = region
											logx.Infof("change-ip: matched via full-scan, region=%s instance=%s scanned=%d", region, instanceId, scanned)
											break
										}
									}
								}
								if instanceId != "" {
									break
								}
							}
							if instanceId != "" {
								break
							}
						}
						if instanceId == "" {
							logx.Infof("change-ip: full-scan no match, region=%s scanned=%d %s", region, scanned, oldIP)
						}
					}
				}
			}
		}
		if instanceId != "" {
			break
		}
	}
	if instanceId == "" || targetRegion == "" {
		return "", "", "", errors.New("未在香港/新加坡找到对应实例")
	}
	return targetRegion, instanceId, oldAllocationId, nil
}

// ChangeMerchantIP 为商户更换公网IP（AWS）：
// 1) 定位实例（EIP或临时公网IP）于香港/新加坡
// 2) 申请新EIP并关联到实例
// 3) 等待生效，更新数据库（merchants.server_ip 与 servers.host），并联动gost
func ChangeMerchantIP(merchantId int) (model.ChangeMerchantIPResp, error) {
	var resp model.ChangeMerchantIPResp

	// 查询商户
	var m entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", merchantId).Get(&m)
	if err != nil {
		return resp, fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return resp, errors.New("商户不存在")
	}
	oldIP := m.ServerIP
	if oldIP == "" {
		return resp, errors.New("商户当前IP为空")
	}

	// 获取商户AWS账号
	acc, err := awscloud.ResolveAwsAccount(context.Background(), merchantId, 0)
	if err != nil {
		return resp, fmt.Errorf("获取商户AWS账号失败: %v", err)
	}
	logx.Infof("acc: %+v", acc)

	// 在香港/新加坡定位实例
	regions := []string{"ap-east-1", "ap-southeast-1"}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	targetRegion, instanceId, oldAllocationId, err := findInstanceByPublicIP(ctx, acc, regions, oldIP)
	logx.Infof("instanceId: %s, targetRegion: %s", instanceId, targetRegion)
	if err != nil {
		return resp, err
	}

	// 申请新EIP
	cli, err := awscloud.NewEc2Client(ctx, acc, targetRegion)
	if err != nil {
		return resp, fmt.Errorf("创建EC2客户端失败: %v", err)
	}
	allocOut, err := cli.AllocateAddress(ctx, &ec2.AllocateAddressInput{Domain: ec2types.DomainTypeVpc})
	if err != nil || allocOut.AllocationId == nil {
		return resp, fmt.Errorf("申请EIP失败: %v", err)
	}
	newAllocationId := *allocOut.AllocationId

	// 若旧IP为EIP，先解绑旧EIP
	if oldAllocationId != "" {
		addrs, derr := cli.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{AllocationIds: []string{oldAllocationId}})
		if derr == nil && len(addrs.Addresses) > 0 && addrs.Addresses[0].AssociationId != nil {
			_, _ = cli.DisassociateAddress(ctx, &ec2.DisassociateAddressInput{AssociationId: addrs.Addresses[0].AssociationId})
		}
	}

	// 绑定新EIP到实例（默认绑定实例的主私网IP）
	_, err = cli.AssociateAddress(ctx, &ec2.AssociateAddressInput{AllocationId: &newAllocationId, InstanceId: &instanceId})
	if err != nil {
		// 失败释放新EIP
		_, _ = cli.ReleaseAddress(ctx, &ec2.ReleaseAddressInput{AllocationId: &newAllocationId})
		return resp, fmt.Errorf("绑定EIP失败: %v", err)
	}

	// 获取新公网IP字符串
	var newIP string
	if addrInfo, derr := cli.DescribeAddresses(ctx, &ec2.DescribeAddressesInput{AllocationIds: []string{newAllocationId}}); derr == nil && len(addrInfo.Addresses) > 0 && addrInfo.Addresses[0].PublicIp != nil {
		newIP = *addrInfo.Addresses[0].PublicIp
	}
	if newIP == "" {
		// 兜底等待实例上报新IP
		newIP = waitInstancePublicIP(ctx, cli, instanceId, 10, 3*time.Second)
	}
	if newIP == "" {
		return resp, errors.New("未获取到新公网IP")
	}

	// 等待实例PublicIp变为新IP（最多重试）
	if ok := waitInstancePublicIPMatch(ctx, cli, instanceId, newIP, 10, 3*time.Second); !ok {
		logx.Errorf("new public ip not reflected in instance describe in time, instance=%s, ip=%s", instanceId, newIP)
	}

	// 更新数据库并联动（事务内写库，事务外联动）
	if err := dbs.DBAdmin.WithTx(func(s *xorm.Session) error {
		// 更新 merchants.server_ip（使用 Where 绑定主键，避免 ID 与 Table 组合导致的引用错误）
		_, err := s.Table("merchants").Where("id = ?", m.Id).Update(map[string]any{
			"server_ip":  newIP,
			"updated_at": time.Now(),
		})
		if err != nil {
			return err
		}
		// 同步 servers.host（商户服务器）
		_, err = s.Table("servers").Where("server_type = ? AND merchant_id = ?", 1, m.Id).Update(map[string]any{
			"host":       newIP,
			"updated_at": time.Now(),
		})
		return err
	}); err != nil {
		return resp, err
	}

	// 事务外联动：更新所有系统服务器gost转发目标
	onMerchantServerIPChanged(m.Id, m.Port, newIP)

	resp = model.ChangeMerchantIPResp{
		OldIP:         oldIP,
		NewIP:         newIP,
		Region:        targetRegion,
		InstanceId:    instanceId,
		OldAllocation: oldAllocationId,
		NewAllocation: newAllocationId,
	}
	return resp, nil
}

// ChangeMerchantGostPort 为商户更换 GOST 转发端口（基础端口）：
// 会同时处理 basePort, basePort+1, basePort+2 三个端口（TCP/WS/HTTP）
// 1) 调用 AWS 安全组开放端口（全端口放行，确保不被拦截）；
// 2) 通过 GOST API 更新商户服务器上的本地转发服务（监听新端口，转发到业务程序端口）；
// 3) 更新所有系统服务器上的 GOST 转发配置（转发目标改为新端口）。
func ChangeMerchantGostPort(merchantId int, newPort int) (model.ChangeGostPortResp, error) {
	var resp model.ChangeGostPortResp

	if newPort < 1 {
		return resp, fmt.Errorf("gost端口不能小于1")
	}
	if newPort > 65533 { // 65533 因为需要 +2
		return resp, fmt.Errorf("gost端口不能大于65533（需要占用连续3个端口）")
	}

	// 查询商户基本信息
	var m entity.Merchants
	has, err := dbs.DBAdmin.Where("id = ?", merchantId).Get(&m)
	if err != nil {
		return resp, fmt.Errorf("查询商户失败: %v", err)
	}
	if !has {
		return resp, errors.New("商户不存在")
	}
	if m.ServerIP == "" {
		return resp, errors.New("商户当前IP为空")
	}

	// 旧端口使用默认值（标准化后固定为 10000/10001/10002）
	oldPort := gostapi.MerchantGostPortTCP

	// 第一步：AWS 安全组全端口放行，确保不会被拦截
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	acc, err := awscloud.ResolveAwsAccount(ctx, merchantId, 0)
	if err != nil {
		return resp, fmt.Errorf("获取商户AWS账号失败: %v", err)
	}

	regions := []string{"ap-east-1", "ap-southeast-1"}
	targetRegion, instanceId, _, err := findInstanceByPublicIP(ctx, acc, regions, m.ServerIP)
	if err != nil {
		return resp, err
	}

	cli, err := awscloud.NewEc2Client(ctx, acc, targetRegion)
	if err != nil {
		return resp, fmt.Errorf("创建EC2客户端失败: %v", err)
	}

	din, err := cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceId}})
	if err != nil {
		return resp, fmt.Errorf("查询实例失败: %v", err)
	}
	if len(din.Reservations) == 0 || len(din.Reservations[0].Instances) == 0 {
		return resp, errors.New("未找到对应实例")
	}
	ins := din.Reservations[0].Instances[0]
	groupIds := make([]string, 0, len(ins.SecurityGroups))
	for _, sg := range ins.SecurityGroups {
		if sg.GroupId != nil && *sg.GroupId != "" {
			groupIds = append(groupIds, *sg.GroupId)
		}
	}
	if len(groupIds) > 0 {
		if err := cloudAwsSvc.OpenRequiredPortsForSecurityGroups(ctx, cli, groupIds, int32(newPort)); err != nil {
			return resp, fmt.Errorf("安全组开放端口失败: %v", err)
		}
	}

	// 第二步：通过 GOST API 更新商户服务器上的本地转发服务
	// 新端口监听 relay+tls，转发到本地业务程序端口（V2: 10443+10010）
	if err := gostapi.UpdateMerchantLocalForwardsWithCustomPorts(m.ServerIP, newPort, gostapi.MerchantAppPortTCP); err != nil {
		return resp, fmt.Errorf("更新商户服务器GOST配置失败: %v", err)
	}

	// 第三步：更新所有系统服务器上的 GOST 转发配置（监听端口=商户端口；转发目标改为新端口）
	updateGostServicesOnSystemServers(m.Id, m.Port, m.ServerIP, m.TunnelIP, newPort)

	resp = model.ChangeGostPortResp{
		MerchantId: m.Id,
		OldPort:    oldPort,
		NewPort:    newPort,
	}
	return resp, nil
}

func waitInstancePublicIP(ctx context.Context, cli *ec2.Client, instanceId string, tries int, interval time.Duration) string {
	for i := 0; i < tries; i++ {
		out, err := cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceId}})
		if err == nil && len(out.Reservations) > 0 && len(out.Reservations[0].Instances) > 0 {
			ins := out.Reservations[0].Instances[0]
			if ins.PublicIpAddress != nil && *ins.PublicIpAddress != "" {
				return *ins.PublicIpAddress
			}
		}
		time.Sleep(interval)
	}
	return ""
}

func waitInstancePublicIPMatch(ctx context.Context, cli *ec2.Client, instanceId, expectIP string, tries int, interval time.Duration) bool {
	for i := 0; i < tries; i++ {
		out, err := cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{instanceId}})
		if err == nil && len(out.Reservations) > 0 && len(out.Reservations[0].Instances) > 0 {
			ins := out.Reservations[0].Instances[0]
			if ins.PublicIpAddress != nil && *ins.PublicIpAddress == expectIP {
				return true
			}
		}
		time.Sleep(interval)
	}
	return false
}
