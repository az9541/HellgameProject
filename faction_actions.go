package main

import (
	"log"
	"math/rand"
)

// executeFactionActions выполняет действия всех фракций
func (sim *WorldSimulator) executeFactionActions() {
	for _, faction := range sim.Factions {
		// Сначала всегда проверяем кандидатуры на захват по текущим плотностям влияния
		var topDomain *DomainState
		var topInfluence float64

		for _, domain := range sim.Domains {
			// Если домен контролируется текущей фракцией - пропускаем
			if domain.ControlledBy == faction.ID {
				continue
			}
			// Проверяем влияние фракции на домен
			if infl, ok := domain.Influence[faction.ID]; ok && infl > InfluenceToTakeOver {
				if infl > topInfluence {
					topInfluence = infl
					topDomain = domain
				}
			}
		}

		// Если есть кандидат — пробуем захват или приводим к войне
		if topDomain != nil {
			if topDomain.ControlledBy != "none" {
				sim.StartWarTrigger(faction, sim.Factions[topDomain.ControlledBy], topDomain)
			} else {
				sim.attemptDomainTakeover(faction, topDomain, topInfluence)
			}
		} else {
			log.Printf("INFO: no takeover candidate for faction=%q (threshold=%.3f), faction influence on domen: %.2f", faction.ID, InfluenceToTakeOver, topInfluence)
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
		sim.EventBus.Publish(GameEvent{
			Type: "DOMAIN_TAKEOVER",
			Tick: sim.GlobalTick,
			Data: map[string]any{
				"attacker": attacker.Name,
				"domain":   domain.Name,
			},
		})
	} else {
		sim.EventBus.Publish(GameEvent{
			Type: "TAKEOVER_FAILED",
			Tick: sim.GlobalTick,
			Data: map[string]any{
				"attacker":    attacker.Name,
				"domain":      domain.Name,
				"probability": probability,
			},
		})
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
	for _, d := range sim.Domains {
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
		Type: "TRADE_ROUTE",
		Tick: sim.GlobalTick,
		Data: map[string]any{
			"from": domain1.Name,
			"to":   domain2.Name,
			"by":   faction.Name,
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

func (sim *WorldSimulator) updateFactionMilitaryForce() {
	factionsInWar := make(map[string]struct{})
	for _, war := range sim.Wars {
		if war == nil || war.IsOver {
			continue
		}
		factionsInWar[war.AttackerID] = struct{}{}
		factionsInWar[war.DefenderID] = struct{}{}
	}
	for _, faction := range sim.Factions {
		if _, ok := factionsInWar[faction.ID]; ok {
			faction.MilitaryForce = minFloat(faction.MilitaryForce+0.1, MaxMilitaryForce)
		} else {
			faction.MilitaryForce = minFloat(faction.MilitaryForce+1, MaxMilitaryForce)
		}
	}
}
