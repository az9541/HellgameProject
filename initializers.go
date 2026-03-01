package main

import "math/rand"

// createInitialFactions создаёт начальные фракции мира
func createInitialFactions() map[string]*FactionState {
	return map[string]*FactionState{
		FactionCorporateConsortium: {
			ID:             FactionCorporateConsortium,
			Name:           "Corporate Consortium",
			Power:          70,
			Territory:      3.5,
			DomainsHeld:    []string{DomainGreed},
			Attitude:       make(map[string]float64),
			Resources:      80,
			MilitaryForce:  75,
			LastActionTime: 0,
		},
		FactionRepentantCommunes: {
			ID:             FactionRepentantCommunes,
			Name:           "Repentant Communes",
			Power:          50,
			Territory:      1.8,
			DomainsHeld:    []string{DomainGluttony},
			Attitude:       make(map[string]float64),
			Resources:      40,
			MilitaryForce:  35,
			LastActionTime: 0,
		},
		FactionNeoTormentors: {
			ID:             FactionNeoTormentors,
			Name:           "Neo-Tormentors",
			Power:          65,
			Territory:      2.5,
			DomainsHeld:    []string{DomainViolence},
			Attitude:       make(map[string]float64),
			Resources:      70,
			MilitaryForce:  85,
			LastActionTime: 0,
		},
		FactionCaravanGuilds: {
			ID:             FactionCaravanGuilds,
			Name:           "Caravan Guilds",
			Power:          45,
			Territory:      0.9,
			DomainsHeld:    []string{DomainLimbo},
			Attitude:       make(map[string]float64),
			Resources:      60,
			MilitaryForce:  40,
			LastActionTime: 0,
		},
		FactionAncientRemnants: {
			ID:             FactionAncientRemnants,
			Name:           "Ancient Remnants",
			Power:          30,
			Territory:      0.3,
			DomainsHeld:    []string{DomainHeresy},
			Attitude:       make(map[string]float64),
			Resources:      50,
			MilitaryForce:  60,
			LastActionTime: 0,
		},
	}
}

// generateDomainTopology создаёт случайный граф доменов с 1-2 соседями каждый.
// Limbo (первый) и Betrayal (последний) не соединены напрямую.
// Граф симметричный.
//
// tl;dr: моделируем хаотичность Ада через случайную топологию.
// Почему так: KPP-уравнение на графе уже поддерживает произвольные соседства,
// нам просто нужна адекватная топология. Ограничения (не более 2 соседей,
// граничные домены не связаны) балансируют хаос с минимальной предсказуемостью.
func generateDomainTopology(domains map[string]*DomainState, allDomainIDs []string) {
	// Инициализируем пустые списки соседей
	for _, domain := range domains {
		domain.AdjacentDomains = []string{}
	}

	// Преобразуем список доменов в порядок: Limbo -> Betrayal
	// allDomainIDs должен быть в правильном порядке из инициализации
	n := len(allDomainIDs)

	// Граф для отслеживания добавленных рёбер (чтобы избежать дубликатов)
	edges := make(map[string]map[string]bool)
	for _, id := range allDomainIDs {
		edges[id] = make(map[string]bool)
	}

	// Для каждого домена генерируем 1-2 случайных соседей
	for i, domainID := range allDomainIDs {
		// Определяем, сколько соседей (1 или 2)
		numNeighbors := 1 + rand.Intn(2) // 1 или 2

		// Собираем кандидатов (все домены кроме себя)
		candidates := []string{}
		for j, candidateID := range allDomainIDs {
			// Пропускаем сам домен
			if i == j {
				continue
			}
			// Граничный случай: Limbo (индекс 0) и Betrayal (индекс n-1) не соединены
			if (i == 0 && j == n-1) || (i == n-1 && j == 0) {
				continue
			}
			candidates = append(candidates, candidateID)
		}

		// Выбираем случайных соседей из кандидатов
		for k := 0; k < numNeighbors && len(candidates) > 0; k++ {
			// Случайный индекс в candidates
			randIdx := rand.Intn(len(candidates))
			neighborID := candidates[randIdx]

			// Проверяем, нет ли уже этого ребра
			if !edges[domainID][neighborID] {
				edges[domainID][neighborID] = true
				edges[neighborID][domainID] = true // симметричное ребро

				// Добавляем в список соседей обоих доменов
				domains[domainID].AdjacentDomains = append(domains[domainID].AdjacentDomains, neighborID)
				domains[neighborID].AdjacentDomains = append(domains[neighborID].AdjacentDomains, domainID)
			}

			// Удаляем кандидата из списка, чтобы не выбрать его дважды
			candidates = append(candidates[:randIdx], candidates[randIdx+1:]...)
		}
	}
}

// updateCreateInitialDomains — обновленная функция с вызовом генерации графа
func createInitialDomains() (map[string]*DomainState, []string) {
	domains := map[string]*DomainState{
		DomainLimbo: {
			ID:           DomainLimbo,
			Name:         "Limbo",
			Stability:    60,
			ControlledBy: FactionCaravanGuilds,
			DangerLevel:  2,
			Population:   5000,
			Mood:         "stable",
			Influence:    make(map[string]float64),
		},
		DomainLust: {
			ID:           DomainLust,
			Name:         "Circle of Lust",
			Stability:    40,
			ControlledBy: FactionNone,
			DangerLevel:  5,
			Population:   3000,
			Mood:         "exploited",
			Influence:    make(map[string]float64),
		},
		DomainGluttony: {
			ID:           DomainGluttony,
			Name:         "Circle of Gluttony",
			Stability:    55,
			ControlledBy: FactionRepentantCommunes,
			DangerLevel:  1,
			Population:   2500,
			Mood:         "hopeful",
			Influence:    make(map[string]float64),
		},
		DomainGreed: {
			ID:           DomainGreed,
			Name:         "Circle of Greed",
			Stability:    55,
			ControlledBy: FactionCorporateConsortium,
			DangerLevel:  3,
			Population:   4000,
			Mood:         "unrest",
			Influence:    make(map[string]float64),
		},
		DomainWrath: {
			ID:           DomainWrath,
			Name:         "Circle of Wrath",
			Stability:    20,
			ControlledBy: FactionNone,
			DangerLevel:  9,
			Population:   6000,
			Mood:         "terrified",
			Influence:    make(map[string]float64),
		},
		DomainHeresy: {
			ID:           DomainHeresy,
			Name:         "Circle of Heresy",
			Stability:    45,
			ControlledBy: FactionAncientRemnants,
			DangerLevel:  3,
			Population:   3000,
			Mood:         "mysterious",
			Influence:    make(map[string]float64),
		},
		DomainViolence: {
			ID:           DomainViolence,
			Name:         "Circle of Violence",
			Stability:    45,
			ControlledBy: FactionNeoTormentors,
			DangerLevel:  4,
			Population:   6000,
			Mood:         "chaotic",
			Influence:    make(map[string]float64),
		},
		DomainFraud: {
			ID:           DomainFraud,
			Name:         "Circle of Fraud",
			Stability:    30,
			ControlledBy: FactionNone,
			DangerLevel:  8,
			Population:   2000,
			Mood:         "deceptive",
			Influence:    make(map[string]float64),
		},
		DomainBetrayance: {
			ID:           DomainBetrayance,
			Name:         "Ninth Circle",
			Stability:    10,
			ControlledBy: FactionNone,
			DangerLevel:  10,
			Population:   500,
			Mood:         "despairing",
			Influence:    make(map[string]float64),
		},
	}

	// Порядок доменов: от первого ко второму (важен для граничного случая)
	allDomainIDs := []string{
		DomainLimbo,
		DomainLust,
		DomainGluttony,
		DomainGreed,
		DomainWrath,
		DomainHeresy,
		DomainViolence,
		DomainFraud,
		DomainBetrayance,
	}

	// Генерируем случайную топологию графа
	generateDomainTopology(domains, allDomainIDs)

	return domains, allDomainIDs
}
