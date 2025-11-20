package engine

import (
	"backtester/types"
	"time"
)

type backtester struct {
	feeds           []*DataFeedConfig
	executionConfig *ExecutionConfig
	portfolioConfig *PortfolioConfig
	strategy        strategy
	allocator       allocator
	broker          broker
	portfolio       *portfolio

	start          time.Time
	curTime        time.Time
	end            time.Time
	feedIndex      map[string]int
	executionIndex map[string]int
}

func newBacktester(feeds []*DataFeedConfig, executionConfig *ExecutionConfig, portfolioConfig *PortfolioConfig, strat strategy, sizing allocator, broker broker, portfolio *portfolio) *backtester {
	start, end := getGlobalTimeRange(feeds)

	return &backtester{
		start:           start,
		end:             end,
		curTime:         start,
		feeds:           feeds,
		executionConfig: executionConfig,
		portfolioConfig: portfolioConfig,
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
			i := b.feedIndex[feed.ticker]
			if i >= len(feed.candles) {
				continue
			}
			curCandle := feed.candles[i]
			if curCandle.Timestamp.Equal(b.curTime) {
				signals = append(signals, b.strategy.OnCandle(curCandle)...)
				b.feedIndex[feed.ticker]++
			}
			b.executionIndex[feed.ticker] = advanceFeedIndex(b.executionConfig.candles[feed.ticker], b.executionIndex[feed.ticker], b.curTime)
		}

		orders := b.allocator.Allocate(signals, b.portfolio.GetPortfolioSnapshot())
		executions := b.broker.Execute(orders, b.getExecutionContext())
		err := b.portfolio.processExecutions(executions)
		if err != nil {
			return err
		}

		// Create a snapshot of the portfolio every day
		if b.curTime.Hour() == 0 && b.curTime.Minute() == 0 {
			curSnapshot := b.portfolio.GetPortfolioSnapshot()
			curSnapshot.Time = b.curTime
			b.portfolio.snapshots = append(b.portfolio.snapshots, curSnapshot)
		}

		// We use time.Minute here because the lowest timeframe we have is minute
		b.curTime = b.curTime.Add(time.Minute)
	}
	return nil
}

func (b *backtester) getExecutionContext() types.ExecutionContext {
	ctx := types.ExecutionContext{CurTime: b.curTime}
	candlesMap := make(map[string][]types.Candle)
	for ticker, feed := range b.executionConfig.candles {
		start := b.executionIndex[ticker] - b.executionConfig.barsBefore
		end := b.executionIndex[ticker] + b.executionConfig.barsAfter
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
		candlesMap[ticker] = candles
	}
	ctx.Candles = candlesMap
	ctx.Portfolio = b.portfolio.GetPortfolioSnapshot()
	return ctx
}

func getGlobalTimeRange(feeds []*DataFeedConfig) (time.Time, time.Time) {
	if len(feeds) == 0 {
		return time.UnixMilli(0), time.UnixMilli(0)
	}

	minStart := feeds[0].start
	maxEnd := feeds[0].end

	for _, f := range feeds[1:] {
		if f.start.Before(minStart) {
			minStart = f.start
		}
		if f.end.After(maxEnd) {
			maxEnd = f.end
		}
	}
	return minStart, maxEnd
}

func (b *backtester) getCurrentTime() time.Time {
	return b.curTime
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
