package engine

import (
	"backtester/types"
	"errors"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

var UnknownSideErr = errors.New("unknown fill side")
var InsufficientBalanceErr = errors.New("insufficient balance when applying order fill")
var ShortSellNotAllowedErr = errors.New("short sell not allowed, broker sold more stock than in portfolio")

type portfolio struct {
	cash              decimal.Decimal
	positions         map[string]*Position
	fills             []Fill
	realizedPnL       decimal.Decimal
	snapshots         []types.PortfolioView
	allowShortSelling bool
}

type Position struct {
	Symbol    string
	Quantity  decimal.Decimal
	AvgCost   decimal.Decimal
	LastPrice decimal.Decimal
}

func newPortfolio(initialCash decimal.Decimal, allowShortSelling bool) *portfolio {
	return &portfolio{
		cash:              initialCash,
		positions:         make(map[string]*Position),
		allowShortSelling: allowShortSelling,
	}
}

func (p *portfolio) GetPortfolioSnapshot(curTime time.Time) types.PortfolioView {
	view := types.PortfolioView{
		Cash:      p.cash,
		Positions: make(map[string]types.PositionSnapshot),
		Time:      curTime,
	}

	for sym, pos := range p.positions {
		view.Positions[sym] = types.PositionSnapshot{
			Symbol:    pos.Symbol,
			Quantity:  pos.Quantity,
			LastPrice: pos.LastPrice,
		}
	}
	return view
}

func (p *portfolio) processExecutions(execs []ExecutionReport) error {
	if len(execs) == 0 {
		return nil
	}
	sort.Slice(execs, func(i, j int) bool { return execs[i].reportTime.Before(execs[j].reportTime) })
	for _, er := range execs {
		if len(er.fills) == 0 {
			continue
		}
		// sort fills by time
		fills := append([]Fill(nil), er.fills...)
		sort.Slice(fills, func(i, j int) bool { return fills[i].Time.Before(fills[j].Time) })

		pos := p.positions[er.symbol]
		if pos == nil {
			// Create new position
			pos = &Position{Symbol: er.symbol}
			p.positions[er.symbol] = pos
		}

		for _, fill := range fills {
			quantity := fill.Qty
			if er.side != types.SideTypeBuy && er.side != types.SideTypeSell {
				return UnknownSideErr
			}

			if er.side == types.SideTypeSell {
				quantity = quantity.Neg()
			}

			cashDelta := fill.Price.Mul(quantity).Neg()
			newCash := p.cash.Add(cashDelta).Sub(fill.Fee)

			if newCash.LessThan(decimal.Zero) {
				return InsufficientBalanceErr
			}
			p.cash = newCash

			oldQty := pos.Quantity
			newQty := oldQty.Add(quantity)
			if !p.allowShortSelling && newQty.IsNegative() {
				return ShortSellNotAllowedErr
			}

			switch {
			case sameSide(oldQty, newQty):
				absOld := oldQty.Abs()
				absNew := newQty.Abs()
				absAdd := quantity.Abs()
				if absNew.GreaterThan(absOld) && !absAdd.IsZero() {
					pos.AvgCost = weightedAvg(pos.AvgCost, absOld, fill.Price, absAdd)
				}
				pos.Quantity = newQty

			case oldQty.IsZero():
				pos.Quantity = newQty
				pos.AvgCost = fill.Price

			case newQty.IsZero():
				pos.Quantity = decimal.Zero
				pos.AvgCost = decimal.Zero

			default:
				pos.Quantity = newQty
				pos.AvgCost = fill.Price
			}

			pos.LastPrice = fill.Price
			p.fills = append(p.fills, fill)
		}
	}
	return nil
}

func sameSide(a, b decimal.Decimal) bool {
	return (a.GreaterThan(decimal.Zero) && b.GreaterThan(decimal.Zero)) ||
		(a.LessThan(decimal.Zero) && b.LessThan(decimal.Zero))
}

func weightedAvg(existingAvgPrice, existingQty, newPrice, newQty decimal.Decimal) decimal.Decimal {
	if existingQty.IsZero() {
		return newPrice
	}
	return existingAvgPrice.Mul(existingQty).
		Add(newPrice.Mul(newQty)).
		Div(existingQty.Add(newQty))
}
