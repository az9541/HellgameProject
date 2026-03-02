package engine

// Отвечает только за пространственную диффузию влияния на графе.
// Никакой борьбы внутри одного домена тут нет
// Входящие аргументы: u[f][d] - влияние фракции f в домене d
// , neighbors[d] - соседи домена d
// , D[f] - коэффициент диффузии для фракции f
// , dt - шаг времени
func applyKPPDiffusionStep(state InfluenceState, factionIDs []string, domains []*DomainState, neighbors map[string][]string,
	diffusionRateByFaction map[string]float64, dt float64, warMaskByDomain map[string]bool) InfluenceState {
	medianPopulation := calculateMedianPopulation(domains)
	nextInflence := state.CopyInfluenceState() // Копируем текущее состояние, чтобы записывать в него результаты

	eff := func(x float64) float64 { // Обрезка по DiffusionThreshold - мелкое влияниене перетекает в другие домены
		return maxFloat(0, x-DiffusionThreshold)
	}

	// Проходимся по каждой фракции
	for _, factionID := range factionIDs {
		D := diffusionRateByFaction[factionID]
		currentFactionInfluence := state[factionID]     // Влияние текущей фракции на все домены, индексированное по domain.ID
		nextFactionInfluence := nextInflence[factionID] // Ссылка на мапу для записи результатов для текущей фракции

		// Проходимся по всем доменам фракции
		for _, domain := range domains {
			influence := currentFactionInfluence[domain.ID]
			if warMaskByDomain[domain.ID] {
				nextFactionInfluence[domain.ID] = influence // Война гасит рост и диффузию, домен остаётся без изменений
				continue
			}
			effectiveInfluence := eff(influence)
			diff := 0.0 // Суммарный эффект от соседей
			// Проходимся по каждому соседу домена
			for _, neighborID := range neighbors[domain.ID] {
				if warMaskByDomain[neighborID] {
					continue // Сосед в состоянии войны не участвует в диффузии
				}
				diff += eff(currentFactionInfluence[neighborID]) - effectiveInfluence // Диффузия зависит от разницы эффективного влияния между соседями
			}
			popScale := 1.0
			if domain.Population > 0 {
				popScale = clamp(medianPopulation/float64(domain.Population), 0.1, 2.0)
			}

			nextFactionInfluence[domain.ID] = influence + dt*(D*diff)*popScale
		}
	}
	return nextInflence
}
