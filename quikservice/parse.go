package quikservice

import (
	"strconv"
	"time"
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

func ParseTime(a any, loc *time.Location) time.Time {
	var t = AsMap(a)
	if t == nil {
		return time.Time{}
	}
	var year = t["year"].(int)
	var month = t["month"].(int)
	var day = t["day"].(int)
	var hour = t["hour"].(int)
	var min = t["min"].(int)
	var sec = t["sec"].(int)
	return time.Date(year, time.Month(month), day, hour, min, sec, 0, loc)
}
