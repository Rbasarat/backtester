package engine

import (
	"backtester/types"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/shopspring/decimal"
)

type backtester struct {
	instruments     []*InstrumentConfig
	executionConfig *ExecutionConfig
	portfolioConfig *PortfolioConfig
	strategy        strategy
	allocator       allocator
	broker          broker
	portfolio       *portfolio

	start               time.Time
	curTime             time.Time
	end                 time.Time
	instrumentFeedIndex map[string]int
	contextFeedIndex    map[string]map[types.Interval]int
	executionIndex      map[string]int
}

func newBacktester(feeds []*InstrumentConfig, executionConfig *ExecutionConfig, portfolioConfig *PortfolioConfig, strat strategy, sizing allocator, broker broker, portfolio *portfolio) *backtester {
	start, end := getGlobalTimeRange(feeds)
	feedIndex := make(map[string]int)
	executionIndex := make(map[string]int)
	contextFeedIndex := make(map[string]map[types.Interval]int)
	for _, feed := range feeds {
		feedIndex[feed.ticker] = 0
		executionIndex[feed.ticker] = -1
		for _, config := range feed.context {
			contextFeedIndex[feed.ticker] = make(map[types.Interval]int)
			contextFeedIndex[feed.ticker][config.interval] = 0
		}
	}

	return &backtester{
		start:               start,
		end:                 end,
		curTime:             start,
		instruments:         feeds,
		executionConfig:     executionConfig,
		portfolioConfig:     portfolioConfig,
		strategy:            strat,
		allocator:           sizing,
		broker:              broker,
		portfolio:           portfolio,
		instrumentFeedIndex: feedIndex,
		contextFeedIndex:    contextFeedIndex,
		executionIndex:      executionIndex,
	}
}

func (b *backtester) run() error {
	bar := initProgressBar(int(b.end.Sub(b.start).Minutes()))
	for !b.curTime.After(b.end) {
		signals := make(map[string][]types.Signal)
		for _, instrument := range b.instruments {
			i := b.instrumentFeedIndex[instrument.ticker]
			if i >= len(instrument.primary.candles) {
				continue
			}
			curCandle := instrument.primary.candles[i]
			// Only send candles when they are fully closed.
			candleCloseTime := curCandle.Timestamp.Add(types.IntervalToTime[instrument.interval])
			if candleCloseTime.Equal(b.curTime) {
				curSignals := signals[instrument.ticker]
				curContexts := b.buildInstrumentContext(instrument, b.curTime)
				curSignals = append(curSignals, b.strategy.OnCandle(curCandle, curContexts)...)
				signals[instrument.ticker] = curSignals
				b.instrumentFeedIndex[instrument.ticker]++
			}
			b.executionIndex[instrument.ticker] = advanceFeedIndex(
				b.executionConfig.candles[instrument.ticker],
				b.executionIndex[instrument.ticker],
				b.curTime,
				b.executionConfig.interval,
			)
		}

		orders := b.allocator.Allocate(signals, b.portfolio.GetPortfolioSnapshot())
		executions := b.broker.Execute(orders, b.buildExecutionContext())
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

func (b *backtester) buildInstrumentContext(inst *InstrumentConfig, curTime time.Time) map[types.Interval][]types.Candle {
	out := make(map[types.Interval][]types.Candle)

	if b.contextFeedIndex[inst.ticker] == nil {
		b.contextFeedIndex[inst.ticker] = make(map[types.Interval]int)
	}

	for _, cfg := range inst.context {
		curIdx := b.contextFeedIndex[inst.ticker][cfg.interval]
		nextIdx := b.findInstrumentContextCandleIndex(cfg.candles, cfg.interval, curTime, curIdx)

		b.contextFeedIndex[inst.ticker][cfg.interval] = nextIdx
		out[cfg.interval] = cfg.candles[:nextIdx]
	}
	return out
}

func (b *backtester) findInstrumentContextCandleIndex(
	candles []types.Candle,
	interval types.Interval,
	curTime time.Time,
	curIndex int,
) int {

	if len(candles) == 0 {
		return 0
	}

	if curIndex < 0 {
		curIndex = 0
	}
	if curIndex > len(candles) {
		return len(candles)
	}

	nextIdx := curIndex
	for nextIdx < len(candles) {
		c := candles[nextIdx]
		closeTime := c.Timestamp.Add(types.IntervalToTime[interval])
		if closeTime.After(curTime) {
			break
		}
		nextIdx++
	}
	return nextIdx
}

// TODO we can use the moveidx for contexts function too for this I think..
func (b *backtester) buildExecutionContext() types.ExecutionContext {
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

func getGlobalTimeRange(feeds []*InstrumentConfig) (time.Time, time.Time) {
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
	for _, feed := range b.instruments {
		if feed.ticker != ticker || len(feed.primary.candles) == 0 {
			continue
		}
		idx := b.instrumentFeedIndex[ticker] - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(feed.primary.candles) {
			idx = len(feed.primary.candles) - 1
		}
		return feed.primary.candles[idx].Close
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
