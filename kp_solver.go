package main

// SolveKPGraph делает явную интеграцию уравнения Колмогорова-Плискунова
// на графе, заданном списком смежности neighbors.
// u: начальное распределение (len = n), neighbors[i] = индексы соседей i.
// D: коэффициент диффузии, r: скорость роста, dt: основной шаг времени,
// substeps: число субшагов внутри одного часового шага (для стабильности).
func SolveKPGraph(u []float64, neighbors [][]int, D, r, dt float64, substeps int) []float64 {
	n := len(u)
	uWork := make([]float64, n)
	copy(uWork, u)
	// Эффективная функция для диффузии, которая обнуляет влияние ниже порога
	eff := func(x float64) float64 {
		return maxFloat(0, x-DiffusionThreshold)
	}

	dtSub := dt / float64(substeps)
	for s := 0; s < substeps; s++ {
		next := make([]float64, n)
		for i := 0; i < n; i++ {
			diff := 0.0
			for _, j := range neighbors[i] {
				// Разница эффективного влияния между соседями
				diff += (eff(uWork[j]) - eff(uWork[i]))
			}
			diffusion := D * diff
			reaction := r * uWork[i] * (1.0 - uWork[i])
			next[i] = uWork[i] + dtSub*(diffusion+reaction)
			if next[i] < 0 {
				next[i] = 0
			}
			if next[i] > 1 {
				next[i] = 1
			}
		}
		uWork = next
	}
	return uWork
}

// Отвечает только за пространственную диффузию влияния на графе.
// Никакой борьбы внутри одного домена тут нет
// Входящие аргументы: u[f][d] - влияние фракции f в домене d
// , neighbors[d] - соседи домена d
// , D[f] - коэффициент диффузии для фракции f
// , dt - шаг времени
func applyKPPDiffusionStep(u [][]float64, neighbors [][]int, D []float64, dt float64) [][]float64 {
	nF := len(u) // Количество фракций в игре
	if nF == 0 {
		return nil
	}

	nD := len(u[0])               // Количество доменов в игре
	next := make([][]float64, nF) // Инициализации новой матрицы влияния после шага диффузии

	eff := func(x float64) float64 { // Обрезка по DiffusionThreshold - мелкое влияниене перетекает в другие домены
		return maxFloat(0, x-DiffusionThreshold)
	}

	for f := 0; f < nF; f++ { // Просто заполняем строки матрицы
		next[f] = make([]float64, nD)
	}

	// Суть цикла: для КАЖДОЙ фракции считаем её КАЖДЫЙ домен и распространяем влияние на КАЖДОГО соседа
	// при условии, что влияние выше эффективного порога перетока.
	for f := 0; f < nF; f++ { // Обрабатываем КАЖДУЮ фракцию. Переносим влияние конкретной фракции по доменам
		for d := 0; d < nD; d++ { // Это как раз цикл по доменам для фракции f.
			diff := 0.0
			for _, j := range neighbors[d] { // В соседей перетекает только влияние выше порога
				diff += eff(u[f][j]) - eff(u[f][d]) // Считаем диффузю
			}
			next[f][d] = u[f][d] + dt*(D[f]*diff) // Заполняем новую матрицу
		}
	}
	// Всё. Вы великолепны.
	return next
}
