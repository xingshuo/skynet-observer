package main

import (
	"strconv"
	"strings"
)

func ParseInt(s, sep string, idx int) int64 {
	suffix := strings.Split(s, sep)[idx]
	val, _ := strconv.ParseInt(suffix, 10, 64)
	return val
}

func ParseFloat(s, sep string, idx int) float64 {
	suffix := strings.Split(s, sep)[idx]
	val, _ := strconv.ParseFloat(suffix, 64)
	return val
}
