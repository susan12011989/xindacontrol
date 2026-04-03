-- 商户服务节点增加部署参数字段（WuKongIM 集群配置 + 远程连接地址）
ALTER TABLE `merchant_service_nodes`
  ADD COLUMN `wk_node_id` INT DEFAULT 0 COMMENT 'WuKongIM 节点ID',
  ADD COLUMN `db_host` VARCHAR(128) DEFAULT '' COMMENT 'DB节点内网IP',
  ADD COLUMN `minio_host` VARCHAR(128) DEFAULT '' COMMENT 'MinIO节点内网IP';
