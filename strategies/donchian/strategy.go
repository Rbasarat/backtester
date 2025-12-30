package donchian

import (
	"backtester/internal/engine"
	"backtester/types"
	"fmt"

	"github.com/shopspring/decimal"
)

type Strategy struct {
	portfolio engine.PortfolioApi
}

func (s *Strategy) Init(api engine.PortfolioApi) error {
	s.portfolio = api
	return nil
}

func (s *Strategy) OnCandle(candle types.Candle, contexts map[types.Interval][]types.Candle) []types.Signal {

	if len(contexts[types.Day]) < 20 {
		return nil
	}

	last20Days := contexts[types.Day][len(contexts[types.Day])-20:]
	highestHigh, _ := donchianHighLow(last20Days)
	_, lowestLow := donchianHighLow(contexts[types.Day][len(contexts[types.Day])-10:])

	var signals []types.Signal

	if candle.Close.GreaterThan(highestHigh) {
		signals = append(signals, types.NewSignal(
			candle.Ticker,
			types.SideTypeBuy,
			highestHigh, // breakout level
			"Break of highest weekly high of preceding 4 weeks",
			candle.Timestamp,
		))
	}

	if candle.Close.LessThan(lowestLow) {
		signals = append(signals, types.NewSignal(
			candle.Ticker,
			types.SideTypeSell,
			lowestLow, // breakout level
			"Break of lowest weekly low of preceding 4 weeks",
			candle.Timestamp,
		))
	}
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

func movingAverageClose(candles []types.Candle) decimal.Decimal {
	var sum []decimal.Decimal
	for _, c := range candles {
		sum = append(sum, c.Close)
	}

	return decimal.Avg(sum[0], sum[1:]...)
}

func ATRWilderOnClose(candles []types.Candle, period int) ([]decimal.Decimal, error) {
	if period <= 0 {
		return nil, fmt.Errorf("period must be > 0")
	}
	n := len(candles)
	if n == 0 {
		return []decimal.Decimal{}, nil
	}
	if n < period+1 {
		// Need at least period+1 candles to have period TR values (starting at index 1)
		out := make([]decimal.Decimal, n)
		return out, nil
	}

	out := make([]decimal.Decimal, n) // default decimal.Zero values

	per := decimal.NewFromInt(int64(period))
	perMinus1 := decimal.NewFromInt(int64(period - 1))

	// Helper abs for decimal
	abs := func(x decimal.Decimal) decimal.Decimal {
		if x.IsNegative() {
			return x.Neg()
		}
		return x
	}

	// Compute TRs (we only have TR from i=1..n-1)
	tr := make([]decimal.Decimal, n)
	for i := 1; i < n; i++ {
		hl := candles[i].High.Sub(candles[i].Low)

		prevClose := candles[i-1].Close
		hpc := abs(candles[i].High.Sub(prevClose))
		lpc := abs(candles[i].Low.Sub(prevClose))

		// max(hl, hpc, lpc)
		m := hl
		if hpc.GreaterThan(m) {
			m = hpc
		}
		if lpc.GreaterThan(m) {
			m = lpc
		}
		tr[i] = m
	}

	// First ATR is SMA of TR[1.. period], aligned at candle index = period
	sum := decimal.Zero
	for i := 1; i <= period; i++ {
		sum = sum.Add(tr[i])
	}
	firstATR := sum.Div(per)
	out[period] = firstATR

	// Wilder smoothing for subsequent candles
	prevATR := firstATR
	for i := period + 1; i < n; i++ {
		// ATR = (prevATR*(period-1) + TR[i]) / period
		prevATR = prevATR.Mul(perMinus1).Add(tr[i]).Div(per)
		out[i] = prevATR
	}
	return out, nil
}
