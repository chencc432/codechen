package models

import "time"

// User 用户模型
// 学习要点：用户表设计，字段约束，索引设置
type User struct {
	BaseModel
	Username    string    `gorm:"uniqueIndex;size:50;not null;comment:用户名" json:"username"`        // 用户名（唯一索引）
	Email       string    `gorm:"uniqueIndex;size:100;not null;comment:邮箱" json:"email"`           // 邮箱（唯一索引）
	Password    string    `gorm:"size:255;not null;comment:密码" json:"-"`                           // 密码（不返回给前端）
	Nickname    string    `gorm:"size:50;comment:昵称" json:"nickname"`                              // 昵称
	Avatar      string    `gorm:"size:255;comment:头像" json:"avatar"`                              // 头像URL
	Phone       string    `gorm:"size:20;comment:手机号" json:"phone"`                               // 手机号
	Status      int       `gorm:"default:1;comment:状态 1-正常 0-禁用" json:"status"`                   // 状态
	LastLoginAt *time.Time `gorm:"comment:最后登录时间" json:"last_login_at"`                           // 最后登录时间
	
	// 关联关系
	Tasks []Task `gorm:"foreignKey:UserID;comment:用户的任务" json:"tasks,omitempty"` // 一对多：用户拥有多个任务
}

// TableName 自定义表名
// 学习要点：GORM 自定义表名的方法
func (User) TableName() string {
	return "users"
}

// UserCreateRequest 创建用户请求
type UserCreateRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"` // 用户名（必填，3-50字符）
	Email    string `json:"email" binding:"required,email"`           // 邮箱（必填，邮箱格式）
	Password string `json:"password" binding:"required,min=6"`        // 密码（必填，最少6位）
	Nickname string `json:"nickname" binding:"max=50"`                // 昵称（最多50字符）
	Phone    string `json:"phone" binding:"max=20"`                   // 手机号（最多20字符）
}

// UserUpdateRequest 更新用户请求
type UserUpdateRequest struct {
	Nickname string `json:"nickname" binding:"max=50"` // 昵称
	Avatar   string `json:"avatar"`                    // 头像
	Phone    string `json:"phone" binding:"max=20"`    // 手机号
}

// UserResponse 用户响应（不包含敏感信息）
type UserResponse struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Nickname    string     `json:"nickname"`
	Avatar      string     `json:"avatar"`
	Phone       string     `json:"phone"`
	Status      int        `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// ToResponse 转换为响应格式
// 学习要点：数据传输对象（DTO）的使用，隐藏敏感字段
func (u *User) ToResponse() UserResponse {
	return UserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Nickname:    u.Nickname,
		Avatar:      u.Avatar,
		Phone:       u.Phone,
		Status:      u.Status,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}