package quik

import (
	"math"
	"strconv"
	"time"

	"github.com/ChizhovVadim/trader/brokers"
	"github.com/ChizhovVadim/trader/brokers/quikservice"
)

func calculateStartTransId() int64 {
	var hour, min, sec = time.Now().Clock()
	return 60*(60*int64(hour)+int64(min)) + int64(sec)
}

func formatPrice(priceStep float64, pricePrecision int, price float64) string {
	if priceStep != 0 {
		price = math.Round(price/priceStep) * priceStep
	}
	return strconv.FormatFloat(price, 'f', pricePrecision, 64)
}

func isToday(d time.Time) bool {
	var y1, m1, d1 = d.Date()
	var y2, m2, d2 = time.Now().Date()
	return y1 == y2 && m1 == m2 && d1 == d2
}

func convertToHistoryCandle(item quikservice.Candle) brokers.HistoryCandle {
	return brokers.HistoryCandle{
		DateTime:   item.Datetime.ToTime(brokers.Moscow),
		OpenPrice:  item.Open,
		HighPrice:  item.High,
		LowPrice:   item.Low,
		ClosePrice: item.Close,
		Volume:     item.Volume,
	}
}

func convertToCandle(item quikservice.Candle) brokers.Candle {
	return brokers.Candle{
		Interval:      "TODO",
		SecurityCode:  item.SecCode,
		HistoryCandle: convertToHistoryCandle(item),
	}
}

func quikTimeframe(timeframe string) (int, bool) {
	if timeframe == "minutes5" {
		return quikservice.CandleIntervalM5, true
	}
	return 0, false
}
