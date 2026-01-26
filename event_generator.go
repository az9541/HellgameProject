package main

import (
	"math/rand"
)

// generateTickEvent генерирует случайное событие для текущего тика
func (sim *WorldSimulator) generateTickEvent(tick int64) *WorldEvent {
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
func (sim *WorldSimulator) generateMysteryEvent(tick int64) *WorldEvent {
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

	return &WorldEvent{
		ID:          generateID(),
		Tick:        tick,
		Type:        "mystery",
		Location:    domain.ID,
		Title:       titles[rand.Intn(len(titles))],
		Description: "Something ancient and unknown has stirred...",
		Consequence: "heresy_danger_level +2",
	}
}

// generateResourceEvent генерирует событие открытия ресурсов
func (sim *WorldSimulator) generateResourceEvent(tick int64) *WorldEvent {
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

	return &WorldEvent{
		ID:          generateID(),
		Tick:        tick,
		Type:        "resource_discovery",
		Location:    domain.ID,
		Title:       "New mineral deposits discovered",
		Description: "Corporate teams have found rich deposits of infernal ore.",
		Consequence: "corporate_consortium power +3",
	}
}

// generateCulturalEvent генерирует культурное событие
func (sim *WorldSimulator) generateCulturalEvent(tick int64) *WorldEvent {
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

	return &WorldEvent{
		ID:          generateID(),
		Tick:        tick,
		Type:        "cultural",
		Location:    domain.ID,
		Title:       "Community gathering brings hope",
		Description: "The communes organize a gathering to celebrate survival and mutual aid.",
		Consequence: domain.Name + " stability +5",
	}
}

// generateDangerEvent генерирует событие опасности
func (sim *WorldSimulator) generateDangerEvent(tick int64) *WorldEvent {
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

	return &WorldEvent{
		ID:          generateID(),
		Tick:        tick,
		Type:        "danger",
		Location:    domain.ID,
		Title:       "Dangerous creature sighting",
		Description: "Reports of a dangerous entity roaming the domain.",
		Consequence: domain.Name + " danger_level +1",
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
