package main

import (
	"fmt"
	"log"
	"math"
	"sort"
)

// initializeFactionInfluence инициализирует влияние фракций на домены
func (sim *WorldSimulator) initializeFactionInfluence() {
	// Каждая фракция имеет минимальное влияние везде
	baseInfluence := 0.1 // 10% везде по умолчанию

	for _, faction := range sim.State.Factions {
		for _, domain := range sim.State.Domains {
			if domain.Influence == nil {
				domain.Influence = make(map[string]float64)
			}

			// Стартовое влияние: выше в своих доменах, ниже в чужих
			if domain.ControlledBy == faction.ID {
				domain.Influence[faction.ID] = 0.8 // 80% в своих
			} else {
				domain.Influence[faction.ID] = baseInfluence // 10% везде
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
		newInfluence[faction.ID] = SimulateFactionExpansion(faction, domainsSlice, 1)
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
				domain.Influence[factionID] = densities[i] / sum
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

// SimulateFactionExpansion симулирует распространение влияния фракции по доменам
func SimulateFactionExpansion(faction *FactionState, domains []*DomainState, ticks int) []float64 {
	n := len(domains)
	if n == 0 || ticks <= 0 {
		return nil
	}

	neighbors := buildNeighborsFromDomains(domains)

	// Начальное распределение: текущее влияние, чтобы динамика была накопительной
	u := make([]float64, n)
	for i := 0; i < n; i++ {
		u[i] = domains[i].Influence[faction.ID]
		if u[i] <= 0 {
			u[i] = MinInfluence
		}
		if domains[i].ControlledBy == faction.ID && u[i] < 1.0 {
			u[i] = 1.0
		}
		u[i] = clamp(u[i], MinInfluence, 1.0)
	}

	// Параметры модели (настройте по вкусу или добавьте поля в FactionState)
	D := minFloat(1.0, 0.2+0.8*(faction.Power/100.0))
	r := minFloat(0.2, 0.01+0.09*(faction.Territory/5.0))
	dt := 1.0 // один час на внешний шаг

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
		if substeps < 1 {
			substeps = 1
		}
		if substeps > 1000 {
			substeps = 1000
		}
	}

	// Интегрируем
	for h := 0; h < ticks; h++ {
		prev := make([]float64, n)
		copy(prev, u)
		u = SolveKPGraph(u, neighbors, D, r, dt, substeps)
		// 1) лог плотностей компактно
		row := ""
		for i := 0; i < n; i++ {
			row += fmt.Sprintf("%.3f", u[i])
			if i < n-1 {
				row += ", "
			}
		}
		//log.Printf("EXPANSION_DENSITIES faction=%q step=%d densities=[%s]", faction.ID, h+1, row)

		// 2) события захвата (пересечение порога)
		/*for i := 0; i < n; i++ {
			if prev[i] <= 0.5 && u[i] > 0.5 {
				fmt.Printf("  TAKEOVER: faction=%s domain=%s idx=%d new_density=%.3f", faction.ID, domains[i].ID, i, u[i])
			}
		}*/

		// 3) агрегаты: count, max, center of mass
		count := 0
		maxv := 0.0
		sum := 0.0
		weightSum := 0.0
		for i, v := range u {
			if v > 0.5 {
				count++
			}
			if v > maxv {
				maxv = v
			}
			sum += v
			weightSum += float64(i) * v
		}
		com := 0.0
		if sum > 0 {
			com = weightSum / sum
		} // центр масс по индексам
		log.Printf("EXPANSION_METRICS=faction=%q step=%d controlled=%d max=%.3f mean=%.3f com=%.2f", faction.ID, h+1, count, maxv, sum/float64(n), com)
	}

	type pair struct {
		id      string
		density float64
	}
	pairs := make([]pair, 0, len(domains))

	for i, d := range domains {
		pairs = append(pairs, pair{id: d.ID, density: u[i]})
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].density > pairs[j].density })

	top := minInt(3, len(pairs))
	for i := 0; i < top; i++ {
		log.Printf("EXPANSION_METRICS=TOP_TAKEOVER_CANDIDATE faction=%q rank=%d domain=%q density=%.3f", faction.ID, i+1, pairs[i].id, pairs[i].density)
	}
	log.Printf("\n_____________________________________")
	return u
}
