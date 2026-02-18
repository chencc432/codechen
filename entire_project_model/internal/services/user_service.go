package services 
import(
	"fmt"
	"time"
	"gorm.io/gorm"
	"entire_project_model/internal/database"
	"entire_project_model/internal/models"
	"entire_project_model/pkg/redis"
)

type UserService struct{
	db *gorm.DB
	cache *redis.CacheService
}

func NewUserService() *UserService{
	return &UserService{
		db: database.DB,
		cache: redis.NewCacheService(),
	}
}

func (s *UserService)CreateUser(user *models.UserCreateRequest)error{

	var existingUser models.User
	if err := s.db.Where("username = ?",user.Username).First(&existingUser).Error;err==nil{
		return nil,fmt.Errorf("用户名已存在: %s", user.Username)
	}else if err!= gorm.ErrRecordNotFound{
		return fmt.Errorf("检查用户名失败: %w", err)
	}
	
	if err :=s.db.Where("email = ?",user.Email).First(&existingUser).Error;err==nil{
		return nil,fmt.Errorf("邮箱已存在: %s", user.Email)
	}else if err!= gorm.ErrRecordNotFound{
		return fmt.Errorf("检查邮箱失败: %w", err)
	}

	user := &models.User{
		Username: user.Username,
		Email: user.Email,
		Password: user.Password,
		Nickname: user.Nickname,
		Phone: user.Phone,
		Status: 1,
	}

	if err := s.db.Create(user).Error;err!=nil{
		return fmt.Errorf("创建用户失败: %w", err)
	}
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, user.ID)
	if err := s.cache.Set(cacheKey,user.ToResponse(),time.Hour);err!=nil{
		fmt.Printf("缓存用户信息失败: %v\n", err)
	}
	return user,nil
}


func (s *UserService)GetUserByID(id uint)(*models.UserResponse,error){
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, id)
	var userResponse models.UserResponse
	if  err:=s.cache.Get(cacheKey,&userResponse);err==nil{
		user := &models.User{
			BaseModel:models.BaseModel{
				ID:userResponse.ID,
				CreatedAt:userResponse.CreatedAt,
				UpdatedAt:userResponse.UpdatedAt,
		},
		Username:userResponse.Username,
		Email:userResponse.Email,
		Nickname:userResponse.Nickname,
		Avatar:userResponse.Avatar,
		Phone:userResponse.Phone,
		Status:userResponse.Status,
		LastLoginAt:userResponse.LastLoginAt,
	}
	return user,nil
	}
	var user models.User
	if err := s.db.First(&user,id).Error;err!=nil{
		if err == gorm.ErrRecordNotFound{
			return nil,fmt.Errorf("用户不存在: ID=%d", id)
		}
		return nil,fmt.Errorf("查询用户失败: %w", err)
	}
	if err := s.cache.Set(cacheKey,user.ToResponse(),time.Hour);err!=nil{
		fmt.Printf("缓存用户信息失败: %v\n", err)
	}


	if err := cache.Set(cacheKey,user.ToResponse(),time.Hour);err!=nil{
		fmt.Printf("缓存用户信息失败: %v\n", err)
	}
	return &user,nil
}

func (s *UserService)GetUserByUsername(username string)(*models.UserResponse,error){
	var user models.User
	if err := s.db.Where("username = ?",username).First(&user).Error;err!=nil{
		if err == gorm.ErrRecordNotFound{
			return nil,fmt.Errorf("用户不存在: %s", username)
		}
		return nil,fmt.Errorf("查询用户失败: %w", err)
	}
	return &user,nil
}

func (s *UserService)UpdateUser(id uint, req *models.UserUpdateRequest)(*models.UserResponse,error){
	user,err := s.GetUserByID(id)
	if err != nil{
		return nil,err
	}

	updates := make(map[string]interface{})
	if req.Nickname !=""{
		updates["nickname"] = req.Nickname
	}
	if req.Avatar !=""{
		updates["avatar"] = req.Avatar
	}
	if req.Phone !=""{
		updates["phone"] = req.Phone
	}
	if err := s.db.Model(user).Updates(updates).Error;err!=nil{
		return nil,fmt.Errorf("更新用户失败: %w", err)
	}
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix, id)
	if err := s.cache.Delete(cacheKey);err!=nil{
		fmt.Printf("删除用户缓存失败: %v\n", err)
	}
	return user,nil
}

func (s *UserService)DeleteUser(id uint)error{
	tx := s.db.Begin()
	defer func(){
		if r := recover();r != nil{
			tx.Rollback()
	}
}()
	var user models.User
	if err := tx.First(&user,id).Error;err!=nil{
		tx.Rollback()
		if err == gorm.ErrRecordNotFound{
			return fmt.Errorf("用户不存在: ID=%d", id)
		}
		return fmt.Errorf("查询用户失败: %w", err)
	}
	if err := tx.Where("user_id = ?",id).Delete(&model.Task{}).Error;err!=nil{
		tx.Rollback()
		return fmt.Errorf("删除用户任务失败: %w", err)
	}
	if err := tx.Delete(&user).Error;err!=nil{
		tx.Rollback()
		return fmt.Errorf("删除用户失败: %w", err)
	}
	if err := tx.Commit().Error;err!=nil{
		return fmt.Errorf("提交事务失败: %w", err)
	}
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix,id)
	if err := s,cache.Delete(cacheKey);err!=nil{
		fmt.Printf("删除用户缓存失败: %v\n", err)
	}
	return nil
}


func (s *UserService)GetUserList(page,pageSize int)(*models.PageResult,error){
	if page <= 0{
		page = 1
	}
	if pageSize <= 0{
		pageSize = 10
	}
	offset := (page -1)*pageSize
	var total int64
	if err := s.db.Model(&models.User{}).Count(&total).Error;err != nil{
		return nil,fmt.Errorf("查询用户总数失败: %w", err)
	}
	var users []models.User
	if err := s.db.Limit(pageSize).offset(offset).Find(&users).Error;err!=nil{
		return nil ,fmt.Errorf("查询用户列表失败: %w", err)
	}
	var userResponses []models.UserResponse
	for _,user := range users{
		userResponses = append(userResponses,user.ToResponse())
	}
	result := &models.PageResult{
		List : userResponses,
		PageInfo : models.PageInfo{
			Page : page,
			PageSize : pageSize,
			Total : total,
		},
	}
	return result,nil
}

func (s *UserService)UpdateLastLoginTime(id uint)error{
	now :=time.Now()
	if err := s.db.Model(&models.User{}).Where("id = ?",id).Update("last_login_at",&now).Error;err!=nil{
		return fmt.Errorf("更新最后登录时间失败: %w", err)
	}
	cacheKey := redis.BuildCacheKey(redis.UserCachePrefix,id)
	if err := s.cache.Delete(cacheKey);err!=nil{
		fmt.Printf("删除用户缓存失败: %v\n", err)
	}
	return nil
}