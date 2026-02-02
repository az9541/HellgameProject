package main

import (
	"math"
)

// minFloat возвращает минимальное из двух float64 значений
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// maxFloat возвращает максимальное из двух float64 значений
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// minInt возвращает минимальное из двух int значений
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt возвращает максимальное из двух int значений
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(x, minV, maxV float64) float64 {
	if x < minV {
		return minV
	}
	if x > maxV {
		return maxV
	}
	return x
}

func makeLog(forceRatio float64) float64 {
	forceFactor := math.Log(forceRatio)
	if forceFactor > 1 {
		forceFactor = 1
	}
	if forceFactor < -1 {
		forceFactor = -1
	}
	return forceFactor
}
