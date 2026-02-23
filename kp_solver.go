package main

func SolveKPGraph(
	faction *FactionState,
	domains []*DomainState,
	D, r, dt float64,
	substeps int,
	warMask map[string]bool,
) map[string]float64 {
	n := len(domains)
	if n == 0 {
		return map[string]float64{}
	}
	if substeps <= 0 {
		substeps = 1
	}

	// Дальше мы составляем матрицы. Для матриц нам нужны индексы, а не ID доменов.
	// Поэтому дальше идёт подготовка к рассчёту.

	// 1. Индексируем domain.ID в idx
	domainIDs := make([]string, n)
	domainIDToIdx := make(map[string]int, n)
	influenceToIdx := make([]float64, n)
	for i, domain := range domains {
		domainIDs[i] = domain.ID
		domainIDToIdx[domain.ID] = i
		// Записываем влияние фракции на домене в индексированный срез
		influenceToIdx[i] = domain.Influence[faction.ID]
	}

	// 2) Соседи в индексном виде (второй проход)
	neighbors := make([][]int, n)
	for i, domain := range domains {
		for _, neighborID := range domain.AdjacentDomains {
			if j, exists := domainIDToIdx[neighborID]; exists {
				// пропускаем self-loop, если сосед - это сам домен
				if j == i {
					continue
				}
				neighbors[i] = append(neighbors[i], j)
			}
		}
	}
	// 3. Индексируем войны
	warIdx := make([]bool, n)
	if warMask != nil {
		for i, domainID := range domainIDs {
			warIdx[i] = warMask[domainID]
		}
	}

	// 4. Составляем рабочие буферы
	curr := make([]float64, n)
	next := make([]float64, n)

	for i := 0; i < n; i++ {
		curr[i] = clamp(influenceToIdx[i], 0.0, 1.0)
	}

	eff := func(x float64) float64 {
		return maxFloat(0, x-DiffusionThreshold)
	}

	dtSub := dt / float64(substeps)

	// 5. Явная интеграция уравнения Колмогорова-Плискунова
	for s := 0; s < substeps; s++ { // Для каждого субшага
		for i := 0; i < n; i++ { // Для каждого домена
			if warIdx[i] {
				next[i] = curr[i] // Война гасит рост и диффузию, домен остаётся без изменений
				continue
			}
			ui := curr[i]
			uiEff := eff(ui)

			diff := 0.0
			for _, j := range neighbors[i] {
				if warIdx[j] {
					continue // Сосед в состоянии войны не участвует в диффузии
				}
				diff += eff(curr[j]) - uiEff // Диффузия зависит от разницы эффективного влияния между соседями
			}
			diffusion := D * diff
			reaction := r * ui * (1.0 - ui)
			next[i] = clamp(ui+dtSub*(diffusion+reaction), 0.0, 1.0) // Обновляем влияние с учётом диффузии и реакции, и обрезаем до [0,1]
		}

		// Свапаем буферы для следующего шага
		curr, next = next, curr
	}

	// 6. Конвертируем результат обратно в мапу ID → влияние
	result := make(map[string]float64, n)
	for i, domainID := range domainIDs {
		result[domainID] = curr[i]
	}
	return result
}

// Отвечает только за пространственную диффузию влияния на графе.
// Никакой борьбы внутри одного домена тут нет
// Входящие аргументы: u[f][d] - влияние фракции f в домене d
// , neighbors[d] - соседи домена d
// , D[f] - коэффициент диффузии для фракции f
// , dt - шаг времени
func applyKPPDiffusionStep(state InfluenceState, factionIDs []string, domains []*DomainState, neighbors map[string][]string,
	diffusionRateByFaction map[string]float64, dt float64, warMaskByDomain map[string]bool) InfluenceState {

	nextInflence := state.Clone() // Копируем текущее состояние, чтобы записывать в него результаты

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
			nextFactionInfluence[domain.ID] = influence + dt*(D*diff) // Обновляем влияние с учётом диффузии
		}
	}
	return nextInflence
}
