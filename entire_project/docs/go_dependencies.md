# 📦 Go依赖包管理完全指南

## 🚀 一次性获取所有依赖包

### 方法1：go mod tidy（最推荐）⭐

```bash
go mod tidy
```

**功能说明：**
- ✅ 下载所有缺失的依赖包
- ✅ 移除不再使用的依赖
- ✅ 自动更新 `go.mod` 和 `go.sum` 文件
- ✅ 解决依赖冲突
- ✅ 整理依赖树结构

**使用场景：**
- 第一次克隆项目后
- 修改代码引入新包后
- 定期清理无用依赖

**执行结果：**
```
go: downloading github.com/gin-gonic/gin v1.9.1
go: downloading gorm.io/gorm v1.25.5
go: downloading github.com/redis/go-redis/v9 v9.3.0
...
```

### 方法2：go mod download

```bash
go mod download
```

**功能说明：**
- ✅ 只下载依赖包到本地缓存
- ✅ 不修改 `go.mod` 文件
- ✅ 加快后续构建速度

**使用场景：**
- CI/CD流水线中预下载依赖
- Docker镜像构建时缓存依赖层
- 离线开发前预下载依赖

**示例：**
```bash
# 下载所有依赖到缓存
go mod download

# 查看下载位置
go env GOMODCACHE
# 输出: C:\Users\YourName\go\pkg\mod
```

### 方法3：go get（安装或更新特定包）

```bash
# 安装单个包
go get github.com/gin-gonic/gin

# 安装指定版本
go get github.com/gin-gonic/gin@v1.9.1

# 更新所有依赖到最新版本
go get -u ./...

# 更新所有依赖到最新的次要版本
go get -u=patch ./...
```

**使用场景：**
- 添加新的依赖包
- 更新某个特定包
- 批量更新所有依赖

## 📋 完整操作流程

### 新项目初始化

```bash
# 1. 创建项目目录
mkdir myproject
cd myproject

# 2. 初始化Go模块
go mod init myproject

# 3. 添加依赖（编写代码后）
go mod tidy

# 4. 运行项目
go run main.go
```

### 克隆已有项目

```bash
# 1. 克隆项目
git clone <repository-url>
cd project

# 2. 下载所有依赖
go mod tidy

# 3. 验证依赖完整性
go mod verify

# 4. 运行项目
go run cmd/server/main.go
```

## 🔧 常用命令速查表

| 命令 | 功能 | 使用场景 |
|------|------|----------|
| `go mod init <name>` | 初始化模块 | 新项目开始 |
| `go mod tidy` | 整理依赖 | ⭐ 最常用，下载+清理 |
| `go mod download` | 下载依赖 | CI/CD、离线开发 |
| `go mod verify` | 验证依赖 | 检查完整性 |
| `go mod graph` | 显示依赖图 | 分析依赖关系 |
| `go mod why <pkg>` | 查看为何需要某包 | 依赖分析 |
| `go get <pkg>` | 添加依赖 | 安装新包 |
| `go get -u <pkg>` | 更新依赖 | 升级版本 |
| `go list -m all` | 列出所有依赖 | 查看依赖列表 |
| `go clean -modcache` | 清理缓存 | 解决缓存问题 |

## 📝 go.mod 文件详解

```go
module task-management-system  // 模块名称

go 1.21  // Go版本要求

require (
    github.com/gin-gonic/gin v1.9.1          // 直接依赖
    github.com/redis/go-redis/v9 v9.3.0
    gorm.io/gorm v1.25.5
)

replace (
    // 替换某个依赖（用于本地开发或fork版本）
    github.com/old/module => github.com/new/module v1.0.0
)

exclude (
    // 排除某个版本
    github.com/bad/module v1.2.3
)
```

### 依赖版本标记说明

- `v1.9.1` - 精确版本号
- `v1.9.0+incompatible` - 不兼容版本（没有go.mod的旧包）
- `v0.0.0-20230101120000-abcdef123456` - 伪版本（基于commit）
- `// indirect` - 间接依赖（传递依赖）

## 🔍 依赖问题排查

### 问题1：下载依赖失败

```bash
# 设置国内代理（推荐）
go env -w GOPROXY=https://goproxy.cn,direct

# 或使用阿里云代理
go env -w GOPROXY=https://mirrors.aliyun.com/goproxy/,direct

# 查看当前代理设置
go env GOPROXY
```

### 问题2：依赖冲突

```bash
# 查看依赖图
go mod graph

# 查看某个包的依赖路径
go mod why github.com/some/package

# 更新所有依赖
go get -u ./...

# 清理并重新下载
go clean -modcache
go mod tidy
```

### 问题3：sum mismatch错误

```bash
# 删除go.sum并重新生成
rm go.sum
go mod tidy

# 验证依赖完整性
go mod verify
```

### 问题4：缓存损坏

```bash
# 清理模块缓存
go clean -modcache

# 重新下载
go mod download
```

## 🎯 最佳实践

### 1. 版本管理策略

```bash
# ✅ 推荐：使用精确版本
require github.com/gin-gonic/gin v1.9.1

# ❌ 不推荐：使用latest
# require github.com/gin-gonic/gin latest
```

### 2. 定期更新依赖

```bash
# 每月更新一次次要版本
go get -u=patch ./...
go mod tidy

# 大版本更新前先测试
go get -u github.com/some/package
go test ./...
```

### 3. 提交代码时

```bash
# 提交go.mod和go.sum
git add go.mod go.sum
git commit -m "Update dependencies"

# ⚠️ 不要提交vendor目录（除非有特殊需求）
```

### 4. CI/CD配置

```yaml
# .github/workflows/go.yml
name: Go CI
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
        with:
          go-version: '1.21'
      
      # 下载依赖
      - name: Download dependencies
        run: go mod download
      
      # 验证依赖
      - name: Verify dependencies
        run: go mod verify
      
      # 构建项目
      - name: Build
        run: go build ./...
      
      # 运行测试
      - name: Test
        run: go test ./...
```

## 🚢 Docker构建优化

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app

# 复制依赖文件
COPY go.mod go.sum ./

# 下载依赖（利用Docker缓存层）
RUN go mod download

# 复制源代码
COPY . .

# 构建
RUN go build -o server cmd/server/main.go

# 运行阶段
FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/server .
CMD ["./server"]
```

## 📊 依赖分析工具

### 1. 查看依赖树

```bash
# 安装依赖分析工具
go install github.com/KyleBanks/depth/cmd/depth@latest

# 查看依赖树
depth github.com/gin-gonic/gin
```

### 2. 查看过时的依赖

```bash
# 安装检查工具
go install github.com/psampaz/go-mod-outdated@latest

# 检查过时依赖
go list -u -m -json all | go-mod-outdated
```

### 3. 依赖许可证检查

```bash
# 安装许可证检查工具
go install github.com/google/go-licenses@latest

# 检查所有依赖的许可证
go-licenses check ./...
```

## 💡 实用技巧

### 1. 使用vendor目录（离线开发）

```bash
# 将所有依赖复制到vendor目录
go mod vendor

# 使用vendor目录构建
go build -mod=vendor ./...
```

### 2. 替换本地依赖（开发调试）

```go
// go.mod
replace github.com/some/module => ../local/path/to/module
```

### 3. 排除特定版本

```go
// go.mod
exclude github.com/buggy/module v1.2.3
```

### 4. 清理未使用的依赖

```bash
# go mod tidy会自动清理
go mod tidy

# 查看变化
git diff go.mod
```

## 📚 学习资源

- [官方文档](https://go.dev/doc/modules/managing-dependencies)
- [Go Modules Wiki](https://github.com/golang/go/wiki/Modules)
- [包管理最佳实践](https://go.dev/blog/using-go-modules)

## 🎯 快速参考

### 日常开发流程

```bash
# 1. 添加新功能，引入新包
import "github.com/new/package"

# 2. 整理依赖
go mod tidy

# 3. 运行测试
go test ./...

# 4. 提交代码
git add go.mod go.sum
git commit -m "Add new feature"
```

### 故障排除流程

```bash
# 1. 清理缓存
go clean -modcache

# 2. 删除go.sum
rm go.sum

# 3. 重新整理
go mod tidy

# 4. 验证依赖
go mod verify

# 5. 运行测试
go test ./...
```

---

**记住：** `go mod tidy` 是你最好的朋友！几乎所有依赖问题都可以通过它解决。🚀