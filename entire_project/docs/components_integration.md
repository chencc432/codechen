# 组件集成说明

## 🔄 各组件配合使用详解

### 1. 整体架构图

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   HTTP客户端     │────│   Gin路由层     │────│   中间件层      │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   处理器层       │────│   服务层        │────│   数据访问层    │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                ┌───────────────┼───────────────┐
                ▼                               ▼
┌─────────────────┐                    ┌─────────────────┐
│   MySQL数据库    │                    │   Redis缓存     │
└─────────────────┘                    └─────────────────┘
```

### 2. 请求处理流程

#### 2.1 读取数据流程（缓存优先）

```go
客户端请求 → Gin路由 → 中间件验证 → 处理器 → 服务层
                                                │
                                                ▼
                                          检查Redis缓存
                                                │
                                    ┌──────────┴──────────┐
                                    ▼                     ▼
                              缓存命中                  缓存未命中
                                    │                     │
                                    ▼                     ▼
                              返回缓存数据              查询MySQL
                                                         │
                                                         ▼
                                                   更新Redis缓存
                                                         │
                                                         ▼
                                                   返回数据库数据
```

#### 2.2 写入数据流程（数据库优先）

```go
客户端请求 → Gin路由 → 中间件验证 → 处理器 → 服务层
                                                │
                                                ▼
                                          开始数据库事务
                                                │
                                                ▼
                                          写入MySQL数据
                                                │
                                    ┌──────────┴──────────┐
                                    ▼                     ▼
                                事务成功                  事务失败
                                    │                     │
                                    ▼                     ▼
                              清除相关缓存              回滚事务
                                    │                     │
                                    ▼                     ▼
                              提交事务                  返回错误
                                    │
                                    ▼
                              返回成功结果
```

### 3. 核心组件配合机制

#### 3.1 Gin + GORM 集成

```go
// 处理器层：HTTP请求处理
func (h *UserHandler) CreateUser(c *gin.Context) {
    // 1. Gin负责请求参数绑定
    var req models.UserCreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, models.NewErrorResponse("参数错误"))
        return
    }
    
    // 2. 调用服务层（业务逻辑）
    user, err := h.userService.CreateUser(&req)
    if err != nil {
        c.JSON(500, models.NewErrorResponse(err.Error()))
        return
    }
    
    // 3. Gin负责响应返回
    c.JSON(200, models.NewSuccessResponse(user.ToResponse()))
}

// 服务层：业务逻辑处理
func (s *UserService) CreateUser(req *models.UserCreateRequest) (*models.User, error) {
    // 3. GORM负责数据库操作
    user := &models.User{
        Username: req.Username,
        Email:    req.Email,
        Password: req.Password,
    }
    
    if err := s.db.Create(user).Error; err != nil {
        return nil, fmt.Errorf("创建用户失败: %w", err)
    }
    
    return user, nil
}
```

#### 3.2 GORM + Redis 集成

```go
// 查询时的缓存策略
func (s *UserService) GetUserByID(id uint) (*models.User, error) {
    // 1. 先查Redis缓存
    cacheKey := fmt.Sprintf("user:%d", id)
    var userResponse models.UserResponse
    if err := s.cache.Get(cacheKey, &userResponse); err == nil {
        // 缓存命中，直接返回
        return s.convertToUser(&userResponse), nil
    }
    
    // 2. 缓存未命中，查询数据库（GORM）
    var user models.User
    if err := s.db.First(&user, id).Error; err != nil {
        if err == gorm.ErrRecordNotFound {
            return nil, fmt.Errorf("用户不存在")
        }
        return nil, fmt.Errorf("查询用户失败: %w", err)
    }
    
    // 3. 将查询结果缓存到Redis
    if err := s.cache.Set(cacheKey, user.ToResponse(), time.Hour); err != nil {
        // 缓存失败不影响主业务逻辑，只记录日志
        log.Printf("缓存用户信息失败: %v", err)
    }
    
    return &user, nil
}

// 更新时的缓存清理策略
func (s *UserService) UpdateUser(id uint, req *models.UserUpdateRequest) error {
    // 1. 先更新数据库（GORM）
    updates := map[string]interface{}{}
    if req.Nickname != "" {
        updates["nickname"] = req.Nickname
    }
    
    if err := s.db.Model(&models.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
        return fmt.Errorf("更新用户失败: %w", err)
    }
    
    // 2. 清除Redis中的缓存
    cacheKey := fmt.Sprintf("user:%d", id)
    if err := s.cache.Delete(cacheKey); err != nil {
        log.Printf("删除用户缓存失败: %v", err)
    }
    
    return nil
}
```

#### 3.3 Redis 高级用法集成

```go
// 分布式锁
func (s *TaskService) CompleteTask(taskID uint, userID uint) error {
    // 使用Redis实现分布式锁，防止重复操作
    lockKey := fmt.Sprintf("task_lock:%d", taskID)
    lockValue := fmt.Sprintf("%d_%d", userID, time.Now().UnixNano())
    
    // 尝试获取锁
    locked, err := s.cache.SetNX(lockKey, lockValue, time.Second*10)
    if err != nil {
        return fmt.Errorf("获取锁失败: %w", err)
    }
    if !locked {
        return fmt.Errorf("任务正在处理中，请稍后再试")
    }
    
    // 确保释放锁
    defer s.releaseLock(lockKey, lockValue)
    
    // 执行任务完成逻辑
    return s.doCompleteTask(taskID, userID)
}

// 计数器功能
func (s *TaskService) IncrementTaskCount(userID uint, status int) error {
    countKey := fmt.Sprintf("user_task_count:%d:%d", userID, status)
    
    // 原子递增
    _, err := s.cache.IncrBy(countKey, 1)
    if err != nil {
        return fmt.Errorf("更新任务计数失败: %w", err)
    }
    
    // 设置过期时间
    return s.cache.SetExpire(countKey, time.Hour*24)
}

// 排行榜功能
func (s *TaskService) UpdateUserRanking(userID uint, score int64) error {
    rankingKey := "user_task_ranking"
    
    // 使用有序集合更新排行榜
    return s.cache.ZAdd(rankingKey, score, userID)
}
```

### 4. 事务处理机制

#### 4.1 数据库事务 + 缓存一致性

```go
func (s *TaskService) CreateTaskWithTags(userID uint, req *TaskCreateRequest) (*models.Task, error) {
    // 开始数据库事务
    tx := s.db.Begin()
    defer func() {
        if r := recover(); r != nil {
            tx.Rollback()
        }
    }()
    
    // 1. 创建任务
    task := &models.Task{
        Title:       req.Title,
        Description: req.Description,
        UserID:      userID,
    }
    
    if err := tx.Create(task).Error; err != nil {
        tx.Rollback()
        return nil, fmt.Errorf("创建任务失败: %w", err)
    }
    
    // 2. 关联标签
    if len(req.TagIDs) > 0 {
        var tags []models.Tag
        if err := tx.Where("id IN ?", req.TagIDs).Find(&tags).Error; err != nil {
            tx.Rollback()
            return nil, fmt.Errorf("查询标签失败: %w", err)
        }
        
        if err := tx.Model(task).Association("Tags").Append(tags); err != nil {
            tx.Rollback()
            return nil, fmt.Errorf("关联标签失败: %w", err)
        }
    }
    
    // 3. 提交数据库事务
    if err := tx.Commit().Error; err != nil {
        return nil, fmt.Errorf("提交事务失败: %w", err)
    }
    
    // 4. 事务提交成功后，处理缓存
    go func() {
        // 异步清理相关缓存
        s.clearUserTasksCache(userID)
        s.updateTaskCountCache(userID, task.Status, 1)
    }()
    
    return task, nil
}
```

### 5. 错误处理和日志集成

```go
// 统一错误处理中间件
func ErrorHandlingMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        defer func() {
            if err := recover(); err != nil {
                // 记录错误日志
                requestID := c.GetString("request_id")
                log.WithFields(logrus.Fields{
                    "request_id": requestID,
                    "error":      err,
                    "path":       c.Request.URL.Path,
                    "method":     c.Request.Method,
                }).Error("发生panic错误")
                
                // 返回统一错误响应
                c.JSON(500, models.NewErrorResponse("内部服务器错误"))
                c.Abort()
            }
        }()
        
        c.Next()
        
        // 检查是否有错误需要记录
        if len(c.Errors) > 0 {
            for _, err := range c.Errors {
                log.WithField("request_id", c.GetString("request_id")).
                    Error(err.Error())
            }
        }
    }
}
```

### 6. 性能监控集成

```go
// 数据库查询性能监控
func DatabaseMetricsMiddleware(db *gorm.DB) {
    db.Callback().Query().Before("gorm:query").Register("metrics:query_start", func(db *gorm.DB) {
        db.Set("query_start_time", time.Now())
    })
    
    db.Callback().Query().After("gorm:query").Register("metrics:query_end", func(db *gorm.DB) {
        if startTime, ok := db.Get("query_start_time"); ok {
            duration := time.Since(startTime.(time.Time))
            
            // 记录慢查询
            if duration > time.Millisecond*100 {
                log.WithFields(logrus.Fields{
                    "duration": duration,
                    "sql":      db.Statement.SQL.String(),
                }).Warn("慢查询检测")
            }
        }
    })
}
```

### 7. 配置管理集成

```go
// 环境变量 + 配置文件集成
type DatabaseConfig struct {
    MySQL MySQLConfig `yaml:"mysql"`
}

func (c *DatabaseConfig) GetDSN() string {
    // 优先使用环境变量
    if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
        return dsn
    }
    
    // 否则使用配置文件
    return c.MySQL.GetMySQLDSN()
}
```

### 8. 健康检查集成

```go
// 健康检查端点
func HealthCheckHandler(db *gorm.DB, redisClient *redis.Client) gin.HandlerFunc {
    return func(c *gin.Context) {
        health := gin.H{
            "status":    "ok",
            "timestamp": time.Now().Unix(),
            "services":  gin.H{},
        }
        
        // 检查数据库连接
        if sqlDB, err := db.DB(); err != nil || sqlDB.Ping() != nil {
            health["services"].(gin.H)["database"] = "down"
            health["status"] = "degraded"
        } else {
            health["services"].(gin.H)["database"] = "up"
        }
        
        // 检查Redis连接
        if _, err := redisClient.Ping(context.Background()).Result(); err != nil {
            health["services"].(gin.H)["redis"] = "down"
            health["status"] = "degraded"
        } else {
            health["services"].(gin.H)["redis"] = "up"
        }
        
        statusCode := 200
        if health["status"] != "ok" {
            statusCode = 503
        }
        
        c.JSON(statusCode, health)
    }
}
```

## 💡 最佳实践

### 1. 数据一致性
- 数据库操作使用事务确保ACID特性
- 缓存更新失败不影响主业务逻辑
- 使用异步方式处理非关键缓存操作

### 2. 性能优化
- 读操作优先查询缓存
- 写操作优先更新数据库再清理缓存
- 使用连接池合理配置数据库连接

### 3. 错误处理
- 统一错误响应格式
- 详细的错误日志记录
- 优雅的错误降级处理

### 4. 监控告警
- 数据库慢查询监控
- Redis命中率监控
- API响应时间监控

这种集成方式确保了各个组件能够协调工作，既保证了数据的一致性，又提供了良好的性能和用户体验。