package brokers

import (
	"context"
	"iter"
	"time"
)

type HistoryCandle struct {
	DateTime   time.Time
	OpenPrice  float64
	HighPrice  float64
	LowPrice   float64
	ClosePrice float64
	Volume     float64
}

type Candle struct {
	Interval     string
	SecurityCode string // or Security
	HistoryCandle
}

type Portfolio struct {
	// MultyBroker использует это поле для маршрутизации клиентов
	Client    string
	Firm      string
	Portfolio string
}

type Security struct {
	// Название инструмента
	Name string
	// Код инструмента
	Code string
	// Код класса
	ClassCode string
	// точность (кол-во знаков после запятой). Если шаг цены может быть не круглым (0.05), то этого будет недостаточно.
	PricePrecision int
	// шаг цены
	PriceStep float64
	// Стоимость шага цены
	PriceStepCost float64
	// Плечо. Для фьючерсов = PriceStepCost/PriceStep.
	Lever float64
}

type Order struct {
	Portfolio Portfolio
	Security  Security
	Volume    int
	Price     float64 //or string?
}

type PortfolioLimits struct {
	// Лимит открытых позиций на начало дня
	StartLimitOpenPos float64
	// Текущие чистые позиции
	UsedLimOpenPos float64
	// Вариац. маржа
	VarMargin float64
	// Накопленная вариационная маржа
	AccVarMargin float64
}

type IMarketData interface {
	GetLastCandles(security Security, timeframe string) iter.Seq2[HistoryCandle, error]
	SubscribeCandles(security Security, timeframe string) error
	//LastPrice(security Security) (float64, error)
}

type IBroker interface {
	Init(context.Context) error
	CheckStatus()
	Close() error
	GetPortfolioLimits(portfolio Portfolio) (PortfolioLimits, error)
	GetPosition(portfolio Portfolio, security Security) (float64, error)
	RegisterOrder(order Order) error
}
