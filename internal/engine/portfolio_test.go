package engine

import (
	"backtester/types"
	"sort"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestPortfolioProcessExecutions(t *testing.T) {
	tests := []struct {
		name           string
		startPortfolio portfolio
		execs          []types.ExecutionReport
		wantPortfolio  portfolio
		wantErr        error
	}{
		{
			name: "open long",
			startPortfolio: portfolio{
				cash:      decimal.NewFromFloat(10000),
				positions: map[string]*Position{},
			},
			execs: []types.ExecutionReport{
				newExecutionReport("AAPL", types.SideTypeBuy, newFill(time.UnixMilli(1), "100", "10", "1.00")),
			},
			wantPortfolio: portfolio{
				cash: decimal.NewFromFloat(8999),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(10),
						AvgCost:   decimal.NewFromFloat(100),
						LastPrice: decimal.NewFromFloat(100),
					},
				},
			},
		},
		{
			name: "scale-in long (avg cost updates)",
			startPortfolio: portfolio{
				cash: decimal.NewFromFloat(10000),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(10),
						AvgCost:   decimal.NewFromFloat(100),
						LastPrice: decimal.NewFromFloat(100),
					},
				},
			},
			execs: []types.ExecutionReport{
				newExecutionReport("AAPL", types.SideTypeBuy, newFill(time.UnixMilli(1).Add(time.Minute), "110", "5", "0")),
			},
			wantPortfolio: portfolio{
				cash: decimal.NewFromFloat(9450),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(15),
						AvgCost:   decimal.NewFromFloat(103.3333333333333333),
						LastPrice: decimal.NewFromFloat(110),
					},
				},
			},
		},
		{
			name: "reduce long",
			startPortfolio: portfolio{
				cash: decimal.NewFromFloat(0),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(10),
						AvgCost:   decimal.NewFromFloat(100),
						LastPrice: decimal.NewFromFloat(100),
					},
				},
			},
			execs: []types.ExecutionReport{
				newExecutionReport("AAPL", types.SideTypeSell, newFill(time.UnixMilli(1).Add(time.Minute), "105", "4", "0.50")),
			},
			wantPortfolio: portfolio{
				cash: decimal.NewFromFloat(419.5),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(6),
						AvgCost:   decimal.NewFromFloat(100),
						LastPrice: decimal.NewFromFloat(105),
					},
				},
			},
		},
		{
			name: "flip long -> short",
			startPortfolio: portfolio{
				cash:              decimal.NewFromFloat(0),
				allowShortSelling: true,
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(5),
						AvgCost:   decimal.NewFromFloat(100),
						LastPrice: decimal.NewFromFloat(100),
					},
				},
			},
			execs: []types.ExecutionReport{
				newExecutionReport("AAPL", types.SideTypeSell, newFill(time.UnixMilli(1).Add(time.Minute), "90", "8", "0")),
			},
			wantPortfolio: portfolio{
				cash: decimal.NewFromFloat(720),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(-3),
						AvgCost:   decimal.NewFromFloat(90),
						LastPrice: decimal.NewFromFloat(90),
					},
				},
			},
		},
		{
			name: "insufficient cash",
			startPortfolio: portfolio{
				cash:      decimal.NewFromFloat(100),
				positions: map[string]*Position{},
			},
			execs: []types.ExecutionReport{
				newExecutionReport("AAPL", types.SideTypeBuy, newFill(time.UnixMilli(1), "10", "20", "0")),
			},
			wantErr: InsufficientBalanceErr,
		},
		{
			name: "no executions → ignored",
			startPortfolio: portfolio{
				cash:      decimal.NewFromFloat(100),
				positions: map[string]*Position{},
			},
			execs: []types.ExecutionReport{
				{symbol: "AAPL", side: types.SideTypeBuy, fills: nil},
			},
			wantPortfolio: portfolio{
				cash:      decimal.NewFromFloat(100),
				positions: map[string]*Position{},
			},
		},
		{
			name: "two symbols updated independently",
			startPortfolio: portfolio{
				cash: decimal.NewFromFloat(20000),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(10),
						AvgCost:   decimal.NewFromFloat(100),
						LastPrice: decimal.NewFromFloat(100),
					},
					"MSFT": {
						Symbol:    "MSFT",
						Quantity:  decimal.NewFromFloat(5),
						AvgCost:   decimal.NewFromFloat(200),
						LastPrice: decimal.NewFromFloat(200),
					},
				},
			},
			execs: []types.ExecutionReport{
				// Buy 5 more AAPL at 110
				newExecutionReport("AAPL", types.SideTypeBuy,
					newFill(time.UnixMilli(1), "110", "5", "0.25"),
				),

				// Sell 2 MSFT at 195
				newExecutionReport("MSFT", types.SideTypeSell,
					newFill(time.UnixMilli(2), "195", "2", "0.10"),
				),
			},
			wantPortfolio: portfolio{
				// cash:
				// start: 20000
				// AAPL buy:   -110*5 - 0.25  = -550.25
				// MSFT sell:  +195*2 - 0.10  = +389.90
				// net cash: 20000 - 550.25 + 389.90 = 19839.65
				cash: decimal.NewFromFloat(19839.65),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(15),
						AvgCost:   decimal.NewFromFloat(103.3333333),
						LastPrice: decimal.NewFromFloat(110),
					},
					"MSFT": {
						Symbol:    "MSFT",
						Quantity:  decimal.NewFromFloat(3),
						AvgCost:   decimal.NewFromFloat(200),
						LastPrice: decimal.NewFromFloat(195),
					},
				},
			},
		},
		{
			name: "multiple executions in single execution report",
			startPortfolio: portfolio{
				cash:      decimal.NewFromFloat(1000),
				positions: map[string]*Position{},
			},
			execs: []types.ExecutionReport{
				{
					orderId: "order-1",
					symbol:  "AAPL",
					side:    types.SideTypeBuy,
					status:  types.OrderFilled,
					fills: []types.Fill{
						{
							Time:  time.UnixMilli(1),
							Price: decimal.NewFromFloat(10),
							Qty:   decimal.NewFromFloat(5),
							Fee:   decimal.NewFromFloat(0.10),
						},
						{
							Time:  time.UnixMilli(2),
							Price: decimal.NewFromFloat(20),
							Qty:   decimal.NewFromFloat(5),
							Fee:   decimal.NewFromFloat(0.20),
						},
					},
					totalFilledQty: decimal.NewFromFloat(10),
					avgFillPrice:   decimal.NewFromFloat(15),
					totalFees:      decimal.NewFromFloat(0.30),
					remainingQty:   decimal.NewFromFloat(0),
					reportTime:     time.UnixMilli(2),
				},
			},
			wantPortfolio: portfolio{
				// cash = 1000 - (10*5 + 0.10) - (20*5 + 0.20)
				//      = 1000 - 50.10 - 100.20 = 849.70
				cash: decimal.NewFromFloat(849.70),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(10),
						AvgCost:   decimal.NewFromFloat(15),
						LastPrice: decimal.NewFromFloat(20),
					},
				},
			},
		},
		{
			name: "two execution reports for same symbol",
			startPortfolio: portfolio{
				cash:      decimal.NewFromFloat(1000),
				positions: map[string]*Position{},
			},
			execs: []types.ExecutionReport{
				{
					orderId: "order-1",
					symbol:  "AAPL",
					side:    types.SideTypeBuy,
					status:  types.OrderFilled,
					fills: []types.Fill{
						{
							Time:  time.UnixMilli(1),
							Price: decimal.NewFromFloat(10),
							Qty:   decimal.NewFromFloat(5),
							Fee:   decimal.NewFromFloat(0.10),
						},
					},
					totalFilledQty: decimal.NewFromFloat(5),
					avgFillPrice:   decimal.NewFromFloat(10),
					totalFees:      decimal.NewFromFloat(0.10),
					remainingQty:   decimal.NewFromFloat(0),
					reportTime:     time.UnixMilli(1),
				},
				{
					orderId: "order-2",
					symbol:  "AAPL",
					side:    types.SideTypeBuy,
					status:  types.OrderFilled,
					fills: []types.Fill{
						{
							Time:  time.UnixMilli(2),
							Price: decimal.NewFromFloat(20),
							Qty:   decimal.NewFromFloat(5),
							Fee:   decimal.NewFromFloat(0.20),
						},
					},
					totalFilledQty: decimal.NewFromFloat(5),
					avgFillPrice:   decimal.NewFromFloat(20),
					totalFees:      decimal.NewFromFloat(0.20),
					remainingQty:   decimal.NewFromFloat(0),
					reportTime:     time.UnixMilli(2),
				},
			},
			wantPortfolio: portfolio{
				cash: decimal.NewFromFloat(849.70),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(10),
						AvgCost:   decimal.NewFromFloat(15),
						LastPrice: decimal.NewFromFloat(20),
					},
				},
			},
		},
		{
			name: "unordered execution reports for same symbol",
			startPortfolio: portfolio{
				cash:              decimal.NewFromFloat(1000),
				positions:         map[string]*Position{},
				allowShortSelling: true,
			},
			execs: []types.ExecutionReport{
				{
					orderId: "order-2",
					symbol:  "AAPL",
					side:    types.SideTypeBuy,
					status:  types.OrderFilled,
					fills: []types.Fill{
						{
							Time:  time.UnixMilli(2),
							Price: decimal.NewFromFloat(20),
							Qty:   decimal.NewFromFloat(5),
							Fee:   decimal.NewFromFloat(0.20),
						},
					},
					totalFilledQty: decimal.NewFromFloat(5),
					avgFillPrice:   decimal.NewFromFloat(20),
					totalFees:      decimal.NewFromFloat(0.20),
					remainingQty:   decimal.NewFromFloat(0),
					reportTime:     time.UnixMilli(2),
				},
				{
					orderId: "order-1",
					symbol:  "AAPL",
					side:    types.SideTypeBuy,
					status:  types.OrderFilled,
					fills: []types.Fill{
						{
							Time:  time.UnixMilli(1),
							Price: decimal.NewFromFloat(10),
							Qty:   decimal.NewFromFloat(5),
							Fee:   decimal.NewFromFloat(0.10),
						},
					},
					totalFilledQty: decimal.NewFromFloat(5),
					avgFillPrice:   decimal.NewFromFloat(10),
					totalFees:      decimal.NewFromFloat(0.10),
					remainingQty:   decimal.NewFromFloat(0),
					reportTime:     time.UnixMilli(1),
				},
			},
			wantPortfolio: portfolio{
				cash: decimal.NewFromFloat(849.70),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(10),
						AvgCost:   decimal.NewFromFloat(15),
						LastPrice: decimal.NewFromFloat(20),
					},
				},
			},
		},
		{
			name: "sell more than position → QuantityExceededErr",
			startPortfolio: portfolio{
				cash: decimal.NewFromFloat(0),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(5),
						AvgCost:   decimal.NewFromFloat(100),
						LastPrice: decimal.NewFromFloat(100),
					},
				},
			},
			execs: []types.ExecutionReport{
				{
					orderId: "order-oversell",
					symbol:  "AAPL",
					side:    types.SideTypeSell,
					status:  types.OrderFilled,
					fills: []types.Fill{
						{
							Time:  time.Date(2025, time.January, 2, 10, 1, 0, 0, time.UTC),
							Price: decimal.NewFromFloat(110),
							Qty:   decimal.NewFromFloat(10),
							Fee:   decimal.NewFromFloat(0),
						},
					},
					totalFilledQty: decimal.NewFromFloat(10),
					avgFillPrice:   decimal.NewFromFloat(110),
					totalFees:      decimal.NewFromFloat(0),
					remainingQty:   decimal.NewFromFloat(0),
					reportTime:     time.Date(2025, time.January, 2, 10, 1, 0, 0, time.UTC),
				},
			},
			wantPortfolio: portfolio{
				// portfolio should be unchanged if you bail out with QuantityExceededErr
				cash: decimal.NewFromFloat(0),
				positions: map[string]*Position{
					"AAPL": {
						Symbol:    "AAPL",
						Quantity:  decimal.NewFromFloat(5),
						AvgCost:   decimal.NewFromFloat(100),
						LastPrice: decimal.NewFromFloat(100),
					},
				},
			},
			wantErr: ShortSellNotAllowedErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {

			err := tc.startPortfolio.processExecutions(tc.execs)
			if tc.wantErr != nil {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if err.Error() != tc.wantErr.Error() {
					t.Fatalf("got error %q, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if want, got := tc.wantPortfolio.cash, tc.startPortfolio.cash; want.Cmp(got) != 0 {
				t.Fatalf("cash mismatch: got %s want %s", got, want)
			}

			for sym, wantPos := range tc.wantPortfolio.positions {
				gotPos := tc.startPortfolio.positions[sym]
				if gotPos == nil {
					t.Fatalf("position for %s missing", sym)
				}
				if !gotPos.Quantity.Equal(wantPos.Quantity) {
					t.Fatalf("qty mismatch: got %s want %s", gotPos.Quantity, wantPos.Quantity)
				}
				if !gotPos.AvgCost.RoundBank(6).Equal(wantPos.AvgCost.RoundBank(6)) {
					t.Fatalf("avgCost mismatch: got %s want %s", gotPos.AvgCost, wantPos.AvgCost)
				}
				if !gotPos.LastPrice.Equal(wantPos.LastPrice) {
					t.Fatalf("lastPrice mismatch: got %s want %s", gotPos.LastPrice, wantPos.LastPrice)
				}
			}

			// Ensure no unexpected positions
			if len(tc.startPortfolio.positions) != len(tc.wantPortfolio.positions) {
				t.Fatalf("unexpected extra positions: got %+v, want %+v", tc.startPortfolio.positions, tc.wantPortfolio.positions)
			}
		})
	}
}

func TestWeightedAvgPrice(t *testing.T) {
	tests := []struct {
		name             string
		existingAvgPrice decimal.Decimal
		existingQty      decimal.Decimal
		newPrice         decimal.Decimal
		newQty           decimal.Decimal
		want             decimal.Decimal
	}{
		{
			name:             "existing qty zero → returns newPrice",
			existingAvgPrice: decimal.RequireFromString("0"),
			existingQty:      decimal.RequireFromString("0"),
			newPrice:         decimal.RequireFromString("123.45"),
			newQty:           decimal.RequireFromString("10"),
			want:             decimal.RequireFromString("123.45"),
		},
		{
			name:             "new qty zero → unchanged average",
			existingAvgPrice: decimal.RequireFromString("100"),
			existingQty:      decimal.RequireFromString("10"),
			newPrice:         decimal.RequireFromString("150"),
			newQty:           decimal.RequireFromString("0"),
			want:             decimal.RequireFromString("100"),
		},
		{
			name:             "simple mix",
			existingAvgPrice: decimal.RequireFromString("100"),
			existingQty:      decimal.RequireFromString("10"),
			newPrice:         decimal.RequireFromString("110"),
			newQty:           decimal.RequireFromString("5"),
			want:             decimal.RequireFromString("103.3333333333333333"),
		},
		{
			name:             "identical prices",
			existingAvgPrice: decimal.RequireFromString("42.00"),
			existingQty:      decimal.RequireFromString("7"),
			newPrice:         decimal.RequireFromString("42.00"),
			newQty:           decimal.RequireFromString("3"),
			want:             decimal.RequireFromString("42.00"),
		},
		{
			name:             "large numbers",
			existingAvgPrice: decimal.RequireFromString("250000.125"),
			existingQty:      decimal.RequireFromString("1000000"),
			newPrice:         decimal.RequireFromString("249999.875"),
			newQty:           decimal.RequireFromString("500000"),
			// compute expected with same decimal ops to avoid literal precision issues
			want: func() decimal.Decimal {
				num := decimal.RequireFromString("250000.125").Mul(decimal.RequireFromString("1000000")).Add(decimal.RequireFromString("249999.875").Mul(decimal.RequireFromString("500000")))
				den := decimal.RequireFromString("1000000").Add(decimal.RequireFromString("500000"))
				return num.Div(den)
			}(),
		},
		{
			name:             "both quantities zero → returns newPrice by convention",
			existingAvgPrice: decimal.RequireFromString("99.99"),
			existingQty:      decimal.RequireFromString("0"),
			newPrice:         decimal.RequireFromString("88.88"),
			newQty:           decimal.RequireFromString("0"),
			want:             decimal.RequireFromString("88.88"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := weightedAvg(tc.existingAvgPrice, tc.existingQty, tc.newPrice, tc.newQty)

			// For cases with repeating decimals, compare using Cmp (exact decimal equality).
			if !got.Equal(tc.want) {
				t.Fatalf("got %s, want %s", got.String(), tc.want.String())
			}
		})
	}
}

// Helper functions

func newFill(t time.Time, price, qty, fee string) types.Fill {
	return types.Fill{Time: t, Price: decimal.RequireFromString(price), Qty: decimal.RequireFromString(qty), Fee: decimal.RequireFromString(fee)}
}

func newExecutionReport(symbol string, side types.Side, fills ...types.Fill) types.ExecutionReport {
	totalQty := decimal.Zero
	totalFees := decimal.Zero
	sum := decimal.Zero

	for _, f := range fills {
		totalQty = totalQty.Add(f.Qty)
		totalFees = totalFees.Add(f.Fee)
		sum = sum.Add(f.Price.Mul(f.Qty))
	}

	avg := decimal.Zero
	if !totalQty.IsZero() {
		avg = sum.Div(totalQty)
	}

	if len(fills) > 0 {
		cp := append([]types.Fill(nil), fills...)
		sort.Slice(cp, func(i, j int) bool { return cp[i].Time.Before(cp[j].Time) })
	}

	return types.ExecutionReport{
		orderId:        "X",
		symbol:         symbol,
		side:           side,
		status:         types.OrderFilled,
		fills:          fills,
		totalFilledQty: totalQty,
		avgFillPrice:   avg,
		totalFees:      totalFees,
		remainingQty:   decimal.Zero,
	}
}
