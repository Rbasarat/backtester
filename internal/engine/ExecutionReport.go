package engine

import (
	"backtester/types"
	"time"

	"github.com/shopspring/decimal"
)

type ExecutionReport struct {
	orderID        string
	symbol         string
	side           types.Side
	status         types.OrderStatus
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
	side types.Side,
	status types.OrderStatus,
	fills []Fill,
	totalFilledQty decimal.Decimal,
	avgFillPrice decimal.Decimal,
	totalFees decimal.Decimal,
	remainingQty decimal.Decimal,
	rejectReason string,
	reportTime time.Time,
) ExecutionReport {
	return ExecutionReport{
		orderID:        orderID,
		symbol:         symbol,
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
