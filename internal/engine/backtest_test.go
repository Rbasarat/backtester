package engine

import (
	"backtester/types"
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

var testInterval = types.OneMinute

func TestBacktest_BacktesterBrokerExecutionContext(t *testing.T) {
	testStrat := allocatorStrategy{}
	testAllocator := &mockAllocator{}
	testBroker := &mockBroker{}

	engine := mockEngine(&testStrat, mockFeed(), testAllocator, testBroker)
	engine.portfolio = newPortfolio(decimal.NewFromInt(1000))

	err := engine.Run()
	if err != nil {
		t.Errorf("Error running backtester: %v", err)
	}
	if len(testBroker.ctx) != 6 {
		t.Errorf("Expected at least 6 executionContext, got %d", len(testBroker.ctx))
	}
	for i, executionCtx := range testBroker.ctx {
		if i == 0 || i == 6 {
			if len(executionCtx.Candles) != 1 {
				t.Fatalf("Expected executionCtx candles on edges of list len 1, got %d", len(executionCtx.Candles))
			}
		} else {
			if len(executionCtx.Candles["AAPL"]) != 2 {
				t.Fatalf("Expected executionCtx candles len 2, got %d", len(executionCtx.Candles))
			}
		}
	}
}

func TestBacktester_GetExecutionContext_ClampsWindow_NoPanics(t *testing.T) {
	tests := []struct {
		name   string
		feed   []types.Candle
		index  int
		before int
		after  int
		wantTS []int64 // UnixMilli timestamps expected
	}{
		{
			name:   "both sides clamp to full feed",
			feed:   mockCandles(0, 3, 1),
			index:  1,
			before: 5,
			after:  5,
			wantTS: []int64{0, 1, 2},
		},
		{
			name:   "end clamp only",
			feed:   mockCandles(0, 3, 1),
			index:  2,
			before: 1,
			after:  2,
			wantTS: []int64{1, 2},
		},
		{
			name:   "start clamp only",
			feed:   mockCandles(0, 3, 1),
			index:  0,
			before: 2,
			after:  1,
			wantTS: []int64{0},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			bt := &backtester{
				curTime: time.UnixMilli(42),
				executionConfig: ExecutionConfig{
					candles:    map[string][]types.Candle{"TICK": tc.feed},
					BarsBefore: tc.before,
					BarsAfter:  tc.after,
				},
				portfolio:      newPortfolio(decimal.NewFromInt(1000)),
				executionIndex: map[string]int{"TICK": tc.index},
			}

			got := bt.getExecutionContext()

			sub := got.Candles["TICK"]
			if len(sub) != len(tc.wantTS) {
				t.Fatalf("len(sub)=%d, want=%d (timestamps=%v)", len(sub), len(tc.wantTS), tc.wantTS)
			}
			for _, ms := range tc.wantTS {
				ts := time.UnixMilli(ms)
				if _, ok := sub[ts]; !ok {
					t.Fatalf("missing timestamp %d (UnixMilli)", ms)
				}
			}
		})
	}
}

func TestBacktester_GetExecutionContext_MixedTickers_Clamping(t *testing.T) {
	aapl := mockCandles(0, 3, 1)   // 0,1,2
	goog := mockCandles(100, 5, 2) // 100..104

	bt := &backtester{
		curTime: time.UnixMilli(9_999),
		executionConfig: ExecutionConfig{
			candles: map[string][]types.Candle{
				"AAPL": aapl,
				"GOOG": goog,
			},
			BarsBefore: 1,
			BarsAfter:  2,
		},
		portfolio: newPortfolio(decimal.NewFromInt(1000)),
		executionIndex: map[string]int{
			"AAPL": 5, // start=4->clamp 3; end=7->clamp 3 => empty
			"GOOG": 0, // start=-1->0; end=2 -> selects 100,101
		},
	}

	got := bt.getExecutionContext()
	want := map[string][]int64{
		"AAPL": {},         // empty after clamping
		"GOOG": {100, 101}, // clamped window
	}

	for ticker, tsList := range want {
		sub := got.Candles[ticker]
		if len(sub) != len(tsList) {
			t.Fatalf("%s: len=%d, want=%d", ticker, len(sub), len(tsList))
		}
		for _, ms := range tsList {
			if _, ok := sub[time.UnixMilli(ms)]; !ok {
				t.Fatalf("%s: missing %d", ticker, ms)
			}
		}
	}

	// Quick structural check using DeepEqual on expected candles for GOOG
	gotGOOG := got.Candles["GOOG"]
	wantGOOG := map[time.Time]types.Candle{
		time.UnixMilli(100): goog[0],
		time.UnixMilli(101): goog[1],
	}
	if !reflect.DeepEqual(gotGOOG, wantGOOG) {
		t.Fatalf("GOOG map mismatch:\n got=%v\nwant=%v", gotGOOG, wantGOOG)
	}
}

func TestBacktester_GetExecutionContext_SlicesAndMapsByTicker(t *testing.T) {
	aaplFeed := mockCandles(0, 5, 1)
	googFeed := mockCandles(100, 6, 2)

	cfg := ExecutionConfig{
		candles: map[string][]types.Candle{
			"AAPL": aaplFeed,
			"GOOG": googFeed,
		},
		BarsBefore: 1,
		BarsAfter:  2,
	}
	idx := map[string]int{
		"AAPL": 2,
		"GOOG": 3,
	}

	bt := &backtester{
		portfolio:       newPortfolio(decimal.NewFromInt(1000)),
		curTime:         time.UnixMilli(9_999),
		executionConfig: cfg,
		executionIndex:  idx,
	}

	got := bt.getExecutionContext()

	// CurTime should be forwarded
	if got.CurTime != bt.curTime {
		t.Errorf("CurTime mismatch: got %v, want %v", got.CurTime, bt.curTime)
	}

	// Expect per-ticker map windows
	want := map[string]map[time.Time]types.Candle{
		"AAPL": {
			time.UnixMilli(1): aaplFeed[1],
			time.UnixMilli(2): aaplFeed[2],
			time.UnixMilli(3): aaplFeed[3],
		},
		"GOOG": {
			time.UnixMilli(102): googFeed[2],
			time.UnixMilli(103): googFeed[3],
			time.UnixMilli(104): googFeed[4],
		},
	}

	if got.Candles == nil {
		t.Fatalf("Candles is nil; expected populated map (did you assign ctx.Candles = candlesMap?)")
	}
	if len(got.Candles) != len(want) {
		t.Fatalf("top-level tickers len(got)=%d, want=%d", len(got.Candles), len(want))
	}

	for ticker, wantMap := range want {
		sub, ok := got.Candles[ticker]
		if !ok {
			t.Errorf("missing ticker %q in Candles", ticker)
		}
		if len(sub) != len(wantMap) {
			t.Errorf("%s: submap len(got)=%d, want=%d", ticker, len(sub), len(wantMap))
		}
		for ts, wantC := range wantMap {
			gotC, ok := sub[ts]
			if !ok {
				t.Errorf("%s: missing timestamp %v", ticker, ts)
			}
			if !reflect.DeepEqual(gotC, wantC) {
				t.Errorf("%s @ %v: got %+v, want %+v", ticker, ts, gotC, wantC)
			}
		}
	}
}

func TestCreateMapFromCandles(t *testing.T) {
	// Simple consecutive timestamps for clarity
	t1 := time.UnixMilli(0)
	t2 := time.UnixMilli(1)
	t3 := time.UnixMilli(2)

	d := func(i int64) decimal.Decimal { return decimal.NewFromInt(i) }

	c1 := types.Candle{AssetId: 1, Open: d(1), Close: d(2), High: d(3), Low: d(0), Volume: d(10), Interval: testInterval, Timestamp: t1}
	c2 := types.Candle{AssetId: 1, Open: d(2), Close: d(3), High: d(4), Low: d(1), Volume: d(20), Interval: testInterval, Timestamp: t2}
	c3 := types.Candle{AssetId: 2, Open: d(3), Close: d(4), High: d(5), Low: d(2), Volume: d(30), Interval: testInterval, Timestamp: t3}
	c2Overwrite := types.Candle{AssetId: 99, Open: d(9), Close: d(9), High: d(9), Low: d(9), Volume: d(99), Interval: testInterval, Timestamp: t2}

	tests := []struct {
		name string
		in   []types.Candle
		want map[time.Time]types.Candle
	}{
		{
			name: "empty slice",
			in:   nil,
			want: map[time.Time]types.Candle{},
		},
		{
			name: "single candle",
			in:   []types.Candle{c1},
			want: map[time.Time]types.Candle{t1: c1},
		},
		{
			name: "multiple unique timestamps",
			in:   []types.Candle{c1, c2, c3},
			want: map[time.Time]types.Candle{t1: c1, t2: c2, t3: c3},
		},
		{
			name: "duplicate timestamp overwrites with last occurrence",
			in:   []types.Candle{c1, c2, c2Overwrite, c3},
			want: map[time.Time]types.Candle{t1: c1, t2: c2Overwrite, t3: c3},
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			got := createMapFromCandles(tc.in)

			if len(got) != len(tc.want) {
				t.Errorf("len(got)=%d, len(want)=%d, got=%v", len(got), len(tc.want), got)
			}

			for ts, wantC := range tc.want {
				gotC, ok := got[ts]
				if !ok {
					t.Errorf("missing timestamp %v in result", ts)
				}
				if !reflect.DeepEqual(gotC, wantC) {
					t.Errorf("for timestamp %v:\n got  %+v\n want %+v", ts, gotC, wantC)
				}
			}
		})
	}
}

func TestBacktest_BacktesterCallsAllocatorAndBroker(t *testing.T) {
	testStrat := allocatorStrategy{}
	testAllocator := &mockAllocator{}
	testBroker := &mockBroker{}
	engine := mockEngine(&testStrat, mockFeed(), testAllocator, testBroker)

	err := engine.Run()
	if err != nil {
		t.Errorf("Error running engine: %v", err)
	}
	// We always call the allocator even if we have 0 signals
	if testAllocator.callCount != 6 {
		t.Errorf("Allocator called %d times, want %d", testAllocator.callCount, 6)
	}
	// We always call the broker even if we have 0 signals
	if testBroker.callCount != 6 {
		t.Errorf("Broker called %d times, want %d", testBroker.callCount, 6)
	}
}

func TestBacktest_advanceFeedIndex(t *testing.T) {
	base := time.UnixMilli(0)
	// helper to build minute-spaced candles: 0m,1m,2m,...
	mkCandles := func(n int) []types.Candle {
		cs := make([]types.Candle, n)
		for i := 0; i < n; i++ {
			cs[i] = types.Candle{Timestamp: base.Add(time.Duration(i) * types.IntervalToTime[testInterval])}
		}
		return cs
	}

	type args struct {
		candles  []types.Candle
		curIndex int
		curTime  time.Time
	}
	tests := []struct {
		name string
		args args
		want int
	}{
		{
			name: "empty feed",
			args: args{
				candles:  nil,
				curIndex: 0,
				curTime:  base,
			},
			want: 0,
		},
		{
			name: "before first candle",
			args: args{
				candles:  mkCandles(3),
				curIndex: 0,
				curTime:  base.Add(-time.Second),
			},
			want: 0,
		},
		{
			name: "exactly at first candle",
			args: args{
				candles:  mkCandles(3),
				curIndex: 0,
				curTime:  base,
			},
			want: 0,
		},
		{
			name: "between first and second candle",
			args: args{
				candles:  mkCandles(3),
				curIndex: 0,
				curTime:  base.Add(30 * time.Second),
			},
			want: 0,
		},
		{
			name: "exactly at second candle",
			args: args{
				candles:  mkCandles(3),
				curIndex: 0,
				curTime:  base.Add(1 * time.Minute),
			},
			want: 1,
		},
		{
			name: "after last candle",
			args: args{
				candles:  mkCandles(3),
				curIndex: 0,
				curTime:  base.Add(10 * time.Minute),
			},
			want: 2,
		},
		{
			name: "starts from a non-zero curIndex, still returns latest <= curTime",
			args: args{
				candles:  mkCandles(5),
				curIndex: 2,
				curTime:  base.Add(3 * time.Minute),
			},
			want: 3,
		},
		{
			name: "curTime equal to current curIndex timestamp",
			args: args{
				candles:  mkCandles(5),
				curIndex: 2,
				curTime:  base.Add(2 * time.Minute),
			},
			want: 2,
		},
		{
			name: "curTime before curIndex timestamp (should not go backwards; returns curIndex-1 if that timestamp <= curTime)",
			args: args{
				candles:  mkCandles(5),
				curIndex: 3,
				curTime:  base.Add(90 * time.Second),
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := advanceFeedIndex(tt.args.candles, tt.args.curIndex, tt.args.curTime)
			if got != tt.want {
				t.Fatalf("advanceFeedIndex() = %d, want %d", got, tt.want)
			}
		})
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
			err := engine.backtester.run()
			if err != nil {
				t.Errorf("Error running engine: %v", err)
			}

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
			args:      NewDataFeedConfig(),
			wantStart: time.UnixMilli(0),
			wantEnd:   time.UnixMilli(0),
		},
		{
			name: "should find min and max in first feed",
			args: NewDataFeedConfig(
				DataFeedConfig{Ticker: "AAPL", Interval: testInterval, Start: time.UnixMilli(1), End: time.UnixMilli(2)},
			),
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(2),
		},
		{
			name: "should find min in first and max in second feed",
			args: NewDataFeedConfig(
				DataFeedConfig{Ticker: "AAPL", Interval: testInterval, Start: time.UnixMilli(1), End: time.UnixMilli(2)},
				DataFeedConfig{Ticker: "AAPL", Interval: testInterval, Start: time.UnixMilli(2), End: time.UnixMilli(3)},
			),
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(3),
		},
		{
			name: "should find min in second and max in first feed",
			args: NewDataFeedConfig(
				DataFeedConfig{Ticker: "AAPL", Interval: testInterval, Start: time.UnixMilli(3), End: time.UnixMilli(6)},
				DataFeedConfig{Ticker: "AAPL", Interval: testInterval, Start: time.UnixMilli(1), End: time.UnixMilli(2)},
			),
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
func mockCandles(startMilli int64, n int, assetID int) []types.Candle {
	out := make([]types.Candle, 0, n)
	for i := 0; i < n; i++ {
		ts := time.UnixMilli(startMilli + int64(i))
		out = append(out, types.Candle{
			AssetId:   assetID,
			Open:      decimal.NewFromInt(int64(i)),
			Close:     decimal.NewFromInt(int64(i + 1)),
			High:      decimal.NewFromInt(int64(i + 2)),
			Low:       decimal.NewFromInt(int64(i)),
			Volume:    decimal.NewFromInt(int64(i * 10)),
			Interval:  testInterval,
			Timestamp: ts,
		})
	}
	return out
}

func mockFeed() []*DataFeedConfig {
	return NewDataFeedConfig(
		DataFeedConfig{
			Ticker:   "AAPL",
			Interval: testInterval,
			Start:    time.UnixMilli(0),
			End:      time.UnixMilli(0).Add(types.IntervalToTime[testInterval] * time.Duration(5)),
		},
	)
}
func mockEngine(strat strategy, feeds []*DataFeedConfig, allocator allocator, broker broker) *Engine {
	db := mockDb{}
	portfolio := NewPortfolioConfig(decimal.NewFromInt(1000))
	executionConfig := NewExecutionConfig(types.OneMinute, 1, 1)
	for _, feed := range feeds {
		executionConfig.candles[feed.Ticker] = feed.candles
	}
	engine := NewEngine(feeds, executionConfig, strat, allocator, broker, portfolio, db)
	return engine
}

type mockBroker struct {
	callCount int
	ctx       []types.ExecutionContext
}

func (m *mockBroker) Execute(orders []types.Order, ctx types.ExecutionContext) []types.ExecutionReport {
	m.callCount++
	m.ctx = append(m.ctx, ctx)
	return nil
}

type mockAllocator struct {
	callCount int
}

func (m *mockAllocator) Init(api PortfolioApi) error {
	return nil
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
