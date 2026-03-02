package main

import (
	"math"
	"math/rand"
)

// minFloat возвращает минимальное из двух float64 значений
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// maxFloat возвращает максимальное из двух float64 значений
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// minInt возвращает минимальное из двух int значений
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// maxInt возвращает максимальное из двух int значений
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func clamp(x, minV, maxV float64) float64 {
	if x < minV {
		return minV
	}
	if x > maxV {
		return maxV
	}
	return x
}

// lerp линейно интерполирует между a и b по параметру t ∈ [0,1].
// t=0 → a, t=1 → b.
func lerp(a, b, t float64) float64 {
	return a + t*(b-a)
}

func makeLog(forceRatio float64) float64 {
	forceFactor := math.Log(forceRatio)
	if forceFactor > 1 {
		forceFactor = 1
	}
	if forceFactor < -1 {
		forceFactor = -1
	}
	return forceFactor
}

func awarenessFromInfluence(influence float64) float64 {
	if influence <= 0 {
		return MinAwareness
	}
	a := MinAwareness + (1.0-MinAwareness)*(math.Log(1+InfluenceToAwarenessFactor*influence)/math.Log(1+InfluenceToAwarenessFactor))
	return clamp(a, MinAwareness, 1.0)
}

func estimateForceWithAwareness(force, awareness float64) float64 {
	// Добавляем рандома. Оценка может быть как в большую сторону, так и в меньшую
	// Если фактор шума положителен, то оценка завышается, если отрицателен — занижается
	maxNoise := 0.4
	noise := (rand.Float64()*2 - 1) * maxNoise * (1 - awareness)
	return force * (1 + noise)
}

// Clone делает глубокую копию FactionState.
func (f *FactionState) Clone() *FactionState {
	if f == nil {
		return nil
	}

	domainsCopy := append([]string(nil), f.DomainsHeld...)

	attitudeCopy := make(map[string]float64, len(f.Attitude))
	for k, v := range f.Attitude {
		attitudeCopy[k] = v
	}

	return &FactionState{
		ID:             f.ID,
		Name:           f.Name,
		Power:          f.Power,
		Territory:      f.Territory,
		DomainsHeld:    domainsCopy,
		Attitude:       attitudeCopy,
		Resources:      f.Resources,
		MilitaryForce:  f.MilitaryForce,
		LastActionTime: f.LastActionTime,
	}
}

// Снимок "живых" фракций в value-map (без указателей).
func makeFactionStatesSnapshot(factions map[string]*FactionState) map[string]FactionState {
	snapshot := make(map[string]FactionState, len(factions))
	for _, faction := range factions {
		if faction == nil {
			continue
		}
		snapshot[faction.ID] = *faction.Clone()
	}
	return snapshot
}

func calculatePoplationScale(domain *DomainState, medianPopulation float64) (popScale float64) {
	// Считаем общую популяцию везде
	popScale = 0.0
	if domain.Population > 0 {
		popScale = clamp(medianPopulation/float64(domain.Population), 0.1, 2.0)
	}
	return popScale
}

func calculateMedianPopulation(domains []*DomainState) float64 {
	totalPopulation := 0.0
	for _, d := range domains {
		totalPopulation += float64(d.Population)
	}
	return totalPopulation / float64(len(domains))
}

func (sim *WorldSimulator) calcDomainImportanceForFaction(domain *DomainState, faction *FactionState) float64 {
	if len(faction.DomainsHeld) == 0 {
		return 1.0 // Нет доменов — нечего считать, максимальная важность
	}
	survivalFactor := 1.0 / float64(len(faction.DomainsHeld))

	overallFactionResources := 0.0
	for _, dID := range faction.DomainsHeld {
		if d, ok := sim.State.Domains[dID]; ok {
			overallFactionResources += d.Resources
		}
	}
	resFactor := 0.0
	if overallFactionResources > 0 {
		resFactor = domain.Resources / overallFactionResources
	}

	overallFactionPopulation := 0.0
	for _, dID := range faction.DomainsHeld {
		if d, ok := sim.State.Domains[dID]; ok {
			overallFactionPopulation += float64(d.Population)
		}
	}
	popFactor := 0.0
	if overallFactionPopulation > 0 {
		popFactor = float64(domain.Population) / overallFactionPopulation
	}

	return DomainImportanceSurvivalWeight*survivalFactor + DomainImportanceResourcesWeight*resFactor + DomainImportancePopulationWeight*popFactor
}
