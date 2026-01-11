// Package middleware HTTP中间件
// 学习要点：中间件设计模式，HTTP头处理，CORS配置
package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CorsMiddleware CORS跨域中间件
// 学习要点：跨域资源共享(CORS)的处理，安全头设置
func CorsMiddleware() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// 设置CORS响应头
		// 学习要点：各种CORS头的含义和作用
		c.Header("Access-Control-Allow-Origin", "*")                                         // 允许所有域名访问
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,HEAD,OPTIONS")  // 允许的HTTP方法
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Length,Content-Type,X-User-ID,Authorization,X-Request-ID") // 允许的请求头
		c.Header("Access-Control-Expose-Headers", "Content-Length,Access-Control-Allow-Origin,Access-Control-Allow-Headers") // 暴露的响应头
		c.Header("Access-Control-Allow-Credentials", "true")                                 // 允许携带认证信息
		c.Header("Access-Control-Max-Age", "86400")                                          // 预检请求缓存时间(秒)
		
		// 处理OPTIONS预检请求
		// 学习要点：CORS预检请求的处理
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		// 继续处理请求
		c.Next()
	})
}