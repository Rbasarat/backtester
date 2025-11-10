package engine

import (
	"backtester/types"
	"time"
)

type backtester struct {
	feeds     []*DataFeedConfig
	strategy  strategy
	allocator allocator
	broker    broker
	portfolio *types.Portfolio

	start     time.Time
	curTime   time.Time
	end       time.Time
	feedIndex map[string]int
}

func newBacktester(feeds []*DataFeedConfig, strat strategy, sizing allocator, broker broker, portfolio *types.Portfolio) *backtester {
	start, end := getGlobalTimeRange(feeds)
	return &backtester{
		start:     start,
		end:       end,
		curTime:   start,
		feeds:     feeds,
		strategy:  strat,
		allocator: sizing,
		broker:    broker,
		portfolio: portfolio,
		feedIndex: make(map[string]int),
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
		}

		orders := b.allocator.Allocate(signals, b.portfolio.GetPortfolioSnapshot())
		b.broker.Execute(orders)

		// We use time.Minute here because the lowest timeframe we have is minute
		b.curTime = b.curTime.Add(time.Minute)
	}
	return nil
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
