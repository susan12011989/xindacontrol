-- 商户存储配置管理
-- 用于管理商户服务器的存储配置（MinIO/OSS/S3/COS），支持从 Control 平台推送到商户服务器

-- 商户存储配置表
CREATE TABLE IF NOT EXISTS `merchant_storage_configs` (
    `id` INT NOT NULL AUTO_INCREMENT,
    `merchant_id` INT NOT NULL COMMENT '商户ID',
    `storage_type` VARCHAR(20) NOT NULL COMMENT '存储类型: minio, aliyunOSS, aws_s3, tencent_cos',
    `name` VARCHAR(64) NOT NULL COMMENT '配置名称',

    -- 通用字段
    `endpoint` VARCHAR(255) DEFAULT '' COMMENT '服务端点',
    `bucket` VARCHAR(128) NOT NULL COMMENT 'Bucket名称',
    `region` VARCHAR(64) DEFAULT '' COMMENT '区域',
    `access_key_id` VARCHAR(255) NOT NULL COMMENT 'AccessKeyId',
    `access_key_secret` VARCHAR(255) NOT NULL COMMENT 'AccessKeySecret (加密存储)',

    -- MinIO/S3 专用
    `upload_url` VARCHAR(255) DEFAULT '' COMMENT '上传URL (MinIO)',
    `download_url` VARCHAR(255) DEFAULT '' COMMENT '下载URL (MinIO)',
    `file_base_url` VARCHAR(255) DEFAULT '' COMMENT '文件访问基础URL',

    -- OSS 专用
    `bucket_url` VARCHAR(255) DEFAULT '' COMMENT 'Bucket URL (OSS)',
    `custom_domain` VARCHAR(255) DEFAULT '' COMMENT '自定义域名CDN',

    -- 状态
    `is_default` TINYINT NOT NULL DEFAULT 0 COMMENT '是否默认配置: 0-否 1-是',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 0-禁用 1-启用',
    `last_push_at` DATETIME DEFAULT NULL COMMENT '最后推送时间',
    `last_push_result` VARCHAR(255) DEFAULT '' COMMENT '最后推送结果',

    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (`id`),
    INDEX `idx_merchant_id` (`merchant_id`),
    INDEX `idx_storage_type` (`storage_type`),
    INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户存储配置';

-- 商户表添加 API Token 字段（用于 Control 推送配置时的身份验证）
ALTER TABLE `merchants` ADD COLUMN IF NOT EXISTS `control_api_token` VARCHAR(64) DEFAULT '' COMMENT 'Control API Token' AFTER `status`;
