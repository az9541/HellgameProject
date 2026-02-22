package main

import (
	"fmt"
	"log"
	"math"
	"sort"
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

	for _, faction := range sim.State.Factions {
		for _, domain := range sim.State.Domains {
			if domain.Influence == nil {
				domain.Influence = make(map[string]float64)
			}

			// Стартовое влияние: выше в своих доменах, ниже в чужих
			if domain.ControlledBy == faction.ID {
				domain.Influence[faction.ID] = BaseOwnDomainInfluence // 80% в своих
			} else {
				domain.Influence[faction.ID] = 0.0
			}
		}
	}
}

func (sim *WorldSimulator) runKPPSimulation() {
	domainKeys := getSortedDomainKeys(sim.State.Domains)
	domains := getDomainsList(domainKeys, sim.State.Domains)
	if len(domains) == 0 || len(sim.State.Factions) == 0 {
		return
	}

	factionIDs := getSortedFactionKeys(sim.State.Factions)
	u := sim.solveExpansionEquations(factionIDs, domains, 1)

	// Переносим результаты обратно в структуру доменов
	for fIdx, factionID := range factionIDs {
		for dIdx := range domains {
			domains[dIdx].Influence[factionID] = u[fIdx][dIdx]
		}
	}
	// Ограничиваем общее влияние на домен, чтобы не уходило за 100%
	for _, domain := range domains {
		capDomainInfluence(domain.Influence)
	}

	for _, factionID := range factionIDs {
		row := ""
		for i := range domains {
			row += fmt.Sprintf("%.3f", domains[i].Influence[factionID])
			if i < len(domains)-1 {
				row += ", "
			}
		}
		log.Printf("EXPANSION_DENSITIES_COUPLED faction=%q tick=%d densities=[%s]",
			factionID, sim.State.GlobalTick, row)
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
	D := minFloat(1.0, 0.002+0.0015*(faction.Power/100.0))
	r := minFloat(0.1, 0.005+0.065*(faction.Territory/5.0))
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

// TODO!! Насущие проблемы:
// 1 - Проценты влияния на домен могут уходить за 100%. Кейс с войной
// 2 - В случае войны влияние не замораживается, а продолжает резко расти для атакующего, пробивая в итоге 100%
// 3 - В целом после войны происходят странные вещи: скачки влияния, уход за 100%, странные колебания.
// 4 - На unclaimed-доменах появляется влияние фракций, которые не имеют с этим доменом связей.
func (sim *WorldSimulator) solveExpansionEquations(
	factionIDs []string,
	domains []*DomainState,
	ticks int,
) [][]float64 {
	factionsMap := sim.State.Factions
	domainsMap := sim.State.Domains
	if len(factionsMap) == 0 || len(domainsMap) == 0 {
		return nil
	}

	nF := len(factionIDs)
	nD := len(domains)
	if nF == 0 || nD == 0 || ticks <= 0 {
		return nil
	}

	neighbors := buildNeighborsFromDomains(domains)

	u := make([][]float64, nF)
	owned := make([][]bool, nF)
	D := make([]float64, nF)
	r := make([]float64, nF)
	wealth := make([]float64, nF)

	for fIdx, factionID := range factionIDs {
		f := sim.State.Factions[factionID]
		u[fIdx] = make([]float64, nD)
		owned[fIdx] = make([]bool, nD)

		D[fIdx] = minFloat(1.0, 0.002+0.01*(f.Power/100.0))
		r[fIdx] = minFloat(0.1, 0.005+0.095*(f.Territory/5.0))
		wealth[fIdx] = sim.factionWealthIndex(f)

		for dIdx, d := range domains {
			val := d.Influence[factionID]
			if val <= 0 {
				val = MinInfluence
			}
			u[fIdx][dIdx] = clamp(val, 0.0, 1.0)
			owned[fIdx][dIdx] = (d.ControlledBy == factionID)
		}
	}

	maxDeg := 0
	for _, nb := range neighbors {
		if len(nb) > maxDeg {
			maxDeg = len(nb)
		}
	}
	maxD := 0.0
	for _, d := range D {
		if d > maxD {
			maxD = d
		}
	}

	substeps := 1
	if maxD > 0 && maxDeg > 0 {
		substeps = int(math.Ceil(maxD * float64(maxDeg) * 2.0))
		substeps = maxInt(1, minInt(1000, substeps))
	}
	dtSub := 1.0 / float64(substeps)

	for h := 0; h < ticks; h++ {
		for s := 0; s < substeps; s++ {
			warMask := make([]bool, nD)
			for d := range domains {
				warMask[d] = sim.getActiveWarForDomain(domains[d].ID) != nil
			}

			u = applyKPPDiffusionStep(u, neighbors, D, dtSub)
			u = applyLVReactionStep(u, r, dtSub, warMask)
			u = applySourceStep(u, owned, wealth, dtSub)
			u = applySpilloverStep(u, owned, neighbors, dtSub)
			clampMatrixInPlace(u, MinInfluence, 1.0)
		}
	}

	return u
}

// Подпитываем влияние фракции на собственых доменах
func applySourceStep(u [][]float64, owned [][]bool, wealth []float64, dt float64) [][]float64 {
	nF := len(u)
	if nF == 0 {
		return nil
	}
	nD := len(u[0])

	next := make([][]float64, nF)
	for f := 0; f < nF; f++ {
		next[f] = make([]float64, nD)
		copy(next[f], u[f])

		for d := 0; d < nD; d++ {
			if !owned[f][d] {
				continue
			}
			next[f][d] += dt * sourceBaseRate * wealth[f] * maxFloat(0, sourceTargetOwned-next[f][d])
		}
	}
	return next
}

func applySpilloverStep(u [][]float64, owned [][]bool, neighbors [][]int, dt float64) [][]float64 {
	nF := len(u)
	if nF == 0 {
		return nil
	}
	nD := len(u[0])

	next := make([][]float64, nF)
	for f := 0; f < nF; f++ {
		next[f] = make([]float64, nD)
		copy(next[f], u[f])

		for d := 0; d < nD; d++ {
			if !owned[f][d] {
				continue
			}
			overflow := dt * spillRate * maxFloat(0, next[f][d]-spillThreshold)
			if overflow <= 0 {
				continue
			}
			count := 0
			for _, j := range neighbors[d] {
				if !owned[f][j] {
					count++
				}
			}
			if count == 0 {
				continue
			}
			share := overflow / float64(count)
			for _, j := range neighbors[d] {
				if !owned[f][j] {
					next[f][j] += share
				}
			}
		}
	}
	return next
}

func clampMatrixInPlace(u [][]float64, minV, maxV float64) {
	for f := range u {
		for d := range u[f] {
			u[f][d] = clamp(u[f][d], minV, maxV)
		}
	}
}

func getSortedFactionKeys(factions map[string]*FactionState) []string {
	keys := make([]string, 0, len(factions))
	for k := range factions {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func capDomainInfluence(influence map[string]float64) {
	total := 0.0
	for _, v := range influence {
		total += maxFloat(0.0, v)
	}
	if total <= 1.0 {
		return
	}
	scale := 1.0 / total
	for factionID, value := range influence {
		influence[factionID] = maxFloat(0.0, value) * scale
	}
}
