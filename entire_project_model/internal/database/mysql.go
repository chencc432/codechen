package database
import(
	"fmt"
	"time"
	"entire_project_model/internal/models"
	"gorm.io/gorm"
	"gorm.io/driver/mysql"
	"gorm.io/gorm/logger"
	"entire_project_model/internal/config"
)


var DB *gorm.DB

func InitMySQL() error{
	cfg := &config.GlobalConfig.Database.MySQL
	dsn := cfg.GetMySQLDSN()
	fmt.Println("DSN:", dsn)
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	var err error
	DB,err = gorm.Open(mysql.Open(dsn), gormConfig)
	if err!=nil{
		return fmt.Errorf("连接MySQL数据库失败: %w", err)
	}
	sqlDB,err := DB.DB()
	if err!=nil{
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetConnMaxLifetime(cfg.GetConnMaxLifetime())
	if err:=sqlDB.Ping();err!=nil{
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}
	fmt.Println("✅ MySQL数据库连接成功")
	return nil
}

func AutoMigrate()error{
	models := []interface{}{
		&models.User{},
		&models.Task{},
		&models.Tag{},
	}
	if err := DB.AutoMigrate(models...); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	fmt.Println("✅ 数据库表结构迁移完成")
	return nil
}

func SeedData() error{
	var userCount int64
	DB.Model(&models.User{}).Count(&userCount)

	if userCount > 0{
		fmt.Println("⚠️ 数据库已存在数据，跳过种子数据初始化")
		return nil
	}

	tx := DB.Begin()
	defer func(){
		if r := recover(); r!=nil{
			tx.Rollback()
		}
	}()

	users := []models.User{
		{
			Username: "admin",
			Email : "admin@example.com",
			Password: "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			Nickname: "管理员",
			Status: 1,
		},
		{
			Username:"testuser",
			Email:"test@example.com",
			Password:"$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy",
			Nickname:"测试用户",
			Status:1,
		},
	}
	for _, user := range users{
		if err := tx.Create(&user).Error; err!=nil{
			tx.Rollback()
			return fmt.Errorf("创建用户失败: %w", err)
	}
}

	tags := []models.Tag{
		{Name: "工作", Color: "#FF5722"},
		{Name: "学习", Color: "#2196F3"},
		{Name: "生活", Color: "#4CAF50"},
		{Name: "紧急", Color: "#F44336"},
	}
	for _, rag :=range tags{
		if err := tx.Create(&rag).Error;err!=nil{
			tx.Rollback()
			return fmt.Errorf("创建标签失败: %w", err)
		}
	}
	now :=time.Now()
	tomorrow := now.Add(24 * time.Hour)
	tasks := []models.Task{
		{
			Title: "完成项目文档",
			Description: "编写项目的README和API文档",
			Status: models.TaskStatusInProgress,
			Priority: models.TaskPriorityHigh,
			DueDate: &tomorrow,
			UserID: users[0].ID,
		},
		{
			Title: "代码重构",
			Description: "重构用户管理模块的代码",
			Status: models.TaskStatusPending,
			Priority: models.TaskPriorityMedium,
			UserID: users[0].ID,
		},
		{
			Title: "学习Golang",
			Description: "深入学习Golang的并发编程",
			Status: models.TaskStatusInProgress,
			Priority: models.TaskPriorityLow,
			UserID: users[1].ID,
		},
	}
	for _, task := range tasks{
		if err := tx.Create(&task).Error;err!=nil{
			tx.Rollback()
			return fmt.Errorf("创建任务失败: %w", err)
		}
	}

	if err := tx.Commit().Error;err!=nil{
		return fmt.Errorf("提交事务失败: %w", err)
	}
	fmt.Println("✅ 种子数据初始化完成")
	return nil


}


func Close()error{
	if DB == nil{
       return nil
	}

	sqlDB, err := DB.DB()
	if err!=nil{
		return err
	}
	return sqlDB.Close()
}