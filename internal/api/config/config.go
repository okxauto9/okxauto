package config

import (
	"okxauto/internal/server"

	"gopkg.in/yaml.v2"
	"os"
)

type Trading struct {
	Mode           string   `yaml:"mode"`
	TradeType      string   `yaml:"trade_type"`
	Leverage       int      `yaml:"leverage"`
	MarginMode     string   `yaml:"margin_mode"`
	ReserveBalance float64  `yaml:"reserve_balance"`
	Symbols        []string `yaml:"symbols"`

	LongPosition struct {
		Enabled     bool    `yaml:"enabled"`
		EntryRange struct {
			Min float64 `yaml:"min"`
			Max float64 `yaml:"max"`
		} `yaml:"entry_range"`
		TakeProfit   float64            `yaml:"take_profit"`
		StopLoss     float64            `yaml:"stop_loss"`
		PositionSize int                `yaml:"position_size"`
		MarginRatio  float64            `yaml:"margin_ratio"`
		AutoMargin   bool               `yaml:"auto_margin"`
		MarginAmount float64            `yaml:"margin_amount"`
		SymbolMarginRatios map[string]float64 `yaml:"symbol_margin_ratios"`
	} `yaml:"long_position"`

	ShortPosition struct {
		Enabled     bool    `yaml:"enabled"`
		EntryRange struct {
			Min float64 `yaml:"min"`
			Max float64 `yaml:"max"`
		} `yaml:"entry_range"`
		TakeProfit   float64            `yaml:"take_profit"`
		StopLoss     float64            `yaml:"stop_loss"`
		PositionSize int                `yaml:"position_size"`
		MarginRatio  float64            `yaml:"margin_ratio"`
		AutoMargin   bool               `yaml:"auto_margin"`
		MarginAmount float64            `yaml:"margin_amount"`
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

type Config struct {
	API struct {
		Key        string `yaml:"key"`
		Secret     string `yaml:"secret"`
		Passphrase string `yaml:"passphrase"`
		Mode       string `yaml:"mode"`
		BaseURL    string `yaml:"base_url"`
	} `yaml:"api"`

	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`

	Trading Trading `yaml:"trading"`
	Server  server.Config `yaml:"server"`
}

func Load(configFile string) (*Config, error) {
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
} 