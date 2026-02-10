-- 为云账号表添加站点类型字段（区分阿里云国内站/国际站）
ALTER TABLE cloud_accounts
ADD COLUMN site_type VARCHAR(10) DEFAULT 'cn' COMMENT '站点类型: cn-国内站, intl-国际站'
AFTER cloud_type;
