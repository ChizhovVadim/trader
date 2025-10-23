package quikservice

import (
	"encoding/json"
	"fmt"
)

func (quik *QuikService) IsConnected() (ResponseJson, error) {
	return quik.MakeQuery("isConnected", "")
}

func (quik *QuikService) MessageInfo(msg string) (ResponseJson, error) {
	return quik.MakeQuery("message", msg)
}

func (quik *QuikService) GetPortfolioInfoEx(
	firmId string,
	clientCode string,
	limitKind int,
) (ResponseJson, error) {
	return quik.MakeQuery("getPortfolioInfoEx",
		fmt.Sprintf("%v|%v|%v", firmId, clientCode, limitKind))
}

func (quik *QuikService) GetFuturesHolding(
	firmId string,
	accId string,
	secCode string,
	posType int,
) (ResponseJson, error) {
	return quik.MakeQuery("getFuturesHolding",
		fmt.Sprintf("%v|%v|%v|%v", firmId, accId, secCode, posType))
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

func (quik *QuikService) SendTransaction(req Transaction) (ResponseJson, error) {
	//Все значения должны передаваться в виде строк
	return quik.MakeQuery("sendTransaction", req)
}

const (
	CandleIntervalM5 int = 5
)

func (quik *QuikService) GetLastCandles(
	classCode string,
	securityCode string,
	interval int,
	count int,
) ([]Candle, error) {
	var incoming, err = quik.ExecuteQuery(
		"get_candles_from_data_source",
		fmt.Sprintf("%v|%v|%v|%v", classCode, securityCode, interval, count))

	var response TResponseJson[[]Candle]
	err = json.Unmarshal([]byte(incoming), &response)
	if err != nil {
		return nil, err
	}
	if response.LuaError != "" {
		return nil, fmt.Errorf("lua error: %v", response.LuaError)
	}
	return response.Data, nil
}

func (quik *QuikService) SubscribeCandles(
	classCode string,
	securityCode string,
	interval int,
) (ResponseJson, error) {
	return quik.MakeQuery(
		"subscribe_to_candles",
		fmt.Sprintf("%v|%v|%v", classCode, securityCode, interval))
}
