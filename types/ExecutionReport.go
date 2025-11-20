package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type ExecutionReport struct {
	TradeId        string
	OrderId        string
	Symbol         string
	side           Side
	status         OrderStatus
	fills          []Fill
	totalFilledQty decimal.Decimal
	avgFillPrice   decimal.Decimal
	totalFees      decimal.Decimal
	remainingQty   decimal.Decimal
	rejectReason   string
	reportTime     time.Time
}

type Fill struct {
	Time  time.Time
	Price decimal.Decimal
	Qty   decimal.Decimal
	Fee   decimal.Decimal
}

func NewFill(time time.Time, price, qty, fee decimal.Decimal) Fill {
	return Fill{
		Time:  time,
		Price: price,
		Qty:   qty,
		Fee:   fee,
	}
}

func NewExecutionReport(
	orderID string,
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
		OrderId:        orderID,
		Symbol:         symbol,
		side:           side,
		status:         status,
		fills:          fills,
		totalFilledQty: totalFilledQty,
		avgFillPrice:   avgFillPrice,
		totalFees:      totalFees,
		remainingQty:   remainingQty,
		rejectReason:   rejectReason,
		reportTime:     reportTime,
	}
}
