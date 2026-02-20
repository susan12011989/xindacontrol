package cloud_aws

import (
	"context"
	"fmt"
	"sort"
	"time"

	awsv2 "github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/ec2"

	awscloud "server/internal/server/cloud/aws"
	"server/internal/server/model"
	"server/pkg/dbs"
	"server/pkg/entity"
)

type metricDef struct {
	Namespace  string
	MetricName string
	DimName    string
	DimValue   string
	Label      string
	Unit       string
	Stat       string // "Average" or "Sum"
}

// GetCloudWatchMetrics 查询 CloudWatch 指标（CPU + EBS IOPS/Queue/Latency）
func GetCloudWatchMetrics(req model.AwsCloudWatchMetricsReq) (*model.AwsCloudWatchMetricsResp, error) {
	// 1. 从 Servers 表拿 instance_id / region / merchant_id
	var server entity.Servers
	has, err := dbs.DBAdmin.Where("id = ?", req.ServerId).Get(&server)
	if err != nil {
		return nil, fmt.Errorf("查询服务器失败: %v", err)
	}
	if !has {
		return nil, fmt.Errorf("服务器不存在")
	}
	if server.AwsInstanceId == "" {
		return nil, fmt.Errorf("该服务器未配置 AWS 实例 ID")
	}
	if server.AwsRegionId == "" {
		return nil, fmt.Errorf("该服务器未配置 AWS 区域")
	}

	// 2. 解析 AWS 账号
	acc, err := awscloud.ResolveAwsAccount(context.Background(), server.MerchantId, server.CloudAccountId)
	if err != nil {
		return nil, fmt.Errorf("解析 AWS 账号失败: %v", err)
	}

	instanceId := server.AwsInstanceId
	regionId := server.AwsRegionId

	// 3. 计算时间范围和精度
	now := time.Now().UTC()
	var startTime time.Time
	var period int32
	switch req.Period {
	case "6h":
		startTime = now.Add(-6 * time.Hour)
		period = 300
	case "24h":
		startTime = now.Add(-24 * time.Hour)
		period = 300
	case "7d":
		startTime = now.Add(-7 * 24 * time.Hour)
		period = 3600
	default: // "1h"
		startTime = now.Add(-1 * time.Hour)
		period = 60
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 4. 查询该实例挂载的 EBS 卷
	ec2cli, err := awscloud.NewEc2Client(ctx, acc, regionId)
	if err != nil {
		return nil, fmt.Errorf("创建 EC2 客户端失败: %v", err)
	}
	din, err := ec2cli.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	})
	if err != nil {
		return nil, fmt.Errorf("查询实例 %s 失败: %v", instanceId, err)
	}
	if len(din.Reservations) == 0 || len(din.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("实例 %s 未找到", instanceId)
	}

	inst := din.Reservations[0].Instances[0]
	var volumeIds []string
	for _, bdm := range inst.BlockDeviceMappings {
		if bdm.Ebs != nil && bdm.Ebs.VolumeId != nil {
			volumeIds = append(volumeIds, *bdm.Ebs.VolumeId)
		}
	}

	// 5. 构建 CloudWatch 查询列表
	var queries []metricDef

	// CPU
	queries = append(queries, metricDef{
		Namespace: "AWS/EC2", MetricName: "CPUUtilization",
		DimName: "InstanceId", DimValue: instanceId,
		Label: "CPU", Unit: "Percent", Stat: "Average",
	})

	// EBS 指标（每个卷）
	for _, vid := range volumeIds {
		shortVol := vid
		if len(vid) > 12 {
			shortVol = vid[len(vid)-8:]
		}
		queries = append(queries,
			metricDef{"AWS/EBS", "VolumeReadOps", "VolumeId", vid,
				fmt.Sprintf("Read IOPS (%s)", shortVol), "Count/Second", "Sum"},
			metricDef{"AWS/EBS", "VolumeWriteOps", "VolumeId", vid,
				fmt.Sprintf("Write IOPS (%s)", shortVol), "Count/Second", "Sum"},
			metricDef{"AWS/EBS", "VolumeQueueLength", "VolumeId", vid,
				fmt.Sprintf("Queue (%s)", shortVol), "Count", "Average"},
			metricDef{"AWS/EBS", "VolumeTotalReadTime", "VolumeId", vid,
				fmt.Sprintf("Read Latency (%s)", shortVol), "Seconds", "Average"},
			metricDef{"AWS/EBS", "VolumeTotalWriteTime", "VolumeId", vid,
				fmt.Sprintf("Write Latency (%s)", shortVol), "Seconds", "Average"},
		)
	}

	// 6. 逐个查询 CloudWatch 指标
	cwcli, err := awscloud.NewCloudWatchClient(ctx, acc, regionId)
	if err != nil {
		return nil, fmt.Errorf("创建 CloudWatch 客户端失败: %v", err)
	}

	resp := &model.AwsCloudWatchMetricsResp{
		InstanceId: instanceId,
		RegionId:   regionId,
		VolumeIds:  volumeIds,
		Period:     period,
		StartTime:  startTime.Unix(),
		EndTime:    now.Unix(),
	}

	for _, q := range queries {
		stat := cwtypes.Statistic(q.Stat)
		out, err := cwcli.GetMetricStatistics(ctx, &cloudwatch.GetMetricStatisticsInput{
			Namespace:  awsv2.String(q.Namespace),
			MetricName: awsv2.String(q.MetricName),
			Dimensions: []cwtypes.Dimension{{
				Name:  awsv2.String(q.DimName),
				Value: awsv2.String(q.DimValue),
			}},
			StartTime:  awsv2.Time(startTime),
			EndTime:    awsv2.Time(now),
			Period:     awsv2.Int32(period),
			Statistics: []cwtypes.Statistic{stat},
		})
		if err != nil {
			continue // 单个指标失败不影响整体
		}

		series := model.MetricSeries{
			MetricName: q.MetricName,
			Label:      q.Label,
			Unit:       q.Unit,
		}
		for _, dp := range out.Datapoints {
			val := 0.0
			if q.Stat == "Average" && dp.Average != nil {
				val = *dp.Average
			} else if q.Stat == "Sum" && dp.Sum != nil {
				// IOPS: Sum of ops in period → divide by period to get ops/sec
				val = *dp.Sum / float64(period)
			}
			series.DataPoints = append(series.DataPoints, model.MetricDataPoint{
				Timestamp: dp.Timestamp.Unix(),
				Value:     val,
			})
		}
		// 按时间升序排列
		sort.Slice(series.DataPoints, func(i, j int) bool {
			return series.DataPoints[i].Timestamp < series.DataPoints[j].Timestamp
		})
		resp.Metrics = append(resp.Metrics, series)
	}

	return resp, nil
}
