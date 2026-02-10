-- 为管理员表添加2FA支持
-- 执行时间: 2024年

ALTER TABLE `server_users` 
ADD COLUMN `two_factor_secret` varchar(32) DEFAULT '' COMMENT '2FA密钥(Base32编码)' AFTER `password`,
ADD COLUMN `two_factor_enabled` tinyint(1) DEFAULT 0 COMMENT '是否启用2FA' AFTER `two_factor_secret`,
ADD INDEX `idx_two_factor_enabled` (`two_factor_enabled`);

-- 说明：
-- two_factor_secret: 存储TOTP的Base32编码密钥，使用Google Authenticator等应用扫码绑定
-- two_factor_enabled: 0=未启用，1=已启用

