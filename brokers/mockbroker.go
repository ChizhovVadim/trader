package brokers

import (
	"context"
	"fmt"
	"iter"
	"log/slog"
)

var _ IBroker = (*MockBroker)(nil)
var _ IMarketData = (*MockBroker)(nil)

type MockBroker struct {
	logger    *slog.Logger
	name      string
	positions map[string]float64
}

func NewMockBroker(logger *slog.Logger, name string) *MockBroker {
	return &MockBroker{
		logger:    logger,
		name:      name,
		positions: make(map[string]float64),
	}
}

func (b *MockBroker) Init(context.Context) error {
	return nil
}

func (b *MockBroker) CheckStatus() {
	fmt.Printf("%10s %10s\n", b.name, "mock")
}

func (b *MockBroker) GetPortfolioLimits(portfolio Portfolio) (PortfolioLimits, error) {
	return PortfolioLimits{
		StartLimitOpenPos: 1_000_000,
	}, nil
}

func (b *MockBroker) GetPosition(portfolio Portfolio, security Security) (float64, error) {
	return b.positions[b.positionKey(portfolio, security)], nil
}

func (b *MockBroker) RegisterOrder(order Order) error {
	b.logger.Info("RegisterOrder",
		"client", order.Portfolio.Client,
		"portfolio", order.Portfolio.Portfolio,
		"security", order.Security.Name,
		"volume", order.Volume,
		"price", order.Price)
	b.positions[b.positionKey(order.Portfolio, order.Security)] += float64(order.Volume)
	return nil
}

func (b *MockBroker) Close() error {
	return nil
}

func (b *MockBroker) positionKey(portfolio Portfolio, security Security) string {
	return portfolio.Portfolio + security.Code
}

func (b *MockBroker) GetLastCandles(security Security, timeframe string) iter.Seq2[HistoryCandle, error] {
	return func(yield func(HistoryCandle, error) bool) {}
}

func (b *MockBroker) SubscribeCandles(security Security, timeframe string) error {
	b.logger.Debug("SubscribeCandles",
		"security", security.Code,
		"timeframe", timeframe)
	return nil
}
