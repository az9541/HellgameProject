package main

func (sim *WorldSimulator) copyFactionStates() map[string]*FactionState {
	result := make(map[string]*FactionState)
	for id, faction := range sim.State.Factions {
		// Копируем DomainsHeld
		domainsCopy := make([]string, len(faction.DomainsHeld))
		copy(domainsCopy, faction.DomainsHeld)

		// Копируем Attitude
		attitudeCopy := make(map[string]float64)
		for k, v := range faction.Attitude {
			attitudeCopy[k] = v
		}

		result[id] = &FactionState{
			ID:            faction.ID,
			Name:          faction.Name,
			Power:         faction.Power,
			Territory:     faction.Territory,
			DomainsHeld:   domainsCopy,
			Attitude:      attitudeCopy,
			Resources:     faction.Resources,
			MilitaryForce: faction.MilitaryForce,
		}
	}
	return result
}

// copyDomainStates создаёт глубокую копию состояний всех доменов
func (sim *WorldSimulator) copyDomainStates() map[string]*DomainState {
	result := make(map[string]*DomainState)
	for id, domain := range sim.State.Domains {
		// Копируем Influence
		influenceCopy := make(map[string]float64)
		for k, v := range domain.Influence {
			influenceCopy[k] = v
		}

		// Копируем AdjacentDomains
		adjacentCopy := make([]string, len(domain.AdjacentDomains))
		copy(adjacentCopy, domain.AdjacentDomains)

		// Копируем Events
		eventsCopy := make([]string, len(domain.Events))
		copy(eventsCopy, domain.Events)

		result[id] = &DomainState{
			ID:              domain.ID,
			Name:            domain.Name,
			Stability:       domain.Stability,
			ControlledBy:    domain.ControlledBy,
			DangerLevel:     domain.DangerLevel,
			Population:      domain.Population,
			Mood:            domain.Mood,
			Influence:       influenceCopy,
			AdjacentDomains: adjacentCopy,
			Events:          eventsCopy,
			Resources:       domain.Resources,
		}
	}
	return result
}

// copyWars создаёт глубокую копию всех войн
func (sim *WorldSimulator) copyWars() map[string]*WarState {
	result := make(map[string]*WarState)
	for id, war := range sim.State.Wars {
		// Копируем WinnersID и LosersID
		winnersCopy := make(map[string]string)
		for k, v := range war.WinnersID {
			winnersCopy[k] = v
		}
		losersCopy := make(map[string]string)
		for k, v := range war.LosersID {
			losersCopy[k] = v
		}
		// Копируем FrozenFactionsDenseties
		densitiesCopy := make(map[string]float64)
		for k, v := range war.FrozenFactionsDenseties {
			densitiesCopy[k] = v
		}

		result[id] = &WarState{
			ID:                      war.ID,
			AttackerID:              war.AttackerID,
			DefenderID:              war.DefenderID,
			DomainID:                war.DomainID,
			StartTick:               war.StartTick,
			LastUpdateTick:          war.LastUpdateTick,
			TicksDuration:           war.TicksDuration,
			FrozenFactionsDenseties: densitiesCopy,
			AttackerCommittedForce:  war.AttackerCommittedForce,
			DefenderCommittedForce:  war.DefenderCommittedForce,
			AttackerCurrentForce:    war.AttackerCurrentForce,
			DefenderCurrentForce:    war.DefenderCurrentForce,
			Momentum:                war.Momentum,
			AttackerMorale:          war.AttackerMorale,
			DefenderMorale:          war.DefenderMorale,
			IsOver:                  war.IsOver,
			WinnersID:               winnersCopy,
			LosersID:                losersCopy,
		}
	}
	return result
}

// copyEventLog создаёт копию журнала событий.
func (sim *WorldSimulator) copyEventLog() []GameEvent {
	result := make([]GameEvent, len(sim.State.EventLog))
	copy(result, sim.State.EventLog)
	return result
}

// CopyInfluenceState создает глубокую копию InfluenceState
func (state InfluenceState) CopyInfluenceState() InfluenceState {
	if state == nil {
		return nil
	}
	out := make(InfluenceState, len(state))
	for factionID, domainInfluence := range state {
		domainInfluenceCopy := make(map[string]float64, len(domainInfluence))
		for domainID, influence := range domainInfluence {
			domainInfluenceCopy[domainID] = influence
		}
		out[factionID] = domainInfluenceCopy
	}
	return out
}
