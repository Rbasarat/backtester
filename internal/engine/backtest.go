package engine

import (
	"backtester/types"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/shopspring/decimal"
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
	feedIndex := make(map[string]int)
	executionIndex := make(map[string]int)
	for _, feed := range feeds {
		feedIndex[feed.ticker] = 0
		executionIndex[feed.ticker] = -1
	}

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
		feedIndex:       feedIndex,
		executionIndex:  executionIndex,
	}
}

func (b *backtester) run() error {
	bar := initProgressBar(int(b.end.Sub(b.start).Minutes()))
	for !b.curTime.After(b.end) {
		signals := make(map[string][]types.Signal)
		for _, feed := range b.feeds {
			i := b.feedIndex[feed.ticker]
			if i >= len(feed.candles) {
				continue
			}
			curCandle := feed.candles[i]
			// Only send candles when they are fully closed.
			candleCloseTime := curCandle.Timestamp.Add(types.IntervalToTime[feed.interval])
			if candleCloseTime.Equal(b.curTime) {
				curSignals := signals[feed.ticker]
				curSignals = append(curSignals, b.strategy.OnCandle(curCandle)...)
				signals[feed.ticker] = curSignals
				b.feedIndex[feed.ticker]++
			}
			b.executionIndex[feed.ticker] = advanceFeedIndex(
				b.executionConfig.candles[feed.ticker],
				b.executionIndex[feed.ticker],
				b.curTime,
				b.executionConfig.interval,
			)
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
		bar.Add(1)
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
func (b *backtester) getLastPriceForTicker(ticker string) decimal.Decimal {
	for _, feed := range b.feeds {
		if feed.ticker != ticker || len(feed.candles) == 0 {
			continue
		}
		idx := b.feedIndex[ticker] - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(feed.candles) {
			idx = len(feed.candles) - 1
		}
		return feed.candles[idx].Close
	}
	return decimal.Zero
}

// Index only goes one way
func advanceFeedIndex(candles []types.Candle, prevIndex int, curTime time.Time, candleInterval types.Interval) int {
	if prevIndex < -1 {
		prevIndex = -1
	}

	nextIdx := prevIndex + 1
	if nextIdx < 0 {
		nextIdx = 0
	}

	candleDuration := types.IntervalToTime[candleInterval]

	for nextIdx < len(candles) {
		closeTime := candles[nextIdx].Timestamp.Add(candleDuration)
		if closeTime.After(curTime) {
			break
		}

		prevIndex = nextIdx
		nextIdx++
	}

	return prevIndex
}

func initProgressBar(maxTicks int) *progressbar.ProgressBar {
	return progressbar.NewOptions(maxTicks,
		//progressbar.OptionSetWriter(ansi.NewAnsiStdout()),
		progressbar.OptionEnableColorCodes(true),
		progressbar.OptionSetElapsedTime(true),
		progressbar.OptionShowElapsedTimeOnFinish(),
		progressbar.OptionSetDescription("Backtesting in progress..."),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "[green]=[reset]",
			SaucerHead:    "[green]>[reset]",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}))
}
