package engine

import (
	"backtester/types"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestCalcNetProfitAndFees(t *testing.T) {
	tests := []struct {
		name     string
		trades   []trade
		wantNet  decimal.Decimal
		wantFees decimal.Decimal
	}{
		{
			name:     "no executions -> zero",
			trades:   []trade{},
			wantNet:  decimal.RequireFromString("0"),
			wantFees: decimal.RequireFromString("0"),
		},
		{
			name: "only buys -> unrealized -> only fees",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0.5"),
							},
						},
					},
					sell: nil,
				},
			},
			// grossProfit = 0 (unrealized), fees = 0.5 -> net = -0.5
			wantNet:  decimal.RequireFromString("-0.5"),
			wantFees: decimal.RequireFromString("0.5"),
		},
		{
			name: "only sells -> unrealized -> only fees",
			trades: []trade{
				{
					buy: nil,
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("50"),
								Quantity: decimal.RequireFromString("2"),
								Fee:      decimal.RequireFromString("0.1"),
							},
						},
					},
				},
			},
			// grossProfit = 0 (unrealized), fees = 0.1 -> net = -0.1
			wantNet:  decimal.RequireFromString("-0.1"),
			wantFees: decimal.RequireFromString("0.1"),
		},
		{
			name: "simple realized long trade (buy then sell, with fees)",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("110"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
				},
			},
			// gross = -100 + 110 = 10, fees = 2 -> net = 8
			wantNet:  decimal.RequireFromString("8"),
			wantFees: decimal.RequireFromString("2"),
		},
		{
			name: "partially closed position still counted as realized (has buy and sell)",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("2"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("110"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
			},
			// gross = -200 + 110 = -90, fees = 0 -> net = -90
			wantNet:  decimal.RequireFromString("-90"),
			wantFees: decimal.RequireFromString("0"),
		},
		{
			name: "multiple trades: some realized, some not",
			trades: []trade{
				// trade1: realized long
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("110"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
				},
				// trade2: realized short (overall loss)
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("60"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("50"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
				// trade3: only buy (unrealized)
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("10"),
								Quantity: decimal.RequireFromString("5"),
								Fee:      decimal.RequireFromString("0.1"),
							},
						},
					},
					sell: nil,
				},
			},
			// trade1: gross 10, fees 2
			// trade2: gross -10, fees 0
			// trade3: gross ignored (unrealized), fees 0.1
			// total gross = 0, total fees = 2.1 -> net = -2.1
			wantNet:  decimal.RequireFromString("-2.1"),
			wantFees: decimal.RequireFromString("2.1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			gotNet, gotFees := calcNetProfitAndFees(tt.trades, &wg)

			if !gotNet.Equal(tt.wantNet) || !gotFees.Equal(tt.wantFees) {
				t.Fatalf(
					"calcNetProfitAndFees() = net %s, fees %s, want net %s, fees %s",
					gotNet.String(), gotFees.String(), tt.wantNet.String(), tt.wantFees.String(),
				)
			}
		})
	}
}

func TestNetAvgProfitPerTrade(t *testing.T) {
	tests := []struct {
		name   string
		trades []trade
		want   decimal.Decimal
	}{
		{
			name:   "no executions => 0",
			trades: []trade{},
			want:   decimal.RequireFromString("0"),
		},
		{
			name: "only buys (no realized trades) => 0",
			// Note: even though fees are present, realizedTrades==0 so function returns 0
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0.5"),
							},
						},
					},
					sell: nil,
				},
			},
			want: decimal.RequireFromString("0"),
		},
		{
			name: "simple realized long trade (buy then sell, with fees)",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("110"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
				},
			},
			// gross = -100 + 110 = 10, fees = 2 -> net = 8, only one realized trade
			want: decimal.RequireFromString("8"),
		},
		{
			name: "partially closed position is treated as realized",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("2"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("110"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
			},
			// gross = -200 + 110 = -90, fees = 0 -> net = -90
			want: decimal.RequireFromString("-90"),
		},
		{
			name: "one realized trade + one unrealized trade (fees from unrealized still counted)",
			trades: []trade{
				// realized trade
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("110"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
				},
				// unrealized trade (only buy)
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("50"),
								Quantity: decimal.RequireFromString("2"),
								Fee:      decimal.RequireFromString("0.5"),
							},
							{
								Price:    decimal.RequireFromString("50"),
								Quantity: decimal.RequireFromString("0"), // extra fill, fee counted
								Fee:      decimal.RequireFromString("0.5"),
							},
						},
					},
					sell: nil,
				},
			},
			// realized trade gross = 10, fees = 2
			// unrealized trade fees = 1, gross ignored (no sell)
			// total gross = 10, total fees = 3, realizedTrades = 1
			// netAvg = (10 - 3) / 1 = 7
			want: decimal.RequireFromString("7"),
		},
		{
			name: "two realized trades, no unrealized",
			trades: []trade{
				// trade1: long
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("110"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
				},
				// trade2: short
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("150"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0.5"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("200"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0.5"),
							},
						},
					},
				},
			},
			// trade1: gross 10, fees 2
			// trade2: gross 50, fees 1
			// total gross = 60, total fees = 3, realizedTrades = 2
			// netAvg = (60 - 3) / 2 = 28.5
			want: decimal.RequireFromString("28.5"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			got := calcNetAvgProfitPerTrade(tt.trades, &wg)

			if !got.Equal(tt.want) {
				t.Fatalf("NetAvgProfitPerTrade() = %s, want %s", got.String(), tt.want.String())
			}
		})
	}
}

func TestCalcCAGRFromSnapshots(t *testing.T) {
	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		snapshots []types.PortfolioView
		want      decimal.Decimal
	}{
		{
			name: "23.86% in 3 years",
			snapshots: []types.PortfolioView{
				{
					Time: baseTime,
					Cash: decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Ticker:          "AAA",
							Quantity:        decimal.RequireFromString("1"),
							LastMarketPrice: decimal.RequireFromString("9000"),
							AvgEntryPrice:   decimal.RequireFromString("0"),
						},
					},
				},
				{
					Time: baseTime.AddDate(3, 0, 0), // approx 1 year later
					Cash: decimal.RequireFromString("5000"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Ticker:          "AAA",
							Quantity:        decimal.RequireFromString("1"),
							LastMarketPrice: decimal.RequireFromString("14000"),
							AvgEntryPrice:   decimal.RequireFromString("0"),
						},
					},
				},
			},
			want: decimal.RequireFromString("0.2385"),
		},
		{
			name: "8.45% in 5 years look to cash and positions",
			snapshots: []types.PortfolioView{
				{
					Time: baseTime,
					Cash: decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Ticker:          "AAA",
							Quantity:        decimal.RequireFromString("1"),
							LastMarketPrice: decimal.RequireFromString("9000"),
							AvgEntryPrice:   decimal.RequireFromString("0"),
						},
					},
				},
				{
					Time: baseTime.AddDate(5, 0, 0), // approx 1 year later
					Cash: decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Ticker:          "AAA",
							Quantity:        decimal.RequireFromString("1"),
							LastMarketPrice: decimal.RequireFromString("14000"),
							AvgEntryPrice:   decimal.RequireFromString("0"),
						},
					},
				},
			},
			want: decimal.RequireFromString("0.0844"),
		},
		{
			name:      "no snapshots -> 0",
			snapshots: nil,
			want:      decimal.RequireFromString("0"),
		},
		{
			name: "single snapshot -> 0",
			snapshots: []types.PortfolioView{
				{
					Time:      baseTime,
					Cash:      decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{},
				},
			},
			want: decimal.RequireFromString("0"),
		},
		{
			name: "flat portfolio over 1 year -> 0",
			snapshots: []types.PortfolioView{
				{
					Time: baseTime,
					Cash: decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Ticker:          "AAA",
							Quantity:        decimal.RequireFromString("10"),
							LastMarketPrice: decimal.RequireFromString("0"),
							AvgEntryPrice:   decimal.RequireFromString("0"),
						},
					},
				},
				{
					Time: baseTime.AddDate(1, 0, 0), // approx 1 year later
					Cash: decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Ticker:          "AAA",
							Quantity:        decimal.RequireFromString("10"),
							LastMarketPrice: decimal.RequireFromString("0"),
							AvgEntryPrice:   decimal.RequireFromString("0"),
						},
					},
				},
			},
			want: decimal.RequireFromString("0"),
		},
		{
			name: "cash grows from 1000 to 1210 in 1 year -> ~21%",
			// CAGR = (1210 / 1000)^(1/1) - 1 = 0.21
			snapshots: []types.PortfolioView{
				{
					Time:      baseTime,
					Cash:      decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{},
				},
				{
					Time:      baseTime.AddDate(1, 0, 0),
					Cash:      decimal.RequireFromString("1210"),
					Positions: map[string]types.PositionSnapshot{},
				},
			},
			want: decimal.RequireFromString("0.2095"),
		},
		{
			name: "portfolio doubles in 2 years from positions only -> ~41.4213%",
			// start: 10 * 100 = 1000
			// end:   10 * 200 = 2000
			// CAGR = (2000 / 1000)^(1/2) - 1 = sqrt(2) - 1 ≈ 0.41421356
			snapshots: []types.PortfolioView{
				{
					Time: baseTime,
					Cash: decimal.RequireFromString("0"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Ticker:          "AAA",
							Quantity:        decimal.RequireFromString("10"),
							LastMarketPrice: decimal.RequireFromString("100"),
							AvgEntryPrice:   decimal.RequireFromString("100"),
						},
					},
				},
				{
					Time: baseTime.AddDate(2, 0, 0), // ~2 years later
					Cash: decimal.RequireFromString("0"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Ticker:          "AAA",
							Quantity:        decimal.RequireFromString("10"),
							LastMarketPrice: decimal.RequireFromString("200"),
							AvgEntryPrice:   decimal.RequireFromString("100"),
						},
					},
				},
			},
			want: decimal.RequireFromString("0.4139"),
		},
		{
			name: "portfolio halves in 1 year -> -50%",
			// start: 1000
			// end: 500
			// CAGR = (0.5)^(1/1) - 1 = -0.5
			snapshots: []types.PortfolioView{
				{
					Time:      baseTime,
					Cash:      decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{},
				},
				{
					Time:      baseTime.AddDate(1, 0, 0),
					Cash:      decimal.RequireFromString("500"),
					Positions: map[string]types.PositionSnapshot{},
				},
			},
			want: decimal.RequireFromString("-0.4993"),
		},
		{
			name: "start value <= 0 -> CAGR 0",
			// start portfolio value is 0 => function should guard and return 0
			snapshots: []types.PortfolioView{
				{
					Time:      baseTime,
					Cash:      decimal.RequireFromString("0"),
					Positions: map[string]types.PositionSnapshot{},
				},
				{
					Time:      baseTime.AddDate(1, 0, 0),
					Cash:      decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{},
				},
			},
			want: decimal.RequireFromString("0"),
		},
	}

	for _, tt := range tests {

		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			got := calcCAGR(tt.snapshots, &wg)
			dec100 := decimal.NewFromInt(100)
			if !got.Mul(dec100).Round(2).Equal(tt.want.Mul(dec100).Round(2)) {
				t.Fatalf("calcCAGR got = %v, want %v", got.Mul(dec100).Round(2), tt.want.Mul(dec100).Round(2))
			}
		})
	}
}

func TestCalcAvgWinLossPerTrade(t *testing.T) {
	tests := []struct {
		name        string
		trades      []trade
		wantAvgWin  decimal.Decimal
		wantAvgLoss decimal.Decimal
	}{
		{
			name:        "no executions -> zero win/loss",
			trades:      []trade{},
			wantAvgWin:  decimal.RequireFromString("0"),
			wantAvgLoss: decimal.RequireFromString("0"),
		},
		{
			name: "only unrealized trades (only buys) -> zero win/loss",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					sell: nil,
				},
			},
			wantAvgWin:  decimal.RequireFromString("0"),
			wantAvgLoss: decimal.RequireFromString("0"),
		},
		{
			name: "single realized winning trade",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("120"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
				},
			},
			// gross = -100 + 120 = 20, fees = 2 -> net = 18
			wantAvgWin:  decimal.RequireFromString("18"),
			wantAvgLoss: decimal.RequireFromString("0"),
		},
		{
			name: "single realized losing trade",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("90"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
				},
			},
			// gross = -100 + 90 = -10, fees = 2 -> net = -12
			wantAvgWin:  decimal.RequireFromString("0"),
			wantAvgLoss: decimal.RequireFromString("12"),
		},
		{
			name: "one winner and one loser",
			trades: []trade{
				// winner
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("120"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
				},
				// loser
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("200"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("2"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("180"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("2"),
							},
						},
					},
				},
			},
			// trade1: gross 20, fees 2 -> net 18
			// trade2: gross -20, fees 4 -> net -24
			// avgWin = 18, avgLoss = 24
			wantAvgWin:  decimal.RequireFromString("18"),
			wantAvgLoss: decimal.RequireFromString("24"),
		},
		{
			name: "realized trade with zero net (ignored for both win/loss)",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
			},
			// net = 0, so neither win nor loss bucket
			wantAvgWin:  decimal.RequireFromString("0"),
			wantAvgLoss: decimal.RequireFromString("0"),
		},
		{
			name: "partially closed position (still treated as realized)",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("2"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("150"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
			},
			// gross = -200 + 150 = -50, net = -50 -> single losing trade
			wantAvgWin:  decimal.RequireFromString("0"),
			wantAvgLoss: decimal.RequireFromString("50"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			gotWin, gotLoss := calcAvgWinLossPerTrade(tt.trades, &wg)

			if !gotWin.Equal(tt.wantAvgWin) {
				t.Fatalf("calcAvgWinLossPerTrade avgWin = %s, want %s", gotWin.String(), tt.wantAvgWin.String())
			}
			if !gotLoss.Equal(tt.wantAvgLoss) {
				t.Fatalf("calcAvgWinLossPerTrade avgLoss = %s, want %s", gotLoss.String(), tt.wantAvgLoss.String())
			}
		})
	}
}

func TestCalcMaxDrawdownMetrics(t *testing.T) {
	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		snapshots    []types.PortfolioView
		wantMaxDD    decimal.Decimal
		wantMaxDDPct decimal.Decimal
		wantMaxDDDur time.Duration
	}{{
		name: "Simple drawdown 30%",
		snapshots: []types.PortfolioView{
			newPv(baseTime, "1000"),
			newPv(baseTime.AddDate(0, 0, 1), "10000"),
			newPv(baseTime.AddDate(0, 0, 2), "7000"),
		},
		wantMaxDD:    decimal.RequireFromString("3000"),
		wantMaxDDPct: decimal.RequireFromString("0.3"),
		wantMaxDDDur: time.Hour * 24 * 1,
	},
		{
			name:         "no snapshots -> zero drawdown and duration",
			snapshots:    nil,
			wantMaxDD:    decimal.RequireFromString("0"),
			wantMaxDDPct: decimal.RequireFromString("0"),
			wantMaxDDDur: 0,
		},
		{
			name: "monotonic up -> zero drawdown and duration",
			// equity: 1000 -> 1200 -> 1500
			snapshots: []types.PortfolioView{
				newPv(baseTime, "1000"),
				newPv(baseTime.AddDate(0, 0, 1), "1200"),
				newPv(baseTime.AddDate(0, 0, 2), "1500"),
			},
			wantMaxDD:    decimal.RequireFromString("0"),
			wantMaxDDPct: decimal.RequireFromString("0"),
			wantMaxDDDur: 0,
		},
		{
			name: "single drawdown with full recovery",
			// equity: 1000 -> 1200 -> 900 -> 1300
			// peaks:  1000    1200    1200   1300
			// dd:        0       0     300      0
			// maxDD = 300 at time (day 2) from peak at (day 1)
			// duration = 1 day
			snapshots: []types.PortfolioView{
				newPv(baseTime, "1000"),
				newPv(baseTime.AddDate(0, 0, 1), "1200"), // peak
				newPv(baseTime.AddDate(0, 0, 2), "900"),  // trough for max DD
				newPv(baseTime.AddDate(0, 0, 3), "1300"),
			},
			wantMaxDD:    decimal.RequireFromString("300"),
			wantMaxDDPct: decimal.RequireFromString("0.25"), // 300 / 1200
			wantMaxDDDur: 24 * time.Hour,                    // 1 day from peak to trough
		},
		{
			name: "multiple peaks with deeper later drawdown",
			// equity: 1000 -> 1500 -> 1300 -> 1600 -> 1200
			// peaks:  1000    1500    1500    1600    1600
			// dd:        0       0     200       0     400
			// maxDD = 400 at last point, peak is 1600 at day 3
			// duration = 1 day (from day 3 to day 4)
			snapshots: []types.PortfolioView{
				newPv(baseTime, "1000"),
				newPv(baseTime.AddDate(0, 0, 1), "1500"),
				newPv(baseTime.AddDate(0, 0, 2), "1300"),
				newPv(baseTime.AddDate(0, 0, 3), "1600"), // new peak
				newPv(baseTime.AddDate(0, 0, 4), "1200"), // trough for max DD
			},
			wantMaxDD:    decimal.RequireFromString("400"),
			wantMaxDDPct: decimal.RequireFromString("0.25"), // 400 / 1600
			wantMaxDDDur: 24 * time.Hour,
		},
		{
			name: "flat then drop with no recovery",
			// equity: 1000 -> 1000 -> 800 -> 700
			// peaks:  1000    1000    1000    1000
			// dd:        0       0     200     300
			// maxDD = 300 from peak at day 0 to trough at day 3
			// duration = 3 days
			snapshots: []types.PortfolioView{
				newPv(baseTime, "1000"), // peak
				newPv(baseTime.AddDate(0, 0, 1), "1000"),
				newPv(baseTime.AddDate(0, 0, 2), "800"),
				newPv(baseTime.AddDate(0, 0, 3), "700"), // trough for max DD
			},
			wantMaxDD:    decimal.RequireFromString("300"),
			wantMaxDDPct: decimal.RequireFromString("0.3"), // 300 / 1000
			wantMaxDDDur: 72 * time.Hour,                   // 3 days
		},
		{
			name: "start at zero equity -> no drawdown",
			// equity: 0 -> -100
			// peak starts at 0 and never > 0, so drawdown is effectively 0 by our guard logic
			snapshots: []types.PortfolioView{
				newPv(baseTime, "0"),
				newPv(baseTime.AddDate(0, 0, 1), "-100"),
			},
			wantMaxDD:    decimal.RequireFromString("0"),
			wantMaxDDPct: decimal.RequireFromString("0"),
			wantMaxDDDur: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			gotDD, gotDDPct, gotDur := calcDrawdownMetrics(tt.snapshots, &wg)

			if !gotDD.Equal(tt.wantMaxDD) {
				t.Fatalf("max drawdown = %s, want %s", gotDD.String(), tt.wantMaxDD.String())
			}
			if !gotDDPct.Equal(tt.wantMaxDDPct) {
				t.Fatalf("max drawdown pct = %s, want %s", gotDDPct.String(), tt.wantMaxDDPct.String())
			}
			if gotDur != tt.wantMaxDDDur {
				t.Fatalf("max drawdown duration = %s, want %s", gotDur, tt.wantMaxDDDur)
			}
		})
	}
}

func TestCalcMaxConsecutiveLosses(t *testing.T) {
	baseTime := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		trades []trade
		want   int
	}{
		{
			name:   "no trades -> 0",
			trades: []trade{},
			want:   0,
		},
		{
			name: "second time is higher max consecutive losses",
			trades: []trade{
				// trade1: loss (100 -> 99)
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(2 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("99"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
				// trade2: win (100 -> 1000)
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(3 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("1000"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
				// trade3: loss (100 -> 99)
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(4 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("99"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
				// trade4: loss (100 -> 99)
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(5 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("99"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
			},
			want: 2, // longest streak of consecutive losses by close time
		},
		{
			name: "only unrealized trades (no buy+sell pair) -> 0",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					sell: nil,
				},
				{
					buy: nil,
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(2 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "three consecutive losing trades",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(2 * time.Hour),
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("90"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
				},
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(3 * time.Hour),
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("200"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(4 * time.Hour),
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("150"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
				},
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(5 * time.Hour),
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("300"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(6 * time.Hour),
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("250"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
				},
			},
			want: 3,
		},
		{
			name: "loss streak broken by win and breakeven",
			trades: []trade{
				// trade1: win
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(2 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("120"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				// trade2: loss
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(3 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(4 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("90"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				// trade3: loss
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(5 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(6 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("80"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				// trade4: breakeven
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(7 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(8 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				// trade5: loss
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(9 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(10 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("90"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				// trade6: loss
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(11 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(12 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("80"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
			},
			want: 2,
		},
		{
			name: "order determined by sell time, not slice order",
			trades: []trade{
				// tradeA: closes second (loss)
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(2 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(4 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("90"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				// tradeB: closes first (loss, deeper)
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(3 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("80"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				// tradeC: closes last (win)
				{
					buy: &types.ExecutionReport{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(5 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					sell: &types.ExecutionReport{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(6 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("120"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
			},
			want: 2, // two losses in a row by close time, then a win
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			got := calcMaxConsecutiveLosses(tt.trades, &wg)

			if got != tt.want {
				t.Fatalf("calcMaxConsecutiveLosses() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestMonthlyReturnsFromSnapshots(t *testing.T) {
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		snapshots []types.PortfolioView
		want      []decimal.Decimal
	}{
		{
			name:      "no snapshots -> empty returns",
			snapshots: nil,
			want:      nil,
		},
		{
			name: "single month, simple +10% return",
			// Jan end: 1000, Feb end: 1100 -> (1100/1000 - 1) = 0.10
			snapshots: []types.PortfolioView{
				newPv(base, "1000"),                  // 2020-01-01
				newPv(base.AddDate(0, 1, 0), "1100"), // 2020-02-01
			},
			want: []decimal.Decimal{
				decimal.RequireFromString("0.10"),
			},
		},
		{
			name: "three consecutive months, +10%, 0%, -10%",
			// Month-end values:
			// Jan: 1000, Feb: 1100, Mar: 1100, Apr: 990
			// Returns: Feb/Jan = +10%, Mar/Feb = 0%, Apr/Mar = -10%
			snapshots: []types.PortfolioView{
				newPv(base, "1000"),                  // Jan
				newPv(base.AddDate(0, 1, 0), "1100"), // Feb
				newPv(base.AddDate(0, 2, 0), "1100"), // Mar
				newPv(base.AddDate(0, 3, 0), "990"),  // Apr
			},
			want: []decimal.Decimal{
				decimal.RequireFromString("0.10"),
				decimal.RequireFromString("0.00"),
				decimal.RequireFromString("-0.10"),
			},
		},
		{
			name: "month with zero month-end value is skipped as start for next return",
			// Jan end: 0 (invalid, so Jan->Feb return skipped)
			// Feb end: 1000, Mar end: 1100
			// Only return: Mar/Feb = (1100/1000 - 1) = 0.10
			snapshots: []types.PortfolioView{
				newPv(base, "0"),                     // Jan
				newPv(base.AddDate(0, 1, 0), "1000"), // Feb
				newPv(base.AddDate(0, 2, 0), "1100"), // Mar
			},
			want: []decimal.Decimal{
				decimal.RequireFromString("0.10"),
			},
		},
		{
			name: "snapshots out of order still pick correct month-end values",
			// Jan end: 1000
			// Feb has two snapshots: 1100 (Feb 1), 1050 (Feb 16),
			// month-end should be 1050.
			// Return: Feb/Jan = (1050/1000 - 1) = 0.05
			snapshots: []types.PortfolioView{
				newPv(base.AddDate(0, 1, 15), "1050"), // 2020-02-16
				newPv(base, "1000"),                   // 2020-01-01
				newPv(base.AddDate(0, 1, 0), "1100"),  // 2020-02-01
			},
			want: []decimal.Decimal{
				decimal.RequireFromString("0.05"),
			},
		},
		{
			name: "simple 1 month 18% return",
			// Jan end: 1000, Feb end: 1180
			// Return: (1180/1000 - 1) = 0.18
			snapshots: []types.PortfolioView{
				newPv(base, "1000"),
				newPv(base.Add(31*24*time.Hour), "1180"), // 2020-02-01
			},
			want: []decimal.Decimal{
				decimal.RequireFromString("0.18"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getMonthlyReturns(tt.snapshots)
			if tt.want == nil {
				if got != nil && len(got) != 0 {
					t.Fatalf("expected nil/empty, got=%v", got)
				}
				return
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len(got)=%d, len(want)=%d, got=%v, want=%v", len(got), len(tt.want), got, tt.want)
			}
			for i := range got {
				if !got[i].Equal(tt.want[i]) {
					t.Fatalf("index %d: got=%s, want=%s", i, got[i].String(), tt.want[i].String())
				}
			}
		})
	}
}

func TestCalcSharpeRatioFromSnapshots(t *testing.T) {
	base := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		snapshots  []types.PortfolioView
		riskFree   decimal.Decimal
		wantSharpe decimal.Decimal
	}{
		{
			name: "monthly snapshots, Sharpe ≈ 1.25",
			snapshots: []types.PortfolioView{
				newPv(base.AddDate(0, 0, 0), "1000.00"),  // Jan start
				newPv(base.AddDate(0, 1, 0), "961.08"),   // Feb
				newPv(base.AddDate(0, 2, 0), "940.76"),   // Mar
				newPv(base.AddDate(0, 3, 0), "937.57"),   // Apr
				newPv(base.AddDate(0, 4, 0), "951.06"),   // May
				newPv(base.AddDate(0, 5, 0), "964.73"),   // Jun
				newPv(base.AddDate(0, 6, 0), "995.75"),   // Jul
				newPv(base.AddDate(0, 7, 0), "1045.45"),  // Aug
				newPv(base.AddDate(0, 8, 0), "1116.20"),  // Sep
				newPv(base.AddDate(0, 9, 0), "1092.60"),  // Oct
				newPv(base.AddDate(0, 10, 0), "1088.90"), // Nov
				newPv(base.AddDate(0, 11, 0), "1123.90"), // Dec
				newPv(base.AddDate(1, 0, 0), "1180.00"),  // end of year
			},
			riskFree:   decimal.RequireFromString("0.03"),
			wantSharpe: decimal.RequireFromString("1.25"),
		},
		{
			name: "less than 2 months -> sharpe = 0",
			snapshots: []types.PortfolioView{
				newPv(base, "1000"),
				newPv(base.AddDate(0, 1, 0), "1010"),
			},
			riskFree:   decimal.RequireFromString("0.00"),
			wantSharpe: decimal.RequireFromString("0"),
		},
		{
			name: "flat portfolio → 0 stdev → sharpe = 0",
			snapshots: []types.PortfolioView{
				newPv(base.AddDate(0, 0, 0), "1000"),
				newPv(base.AddDate(0, 1, 0), "1000"),
				newPv(base.AddDate(0, 2, 0), "1000"),
				newPv(base.AddDate(0, 3, 0), "1000"),
			},
			riskFree:   decimal.RequireFromString("0.01"),
			wantSharpe: decimal.RequireFromString("0"),
		}, {
			name: "18% return, 3% rf, 12% vol → sharpe ≈ 1.25",
			snapshots: []types.PortfolioView{
				newPv(base.AddDate(0, 0, 0), "133.33"),  // Month 0
				newPv(base.AddDate(0, 1, 0), "139.46"),  // Month 1
				newPv(base.AddDate(0, 2, 0), "137.06"),  // Month 2
				newPv(base.AddDate(0, 3, 0), "143.36"),  // Month 3
				newPv(base.AddDate(0, 4, 0), "140.89"),  // Month 4
				newPv(base.AddDate(0, 5, 0), "147.37"),  // Month 5
				newPv(base.AddDate(0, 6, 0), "144.83"),  // Month 6
				newPv(base.AddDate(0, 7, 0), "151.50"),  // Month 7
				newPv(base.AddDate(0, 8, 0), "148.88"),  // Month 8
				newPv(base.AddDate(0, 9, 0), "155.73"),  // Month 9
				newPv(base.AddDate(0, 10, 0), "153.05"), // Month 10
				newPv(base.AddDate(0, 11, 0), "160.09"), // Month 11
				newPv(base.AddDate(0, 12, 0), "157.33"), // Month 12
			},
			riskFree:   decimal.RequireFromString("0.03"),
			wantSharpe: decimal.RequireFromString("1.2499"),
		},
		{
			name: "constant % monthly growth → zero volatility → sharpe = 0",
			snapshots: []types.PortfolioView{
				newPv(base.AddDate(0, 0, 0), "1000"),         // start
				newPv(base.AddDate(0, 1, 0), "1050"),         // 1000 * 1.05
				newPv(base.AddDate(0, 2, 0), "1102.5"),       // 1050 * 1.05
				newPv(base.AddDate(0, 3, 0), "1157.625"),     // 1102.5 * 1.05
				newPv(base.AddDate(0, 4, 0), "1215.50625"),   // 1157.625 * 1.05
				newPv(base.AddDate(0, 5, 0), "1276.2815625"), // 1215.50625 * 1.05
			},
			riskFree:   decimal.RequireFromString("0.02"),
			wantSharpe: decimal.RequireFromString("0"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			got := calcSharpeRatio(tt.snapshots, tt.riskFree, &wg)
			if !got.Round(4).Equal(tt.wantSharpe.Round(4)) {
				t.Fatalf("got=%s, want=%s", got.Round(4), tt.wantSharpe.Round(4))
			}
		})
	}
}

func TestExecutionsToTrades(t *testing.T) {
	baseTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		executions []types.ExecutionReport
		wantTrades []trade
	}{
		{
			name: "single long trade (buy then sell)",
			executions: []types.ExecutionReport{
				{
					Ticker:         "AAPL",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime,
					TotalFilledQty: decimal.NewFromInt(10),
				},
				{
					Ticker:         "AAPL",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(time.Minute),
					TotalFilledQty: decimal.NewFromInt(10),
				},
			},
			wantTrades: []trade{
				{
					buy: &types.ExecutionReport{
						Ticker:         "AAPL",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime,
						TotalFilledQty: decimal.NewFromInt(10),
					},
					sell: &types.ExecutionReport{
						Ticker:         "AAPL",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(time.Minute),
						TotalFilledQty: decimal.NewFromInt(10),
					},
					qty: decimal.NewFromInt(10),
				},
			},
		},
		{
			name: "single short trade (sell then buy)",
			executions: []types.ExecutionReport{
				{
					Ticker:         "MSFT",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime,
					TotalFilledQty: decimal.NewFromInt(5),
				},
				{
					Ticker:         "MSFT",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime.Add(time.Minute),
					TotalFilledQty: decimal.NewFromInt(5),
				},
			},
			wantTrades: []trade{
				{
					// normalized so buy is the buy leg, sell is the sell leg,
					// regardless of chronological order
					buy: &types.ExecutionReport{
						Ticker:         "MSFT",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(time.Minute),
						TotalFilledQty: decimal.NewFromInt(5),
					},
					sell: &types.ExecutionReport{
						Ticker:         "MSFT",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime,
						TotalFilledQty: decimal.NewFromInt(5),
					},
					qty: decimal.NewFromInt(5),
				},
			},
		},
		{
			name: "mixed long and short across tickers",
			executions: []types.ExecutionReport{
				// AAPL long
				{
					Ticker:         "AAPL",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime,
					TotalFilledQty: decimal.NewFromInt(10),
				},
				{
					Ticker:         "AAPL",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(time.Minute),
					TotalFilledQty: decimal.NewFromInt(10),
				},
				// MSFT short
				{
					Ticker:         "MSFT",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(30 * time.Second),
					TotalFilledQty: decimal.NewFromInt(3),
				},
				{
					Ticker:         "MSFT",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime.Add(2 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(3),
				},
			},
			// this assumes executionsToTrades returns trades in the order the
			// pairs are completed: first AAPL, then MSFT
			wantTrades: []trade{
				{
					buy: &types.ExecutionReport{
						Ticker:         "AAPL",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime,
						TotalFilledQty: decimal.NewFromInt(10),
					},
					sell: &types.ExecutionReport{
						Ticker:         "AAPL",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(time.Minute),
						TotalFilledQty: decimal.NewFromInt(10),
					},
					qty: decimal.NewFromInt(10),
				},
				{
					buy: &types.ExecutionReport{
						Ticker:         "MSFT",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(2 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(3),
					},
					sell: &types.ExecutionReport{
						Ticker:         "MSFT",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(30 * time.Second),
						TotalFilledQty: decimal.NewFromInt(3),
					},
					qty: decimal.NewFromInt(3),
				},
			},
		},
		{
			name: "unmatched final buy results in trade with only buy side",
			executions: []types.ExecutionReport{
				{
					Ticker:         "AAPL",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime,
					TotalFilledQty: decimal.NewFromInt(10),
				},
				{
					Ticker:         "AAPL",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(time.Minute),
					TotalFilledQty: decimal.NewFromInt(10),
				},
				// unmatched buy (open long)
				{
					Ticker:         "AAPL",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime.Add(2 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(5),
				},
			},
			wantTrades: []trade{
				{
					buy: &types.ExecutionReport{
						Ticker:         "AAPL",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime,
						TotalFilledQty: decimal.NewFromInt(10),
					},
					sell: &types.ExecutionReport{
						Ticker:         "AAPL",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(time.Minute),
						TotalFilledQty: decimal.NewFromInt(10),
					},
					qty: decimal.NewFromInt(10),
				},
				{
					buy: &types.ExecutionReport{
						Ticker:         "AAPL",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(2 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(5),
					},
					sell: nil,
					qty:  decimal.Zero,
				},
			},
		},
		{
			name: "unmatched final sell results in trade with only sell side",
			executions: []types.ExecutionReport{
				{
					Ticker:         "MSFT",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime,
					TotalFilledQty: decimal.NewFromInt(4),
				},
				{
					Ticker:         "MSFT",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime.Add(time.Minute),
					TotalFilledQty: decimal.NewFromInt(4),
				},
				// unmatched sell (open short)
				{
					Ticker:         "MSFT",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(2 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(2),
				},
			},
			wantTrades: []trade{
				{
					buy: &types.ExecutionReport{
						Ticker:         "MSFT",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(time.Minute),
						TotalFilledQty: decimal.NewFromInt(4),
					},
					sell: &types.ExecutionReport{
						Ticker:         "MSFT",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime,
						TotalFilledQty: decimal.NewFromInt(4),
					},
					qty: decimal.NewFromInt(4),
				},
				{
					buy: nil,
					sell: &types.ExecutionReport{
						Ticker:         "MSFT",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(2 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(2),
					},
					qty: decimal.Zero,
				},
			},
		},
		{
			name: "short then long, both trades completed",
			executions: []types.ExecutionReport{
				// First: short 5, then cover
				{
					Ticker:         "TSLA",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime,
					TotalFilledQty: decimal.NewFromInt(5),
				},
				{
					Ticker:         "TSLA",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime.Add(1 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(5),
				},
				// Then: long 10, then close
				{
					Ticker:         "TSLA",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime.Add(2 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(10),
				},
				{
					Ticker:         "TSLA",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(3 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(10),
				},
			},
			wantTrades: []trade{
				{
					// normalized: buy is the buy leg, sell is the sell leg
					buy: &types.ExecutionReport{
						Ticker:         "TSLA",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(1 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(5),
					},
					sell: &types.ExecutionReport{
						Ticker:         "TSLA",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime,
						TotalFilledQty: decimal.NewFromInt(5),
					},
					qty: decimal.NewFromInt(5),
				},
				{
					buy: &types.ExecutionReport{
						Ticker:         "TSLA",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(2 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(10),
					},
					sell: &types.ExecutionReport{
						Ticker:         "TSLA",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(3 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(10),
					},
					qty: decimal.NewFromInt(10),
				},
			},
		},
		{
			name: "long then short, both trades completed",
			executions: []types.ExecutionReport{
				// First: long 7, then close
				{
					Ticker:         "GOOG",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime,
					TotalFilledQty: decimal.NewFromInt(7),
				},
				{
					Ticker:         "GOOG",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(1 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(7),
				},
				// Then: short 3, then cover
				{
					Ticker:         "GOOG",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(2 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(3),
				},
				{
					Ticker:         "GOOG",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime.Add(3 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(3),
				},
			},
			wantTrades: []trade{
				{
					buy: &types.ExecutionReport{
						Ticker:         "GOOG",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime,
						TotalFilledQty: decimal.NewFromInt(7),
					},
					sell: &types.ExecutionReport{
						Ticker:         "GOOG",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(1 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(7),
					},
					qty: decimal.NewFromInt(7),
				},
				{
					buy: &types.ExecutionReport{
						Ticker:         "GOOG",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(3 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(3),
					},
					sell: &types.ExecutionReport{
						Ticker:         "GOOG",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(2 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(3),
					},
					qty: decimal.NewFromInt(3),
				},
			},
		},
		{
			name: "short then long, with unfinished long",
			executions: []types.ExecutionReport{
				// Completed short: sell 5 then buy 5
				{
					Ticker:         "TSLA",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime,
					TotalFilledQty: decimal.NewFromInt(5),
				},
				{
					Ticker:         "TSLA",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime.Add(1 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(5),
				},
				// Long 10, only 6 gets closed → open long 4
				{
					Ticker:         "TSLA",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime.Add(2 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(10),
				},
				{
					Ticker:         "TSLA",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(3 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(6),
				},
			},
			wantTrades: []trade{
				{
					buy: &types.ExecutionReport{
						Ticker:         "TSLA",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(1 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(5),
					},
					sell: &types.ExecutionReport{
						Ticker:         "TSLA",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime,
						TotalFilledQty: decimal.NewFromInt(5),
					},
					qty: decimal.NewFromInt(5),
				},
				{
					buy: &types.ExecutionReport{
						Ticker:         "TSLA",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(2 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(10),
					},
					sell: &types.ExecutionReport{
						Ticker:         "TSLA",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(3 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(6),
					},
					qty: decimal.NewFromInt(6),
				},
				{
					// open long of 4 → buy-only trade with qty = 0
					buy: &types.ExecutionReport{
						Ticker:         "TSLA",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(2 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(10),
					},
					sell: nil,
					qty:  decimal.Zero,
				},
			},
		},
		{
			name: "long then short, with unfinished short",
			executions: []types.ExecutionReport{
				// Completed long: buy 5 then sell 5
				{
					Ticker:         "GOOG",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime,
					TotalFilledQty: decimal.NewFromInt(5),
				},
				{
					Ticker:         "GOOG",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(1 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(5),
				},
				// Short 10, only 6 gets covered → open short 4
				{
					Ticker:         "GOOG",
					Side:           types.SideTypeSell,
					ReportTime:     baseTime.Add(2 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(10),
				},
				{
					Ticker:         "GOOG",
					Side:           types.SideTypeBuy,
					ReportTime:     baseTime.Add(3 * time.Minute),
					TotalFilledQty: decimal.NewFromInt(6),
				},
			},
			wantTrades: []trade{
				{
					buy: &types.ExecutionReport{
						Ticker:         "GOOG",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime,
						TotalFilledQty: decimal.NewFromInt(5),
					},
					sell: &types.ExecutionReport{
						Ticker:         "GOOG",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(1 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(5),
					},
					qty: decimal.NewFromInt(5),
				},
				{
					buy: &types.ExecutionReport{
						Ticker:         "GOOG",
						Side:           types.SideTypeBuy,
						ReportTime:     baseTime.Add(3 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(6),
					},
					sell: &types.ExecutionReport{
						Ticker:         "GOOG",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(2 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(10),
					},
					qty: decimal.NewFromInt(6),
				},
				{
					// open short of 4 → sell-only trade with qty = 0
					buy: nil,
					sell: &types.ExecutionReport{
						Ticker:         "GOOG",
						Side:           types.SideTypeSell,
						ReportTime:     baseTime.Add(2 * time.Minute),
						TotalFilledQty: decimal.NewFromInt(10),
					},
					qty: decimal.Zero,
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &portfolio{
				executions: tc.executions,
			}

			got := executionsToTrades(p)

			if len(got) != len(tc.wantTrades) {
				t.Fatalf("executionsToTrades() returned %d trades, want %d", len(got), len(tc.wantTrades))
			}
			for i := range tc.wantTrades {
				if !reflect.DeepEqual(got[i], tc.wantTrades[i]) {
					t.Fatalf("executionsToTrades() returned trade %+v, want %+v", got[i], tc.wantTrades[i])
				}
			}
		})
	}
}

func TestWinLossRatioDecimal(t *testing.T) {
	tests := []struct {
		name        string
		trades      []trade
		wantWins    decimal.Decimal
		wantLosses  decimal.Decimal
		wantWinRate decimal.Decimal
	}{
		{
			name:        "no trades",
			trades:      []trade{},
			wantWins:    decimal.RequireFromString("0"),
			wantLosses:  decimal.RequireFromString("0"),
			wantWinRate: decimal.RequireFromString("0"),
		},
		{
			name: "one winning trade",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						TotalFilledQty: decimal.RequireFromString("1"),
						AvgFillPrice:   decimal.RequireFromString("100"),
						TotalFees:      decimal.RequireFromString("0"),
					},
					sell: &types.ExecutionReport{
						TotalFilledQty: decimal.RequireFromString("1"),
						AvgFillPrice:   decimal.RequireFromString("110"),
						TotalFees:      decimal.RequireFromString("0"),
					},
				},
			},
			wantWins:    decimal.RequireFromString("1"),
			wantLosses:  decimal.RequireFromString("0"),
			wantWinRate: decimal.RequireFromString("1"),
		},
		{
			name: "one winning and one losing trade",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						TotalFilledQty: decimal.RequireFromString("1"),
						AvgFillPrice:   decimal.RequireFromString("100"),
						TotalFees:      decimal.RequireFromString("1"),
					},
					sell: &types.ExecutionReport{
						TotalFilledQty: decimal.RequireFromString("1"),
						AvgFillPrice:   decimal.RequireFromString("110"),
						TotalFees:      decimal.RequireFromString("1"),
					},
				},
				{
					buy: &types.ExecutionReport{
						TotalFilledQty: decimal.RequireFromString("1"),
						AvgFillPrice:   decimal.RequireFromString("100"),
						TotalFees:      decimal.RequireFromString("0"),
					},
					sell: &types.ExecutionReport{
						TotalFilledQty: decimal.RequireFromString("1"),
						AvgFillPrice:   decimal.RequireFromString("95"),
						TotalFees:      decimal.RequireFromString("0"),
					},
				},
			},
			wantWins:    decimal.RequireFromString("1"),
			wantLosses:  decimal.RequireFromString("1"),
			wantWinRate: decimal.RequireFromString("0.5"),
		},
		{
			name: "partial fill uses smaller qty and counts as win",
			trades: []trade{
				{
					buy: &types.ExecutionReport{
						TotalFilledQty: decimal.RequireFromString("2"),
						AvgFillPrice:   decimal.RequireFromString("100"),
						TotalFees:      decimal.RequireFromString("0"),
					},
					sell: &types.ExecutionReport{
						TotalFilledQty: decimal.RequireFromString("1"),
						AvgFillPrice:   decimal.RequireFromString("120"),
						TotalFees:      decimal.RequireFromString("0"),
					},
				},
			},
			wantWins:    decimal.RequireFromString("1"),
			wantLosses:  decimal.RequireFromString("0"),
			wantWinRate: decimal.RequireFromString("1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			gotWinRate := calcWinLossRatio(tt.trades, &wg)

			if !gotWinRate.Equal(tt.wantWinRate) {
				t.Errorf("winRate = %s, want %s", gotWinRate, tt.wantWinRate)
			}
		})
	}
}

// Helper functions
func newPv(t time.Time, cashStr string) types.PortfolioView {
	return types.PortfolioView{
		Time:      t,
		Cash:      decimal.RequireFromString(cashStr),
		Positions: map[string]types.PositionSnapshot{},
	}
}
