package engine

import (
	"backtester/types"
	"time"
)

type backtester struct {
	feeds           []*DataFeedConfig
	executionConfig ExecutionConfig
	strategy        strategy
	allocator       allocator
	broker          broker
	portfolio       *types.Portfolio

	start          time.Time
	curTime        time.Time
	end            time.Time
	feedIndex      map[string]int
	executionIndex map[string]int
}

func newBacktester(feeds []*DataFeedConfig, executionConfig ExecutionConfig, strat strategy, sizing allocator, broker broker, portfolio *types.Portfolio) *backtester {
	start, end := getGlobalTimeRange(feeds)

	return &backtester{
		start:           start,
		end:             end,
		curTime:         start,
		feeds:           feeds,
		executionConfig: executionConfig,
		strategy:        strat,
		allocator:       sizing,
		broker:          broker,
		portfolio:       portfolio,
		feedIndex:       make(map[string]int),
		executionIndex:  make(map[string]int),
	}
}

func (b *backtester) run() error {
	for !b.curTime.After(b.end) {
		var signals []types.Signal
		for _, feed := range b.feeds {
			i := b.feedIndex[feed.Ticker]
			if i >= len(feed.candles) {
				continue
			}
			curCandle := feed.candles[i]
			if curCandle.Timestamp.Equal(b.curTime) {
				signals = append(signals, b.strategy.OnCandle(curCandle)...)
				b.feedIndex[feed.Ticker]++
			}
			b.executionIndex[feed.Ticker] = advanceFeedIndex(b.executionConfig.candles[feed.Ticker], b.executionIndex[feed.Ticker], b.curTime)
		}

		orders := b.allocator.Allocate(signals, b.portfolio.GetPortfolioSnapshot())
		b.broker.Execute(orders, b.getExecutionContext())

		// We use time.Minute here because the lowest timeframe we have is minute
		b.curTime = b.curTime.Add(time.Minute)
	}
	return nil
}

func (b *backtester) getExecutionContext() types.ExecutionContext {
	ctx := types.ExecutionContext{CurTime: b.curTime}
	candlesMap := make(map[string]map[time.Time]types.Candle)
	for ticker, feed := range b.executionConfig.candles {
		start := b.executionIndex[ticker] - b.executionConfig.BarsBefore
		end := b.executionIndex[ticker] + b.executionConfig.BarsAfter
		if start < 0 {
			start = 0
		}
		if end > len(feed) {
			end = len(feed)
		}
		if start > end {
			start = end
		}
		candles := feed[start:end]
		candlesMap[ticker] = createMapFromCandles(candles)
	}
	ctx.Candles = candlesMap
	return ctx
}

func createMapFromCandles(candles []types.Candle) map[time.Time]types.Candle {
	candlesToTime := make(map[time.Time]types.Candle)
	for _, candle := range candles {
		candlesToTime[candle.Timestamp] = candle
	}
	return candlesToTime
}

func getGlobalTimeRange(feeds []*DataFeedConfig) (time.Time, time.Time) {
	if len(feeds) == 0 {
		return time.UnixMilli(0), time.UnixMilli(0)
	}

	minStart := feeds[0].Start
	maxEnd := feeds[0].End

	for _, f := range feeds[1:] {
		if f.Start.Before(minStart) {
			minStart = f.Start
		}
		if f.End.After(maxEnd) {
			maxEnd = f.End
		}
	}
	return minStart, maxEnd
}

// Index only goes one way
func advanceFeedIndex(candles []types.Candle, curIndex int, curTime time.Time) int {
	for curIndex < len(candles) && !candles[curIndex].Timestamp.After(curTime) {
		curIndex++
	}
	// curIndex is now at the first candle *after* curTime, so latest usable is curIndex-1.
	if curIndex == 0 {
		return 0
	}
	return curIndex - 1
}
