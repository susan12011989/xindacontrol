package cloud_monitor

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"server/internal/server/cloud/aliyun"
	"server/internal/server/model"
	"server/pkg/entity"

	cms20190101 "github.com/alibabacloud-go/cms-20190101/v9/client"
	"github.com/alibabacloud-go/tea/tea"
)

// 阿里云 CMS 指标定义（acs_ecs_dashboard 命名空间）
type aliyunMetricDef struct {
	MetricName string
	Label      string
	Unit       string
}

var aliyunMetrics = []aliyunMetricDef{
	{"CPUUtilization", "CPU", "Percent"},
	{"DiskReadIOPS", "Disk Read IOPS", "Count/Second"},
	{"DiskWriteIOPS", "Disk Write IOPS", "Count/Second"},
	{"diskio_queue_length", "Disk Queue Length", "Count"},
	{"disk_readLatency", "Disk Read Latency", "Milliseconds"},
	{"disk_writeLatency", "Disk Write Latency", "Milliseconds"},
}

func getAliyunMetrics(server entity.Servers, instanceId, regionId, period string) (*model.CloudMonitorMetricsResp, error) {
	// 1. 解析阿里云账号
	acc, err := resolveAliyunAccount(server.MerchantId, server.CloudAccountId)
	if err != nil {
		return nil, fmt.Errorf("解析阿里云账号失败: %v", err)
	}

	// 2. 创建 CMS 客户端
	cmsClient, err := aliyun.NewCmsClient(acc.AccessKey, acc.AccessSecret, regionId)
	if err != nil {
		return nil, fmt.Errorf("创建 CMS 客户端失败: %v", err)
	}

	// 3. 计算时间范围和粒度
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

	dimensions := fmt.Sprintf(`[{"instanceId":"%s"}]`, instanceId)

	resp := &model.CloudMonitorMetricsResp{
		CloudType:  "aliyun",
		InstanceId: instanceId,
		RegionId:   regionId,
		Period:     periodSec,
		StartTime:  startTime.Unix(),
		EndTime:    now.Unix(),
	}

	// 4. 逐个查询指标
	for _, m := range aliyunMetrics {
		request := &cms20190101.DescribeMetricListRequest{
			Namespace:  tea.String("acs_ecs_dashboard"),
			MetricName: tea.String(m.MetricName),
			Dimensions: tea.String(dimensions),
			StartTime:  tea.String(fmt.Sprintf("%d", startTime.UnixMilli())),
			EndTime:    tea.String(fmt.Sprintf("%d", now.UnixMilli())),
			Period:     tea.String(fmt.Sprintf("%d", periodSec)),
		}

		out, err := cmsClient.DescribeMetricList(request)
		if err != nil {
			continue // 单个指标失败不影响整体
		}

		series := model.MetricSeries{
			MetricName: m.MetricName,
			Label:      m.Label,
			Unit:       m.Unit,
		}

		// 解析阿里云 CMS 响应（Datapoints 是 JSON 字符串）
		if out.Body != nil && out.Body.Datapoints != nil {
			var datapoints []struct {
				Timestamp int64   `json:"timestamp"`
				Average   float64 `json:"Average"`
				Value     float64 `json:"Value"`
			}
			if err := json.Unmarshal([]byte(*out.Body.Datapoints), &datapoints); err == nil {
				for _, dp := range datapoints {
					val := dp.Average
					if val == 0 {
						val = dp.Value
					}
					series.DataPoints = append(series.DataPoints, model.MetricDataPoint{
						Timestamp: dp.Timestamp / 1000, // 阿里云返回毫秒，转为秒
						Value:     val,
					})
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

func resolveAliyunAccount(merchantId int, cloudAccountId int64) (*aliyun.CloudAccountInfo, error) {
	if cloudAccountId > 0 {
		return aliyun.GetSystemCloudAccount(cloudAccountId)
	}
	if merchantId > 0 {
		return aliyun.GetMerchantCloud(merchantId)
	}
	return nil, fmt.Errorf("cloud_account_id 或 merchant_id 必须提供一个")
}
