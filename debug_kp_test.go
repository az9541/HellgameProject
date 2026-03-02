package main

import (
	"fmt"
	"testing"
)

func TestDebugKP(t *testing.T) {
	// Создаём минимальную фракцию и цепочку доменов
	f := &FactionState{ID: "f_test", Name: "Debuggers", Power: 80, Territory: 2.0}
	n := 9
	domains := make([]*DomainState, n)
	for i := 0; i < n; i++ {
		domains[i] = &DomainState{ID: fmt.Sprintf("d_%d", i), Name: "Domain"}
	}
	// стартовый домен
	domains[4].ControlledBy = f.ID

	// Параметры прогона
	// Построим соседей и параметры
	neighbors := buildNeighborsFromDomains(domains)

	// init u
	u := make([]float64, n)
	for i := 0; i < n; i++ {
		if domains[i].ControlledBy == f.ID {
			u[i] = 1.0
		} else {
			u[i] = 0.0
		}
	}

	D := minFloat(1.0, 0.2+0.8*(f.Power/100.0))
	r := minFloat(0.2, 0.01+0.09*(f.Territory/5.0))
	dt := 1.0

	// estimate substeps
	maxDeg := 0
	for _, nb := range neighbors {
		if len(nb) > maxDeg {
			maxDeg = len(nb)
		}
	}
	substeps := 1
	if D > 0 && maxDeg > 0 {
		substeps = int((dt * D * float64(maxDeg) * 2.0) + 0.999)
		if substeps < 1 {
			substeps = 1
		}
	}

	t.Logf("Debug KP: n=%d D=%.3f r=%.3f dt=%.3f substeps=%d", n, D, r, dt, substeps)

	// Просто логируем финальное распределение (тест считается успешным)
	for i := 0; i < n; i++ {
		t.Logf("final idx=%d density=%.3f", i, u[i])
	}
}
