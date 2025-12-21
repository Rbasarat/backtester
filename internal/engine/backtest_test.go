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

func TestBacktester_PassesSignalsMapKeyedByTicker(t *testing.T) {
	instruments := []*InstrumentConfig{
		{
			ticker:   "AAPL",
			interval: testInterval,
			start:    time.UnixMilli(0).UTC(),
			end:      time.UnixMilli(60_000).UTC(),
		},
		{
			ticker:   "GOOG",
			interval: testInterval,
			start:    time.UnixMilli(0).UTC(),
			end:      time.UnixMilli(120_000).UTC(),
		},
	}

	strat := &tickerTaggingStrategy{}
	recAlloc := &recordingAllocator{}
	testBroker := &mockBroker{}

	engine := mockEngine(strat, instruments, recAlloc, testBroker)

	if err := engine.Run(); err != nil {
		t.Fatalf("Error running engine: %v", err)
	}

	// Flatten all allocator calls into a single aggregated map[ticker][]Signal
	agg := make(map[string][]types.Signal)
	for _, call := range recAlloc.calls {
		for ticker, sigs := range call {
			agg[ticker] = append(agg[ticker], sigs...)
		}
	}

	// We expect only these tickers to appear as keys.
	for ticker := range agg {
		if ticker != "AAPL" && ticker != "GOOG" {
			t.Fatalf("unexpected ticker key in signals map: %q", ticker)
		}
	}

	for ticker, sigs := range agg {
		if len(sigs) == 0 {
			t.Fatalf("no signals recorded for ticker %q", ticker)
		}
		for _, s := range sigs {
			gotTag := s.CreatedAt.UnixMilli()
			switch ticker {
			case "AAPL":
				if gotTag != 0 {
					t.Errorf("AAPL signal has tag %d, wantIndex 0", gotTag)
				}
			case "GOOG":
				if gotTag != 1 {
					t.Errorf("GOOG signal has tag %d, wantIndex 1", gotTag)
				}
			}
		}
	}
}

func TestBacktest_BacktesterBrokerUpdatePortfolio(t *testing.T) {
	testStrat := allocatorStrategy{}
	testAllocator := &mockAllocator{}
	testBroker := &mockBroker{
		reports: []types.ExecutionReport{newExecutionReport("AAPL", types.SideTypeBuy, newFill(time.UnixMilli(1), "100", "1", "1.00"))},
	}

	engine := mockEngine(&testStrat, mockInstrument(), testAllocator, testBroker)

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

	engine := mockEngine(&testStrat, mockInstrument(), testAllocator, testBroker)

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

			got := bt.buildExecutionContext()

			sub := got.Candles["TICK"]
			if len(sub) != len(tc.wantTS) {
				t.Fatalf("len(sub)=%d, wantIndex=%d (timestamps=%v)", len(sub), len(tc.wantTS), tc.wantTS)
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

	got := bt.buildExecutionContext()
	want := map[string][]int64{
		"AAPL": {},         // empty after clamping
		"GOOG": {100, 101}, // clamped window
	}

	for ticker, tsList := range want {
		sub := got.Candles[ticker]
		if len(sub) != len(tsList) {
			t.Fatalf("%s: len=%d, wantIndex=%d", ticker, len(sub), len(tsList))
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
		t.Fatalf("GOOG slice mismatch:\n got=%v\nwantIndex=%v", gotGOOG, wantGOOG)
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
	got := bt.buildExecutionContext()

	// CurTime should be forwarded
	if got.CurTime != bt.curTime {
		t.Errorf("CurTime mismatch: got %v, wantIndex %v", got.CurTime, bt.curTime)
	}

	// Expect per-ticker slice windows:
	// AAPL: index 2 with 1 before, 2 after -> [1,2,3]
	// GOOG: index 3 with 1 before, 2 after -> [2,3,4]
	want := map[string][]types.Candle{
		"AAPL": {aaplFeed[1], aaplFeed[2], aaplFeed[3]},
		"GOOG": {googFeed[2], googFeed[3], googFeed[4]},
	}

	if got.Candles == nil {
		t.Fatalf("candles is nil; expected populated map (did you assign ctx.candles?)")
	}
	if len(got.Candles) != len(want) {
		t.Fatalf("top-level tickers len(got)=%d, wantIndex=%d", len(got.Candles), len(want))
	}

	for ticker, wantSlice := range want {
		sub, ok := got.Candles[ticker]
		if !ok {
			t.Errorf("missing ticker %q in candles", ticker)
			continue
		}
		if len(sub) != len(wantSlice) {
			t.Errorf("%s: slice len(got)=%d, wantIndex=%d", ticker, len(sub), len(wantSlice))
			continue
		}
		for i := range wantSlice {
			if !reflect.DeepEqual(sub[i], wantSlice[i]) {
				t.Errorf("%s[%d]: got %+v, wantIndex %+v", ticker, i, sub[i], wantSlice[i])
			}
		}
	}
}

func TestBacktest_BacktesterCallsAllocatorAndBroker(t *testing.T) {
	testStrat := allocatorStrategy{}
	testAllocator := &mockAllocator{}
	testBroker := &mockBroker{}
	engine := mockEngine(&testStrat, mockInstrument(), testAllocator, testBroker)

	err := engine.Run()
	if err != nil {
		t.Errorf("Error running engine: %v", err)
	}
	// We always call the allocator even if we have 0 signals
	if testAllocator.callCount != 6 {
		t.Errorf("Allocator called %d times, wantIndex %d", testAllocator.callCount, 6)
	}
	// We always call the broker even if we have 0 signals
	if testBroker.callCount != 6 {
		t.Errorf("Broker called %d times, wantIndex %d", testBroker.callCount, 6)
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
				t.Fatalf("advanceFeedIndex() = %d, wantIndex %d", got, tt.want)
			}
		})
	}

}

func TestBacktest_AllPrimaryInstrumentsSendSameTimestampPerTick(t *testing.T) {

	tests := []struct {
		name        string
		feeds       []*InstrumentConfig
		wantCandles map[time.Time][]types.Candle
	}{
		{
			name: "feed with no candles",
			feeds: []*InstrumentConfig{
				{ticker: "A", start: newTime(0), end: newTime(2), primary: TimeframeConfig{candles: nil}},
			},
			wantCandles: map[time.Time][]types.Candle{},
		},
		{
			name: "all instruments send same timestamp per tick",
			feeds: []*InstrumentConfig{
				{
					ticker: "A",
					start:  newTime(0),
					end:    newTime(2),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
						mockCandle(1, newTime(1), testInterval),
						mockCandle(1, newTime(2), testInterval)},
					}},
			},
			wantCandles: map[time.Time][]types.Candle{
				newTime(0): {mockCandle(1, newTime(0), testInterval)},
				newTime(1): {mockCandle(1, newTime(1), testInterval)},
				newTime(2): {mockCandle(1, newTime(2), testInterval)},
			},
		},
		{
			name: "one feed is subset of max range",
			feeds: []*InstrumentConfig{
				{
					ticker: "A",
					start:  newTime(0),
					end:    newTime(2),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
						mockCandle(1, newTime(1), testInterval),
						mockCandle(1, newTime(2), testInterval)},
					},
				},
				{
					ticker: "B",
					start:  newTime(1),
					end:    newTime(1),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(2, newTime(1), testInterval),
					}},
				},
			},
			wantCandles: map[time.Time][]types.Candle{
				newTime(0): {mockCandle(1, newTime(0), testInterval)},
				newTime(1): {mockCandle(1, newTime(1), testInterval), mockCandle(2, newTime(1), testInterval)},
				newTime(2): {mockCandle(1, newTime(2), testInterval)},
			},
		},
		{
			name: "two instruments send same timestamp per tick",
			feeds: []*InstrumentConfig{
				{
					ticker: "A",
					start:  newTime(0),
					end:    newTime(2),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
						mockCandle(1, newTime(1), testInterval),
						mockCandle(1, newTime(2), testInterval)},
					},
				},
				{
					ticker: "B",
					start:  newTime(0),
					end:    newTime(2),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(2, newTime(0), testInterval),
						mockCandle(2, newTime(1), testInterval),
						mockCandle(2, newTime(2), testInterval)},
					},
				},
			},
			wantCandles: map[time.Time][]types.Candle{
				newTime(0): {mockCandle(1, newTime(0), testInterval), mockCandle(2, newTime(0), testInterval)},
				newTime(1): {mockCandle(1, newTime(1), testInterval), mockCandle(2, newTime(1), testInterval)},
				newTime(2): {mockCandle(1, newTime(2), testInterval), mockCandle(2, newTime(2), testInterval)},
			},
		},
		{
			name: "irregular intervals vs dense feed",
			feeds: []*InstrumentConfig{
				{ticker: "A", start: newTime(0), end: newTime(5),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
						mockCandle(1, newTime(3), testInterval),
						mockCandle(1, newTime(5), testInterval)},
					},
				},
				{ticker: "B", start: newTime(0), end: newTime(5),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(2, newTime(0), testInterval), mockCandle(2, newTime(1), testInterval), mockCandle(2, newTime(2), testInterval),
						mockCandle(2, newTime(3), testInterval), mockCandle(2, newTime(4), testInterval), mockCandle(2, newTime(5), testInterval)},
					},
				},
			},
			wantCandles: map[time.Time][]types.Candle{
				newTime(0): {mockCandle(1, newTime(0), testInterval), mockCandle(2, newTime(0), testInterval)},
				newTime(1): {mockCandle(2, newTime(1), testInterval)},
				newTime(2): {mockCandle(2, newTime(2), testInterval)},
				newTime(3): {mockCandle(1, newTime(3), testInterval), mockCandle(2, newTime(3), testInterval)},
				newTime(4): {mockCandle(2, newTime(4), testInterval)},
				newTime(5): {mockCandle(1, newTime(5), testInterval), mockCandle(2, newTime(5), testInterval)},
			},
		},
		{
			name: "one feed overlaps the other",
			feeds: []*InstrumentConfig{
				{
					ticker: "A",
					start:  newTime(0),
					end:    newTime(2),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
						mockCandle(1, newTime(1), testInterval),
						mockCandle(1, newTime(2), testInterval),
					}},
				},
				{
					ticker: "B",
					start:  newTime(1),
					end:    newTime(3),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(2, newTime(1), testInterval),
						mockCandle(2, newTime(2), testInterval),
						mockCandle(2, newTime(3), testInterval),
					}},
				},
			},
			wantCandles: map[time.Time][]types.Candle{
				newTime(0): {mockCandle(1, newTime(0), testInterval)},
				newTime(1): {mockCandle(1, newTime(1), testInterval), mockCandle(2, newTime(1), testInterval)},
				newTime(2): {mockCandle(1, newTime(2), testInterval), mockCandle(2, newTime(2), testInterval)},
				newTime(3): {mockCandle(2, newTime(3), testInterval)},
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
			// We do it like this because we wantIndex to verify that the candle timestamp == curTime in the backtest loop
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
						t.Errorf("candles amount got %d, wantIndex %d for time %v",
							len(gotCandle), len(wantCandle), curTime)
					}
					for _, candle := range gotCandle {
						if !curTime.Equal(candle.Timestamp) {
							t.Errorf("candles mismatch for time %v got %v, wantIndex %v", curTime, candle.Timestamp, curTime)
						}
					}
				} else {
					t.Errorf("createdAt %v does not exist in gotCandles", curTime)
				}
			}
		})
	}
}

func TestBacktester_buildInstrumentContext_ReturnsExpectedContext(t *testing.T) {
	tests := []struct {
		name       string
		inst       *InstrumentConfig
		curTime    time.Time
		preIndexes map[string]map[types.Interval]int // pre-seeded state (optional)

		want map[types.Interval][]types.Candle
	}{
		{
			name: "returns candles closed at or before curTime for one interval",
			inst: &InstrumentConfig{
				ticker: "A",
				context: []TimeframeConfig{
					{
						interval: testInterval, // assume 1m
						candles: []types.Candle{
							mockCandle(1, newTime(0), testInterval), // closes at t=1
							mockCandle(2, newTime(1), testInterval), // closes at t=2
							mockCandle(3, newTime(2), testInterval), // closes at t=3
						},
					},
				},
			},
			curTime: newTime(2), // include first two context candles (close times 1 and 2)

			// Seed state so the function reads from the index map (even though we won't assert it after)
			preIndexes: map[string]map[types.Interval]int{
				"A": {testInterval: 0},
			},
			want: map[types.Interval][]types.Candle{
				testInterval: {
					mockCandle(1, newTime(0), testInterval),
					mockCandle(2, newTime(1), testInterval),
				},
			},
		},
		{
			name:       "no context configs returns empty map",
			inst:       &InstrumentConfig{ticker: "A", context: nil},
			curTime:    newTime(10),
			preIndexes: nil,
			want:       map[types.Interval][]types.Candle{},
		},
		{
			name: "curTime before first candle close returns empty slice",
			inst: &InstrumentConfig{
				ticker: "A",
				context: []TimeframeConfig{{
					interval: testInterval,
					candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
					},
				}},
			},
			curTime:    newTime(0),
			preIndexes: nil,
			want: map[types.Interval][]types.Candle{
				testInterval: {},
			},
		},
		{
			name: "curTime exactly at first candle close includes first candle",
			inst: &InstrumentConfig{
				ticker: "A",
				context: []TimeframeConfig{{
					interval: testInterval,
					candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
						mockCandle(2, newTime(1), testInterval),
					},
				}},
			},
			curTime:    newTime(1),
			preIndexes: nil,
			want: map[types.Interval][]types.Candle{
				testInterval: {
					mockCandle(1, newTime(0), testInterval),
				},
			},
		},
		{
			name: "curTime after all closes returns all candles",
			inst: &InstrumentConfig{
				ticker: "A",
				context: []TimeframeConfig{{
					interval: testInterval,
					candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
						mockCandle(2, newTime(1), testInterval),
					},
				}},
			},
			curTime:    newTime(999),
			preIndexes: nil,
			want: map[types.Interval][]types.Candle{
				testInterval: {
					mockCandle(1, newTime(0), testInterval),
					mockCandle(2, newTime(1), testInterval),
				},
			},
		},
		{
			name: "multiple intervals return both contexts",
			inst: &InstrumentConfig{
				ticker: "A",
				context: []TimeframeConfig{
					{
						interval: testInterval,
						candles: []types.Candle{
							mockCandle(1, newTime(0), testInterval), // closes at t=1
							mockCandle(2, newTime(1), testInterval), // closes at t=2
						},
					},
					{
						interval: types.FiveMinutes, // e.g. 5m
						candles: []types.Candle{
							mockCandle(10, newTime(0), types.FiveMinutes), // closes at t=5
						},
					},
				},
			},
			curTime:    newTime(2),
			preIndexes: nil,
			want: map[types.Interval][]types.Candle{
				testInterval: {
					mockCandle(1, newTime(0), testInterval),
					mockCandle(2, newTime(1), testInterval),
				},
				types.FiveMinutes: {},
			},
		},
		{
			name: "ticker map missing does not panic and still returns context",
			inst: &InstrumentConfig{
				ticker: "A",
				context: []TimeframeConfig{{
					interval: testInterval,
					candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
					},
				}},
			},
			curTime:    newTime(1),
			preIndexes: map[string]map[types.Interval]int{}, // b.contextFeedIndex empty map, no "A"
			want: map[types.Interval][]types.Candle{
				testInterval: {
					mockCandle(1, newTime(0), testInterval),
				},
			},
		},
		{
			name: "two context intervals: fast has one candle closed, slow has none yet",
			inst: &InstrumentConfig{
				ticker: "A",
				context: []TimeframeConfig{
					{
						interval: testInterval, // e.g. 1m
						candles: []types.Candle{
							mockCandle(1, newTime(0), testInterval),
							mockCandle(2, newTime(1), testInterval),
						},
					},
					{
						interval: types.FiveMinutes,
						candles: []types.Candle{
							mockCandle(10, newTime(0), types.FiveMinutes),
						},
					},
				},
			},
			curTime:    newTime(1), // at t=1: 1m candle@t=0 is closed, 5m candle@t=0 is NOT closed
			preIndexes: nil,
			want: map[types.Interval][]types.Candle{
				testInterval: {
					mockCandle(1, newTime(0), testInterval),
				},
				types.FiveMinutes: {}, // not closed yet
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &backtester{
				contextFeedIndex: make(map[string]map[types.Interval]int),
			}

			for ticker, m := range tt.preIndexes {
				b.contextFeedIndex[ticker] = make(map[types.Interval]int)
				for interval, idx := range m {
					b.contextFeedIndex[ticker][interval] = idx
				}
			}

			got := b.buildInstrumentContext(tt.inst, tt.curTime)

			if len(got) != len(tt.want) {
				t.Fatalf("got %d intervals, want %d", len(got), len(tt.want))
			}

			for interval, wantCandles := range tt.want {
				gotCandles, ok := got[interval]
				if !ok {
					t.Fatalf("missing interval %v in result", interval)
				}

				if len(gotCandles) != len(wantCandles) {
					t.Fatalf("interval %v: got %d candles, want %d",
						interval, len(gotCandles), len(wantCandles))
				}

				for i := range wantCandles {
					g := gotCandles[i]
					w := wantCandles[i]

					if !g.Timestamp.Equal(w.Timestamp) {
						t.Fatalf("interval %v candle %d: timestamp mismatch, got %v want %v",
							interval, i, g.Timestamp, w.Timestamp)
					}
					if g.Interval != w.Interval {
						t.Fatalf("interval %v candle %d: interval mismatch, got %v want %v",
							interval, i, g.Interval, w.Interval)
					}
				}
			}
		})
	}
}

func TestBacktest_AllInstrumentContextsSendSameTimestampPerTick(t *testing.T) {
	tests := []struct {
		name        string
		feeds       []*InstrumentConfig
		wantContext map[types.Interval][][]types.Candle
	}{
		{
			name: "instrument with no context",
			feeds: []*InstrumentConfig{
				{ticker: "A", interval: testInterval, start: newTime(0), end: newTime(2), primary: TimeframeConfig{candles: nil}},
			},
			wantContext: make(map[types.Interval][][]types.Candle),
		},
		{
			name: "single interval context accumulates per tick",
			feeds: []*InstrumentConfig{
				{
					ticker: "A",
					interval: testInterval,
					start:  newTime(0),
					end:    newTime(2),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
						mockCandle(1, newTime(1), testInterval),
					}},
					context: []TimeframeConfig{{
						interval: testInterval,
						candles: []types.Candle{
							mockCandle(99, newTime(0), testInterval),
							mockCandle(99, newTime(1), testInterval),
						},
					}},
				},
			},
			wantContext: map[types.Interval][][]types.Candle{
				testInterval: {
					{
						mockCandle(99, newTime(0), testInterval),
					},
					{
						mockCandle(99, newTime(0), testInterval),
						mockCandle(99, newTime(1), testInterval),
					},
				},
			},
		},
		{
			name: "slow interval stays empty until candle closes",
			feeds: []*InstrumentConfig{
				{
					ticker: "A",
					interval: testInterval,
					start:  newTime(0),
					end:    newTime(5),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
						mockCandle(1, newTime(1), testInterval),
						mockCandle(1, newTime(2), testInterval),
						mockCandle(1, newTime(3), testInterval),
						mockCandle(1, newTime(4), testInterval),
					}},
					context: []TimeframeConfig{{
						interval: types.FiveMinutes,
						candles: []types.Candle{
							mockCandle(99, newTime(0), types.FiveMinutes),
						},
					}},
				},
			},
			wantContext: map[types.Interval][][]types.Candle{
				types.FiveMinutes: {
					{},
					{},
					{},
					{},
					{
						mockCandle(99, newTime(0), types.FiveMinutes),
					},
				},
			},
		},
		{
			name: "multiple context intervals tracked independently",
			feeds: []*InstrumentConfig{
				{
					ticker: "A",
					interval: testInterval,
					start:  newTime(0),
					end:    newTime(3),
					primary: TimeframeConfig{candles: []types.Candle{
						mockCandle(1, newTime(0), testInterval),
						mockCandle(1, newTime(1), testInterval),
						mockCandle(1, newTime(2), testInterval),
					}},
					context: []TimeframeConfig{
						{
							interval: testInterval,
							candles: []types.Candle{
								mockCandle(99, newTime(0), testInterval),
								mockCandle(99, newTime(1), testInterval),
								mockCandle(99, newTime(2), testInterval),
							},
						},
						{
							interval: types.ThreeMinutes,
							candles: []types.Candle{
								mockCandle(77, newTime(0), types.ThreeMinutes),
							},
						},
					},
				},
			},
			wantContext: map[types.Interval][][]types.Candle{
				testInterval: {
					{
						mockCandle(99, newTime(0), testInterval),
					},
					{
						mockCandle(99, newTime(0), testInterval),
						mockCandle(99, newTime(1), testInterval),
					},
					{
						mockCandle(99, newTime(0), testInterval),
						mockCandle(99, newTime(1), testInterval),
						mockCandle(99, newTime(2), testInterval),
					},
				},
				types.ThreeMinutes: {
					{},
					{},
					{
						mockCandle(77, newTime(0), types.ThreeMinutes),
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			strat := newCandlesReceivedStrategy()
			testAllocator := &mockAllocator{}
			testBroker := &mockBroker{}
			engine := mockEngine(strat, tt.feeds, testAllocator, testBroker)

			err := engine.backtester.run()
			if err != nil {
				t.Errorf("Error running engine: %v", err)
			}

			for interval, wantContextsByInterval := range tt.wantContext {
				contextsReceivedByInterval, ok := strat.contextsReceived[interval]
				if !ok {
					t.Fatalf("expected to receive a context for interval %v but got none", interval)
				}

				if len(contextsReceivedByInterval) != len(wantContextsByInterval) {
					t.Fatalf("interval %v: got %d context calls, wantIndex %d",
						interval, len(contextsReceivedByInterval), len(wantContextsByInterval))
				}

				// Contexts send in chronological order
				for callIdx := range wantContextsByInterval {
					gotCtx := contextsReceivedByInterval[callIdx]
					wantCtx := wantContextsByInterval[callIdx]

					if len(gotCtx) != len(wantCtx) {
						t.Fatalf("interval %v call %d: got %d candles, wantIndex %d",
							interval, callIdx, len(gotCtx), len(wantCtx))
					}

					for curWantCandlesIdx := range wantCtx {
						gotCandles := gotCtx[curWantCandlesIdx]
						wantCandles := wantCtx[curWantCandlesIdx]

						if !gotCandles.Timestamp.Equal(wantCandles.Timestamp) {
							t.Fatalf(
								"interval %v call %d candle %d: timestamp mismatch, got %v wantIndex %v",
								interval, callIdx, curWantCandlesIdx,
								gotCandles.Timestamp, wantCandles.Timestamp,
							)
						}

						if gotCandles.Interval != wantCandles.Interval {
							t.Fatalf(
								"interval %v call %d candle %d: interval mismatch, got %v wantIndex %v",
								interval, callIdx, curWantCandlesIdx,
								gotCandles.Interval, wantCandles.Interval,
							)
						}
					}
				}
			}

			// Check if there are contexts we did not ask for
			for interval, gotPerCall := range strat.contextsReceived {
				if _, expected := tt.wantContext[interval]; !expected && len(gotPerCall) > 0 {
					t.Fatalf("received unexpected contexts for interval %v: %d calls", interval, len(gotPerCall))
				}
			}

		})
	}
}

func TestBacktester_moveInstrumentContextCandleIndex(t *testing.T) {
	// Helper to build candles with just Timestamp populated (other fields irrelevant for this function)
	candle := func(ts time.Time) types.Candle {
		return types.Candle{
			Open:      decimal.Zero,
			Close:     decimal.Zero,
			High:      decimal.Zero,
			Low:       decimal.Zero,
			Volume:    decimal.Zero,
			Timestamp: ts,
		}
	}

	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	// Make 5 one-minute candles starting at base
	// Candle i: Timestamp = base + i minutes
	candles1m := []types.Candle{
		candle(base.Add(0 * time.Minute)),
		candle(base.Add(1 * time.Minute)),
		candle(base.Add(2 * time.Minute)),
		candle(base.Add(3 * time.Minute)),
		candle(base.Add(4 * time.Minute)),
	}

	b := &backtester{}

	tests := []struct {
		name      string
		candles   []types.Candle
		interval  types.Interval
		curTime   time.Time
		curIndex  int
		wantIndex int
	}{
		{
			name:      "empty candles returns 0",
			candles:   nil,
			interval:  types.OneMinute,
			curTime:   base,
			curIndex:  0,
			wantIndex: 0,
		},
		{
			name:      "negative curIndex clamps to 0",
			candles:   candles1m,
			interval:  types.OneMinute,
			curTime:   base.Add(0*time.Minute + 30*time.Second),
			curIndex:  -10,
			wantIndex: 0,
		},
		{
			name:      "curIndex == len(candles) returns len(candles)",
			candles:   candles1m,
			interval:  types.OneMinute,
			curTime:   base.Add(100 * time.Minute),
			curIndex:  len(candles1m),
			wantIndex: len(candles1m),
		},
		{
			name:      "curIndex > len(candles) returns curIndex unchanged",
			candles:   candles1m,
			interval:  types.OneMinute,
			curTime:   base.Add(100 * time.Minute),
			curIndex:  len(candles1m) + 7,
			wantIndex: len(candles1m),
		},
		{
			name:     "does not advance when first candle closeTime is after curTime",
			candles:  candles1m,
			interval: types.OneMinute,
			// Candle[0] closes at base+1m; curTime is before that
			curTime:   base.Add(59 * time.Second),
			curIndex:  0,
			wantIndex: 0,
		},
		{
			name:     "advances exactly one when curTime equals first candle closeTime",
			candles:  candles1m,
			interval: types.OneMinute,
			// Candle[0] closeTime == base+1m; After(curTime) is false when equal, so it advances
			curTime:   base.Add(1 * time.Minute),
			curIndex:  0,
			wantIndex: 1,
		},
		{
			name:     "advances multiple candles until next closeTime is after curTime",
			candles:  candles1m,
			interval: types.OneMinute,
			// closeTimes: [1m,2m,3m,4m,5m]; curTime=3m means it should advance through idx 0,1,2 => nextIdx=3
			curTime:   base.Add(3 * time.Minute),
			curIndex:  0,
			wantIndex: 3,
		},
		{
			name:     "starting from middle index only advances from curIndex forward",
			candles:  candles1m,
			interval: types.OneMinute,
			// From index 2: candle[2] closes at 3m (<= curTime), candle[3] closes at 4m (> curTime)
			curTime:   base.Add(3 * time.Minute),
			curIndex:  2,
			wantIndex: 3,
		},
		{
			name:     "advances to end when curTime is after all closeTimes",
			candles:  candles1m,
			interval: types.OneMinute,
			// last candle starts at 4m, closes at 5m
			curTime:   base.Add(10 * time.Minute),
			curIndex:  0,
			wantIndex: len(candles1m),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := b.findInstrumentContextCandleIndex(tt.candles, tt.interval, tt.curTime, tt.curIndex)
			if got != tt.wantIndex {
				t.Fatalf("findInstrumentContextCandleIndex(...) = %d, wantIndex %d", got, tt.wantIndex)
			}
		})
	}
}

func TestBacktest_ShouldSendCandlesInOrder(t *testing.T) {
	testStrat := newCandlesReceivedStrategy()
	testAllocator := &mockAllocator{}
	testBroker := &mockBroker{}
	engine := mockEngine(testStrat, mockInstrument(), testAllocator, testBroker)

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
		args      []*InstrumentConfig
		wantStart time.Time
		wantEnd   time.Time
	}{
		{
			name:      "should return 0",
			args:      Instruments(),
			wantStart: time.UnixMilli(0),
			wantEnd:   time.UnixMilli(0),
		},
		{
			name: "should find min and max in first feed",
			args: Instruments(
				&InstrumentConfig{ticker: "AAPL", interval: testInterval, start: time.UnixMilli(1), end: time.UnixMilli(2)},
			),
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(2),
		},
		{
			name: "should find min in first and max in second feed",
			args: Instruments(
				&InstrumentConfig{ticker: "AAPL", interval: testInterval, start: time.UnixMilli(1), end: time.UnixMilli(2)},
				&InstrumentConfig{ticker: "AAPL", interval: testInterval, start: time.UnixMilli(2), end: time.UnixMilli(3)},
			),
			wantStart: time.UnixMilli(1),
			wantEnd:   time.UnixMilli(3),
		},
		{
			name: "should find min in second and max in first feed",
			args: Instruments(
				&InstrumentConfig{ticker: "AAPL", interval: testInterval, start: time.UnixMilli(3), end: time.UnixMilli(6)},
				&InstrumentConfig{ticker: "AAPL", interval: testInterval, start: time.UnixMilli(1), end: time.UnixMilli(2)},
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

	feed := &InstrumentConfig{
		ticker:  "BTCUSDT",
		primary: TimeframeConfig{candles: candles},
		start:   candles[0].Timestamp,
		end:     candles[len(candles)-1].Timestamp,
	}

	tests := []struct {
		name      string
		feeds     []*InstrumentConfig
		feedIndex map[string]int
		ticker    string
		want      decimal.Decimal
	}{
		{
			name:      "normal case (index 1 -> use candle[0])",
			feeds:     []*InstrumentConfig{feed},
			feedIndex: map[string]int{"BTCUSDT": 1},
			ticker:    "BTCUSDT",
			want:      decimal.NewFromInt(100),
		},
		{
			name:      "index 0 -> idx - 1 = -1 -> clamped to 0",
			feeds:     []*InstrumentConfig{feed},
			feedIndex: map[string]int{"BTCUSDT": 0},
			ticker:    "BTCUSDT",
			want:      decimal.NewFromInt(100),
		},
		{
			name:      "index beyond end -> clamped to last candle",
			feeds:     []*InstrumentConfig{feed},
			feedIndex: map[string]int{"BTCUSDT": 10},
			ticker:    "BTCUSDT",
			want:      decimal.NewFromInt(300),
		},
		{
			name:      "ticker not found -> decimal.Zero",
			feeds:     []*InstrumentConfig{feed},
			feedIndex: map[string]int{},
			ticker:    "ETHUSDT",
			want:      decimal.Zero,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &backtester{
				instruments:         tt.feeds,
				instrumentFeedIndex: tt.feedIndex,
			}

			got := b.getLastPriceForTicker(tt.ticker)
			if !got.Equal(tt.want) {
				t.Fatalf("getLastPriceForTicker(%q) = %s, wantIndex %s",
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

func newTime(i int) time.Time {
	base := time.UnixMilli(0).UTC()
	return base.Add(time.Duration(i) * time.Minute)
}

func mockInstrument() []*InstrumentConfig {
	return Instruments(
		Instrument("AAPL", time.UnixMilli(0), time.UnixMilli(0).Add(types.IntervalToTime[testInterval]*time.Duration(5)), testInterval))
}

func mockEngine(strat strategy, feeds []*InstrumentConfig, allocator allocator, broker broker) *Engine {
	db := mockDb{
		assets: make(map[string]*types.Asset),
	}
	newPortfolio := NewPortfolioConfig(decimal.NewFromInt(100000), false)
	executionConfig := NewExecutionConfig(types.OneMinute, 1, 1)
	for i, feed := range feeds {
		db.assets[feed.ticker] = &types.Asset{
			Id:     i,
			Ticker: feed.ticker,
			Type:   types.AssetTypeStock,
		}
		executionConfig.candles[feed.ticker] = feed.primary.candles
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

func (m *mockAllocator) Allocate(signals map[string][]types.Signal, view types.PortfolioView) []types.Order {
	m.callCount++
	return nil
}

type mockDb struct {
	assets map[string]*types.Asset
}

func (m mockDb) GetAssetByTicker(ticker string, ctx context.Context) (*types.Asset, error) {
	return m.assets[ticker], nil
}

func mockCandle(assetId int, ts time.Time, interval types.Interval) types.Candle {
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
	// list of lists since we send a full context on each onCandle
	contextsReceived map[types.Interval][][]types.Candle
}

func newCandlesReceivedStrategy() *candlesReceivedStrategy {
	return &candlesReceivedStrategy{contextsReceived: make(map[types.Interval][][]types.Candle)}
}

func (t *candlesReceivedStrategy) Init(api PortfolioApi) error {
	return nil
}
func (t *candlesReceivedStrategy) OnCandle(candle types.Candle, contexts map[types.Interval][]types.Candle) []types.Signal {
	t.receivedCandles = append(t.receivedCandles, candle)
	t.receivedCount++
	for interval, candles := range contexts {
		curContexts := t.contextsReceived[interval]
		curContexts = append(curContexts, candles)
		t.contextsReceived[interval] = curContexts
	}
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
func (s *candlesParallelismStrategy) OnCandle(candle types.Candle, contexts map[types.Interval][]types.Candle) []types.Signal {
	// Send the candle
	s.sent <- candle
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

func (a *allocatorStrategy) OnCandle(candle types.Candle, contexts map[types.Interval][]types.Candle) []types.Signal {
	var signals []types.Signal
	if a.allocatorCalled < a.callAllocator {
		for i := range a.callAllocator {
			signals = append(signals, types.Signal{CreatedAt: time.UnixMilli(int64(i))})
		}
		a.allocatorCalled++
	}
	return signals
}

// tickerTaggingStrategy emits a single Signal per candle whose CreatedAt
// encodes the candle's AssetId, so we can later verify that signals were
// grouped under the correct ticker.
type tickerTaggingStrategy struct{}

func (s *tickerTaggingStrategy) Init(api PortfolioApi) error {
	return nil
}

func (s *tickerTaggingStrategy) OnCandle(candle types.Candle, contexts map[types.Interval][]types.Candle) []types.Signal {
	return []types.Signal{
		{
			// Use AssetId as a tag via UnixMilli so we can assert later.
			CreatedAt: time.UnixMilli(int64(candle.AssetId)),
		},
	}
}

// recordingAllocator records every signals map it receives from the backtester.
type recordingAllocator struct {
	calls []map[string][]types.Signal
}

func (a *recordingAllocator) Init(api PortfolioApi) error {
	return nil
}

func (a *recordingAllocator) Allocate(signals map[string][]types.Signal, view types.PortfolioView) []types.Order {
	cp := make(map[string][]types.Signal, len(signals))
	for k, v := range signals {
		sigsCopy := make([]types.Signal, len(v))
		copy(sigsCopy, v)
		cp[k] = sigsCopy
	}
	a.calls = append(a.calls, cp)
	return nil
}
