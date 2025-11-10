package types

type TradeStatus string

const (
	TradeStateBuying    TradeStatus = "TradeStateBuying"
	TradeStateNotFilled TradeStatus = "TradeStateNotFilled"
	TradeStateBought    TradeStatus = "TradeStateBought"
	TradeStateSelling   TradeStatus = "TradeStateSelling"
	TradeStateSold      TradeStatus = "TradeStateSold"
	TradeStateCancelled TradeStatus = "TradeStateCancelled"
)

type Trade struct {
	ID        int
	Symbol    string
	State     TradeStatus
	BuyOrder  *Order
	SellOrder *Order
}
