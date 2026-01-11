# 任务管理系统 (Task Management System)

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![MySQL](https://img.shields.io/badge/mysql-%2300f.svg?style=for-the-badge&logo=mysql&logoColor=white)
![Redis](https://img.shields.io/badge/redis-%23DD0031.svg?style=for-the-badge&logo=redis&logoColor=white)
![Gin](https://img.shields.io/badge/gin-00ADD8?style=for-the-badge&logo=go&logoColor=white)

> 🎓 **教学项目** - 专为Golang后端开发初学者设计的完整实践项目，涵盖现代后端开发的核心技术栈。

## 📋 项目简介

这是一个基于Golang的任务管理系统，展示了现代后端开发的最佳实践。项目采用清晰的分层架构，集成了主流的技术栈，非常适合用于学习和实践Golang后端开发。

### 🎯 学习目标

- ✅ 掌握Golang项目的标准结构和组织方式
- ✅ 学会使用Gin框架构建RESTful API
- ✅ 理解GORM的高级用法和数据库设计
- ✅ 掌握Redis缓存策略和使用场景
- ✅ 学习中间件的设计和应用
- ✅ 了解错误处理和日志记录的最佳实践
- ✅ 体验代码自动生成工具的威力

## 🛠 技术栈

### 核心框架
- **[Gin](https://gin-gonic.com/)** - 高性能的HTTP Web框架
- **[GORM](https://gorm.io/)** - 功能丰富的ORM库
- **[Redis](https://redis.io/)** - 内存数据结构存储

### 数据库
- **MySQL** - 关系型数据库
- **Redis** - 缓存和会话存储

### 工具库
- **[Viper](https://github.com/spf13/viper)** - 配置管理
- **[Logrus](https://github.com/sirupsen/logrus)** - 结构化日志
- **[Swaggo](https://github.com/swaggo/swag)** - API文档生成

## 🏗 项目结构

```
task-management-system/
├── cmd/                    # 主要应用程序入口
│   └── server/
│       └── main.go        # 服务器启动入口
├── internal/              # 私有应用程序代码
│   ├── config/           # 配置管理
│   ├── database/         # 数据库连接和迁移
│   ├── handlers/         # HTTP处理器
│   ├── middleware/       # HTTP中间件
│   ├── models/           # 数据模型
│   └── services/         # 业务逻辑层
├── pkg/                   # 可重用的库代码
│   ├── redis/           # Redis客户端封装
│   └── utils/           # 工具函数
├── api/                   # API定义
├── configs/              # 配置文件
├── scripts/              # 构建和部署脚本
├── test/                 # 测试文件
├── docs/                 # 文档
└── examples/             # 示例代码
```

## 🚀 快速开始

### 环境要求

- Go 1.21+
- MySQL 8.0+
- Redis 6.0+

### 安装步骤

1. **克隆项目**
   ```bash
   git clone <repository-url>
   cd task-management-system
   ```

2. **安装依赖**
   ```bash
   go mod tidy
   ```

3. **配置数据库**
   ```bash
   # 创建MySQL数据库
   mysql -u root -p
   CREATE DATABASE task_management;
   ```

4. **启动Redis**
   ```bash
   redis-server
   ```

5. **配置文件**
   ```bash
   # 复制配置模板
   cp configs/config.yaml.example configs/config.yaml
   # 修改数据库连接信息
   vim configs/config.yaml
   ```

6. **运行项目**
   ```bash
   # 开发模式
   go run cmd/server/main.go
   
   # 或者构建后运行
   go build -o bin/server cmd/server/main.go
   ./bin/server
   ```

7. **访问应用**
   - API服务: http://localhost:8080
   - 健康检查: http://localhost:8080/health
   - API文档: http://localhost:8080/swagger/index.html

## 📊 数据库设计

### 核心表结构

```sql
-- 用户表
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    nickname VARCHAR(50),
    avatar VARCHAR(255),
    phone VARCHAR(20),
    status INT DEFAULT 1,
    last_login_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

-- 任务表
CREATE TABLE tasks (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    title VARCHAR(200) NOT NULL,
    description TEXT,
    status INT DEFAULT 0,
    priority INT DEFAULT 2,
    start_time TIMESTAMP NULL,
    end_time TIMESTAMP NULL,
    due_date TIMESTAMP NULL,
    user_id BIGINT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    FOREIGN KEY (user_id) REFERENCES users(id)
);

-- 标签表
CREATE TABLE tags (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    name VARCHAR(50) UNIQUE NOT NULL,
    color VARCHAR(7),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

-- 任务标签关联表
CREATE TABLE task_tags (
    task_id BIGINT,
    tag_id BIGINT,
    PRIMARY KEY (task_id, tag_id),
    FOREIGN KEY (task_id) REFERENCES tasks(id),
    FOREIGN KEY (tag_id) REFERENCES tags(id)
);
```

## 🔧 组件配合使用说明

### 1. 数据流向

```
HTTP请求 → Gin路由 → 中间件 → 处理器 → 服务层 → 数据层 → Redis缓存
                                                ↓
                                            MySQL数据库
```

### 2. 各组件职责

| 组件 | 职责 | 关键特性 |
|------|------|----------|
| **Gin** | HTTP路由和请求处理 | 高性能、中间件支持、参数绑定 |
| **GORM** | 数据库ORM操作 | 自动迁移、关联查询、事务支持 |
| **Redis** | 缓存和会话存储 | 高性能缓存、计数器、分布式锁 |
| **MySQL** | 数据持久化存储 | 事务支持、复杂查询、数据一致性 |

### 3. 缓存策略

```go
// 查询优先级：缓存 → 数据库 → 更新缓存
func (s *UserService) GetUserByID(id uint) (*models.User, error) {
    // 1. 尝试从Redis获取
    if user := s.getFromCache(id); user != nil {
        return user, nil
    }
    
    // 2. 从MySQL查询
    user, err := s.getFromDB(id)
    if err != nil {
        return nil, err
    }
    
    // 3. 更新Redis缓存
    s.setToCache(id, user)
    return user, nil
}
```

## 📁 核心代码解析

### 1. 配置管理 (`internal/config/`)

```go
// 学习要点：Viper配置管理，多环境配置
type Config struct {
    Server   ServerConfig   `yaml:"server"`
    Database DatabaseConfig `yaml:"database"`  
    Redis    RedisConfig    `yaml:"redis"`
}

func Load(configPath string) error {
    viper.SetConfigFile(configPath)
    viper.AutomaticEnv() // 支持环境变量覆盖
    return viper.ReadInConfig()
}
```

### 2. 数据模型 (`internal/models/`)

```go
// 学习要点：GORM模型设计，关联关系
type Task struct {
    BaseModel
    Title       string     `gorm:"size:200;not null" json:"title"`
    Description string     `gorm:"type:text" json:"description"`
    Status      int        `gorm:"default:0" json:"status"`
    UserID      uint       `gorm:"not null" json:"user_id"`
    
    // 关联关系
    User User   `gorm:"foreignKey:UserID" json:"user,omitempty"`
    Tags []Tag  `gorm:"many2many:task_tags" json:"tags,omitempty"`
}
```

### 3. 服务层 (`internal/services/`)

```go
// 学习要点：业务逻辑封装，缓存策略
type TaskService struct {
    db    *gorm.DB
    cache *redis.CacheService
}

func (s *TaskService) CreateTask(userID uint, req *TaskCreateRequest) (*Task, error) {
    // 事务处理
    tx := s.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // 创建任务
    task := &Task{...}
    if err := tx.Create(task).Error; err != nil {
        tx.Rollback()
        return nil, err
    }
    
    // 提交事务
    return task, tx.Commit().Error
}
```

### 4. API处理器 (`internal/handlers/`)

```go
// 学习要点：HTTP处理器设计，参数验证
func (h *TaskHandler) CreateTask(c *gin.Context) {
    var req models.TaskCreateRequest
    
    // 参数绑定和验证
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, models.NewErrorResponse("参数错误: "+err.Error()))
        return
    }
    
    // 调用服务层
    task, err := h.taskService.CreateTask(userID, &req)
    if err != nil {
        c.JSON(500, models.NewErrorResponse(err.Error()))
        return
    }
    
    c.JSON(200, models.NewSuccessResponse(task))
}
```

## 🔄 GORM 代码自动生成

### 生成查询代码

```bash
# 运行生成器
go run scripts/generate.go
```

### 使用生成的代码

```go
import "task-management-system/internal/query"

// 创建查询实例
q := query.Use(db)

// 类型安全的查询
users := q.User.Where(q.User.Status.Eq(1)).Find()

// 复杂查询
tasks := q.Task.
    Where(q.Task.Priority.Gte(3)).
    Preload(q.Task.User).
    Order(q.Task.CreatedAt.Desc()).
    Limit(10).Find()
```

### 生成代码优势

- ✅ **类型安全**: 编译时检查字段名和类型
- ✅ **IDE支持**: 自动完成和重构支持
- ✅ **性能优化**: 预编译查询语句
- ✅ **防SQL注入**: 自动参数绑定

## 📡 API 接口文档

### 用户管理

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/users` | 创建用户 |
| GET | `/api/v1/users` | 获取用户列表 |
| GET | `/api/v1/users/{id}` | 获取用户详情 |
| PUT | `/api/v1/users/{id}` | 更新用户信息 |
| DELETE | `/api/v1/users/{id}` | 删除用户 |

### 任务管理

| 方法 | 路径 | 描述 |
|------|------|------|
| POST | `/api/v1/tasks` | 创建任务 |
| GET | `/api/v1/tasks` | 查询任务列表 |
| GET | `/api/v1/tasks/{id}` | 获取任务详情 |
| PUT | `/api/v1/tasks/{id}` | 更新任务 |
| DELETE | `/api/v1/tasks/{id}` | 删除任务 |
| POST | `/api/v1/tasks/{id}/complete` | 标记任务完成 |

### 示例请求

```bash
# 创建用户
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com", 
    "password": "123456",
    "nickname": "测试用户"
  }'

# 创建任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "X-User-ID: 1" \
  -d '{
    "title": "学习Golang",
    "description": "深入学习Golang后端开发",
    "priority": 3,
    "due_date": "2024-12-31T23:59:59Z"
  }'
```

## 🧪 测试

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/services

# 运行测试并查看覆盖率
go test -cover ./...
```

### 测试示例

```go
func TestUserService_CreateUser(t *testing.T) {
    // 设置测试数据库
    db := setupTestDB()
    defer teardownTestDB(db)
    
    service := NewUserService(db)
    
    req := &UserCreateRequest{
        Username: "testuser",
        Email:    "test@example.com",
        Password: "123456",
    }
    
    user, err := service.CreateUser(req)
    assert.NoError(t, err)
    assert.Equal(t, "testuser", user.Username)
}
```

## 📈 性能优化

### 1. 数据库优化

```sql
-- 创建索引
CREATE INDEX idx_tasks_user_id ON tasks(user_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_created_at ON tasks(created_at);
```

### 2. 缓存优化

```go
// 缓存热点数据
func (s *UserService) GetPopularUsers() ([]User, error) {
    cacheKey := "popular_users"
    
    // 尝试从缓存获取
    if users := s.cache.Get(cacheKey); users != nil {
        return users, nil
    }
    
    // 从数据库查询并缓存
    users, err := s.db.Find(&[]User{}).Error
    if err == nil {
        s.cache.Set(cacheKey, users, time.Hour)
    }
    
    return users, err
}
```

### 3. 连接池配置

```yaml
database:
  mysql:
    max_idle_conns: 10
    max_open_conns: 100
    conn_max_lifetime: 3600
```

## 🔧 部署

### Docker 部署

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
COPY --from=builder /app/configs ./configs

CMD ["./server"]
```

### Docker Compose

```yaml
version: '3.8'
services:
  app:
    build: .
    ports:
      - "8080:8080"
    depends_on:
      - mysql
      - redis
    environment:
      - CONFIG_PATH=configs/config.yaml

  mysql:
    image: mysql:8.0
    environment:
      MYSQL_DATABASE: task_management
      MYSQL_ROOT_PASSWORD: 123456
    ports:
      - "3306:3306"

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
```

## 📚 学习资源

### 推荐阅读

- [Gin官方文档](https://gin-gonic.com/docs/)
- [GORM指南](https://gorm.io/docs/)
- [Go语言规范](https://golang.org/ref/spec)
- [Redis命令参考](https://redis.io/commands)

### 进阶学习

1. **微服务架构**: 学习如何拆分单体应用
2. **消息队列**: 集成RabbitMQ或Kafka
3. **监控系统**: Prometheus + Grafana
4. **API网关**: Kong或Traefik
5. **容器编排**: Kubernetes部署

## 🤝 贡献指南

1. Fork项目
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add some amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 开启Pull Request

## 📄 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 📞 联系方式

- 项目链接: [https://github.com/your-username/task-management-system](https://github.com/your-username/task-management-system)
- 问题反馈: [Issues](https://github.com/your-username/task-management-system/issues)

## 🙏 致谢

感谢以下开源项目为本项目提供的支持：

- [Gin Web Framework](https://github.com/gin-gonic/gin)
- [GORM](https://github.com/go-gorm/gorm)
- [Redis](https://redis.io/)
- [Viper](https://github.com/spf13/viper)

---

⭐ **如果这个项目对你有帮助，请点个Star支持一下！**