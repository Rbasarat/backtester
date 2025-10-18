package repository

import (
	sqlc "backtester/internal/repository/sqlc/generated"
	"backtester/types"
	"context"
	"database/sql"
	"errors"
	"github.com/shopspring/decimal"
	"testing"
	"time"
)

var testInterval = types.OneMinute
var startTime = time.UnixMilli(0)
var endTime = startTime.Add(time.Minute * 5)

type mockCandlesRepository struct {
	sqlError error
	candles  []sqlc.GetAggregatesRow
}

func TestDatabase_GetCandles(t *testing.T) {
	type args struct {
		assetId  int
		interval types.Interval
		start    time.Time
		end      time.Time
	}
	tests := []struct {
		name    string
		args    args
		want    []types.Candle
		sqlErr  error
		wantErr error
	}{
		{"should throw ErrCandlesNotFound", args{999, testInterval, startTime, endTime}, nil, nil, ErrNoCandles},
		{"should throw ErrCandlesNotFound", args{999, testInterval, startTime, endTime}, nil, sql.ErrNoRows, ErrNoCandles},
		{"should throw ErrIntervalNotSupported", args{999, types.Month, startTime, endTime}, nil, nil, ErrIntervalNotSupported},
		{"should return candles", args{999, testInterval, startTime, endTime}, mockCandles(999, startTime, endTime), nil, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := &Database{
				candles: mockCandlesRepository{
					sqlError: tt.sqlErr,
				},
			}
			got, err := db.GetCandles(tt.args.assetId, tt.args.interval, tt.args.start, tt.args.end, context.Background())

			if err != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("GetCandles() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			for i := 0; i < len(tt.want); i++ {
				if got[i].AssetId != tt.args.assetId {
					t.Errorf("GetCandles() %s assetId got = %v, want %v", got[i].Timestamp, got[i].AssetId, tt.want[i].AssetId)
					break
				}
				if got[i].Interval != tt.args.interval {
					t.Errorf("GetCandles() %s interval got = %v, want %v", got[i].Timestamp, got[i].Interval, tt.want[i].Interval)
					break
				}
				if !got[i].High.Equal(tt.want[i].High) {
					t.Errorf("GetCandles() %s high got = %v, want %v", got[i].Timestamp, got[i].High, tt.want[i].High)
					break
				}
			}
		})
	}
}

func (m mockCandlesRepository) GetAggregates(_ context.Context, arg sqlc.GetAggregatesParams) ([]sqlc.GetAggregatesRow, error) {
	if m.sqlError != nil {
		return []sqlc.GetAggregatesRow{}, m.sqlError
	}
	var candles []sqlc.GetAggregatesRow
	i := *arg.Starttime
	for i.Before(*arg.Endtime) {
		candles = append(candles, sqlc.GetAggregatesRow{
			Bucket:  &i,
			AssetID: arg.AssetID,
			Open:    decimal.NewFromInt(i.UnixMilli()),
			High:    decimal.NewFromInt(i.UnixMilli()),
			Low:     decimal.NewFromInt(i.UnixMilli()),
			Close:   decimal.NewFromInt(i.UnixMilli()),
			Volume:  decimal.NewFromInt(i.UnixMilli()),
		})
		i = i.Add(types.IntervalToTime[testInterval])
	}
	return candles, nil
}

func mockCandles(assetId int, start, end time.Time) []types.Candle {
	var candles []types.Candle
	i := start
	for i.Before(end) {
		candles = append(candles, types.Candle{
			Timestamp: i,
			Interval:  testInterval,
			AssetId:   assetId,
			Open:      decimal.NewFromInt(i.UnixMilli()),
			High:      decimal.NewFromInt(i.UnixMilli()),
			Low:       decimal.NewFromInt(i.UnixMilli()),
			Close:     decimal.NewFromInt(i.UnixMilli()),
			Volume:    decimal.NewFromInt(i.UnixMilli()),
		})
		i = i.Add(types.IntervalToTime[testInterval])
	}
	return candles
}
