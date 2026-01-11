// Package examples DAO层使用示例和测试
// 学习要点：DAO层的测试方法，Mock的使用
package examples

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"task-management-system/internal/dao"
	"task-management-system/internal/models"
)

// MockUserDAO Mock DAO用于测试
// 学习要点：接口Mock，测试隔离
type MockUserDAO struct {
	mock.Mock
}

func (m *MockUserDAO) Create(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	// 模拟设置ID
	user.ID = 1
	return args.Error(0)
}

func (m *MockUserDAO) GetByID(ctx context.Context, id uint) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserDAO) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserDAO) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserDAO) Update(ctx context.Context, user *models.User) error {
	args := m.Called(ctx, user)
	return args.Error(0)
}

func (m *MockUserDAO) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserDAO) List(ctx context.Context, offset, limit int) ([]models.User, int64, error) {
	args := m.Called(ctx, offset, limit)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserDAO) ListByStatus(ctx context.Context, status int) ([]models.User, error) {
	args := m.Called(ctx, status)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserDAO) Search(ctx context.Context, keyword string, offset, limit int) ([]models.User, int64, error) {
	args := m.Called(ctx, keyword, offset, limit)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserDAO) GetActiveUsers(ctx context.Context) ([]models.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserDAO) GetUsersWithTasks(ctx context.Context) ([]models.User, error) {
	args := m.Called(ctx)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserDAO) CountByStatus(ctx context.Context, status int) (int64, error) {
	args := m.Called(ctx, status)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockUserDAO) BatchCreate(ctx context.Context, users []models.User) error {
	args := m.Called(ctx, users)
	return args.Error(0)
}

func (m *MockUserDAO) BatchUpdateStatus(ctx context.Context, ids []uint, status int) error {
	args := m.Called(ctx, ids, status)
	return args.Error(0)
}

func (m *MockUserDAO) WithTx(tx interface{}) dao.UserDAO {
	args := m.Called(tx)
	return args.Get(0).(dao.UserDAO)
}

// TestUserService 测试用户服务
// 学习要点：使用Mock进行服务层测试
func TestUserServiceWithDAO_CreateUser(t *testing.T) {
	// 1. 准备Mock DAO
	mockDAO := new(MockUserDAO)
	
	// 2. 设置Mock预期
	ctx := context.Background()
	
	// 期望检查用户名不存在
	mockDAO.On("GetByUsername", ctx, "testuser").
		Return(nil, assert.AnError)
	
	// 期望检查邮箱不存在
	mockDAO.On("GetByEmail", ctx, "test@example.com").
		Return(nil, assert.AnError)
	
	// 期望创建用户成功
	mockDAO.On("Create", ctx, mock.MatchedBy(func(user *models.User) bool {
		return user.Username == "testuser" && user.Email == "test@example.com"
	})).Return(nil)
	
	// 3. 创建服务实例（使用Mock DAO）
	service := &UserServiceWithDAO{
		userDAO: mockDAO,
	}
	
	// 4. 调用被测试的方法
	req := &models.UserCreateRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "123456",
		Nickname: "测试用户",
	}
	
	user, err := service.CreateUser(ctx, req)
	
	// 5. 验证结果
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, uint(1), user.ID) // Mock设置的ID
	
	// 6. 验证Mock调用
	mockDAO.AssertExpectations(t)
}

// TestUserService_CreateUser_DuplicateUsername 测试重复用户名
func TestUserServiceWithDAO_CreateUser_DuplicateUsername(t *testing.T) {
	mockDAO := new(MockUserDAO)
	ctx := context.Background()
	
	// 模拟用户名已存在
	existingUser := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    "existing@example.com",
	}
	
	mockDAO.On("GetByUsername", ctx, "testuser").
		Return(existingUser, nil)
	
	service := &UserServiceWithDAO{
		userDAO: mockDAO,
	}
	
	req := &models.UserCreateRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "123456",
	}
	
	user, err := service.CreateUser(ctx, req)
	
	// 验证返回错误
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "用户名已存在")
	
	mockDAO.AssertExpectations(t)
}

// BenchmarkDAO 性能基准测试
// 学习要点：DAO层的性能测试方法
func BenchmarkUserDAO_GetByID(b *testing.B) {
	// 注意：这里应该使用真实的数据库连接进行基准测试
	// 为了演示，我们使用Mock
	
	mockDAO := new(MockUserDAO)
	ctx := context.Background()
	
	user := &models.User{
		ID:       1,
		Username: "benchuser",
		Email:    "bench@example.com",
	}
	
	// 设置Mock返回
	mockDAO.On("GetByID", ctx, uint(1)).Return(user, nil)
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		_, err := mockDAO.GetByID(ctx, 1)
		if err != nil {
			b.Fatalf("获取用户失败: %v", err)
		}
	}
}

// DAOUsageExamples DAO使用示例
// 学习要点：实际项目中如何使用DAO
func DAOUsageExamples() {
	examples := `
🏗️ DAO层使用示例指南

1. 📝 定义DAO接口
   type UserDAO interface {
       Create(ctx context.Context, user *User) error
       GetByID(ctx context.Context, id uint) (*User, error)
       // ... 更多方法
   }

2. 🔧 实现DAO接口
   type userDAO struct {
       db *gorm.DB
   }
   
   func (d *userDAO) Create(ctx context.Context, user *User) error {
       return d.db.WithContext(ctx).Create(user).Error
   }

3. 🏢 在服务层使用DAO
   type UserService struct {
       userDAO UserDAO
   }
   
   func (s *UserService) CreateUser(req *CreateRequest) (*User, error) {
       // 业务逻辑验证
       if err := s.validate(req); err != nil {
           return nil, err
       }
       
       // 通过DAO操作数据
       user := &User{Username: req.Username}
       return user, s.userDAO.Create(ctx, user)
   }

4. 🧪 编写单元测试
   func TestCreateUser(t *testing.T) {
       mockDAO := &MockUserDAO{}
       mockDAO.On("Create", mock.Anything, mock.Anything).Return(nil)
       
       service := &UserService{userDAO: mockDAO}
       user, err := service.CreateUser(req)
       
       assert.NoError(t, err)
       mockDAO.AssertExpectations(t)
   }

5. 🔄 事务处理
   func (s *UserService) TransferData() error {
       return s.db.Transaction(func(tx *gorm.DB) error {
           userDAO := s.userDAO.WithTx(tx)
           taskDAO := s.taskDAO.WithTx(tx)
           
           // 在同一事务中操作多个DAO
           return s.complexOperation(userDAO, taskDAO)
       })
   }

📈 DAO的优势:
✅ 职责分离 - 数据访问与业务逻辑分离
✅ 易于测试 - 接口编程，便于Mock
✅ 代码复用 - 统一的数据访问方法
✅ 维护性好 - 数据访问逻辑集中管理
✅ 扩展性强 - 易于添加新的查询方法

🎯 使用建议:
• 复杂项目推荐使用DAO层
• 简单项目可以Service直接使用ORM
• 根据团队规模和项目复杂度选择
• 保持接口设计的简洁和一致性
`
	
	fmt.Println(examples)
}

// TestDataIntegrity 数据完整性测试示例
func TestUserDAO_DataIntegrity(t *testing.T) {
	// 注意：这个测试需要真实的数据库环境
	// 这里只是展示测试思路
	
	t.Run("创建用户应该设置正确的默认值", func(t *testing.T) {
		// 测试数据完整性
		// 例如：创建用户时应该自动设置创建时间、默认状态等
	})
	
	t.Run("删除用户应该是软删除", func(t *testing.T) {
		// 测试软删除功能
		// 验证deleted_at字段被正确设置
	})
	
	t.Run("批量操作应该保持数据一致性", func(t *testing.T) {
		// 测试批量操作的数据一致性
	})
}

// PerformanceTest 性能测试示例
func TestUserDAO_Performance(t *testing.T) {
	// 跳过性能测试，除非明确指定
	if testing.Short() {
		t.Skip("跳过性能测试")
	}
	
	t.Run("大量数据查询性能", func(t *testing.T) {
		// 测试在大数据量情况下的查询性能
		start := time.Now()
		
		// 执行查询操作
		// ...
		
		duration := time.Since(start)
		
		// 验证性能指标
		if duration > time.Second {
			t.Errorf("查询时间过长: %v", duration)
		}
	})
}
`
	
	fmt.Println(examples)
}