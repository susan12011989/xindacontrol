-- TLS 证书按商户隔离：每个商户独立一套 CA + Server 证书
-- 添加 merchant_id 字段
ALTER TABLE `tls_certificates` ADD COLUMN `merchant_id` INT NOT NULL DEFAULT 0
  COMMENT '商户ID，0表示旧全局证书' AFTER `id`;

-- 删除旧唯一索引（name），改为 (name, merchant_id) 组合唯一
ALTER TABLE `tls_certificates` DROP INDEX `idx_name`;
ALTER TABLE `tls_certificates` ADD UNIQUE INDEX `idx_name_merchant` (`name`, `merchant_id`);
