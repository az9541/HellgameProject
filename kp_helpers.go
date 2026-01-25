package main

// buildNeighborsFromDomains возвращает список соседей для каждого домена в срезе
// Варианты поведения:
//   - Если у объектов DomainState нет явной информации о соседях, используем
//     упрощённую модель: домены считаются расположенными в цепочке в порядке
//     элементов среза, и соседями являются предыдущий и следующий индекс.
//   - Такая модель совместима с текущей реализацией, которая использует
//     индексную дискретизацию (i-1, i+1). Если в будущем появится поле с
//     информацией о связях (например, AdjacentIDs []string), функцию можно
//     расширить для построения графа по идентификаторам доменов.
func buildNeighborsFromDomains(domains []*DomainState) [][]int {
	n := len(domains)
	neighbors := make([][]int, n)

	for i := 0; i < n; i++ {
		if i > 0 {
			neighbors[i] = append(neighbors[i], i-1)
		}
		if i < n-1 {
			neighbors[i] = append(neighbors[i], i+1)
		}
	}

	return neighbors
}
