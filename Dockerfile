# ============================================
# 多阶段构建 Dockerfile for Control Server
# ============================================

# Stage 1: 构建后端
FROM golang:1.22-alpine AS backend-builder

WORKDIR /build

# 安装必要的构建工具
RUN apk add --no-cache git

# 复制 go.mod 和 go.sum 先下载依赖（利用缓存）
COPY server/go.mod server/go.sum ./
RUN go mod download

# 复制源代码
COPY server/ .

# 编译后端
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=$(date +%Y%m%d) -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o server ./apps/server/main.go

# Stage 2: 构建前端
FROM node:20-alpine AS frontend-builder

WORKDIR /build

# 安装 pnpm
RUN npm install -g pnpm

# 复制 package.json 先下载依赖（利用缓存）
COPY front/package.json front/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

# 复制源代码
COPY front/ .

# 构建前端
RUN pnpm build

# Stage 3: 最终运行镜像
FROM alpine:3.19

WORKDIR /app

# 安装运行时依赖
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    && cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone

# 创建必要的目录
RUN mkdir -p /app/logs /app/assets /app/versions /app/dist

# 从构建阶段复制文件
COPY --from=backend-builder /build/server /app/server
COPY --from=frontend-builder /build/dist /app/dist

# 复制配置文件模板
COPY server/config.yaml.docker /app/config.yaml

# 设置权限
RUN chmod +x /app/server

# 暴露端口
EXPOSE 58181

# 健康检查
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:58181/ping || exit 1

# 环境变量（可被 docker-compose 或运行时覆盖）
ENV CONFIG_PATH=/app/config.yaml \
    DB_HOST=127.0.0.1:3306 \
    DB_USER=root \
    DB_PASSWORD= \
    DB_NAME=control \
    REDIS_HOST=127.0.0.1:6379 \
    REDIS_PASSWORD= \
    REDIS_DB=10 \
    LISTEN_ADDR=0.0.0.0:58181 \
    ASSETS_DIR=/app/assets \
    VERSIONS_DIR=/app/versions \
    LOG_PATH=/app/logs

# 数据卷
VOLUME ["/app/logs", "/app/assets", "/app/versions"]

# 启动命令
CMD ["/app/server"]
