// Package examples 生成代码使用示例
// 学习要点：GORM Gen生成代码的实际使用，与传统方式对比
package examples

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"
	"task-management-system/internal/models"
	// "task-management-system/internal/query" // 注释：生成的查询代码包，需要先运行生成器
)

// GeneratedCodeExample 生成代码使用示例
type GeneratedCodeExample struct {
	db *gorm.DB
	// q  *query.Query // 注释：生成的查询实例
}

// NewGeneratedCodeExample 创建示例实例
func NewGeneratedCodeExample(db *gorm.DB) *GeneratedCodeExample {
	return &GeneratedCodeExample{
		db: db,
		// q:  query.Use(db), // 注释：使用生成的查询实例
	}
}

// 以下是使用生成代码的示例（注释掉是因为需要先运行生成器）

/*
// 🚀 基础查询示例
func (e *GeneratedCodeExample) BasicQueryExamples() {
	fmt.Println("=== 基础查询示例 ===")
	
	// 1. 简单查询 - 类型安全
	users, err := e.q.User.Where(e.q.User.Status.Eq(1)).Find()
	if err != nil {
		fmt.Printf("查询用户失败: %v\n", err)
		return
	}
	fmt.Printf("活跃用户数量: %d\n", len(users))
	
	// 对比传统方式:
	// var users []models.User
	// e.db.Where("status = ?", 1).Find(&users)  // 字符串拼接，容易出错
	
	// 2. 条件组合查询
	tasks, err := e.q.Task.Where(
		e.q.Task.Status.Eq(models.TaskStatusPending),    // 状态=待处理
		e.q.Task.Priority.Gte(models.TaskPriorityHigh),  // 优先级>=高
		e.q.Task.DueDate.IsNotNull(),                    // 有截止日期
	).Find()
	
	if err != nil {
		fmt.Printf("查询任务失败: %v\n", err)
		return
	}
	fmt.Printf("高优先级待处理任务: %d\n", len(tasks))
	
	// 3. 模糊查询
	searchUsers, err := e.q.User.Where(
		e.q.User.Or(
			e.q.User.Username.Like("%admin%"),
			e.q.User.Email.Like("%admin%"),
		),
	).Find()
	
	if err != nil {
		fmt.Printf("模糊查询失败: %v\n", err)
		return
	}
	fmt.Printf("包含'admin'的用户: %d\n", len(searchUsers))
}

// 🔍 复杂查询示例
func (e *GeneratedCodeExample) ComplexQueryExamples() {
	fmt.Println("=== 复杂查询示例 ===")
	
	// 1. 预加载关联数据 - 避免N+1问题
	tasksWithRelations, err := e.q.Task.
		Preload(e.q.Task.User).     // 预加载用户信息
		Preload(e.q.Task.Tags).     // 预加载标签信息
		Where(e.q.Task.Status.Neq(models.TaskStatusCancelled)).
		Order(e.q.Task.Priority.Desc(), e.q.Task.CreatedAt.Desc()).
		Limit(10).
		Find()
	
	if err != nil {
		fmt.Printf("复杂查询失败: %v\n", err)
		return
	}
	
	fmt.Printf("获取任务及关联数据: %d\n", len(tasksWithRelations))
	for _, task := range tasksWithRelations {
		fmt.Printf("  任务: %s, 用户: %s, 标签数: %d\n", 
			task.Title, task.User.Username, len(task.Tags))
	}
	
	// 2. 子查询 - 查找有任务的用户
	usersWithTasks, err := e.q.User.Where(
		e.q.User.ID.In(
			e.q.Task.Select(e.q.Task.UserID).
			Where(e.q.Task.Status.Neq(models.TaskStatusCancelled)),
		),
	).Find()
	
	if err != nil {
		fmt.Printf("子查询失败: %v\n", err)
		return
	}
	fmt.Printf("有任务的用户数: %d\n", len(usersWithTasks))
	
	// 3. 多表连接查询
	type UserTaskStat struct {
		UserID       uint   `json:"user_id"`
		Username     string `json:"username"`
		TaskCount    int64  `json:"task_count"`
		CompletedCount int64 `json:"completed_count"`
	}
	
	var stats []UserTaskStat
	err = e.q.User.
		Select(
			e.q.User.ID.As("user_id"),
			e.q.User.Username.As("username"),
			e.q.Task.ID.Count().As("task_count"),
		).
		LeftJoin(e.q.Task, e.q.User.ID.EqCol(e.q.Task.UserID)).
		Group(e.q.User.ID, e.q.User.Username).
		Scan(&stats)
	
	if err != nil {
		fmt.Printf("连接查询失败: %v\n", err)
		return
	}
	
	fmt.Println("用户任务统计:")
	for _, stat := range stats {
		fmt.Printf("  %s: %d 个任务\n", stat.Username, stat.TaskCount)
	}
}

// 📊 聚合查询示例
func (e *GeneratedCodeExample) AggregationExamples() {
	fmt.Println("=== 聚合查询示例 ===")
	
	// 1. 统计查询
	totalUsers, err := e.q.User.Count()
	if err != nil {
		fmt.Printf("统计用户失败: %v\n", err)
		return
	}
	fmt.Printf("总用户数: %d\n", totalUsers)
	
	// 2. 分组统计
	type StatusCount struct {
		Status int   `json:"status"`
		Count  int64 `json:"count"`
	}
	
	var statusStats []StatusCount
	err = e.q.Task.
		Select(e.q.Task.Status, e.q.Task.ID.Count().As("count")).
		Group(e.q.Task.Status).
		Scan(&statusStats)
	
	if err != nil {
		fmt.Printf("分组统计失败: %v\n", err)
		return
	}
	
	fmt.Println("任务状态统计:")
	for _, stat := range statusStats {
		statusText := getStatusText(stat.Status)
		fmt.Printf("  %s: %d 个\n", statusText, stat.Count)
	}
	
	// 3. 时间范围统计
	startDate := time.Now().AddDate(0, 0, -7) // 7天前
	recentTaskCount, err := e.q.Task.
		Where(e.q.Task.CreatedAt.Gte(startDate)).
		Count()
	
	if err != nil {
		fmt.Printf("时间范围统计失败: %v\n", err)
		return
	}
	fmt.Printf("最近7天创建的任务: %d\n", recentTaskCount)
	
	// 4. 高级聚合 - 平均值、最大值、最小值
	type TaskStats struct {
		TotalTasks   int64     `json:"total_tasks"`
		AvgPriority  float64   `json:"avg_priority"`
		MaxPriority  int       `json:"max_priority"`
		MinPriority  int       `json:"min_priority"`
		LatestCreate time.Time `json:"latest_create"`
	}
	
	var taskStats TaskStats
	err = e.q.Task.
		Select(
			e.q.Task.ID.Count().As("total_tasks"),
			e.q.Task.Priority.Avg().As("avg_priority"),
			e.q.Task.Priority.Max().As("max_priority"),
			e.q.Task.Priority.Min().As("min_priority"),
			e.q.Task.CreatedAt.Max().As("latest_create"),
		).
		Scan(&taskStats)
	
	if err != nil {
		fmt.Printf("高级聚合查询失败: %v\n", err)
		return
	}
	
	fmt.Printf("任务统计信息:\n")
	fmt.Printf("  总任务数: %d\n", taskStats.TotalTasks)
	fmt.Printf("  平均优先级: %.2f\n", taskStats.AvgPriority)
	fmt.Printf("  最高优先级: %d\n", taskStats.MaxPriority)
	fmt.Printf("  最低优先级: %d\n", taskStats.MinPriority)
	fmt.Printf("  最新创建时间: %s\n", taskStats.LatestCreate.Format("2006-01-02 15:04:05"))
}

// 📝 增删改操作示例
func (e *GeneratedCodeExample) CUDOperationExamples() {
	fmt.Println("=== 增删改操作示例 ===")
	
	// 1. 创建操作
	newUser := &models.User{
		Username: "generated_user",
		Email:    "generated@example.com",
		Password: "password123",
		Status:   1,
	}
	
	err := e.q.User.Create(newUser)
	if err != nil {
		fmt.Printf("创建用户失败: %v\n", err)
		return
	}
	fmt.Printf("创建用户成功, ID: %d\n", newUser.ID)
	
	// 2. 批量创建
	newTasks := []*models.Task{
		{Title: "生成的任务1", Priority: 2, UserID: newUser.ID},
		{Title: "生成的任务2", Priority: 3, UserID: newUser.ID},
		{Title: "生成的任务3", Priority: 1, UserID: newUser.ID},
	}
	
	err = e.q.Task.CreateInBatches(newTasks, 100)
	if err != nil {
		fmt.Printf("批量创建任务失败: %v\n", err)
		return
	}
	fmt.Printf("批量创建 %d 个任务成功\n", len(newTasks))
	
	// 3. 更新操作
	result, err := e.q.User.
		Where(e.q.User.ID.Eq(newUser.ID)).
		Update(e.q.User.Nickname, "更新的昵称")
	
	if err != nil {
		fmt.Printf("更新用户失败: %v\n", err)
		return
	}
	fmt.Printf("更新用户成功, 影响行数: %d\n", result.RowsAffected)
	
	// 4. 批量更新
	result, err = e.q.Task.
		Where(e.q.Task.UserID.Eq(newUser.ID)).
		Update(e.q.Task.Status, models.TaskStatusInProgress)
	
	if err != nil {
		fmt.Printf("批量更新任务失败: %v\n", err)
		return
	}
	fmt.Printf("批量更新任务状态成功, 影响行数: %d\n", result.RowsAffected)
	
	// 5. 条件删除
	result, err = e.q.Task.
		Where(e.q.Task.UserID.Eq(newUser.ID)).
		Delete()
	
	if err != nil {
		fmt.Printf("删除任务失败: %v\n", err)
		return
	}
	fmt.Printf("删除任务成功, 影响行数: %d\n", result.RowsAffected)
	
	// 6. 删除用户
	result, err = e.q.User.Where(e.q.User.ID.Eq(newUser.ID)).Delete()
	if err != nil {
		fmt.Printf("删除用户失败: %v\n", err)
		return
	}
	fmt.Printf("删除用户成功, 影响行数: %d\n", result.RowsAffected)
}

// 🔧 事务操作示例
func (e *GeneratedCodeExample) TransactionExamples() {
	fmt.Println("=== 事务操作示例 ===")
	
	// 事务中的复杂操作
	err := e.q.Transaction(func(tx *query.Query) error {
		// 1. 创建用户
		user := &models.User{
			Username: "tx_user",
			Email:    "tx@example.com",
			Status:   1,
		}
		
		if err := tx.User.Create(user); err != nil {
			return fmt.Errorf("创建用户失败: %w", err)
		}
		
		// 2. 为用户创建多个任务
		tasks := []*models.Task{
			{Title: "事务任务1", UserID: user.ID, Priority: 2},
			{Title: "事务任务2", UserID: user.ID, Priority: 3},
		}
		
		if err := tx.Task.CreateInBatches(tasks, 100); err != nil {
			return fmt.Errorf("创建任务失败: %w", err)
		}
		
		// 3. 更新用户的任务统计（模拟）
		_, err := tx.User.
			Where(tx.User.ID.Eq(user.ID)).
			Update(tx.User.UpdatedAt, time.Now())
			
		if err != nil {
			return fmt.Errorf("更新用户时间戳失败: %w", err)
		}
		
		fmt.Printf("事务操作成功: 创建用户 %d, 创建任务 %d 个\n", 
			user.ID, len(tasks))
		
		return nil
	})
	
	if err != nil {
		fmt.Printf("事务操作失败: %v\n", err)
		return
	}
	
	fmt.Println("事务操作全部成功！")
}

// 📈 性能对比示例
func (e *GeneratedCodeExample) PerformanceComparison() {
	fmt.Println("=== 性能对比示例 ===")
	
	// 1. 生成代码方式 - 预编译优化
	start := time.Now()
	for i := 0; i < 100; i++ {
		_, err := e.q.User.Where(e.q.User.Status.Eq(1)).Limit(10).Find()
		if err != nil {
			fmt.Printf("生成代码查询失败: %v\n", err)
			return
		}
	}
	generatedDuration := time.Since(start)
	
	// 2. 传统方式 - 字符串拼接
	start = time.Now()
	for i := 0; i < 100; i++ {
		var users []models.User
		err := e.db.Where("status = ?", 1).Limit(10).Find(&users).Error
		if err != nil {
			fmt.Printf("传统方式查询失败: %v\n", err)
			return
		}
	}
	traditionalDuration := time.Since(start)
	
	fmt.Printf("性能对比结果 (100次查询):\n")
	fmt.Printf("  生成代码方式: %v\n", generatedDuration)
	fmt.Printf("  传统方式: %v\n", traditionalDuration)
	fmt.Printf("  性能提升: %.2f%%\n", 
		float64(traditionalDuration-generatedDuration)/float64(traditionalDuration)*100)
}

// 🎯 最佳实践示例
func (e *GeneratedCodeExample) BestPracticeExamples() {
	fmt.Println("=== 最佳实践示例 ===")
	
	// 1. 分页查询的最佳实践
	page := 1
	pageSize := 10
	offset := (page - 1) * pageSize
	
	tasks, count, err := e.q.Task.
		Where(e.q.Task.Status.Neq(models.TaskStatusCancelled)).
		Preload(e.q.Task.User).           // 预加载关联数据
		Order(e.q.Task.CreatedAt.Desc()). // 排序
		FindByPage(offset, pageSize)       // 分页查询
	
	if err != nil {
		fmt.Printf("分页查询失败: %v\n", err)
		return
	}
	
	fmt.Printf("分页查询结果: 第%d页, 每页%d条, 总数%d, 实际%d条\n", 
		page, pageSize, count, len(tasks))
	
	// 2. 动态查询条件的最佳实践
	query := e.q.Task
	
	// 根据条件动态添加WHERE子句
	status := models.TaskStatusPending
	if status != 0 {
		query = query.Where(e.q.Task.Status.Eq(status))
	}
	
	keyword := "重要"
	if keyword != "" {
		query = query.Where(
			e.q.Task.Or(
				e.q.Task.Title.Like("%"+keyword+"%"),
				e.q.Task.Description.Like("%"+keyword+"%"),
			),
		)
	}
	
	results, err := query.Find()
	if err != nil {
		fmt.Printf("动态查询失败: %v\n", err)
		return
	}
	
	fmt.Printf("动态查询结果: %d 条记录\n", len(results))
	
	// 3. 错误处理的最佳实践
	user, err := e.q.User.Where(e.q.User.ID.Eq(999999)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fmt.Println("用户不存在，这是正常情况")
		} else {
			fmt.Printf("查询用户时发生错误: %v\n", err)
		}
		return
	}
	
	fmt.Printf("找到用户: %s\n", user.Username)
}

// RunAllGeneratedExamples 运行所有生成代码示例
func (e *GeneratedCodeExample) RunAllGeneratedExamples() {
	fmt.Println("🚀 开始运行生成代码示例...")
	
	e.BasicQueryExamples()
	e.ComplexQueryExamples()
	e.AggregationExamples()
	e.CUDOperationExamples()
	e.TransactionExamples()
	e.PerformanceComparison()
	e.BestPracticeExamples()
	
	fmt.Println("✅ 所有生成代码示例运行完成！")
}
*/

// ManualQueryComparison 手动查询对比（不依赖生成代码）
// 学习要点：传统GORM查询方式，用于对比学习
func (e *GeneratedCodeExample) ManualQueryComparison() {
	fmt.Println("=== 传统查询方式示例 ===")
	
	// 1. 基础查询
	var users []models.User
	if err := e.db.Where("status = ?", 1).Find(&users).Error; err != nil {
		fmt.Printf("查询活跃用户失败: %v\n", err)
		return
	}
	fmt.Printf("活跃用户数: %d\n", len(users))
	
	// 2. 复杂查询
	var tasks []models.Task
	if err := e.db.
		Preload("User").
		Preload("Tags").
		Where("status != ? AND priority >= ?", models.TaskStatusCancelled, models.TaskPriorityHigh).
		Order("priority DESC, created_at DESC").
		Limit(10).
		Find(&tasks).Error; err != nil {
		fmt.Printf("复杂查询失败: %v\n", err)
		return
	}
	fmt.Printf("高优先级任务数: %d\n", len(tasks))
	
	// 3. 聚合查询
	type StatusStat struct {
		Status int   `json:"status"`
		Count  int64 `json:"count"`
	}
	
	var stats []StatusStat
	if err := e.db.Model(&models.Task{}).
		Select("status, COUNT(*) as count").
		Group("status").
		Scan(&stats).Error; err != nil {
		fmt.Printf("聚合查询失败: %v\n", err)
		return
	}
	
	fmt.Println("任务状态统计:")
	for _, stat := range stats {
		fmt.Printf("  状态 %d: %d 个任务\n", stat.Status, stat.Count)
	}
	
	// 4. 子查询
	var activeUsers []models.User
	if err := e.db.Where("id IN (?)", 
		e.db.Model(&models.Task{}).
			Select("DISTINCT user_id").
			Where("status != ?", models.TaskStatusCancelled),
	).Find(&activeUsers).Error; err != nil {
		fmt.Printf("子查询失败: %v\n", err)
		return
	}
	fmt.Printf("有活跃任务的用户数: %d\n", len(activeUsers))
}

// 辅助函数
func getStatusText(status int) string {
	switch status {
	case models.TaskStatusPending:
		return "待处理"
	case models.TaskStatusInProgress:
		return "进行中"
	case models.TaskStatusCompleted:
		return "已完成"
	case models.TaskStatusCancelled:
		return "已取消"
	default:
		return "未知状态"
	}
}

// CodeGenerationGuide 代码生成指南
func CodeGenerationGuide() {
	guide := `
📚 GORM Gen 代码生成完整指南

🔧 1. 安装和配置
   go get -u gorm.io/gen

🚀 2. 运行生成器
   cd scripts
   go run generate.go

📁 3. 生成的文件结构
   internal/query/
   ├── gen.go           # 主查询文件
   ├── users.gen.go     # 用户查询代码  
   ├── tasks.gen.go     # 任务查询代码
   └── tags.gen.go      # 标签查询代码

💡 4. 使用生成的代码
   import "task-management-system/internal/query"
   
   q := query.Use(db)
   
   // 类型安全的查询
   users, err := q.User.Where(
       q.User.Status.Eq(1),
       q.User.Username.Like("%admin%"),
   ).Find()

✨ 5. 主要优势
   ✅ 类型安全 - 编译时检查
   ✅ IDE支持 - 自动完成和重构
   ✅ 性能优化 - 预编译查询
   ✅ 代码生成 - 减少手写代码

🎯 6. 最佳实践
   • 复杂查询使用生成代码
   • 简单CRUD可以混用
   • 特殊需求使用原生SQL
   • 定期重新生成以同步表结构

📖 7. 学习建议
   • 先掌握传统GORM用法
   • 理解生成代码的原理
   • 在实际项目中逐步应用
   • 根据团队情况选择使用范围
`
	fmt.Println(guide)
}