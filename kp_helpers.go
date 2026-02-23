package main

// buildNeighborsFromDomains конвертирует список доменов с явно определёнными
// AdjacentDomains в список смежности для KPP-уравнения.
//
// tl;dr: преобразует граф доменов (ID-ориентированный) в индексный граф
// для SolveKPGraph. domains[i].AdjacentDomains содержит ID соседей,
// нам нужны их индексы в срезе.
// Почему: SolveKPGraph работает с индексами (матрица смежности),
// а DomainState хранит ID. Нужна конверсия.
func buildNeighborsFromDomains(domains []*DomainState) map[string][]string {
	n := len(domains)
	neighbors := make(map[string][]string, n)

	for _, domain := range domains {
		neighbors[domain.ID] = append([]string(nil), domain.AdjacentDomains...)
	}
	return neighbors
}
