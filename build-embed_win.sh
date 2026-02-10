#!/bin/bash


set -e  # 遇到错误立即退出

echo "======================================"
echo " 总控后台内嵌式构建"
echo "======================================"
echo ""

# 1. 构建前端
echo ">>> 步骤 1/4: 构建前端项目..."
cd front

# 检查是否存在 .env.production 文件
if [ ! -f ".env.production" ]; then
    echo "创建 .env.production 文件..."
    cat > .env.production << 'EOF'
# 生产环境配置

# 页面标题
VITE_APP_TITLE = "总控后台"

# 公共基础路径（默认 / ）
VITE_PUBLIC_PATH = "/"

# API 基础路径（与后端的 /server/v1 前缀保持一致）
VITE_BASE_URL = "/server/v1"
EOF
fi

# 检查是否安装了依赖
if [ ! -d "node_modules" ]; then
    echo "安装前端依赖..."
    pnpm install
fi

# 构建前端
echo "正在构建前端..."
pnpm build

echo "✓ 前端构建完成"
echo ""

# 2. 复制前端构建产物到后端embed目录
echo ">>> 步骤 2/4: 复制前端构建产物..."
cd ..

# 删除旧的embed dist目录
rm -rf server/internal/server/static/dist

# 复制新的构建产物
cp -r front/dist server/internal/server/static/

echo "✓ 构建产物复制完成"
echo ""

# 3. 构建后端
echo ">>> 步骤 3/4: 构建后端Go程序..."
cd server

# 获取版本信息
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date '+%Y-%m-%d %H:%M:%S')
GO_VERSION=$(go version | awk '{print $3}')

# 构建Go二进制文件
echo "正在构建Go程序..."
echo "  版本: $VERSION"
echo "  构建时间: $BUILD_TIME"
echo "  Go版本: $GO_VERSION"

cd apps/server

go build -ldflags "\
    -X 'main.Version=$VERSION' \
    -X 'main.BuildTime=$BUILD_TIME' \
    -X 'main.GoVersion=$GO_VERSION' \
    -w -s" \
    -o ../../bin/server.exe \
    main.go

cd ../..

echo "✓ 后端构建完成"
echo ""

# 4. 显示构建结果
echo ">>> 步骤 4/4: 构建完成！"
echo ""
echo "======================================"
echo "  构建信息"
echo "======================================"
echo "二进制文件: server/bin/server.exe"
echo "文件大小: $(du -h server/bin/server.exe | cut -f1)"
echo "版本: $VERSION"
echo "构建时间: $BUILD_TIME"
echo ""
echo "======================================"
echo "  运行说明"
echo "======================================"
echo "1. 确保 config.yaml 配置正确"
echo "2. 运行以下命令启动服务："
echo ""
echo "   cd server/bin"
echo "   ./server.exe"
echo ""
echo "3. 访问 http://localhost:8080"
echo "   (端口取决于 config.yaml 中的配置)"
echo ""
echo "4. 查看版本信息："
echo "   ./server.exe -version"
echo ""
echo "======================================"

