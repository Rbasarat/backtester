package engine

import (
	"backtester/types"
	"context"
	"github.com/shopspring/decimal"
	"reflect"
	"testing"
	"time"
)

var testInterval = types.OneMinute

func TestBacktest_BacktesterCallsAllocatorAndBroker(t *testing.T) {
	testStrat := allocatorStrategy{}
	testAllocator := &mockAllocator{}
	testBroker := &mockBroker{}
	engine := mockEngine(&testStrat, mockFeed(), testAllocator, testBroker)

	engine.Run()
	// We always call the allocator even if we have 0 signals
	if testAllocator.callCount != 6 {
		t.Errorf("Allocator called %d times, want %d", testAllocator.callCount, 6)
	}
	// We always call the broker even if we have 0 signals
	if testBroker.callCount != 6 {
		t.Errorf("Broker called %d times, want %d", testBroker.callCount, 6)
	}

}

func TestBacktest_AllFeedsSendSameTimestampPerTick(t *testing.T) {
	base := time.UnixMilli(0).UTC()
	newTime := func(i int) time.Time { return base.Add(time.Duration(i) * time.Minute) }
	tests := []struct {
		name        string
		feeds       []*DataFeedConfig
		wantCandles map[time.Time][]types.Candle
	}{
		{
			name: "feed with no candles",
			feeds: []*DataFeedConfig{
				{Ticker: "A", Start: newTime(0), End: newTime(2), candles: nil},
			},
			wantCandles: map[time.Time][]types.Candle{},
		},
		{
			name: "all feeds send same timestamp per tick",
			feeds: []*DataFeedConfig{
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
			feeds: []*DataFeedConfig{
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
			feeds: []*DataFeedConfig{
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
			feeds: []*DataFeedConfig{
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
			feeds: []*DataFeedConfig{
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
			testAllocator := &mockAllocator{}
			testBroker := &mockBroker{}
			engine := mockEngine(sp, tt.feeds, testAllocator, testBroker)

			gotCandles := make(map[time.Time][]types.Candle)
			// We do it like this because we want to verify that the candle timestamp == curTime in the backtest loop
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
	testAllocator := &mockAllocator{}
	testBroker := &mockBroker{}
	engine := mockEngine(&testStrat, mockFeed(), testAllocator, testBroker)

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
		args      []*DataFeedConfig
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "should return 0",
			args:      []*DataFeedConfig{},
			wantStart: time.UnixMilli(0),
			wantEnd:   time.UnixMilli(0),
		},
		{
			name:      "should find min and max in first feed",
			args:      []*DataFeedConfig{NewDataFeed("AAPL", testInterval, time.UnixMilli(1), time.UnixMilli(2))},
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(2),
		},
		{
			name: "should find min in first and max in second feed",
			args: []*DataFeedConfig{
				NewDataFeed("AAPL", testInterval, time.UnixMilli(1), time.UnixMilli(2)),
				NewDataFeed("AAPL", testInterval, time.UnixMilli(1), time.UnixMilli(3)),
			},
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(3),
		},
		{
			name: "should find min in second and max in first feed",
			args: []*DataFeedConfig{
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

// ----------------Helper functions----------------
func mockFeed() []*DataFeedConfig {
	return []*DataFeedConfig{NewDataFeed(
		"AAPL",
		testInterval,
		time.UnixMilli(0),
		time.UnixMilli(0).Add(types.IntervalToTime[testInterval]*time.Duration(5)),
	)}
}
func mockEngine(strat strategy, feed []*DataFeedConfig, allocator allocator, broker broker) *Engine {
	db := mockDb{}
	portfolio := NewPortfolioConfig(decimal.NewFromInt(1000))
	return NewEngine(feed, strat, allocator, broker, portfolio, db)
}

type mockBroker struct {
	callCount int
}

func (m *mockBroker) Execute(orders []types.Order) {
	m.callCount++
}

type mockAllocator struct {
	callCount int
}

func (m *mockAllocator) Allocate(signals []types.Signal, view types.PortfolioView) []types.Order {
	m.callCount++
	return nil
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

func (t *candlesReceivedStrategy) Init(api PortfolioApi) error {
	return nil
}
func (t *candlesReceivedStrategy) OnCandle(candle types.Candle) []types.Signal {
	t.receivedCandles = append(t.receivedCandles, candle)
	t.receivedCount++
	return nil
}

type candlesParallelismStrategy struct {
	batch []types.Candle
	sent  chan types.Candle
	ack   chan struct{}
}

func newCandlesParallelismStrategy() *candlesParallelismStrategy {
	return &candlesParallelismStrategy{sent: make(chan types.Candle), ack: make(chan struct{})}
}

func (s *candlesParallelismStrategy) Init(api PortfolioApi) error {
	return nil
}
func (s *candlesParallelismStrategy) OnCandle(c types.Candle) []types.Signal {
	// Send the candle
	s.sent <- c
	// Then wait for wrap up after all candles are received
	<-s.ack
	return nil
}

type allocatorStrategy struct {
	callAllocator   int
	allocatorCalled int
}

func (a *allocatorStrategy) Init(api PortfolioApi) error {
	return nil
}

func (a *allocatorStrategy) OnCandle(candle types.Candle) []types.Signal {
	var signals []types.Signal
	if a.allocatorCalled < a.callAllocator {
		for i := range a.callAllocator {
			signals = append(signals, types.Signal{Time: time.UnixMilli(int64(i))})
		}
		a.allocatorCalled++
	}
	return signals
}
