package engine

import (
	"backtester/internal/repository"
	"backtester/types"
	"context"
	"time"
)

type DataFeed struct {
	Ticker   string
	Interval types.Interval
	Start    time.Time
	End      time.Time
}

func (df *DataFeed) GetData(db repository.Database) ([]types.Candle, error) {
	ctx := context.Background()
	ticker, err := db.GetAssetByTicker(df.Ticker, ctx)
	if err != nil {
		return nil, err
	}
	candles, err := db.GetCandles(ticker.Id, df.Interval, df.Start, df.End, ctx)
	if err != nil {
		return nil, err
	}
	return candles, nil
}
