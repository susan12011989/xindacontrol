-- 006_resource_grouping.sql
-- 资源分组与标签功能

-- 资源标签表
CREATE TABLE IF NOT EXISTS `resource_tags` (
    `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(32) NOT NULL COMMENT '标签名',
    `color` VARCHAR(16) DEFAULT '' COMMENT '显示颜色',
    `description` VARCHAR(128) DEFAULT '' COMMENT '描述',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE INDEX `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源标签';

-- 资源-标签关联表（多对多）
CREATE TABLE IF NOT EXISTS `resource_tag_relations` (
    `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `tag_id` INT NOT NULL COMMENT '标签ID',
    `resource_type` VARCHAR(32) NOT NULL COMMENT '资源类型: oss_config / gost_server / storage_config',
    `resource_id` INT NOT NULL COMMENT '资源ID',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE INDEX `idx_tag_resource` (`tag_id`, `resource_type`, `resource_id`),
    INDEX `idx_resource` (`resource_type`, `resource_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='资源标签关联';
