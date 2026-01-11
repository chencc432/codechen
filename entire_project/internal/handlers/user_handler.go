// Package handlers HTTP请求处理器
// 学习要点：HTTP处理器设计，请求参数验证，响应格式标准化
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"task-management-system/internal/models"
	"task-management-system/internal/services"
)

// UserHandler 用户处理器结构体
// 学习要点：控制器模式，依赖注入，职责分离
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler 创建用户处理器实例
func NewUserHandler() *UserHandler {
	return &UserHandler{
		userService: services.NewUserService(),
	}
}

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户账户
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param user body models.UserCreateRequest true "用户信息"
// @Success 200 {object} models.Response{data=models.UserResponse} "创建成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req models.UserCreateRequest
	
	// 绑定并验证请求参数
	// 学习要点：Gin的参数绑定和验证功能
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("请求参数错误: "+err.Error()))
		return
	}
	
	// 调用服务层创建用户
	user, err := h.userService.CreateUser(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}
	
	// 返回成功响应（不包含密码等敏感信息）
	c.JSON(http.StatusOK, models.NewSuccessResponse(user.ToResponse()))
}

// GetUser 获取用户信息
// @Summary 获取用户信息
// @Description 根据用户ID获取用户详细信息
// @Tags 用户管理
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} models.Response{data=models.UserResponse} "获取成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 404 {object} models.Response "用户不存在"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUser(c *gin.Context) {
	// 获取路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户ID格式错误"))
		return
	}
	
	// 调用服务层获取用户
	user, err := h.userService.GetUserByID(uint(id))
	if err != nil {
		// 根据错误类型返回不同的状态码
		// 学习要点：错误处理的最佳实践
		if err.Error() == "用户不存在" {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(err.Error()))
		} else {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		}
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(user.ToResponse()))
}

// UpdateUser 更新用户信息
// @Summary 更新用户信息
// @Description 更新用户的基本信息
// @Tags 用户管理
// @Accept json
// @Produce json
// @Param id path int true "用户ID"
// @Param user body models.UserUpdateRequest true "更新的用户信息"
// @Success 200 {object} models.Response{data=models.UserResponse} "更新成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 404 {object} models.Response "用户不存在"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/users/{id} [put]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	// 获取路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户ID格式错误"))
		return
	}
	
	// 绑定请求参数
	var req models.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("请求参数错误: "+err.Error()))
		return
	}
	
	// 调用服务层更新用户
	user, err := h.userService.UpdateUser(uint(id), &req)
	if err != nil {
		if err.Error() == "用户不存在" {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(err.Error()))
		} else {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		}
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(user.ToResponse()))
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 软删除用户账户
// @Tags 用户管理
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} models.Response "删除成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 404 {object} models.Response "用户不存在"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	// 获取路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户ID格式错误"))
		return
	}
	
	// 调用服务层删除用户
	if err := h.userService.DeleteUser(uint(id)); err != nil {
		if err.Error() == "用户不存在" {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(err.Error()))
		} else {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		}
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse("用户删除成功"))
}

// GetUserList 获取用户列表
// @Summary 获取用户列表
// @Description 分页获取用户列表
// @Tags 用户管理
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} models.Response{data=models.PageResult} "获取成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/users [get]
func (h *UserHandler) GetUserList(c *gin.Context) {
	// 获取查询参数
	// 学习要点：查询参数的获取和默认值处理
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	
	// 参数验证
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 { // 限制最大页面大小
		pageSize = 10
	}
	
	// 调用服务层获取用户列表
	result, err := h.userService.GetUserList(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(result))
}

// GetUserByUsername 根据用户名获取用户
// @Summary 根据用户名获取用户
// @Description 根据用户名查找用户信息
// @Tags 用户管理
// @Produce json
// @Param username path string true "用户名"
// @Success 200 {object} models.Response{data=models.UserResponse} "获取成功"
// @Failure 404 {object} models.Response "用户不存在"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/users/username/{username} [get]
func (h *UserHandler) GetUserByUsername(c *gin.Context) {
	username := c.Param("username")
	if username == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户名不能为空"))
		return
	}
	
	// 调用服务层获取用户
	user, err := h.userService.GetUserByUsername(username)
	if err != nil {
		if err.Error() == "用户不存在" {
			c.JSON(http.StatusNotFound, models.NewErrorResponse(err.Error()))
		} else {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		}
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse(user.ToResponse()))
}

// UpdateLastLoginTime 更新最后登录时间
// @Summary 更新最后登录时间
// @Description 记录用户的最后登录时间
// @Tags 用户管理
// @Produce json
// @Param id path int true "用户ID"
// @Success 200 {object} models.Response "更新成功"
// @Failure 400 {object} models.Response "请求参数错误"
// @Failure 500 {object} models.Response "内部服务器错误"
// @Router /api/v1/users/{id}/login [post]
func (h *UserHandler) UpdateLastLoginTime(c *gin.Context) {
	// 获取路径参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("用户ID格式错误"))
		return
	}
	
	// 调用服务层更新登录时间
	if err := h.userService.UpdateLastLoginTime(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}
	
	c.JSON(http.StatusOK, models.NewSuccessResponse("登录时间更新成功"))
}