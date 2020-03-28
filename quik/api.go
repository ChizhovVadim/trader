package quik

import (
	"context"
	"fmt"
	"sync/atomic"
)

type Nullable struct {
	valid bool
}

func (n *Nullable) Valid() bool {
	return n.valid
}

func (n *Nullable) SetValid(v bool) {
	n.valid = v
}

func (quik *QuikService) IsConnected(ctx context.Context) (bool, error) {
	var resp string
	var err = quik.executeQuery(ctx,
		"isConnected",
		"",
		&resp)
	return resp == "1", err
}

type GetPortfolioInfoExRequest struct {
	FirmId     string
	ClientCode string
	LimitKind  int
}

type GetPortfolioInfoExResponse struct {
	Nullable
	StartLimitOpenPos string `json:"start_limit_open_pos"`
}

func (quik *QuikService) GetPortfolioInfoEx(ctx context.Context,
	req GetPortfolioInfoExRequest) (GetPortfolioInfoExResponse, error) {
	var resp GetPortfolioInfoExResponse
	var err = quik.executeQuery(ctx,
		"getPortfolioInfoEx",
		fmt.Sprintf("%v|%v|%v", req.FirmId, req.ClientCode, req.LimitKind),
		&resp)
	return resp, err
}

type GetFuturesHoldingRequest struct {
	FirmId  string
	AccId   string
	SecCode string
	PosType int
}

type GetFuturesHoldingResponse struct {
	Nullable
	TotalNet float64 `json:"totalnet"`
}

func (quik *QuikService) GetFuturesHolding(ctx context.Context,
	req GetFuturesHoldingRequest) (GetFuturesHoldingResponse, error) {
	var resp GetFuturesHoldingResponse
	var err = quik.executeQuery(ctx,
		"getFuturesHolding",
		fmt.Sprintf("%v|%v|%v|%v", req.FirmId, req.AccId, req.SecCode, req.PosType),
		&resp)
	return resp, err
}

type Transaction struct {
	TRANS_ID    string
	ACTION      string
	ACCOUNT     string
	CLASSCODE   string
	SECCODE     string
	QUANTITY    string
	OPERATION   string
	PRICE       string
	CLIENT_CODE string
}

func (quik *QuikService) SendTransaction(ctx context.Context,
	req Transaction) error {
	var transId = atomic.AddInt64(&quik.transId, 1)
	req.TRANS_ID = fmt.Sprintf("%v", transId)
	req.CLIENT_CODE = req.TRANS_ID
	var resp bool
	return quik.executeQuery(ctx,
		"sendTransaction",
		req,
		&resp)
}

type CandleInterval int

const (
	CandleIntervalM5 = 5
)

type Candle struct {
	Low       float64        `json:"low"`
	Close     float64        `json:"close"`
	High      float64        `json:"high"`
	Open      float64        `json:"open"`
	Volume    int            `json:"volume"`
	Datetime  QuikDateTime   `json:"datetime"`
	SecCode   string         `json:"sec"`
	ClassCode string         `json:"class"`
	Interval  CandleInterval `json:"interval"`
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

func (quik *QuikService) GetLastCandles(ctx context.Context,
	classCode, securityCode string, interval CandleInterval, count int) ([]Candle, error) {
	var resp []Candle
	var err = quik.executeQuery(ctx,
		"get_candles_from_data_source",
		fmt.Sprintf("%v|%v|%v|%v", classCode, securityCode, interval, count),
		&resp)
	return resp, err
}

func (quik *QuikService) SubscribeCandles(ctx context.Context,
	classCode, securityCode string, interval CandleInterval) error {
	var resp string
	var err = quik.executeQuery(ctx,
		"subscribe_to_candles",
		fmt.Sprintf("%v|%v|%v", classCode, securityCode, interval),
		&resp)
	return err
}

type TradeEventData struct {
	TradeNum int64        `json:"trade_num"`
	Account  string       `json:"account"`
	Price    float64      `json:"price"`
	Quantity int          `json:"qty"`
	SecCode  string       `json:"sec_code"`
	DateTime QuikDateTime `json:"datetime"`
	TransID  int64        `json:"trans_id"`
}

type NewCandleEventData Candle

type ConnectedEventData struct{}

type DisconnectedEventData struct{}
