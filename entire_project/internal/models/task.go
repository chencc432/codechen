package models

import "time"

// TaskStatus 任务状态枚举
// 学习要点：使用常量定义状态枚举，提高代码可维护性
const (
	TaskStatusPending    = 0 // 待处理
	TaskStatusInProgress = 1 // 进行中
	TaskStatusCompleted  = 2 // 已完成
	TaskStatusCancelled  = 3 // 已取消
)

// TaskPriority 任务优先级枚举
const (
	TaskPriorityLow    = 1 // 低优先级
	TaskPriorityMedium = 2 // 中优先级
	TaskPriorityHigh   = 3 // 高优先级
	TaskPriorityUrgent = 4 // 紧急
)

// Task 任务模型
// 学习要点：复杂模型设计，外键关系，多对多关系
type Task struct {
	BaseModel
	Title       string     `gorm:"size:200;not null;comment:任务标题" json:"title"`                    // 任务标题
	Description string     `gorm:"type:text;comment:任务描述" json:"description"`                      // 任务描述
	Status      int        `gorm:"default:0;comment:任务状态 0-待处理 1-进行中 2-已完成 3-已取消" json:"status"`   // 任务状态
	Priority    int        `gorm:"default:2;comment:优先级 1-低 2-中 3-高 4-紧急" json:"priority"`        // 优先级
	StartTime   *time.Time `gorm:"comment:开始时间" json:"start_time"`                                 // 开始时间
	EndTime     *time.Time `gorm:"comment:结束时间" json:"end_time"`                                   // 结束时间
	DueDate     *time.Time `gorm:"comment:截止日期" json:"due_date"`                                   // 截止日期
	UserID      uint       `gorm:"not null;comment:创建用户ID" json:"user_id"`                        // 创建用户ID（外键）
	
	// 关联关系
	User User    `gorm:"foreignKey:UserID;comment:任务创建者" json:"user,omitempty"`        // 多对一：任务属于一个用户
	Tags []Tag   `gorm:"many2many:task_tags;comment:任务标签" json:"tags,omitempty"`      // 多对多：任务可以有多个标签
}

// TableName 自定义表名
func (Task) TableName() string {
	return "tasks"
}

// Tag 标签模型
// 学习要点：标签系统设计，多对多关系
type Tag struct {
	BaseModel
	Name  string `gorm:"uniqueIndex;size:50;not null;comment:标签名称" json:"name"`  // 标签名称（唯一索引）
	Color string `gorm:"size:7;comment:标签颜色" json:"color"`                      // 标签颜色（十六进制）
	
	// 关联关系
	Tasks []Task `gorm:"many2many:task_tags;comment:标签的任务" json:"tasks,omitempty"` // 多对多：标签可以属于多个任务
}

// TableName 自定义表名
func (Tag) TableName() string {
	return "tags"
}

// TaskCreateRequest 创建任务请求
type TaskCreateRequest struct {
	Title       string     `json:"title" binding:"required,max=200"`       // 任务标题（必填）
	Description string     `json:"description"`                            // 任务描述
	Priority    int        `json:"priority" binding:"min=1,max=4"`         // 优先级（1-4）
	DueDate     *time.Time `json:"due_date"`                               // 截止日期
	TagIDs      []uint     `json:"tag_ids"`                                // 标签ID列表
}

// TaskUpdateRequest 更新任务请求
type TaskUpdateRequest struct {
	Title       *string    `json:"title" binding:"omitempty,max=200"`      // 任务标题
	Description *string    `json:"description"`                            // 任务描述
	Status      *int       `json:"status" binding:"omitempty,min=0,max=3"` // 任务状态（0-3）
	Priority    *int       `json:"priority" binding:"omitempty,min=1,max=4"` // 优先级（1-4）
	StartTime   *time.Time `json:"start_time"`                             // 开始时间
	EndTime     *time.Time `json:"end_time"`                               // 结束时间
	DueDate     *time.Time `json:"due_date"`                               // 截止日期
	TagIDs      []uint     `json:"tag_ids"`                                // 标签ID列表
}

// TaskQueryRequest 任务查询请求
type TaskQueryRequest struct {
	Status   *int   `form:"status"`    // 任务状态
	Priority *int   `form:"priority"`  // 优先级
	TagID    *uint  `form:"tag_id"`    // 标签ID
	UserID   *uint  `form:"user_id"`   // 用户ID
	Keyword  string `form:"keyword"`   // 关键词搜索
	Page     int    `form:"page"`      // 页码
	PageSize int    `form:"page_size"` // 每页数量
}

// TagCreateRequest 创建标签请求
type TagCreateRequest struct {
	Name  string `json:"name" binding:"required,max=50"`  // 标签名称（必填）
	Color string `json:"color" binding:"omitempty,len=7"` // 标签颜色（7位十六进制）
}

// GetStatusText 获取状态文本
// 学习要点：枚举值的文本转换
func (t *Task) GetStatusText() string {
	switch t.Status {
	case TaskStatusPending:
		return "待处理"
	case TaskStatusInProgress:
		return "进行中"
	case TaskStatusCompleted:
		return "已完成"
	case TaskStatusCancelled:
		return "已取消"
	default:
		return "未知状态"
	}
}

// GetPriorityText 获取优先级文本
func (t *Task) GetPriorityText() string {
	switch t.Priority {
	case TaskPriorityLow:
		return "低"
	case TaskPriorityMedium:
		return "中"
	case TaskPriorityHigh:
		return "高"
	case TaskPriorityUrgent:
		return "紧急"
	default:
		return "未知"
	}
}

// IsOverdue 判断任务是否过期
// 学习要点：业务逻辑方法的设计
func (t *Task) IsOverdue() bool {
	if t.DueDate == nil {
		return false
	}
	return time.Now().After(*t.DueDate) && t.Status != TaskStatusCompleted
}