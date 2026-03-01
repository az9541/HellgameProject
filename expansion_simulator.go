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
	state := sim.solveExpansionEquations(factionIDs, domains, 1)
	if state == nil {
		return
	}
	ApplyInfluenceStateToDomains(state, factionIDs, domains)
	// Ограничиваем общее влияние на домен, чтобы не уходило за 100%
	for _, domain := range domains {
		capDomainInfluence(domain.Influence)
	}

	if !sim.cfg.DisableKPPTickLogs { // Отключем спам логов KPP при батчевом дебаге
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
	if faction == nil || n == 0 || ticks <= 0 {
		return nil
	}

	neighbors := buildNeighborsFromDomains(domains)

	// Стабильный порядок доменов для конвертации map -> []float64 на выходе
	domainIDs := make([]string, n)
	for i, d := range domains {
		domainIDs[i] = d.ID
	}

	ownedByDomain := make(map[string]bool, n)
	hasOwned := false
	for _, d := range domains {
		isOwned := d.ControlledBy == faction.ID
		ownedByDomain[d.ID] = isOwned
		if isOwned {
			hasOwned = true
		}
	}

	// Текущее влияние фракции по domainID
	influence := make(map[string]float64, n)
	for _, d := range domains {
		v := d.Influence[faction.ID]
		if v <= 0 {
			v = MinInfluence
		}
		influence[d.ID] = clamp(v, MinInfluence, 1.0)
	}

	// KPP-параметры (как было)
	D := minFloat(1.0, 0.002+0.0015*(faction.Power/100.0))
	r := minFloat(0.1, 0.005+0.065*(faction.Territory/5.0))
	dt := 1.0

	// Число субшагов для устойчивости
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
	dtSub := dt / float64(substeps)

	eff := func(x float64) float64 {
		return maxFloat(0, x-DiffusionThreshold)
	}

	for h := 0; h < ticks; h++ {
		// 1) KPP (диффузия + реакция) на субшагах
		for s := 0; s < substeps; s++ {
			next := make(map[string]float64, n)

			for _, domainID := range domainIDs {
				ui := influence[domainID]
				uiEff := eff(ui)

				diff := 0.0
				for _, neighborID := range neighbors[domainID] {
					diff += eff(influence[neighborID]) - uiEff
				}

				diffusion := D * diff
				reaction := r * ui * (1.0 - ui)
				next[domainID] = ui + dtSub*(diffusion+reaction)
			}

			influence = next
		}

		// 2) Source
		if hasOwned {
			wealth := faction.WealthIndex
			for _, domainID := range domainIDs {
				if ownedByDomain[domainID] {
					influence[domainID] += dt * sourceBaseRate * wealth * maxFloat(0, sourceTargetOwned-influence[domainID])
				}
			}
		}

		// 3) Spillover
		for _, domainID := range domainIDs {
			if !ownedByDomain[domainID] {
				continue
			}

			overflow := spillRate * maxFloat(0, influence[domainID]-spillThreshold)
			if overflow <= 0 {
				continue
			}

			count := 0
			for _, neighborID := range neighbors[domainID] {
				if !ownedByDomain[neighborID] {
					count++
				}
			}
			if count == 0 {
				continue
			}

			share := overflow / float64(count)
			for _, neighborID := range neighbors[domainID] {
				if !ownedByDomain[neighborID] {
					influence[neighborID] += share
				}
			}
		}

		// 4) Clamp
		for _, domainID := range domainIDs {
			influence[domainID] = clamp(influence[domainID], MinInfluence, 1.0)
		}
	}

	// map -> []float64 в порядке domains
	out := make([]float64, n)
	for i, domainID := range domainIDs {
		out[i] = influence[domainID]
	}
	return out
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
) InfluenceState {
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

	// 1) Состояние влияния в формате solver-а
	state := BuildInfluenceState(factionIDs, domains)

	// 2) Параметры по фракциям в map-формате
	ownedByFactionDomain := make(map[string]map[string]bool, nF)
	diffusionRateByFaction := make(map[string]float64, nF) // D
	growthRateByFaction := make(map[string]float64, nF)    // r
	wealthByFaction := make(map[string]float64, nF)

	for _, factionID := range factionIDs {
		faction := sim.State.Factions[factionID]
		if faction == nil {
			continue
		}

		ownedByFactionDomain[factionID] = make(map[string]bool, nD)
		kppParams := NewKPPParameters(faction)
		diffusionRateByFaction[factionID] = kppParams.Diffusion
		growthRateByFaction[factionID] = kppParams.Growth
		wealthByFaction[factionID] = faction.WealthIndex
		for _, d := range domains {
			ownedByFactionDomain[factionID][d.ID] = (d.ControlledBy == factionID)
		}
	}
	// 3) Стабильность явной схемы: считаем substeps как раньше
	maxDeg := 0
	for _, nb := range neighbors {
		if len(nb) > maxDeg {
			maxDeg = len(nb)
		}
	}
	maxD := 0.0 // Максимальная D среди фракций, влияет на число субшагов для устойчивости
	for _, factionID := range factionIDs {
		if diffusionRateByFaction[factionID] > maxD {
			maxD = diffusionRateByFaction[factionID]
		}
	}
	substeps := 1
	if maxD > 0 && maxDeg > 0 { // Если есть диффузия и есть связи между доменами, то нужно разбивать на субшаги для устойчивости
		// Простыми словами, устойчивость это когда влияние не начинает "скакать" и уходить за 100% из-за слишком большого шага.
		// Чем больше D и чем больше соседей, тем быстрее может расти влияние, и тем меньше должен быть шаг для устойчивости.
		substeps = int(math.Ceil(maxD * float64(maxDeg) * 2.0))
		substeps = maxInt(1, minInt(1000, substeps))
	}
	dtSub := 1.0 / float64(substeps)

	factionsSnapshot := makeFactionStatesSnapshot(factionsMap)
	// 4) Основной цикл интеграции
	for h := 0; h < ticks; h++ {
		for s := 0; s < substeps; s++ {
			warMaskByDomain := make(map[string]bool, nD)
			for _, d := range domains {
				warMaskByDomain[d.ID] = sim.getActiveWarForDomain(d.ID) != nil
			}

			state = applyKPPDiffusionStep(
				state,
				factionIDs,
				domains,
				neighbors,
				diffusionRateByFaction,
				dtSub,
				warMaskByDomain,
			)

			state = applyLVReactionStep(
				state,
				factionIDs,
				domains,
				growthRateByFaction,
				dtSub,
				warMaskByDomain,
			)

			state = applySourceStep(
				state,
				factionIDs,
				domains,
				ownedByFactionDomain,
				wealthByFaction,
				dtSub,
				warMaskByDomain,
			)

			state = applySpilloverStep(
				state,
				factionsSnapshot,
				domains,
				ownedByFactionDomain,
				neighbors,
				dtSub,
				warMaskByDomain,
			)

			clampInfluenceInPlace(state, factionIDs, domains, 0.0, 1.0)
		}
	}

	return state
}

// Подпитываем влияние фракции на собственых доменах
func applySourceStep(state InfluenceState, factionIDs []string, domains []*DomainState,
	factionOwnedDomains map[string]map[string]bool, wealthByFaction map[string]float64,
	dt float64, warMaskByDomain map[string]bool) InfluenceState {
	const warSuppressionForSource = 0.2 // Война сильно подавляет генерацию влияния в своих доменах
	nextInfluence := state.CopyInfluenceState()

	for _, factionID := range factionIDs {
		wealth := wealthByFaction[factionID]
		for _, domain := range domains {
			if !factionOwnedDomains[factionID][domain.ID] {
				continue
			}
			if warMaskByDomain[domain.ID] {
				nextInfluence[factionID][domain.ID] += dt * sourceBaseRate *
					wealth * maxFloat(0, sourceTargetOwned-nextInfluence[factionID][domain.ID]) * warSuppressionForSource // Война подавляет генерацию влияния в своих доменах
			} else {
				nextInfluence[factionID][domain.ID] += dt * sourceBaseRate * wealth * maxFloat(0, sourceTargetOwned-nextInfluence[factionID][domain.ID])
			}
		}
	}
	return nextInfluence
}

func applySpilloverStep(state InfluenceState, factionStates map[string]FactionState,
	domains []*DomainState, factionOwnedDomains map[string]map[string]bool,
	neighbors map[string][]string, dt float64, warMaskByDomain map[string]bool) InfluenceState {
	type spillCandidate struct {
		domainID       string
		attractiveness float64
	}
	nextInfluence := state.CopyInfluenceState()

	domainByID := make(map[string]*DomainState, len(domains))
	for _, d := range domains {
		domainByID[d.ID] = d
	}
	for factionID, factionState := range factionStates {
		for _, sourceDomain := range domains {
			sourceID := sourceDomain.ID
			if warMaskByDomain[sourceID] {
				continue
			}
			if !factionOwnedDomains[factionID][sourceID] {
				continue
			}

			sourceValue := state[factionID][sourceID]
			overflow := spillRate * dt * maxFloat(0, sourceValue-spillThreshold)
			if overflow <= 0 {
				continue
			}

			candidates := make([]spillCandidate, 0, len(neighbors[sourceID]))
			totalAttractiveness := 0.0

			for _, neighborID := range neighbors[sourceID] {
				if factionOwnedDomains[factionID][neighborID] {
					continue
				}
				if warMaskByDomain[neighborID] {
					continue
				}

				neighborDomain, ok := domainByID[neighborID]
				if !ok {
					continue
				}

				neighborInfluence := state[factionID][neighborID]
				attractiveness := maxFloat(0.0, calcDomainAttractiveness(
					factionState.Resources,
					neighborDomain,
					neighborInfluence,
					0,
				))

				candidates = append(candidates, spillCandidate{
					domainID:       neighborID,
					attractiveness: attractiveness,
				})
				totalAttractiveness += attractiveness
			}

			if len(candidates) == 0 {
				continue
			}

			// Убираем влияние из источника
			nextInfluence[factionID][sourceID] -= overflow
			if totalAttractiveness <= 0 {
				share := overflow / float64(len(candidates))
				for _, candidate := range candidates {
					nextInfluence[factionID][candidate.domainID] += share
				}
				continue
			}

			for _, candidate := range candidates {
				share := overflow * (candidate.attractiveness / totalAttractiveness)
				nextInfluence[factionID][candidate.domainID] += share
			}
		}
	}
	return nextInfluence
}

func clampInfluenceInPlace(state InfluenceState, factionIDs []string, domains []*DomainState, minV, maxV float64) {
	for _, factionID := range factionIDs {
		for _, domain := range domains {
			state[factionID][domain.ID] = clamp(state[factionID][domain.ID], minV, maxV)
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

func buildOwnedByFactionDomain(
	factionIDs []string,
	domains []*DomainState,
	factions map[string]*FactionState,
) map[string]map[string]bool {
	owned := make(map[string]map[string]bool, len(factionIDs))

	domainInScope := make(map[string]struct{}, len(domains))
	for _, d := range domains {
		domainInScope[d.ID] = struct{}{}
	}

	for _, factionID := range factionIDs {
		row := make(map[string]bool, len(domains))
		if f := factions[factionID]; f != nil {
			for _, domainID := range f.DomainsHeld {
				if _, ok := domainInScope[domainID]; ok {
					row[domainID] = true
				}
			}
		}
		owned[factionID] = row
	}

	return owned
}
