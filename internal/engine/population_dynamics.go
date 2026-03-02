package engine

// Вычисляем скалярное поле потенциала. Зависит с одной стороны от стабильности, с другой - от опасности.
func (domain *DomainState) calculateMigrationPotential() (migrationPotential float64) {
	migrationPotential = PopPotentialStabilityWeight*domain.Stability -
		PopPotentialDangerWeight*float64(domain.DangerLevel)

	return
}

// UpdateDomainPopulation обновляет население доменов, решая уравнение Конвекции-Диффузии-Реакции на графе.
// dP/dt = D*\nabla^2 P - \nabla*(\mu*P*\nabla U) + r*P*(1 - P/K)
func (sim *WorldSimulator) UpdateDomainPopulation() {
	if len(sim.State.Domains) == 0 {
		return
	}

	// 1. Получаем снапшот доменов
	// Это нужно, чтобы расчеты (dP/dt) зависели только от состояния на начало тика (t),.
	snapshot := sim.CopyDomainStates()

	// Массив для накопления изменений населения (dP/dt)
	deltas := make(map[string]float64)

	// 2. Считаем потоки и реакции, читая ТОЛЬКО из snapshot
	for id, domainSnap := range snapshot {
		// population: Текущее население домена i
		population := float64(domainSnap.Population)

		// migrationPotential: Потенциал домена i
		migrationPotential := domainSnap.calculateMigrationPotential()

		// domainMaxCapacity: Емкость домена i (зависит от ресурсов)
		domainMaxCapacity := PopBaseCapacity * (0.5 + domainSnap.Resources/200.0)

		// --- РЕАКЦИЯ: r * P * (1 - P/K) ---
		// reaction: Коэффициент роста. Если потенциал отрицательный, начинается вымирание.
		reaction := PopBaseGrowthRate
		if migrationPotential < 0 {
			// Смертность пропорциональна негативному потенциалу (ограничена 5% за тик)
			reaction = clamp(PopBaseGrowthRate*(migrationPotential/50.0), -0.05, PopBaseGrowthRate)
		}
		reactionGrowth := reaction * population * (1.0 - population/domainMaxCapacity)
		deltas[id] += reactionGrowth

		// Считаем потоки с соседями (Диффузия и Конвекция)
		for _, neighborID := range domainSnap.AdjacentDomains {
			neighborSnap := snapshot[neighborID]

			// neighborPopulation: Население соседа j
			neighborPopulation := float64(neighborSnap.Population)

			// neighborMigrationPotential: Потенциал соседа j
			neighborMigrationPotential := neighborSnap.calculateMigrationPotential()

			// Дискретный аналог лапласиана на графе: сумма разностей (P_j - P_i).
			// Делим на 2, так как цикл пройдет по ребру i-j дважды (от i к j, и от j к i).
			// Это закон Фика. Диффузия. Поток пропорционален разности концентраций (населений) и коэффициенту диффузии.
			diffusionFlux := (PopDiffusionCoeff * (neighborPopulation - population)) / 2.0
			deltas[id] += diffusionFlux

			// Направленный поток по градиенту потенциала (\nabla U = U_j - U_i).
			// Люди бегут только туда, где потенциал строго больше (U_j > U_i).
			// Это конвекция. Миграция происходит в сторону более привлекательных доменов.
			if neighborMigrationPotential > migrationPotential {
				// Поток из i в j: \mu * P_i * (U_j - U_i)
				convectionFlux := PopConvectionMobility * population * (neighborMigrationPotential - migrationPotential)

				// Убыль из текущего домена (i)
				deltas[id] -= convectionFlux

				// Прибыль в соседний домен (j).
				// Это гарантирует закон сохранения массы (беженцы не исчезают).
				deltas[neighborID] += convectionFlux
			}
		}
	}

	// 3. Применяем накопленные изменения (dP/dt) к ОРИГИНАЛЬНЫМ доменам (P(t+dt) = P(t) + dP)
	for id, deltaP := range deltas {
		domain := sim.State.Domains[id]
		newPopulation := float64(domain.Population) + deltaP

		// Защита от вымирания в ноль (физическое ограничение P >= 0)
		if newPopulation < 0 {
			newPopulation = 0
		}
		domain.Population = int(newPopulation)
	}
}
