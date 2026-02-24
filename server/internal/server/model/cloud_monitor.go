package model

// ========== 统一云监控 (AWS CloudWatch / 阿里云云监控 / 腾讯云云监控) ==========

// CloudMonitorMetricsReq 统一云监控请求
type CloudMonitorMetricsReq struct {
	ServerId int    `form:"server_id" binding:"required"`
	Period   string `form:"period"` // "1h","6h","24h","7d", default "1h"
}

// CloudMonitorMetricsResp 统一云监控响应
type CloudMonitorMetricsResp struct {
	CloudType  string         `json:"cloud_type"`            // "aws", "aliyun", "tencent"
	InstanceId string         `json:"instance_id"`           // 云实例ID
	RegionId   string         `json:"region_id"`             // 云区域
	VolumeIds  []string       `json:"volume_ids,omitempty"`  // 磁盘卷ID（AWS 专属）
	Period     int32          `json:"period"`                // 数据点粒度（秒）
	StartTime  int64          `json:"start_time"`            // Unix seconds
	EndTime    int64          `json:"end_time"`              // Unix seconds
	Metrics    []MetricSeries `json:"metrics"`               // 复用 MetricSeries
}
