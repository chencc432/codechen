package dao

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"task-management-system/internal/models"
)

// TaskDAO 任务数据访问接口
// 学习要点：复杂业务对象的DAO设计，关联查询
type TaskDAO interface {
	// 基础CRUD操作
	Create(ctx context.Context, task *models.Task) error
	GetByID(ctx context.Context, id uint) (*models.Task, error)
	GetByIDWithAssociations(ctx context.Context, id uint) (*models.Task, error)
	Update(ctx context.Context, task *models.Task) error
	Delete(ctx context.Context, id uint) error
	
	// 查询操作
	List(ctx context.Context, offset, limit int) ([]models.Task, int64, error)
	ListByUserID(ctx context.Context, userID uint, offset, limit int) ([]models.Task, int64, error)
	ListByStatus(ctx context.Context, status int, offset, limit int) ([]models.Task, int64, error)
	Search(ctx context.Context, keyword string, offset, limit int) ([]models.Task, int64, error)
	
	// 复杂查询
	GetTasksByFilter(ctx context.Context, filter TaskFilter) ([]models.Task, int64, error)
	GetOverdueTasks(ctx context.Context) ([]models.Task, error)
	GetTasksByPriority(ctx context.Context, priority int) ([]models.Task, error)
	GetTasksByTag(ctx context.Context, tagID uint, offset, limit int) ([]models.Task, int64, error)
	GetTasksByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.Task, error)
	
	// 统计查询
	CountByStatus(ctx context.Context, status int) (int64, error)
	CountByUserID(ctx context.Context, userID uint) (int64, error)
	CountOverdue(ctx context.Context) (int64, error)
	GetStatusStats(ctx context.Context) (map[int]int64, error)
	GetUserTaskStats(ctx context.Context, userID uint) (map[int]int64, error)
	
	// 关联操作
	AddTags(ctx context.Context, taskID uint, tagIDs []uint) error
	RemoveTags(ctx context.Context, taskID uint, tagIDs []uint) error
	ReplaceTags(ctx context.Context, taskID uint, tagIDs []uint) error
	
	// 批量操作
	BatchUpdateStatus(ctx context.Context, ids []uint, status int) error
	BatchDelete(ctx context.Context, ids []uint) error
	
	// 事务支持
	WithTx(tx *gorm.DB) TaskDAO
}

// TaskFilter 任务查询过滤器
// 学习要点：查询条件封装，复杂查询参数管理
type TaskFilter struct {
	UserID     *uint      `json:"user_id"`
	Status     *int       `json:"status"`
	Priority   *int       `json:"priority"`
	TagID      *uint      `json:"tag_id"`
	Keyword    string     `json:"keyword"`
	StartDate  *time.Time `json:"start_date"`
	EndDate    *time.Time `json:"end_date"`
	IsOverdue  *bool      `json:"is_overdue"`
	OrderBy    string     `json:"order_by"`    // created_at, priority, due_date
	OrderDesc  bool       `json:"order_desc"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
}

// taskDAO 任务DAO实现
type taskDAO struct {
	db *gorm.DB
}

// NewTaskDAO 创建任务DAO实例
func NewTaskDAO(db *gorm.DB) TaskDAO {
	return &taskDAO{db: db}
}

// Create 创建任务
func (d *taskDAO) Create(ctx context.Context, task *models.Task) error {
	if err := d.db.WithContext(ctx).Create(task).Error; err != nil {
		return fmt.Errorf("创建任务失败: %w", err)
	}
	return nil
}

// GetByID 根据ID获取任务
func (d *taskDAO) GetByID(ctx context.Context, id uint) (*models.Task, error) {
	var task models.Task
	err := d.db.WithContext(ctx).First(&task, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在: ID=%d", id)
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	return &task, nil
}

// GetByIDWithAssociations 根据ID获取任务（包含关联数据）
// 学习要点：预加载关联数据，避免N+1查询问题
func (d *taskDAO) GetByIDWithAssociations(ctx context.Context, id uint) (*models.Task, error) {
	var task models.Task
	err := d.db.WithContext(ctx).
		Preload("User").
		Preload("Tags").
		First(&task, id).Error
		
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("任务不存在: ID=%d", id)
		}
		return nil, fmt.Errorf("查询任务失败: %w", err)
	}
	return &task, nil
}

// Update 更新任务
func (d *taskDAO) Update(ctx context.Context, task *models.Task) error {
	if err := d.db.WithContext(ctx).Save(task).Error; err != nil {
		return fmt.Errorf("更新任务失败: %w", err)
	}
	return nil
}

// Delete 删除任务
func (d *taskDAO) Delete(ctx context.Context, id uint) error {
	if err := d.db.WithContext(ctx).Delete(&models.Task{}, id).Error; err != nil {
		return fmt.Errorf("删除任务失败: %w", err)
	}
	return nil
}

// List 获取任务列表
func (d *taskDAO) List(ctx context.Context, offset, limit int) ([]models.Task, int64, error) {
	var tasks []models.Task
	var total int64
	
	// 统计总数
	if err := d.db.WithContext(ctx).Model(&models.Task{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计任务总数失败: %w", err)
	}
	
	// 查询列表（预加载关联数据）
	if err := d.db.WithContext(ctx).
		Preload("User").
		Preload("Tags").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询任务列表失败: %w", err)
	}
	
	return tasks, total, nil
}

// ListByUserID 根据用户ID获取任务列表
func (d *taskDAO) ListByUserID(ctx context.Context, userID uint, offset, limit int) ([]models.Task, int64, error) {
	var tasks []models.Task
	var total int64
	
	query := d.db.WithContext(ctx).Model(&models.Task{}).Where("user_id = ?", userID)
	
	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计用户任务总数失败: %w", err)
	}
	
	// 查询列表
	if err := query.
		Preload("Tags").
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询用户任务列表失败: %w", err)
	}
	
	return tasks, total, nil
}

// GetTasksByFilter 根据过滤器获取任务
// 学习要点：动态查询构建，复杂条件组合
func (d *taskDAO) GetTasksByFilter(ctx context.Context, filter TaskFilter) ([]models.Task, int64, error) {
	var tasks []models.Task
	var total int64
	
	query := d.db.WithContext(ctx).Model(&models.Task{})
	
	// 构建查询条件
	if filter.UserID != nil {
		query = query.Where("user_id = ?", *filter.UserID)
	}
	
	if filter.Status != nil {
		query = query.Where("status = ?", *filter.Status)
	}
	
	if filter.Priority != nil {
		query = query.Where("priority = ?", *filter.Priority)
	}
	
	if filter.TagID != nil {
		query = query.Joins("JOIN task_tags ON tasks.id = task_tags.task_id").
			Where("task_tags.tag_id = ?", *filter.TagID)
	}
	
	if filter.Keyword != "" {
		searchPattern := "%" + filter.Keyword + "%"
		query = query.Where("title LIKE ? OR description LIKE ?", searchPattern, searchPattern)
	}
	
	if filter.StartDate != nil {
		query = query.Where("created_at >= ?", *filter.StartDate)
	}
	
	if filter.EndDate != nil {
		query = query.Where("created_at <= ?", *filter.EndDate)
	}
	
	if filter.IsOverdue != nil && *filter.IsOverdue {
		query = query.Where("due_date < ? AND status != ?", time.Now(), models.TaskStatusCompleted)
	}
	
	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计过滤任务总数失败: %w", err)
	}
	
	// 排序
	orderBy := "created_at"
	if filter.OrderBy != "" {
		orderBy = filter.OrderBy
	}
	if filter.OrderDesc {
		orderBy += " DESC"
	}
	
	// 分页
	offset := 0
	limit := 10
	if filter.Page > 0 && filter.PageSize > 0 {
		offset = (filter.Page - 1) * filter.PageSize
		limit = filter.PageSize
	}
	
	// 查询列表
	if err := query.
		Preload("User").
		Preload("Tags").
		Order(orderBy).
		Offset(offset).
		Limit(limit).
		Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询过滤任务失败: %w", err)
	}
	
	return tasks, total, nil
}

// GetOverdueTasks 获取过期任务
// 学习要点：时间查询，业务逻辑查询
func (d *taskDAO) GetOverdueTasks(ctx context.Context) ([]models.Task, error) {
	var tasks []models.Task
	
	now := time.Now()
	err := d.db.WithContext(ctx).
		Where("due_date < ? AND status != ?", now, models.TaskStatusCompleted).
		Preload("User").
		Find(&tasks).Error
		
	if err != nil {
		return nil, fmt.Errorf("查询过期任务失败: %w", err)
	}
	
	return tasks, nil
}

// GetTasksByTag 根据标签获取任务
// 学习要点：多对多关系查询，JOIN操作
func (d *taskDAO) GetTasksByTag(ctx context.Context, tagID uint, offset, limit int) ([]models.Task, int64, error) {
	var tasks []models.Task
	var total int64
	
	query := d.db.WithContext(ctx).
		Table("tasks").
		Joins("JOIN task_tags ON tasks.id = task_tags.task_id").
		Where("task_tags.tag_id = ?", tagID)
	
	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计标签任务总数失败: %w", err)
	}
	
	// 查询列表
	if err := query.
		Preload("User").
		Preload("Tags").
		Offset(offset).
		Limit(limit).
		Order("tasks.created_at DESC").
		Find(&tasks).Error; err != nil {
		return nil, 0, fmt.Errorf("查询标签任务失败: %w", err)
	}
	
	return tasks, total, nil
}

// GetStatusStats 获取状态统计
// 学习要点：GROUP BY聚合查询，统计报表
func (d *taskDAO) GetStatusStats(ctx context.Context) (map[int]int64, error) {
	type StatusCount struct {
		Status int   `json:"status"`
		Count  int64 `json:"count"`
	}
	
	var results []StatusCount
	err := d.db.WithContext(ctx).
		Model(&models.Task{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Find(&results).Error
		
	if err != nil {
		return nil, fmt.Errorf("查询状态统计失败: %w", err)
	}
	
	stats := make(map[int]int64)
	for _, result := range results {
		stats[result.Status] = result.Count
	}
	
	return stats, nil
}

// GetUserTaskStats 获取用户任务统计
func (d *taskDAO) GetUserTaskStats(ctx context.Context, userID uint) (map[int]int64, error) {
	type StatusCount struct {
		Status int   `json:"status"`
		Count  int64 `json:"count"`
	}
	
	var results []StatusCount
	err := d.db.WithContext(ctx).
		Model(&models.Task{}).
		Select("status, COUNT(*) as count").
		Where("user_id = ?", userID).
		Group("status").
		Find(&results).Error
		
	if err != nil {
		return nil, fmt.Errorf("查询用户任务统计失败: %w", err)
	}
	
	stats := make(map[int]int64)
	for _, result := range results {
		stats[result.Status] = result.Count
	}
	
	return stats, nil
}

// AddTags 为任务添加标签
// 学习要点：多对多关系操作，关联表管理
func (d *taskDAO) AddTags(ctx context.Context, taskID uint, tagIDs []uint) error {
	if len(tagIDs) == 0 {
		return nil
	}
	
	task, err := d.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	
	var tags []models.Tag
	if err := d.db.WithContext(ctx).Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
		return fmt.Errorf("查询标签失败: %w", err)
	}
	
	if err := d.db.WithContext(ctx).Model(task).Association("Tags").Append(tags); err != nil {
		return fmt.Errorf("添加任务标签失败: %w", err)
	}
	
	return nil
}

// ReplaceTags 替换任务标签
func (d *taskDAO) ReplaceTags(ctx context.Context, taskID uint, tagIDs []uint) error {
	task, err := d.GetByID(ctx, taskID)
	if err != nil {
		return err
	}
	
	// 清除现有标签
	if err := d.db.WithContext(ctx).Model(task).Association("Tags").Clear(); err != nil {
		return fmt.Errorf("清除任务标签失败: %w", err)
	}
	
	// 添加新标签
	if len(tagIDs) > 0 {
		var tags []models.Tag
		if err := d.db.WithContext(ctx).Where("id IN ?", tagIDs).Find(&tags).Error; err != nil {
			return fmt.Errorf("查询标签失败: %w", err)
		}
		
		if err := d.db.WithContext(ctx).Model(task).Association("Tags").Append(tags); err != nil {
			return fmt.Errorf("添加任务标签失败: %w", err)
		}
	}
	
	return nil
}

// BatchUpdateStatus 批量更新任务状态
func (d *taskDAO) BatchUpdateStatus(ctx context.Context, ids []uint, status int) error {
	if len(ids) == 0 {
		return nil
	}
	
	if err := d.db.WithContext(ctx).Model(&models.Task{}).
		Where("id IN ?", ids).
		Update("status", status).Error; err != nil {
		return fmt.Errorf("批量更新任务状态失败: %w", err)
	}
	return nil
}

// CountByStatus 根据状态统计任务数
func (d *taskDAO) CountByStatus(ctx context.Context, status int) (int64, error) {
	var count int64
	if err := d.db.WithContext(ctx).Model(&models.Task{}).Where("status = ?", status).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计任务数失败: %w", err)
	}
	return count, nil
}

// CountByUserID 统计用户任务数
func (d *taskDAO) CountByUserID(ctx context.Context, userID uint) (int64, error) {
	var count int64
	if err := d.db.WithContext(ctx).Model(&models.Task{}).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计用户任务数失败: %w", err)
	}
	return count, nil
}

// CountOverdue 统计过期任务数
func (d *taskDAO) CountOverdue(ctx context.Context) (int64, error) {
	var count int64
	now := time.Now()
	if err := d.db.WithContext(ctx).Model(&models.Task{}).
		Where("due_date < ? AND status != ?", now, models.TaskStatusCompleted).
		Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计过期任务数失败: %w", err)
	}
	return count, nil
}

// WithTx 使用事务
func (d *taskDAO) WithTx(tx *gorm.DB) TaskDAO {
	return &taskDAO{db: tx}
}

// 其他方法的实现...
func (d *taskDAO) ListByStatus(ctx context.Context, status int, offset, limit int) ([]models.Task, int64, error) {
	// 实现类似 ListByUserID 的逻辑
	return nil, 0, nil
}

func (d *taskDAO) Search(ctx context.Context, keyword string, offset, limit int) ([]models.Task, int64, error) {
	// 实现搜索逻辑
	return nil, 0, nil
}

func (d *taskDAO) GetTasksByPriority(ctx context.Context, priority int) ([]models.Task, error) {
	// 实现按优先级查询逻辑
	return nil, nil
}

func (d *taskDAO) GetTasksByDateRange(ctx context.Context, startDate, endDate time.Time) ([]models.Task, error) {
	// 实现按日期范围查询逻辑
	return nil, nil
}

func (d *taskDAO) RemoveTags(ctx context.Context, taskID uint, tagIDs []uint) error {
	// 实现移除标签逻辑
	return nil
}

func (d *taskDAO) BatchDelete(ctx context.Context, ids []uint) error {
	// 实现批量删除逻辑
	return nil
}

// 确保实现了接口
var _ TaskDAO = (*taskDAO)(nil)