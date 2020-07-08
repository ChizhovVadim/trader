package main

import (
	"context"
	"errors"
	"log"
	"math"
	"strconv"
	"time"

	"github.com/ChizhovVadim/trader/quik"
	"golang.org/x/sync/errgroup"
)

const ClassCode = "SPBFUT"

type Candle struct {
	SecurityCode string
	DateTime     time.Time
	ClosePrice   float64
	Volume       float64
}

type Advice struct {
	SecurityCode string
	DateTime     time.Time
	Price        float64
	Position     float64
}

type IAdvisorService interface {
	GetSecurities() ([]string, error)
	GetAdvices(ctx context.Context, security string, advices chan<- Advice) error
	PublishCandles(ctx context.Context, candles <-chan Candle) error
}

type Strategy struct {
	logger         *log.Logger
	quikService    *quik.QuikService
	client         Client
	advisorService IAdvisorService
}

func (s *Strategy) Run(ctx context.Context) error {
	s.logger.Println("Strategy starting...")

	s.logger.Println("Check connection")
	connected, err := s.quikService.IsConnected(ctx)
	if err != nil {
		return err
	}
	if !connected {
		return errors.New("quik is not connected")
	}

	s.logger.Println("Init portfolio...")
	amount, err := s.getAmount(ctx)
	if err != nil {
		return err
	}
	availableAmount := s.getAvailableAmount(amount)
	s.logger.Println("Init portfolio",
		"Amount", amount,
		"AvailableAmount", availableAmount)
	if availableAmount == 0 {
		return errors.New("availableAmount zero")
	}

	securities, err := s.advisorService.GetSecurities()
	if err != nil {
		return err
	}
	s.logger.Println("Init securities",
		"Securities", securities)

	g, ctx := errgroup.WithContext(ctx)

	if s.client.PublishCandles {
		s.logger.Println("PublishCandles")

		var candles = make(chan Candle, 128)

		g.Go(func() error {
			return s.advisorService.PublishCandles(ctx, candles)
		})

		for _, security := range securities {
			var security = security
			g.Go(func() error {
				return s.getCandles(ctx, security, candles)
			})
		}
	}

	for _, security := range securities {
		var security = security
		var advices = make(chan Advice)

		g.Go(func() error {
			return s.executeAdvices(ctx, s.client.Portfolio, security, availableAmount, advices)
		})

		g.Go(func() error {
			return s.advisorService.GetAdvices(ctx, security, advices)
		})
	}

	s.logger.Println("Strategy started.")
	defer s.logger.Println("Strategy stopped.")

	return g.Wait()
}

func (s *Strategy) getAmount(ctx context.Context) (float64, error) {
	resp, err := s.quikService.GetPortfolioInfoEx(ctx, quik.GetPortfolioInfoExRequest{
		FirmId:     s.client.Firm,
		ClientCode: s.client.Portfolio,
	})
	if err != nil {
		return 0, err
	}
	if !resp.Valid() {
		return 0, errors.New("portfolio not found")
	}
	amount, err := strconv.ParseFloat(resp.StartLimitOpenPos, 64)
	if err != nil {
		return 0, err
	}
	return amount, nil
}

func (s *Strategy) getAvailableAmount(quikAmount float64) float64 {
	var amount float64
	if s.client.Amount > 0 {
		amount = s.client.Amount
	} else {
		amount = quikAmount
	}
	if s.client.MaxAmount > 0 {
		amount = math.Min(amount, s.client.MaxAmount)
	}
	if 0 < s.client.Weight && s.client.Weight < 1 {
		amount *= s.client.Weight
	}
	return amount
}

func (s *Strategy) executeAdvices(ctx context.Context,
	portfolio, security string, amount float64, advices <-chan Advice) error {
	security, err := EncodeSecurity(security)
	if err != nil {
		return err
	}

	pos, err := s.quikService.GetFuturesHolding(ctx, quik.GetFuturesHoldingRequest{
		FirmId:  s.client.Firm,
		AccId:   s.client.Portfolio,
		SecCode: security,
	})
	if err != nil {
		return err
	}
	var strategyPosition = int(pos.TotalNet)
	s.logger.Println("Init position",
		"Portfolio", s.client.Portfolio,
		"Security", security,
		"Position", strategyPosition)

	var basePrice float64

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case advice := <-advices:
			if time.Since(advice.DateTime) >= 9*time.Minute {
				continue
			}
			if basePrice == 0 {
				basePrice = advice.Price
				s.logger.Println("Init base price",
					"Advice", advice)
			}
			var position = amount / basePrice * advice.Position
			var volume = int(position - float64(strategyPosition)) //TODO размер лота
			if volume == 0 {
				continue
			}
			s.logger.Println("New advice",
				"Advice", advice)
			if !s.checkPosition(ctx, security, strategyPosition) {
				continue
			}
			err = s.registerOrder(ctx, portfolio, security, volume, advice.Price)
			if err != nil {
				s.logger.Println(err)
				continue
			}
			strategyPosition += volume
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(30 * time.Second):
				s.checkPosition(ctx, security, strategyPosition)
			}
		}
	}
}

func (s *Strategy) checkPosition(ctx context.Context, security string, strategyPosition int) bool {
	pos, err := s.quikService.GetFuturesHolding(ctx, quik.GetFuturesHoldingRequest{
		FirmId:  s.client.Firm,
		AccId:   s.client.Portfolio,
		SecCode: security,
	})
	if err != nil {
		s.logger.Println(err)
		return false
	}
	var traderPosition = int(pos.TotalNet)
	var ok = strategyPosition == traderPosition
	var status string
	if ok {
		status = "+"
	} else {
		status = "!"
	}
	s.logger.Println("Check position",
		"Portfolio", s.client.Portfolio,
		"Security", security,
		"StrategyPosition", strategyPosition,
		"TraderPosition", traderPosition,
		"Status", status)
	return ok
}

func (s *Strategy) registerOrder(ctx context.Context,
	portfolio, security string, volume int, price float64) error {
	const Slippage = 0.001
	if volume > 0 {
		price = price * (1 + Slippage)
	} else {
		price = price * (1 - Slippage)
	}
	price = math.Round(price) //TODO шаг цены, планка
	s.logger.Println("Register order",
		"Portfolio", portfolio,
		"Security", security,
		"Price", price,
		"Volume", volume)
	var trans = quik.Transaction{
		ACTION:    "NEW_ORDER",
		SECCODE:   security,
		CLASSCODE: ClassCode,
		ACCOUNT:   portfolio,
		PRICE:     strconv.Itoa(int(price)),
	}
	if volume > 0 {
		trans.OPERATION = "B"
		trans.QUANTITY = strconv.Itoa(volume)
	} else {
		trans.OPERATION = "S"
		trans.QUANTITY = strconv.Itoa(-volume)
	}
	return s.quikService.SendTransaction(ctx, trans)
}

func (s *Strategy) getCandles(ctx context.Context, securityName string, candles chan<- Candle) error {
	const interval = quik.CandleIntervalM5

	securityCode, err := EncodeSecurity(securityName)
	if err != nil {
		return err
	}

	lastQuikCandles, err := s.quikService.GetLastCandles(ctx,
		ClassCode, securityCode, interval, 0)
	if err != nil {
		return err
	}

	var lastCandles = make([]Candle, len(lastQuikCandles))
	for i := range lastQuikCandles {
		lastCandles[i] = convertQuikCandle(securityName, lastQuikCandles[i], Moscow)
	}

	// последний бар за сегодня может быть не завершен
	if len(lastCandles) > 0 && isToday(lastCandles[len(lastCandles)-1].DateTime) {
		lastCandles = lastCandles[:len(lastCandles)-1]
	}

	if len(lastCandles) == 0 {
		s.logger.Println("Ready candles empty")
	} else {
		s.logger.Println("Ready candles",
			"First", lastCandles[0],
			"Last", lastCandles[len(lastCandles)-1])
	}

	for _, candle := range lastCandles {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case candles <- candle:
		}
	}

	var unsubscribe = s.quikService.Events().Subscribe(quik.HandlerFunc(func(msg interface{}) {
		if candle, ok := msg.(*quik.NewCandleEventData); ok {
			if candle.SecCode == securityCode && candle.Interval == interval {
				select {
				case <-ctx.Done():
				case candles <- convertQuikCandle(securityName, quik.Candle(*candle), Moscow):
				}
			}
		}
	}))
	defer unsubscribe()

	err = s.quikService.SubscribeCandles(ctx, ClassCode, securityCode, interval)
	if err != nil {
		return err
	}

	<-ctx.Done()
	return ctx.Err()
}

func convertQuikCandle(security string, candle quik.Candle, loc *time.Location) Candle {
	return Candle{
		SecurityCode: security,
		DateTime:     convertQuikDateTime(candle.Datetime, loc),
		ClosePrice:   candle.Close,
		Volume:       candle.Volume,
	}
}

func convertQuikDateTime(t quik.QuikDateTime, loc *time.Location) time.Time {
	//TODO ms
	return time.Date(t.Year, time.Month(t.Month), t.Day, t.Hour, t.Min, t.Sec, 0, loc)
}

var Moscow = initMoscow()

func initMoscow() *time.Location {
	var loc, err = time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.FixedZone("MSK", int(3*time.Hour/time.Second))
	}
	return loc
}

func isToday(d time.Time) bool {
	var y1, m1, d1 = d.Date()
	var y2, m2, d2 = time.Now().Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}
