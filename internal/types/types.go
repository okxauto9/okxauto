package types

// 交易信号
type Signal struct {
	Symbol    string
	Strategy  string
	Action    string
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

// 策略接口
type Strategy interface {
	Name() string
	Initialize() error
	ProcessTick(tick *Tick) (*Signal, error)
	Stop()
} 
 