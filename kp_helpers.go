package main

// buildNeighborsFromDomains конвертирует список доменов с явно определёнными
// AdjacentDomains в список смежности для KPP-уравнения.
//
// tl;dr: преобразует граф доменов (ID-ориентированный) в индексный граф
// для SolveKPGraph. domains[i].AdjacentDomains содержит ID соседей,
// нам нужны их индексы в срезе.
// Почему: SolveKPGraph работает с индексами (матрица смежности),
// а DomainState хранит ID. Нужна конверсия.
func buildNeighborsFromDomains(domains []*DomainState) [][]int {
	n := len(domains)
	neighbors := make([][]int, n)

	// Создаём маппинг ID → индекс
	domainIDToIndex := make(map[string]int)
	for i, domain := range domains {
		domainIDToIndex[domain.ID] = i
	}

	// Для каждого домена конвертируем AdjacentDomains (ID) в индексы
	for i, domain := range domains {
		for _, neighborID := range domain.AdjacentDomains {
			if j, exists := domainIDToIndex[neighborID]; exists {
				neighbors[i] = append(neighbors[i], j)
			}
		}
	}

	return neighbors
}
