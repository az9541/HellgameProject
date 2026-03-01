package main

import (
	"math"
	"sort"
)

// UpdateDomainStability обновляет стабильность всех доменов
func (sim *WorldSimulator) UpdateDomainStability() {
	for _, domain := range sim.State.Domains {
		stabilityRegen := 0.1 // Базовое восстановление стабильности
		stabilityRegenModifier := 1.1
		domainEffects := sim.State.TimedEffects[domain.ID]

		// Заполняем эффекты в домене, активные на текущем тике
		activeEffects := domainEffects[:0]
		for _, effect := range domainEffects {
			if sim.State.GlobalTick >= effect.StartTick && sim.State.GlobalTick < effect.StartTick+effect.Duration {
				activeEffects = append(activeEffects, effect)
			}
		}

		// Война снижает стабильность
		if sim.getActiveWarForDomain(domain.ID) != nil {
			stabilityRegenModifier *= 0.5
		}
		// TODO!!! Учесть эффекты корректно. Не просто обновлять curvedEffect для одной фракции.
		curvedEffect := 1.0
		for _, effect := range activeEffects {
			effectCoefficient := float64(sim.State.GlobalTick-effect.StartTick) / float64(effect.Duration)
			curvedEffect = 1 - effect.BasePenalty*math.Pow(10, -effect.DecayRate*effectCoefficient)
		}
		// Модификатор владельца домена
		switch domain.ControlledBy {
		case FactionNone:
			stabilityRegenModifier *= 0.5
		case FactionCorporateConsortium:
			stabilityRegenModifier *= 1.05
		case FactionRepentantCommunes:
			stabilityRegenModifier *= 1.1
		case FactionNeoTormentors:
			stabilityRegenModifier *= 1.02
		}
		stabilityRegenModifier *= curvedEffect
		// Модификатор популяции
		switch {
		case domain.Population >= 6000:
			stabilityRegenModifier *= 0.85
		case domain.Population >= 4000:
			stabilityRegenModifier *= 0.92
		case domain.Population >= 2000:
			stabilityRegenModifier *= 0.93
		case domain.Population >= 1000:
			stabilityRegenModifier *= 0.95
		default:
			stabilityRegenModifier *= 1.0
		}

		// Модификатор уровня опасности
		switch {
		case domain.DangerLevel >= 9:
			stabilityRegenModifier *= 0.9
		case domain.DangerLevel >= 7:
			stabilityRegenModifier *= 1
		case domain.DangerLevel >= 5:
			stabilityRegenModifier *= 1.02
		case domain.DangerLevel >= 3:
			stabilityRegenModifier *= 1.03
		case domain.DangerLevel >= 1:
			stabilityRegenModifier *= 1.05
		default:
			stabilityRegenModifier *= 1.1
		}

		finalStabilityChange := stabilityRegen * stabilityRegenModifier
		stabilityFactor := 1 + (finalStabilityChange - stabilityRegen)
		domain.Stability = clamp(domain.Stability*stabilityFactor, 0, 100)
		sim.State.TimedEffects[domain.ID] = activeEffects
	}
}

func (sim *WorldSimulator) UpdateDomainDanger() {
	for _, domain := range sim.State.Domains {
		dangerChange := 0.0
		if sim.getActiveWarForDomain(domain.ID) != nil {
			dangerChange += 0.2
		}
		switch domain.ControlledBy {
		case FactionNone:
			dangerChange += 0.1
		case FactionCorporateConsortium:
			dangerChange -= 0.05
		case FactionRepentantCommunes:
			dangerChange -= 0.1
		case FactionNeoTormentors:
			dangerChange += 0.01
		}
		switch {
		case domain.Population >= 6000:
			dangerChange -= 0.1
		case domain.Population >= 4000:
			dangerChange -= 0.05
		case domain.Population >= 2000:
			dangerChange -= 0.025
		case domain.Population >= 1000:
			dangerChange += 0.025
		default:
			dangerChange += 0.01
		}
		domain.DangerLevel = clamp(domain.DangerLevel+dangerChange, 0, 10)
	}
}
func (sim *WorldSimulator) UpdateDomainResources() {
	for _, domain := range sim.State.Domains {
		resRegen := 2.0      // Базовое восстановление ресурсов
		resMultiplier := 1.0 // Модификатор восстановления

		if sim.getActiveWarForDomain(domain.ID) != nil {
			resMultiplier = 0.5 // Война является серьёзным препятствием для восстановления ресурсов
		}
		if domain.ControlledBy == FactionNone {
			resMultiplier *= 1.35 // Неконтролируемые домены восстанавливают ресурсы быстрее
		}
		switch {
		case domain.Stability >= 80:
			resMultiplier *= 3.0
		case domain.Stability >= 60:
			resMultiplier *= 1.5
		case domain.Stability >= 40:
			resMultiplier *= 1.2
		case domain.Stability >= 20:
			resMultiplier *= 0.8
		default:
			resMultiplier *= 0.4
		}
		switch {
		case domain.Population >= 6000:
			resMultiplier *= 2.0
		case domain.Population >= 4000:
			resMultiplier *= 1.5
		case domain.Population >= 2000:
			resMultiplier *= 1.2
		case domain.Population >= 1000:
			resMultiplier *= 0.8
		default:
			resMultiplier *= 0.5
		}
		switch {
		case domain.DangerLevel >= 9:
			resMultiplier *= 0.85
		case domain.DangerLevel >= 7:
			resMultiplier *= 1.1
		case domain.DangerLevel >= 5:
			resMultiplier *= 1.25
		case domain.DangerLevel >= 3:
			resMultiplier *= 1.1
		case domain.DangerLevel >= 1:
			resMultiplier *= 0.9
		default:
			resMultiplier *= 0.9
		}
		resMultiplier = 1 + (resMultiplier-1)*0.6
		domain.Resources = minFloat(domain.Resources+resRegen*resMultiplier, 100)
	}
}

// syncFactionDomains перестраивает DomainsHeld у всех фракций на основе current ControlledBy
func (sim *WorldSimulator) syncFactionDomains() {
	// очистить списки
	for _, f := range sim.State.Factions {
		f.DomainsHeld = f.DomainsHeld[:0]
	}
	// заполнить заново
	for _, d := range sim.State.Domains {
		if f := sim.State.Factions[d.ControlledBy]; f != nil {
			f.DomainsHeld = append(f.DomainsHeld, d.ID)
		}
	}
}

// transferDomainControl передаёт контроль над доменом новой фракции
func (sim *WorldSimulator) transferDomainControl(domain *DomainState, newOwner *FactionState) {
	oldOwner := sim.State.Factions[domain.ControlledBy]

	if newOwner != nil && oldOwner != nil && oldOwner.ID == newOwner.ID {
		return // ничего не менять
	}

	if oldOwner != nil {
		oldOwner.removeDomain(domain.ID)
	}

	if newOwner == nil {
		domain.ControlledBy = FactionNone
		return
	}

	domain.ControlledBy = newOwner.ID
	newOwner.addDomain(domain.ID)
}

// getSortedDomainKeys возвращает отсортированные ключи доменов (для детерминированного порядка)
func getSortedDomainKeys(domains map[string]*DomainState) []string {
	keys := make([]string, 0, len(domains))
	for k := range domains {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// getDomainsList возвращает список доменов в порядке ключей
func getDomainsList(keys []string, domains map[string]*DomainState) []*DomainState {
	domainsSlice := make([]*DomainState, 0, len(keys))
	for _, k := range keys {
		domainsSlice = append(domainsSlice, domains[k])
	}
	return domainsSlice
}
