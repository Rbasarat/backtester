package main

import (
	"backtester/internal/engine"
	"backtester/internal/repository"
	"backtester/types"
	"context"
	"fmt"
	"github.com/shopspring/decimal"
	"log"
	"time"
)

const (
	dburl  = "postgresql://moneymaker:moneymaker@localhost:5432/moneymaker"
	ticker = "AAPL"
)

func main() {
	db, err := repository.NewDatabase(dburl)
	if err != nil {
		log.Fatal(err)
	}
	ctx := context.Background()
	asset, err := db.GetAssetByTicker(ticker, ctx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(asset)
	fmt.Println("-------")

	start := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2022, 1, 10, 0, 0, 0, 0, time.UTC)
	//candles, err := db.GetAggregates(98413, types.Day, start, end, ctx)
	//if err != nil {
	//	log.Fatal(err)
	//}
	//for _, candle := range candles {
	//	fmt.Println(candle)
	//}
	feeds := []*engine.DataFeedConfig{
		engine.NewDataFeed("AAPL", types.Day, start, end),
	}
	eng := engine.NewEngine(
		feeds,
		strat{},
		allo{},
		broker{},
		engine.NewPortfolioConfig(decimal.NewFromInt(1000)),
		&db,
	)
	eng.Run()

}

type strat struct {
}

func (s strat) Init(api engine.PortfolioApi) error {
	return nil
}

func (s strat) OnCandle(candle types.Candle) []types.Signal {
	return nil
}

type allo struct{}

func (a allo) Allocate(signals []types.Signal, view types.PortfolioView) []types.Order {
	return nil
}

type broker struct{}

func (b broker) Execute(orders []types.Order) {

}
