package donchian

import (
	"backtester/types"
	"time"

	"github.com/shopspring/decimal"
)

type Broker struct {
}

// ibkrNetherlandsFixedUSDFee computes the commission for a trade in USD-denominated
// Netherlands stocks using IBKR "Fixed - IB SmartRouting" pricing.
//
// Schedule (per IBKR, Netherlands, USD):
//   - 0.05% of trade value
//   - Minimum per order: USD 1.70
//   - Maximum per order: USD 39.00
//
// tradeValue = price * quantity (in USD)
func ibkrNetherlandsFixedUSDFee(tradeValue decimal.Decimal) decimal.Decimal {
	if tradeValue.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	// 0.05% = 0.0005
	rate := decimal.NewFromFloat(0.0005)
	fee := tradeValue.Mul(rate)

	minFee := decimal.RequireFromString("1.70")
	maxFee := decimal.RequireFromString("39")

	if fee.LessThan(minFee) {
		fee = minFee
	}
	if fee.GreaterThan(maxFee) {
		fee = maxFee
	}
	return fee
}

func ibkrForexTier1LowestUSDFee(tradeValueUSD decimal.Decimal) decimal.Decimal {
	if tradeValueUSD.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	// 0.20 basis point = 0.20 * 0.0001 = 0.00002
	rate := decimal.RequireFromString("0.00002")
	fee := tradeValueUSD.Mul(rate)

	minFee := decimal.RequireFromString("2.00")
	if fee.LessThan(minFee) {
		fee = minFee
	}

	return fee
}

// Execute fills all orders at the OPEN of the next available candle for that ticker.
// Fee model (IBKR Netherlands, USD, Fixed - SmartRouting):
//   - Commission = 0.05% of trade value
//   - Min fee per order = 3.00 USD
//   - Max fee per order = 29.00 USD
//
// - No slippage
// - Buys: check remaining cash, reject if insufficient (price * qty + fee)
// - Sells: always allowed, proceeds added to remaining cash (minus fee)
// - Does NOT mutate the portfolio directly; engine applies reports.
func (b *Broker) Execute(orders []types.Order, ctx types.ExecutionContext) []types.ExecutionReport {
	var execReports []types.ExecutionReport
	remainingCash := ctx.Portfolio.Cash

	for _, order := range orders {
		candles, ok := ctx.Candles[order.Ticker]
		if !ok || len(candles) == 0 {
			// No data for this ticker
			report := *types.NewExecutionReport(
				order.Ticker,
				order.Side,
				types.OrderRejected,
				[]types.Fill{},
				decimal.Zero, // filledQty
				decimal.Zero, // avgPrice
				decimal.Zero, // fee
				decimal.Zero, // slippage
				"No market data for ticker",
				order.SignalReason,
				ctx.CurTime,
			)
			execReports = append(execReports, report)
			continue
		}

		nextCandle := getNextCandle(ctx.CurTime, candles)
		if nextCandle == nil {
			// No candle strictly after CurTime -> cannot execute
			report := *types.NewExecutionReport(
				order.Ticker,
				order.Side,
				types.OrderRejected,
				[]types.Fill{},
				decimal.Zero,
				decimal.Zero,
				decimal.Zero,
				decimal.Zero,
				"No future candle available for execution",
				order.SignalReason,
				ctx.CurTime,
			)
			execReports = append(execReports, report)
			continue
		}

		// Sanity: non-positive quantity -> reject
		if order.Quantity.LessThanOrEqual(decimal.Zero) {
			report := *types.NewExecutionReport(
				order.Ticker,
				order.Side,
				types.OrderRejected,
				[]types.Fill{},
				decimal.Zero,
				decimal.Zero,
				decimal.Zero,
				decimal.Zero,
				"Non-positive order quantity",
				order.SignalReason,
				ctx.CurTime,
			)
			execReports = append(execReports, report)
			continue
		}

		fillPrice := nextCandle.Open
		fillTime := nextCandle.Timestamp
		tradeValue := fillPrice.Mul(order.Quantity)

		// Compute IBKR Netherlands fee for this trade value
		fee := ibkrNetherlandsFixedUSDFee(tradeValue)

		// Pre-check / update cash for this order
		switch order.Side {
		case types.SideTypeBuy:
			// Total cash needed: price * qty + fee
			totalCost := tradeValue.Add(fee)
			if totalCost.GreaterThan(remainingCash) {
				report := *types.NewExecutionReport(
					order.Ticker,
					order.Side,
					types.OrderRejected,
					[]types.Fill{},
					decimal.Zero,
					decimal.Zero,
					decimal.Zero,
					decimal.Zero,
					"Not enough cash available for buy",
					order.SignalReason,
					ctx.CurTime,
				)
				execReports = append(execReports, report)
				continue
			}
			remainingCash = remainingCash.Sub(totalCost)

		case types.SideTypeSell:
			// Proceeds from sale minus fee
			proceeds := tradeValue
			remainingCash = remainingCash.Add(proceeds).Sub(fee)
		}

		// Build fill object
		fill := types.NewFill(
			fillTime,
			fillPrice,
			order.Quantity,
			fee,
		)

		// Successful fill
		report := *types.NewExecutionReport(
			order.Ticker,
			order.Side,
			types.OrderFilled,
			[]types.Fill{fill},
			order.Quantity,     // filledQty
			fillPrice,          // avgPrice
			fee,                // fee
			decimal.Zero,       // slippage (none)
			"",                 // message
			order.SignalReason, // signal reason
			fillTime,           // report time = fill time
		)

		execReports = append(execReports, report)
	}

	return execReports
}

// getNextCandle returns the first candle with Timestamp strictly AFTER curTime.
// With weekly candles and CurTime at the last bar's time, this will be the next Monday open.
func getNextCandle(curTime time.Time, candles []types.Candle) *types.Candle {
	for i := range candles {
		if candles[i].Timestamp.After(curTime) {
			return &candles[i]
		}
	}
	return nil
}
