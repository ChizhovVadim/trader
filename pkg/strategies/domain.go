package strategies

import (
	"time"

	"github.com/ChizhovVadim/trader/pkg/brokers"
)

type Indicator interface {
	Add(d time.Time, value float64) bool
	Value() float64
}

type Advisor func(dateTime time.Time, closePrice float64) (prediction float64, ok bool)

type Optional[T any] struct {
	Value    T
	HasValue bool
}

func (m *Optional[T]) SetValue(value T) {
	m.Value = value
	m.HasValue = true
}

type Portfolio struct {
	Portfolio       brokers.Portfolio
	AmountAvailable Optional[float64]
}

type SizeConfig struct {
	LongLever  float64 `xml:",attr"`
	ShortLever float64 `xml:",attr"`
	MaxLever   float64 `xml:",attr"`
	Weight     float64 `xml:",attr"`
}
