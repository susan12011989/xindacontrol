ALTER TABLE `servers`
  ADD COLUMN `forward_type` TINYINT NOT NULL DEFAULT 1 COMMENT '转发类型:1-加密(relay+tls) 2-直连(tcp)' AFTER
  `server_type`;