// Package examples 查询示例代码
// 学习要点：GORM Gen生成的查询代码使用示例，复杂查询构建
package examples

import (
	"fmt"
	"time"

	"gorm.io/gorm"
	"task-management-system/internal/models"
	// "task-management-system/internal/query" // 注释：这是生成的查询代码包
)

// QueryExamples 查询示例结构体
type QueryExamples struct {
	db *gorm.DB
	// q  *query.Query // 注释：生成的查询实例
}

// NewQueryExamples 创建查询示例实例
func NewQueryExamples(db *gorm.DB) *QueryExamples {
	return &QueryExamples{
		db: db,
		// q:  query.Use(db), // 注释：使用生成的查询实例
	}
}

// 以下是GORM Gen生成查询代码的使用示例
// 注释掉是因为需要先运行生成器才能使用

/*
// UserQueryExamples 用户查询示例
func (e *QueryExamples) UserQueryExamples() {
	// 学习要点：基础查询操作
	
	// 1. 简单查询 - 根据ID查找用户
	user, err := e.q.User.Where(e.q.User.ID.Eq(1)).First()
	if err != nil {
		fmt.Printf("查询用户失败: %v\n", err)
		return
	}
	fmt.Printf("用户信息: %+v\n", user)
	
	// 2. 条件查询 - 查找活跃用户
	activeUsers, err := e.q.User.Where(e.q.User.Status.Eq(1)).Find()
	if err != nil {
		fmt.Printf("查询活跃用户失败: %v\n", err)
		return
	}
	fmt.Printf("活跃用户数量: %d\n", len(activeUsers))
	
	// 3. 模糊查询 - 根据用户名模糊搜索
	searchUsers, err := e.q.User.Where(e.q.User.Username.Like("%admin%")).Find()
	if err != nil {
		fmt.Printf("模糊查询用户失败: %v\n", err)
		return
	}
	fmt.Printf("搜索结果: %d 个用户\n", len(searchUsers))
	
	// 4. 多条件查询 - AND条件
	specificUsers, err := e.q.User.Where(
		e.q.User.Status.Eq(1),
		e.q.User.Username.Neq(""),
	).Find()
	if err != nil {
		fmt.Printf("多条件查询失败: %v\n", err)
		return
	}
	fmt.Printf("特定条件用户数量: %d\n", len(specificUsers))
	
	// 5. OR条件查询
	orUsers, err := e.q.User.Where(
		e.q.User.Or(
			e.q.User.Username.Eq("admin"),
			e.q.User.Email.Like("%admin%"),
		),
	).Find()
	if err != nil {
		fmt.Printf("OR查询失败: %v\n", err)
		return
	}
	fmt.Printf("OR查询结果: %d 个用户\n", len(orUsers))
}

// TaskQueryExamples 任务查询示例
func (e *QueryExamples) TaskQueryExamples() {
	// 学习要点：关联查询，复杂条件构建
	
	// 1. 基础任务查询
	tasks, err := e.q.Task.Where(e.q.Task.Status.Eq(models.TaskStatusPending)).Find()
	if err != nil {
		fmt.Printf("查询待处理任务失败: %v\n", err)
		return
	}
	fmt.Printf("待处理任务数量: %d\n", len(tasks))
	
	// 2. 预加载关联数据
	tasksWithUser, err := e.q.Task.Preload(e.q.Task.User).Preload(e.q.Task.Tags).Find()
	if err != nil {
		fmt.Printf("预加载任务关联数据失败: %v\n", err)
		return
	}
	fmt.Printf("带关联数据的任务数量: %d\n", len(tasksWithUser))
	
	// 3. 时间范围查询 - 查找本周创建的任务
	weekStart := time.Now().AddDate(0, 0, -7)
	recentTasks, err := e.q.Task.Where(e.q.Task.CreatedAt.Gte(weekStart)).Find()
	if err != nil {
		fmt.Printf("查询最近任务失败: %v\n", err)
		return
	}
	fmt.Printf("本周新建任务数量: %d\n", len(recentTasks))
	
	// 4. 排序查询 - 按优先级和创建时间排序
	sortedTasks, err := e.q.Task.Order(
		e.q.Task.Priority.Desc(),
		e.q.Task.CreatedAt.Desc(),
	).Limit(10).Find()
	if err != nil {
		fmt.Printf("排序查询失败: %v\n", err)
		return
	}
	fmt.Printf("排序后的前10个任务数量: %d\n", len(sortedTasks))
	
	// 5. 聚合查询 - 统计各状态任务数量
	var statusStats []struct {
		Status int
		Count  int64
	}
	
	err = e.q.Task.Select(
		e.q.Task.Status,
		e.q.Task.ID.Count().As("count"),
	).Group(e.q.Task.Status).Scan(&statusStats)
	if err != nil {
		fmt.Printf("聚合查询失败: %v\n", err)
		return
	}
	
	fmt.Println("任务状态统计:")
	for _, stat := range statusStats {
		fmt.Printf("  状态 %d: %d 个任务\n", stat.Status, stat.Count)
	}
	
	// 6. 子查询 - 查找有任务的用户
	usersWithTasks, err := e.q.User.Where(
		e.q.User.ID.In(
			e.q.Task.Select(e.q.Task.UserID).Where(e.q.Task.Status.Neq(models.TaskStatusCancelled)),
		),
	).Find()
	if err != nil {
		fmt.Printf("子查询失败: %v\n", err)
		return
	}
	fmt.Printf("有任务的用户数量: %d\n", len(usersWithTasks))
}

// ComplexQueryExamples 复杂查询示例
func (e *QueryExamples) ComplexQueryExamples() {
	// 学习要点：复杂业务查询，多表联查，性能优化
	
	// 1. 分页查询 - 用户的任务列表
	userID := uint(1)
	page := 1
	pageSize := 10
	offset := (page - 1) * pageSize
	
	userTasks, total, err := e.q.Task.Where(e.q.Task.UserID.Eq(userID)).
		Preload(e.q.Task.Tags).
		Order(e.q.Task.CreatedAt.Desc()).
		FindByPage(offset, pageSize)
	if err != nil {
		fmt.Printf("分页查询失败: %v\n", err)
		return
	}
	
	fmt.Printf("用户 %d 的任务: 第 %d 页，共 %d 个任务\n", userID, page, total)
	for _, task := range userTasks {
		fmt.Printf("  - %s (优先级: %d)\n", task.Title, task.Priority)
	}
	
	// 2. 连表查询 - 查找高优先级任务及其创建者
	highPriorityTasks, err := e.q.Task.
		Select(e.q.Task.ALL, e.q.User.Username, e.q.User.Email).
		LeftJoin(e.q.User, e.q.Task.UserID.EqCol(e.q.User.ID)).
		Where(e.q.Task.Priority.Gte(models.TaskPriorityHigh)).
		Find()
	if err != nil {
		fmt.Printf("连表查询失败: %v\n", err)
		return
	}
	fmt.Printf("高优先级任务数量: %d\n", len(highPriorityTasks))
	
	// 3. 事务操作 - 批量更新任务状态
	err = e.q.Transaction(func(tx *query.Query) error {
		// 将所有过期的待处理任务标记为已取消
		dueDate := time.Now()
		_, err := tx.Task.
			Where(tx.Task.Status.Eq(models.TaskStatusPending)).
			Where(tx.Task.DueDate.Lt(dueDate)).
			Update(tx.Task.Status, models.TaskStatusCancelled)
		return err
	})
	if err != nil {
		fmt.Printf("事务操作失败: %v\n", err)
		return
	}
	fmt.Println("✅ 过期任务状态更新完成")
	
	// 4. 原生SQL查询 - 复杂统计查询
	type UserTaskStat struct {
		UserID       uint   `json:"user_id"`
		Username     string `json:"username"`
		TotalTasks   int64  `json:"total_tasks"`
		CompletedTasks int64  `json:"completed_tasks"`
		CompletionRate float64 `json:"completion_rate"`
	}
	
	var stats []UserTaskStat
	err = e.db.Raw(`
		SELECT 
			u.id as user_id,
			u.username,
			COUNT(t.id) as total_tasks,
			SUM(CASE WHEN t.status = ? THEN 1 ELSE 0 END) as completed_tasks,
			ROUND(
				SUM(CASE WHEN t.status = ? THEN 1 ELSE 0 END) * 100.0 / COUNT(t.id), 2
			) as completion_rate
		FROM users u
		LEFT JOIN tasks t ON u.id = t.user_id
		WHERE u.deleted_at IS NULL AND (t.deleted_at IS NULL OR t.id IS NULL)
		GROUP BY u.id, u.username
		HAVING COUNT(t.id) > 0
		ORDER BY completion_rate DESC
	`, models.TaskStatusCompleted, models.TaskStatusCompleted).Scan(&stats).Error
	
	if err != nil {
		fmt.Printf("原生SQL查询失败: %v\n", err)
		return
	}
	
	fmt.Println("用户任务完成率统计:")
	for _, stat := range stats {
		fmt.Printf("  %s: %d/%d 任务 (%.2f%%)\n", 
			stat.Username, stat.CompletedTasks, stat.TotalTasks, stat.CompletionRate)
	}
}

// BatchOperationExamples 批量操作示例
func (e *QueryExamples) BatchOperationExamples() {
	// 学习要点：批量操作，性能优化，事务处理
	
	// 1. 批量插入
	newTasks := []*models.Task{
		{Title: "批量任务1", Description: "演示批量插入", Priority: models.TaskPriorityMedium, UserID: 1},
		{Title: "批量任务2", Description: "演示批量插入", Priority: models.TaskPriorityMedium, UserID: 1},
		{Title: "批量任务3", Description: "演示批量插入", Priority: models.TaskPriorityMedium, UserID: 1},
	}
	
	err := e.q.Task.CreateInBatches(newTasks, 100)
	if err != nil {
		fmt.Printf("批量插入失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 批量插入 %d 个任务完成\n", len(newTasks))
	
	// 2. 批量更新
	affected, err := e.q.Task.
		Where(e.q.Task.Title.Like("批量任务%")).
		Update(e.q.Task.Description, "批量操作演示任务")
	if err != nil {
		fmt.Printf("批量更新失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 批量更新 %d 个任务完成\n", affected.RowsAffected)
	
	// 3. 批量删除（软删除）
	affected, err = e.q.Task.
		Where(e.q.Task.Title.Like("批量任务%")).
		Delete()
	if err != nil {
		fmt.Printf("批量删除失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 批量删除 %d 个任务完成\n", affected.RowsAffected)
}

// RunAllExamples 运行所有示例
func (e *QueryExamples) RunAllExamples() {
	fmt.Println("=== GORM Gen 查询示例演示 ===")
	
	fmt.Println("\n1. 用户查询示例:")
	e.UserQueryExamples()
	
	fmt.Println("\n2. 任务查询示例:")
	e.TaskQueryExamples()
	
	fmt.Println("\n3. 复杂查询示例:")
	e.ComplexQueryExamples()
	
	fmt.Println("\n4. 批量操作示例:")
	e.BatchOperationExamples()
	
	fmt.Println("\n=== 演示完成 ===")
}
*/

// ManualQueryExamples 手动查询示例（不依赖生成的代码）
// 学习要点：传统GORM查询方式，与生成代码的对比
func (e *QueryExamples) ManualQueryExamples() {
	fmt.Println("=== 手动查询示例（传统GORM方式）===")
	
	// 1. 基础查询
	var users []models.User
	if err := e.db.Where("status = ?", 1).Find(&users).Error; err != nil {
		fmt.Printf("查询用户失败: %v\n", err)
		return
	}
	fmt.Printf("活跃用户数量: %d\n", len(users))
	
	// 2. 关联查询
	var tasks []models.Task
	if err := e.db.Preload("User").Preload("Tags").Find(&tasks).Error; err != nil {
		fmt.Printf("查询任务失败: %v\n", err)
		return
	}
	fmt.Printf("任务总数: %d\n", len(tasks))
	
	// 3. 统计查询
	var count int64
	if err := e.db.Model(&models.Task{}).Where("status = ?", models.TaskStatusCompleted).Count(&count).Error; err != nil {
		fmt.Printf("统计失败: %v\n", err)
		return
	}
	fmt.Printf("已完成任务数: %d\n", count)
	
	// 4. 复杂条件查询
	var highPriorityTasks []models.Task
	if err := e.db.Where("priority >= ? AND status != ?", models.TaskPriorityHigh, models.TaskStatusCancelled).
		Order("priority DESC, created_at DESC").
		Limit(10).
		Find(&highPriorityTasks).Error; err != nil {
		fmt.Printf("复杂查询失败: %v\n", err)
		return
	}
	fmt.Printf("高优先级任务数: %d\n", len(highPriorityTasks))
	
	fmt.Println("=== 手动查询示例完成 ===")
}

// ComparisonExamples GORM传统方式 vs Gen生成代码对比
func (e *QueryExamples) ComparisonExamples() {
	fmt.Println(`
=== GORM 传统方式 vs Gen 生成代码对比 ===

1. 传统GORM查询:
   db.Where("status = ? AND priority >= ?", 1, 3).Find(&tasks)

2. Gen生成查询:
   q.Task.Where(q.Task.Status.Eq(1), q.Task.Priority.Gte(3)).Find()

优势对比:
✅ Gen方式优势:
  - 类型安全：编译时检查字段名和类型
  - IDE支持：自动完成和重构支持  
  - 可读性强：语义化的方法名
  - 性能优化：预编译查询语句
  - 防SQL注入：自动参数绑定

❌ 传统方式不足:
  - 字符串拼接容易出错
  - 运行时才能发现错误
  - IDE支持有限
  - 维护成本高

🎯 使用建议:
  - 新项目推荐使用GORM Gen
  - 复杂查询可结合原生SQL
  - 简单查询优先使用生成代码
  - 性能敏感场景考虑原生SQL

📚 学习要点:
  - 理解两种方式的适用场景
  - 掌握生成代码的配置和使用
  - 学会在项目中合理选择查询方式
	`)
}