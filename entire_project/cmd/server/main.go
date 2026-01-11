// Package main 主程序入口
// 学习要点：Go应用程序的启动流程，资源初始化与清理
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"task-management-system/internal/config"
	"task-management-system/internal/database"
	"task-management-system/internal/handlers"
	"task-management-system/pkg/redis"
)

// @title 任务管理系统API
// @version 1.0
// @description 基于Golang的任务管理系统，演示GORM、Redis、Gin的集成使用
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1

func main() {
	// 打印启动横幅
	printBanner()
	
	// 1. 加载配置
	// 学习要点：配置文件的加载顺序，环境变量的优先级
	if err := initConfig(); err != nil {
		log.Fatalf("配置初始化失败: %v", err)
	}
	fmt.Println("✅ 配置加载完成")
	
	// 2. 初始化数据库连接
	// 学习要点：数据库连接的初始化，连接池配置
	if err := initDatabase(); err != nil {
		log.Fatalf("数据库初始化失败: %v", err)
	}
	fmt.Println("✅ 数据库初始化完成")
	
	// 3. 初始化Redis连接
	// 学习要点：Redis连接初始化，缓存系统集成
	if err := initRedis(); err != nil {
		log.Fatalf("Redis初始化失败: %v", err)
	}
	fmt.Println("✅ Redis初始化完成")
	
	// 4. 设置路由
	// 学习要点：HTTP路由的设置，中间件的应用
	router := handlers.SetupRoutes()
	fmt.Println("✅ 路由设置完成")
	
	// 5. 启动HTTP服务器
	// 学习要点：HTTP服务器的启动，端口配置
	serverAddr := fmt.Sprintf(":%d", config.GlobalConfig.Server.Port)
	fmt.Printf("🚀 服务器启动成功，监听端口: %s\n", serverAddr)
	fmt.Printf("📖 API文档地址: http://localhost%s/swagger/index.html\n", serverAddr)
	fmt.Printf("🔍 健康检查: http://localhost%s/health\n", serverAddr)
	
	// 6. 优雅关闭处理
	// 学习要点：信号处理，资源清理，优雅关闭
	go handleGracefulShutdown()
	
	// 启动HTTP服务器
	if err := http.ListenAndServe(serverAddr, router); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}

// initConfig 初始化配置
func initConfig() error {
	// 默认配置文件路径
	configPath := "configs/config.yaml"
	
	// 检查环境变量中是否指定了配置文件路径
	if envConfigPath := os.Getenv("CONFIG_PATH"); envConfigPath != "" {
		configPath = envConfigPath
	}
	
	return config.Load(configPath)
}

// initDatabase 初始化数据库
func initDatabase() error {
	// 初始化MySQL连接
	if err := database.InitMySQL(); err != nil {
		return err
	}
	
	// 自动迁移数据库表结构
	if err := database.AutoMigrate(); err != nil {
		return err
	}
	
	// 初始化种子数据
	if err := database.SeedData(); err != nil {
		return err
	}
	
	return nil
}

// initRedis 初始化Redis
func initRedis() error {
	return redis.InitRedis()
}

// handleGracefulShutdown 处理优雅关闭
// 学习要点：信号处理，资源清理，优雅关闭模式
func handleGracefulShutdown() {
	// 创建信号通道
	quit := make(chan os.Signal, 1)
	
	// 监听系统信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	
	// 等待信号
	<-quit
	fmt.Println("\n🛑 收到关闭信号，开始优雅关闭...")
	
	// 设置关闭超时
	timeout := 30 * time.Second
	fmt.Printf("⏰ 等待现有连接处理完毕（最多等待 %v）...\n", timeout)
	
	// 关闭数据库连接
	if err := database.Close(); err != nil {
		fmt.Printf("❌ 关闭数据库连接失败: %v\n", err)
	} else {
		fmt.Println("✅ 数据库连接已关闭")
	}
	
	// 关闭Redis连接
	if err := redis.Close(); err != nil {
		fmt.Printf("❌ 关闭Redis连接失败: %v\n", err)
	} else {
		fmt.Println("✅ Redis连接已关闭")
	}
	
	fmt.Println("👋 服务器已优雅关闭")
	os.Exit(0)
}

// printBanner 打印启动横幅
func printBanner() {
	banner := `
	╔══════════════════════════════════════════════╗
	║          任务管理系统 v1.0                    ║
	║      Task Management System                  ║
	║                                             ║
	║  技术栈:                                     ║
	║  🔧 Golang + Gin + GORM + Redis + MySQL    ║
	║  📦 教学项目 - 后端开发最佳实践              ║
	║                                             ║
	║  学习要点:                                   ║
	║  • RESTful API 设计                        ║
	║  • 数据库建模与关系设计                      ║
	║  • 缓存策略与Redis集成                       ║
	║  • 中间件与错误处理                          ║
	║  • 项目结构与代码组织                        ║
	╚══════════════════════════════════════════════╝
	`
	fmt.Println(banner)
}

// 编译信息（可以在编译时通过 ldflags 注入）
var (
	Version   = "1.0.0"
	BuildTime = "unknown"
	GitCommit = "unknown"
)

// GetBuildInfo 获取构建信息
func GetBuildInfo() map[string]string {
	return map[string]string{
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
	}
}