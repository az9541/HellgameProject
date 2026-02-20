package main

import (
	"fmt"
	"log"
	"math"
)

const (
	sourceTargetOwned      = 0.9
	sourceBaseRate         = 0.03
	spillThreshold         = 0.75
	spillRate              = 0.15
	normalizationSmoothing = 0.3
)

// initializeFactionInfluence инициализирует влияние фракций на домены
func (sim *WorldSimulator) initializeFactionInfluence() {
	// Каждая фракция имеет минимальное влияние везде
	baseInfluence := 0.05 // 5% везде по умолчанию

	for _, faction := range sim.State.Factions {
		for _, domain := range sim.State.Domains {
			if domain.Influence == nil {
				domain.Influence = make(map[string]float64)
			}

			// Стартовое влияние: выше в своих доменах, ниже в чужих
			if domain.ControlledBy == faction.ID {
				domain.Influence[faction.ID] = 0.8 // 80% в своих
			} else {
				domain.Influence[faction.ID] = baseInfluence // 5% везде
			}
		}
	}
}

// runKPPSimulation выполняет один шаг KPP (одно обновление влияния фракций)
func (sim *WorldSimulator) runKPPSimulation() {
	// Пересчитываем физику для каждой фракции один раз
	keys := getSortedDomainKeys(sim.State.Domains)
	domainsSlice := getDomainsList(keys, sim.State.Domains)
	if len(domainsSlice) == 0 || len(sim.State.Factions) == 0 {
		return
	}

	newInfluence := make(map[string][]float64, len(sim.State.Factions))
	for _, faction := range sim.State.Factions {
		newInfluence[faction.ID] = sim.SimulateFactionExpansion(faction, domainsSlice, 1)
	}

	// Нормализуем влияние так, чтобы сумма по домену была 1.0
	for i, domain := range domainsSlice {
		sum := 0.0
		for _, densities := range newInfluence {
			if i < len(densities) {
				sum += densities[i]
			}
		}
		if sum <= 0 {
			equal := 1.0 / float64(len(newInfluence))
			for factionID := range newInfluence {
				domain.Influence[factionID] = equal
			}
			continue
		}
		for factionID, densities := range newInfluence {
			if i < len(densities) {
				newNorm := densities[i] / sum
				old := domain.Influence[factionID]
				domain.Influence[factionID] = (1-normalizationSmoothing)*old + normalizationSmoothing*newNorm
			}
		}
	}

	for factionID := range newInfluence {
		row := ""
		for i := 0; i < len(domainsSlice); i++ {
			row += fmt.Sprintf("%.3f", domainsSlice[i].Influence[factionID])
			if i < len(domainsSlice)-1 {
				row += ", "
			}
		}
		log.Printf("EXPANSION_DENSITIES_NORMALIZED faction=%q tick=%d densities=[%s]", factionID, sim.State.GlobalTick, row)
	}
}

// SimulateFactionExpansion симулирует один шаг распространения влияния фракции по доменам.
// Общая схема:
// 1) Берём текущее влияние фракции по доменам (u).
// 2) Делаем шаг KPP на графе доменов (диффузия + реакция).
// 3) Подпитываем собственные домены генератором влияния.
// 4) Переносим избыток (spillover) из своих доменов в соседние чужие.
// 5) Клампим значения и возвращаем массив u.
func (sim *WorldSimulator) SimulateFactionExpansion(faction *FactionState, domains []*DomainState, ticks int) []float64 {
	n := len(domains)
	if n == 0 || ticks <= 0 {
		return nil
	}

	neighbors := buildNeighborsFromDomains(domains)
	// Карта владения доменами для текущей фракции (ключ — указатель на домен)
	ownedMask := make(map[*DomainState]bool)
	for _, i := range domains {
		if i.ControlledBy == faction.ID {
			ownedMask[i] = true
		}
	}
	// Быстрая проверка: есть ли вообще свои домены
	hasOwned := false
	for _, d := range domains {
		if ownedMask[d] {
			hasOwned = true
			break
		}
	}

	// Инициализируем слайс для плотностей влияния по всем доменам
	u := make([]float64, n)
	for i := 0; i < n; i++ {
		u[i] = domains[i].Influence[faction.ID]
		if u[i] <= 0 {
			u[i] = MinInfluence
		}
		u[i] = clamp(u[i], MinInfluence, 1.0)
	}

	// Коэффициенты модели (D — диффузия, r — локальный рост)
	D := minFloat(1.0, 0.002+0.01*(faction.Power/100.0))
	r := minFloat(0.1, 0.005+0.045*(faction.Territory/5.0))
	dt := 1.0 // одна временная единица на шаг

	// Оценка стабильности явной схемы — число субшагов внутри часа
	maxDeg := 0
	for _, nb := range neighbors {
		if len(nb) > maxDeg {
			maxDeg = len(nb)
		}
	}
	substeps := 1
	if D > 0 && maxDeg > 0 {
		substeps = int(math.Ceil(dt * D * float64(maxDeg) * 2.0))
		substeps = maxInt(1, minInt(1000, substeps))
	}

	// Интегрируем ticks шагов
	for h := 0; h < ticks; h++ {
		// 1) Диффузия + реакция (KPP на графе)
		u = SolveKPGraph(u, neighbors, D, r, dt, substeps)
		// 2) Генератор влияния на собственных доменах
		if hasOwned {
			wealth := sim.factionWealthIndex(faction)
			for i, d := range domains {
				if ownedMask[d] {
					u[i] += dt * sourceBaseRate * wealth * maxFloat(0, sourceTargetOwned-u[i])
				}
			}
		}
		// 3) Spillover: избыток из своих доменов в соседние чужие
		for i, d := range domains {
			if !ownedMask[d] { // Чужой домен не может быть источником избыточного влияния
				continue
			}
			// Коэффициент перетока начинает работать только после достижения порога spillThreshold
			// Это значит, что если влияние u на домене i меньше порогового значения,
			// то избыток не будет переноситься вообще
			overflow := spillRate * maxFloat(0, u[i]-spillThreshold)
			if overflow <= 0 {
				continue
			}
			count := 0
			// Считаем количество соседних доменов, которые не принадлежат фракции (кандидаты для spillover)
			for _, j := range neighbors[i] {
				if !ownedMask[domains[j]] {
					count++
				}
			}
			// Нет соседей для перетока, избыток пропадает
			// В будущем можно подумать над тем, что с ним делать
			if count == 0 {
				continue
			}
			// Делим избыток поровну между соседними чужими доменами
			// В будущем можно добавить более сложную логику распределения, например,
			// с учётом текущего влияния на соседях, стабильности и других факторов
			share := overflow / float64(count)
			for _, j := range neighbors[i] {
				if !ownedMask[domains[j]] {
					u[j] += share
				}
			}
		}
		// 4) Границы значений
		for i := 0; i < n; i++ {
			u[i] = clamp(u[i], MinInfluence, 1.0)
		}
	}
	return u
}
