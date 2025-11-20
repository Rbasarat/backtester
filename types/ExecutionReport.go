package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type ExecutionReport struct {
	TradeId        string
	Symbol         string
	Side           Side
	Status         OrderStatus
	Fills          []Fill
	TotalFilledQty decimal.Decimal
	AvgFillPrice   decimal.Decimal
	TotalFees      decimal.Decimal
	RemainingQty   decimal.Decimal
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

func NewExecutionReport(
	tradeID string,
	symbol string,
	side Side,
	status OrderStatus,
	fills []Fill,
	totalFilledQty decimal.Decimal,
	avgFillPrice decimal.Decimal,
	totalFees decimal.Decimal,
	remainingQty decimal.Decimal,
	rejectReason string,
	reportTime time.Time,
) ExecutionReport {
	return ExecutionReport{
		TradeId:        tradeID,
		Symbol:         symbol,
		Side:           side,
		Status:         status,
		Fills:          fills,
		TotalFilledQty: totalFilledQty,
		AvgFillPrice:   avgFillPrice,
		TotalFees:      totalFees,
		RemainingQty:   remainingQty,
		RejectReason:   rejectReason,
		ReportTime:     reportTime,
	}
}
