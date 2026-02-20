-- 007_cloud_instance_binding.sql
-- 云实例商户绑定

CREATE TABLE IF NOT EXISTS `cloud_instance_bindings` (
    `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `instance_id` VARCHAR(64) NOT NULL COMMENT '云实例ID',
    `region_id` VARCHAR(64) NOT NULL COMMENT '区域ID',
    `cloud_type` VARCHAR(16) NOT NULL DEFAULT 'aliyun' COMMENT '云类型',
    `merchant_id` INT NOT NULL COMMENT '商户ID',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE INDEX `idx_instance` (`instance_id`, `cloud_type`),
    INDEX `idx_merchant` (`merchant_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='云实例商户绑定';
