package engine

import (
	"backtester/types"
	"github.com/shopspring/decimal"
	"time"
)

type DataFeedConfig struct {
	Ticker   string
	Interval types.Interval
	Start    time.Time
	End      time.Time
	candles  []types.Candle
}

func NewDataFeed(ticker string, interval types.Interval, start, end time.Time) *DataFeedConfig {
	return &DataFeedConfig{
		Ticker:   ticker,
		Interval: interval,
		Start:    start,
		End:      end,
		candles:  []types.Candle{},
	}
}

type PortfolioConfig struct {
	InitialCash decimal.Decimal
}

func NewPortfolioConfig(initialCash decimal.Decimal) *PortfolioConfig {
	return &PortfolioConfig{
		InitialCash: initialCash,
	}
}
