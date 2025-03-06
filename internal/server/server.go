package server

import (
	"context"
	"net/http"
	"time"

	"okxauto/internal/database"
	"okxauto/internal/trading"

	"github.com/gin-gonic/gin"
)

type Server struct {
	config *Config
	engine *trading.Engine
	db     *database.Database
	router *gin.Engine
	server *http.Server
}

// Config 定义服务器配置
type Config struct {
	Port     string `yaml:"port"`
	JWTKey   string `yaml:"jwt_key"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Grid     struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"grid_strategy"`
	RSI struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"rsi_strategy"`
}

func New(config *Config, db *database.Database, engine *trading.Engine) *Server {
	s := &Server{
		config: config,
		engine: engine,
		db:     db,
		router: gin.Default(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// 中间件
	s.router.Use(s.corsMiddleware())
	s.router.Use(s.authMiddleware())

	// API路由
	api := s.router.Group("/api")
	{
		// 认证相关
		api.POST("/login", s.handleLogin)

		// 交易相关
		trades := api.Group("/trades")
		{
			trades.GET("/history", s.handleGetTradeHistory)
			trades.GET("/active", s.handleGetActiveTrades)
		}

		// 策略相关
		strategies := api.Group("/strategies")
		{
			strategies.GET("/", s.handleGetStrategies)
			strategies.POST("/:name/enable", s.handleEnableStrategy)
			strategies.POST("/:name/disable", s.handleDisableStrategy)
			strategies.PUT("/:name/config", s.handleUpdateStrategyConfig)
		}

		// 系统相关
		system := api.Group("/system")
		{
			system.GET("/status", s.handleGetSystemStatus)
			system.GET("/balance", s.handleGetBalance)
		}
	}
}

func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: s.router,
	}

	return s.server.ListenAndServe()
}

func (s *Server) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(ctx)
}
