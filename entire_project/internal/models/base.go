// Package models 数据模型定义
// 学习要点：GORM 模型设计，数据库表关系，软删除，时间戳
package models

import (
	"time"

	"gorm.io/gorm"
)

// BaseModel 基础模型
// 学习要点：GORM 的基础模型设计，包含常用字段
type BaseModel struct {
	ID        uint           `gorm:"primarykey;comment:主键ID" json:"id"`                    // 主键ID
	CreatedAt time.Time      `gorm:"comment:创建时间" json:"created_at"`                       // 创建时间
	UpdatedAt time.Time      `gorm:"comment:更新时间" json:"updated_at"`                       // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"index;comment:删除时间" json:"deleted_at,omitempty"`       // 删除时间（软删除）
}

// TableName 接口用于自定义表名
type TableName interface {
	TableName() string
}

// Response 统一响应结构
// 学习要点：API 响应标准化
type Response struct {
	Code    int         `json:"code"`    // 状态码：200成功，其他失败
	Message string      `json:"message"` // 响应消息
	Data    interface{} `json:"data"`    // 响应数据
}

// PageInfo 分页信息
// 学习要点：分页查询的标准结构
type PageInfo struct {
	Page     int   `json:"page"`      // 当前页码
	PageSize int   `json:"page_size"` // 每页数量
	Total    int64 `json:"total"`     // 总记录数
}

// PageResult 分页结果
type PageResult struct {
	List     interface{} `json:"list"`      // 数据列表
	PageInfo PageInfo    `json:"page_info"` // 分页信息
}

// NewResponse 创建响应
func NewResponse(code int, message string, data interface{}) Response {
	return Response{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

// NewSuccessResponse 创建成功响应
func NewSuccessResponse(data interface{}) Response {
	return NewResponse(200, "操作成功", data)
}

// NewErrorResponse 创建错误响应
func NewErrorResponse(message string) Response {
	return NewResponse(500, message, nil)
}