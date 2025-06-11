package main

import (
	"encoding/json"
)

func safeDivide(a interface{}, b float64) float64 {
	if b == 0 {
		return 0
	}

	var af float64
	switch v := a.(type) {
	case float64:
		af = v
	case int:
		af = float64(v)
	case int64:
		af = float64(v)
	case json.Number:
		f, _ := v.Float64()
		af = f
	default:
		return 0
	}

	return af / b
}
