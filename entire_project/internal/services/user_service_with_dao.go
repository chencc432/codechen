// Package services 使用DAO模式的服务层示例
// 学习要点：服务层如何使用DAO，业务逻辑与数据访问分离
package services

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"task-management-system/internal/dao"
	"task-management-system/internal/models"
	"task-management-system/pkg/redis"
)

// UserServiceWithDAO 使用DAO模式的用户服务
// 学习要点：依赖注入，接口编程，测试友好
type UserServiceWithDAO struct {
	userDAO dao.UserDAO
	taskDAO dao.TaskDAO
	cache   *redis.CacheService
	db      *gorm.DB
}

// NewUserServiceWithDAO 创建使用DAO的用户服务实例
func NewUserServiceWithDAO(db *gorm.DB, cache *redis.CacheService) *UserServiceWithDAO {
	return &UserServiceWithDAO{
		userDAO: dao.NewUserDAO(db),
		taskDAO: dao.NewTaskDAO(db),
		cache:   cache,
		db:      db,
	}
}

// CreateUser 创建用户
// 学习要点：业务逻辑处理，数据验证，事务使用
func (s *UserServiceWithDAO) CreateUser(ctx context.Context, req *models.UserCreateRequest) (*models.User, error) {
	// 1. 数据验证（业务逻辑层职责）
	if err := s.validateUserCreate(ctx, req); err != nil {
		return nil, err
	}
	
	// 2. 构建用户对象
	user := &models.User{
		Username: req.Username,
		Email:    req.Email,
		Password: req.Password, // 实际项目中应该加密
		Nickname: req.Nickname,
		Phone:    req.Phone,
		Status:   1, // 默认状态
	}
	
	// 3. 保存到数据库（通过DAO）
	if err := s.userDAO.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}
	
	// 4. 缓存用户信息（业务逻辑）
	s.cacheUser(user)
	
	return user, nil
}

// GetUserByID 根据ID获取用户
// 学习要点：缓存策略，降级处理
func (s *UserServiceWithDAO) GetUserByID(ctx context.Context, id uint) (*models.User, error) {
	// 1. 先尝试从缓存获取
	if user := s.getUserFromCache(id); user != nil {
		return user, nil
	}
	
	// 2. 从数据库查询（通过DAO）
	user, err := s.userDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// 3. 更新缓存
	s.cacheUser(user)
	
	return user, nil
}

// UpdateUser 更新用户
// 学习要点：部分更新，缓存失效
func (s *UserServiceWithDAO) UpdateUser(ctx context.Context, id uint, req *models.UserUpdateRequest) (*models.User, error) {
	// 1. 获取现有用户
	user, err := s.userDAO.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	
	// 2. 更新字段（业务逻辑）
	if req.Nickname != "" {
		user.Nickname = req.Nickname
	}
	if req.Avatar != "" {
		user.Avatar = req.Avatar
	}
	if req.Phone != "" {
		user.Phone = req.Phone
	}
	
	// 3. 保存更新
	if err := s.userDAO.Update(ctx, user); err != nil {
		return nil, err
	}
	
	// 4. 清除缓存
	s.clearUserCache(id)
	
	return user, nil
}

// DeleteUser 删除用户
// 学习要点：关联数据处理，事务使用
func (s *UserServiceWithDAO) DeleteUser(ctx context.Context, id uint) error {
	// 使用事务确保数据一致性
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 创建带事务的DAO实例
		userDAO := s.userDAO.WithTx(tx)
		taskDAO := s.taskDAO.WithTx(tx)
		
		// 1. 检查用户是否存在
		_, err := userDAO.GetByID(ctx, id)
		if err != nil {
			return err
		}
		
		// 2. 先删除用户的所有任务
		tasks, _, err := taskDAO.ListByUserID(ctx, id, 0, 1000)
		if err != nil {
			return fmt.Errorf("查询用户任务失败: %w", err)
		}
		
		taskIDs := make([]uint, len(tasks))
		for i, task := range tasks {
			taskIDs[i] = task.ID
		}
		
		if len(taskIDs) > 0 {
			if err := taskDAO.BatchDelete(ctx, taskIDs); err != nil {
				return fmt.Errorf("删除用户任务失败: %w", err)
			}
		}
		
		// 3. 删除用户
		if err := userDAO.Delete(ctx, id); err != nil {
			return err
		}
		
		// 4. 清除相关缓存（异步处理，不影响事务）
		go func() {
			s.clearUserCache(id)
			s.clearUserTasksCache(id)
		}()
		
		return nil
	})
}

// GetUserList 获取用户列表
func (s *UserServiceWithDAO) GetUserList(ctx context.Context, page, pageSize int) (*models.PageResult, error) {
	// 计算偏移量
	offset := (page - 1) * pageSize
	
	// 通过DAO查询
	users, total, err := s.userDAO.List(ctx, offset, pageSize)
	if err != nil {
		return nil, err
	}
	
	// 转换为响应格式
	var userResponses []models.UserResponse
	for _, user := range users {
		userResponses = append(userResponses, user.ToResponse())
	}
	
	return &models.PageResult{
		List: userResponses,
		PageInfo: models.PageInfo{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}, nil
}

// SearchUsers 搜索用户
// 学习要点：搜索功能实现，缓存策略
func (s *UserServiceWithDAO) SearchUsers(ctx context.Context, keyword string, page, pageSize int) (*models.PageResult, error) {
	// 检查缓存（可选）
	cacheKey := fmt.Sprintf("search_users:%s:%d:%d", keyword, page, pageSize)
	if result := s.getSearchResultFromCache(cacheKey); result != nil {
		return result, nil
	}
	
	// 通过DAO搜索
	offset := (page - 1) * pageSize
	users, total, err := s.userDAO.Search(ctx, keyword, offset, pageSize)
	if err != nil {
		return nil, err
	}
	
	// 构建结果
	var userResponses []models.UserResponse
	for _, user := range users {
		userResponses = append(userResponses, user.ToResponse())
	}
	
	result := &models.PageResult{
		List: userResponses,
		PageInfo: models.PageInfo{
			Page:     page,
			PageSize: pageSize,
			Total:    total,
		},
	}
	
	// 缓存搜索结果（短时间）
	s.cacheSearchResult(cacheKey, result, 5*time.Minute)
	
	return result, nil
}

// GetActiveUsers 获取活跃用户
// 学习要点：业务逻辑查询，缓存优化
func (s *UserServiceWithDAO) GetActiveUsers(ctx context.Context) ([]models.UserResponse, error) {
	// 检查缓存
	cacheKey := "active_users"
	if users := s.getActiveUsersFromCache(cacheKey); users != nil {
		return users, nil
	}
	
	// 通过DAO查询
	users, err := s.userDAO.GetActiveUsers(ctx)
	if err != nil {
		return nil, err
	}
	
	// 转换格式
	var responses []models.UserResponse
	for _, user := range users {
		responses = append(responses, user.ToResponse())
	}
	
	// 缓存结果
	s.cacheActiveUsers(cacheKey, responses, 30*time.Minute)
	
	return responses, nil
}

// GetUserStats 获取用户统计
// 学习要点：统计查询，多DAO协作
func (s *UserServiceWithDAO) GetUserStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	// 总用户数
	totalUsers, _, err := s.userDAO.List(ctx, 0, 1)
	if err != nil {
		return nil, err
	}
	stats["total_users"] = len(totalUsers)
	
	// 活跃用户数
	activeCount, err := s.userDAO.CountByStatus(ctx, 1)
	if err != nil {
		return nil, err
	}
	stats["active_users"] = activeCount
	
	// 禁用用户数
	disabledCount, err := s.userDAO.CountByStatus(ctx, 0)
	if err != nil {
		return nil, err
	}
	stats["disabled_users"] = disabledCount
	
	// 有任务的用户数
	usersWithTasks, err := s.userDAO.GetUsersWithTasks(ctx)
	if err != nil {
		return nil, err
	}
	stats["users_with_tasks"] = len(usersWithTasks)
	
	return stats, nil
}

// 私有方法：业务逻辑辅助方法

// validateUserCreate 验证用户创建请求
func (s *UserServiceWithDAO) validateUserCreate(ctx context.Context, req *models.UserCreateRequest) error {
	// 检查用户名是否已存在
	if _, err := s.userDAO.GetByUsername(ctx, req.Username); err == nil {
		return fmt.Errorf("用户名已存在: %s", req.Username)
	}
	
	// 检查邮箱是否已存在
	if _, err := s.userDAO.GetByEmail(ctx, req.Email); err == nil {
		return fmt.Errorf("邮箱已存在: %s", req.Email)
	}
	
	// 其他业务验证...
	
	return nil
}

// 缓存相关方法
func (s *UserServiceWithDAO) cacheUser(user *models.User) {
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, user.ID)
	if err := s.cache.Set(cacheKey, user.ToResponse(), time.Hour); err != nil {
		// 记录日志，不影响主流程
		fmt.Printf("缓存用户信息失败: %v\n", err)
	}
}

func (s *UserServiceWithDAO) getUserFromCache(id uint) *models.User {
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, id)
	var userResponse models.UserResponse
	if err := s.cache.Get(cacheKey, &userResponse); err != nil {
		return nil
	}
	
	// 转换为完整用户对象（简化处理）
	return &models.User{
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
}

func (s *UserServiceWithDAO) clearUserCache(id uint) {
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, id)
	if err := s.cache.Delete(cacheKey); err != nil {
		fmt.Printf("清除用户缓存失败: %v\n", err)
	}
}

func (s *UserServiceWithDAO) clearUserTasksCache(userID uint) {
	cacheKey := redis.BuildCacheKey(redis.UserTasksPrefix, userID)
	if err := s.cache.Delete(cacheKey); err != nil {
		fmt.Printf("清除用户任务缓存失败: %v\n", err)
	}
}

func (s *UserServiceWithDAO) getSearchResultFromCache(key string) *models.PageResult {
	var result models.PageResult
	if err := s.cache.Get(key, &result); err != nil {
		return nil
	}
	return &result
}

func (s *UserServiceWithDAO) cacheSearchResult(key string, result *models.PageResult, duration time.Duration) {
	if err := s.cache.Set(key, result, duration); err != nil {
		fmt.Printf("缓存搜索结果失败: %v\n", err)
	}
}

func (s *UserServiceWithDAO) getActiveUsersFromCache(key string) []models.UserResponse {
	var users []models.UserResponse
	if err := s.cache.Get(key, &users); err != nil {
		return nil
	}
	return users
}

func (s *UserServiceWithDAO) cacheActiveUsers(key string, users []models.UserResponse, duration time.Duration) {
	if err := s.cache.Set(key, users, duration); err != nil {
		fmt.Printf("缓存活跃用户失败: %v\n", err)
	}
}