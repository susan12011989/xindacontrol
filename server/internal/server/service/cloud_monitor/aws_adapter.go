package cloud_monitor

import (
	"server/internal/server/model"
	"server/internal/server/service/cloud_aws"
	"server/pkg/entity"
)

// getAwsMetrics 调用现有 AWS CloudWatch 实现，转换为统一响应格式
func getAwsMetrics(server entity.Servers, period string) (*model.CloudMonitorMetricsResp, error) {
	awsResp, err := cloud_aws.GetCloudWatchMetrics(model.AwsCloudWatchMetricsReq{
		ServerId: server.Id,
		Period:   period,
	})
	if err != nil {
		return nil, err
	}

	return &model.CloudMonitorMetricsResp{
		CloudType:  "aws",
		InstanceId: awsResp.InstanceId,
		RegionId:   awsResp.RegionId,
		VolumeIds:  awsResp.VolumeIds,
		Period:     awsResp.Period,
		StartTime:  awsResp.StartTime,
		EndTime:    awsResp.EndTime,
		Metrics:    awsResp.Metrics,
	}, nil
}
