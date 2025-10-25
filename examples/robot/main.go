package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/ChizhovVadim/trader/pkg/brokers"
	"github.com/ChizhovVadim/trader/pkg/brokers/quik"
	"github.com/ChizhovVadim/trader/pkg/moex"
	"github.com/ChizhovVadim/trader/pkg/strategies"
)

func main() {
	var logger = slog.New(
		slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	var err = robotHandler(logger)
	if err != nil {
		logger.Error("run failed",
			"error", err)
		return
	}
}

func robotHandler(logger *slog.Logger) error {
	var trader = strategies.NewTrader(logger)
	defer trader.Close()

	trader.Broker.Add("paper", brokers.NewMockBroker(logger, "paper"))         // Для сделок
	var marketData = quik.NewQuikBroker(logger, "quik", 34132, trader.Inbox()) // Для получения баров
	trader.Broker.Add("quik", marketData)

	var security, err = moex.GetSecurityInfo("Si-12.25")
	if err != nil {
		return err
	}
	trader.AddSignal(strategies.NewSignalService(logger, "signal", marketData, security, "minutes5",
		&AdvisorSample{}, strategies.SizeConfig{MaxLever: 5, LongLever: 5, ShortLever: 5, Weight: 1}))

	var portfolio = &strategies.Portfolio{Portfolio: brokers.Portfolio{Client: "paper", Portfolio: "test"}}
	trader.AddPortfolio(strategies.NewPortfolioService(logger, trader.Broker, portfolio, 0, 0))
	trader.AddStrategy(strategies.NewStrategyService(logger, trader.Broker, portfolio, security, "signal"))
	return trader.Run(context.Background())
}
