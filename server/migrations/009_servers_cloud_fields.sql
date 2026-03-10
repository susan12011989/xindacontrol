-- 为 servers 表添加云实例追踪字段
ALTER TABLE `servers` ADD COLUMN `cloud_type` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '云类型: aws,aliyun,tencent' AFTER `cloud_account_id`;
ALTER TABLE `servers` ADD COLUMN `cloud_instance_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '云实例ID' AFTER `cloud_type`;
ALTER TABLE `servers` ADD COLUMN `cloud_region_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '云区域ID' AFTER `cloud_instance_id`;
