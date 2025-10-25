package strategies

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/ChizhovVadim/trader/pkg/brokers"
)

type StrategyService struct {
	logger          *slog.Logger
	broker          brokers.IBroker
	portfolio       *Portfolio
	security        brokers.Security
	signalName      string
	plannedPosition Optional[int]
}

func NewStrategyService(
	logger *slog.Logger,
	broker brokers.IBroker,
	portfolio *Portfolio,
	security brokers.Security,
	signalName string,
) *StrategyService {
	logger = logger.With(
		"client", portfolio.Portfolio.Client,
		"portfolio", portfolio.Portfolio.Portfolio,
		"security", security.Name,
		"signal", signalName)
	return &StrategyService{
		logger:     logger,
		broker:     broker,
		portfolio:  portfolio,
		security:   security,
		signalName: signalName,
	}
}

func (s *StrategyService) getBrokerPos() (float64, error) {
	return s.broker.GetPosition(s.portfolio.Portfolio, s.security)
}

func (s *StrategyService) Init() error {
	brokerPos, err := s.getBrokerPos()
	if err != nil {
		return err
	}
	s.plannedPosition.SetValue(int(brokerPos))
	s.logger.Info("Init strategy",
		"Position", s.plannedPosition.Value)
	return nil
}

func (s *StrategyService) CheckStatus() {
	brokerPos, err := s.getBrokerPos()
	if err != nil {
		return
	}
	var status string
	if s.plannedPosition.HasValue && s.plannedPosition.Value == int(brokerPos) {
		status = "+"
	} else {
		status = "!"
	}
	fmt.Printf("%10v %10v %10v planned: %6v broker: %6v %v\n",
		s.portfolio.Portfolio.Client,
		s.portfolio.Portfolio.Portfolio,
		s.security.Name,
		s.plannedPosition.Value,
		int(brokerPos),
		status)
}

func (s *StrategyService) OnSignal(signal Signal) bool {
	var orderRegistered bool
	var err = s.on_signal_impl(signal, &orderRegistered)
	if err != nil {
		s.logger.Warn("OnSignal failed",
			"error", err)
	}
	return orderRegistered
}

func (s *StrategyService) on_signal_impl(signal Signal, orderRegistered *bool) error {
	// стратегия следит только за своими сигналами
	if !(signal.SecurityCode == s.security.Code &&
		signal.Name == s.signalName) {
		return nil
	}
	// считаем, что сигнал слишком старый
	if signal.Deadline.Before(time.Now()) {
		return nil
	}
	if !s.portfolio.AmountAvailable.HasValue {
		return nil
	}
	if !signal.ContractsPerAmount.HasValue {
		return nil
	}
	if !s.plannedPosition.HasValue {
		return nil
	}
	var idealPos = signal.ContractsPerAmount.Value * s.portfolio.AmountAvailable.Value
	var volume = int(idealPos - float64(s.plannedPosition.Value))
	// изменение позиции не требуется
	if volume == 0 {
		return nil
	}
	brokerPos, err := s.getBrokerPos()
	if err != nil {
		return err
	}
	if s.plannedPosition.Value != int(brokerPos) {
		return fmt.Errorf("check position failed")
	}
	err = s.broker.RegisterOrder(brokers.Order{
		Portfolio: s.portfolio.Portfolio,
		Security:  s.security,
		Volume:    volume,
		Price:     priceWithSlippage(signal.Price, volume),
	})
	if err != nil {
		return err
	}
	s.plannedPosition.Value += volume
	*orderRegistered = true
	return nil
}

func priceWithSlippage(price float64, volume int) float64 {
	const Slippage = 0.001
	if volume > 0 {
		return price * (1 + Slippage)
	} else {
		return price * (1 - Slippage)
	}
}
