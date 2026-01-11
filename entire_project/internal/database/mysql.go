// Package database 数据库连接管理
// 学习要点：GORM 数据库连接，连接池配置，自动迁移
package database

import (
	"fmt"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"task-management-system/internal/config"
	"task-management-system/internal/models"
)

// DB 全局数据库连接实例
var DB *gorm.DB

// InitMySQL 初始化MySQL数据库连接
// 学习要点：数据库连接初始化，连接池配置，错误处理
func InitMySQL() error {
	cfg := &config.GlobalConfig.Database.MySQL
	
	// 构建DSN（数据源名称）
	dsn := cfg.GetMySQLDSN()
	
	// 配置GORM
	gormConfig := &gorm.Config{
		// 配置日志级别
		Logger: logger.Default.LogMode(logger.Info),
		// 禁用外键约束（在代码中维护关系）
		DisableForeignKeyConstraintWhenMigrating: true,
	}
	
	// 连接数据库
	var err error
	DB, err = gorm.Open(mysql.Open(dsn), gormConfig)
	if err != nil {
		return fmt.Errorf("连接MySQL数据库失败: %w", err)
	}
	
	// 获取底层的sql.DB对象来配置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}
	
	// 配置连接池
	// 学习要点：数据库连接池的配置对性能的影响
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)                        // 设置最大空闲连接数
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)                        // 设置最大打开连接数
	sqlDB.SetConnMaxLifetime(cfg.GetConnMaxLifetime())             // 设置连接最大生命周期
	
	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}
	
	fmt.Println("✅ MySQL数据库连接成功")
	return nil
}

// AutoMigrate 自动迁移数据库表结构
// 学习要点：GORM 的自动迁移功能，数据库版本管理
func AutoMigrate() error {
	// 需要迁移的模型列表
	models := []interface{}{
		&models.User{},  // 用户表
		&models.Task{},  // 任务表
		&models.Tag{},   // 标签表
	}
	
	// 执行自动迁移
	if err := DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	
	fmt.Println("✅ 数据库表结构迁移完成")
	return nil
}

// SeedData 初始化种子数据
// 学习要点：数据库种子数据的创建，测试数据准备
func SeedData() error {
	// 检查是否已存在数据
	var userCount int64
	DB.Model(&models.User{}).Count(&userCount)
	if userCount > 0 {
		fmt.Println("⚠️  数据库已存在数据，跳过种子数据初始化")
		return nil
	}
	
	// 开始事务
	tx := DB.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()
	
	// 创建示例用户
	users := []models.User{
		{
			Username: "admin",
			Email:    "admin@example.com",
			Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // 密码: secret
			Nickname: "管理员",
			Status:   1,
		},
		{
			Username: "testuser",
			Email:    "test@example.com",
			Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy", // 密码: secret
			Nickname: "测试用户",
			Status:   1,
		},
	}
	
	for _, user := range users {
		if err := tx.Create(&user).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("创建用户失败: %w", err)
		}
	}
	
	// 创建示例标签
	tags := []models.Tag{
		{Name: "工作", Color: "#FF5722"},
		{Name: "学习", Color: "#2196F3"},
		{Name: "生活", Color: "#4CAF50"},
		{Name: "紧急", Color: "#F44336"},
	}
	
	for _, tag := range tags {
		if err := tx.Create(&tag).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("创建标签失败: %w", err)
		}
	}
	
	// 创建示例任务
	now := time.Now()
	tomorrow := now.Add(24 * time.Hour)
	tasks := []models.Task{
		{
			Title:       "完成项目文档",
			Description: "编写项目的README和API文档",
			Status:      models.TaskStatusInProgress,
			Priority:    models.TaskPriorityHigh,
			DueDate:     &tomorrow,
			UserID:      users[0].ID,
		},
		{
			Title:       "代码重构",
			Description: "重构用户管理模块的代码",
			Status:      models.TaskStatusPending,
			Priority:    models.TaskPriorityMedium,
			UserID:      users[0].ID,
		},
		{
			Title:       "学习Golang",
			Description: "深入学习Golang的并发编程",
			Status:      models.TaskStatusInProgress,
			Priority:    models.TaskPriorityLow,
			UserID:      users[1].ID,
		},
	}
	
	for _, task := range tasks {
		if err := tx.Create(&task).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("创建任务失败: %w", err)
		}
	}
	
	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}
	
	fmt.Println("✅ 种子数据初始化完成")
	return nil
}

// Close 关闭数据库连接
func Close() error {
	if DB == nil {
		return nil
	}
	
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	
	return sqlDB.Close()
}