# 📚 DAO层与代码生成详解

## 🤔 为什么需要DAO层？

### 1. 什么是DAO层

**DAO (Data Access Object)** 是数据访问对象层，是一种经典的设计模式：

```
┌─────────────────────────────────────────────────────────────┐
│                    为什么需要DAO层？                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  没有DAO层的问题：                                            │
│  ❌ 业务逻辑与数据访问混合                                     │
│  ❌ 数据库操作分散在各个服务中                                  │
│  ❌ 难以进行单元测试                                           │
│  ❌ 数据库变更影响业务逻辑                                     │
│  ❌ 代码重复，维护困难                                         │
│                                                             │
│  使用DAO层的优势：                                            │
│  ✅ 数据访问逻辑集中管理                                       │
│  ✅ 业务逻辑与数据访问分离                                     │
│  ✅ 便于单元测试和Mock                                         │
│  ✅ 数据库无关性，易于切换                                     │
│  ✅ 代码复用，统一规范                                         │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

### 2. 架构对比

#### 🔴 没有DAO层的架构（我们原项目）
```go
// 服务层直接操作数据库
func (s *UserService) CreateUser(req *UserCreateRequest) (*User, error) {
    // 业务验证
    if err := s.validateUser(req); err != nil {
        return nil, err
    }
    
    // 直接使用GORM操作数据库 ❌
    user := &User{Username: req.Username, Email: req.Email}
    if err := s.db.Create(user).Error; err != nil {
        return nil, err
    }
    
    // 缓存处理
    s.cache.Set("user:"+user.ID, user)
    
    return user, nil
}
```

#### 🟢 使用DAO层的架构
```go
// 服务层专注业务逻辑
func (s *UserService) CreateUser(ctx context.Context, req *UserCreateRequest) (*User, error) {
    // 业务验证
    if err := s.validateUser(ctx, req); err != nil {
        return nil, err
    }
    
    // 通过DAO操作数据库 ✅
    user := &User{Username: req.Username, Email: req.Email}
    if err := s.userDAO.Create(ctx, user); err != nil {
        return nil, err
    }
    
    // 缓存处理
    s.cache.Set("user:"+user.ID, user)
    
    return user, nil
}

// DAO层专注数据访问
func (d *userDAO) Create(ctx context.Context, user *User) error {
    return d.db.WithContext(ctx).Create(user).Error
}
```

## 🏗️ DAO层的具体作用

### 1. 数据访问抽象化

```go
// UserDAO接口 - 抽象所有用户数据操作
type UserDAO interface {
    Create(ctx context.Context, user *User) error
    GetByID(ctx context.Context, id uint) (*User, error)
    Update(ctx context.Context, user *User) error
    Delete(ctx context.Context, id uint) error
    List(ctx context.Context, offset, limit int) ([]User, int64, error)
    // ... 更多方法
}

// 优势：
// ✅ 接口编程，易于测试和Mock
// ✅ 统一的数据访问规范
// ✅ 便于切换不同的存储实现
```

### 2. 复杂查询封装

```go
// 在DAO中封装复杂查询
func (d *taskDAO) GetTasksByFilter(ctx context.Context, filter TaskFilter) ([]Task, int64, error) {
    query := d.db.WithContext(ctx).Model(&Task{})
    
    // 动态构建查询条件
    if filter.Status != nil {
        query = query.Where("status = ?", *filter.Status)
    }
    if filter.Priority != nil {
        query = query.Where("priority = ?", *filter.Priority)
    }
    if filter.Keyword != "" {
        query = query.Where("title LIKE ? OR description LIKE ?", 
            "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
    }
    
    // 执行查询
    var tasks []Task
    var total int64
    
    query.Count(&total)
    query.Offset(filter.Offset).Limit(filter.Limit).Find(&tasks)
    
    return tasks, total, nil
}

// 服务层调用简洁
func (s *TaskService) SearchTasks(filter TaskFilter) (*PageResult, error) {
    tasks, total, err := s.taskDAO.GetTasksByFilter(ctx, filter)
    // ... 处理结果
}
```

### 3. 事务支持

```go
// DAO支持事务传递
func (d *userDAO) WithTx(tx *gorm.DB) UserDAO {
    return &userDAO{db: tx}
}

// 服务层使用事务
func (s *UserService) CreateUserWithProfile(ctx context.Context, req *CreateRequest) error {
    return s.db.Transaction(func(tx *gorm.DB) error {
        // 使用事务DAO
        userDAO := s.userDAO.WithTx(tx)
        profileDAO := s.profileDAO.WithTx(tx)
        
        // 创建用户
        user := &User{Username: req.Username}
        if err := userDAO.Create(ctx, user); err != nil {
            return err
        }
        
        // 创建用户资料
        profile := &Profile{UserID: user.ID, RealName: req.RealName}
        if err := profileDAO.Create(ctx, profile); err != nil {
            return err
        }
        
        return nil
    })
}
```

### 4. 便于单元测试

```go
// Mock DAO接口用于测试
type MockUserDAO struct{}

func (m *MockUserDAO) Create(ctx context.Context, user *User) error {
    // 模拟创建成功
    user.ID = 123
    return nil
}

func (m *MockUserDAO) GetByID(ctx context.Context, id uint) (*User, error) {
    // 模拟查询结果
    return &User{ID: id, Username: "testuser"}, nil
}

// 测试服务层业务逻辑
func TestUserService_CreateUser(t *testing.T) {
    // 使用Mock DAO
    service := &UserService{
        userDAO: &MockUserDAO{},
    }
    
    user, err := service.CreateUser(ctx, &CreateRequest{
        Username: "test",
        Email:    "test@example.com",
    })
    
    assert.NoError(t, err)
    assert.Equal(t, uint(123), user.ID)
}
```

## 🛠️ GORM 代码生成详解

### 1. 什么是GORM代码生成？

GORM Gen 是一个**代码生成工具**，它可以：

- 📝 **自动生成DAO层代码**
- 🔍 **生成类型安全的查询方法**
- ⚡ **提供高性能的查询构建器**
- 🧪 **支持编译时类型检查**

### 2. 代码生成过程

```go
// scripts/generate.go - 生成器配置
func main() {
    // 1. 连接数据库
    db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
    
    // 2. 创建生成器
    g := gen.NewGenerator(gen.Config{
        OutPath:      "../internal/query",    // 输出目录
        OutFile:      "gen.go",               // 主文件名
        ModelPkgPath: "../internal/models",   // 模型包路径
        Mode: gen.WithoutContext |            // 生成模式
             gen.WithDefaultQuery |
             gen.WithQueryInterface,
    })
    
    // 3. 设置数据库
    g.UseDB(db)
    
    // 4. 生成所有表的查询代码
    g.ApplyBasic(g.GenerateAllTable()...)
    
    // 5. 执行生成
    g.Execute()
}
```

### 3. 运行代码生成

```bash
# 进入scripts目录
cd scripts

# 运行生成器
go run generate.go

# 生成的文件结构
internal/query/
├── gen.go           # 主查询文件
├── users.gen.go     # 用户查询代码
├── tasks.gen.go     # 任务查询代码
└── tags.gen.go      # 标签查询代码
```

### 4. 生成的代码示例

#### 生成的查询结构
```go
// internal/query/users.gen.go (自动生成)
type userDo struct {
    *gen.DO
    Username field.String
    Email    field.String
    Status   field.Int32
    // ... 更多字段
}

// 类型安全的查询方法
func (u *userDo) Where(conds ...field.Expr) *userDo {
    return u.withDO(u.DO.Where(conds...))
}

func (u *userDo) First() (*model.User, error) {
    return u.DO.First()
}

func (u *userDo) Find() ([]*model.User, error) {
    return u.DO.Find()
}

// 更多查询方法...
```

#### 使用生成的代码
```go
// 使用生成的查询代码
import "task-management-system/internal/query"

func ExampleGeneratedQuery() {
    // 1. 创建查询实例
    q := query.Use(db)
    
    // 2. 类型安全的查询 ✅
    users, err := q.User.Where(
        q.User.Status.Eq(1),                    // 状态等于1
        q.User.Username.Like("%admin%"),        // 用户名包含admin
    ).Find()
    
    // 3. 复杂查询
    tasks, err := q.Task.
        Where(q.Task.Priority.Gte(3)).          // 优先级>=3
        Preload(q.Task.User).                   // 预加载用户
        Order(q.Task.CreatedAt.Desc()).         // 按创建时间倒序
        Limit(10).                              // 限制10条
        Find()
    
    // 4. 聚合查询
    count, err := q.Task.Where(q.Task.Status.Eq(2)).Count()
    
    // 5. 子查询
    activeUsers, err := q.User.Where(
        q.User.ID.In(
            q.Task.Select(q.Task.UserID).Where(q.Task.Status.Neq(3)),
        ),
    ).Find()
}
```

## 💡 传统方式 vs 生成代码对比

### 1. 查询安全性对比

```go
// ❌ 传统方式 - 字符串拼接，容易出错
db.Where("status = ? AND priority >= ?", 1, 3).Find(&tasks)
//    ↑ 字段名写错了，运行时才发现
db.Where("statuss = ? AND priority >= ?", 1, 3).Find(&tasks)

// ✅ 生成代码 - 类型安全，编译时检查
q.Task.Where(
    q.Task.Status.Eq(1),      // 编译时就能发现字段名错误
    q.Task.Priority.Gte(3),   // 类型不匹配也会报错
).Find()
```

### 2. IDE支持对比

```go
// ❌ 传统方式 - 没有自动完成
db.Where("user_na", 1).Find(&users)  // IDE无法提示字段名

// ✅ 生成代码 - 完整的IDE支持
q.User.Where(q.User.Username.   // IDE会自动提示所有可用方法
//                    ↑ 自动完成: Eq, Neq, Like, In, etc.
```

### 3. 重构支持对比

```go
// ❌ 传统方式 - 重构困难
// 如果将User.Status改名为User.State，需要手动找到所有相关字符串
db.Where("status = ?", 1)  // 重构时容易遗漏
db.Select("status")        // 需要手动修改
db.Order("status DESC")    // 可能忘记更新

// ✅ 生成代码 - 重构友好
q.User.Where(q.User.Status.Eq(1))  // 重命名字段时IDE会自动更新
q.User.Select(q.User.Status)       // 重构工具全部更新
q.User.Order(q.User.Status.Desc()) // 不会遗漏
```

### 4. 性能对比

```go
// ❌ 传统方式 - 运行时解析
for i := 0; i < 1000; i++ {
    db.Where("status = ?", 1).Find(&users)  // 每次都要解析SQL
}

// ✅ 生成代码 - 预编译优化
q := query.Use(db)
for i := 0; i < 1000; i++ {
    q.User.Where(q.User.Status.Eq(1)).Find()  // 预编译，性能更好
}
```

## 🎯 实际使用建议

### 1. 何时使用DAO层

**✅ 适合使用DAO的场景：**
- 大型项目，多人协作
- 复杂的数据查询逻辑
- 需要高度可测试性
- 可能切换数据库类型
- 对代码规范要求高

**❌ 可以不使用DAO的场景：**
- 小型项目，简单CRUD
- 原型阶段，快速迭代
- 单人开发，代码量小
- 数据访问逻辑简单

### 2. 何时使用代码生成

**✅ 推荐使用生成代码：**
- 新项目，从零开始
- 对类型安全要求高
- 复杂查询较多
- 团队对代码质量要求高

**❌ 慎重使用生成代码：**
- 现有项目，迁移成本高
- 团队对新工具接受度低
- 简单查询为主
- 自定义查询需求多

### 3. 混合使用方案

```go
// 推荐：根据场景选择合适的方式

type TaskService struct {
    db      *gorm.DB
    taskDAO dao.TaskDAO      // 复杂业务逻辑使用DAO
    query   *query.Query     // 简单查询使用生成代码
}

func (s *TaskService) GetTaskList(filter TaskFilter) ([]Task, error) {
    // 简单查询使用生成代码
    return s.query.Task.Where(
        s.query.Task.Status.Eq(filter.Status),
    ).Find()
}

func (s *TaskService) ComplexBusinessLogic(ctx context.Context) error {
    // 复杂业务逻辑使用DAO
    return s.taskDAO.ComplexOperation(ctx)
}

func (s *TaskService) CustomQuery() ([]Task, error) {
    // 特殊需求使用原生SQL
    var tasks []Task
    return tasks, s.db.Raw("SELECT * FROM tasks WHERE custom_logic").Scan(&tasks).Error
}
```

## 📝 总结

### DAO层的价值
1. **职责分离**: 数据访问与业务逻辑分离
2. **代码复用**: 统一的数据访问接口
3. **易于测试**: 接口编程，便于Mock
4. **维护性**: 数据访问逻辑集中管理

### 代码生成的价值
1. **类型安全**: 编译时检查，减少运行时错误
2. **开发效率**: IDE支持，自动完成
3. **重构友好**: 自动更新相关代码
4. **性能优化**: 预编译查询语句

### 选择建议
- **小项目**: Service直接使用GORM，简单高效
- **中项目**: 使用代码生成，保证质量和效率
- **大项目**: DAO + 代码生成混合使用，分层清晰

记住：**没有银弹，选择合适的架构才是最好的架构！**