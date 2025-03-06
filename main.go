/*
 * @Author: okxauto9@gmail.com
 * @Date: 2025-02-04 22:34:12
 * @LastEditors: xxxxx@xx.com
 * @LastEditTime: 2025-03-05 18:14:53
 * @FilePath: \aaa8\main.go
 * @Description:
 *
 * Copyright (c) 2025 by okxauto9@gmail.com, All Rights Reserved.
 */
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"okxauto/internal/api"
	"okxauto/internal/api/config"
	"okxauto/internal/database"
	"okxauto/internal/server"
	"okxauto/internal/trading"
)

func main() {
	configFile := flag.String("config", "config/config.yaml", "配置文件路径")
	flag.Parse()

	// 加载配置
	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 初始化数据库
	db, err := database.New("data/trades.db")
	if err != nil {
		log.Fatalf("初始化数据库失败: %v", err)
	}
	defer db.Close()

	// 初始化数据库表结构
	if err := db.Initialize(); err != nil {
		log.Fatalf("初始化数据库表结构失败: %v", err)
	}

	// 创建API客户端
	apiClient := api.NewOKXClient(
		cfg.API.Key,
		cfg.API.Secret,
		cfg.API.Passphrase,
		cfg.API.Mode,
	)

	// 创建交易引擎
	tradingConfig := trading.Config{
		Mode:           cfg.Trading.Mode,
		TradeType:      cfg.Trading.TradeType,
		Leverage:       cfg.Trading.Leverage,
		MarginMode:     cfg.Trading.MarginMode,
		ReserveBalance: cfg.Trading.ReserveBalance,
		Symbols:        cfg.Trading.Symbols,
		LongPosition:   cfg.Trading.LongPosition,
		ShortPosition:  cfg.Trading.ShortPosition,
		Grid:           cfg.Trading.Grid,
		RSI:            cfg.Trading.RSI,
	}

	engine, err := trading.NewEngine(apiClient, db, tradingConfig)
	if err != nil {
		log.Fatalf("创建交易引擎失败: %v", err)
	}

	// 启动交易引擎
	if err := engine.Start(); err != nil {
		log.Fatalf("启动交易引擎失败: %v", err)
	}
	defer engine.Stop()

	// 创建并启动HTTP服务器
	srv := server.New(&cfg.Server, db, engine)
	go func() {
		if err := srv.Start(); err != nil {
			log.Printf("HTTP服务器停止: %v", err)
		}
	}()

	// 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("正在关闭服务...")
}
