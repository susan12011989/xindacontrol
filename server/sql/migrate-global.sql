-- 全局管理相关表

-- OSS URL 管理表
CREATE TABLE IF NOT EXISTS `global_oss_url` (
  `id` int(11) NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `url` varchar(512) NOT NULL DEFAULT '' COMMENT 'OSS URL地址',
  `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  KEY `idx_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='全局OSS URL管理表';

