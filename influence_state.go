package main

type InfluenceState map[string]map[string]float64 //factionID -> domainID -> influence

// BuildInfluenceState собирает текущее влияние фракций на домены в структуру InfluenceState для удобного доступа и передачи в функции расчёта.
func BuildInfluenceState(factionIDs []string, domains []*DomainState) InfluenceState {
	state := make(InfluenceState, len(factionIDs))
	for _, factionID := range factionIDs {
		factionInfluence := make(map[string]float64, len(domains)) // Влияние фракции на ВСЕ домены
		for _, domain := range domains {
			influence := domain.Influence[factionID] // Получаем влияние фракции на конкретный домен
			if influence <= 0 {
				influence = 0.0
			}
			factionInfluence[domain.ID] = influence // Сохраняем влияние фракции на домен в мапу
		}
		state[factionID] = factionInfluence // Сохраняем мапу влияния для фракции в общем состоянии
	}
	return state
}

// Clone создает глубокую копию InfluenceState
func (state InfluenceState) Clone() InfluenceState {
	if state == nil {
		return nil
	}
	out := make(InfluenceState, len(state))
	for factionID, domainInfluence := range state {
		domainInfluenceCopy := make(map[string]float64, len(domainInfluence))
		for domainID, influence := range domainInfluence {
			domainInfluenceCopy[domainID] = influence
		}
		out[factionID] = domainInfluenceCopy
	}
	return out
}

// ApplyInfluenceStateToDomains применяет данные из InfluenceState обратно в структуру доменов, обновляя влияние каждой фракции на каждый домен.
func ApplyInfluenceStateToDomains(state InfluenceState, factionIDs []string, domains []*DomainState) {
	for _, domain := range domains { // Походимся по каждому домену
		if domain.Influence == nil {
			domain.Influence = make(map[string]float64)
		}
		for _, factionID := range factionIDs {
			domain.Influence[factionID] = state[factionID][domain.ID] // Обновляем влияние каждой фракцции в domain из InfluenceState
		}
	}
}
