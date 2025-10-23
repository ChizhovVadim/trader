package brokers

import (
	"context"
	"fmt"
	"log/slog"
)

var _ IBroker = (*MultyBroker)(nil)

type MultyBroker struct {
	logger  *slog.Logger
	brokers map[string]IBroker
}

func NewMultyBroker(logger *slog.Logger) *MultyBroker {
	return &MultyBroker{
		logger:  logger,
		brokers: make(map[string]IBroker),
	}
}

func (b *MultyBroker) Add(key string, broker IBroker) {
	b.brokers[key] = broker
}

func (b *MultyBroker) Get(key string) IBroker {
	return b.brokers[key]
}

func (b *MultyBroker) Init(ctx context.Context) error {
	for _, child := range b.brokers {
		var err = child.Init(ctx)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *MultyBroker) CheckStatus() {
	for _, child := range b.brokers {
		child.CheckStatus()
	}
	fmt.Println("Total brokers:", len(b.brokers))
}

func (b *MultyBroker) GetPortfolioLimits(portfolio Portfolio) (PortfolioLimits, error) {
	return b.brokers[portfolio.Client].GetPortfolioLimits(portfolio)
}

func (b *MultyBroker) GetPosition(portfolio Portfolio, security Security) (float64, error) {
	return b.brokers[portfolio.Client].GetPosition(portfolio, security)
}

func (b *MultyBroker) RegisterOrder(order Order) error {
	return b.brokers[order.Portfolio.Client].RegisterOrder(order)
}

func (b *MultyBroker) Close() error {
	for _, broker := range b.brokers {
		broker.Close()
	}
	//TODO errors.Join
	return nil
}
