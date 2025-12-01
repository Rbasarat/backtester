package donchian

import (
	"backtester/internal/engine"
	"backtester/types"

	"github.com/shopspring/decimal"
)

type Strategy struct {
	// store WEEKLY candles per ticker
	history   map[string][]types.Candle
	portfolio engine.PortfolioApi
	stopLoss  map[string]decimal.Decimal
}

func (s *Strategy) Init(api engine.PortfolioApi) error {
	s.portfolio = api
	s.history = make(map[string][]types.Candle)
	s.stopLoss = make(map[string]decimal.Decimal)
	return nil
}

func (s *Strategy) OnCandle(candle types.Candle) []types.Signal {
	hist := s.history[candle.Ticker]
	hist = append(hist, candle)
	s.history[candle.Ticker] = hist

	// Need at least 5 weekly candles:
	//  - 4 *completed* weeks for the channel
	//  - current week for possible breakout
	if len(hist) < 21 {
		return nil
	}

	// Last 4 COMPLETED weeks (excluding current)
	last4Weeks := hist[len(hist)-21 : len(hist)-1]
	highestHigh, lowestLow := donchianHighLow(last4Weeks)

	var signals []types.Signal

	// BUY SETUP & ENTRY:
	// "Buy a break of the highest weekly high of the preceding 4 weeks"
	// Use the breakout level as the signal price.
	if candle.High.GreaterThan(highestHigh) {
		signals = append(signals, types.NewSignal(
			candle.Ticker,
			types.SideTypeBuy,
			highestHigh, // breakout level
			"Break of highest weekly high of preceding 4 weeks (entry/stop-and-reverse BUY)",
			candle.Timestamp,
		))
		// Also set the stop loss
		s.stopLoss[candle.Ticker] = candle.Close.Sub(calcATR(hist, 20).Mul(decimal.NewFromFloat(2)))
	}

	if candle.Low.LessThan(lowestLow) {
		signals = append(signals, types.NewSignal(
			candle.Ticker,
			types.SideTypeSell,
			lowestLow, // breakout level
			"Break of lowest weekly low of preceding 4 weeks (entry/stop-and-reverse SELL)",
			candle.Timestamp,
		))
		// Reset/flip stop loss here if you set one for shorts
		s.stopLoss[candle.Ticker] = decimal.Zero
	}
	//else if candle.Close.LessThan(s.stopLoss[candle.Ticker]) {
	//	// ATR stop-loss exit
	//	signals = append(signals, types.NewSignal(
	//		candle.Ticker,
	//		types.SideTypeSell,
	//		s.stopLoss[candle.Ticker], // stop level
	//		"ATR(20) stop-loss SELL",
	//		candle.Timestamp,
	//	))
	//	s.stopLoss[candle.Ticker] = decimal.Zero
	//}

	return signals
}

// Utility: Donchian Channel High/Low
func donchianHighLow(candles []types.Candle) (decimal.Decimal, decimal.Decimal) {
	if len(candles) == 0 {
		return decimal.Zero, decimal.Zero
	}

	highest := candles[0].High
	lowest := candles[0].Low

	for _, c := range candles {
		if c.High.GreaterThan(highest) {
			highest = c.High
		}
		if c.Low.LessThan(lowest) {
			lowest = c.Low
		}
	}
	return highest, lowest
}

func calcATR(candles []types.Candle, period int) decimal.Decimal {
	if len(candles) < period+1 {
		return decimal.Zero // need enough data (prev candle + period)
	}

	var trueRanges []decimal.Decimal

	for i := 1; i < len(candles); i++ {
		high := candles[i].High
		low := candles[i].Low
		prevClose := candles[i-1].Close

		range1 := high.Sub(low)
		range2 := high.Sub(prevClose).Abs()
		range3 := low.Sub(prevClose).Abs()

		maxTrueRange := decimal.Max(range1, range2, range3)
		trueRanges = append(trueRanges, maxTrueRange)
	}

	atr := decimal.Zero
	for _, tr := range trueRanges[:period] {
		atr = atr.Add(tr)
	}
	atr = atr.Div(decimal.NewFromInt(int64(period)))

	for i := period; i < len(trueRanges); i++ {
		atr = (atr.Mul(decimal.NewFromInt(int64(period - 1))).Add(trueRanges[i])).
			Div(decimal.NewFromInt(int64(period)))
	}

	return atr
}
