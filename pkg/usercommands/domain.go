package usercommands

type ExitUserCmd struct{}
type CheckStatusUserCmd struct{}

type InitLimitsUserCmd struct {
	Client string
}

// Принудительная ребалансировка.
// Например, при вводе/выводе средств.
type RebalanceUserCmd struct {
	Client string
}

// Закрыть все позиции.
// Например, перед экспирацией, длинными выходными/праздниками.
type CloseAllUserCmd struct {
	Client string
}
