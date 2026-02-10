package main

import (
	"math/rand"
)

// generateTickEvent генерирует случайное событие для текущего тика
func (sim *WorldSimulator) generateTickEvent(tick int64) *GameEvent {
	eventType := rand.Intn(5)

	switch eventType {
	case 0:
		return sim.generateMysteryEvent(tick)
	case 1:
		return sim.generateResourceEvent(tick)
	case 2:
		return sim.generateCulturalEvent(tick)
	case 3:
		return sim.generateDangerEvent(tick)
	default:
		return nil
	}
}

// generateMysteryEvent генерирует мистическое событие
func (sim *WorldSimulator) generateMysteryEvent(tick int64) *GameEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		domains = append(domains, d)
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	titles := []string{
		"Ancient entity stirs in the shadows",
		"A mysterious figure appears in the mist",
		"Strange markings discovered on ancient stones",
		"Whispers of something forgotten echo through the domain",
	}

	return &GameEvent{
		Type:      "mystery",
		EventKind: EventKindWorld,
		Tick:      tick,
		EventData: WorldEventData{
			Meta: WorldEventMeta{
				Location:    domain.ID,
				Title:       titles[rand.Intn(len(titles))],
				Description: "Something mysterious is happening in " + domain.Name + ". The inhabitants are uneasy.",
				Consequence: domain.Name + " stability -3",
			},
		},
	}

}

// generateResourceEvent генерирует событие открытия ресурсов
func (sim *WorldSimulator) generateResourceEvent(tick int64) *GameEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		if d.ControlledBy == FactionCorporateConsortium {
			domains = append(domains, d)
		}
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	return &GameEvent{
		Type:      "resource_discovery",
		EventKind: EventKindGeneric,
		Tick:      tick,
		EventData: WorldEventData{
			Meta: WorldEventMeta{
				Location: domain.ID,
				Title:    "Valuable resource discovered",
			},
		},
	}
}

// generateCulturalEvent генерирует культурное событие
func (sim *WorldSimulator) generateCulturalEvent(tick int64) *GameEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		if d.ControlledBy == FactionRepentantCommunes {
			domains = append(domains, d)
		}
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	return &GameEvent{
		Type:      "cultural_event",
		Tick:      tick,
		EventKind: EventKindGeneric,
		EventData: WorldEventData{
			Meta: WorldEventMeta{
				Location:    domain.ID,
				Title:       "Cultural festival",
				Description: "A grand cultural festival is taking place in " + domain.Name + ". It attracts visitors from all over the world.",
				Consequence: domain.Name + " stability +5",
			},
		},
	}
}

// generateDangerEvent генерирует событие опасности
func (sim *WorldSimulator) generateDangerEvent(tick int64) *GameEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		if d.DangerLevel > 5 {
			domains = append(domains, d)
		}
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	return &GameEvent{
		Type:      "danger_event",
		Tick:      tick,
		EventKind: EventKindGeneric,
		EventData: WorldEventData{
			Meta: WorldEventMeta{
				Location:    domain.ID,
				Title:       "Dangerous situation unfolds",
				Description: "A dangerous situation is unfolding in " + domain.Name + ". The inhabitants are on high alert.",
				Consequence: domain.Name + " stability -5",
			},
		},
	}
}

// generateID генерирует уникальный ID для события
func generateID() string {
	chars := "abcdefghijklmnopqrstuvwxyz"
	id := "evt_"
	for i := 0; i < 3; i++ {
		id += string(chars[rand.Intn(len(chars))])
	}
	return id
}
