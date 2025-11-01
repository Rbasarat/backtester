package engine

import (
	"backtester/types"
	"sync"
	"time"
)

type backtester struct {
	curTime  time.Time
	start    time.Time
	end      time.Time
	feeds    []*DataFeed
	strategy strategy
}

func newBacktester(feeds []*DataFeed, strat strategy) *backtester {
	start, end := getGlobalTimeRange(feeds)
	return &backtester{
		start:    start,
		end:      end,
		curTime:  start,
		feeds:    feeds,
		strategy: strat,
	}
}

func (e *backtester) run() error {
	feedCursor := make(map[string]int)

	wg := &sync.WaitGroup{}
	for !e.curTime.After(e.end) {
		for _, feed := range e.feeds {
			i := feedCursor[feed.Ticker]
			if i >= len(feed.candles) {
				continue
			}
			curCandle := feed.candles[i]
			if curCandle.Timestamp.Equal(e.curTime) {
				wg.Add(1)
				go func(candle types.Candle) {
					defer wg.Done()
					e.strategy.OnCandle(candle)
				}(curCandle)
				feedCursor[feed.Ticker]++
			}
		}
		wg.Wait()

		// We use time.Minute here because the lowest timeframe we have is minute
		e.curTime = e.curTime.Add(time.Minute)
	}
	return nil
}

func getGlobalTimeRange(feeds []*DataFeed) (time.Time, time.Time) {
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
