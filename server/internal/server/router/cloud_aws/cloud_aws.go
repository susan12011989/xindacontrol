package cloud_aws

import (
	"context"
	"fmt"
	"net/http"
	awscloud "server/internal/server/cloud/aws"
	"server/internal/server/middleware"
	"server/internal/server/model"
	cloudService "server/internal/server/service/cloud_aws"
	"server/internal/server/utils"
	"server/pkg/consts"
	"server/pkg/result"
	"strconv"
	"strings"
	"sync"

	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/gin-gonic/gin"
)

// Routes AWS 服务API路由
func Routes(ge gin.IRouter) {
	group := ge.Group("/aws", middleware.Authorization)

	// EC2
	group.GET("/ec2/instance", listEc2Instance)
	group.POST("/ec2/instance/operate", operateEc2Instance)
	group.POST("/ec2/instance/create", createEc2Instance)
	group.POST("/ec2/instance/modify", modifyEc2Instance)
	group.POST("/ec2/volume/resize/stream", resizeVolumeStream)
	group.GET("/ec2/volumes", listVolumes)
	group.GET("/ec2/volumes/usage", getVolumeUsage)
	group.GET("/ec2/images", listImages)
	group.GET("/ec2/instance-types", listInstanceTypes)
	group.GET("/ec2/subnets", listSubnets)
	group.GET("/ec2/security-groups/options", listSecurityGroupsOptions)

	// Security Group
	group.GET("/ec2/security-group", listSecurityGroup)
	group.GET("/ec2/security-group/attribute", describeSecurityGroup)
	group.POST("/ec2/security-group/authorize", authorizeSecurityGroup)
	group.POST("/ec2/security-group/revoke", revokeSecurityGroup)

	// EIP
	group.GET("/ec2/eip", listEip)
	group.POST("/ec2/eip/operate", operateEip)
	group.POST("/ec2/eip/allocate", allocateEip)
	group.GET("/ec2/instance/describe", describeInstance)

	// S3
	group.GET("/s3/buckets", listBuckets)
	group.GET("/s3/objects", listObjects)
	group.POST("/s3/object/upload", uploadObject)
	group.GET("/s3/object/download", downloadObject)
	group.DELETE("/s3/object", deleteObject)
	group.POST("/s3/bucket", createBucket)
	group.DELETE("/s3/bucket", deleteBucket)
	group.POST("/s3/bucket/set-public", setBucketPublic)

	// Billing
	group.GET("/billing/cost-usage", getCostAndUsage)

	// CloudWatch Monitoring
	group.GET("/cloudwatch/metrics", getCloudWatchMetrics)
}

func listEc2Instance(c *gin.Context) {
	var req model.AwsListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if len(req.RegionId) == 0 {
		result.GErr(c, fmt.Errorf("region_id不能为空"))
		return
	}
	// 统一解析账号
	acc, errAcc := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if errAcc != nil {
		result.GErr(c, errAcc)
		return
	}
	type regionResult struct {
		Region    string
		Instances []any
		Err       error
	}
	ch := make(chan regionResult, len(req.RegionId))
	var wg sync.WaitGroup
	for _, region := range req.RegionId {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			instances, err := cloudService.ListEc2Instances(acc, r)
			wrapped := make([]any, 0, len(instances))
			// 补充每台实例的 CPU 和内存信息（DescribeInstanceTypes 批量聚合），并注入到返回结构
			if err == nil && len(instances) > 0 {
				// 收集规格
				type set map[string]struct{}
				uniq := set{}
				for _, ins := range instances {
					if s := string(ins.InstanceType); s != "" {
						uniq[s] = struct{}{}
					}
				}
				// 统一使用账号信息解析 vCPU 和内存
				infoMap, derr := cloudService.ResolveInstanceTypesInfoWithAccount(acc, r, uniq)
				if derr == nil {
					type instanceWithInfo struct {
						ec2types.Instance
						VCpu      int32 `json:"vcpu"`
						MemoryMiB int64 `json:"memory_mib"`
					}
					for _, ins := range instances {
						info := infoMap[string(ins.InstanceType)]
						wrapped = append(wrapped, instanceWithInfo{
							Instance:  ins,
							VCpu:      info.VCpu,
							MemoryMiB: info.MemoryMiB,
						})
					}
				} else {
					// 回退：不带 CPU/内存注入，直接使用原始实例
					for _, ins := range instances {
						wrapped = append(wrapped, ins)
					}
				}
			}
			if err == nil && len(instances) == 0 {
				wrapped = []any{}
			}
			ch <- regionResult{Region: r, Instances: wrapped, Err: err}
		}(region)
	}
	go func() { wg.Wait(); close(ch) }()
	var all []any
	var errors []string
	for v := range ch {
		if v.Err != nil {
			errors = append(errors, fmt.Sprintf("region %s: %s", v.Region, v.Err.Error()))
			continue
		}
		all = append(all, v.Instances...)
	}
	if len(errors) > 0 {
		result.GErr(c, fmt.Errorf("部分区域请求失败: %s", strings.Join(errors, "; ")))
		return
	}
	result.GOK(c, gin.H{"list": all, "total": len(all)})
}

func operateEc2Instance(c *gin.Context) {
	var req model.AwsOperateEc2Req
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	err = cloudService.OperateEc2Instance(acc, req.RegionId, req.InstanceId, req.Operation)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func createEc2Instance(c *gin.Context) {
	var req model.AwsCreateEc2InstanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	id, err := cloudService.CreateEc2Instance(acc, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"instance_id": id})
}

func modifyEc2Instance(c *gin.Context) {
	var req model.AwsModifyEc2InstanceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	// 参数互斥校验：不能同时传或都不传
	if (req.CloudAccountId > 0 && req.MerchantId > 0) || (req.CloudAccountId == 0 && req.MerchantId == 0) {
		result.GParamErr(c, fmt.Errorf("cloud_account_id 与 merchant_id 不能同时传，且必须提供一个"))
		return
	}
	if err := cloudService.ModifyEc2Instance(req); err != nil { // 保持原签名，这里不需要客户端
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// 保留 stream 版本，普通版本已移除

// 扩容 EBS 卷（SSE流式输出）
func resizeVolumeStream(c *gin.Context) {
	var req model.AwsResizeVolumeReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if (req.CloudAccountId > 0 && req.MerchantId > 0) || (req.CloudAccountId == 0 && req.MerchantId == 0) {
		result.GParamErr(c, fmt.Errorf("cloud_account_id 与 merchant_id 不能同时传，且必须提供一个"))
		return
	}

	// 开启SSE
	result.GStream(c)

	// 解析账号
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GStreamData(c, gin.H{"error": err.Error()})
		result.GStreamEnd(c, false, "解析账号失败")
		return
	}

	// EC2 client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	ec2cli, err := awscloud.NewEc2Client(ctx, acc, req.RegionId)
	if err != nil {
		result.GStreamData(c, gin.H{"error": err.Error()})
		result.GStreamEnd(c, false, "创建EC2客户端失败")
		return
	}

	// 解析卷ID
	volumeId := strings.TrimSpace(req.VolumeId)
	if volumeId == "" {
		if req.InstanceId == "" {
			result.GStreamEnd(c, false, "缺少 volume_id 或 instance_id")
			return
		}
		din, derr := ec2cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{InstanceIds: []string{req.InstanceId}})
		if derr != nil || len(din.Reservations) == 0 || len(din.Reservations[0].Instances) == 0 {
			result.GStreamEnd(c, false, "实例不存在或描述失败")
			return
		}
		ins := din.Reservations[0].Instances[0]
		targetDevice := strings.TrimSpace(req.DeviceName)
		if targetDevice == "" && ins.RootDeviceName != nil {
			targetDevice = *ins.RootDeviceName
		}
		for _, bdm := range ins.BlockDeviceMappings {
			if bdm.Ebs == nil || bdm.Ebs.VolumeId == nil {
				continue
			}
			if targetDevice == "" || (bdm.DeviceName != nil && *bdm.DeviceName == targetDevice) {
				volumeId = *bdm.Ebs.VolumeId
				if targetDevice != "" {
					break
				}
			}
		}
		if volumeId == "" {
			result.GStreamEnd(c, false, "未找到目标卷")
			return
		}
	}
	result.GStreamData(c, gin.H{"step": "resolve_volume", "volume_id": volumeId})

	// ModifyVolume
	_, err = ec2cli.ModifyVolume(ctx, &ec2.ModifyVolumeInput{VolumeId: &volumeId, Size: &req.NewSizeGiB})
	if err != nil {
		result.GStreamEnd(c, false, fmt.Sprintf("ModifyVolume 失败: %v", err))
		return
	}
	result.GStreamData(c, gin.H{"step": "modify_submitted", "target_size_gib": req.NewSizeGiB})

	// 轮询修改状态并输出进度
	deadline := time.Now().Add(10 * time.Minute)
	for time.Now().Before(deadline) {
		ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
		out, derr := ec2cli.DescribeVolumesModifications(ctx2, &ec2.DescribeVolumesModificationsInput{VolumeIds: []string{volumeId}})
		cancel2()
		if derr == nil && len(out.VolumesModifications) > 0 {
			vm := out.VolumesModifications[0]
			state := string(vm.ModificationState)
			progress := 0
			if vm.Progress != nil {
				progress = int(*vm.Progress)
			}
			result.GStreamData(c, gin.H{"step": "modify_status", "state": state, "progress": progress})
			if state == "optimizing" || state == "completed" {
				break
			}
		}
		time.Sleep(5 * time.Second)
	}

	if !req.ExpandFS {
		result.GStreamEnd(c, true, "卷扩容完成（未开启文件系统扩展）")
		return
	}

	// 实例内扩展：优先 SSM，失败回退 SSH
	result.GStreamData(c, gin.H{"step": "expand_fs_start"})
	// SSM 客户端
	ssmcli, serr := awscloud.NewSsmClient(context.Background(), acc, req.RegionId)
	if serr == nil {
		commands := []string{
			"set -euxo pipefail",
			"ROOT=$(findmnt -n -o SOURCE /)",
			"PARENT=$ROOT; PARTNUM=",
			"if [[ $ROOT =~ ^/dev/nvme[0-9]+n[0-9]+p[0-9]+$ ]]; then PARENT=${ROOT%p*}; PARTNUM=${ROOT##*p}; fi",
			"if [[ $ROOT =~ ^/dev/[a-z]+[0-9]+$ ]]; then PARENT=$(echo $ROOT | sed -E 's/[0-9]+$//'); PARTNUM=$(echo $ROOT | sed -E 's/^.*([0-9]+)$/\\1/'); fi",
			"if ! command -v growpart >/dev/null 2>&1; then if command -v yum >/dev/null 2>&1; then yum install -y cloud-utils-growpart || true; fi; fi",
			"if ! command -v growpart >/dev/null 2>&1; then if command -v apt-get >/dev/null 2>&1; then apt-get update && apt-get install -y cloud-guest-utils || true; fi; fi",
			"if command -v growpart >/dev/null 2>&1 && [ -n \"$PARTNUM\" ]; then growpart $PARENT $PARTNUM || true; fi",
			"FSTYPE=$(findmnt -n -o FSTYPE /)",
			"if [ \"$FSTYPE\" = \"xfs\" ]; then xfs_growfs /; else resize2fs \"$ROOT\"; fi",
		}
		in := &ssm.SendCommandInput{
			DocumentName:   strPtr("AWS-RunShellScript"),
			InstanceIds:    []string{req.InstanceId},
			Comment:        strPtr("Expand partition and filesystem after EBS resize"),
			TimeoutSeconds: int32Ptr(600),
			Parameters:     map[string][]string{"commands": commands},
		}
		out, sendErr := ssmcli.SendCommand(context.Background(), in)
		if sendErr == nil && out.Command != nil && out.Command.CommandId != nil {
			cmdId := *out.Command.CommandId
			result.GStreamData(c, gin.H{"step": "ssm_send", "command_id": cmdId})
			// 轮询SSM命令状态
			sDeadline := time.Now().Add(10 * time.Minute)
			for time.Now().Before(sDeadline) {
				time.Sleep(5 * time.Second)
				inv, invErr := ssmcli.GetCommandInvocation(context.Background(), &ssm.GetCommandInvocationInput{CommandId: &cmdId, InstanceId: &req.InstanceId})
				if invErr == nil {
					st := inv.Status
					result.GStreamData(c, gin.H{"step": "ssm_status", "status": string(st)})
					if st == ssmtypes.CommandInvocationStatusSuccess {
						result.GStreamEnd(c, true, "扩容完成")
						return
					}
					if st == ssmtypes.CommandInvocationStatusFailed || st == ssmtypes.CommandInvocationStatusCancelled || st == ssmtypes.CommandInvocationStatusTimedOut {
						break // 改用SSH
					}
				}
			}
		}
	}

	// 回退 SSH：root/DefaultPassword
	result.GStreamData(c, gin.H{"step": "ssh_fallback"})
	// 解析实例IP
	ip := ""
	din, derr := ec2cli.DescribeInstances(context.Background(), &ec2.DescribeInstancesInput{InstanceIds: []string{req.InstanceId}})
	if derr == nil && len(din.Reservations) > 0 && len(din.Reservations[0].Instances) > 0 {
		ins := din.Reservations[0].Instances[0]
		if ins.PublicIpAddress != nil {
			ip = *ins.PublicIpAddress
		} else if ins.PrivateIpAddress != nil {
			ip = *ins.PrivateIpAddress
		}
	}
	if ip == "" {
		result.GStreamEnd(c, false, "无法解析实例IP")
		return
	}
	client := &utils.SSHClient{Host: ip, Port: 22, Username: "root", Password: consts.DefaultPassword}
	script := strings.Join([]string{
		"set -euxo pipefail",
		"ROOT=$(findmnt -n -o SOURCE /)",
		"PARENT=$ROOT; PARTNUM=",
		"if [[ $ROOT =~ ^/dev/nvme[0-9]+n[0-9]+p[0-9]+$ ]]; then PARENT=${ROOT%p*}; PARTNUM=${ROOT##*p}; fi",
		"if [[ $ROOT =~ ^/dev/[a-z]+[0-9]+$ ]]; then PARENT=$(echo $ROOT | sed -E 's/[0-9]+$//'); PARTNUM=$(echo $ROOT | sed -E 's/^.*([0-9]+)$/\\1/'); fi",
		"if ! command -v growpart >/dev/null 2>&1; then if command -v yum >/dev/null 2>&1; then yum install -y cloud-utils-growpart || true; fi; fi",
		"if ! command -v growpart >/dev/null 2>&1; then if command -v apt-get >/dev/null 2>&1; then apt-get update && apt-get install -y cloud-guest-utils || true; fi; fi",
		"if command -v growpart >/dev/null 2>&1 && [ -n \"$PARTNUM\" ]; then growpart $PARENT $PARTNUM || true; fi",
		"FSTYPE=$(findmnt -n -o FSTYPE /)",
		"if [ \"$FSTYPE\" = \"xfs\" ]; then xfs_growfs /; else resize2fs \"$ROOT\"; fi",
	}, " && ")
	if _, err := client.ExecuteCommandWithTimeout(script, 2*time.Minute); err != nil {
		result.GStreamEnd(c, false, fmt.Sprintf("SSH 扩展失败: %v", err))
		return
	}
	result.GStreamEnd(c, true, "扩容完成")
}

// 列举卷信息
func listVolumes(c *gin.Context) {
	var req model.AwsListVolumesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	// 参数互斥校验：不能同时传或都不传
	if (req.CloudAccountId > 0 && req.MerchantId > 0) || (req.CloudAccountId == 0 && req.MerchantId == 0) {
		result.GParamErr(c, fmt.Errorf("cloud_account_id 与 merchant_id 不能同时传，且必须提供一个"))
		return
	}
	items, err := cloudService.ListVolumes(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"list": items, "total": len(items)})
}

// 获取卷使用率（SSM）
func getVolumeUsage(c *gin.Context) {
	var req model.AwsGetVolumeUsageReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if (req.CloudAccountId > 0 && req.MerchantId > 0) || (req.CloudAccountId == 0 && req.MerchantId == 0) {
		result.GParamErr(c, fmt.Errorf("cloud_account_id 与 merchant_id 不能同时传，且必须提供一个"))
		return
	}
	items, err := cloudService.GetVolumeUsage(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"list": items, "total": len(items)})
}

func listImages(c *gin.Context) {
	var req model.AwsListImagesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	items, err := cloudService.ListImages(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"list": items})
}

func listInstanceTypes(c *gin.Context) {
	var req model.AwsListInstanceTypesReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	items, err := cloudService.ListInstanceTypes(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"list": items})
}

func listSubnets(c *gin.Context) {
	var req model.AwsListSubnetsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	items, err := cloudService.ListSubnets(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"list": items})
}

func listSecurityGroupsOptions(c *gin.Context) {
	var req model.AwsListSecurityGroupsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	items, err := cloudService.ListSecurityGroupOptions(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"list": items})
}

func listSecurityGroup(c *gin.Context) {
	var req model.AwsListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if len(req.RegionId) == 0 {
		result.GErr(c, fmt.Errorf("region_id不能为空"))
		return
	}
	acc, errAcc := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if errAcc != nil {
		result.GErr(c, errAcc)
		return
	}
	type regionResult struct {
		Region string
		Groups any
		Err    error
	}
	ch := make(chan regionResult, len(req.RegionId))
	var wg sync.WaitGroup
	for _, region := range req.RegionId {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			groups, err := cloudService.ListSecurityGroups(acc, r)
			ch <- regionResult{Region: r, Groups: groups, Err: err}
		}(region)
	}
	go func() { wg.Wait(); close(ch) }()
	var all []any
	var errors []string
	for v := range ch {
		if v.Err != nil {
			errors = append(errors, fmt.Sprintf("region %s: %s", v.Region, v.Err.Error()))
			continue
		}
		all = append(all, v.Groups)
	}
	if len(errors) > 0 {
		result.GErr(c, fmt.Errorf("部分区域请求失败: %s", strings.Join(errors, "; ")))
		return
	}
	result.GOK(c, gin.H{"list": all, "total": len(all)})
}

func describeSecurityGroup(c *gin.Context) {
	var req model.AwsDescribeSecurityGroupReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	out, err := cloudService.DescribeSecurityGroup(acc, req.RegionId, req.GroupId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, out)
}

func authorizeSecurityGroup(c *gin.Context) {
	var req model.AwsAuthorizeSecurityGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	err = cloudService.AuthorizeSecurityGroupIngress(acc, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func revokeSecurityGroup(c *gin.Context) {
	var req model.AwsAuthorizeSecurityGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	err = cloudService.RevokeSecurityGroupIngress(acc, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func listEip(c *gin.Context) {
	var req model.AwsListReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	if len(req.RegionId) == 0 {
		result.GErr(c, fmt.Errorf("region_id不能为空"))
		return
	}
	acc, errAcc := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if errAcc != nil {
		result.GErr(c, errAcc)
		return
	}
	type regionResult struct {
		Region string
		Eips   []any
		Err    error
	}
	ch := make(chan regionResult, len(req.RegionId))
	var wg sync.WaitGroup
	for _, region := range req.RegionId {
		wg.Add(1)
		go func(r string) {
			defer wg.Done()
			eips, err := cloudService.ListAddresses(acc, r)
			wrapped := make([]any, 0, len(eips))
			if err == nil {
				type addressWithRegion struct {
					ec2types.Address
					Region string `json:"Region"`
				}
				for _, e := range eips {
					wrapped = append(wrapped, addressWithRegion{Address: e, Region: r})
				}
			}
			ch <- regionResult{Region: r, Eips: wrapped, Err: err}
		}(region)
	}
	go func() { wg.Wait(); close(ch) }()
	var all []any
	var errors []string
	for v := range ch {
		if v.Err != nil {
			errors = append(errors, fmt.Sprintf("region %s: %s", v.Region, v.Err.Error()))
			continue
		}
		all = append(all, v.Eips...)
	}
	if len(errors) > 0 {
		result.GErr(c, fmt.Errorf("部分区域请求失败: %s", strings.Join(errors, "; ")))
		return
	}
	result.GOK(c, gin.H{"list": all, "total": len(all)})
}

func operateEip(c *gin.Context) {
	var req model.AwsOperateEipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	err = cloudService.OperateAddress(acc, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func allocateEip(c *gin.Context) {
	var req model.AwsAllocateEipReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	id, err := cloudService.AllocateAddress(acc, req.RegionId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"allocation_id": id})
}

func describeInstance(c *gin.Context) {
	var req model.AwsDescribeInstanceReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	data, err := cloudService.DescribeInstance(acc, req.RegionId, req.InstanceId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

func listBuckets(c *gin.Context) {
	var req model.AwsS3ListBucketsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	// 使用新的函数获取 bucket 列表及其区域信息
	buckets, err := cloudService.ListBucketsWithLocation(acc)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, gin.H{"list": buckets, "total": len(buckets)})
}

func listObjects(c *gin.Context) {
	var req model.AwsS3ListObjectsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	data, err := cloudService.ListObjects(acc, req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}

func uploadObject(c *gin.Context) {
	bucket := c.PostForm("bucket")
	key := c.PostForm("object_key")
	region := c.PostForm("region_id")
	merchantId, _ := strconvAtoi(c.PostForm("merchant_id"))
	cloudAccountId, _ := strconvParseInt64(c.PostForm("cloud_account_id"))
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		result.GParamErr(c, err)
		return
	}
	defer file.Close()
	acc, err := awscloud.ResolveAwsAccount(c, merchantId, cloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if err := cloudService.UploadObject(acc, region, bucket, key, file); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func setBucketPublic(c *gin.Context) {
	var req model.AwsS3SetBucketPublicReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if err := cloudService.SetBucketPublicAccess(acc, req.RegionId, req.Bucket, req.Public); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func createBucket(c *gin.Context) {
	var req model.AwsS3CreateBucketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if err := cloudService.CreateBucket(acc, req.RegionId, req.Bucket); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func deleteBucket(c *gin.Context) {
	var req model.AwsS3DeleteBucketReq
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if err := cloudService.DeleteBucket(acc, req.RegionId, req.Bucket); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

func downloadObject(c *gin.Context) {
	var req model.OssDownloadReq // 复用字段名结构
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	data, contentType, filename, err := cloudService.DownloadObject(acc, req.RegionId, req.Bucket, req.ObjectKey)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if req.Filename != "" {
		filename = req.Filename
	}
	if req.Attachment == 1 {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%q", filename))
	}
	c.Data(http.StatusOK, contentType, data)
}

func deleteObject(c *gin.Context) {
	var req model.OssDownloadReq // 复用字段名结构
	if err := c.ShouldBindJSON(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	if err := cloudService.DeleteObject(acc, req.RegionId, req.Bucket, req.ObjectKey); err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, nil)
}

// getCostAndUsage AWS 账单查询（Cost Explorer）
func getCostAndUsage(c *gin.Context) {
	var req struct {
		MerchantId       int      `form:"merchant_id"`
		CloudAccountId   int64    `form:"cloud_account_id"`
		RegionId         string   `form:"region_id"`
		Start            string   `form:"start" binding:"required"` // YYYY-MM-DD
		End              string   `form:"end" binding:"required"`   // YYYY-MM-DD
		Granularity      string   `form:"granularity"`              // DAILY|MONTHLY
		Metrics          []string `form:"metrics[]"`
		GroupByKey       string   `form:"group_by_key"`      // e.g. SERVICE
		ExcludeEstimated int      `form:"exclude_estimated"` // 1=仅已出账
	}
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	// 校验日期
	if _, err := time.Parse("2006-01-02", req.Start); err != nil {
		result.GParamErr(c, fmt.Errorf("start 日期格式错误"))
		return
	}
	if _, err := time.Parse("2006-01-02", req.End); err != nil {
		result.GParamErr(c, fmt.Errorf("end 日期格式错误"))
		return
	}
	acc, err := awscloud.ResolveAwsAccount(c, req.MerchantId, req.CloudAccountId)
	if err != nil {
		result.GErr(c, err)
		return
	}
	out, err := cloudService.GetCostAndUsage(acc, req.RegionId, req.Start, req.End, req.Granularity, req.Metrics, req.GroupByKey, req.ExcludeEstimated == 1)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, out)
}

func strconvAtoi(s string) (int, error) {
	if s == "" {
		return 0, nil
	}
	return strconv.Atoi(s)
}
func strconvParseInt64(s string) (int64, error) {
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	return v, err
}

func strPtr(s string) *string { return &s }
func int32Ptr(v int32) *int32 { return &v }

// ========== CloudWatch ==========

func getCloudWatchMetrics(c *gin.Context) {
	var req model.AwsCloudWatchMetricsReq
	if err := c.ShouldBindQuery(&req); err != nil {
		result.GParamErr(c, err)
		return
	}
	data, err := cloudService.GetCloudWatchMetrics(req)
	if err != nil {
		result.GErr(c, err)
		return
	}
	result.GOK(c, data)
}
