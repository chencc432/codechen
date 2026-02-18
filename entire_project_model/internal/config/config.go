package config

import (
	"fmt"
	"time"
	"github.com/spf13/viper"
	net/url

)

type ServerConfig struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"`

}

type DatabaseConfig struct {
	MySQL MySQLConfig `yaml:"mysql"`
}

type MySQLConfig struct {
	Host string `yaml:"host"`
	Port int `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	DBName string `yaml:"dbname"`
	Charset string `yaml:"charset"`
	ParseTime bool `yaml:"parse_time"`
	Loc string `yaml:"loc"`
	MaxIdleConns int `yaml:"max_idle_conns"`
	MaxOpenConns int `yaml:"max_open_conns"`
	ConnMaxLifetime int `yaml:"conn_max_lifetime"`
    Socket string `yaml:"socket"`
	
}

type RedisConfig struct {
	Host string `yaml:"host"`
	Port int `yaml:"port"`
	Password string `yaml:"password"`
	DB int `yaml:"db"`
	PoolSize int `yaml:"pool_size"`
	MinIdleConns int `yaml:"min_idle_conns"`
}

type Config struct {
	Server ServerConfig `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	Redis RedisConfig `yaml:"redis"`
	Log LogConfig `yaml:"log"`
}

type LogConfig struct{
	Level string `yaml:"level"`
	FilePath string `yaml:"file_path"`
	MaxSize int `yaml:"max_size"`
	MaxBackups int `yaml:"max_backups"`
	MaxAge int `yaml:"max_age"`
}


var GlobalConfig *Config
func Load(configPath string)error{
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")
	viper.SetEnvPrefix("Task")
	viper.AutomaticEnv()
	if err:=viper.ReadInConfig();err!=nil{
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	GlobalConfig = &Config{

	}
	if err := viper.Unmarshal(GlobalConfig);err!=nil{
		return fmt.Errorf("解析配置文件失败: %w", err)
	}
	return nil
}

func (c *MySQLConfig) GetMySQLDSN() string {
	loc := url.QueryEscape(c.Loc)
	if c.Socket != "" {
		return fmt.Sprintf("%s:%s@unix(%s)/%s?charset=%s&parseTime=%t&loc=%s", c.Username, c.Password, c.Socket, c.DBName, c.Charset, c.ParseTime, loc
	)
}
return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=%t&loc=%s", c.Username, c.Password, c.Host, c.Port, c.DBName, c.Charset, c.ParseTime, loc
}

func (c *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)

}

func (c *MySQLConfig) GetConnMaxLifetime() time.Duration {
	return time.Duration(c.ConnMaxLifetime) * time.Second
}