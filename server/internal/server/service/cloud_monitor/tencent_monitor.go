package cloud_monitor

import (
	"fmt"
	"sort"
	"time"

	tencentCloud "server/internal/server/cloud/tencent"
	"server/internal/server/model"
	"server/pkg/entity"

	"github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/common"
	monitor "github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/monitor/v20180724"
)

// 腾讯云监控指标定义（QCE/CVM 命名空间）
type tencentMetricDef struct {
	MetricName string
	Label      string
	Unit       string
}

var tencentMetrics = []tencentMetricDef{
	{"CpuUsage", "CPU", "Percent"},
	{"DiskReadIops", "Disk Read IOPS", "Count/Second"},
	{"DiskWriteIops", "Disk Write IOPS", "Count/Second"},
	{"CvmDiskUsage", "Disk Usage", "Percent"},
}

func getTencentMetrics(server entity.Servers, instanceId, regionId, period string) (*model.CloudMonitorMetricsResp, error) {
	// 1. 解析腾讯云账号
	acc, err := tencentCloud.GetCloudCredentials(server.MerchantId, server.CloudAccountId)
	if err != nil {
		return nil, fmt.Errorf("解析腾讯云账号失败: %v", err)
	}

	// 2. 创建 Monitor 客户端
	monitorClient, err := tencentCloud.NewMonitorClient(acc.AccessKey, acc.AccessSecret, regionId)
	if err != nil {
		return nil, fmt.Errorf("创建 Monitor 客户端失败: %v", err)
	}

	// 3. 计算时间范围
	now := time.Now().UTC()
	var startTime time.Time
	var periodSec int32
	switch period {
	case "6h":
		startTime = now.Add(-6 * time.Hour)
		periodSec = 300
	case "24h":
		startTime = now.Add(-24 * time.Hour)
		periodSec = 300
	case "7d":
		startTime = now.Add(-7 * 24 * time.Hour)
		periodSec = 3600
	default: // "1h"
		startTime = now.Add(-1 * time.Hour)
		periodSec = 60
	}

	startStr := startTime.Format("2006-01-02T15:04:05+00:00")
	endStr := now.Format("2006-01-02T15:04:05+00:00")

	resp := &model.CloudMonitorMetricsResp{
		CloudType:  "tencent",
		InstanceId: instanceId,
		RegionId:   regionId,
		Period:     periodSec,
		StartTime:  startTime.Unix(),
		EndTime:    now.Unix(),
	}

	// 4. 逐个查询指标
	for _, m := range tencentMetrics {
		request := monitor.NewGetMonitorDataRequest()
		request.Namespace = common.StringPtr("QCE/CVM")
		request.MetricName = common.StringPtr(m.MetricName)
		request.Period = common.Uint64Ptr(uint64(periodSec))
		request.StartTime = common.StringPtr(startStr)
		request.EndTime = common.StringPtr(endStr)
		request.Instances = []*monitor.Instance{
			{
				Dimensions: []*monitor.Dimension{
					{
						Name:  common.StringPtr("InstanceId"),
						Value: common.StringPtr(instanceId),
					},
				},
			},
		}

		out, err := monitorClient.GetMonitorData(request)
		if err != nil {
			continue // 单个指标失败不影响整体
		}

		series := model.MetricSeries{
			MetricName: m.MetricName,
			Label:      m.Label,
			Unit:       m.Unit,
		}

		if out.Response != nil && len(out.Response.DataPoints) > 0 {
			dp := out.Response.DataPoints[0]
			if dp.Timestamps != nil && dp.Values != nil {
				for i, ts := range dp.Timestamps {
					if i < len(dp.Values) {
						series.DataPoints = append(series.DataPoints, model.MetricDataPoint{
							Timestamp: int64(*ts),
							Value:     *dp.Values[i],
						})
					}
				}
			}
		}

		sort.Slice(series.DataPoints, func(i, j int) bool {
			return series.DataPoints[i].Timestamp < series.DataPoints[j].Timestamp
		})
		resp.Metrics = append(resp.Metrics, series)
	}

	return resp, nil
}
