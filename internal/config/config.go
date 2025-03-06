package config

import (
	"fmt"
	"io/ioutil"

	"okxauto/internal/server"

	"gopkg.in/yaml.v2"
)

type Config struct {
	API struct {
		Key        string `yaml:"key"`
		Secret     string `yaml:"secret"`
		Passphrase string `yaml:"passphrase"`
		Mode       string `yaml:"mode"`
	} `yaml:"api"`

	Database struct {
		Path string `yaml:"path"`
	} `yaml:"database"`

	Trading struct {
		Mode       string   `yaml:"mode"`
		TradeType  string   `yaml:"trade_type"`
		Leverage   int      `yaml:"leverage"`
		MarginMode string   `yaml:"margin_mode"`
		Symbols    []string `yaml:"symbols"`
		Grid      struct {
			Enabled     bool    `yaml:"enabled"`
			UpperPrice  float64 `yaml:"upper_price"`
			LowerPrice  float64 `yaml:"lower_price"`
			GridNumber  int     `yaml:"grid_number"`
			TotalAmount float64 `yaml:"total_amount"`
		} `yaml:"grid_strategy"`
		RSI struct {
			Enabled            bool    `yaml:"enabled"`
			Period             int     `yaml:"period"`
			OverboughtThreshold float64 `yaml:"overbought_threshold"`
			OversoldThreshold  float64 `yaml:"oversold_threshold"`
		} `yaml:"rsi_strategy"`
	} `yaml:"trading"`

	Server server.Config `yaml:"server"`
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