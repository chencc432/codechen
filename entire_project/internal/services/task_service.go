package services

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"task-management-system/internal/database"
	"task-management-system/internal/models"
	"task-management-system/pkg/redis"
)

// TaskService 任务服务结构体
// 学习要点：复杂业务逻辑处理，多表关联查询，缓存策略
type TaskService struct {
	db    *gorm.DB
	cache *redis.CacheService
}

// NewTaskService 创建任务服务实例
func NewTaskService() *TaskService {
	return &TaskService{
		db:    database.DB,
		cache: redis.NewCacheService(),
	}
}

// CreateTask 创建任务
// 学习要点：关联数据处理，事务管理，多对多关系
func (s *TaskService) CreateTask(userID uint, req *models.TaskCreateRequest) (*models.Task, error) {
	// 验证用户是否存在
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在: ID=%d", userID)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}
	
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// 创建任务
	task := &models.Task{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
		Status:      models.TaskStatusPending, // 默认状态为待处理
		UserID:      userID,
	}
	
	if err := tx.Create(task).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}
	
	// 关联标签（多对多关系）
	if len(req.TagIDs) > 0 {
		var tags []models.Tag
		if err := tx.Where("id IN ?", req.TagIDs).Find(&tags).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("查询标签失败: %w", err)
		}
		
		// 关联标签到任务
		if err := tx.Model(task).Association("Tags").Append(tags); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("关联标签失败: %w", err)
		}
	}
	
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	
	// 预加载关联数据
	if err := s.db.Preload("User").Preload("Tags").First(task, task.ID).Error; err != nil {
		fmt.Printf("预加载任务关联数据失败: %v\n", err)
	}
	
	// 缓存任务信息
	cacheKey := redis.BuildCacheKey(redis.TaskCachePrefix, task.ID)
	if err := s.cache.Set(cacheKey, task, time.Hour); err != nil {
		fmt.Printf("缓存任务信息失败: %v\n", err)
	}
	
	// 清除用户任务列表缓存
	userTasksKey := redis.BuildCacheKey(redis.UserTasksPrefix, userID)
	if err := s.cache.Delete(userTasksKey); err != nil {
		fmt.Printf("清除用户任务缓存失败: %v\n", err)
	}
	
	// 更新任务统计计数器
	s.updateTaskStats(userID, models.TaskStatusPending, 1)
	
	return task, nil
}

// GetTaskByID 根据ID获取任务
// 学习要点：预加载关联数据，缓存策略
func (s *TaskService) GetTaskByID(id uint) (*models.Task, error) {
	cacheKey := redis.BuildCacheKey(redis.TaskCachePrefix, id)
	
	// 尝试从缓存获取
	var task models.Task
	if err := s.cache.Get(cacheKey, &task); err == nil {
		return &task, nil
	}
	
	// 缓存未命中，从数据库查询（预加载关联数据）
	if err := s.db.Preload("User").Preload("Tags").First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在: ID=%d", id)
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	
	// 更新缓存
	if err := s.cache.Set(cacheKey, &task, time.Hour); err != nil {
		fmt.Printf("缓存任务信息失败: %v\n", err)
	}
	
	return &task, nil
}

// UpdateTask 更新任务
// 学习要点：部分更新，状态变更处理，关联数据更新
func (s *TaskService) UpdateTask(id uint, userID uint, req *models.TaskUpdateRequest) (*models.Task, error) {
	// 获取现有任务
	task, err := s.GetTaskByID(id)
	if err != nil {
		return nil, err
	}
	
	// 验证权限（只有任务创建者可以修改）
	if task.UserID != userID {
		return nil, fmt.Errorf("没有权限修改此任务")
	}
	
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// 记录状态变更（用于统计计数器更新）
	oldStatus := task.Status
	
	// 准备更新数据
	updates := make(map[string]interface{})
	if req.Title != nil {
		updates["title"] = *req.Title
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
		// 状态变更时的特殊处理
		switch *req.Status {
		case models.TaskStatusInProgress:
			if task.StartTime == nil {
				now := time.Now()
				updates["start_time"] = &now
			}
		case models.TaskStatusCompleted:
			if task.EndTime == nil {
				now := time.Now()
				updates["end_time"] = &now
			}
		}
	}
	if req.Priority != nil {
		updates["priority"] = *req.Priority
	}
	if req.StartTime != nil {
		updates["start_time"] = req.StartTime
	}
	if req.EndTime != nil {
		updates["end_time"] = req.EndTime
	}
	if req.DueDate != nil {
		updates["due_date"] = req.DueDate
	}
	
	// 执行更新
	if err := tx.Model(task).Updates(updates).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新任务失败: %w", err)
	}
	
	// 更新标签关联
	if req.TagIDs != nil {
		// 清除现有标签关联
		if err := tx.Model(task).Association("Tags").Clear(); err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("清除标签关联失败: %w", err)
		}
		
		// 添加新的标签关联
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
	}
	
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}
	
	// 重新加载任务数据
	if err := s.db.Preload("User").Preload("Tags").First(task, id).Error; err != nil {
		return nil, fmt.Errorf("重新加载任务数据失败: %w", err)
	}
	
	// 删除缓存（让下次查询时重新缓存）
	cacheKey := redis.BuildCacheKey(redis.TaskCachePrefix, id)
	if err := s.cache.Delete(cacheKey); err != nil {
		fmt.Printf("删除任务缓存失败: %v\n", err)
	}
	
	// 清除用户任务列表缓存
	userTasksKey := redis.BuildCacheKey(redis.UserTasksPrefix, userID)
	if err := s.cache.Delete(userTasksKey); err != nil {
		fmt.Printf("清除用户任务缓存失败: %v\n", err)
	}
	
	// 更新任务统计（如果状态发生变更）
	if req.Status != nil && oldStatus != *req.Status {
		s.updateTaskStats(userID, oldStatus, -1)    // 减少原状态计数
		s.updateTaskStats(userID, *req.Status, 1)   // 增加新状态计数
	}
	
	return task, nil
}

// DeleteTask 删除任务
func (s *TaskService) DeleteTask(id uint, userID uint) error {
	// 获取任务
	task, err := s.GetTaskByID(id)
	if err != nil {
		return err
	}
	
	// 验证权限
	if task.UserID != userID {
		return fmt.Errorf("没有权限删除此任务")
	}
	
	// 开始事务
	tx := s.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// 删除标签关联
	if err := tx.Model(task).Association("Tags").Clear(); err != nil {
		tx.Rollback()
		return fmt.Errorf("清除标签关联失败: %w", err)
	}
	
	// 软删除任务
	if err := tx.Delete(task).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("删除任务失败: %w", err)
	}
	
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	
	// 删除相关缓存
	cacheKey := redis.BuildCacheKey(redis.TaskCachePrefix, id)
	if err := s.cache.Delete(cacheKey); err != nil {
		fmt.Printf("删除任务缓存失败: %v\n", err)
	}
	
	// 清除用户任务列表缓存
	userTasksKey := redis.BuildCacheKey(redis.UserTasksPrefix, userID)
	if err := s.cache.Delete(userTasksKey); err != nil {
		fmt.Printf("清除用户任务缓存失败: %v\n", err)
	}
	
	// 更新任务统计
	s.updateTaskStats(userID, task.Status, -1)
	
	return nil
}

// QueryTasks 查询任务（支持多种过滤条件和分页）
// 学习要点：复杂查询构建，动态条件，关联查询优化
func (s *TaskService) QueryTasks(req *models.TaskQueryRequest) (*models.PageResult, error) {
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	
	// 构建查询
	query := s.db.Model(&models.Task{})
	
	// 添加查询条件
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	if req.Priority != nil {
		query = query.Where("priority = ?", *req.Priority)
	}
	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}
	if req.TagID != nil {
		// 关联查询：查找包含指定标签的任务
		query = query.Joins("JOIN task_tags ON tasks.id = task_tags.task_id").
			Where("task_tags.tag_id = ?", *req.TagID)
	}
	if req.Keyword != "" {
		// 模糊搜索：标题或描述包含关键词
		keyword := "%" + req.Keyword + "%"
		query = query.Where("title LIKE ? OR description LIKE ?", keyword, keyword)
	}
	
	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("查询任务总数失败: %w", err)
	}
	
	// 分页查询
	var tasks []models.Task
	offset := (req.Page - 1) * req.PageSize
	if err := query.Preload("User").Preload("Tags").
		Limit(req.PageSize).Offset(offset).
		Order("created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, fmt.Errorf("查询任务列表失败: %w", err)
	}
	
	// 构建返回结果
	result := &models.PageResult{
		List: tasks,
		PageInfo: models.PageInfo{
			Page:     req.Page,
			PageSize: req.PageSize,
			Total:    total,
		},
	}
	
	return result, nil
}

// GetUserTaskStats 获取用户任务统计
// 学习要点：统计查询，缓存计数器使用
func (s *TaskService) GetUserTaskStats(userID uint) (map[string]int64, error) {
	stats := make(map[string]int64)
	
	// 尝试从缓存获取统计数据
	cacheKey := redis.BuildCacheKey(redis.TaskCountPrefix, userID)
	if err := s.cache.HGetAll(cacheKey); err == nil {
		// 如果缓存存在且有数据，直接返回
		// 这里简化处理，实际项目中需要转换数据类型
	}
	
	// 从数据库统计各状态的任务数量
	statusList := []int{
		models.TaskStatusPending,
		models.TaskStatusInProgress,
		models.TaskStatusCompleted,
		models.TaskStatusCancelled,
	}
	
	for _, status := range statusList {
		var count int64
		if err := s.db.Model(&models.Task{}).
			Where("user_id = ? AND status = ?", userID, status).
			Count(&count).Error; err != nil {
			return nil, fmt.Errorf("统计状态 %d 的任务数量失败: %w", status, err)
		}
		
		// 根据状态设置统计键名
		var key string
		switch status {
		case models.TaskStatusPending:
			key = "pending"
		case models.TaskStatusInProgress:
			key = "in_progress"
		case models.TaskStatusCompleted:
			key = "completed"
		case models.TaskStatusCancelled:
			key = "cancelled"
		}
		stats[key] = count
		
		// 更新缓存计数器
		countKey := fmt.Sprintf("%s%d:%s", redis.TaskCountPrefix, userID, key)
		if err := s.cache.Set(countKey, count, time.Hour*24); err != nil {
			fmt.Printf("缓存任务统计失败: %v\n", err)
		}
	}
	
	// 统计总任务数
	var totalCount int64
	if err := s.db.Model(&models.Task{}).
		Where("user_id = ?", userID).
		Count(&totalCount).Error; err != nil {
		return nil, fmt.Errorf("统计总任务数量失败: %w", err)
	}
	stats["total"] = totalCount
	
	// 统计过期任务数
	var overdueCount int64
	if err := s.db.Model(&models.Task{}).
		Where("user_id = ? AND due_date < ? AND status != ?", 
			userID, time.Now(), models.TaskStatusCompleted).
		Count(&overdueCount).Error; err != nil {
		return nil, fmt.Errorf("统计过期任务数量失败: %w", err)
	}
	stats["overdue"] = overdueCount
	
	return stats, nil
}

// updateTaskStats 更新任务统计计数器
// 学习要点：Redis计数器的使用，原子操作
func (s *TaskService) updateTaskStats(userID uint, status int, delta int64) {
	var key string
	switch status {
	case models.TaskStatusPending:
		key = "pending"
	case models.TaskStatusInProgress:
		key = "in_progress"
	case models.TaskStatusCompleted:
		key = "completed"
	case models.TaskStatusCancelled:
		key = "cancelled"
	default:
		return
	}
	
	countKey := fmt.Sprintf("%s%d:%s", redis.TaskCountPrefix, userID, key)
	if delta > 0 {
		if _, err := s.cache.IncrBy(countKey, delta); err != nil {
			fmt.Printf("增加任务统计计数失败: %v\n", err)
		}
	} else {
		if _, err := s.cache.DecrBy(countKey, -delta); err != nil {
			fmt.Printf("减少任务统计计数失败: %v\n", err)
		}
	}
	
	// 设置过期时间
	if err := s.cache.SetExpire(countKey, time.Hour*24); err != nil {
		fmt.Printf("设置统计计数过期时间失败: %v\n", err)
	}
}