package moex

import "time"

const FuturesClassCode = "SPBFUT"

var Moscow = initMoscow()

func initMoscow() *time.Location {
	var loc, err = time.LoadLocation("Europe/Moscow")
	if err != nil {
		loc = time.FixedZone("MSK", int(3*time.Hour/time.Second))
	}
	return loc
}

func IsMainFortsSession(d time.Time) bool {
	return d.Hour() >= 10 && d.Hour() <= 18
}
