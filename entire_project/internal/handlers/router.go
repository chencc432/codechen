package handlers

import (
	"github.com/gin-gonic/gin"
	"task-management-system/internal/middleware"
)

// SetupRoutes 设置路由
// 学习要点：路由组织，中间件应用，RESTful API设计
func SetupRoutes() *gin.Engine {
	// 创建Gin路由器
	r := gin.New()
	
	// 添加中间件
	// 学习要点：中间件的使用顺序很重要
	r.Use(gin.Logger())                          // 日志中间件
	r.Use(gin.Recovery())                        // 恢复中间件（防止panic导致程序崩溃）
	r.Use(middleware.CorsMiddleware())           // CORS中间件
	r.Use(middleware.RequestIDMiddleware())      // 请求ID中间件
	
	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "任务管理系统运行正常",
		})
	})
	
	// 创建处理器实例
	userHandler := NewUserHandler()
	taskHandler := NewTaskHandler()
	
	// API路由组
	// 学习要点：路由组的使用，版本控制
	api := r.Group("/api")
	{
		// v1版本路由
		v1 := api.Group("/v1")
		{
			// 用户相关路由
			// 学习要点：RESTful风格的路由设计
			users := v1.Group("/users")
			{
				users.POST("", userHandler.CreateUser)                           // 创建用户
				users.GET("", userHandler.GetUserList)                          // 获取用户列表
				users.GET("/:id", userHandler.GetUser)                          // 获取单个用户
				users.PUT("/:id", userHandler.UpdateUser)                       // 更新用户
				users.DELETE("/:id", userHandler.DeleteUser)                    // 删除用户
				users.GET("/username/:username", userHandler.GetUserByUsername) // 根据用户名获取用户
				users.POST("/:id/login", userHandler.UpdateLastLoginTime)       // 更新登录时间
				
				// 用户相关的任务路由
				// 学习要点：嵌套资源的路由设计
				users.GET("/:user_id/tasks", taskHandler.GetUserTasks)          // 获取用户任务列表
				users.GET("/:user_id/tasks/stats", taskHandler.GetUserTaskStats) // 获取用户任务统计
			}
			
			// 任务相关路由
			tasks := v1.Group("/tasks")
			{
				tasks.POST("", taskHandler.CreateTask)                          // 创建任务
				tasks.GET("", taskHandler.QueryTasks)                           // 查询任务列表
				tasks.GET("/:id", taskHandler.GetTask)                          // 获取任务详情
				tasks.PUT("/:id", taskHandler.UpdateTask)                       // 更新任务
				tasks.DELETE("/:id", taskHandler.DeleteTask)                    // 删除任务
				tasks.POST("/:id/complete", taskHandler.MarkTaskComplete)       // 标记任务完成
			}
			
			// 标签相关路由
			tags := v1.Group("/tags")
			{
				// 标签相关的任务路由
				tags.GET("/:tag_id/tasks", taskHandler.GetTasksByTag)           // 根据标签获取任务
				
				// TODO: 这里可以添加标签的CRUD操作
				// tags.POST("", tagHandler.CreateTag)                          // 创建标签
				// tags.GET("", tagHandler.GetTagList)                          // 获取标签列表
				// tags.GET("/:id", tagHandler.GetTag)                          // 获取标签详情
				// tags.PUT("/:id", tagHandler.UpdateTag)                       // 更新标签
				// tags.DELETE("/:id", tagHandler.DeleteTag)                    // 删除标签
			}
		}
	}
	
	// 管理员路由组（需要管理员权限）
	// 学习要点：权限控制，中间件链式调用
	admin := api.Group("/admin")
	// admin.Use(middleware.AdminAuthMiddleware()) // 管理员认证中间件（暂未实现）
	{
		adminV1 := admin.Group("/v1")
		{
			// 管理员专用的用户管理接口
			adminUsers := adminV1.Group("/users")
			{
				adminUsers.GET("/stats", func(c *gin.Context) {
					// TODO: 实现用户统计接口
					c.JSON(200, gin.H{"message": "用户统计接口待实现"})
				})
			}
			
			// 管理员专用的任务管理接口
			adminTasks := adminV1.Group("/tasks")
			{
				adminTasks.GET("/stats", func(c *gin.Context) {
					// TODO: 实现任务统计接口
					c.JSON(200, gin.H{"message": "任务统计接口待实现"})
				})
			}
		}
	}
	
	return r
}

// RouteInfo 路由信息结构
// 学习要点：路由文档化，API接口清单
type RouteInfo struct {
	Method      string `json:"method"`      // HTTP方法
	Path        string `json:"path"`        // 路径
	Handler     string `json:"handler"`     // 处理函数
	Description string `json:"description"` // 描述
}

// GetRouteList 获取路由列表
// 学习要点：动态获取路由信息，API文档生成
func GetRouteList(r *gin.Engine) []RouteInfo {
	var routes []RouteInfo
	
	// 遍历所有路由
	for _, route := range r.Routes() {
		routes = append(routes, RouteInfo{
			Method:      route.Method,
			Path:        route.Path,
			Handler:     route.Handler,
			Description: "", // 可以从注释或其他地方获取描述
		})
	}
	
	return routes
}