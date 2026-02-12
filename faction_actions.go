package main

import (
	"math/rand"
	"sort"
)

// executeFactionActions выполняет действия всех фракций
func (sim *WorldSimulator) executeFactionActions() {
	for _, faction := range sim.State.Factions {
		// Сначала всегда проверяем кандидатуры на захват по текущим плотностям влияния
		topDomains := faction.getTopFactionDomainInfluence(sim)
		if len(topDomains) == 0 {
			continue
		}
		activeWars := faction.getActiveWars(sim)
		for _, dom := range topDomains {
			if dom.ControlledBy == "none" {
				gameEventBuillder := NewBuilderGenericEvent()
				gameEventBuillder.SetType("TAKEOVER_ATTEMPT").SetTick(sim.State.GlobalTick).SetData(GenericEventData{
					EventKind: EventKindGeneric,
					EventData: map[string]any{
						"faction":     faction.Name,
						"domain":      dom.Name,
						"influence":   dom.Influence[faction.ID],
						"description": "Domain is unclaimed but has high influence. Faction attempts to take it over without war.",
					},
				})
				sim.EventBus.Publish(gameEventBuillder.Build())
				sim.attemptDomainTakeover(faction, dom, dom.Influence[faction.ID])
				continue
			}
			attractiveness := faction.calcDomainAttractiveness(dom, dom.Influence[faction.ID], len(activeWars))
			if attractiveness <= TEMPDomainAttractivnessThreshold {
				gameEventBuillder := NewBuilderGenericEvent()
				gameEventBuillder.SetType("WAR_AVOIDED").SetTick(sim.State.GlobalTick).SetData(GenericEventData{
					EventKind: EventKindGeneric,
					EventData: map[string]any{
						"pretender":         faction.Name,
						"domain_controller": sim.State.Factions[dom.ControlledBy].Name,
						"domain":            dom.Name,
						"attractiveness":    attractiveness,
						"description":       "Attractiveness is too low to justify war. Faction decides to avoid conflict for now.",
					},
				})
				sim.EventBus.Publish(gameEventBuillder.Build())
				continue
			}
			gameEventBuillder := NewBuilderGenericEvent()
			gameEventBuillder.SetType("WAR_PROBABILITY").SetTick(sim.State.GlobalTick).SetData(GenericEventData{
				EventKind: EventKindGeneric,
				EventData: map[string]any{
					"pretender":         faction.Name,
					"domain_controller": sim.State.Factions[dom.ControlledBy].Name,
					"domain":            dom.Name,
					"attractiveness":    attractiveness,
					"description":       "Attractiveness is high enough to consider war. Evaluating further conditions...",
				},
			})
			sim.EventBus.Publish(gameEventBuillder.Build())
			warStarted := sim.StartWarTrigger(faction, sim.State.Factions[dom.ControlledBy], dom)
			if warStarted {
				break // Если война началась, не рассматриваем другие домены в этом тике
			} else {
				continue // Если война не началась, продолжаем рассматривать другие домены
			}
		}

		// Отдельно — случайные второстепенные действия (торговля, ресурсы)
		if rand.Float64() < 0.4 { // 40% шанс на побочное действие
			action := rand.Intn(3)
			switch action {
			case 1:
				sim.establishTradeRoute(faction)
			case 2:
				faction.Resources = minFloat(faction.Resources+5, 100)
			}
		}
	}
}

// attemptDomainTakeover пытается захватить домен
func (sim *WorldSimulator) attemptDomainTakeover(attacker *FactionState, domain *DomainState, influence float64) {
	baseProbability := (attacker.MilitaryForce / 100) * (1 - float64(domain.DangerLevel)/20)
	probability := baseProbability * (1.0 + influence)
	if probability >= 0.6 {
		sim.transferDomainControl(domain, attacker)
		sim.EmitEvent(GameEvent{
			Type:      "DOMAIN_TAKEOVER",
			Tick:      sim.State.GlobalTick,
			EventKind: EventKindGeneric,
			EventData: GenericEventData{
				EventKind: EventKindGeneric,
				EventData: map[string]any{
					"attacker": attacker.Name,
					"domain":   domain.Name,
				},
			}})
	} else {
		sim.EventBus.Publish(GameEvent{
			Type:      "TAKEOVER_FAILED",
			Tick:      sim.State.GlobalTick,
			EventKind: EventKindGeneric,
			EventData: GenericEventData{
				EventKind: EventKindGeneric,
				EventData: map[string]any{
					"attacker":    attacker.Name,
					"domain":      domain.Name,
					"probability": probability,
				},
			}})
	}
}

// resolveFactionWar больше не разрешает войну мгновенно — только инициирует её.
func (sim *WorldSimulator) resolveFactionWar(attacker, defender *FactionState, domain *DomainState) string {
	sim.StartWarTrigger(attacker, defender, domain)
	return "war_started"
}

// establishTradeRoute устанавливает торговый маршрут между двумя доменами
func (sim *WorldSimulator) establishTradeRoute(faction *FactionState) {
	// Выбираем два рандомных домена
	domains := make([]*DomainState, 0)
	for _, d := range sim.State.Domains {
		domains = append(domains, d)
	}

	if len(domains) < 2 {
		return
	}

	domain1 := domains[rand.Intn(len(domains))]
	domain2 := domains[rand.Intn(len(domains))]

	if domain1.ID == domain2.ID {
		return
	}

	// Устанавливаются торговые связи, даются плюшки
	// Сейчас торговая связь устанавливается просто по велению рандома, но мы это поправим
	domain1.Stability = minFloat(domain1.Stability+10, 100)
	domain2.Stability = minFloat(domain2.Stability+10, 100)
	faction.Resources += 10
	sim.EventBus.Publish(GameEvent{
		Type:      "TRADE_ROUTE",
		Tick:      sim.State.GlobalTick,
		EventKind: EventKindGeneric,
		EventData: GenericEventData{
			EventKind: EventKindGeneric,
			EventData: map[string]any{
				"from": domain1.Name,
				"to":   domain2.Name,
				"by":   faction.Name,
			},
		},
	})
}

// addDomain добавляет домен в список контролируемых фракцией
func (faction *FactionState) addDomain(id string) {
	if faction.hasDomain(id) {
		return
	}
	faction.DomainsHeld = append(faction.DomainsHeld, id)
}

// removeDomain удаляет домен из списка контролируемых фракцией
func (faction *FactionState) removeDomain(id string) {
	out := faction.DomainsHeld[:0]
	for _, d := range faction.DomainsHeld {
		if d != id {
			out = append(out, d)
		}
	}
	faction.DomainsHeld = out
}

// hasDomain проверяет, контролирует ли фракция домен
func (faction *FactionState) hasDomain(id string) bool {
	for _, d := range faction.DomainsHeld {
		if d == id {
			return true
		}
	}
	return false
}

func (faction *FactionState) getTopFactionDomainInfluence(sim *WorldSimulator) []*DomainState {
	topDomsSlice := make([]*DomainState, 0, len(sim.State.Domains))

	for _, domain := range sim.State.Domains {
		// Если домен контролируется текущей фракцией - пропускаем
		if domain.ControlledBy == faction.ID {
			continue
		}
		// Проверяем, может ли фракция достичь этого домена
		reachable, _ := faction.canReachDomain(domain, sim)
		if !reachable {
			continue
		}
		// Проверяем, что у фракции есть влияние на домен и оно выше порога для захвата
		if infl, ok := domain.Influence[faction.ID]; ok && infl > InfluenceToTakeOver {
			topDomsSlice = append(topDomsSlice, domain)
		}
	}
	// Сортируем домены по влиянию в порядке убывания
	sort.Slice(topDomsSlice, func(i, j int) bool {
		inflI := topDomsSlice[i].Influence[faction.ID]
		inflJ := topDomsSlice[j].Influence[faction.ID]
		return inflI > inflJ
	})
	return topDomsSlice
}

func (faction *FactionState) getActiveWars(sim *WorldSimulator) []*WarState {
	activeWars := make([]*WarState, 0)
	for _, war := range sim.State.Wars {
		if war == nil || war.IsOver {
			continue
		}
		if war.AttackerID == faction.ID || war.DefenderID == faction.ID {
			activeWars = append(activeWars, war)
		}
	}
	return activeWars
}

func (sim *WorldSimulator) updateFactionMilitaryForce() {
	factionsInWar := make(map[string]struct{})
	for _, war := range sim.State.Wars {
		if war == nil || war.IsOver {
			continue
		}
		factionsInWar[war.AttackerID] = struct{}{}
		factionsInWar[war.DefenderID] = struct{}{}
	}
	for _, faction := range sim.State.Factions {
		if _, ok := factionsInWar[faction.ID]; ok {
			faction.MilitaryForce = minFloat(faction.MilitaryForce+0.1, MaxMilitaryForce)
		} else {
			faction.MilitaryForce = minFloat(faction.MilitaryForce+1, MaxMilitaryForce)
		}
	}
}

func (faction *FactionState) canReachDomain(domain *DomainState, sim *WorldSimulator) (bool, []*DomainState) {
	if domain == nil {
		return false, nil
	}
	footholds := make([]*DomainState, 0)
	var isReachable bool
	if domain.ControlledBy == faction.ID {
		isReachable = true
		footholds = append(footholds, domain)
	}
	for _, neighborID := range domain.AdjacentDomains {
		if neighbor, ok := sim.State.Domains[neighborID]; ok && neighbor.ControlledBy == faction.ID {
			isReachable = true
			footholds = append(footholds, neighbor)
		}
	}
	return isReachable, footholds
}

func (faction *FactionState) calcDomainAttractiveness(domain *DomainState, influence float64, activeWars int) float64 {
	popFactor := clamp(float64(domain.Population)/10000.0, 0.1, 1)
	stabFactor := clamp(domain.Stability/100.0, 0.1, 1)
	inflFactor := clamp(influence, 0, 1) * 2.0
	dangerFactor := 3.0 - clamp(float64(domain.DangerLevel)/10.0, 0, 0.9)
	warPenalty := 1.0 - clamp(float64(activeWars)*0.2, 0, 0.8)
	return popFactor * stabFactor * inflFactor * dangerFactor * warPenalty
}
