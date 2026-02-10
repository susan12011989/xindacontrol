-- 服务器运维管理模块数据库迁移文件

-- 1. 服务器配置表
CREATE TABLE IF NOT EXISTS `servers` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `server_type` TINYINT NOT NULL DEFAULT 1 COMMENT '服务器类型: 1-商户服务器 2-系统服务器',
  `merchant_id` INT DEFAULT NULL COMMENT '商户ID（商户服务器必填，系统服务器为NULL）',
  `name` VARCHAR(64) NOT NULL COMMENT '服务器名称',
  `host` VARCHAR(128) NOT NULL COMMENT '服务器地址',
  `port` INT NOT NULL DEFAULT 22 COMMENT 'SSH端口',
  `username` VARCHAR(32) NOT NULL COMMENT 'SSH用户名',
  `auth_type` TINYINT NOT NULL DEFAULT 1 COMMENT '认证方式: 1-密码 2-密钥',
  `password` VARCHAR(128) DEFAULT '' COMMENT 'SSH密码',
  `private_key` TEXT COMMENT 'SSH私钥',
  `deploy_path` VARCHAR(255) DEFAULT '/opt/teamgram/bin' COMMENT '部署目录',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 0-禁用 1-启用',
  `description` VARCHAR(255) DEFAULT '' COMMENT '描述',
  `tags` VARCHAR(255) DEFAULT '' COMMENT '标签',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_server_type` (`server_type`),
  INDEX `idx_merchant_id` (`merchant_id`),
  INDEX `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='服务器配置表';

-- 2. 部署配置表
CREATE TABLE IF NOT EXISTS `deploy_configs` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `server_id` INT NOT NULL COMMENT '服务器ID',
  `name` VARCHAR(64) NOT NULL COMMENT '配置名称',
  `service_name` VARCHAR(64) NOT NULL COMMENT '服务名称',
  `deploy_path` VARCHAR(255) NOT NULL COMMENT '部署目录',
  `start_command` TEXT NOT NULL COMMENT '启动命令',
  `stop_command` TEXT NOT NULL COMMENT '停止命令',
  `restart_command` TEXT COMMENT '重启命令',
  `status_command` TEXT COMMENT '状态查询命令',
  `log_path` VARCHAR(255) COMMENT '日志路径',
  `pre_deploy_script` TEXT COMMENT '部署前脚本',
  `post_deploy_script` TEXT COMMENT '部署后脚本',
  `env_vars` TEXT COMMENT '环境变量(JSON)',
  `start_order` INT NOT NULL DEFAULT 0 COMMENT '启动顺序',
  `sleep_after` INT NOT NULL DEFAULT 1 COMMENT '启动后等待秒数',
  `service_group` VARCHAR(32) DEFAULT '' COMMENT '服务分组',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX `idx_server_id` (`server_id`),
  INDEX `idx_start_order` (`start_order`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部署配置表';

-- 3. 部署历史表
CREATE TABLE IF NOT EXISTS `deploy_history` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `server_id` INT NOT NULL COMMENT '服务器ID',
  `config_id` INT DEFAULT NULL COMMENT '配置ID',
  `action` VARCHAR(32) NOT NULL COMMENT '操作类型: start_all/stop_all/restart_all/start/stop/restart/status/logs',
  `service_name` VARCHAR(64) DEFAULT '' COMMENT '服务名称',
  `operator` VARCHAR(32) NOT NULL COMMENT '操作人',
  `status` TINYINT NOT NULL DEFAULT 0 COMMENT '状态: 0-执行中 1-成功 2-失败',
  `output` TEXT COMMENT '执行输出',
  `error_msg` TEXT COMMENT '错误信息',
  `duration` INT DEFAULT 0 COMMENT '执行时长(秒)',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX `idx_server_id` (`server_id`),
  INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部署历史表';

-- 4. Docker操作历史表
CREATE TABLE IF NOT EXISTS `docker_operation_history` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `server_id` INT NOT NULL COMMENT '服务器ID',
  `merchant_id` INT NOT NULL COMMENT '商户ID',
  `container_id` VARCHAR(64) NOT NULL COMMENT '容器ID',
  `container_name` VARCHAR(128) DEFAULT '' COMMENT '容器名称',
  `action` VARCHAR(32) NOT NULL COMMENT '操作: start/stop/restart/remove/logs',
  `operator` VARCHAR(32) NOT NULL COMMENT '操作人',
  `params` TEXT COMMENT '操作参数(JSON)',
  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1-成功 2-失败',
  `output` TEXT COMMENT '执行输出',
  `error_msg` TEXT COMMENT '错误信息',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  INDEX `idx_server_id` (`server_id`),
  INDEX `idx_merchant_id` (`merchant_id`),
  INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='Docker操作历史表';

ALTER TABLE control.merchants ADD port INT NOT NULL COMMENT '占用端口';
ALTER TABLE control.merchants ADD CONSTRAINT port_unique UNIQUE KEY (port);
ALTER TABLE control.admin_users ADD `role` varchar(32) DEFAULT NULL COMMENT '用户角色';

-- 5. CloudAccounts 表新增字段与索引
ALTER TABLE control.cloud_accounts ADD `account_type` VARCHAR(32) NOT NULL COMMENT '账号类型: system, merchant';
ALTER TABLE control.cloud_accounts ADD `merchant_id` INT DEFAULT NULL COMMENT '商户ID';
ALTER TABLE control.cloud_accounts ADD INDEX `idx_account_type` (`account_type`);
ALTER TABLE control.cloud_accounts ADD INDEX `idx_merchant_id` (`merchant_id`);
ALTER TABLE control.cloud_accounts ADD UNIQUE KEY `uq_cloud_account_scope` (`account_type`, `merchant_id`, `cloud_type`);

ALTER TABLE control.merchants ADD server_ip varchar(32) NULL COMMENT '服务器ip';

ALTER TABLE control.cloud_accounts DROP INDEX uq_cloud_account_scope;
