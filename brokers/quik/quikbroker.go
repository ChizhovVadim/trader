package quik

import (
	"github.com/ChizhovVadim/trader/brokers"
	"github.com/ChizhovVadim/trader/brokers/quikservice"

	"context"
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"strconv"
	"sync/atomic"
)

var _ brokers.IBroker = (*QuikBroker)(nil)
var _ brokers.IMarketData = (*QuikBroker)(nil)

type QuikBroker struct {
	logger              *slog.Logger
	name                string
	quikService         *quikservice.QuikService
	marketDataCallbacks chan<- any
	transId             int64
}

func NewQuikBroker(
	logger *slog.Logger,
	name string,
	port int,
	marketDataCallbacks chan<- any,
) *QuikBroker {
	return &QuikBroker{
		logger:              logger,
		name:                name,
		quikService:         quikservice.New(nil, port, 1),
		marketDataCallbacks: marketDataCallbacks,
		transId:             calculateStartTransId(),
	}
}

func (b *QuikBroker) handleCallbacks(ctx context.Context, cj quikservice.CallbackJson) {
	if cj.Command == "NewCandle" {
		if cj.Data != nil && b.marketDataCallbacks != nil {
			var newCandle quikservice.Candle
			var err = json.Unmarshal(*cj.Data, &newCandle)
			if err != nil {
				return //err
			}
			// TODO можно фильтровать слишком ранние бары
			select {
			case <-ctx.Done():
				//return ctx.Err()
			case b.marketDataCallbacks <- convertToCandle(newCandle):
			}
		}
		return
	}
}

func (b *QuikBroker) Init(ctx context.Context) error {
	if err := b.quikService.Init(ctx, b.handleCallbacks); err != nil {
		return err
	}
	resp, err := b.quikService.IsConnected()
	if err != nil {
		return err
	}
	res, _ := quikservice.ParseInt(resp.Data)
	if !(res == 1) {
		return errors.New("trader is not connected")
	}
	return nil
}

func (b *QuikBroker) CheckStatus() {
	fmt.Printf("%10s %10s\n", b.name, "quik")
}

func (b *QuikBroker) Close() error {
	return b.quikService.Close()
}

func (b *QuikBroker) GetPortfolioLimits(portfolio brokers.Portfolio) (brokers.PortfolioLimits, error) {
	resp, err := b.quikService.GetPortfolioInfoEx(portfolio.Firm, portfolio.Portfolio, 0)
	if err != nil {
		return brokers.PortfolioLimits{}, err
	}
	var data = quikservice.AsMap(resp.Data)
	if data == nil {
		return brokers.PortfolioLimits{}, errors.New("portfolio not found")
	}
	startLimitOpenPos, ok := quikservice.ParseFloat(data["start_limit_open_pos"])
	if !ok {
		return brokers.PortfolioLimits{}, errors.New("parse start_limit_open_pos")
	}
	usedLimOpenPos, _ := quikservice.ParseFloat(data["used_lim_open_pos"])
	varMargin, _ := quikservice.ParseFloat(data["varmargin"])
	accVarMargin, _ := quikservice.ParseFloat(data["fut_accured_int"])
	return brokers.PortfolioLimits{
		StartLimitOpenPos: startLimitOpenPos,
		UsedLimOpenPos:    usedLimOpenPos,
		VarMargin:         varMargin,
		AccVarMargin:      accVarMargin,
	}, nil
}

func (b *QuikBroker) GetPosition(portfolio brokers.Portfolio, security brokers.Security) (float64, error) {
	if security.ClassCode == brokers.FuturesClassCode {
		resp, err := b.quikService.GetFuturesHolding(portfolio.Firm, portfolio.Portfolio, security.Code, 0)
		if err != nil {
			return 0, err
		}
		var data = quikservice.AsMap(resp.Data)
		if data == nil {
			b.logger.Warn("empty position",
				"client", portfolio.Client,
				"portfolio", portfolio.Portfolio,
				"security", security.Name,
			)
			return 0, nil
		}
		pos, ok := quikservice.ParseFloat(data["totalnet"])
		if !ok {
			return 0, fmt.Errorf("GetFuturesHolding bad response")
		}
		return pos, nil
	} else {
		return 0, fmt.Errorf("not supported classcode %v", security.ClassCode)
	}
}

func (b *QuikBroker) RegisterOrder(order brokers.Order) error {
	var sPrice = formatPrice(order.Security.PriceStep, order.Security.PricePrecision, order.Price)
	b.logger.Info("RegisterOrder",
		"client", order.Portfolio.Client,
		"portfolio", order.Portfolio.Portfolio,
		"security", order.Security.Name,
		"volume", order.Volume,
		"price", sPrice)

	var transId = atomic.AddInt64(&b.transId, 1)
	var strTransId = fmt.Sprintf("%v", transId)
	var trans = quikservice.Transaction{
		TRANS_ID:    strTransId,
		ACTION:      "NEW_ORDER",
		SECCODE:     order.Security.Code,
		CLASSCODE:   order.Security.ClassCode,
		ACCOUNT:     order.Portfolio.Portfolio,
		PRICE:       sPrice,
		CLIENT_CODE: strTransId,
	}
	if order.Volume > 0 {
		trans.OPERATION = "B"
		trans.QUANTITY = strconv.Itoa(order.Volume)
	} else {
		trans.OPERATION = "S"
		trans.QUANTITY = strconv.Itoa(-order.Volume)
	}
	_, err := b.quikService.SendTransaction(trans)
	return err
}

func (b *QuikBroker) GetLastCandles(security brokers.Security, timeframe string) iter.Seq2[brokers.HistoryCandle, error] {
	return func(yield func(brokers.HistoryCandle, error) bool) {
		var candles, err = b.getLastCandles_Impl(security, timeframe)
		if err != nil {
			yield(brokers.HistoryCandle{}, err)
			return
		}
		for _, item := range candles {
			var candle = convertToHistoryCandle(item)
			if !yield(candle, nil) {
				return
			}
		}
	}
}

func (b *QuikBroker) getLastCandles_Impl(security brokers.Security, timeframe string) ([]quikservice.Candle, error) {
	var candleInterval, ok = quikTimeframe(timeframe)
	if !ok {
		return nil, fmt.Errorf("timeframe not supported %v", timeframe)
	}
	const count = 5_000 // Если не указывать размер, то может прийти слишком много баров и unmarshal большой json
	var candles, err = b.quikService.GetLastCandles(security.ClassCode, security.Code, candleInterval, count)
	if err != nil {
		return nil, err
	}
	// последний бар за сегодня может быть не завершен
	if len(candles) > 0 &&
		isToday(candles[len(candles)-1].Datetime.ToTime(brokers.Moscow)) {
		candles = candles[:len(candles)-1]
	}
	return candles, nil
}

func (b *QuikBroker) SubscribeCandles(security brokers.Security, timeframe string) error {
	var candleInterval, ok = quikTimeframe(timeframe)
	if !ok {
		return fmt.Errorf("timeframe not supported %v", timeframe)
	}
	b.logger.Debug("SubscribeCandles",
		"security", security.Code,
		"timeframe", timeframe)
	_, err := b.quikService.SubscribeCandles(security.ClassCode, security.Code, candleInterval)
	return err
}
