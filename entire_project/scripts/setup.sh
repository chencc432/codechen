#!/bin/bash

# 任务管理系统环境设置脚本
# 学习要点：项目环境自动化设置，依赖检查，数据初始化

set -e  # 遇到错误立即退出

echo "🚀 开始设置任务管理系统开发环境..."

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 打印彩色消息
print_message() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 检查命令是否存在
check_command() {
    if ! command -v $1 &> /dev/null; then
        print_error "$1 未安装或不在PATH中"
        return 1
    else
        print_message "$1 检查通过 ✅"
        return 0
    fi
}

# 1. 检查系统依赖
print_message "检查系统依赖..."

# 检查Go环境
if ! check_command go; then
    print_error "请先安装Go语言环境"
    echo "下载地址: https://golang.org/dl/"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
print_message "Go版本: $GO_VERSION"

# 检查MySQL
if ! check_command mysql; then
    print_warning "MySQL客户端未找到，请确保MySQL服务器已启动"
else
    print_message "MySQL客户端检查通过"
fi

# 检查Redis
if ! check_command redis-cli; then
    print_warning "Redis客户端未找到，请确保Redis服务器已启动"
else
    print_message "Redis客户端检查通过"
fi

# 2. 检查并创建目录结构
print_message "检查项目目录结构..."

DIRS=(
    "logs"
    "tmp" 
    "bin"
    "internal/query"
)

for dir in "${DIRS[@]}"; do
    if [ ! -d "$dir" ]; then
        mkdir -p "$dir"
        print_message "创建目录: $dir ✅"
    fi
done

# 3. 安装Go依赖
print_message "安装Go依赖包..."

if [ -f "go.mod" ]; then
    go mod tidy
    print_message "依赖包安装完成 ✅"
else
    print_error "go.mod 文件不存在"
    exit 1
fi

# 4. 检查配置文件
print_message "检查配置文件..."

CONFIG_FILE="configs/config.yaml"
CONFIG_EXAMPLE="configs/config.yaml.example"

if [ ! -f "$CONFIG_FILE" ]; then
    if [ -f "$CONFIG_EXAMPLE" ]; then
        cp "$CONFIG_EXAMPLE" "$CONFIG_FILE"
        print_message "已从示例文件创建配置文件"
        print_warning "请修改 $CONFIG_FILE 中的数据库配置"
    else
        print_warning "配置文件不存在，使用默认配置"
    fi
else
    print_message "配置文件检查完成 ✅"
fi

# 5. 数据库设置
print_message "设置数据库..."

read -p "是否需要创建数据库？(y/n): " create_db
if [[ $create_db == "y" || $create_db == "Y" ]]; then
    read -p "请输入MySQL root密码: " -s mysql_password
    echo
    
    # 创建数据库
    mysql -u root -p$mysql_password -e "CREATE DATABASE IF NOT EXISTS task_management CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;" 2>/dev/null
    
    if [ $? -eq 0 ]; then
        print_message "数据库创建成功 ✅"
    else
        print_error "数据库创建失败，请手动创建"
    fi
fi

# 6. Redis设置
print_message "检查Redis连接..."

redis-cli ping > /dev/null 2>&1
if [ $? -eq 0 ]; then
    print_message "Redis连接正常 ✅"
else
    print_warning "Redis连接失败，请确保Redis服务已启动"
    print_message "启动Redis命令: redis-server"
fi

# 7. 构建项目
print_message "构建项目..."

go build -o bin/server cmd/server/main.go
if [ $? -eq 0 ]; then
    print_message "项目构建成功 ✅"
else
    print_error "项目构建失败"
    exit 1
fi

# 8. 生成查询代码（可选）
read -p "是否生成GORM查询代码？(y/n): " generate_code
if [[ $generate_code == "y" || $generate_code == "Y" ]]; then
    print_message "生成GORM查询代码..."
    
    # 先启动项目确保数据库表已创建
    print_message "正在启动服务器以创建数据库表..."
    
    # 后台启动服务器
    ./bin/server &
    SERVER_PID=$!
    
    # 等待服务器启动
    sleep 5
    
    # 停止服务器
    kill $SERVER_PID 2>/dev/null || true
    
    # 运行代码生成器
    if [ -f "scripts/generate.go" ]; then
        cd scripts
        go run generate.go
        cd ..
        print_message "查询代码生成完成 ✅"
    else
        print_warning "代码生成器文件不存在"
    fi
fi

# 9. 创建启动脚本
print_message "创建启动脚本..."

cat > start.sh << 'EOF'
#!/bin/bash

echo "🚀 启动任务管理系统..."

# 检查配置文件
if [ ! -f "configs/config.yaml" ]; then
    echo "❌ 配置文件不存在"
    exit 1
fi

# 检查二进制文件
if [ ! -f "bin/server" ]; then
    echo "📦 构建项目..."
    go build -o bin/server cmd/server/main.go
fi

# 启动服务器
echo "🌟 服务器启动中..."
./bin/server
EOF

chmod +x start.sh
print_message "启动脚本创建完成 ✅"

# 10. 创建停止脚本
cat > stop.sh << 'EOF'
#!/bin/bash

echo "🛑 停止任务管理系统..."

# 查找并停止服务器进程
PID=$(pgrep -f "bin/server")
if [ ! -z "$PID" ]; then
    kill $PID
    echo "✅ 服务器已停止 (PID: $PID)"
else
    echo "⚠️  没有找到运行中的服务器进程"
fi
EOF

chmod +x stop.sh
print_message "停止脚本创建完成 ✅"

# 11. 创建开发工具脚本
cat > dev.sh << 'EOF'
#!/bin/bash

# 开发工具脚本

case "$1" in
    "build")
        echo "📦 构建项目..."
        go build -o bin/server cmd/server/main.go
        ;;
    "test")
        echo "🧪 运行测试..."
        go test ./...
        ;;
    "fmt")
        echo "🎨 格式化代码..."
        go fmt ./...
        ;;
    "clean")
        echo "🧹 清理项目..."
        rm -rf bin/* logs/* tmp/*
        ;;
    "gen")
        echo "🔧 生成查询代码..."
        cd scripts && go run generate.go
        ;;
    *)
        echo "使用方法: $0 {build|test|fmt|clean|gen}"
        echo ""
        echo "命令说明:"
        echo "  build  - 构建项目"
        echo "  test   - 运行测试"
        echo "  fmt    - 格式化代码"  
        echo "  clean  - 清理生成文件"
        echo "  gen    - 生成查询代码"
        ;;
esac
EOF

chmod +x dev.sh
print_message "开发工具脚本创建完成 ✅"

# 12. 设置完成总结
print_message "🎉 环境设置完成！"
echo ""
echo -e "${BLUE}接下来的步骤:${NC}"
echo "1. 修改配置文件: configs/config.yaml"
echo "2. 启动服务器: ./start.sh"
echo "3. 访问API文档: http://localhost:8080/swagger/index.html"
echo "4. 健康检查: http://localhost:8080/health"
echo ""
echo -e "${BLUE}常用命令:${NC}"
echo "  ./start.sh         - 启动服务器"
echo "  ./stop.sh          - 停止服务器"
echo "  ./dev.sh build     - 构建项目"
echo "  ./dev.sh test      - 运行测试"
echo "  ./dev.sh fmt       - 格式化代码"
echo "  ./dev.sh clean     - 清理文件"
echo "  ./dev.sh gen       - 生成查询代码"
echo ""
print_message "Happy Coding! 🚀"