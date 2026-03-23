package gostapi

import "fmt"

// MerchantNginxConfigTemplate 生成商户服务器的 nginx 配置
// GOST(10443) → nginx(8080) → 按路径分发到各业务程序
func MerchantNginxConfigTemplate() string {
	return fmt.Sprintf(`# 商户服务器 nginx 路径分发配置
# GOST relay+tls (:%d) → nginx (:%d) → 业务程序
# 由 tsdd-control 自动生成

server {
    listen %d;

    # WebSocket 长连接 → WuKongIM
    # App/Web: wss://系统服务器:443/ws
    location /ws {
        proxy_pass http://127.0.0.1:%d;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
    }

    # HTTP API → tsdd-server
    # App/Web/PC: https://系统服务器:443/api/v1/...
    location /api/ {
        proxy_pass http://127.0.0.1:%d/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        client_max_body_size 1000m;
        client_body_buffer_size 500m;
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    # MinIO S3 → 图片/文件上传下载
    # App/Web: https://系统服务器:443/s3/...
    location /s3/ {
        proxy_pass http://127.0.0.1:%d/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        client_max_body_size 1000m;
        client_body_buffer_size 500m;
        proxy_connect_timeout 60s;
        proxy_send_timeout 300s;
        proxy_read_timeout 300s;
    }

    # WuKongIM 管理后台（可选，仅内部使用）
    location /manager/ {
        proxy_pass http://127.0.0.1:%d/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # 健康检查
    location /health {
        return 200 'ok';
        add_header Content-Type text/plain;
    }
}
`,
		MerchantUnifiedPort, MerchantNginxPort,
		MerchantNginxPort,
		MerchantAppPortWS,
		MerchantAppPortHTTP,
		MerchantAppPortMinIO,
		MerchantAppPortWKMgr,
	)
}

// SystemNginxConfigTemplate 生成系统服务器的 nginx TLS 终结 + 缓存配置
// 所有客户端 → 443(TLS) → nginx(缓存+路径分发) → GOST relay+tls → 商户
// 图片/视频等静态文件在系统服务器 nginx 层缓存，命中后不走 GOST 链路
func SystemNginxConfigTemplate(certPath, keyPath string, gostRelayPort int) string {
	return fmt.Sprintf(`# 系统服务器 TLS 终结 + 媒体缓存 + 路径分发
# App/Web/PC → :443 (TLS) → nginx → GOST relay+tls → 商户
# 图片/文件缓存在系统服务器，减少 GOST 链路负载
# 由 tsdd-control 自动生成

# 缓存配置（图片/视频 7 天，最大 2GB）
proxy_cache_path /var/cache/nginx/media_cache
    levels=1:2
    keys_zone=media_cache:20m
    max_size=2g
    inactive=7d
    use_temp_path=off;

server {
    listen 443 ssl;

    ssl_certificate     %s;
    ssl_certificate_key %s;
    ssl_protocols       TLSv1.2 TLSv1.3;
    ssl_ciphers         HIGH:!aNULL:!MD5;

    # WebSocket 长连接 → GOST relay → 商户 WS
    location /ws {
        proxy_pass http://127.0.0.1:%d;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_read_timeout 86400;
        proxy_send_timeout 86400;
    }

    # HTTP API → GOST relay → 商户 tsdd-server
    location /api/ {
        proxy_pass http://127.0.0.1:%d;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-Proto https;
        client_max_body_size 1000m;
        client_body_buffer_size 500m;
    }

    # MinIO S3 — 图片/文件（带缓存）
    # 下载命中缓存直接返回，不走 GOST 链路，大幅降低延迟
    location /s3/ {
        proxy_pass http://127.0.0.1:%d;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        client_max_body_size 1000m;
        client_body_buffer_size 500m;

        # 仅缓存 GET 请求（下载），不缓存 PUT（上传）
        proxy_cache media_cache;
        proxy_cache_methods GET HEAD;
        proxy_cache_valid 200 7d;
        proxy_cache_valid 304 7d;
        proxy_cache_key $uri$is_args$args;
        proxy_cache_use_stale error timeout updating http_500 http_502 http_503 http_504;
        proxy_ignore_headers Cache-Control Expires Set-Cookie;
        add_header X-Cache-Status $upstream_cache_status;

        # 防缓存击穿：同一 URL 并发请求时，只放 1 个穿透到后端
        # 其余请求等待第 1 个完成后直接拿缓存结果
        # 2 万人群同时下载同一张图 → 只有 1 个请求走 GOST，其余等缓存
        proxy_cache_lock on;
        proxy_cache_lock_timeout 10s;
        proxy_cache_lock_age 15s;

        # PUT/POST 请求（上传）不缓存，直接透传
        proxy_no_cache $request_method;
        proxy_cache_bypass $request_method;
    }

    # 健康检查
    location /health {
        return 200 'ok';
        add_header Content-Type text/plain;
    }
}
`,
		certPath, keyPath,
		gostRelayPort, // /ws → GOST local relay
		gostRelayPort, // /api → GOST local relay
		gostRelayPort, // /s3 → GOST local relay (缓存层)
	)
}
