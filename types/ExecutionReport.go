package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type ExecutionReport struct {
	Ticker         string
	Side           Side
	Status         OrderStatus
	Fills          []Fill
	TotalFilledQty decimal.Decimal
	AvgFillPrice   decimal.Decimal
	TotalFees      decimal.Decimal
	RemainingQty   decimal.Decimal
	SignalReason   string
	RejectReason   string
	ReportTime     time.Time
}

type Fill struct {
	Time     time.Time
	Price    decimal.Decimal
	Quantity decimal.Decimal
	Fee      decimal.Decimal
}

func NewFill(time time.Time, price, qty, fee decimal.Decimal) Fill {
	return Fill{
		Time:     time,
		Price:    price,
		Quantity: qty,
		Fee:      fee,
	}
}

func NewExecutionReport(ticker string, side Side, status OrderStatus, fills []Fill, totalFilledQty decimal.Decimal, avgFillPrice decimal.Decimal, totalFees decimal.Decimal, remainingQty decimal.Decimal, signalReason string, rejectReason string, reportTime time.Time) *ExecutionReport {
	return &ExecutionReport{Ticker: ticker, Side: side, Status: status, Fills: fills, TotalFilledQty: totalFilledQty, AvgFillPrice: avgFillPrice, TotalFees: totalFees, RemainingQty: remainingQty, SignalReason: signalReason, RejectReason: rejectReason, ReportTime: reportTime}

}
