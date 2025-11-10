package types

import (
	"github.com/shopspring/decimal"
	"time"
)

type Side string
type OrderType string

const (
	SideTypeBuy  Side = "BUY"
	SideTypeSell Side = "SELL"

	TypeLimit           OrderType = "LIMIT"
	TypeMarket          OrderType = "MARKET"
	TypeLimitMaker      OrderType = "LIMIT_MAKER"
	TypeStopLoss        OrderType = "STOP_LOSS"
	TypeStopLossLimit   OrderType = "STOP_LOSS_LIMIT"
	TypeTakeProfit      OrderType = "TAKE_PROFIT"
	TypeTakeProfitLimit OrderType = "TAKE_PROFIT_LIMIT"
)

type Order struct {
	Timestamp        time.Time
	ID               int
	Symbol           string
	Price            decimal.Decimal
	OrigQuantity     decimal.Decimal
	ExecutedQuantity decimal.Decimal
	Type             OrderType
	Side             Side
	Fee              decimal.Decimal
}
