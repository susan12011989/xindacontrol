package deploy

import (
	"fmt"
	"server/internal/server/model"
)

// GenerateComposeByRole 根据节点角色生成对应的 docker-compose.yml
func GenerateComposeByRole(config model.DeployConfig) string {
	switch config.NodeRole {
	case "db":
		return generateDBCompose(config)
	case "app":
		return generateAppCompose(config)
	default: // "allinone" 或空
		return generateAllinoneCompose(config)
	}
}

// GenerateEnvByRole 根据节点角色生成 .env 文件
func GenerateEnvByRole(config model.DeployConfig) string {
	switch config.NodeRole {
	case "db":
		return generateDBEnv(config)
	case "app":
		return generateAppEnv(config)
	default:
		return generateAllinoneEnv(config)
	}
}

// ==================== DB 节点 ====================
// 只运行 MySQL + Redis + MinIO，监听内网供 App 节点访问

func generateDBCompose(config model.DeployConfig) string {
	return `version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: tsdd-mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: tsdd
    volumes:
      - /data/db/mysql:/var/lib/mysql
    ports:
      - "3306:3306"
    command: >
      --character-set-server=utf8mb4
      --collation-server=utf8mb4_unicode_ci
      --default-time-zone=+08:00
      --innodb-buffer-pool-size=8G
      --max-connections=500
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: tsdd-redis
    restart: always
    volumes:
      - /data/db/redis:/data
    ports:
      - "6379:6379"
    command: redis-server --maxmemory 4gb --maxmemory-policy allkeys-lru
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    container_name: tsdd-minio
    restart: always
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - /data/minio:/data
    ports:
      - "9000:9000"
      - "9001:9001"
    command: server /data --console-address ":9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3
`
}

func generateDBEnv(config model.DeployConfig) string {
	return fmt.Sprintf(`MYSQL_ROOT_PASSWORD=%s
MINIO_ROOT_USER=%s
MINIO_ROOT_PASSWORD=%s
`, config.MySQLPassword, config.MinioUser, config.MinioPassword)
}

// ==================== App 节点 ====================
// 运行 WuKongIM(集群模式) + tsdd-server + web + manager
// 连接远程 DB 节点的 MySQL/Redis/MinIO

func generateAppCompose(config model.DeployConfig) string {
	dbHost := config.DBHost
	redisHost := config.RedisHost
	minioHost := config.MinioHost
	if dbHost == "" {
		dbHost = "127.0.0.1"
	}
	if redisHost == "" {
		redisHost = dbHost
	}
	if minioHost == "" {
		minioHost = dbHost
	}

	// WuKongIM 集群环境变量
	wkClusterEnv := ""
	if config.WKNodeId > 0 {
		wkClusterEnv = fmt.Sprintf(`      WK_CLUSTER_NODEID: "%d"
      WK_CLUSTER_ADDR: "tcp://0.0.0.0:11110"
      WK_CLUSTER_SERVERADDR: "${PRIVATE_IP}:11110"
      WK_CLUSTER_APIURL: "http://${PRIVATE_IP}:5002"`, config.WKNodeId)

		if config.WKSeedNode != "" {
			// 加入已有集群
			wkClusterEnv += fmt.Sprintf(`
      WK_CLUSTER_SEED: "%s"`, config.WKSeedNode)
		} else {
			// 首个节点，使用 initNodes 初始化
			wkClusterEnv += fmt.Sprintf(`
      WK_CLUSTER_INITNODES: "%d@${PRIVATE_IP}:11110"`, config.WKNodeId)
		}

		wkClusterEnv += `
      WK_CLUSTER_SLOTCOUNT: "128"
      WK_CLUSTER_SLOTREPLICACOUNT: "3"
      WK_CLUSTER_CHANNELREPLICACOUNT: "3"`
	}

	// tsdd-server 的 Control 面板回调配置
	controlEnv := ""
	if config.ControlAPIUsername != "" {
		controlEnv = fmt.Sprintf(`      TS_CONTROL_APIUSERNAME: "%s"
      TS_CONTROL_APIPASSWORD: "%s"`, config.ControlAPIUsername, config.ControlAPIPassword)
	}

	return fmt.Sprintf(`version: '3.8'

services:
  wukongim:
    image: registry.cn-shanghai.aliyuncs.com/wukongim/wukongim:latest
    container_name: tsdd-wukongim
    restart: always
    network_mode: host
    environment:
      WK_MODE: release
      WK_ADDR: "tcp://0.0.0.0:5001"
      WK_HTTPADDR: "0.0.0.0:5002"
      WK_EXTERNAL_IP: ${EXTERNAL_IP}
      WK_EXTERNAL_TCP_ADDR: "${EXTERNAL_IP}:10000"
      WK_EXTERNAL_WS_ADDR: "ws://${EXTERNAL_IP}:10003"
      WK_WEBHOOK_GRPCADDR: "127.0.0.1:6979"
      WK_DATASOURCE_ADDR: "http://127.0.0.1:%d/v1/datasource"
%s
    volumes:
      - /data/db/wukongim:/root/wukongim

  tsdd-server:
    image: tsdd-server-local:latest
    container_name: tsdd-server
    restart: always
    network_mode: host
    environment:
      TS_MODE: release
      TS_ADDR: ":%d"
      TS_EXTERNAL_IP: ${EXTERNAL_IP}
      TS_EXTERNAL_BASEURL: "http://${EXTERNAL_IP}:%d"
      TS_DB_MYSQLADDR: "root:${MYSQL_ROOT_PASSWORD}@tcp(%s:3306)/tsdd?charset=utf8mb4&parseTime=true&loc=Local"
      TS_DB_REDISADDR: "%s:6379"
      TS_WUKONGIM_APIURL: "http://127.0.0.1:5002"
      TS_MINIO_URL: "http://%s:9000"
      TS_MINIO_ACCESSKEYID: ${MINIO_ROOT_USER}
      TS_MINIO_SECRETACCESSKEY: ${MINIO_ROOT_PASSWORD}
      TS_MINIO_UPLOADURL: "http://%s:9000"
%s
    volumes:
      - /opt/tsdd/assets:/home/tsdd/tsdd/assets:ro
      - /opt/tsdd/TangSengDaoDaoServer:/home/tsdd/tsdd/TangSengDaoDaoServer
    depends_on:
      - wukongim

  web:
    image: tsdd-web:custom
    container_name: tsdd-web
    restart: always
    environment:
      API_URL: "http://${EXTERNAL_IP}:%d"
      WS_URL: "ws://${EXTERNAL_IP}:5200"
    ports:
      - "82:80"
      - "443:443"
    volumes:
      - /opt/tsdd/ssl:/etc/nginx/ssl:ro
      - /opt/tsdd/nginx.conf.template:/nginx.conf.template:ro
      - /opt/tsdd/web:/usr/share/nginx/html:ro
    depends_on:
      - tsdd-server

  manager:
    image: tsdd-manager:custom
    container_name: tsdd-manager
    restart: always
    environment:
      API_URL: "http://127.0.0.1:%d/v1/"
    ports:
      - "%d:80"
    depends_on:
      - tsdd-server
`,
		config.APIPort, // WK_DATASOURCE_ADDR
		wkClusterEnv,   // WuKongIM cluster env
		config.APIPort, // TS_ADDR
		config.APIPort, // TS_EXTERNAL_BASEURL
		dbHost,         // MySQL host
		redisHost,      // Redis host
		minioHost,      // MinIO URL
		minioHost,      // MinIO upload URL
		controlEnv,     // Control API
		config.APIPort, // web API_URL
		config.APIPort, // manager API_URL
		config.ManagerPort,
	)
}

func generateAppEnv(config model.DeployConfig) string {
	return fmt.Sprintf(`EXTERNAL_IP=%s
PRIVATE_IP=%s
MYSQL_ROOT_PASSWORD=%s
MINIO_ROOT_USER=%s
MINIO_ROOT_PASSWORD=%s
`, config.ExternalIP, config.DBHost, config.MySQLPassword, config.MinioUser, config.MinioPassword)
}

// ==================== All-in-one 节点 ====================
// 所有服务在同一台机器上，支持可选的 WuKongIM 集群配置（为后续扩容做准备）

func generateAllinoneCompose(config model.DeployConfig) string {
	// WuKongIM 集群环境变量（可选）
	wkClusterEnv := ""
	if config.WKNodeId > 0 {
		wkClusterEnv = fmt.Sprintf(`      WK_CLUSTER_NODEID: "%d"
      WK_CLUSTER_ADDR: "tcp://0.0.0.0:11110"
      WK_CLUSTER_SERVERADDR: "${PRIVATE_IP}:11110"
      WK_CLUSTER_APIURL: "http://${PRIVATE_IP}:5002"
      WK_CLUSTER_INITNODES: "%d@${PRIVATE_IP}:11110"
      WK_CLUSTER_SLOTCOUNT: "128"
      WK_CLUSTER_SLOTREPLICACOUNT: "3"
      WK_CLUSTER_CHANNELREPLICACOUNT: "3"`, config.WKNodeId, config.WKNodeId)
	}

	controlEnv := ""
	if config.ControlAPIUsername != "" {
		controlEnv = fmt.Sprintf(`      TS_CONTROL_APIUSERNAME: "%s"
      TS_CONTROL_APIPASSWORD: "%s"`, config.ControlAPIUsername, config.ControlAPIPassword)
	}

	return fmt.Sprintf(`version: '3.8'

services:
  mysql:
    image: mysql:8.0
    container_name: tsdd-mysql
    restart: always
    environment:
      MYSQL_ROOT_PASSWORD: ${MYSQL_ROOT_PASSWORD}
      MYSQL_DATABASE: tsdd
    volumes:
      - /data/db/mysql:/var/lib/mysql
    ports:
      - "3306:3306"
    command: >
      --character-set-server=utf8mb4
      --collation-server=utf8mb4_unicode_ci
      --default-time-zone=+08:00
    healthcheck:
      test: ["CMD", "mysqladmin", "ping", "-h", "localhost"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    container_name: tsdd-redis
    restart: always
    volumes:
      - /data/db/redis:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    container_name: tsdd-minio
    restart: always
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - /data/minio:/data
    ports:
      - "9000:9000"
      - "9001:9001"
    command: server /data --console-address ":9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 30s
      timeout: 20s
      retries: 3

  wukongim:
    image: registry.cn-shanghai.aliyuncs.com/wukongim/wukongim:latest
    container_name: tsdd-wukongim
    restart: always
    environment:
      WK_MODE: release
      WK_ADDR: "tcp://0.0.0.0:5001"
      WK_HTTPADDR: "0.0.0.0:5002"
      WK_EXTERNAL_IP: ${EXTERNAL_IP}
      WK_EXTERNAL_TCP_ADDR: "${EXTERNAL_IP}:10000"
      WK_EXTERNAL_WS_ADDR: "ws://${EXTERNAL_IP}:10003"
      WK_WEBHOOK_GRPCADDR: "tsdd-server:6979"
      WK_DATASOURCE_ADDR: "http://tsdd-server:%d/v1/datasource"
%s
    volumes:
      - /data/db/wukongim:/root/wukongim
    ports:
      - "5001:5001"
      - "5002:5002"
      - "5200:5200"
      - "5300:5300"
      - "10000:10000"
      - "10003:10003"
      - "11110:11110"
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy

  tsdd-server:
    image: tsdd-server-local:latest
    container_name: tsdd-server
    restart: always
    environment:
      TS_MODE: release
      TS_ADDR: ":%d"
      TS_EXTERNAL_IP: ${EXTERNAL_IP}
      TS_EXTERNAL_BASEURL: "http://${EXTERNAL_IP}:%d"
      TS_DB_MYSQLADDR: "root:${MYSQL_ROOT_PASSWORD}@tcp(mysql:3306)/tsdd?charset=utf8mb4&parseTime=true&loc=Local"
      TS_DB_REDISADDR: "redis:6379"
      TS_WUKONGIM_APIURL: "http://wukongim:5002"
      TS_MINIO_URL: "http://minio:9000"
      TS_MINIO_ACCESSKEYID: ${MINIO_ROOT_USER}
      TS_MINIO_SECRETACCESSKEY: ${MINIO_ROOT_PASSWORD}
      TS_MINIO_UPLOADURL: "http://minio:9000"
%s
    ports:
      - "%d:%d"
      - "6979:6979"
    depends_on:
      mysql:
        condition: service_healthy
      redis:
        condition: service_healthy
      minio:
        condition: service_healthy
      wukongim:
        condition: service_started
    volumes:
      - /opt/tsdd/assets:/home/tsdd/tsdd/assets:ro
      - /opt/tsdd/TangSengDaoDaoServer:/home/tsdd/tsdd/TangSengDaoDaoServer

  manager:
    image: tsdd-manager:custom
    container_name: tsdd-manager
    restart: always
    environment:
      API_URL: "http://tsdd-server:%d/v1/"
    ports:
      - "%d:80"
    depends_on:
      - tsdd-server

  web:
    image: tsdd-web:custom
    container_name: tsdd-web
    restart: always
    environment:
      API_URL: "http://${EXTERNAL_IP}:%d"
      WS_URL: "ws://${EXTERNAL_IP}:5200"
    ports:
      - "%d:80"
      - "443:443"
    volumes:
      - /opt/tsdd/ssl:/etc/nginx/ssl:ro
      - /opt/tsdd/nginx.conf.template:/nginx.conf.template:ro
      - /opt/tsdd/web:/usr/share/nginx/html:ro
    depends_on:
      - tsdd-server
`,
		config.APIPort,    // WK_DATASOURCE_ADDR
		wkClusterEnv,      // WuKongIM cluster env
		config.APIPort,    // TS_ADDR
		config.APIPort,    // TS_EXTERNAL_BASEURL
		controlEnv,        // Control API
		config.APIPort,    // host port mapping
		config.APIPort,    // container port mapping
		config.APIPort,    // manager API_URL
		config.ManagerPort,
		config.APIPort,    // web API_URL
		config.WebPort,
	)
}

func generateAllinoneEnv(config model.DeployConfig) string {
	privateIP := config.ExternalIP
	if config.DBHost != "" {
		privateIP = config.DBHost
	}
	return fmt.Sprintf(`EXTERNAL_IP=%s
PRIVATE_IP=%s
MYSQL_ROOT_PASSWORD=%s
MINIO_ROOT_USER=%s
MINIO_ROOT_PASSWORD=%s
ADMIN_PASSWORD=%s
SMS_CODE=%s
`, config.ExternalIP, privateIP, config.MySQLPassword, config.MinioUser, config.MinioPassword, config.AdminPassword, config.SMSCode)
}
