-- 告警通知系统
-- 执行位置: control 数据库

-- 1. 告警规则表
CREATE TABLE IF NOT EXISTS `alert_rules` (
    `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
    `name` VARCHAR(64) NOT NULL COMMENT '规则名称',
    `type` VARCHAR(32) NOT NULL COMMENT '告警类型: merchant_expired, server_down, cpu_high, memory_high, disk_high',
    `threshold` DECIMAL(10,2) DEFAULT 0 COMMENT '阈值（如 CPU > 80）',
    `merchant_id` INT DEFAULT 0 COMMENT '商户ID（0表示全局规则）',
    `notify_type` VARCHAR(32) NOT NULL DEFAULT 'webhook' COMMENT '通知方式: webhook, email, sms',
    `notify_url` VARCHAR(512) DEFAULT '' COMMENT 'Webhook URL',
    `notify_email` VARCHAR(128) DEFAULT '' COMMENT '通知邮箱',
    `notify_phone` VARCHAR(32) DEFAULT '' COMMENT '通知手机号',
    `interval_minutes` INT NOT NULL DEFAULT 60 COMMENT '告警间隔（分钟），避免频繁告警',
    `status` TINYINT(1) NOT NULL DEFAULT 1 COMMENT '状态：1-启用 0-禁用',
    `description` VARCHAR(255) DEFAULT '' COMMENT '规则描述',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_type` (`type`),
    INDEX `idx_merchant_id` (`merchant_id`),
    INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警规则表';

-- 2. 告警日志表
CREATE TABLE IF NOT EXISTS `alert_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `rule_id` INT NOT NULL COMMENT '规则ID',
    `rule_name` VARCHAR(64) NOT NULL COMMENT '规则名称',
    `type` VARCHAR(32) NOT NULL COMMENT '告警类型',
    `level` VARCHAR(16) NOT NULL DEFAULT 'warning' COMMENT '告警级别: info, warning, error, critical',
    `target_type` VARCHAR(32) NOT NULL COMMENT '目标类型: merchant, server',
    `target_id` INT NOT NULL DEFAULT 0 COMMENT '目标ID',
    `target_name` VARCHAR(128) DEFAULT '' COMMENT '目标名称',
    `message` VARCHAR(512) NOT NULL COMMENT '告警消息',
    `detail` TEXT COMMENT '告警详情（JSON格式）',
    `notify_status` VARCHAR(16) NOT NULL DEFAULT 'pending' COMMENT '通知状态: pending, sent, failed',
    `notify_result` VARCHAR(512) DEFAULT '' COMMENT '通知结果',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_rule_id` (`rule_id`),
    INDEX `idx_type` (`type`),
    INDEX `idx_target` (`target_type`, `target_id`),
    INDEX `idx_created_at` (`created_at`),
    INDEX `idx_notify_status` (`notify_status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='告警日志表';

-- 3. 示例规则
-- INSERT INTO `alert_rules` (`name`, `type`, `threshold`, `notify_type`, `notify_url`, `description`) VALUES
-- ('商户即将过期告警', 'merchant_expired', 3, 'webhook', 'https://example.com/webhook', '商户服务即将在3天内过期'),
-- ('CPU使用率告警', 'cpu_high', 80, 'webhook', 'https://example.com/webhook', 'CPU使用率超过80%'),
-- ('内存使用率告警', 'memory_high', 85, 'webhook', 'https://example.com/webhook', '内存使用率超过85%'),
-- ('磁盘使用率告警', 'disk_high', 90, 'webhook', 'https://example.com/webhook', '磁盘使用率超过90%');
