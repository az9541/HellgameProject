package main

// Если KPP - это за распространение влияния по доменам, то LV - это распространение влияние ВНУТРИ ОДНОГО домена
// Пара примеров, как влияют параметры на рост влияния внутри домена:
// - Если в домене ни у кого ещё нет влияния. Тогда crowding будет 0, и рост будет максимально возможным, определяемым growthRateByFaction. Это моделирует ситуацию, когда фракция начинает распространяться в новом домене без конкуренции.
// - Если в домене уже есть влияние от других фракций, то crowding будет больше 0, и рост будет замедляться. Это моделирует конкуренцию за ресурсы внутри домена, когда фракции мешают друг другу расти.
// - Если в домене идёт война (warMaskByDomain[domain.ID] == true), то warScale будет 0, и рост будет полностью подавлен. Это моделирует ситуацию, когда из-за войны внутри домена невозможно эффективно распространять влияние.
// - Если в домене уже много влияния от других фракций, то competition будет высоким, и рост будет ещё больше подавляться. Это моделирует сильную конкуренцию внутри домена, когда фракции активно мешают друг другу расти.
func applyLVReactionStep(
	state InfluenceState, factionIDs []string, domains []*DomainState,
	growthRateByFaction map[string]float64, dt float64,
	warMaskByDomain map[string]bool) InfluenceState {

	nextInfluence := state.CopyInfluenceState() // Копируем текущее состояние, чтобы записывать в него результаты

	const (
		lvCapacityK        = 1.0
		lvCompetitionAlpha = 0.05
		lvCrowdingWeight   = 0.8
		lvWarSuppression   = 0.05 // Война сильно подавляет рост внутри домена
	)

	// Цикл по всем доменам. Внутри каждого домена рассчитываем рост влияния каждой фракции с учётом конкуренции, заполненности и войны.
	for _, domain := range domains {
		warScale := 1.0
		if warMaskByDomain[domain.ID] {
			warScale = lvWarSuppression // Война подавляет рост внутри домена
		}

		totalCrowding := 0.0 // Суммарная заполненность домена влиянием всех фракций, влияет на снижение роста
		for _, factionID := range factionIDs {
			totalCrowding += lvCrowdingWeight * state[factionID][domain.ID] // Чем больше всего влияния в домене, тем меньше рост для всех (конкуренция за ресурсы)
		}

		// Считаем рост для каждой фракции в домене с учётом конкуренции от других фракций.
		for _, factionID := range factionIDs {
			competition := 0.0
			for _, otherFactionID := range factionIDs {
				if otherFactionID == factionID {
					continue
				}
				competition += lvCompetitionAlpha * state[otherFactionID][domain.ID] // Конкуренция от других фракций, снижает рост
			}
			competition *= warScale // Война усиливает конкуренцию, подавляя рост

			influence := state[factionID][domain.ID]
			crowding := 1.0 - totalCrowding/lvCapacityK                                                                   // Чем больше всего влияния, тем меньше рост (логистический рост)
			influenceGrowthRate := influence * growthRateByFaction[factionID] * crowding * (1.0 - competition) * warScale // Рост с учётом всех факторов
			nextInfluence[factionID][domain.ID] = influence + dt*influenceGrowthRate                                      // Обновляем влияние с учётом роста
		}
	}
	return nextInfluence
}
