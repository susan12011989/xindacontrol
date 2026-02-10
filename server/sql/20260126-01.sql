-- 为 servers 表添加辅助IP字段（仅系统服务器使用，用于IP批量内嵌上传）
ALTER TABLE `servers` ADD COLUMN `auxiliary_ip` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '辅助IP,仅系统服务器使用' AFTER `host`;

-- IP嵌入选择记录表（用于记录工具页选择的IP，每次覆盖保存）
CREATE TABLE IF NOT EXISTS `ip_embed_selections` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `key_name` VARCHAR(64) NOT NULL COMMENT '配置键名',
  `selected_ips` TEXT COMMENT '选中的IP列表(JSON数组)',
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_key_name` (`key_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='IP嵌入选择记录';
