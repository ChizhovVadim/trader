package quikservice

import "fmt"

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
