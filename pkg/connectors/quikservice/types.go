package quikservice

import (
	"encoding/json"
	"time"
)

type RequestJson struct {
	Id          int64  `json:"id"`
	Command     string `json:"cmd"`
	CreatedTime int64  `json:"t"`
	Data        any    `json:"data"`
}

type TResponseJson[T any] struct {
	Id          int64   `json:"id"`
	Command     string  `json:"cmd"`
	CreatedTime float64 `json:"t"`
	Data        T       `json:"data"`
	LuaError    string  `json:"lua_error"`
}

type ResponseJson = TResponseJson[any]

type CallbackJson struct {
	Command     string           `json:"cmd"`
	CreatedTime float64          `json:"t"`
	Data        *json.RawMessage `json:"data"`
	LuaError    string           `json:"lua_error"`
}

type Candle struct {
	Low       float64      `json:"low"`
	Close     float64      `json:"close"`
	High      float64      `json:"high"`
	Open      float64      `json:"open"`
	Volume    float64      `json:"volume"`
	Datetime  QuikDateTime `json:"datetime"`
	SecCode   string       `json:"sec"`
	ClassCode string       `json:"class"`
	Interval  int          `json:"interval"`
}

type QuikDateTime struct {
	Ms    int `json:"ms"`
	Sec   int `json:"sec"`
	Min   int `json:"min"`
	Hour  int `json:"hour"`
	Day   int `json:"day"`
	Month int `json:"month"`
	Year  int `json:"year"`
}

func (t *QuikDateTime) ToTime(loc *time.Location) time.Time {
	return time.Date(t.Year, time.Month(t.Month), t.Day, t.Hour, t.Min, t.Sec, 0, loc)
}
