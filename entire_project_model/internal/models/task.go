package models
import "time"

const (
	TaskStatusPending    = 0 // 待处理
	TaskStatusInProgress = 1 // 进行中
	TaskStatusCompleted  = 2 // 已完成
	TaskStatusCancelled  = 3 // 已取消
)

const(
	TaskPriorityLow    = 1 // 低优先级
	TaskPriorityMedium = 2 // 中优先级
	TaskPriorityHigh   = 3 // 高优先级
	TaskPriorityUrgent = 4 // 紧急
)

type Task struct{
	BaseModel
	Title string `gorm:"size:200;not null;comment:任务标题" json:"title"`
	Description string `gorm:"type:text;comment:任务描述" json:"description"`
	Status int `gorm:"default:0;comment:任务状态 0-待处理 1-进行中 2-已完成 3-已取消" json:"status"`
	Priority int `gorm:"default:2;comment:优先级 1-低 2-中 3-高 4-紧急" json:"priority"`
	StartTime  *time.Time `gorm:"comment:开始时间" json:"start_time"`
	EndTime *time.Time `gorm:"comment:结束时间" json:"end_time"`
	DueDate *time.Time `gorm:"comment:截止日期" json:"due_date"`
	UserID uint `gorm:"not null;comment:创建用户ID" json:"user_id"`

	User User `gorm:"foreignKey:UserID;comment:任务创建者" json:"user,omitempty"`
	Tags []Tag `gorm:"many2many:task_tags;comment:任务标签" json:"tags,omitempty"`
}

func (Task)TableName() string{
	return "tasks"
}


type Tag struct{
	BaseModel
	Name string `gorm:"uniqueIndex;size:50;not null;comment:标签名称" json:"name"`
	Color string `gorm:"size:7;comment:标签颜色" json:"color"`
	Tasks []Task `gorm:"many2many:task_tags;comment:标签的任务" json:"tasks,omitempty"`

}

func (Tag)TableName() string{
	return "tags"
}

type TaskCreateRequest struct{
	Title string `json:"title" binding:"required,max=200"`
	Description string `json:"description"`
	Priority int `json:"priority" binding:"min=1,max=4"`
	DueDate *time.Time `json:"due_date"`
	TagIDs []uint `json:"tag_ids"`
}


type TaskUpdateRequest struct{
	Title *string `json:"title" binding:"omitempty,max=200"`
	Description *string `json:"description"`
	Status *int `json:"status" binding:"omitempty,min=0,max=3"`
	Priority *int `json:"priority" binding:"omitempty,min=1,max=4"`
	StartTime *time.Time `json:"start_time"`
	EndTime *time.Time `json:"end_time"`
	DueDate *time.Time `json:"due_date"`
	TagIDs []uint `json:"tag_ids"`
}


type TaskQueryRequest struct{
	Status *int `form:"status"`
	Priority *int `form:"priority"`
	TagID *uint `form:"tag_id"`
	UserID *uint `form:"user_id"`
	Keyword string `form:"keyword"`
	Page int `form:"page"`
	PageSize int `form:"page_size"`
}

type TagCreateRequest struct{
	Name string `json:"name" binding:"required,max=50"`
	Color string `json:"color" binding:"omitempty,len=7"`
}

func (t *Task)GetStatusText() string{
	switch t.Status{
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

func (t *Task)GetPriorityText() string{
	switch t.Priority{
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

func (t *Task)IsOverdue() bool{
	if t.DueDate == nil{
		return false
	}
	return time.Now().After(*t.DueDate) && t.Status != TaskStatusCompleted
}