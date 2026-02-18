package services

import(
	"fmt"
	"time"
	"gorm.io/gorm"
	"entire_project_model/internal/models"
	"entire_project_model/internal/database"
	"entire_project_model/pkg/redis"
)

type TaskService struct{
	db *gorm.DB
	cache *redis.CacheService
}

func NewTaskService() *TaskService{
	return &TaskService{
		db: database.DB,
		cache: redis.NewCacheService(),
	}
}

func(s *TaskService)CreateTask(userId uint, req *models.TaskCreateRequest) (*models.Task, error){
	//查数据库验证是否存在用户
	//开始事务创建任务
	  //任务结构体是怎样的
	  //关联任务标签，看看有没有这个任务，并且关联
	//提交事务
	//预加载关联目的是放到缓存
	//缓存任务信息
	//清除原来用户列表缓存
	//更新任务统计计数器
	var user models.User
	if  err:= s.db.First(&user,userId).Error;err!=nil{
		if err == gorm.ErrRecordNotFound{
			return nil, fmt.Errorf("用户不存在: ID=%d", userId)
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	tx := s.db.Begin()
	defer func(){
		if r := recover(); r !=nil{
			tx.Rollback()
			return nil, fmt.Errorf("创建任务失败: %w", err)
		}
	}()

	task := &models.Task{
		Title: req.Title,
		Description: req.Description,
		Priority: req.Priority,
		DueDate: req.DueDate,
		Status: models.TaskStatusPending,
		UserID: userId,
	}
	if err :=tx.Create(task).Error;err!=nil{
		tx.Rollback()
		return nil,fmt.Errorf("创建任务失败: %w", err)
	}

	if len(req.TagIDs)>0{
		var tags []models.Tag
		if err := tx.Where("id IN ?",req.TagIDs).Find(&tags).Error;err!=nil{
			tx.Rollback()
			return nil,fmt.Errorf("查询标签失败: %w", err)
		}
		if err := tx.Model(task).Association("Tags").Append(tags).Error;err!=nil{
			tx.Rollback()
			return nil,fmt.Errorf("关联标签失败: %w", err)
	    }
	}

	if err :=tx.commit().Error;err!=nil{
		return nil,fmt.Errorf("提交事务失败: %w", err)
	}
	
	if err := s.db.Preload("User").Preload("Tags").First(task,taskID).Error;err!=nil{
		return nil,fmt.Errorf("重新加载任务数据失败: %w", err)
	}

	cacheKey := redis.BuildCacheKey(redis.TaskCachePrefix,task.Id)
	if err := s.cache.Set(cacheKey,task,time.Hour);err!=nil{
		fmt.Printf("缓存任务信息失败: %v\n", err)
	}
	userTasksKey := redis.BuildCacheKey(redis.UserTasksPrefix,userId)
	if err := s.cache.Delete(userTasksKey);err!=nil{
		fmt.Printf("清除用户任务列表缓存失败: %v\n", err)
	}
	s.updateTaskStats(userId,models.TaskStatusPending,1)
	return task,nil
}

func (s *TaskService)GetTaskByID(id uint)(*models.Task,error){
	cacheKey := redis.BuildCacheKey(redis.TaskCachePrefix,id)

	var task models.Task
	if err:= s.cache.Get(cacheKey,&task);err ==nil{
		return &task,nil
	}

	if err:= s.db.Preload("User").Preload("Tags").First(&task,id).Error;err!=nil{
		if err == gorm.ErrRecordNotFound{
			return nil, fmt.Errorf("任务不存在: ID=%d", id)
		}
		return nil,fmt.Errorf("查询任务失败: %w", err)
	}
	if err := s.cache.Set(cacheKey,&task,time.Hour);err!=nil{
		fmt.Printf("缓存任务信息失败: %v\n", err)
	}
	return &task,nil
}

func (s *TaskService)UpdateTask(id uint, req *models.TaskUpdateRequest)(*models.Task,error){
	//获取现有任务
	//验证权限
	//开始事务
	//记录状态变更
	//准备更新数据
	//执行更新
	//更新标签关联
	//提交事务
	//重新加载任务数据
	//删除缓存
	//清除用户任务列表缓存
	//更新任务统计计数器
	task,err := s.GetTaskByID(id)
	if err !=nil{
		return nil,err
	}
	if task.UserID != userID{
		return nil,fmt.Errorf("没有权限修改此任务")
	}

	tx := s.db.Begin()
	defer func(){
		if r :=recover();r !=nil{
			tx.Rollback()
	    }
    }()
    oldStatus := task.Status

	updates := make(map[string]interface{})
	if req.Title !=nil{
		updates["title"] = *req.Title
	}
	if req.Description !=nil{
		updates["description"] = *req.Description
	}
	if req.Status !=nil{
		updates["status"] = *req.Status
		switch *req.Status{
		case models.TaskStatusInProgress:
			if task.StartTime ==nil{
				now := time.Now()
				updates["start_time"] = &now
			}
		case models.TaskStatusCompleted:
			if task.EndTime =nil{
				now := time.Now()
				updates["end_time"] = &now
			}
		}
	}
	if req.Priority !=nil{
		updates["priority"] = *req.Priority
	}
	if req.StartTime !=nil{
		updates["start_time"] = req.StartTime
	}
	if req.EndTime !=nil{
		updates["end_time"] = req.EndTime
	}
	if req.DueDate !=nil{
		updates["due_date"] = req.DueDate
	}
	if err := tx.Model(task).Updates(updates).Error;err!=nil{
		tx.Rollback()
		return nil,fmt.Errorf("更新任务失败: %w", err)
	}

	if req.TagIDs !=nil{
		if err := tx.Model(task).Association("Tags").Clear();err!=nil{
			tx.Rollback()
			return nil,fmt.Errorf("清除标签关联失败: %w", err)
	    }

		if len(req.TagIDs)>0{
			var tags []models.Tag
			if err := tx.Where("id IN ?",req.TagIDs).Find(&tags).Error;err!=nil{
				tx.Rollback()
				return nil,fmt.Errorf("查询标签失败: %w", err)
			}
			if err := tx.Model(task).Association("Tags").Append(tags);err!=nil{
				tx.Rollback()
				return nil,fmt.Errorf("关联标签失败: %w", err)
			}
		}
	}
	if err := tx.commit().Error;err!=nil{
		return nil,fmt.Errorf("提交事务失败: %w", err)
	}
	if err := s.db.Preload("User").Preload("Tags").First(task,id).Error;err!=nil{
		return nil,fmt.Errorf("重新加载任务数据失败: %w", err)
	}
	cacheKey := redis.BuildCacheKey(redis.TaskCachePrefix,id)
	if err := s.cache.Delete(cacheKey);err !=nil{
		fmt.Printf("删除任务缓存失败: %v\n", err)
	}
	if req.Status != nil && oldStatus != *req.Status{
		s.updateTaskStats(userID,oldStatus,-1)
		s.updateTaskStats(userID,*req.Status,1)
	}
	return task,nil

}
func (s *TaskService)DeleteTask(id uint,userID uint)error{
	//获取任务
	//验证权限
	//开始事务
	//删除标签关联
	//软删除任务
	//提交事务
	//删除缓存
	//清除用户任务列表缓存
	//更新任务统计计数器
	task,err := s.GetTaskById(id)
	if err!=nil{
		return err
	}
	if task.UserID != userID{
		return fmt.Errorf("没有权限删除此任务")
	}
	tx :=s.db.Begin()
	defer func(){
		if r:=recover();r!=nil{
			tx.Rollback()
		}
	}()
	if err := tx.Model(task).Association("Tags").Clear();err!=nil{
		tx.Rollback()
		return fmt.Errorf("清除标签关联失败: %w", err)
	}
	if err := tx.Delete(task).Error;err!=nil{
		tx.Rollback()
		return fmt.Errorf("删除任务失败: %w", err)
	}
	if err := tx.commit().Error;err!=nil{
		return fmt.Errorf("提交事务失败: %w", err)
	}
	cacheKey := redis.BuildCacheKey(redis.TaskCachePrefix,id)
	if err := s.cache.Delete(cacheKey);err !=nil{
		fmt.Printf("删除任务缓存失败: %v\n", err)
	}
	userTasksKey := redis.BuildCacheKey(redis.UserTasksPrefix,userID)
	if err := s.cache.Delete(userTasksKey);err !=nil{
		fmt.Printf("清除用户任务列表缓存失败: %v\n", err)
	}
	s.updateTaskStats(userID,task.Status,-1)
	return nil
}
func (s *TaskService)QueryTasks(req *models.TaskQueryRequest)([]*models.Task,int64,error){
	//验证过滤条件