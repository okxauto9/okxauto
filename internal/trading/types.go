package trading

import (
	"okxauto/internal/types"
)

// 策略接口
type Strategy interface {
	Name() string
	Initialize() error
	ProcessTick(tick *types.Tick) (*types.Signal, error)
	Stop()
}

// 交易信号
type Signal struct {
	Symbol    string
	Strategy  string
	Action    string  // "buy" or "sell"
	Price     float64
	Amount    float64
	Timestamp int64
}

// 行情数据
type Tick struct {
	Symbol    string
	Price     float64
	Volume    float64
	Timestamp int64
}

// 策略配置
type StrategyConfig struct {
	Enabled bool
	Symbol  string
}

// Config 定义交易引擎配置
type Config struct {
	Mode           string   `yaml:"mode"`        // simulation or live
	TradeType      string   `yaml:"trade_type"`  // spot或futures
	Leverage       int      `yaml:"leverage"`    // 合约杠杆倍数
	MarginMode     string   `yaml:"margin_mode"` // 合约保证金模式
	ReserveBalance float64  `yaml:"reserve_balance"` // 添加预留余额字段
	Symbols        []string `yaml:"symbols"`

	// 添加做多配置
	LongPosition struct {
		Enabled     bool    `yaml:"enabled"`
		EntryRange struct {
			Min float64 `yaml:"min"`
			Max float64 `yaml:"max"`
		} `yaml:"entry_range"`
		TakeProfit   float64 `yaml:"take_profit"`
		StopLoss     float64 `yaml:"stop_loss"`
		PositionSize int     `yaml:"position_size"`
		MarginRatio  float64 `yaml:"margin_ratio"`
		AutoMargin   bool    `yaml:"auto_margin"`
		MarginAmount float64 `yaml:"margin_amount"`
		SymbolMarginRatios map[string]float64 `yaml:"symbol_margin_ratios"`
	} `yaml:"long_position"`

	// 添加做空配置
	ShortPosition struct {
		Enabled     bool    `yaml:"enabled"`
		EntryRange struct {
			Min float64 `yaml:"min"`
			Max float64 `yaml:"max"`
		} `yaml:"entry_range"`
		TakeProfit   float64 `yaml:"take_profit"`
		StopLoss     float64 `yaml:"stop_loss"`
		PositionSize int     `yaml:"position_size"`
		MarginRatio  float64 `yaml:"margin_ratio"`
		AutoMargin   bool    `yaml:"auto_margin"`
		MarginAmount float64 `yaml:"margin_amount"`
		SymbolMarginRatios map[string]float64 `yaml:"symbol_margin_ratios"`
	} `yaml:"short_position"`

	Grid struct {
		Enabled     bool    `yaml:"enabled"`
		UpperPrice  float64 `yaml:"upper_price"`
		LowerPrice  float64 `yaml:"lower_price"`
		GridNumber  int     `yaml:"grid_number"`
		TotalAmount float64 `yaml:"total_amount"`
	} `yaml:"grid_strategy"`

	RSI struct {
		Enabled             bool    `yaml:"enabled"`
		Period              int     `yaml:"period"`
		OverboughtThreshold float64 `yaml:"overbought_threshold"`
		OversoldThreshold   float64 `yaml:"oversold_threshold"`
	} `yaml:"rsi_strategy"`
}

// Position 持仓信息
type Position struct {
	Symbol    string  `json:"symbol"`
	PosSide   string  `json:"posSide"`   // long/short
	Position  float64 `json:"pos"`        // 持仓数量
	AvgPrice  float64 `json:"avgPx"`      // 开仓均价
	UnrealPnL float64 `json:"upl"`        // 未实现盈亏
	PnLRatio  float64 `json:"uplRatio"`   // 收益率
} 