package strategies

import (
	"fmt"
	"log/slog"

	"github.com/ChizhovVadim/trader/pkg/brokers"
)

type PortfolioService struct {
	logger    *slog.Logger
	broker    brokers.IBroker
	portfolio *Portfolio
	maxAmount float64
	weight    float64
}

func NewPortfolioService(
	logger *slog.Logger,
	broker brokers.IBroker,
	portfolio *Portfolio,
	maxAmount float64,
	weight float64,
) *PortfolioService {
	logger = logger.With(
		"client", portfolio.Portfolio.Client,
		"portfolio", portfolio.Portfolio.Portfolio)
	return &PortfolioService{
		logger:    logger,
		broker:    broker,
		portfolio: portfolio,
		maxAmount: maxAmount,
		weight:    weight,
	}
}

func (s *PortfolioService) Init() error {
	var limits, err = s.broker.GetPortfolioLimits(s.portfolio.Portfolio)
	if err != nil {
		return err
	}
	var availableAmount = limits.StartLimitOpenPos
	if s.weight != 0 {
		availableAmount *= s.weight
	}
	if s.maxAmount != 0 {
		availableAmount = min(availableAmount, s.maxAmount)
	}
	s.logger.Info("Init portfolio",
		"amount", limits.StartLimitOpenPos,
		"availableAmount", availableAmount)
	s.portfolio.AmountAvailable.SetValue(availableAmount)
	return nil
}

func (s *PortfolioService) CheckStatus() {
	var limits, err = s.broker.GetPortfolioLimits(s.portfolio.Portfolio)
	if err != nil {
		fmt.Println(err)
		return
	}
	var varMargin = limits.AccVarMargin + limits.VarMargin
	var varMarginRatio = varMargin / limits.StartLimitOpenPos
	var usedRatio = limits.UsedLimOpenPos / limits.StartLimitOpenPos

	fmt.Printf("%10v %10v start: %10.0f available: %10.0f varmargin: %10.0f varmargin: %.1f used: %.1f\n",
		//fmt.Fprintf(w, "%v\t%v\t%v\t%v\t%.0f\t%.1f\t%.1f\t\n",
		s.portfolio.Portfolio.Client,
		s.portfolio.Portfolio.Portfolio,
		limits.StartLimitOpenPos,
		s.portfolio.AmountAvailable.Value,
		varMargin,
		varMarginRatio*100,
		usedRatio*100,
	)
}
