package strategies

import (
	"log"
	"sync"
	"time"

	"okxauto/internal/api"
	"okxauto/internal/types"
)

type GridStrategy struct {
	api         *api.OKXClient
	symbol      string
	config      GridConfig
	gridLevels []float64
	positions   map[float64]float64
	mu          sync.RWMutex
}

type GridConfig struct {
	Enabled     bool    `yaml:"enabled"`
	UpperPrice  float64 `yaml:"upper_price"`
	LowerPrice  float64 `yaml:"lower_price"`
	GridNumber  int     `yaml:"grid_number"`
	TotalAmount float64 `yaml:"total_amount"`
}

func NewGridStrategy(api *api.OKXClient, symbol string, config GridConfig) *GridStrategy {
	return &GridStrategy{
		api:        api,
		symbol:     symbol,
		config:     config,
		positions:  make(map[float64]float64),
	}
}

func (s *GridStrategy) Name() string {
	return "Grid"
}

func (s *GridStrategy) Initialize() error {
	s.gridLevels = s.calculateGridLevels()
	return nil
}

func (s *GridStrategy) ProcessTick(tick *types.Tick) (*types.Signal, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	currentPrice := tick.Price
	log.Printf("[Grid-%s] 分析价格: %.2f", s.symbol, currentPrice)

	// 检查价格是否在网格范围内
	if currentPrice < s.config.LowerPrice {
		log.Printf("[Grid-%s] 价格(%.2f)低于网格下限(%.2f), 等待价格回升", 
			s.symbol, currentPrice, s.config.LowerPrice)
		return nil, nil
	}
	if currentPrice > s.config.UpperPrice {
		log.Printf("[Grid-%s] 价格(%.2f)高于网格上限(%.2f), 等待价格回落", 
			s.symbol, currentPrice, s.config.UpperPrice)
		return nil, nil
	}

	// 找到当前价格所在的网格
	gridIndex := -1
	for i := 0; i < len(s.gridLevels)-1; i++ {
		if currentPrice >= s.gridLevels[i] && currentPrice < s.gridLevels[i+1] {
			gridIndex = i
			break
		}
	}

	if gridIndex == -1 {
		log.Printf("[Grid-%s] 价格(%.2f)未落在任何网格中", s.symbol, currentPrice)
		return nil, nil
	}

	log.Printf("[Grid-%s] 价格(%.2f)位于网格%d [%.2f - %.2f]", 
		s.symbol, currentPrice, gridIndex, s.gridLevels[gridIndex], s.gridLevels[gridIndex+1])

	signal := s.checkGridSignal(currentPrice)
	if signal != nil {
		log.Printf("[Grid-%s] 触发%s信号: 价格=%.2f, 数量=%.2f", 
			s.symbol, signal.Action, signal.Price, signal.Amount)
	} else {
		log.Printf("[Grid-%s] 当前价格未触发交易信号", s.symbol)
	}
	
	return signal, nil
}

func (s *GridStrategy) Stop() {
	// 清理所有持仓
}

func (s *GridStrategy) calculateGridLevels() []float64 {
	levels := make([]float64, s.config.GridNumber+1)
	interval := (s.config.UpperPrice - s.config.LowerPrice) / float64(s.config.GridNumber)
	
	for i := 0; i <= s.config.GridNumber; i++ {
		levels[i] = s.config.LowerPrice + float64(i)*interval
	}
	
	return levels
}

func (s *GridStrategy) checkGridSignal(currentPrice float64) *types.Signal {
	for i := 0; i < len(s.gridLevels)-1; i++ {
		lower := s.gridLevels[i]
		upper := s.gridLevels[i+1]
		
		if currentPrice >= lower && currentPrice < upper {
			// 合约交易使用张数，最小为1张
			gridAmount := 1 // 固定为1张合约
			gridRange := upper - lower
			
			// 价格接近下边界，产生买入信号
			if currentPrice-lower < gridRange*0.1 {
				log.Printf("[Grid-%s] 价格(%.2f)接近网格下边界(%.2f), 考虑买入", 
					s.symbol, currentPrice, lower)
				return &types.Signal{
					Symbol:    s.symbol,
					Strategy:  s.Name(),
					Action:    "buy",
					Price:    currentPrice,
					Amount:   float64(gridAmount), // 使用整数张数
					Timestamp: time.Now().Unix(),
				}
			}
			
			// 价格接近上边界，产生卖出信号
			if upper-currentPrice < gridRange*0.1 {
				log.Printf("[Grid-%s] 价格(%.2f)接近网格上边界(%.2f), 考虑卖出", 
					s.symbol, currentPrice, upper)
				return &types.Signal{
					Symbol:    s.symbol,
					Strategy:  s.Name(),
					Action:    "sell",
					Price:    currentPrice,
					Amount:   float64(gridAmount), // 使用整数张数
					Timestamp: time.Now().Unix(),
				}
			}
		}
	}
	
	return nil
} 