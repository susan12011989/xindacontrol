-- 打包中心模块数据库迁移文件
-- 用于管理商户客户端打包配置、构建任务和产物

-- 1. 商户打包配置表
CREATE TABLE IF NOT EXISTS `build_merchants` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `merchant_id` INT DEFAULT NULL COMMENT '关联商户ID（可选）',
  `name` VARCHAR(64) NOT NULL COMMENT '配置名称',
  `app_name` VARCHAR(64) NOT NULL COMMENT '应用名称',
  `short_name` VARCHAR(32) NOT NULL COMMENT '应用短名称',

  -- Android 配置
  `android_package` VARCHAR(128) NOT NULL COMMENT 'Android 包名',
  `android_version_code` INT DEFAULT 1 COMMENT '版本号',
  `android_version_name` VARCHAR(32) DEFAULT '1.0.0' COMMENT '版本名',

  -- iOS 配置
  `ios_bundle_id` VARCHAR(128) NOT NULL COMMENT 'iOS Bundle ID',
  `ios_version` VARCHAR(32) DEFAULT '1.0.0' COMMENT 'iOS 版本',
  `ios_build` VARCHAR(32) DEFAULT '1' COMMENT 'iOS Build',

  -- Windows 配置
  `windows_app_name` VARCHAR(64) DEFAULT '' COMMENT 'Windows 应用名',
  `windows_version` VARCHAR(32) DEFAULT '1.0.0' COMMENT 'Windows 版本',

  -- macOS 配置
  `macos_bundle_id` VARCHAR(128) DEFAULT '' COMMENT 'macOS Bundle ID',
  `macos_app_name` VARCHAR(64) DEFAULT '' COMMENT 'macOS 应用名',
  `macos_version` VARCHAR(32) DEFAULT '1.0.0' COMMENT 'macOS 版本',

  -- 服务器配置
  `server_api_url` VARCHAR(256) DEFAULT '' COMMENT 'API 地址',
  `server_ws_host` VARCHAR(128) DEFAULT '' COMMENT 'WebSocket 主机',
  `server_ws_port` INT DEFAULT 5100 COMMENT 'WebSocket 端口',
  `enterprise_code` VARCHAR(16) DEFAULT '' COMMENT '企业号（优先使用）',

  -- 资源文件（存储 OSS 路径）
  `icon_url` VARCHAR(512) DEFAULT '' COMMENT '图标 URL (1024x1024)',
  `logo_url` VARCHAR(512) DEFAULT '' COMMENT 'Logo URL',
  `splash_url` VARCHAR(512) DEFAULT '' COMMENT '启动图 URL',

  -- 签名配置（存储 OSS 路径）
  `android_keystore_url` VARCHAR(512) DEFAULT '' COMMENT 'Android keystore 文件 URL',
  `android_keystore_password` VARCHAR(128) DEFAULT '' COMMENT 'Keystore 密码（加密存储）',
  `android_key_alias` VARCHAR(64) DEFAULT '' COMMENT 'Key 别名',
  `android_key_password` VARCHAR(128) DEFAULT '' COMMENT 'Key 密码（加密存储）',

  -- 推送配置
  `push_mi_app_id` VARCHAR(64) DEFAULT '' COMMENT '小米推送 AppID',
  `push_mi_app_key` VARCHAR(64) DEFAULT '' COMMENT '小米推送 AppKey',
  `push_oppo_app_key` VARCHAR(64) DEFAULT '' COMMENT 'OPPO 推送 AppKey',
  `push_oppo_app_secret` VARCHAR(128) DEFAULT '' COMMENT 'OPPO 推送 AppSecret',
  `push_vivo_app_id` VARCHAR(64) DEFAULT '' COMMENT 'VIVO 推送 AppID',
  `push_vivo_app_key` VARCHAR(64) DEFAULT '' COMMENT 'VIVO 推送 AppKey',
  `push_hms_app_id` VARCHAR(64) DEFAULT '' COMMENT '华为推送 AppID',

  `description` VARCHAR(255) DEFAULT '' COMMENT '备注',
  `status` TINYINT DEFAULT 1 COMMENT '状态: 0-禁用 1-启用',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX `idx_merchant_id` (`merchant_id`),
  INDEX `idx_status` (`status`),
  INDEX `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='商户打包配置表';

-- 2. 构建任务表
CREATE TABLE IF NOT EXISTS `build_tasks` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `build_merchant_id` INT NOT NULL COMMENT '打包配置ID',
  `merchant_name` VARCHAR(64) DEFAULT '' COMMENT '商户名称（冗余，方便查询）',
  `platforms` VARCHAR(128) NOT NULL COMMENT '目标平台，逗号分隔: android,ios,windows,macos',
  `status` TINYINT DEFAULT 0 COMMENT '状态: 0-排队中 1-构建中 2-成功 3-失败 4-已取消',
  `progress` INT DEFAULT 0 COMMENT '进度 0-100',
  `current_step` VARCHAR(128) DEFAULT '' COMMENT '当前步骤描述',
  `operator` VARCHAR(32) NOT NULL COMMENT '操作人',

  -- 版本覆盖（可选，不填则使用配置中的版本）
  `override_android_version_code` INT DEFAULT NULL COMMENT '覆盖 Android 版本号',
  `override_android_version_name` VARCHAR(32) DEFAULT NULL COMMENT '覆盖 Android 版本名',
  `override_ios_version` VARCHAR(32) DEFAULT NULL COMMENT '覆盖 iOS 版本',
  `override_ios_build` VARCHAR(32) DEFAULT NULL COMMENT '覆盖 iOS Build',

  -- 时间记录
  `started_at` DATETIME DEFAULT NULL COMMENT '开始时间',
  `finished_at` DATETIME DEFAULT NULL COMMENT '完成时间',
  `duration` INT DEFAULT 0 COMMENT '耗时（秒）',

  -- 日志和错误
  `error_msg` TEXT COMMENT '错误信息',
  `log_url` VARCHAR(512) DEFAULT '' COMMENT '完整日志文件 URL',

  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,

  INDEX `idx_build_merchant_id` (`build_merchant_id`),
  INDEX `idx_status` (`status`),
  INDEX `idx_operator` (`operator`),
  INDEX `idx_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='构建任务表';

-- 3. 构建产物表
CREATE TABLE IF NOT EXISTS `build_artifacts` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `task_id` INT NOT NULL COMMENT '构建任务ID',
  `build_merchant_id` INT NOT NULL COMMENT '打包配置ID',
  `merchant_name` VARCHAR(64) DEFAULT '' COMMENT '商户名称',
  `platform` VARCHAR(32) NOT NULL COMMENT '平台: android/ios/windows/macos',
  `file_name` VARCHAR(256) NOT NULL COMMENT '文件名',
  `file_size` BIGINT DEFAULT 0 COMMENT '文件大小（字节）',
  `file_url` VARCHAR(512) NOT NULL COMMENT '文件下载地址',
  `version` VARCHAR(32) DEFAULT '' COMMENT '版本号',
  `expires_at` DATETIME NOT NULL COMMENT '过期时间（创建后24小时）',
  `download_count` INT DEFAULT 0 COMMENT '下载次数',
  `is_deleted` TINYINT DEFAULT 0 COMMENT '是否已删除: 0-否 1-是',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,

  INDEX `idx_task_id` (`task_id`),
  INDEX `idx_build_merchant_id` (`build_merchant_id`),
  INDEX `idx_platform` (`platform`),
  INDEX `idx_expires_at` (`expires_at`),
  INDEX `idx_is_deleted` (`is_deleted`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='构建产物表';

-- 4. 构建统计表（按日汇总）
CREATE TABLE IF NOT EXISTS `build_stats_daily` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `date` DATE NOT NULL COMMENT '统计日期',
  `total_builds` INT DEFAULT 0 COMMENT '总构建次数',
  `success_builds` INT DEFAULT 0 COMMENT '成功次数',
  `failed_builds` INT DEFAULT 0 COMMENT '失败次数',
  `cancelled_builds` INT DEFAULT 0 COMMENT '取消次数',
  `android_builds` INT DEFAULT 0 COMMENT 'Android 构建次数',
  `ios_builds` INT DEFAULT 0 COMMENT 'iOS 构建次数',
  `windows_builds` INT DEFAULT 0 COMMENT 'Windows 构建次数',
  `macos_builds` INT DEFAULT 0 COMMENT 'macOS 构建次数',
  `total_duration` INT DEFAULT 0 COMMENT '总耗时（秒）',
  `avg_duration` INT DEFAULT 0 COMMENT '平均耗时（秒）',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  UNIQUE KEY `uk_date` (`date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='构建统计日报表';

-- 5. 构建服务器表（支持多台构建机器）
CREATE TABLE IF NOT EXISTS `build_servers` (
  `id` INT AUTO_INCREMENT PRIMARY KEY,
  `name` VARCHAR(64) NOT NULL COMMENT '服务器名称',
  `host` VARCHAR(128) NOT NULL COMMENT '服务器地址',
  `port` INT DEFAULT 22 COMMENT 'SSH 端口',
  `username` VARCHAR(32) NOT NULL COMMENT 'SSH 用户名',
  `auth_type` TINYINT DEFAULT 1 COMMENT '认证方式: 1-密码 2-密钥',
  `password` VARCHAR(128) DEFAULT '' COMMENT 'SSH 密码（加密存储）',
  `private_key` TEXT COMMENT 'SSH 私钥',
  `work_dir` VARCHAR(255) DEFAULT '/opt/build' COMMENT '工作目录',
  `platforms` VARCHAR(128) DEFAULT 'android' COMMENT '支持的平台，逗号分隔',
  `max_concurrent` INT DEFAULT 1 COMMENT '最大并发构建数',
  `current_tasks` INT DEFAULT 0 COMMENT '当前任务数',
  `status` TINYINT DEFAULT 1 COMMENT '状态: 0-离线 1-在线 2-忙碌',
  `last_heartbeat` DATETIME DEFAULT NULL COMMENT '最后心跳时间',
  `description` VARCHAR(255) DEFAULT '' COMMENT '备注',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

  INDEX `idx_status` (`status`),
  INDEX `idx_platforms` (`platforms`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='构建服务器表';
