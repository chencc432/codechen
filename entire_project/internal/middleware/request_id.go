package middleware

import (
	"github.com/gin-gonic/gin"
	"task-management-system/pkg/utils"
)

// RequestIDMiddleware 请求ID中间件
// 学习要点：分布式链路追踪的基础，请求唯一标识
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 尝试从请求头获取请求ID
		requestID := c.GetHeader("X-Request-ID")
		
		// 如果请求头没有提供请求ID，则生成一个新的
		if requestID == "" {
			requestID = utils.GenerateRequestID()
		}
		
		// 设置请求ID到上下文中
		c.Set("request_id", requestID)
		
		// 在响应头中返回请求ID
		c.Header("X-Request-ID", requestID)
		
		// 继续处理请求
		c.Next()
	}
}