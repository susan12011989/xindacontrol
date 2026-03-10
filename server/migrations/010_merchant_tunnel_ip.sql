-- 商户隧道 IP（系统服务器上为每个商户分配的独立 IP，用于多商户端口隔离）
ALTER TABLE merchants ADD COLUMN tunnel_ip VARCHAR(128) DEFAULT '' COMMENT '隧道IP-系统服务器分配' AFTER port;
