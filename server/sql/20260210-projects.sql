-- 项目管理表
-- 用于按项目管理商户和 GOST 服务器

-- 项目表
CREATE TABLE IF NOT EXISTS `projects` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(64) NOT NULL COMMENT '项目名称',
  `description` VARCHAR(255) DEFAULT '' COMMENT '项目描述',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 0-禁用 1-启用',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='项目表';

-- 项目 GOST 服务器关联表
CREATE TABLE IF NOT EXISTS `project_gost_servers` (
  `id` INT NOT NULL AUTO_INCREMENT,
  `project_id` INT NOT NULL COMMENT '项目ID',
  `server_id` INT NOT NULL COMMENT '服务器ID',
  `is_primary` TINYINT NOT NULL DEFAULT 0 COMMENT '是否主服务器',
  `priority` INT NOT NULL DEFAULT 0 COMMENT '优先级,数字越小越高',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 0-禁用 1-启用',
  `remark` VARCHAR(255) DEFAULT '' COMMENT '备注',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_project_server` (`project_id`, `server_id`),
  KEY `idx_project_id` (`project_id`),
  KEY `idx_server_id` (`server_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='项目GOST服务器关联表';

-- 商户表添加 project_id 字段
ALTER TABLE `merchants` ADD COLUMN `project_id` INT DEFAULT 0 COMMENT '所属项目ID' AFTER `id`;
ALTER TABLE `merchants` ADD INDEX `idx_project_id` (`project_id`);
