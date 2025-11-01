package engine

import (
	"backtester/types"
	"time"
)

type DataFeed struct {
	Ticker   string
	Interval types.Interval
	Start    time.Time
	End      time.Time
	candles  []types.Candle
}

func NewDataFeed(ticker string, interval types.Interval, start, end time.Time) *DataFeed {
	return &DataFeed{
		Ticker:   ticker,
		Interval: interval,
		Start:    start,
		End:      end,
		candles:  []types.Candle{},
	}
}
