package models

import(
	"time"
	"gorm.io/gorm"
)

type BaseModel struct{
	ID        uint           `gorm:"primarykey;comment:主键ID" json:"id"`                    // 主键ID
	CreatedAt time.Time      `gorm:"comment:创建时间" json:"created_at"`                       // 创建时间
	UpdatedAt time.Time      `gorm:"comment:更新时间" json:"updated_at"`                       // 更新时间
	DeletedAt gorm.DeletedAt `gorm:"index;comment:删除时间" json:"deleted_at,omitempty"`       // 删除时间（软删除）
}



type TableName interface{
	TableName() string
}

type Response struct{
	Code    int         `json:"code"`    // 状态码：200成功，其他失败
	Message string      `json:"message"` // 响应消息
	Data    interface{} `json:"data"`    // 响应数据
}


type PageInfo struct{
	Page  int `json:"page"`      // 当前页码
	PageSize int `json:"page_size"` // 每页数量
	Total int64 `json:"total"` // 总记录数
}

type PageResult struct{
	List     interface{} `json:"list"`      // 数据列表
	PageInfo PageInfo    `json:"page_info"` // 分页信息
}

func NewResponse(code int,message string,data interface{}) Response {
	return Response{
		Code: code,
		Message: message,
		Data:data,
	}
}

func NewSuccessResponse(data interface{})Response{
	return NewResponse(200,"操作成功",data)
}

func NewErrorResponse(message string)Response{
	return NewResponse(500,message,nil)
}