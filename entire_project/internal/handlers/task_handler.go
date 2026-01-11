package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"task-management-system/internal/models"
	"task-management-system/internal/services"
)

// TaskHandler 任务处理器结构体
// 学习要点：复杂业务逻辑的HTTP处理，多条件查询，权限验证
type TaskHandler struct {
	taskService *services.TaskService
}

// NewTaskHandler 创建任务处理器实例
func NewTaskHandler() *TaskHandler {
	return &TaskHandler{
		taskService: services.NewTaskService(),
	}
}

// CreateTask 创建任务
// @Summary 创建任务
// @Description 为指定用户创建新任务
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param task body models.TaskCreateRequest true "任务信息"
// @Success 200 {object} models.Response{data=models.Task} "创建成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/tasks [post]
func (h *TaskHandler) CreateTask(c *gin.Context) {
	// 绑定请求参数
	var req models.TaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("请求参数错误: "+err.Error()))
		return
	}
	
	// 获取用户ID（实际项目中应该从JWT token中获取）
	// 这里为了演示，从header中获取
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse("请先登录"))
		return
	}
	
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户ID格式错误"))
		return
	}
	
	// 调用服务层创建任务
	task, err := h.taskService.CreateTask(uint(userID), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(task))
}

// GetTask 获取任务详情
// @Summary 获取任务详情
// @Description 根据任务ID获取任务详细信息
// @Tags 任务管理
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} models.Response{data=models.Task} "获取成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 404 {object} models.Response "任务不存在"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/tasks/{id} [get]
func (h *TaskHandler) GetTask(c *gin.Context) {
	// 获取路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("任务ID格式错误"))
		return
	}
	
	// 调用服务层获取任务
	task, err := h.taskService.GetTaskByID(uint(id))
	if err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(err.Error()))
		} else {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		}
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(task))
}

// UpdateTask 更新任务
// @Summary 更新任务
// @Description 更新任务信息
// @Tags 任务管理
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Param task body models.TaskUpdateRequest true "更新的任务信息"
// @Success 200 {object} models.Response{data=models.Task} "更新成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 403 {object} models.Response "权限不足"
// @Failure 404 {object} models.Response "任务不存在"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/tasks/{id} [put]
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	// 获取路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("任务ID格式错误"))
		return
	}
	
	// 获取用户ID
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse("请先登录"))
		return
	}
	
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户ID格式错误"))
		return
	}
	
	// 绑定请求参数
	var req models.TaskUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("请求参数错误: "+err.Error()))
		return
	}
	
	// 调用服务层更新任务
	task, err := h.taskService.UpdateTask(uint(id), uint(userID), &req)
	if err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(err.Error()))
		} else if err.Error() == "没有权限修改此任务" {
			c.JSON(http.StatusForbidden, models.NewErrorResponse(err.Error()))
		} else {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		}
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(task))
}

// DeleteTask 删除任务
// @Summary 删除任务
// @Description 删除指定的任务
// @Tags 任务管理
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} models.Response "删除成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 403 {object} models.Response "权限不足"
// @Failure 404 {object} models.Response "任务不存在"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/tasks/{id} [delete]
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	// 获取路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("任务ID格式错误"))
		return
	}
	
	// 获取用户ID
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse("请先登录"))
		return
	}
	
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户ID格式错误"))
		return
	}
	
	// 调用服务层删除任务
	if err := h.taskService.DeleteTask(uint(id), uint(userID)); err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(err.Error()))
		} else if err.Error() == "没有权限删除此任务" {
			c.JSON(http.StatusForbidden, models.NewErrorResponse(err.Error()))
		} else {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		}
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse("任务删除成功"))
}

// QueryTasks 查询任务列表
// @Summary 查询任务列表
// @Description 根据条件查询任务列表（支持分页和多种过滤条件）
// @Tags 任务管理
// @Produce json
// @Param status query int false "任务状态 (0-待处理,1-进行中,2-已完成,3-已取消)"
// @Param priority query int false "优先级 (1-低,2-中,3-高,4-紧急)"
// @Param tag_id query int false "标签ID"
// @Param user_id query int false "用户ID"
// @Param keyword query string false "搜索关键词"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} models.Response{data=models.PageResult} "查询成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/tasks [get]
func (h *TaskHandler) QueryTasks(c *gin.Context) {
	// 构建查询请求
	var req models.TaskQueryRequest
	
	// 绑定查询参数
	// 学习要点：复杂查询参数的处理，类型转换，默认值设置
	if c.ShouldBindQuery(&req) != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("查询参数格式错误"))
		return
	}
	
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}
	
	// 调用服务层查询任务
	result, err := h.taskService.QueryTasks(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(result))
}

// GetUserTasks 获取用户的任务列表
// @Summary 获取用户的任务列表
// @Description 获取指定用户的所有任务
// @Tags 任务管理
// @Produce json
// @Param user_id path int true "用户ID"
// @Param status query int false "任务状态过滤"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} models.Response{data=models.PageResult} "获取成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/users/{user_id}/tasks [get]
func (h *TaskHandler) GetUserTasks(c *gin.Context) {
	// 获取用户ID
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户ID格式错误"))
		return
	}
	
	// 构建查询请求
	var req models.TaskQueryRequest
	req.UserID = new(uint)
	*req.UserID = uint(userID)
	
	// 绑定其他查询参数
	c.ShouldBindQuery(&req)
	
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}
	
	// 调用服务层查询任务
	result, err := h.taskService.QueryTasks(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(result))
}

// GetUserTaskStats 获取用户任务统计
// @Summary 获取用户任务统计
// @Description 获取用户各状态任务的统计信息
// @Tags 任务管理
// @Produce json
// @Param user_id path int true "用户ID"
// @Success 200 {object} models.Response{data=map[string]int64} "获取成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/users/{user_id}/tasks/stats [get]
func (h *TaskHandler) GetUserTaskStats(c *gin.Context) {
	// 获取用户ID
	userIDStr := c.Param("user_id")
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户ID格式错误"))
		return
	}
	
	// 调用服务层获取统计信息
	stats, err := h.taskService.GetUserTaskStats(uint(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(stats))
}

// GetTasksByTag 根据标签获取任务列表
// @Summary 根据标签获取任务列表
// @Description 获取包含指定标签的所有任务
// @Tags 任务管理
// @Produce json
// @Param tag_id path int true "标签ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} models.Response{data=models.PageResult} "获取成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/tags/{tag_id}/tasks [get]
func (h *TaskHandler) GetTasksByTag(c *gin.Context) {
	// 获取标签ID
	tagIDStr := c.Param("tag_id")
	tagID, err := strconv.ParseUint(tagIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("标签ID格式错误"))
		return
	}
	
	// 构建查询请求
	var req models.TaskQueryRequest
	req.TagID = new(uint)
	*req.TagID = uint(tagID)
	
	// 绑定分页参数
	c.ShouldBindQuery(&req)
	
	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}
	
	// 调用服务层查询任务
	result, err := h.taskService.QueryTasks(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(result))
}

// MarkTaskComplete 标记任务为完成
// @Summary 标记任务为完成
// @Description 将任务状态设置为已完成
// @Tags 任务管理
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} models.Response{data=models.Task} "操作成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 403 {object} models.Response "权限不足"
// @Failure 404 {object} models.Response "任务不存在"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/tasks/{id}/complete [post]
func (h *TaskHandler) MarkTaskComplete(c *gin.Context) {
	// 获取任务ID
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("任务ID格式错误"))
		return
	}
	
	// 获取用户ID
	userIDStr := c.GetHeader("X-User-ID")
	if userIDStr == "" {
		c.JSON(http.StatusUnauthorized, models.NewErrorResponse("请先登录"))
		return
	}
	
	userID, err := strconv.ParseUint(userIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户ID格式错误"))
		return
	}
	
	// 构建更新请求
	status := models.TaskStatusCompleted
	req := models.TaskUpdateRequest{
		Status: &status,
	}
	
	// 调用服务层更新任务
	task, err := h.taskService.UpdateTask(uint(id), uint(userID), &req)
	if err != nil {
		if err.Error() == "任务不存在" {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(err.Error()))
		} else if err.Error() == "没有权限修改此任务" {
			c.JSON(http.StatusForbidden, models.NewErrorResponse(err.Error()))
		} else {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		}
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(task))
}