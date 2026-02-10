-- 商户独立 OSS 和 GOST 配置
-- 执行位置: control 数据库

-- 1. 商户 OSS 配置表（每商户可配置多个 OSS）
CREATE TABLE IF NOT EXISTS `merchant_oss_configs` (
    `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `merchant_id` INT NOT NULL COMMENT '商户ID',
    `name` VARCHAR(64) NOT NULL COMMENT '配置名称，如：主OSS、备用OSS',
    `cloud_type` VARCHAR(20) NOT NULL COMMENT '云类型: aliyun, tencent, aws, minio',
    `endpoint` VARCHAR(255) NOT NULL COMMENT 'OSS Endpoint',
    `access_key` VARCHAR(255) NOT NULL COMMENT 'AccessKey',
    `secret_key` VARCHAR(255) NOT NULL COMMENT 'SecretKey',
    `bucket` VARCHAR(128) NOT NULL COMMENT 'Bucket名称',
    `region` VARCHAR(64) DEFAULT '' COMMENT '区域',
    `custom_domain` VARCHAR(255) DEFAULT '' COMMENT '自定义域名（CDN）',
    `is_default` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否默认OSS',
    `status` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '状态：1-启用 0-禁用',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_merchant_id` (`merchant_id`),
    INDEX `idx_merchant_default` (`merchant_id`, `is_default`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户OSS配置表';

-- 2. 商户 GOST 服务器关联表（每商户可配置多个 GOST 转发服务器）
CREATE TABLE IF NOT EXISTS `merchant_gost_servers` (
    `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `merchant_id` INT NOT NULL COMMENT '商户ID',
    `server_id` INT NOT NULL COMMENT '服务器ID（关联servers表）',
    `cloud_type` VARCHAR(20) NOT NULL DEFAULT '' COMMENT '云类型: aliyun, tencent, aws',
    `region` VARCHAR(64) DEFAULT '' COMMENT '区域/地区',
    `listen_port` INT NOT NULL DEFAULT 0 COMMENT '在此服务器上监听的端口（0表示使用商户默认端口）',
    `is_primary` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否主转发服务器',
    `priority` INT NOT NULL DEFAULT 0 COMMENT '优先级（数字越小优先级越高）',
    `status` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '状态：1-启用 0-禁用',
    `remark` VARCHAR(255) DEFAULT '' COMMENT '备注',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_merchant_server` (`merchant_id`, `server_id`),
    INDEX `idx_merchant_id` (`merchant_id`),
    INDEX `idx_server_id` (`server_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户GOST服务器关联表';

-- 3. 修改 servers 表，添加用途标识
ALTER TABLE `servers`
    ADD COLUMN `usage_type` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '用途：0-通用 1-商户专属GOST 2-系统共享GOST' AFTER `server_type`;

-- 4. 示例数据
-- INSERT INTO `merchant_oss_configs` (`merchant_id`, `name`, `cloud_type`, `endpoint`, `access_key`, `secret_key`, `bucket`, `region`, `is_default`) VALUES
-- (1, '主存储', 'aliyun', 'oss-cn-hangzhou.aliyuncs.com', 'your-access-key', 'your-secret-key', 'merchant-1-bucket', 'cn-hangzhou', 1);

-- INSERT INTO `merchant_gost_servers` (`merchant_id`, `server_id`, `cloud_type`, `region`, `is_primary`, `priority`) VALUES
-- (1, 10, 'aliyun', 'cn-hangzhou', 1, 0),
-- (1, 11, 'tencent', 'ap-guangzhou', 0, 1);