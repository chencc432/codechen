package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"todolist/controllers"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	r := gin.Default()
	r.Use(gin.Logger())
	api := r.Group("/api")
	{
		todos := api.Group("todos")
		{
			todos.GET("", controllers.GetTodos)
			todos.GET("/:id", controllers.GetTodo)
			todos.POST("", controllers.CreateTodo)
			todos.PUT("/:id", controllers.UpdateTodo)
			todos.DELETE("/:id", controllers.DeleteTodo)
		}
	}
	return r
}

func main() {
	r := setupRouter()

	srv := &http.Server{
		Addr:    ":8080",
		Handler: r,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed { //监听端口并启动服务
			log.Fatalf("ListenAndServe: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit //会阻塞在这里
	log.Println("正在关闭服务器")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Shutdown: %v", err)
	}
	log.Println("服务器已优雅关闭")

}
