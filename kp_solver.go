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

    dtSub := dt / float64(substeps)
    for s := 0; s < substeps; s++ {
        next := make([]float64, n)
        for i := 0; i < n; i++ {
            diff := 0.0
            for _, j := range neighbors[i] {
                diff += (uWork[j] - uWork[i])
            }
            diffusion := D * diff
            reaction := r * uWork[i] * (1.0 - uWork[i])
            next[i] = uWork[i] + dtSub*(diffusion+reaction)
            if next[i] < 0 { next[i] = 0 }
            if next[i] > 1 { next[i] = 1 }
        }
        uWork = next
    }
    return uWork
}
