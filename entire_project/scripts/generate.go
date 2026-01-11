// Package main GORM代码生成器
// 学习要点：GORM Gen的使用，自动代码生成，查询构建器
package main

import (
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gen"
	"gorm.io/gorm"
	"task-management-system/internal/config"
	"task-management-system/internal/models"
)

func main() {
	// 加载配置
	if err := config.Load("../configs/config.yaml"); err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	cfg := &config.GlobalConfig.Database.MySQL
	dsn := cfg.GetMySQLDSN()
	
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 创建GORM Gen实例
	// 学习要点：Gen生成器的配置，输出目录设置
	g := gen.NewGenerator(gen.Config{
		OutPath:      "../internal/query",    // 生成代码的输出目录
		OutFile:      "gen.go",               // 生成的主文件名
		ModelPkgPath: "../internal/models",   // 模型包路径
		
		// 生成模式配置
		Mode: gen.WithoutContext |           // 不生成带context的方法
			gen.WithDefaultQuery |            // 生成默认查询实例
			gen.WithQueryInterface,           // 生成查询接口
			
		// 字段可为空时生成指针类型
		FieldNullable:     true,
		FieldCoverable:    false,
		FieldSignable:     false,
		FieldWithIndexTag: false,
		FieldWithTypeTag:  true,
	})

	// 设置数据库连接
	g.UseDB(db)

	// 生成基础模型的查询代码
	// 学习要点：不同模型的查询方法生成
	
	// 1. 生成User模型的查询方法
	// 学习要点：自定义查询方法生成，复杂查询构建
	user := g.GenerateModel("users")
	
	// 为User添加自定义方法
	g.ApplyBasic(
		// 生成所有字段
		g.GenerateAllTable()...,
	)
	
	// 自定义查询方法
	// 学习要点：方法级别的查询生成，WHERE条件构建
	userQuery := g.GenerateModel("users", gen.FieldRelate(field.HasMany, "Tasks", models.Task{}, &field.RelateConfig{
		RelatePointer: true,
		GORMTag: map[string][]string{
			"foreignKey": {"UserID"},
		},
	}))
	
	// 为用户模型生成特定的查询方法
	g.ApplyInterface(func(models.User) {}, userQuery)

	// 2. 生成Task模型的查询方法
	taskQuery := g.GenerateModel("tasks", 
		gen.FieldRelate(field.BelongsTo, "User", models.User{}, &field.RelateConfig{
			RelatePointer: true,
			GORMTag: map[string][]string{
				"foreignKey": {"UserID"},
			},
		}),
		gen.FieldRelate(field.Many2Many, "Tags", models.Tag{}, &field.RelateConfig{
			RelatePointer: true,
			GORMTag: map[string][]string{
				"many2many": {"task_tags"},
			},
		}),
	)
	
	g.ApplyInterface(func(models.Task) {}, taskQuery)

	// 3. 生成Tag模型的查询方法
	tagQuery := g.GenerateModel("tags", 
		gen.FieldRelate(field.Many2Many, "Tasks", models.Task{}, &field.RelateConfig{
			RelatePointer: true,
			GORMTag: map[string][]string{
				"many2many": {"task_tags"},
			},
		}),
	)
	
	g.ApplyInterface(func(models.Tag) {}, tagQuery)

	// 执行代码生成
	// 学习要点：代码生成的执行过程
	g.Execute()
	
	log.Println("✅ GORM查询代码生成完成！")
	log.Println("📁 生成的文件位于: internal/query/")
	log.Println("🔍 使用方式：")
	log.Println("   import \"task-management-system/internal/query\"")
	log.Println("   q := query.Use(db)")
	log.Println("   users := q.User.Where(q.User.Status.Eq(1)).Find()")
}