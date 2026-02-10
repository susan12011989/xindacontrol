-- 系统云账号表
CREATE TABLE IF NOT EXISTS `cloud_accounts` (
  `id` BIGINT(20) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `account_type` VARCHAR(32) NOT NULL COMMENT '账号类型: system, merchant',
  `merchant_id` INT DEFAULT NULL COMMENT '商户ID',
  `name` VARCHAR(100) NOT NULL COMMENT '账号名称',
  `cloud_type` VARCHAR(20) NOT NULL COMMENT '云类型: aliyun, aws',
  `access_key_id` VARCHAR(255) NOT NULL COMMENT 'AccessKeyId',
  `access_key_secret` VARCHAR(255) NOT NULL COMMENT 'AccessKeySecret',
  `region` VARCHAR(50) DEFAULT '' COMMENT '默认区域',
  `description` VARCHAR(500) DEFAULT '' COMMENT '描述',
  `status` TINYINT(1) DEFAULT 1 COMMENT '状态: 0-禁用 1-启用',
  `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  INDEX `idx_cloud_type` (`cloud_type`),
  INDEX `idx_status` (`status`),
  INDEX `idx_account_type` (`account_type`),
  INDEX `idx_merchant_id` (`merchant_id`),
  UNIQUE KEY `uq_cloud_account_scope` (`account_type`, `merchant_id`, `cloud_type`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统云账号表';

