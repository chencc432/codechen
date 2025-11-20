package main

import (
	"github.com/gin-gonic/gin"
)

type LoginForm struct {
	Username string `json:"username"`
	Password string `json:"password"`
	//PP       string `json:"pp" binding:"required"`
}

func main() {

	router := gin.Default()
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello, world!",
		})
	})
	router.POST("/login", func(c *gin.Context) {
		var form LoginForm
		if err := c.ShouldBindJSON(&form); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}
		c.JSON(200, gin.H{
			"message": "Login success!",
		})

	})
	router.GET("/p", func(c *gin.Context) {
		c.String(200, "Hello, Gin!")
	})

	router.GET("/search", func(c *gin.Context) {
		query := c.Query("q")               // 获取参数，没有则返回空字符串
		page := c.DefaultQuery("page", "1") // 获取参数，没有则使用默认值

		c.JSON(200, gin.H{
			"query": query,
			"page":  page,
		})
	})
	router.Run(":8080")
}
