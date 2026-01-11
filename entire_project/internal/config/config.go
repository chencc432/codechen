// Package config 配置管理模块
// 学习要点：使用 viper 进行配置管理，支持多种配置格式
package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config 应用程序配置结构体
// 学习要点：使用结构体标签进行配置映射，支持多种配置源
type Config struct {
	Server   ServerConfig   `yaml:"server"`   // 服务器配置
	Database DatabaseConfig `yaml:"database"` // 数据库配置
	Redis    RedisConfig    `yaml:"redis"`    // Redis配置
	Log      LogConfig      `yaml:"log"`      // 日志配置
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `yaml:"port"` // 服务端口
	Mode string `yaml:"mode"` // 运行模式
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	MySQL MySQLConfig `yaml:"mysql"` // MySQL配置
}

// MySQLConfig MySQL数据库配置
// 学习要点：数据库连接参数的配置管理
type MySQLConfig struct {
	Host            string `yaml:"host"`              // 主机地址
	Port            int    `yaml:"port"`              // 端口号
	Username        string `yaml:"username"`          // 用户名
	Password        string `yaml:"password"`          // 密码
	DBName          string `yaml:"dbname"`            // 数据库名
	Charset         string `yaml:"charset"`           // 字符集
	ParseTime       bool   `yaml:"parse_time"`        // 解析时间
	Loc             string `yaml:"loc"`               // 时区
	MaxIdleConns    int    `yaml:"max_idle_conns"`    // 最大空闲连接数
	MaxOpenConns    int    `yaml:"max_open_conns"`    // 最大打开连接数
	ConnMaxLifetime int    `yaml:"conn_max_lifetime"` // 连接最大生命周期
}

// RedisConfig Redis配置
// 学习要点：Redis 连接池配置
type RedisConfig struct {
	Host         string `yaml:"host"`           // 主机地址
	Port         int    `yaml:"port"`           // 端口号
	Password     string `yaml:"password"`       // 密码
	DB           int    `yaml:"db"`             // 数据库编号
	PoolSize     int    `yaml:"pool_size"`      // 连接池大小
	MinIdleConns int    `yaml:"min_idle_conns"` // 最小空闲连接数
}

// LogConfig 日志配置
type LogConfig struct {
	Level      string `yaml:"level"`       // 日志级别
	FilePath   string `yaml:"file_path"`   // 日志文件路径
	MaxSize    int    `yaml:"max_size"`    // 单个日志文件最大大小
	MaxBackups int    `yaml:"max_backups"` // 保留的日志文件数量
	MaxAge     int    `yaml:"max_age"`     // 日志文件保留天数
}

// GlobalConfig 全局配置实例
var GlobalConfig *Config

// Load 加载配置文件
// 学习要点：配置文件的读取和解析，错误处理
func Load(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 设置环境变量前缀
	viper.SetEnvPrefix("TASK")
	viper.AutomaticEnv()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置到结构体
	GlobalConfig = &Config{}
	if err := viper.Unmarshal(GlobalConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	return nil
}

// GetMySQLDSN 获取MySQL数据源名称
// 学习要点：DSN 字符串的构建，数据库连接字符串格式
func (c *MySQLConfig) GetMySQLDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.DBName,
		c.Charset,
		c.ParseTime,
		c.Loc,
	)
}

// GetRedisAddr 获取Redis地址
func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

// GetConnMaxLifetime 获取连接最大生命周期
func (c *MySQLConfig) GetConnMaxLifetime() time.Duration {
	return time.Duration(c.ConnMaxLifetime) * time.Second
}