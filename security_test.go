package main

import (
	"testing"
)

func TestEncodeSecurity(t *testing.T) {
	var tests = []struct {
		name string
		code string
	}{
		{
			name: "Si-6.20",
			code: "SiM0",
		},
	}

	for _, test := range tests {
		var code, err = EncodeSecurity(test.name)
		if err != nil {
			t.Error(test, err)
			continue
		}
		if code != test.code {
			t.Error(test, code)
			continue
		}
	}
}
