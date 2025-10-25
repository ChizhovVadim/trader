package strategies

import (
	"fmt"
	"iter"
	"log/slog"
	"time"

	"github.com/ChizhovVadim/trader/pkg/brokers"
)

type Signal struct {
	Name               string
	DateTime           time.Time
	SecurityCode       string
	Price              float64
	Prediction         float64
	ContractsPerAmount Optional[float64]
	Deadline           time.Time
}

type SignalService struct {
	logger         *slog.Logger
	name           string
	marketData     brokers.IMarketData
	security       brokers.Security
	candleInterval string
	ind            Indicator
	sizeConfig     SizeConfig
	start          time.Time
	baseCandle     brokers.Candle
	lastSignal     Signal
}

func NewSignalService(
	logger *slog.Logger,
	name string,
	marketData brokers.IMarketData,
	security brokers.Security,
	candleInterval string,
	ind Indicator,
	sizeConfig SizeConfig,
) *SignalService {
	logger = logger.With(
		"name", name,
		"security", security.Name)
	return &SignalService{
		logger:         logger,
		name:           name,
		marketData:     marketData,
		security:       security,
		candleInterval: candleInterval,
		ind:            ind,
		sizeConfig:     sizeConfig,
		start:          time.Now().Add(-10 * time.Minute),
	}
}

func (s *SignalService) Init() error {
	if err := s.AddHistoryCandles(s.marketData.GetLastCandles(s.security, s.candleInterval)); err != nil {
		return err
	}
	// тк можем подписаться на несколько инструментов,
	// то подписываемся в отдельной горутине,
	// чтобы сразу начать читать бары из первой подписки и не заблокироваться.
	go func() {
		var err = s.marketData.SubscribeCandles(s.security, s.candleInterval)
		if err != nil {
			s.logger.Error("marketData.SubscribeCandles", "error", err)
			return
		}
	}()
	return nil
}

func (s *SignalService) CheckStatus() {
	fmt.Printf("%10v %10v %16v %8v %.4f\n",
		s.name,
		s.security.Name,
		s.lastSignal.DateTime.Format("2006-01-02 15:04"),
		s.lastSignal.Price,
		s.lastSignal.Prediction,
	)
}

func (s *SignalService) OnCandle(candle brokers.Candle) Signal {
	// советник следит только за своими барами
	if !( /*TODO s.candleInterval == candle.Interval &&*/
	s.security.Code == candle.SecurityCode) {
		return Signal{}
	}
	if !s.ind.Add(candle.DateTime, candle.ClosePrice) {
		return Signal{}
	}
	var freshCandle = candle.DateTime.After(s.start) // TODO and main forts session?
	if s.baseCandle.DateTime.IsZero() && freshCandle {
		s.baseCandle = candle
		s.logger.Debug("Init base price",
			"DateTime", s.baseCandle.DateTime,
			"Price", s.baseCandle.ClosePrice)
	}
	s.lastSignal = s.makeSignal(candle.DateTime, candle.ClosePrice)
	if freshCandle {
		s.logger.Debug("New signal",
			"Signal", s.lastSignal)
	}
	return s.lastSignal
}

func (s *SignalService) makeSignal(dt time.Time, price float64) Signal {
	var signal = Signal{
		Name:         s.name,
		SecurityCode: s.security.Code,
		DateTime:     dt,
		Price:        price,
		Prediction:   s.ind.Value(),
	}
	if !s.baseCandle.DateTime.IsZero() {
		var position = applySize(signal.Prediction, s.sizeConfig)
		signal.ContractsPerAmount.SetValue(position / (s.baseCandle.ClosePrice * s.security.Lever))
		signal.Deadline = dt.Add(9 * time.Minute) // от открытия бара или 4 минуты от закрытия.
	}
	return signal
}

func (s *SignalService) AddHistoryCandles(historyCandles iter.Seq2[brokers.HistoryCandle, error]) error {
	var (
		firstCandle brokers.HistoryCandle
		lastCandle  brokers.HistoryCandle
		size        int
	)
	for candle, err := range historyCandles {
		if err != nil {
			return err
		}
		if !s.ind.Add(candle.DateTime, candle.ClosePrice) {
			continue
		}
		if size == 0 {
			firstCandle = candle
		}
		size += 1
		lastCandle = candle
	}
	if size == 0 {
		s.logger.Warn("History candles empty")
	} else {
		s.logger.Debug("History candles",
			"First", firstCandle,
			"Last", lastCandle,
			"Size", size)
		s.lastSignal = s.makeSignal(lastCandle.DateTime, lastCandle.ClosePrice)
		s.logger.Info("Init signal",
			"DateTime", s.lastSignal.DateTime,
			"Price", s.lastSignal.Price,
			"Prediction", s.lastSignal.Prediction,
		)
	}
	return nil
}

func applySize(pos float64, config SizeConfig) float64 {
	if pos > 0 {
		pos *= config.LongLever
	} else {
		pos *= config.ShortLever
	}
	pos = config.Weight * max(-config.MaxLever, min(config.MaxLever, pos))
	return pos
}
