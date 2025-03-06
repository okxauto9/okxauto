package api

import (
    "fmt"
    "log"
    "time"
    "strings"
)

// RetryConfig 重试配置
type RetryConfig struct {
    MaxRetries  int
    DelayMillis int
}

// DefaultRetryConfig 默认重试配置
var DefaultRetryConfig = RetryConfig{
    MaxRetries:  3,
    DelayMillis: 1000, // 1秒
}

// IsTemporaryError 判断是否为临时性错误
func IsTemporaryError(err error) bool {
    if err == nil {
        return false
    }
    
    errMsg := strings.ToLower(err.Error())
    return strings.Contains(errMsg, "upgrading") ||
           strings.Contains(errMsg, "try again") ||
           strings.Contains(errMsg, "timeout") ||
           strings.Contains(errMsg, "too many requests")
}

// RetryOperation 重试执行操作
func RetryOperation(operation func() error, config RetryConfig) error {
    var lastErr error
    
    for i := 0; i < config.MaxRetries; i++ {
        if i > 0 {
            log.Printf("重试操作 (第%d次)...", i+1)
            time.Sleep(time.Duration(config.DelayMillis) * time.Millisecond)
        }
        
        err := operation()
        if err == nil {
            return nil
        }
        
        lastErr = err
        if !IsTemporaryError(err) {
            return err // 如果不是临时性错误，直接返回
        }
        
        log.Printf("操作失败 (尝试 %d/%d): %v", i+1, config.MaxRetries, err)
    }
    
    return fmt.Errorf("达到最大重试次数 (%d): %v", config.MaxRetries, lastErr)
} 