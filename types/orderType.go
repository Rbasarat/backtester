package types

type Side string

type Direction string
type OrderType string

type OrderStatus string

const (
	OrderAccepted        OrderStatus = "ORDER_ACCEPTED"
	OrderPartiallyFilled             = "ORDER_PARTIALLY_FILLED"
	OrderFilled                      = "ORDER_FILLED"
	OrderRejected                    = "ORDER_REJECTED"
	OrderExpired                     = "ORDER_EXPIRED"
	OrderCanceled                    = "ORDER_CANCELED"

	SideTypeBuy  Side = "BUY"
	SideTypeSell Side = "SELL"

	DirectionLong  Direction = "LONG"
	DirectionShort Direction = "SHORT"

	TypeLimit           OrderType = "LIMIT"
	TypeMarket          OrderType = "MARKET"
	TypeLimitMaker      OrderType = "LIMIT_MAKER"
	TypeStopLoss        OrderType = "STOP_LOSS"
	TypeStopLossLimit   OrderType = "STOP_LOSS_LIMIT"
	TypeTakeProfit      OrderType = "TAKE_PROFIT"
	TypeTakeProfitLimit OrderType = "TAKE_PROFIT_LIMIT"
)
