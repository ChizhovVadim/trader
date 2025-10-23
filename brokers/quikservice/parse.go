package quikservice

import (
	"strconv"
)

func AsMap(a any) map[string]any {
	if a == nil {
		return nil
	}
	if res, ok := a.(map[string]any); ok {
		return res
	}
	return nil
}

func ParseInt(a any) (int, bool) {
	switch v := a.(type) {
	case int:
		return v, true
	case float64:
		return int(v), true
	case string:
		var f, err = strconv.Atoi(v)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}

func ParseFloat(a any) (float64, bool) {
	switch v := a.(type) {
	case float64:
		return v, true
	case string:
		var f, err = strconv.ParseFloat(v, 64)
		if err != nil {
			return 0, false
		}
		return f, true
	default:
		return 0, false
	}
}
