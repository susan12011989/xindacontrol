-- 009_cloud_monitoring_fields.sql
-- 通用化云监控字段（支持 AWS/阿里云/腾讯云）

ALTER TABLE `servers`
  ADD COLUMN IF NOT EXISTS `cloud_type` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '云类型: aws,aliyun,tencent',
  ADD COLUMN IF NOT EXISTS `cloud_instance_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '云实例ID',
  ADD COLUMN IF NOT EXISTS `cloud_region_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '云区域ID';

-- 回填现有 AWS 数据
UPDATE `servers`
SET `cloud_type` = 'aws',
    `cloud_instance_id` = `aws_instance_id`,
    `cloud_region_id` = `aws_region_id`
WHERE `aws_instance_id` != '' AND `cloud_instance_id` = '';
