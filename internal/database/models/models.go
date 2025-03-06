package models

import "time"

// User 用户信息
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Trade 交易记录
type Trade struct {
	ID        int64     `db:"id"`
	Symbol    string    `db:"symbol"`     // 交易对
	Side      string    `db:"side"`       // 买卖方向
	Price     float64   `db:"price"`      // 价格
	Amount    float64   `db:"amount"`     // 数量
	Strategy  string    `db:"strategy"`   // 策略名称
	Status    string    `db:"status"`     // 状态
	OrderID   string    `db:"order_id"`   // 订单ID
	TradeType string    `db:"trade_type"` // 交易类型：spot/futures
	CreatedAt time.Time `db:"created_at"` // 创建时间
}

// Signal 交易信号
type Signal struct {
	ID        int64     `json:"id"`
	Symbol    string    `json:"symbol"`
	Strategy  string    `json:"strategy"`
	Action    string    `json:"action"`
	Price     float64   `json:"price"`
	CreatedAt time.Time `json:"created_at"`
} 