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

func TestBacktest_BacktesterBrokerUpdatePortfolio(t *testing.T) {
	testStrat := allocatorStrategy{}
	testAllocator := &mockAllocator{}
	testBroker := &mockBroker{
		reports: []types.ExecutionReport{newExecutionReport("AAPL", types.SideTypeBuy, newFill(time.UnixMilli(1), "100", "1", "1.00"))},
	}

	engine := mockEngine(&testStrat, mockFeed(), testAllocator, testBroker)

	err := engine.Run()
	if err != nil {
		t.Fatalf("Error running backtester: %v", err)
	}
	portfolioSnapshot := engine.portfolio.GetPortfolioSnapshot()
	pos, ok := portfolioSnapshot.Positions["AAPL"]
	if !ok {
		t.Fatalf("expected portfolio position AAPL but was nil")
	}
	if !pos.Quantity.Equal(decimal.NewFromFloat(6)) {
		t.Errorf("expected position AAPL to have quantity 6, got %v", pos.Quantity)
	}

}

func TestBacktest_BacktesterBrokerExecutionContext(t *testing.T) {
	testStrat := allocatorStrategy{}
	testAllocator := &mockAllocator{}
	testBroker := &mockBroker{}

	engine := mockEngine(&testStrat, mockFeed(), testAllocator, testBroker)

	err := engine.Run()
	if err != nil {
		t.Errorf("Error running backtester: %v", err)
	}
	expectedLens := []int{0, 1, 2, 2, 2, 2}
	if len(testBroker.ctx) != len(expectedLens) {
		t.Fatalf("Expected %d execution contexts, got %d", len(expectedLens), len(testBroker.ctx))
	}
	for i, executionCtx := range testBroker.ctx {
		got := len(executionCtx.Candles["AAPL"])
		if got != expectedLens[i] {
			t.Fatalf("Expected executionCtx %d to have %d candles, got %d", i, expectedLens[i], got)
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
		t.Run(tc.name, func(t *testing.T) {
			bt := &backtester{
				curTime: time.UnixMilli(42),
				executionConfig: &ExecutionConfig{
					candles:    map[string][]types.Candle{"TICK": tc.feed},
					barsBefore: tc.before,
					barsAfter:  tc.after,
				},
				portfolio:      newPortfolio(decimal.NewFromInt(1000), true),
				executionIndex: map[string]int{"TICK": tc.index},
			}
			bt.portfolio.backtesterApi = bt

			got := bt.getExecutionContext()

			sub := got.Candles["TICK"]
			if len(sub) != len(tc.wantTS) {
				t.Fatalf("len(sub)=%d, want=%d (timestamps=%v)", len(sub), len(tc.wantTS), tc.wantTS)
			}
			for _, ms := range tc.wantTS {
				ts := time.UnixMilli(ms)
				found := false
				for _, candle := range sub {
					if candle.Timestamp.Equal(ts) {
						found = true
						break
					}
				}
				if !found {
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
		executionConfig: &ExecutionConfig{
			candles: map[string][]types.Candle{
				"AAPL": aapl,
				"GOOG": goog,
			},
			barsBefore: 1,
			barsAfter:  2,
		},
		portfolio: newPortfolio(decimal.NewFromInt(1000), true),
		executionIndex: map[string]int{
			"AAPL": 5, // start=4->clamp 3; end=7->clamp 3 => empty
			"GOOG": 0, // start=-1->0; end=2 -> selects 100,101
		},
	}
	bt.portfolio.backtesterApi = bt

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
			found := false
			for _, candle := range sub {
				if candle.Timestamp.Equal(time.UnixMilli(ms)) {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("%s: missing timestamp %d (UnixMilli)", ticker, ms)
			}
		}
	}

	// Quick structural check using DeepEqual on expected candles for GOOG
	gotGOOG := got.Candles["GOOG"]
	wantGOOG := []types.Candle{
		goog[0],
		goog[1],
	}
	if !reflect.DeepEqual(gotGOOG, wantGOOG) {
		t.Fatalf("GOOG slice mismatch:\n got=%v\nwant=%v", gotGOOG, wantGOOG)
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
		barsBefore: 1,
		barsAfter:  2,
	}
	idx := map[string]int{
		"AAPL": 2,
		"GOOG": 3,
	}

	bt := &backtester{
		portfolio:       newPortfolio(decimal.NewFromInt(1000), true),
		curTime:         time.UnixMilli(9_999),
		executionConfig: &cfg,
		executionIndex:  idx,
	}
	bt.portfolio.backtesterApi = bt
	got := bt.getExecutionContext()

	// CurTime should be forwarded
	if got.CurTime != bt.curTime {
		t.Errorf("CurTime mismatch: got %v, want %v", got.CurTime, bt.curTime)
	}

	// Expect per-ticker slice windows:
	// AAPL: index 2 with 1 before, 2 after -> [1,2,3]
	// GOOG: index 3 with 1 before, 2 after -> [2,3,4]
	want := map[string][]types.Candle{
		"AAPL": {aaplFeed[1], aaplFeed[2], aaplFeed[3]},
		"GOOG": {googFeed[2], googFeed[3], googFeed[4]},
	}

	if got.Candles == nil {
		t.Fatalf("Candles is nil; expected populated map (did you assign ctx.Candles?)")
	}
	if len(got.Candles) != len(want) {
		t.Fatalf("top-level tickers len(got)=%d, want=%d", len(got.Candles), len(want))
	}

	for ticker, wantSlice := range want {
		sub, ok := got.Candles[ticker]
		if !ok {
			t.Errorf("missing ticker %q in Candles", ticker)
			continue
		}
		if len(sub) != len(wantSlice) {
			t.Errorf("%s: slice len(got)=%d, want=%d", ticker, len(sub), len(wantSlice))
			continue
		}
		for i := range wantSlice {
			if !reflect.DeepEqual(sub[i], wantSlice[i]) {
				t.Errorf("%s[%d]: got %+v, want %+v", ticker, i, sub[i], wantSlice[i])
			}
		}
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
				curIndex: -1,
				curTime:  base,
			},
			want: -1,
		},
		{
			name: "before first candle close",
			args: args{
				candles:  mkCandles(3),
				curIndex: -1,
				curTime:  base.Add(30 * time.Second),
			},
			want: -1,
		},
		{
			name: "exactly at first candle close",
			args: args{
				candles:  mkCandles(3),
				curIndex: -1,
				curTime:  base.Add(1 * time.Minute),
			},
			want: 0,
		},
		{
			name: "after last candle close",
			args: args{
				candles:  mkCandles(3),
				curIndex: -1,
				curTime:  base.Add(10 * time.Minute),
			},
			want: 2,
		},
		{
			name: "starts from a non-zero curIndex, still returns latest closed <= curTime",
			args: args{
				candles:  mkCandles(5),
				curIndex: 1,
				curTime:  base.Add(4 * time.Minute),
			},
			want: 3,
		},
		{
			name: "does not go backwards when curTime is before next close",
			args: args{
				candles:  mkCandles(5),
				curIndex: 3,
				curTime:  base.Add(3*time.Minute + 30*time.Second),
			},
			want: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := advanceFeedIndex(tt.args.candles, tt.args.curIndex, tt.args.curTime, testInterval)
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
				{ticker: "A", start: newTime(0), end: newTime(2), candles: nil},
			},
			wantCandles: map[time.Time][]types.Candle{},
		},
		{
			name: "all feeds send same timestamp per tick",
			feeds: []*DataFeedConfig{
				{
					ticker: "A",
					start:  newTime(0),
					end:    newTime(2),
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
					ticker: "A",
					start:  newTime(0),
					end:    newTime(2),
					candles: []types.Candle{
						mockCandle(1, newTime(0)),
						mockCandle(1, newTime(1)),
						mockCandle(1, newTime(2))},
				},
				{
					ticker: "B",
					start:  newTime(1),
					end:    newTime(1),
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
					ticker: "A",
					start:  newTime(0),
					end:    newTime(2),
					candles: []types.Candle{
						mockCandle(1, newTime(0)),
						mockCandle(1, newTime(1)),
						mockCandle(1, newTime(2))},
				},
				{
					ticker: "B",
					start:  newTime(0),
					end:    newTime(2),
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
				{ticker: "A", start: newTime(0), end: newTime(5),
					candles: []types.Candle{
						mockCandle(1, newTime(0)), mockCandle(1, newTime(3)), mockCandle(1, newTime(5)),
					},
				},
				{ticker: "B", start: newTime(0), end: newTime(5),
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
					ticker: "A",
					start:  newTime(0),
					end:    newTime(2),
					candles: []types.Candle{
						mockCandle(1, newTime(0)),
						mockCandle(1, newTime(1)),
						mockCandle(1, newTime(2))},
				},
				{
					ticker: "B",
					start:  newTime(1),
					end:    newTime(3),
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
					t.Errorf("createdAt %v does not exist in gotCandles", curTime)
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
			args:      NewDataFeedConfigs(),
			wantStart: time.UnixMilli(0),
			wantEnd:   time.UnixMilli(0),
		},
		{
			name: "should find min and max in first feed",
			args: NewDataFeedConfigs(
				&DataFeedConfig{ticker: "AAPL", interval: testInterval, start: time.UnixMilli(1), end: time.UnixMilli(2)},
			),
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(2),
		},
		{
			name: "should find min in first and max in second feed",
			args: NewDataFeedConfigs(
				&DataFeedConfig{ticker: "AAPL", interval: testInterval, start: time.UnixMilli(1), end: time.UnixMilli(2)},
				&DataFeedConfig{ticker: "AAPL", interval: testInterval, start: time.UnixMilli(2), end: time.UnixMilli(3)},
			),
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(3),
		},
		{
			name: "should find min in second and max in first feed",
			args: NewDataFeedConfigs(
				&DataFeedConfig{ticker: "AAPL", interval: testInterval, start: time.UnixMilli(3), end: time.UnixMilli(6)},
				&DataFeedConfig{ticker: "AAPL", interval: testInterval, start: time.UnixMilli(1), end: time.UnixMilli(2)},
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

func TestGetLastPriceForTicker(t *testing.T) {
	// Common candle data used in all tests
	candles := []types.Candle{
		{Timestamp: time.Unix(0, 0), Close: decimal.NewFromInt(100)},
		{Timestamp: time.Unix(60, 0), Close: decimal.NewFromInt(200)},
		{Timestamp: time.Unix(120, 0), Close: decimal.NewFromInt(300)},
	}

	feed := &DataFeedConfig{
		ticker:  "BTCUSDT",
		candles: candles,
		start:   candles[0].Timestamp,
		end:     candles[len(candles)-1].Timestamp,
	}

	tests := []struct {
		name      string
		feeds     []*DataFeedConfig
		feedIndex map[string]int
		ticker    string
		want      decimal.Decimal
	}{
		{
			name:      "normal case (index 1 -> use candle[0])",
			feeds:     []*DataFeedConfig{feed},
			feedIndex: map[string]int{"BTCUSDT": 1},
			ticker:    "BTCUSDT",
			want:      decimal.NewFromInt(100),
		},
		{
			name:      "index 0 -> idx - 1 = -1 -> clamped to 0",
			feeds:     []*DataFeedConfig{feed},
			feedIndex: map[string]int{"BTCUSDT": 0},
			ticker:    "BTCUSDT",
			want:      decimal.NewFromInt(100),
		},
		{
			name:      "index beyond end -> clamped to last candle",
			feeds:     []*DataFeedConfig{feed},
			feedIndex: map[string]int{"BTCUSDT": 10},
			ticker:    "BTCUSDT",
			want:      decimal.NewFromInt(300),
		},
		{
			name:      "ticker not found -> decimal.Zero",
			feeds:     []*DataFeedConfig{feed},
			feedIndex: map[string]int{},
			ticker:    "ETHUSDT",
			want:      decimal.Zero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &backtester{
				feeds:     tt.feeds,
				feedIndex: tt.feedIndex,
			}

			got := b.getLastPriceForTicker(tt.ticker)
			if !got.Equal(tt.want) {
				t.Fatalf("getLastPriceForTicker(%q) = %s, want %s",
					tt.ticker, got.String(), tt.want.String())
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
	return NewDataFeedConfigs(
		NewDataFeedConfig("AAPL", testInterval, time.UnixMilli(0), time.UnixMilli(0).Add(types.IntervalToTime[testInterval]*time.Duration(5))))
}
func mockEngine(strat strategy, feeds []*DataFeedConfig, allocator allocator, broker broker) *Engine {
	db := mockDb{}
	newPortfolio := NewPortfolioConfig(decimal.NewFromInt(100000), false)
	executionConfig := NewExecutionConfig(types.OneMinute, 1, 1)
	for _, feed := range feeds {
		executionConfig.candles[feed.ticker] = feed.candles
	}
	reportingConfig := NewReportingConfig(decimal.NewFromFloat(0.03), false, "", "")
	engine := NewEngine(feeds, executionConfig, reportingConfig, strat, allocator, broker, newPortfolio, db)
	return engine
}

type mockBroker struct {
	callCount int
	ctx       []types.ExecutionContext
	reports   []types.ExecutionReport
}

func (m *mockBroker) Execute(orders []types.Order, ctx types.ExecutionContext) []types.ExecutionReport {
	m.callCount++
	m.ctx = append(m.ctx, ctx)
	return m.reports
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

func (m mockDb) GetAggregates(assetId int, ticker string, interval types.Interval, start, end time.Time, ctx context.Context) ([]types.Candle, error) {
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
			signals = append(signals, types.Signal{CreatedAt: time.UnixMilli(int64(i))})
		}
		a.allocatorCalled++
	}
	return signals
}
