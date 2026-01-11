// Package dao 数据访问层
// 学习要点：DAO模式实现，数据访问接口抽象，Repository模式
package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"task-management-system/internal/models"
)

// UserDAO 用户数据访问接口
// 学习要点：接口定义，抽象数据访问操作
type UserDAO interface {
	// 基础CRUD操作
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uint) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uint) error
	
	// 查询操作
	List(ctx context.Context, offset, limit int) ([]models.User, int64, error)
	ListByStatus(ctx context.Context, status int) ([]models.User, error)
	Search(ctx context.Context, keyword string, offset, limit int) ([]models.User, int64, error)
	
	// 业务相关查询
	GetActiveUsers(ctx context.Context) ([]models.User, error)
	GetUsersWithTasks(ctx context.Context) ([]models.User, error)
	CountByStatus(ctx context.Context, status int) (int64, error)
	
	// 批量操作
	BatchCreate(ctx context.Context, users []models.User) error
	BatchUpdateStatus(ctx context.Context, ids []uint, status int) error
	
	// 事务支持
	WithTx(tx *gorm.DB) UserDAO
}

// userDAO 用户DAO实现
// 学习要点：接口具体实现，GORM数据库操作封装
type userDAO struct {
	db *gorm.DB
}

// NewUserDAO 创建用户DAO实例
func NewUserDAO(db *gorm.DB) UserDAO {
	return &userDAO{db: db}
}

// Create 创建用户
// 学习要点：数据创建操作，错误处理
func (d *userDAO) Create(ctx context.Context, user *models.User) error {
	if err := d.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("创建用户失败: %w", err)
	}
	return nil
}

// GetByID 根据ID获取用户
// 学习要点：单条记录查询，记录不存在处理
func (d *userDAO) GetByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := d.db.WithContext(ctx).First(&user, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在: ID=%d", id)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// GetByUsername 根据用户名获取用户
// 学习要点：条件查询，唯一字段查询
func (d *userDAO) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := d.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在: username=%s", username)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (d *userDAO) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := d.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在: email=%s", email)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// Update 更新用户
// 学习要点：数据更新操作，部分字段更新
func (d *userDAO) Update(ctx context.Context, user *models.User) error {
	if err := d.db.WithContext(ctx).Save(user).Error; err != nil {
		return fmt.Errorf("更新用户失败: %w", err)
	}
	return nil
}

// Delete 删除用户
// 学习要点：软删除操作
func (d *userDAO) Delete(ctx context.Context, id uint) error {
	if err := d.db.WithContext(ctx).Delete(&models.User{}, id).Error; err != nil {
		return fmt.Errorf("删除用户失败: %w", err)
	}
	return nil
}

// List 获取用户列表
// 学习要点：分页查询，总数统计
func (d *userDAO) List(ctx context.Context, offset, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	
	// 统计总数
	if err := d.db.WithContext(ctx).Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计用户总数失败: %w", err)
	}
	
	// 查询列表
	if err := d.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("查询用户列表失败: %w", err)
	}
	
	return users, total, nil
}

// ListByStatus 根据状态获取用户列表
// 学习要点：条件查询，列表操作
func (d *userDAO) ListByStatus(ctx context.Context, status int) ([]models.User, error) {
	var users []models.User
	if err := d.db.WithContext(ctx).Where("status = ?", status).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("根据状态查询用户失败: %w", err)
	}
	return users, nil
}

// Search 搜索用户
// 学习要点：模糊查询，多字段搜索
func (d *userDAO) Search(ctx context.Context, keyword string, offset, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	
	query := d.db.WithContext(ctx).Model(&models.User{})
	if keyword != "" {
		searchPattern := "%" + keyword + "%"
		query = query.Where("username LIKE ? OR email LIKE ? OR nickname LIKE ?", 
			searchPattern, searchPattern, searchPattern)
	}
	
	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计搜索结果总数失败: %w", err)
	}
	
	// 查询列表
	if err := query.Offset(offset).Limit(limit).Find(&users).Error; err != nil {
		return nil, 0, fmt.Errorf("搜索用户失败: %w", err)
	}
	
	return users, total, nil
}

// GetActiveUsers 获取活跃用户
// 学习要点：业务相关查询，复杂条件
func (d *userDAO) GetActiveUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User
	
	// 查询状态为1且最近30天有登录的用户
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	err := d.db.WithContext(ctx).
		Where("status = ? AND (last_login_at IS NULL OR last_login_at > ?)", 1, thirtyDaysAgo).
		Find(&users).Error
		
	if err != nil {
		return nil, fmt.Errorf("查询活跃用户失败: %w", err)
	}
	
	return users, nil
}

// GetUsersWithTasks 获取有任务的用户
// 学习要点：关联查询，EXISTS子查询
func (d *userDAO) GetUsersWithTasks(ctx context.Context) ([]models.User, error) {
	var users []models.User
	
	err := d.db.WithContext(ctx).
		Where("EXISTS (SELECT 1 FROM tasks WHERE tasks.user_id = users.id AND tasks.deleted_at IS NULL)").
		Find(&users).Error
		
	if err != nil {
		return nil, fmt.Errorf("查询有任务的用户失败: %w", err)
	}
	
	return users, nil
}

// CountByStatus 根据状态统计用户数
// 学习要点：聚合查询，统计操作
func (d *userDAO) CountByStatus(ctx context.Context, status int) (int64, error) {
	var count int64
	if err := d.db.WithContext(ctx).Model(&models.User{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计用户数失败: %w", err)
	}
	return count, nil
}

// BatchCreate 批量创建用户
// 学习要点：批量操作，性能优化
func (d *userDAO) BatchCreate(ctx context.Context, users []models.User) error {
	if len(users) == 0 {
		return nil
	}
	
	if err := d.db.WithContext(ctx).CreateInBatches(users, 100).Error; err != nil {
		return fmt.Errorf("批量创建用户失败: %w", err)
	}
	return nil
}

// BatchUpdateStatus 批量更新用户状态
// 学习要点：批量更新，IN查询
func (d *userDAO) BatchUpdateStatus(ctx context.Context, ids []uint, status int) error {
	if len(ids) == 0 {
		return nil
	}
	
	if err := d.db.WithContext(ctx).Model(&models.User{}).
		Where("id IN ?", ids).
		Update("status", status).Error; err != nil {
		return fmt.Errorf("批量更新用户状态失败: %w", err)
	}
	return nil
}

// WithTx 使用事务
// 学习要点：事务支持，依赖注入
func (d *userDAO) WithTx(tx *gorm.DB) UserDAO {
	return &userDAO{db: tx}
}

// UserDAOInterface 确保实现了接口
var _ UserDAO = (*userDAO)(nil)