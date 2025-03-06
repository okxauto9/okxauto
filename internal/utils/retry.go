/*
 * @Author: okxauto9@gmail.com
 * @Date: 2025-02-14 19:01:55
 * @LastEditors: okxauto9@gmail.com
 * @LastEditTime: 2025-02-14 19:02:43
 * @FilePath: \okxauto\internal\utils\retry.go
 * @Description:
 *
 * Copyright (c) 2025 by okxauto9@gmail.com, All Rights Reserved.
 */
package utils

import (
	"time"
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

// RetryOperation 重试执行操作
func RetryOperation(operation func() error, config RetryConfig) error {
	var lastErr error

	for i := 0; i < config.MaxRetries; i++ {
		if i > 0 {
			time.Sleep(time.Duration(config.DelayMillis) * time.Millisecond)
		}

		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	return lastErr
}
