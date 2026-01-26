package main

// copyFactionStates создаёт глубокую копию состояний всех фракций
func (sim *WorldSimulator) copyFactionStates() map[string]*FactionState {
	copy := make(map[string]*FactionState)
	for id, faction := range sim.Factions {
		copy[id] = &FactionState{
			ID:            faction.ID,
			Name:          faction.Name,
			Power:         faction.Power,
			Territory:     faction.Territory,
			DomainsHeld:   append([]string{}, faction.DomainsHeld...),
			Attitude:      faction.Attitude,
			Resources:     faction.Resources,
			MilitaryForce: faction.MilitaryForce,
		}
	}
	return copy
}

// copyDomainStates создаёт глубокую копию состояний всех доменов
func (sim *WorldSimulator) copyDomainStates() map[string]*DomainState {
	copy := make(map[string]*DomainState)
	for id, domain := range sim.Domains {
		copy[id] = &DomainState{
			ID:           domain.ID,
			Name:         domain.Name,
			Stability:    domain.Stability,
			ControlledBy: domain.ControlledBy,
			DangerLevel:  domain.DangerLevel,
			Population:   domain.Population,
			Mood:         domain.Mood,
		}
	}
	return copy
}
