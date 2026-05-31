package util

import "math"

func Clamp(v, min, max float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Min(math.Max(v, min), max)
}

func RoundVal(v float64, places int) float64 {
	pow := math.Pow10(places)
	return math.Round(v*pow) / pow
}

func IsAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
