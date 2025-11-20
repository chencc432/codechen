package controllers

import (
	"net/http"
	"strconv"

	"todolist/models"

	"github.com/gin-gonic/gin"
)

func GetTodos(c *gin.Context) {
	todos := models.GetAllTodos()
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   todos,
	})
}

func GetTodo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid id",
		})
		return
	}
	todo, ok := models.GetTodoByID(uint(id))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "todo not found",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   todo,
	})

}

func CreateTodo(c *gin.Context) {
	var todo models.Todo
	if err := c.ShouldBindJSON(&todo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}
	newTodo := models.GreateTodo(todo)
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   newTodo,
	})
}

func UpdateTodo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid id",
		})
		return
	}
	var todo models.Todo
	if err := c.ShouldBindJSON(&todo); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": err.Error(),
		})
		return
	}
	updatedTodo, found := models.UpdateTodo(uint(id), todo)
	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "todo not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   updatedTodo,
	})
}

func DeleteTodo(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "error",
			"message": "invalid id",
		})
		return
	}
	found := models.DeleteTodo(uint(id))
	if !found {
		c.JSON(http.StatusNotFound, gin.H{
			"status":  "error",
			"message": "todo not found",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "todo deleted",
	})

}
