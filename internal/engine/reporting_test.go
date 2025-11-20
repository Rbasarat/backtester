package engine

import (
	"backtester/types"
	"sync"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestCalcNetProfit(t *testing.T) {
	tests := []struct {
		name       string
		executions map[string][]types.ExecutionReport
		want       decimal.Decimal
	}{
		{
			name:       "no executions -> zero",
			executions: map[string][]types.ExecutionReport{},
			want:       decimal.RequireFromString("0"),
		},
		{
			name: "only buys -> unrealized -> zero",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0.5"),
							},
						},
					},
				},
			},
			want: decimal.RequireFromString("-0.5"),
		},
		{
			name: "only sells -> unrealized -> zero",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
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
			want: decimal.RequireFromString("-0.1"),
		},
		{
			name: "simple realized long trade (buy then sell, with fees)",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					{
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
			want: decimal.RequireFromString("8"),
		},
		{
			name: "partially closed position still counted as realized (has buy and sell)",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("2"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					{
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
			want: decimal.RequireFromString("-90"),
		},
		{
			name: "multiple trades: some realized, some not",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					{
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
				"trade2": {
					{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("50"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("60"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
				},
				"trade3": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("10"),
								Quantity: decimal.RequireFromString("5"),
								Fee:      decimal.RequireFromString("0.1"),
							},
						},
					},
				},
			},
			want: decimal.RequireFromString("-2.1"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			got := calcNetProfit(tt.executions, &wg)

			if !got.Equal(tt.want) {
				t.Fatalf("calcNetProfit() = %s, want %s", got.String(), tt.want.String())
			}
		})
	}
}

func TestNetAvgProfitPerTrade(t *testing.T) {
	tests := []struct {
		name       string
		executions map[string][]types.ExecutionReport
		want       decimal.Decimal
	}{
		{
			name:       "no executions => 0",
			executions: map[string][]types.ExecutionReport{},
			want:       decimal.RequireFromString("0"),
		},
		{
			name: "only buys (no realized trades) => 0",
			// Note: even though fees are present, realizedTrades==0 so function returns 0
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0.5"),
							},
						},
					},
				},
			},
			want: decimal.RequireFromString("0"),
		},
		{
			name: "simple realized long trade (buy then sell, with fees)",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					{
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
			want: decimal.RequireFromString("8"),
		},
		{
			name: "partially closed position is treated as realized",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("2"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					{
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
			want: decimal.RequireFromString("-90"),
		},
		{
			name: "one realized trade + one unrealized trade (fees from unrealized still counted)",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					{
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
				"trade2": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("50"),
								Quantity: decimal.RequireFromString("2"),
								Fee:      decimal.RequireFromString("0.5"),
							},
							{
								Price:    decimal.RequireFromString("50"),
								Quantity: decimal.RequireFromString("0"), // example extra fill, not necessary
								Fee:      decimal.RequireFromString("0.5"),
							},
						},
					},
				},
			},
			want: decimal.RequireFromString("7"),
		},
		{
			name: "two realized trades, no unrealized",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					{
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
				"trade2": {
					{
						Side: types.SideTypeSell,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("200"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0.5"),
							},
						},
					},
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("150"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0.5"),
							},
						},
					},
				},
			},
			want: decimal.RequireFromString("28.5"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			got := calcNetAvgProfitPerTrade(tt.executions, &wg)

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
							Symbol:        "AAA",
							Quantity:      decimal.RequireFromString("1"),
							LastPrice:     decimal.RequireFromString("9000"),
							AvgEntryPrice: decimal.RequireFromString("0"),
						},
					},
				},
				{
					Time: baseTime.AddDate(3, 0, 0), // approx 1 year later
					Cash: decimal.RequireFromString("5000"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Symbol:        "AAA",
							Quantity:      decimal.RequireFromString("1"),
							LastPrice:     decimal.RequireFromString("14000"),
							AvgEntryPrice: decimal.RequireFromString("0"),
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
							Symbol:        "AAA",
							Quantity:      decimal.RequireFromString("1"),
							LastPrice:     decimal.RequireFromString("9000"),
							AvgEntryPrice: decimal.RequireFromString("0"),
						},
					},
				},
				{
					Time: baseTime.AddDate(5, 0, 0), // approx 1 year later
					Cash: decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Symbol:        "AAA",
							Quantity:      decimal.RequireFromString("1"),
							LastPrice:     decimal.RequireFromString("14000"),
							AvgEntryPrice: decimal.RequireFromString("0"),
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
							Symbol:        "AAA",
							Quantity:      decimal.RequireFromString("10"),
							LastPrice:     decimal.RequireFromString("0"),
							AvgEntryPrice: decimal.RequireFromString("0"),
						},
					},
				},
				{
					Time: baseTime.AddDate(1, 0, 0), // approx 1 year later
					Cash: decimal.RequireFromString("1000"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Symbol:        "AAA",
							Quantity:      decimal.RequireFromString("10"),
							LastPrice:     decimal.RequireFromString("0"),
							AvgEntryPrice: decimal.RequireFromString("0"),
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
							Symbol:        "AAA",
							Quantity:      decimal.RequireFromString("10"),
							LastPrice:     decimal.RequireFromString("100"),
							AvgEntryPrice: decimal.RequireFromString("100"),
						},
					},
				},
				{
					Time: baseTime.AddDate(2, 0, 0), // ~2 years later
					Cash: decimal.RequireFromString("0"),
					Positions: map[string]types.PositionSnapshot{
						"AAA": {
							Symbol:        "AAA",
							Quantity:      decimal.RequireFromString("10"),
							LastPrice:     decimal.RequireFromString("200"),
							AvgEntryPrice: decimal.RequireFromString("100"),
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
		tt := tt // capture range var
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
		executions  map[string][]types.ExecutionReport
		wantAvgWin  decimal.Decimal
		wantAvgLoss decimal.Decimal
	}{
		{
			name:        "no executions -> zero win/loss",
			executions:  map[string][]types.ExecutionReport{},
			wantAvgWin:  decimal.RequireFromString("0"),
			wantAvgLoss: decimal.RequireFromString("0"),
		},
		{
			name: "only unrealized trades (only buys) -> zero win/loss",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
				},
			},
			wantAvgWin:  decimal.RequireFromString("0"),
			wantAvgLoss: decimal.RequireFromString("0"),
		},
		{
			name: "single realized winning trade",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					{
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
			wantAvgWin:  decimal.RequireFromString("18"),
			wantAvgLoss: decimal.RequireFromString("0"),
		},
		{
			name: "single realized losing trade",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					{
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
			wantAvgWin:  decimal.RequireFromString("0"),
			wantAvgLoss: decimal.RequireFromString("12"),
		},
		{
			name: "one winner and one loser",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("1"),
							},
						},
					},
					{
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
				"trade2": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("200"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("2"),
							},
						},
					},
					{
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
			wantAvgWin:  decimal.RequireFromString("18"),
			wantAvgLoss: decimal.RequireFromString("24"),
		},
		{
			name: "realized trade with zero net (ignored for both win/loss)",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					{
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
			wantAvgWin:  decimal.RequireFromString("0"),
			wantAvgLoss: decimal.RequireFromString("0"),
		},
		{
			name: "partially closed position (still treated as realized)",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side: types.SideTypeBuy,
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("2"),
								Fee:      decimal.RequireFromString("0"),
							},
						},
					},
					{
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
			wantAvgWin:  decimal.RequireFromString("0"),
			wantAvgLoss: decimal.RequireFromString("50"),
		},
	}

	for _, tt := range tests {
		tt := tt // capture range var
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			gotWin, gotLoss := calcAvgWinLossPerTrade(tt.executions, &wg)

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
		tt := tt // capture range var
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
		name       string
		executions map[string][]types.ExecutionReport
		want       int
	}{
		{
			name:       "no trades -> 0",
			executions: map[string][]types.ExecutionReport{},
			want:       0,
		},
		{
			name: "second time is higher max consecutive losses",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0")},
						},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(2 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("99"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0")},
						},
					},
				},
				"trade2": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0")},
						},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(3 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("1000"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0")},
						},
					},
				},
				"trade3": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0")},
						},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(4 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("99"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0")},
						},
					},
				},
				"trade4": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0")},
						},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(5 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("99"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0")},
						},
					},
				},
			},
			want: 2,
		},
		{
			name: "only unrealized trades (no buy+sell pair) -> 0",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0")},
						},
					},
				},
				"trade2": {
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(2 * time.Hour),
						Fills: []types.Fill{
							{
								Price:    decimal.RequireFromString("100"),
								Quantity: decimal.RequireFromString("1"),
								Fee:      decimal.RequireFromString("0")},
						},
					},
				},
			},
			want: 0,
		},
		{
			name: "three consecutive losing trades",
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(2 * time.Hour), // closeTime
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("90"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
				},
				"trade2": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(3 * time.Hour),
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("200"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(4 * time.Hour),
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("150"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
				},
				"trade3": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(5 * time.Hour),
						Fills: []types.Fill{
							{Price: decimal.RequireFromString("300"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")},
						},
					},
					{
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
			executions: map[string][]types.ExecutionReport{
				"trade1": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(2 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("120"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				"trade2": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(3 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(4 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("90"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				"trade3": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(5 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(6 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("80"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				"trade4": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(7 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(8 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				"trade5": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(9 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(10 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("90"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
				"trade6": {
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(11 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(12 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("80"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
				},
			},
			want: 2,
		},
		{
			name: "order determined by sell time, not map key",
			executions: map[string][]types.ExecutionReport{
				"tradeA": { // closes second
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(2 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(4 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("90"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}}, // loss
					},
				},
				"tradeB": { // closes first
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(1 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(3 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("80"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}}, // deeper loss
					},
				},
				"tradeC": { // closes last
					{
						Side:       types.SideTypeBuy,
						ReportTime: baseTime.Add(5 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("100"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}},
					},
					{
						Side:       types.SideTypeSell,
						ReportTime: baseTime.Add(6 * time.Hour),
						Fills:      []types.Fill{{Price: decimal.RequireFromString("120"), Quantity: decimal.RequireFromString("1"), Fee: decimal.RequireFromString("0")}}, // win
					},
				},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		tt := tt // capture range var
		t.Run(tt.name, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)

			got := calcMaxConsecutiveLosses(tt.executions, &wg)

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

// Helper functions
func newPv(t time.Time, cashStr string) types.PortfolioView {
	return types.PortfolioView{
		Time:      t,
		Cash:      decimal.RequireFromString(cashStr),
		Positions: map[string]types.PositionSnapshot{},
	}
}
