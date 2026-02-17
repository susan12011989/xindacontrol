-- TLS 证书管理
-- 运行: mysql -u root -p control < 005_tls_certificate.sql

-- TLS 证书表（存储 CA 和服务器证书）
CREATE TABLE IF NOT EXISTS `tls_certificates` (
    `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `name` VARCHAR(64) NOT NULL COMMENT '证书名称，如 gost-ca, gost-server',
    `cert_type` TINYINT NOT NULL DEFAULT 1 COMMENT '证书类型:1-CA根证书 2-服务器证书',
    `cert_pem` TEXT NOT NULL COMMENT '证书内容(PEM格式)',
    `key_pem` TEXT NOT NULL COMMENT '私钥内容(PEM格式)',
    `fingerprint` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '证书SHA-256指纹(供App端Pinning)',
    `expires_at` DATETIME NOT NULL COMMENT '过期时间',
    `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态:0-停用 1-启用',
    `created_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `updated_at` DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE INDEX `idx_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='TLS证书管理表';

-- 服务器表增加 TLS 状态字段
ALTER TABLE `servers` ADD COLUMN `tls_enabled` TINYINT NOT NULL DEFAULT 0 COMMENT '客户端TLS:0-未启用 1-已启用' AFTER `status`;
ALTER TABLE `servers` ADD COLUMN `tls_deployed_at` DATETIME DEFAULT NULL COMMENT 'TLS证书部署时间' AFTER `tls_enabled`;