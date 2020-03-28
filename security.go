package main

import (
	"fmt"
	"strconv"
	"strings"
)

// Sample: "Si-3.17" -> "SiH7"
// http://moex.com/s205
func EncodeSecurity(securityName string) (string, error) {
	const MonthCodes = "FGHJKMNQUVXZ"
	var parts = strings.SplitN(securityName, "-", 2)
	var name = parts[0]
	parts = strings.SplitN(parts[1], ".", 2)
	month, err := strconv.Atoi(parts[0])
	if err != nil {
		return "", err
	}
	year, err := strconv.Atoi(parts[1])
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%v%v%v", name, string(MonthCodes[month-1]), year%10), nil
}
