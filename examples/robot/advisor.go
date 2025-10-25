package main

import "time"

// TODO
type AdvisorSample struct {
}

func (a *AdvisorSample) Add(d time.Time, price float64) bool {
	return true
}

func (a *AdvisorSample) Value() float64 {
	return 0
}
