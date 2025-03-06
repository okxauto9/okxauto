package server

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// 登录处理
func (s *Server) handleLogin(c *gin.Context) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	if req.Username != s.config.Username || req.Password != s.config.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 生成JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": req.Username,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(s.config.JWTKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
	})
}

// 获取交易历史
func (s *Server) handleGetTradeHistory(c *gin.Context) {
	// 获取查询参数
	limit := 100 // 默认返回最近100条记录
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// 获取交易历史
	trades, err := s.db.GetTradeHistory(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取交易历史失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"trades": trades,
	})
}

// 获取系统状态
func (s *Server) handleGetSystemStatus(c *gin.Context) {
	status := struct {
		Running    bool      `json:"running"`
		StartTime  time.Time `json:"start_time"`
		Strategies []string  `json:"strategies"`
	}{
		Running:    true,
		StartTime:  time.Now(),
		Strategies: []string{"Grid", "RSI"},
	}

	c.JSON(http.StatusOK, status)
}

// 获取账户余额
func (s *Server) handleGetBalance(c *gin.Context) {
	balances, err := s.engine.GetBalance()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"balances": balances,
	})
}

// 启用策略
func (s *Server) handleEnableStrategy(c *gin.Context) {
	strategyName := c.Param("name")
	err := s.engine.EnableStrategy(strategyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "策略已启用"})
}

// 禁用策略
func (s *Server) handleDisableStrategy(c *gin.Context) {
	strategyName := c.Param("name")
	err := s.engine.DisableStrategy(strategyName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "策略已禁用"})
}

// 更新策略配置
func (s *Server) handleUpdateStrategyConfig(c *gin.Context) {
	strategyName := c.Param("name")
	var config map[string]interface{}

	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的配置参数"})
		return
	}

	err := s.engine.UpdateStrategyConfig(strategyName, config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "配置已更新"})
}

// 获取活跃交易
func (s *Server) handleGetActiveTrades(c *gin.Context) {
	// 实现获取活跃交易的逻辑
	c.JSON(http.StatusOK, gin.H{
		"trades": []interface{}{}, // 返回空列表或实际的活跃交易
	})
}

// 获取策略列表
func (s *Server) handleGetStrategies(c *gin.Context) {
	// 从交易引擎配置中获取策略状态
	engineConfig := s.engine.GetConfig()
	
	strategies := []struct {
		Name    string `json:"name"`
		Enabled bool   `json:"enabled"`
	}{
		{Name: "Grid", Enabled: engineConfig.Grid.Enabled},
		{Name: "RSI", Enabled: engineConfig.RSI.Enabled},
	}

	c.JSON(http.StatusOK, strategies)
} 