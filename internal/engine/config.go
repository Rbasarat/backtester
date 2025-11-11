package engine

import (
	"backtester/types"
	"time"

	"github.com/shopspring/decimal"
)

type DataFeedConfig struct {
	Ticker   string
	Interval types.Interval
	Start    time.Time
	End      time.Time
	candles  []types.Candle
}

func NewDataFeedConfig(feeds ...DataFeedConfig) []*DataFeedConfig {
	var dataFeeds []*DataFeedConfig
	for _, feed := range feeds {
		dataFeeds = append(dataFeeds, &feed)
	}
	return dataFeeds
}

type PortfolioConfig struct {
	InitialCash decimal.Decimal
}

func NewPortfolioConfig(initialCash decimal.Decimal) *PortfolioConfig {
	return &PortfolioConfig{
		InitialCash: initialCash,
	}
}

type ExecutionConfig struct {
	Interval   types.Interval
	BarsBefore int
	BarsAfter  int
	candles    map[string][]types.Candle
}

func NewExecutionConfig(executionInterval types.Interval, barsBefore, barsAfter int) ExecutionConfig {
	return ExecutionConfig{
		Interval:   executionInterval,
		BarsBefore: barsBefore,
		BarsAfter:  barsAfter,
		candles:    make(map[string][]types.Candle),
	}
}
