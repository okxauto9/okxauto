package strategies

import (
	"fmt"
	"log"
	"math"
	"strconv"
	"sync"
	"time"

	"okxauto/internal/api"
	"okxauto/internal/types"
)

type RSIStrategy struct {
	api         *api.OKXClient
	symbol      string
	config      RSIConfig
	prices      []float64
	lastRSI     float64    // 记录上一次的RSI值
	signalCount int        // 信号计数器
	mu          sync.RWMutex
}

type RSIConfig struct {
	Enabled             bool    `yaml:"enabled"`
	Period              int     `yaml:"period"`              // RSI周期
	OverboughtThreshold float64 `yaml:"overbought_threshold"` // 超买阈值
	OversoldThreshold   float64 `yaml:"oversold_threshold"`   // 超卖阈值
	SignalConfirmation  int     `yaml:"signal_confirmation"`  // 信号确认次数
	MinChange           float64 `yaml:"min_change"`           // 最小变化幅度
}

func NewRSIStrategy(api *api.OKXClient, symbol string, config RSIConfig) *RSIStrategy {
	return &RSIStrategy{
		api:     api,
		symbol:  symbol,
		config:  config,
		prices:  make([]float64, 0, config.Period*3), // 预分配3倍周期的容量
	}
}

func (s *RSIStrategy) Name() string {
	return "RSI"
}

// Initialize 初始化策略,获取足够的历史数据
func (s *RSIStrategy) Initialize() error {
	log.Printf("[RSI-%s] 开始初始化策略...", s.symbol)
	
	// 获取足够的历史数据用于计算RSI
	lookback := s.config.Period * 3 // 获取3倍周期的数据
	
	// 使用K线数据API获取历史数据
	// 注意：这里不再需要endTime变量，因为我们默认获取最新的数据
	candles, err := s.api.GetKlines(s.symbol, "1m", lookback)
	if err != nil {
		return fmt.Errorf("获取K线数据失败: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// 清空旧数据
	s.prices = s.prices[:0]
	
	// 处理K线数据
	for _, candle := range candles {
		price, err := strconv.ParseFloat(candle.Close, 64)
		if err != nil {
			log.Printf("[RSI-%s] 解析价格数据失败: %v", s.symbol, err)
			continue
		}
		s.prices = append(s.prices, price)
	}

	// 计算初始RSI值
	if len(s.prices) >= s.config.Period {
		s.lastRSI = s.calculateRSI()
		log.Printf("[RSI-%s] 初始RSI值: %.2f", s.symbol, s.lastRSI)
	}

	log.Printf("[RSI-%s] 策略初始化完成,历史数据数量: %d", s.symbol, len(s.prices))
	return nil
}

// ProcessTick 处理新的价格数据
func (s *RSIStrategy) ProcessTick(tick *types.Tick) (*types.Signal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("[RSI-%s] 当前价格: %.2f", s.symbol, tick.Price)

	// 更新价格数据
	s.prices = append(s.prices, tick.Price)
	if len(s.prices) > s.config.Period*3 {
		s.prices = s.prices[1:]
	}

	// 如果价格数据不足,等待更多数据
	if len(s.prices) < s.config.Period {
		log.Printf("[RSI-%s] 等待更多价格数据: %d/%d", 
			s.symbol, len(s.prices), s.config.Period)
		return nil, nil
	}

	// 计算当前RSI值
	currentRSI := s.calculateRSI()
	
	// 计算RSI变化
	rsiChange := currentRSI - s.lastRSI
	s.lastRSI = currentRSI

	log.Printf("[RSI-%s] 当前RSI: %.2f, 变化: %.2f", s.symbol, currentRSI, rsiChange)

	// 生成交易信号
	var signal *types.Signal

	// 超买条件检查
	if currentRSI >= s.config.OverboughtThreshold {
		if math.Abs(rsiChange) >= s.config.MinChange {
			s.signalCount++
			if s.signalCount >= s.config.SignalConfirmation {
				signal = &types.Signal{
					Symbol:    s.symbol,
					Strategy:  s.Name(),
					Action:    "sell",
					Price:     tick.Price,
					Amount:    1, // 使用配置中的仓位大小
					Timestamp: time.Now().Unix(),
				}
				log.Printf("[RSI-%s] 触发卖出信号 - RSI: %.2f, 价格: %.2f", 
					s.symbol, currentRSI, tick.Price)
				s.signalCount = 0
			}
		}
	} else if currentRSI <= s.config.OversoldThreshold {
		// 超卖条件检查
		if math.Abs(rsiChange) >= s.config.MinChange {
			s.signalCount++
			if s.signalCount >= s.config.SignalConfirmation {
				signal = &types.Signal{
					Symbol:    s.symbol,
					Strategy:  s.Name(),
					Action:    "buy",
					Price:     tick.Price,
					Amount:    1, // 使用配置中的仓位大小
					Timestamp: time.Now().Unix(),
				}
				log.Printf("[RSI-%s] 触发买入信号 - RSI: %.2f, 价格: %.2f", 
					s.symbol, currentRSI, tick.Price)
				s.signalCount = 0
			}
		}
	} else {
		// RSI在正常范围内,重置信号计数
		s.signalCount = 0
	}

	return signal, nil
}

// calculateRSI 使用Wilder's RSI计算方法
func (s *RSIStrategy) calculateRSI() float64 {
	if len(s.prices) < s.config.Period+1 {
		return 50.0
	}

	var avgGain, avgLoss float64
	
	// 计算第一个平均涨跌幅
	for i := 1; i <= s.config.Period; i++ {
		change := s.prices[i] - s.prices[i-1]
		if change >= 0 {
			avgGain += change
		} else {
			avgLoss -= change
		}
	}
	
	avgGain = avgGain / float64(s.config.Period)
	avgLoss = avgLoss / float64(s.config.Period)

	// 使用Wilder's平滑方法计算后续值
	for i := s.config.Period + 1; i < len(s.prices); i++ {
		change := s.prices[i] - s.prices[i-1]
		if change >= 0 {
			avgGain = (avgGain*float64(s.config.Period-1) + change) / float64(s.config.Period)
			avgLoss = (avgLoss*float64(s.config.Period-1)) / float64(s.config.Period)
		} else {
			avgGain = (avgGain*float64(s.config.Period-1)) / float64(s.config.Period)
			avgLoss = (avgLoss*float64(s.config.Period-1) - change) / float64(s.config.Period)
		}
	}

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	return 100.0 - (100.0 / (1.0 + rs))
}

func (s *RSIStrategy) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.prices = nil
	s.signalCount = 0
	log.Printf("[RSI-%s] 策略已停止", s.symbol)
} 