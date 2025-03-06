package trading

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"okxauto/internal/api"
	"okxauto/internal/database"
	dbmodels "okxauto/internal/database/models"
	"okxauto/internal/models"
	"okxauto/internal/trading/strategies"
	"okxauto/internal/types"
)

type Engine struct {
	api        *api.OKXClient
	db         *database.Database
	config     *Config
	strategies []types.Strategy
	signals    chan *types.Signal
	stopChan   chan struct{}
	wg         sync.WaitGroup
}

func NewEngine(apiClient *api.OKXClient, db *database.Database, config Config) (*Engine, error) {
	engine := &Engine{
		api:      apiClient,
		db:       db,
		config:   &config,
		signals:  make(chan *types.Signal, 100),
		stopChan: make(chan struct{}),
	}

	// 根据交易类型选择合适的交易对
	var symbols []string
	if config.TradeType == "futures" {
		// 使用合约交易对
		for _, symbol := range config.Symbols {
			if strings.HasSuffix(symbol, "-SWAP") {
				symbols = append(symbols, symbol)
			}
		}
	} else {
		// 使用现货交易对
		for _, symbol := range config.Symbols {
			if !strings.HasSuffix(symbol, "-SWAP") {
				symbols = append(symbols, symbol)
			}
		}
	}

	// 初始化策略
	if config.Grid.Enabled {
		for _, symbol := range symbols {
			strategy := strategies.NewGridStrategy(apiClient, symbol, strategies.GridConfig{
				Enabled:     config.Grid.Enabled,
				UpperPrice:  config.Grid.UpperPrice,
				LowerPrice:  config.Grid.LowerPrice,
				GridNumber:  config.Grid.GridNumber,
				TotalAmount: config.Grid.TotalAmount,
			})
			engine.strategies = append(engine.strategies, strategy)
		}
	}

	if config.RSI.Enabled {
		for _, symbol := range symbols {
			strategy := strategies.NewRSIStrategy(apiClient, symbol, strategies.RSIConfig{
				Enabled:             config.RSI.Enabled,
				Period:              config.RSI.Period,
				OverboughtThreshold: config.RSI.OverboughtThreshold,
				OversoldThreshold:   config.RSI.OversoldThreshold,
			})
			engine.strategies = append(engine.strategies, strategy)
		}
	}

	return engine, nil
}

func (e *Engine) Start() error {
	// 启动策略
	for _, strategy := range e.strategies {
		if err := strategy.Initialize(); err != nil {
			return err
		}
	}

	// 启动信号处理
	e.wg.Add(1)
	go e.processSignals()

	// 启动行情更新
	for _, symbol := range e.config.Symbols {
		e.wg.Add(1)
		go e.updateMarketData(symbol)
	}

	// 启动保证金检查定时器
	go func() {
		//ticker := time.NewTicker(1 * time.Minute)
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-e.stopChan:
				return
			case <-ticker.C:
				for _, symbol := range e.config.Symbols {
					if err := e.checkAndAdjustMargin(symbol); err != nil {
						log.Printf("检查保证金失败: %v", err)
					}
				}
			}
		}
	}()

	return nil
}

func (e *Engine) Stop() {
	close(e.stopChan)
	e.wg.Wait()

	for _, strategy := range e.strategies {
		strategy.Stop()
	}
}

func (e *Engine) processSignals() {
	defer e.wg.Done()

	for {
		select {
		case <-e.stopChan:
			return
		case signal := <-e.signals:
			if err := e.executeSignal(signal); err != nil {
				log.Printf("执行信号失败: %v", err)
			}
		}
	}
}

func (e *Engine) executeSignal(signal *types.Signal) error {
	log.Printf("[%s] 开始执行交易信号: %s %.2f@%.2f",
		signal.Symbol, signal.Action, signal.Amount, signal.Price)

	// 检查账户资金
	if e.config.TradeType == "futures" {
		// 获取持仓信息
		positions, err := e.api.GetPositions(signal.Symbol)
		if err != nil {
			log.Printf("[%s] 获取持仓信息失败: %v", signal.Symbol, err)
		} else {
			for _, pos := range positions {
				log.Printf("[%s] 当前持仓: 方向=%s, 数量=%.4f, 均价=%.4f",
					signal.Symbol, pos.PosSide, pos.Position, pos.AvgPrice)
			}
		}

		balances, err := e.api.GetBalances()
		if err != nil {
			return fmt.Errorf("获取余额失败: %v", err)
		}

		// 打印所有余额信息
		log.Printf("[%s] 当前账户余额:", signal.Symbol)
		for _, balance := range balances {
			log.Printf("  币种: %s, 总额: %s, 可用: %s, 冻结: %s",
				balance.Currency, balance.Balance, balance.Available, balance.Frozen)
		}

		// 检查USDT余额
		hasUSDT := false
		for _, balance := range balances {
			if balance.Currency == "USDT" {
				available, _ := strconv.ParseFloat(balance.Available, 64)
				required := signal.Amount * signal.Price / float64(e.config.Leverage)
				if available < required {
					log.Printf("[%s] USDT余额不足，无法开仓: 需要 %.2f USDT (考虑%d倍杠杆), 可用 %.2f USDT",
						signal.Symbol, required, e.config.Leverage, available)
					return fmt.Errorf("USDT余额不足")
				}
				hasUSDT = true
				log.Printf("[%s] USDT余额充足，可以开仓: 需要 %.2f USDT, 可用 %.2f USDT",
					signal.Symbol, required, available)
				break
			}
		}

		if !hasUSDT {
			log.Printf("[%s] 未找到USDT余额，无法开仓", signal.Symbol)
			return fmt.Errorf("未找到USDT余额")
		}
	}

	// 设置杠杆倍数
	posSide := "long"
	if signal.Action == "sell" {
		posSide = "short"
	}

	// 在这里声明err变量
	var err error
	err = e.api.SetLeverage(signal.Symbol,
		fmt.Sprintf("%d", e.config.Leverage),
		e.config.MarginMode,
		posSide)

	if err != nil {
		log.Printf("[%s] 设置杠杆倍数失败: %v", signal.Symbol, err)
		return err
	}
	log.Printf("[%s] 设置杠杆倍数成功: %d", signal.Symbol, e.config.Leverage)

	// 计算所需保证金
	margin := signal.Price * signal.Amount / float64(e.config.Leverage)

	// 检查余额是否充足
	if err := e.checkBalance(margin); err != nil {
		log.Printf("[%s] 下单失败: %v", signal.Symbol, err)
		return err
	}

	// 创建订单请求
	orderReq := &api.PlaceOrderRequest{
		InstId:  signal.Symbol,
		TdMode:  e.config.MarginMode,
		Side:    api.OrderSide(signal.Action),
		OrdType: api.Market,
		Sz:      "200", // 固定为1张合约
	}

	// 设置合约特有参数
	if e.config.TradeType == "futures" {
		orderReq.PosSide = posSide
		orderReq.Lever = fmt.Sprintf("%d", e.config.Leverage)
		log.Printf("[%s] 合约交易模式: 杠杆=%d, 保证金模式=%s, 持仓方向=%s, 数量=%s张",
			signal.Symbol, e.config.Leverage, e.config.MarginMode, orderReq.PosSide, orderReq.Sz)
	}

	// 打印完整的订单请求
	reqJSON, _ := json.MarshalIndent(orderReq, "", "  ")
	log.Printf("[%s] 发送下单请求: %s", signal.Symbol, string(reqJSON))

	// 执行订单
	var resp *api.OrderResponse
	err = api.RetryOperation(func() error {
		var placeErr error
		resp, placeErr = e.api.PlaceOrder(orderReq)
		return placeErr
	}, api.RetryConfig{
		MaxRetries:  3,
		DelayMillis: 1000,
	})

	if err != nil {
		log.Printf("[%s] 下单失败: %v", signal.Symbol, err)
		return err
	}

	log.Printf("[%s] 下单成功 - OrderID: %s", signal.Symbol, resp.OrderId)

	// 保存交易记录
	trade := &dbmodels.Trade{
		Symbol:    signal.Symbol,
		Side:      signal.Action,
		Price:     signal.Price,
		Amount:    signal.Amount,
		Strategy:  signal.Strategy,
		Status:    "filled",
		OrderID:   resp.OrderId,
		TradeType: e.config.TradeType,
		CreatedAt: time.Now(),
	}

	if err := e.db.SaveTrade(trade); err != nil {
		log.Printf("[%s] 保存交易记录失败: %v", signal.Symbol, err)
	}

	log.Printf("[%s] 交易完成: %s %.2f@%.2f",
		signal.Symbol, signal.Action, signal.Amount, signal.Price)
	return nil
}

func (e *Engine) updateMarketData(symbol string) {
	defer e.wg.Done()

	// 市场数据更新间隔
	marketTicker := time.NewTicker(1 * time.Second)
	// 止盈止损检查间隔
	pnlTicker := time.NewTicker(1 * time.Second)

	defer marketTicker.Stop()
	defer pnlTicker.Stop()

	log.Printf("[%s] 开始监控交易对，检查间隔: 5秒", symbol)

	// 记录上次开仓状态，避免重复开仓
	var lastLongEntry, lastShortEntry bool

	for {
		select {
		case <-e.stopChan:
			log.Printf("[%s] 停止监控交易对", symbol)
			return
		case <-marketTicker.C:
			// 获取最新价格
			candles, err := e.api.GetKlines(symbol, "1m", 1)
			if err != nil {
				log.Printf("[%s] 获取K线数据失败: %v", symbol, err)
				continue
			}

			if len(candles) == 0 {
				log.Printf("[%s] 未获取到K线数据", symbol)
				continue
			}

			price, _ := strconv.ParseFloat(candles[0].Close, 64)

			// 检查做多条件
			if e.config.LongPosition.Enabled && !lastLongEntry {
				if price >= e.config.LongPosition.EntryRange.Min &&
					price <= e.config.LongPosition.EntryRange.Max {
					// 触发做多信号
					signal := &types.Signal{
						Symbol:    symbol,
						Strategy:  "LongPosition",
						Action:    "buy",
						Price:     price,
						Amount:    float64(e.config.LongPosition.PositionSize),
						Timestamp: time.Now().Unix(),
					}
					log.Printf("[%s] 价格 %.4f 在做多区间内，触发做多信号", symbol, price)
					e.signals <- signal
					lastLongEntry = true
				}
			}

			// 检查做空条件
			if e.config.ShortPosition.Enabled && !lastShortEntry {
				if price >= e.config.ShortPosition.EntryRange.Min &&
					price <= e.config.ShortPosition.EntryRange.Max {
					// 触发做空信号
					signal := &types.Signal{
						Symbol:    symbol,
						Strategy:  "ShortPosition",
						Action:    "sell",
						Price:     price,
						Amount:    float64(e.config.ShortPosition.PositionSize),
						Timestamp: time.Now().Unix(),
					}
					log.Printf("[%s] 价格 %.4f 在做空区间内，触发做空信号", symbol, price)
					e.signals <- signal
					lastShortEntry = true
				}
			}

			// 重置开仓状态的条件
			if price < e.config.LongPosition.EntryRange.Min ||
				price > e.config.LongPosition.EntryRange.Max {
				lastLongEntry = false
			}
			if price < e.config.ShortPosition.EntryRange.Min ||
				price > e.config.ShortPosition.EntryRange.Max {
				lastShortEntry = false
			}

			// 继续执行现有的策略处理...
			for _, strategy := range e.strategies {
				signal, err := strategy.ProcessTick(&types.Tick{
					Symbol:    symbol,
					Price:     price,
					Timestamp: time.Now().Unix(),
				})
				if err != nil {
					log.Printf("[%s-%s] 策略处理失败: %v", symbol, strategy.Name(), err)
					continue
				}
				if signal != nil {
					e.signals <- signal
				}
			}
		case <-pnlTicker.C:
			// 检查止盈止损
			if err := e.checkPositionPnL(symbol); err != nil {
				log.Printf("[%s] 检查止盈止损失败: %v", symbol, err)
			}
		}
	}
}

// 修改 checkPositionPnL 方法使用 api.Position
func (e *Engine) checkPositionPnL(symbol string) error {
	positions, err := e.api.GetPositions(symbol)
	if err != nil {
		log.Printf("[%s] 获取持仓信息失败: %v", symbol, err)
		return fmt.Errorf("获取持仓信息失败: %v", err)
	}

	if len(positions) == 0 {
		log.Printf("[%s] 当前无持仓", symbol)
		return nil
	}

	for _, pos := range positions {
		log.Printf("[%s] 检查持仓: 方向=%s, 数量=%.4f, 收益率=%.2f%%",
			symbol, pos.PosSide, pos.Position, pos.PnLRatio*100)

		// 检查多头持仓
		if pos.PosSide == "long" && pos.Position > 0 {
			log.Printf("[%s] 多头持仓 - 止盈点=%.2f%%, 止损点=%.2f%%",
				symbol, e.config.LongPosition.TakeProfit*100, e.config.LongPosition.StopLoss*100)

			if pos.PnLRatio >= e.config.LongPosition.TakeProfit {
				log.Printf("[%s] 多头达到止盈点 %.2f%% >= %.2f%%, 执行平仓",
					symbol, pos.PnLRatio*100, e.config.LongPosition.TakeProfit*100)
				return e.closeLongPosition(symbol, pos)
			}
			if pos.PnLRatio <= -e.config.LongPosition.StopLoss {
				log.Printf("[%s] 多头达到止损点 %.2f%% <= -%.2f%%, 执行平仓",
					symbol, pos.PnLRatio*100, e.config.LongPosition.StopLoss*100)
				return e.closeLongPosition(symbol, pos)
			}
		}

		// 检查空头持仓
		if pos.PosSide == "short" && pos.Position > 0 {
			if pos.PnLRatio >= e.config.ShortPosition.TakeProfit {
				log.Printf("[%s] 空头达到止盈点 %.2f%%, 执行平仓", symbol, pos.PnLRatio*100)
				return e.closeShortPosition(symbol, pos)
			}
			if pos.PnLRatio <= -e.config.ShortPosition.StopLoss {
				log.Printf("[%s] 空头达到止损点 %.2f%%, 执行平仓", symbol, pos.PnLRatio*100)
				return e.closeShortPosition(symbol, pos)
			}
		}
	}
	return nil
}

// 修改平仓方法使用正确的 Position 类型
func (e *Engine) closeLongPosition(symbol string, pos *models.Position) error {
	// 确保数量为整数
	sz := int(math.Abs(pos.Position))
	if sz < 200 {
		sz = 200 // 确保至少为1
	}

	orderReq := &api.PlaceOrderRequest{
		InstId:  symbol,
		TdMode:  e.config.MarginMode,
		Side:    "sell",           // 平多需要卖出
		PosSide: "long",           // 平多仓
		OrdType: "market",         // 使用市价单
		Sz:      strconv.Itoa(sz), // 使用整数字符串
		ClOrdId: fmt.Sprintf("close%d", time.Now().UnixNano()/1000000),
	}

	log.Printf("[%s] 准备平多头仓位 - 订单参数: %+v", symbol, orderReq)
	resp, err := e.api.PlaceOrder(orderReq)
	if err != nil {
		log.Printf("[%s] 平多头仓位失败: %v", symbol, err)
		return fmt.Errorf("平多头仓位失败: %v", err)
	}

	log.Printf("[%s] 平多头仓位成功 - OrderID: %s, 数量: %d, 收益率: %.2f%%",
		symbol, resp.OrderId, sz, pos.PnLRatio*100)
	return nil
}

func (e *Engine) closeShortPosition(symbol string, pos *models.Position) error {
	// 确保数量为整数
	sz := int(math.Abs(pos.Position))
	if sz < 200 {
		sz = 200 // 确保至少为1
	}

	orderReq := &api.PlaceOrderRequest{
		InstId:  symbol,
		TdMode:  e.config.MarginMode,
		Side:    "buy",            // 平空需要买入
		PosSide: "short",          // 平空仓
		OrdType: "market",         // 使用市价单
		Sz:      strconv.Itoa(sz), // 使用整数字符串
		ClOrdId: fmt.Sprintf("close%d", time.Now().UnixNano()/1000000),
	}

	log.Printf("[%s] 准备平空头仓位 - 订单参数: %+v", symbol, orderReq)
	resp, err := e.api.PlaceOrder(orderReq)
	if err != nil {
		log.Printf("[%s] 平空头仓位失败: %v", symbol, err)
		return fmt.Errorf("平空头仓位失败: %v", err)
	}

	log.Printf("[%s] 平空头仓位成功 - OrderID: %s, 数量: %d, 收益率: %.2f%%",
		symbol, resp.OrderId, sz, pos.PnLRatio*100)
	return nil
}

// GetBalance 获取账户余额
func (e *Engine) GetBalance() ([]*api.Balance, error) {
	// 获取所有货币的余额
	return e.api.GetBalances()
}

// EnableStrategy 启用策略
func (e *Engine) EnableStrategy(name string) error {
	for _, strategy := range e.strategies {
		if strategy.Name() == name {
			return strategy.Initialize()
		}
	}
	return fmt.Errorf("策略不存在: %s", name)
}

// DisableStrategy 禁用策略
func (e *Engine) DisableStrategy(name string) error {
	for _, strategy := range e.strategies {
		if strategy.Name() == name {
			strategy.Stop()
			return nil
		}
	}
	return fmt.Errorf("策略不存在: %s", name)
}

// UpdateStrategyConfig 更新策略配置
func (e *Engine) UpdateStrategyConfig(name string, config map[string]interface{}) error {
	// 这里需要根据实际情况实现配置更新逻辑
	return fmt.Errorf("暂不支持更新策略配置")
}

// GetConfig 返回交易引擎配置
func (e *Engine) GetConfig() *Config {
	return e.config
}

// checkBalance 检查是否有足够的可用余额
func (e *Engine) checkBalance(requiredAmount float64) error {
	balances, err := e.api.GetBalances()
	if err != nil {
		return fmt.Errorf("获取余额失败: %v", err)
	}

	// 查找USDT余额
	var usdtBalance float64
	for _, balance := range balances {
		if balance.Currency == "USDT" {
			usdtBalance, err = strconv.ParseFloat(balance.Available, 64)
			if err != nil {
				return fmt.Errorf("解析USDT余额失败: %v", err)
			}
			break
		}
	}

	// 计算实际可用余额
	availableBalance := usdtBalance - e.config.ReserveBalance

	if availableBalance < requiredAmount {
		return fmt.Errorf("可用USDT余额不足: 需要 %.2f USDT, 实际可用 %.2f USDT (总余额: %.2f USDT, 预留: %.2f USDT)",
			requiredAmount, availableBalance, usdtBalance, e.config.ReserveBalance)
	}

	log.Printf("USDT余额充足: 需要 %.2f USDT, 实际可用 %.2f USDT (总余额: %.2f USDT, 预留: %.2f USDT)",
		requiredAmount, availableBalance, usdtBalance, e.config.ReserveBalance)
	return nil
}

// 修改计算保证金率的方法，直接使用持仓的保证金率，保持百分比形式
func (e *Engine) calculateMarginRatio(pos *models.Position) (float64, error) {
	// 直接使用持仓的保证金率，API返回的值需要乘以100
	marginRatio := pos.MarginRatio * 100

	log.Printf("保证金率计算 - 交易对: %s, 方向: %s, 持仓数量: %.4f, 持仓均价: %.4f, 保证金率: %.4f%%",
		pos.Symbol, pos.PosSide, pos.Position, pos.AvgPrice, marginRatio)

	return marginRatio, nil
}

// 新增保证金检查方法
func (e *Engine) checkAndAdjustMargin(symbol string) error {
	// 获取当前持仓
	positions, err := e.api.GetPositions(symbol)
	if err != nil {
		return fmt.Errorf("获取持仓失败: %v", err)
	}

	for _, pos := range positions {
		// 获取当前保证金率，API返回的值需要乘以100
		marginRatio := pos.MarginRatio * 100

		// 根据持仓方向判断使用多空配置
		var configRatio float64
		var autoMargin bool
		var marginAmount float64
		var symbolMarginRatio float64

		if pos.PosSide == "long" {
			// 检查是否有针对该交易对的特定保证金率配置
			if ratio, ok := e.config.LongPosition.SymbolMarginRatios[symbol]; ok {
				symbolMarginRatio = ratio * 100
				log.Printf("使用交易对 %s 的特定做多保证金率配置: %.2f%%", symbol, ratio*100)
			} else {
				symbolMarginRatio = e.config.LongPosition.MarginRatio * 100
				log.Printf("使用默认做多保证金率配置: %.2f%%", symbolMarginRatio)
			}
			configRatio = symbolMarginRatio
			autoMargin = e.config.LongPosition.AutoMargin
			marginAmount = e.config.LongPosition.MarginAmount
		} else {
			// 检查是否有针对该交易对的特定保证金率配置
			if ratio, ok := e.config.ShortPosition.SymbolMarginRatios[symbol]; ok {
				symbolMarginRatio = ratio * 100
				log.Printf("使用交易对 %s 的特定做空保证金率配置: %.2f%%", symbol, ratio*100)
			} else {
				symbolMarginRatio = e.config.ShortPosition.MarginRatio * 100
				log.Printf("使用默认做空保证金率配置: %.2f%%", symbolMarginRatio)
			}
			configRatio = symbolMarginRatio
			autoMargin = e.config.ShortPosition.AutoMargin
			marginAmount = e.config.ShortPosition.MarginAmount
		}

		log.Printf("%s %s仓位当前保证金率: %.4f%%, 配置保证金率: %.4f%%",
			symbol, pos.PosSide, marginRatio, configRatio)

		// 检查是否需要追加保证金
		if marginRatio < configRatio {
			if !autoMargin {
				log.Printf("警告: %s %s仓位保证金率(%.4f%%)低于设定值(%.4f%%)",
					symbol, pos.PosSide, marginRatio, configRatio)
				continue
			}

			// 使用配置的固定保证金数量
			addAmount := marginAmount

			// 追加保证金
			err = e.addMargin(symbol, pos.PosSide, addAmount)
			if err != nil {
				return fmt.Errorf("追加保证金失败: %v", err)
			}

			log.Printf("%s %s仓位追加保证金 %.4f USDT, 保证金率从 %.4f%% 提升至 %.4f%%",
				symbol, pos.PosSide, addAmount, marginRatio, configRatio)
		} else {
			log.Printf("%s %s仓位保证金率 %.4f%% 高于设定值 %.4f%%, 无需追加保证金",
				symbol, pos.PosSide, marginRatio, configRatio)
		}
	}

	return nil
}

// 追加保证金
func (e *Engine) addMargin(symbol, posSide string, amount float64) error {
	// 检查可用余额
	if err := e.checkBalance(amount); err != nil {
		return err
	}

	// 调用API追加保证金
	params := map[string]string{
		"instId":  symbol,
		"posSide": posSide,
		"amt":     strconv.FormatFloat(amount, 'f', 4, 64),
		"type":    "add",
	}

	_, err := e.api.AddMargin(params)
	return err
}
