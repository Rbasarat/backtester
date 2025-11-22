package engine

import (
	"backtester/types"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/shopspring/decimal"
)

type Report struct {
	// Meta / period info
	StartDate   time.Time
	TotalPeriod time.Duration
	TotalTrades int

	// Absolute performance
	NetProfit            decimal.Decimal
	NetAvgProfitPerTrade decimal.Decimal
	CAGR                 decimal.Decimal

	// Trade-level distribution metrics
	AvgWin  decimal.Decimal
	AvgLoss decimal.Decimal

	// Drawdown & loss streak metrics
	MaxDrawdown          decimal.Decimal
	MaxDrawdownPercent   decimal.Decimal
	MaxDrawdownDays      time.Duration
	MaxConsecutiveLosses int

	// Risk-adjusted metrics
	SharpeRatio  decimal.Decimal
	SortinoRatio decimal.Decimal
	ProfitFactor decimal.Decimal

	// Costs
	TotalFees decimal.Decimal

	// TODO: UPI (brent pentfold book)
}

type trade struct {
	buy  *types.ExecutionReport
	sell *types.ExecutionReport
}

func (e *Engine) printReport(report *Report) {

	fmt.Println("===== Trading Report =====")
	fmt.Printf("Start Date:            %s\n", report.StartDate.Format("2006-01-02"))
	fmt.Printf("Total Period:          %d days\n", report.TotalPeriod/(24*time.Hour))
	fmt.Printf("Total Trades:          %d\n", report.TotalTrades)

	fmt.Println("\n-- Absolute Performance --")
	fmt.Printf("Net Profit:            %s\n", report.NetProfit)
	fmt.Printf("Avg Profit/Trade:      %s\n", report.NetAvgProfitPerTrade)
	fmt.Printf("CAGR:                  %s\n", report.CAGR)

	fmt.Println("\n-- Trade-Level Metrics --")
	fmt.Printf("Avg Win:               %s\n", report.AvgWin)
	fmt.Printf("Avg Loss:              %s\n", report.AvgLoss)

	fmt.Println("\n-- Drawdown Metrics --")
	fmt.Printf("Max Drawdown:          %s\n", report.MaxDrawdown)
	fmt.Printf("Max Drawdown %%:        %s\n", report.MaxDrawdownPercent)
	fmt.Printf("Max Drawdown Days:     %v\n", report.MaxDrawdownDays)
	fmt.Printf("Max Consecutive Losses:%d\n", report.MaxConsecutiveLosses)

	fmt.Println("\n-- Risk-Adjusted Metrics --")
	fmt.Printf("Sharpe Ratio:          %s\n", report.SharpeRatio)
	fmt.Printf("Sortino Ratio:         %s\n", report.SortinoRatio)
	fmt.Printf("Profit Factor:         %s\n", report.ProfitFactor)

	fmt.Println("\n-- Costs --")
	fmt.Printf("Total Fees:            %s\n", report.TotalFees)

	fmt.Println("==========================")
}

func (e *Engine) generateReport(start, end time.Time, results *portfolio) *Report {
	trades := executionsToTrades(results)
	//
	report := &Report{}
	report.StartDate = start
	report.TotalPeriod = end.Sub(start).Truncate(time.Hour * 24)
	report.TotalTrades = len(trades)

	var wg sync.WaitGroup
	wg.Add(7)
	go func() {
		report.NetProfit = calcNetProfit(trades, &wg)
	}()
	go func() {
		report.NetAvgProfitPerTrade = calcNetAvgProfitPerTrade(trades, &wg)
	}()
	go func() {
		report.AvgWin, report.AvgLoss = calcAvgWinLossPerTrade(trades, &wg)
	}()
	go func() {
		report.CAGR = calcCAGR(results.snapshots, &wg)
	}()
	go func() {
		report.MaxDrawdown, report.MaxDrawdownPercent, report.MaxDrawdownDays = calcDrawdownMetrics(results.snapshots, &wg)
	}()
	go func() {
		report.MaxConsecutiveLosses = calcMaxConsecutiveLosses(trades, &wg)
	}()
	go func() {
		report.SharpeRatio = calcSharpeRatio(results.snapshots, e.reportingConfig.sharpeRiskFreeRate, &wg)
	}()
	wg.Wait()

	return report
}

func calcNetProfit(trades []trade, wg *sync.WaitGroup) decimal.Decimal {
	defer wg.Done()

	grossProfit := decimal.Zero
	totalFees := decimal.Zero

	for _, tr := range trades {
		hasBuy, hasSell := false, false
		curGrossProfit := decimal.Zero
		curFees := decimal.Zero

		processReport := func(report *types.ExecutionReport) {
			if report == nil {
				return
			}

			for _, fill := range report.Fills {
				curFees = curFees.Add(fill.Fee)
				value := fill.Quantity.Mul(fill.Price)

				switch report.Side {
				case types.SideTypeBuy:
					curGrossProfit = curGrossProfit.Sub(value)
					hasBuy = true
				case types.SideTypeSell:
					curGrossProfit = curGrossProfit.Add(value)
					hasSell = true
				}
			}
		}

		// process both legs (some trades may be partial: one of these is nil)
		processReport(tr.buy)
		processReport(tr.sell)

		// Only realize PnL when the trade has both sides
		if hasBuy && hasSell {
			grossProfit = grossProfit.Add(curGrossProfit)
		}

		// Always subtract fees, even for open trades
		totalFees = totalFees.Add(curFees)
	}

	return grossProfit.Sub(totalFees)
}

func calcNetAvgProfitPerTrade(trades []trade, wg *sync.WaitGroup) decimal.Decimal {
	defer wg.Done()

	grossProfit := decimal.Zero
	totalFees := decimal.Zero
	realizedTrades := 0

	for _, tr := range trades {
		hasBuy, hasSell := false, false
		curGrossProfit := decimal.Zero
		curFees := decimal.Zero

		processReport := func(report *types.ExecutionReport) {
			if report == nil {
				return
			}

			for _, fill := range report.Fills {
				curFees = curFees.Add(fill.Fee)
				value := fill.Quantity.Mul(fill.Price)

				switch report.Side {
				case types.SideTypeBuy:
					curGrossProfit = curGrossProfit.Sub(value)
					hasBuy = true
				case types.SideTypeSell:
					curGrossProfit = curGrossProfit.Add(value)
					hasSell = true
				}
			}
		}

		// process both legs (some trades may be partial: one of these is nil)
		processReport(tr.buy)
		processReport(tr.sell)

		if hasBuy && hasSell {
			grossProfit = grossProfit.Add(curGrossProfit)
			realizedTrades++
		}

		// Always take fees into account even if the trade is not closed
		totalFees = totalFees.Add(curFees)
	}

	if realizedTrades == 0 {
		return decimal.Zero
	}

	netProfit := grossProfit.Sub(totalFees)
	return netProfit.Div(decimal.NewFromInt(int64(realizedTrades)))
}

func calcCAGR(snapshots []types.PortfolioView, wg *sync.WaitGroup) decimal.Decimal {
	defer wg.Done()
	if len(snapshots) < 2 {
		return decimal.Zero
	}

	startSnap := snapshots[0]
	endSnap := snapshots[len(snapshots)-1]

	startVal := portfolioValue(startSnap)
	endVal := portfolioValue(endSnap)

	// If starting value is <= 0, CAGR is not well-defined
	if !startVal.GreaterThan(decimal.Zero) {
		return decimal.Zero
	}

	// time difference in years (using 365.25 days to account for leap years)
	duration := endSnap.Time.Sub(startSnap.Time)
	if duration <= 0 {
		return decimal.Zero
	}
	years := duration.Hours() / (24.0 * 365.25)

	if years <= 0 {
		return decimal.Zero
	}

	ratio := endVal.Div(startVal)

	if !ratio.GreaterThan(decimal.Zero) {
		return decimal.Zero
	}

	cagrFloat := math.Pow(ratio.InexactFloat64(), 1.0/years) - 1.0

	return decimal.NewFromFloat(cagrFloat)
}

func calcAvgWinLossPerTrade(trades []trade, wg *sync.WaitGroup) (decimal.Decimal, decimal.Decimal) {
	defer wg.Done()

	sumWins := decimal.Zero
	sumLosses := decimal.Zero // store absolute loss amounts
	winCount := 0
	lossCount := 0

	for _, tr := range trades {
		hasBuy, hasSell := false, false
		curGrossProfit := decimal.Zero
		curFees := decimal.Zero

		processReport := func(report *types.ExecutionReport) {
			if report == nil {
				return
			}

			for _, fill := range report.Fills {
				curFees = curFees.Add(fill.Fee)

				value := fill.Quantity.Mul(fill.Price)
				switch report.Side {
				case types.SideTypeBuy:
					curGrossProfit = curGrossProfit.Sub(value)
					hasBuy = true
				case types.SideTypeSell:
					curGrossProfit = curGrossProfit.Add(value)
					hasSell = true
				}
			}
		}

		processReport(tr.buy)
		processReport(tr.sell)

		// Only realized trades (have both a buy and a sell)
		if hasBuy && hasSell {
			net := curGrossProfit.Sub(curFees)

			switch {
			case net.GreaterThan(decimal.Zero):
				sumWins = sumWins.Add(net)
				winCount++
			case net.LessThan(decimal.Zero):
				sumLosses = sumLosses.Add(net.Abs())
				lossCount++
			}
		}
	}

	avgWin := decimal.Zero
	avgLoss := decimal.Zero

	if winCount > 0 {
		avgWin = sumWins.Div(decimal.NewFromInt(int64(winCount)))
	}
	if lossCount > 0 {
		avgLoss = sumLosses.Div(decimal.NewFromInt(int64(lossCount)))
	}

	return avgWin, avgLoss
}

func calcDrawdownMetrics(
	snapshots []types.PortfolioView,
	wg *sync.WaitGroup,
) (decimal.Decimal, decimal.Decimal, time.Duration) {
	defer wg.Done()

	if len(snapshots) == 0 {
		return decimal.Zero, decimal.Zero, 0
	}

	// Assume snapshots are in chronological order.
	// If not guaranteed, you should sort them by time first.

	peak := decimal.Zero
	var peakTime time.Time

	maxDD := decimal.Zero
	maxDDPct := decimal.Zero
	var maxDDDuration time.Duration

	for i, snap := range snapshots {
		equity := portfolioValue(snap)

		// Initialize peak with first snapshot that has a value
		if i == 0 || equity.GreaterThan(peak) || peak.IsZero() {
			peak = equity
			peakTime = snap.Time
		}

		if peak.GreaterThan(decimal.Zero) {
			dd := peak.Sub(equity) // absolute drawdown

			if dd.GreaterThan(maxDD) {
				maxDD = dd
				maxDDPct = dd.Div(peak)
				maxDDDuration = snap.Time.Sub(peakTime)
			}
		}
	}

	return maxDD, maxDDPct, maxDDDuration
}

func calcMaxConsecutiveLosses(trades []trade, wg *sync.WaitGroup) int {
	defer wg.Done()

	type tradeResult struct {
		closeTime time.Time
		netPnL    decimal.Decimal
	}

	var tradeResults []tradeResult

	for _, tr := range trades {
		hasBuy, hasSell := false, false
		curGrossProfit := decimal.Zero
		curFees := decimal.Zero
		var closeTime time.Time

		processReport := func(report *types.ExecutionReport) {
			if report == nil {
				return
			}

			// use the latest leg time as "close" time
			if report.ReportTime.After(closeTime) {
				closeTime = report.ReportTime
			}

			for _, fill := range report.Fills {
				curFees = curFees.Add(fill.Fee)

				value := fill.Quantity.Mul(fill.Price)
				switch report.Side {
				case types.SideTypeBuy:
					curGrossProfit = curGrossProfit.Sub(value)
					hasBuy = true
				case types.SideTypeSell:
					curGrossProfit = curGrossProfit.Add(value)
					hasSell = true
				}
			}
		}

		processReport(tr.buy)
		processReport(tr.sell)

		// Only realized trades (have both a buy and a sell)
		if hasBuy && hasSell && !closeTime.IsZero() {
			netPnL := curGrossProfit.Sub(curFees)
			tradeResults = append(tradeResults, tradeResult{
				closeTime: closeTime,
				netPnL:    netPnL,
			})
		}
	}

	// Sort realized trades by close time
	sort.Slice(tradeResults, func(i, j int) bool {
		return tradeResults[i].closeTime.Before(tradeResults[j].closeTime)
	})

	maxLossStreak := 0
	currentStreak := 0

	for _, tr := range tradeResults {
		if tr.netPnL.LessThan(decimal.Zero) {
			currentStreak++
			if currentStreak > maxLossStreak {
				maxLossStreak = currentStreak
			}
		} else {
			currentStreak = 0
		}
	}

	return maxLossStreak
}

func calcSharpeRatio(
	snapshots []types.PortfolioView,
	annualRiskFree decimal.Decimal,
	wg *sync.WaitGroup,
) decimal.Decimal {
	defer wg.Done()
	monthlyReturns := getMonthlyReturns(snapshots)
	if len(monthlyReturns) < 2 {
		// Need at least 2 months to compute stddev
		return decimal.Zero
	}

	// Convert annual risk-free to *monthly* risk-free:
	// rf_monthly = (1 + rf_annual)^(1/12) - 1
	rfAnnualFloat := annualRiskFree.InexactFloat64()
	rfMonthlyFloat := math.Pow(1.0+rfAnnualFloat, 1.0/12.0) - 1.0

	// Build slice of monthly *excess* returns in float64
	excess := make([]float64, 0, len(monthlyReturns))
	for _, r := range monthlyReturns {
		rFloat := r.InexactFloat64()
		excess = append(excess, rFloat-rfMonthlyFloat)
	}

	if len(excess) < 2 {
		return decimal.Zero
	}

	// Mean of monthly excess returns
	var sum float64
	for _, x := range excess {
		sum += x
	}
	meanMonthlyExcess := sum / float64(len(excess))

	// Sample standard deviation of monthly excess returns
	var varianceSum float64
	for _, x := range excess {
		diff := x - meanMonthlyExcess
		varianceSum += diff * diff
	}
	stdMonthly := math.Sqrt(varianceSum / float64(len(excess)-1))
	if stdMonthly == 0 {
		return decimal.Zero
	}

	// Monthly Sharpe, then annualize by sqrt(12)
	sharpeMonthly := meanMonthlyExcess / stdMonthly
	sharpeAnnual := sharpeMonthly * math.Sqrt(12.0)

	return decimal.NewFromFloat(sharpeAnnual)
}

func getMonthlyReturns(snapshots []types.PortfolioView) []decimal.Decimal {
	if len(snapshots) == 0 {
		return nil
	}

	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Time.Before(snapshots[j].Time)
	})

	type monthKey struct {
		year  int
		month time.Month
	}

	type monthBounds struct {
		first types.PortfolioView
		last  types.PortfolioView
	}

	months := make(map[monthKey]*monthBounds)

	// Find first/last snapshot in each calendar month
	for _, snap := range snapshots {
		y, m, _ := snap.Time.Date()
		key := monthKey{year: y, month: m}

		if b, ok := months[key]; !ok {
			months[key] = &monthBounds{
				first: snap,
				last:  snap,
			}
		} else {
			if snap.Time.Before(b.first.Time) {
				b.first = snap
			}
			if snap.Time.After(b.last.Time) {
				b.last = snap
			}
		}
	}

	// Sort months chronologically
	keys := make([]monthKey, 0, len(months))
	for k := range months {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].year != keys[j].year {
			return keys[i].year < keys[j].year
		}
		return keys[i].month < keys[j].month
	})

	// Collect month-end values (using the "last" snapshot in each month)
	monthEnds := make([]decimal.Decimal, 0, len(keys))
	for _, k := range keys {
		b := months[k]
		endVal := portfolioValue(b.last)
		monthEnds = append(monthEnds, endVal)
	}

	// Now compute returns BETWEEN consecutive month-end values
	if len(monthEnds) < 2 {
		return nil
	}

	returns := make([]decimal.Decimal, 0, len(monthEnds)-1)
	prev := monthEnds[0]

	for i := 1; i < len(monthEnds); i++ {
		curr := monthEnds[i]

		if !prev.GreaterThan(decimal.Zero) {
			prev = curr
			continue
		}

		r := curr.Div(prev).Sub(decimal.NewFromInt(1))
		returns = append(returns, r)

		prev = curr
	}

	return returns
}

// Helper functions
func executionsToTrades(p *portfolio) []trade {
	// Group executions by ticker so we don't accidentally pair
	// different symbols together.
	execsByTicker := make(map[string][]types.ExecutionReport)
	for _, exec := range p.executions {
		execsByTicker[exec.Ticker] = append(execsByTicker[exec.Ticker], exec)
	}

	var trades []trade

	for _, execs := range execsByTicker {
		// Sort executions for this ticker by time
		sort.Slice(execs, func(i, j int) bool {
			return execs[i].ReportTime.Before(execs[j].ReportTime)
		})

		// Pair them off 2-by-2: [0,1], [2,3], ...
		for i := 0; i < len(execs); i += 2 {
			// Normal pair
			if i+1 < len(execs) {
				a := &execs[i]
				b := &execs[i+1]

				var newTrade trade
				if a.Side == types.SideTypeBuy {
					newTrade.buy = a
					newTrade.sell = b
				} else {
					newTrade.buy = b
					newTrade.sell = a
				}
				trades = append(trades, newTrade)

				continue
			}

			// Leftover single execution → partial trade
			last := &execs[i]
			var partial trade
			if last.Side == types.SideTypeBuy {
				partial.buy = last
				partial.sell = nil
			} else {
				partial.buy = nil
				partial.sell = last
			}
			trades = append(trades, partial)
		}
	}

	// Sort resulting trades by the earliest non-nil leg time
	sort.Slice(trades, func(i, j int) bool {
		return tradeTime(trades[i]).Before(tradeTime(trades[j]))
	})

	return trades
}

// tradeTime returns the earliest non-nil leg time for a trade.
// Used for sorting trades chronologically.
func tradeTime(t trade) time.Time {
	if t.buy != nil && t.sell != nil {
		if t.buy.ReportTime.Before(t.sell.ReportTime) {
			return t.buy.ReportTime
		}
		return t.sell.ReportTime
	}
	if t.buy != nil {
		return t.buy.ReportTime
	}
	if t.sell != nil {
		return t.sell.ReportTime
	}
	// Should not happen, but give a zero time as a fallback.
	return time.Time{}
}

func portfolioValue(view types.PortfolioView) decimal.Decimal {
	value := view.Cash

	for _, pos := range view.Positions {
		posVal := pos.Quantity.Mul(pos.LastPrice)
		value = value.Add(posVal)
	}
	return value
}
