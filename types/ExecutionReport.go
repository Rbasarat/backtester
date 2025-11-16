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

type OrderStatus string

const (
	OrderAccepted        OrderStatus = "ORDER_ACCEPTED"
	OrderPartiallyFilled             = "ORDER_PARTIALLY_FILLED"
	OrderFilled                      = "ORDER_FILLED"
	OrderRejected                    = "ORDER_REJECTED"
	OrderExpired                     = "ORDER_EXPIRED"
	OrderCanceled                    = "ORDER_CANCELED"
)
