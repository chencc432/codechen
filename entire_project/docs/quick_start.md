# 🚀 快速入门指南

本指南将帮助你快速运行和理解任务管理系统项目，适合Golang后端开发初学者。

## 📋 环境要求

- **Go语言**: 1.21 或更高版本
- **MySQL**: 8.0 或更高版本  
- **Redis**: 6.0 或更高版本
- **操作系统**: Windows/macOS/Linux

## ⚡ 5分钟快速启动

### 第一步：下载并配置

```bash
# 1. 进入项目目录
cd task-management-system

# 2. 安装依赖
go mod tidy

# 3. 复制配置文件
copy configs\config.yaml configs\my-config.yaml  # Windows
# 或
cp configs/config.yaml configs/my-config.yaml     # Linux/macOS
```

### 第二步：配置数据库

**修改 `configs/config.yaml` 文件：**

```yaml
database:
  mysql:
    host: localhost
    port: 3306
    username: root           # 修改为你的用户名
    password: "your_password" # 修改为你的密码
    dbname: task_management
```

**创建数据库：**

```sql
-- 连接MySQL后执行
CREATE DATABASE task_management CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 第三步：启动Redis

```bash
# Windows (需要先安装Redis)
redis-server

# macOS (使用Homebrew)
brew services start redis

# Linux (Ubuntu/Debian)
sudo systemctl start redis-server
```

### 第四步：运行项目

```bash
# 直接运行
go run cmd/server/main.go

# 或构建后运行
go build -o bin/server cmd/server/main.go
./bin/server  # Linux/macOS
bin\server.exe  # Windows
```

### 第五步：验证运行

访问以下链接验证项目启动成功：

- **健康检查**: http://localhost:8080/health
- **API文档**: http://localhost:8080/swagger/index.html (如果配置了Swagger)

## 🧪 测试API接口

### 1. 创建用户

```bash
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "123456",
    "nickname": "测试用户"
  }'
```

**预期响应：**
```json
{
  "code": 200,
  "message": "操作成功",
  "data": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com",
    "nickname": "测试用户",
    "created_at": "2024-01-01T12:00:00Z"
  }
}
```

### 2. 创建任务

```bash
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 1" \
  -d '{
    "title": "学习Golang",
    "description": "完成Golang基础教程",
    "priority": 3,
    "due_date": "2024-12-31T23:59:59Z"
  }'
```

### 3. 查询任务列表

```bash
curl http://localhost:8080/api/v1/tasks
```

## 🔍 项目结构理解

### 核心文件说明

```
task-management-system/
├── cmd/server/main.go      # 🎯 主程序入口
├── internal/
│   ├── config/            # ⚙️  配置管理
│   ├── models/            # 📊 数据模型  
│   ├── services/          # 🔧 业务逻辑
│   ├── handlers/          # 🌐 HTTP处理
│   └── database/          # 💾 数据库
├── pkg/
│   ├── redis/             # 🔥 Redis缓存
│   └── utils/             # 🛠️ 工具函数
└── configs/config.yaml    # 📋 配置文件
```

### 请求处理流程

```
HTTP请求 → Gin路由 → 中间件 → 处理器 → 服务层 → 数据层
    ↓         ↓        ↓       ↓       ↓       ↓
  客户端   → 路由分发 → 验证日志 → 参数处理 → 业务逻辑 → 数据库/缓存
```

## 📚 核心概念学习

### 1. 数据模型关系

```go
// 用户 (User) - 一对多 - 任务 (Task)
type User struct {
    ID    uint   `json:"id"`
    Tasks []Task `json:"tasks"` // 一个用户有多个任务
}

// 任务 (Task) - 多对多 - 标签 (Tag)  
type Task struct {
    ID     uint  `json:"id"`
    UserID uint  `json:"user_id"` // 属于一个用户
    Tags   []Tag `json:"tags"`    // 一个任务可以有多个标签
}
```

### 2. 缓存策略

```go
// 查询优先级：缓存 → 数据库 → 更新缓存
func GetUser(id uint) (*User, error) {
    // 1. 先查Redis缓存
    if user := cache.Get("user:" + id); user != nil {
        return user, nil
    }
    
    // 2. 查MySQL数据库
    user := db.First(&User{}, id)
    
    // 3. 更新Redis缓存
    cache.Set("user:" + id, user, 1*time.Hour)
    
    return user, nil
}
```

### 3. 错误处理模式

```go
// 统一错误响应格式
type Response struct {
    Code    int         `json:"code"`    // 200-成功，其他-失败
    Message string      `json:"message"` // 错误描述
    Data    interface{} `json:"data"`    // 返回数据
}

// 使用示例
if err != nil {
    c.JSON(500, Response{
        Code:    500,
        Message: "内部服务器错误: " + err.Error(),
        Data:    nil,
    })
    return
}
```

## 🎓 学习建议

### 初学者路径

1. **第一周**: 理解项目结构，运行基本功能
2. **第二周**: 学习数据模型设计，理解关联关系  
3. **第三周**: 深入服务层，掌握业务逻辑处理
4. **第四周**: 学习缓存策略，理解性能优化

### 进阶学习

1. **自动生成**: 学习GORM代码生成器
2. **性能优化**: 数据库索引，Redis缓存策略
3. **监控告警**: 日志记录，性能监控
4. **部署运维**: Docker容器化，CI/CD

## 🐛 常见问题解决

### 1. 数据库连接失败

**错误现象**：
```
连接MySQL数据库失败: dial tcp 127.0.0.1:3306: connect: connection refused
```

**解决方案**：
- 检查MySQL服务是否启动
- 验证配置文件中的用户名密码
- 确认数据库已创建

### 2. Redis连接失败  

**错误现象**：
```
连接Redis失败: dial tcp 127.0.0.1:6379: connect: connection refused
```

**解决方案**：
- 启动Redis服务
- 检查Redis配置和端口

### 3. 端口被占用

**错误现象**：
```
listen tcp :8080: bind: address already in use
```

**解决方案**：
```bash
# 查找占用进程
netstat -ano | findstr :8080  # Windows
lsof -i :8080                 # Linux/macOS

# 修改配置文件端口
server:
  port: 8081  # 改为其他端口
```

### 4. 模块依赖问题

**错误现象**：
```
go: module task-management-system: git ls-remote failed
```

**解决方案**：
```bash
# 清理模块缓存
go clean -modcache

# 重新下载依赖
go mod tidy
```

## 📞 获取帮助

- **查看日志**: 项目会在控制台输出详细日志
- **健康检查**: 访问 `/health` 端点检查服务状态  
- **配置检查**: 确认 `configs/config.yaml` 配置正确
- **端口测试**: 使用 `telnet localhost 8080` 测试端口连通性

## 🎉 下一步

恭喜！你已经成功运行了项目。接下来可以：

1. 📖 阅读详细文档了解更多功能
2. 🔧 修改代码添加新功能
3. 🧪 编写测试用例
4. 🚀 学习部署和运维

Happy Coding! 🚀