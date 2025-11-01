package engine

import (
	"backtester/types"
	"context"
	"github.com/shopspring/decimal"
	"reflect"
	"testing"
	"time"
)

func TestBacktest_AllFeedsSendSameTimestampPerTick(t *testing.T) {
	base := time.UnixMilli(0).UTC()
	newTime := func(i int) time.Time { return base.Add(time.Duration(i) * time.Minute) }
	tests := []struct {
		name        string
		feeds       []*DataFeed
		wantCandles map[time.Time][]types.Candle
	}{
		{
			name: "feed with no candles",
			feeds: []*DataFeed{
				{Ticker: "A", Start: newTime(0), End: newTime(2), candles: nil},
			},
			wantCandles: map[time.Time][]types.Candle{},
		},
		{
			name: "all feeds send same timestamp per tick",
			feeds: []*DataFeed{
				{
					Ticker: "A",
					Start:  newTime(0),
					End:    newTime(2),
					candles: []types.Candle{
						mockCandle(1, newTime(0)),
						mockCandle(1, newTime(1)),
						mockCandle(1, newTime(2))},
				},
			},
			wantCandles: map[time.Time][]types.Candle{
				newTime(0): {mockCandle(1, newTime(0))},
				newTime(1): {mockCandle(1, newTime(1))},
				newTime(2): {mockCandle(1, newTime(2))},
			},
		},
		{
			name: "one feed is subset of max range",
			feeds: []*DataFeed{
				{
					Ticker: "A",
					Start:  newTime(0),
					End:    newTime(2),
					candles: []types.Candle{
						mockCandle(1, newTime(0)),
						mockCandle(1, newTime(1)),
						mockCandle(1, newTime(2))},
				},
				{
					Ticker: "B",
					Start:  newTime(1),
					End:    newTime(1),
					candles: []types.Candle{
						mockCandle(1, newTime(1))},
				},
			},
			wantCandles: map[time.Time][]types.Candle{
				newTime(0): {mockCandle(1, newTime(0))},
				newTime(1): {mockCandle(1, newTime(1)), mockCandle(2, newTime(1))},
				newTime(2): {mockCandle(1, newTime(2))},
			},
		},
		{
			name: "two feeds send same timestamp per tick",
			feeds: []*DataFeed{
				{
					Ticker: "A",
					Start:  newTime(0),
					End:    newTime(2),
					candles: []types.Candle{
						mockCandle(1, newTime(0)),
						mockCandle(1, newTime(1)),
						mockCandle(1, newTime(2))},
				},
				{
					Ticker: "B",
					Start:  newTime(0),
					End:    newTime(2),
					candles: []types.Candle{
						mockCandle(2, newTime(0)),
						mockCandle(2, newTime(1)),
						mockCandle(2, newTime(2))},
				},
			},
			wantCandles: map[time.Time][]types.Candle{
				newTime(0): {mockCandle(1, newTime(0)), mockCandle(2, newTime(0))},
				newTime(1): {mockCandle(1, newTime(1)), mockCandle(2, newTime(1))},
				newTime(2): {mockCandle(1, newTime(2)), mockCandle(2, newTime(2))},
			},
		},
		{
			name: "irregular intervals vs dense feed",
			feeds: []*DataFeed{
				{Ticker: "A", Start: newTime(0), End: newTime(5),
					candles: []types.Candle{
						mockCandle(1, newTime(0)), mockCandle(1, newTime(3)), mockCandle(1, newTime(5)),
					},
				},
				{Ticker: "B", Start: newTime(0), End: newTime(5),
					candles: []types.Candle{
						mockCandle(2, newTime(0)), mockCandle(2, newTime(1)), mockCandle(2, newTime(2)),
						mockCandle(2, newTime(3)), mockCandle(2, newTime(4)), mockCandle(2, newTime(5)),
					},
				},
			},
			wantCandles: map[time.Time][]types.Candle{
				newTime(0): {mockCandle(1, newTime(0)), mockCandle(2, newTime(0))},
				newTime(1): {mockCandle(2, newTime(1))},
				newTime(2): {mockCandle(2, newTime(2))},
				newTime(3): {mockCandle(1, newTime(3)), mockCandle(2, newTime(3))},
				newTime(4): {mockCandle(2, newTime(4))},
				newTime(5): {mockCandle(1, newTime(5)), mockCandle(2, newTime(5))},
			},
		},
		{
			name: "one feed overlaps the other",
			feeds: []*DataFeed{
				{
					Ticker: "A",
					Start:  newTime(0),
					End:    newTime(2),
					candles: []types.Candle{
						mockCandle(1, newTime(0)),
						mockCandle(1, newTime(1)),
						mockCandle(1, newTime(2))},
				},
				{
					Ticker: "B",
					Start:  newTime(1),
					End:    newTime(3),
					candles: []types.Candle{
						mockCandle(2, newTime(1)),
						mockCandle(2, newTime(2)),
						mockCandle(2, newTime(3))},
				},
			},
			wantCandles: map[time.Time][]types.Candle{
				newTime(0): {mockCandle(1, newTime(0))},
				newTime(1): {mockCandle(1, newTime(1)), mockCandle(2, newTime(1))},
				newTime(2): {mockCandle(1, newTime(2)), mockCandle(2, newTime(2))},
				newTime(3): {mockCandle(2, newTime(3))},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sp := newCandlesParallelismStrategy()
			engine := mockEngine(sp, tt.feeds)
			gotCandles := make(map[time.Time][]types.Candle)
			go func() {
				for candle := range sp.sent {
					gotCandles[engine.backtester.curTime] = append(gotCandles[engine.backtester.curTime], candle)
					sp.ack <- struct{}{}
				}
			}()
			engine.backtester.run()

			for curTime, wantCandle := range tt.wantCandles {
				if gotCandle, ok := gotCandles[curTime]; ok {
					if len(wantCandle) != len(gotCandle) {
						t.Errorf("candles amount got %d, want %d for time %v",
							len(gotCandle), len(wantCandle), curTime)
					}
					for _, candle := range gotCandle {
						if !curTime.Equal(candle.Timestamp) {
							t.Errorf("candles mismatch for time %v got %v, want %v", curTime, candle.Timestamp, curTime)
						}
					}
				} else {
					t.Errorf("Time %v does not exist in gotCandles", curTime)
				}
			}
		})
	}
}

func TestBacktest_ShouldSendCandlesInOrder(t *testing.T) {
	testStrat := candlesReceivedStrategy{}
	engine := mockEngine(&testStrat, mockFeed())
	err := engine.Run()
	if err != nil {
		t.Errorf("Error running engine: %v", err)
	}
	if testStrat.receivedCount != 5 {
		t.Errorf("Expected 5 candles to be received but got %v", testStrat.receivedCount)
	}
	for i, candle := range testStrat.receivedCandles {
		if candle.Timestamp.UnixMilli() != (time.Duration(i) * types.IntervalToTime[testInterval]).Milliseconds() {
			t.Errorf("Expected candle unix timestamp to be %v but got %v", i, candle.Timestamp.UnixMilli())
		}
	}
}

func TestBacktest_getGlobalTimeRange(t *testing.T) {

	tests := []struct {
		name      string
		args      []*DataFeed
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "should return 0",
			args:      []*DataFeed{},
			wantStart: time.UnixMilli(0),
			wantEnd:   time.UnixMilli(0),
		},
		{
			name:      "should find min and max in first feed",
			args:      []*DataFeed{NewDataFeed("AAPL", testInterval, time.UnixMilli(1), time.UnixMilli(2))},
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(2),
		},
		{
			name: "should find min in first and max in second feed",
			args: []*DataFeed{
				NewDataFeed("AAPL", testInterval, time.UnixMilli(1), time.UnixMilli(2)),
				NewDataFeed("AAPL", testInterval, time.UnixMilli(1), time.UnixMilli(3)),
			},
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(3),
		},
		{
			name: "should find min in second and max in first feed",
			args: []*DataFeed{
				NewDataFeed("AAPL", testInterval, time.UnixMilli(3), time.UnixMilli(6)),
				NewDataFeed("AAPL", testInterval, time.UnixMilli(1), time.UnixMilli(2)),
			},
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(6),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := getGlobalTimeRange(tt.args)
			if !reflect.DeepEqual(got, tt.wantStart) {
				t.Errorf("getGlobalTimeRange() got = %v, wantStart %v", got, tt.wantStart)
			}
			if !reflect.DeepEqual(got1, tt.wantEnd) {
				t.Errorf("getGlobalTimeRange() got1 = %v, wantStart %v", got1, tt.wantEnd)
			}
		})
	}
}

var testInterval = types.OneMinute

// ----------------Helper functions----------------
func mockFeed() []*DataFeed {
	return []*DataFeed{NewDataFeed(
		"AAPL",
		testInterval,
		time.UnixMilli(0),
		time.UnixMilli(0).Add(types.IntervalToTime[testInterval]*time.Duration(5)),
	)}
}
func mockEngine(strat strategy, feed []*DataFeed) *Engine {
	db := mockDb{}
	return NewEngine(feed, strat, db)
}

type mockDb struct {
}

func (m mockDb) GetAssetByTicker(ticker string, ctx context.Context) (*types.Asset, error) {
	return &types.Asset{
		Id:     1,
		Ticker: "AAPL",
		Name:   "Apple Inc.",
		Type:   types.AssetTypeStock,
	}, nil
}

func mockCandle(assetId int, ts time.Time) types.Candle {
	return types.Candle{AssetId: assetId, Timestamp: ts}
}

func (m mockDb) GetAggregates(assetId int, interval types.Interval, start, end time.Time, ctx context.Context) ([]types.Candle, error) {
	var candles []types.Candle
	curTime := start
	for curTime.Before(end) {
		candles = append(candles, types.Candle{
			AssetId:   assetId,
			Open:      decimal.NewFromInt(curTime.UnixMilli()),
			Close:     decimal.NewFromInt(curTime.UnixMilli()),
			High:      decimal.NewFromInt(curTime.UnixMilli()),
			Low:       decimal.NewFromInt(curTime.UnixMilli()),
			Volume:    decimal.NewFromInt(curTime.UnixMilli()),
			Interval:  interval,
			Timestamp: curTime,
		})
		curTime = curTime.Add(types.IntervalToTime[interval])
	}
	return candles, nil
}

// Test strategies
type candlesReceivedStrategy struct {
	receivedCandles []types.Candle
	receivedCount   int
}

func (t *candlesReceivedStrategy) Init(api StrategyAPI) error {
	return nil
}
func (t *candlesReceivedStrategy) OnCandle(candle types.Candle) {
	t.receivedCandles = append(t.receivedCandles, candle)
	t.receivedCount++
}

type candlesParallelismStrategy struct {
	batch []types.Candle
	sent  chan types.Candle // Chan to receive candles for the current step
	ack   chan struct{}     // chan to tell onCandle to finish "processing"
}

func newCandlesParallelismStrategy() *candlesParallelismStrategy {
	return &candlesParallelismStrategy{sent: make(chan types.Candle), ack: make(chan struct{})}
}

func (s *candlesParallelismStrategy) Init(api StrategyAPI) error {
	return nil
}
func (s *candlesParallelismStrategy) OnCandle(c types.Candle) {
	// Send the candle
	s.sent <- c
	// Then wait for wrap up after all candles are verified and receives
	<-s.ack
}
