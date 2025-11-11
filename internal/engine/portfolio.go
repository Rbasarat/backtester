package engine

import (
	"backtester/types"
	"time"

	"github.com/shopspring/decimal"
)

type portfolio struct {
	Cash         decimal.Decimal
	Positions    map[string]*Position
	OpenTrades   map[string][]*types.Trade
	ClosedTrades map[string][]*types.Trade
}

type Position struct {
	Symbol     string
	Side       types.Side
	Quantity   decimal.Decimal
	EntryPrice decimal.Decimal
	EntryTime  time.Time
	LastPrice  decimal.Decimal
}

func newPortfolio(initialCash decimal.Decimal) *portfolio {
	return &portfolio{
		Cash:         initialCash,
		Positions:    make(map[string]*Position), //TODO: we can make this a list to allow for scaling in/out if needed later
		OpenTrades:   make(map[string][]*types.Trade),
		ClosedTrades: make(map[string][]*types.Trade),
	}
}

func (p *portfolio) GetPortfolioSnapshot() types.PortfolioView {
	view := types.PortfolioView{
		Cash:      p.Cash,
		Positions: make(map[string]types.PositionSnapshot),
	}

	for sym, pos := range p.Positions {
		view.Positions[sym] = types.PositionSnapshot{
			Symbol:    pos.Symbol,
			Side:      pos.Side,
			Quantity:  pos.Quantity,
			LastPrice: pos.LastPrice,
			EntryTime: pos.EntryTime,
		}
	}
	return view
}
