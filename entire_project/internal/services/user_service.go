// Package services 业务逻辑层
// 学习要点：服务层设计，业务逻辑封装，缓存策略
package services

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"task-management-system/internal/database"
	"task-management-system/internal/models"
	"task-management-system/pkg/redis"
)

// UserService 用户服务结构体
// 学习要点：服务层结构设计，依赖注入
type UserService struct {
	db    *gorm.DB
	cache *redis.CacheService
}

// NewUserService 创建用户服务实例
func NewUserService() *UserService {
	return &UserService{
		db:    database.DB,
		cache: redis.NewCacheService(),
	}
}

// CreateUser 创建用户
// 学习要点：数据验证，事务处理，密码加密
func (s *UserService) CreateUser(req *models.UserCreateRequest) (*models.User, error) {
	// 检查用户名是否已存在
	var existingUser models.User
	if err := s.db.Where("username = ?", req.Username).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("用户名已存在: %s", req.Username)
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("检查用户名失败: %w", err)
	}
	
	// 检查邮箱是否已存在
	if err := s.db.Where("email = ?", req.Email).First(&existingUser).Error; err == nil {
		return nil, fmt.Errorf("邮箱已存在: %s", req.Email)
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("检查邮箱失败: %w", err)
	}
	
	// TODO: 在实际项目中应该加密密码
	// password, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	// if err != nil {
	//     return nil, fmt.Errorf("密码加密失败: %w", err)
	// }
	
	// 创建用户
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // 实际项目中应该存储加密后的密码
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Status:   1, // 默认状态为正常
	}
	
	// 保存到数据库
	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}
	
	// 缓存用户信息（缓存1小时）
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, user.ID)
	if err := s.cache.Set(cacheKey, user.ToResponse(), time.Hour); err != nil {
		// 缓存失败不影响主业务逻辑，只记录日志
		fmt.Printf("缓存用户信息失败: %v\n", err)
	}
	
	return user, nil
}

// GetUserByID 根据ID获取用户
// 学习要点：缓存优先策略，缓存穿透处理
func (s *UserService) GetUserByID(id uint) (*models.User, error) {
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, id)
	
	// 先尝试从缓存获取
	var userResponse models.UserResponse
	if err := s.cache.Get(cacheKey, &userResponse); err == nil {
		// 从响应格式转换为完整用户模型
		user := &models.User{
			BaseModel: models.BaseModel{
				ID:        userResponse.ID,
				CreatedAt: userResponse.CreatedAt,
				UpdatedAt: userResponse.UpdatedAt,
			},
			Username:    userResponse.Username,
			Email:       userResponse.Email,
			Nickname:    userResponse.Nickname,
			Avatar:      userResponse.Avatar,
			Phone:       userResponse.Phone,
			Status:      userResponse.Status,
			LastLoginAt: userResponse.LastLoginAt,
		}
		return user, nil
	}
	
	// 缓存未命中，从数据库查询
	var user models.User
	if err := s.db.First(&user, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在: ID=%d", id)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	
	// 更新缓存
	if err := s.cache.Set(cacheKey, user.ToResponse(), time.Hour); err != nil {
		fmt.Printf("缓存用户信息失败: %v\n", err)
	}
	
	return &user, nil
}

// GetUserByUsername 根据用户名获取用户
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	var user models.User
	if err := s.db.Where("username = ?", username).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在: %s", username)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	return &user, nil
}

// UpdateUser 更新用户信息
// 学习要点：部分更新，缓存更新策略
func (s *UserService) UpdateUser(id uint, req *models.UserUpdateRequest) (*models.User, error) {
	// 查找用户
	user, err := s.GetUserByID(id)
	if err != nil {
		return nil, err
	}
	
	// 更新字段
	updates := make(map[string]interface{})
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}
	
	// 执行更新
	if err := s.db.Model(user).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新用户失败: %w", err)
	}
	
	// 删除缓存（让下次查询时重新缓存）
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, id)
	if err := s.cache.Delete(cacheKey); err != nil {
		fmt.Printf("删除用户缓存失败: %v\n", err)
	}
	
	return user, nil
}

// DeleteUser 删除用户（软删除）
// 学习要点：软删除，关联数据处理，缓存清理
func (s *UserService) DeleteUser(id uint) error {
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// 检查用户是否存在
	var user models.User
	if err := tx.First(&user, id).Error; err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("用户不存在: ID=%d", id)
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}
	
	// 软删除用户的所有任务
	if err := tx.Where("user_id = ?", id).Delete(&models.Task{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除用户任务失败: %w", err)
	}
	
	// 软删除用户
	if err := tx.Delete(&user).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除用户失败: %w", err)
	}
	
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	
	// 删除相关缓存
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, id)
	if err := s.cache.Delete(cacheKey); err != nil {
		fmt.Printf("删除用户缓存失败: %v\n", err)
	}
	
	// 删除用户任务列表缓存
	userTasksKey := redis.BuildCacheKey(redis.UserTasksPrefix, id)
	if err := s.cache.Delete(userTasksKey); err != nil {
		fmt.Printf("删除用户任务缓存失败: %v\n", err)
	}
	
	return nil
}

// GetUserList 获取用户列表（分页）
// 学习要点：分页查询，列表缓存策略
func (s *UserService) GetUserList(page, pageSize int) (*models.PageResult, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	
	// 计算偏移量
	offset := (page - 1) * pageSize
	
	// 查询总数
	var total int64
	if err := s.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询用户总数失败: %w", err)
	}
	
	// 查询用户列表
	var users []models.User
	if err := s.db.Limit(pageSize).Offset(offset).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("查询用户列表失败: %w", err)
	}
	
	// 转换为响应格式
	var userResponses []models.UserResponse
	for _, user := range users {
		userResponses = append(userResponses, user.ToResponse())
	}
	
	// 构建分页结果
	result := &models.PageResult{
		List: userResponses,
		PageInfo: models.PageInfo{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}
	
	return result, nil
}

// UpdateLastLoginTime 更新用户最后登录时间
func (s *UserService) UpdateLastLoginTime(id uint) error {
	now := time.Now()
	if err := s.db.Model(&models.User{}).Where("id = ?", id).Update("last_login_at", &now).Error; err != nil {
		return fmt.Errorf("更新最后登录时间失败: %w", err)
	}
	
	// 删除缓存
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, id)
	if err := s.cache.Delete(cacheKey); err != nil {
		fmt.Printf("删除用户缓存失败: %v\n", err)
	}
	
	return nil
}