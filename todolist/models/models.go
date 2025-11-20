package models

import (
	"time"
)

type Todo struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Completed   bool      `json:"completed"`
	CreateAt    time.Time `json:"create_at"`
	UpdateAt    time.Time `json:"update_at"`
}

var todos = []Todo{
	{
		ID:          1,
		Title:       "Todo 1",
		Description: "Todo 1 Description",
		Completed:   false,
		CreateAt:    time.Now(),
		UpdateAt:    time.Now(),
	},
	{
		ID:          2,
		Title:       "Todo 2",
		Description: "Todo 2 Description",
		Completed:   true,
		CreateAt:    time.Now(),
		UpdateAt:    time.Now(),
	},
}

func GetAllTodos() []Todo {
	return todos
}

func GetTodoByID(id uint) (Todo, bool) {
	for _, todo := range todos {
		if todo.ID == id {
			return todo, true
		}
	}
	return Todo{}, false //返回一个空的Todo和false
}

func GreateTodo(todo Todo) Todo {
	todo.ID = uint(len(todos) + 1)
	todo.CreateAt = time.Now()
	todo.UpdateAt = time.Now()
	todos = append(todos, todo)
	return todo
}

func UpdateTodo(id uint, updatedTodo Todo) (Todo, bool) {
	for i, todo := range todos {
		if todo.ID == id {
			updatedTodo.ID = id
			updatedTodo.CreateAt = todo.CreateAt
			todos[i] = updatedTodo
			return updatedTodo, true
		}
	}
	return Todo{}, false
}

func DeleteTodo(id uint) bool {
	for i, todo := range todos {
		if todo.ID == id {
			todos = append(todos[:i], todos[i+1:]...) //删除该元素
			return true
		}
	}
	return false
}
