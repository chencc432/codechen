// Package utils 工具函数集合
// 学习要点：工具函数的组织，UUID生成，随机数生成
package utils

import (
	"crypto/rand"
	"fmt"
	"time"
)

// GenerateRequestID 生成请求ID
// 学习要点：分布式系统中的请求追踪标识生成
func GenerateRequestID() string {
	// 使用时间戳 + 随机数生成请求ID
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)
	
	return fmt.Sprintf("%d-%x", timestamp, randomBytes)
}

// GenerateUUID 生成简单的UUID（用于演示）
// 学习要点：UUID的概念和基本生成方法
// 注意：生产环境建议使用专业的UUID库，如github.com/google/uuid
func GenerateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// GenerateShortID 生成短ID（8位随机字符串）
// 学习要点：短ID生成，适用于不需要全局唯一的场景
func GenerateShortID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 8)
	for i := range b {
		randomByte := make([]byte, 1)
		rand.Read(randomByte)
		b[i] = charset[int(randomByte[0])%len(charset)]
	}
	return string(b)
}