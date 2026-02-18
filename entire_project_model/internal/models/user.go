package models

import "time"

type User struct{
	BaseModel
	Username string `gorm:"uniqueIndex;size:50;not null;comment:用户名" json:"username"`
	Email string `gorm:"uniqueIndex;size:100;not null;comment:邮箱" json:"email"`
	Password string `gorm:"size:255;not null;comment:密码" json:"-"`
	Nickname string `gorm:"size:50;comment:昵称" json:"nickname"`
	Avatar strig `gorm:"size:255;comment:头像" json:"avatar"`
	phone string `gorm:"size:20;comment:手机号" json:"phone"`
	Status int `gorm:"default:1;comment:状态 1-正常 0-禁用" json:"status"`
	LastLoginAt *time.Time `gorm:"comment:最后登录时间" json:"last_login_at"`
	Task []Task `gorm:"foreignKey:UserID;comment:用户的任务" json:"tasks,omitempty"`
}


func (User)TableName() string{
	return "users"
}

type UserCreateRequest struct{
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	Nickname string `json:"nickname" binding:"max=50"`
	Phone string `json:"phone" binding:"max=20"`
}

type UserUpdateRequest struct{
	Nickname string `json:"nickname" binding:"max=50"`
	Avatar string `json:"avatar"`
	Phone string `json:"phone" binding:"max=20"`
}

type UserResponse struct{
	ID uint `json:"id"`
	Username string `json:"username"`
	Email string `json:"email"`
	Nickname string `json:"nickname"`
	Avatar string `json:"avatar"`
	Phone string `json:"phone"`
	Status int `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}


func (u *User)ToResponse() UserResponse{
	return UserResponse{
		ID: u.ID,
		Username: u.Username,
		Email: u.Email,
		Nickname: u.Nickname,
		Avatar: u.Avatar,
		Phone: u.Phone,
		Status: u.Status,
		LastLoginAt: u.LastLoginAt,
		CreatedAt: u.CreatedAt,
		UpdatedAt: u.UpdatedAt,
	}
}