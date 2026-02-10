-- 操作审计日志表
-- 执行位置: control 数据库

CREATE TABLE IF NOT EXISTS `audit_logs` (
    `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    `user_id` INT NOT NULL COMMENT '操作用户ID',
    `username` VARCHAR(32) NOT NULL COMMENT '操作用户名',
    `action` VARCHAR(64) NOT NULL COMMENT '操作类型: create_merchant, delete_server, change_ip...',
    `target_type` VARCHAR(32) NOT NULL COMMENT '目标类型: merchant, server, cloud_account, oss_config, gost_server',
    `target_id` INT NOT NULL DEFAULT 0 COMMENT '目标ID',
    `target_name` VARCHAR(128) DEFAULT '' COMMENT '目标名称（便于显示）',
    `detail` TEXT COMMENT '操作详情（JSON格式）',
    `ip` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作IP',
    `user_agent` VARCHAR(512) DEFAULT '' COMMENT '浏览器UA',
    `status` VARCHAR(16) NOT NULL DEFAULT 'success' COMMENT '操作状态: success, failed',
    `error_msg` VARCHAR(512) DEFAULT '' COMMENT '错误信息（如果失败）',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '操作时间',
    PRIMARY KEY (`id`),
    INDEX `idx_user_id` (`user_id`),
    INDEX `idx_action` (`action`),
    INDEX `idx_target_type` (`target_type`),
    INDEX `idx_target_id` (`target_type`, `target_id`),
    INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='操作审计日志表';

-- 示例数据
-- INSERT INTO `audit_logs` (`user_id`, `username`, `action`, `target_type`, `target_id`, `target_name`, `detail`, `ip`) VALUES
-- (1, 'admin', 'create_merchant', 'merchant', 10, '测试商户', '{"port": 20000, "server_ip": "1.2.3.4"}', '192.168.1.1');