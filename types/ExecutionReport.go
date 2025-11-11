package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type ExecutionReport struct {
	OrderID        string
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
	Time  time.Time
	Price decimal.Decimal
	Qty   decimal.Decimal
	Fee   decimal.Decimal
}

type OrderStatus int

const (
	OrderAccepted OrderStatus = iota
	OrderPartiallyFilled
	OrderFilled
	OrderRejected
	OrderExpired
	OrderCanceled
)
