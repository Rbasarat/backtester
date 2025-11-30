package repository

import (
	sqlc "backtester/internal/repository/sqlc/generated"
	"backtester/types"
	"context"
	"database/sql"
	"errors"
	"time"
)

var bucketToInterval = map[types.Interval]string{
	types.OneMinute:     "1 minute",
	types.FiveMinutes:   "5 minutes",
	types.ThirtyMinutes: "30 minute",
	types.Hour:          "1 hour",
	types.FourHours:     "4 hours",
	types.Day:           "1 day",
	types.Week:          "1 week",
}

func (db *Database) GetAggregates(assetId int, ticker string, interval types.Interval, start, end time.Time, ctx context.Context) ([]types.Candle, error) {
	bucket, ok := bucketToInterval[interval]
	if !ok {
		return nil, ErrIntervalNotSupported
	}
	args := sqlc.GetAggregatesParams{
		TimeBucket: bucket,
		AssetID:    int32(assetId),
		Starttime:  &start,
		Endtime:    &end,
	}
	candles, err := db.candles.GetAggregates(ctx, args)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNoCandles
		}
		return nil, err
	}
	if len(candles) == 0 {
		return nil, ErrNoCandles
	}
	return convertCandles(candles, interval, ticker), nil
}

func convertCandles(candleDAOs []sqlc.GetAggregatesRow, interval types.Interval, ticker string) []types.Candle {
	var candles []types.Candle
	for _, dao := range candleDAOs {
		candles = append(candles, types.Candle{
			AssetId:   int(dao.AssetID),
			Ticker:    ticker,
			Open:      dao.Open,
			Close:     dao.Close,
			High:      dao.High,
			Low:       dao.Low,
			Volume:    dao.Volume,
			Interval:  interval,
			Timestamp: *dao.Bucket,
		})
	}
	return candles
}
