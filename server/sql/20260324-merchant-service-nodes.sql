-- 商户服务节点表：支持单机/多机部署模式
-- 单机商户: 一条 role='all' 记录，host 为服务器 IP
-- 多机商户: 多条记录，按角色指向不同服务器

CREATE TABLE IF NOT EXISTS `merchant_service_nodes` (
    `id` INT NOT NULL AUTO_INCREMENT,
    `merchant_id` INT NOT NULL COMMENT '商户ID',
    `role` VARCHAR(32) NOT NULL COMMENT '服务角色: all, im, api, minio, web',
    `host` VARCHAR(128) NOT NULL COMMENT '服务器地址(IP或内网域名)',
    `server_id` INT NOT NULL DEFAULT 0 COMMENT '关联servers表ID(可选)',
    `is_primary` TINYINT NOT NULL DEFAULT 0 COMMENT '是否主节点(用于API调用和SSH管理)',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:0-停用 1-启用',
    `remark` VARCHAR(255) DEFAULT '' COMMENT '备注',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uk_merchant_server` (`merchant_id`, `server_id`),
    KEY `idx_merchant_id` (`merchant_id`),
    KEY `idx_server_id` (`server_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户服务节点(支持单机/多机部署)';

-- 为现有商户迁移数据：将 merchants.server_ip 写入 merchant_service_nodes（role=all）
INSERT INTO `merchant_service_nodes` (`merchant_id`, `role`, `host`, `is_primary`, `status`)
SELECT `id`, 'all', `server_ip`, 1, 1
FROM `merchants`
WHERE `server_ip` != '' AND `server_ip` IS NOT NULL
ON DUPLICATE KEY UPDATE `host` = VALUES(`host`);
