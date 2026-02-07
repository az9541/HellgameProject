package main

import (
	"math"
	"math/rand"
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

func awarenessFromInfluence(influence float64) float64 {
	if influence <= 0 {
		return MinAwareness
	}
	a := MinAwareness + (1.0-MinAwareness)*(math.Log(1+InfluenceToAwarenessFactor*influence)/math.Log(1+InfluenceToAwarenessFactor))
	return clamp(a, MinAwareness, 1.0)
}

func estimateForceWithAwareness(force, awareness float64) float64 {
	// Добавляем рандома. Оценка может быть как в большую сторону, так и в меньшую
	// Если фактор шума положителен, то оценка завышается, если отрицателен — занижается
	maxNoise := 0.4
	noise := (rand.Float64()*2 - 1) * maxNoise * (1 - awareness)
	return force * (1 + noise)
}
