package strategies

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/ChizhovVadim/trader/pkg/brokers"
	"github.com/ChizhovVadim/trader/pkg/usercommands"
)

type Trader struct {
	logger     *slog.Logger
	inbox      chan any
	Broker     *brokers.MultyBroker
	signals    []*SignalService
	portfolios []*PortfolioService
	strategies []*StrategyService
}

func NewTrader(
	logger *slog.Logger,
) *Trader {
	return &Trader{
		logger: logger,
		inbox:  make(chan any),
		Broker: brokers.NewMultyBroker(logger),
	}
}

func (app *Trader) Close() error {
	return app.Broker.Close()
}

func (app *Trader) Inbox() chan<- any {
	return app.inbox
}

func (app *Trader) AddSignal(signal *SignalService) {
	app.signals = append(app.signals, signal)
}

func (app *Trader) AddStrategy(strategy *StrategyService) {
	app.strategies = append(app.strategies, strategy)
}

// Каждый сигнал торгуем в каждом портфеле
func (app *Trader) AddStrategiesForAllSignalPortfolioPairs() {
	for _, signal := range app.signals {
		for _, portfolio := range app.portfolios {
			app.AddStrategy(NewStrategyService(app.logger, portfolio.broker, portfolio.portfolio, signal.security, signal.name))
		}
	}
}

func (app *Trader) AddPortfolio(portfolio *PortfolioService) {
	app.portfolios = append(app.portfolios, portfolio)
}

func (app *Trader) checkStatus() {
	app.Broker.CheckStatus()

	for _, signal := range app.signals {
		signal.CheckStatus()
	}
	fmt.Println("Total signals:", len(app.signals))

	for _, portfolio := range app.portfolios {
		portfolio.CheckStatus()
	}
	fmt.Println("Total portfolios:", len(app.portfolios))

	for _, strategy := range app.strategies {
		strategy.CheckStatus()
	}
	fmt.Println("Total strategies:", len(app.strategies))
}

func (app *Trader) init(ctx context.Context) error {
	app.logger.Info("Strategies starting...")
	if err := app.Broker.Init(ctx); err != nil {
		return err
	}
	for _, portfolio := range app.portfolios {
		var err = portfolio.Init()
		if err != nil {
			return err
		}
	}
	for _, strategy := range app.strategies {
		var err = strategy.Init()
		if err != nil {
			return err
		}
	}
	// сигналы последние, тк они подписываются на бары
	for _, signal := range app.signals {
		var err = signal.Init()
		if err != nil {
			return err
		}
	}
	app.logger.Info("Strategies started.")
	return nil
}

func (app *Trader) Run(ctx context.Context) error {
	if err := app.init(ctx); err != nil {
		return err
	}
	go func() {
		var err = usercommands.Handle(ctx, app.inbox)
		if err != nil {
			app.logger.Error("usercommands.Handle", "error", err)
			return
		}
	}()
	return app.eventLoop(ctx)
}

func (app *Trader) onCandle(candle brokers.Candle) bool {
	var orderRegistered bool
	for _, signalStrategy := range app.signals {
		var signal = signalStrategy.OnCandle(candle)
		if signal.DateTime.IsZero() {
			continue
		}
		for _, strategy := range app.strategies {
			if strategy.OnSignal(signal) {
				orderRegistered = true
			}
		}
	}
	return orderRegistered
}

func (app *Trader) eventLoop(ctx context.Context) error {
	var shouldCheckStatus = time.After(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-shouldCheckStatus:
			shouldCheckStatus = nil
			app.checkStatus()
		case msg, ok := <-app.inbox:
			if !ok {
				app.inbox = nil
				continue
			}
			switch msg := msg.(type) {
			case usercommands.ExitUserCmd:
				return nil
			case usercommands.CheckStatusUserCmd:
				app.checkStatus()
			case brokers.Candle:
				if app.onCandle(msg) {
					if shouldCheckStatus == nil {
						shouldCheckStatus = time.After(10 * time.Second)
					}
				}
			}
		}
	}
}
