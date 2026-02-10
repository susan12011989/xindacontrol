-- 版本管理系统数据库迁移
-- 运行: mysql -u root -p control < 004_version_management.sql

-- 服务版本注册表
CREATE TABLE IF NOT EXISTS `service_versions` (
    `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `service_name` VARCHAR(32) NOT NULL COMMENT '服务名称:server/wukongim',
    `version` VARCHAR(64) NOT NULL COMMENT '版本号',
    `file_hash` VARCHAR(64) NOT NULL COMMENT '文件SHA256',
    `file_size` BIGINT NOT NULL DEFAULT 0 COMMENT '文件大小',
    `file_path` VARCHAR(255) NOT NULL COMMENT '存储路径',
    `changelog` TEXT COMMENT '更新日志',
    `is_current` TINYINT NOT NULL DEFAULT 0 COMMENT '是否当前版本',
    `uploaded_by` VARCHAR(32) DEFAULT '' COMMENT '上传者',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_service_name` (`service_name`),
    UNIQUE INDEX `idx_service_version` (`service_name`, `version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务版本注册表';

-- 部署记录表
CREATE TABLE IF NOT EXISTS `deployment_records` (
    `id` BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `server_id` INT NOT NULL COMMENT '服务器ID',
    `service_name` VARCHAR(32) NOT NULL COMMENT '服务名称',
    `version_id` INT NOT NULL COMMENT '版本ID',
    `previous_version_id` INT DEFAULT 0 COMMENT '上一版本ID',
    `action` VARCHAR(16) NOT NULL COMMENT '操作:deploy/rollback',
    `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态:0-进行中 1-成功 2-失败',
    `operator` VARCHAR(32) DEFAULT '' COMMENT '操作人',
    `backup_path` VARCHAR(255) DEFAULT '' COMMENT '备份路径',
    `output` TEXT COMMENT '执行输出',
    `started_at` DATETIME COMMENT '开始时间',
    `completed_at` DATETIME COMMENT '完成时间',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_server_id` (`server_id`),
    INDEX `idx_server_service` (`server_id`, `service_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部署记录表';

-- 确保版本存储目录存在（需要在服务器上手动执行）
-- mkdir -p /opt/control/versions/server
-- mkdir -p /opt/control/versions/wukongim
-- chown -R <运行用户> /opt/control/versions
