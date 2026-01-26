package main

import (
	"sort"
)

// updateDomainStability обновляет стабильность всех доменов
func (sim *WorldSimulator) updateDomainStability() {
	for _, domain := range sim.Domains {
		controller := sim.Factions[domain.ControlledBy]
		if controller == nil {
			domain.Stability = maxFloat(domain.Stability-2, 0) // Контроля нет - уходим в хаос
			continue
		}

		// Стабильность доменов в зависимости от того, кто их контроллирует
		if controller.ID == FactionCorporateConsortium {
			// Corporate = stable but exploitative
			domain.Stability = minFloat(domain.Stability+1, 80)
		} else if controller.ID == FactionRepentantCommunes {
			// Communes = moderate stability, good morale
			domain.Stability = minFloat(domain.Stability+2, 90)
		} else if controller.ID == FactionNeoTormentors {
			// Neo-Tormentors = oppressive but effective
			domain.Stability = minFloat(domain.Stability+0.5, 70)
		}

		// Danger level decreases with stability
		if domain.Stability > 70 {
			domain.DangerLevel = maxInt(domain.DangerLevel-1, 1)
		} else if domain.Stability < 30 {
			domain.DangerLevel = minInt(domain.DangerLevel+1, 10)
		}
	}
}

// syncFactionDomains перестраивает DomainsHeld у всех фракций на основе current ControlledBy
func (sim *WorldSimulator) syncFactionDomains() {
	// очистить списки
	for _, f := range sim.Factions {
		f.DomainsHeld = f.DomainsHeld[:0]
	}
	// заполнить заново
	for _, d := range sim.Domains {
		if f := sim.Factions[d.ControlledBy]; f != nil {
			f.DomainsHeld = append(f.DomainsHeld, d.ID)
		}
	}
}

// transferDomainControl передаёт контроль над доменом новой фракции
func (sim *WorldSimulator) transferDomainControl(domain *DomainState, newOwner *FactionState) {
	oldOwner := sim.Factions[domain.ControlledBy]

	if newOwner != nil && oldOwner != nil && oldOwner.ID == newOwner.ID {
		return // ничего не менять
	}

	if oldOwner != nil {
		oldOwner.removeDomain(domain.ID)
	}

	if newOwner == nil {
		domain.ControlledBy = "none"
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
