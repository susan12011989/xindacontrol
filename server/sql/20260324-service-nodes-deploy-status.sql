-- 商户服务节点增加部署状态字段
ALTER TABLE `merchant_service_nodes`
  ADD COLUMN `deploy_status` VARCHAR(32) DEFAULT '' COMMENT '部署状态: pending/deploying/success/failed',
  ADD COLUMN `deploy_error` TEXT COMMENT '部署错误信息',
  ADD COLUMN `deploy_output` TEXT COMMENT '部署输出摘要',
  ADD COLUMN `last_deploy_at` DATETIME DEFAULT NULL COMMENT '最近部署时间';
