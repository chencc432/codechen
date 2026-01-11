// Package redis Redis缓存管理
// 学习要点：Redis连接管理，缓存操作封装，连接池配置
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"task-management-system/internal/config"
)

// Client Redis客户端实例
var Client *redis.Client

// InitRedis 初始化Redis连接
// 学习要点：Redis客户端初始化，连接池配置
func InitRedis() error {
	cfg := &config.GlobalConfig.Redis
	
	// 创建Redis客户端
	Client = redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),    // Redis服务器地址
		Password:     cfg.Password,          // 密码
		DB:           cfg.DB,                // 数据库编号
		PoolSize:     cfg.PoolSize,          // 连接池大小
		MinIdleConns: cfg.MinIdleConns,      // 最小空闲连接数
	})
	
	// 测试连接
	ctx := context.Background()
	if err := Client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("连接Redis失败: %w", err)
	}
	
	fmt.Println("✅ Redis连接成功")
	return nil
}

// CacheService Redis缓存服务结构体
// 学习要点：服务层设计，缓存操作封装
type CacheService struct {
	client *redis.Client
	ctx    context.Context
}

// NewCacheService 创建缓存服务实例
func NewCacheService() *CacheService {
	return &CacheService{
		client: Client,
		ctx:    context.Background(),
	}
}

// Set 设置缓存
// 学习要点：泛型的使用，JSON序列化，过期时间设置
func (c *CacheService) Set(key string, value interface{}, expiration time.Duration) error {
	// 序列化为JSON
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化数据失败: %w", err)
	}
	
	// 设置缓存
	if err := c.client.Set(c.ctx, key, jsonBytes, expiration).Err(); err != nil {
		return fmt.Errorf("设置缓存失败: %w", err)
	}
	
	return nil
}

// Get 获取缓存
// 学习要点：反序列化，缓存未命中处理
func (c *CacheService) Get(key string, dest interface{}) error {
	// 获取缓存
	jsonStr, err := c.client.Get(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("缓存键不存在: %s", key)
		}
		return fmt.Errorf("获取缓存失败: %w", err)
	}
	
	// 反序列化
	if err := json.Unmarshal([]byte(jsonStr), dest); err != nil {
		return fmt.Errorf("反序列化数据失败: %w", err)
	}
	
	return nil
}

// Delete 删除缓存
func (c *CacheService) Delete(key string) error {
	if err := c.client.Del(c.ctx, key).Err(); err != nil {
		return fmt.Errorf("删除缓存失败: %w", err)
	}
	return nil
}

// Exists 检查键是否存在
func (c *CacheService) Exists(key string) (bool, error) {
	count, err := c.client.Exists(c.ctx, key).Result()
	if err != nil {
		return false, fmt.Errorf("检查键存在失败: %w", err)
	}
	return count > 0, nil
}

// SetExpire 设置键的过期时间
func (c *CacheService) SetExpire(key string, expiration time.Duration) error {
	if err := c.client.Expire(c.ctx, key, expiration).Err(); err != nil {
		return fmt.Errorf("设置过期时间失败: %w", err)
	}
	return nil
}

// GetTTL 获取键的剩余过期时间
func (c *CacheService) GetTTL(key string) (time.Duration, error) {
	ttl, err := c.client.TTL(c.ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("获取TTL失败: %w", err)
	}
	return ttl, nil
}

// IncrBy 增加数值
// 学习要点：原子操作，计数器实现
func (c *CacheService) IncrBy(key string, value int64) (int64, error) {
	result, err := c.client.IncrBy(c.ctx, key, value).Result()
	if err != nil {
		return 0, fmt.Errorf("增加数值失败: %w", err)
	}
	return result, nil
}

// DecrBy 减少数值
func (c *CacheService) DecrBy(key string, value int64) (int64, error) {
	result, err := c.client.DecrBy(c.ctx, key, value).Result()
	if err != nil {
		return 0, fmt.Errorf("减少数值失败: %w", err)
	}
	return result, nil
}

// HSet 设置哈希字段
// 学习要点：Redis哈希操作，复杂数据结构缓存
func (c *CacheService) HSet(key, field string, value interface{}) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("序列化哈希值失败: %w", err)
	}
	
	if err := c.client.HSet(c.ctx, key, field, jsonBytes).Err(); err != nil {
		return fmt.Errorf("设置哈希字段失败: %w", err)
	}
	return nil
}

// HGet 获取哈希字段
func (c *CacheService) HGet(key, field string, dest interface{}) error {
	jsonStr, err := c.client.HGet(c.ctx, key, field).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("哈希字段不存在: %s.%s", key, field)
		}
		return fmt.Errorf("获取哈希字段失败: %w", err)
	}
	
	if err := json.Unmarshal([]byte(jsonStr), dest); err != nil {
		return fmt.Errorf("反序列化哈希值失败: %w", err)
	}
	
	return nil
}

// HDel 删除哈希字段
func (c *CacheService) HDel(key string, fields ...string) error {
	if err := c.client.HDel(c.ctx, key, fields...).Err(); err != nil {
		return fmt.Errorf("删除哈希字段失败: %w", err)
	}
	return nil
}

// HGetAll 获取哈希的所有字段
func (c *CacheService) HGetAll(key string) (map[string]string, error) {
	result, err := c.client.HGetAll(c.ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("获取哈希所有字段失败: %w", err)
	}
	return result, nil
}

// LPush 从列表左侧推入元素
// 学习要点：Redis列表操作，队列实现
func (c *CacheService) LPush(key string, values ...interface{}) error {
	// 序列化所有值
	serializedValues := make([]interface{}, len(values))
	for i, v := range values {
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("序列化列表元素失败: %w", err)
		}
		serializedValues[i] = jsonBytes
	}
	
	if err := c.client.LPush(c.ctx, key, serializedValues...).Err(); err != nil {
		return fmt.Errorf("推入列表失败: %w", err)
	}
	return nil
}

// RPop 从列表右侧弹出元素
func (c *CacheService) RPop(key string, dest interface{}) error {
	jsonStr, err := c.client.RPop(c.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("列表为空: %s", key)
		}
		return fmt.Errorf("弹出列表元素失败: %w", err)
	}
	
	if err := json.Unmarshal([]byte(jsonStr), dest); err != nil {
		return fmt.Errorf("反序列化列表元素失败: %w", err)
	}
	
	return nil
}

// LLen 获取列表长度
func (c *CacheService) LLen(key string) (int64, error) {
	length, err := c.client.LLen(c.ctx, key).Result()
	if err != nil {
		return 0, fmt.Errorf("获取列表长度失败: %w", err)
	}
	return length, nil
}

// Close 关闭Redis连接
func Close() error {
	if Client == nil {
		return nil
	}
	return Client.Close()
}

// 缓存键常量定义
// 学习要点：缓存键的标准化管理，避免键名冲突
const (
	UserCachePrefix     = "user:"        // 用户缓存前缀
	TaskCachePrefix     = "task:"        // 任务缓存前缀
	UserTasksPrefix     = "user_tasks:"  // 用户任务列表前缀
	TaskCountPrefix     = "task_count:"  // 任务统计前缀
	LoginAttemptsPrefix = "login_attempts:" // 登录尝试次数前缀
)

// BuildCacheKey 构建缓存键
func BuildCacheKey(prefix string, id interface{}) string {
	return fmt.Sprintf("%s%v", prefix, id)
}