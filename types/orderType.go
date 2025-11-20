package types

type Side string

type Direction string
type OrderType string

type OrderStatus string

const (
	OrderAccepted        OrderStatus = "ORDER_ACCEPTED"
	OrderPartiallyFilled OrderStatus = "ORDER_PARTIALLY_FILLED"
	OrderFilled          OrderStatus = "ORDER_FILLED"
	OrderRejected        OrderStatus = "ORDER_REJECTED"
	OrderExpired         OrderStatus = "ORDER_EXPIRED"
	OrderCanceled        OrderStatus = "ORDER_CANCELED"

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
