package main

import (
	"fmt"
	"log"
	"math/rand"
	"slices"
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
				sim.resolveFactionWar(faction, sim.Factions[topDomain.ControlledBy], topDomain)
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
		log.Printf("EVENT=DOMAIN_TAKEOVER tick=%d attacker=%q domain=%q", sim.GlobalTick, attacker.Name, domain.Name)
	} else {
		log.Printf("EVENT=TAKEOVER_FAILED tick=%d attacker=%q domain=%q probability=%.4f", sim.GlobalTick, attacker.Name, domain.Name, probability)
	}
}

// resolveFactionWar разрешает войну между двумя фракциями за домен
func (sim *WorldSimulator) resolveFactionWar(attacker, defender *FactionState, domain *DomainState) string {
	// Базовые силы с учётом влияния на домене
	baseAttackerStrength := attacker.MilitaryForce * (1.0 + domain.Influence[attacker.ID])
	baseDefenderStrength := defender.MilitaryForce * (1.0 + domain.Influence[defender.ID])

	// Проверка: атакующий должен иметь минимальное соотношение сил
	strengthRatio := baseAttackerStrength / baseDefenderStrength
	if strengthRatio < MinAttackStrengthRatio {
		// Атакующий слишком слаб - отказывается от атаки
		log.Printf("EVENT=WAR_ABORTED tick=%d attacker=%q defender=%q domain=%q reason=insufficient_strength ratio=%.3f (min=%.3f)",
			sim.GlobalTick, attacker.Name, defender.Name, domain.Name, strengthRatio, MinAttackStrengthRatio)
		return "war_aborted"
	}

	// Добавляем случайный фактор (10% вариация)
	randomFactor := 0.9 + rand.Float64()*0.2 // от 0.9 до 1.1
	attackerStrength := baseAttackerStrength * randomFactor
	defenderStrength := baseDefenderStrength * (0.9 + rand.Float64()*0.2)

	// Логируем начало конфликта
	log.Printf("EVENT=WAR_STARTED tick=%d attacker=%q defender=%q domain=%q a_str=%.1f d_str=%.1f ratio=%.3f",
		sim.GlobalTick, attacker.Name, defender.Name, domain.Name, attackerStrength, defenderStrength, strengthRatio)

	// Обе стороны тратят ресурсы на войну
	attacker.Resources = maxFloat(attacker.Resources-WarResourceCost, 0)
	defender.Resources = maxFloat(defender.Resources-WarResourceCost, 0)

	// Вычисляем соотношение сил в битве для определения интенсивности
	battleRatio := attackerStrength / defenderStrength
	victoryMargin := battleRatio - 1.0 // насколько больше победитель (0.0 = равные силы)

	if attackerStrength > defenderStrength {
		// ========== АТАКУЮЩИЙ ПОБЕДИЛ ==========

		// Передаём домен атакующему
		sim.transferDomainControl(domain, attacker)

		// Динамическое изменение power в зависимости от победы
		// Чем больше превосходство - тем больше бонус, но с убывающей отдачей
		powerGainMultiplier := 1.0 + victoryMargin*0.5 // от 1.0 до ~1.5
		powerGain := BasePowerGain * powerGainMultiplier

		// Поражение защитника зависит от того, насколько он был слабее
		powerLossMultiplier := 1.0 + (1.0/strengthRatio-1.0)*0.3 // больше потерь если был намного слабее
		powerLoss := BasePowerLoss * powerLossMultiplier

		attacker.Power = minFloat(attacker.Power+powerGain, 100)
		defender.Power = maxFloat(defender.Power-powerLoss, 0)

		// Потери военной силы пропорциональны интенсивности битвы
		// Чем ровнее была битва - тем больше потерь
		if victoryMargin < 0.2 { // очень близкая победа
			attacker.MilitaryForce = maxFloat(attacker.MilitaryForce-3, 0)
			defender.MilitaryForce = maxFloat(defender.MilitaryForce-5, 0)
		} else if victoryMargin < 0.5 { // средняя победа
			attacker.MilitaryForce = maxFloat(attacker.MilitaryForce-2, 0)
			defender.MilitaryForce = maxFloat(defender.MilitaryForce-4, 0)
		} else { // лёгкая победа
			attacker.MilitaryForce = maxFloat(attacker.MilitaryForce-1, 0)
			defender.MilitaryForce = maxFloat(defender.MilitaryForce-3, 0)
		}

		// Последствия для домена зависят от интенсивности битвы
		stabilityLoss := 10.0 + victoryMargin*10.0 // от 10 до 20
		domain.Stability = maxFloat(domain.Stability-stabilityLoss, 0)
		domain.Mood = "conquered"

		// Создаем событие
		warEvent := WorldEvent{
			ID:       generateID(),
			Tick:     sim.GlobalTick,
			Type:     "faction_war",
			Location: domain.ID,
			Title:    fmt.Sprintf("%s conquered %s", attacker.Name, domain.Name),
			Description: fmt.Sprintf("After a fierce battle, %s seized control from %s. Victory margin: %.1f%%",
				attacker.Name, defender.Name, victoryMargin*100),
			Consequence: fmt.Sprintf("Power: %s +%.1f, %s -%.1f", attacker.Name, powerGain, defender.Name, powerLoss),
			Factions:    []string{attacker.ID, defender.ID},
		}
		sim.EventLog = append(sim.EventLog, warEvent)

		log.Printf("EVENT=WAR_RESULT tick=%d result=VICTORY attacker=%q domain=%q margin=%.3f power_gain=%.1f power_loss=%.1f",
			sim.GlobalTick, attacker.Name, domain.Name, victoryMargin, powerGain, powerLoss)
		return "attacker_victory"

	} else {
		// ========== ЗАЩИТНИК ОТБИЛСЯ ==========

		// Вычисляем насколько защитник был сильнее
		defenseMargin := defenderStrength/attackerStrength - 1.0

		// Динамическое изменение power
		// Защитник получает бонус, но меньше чем при победе атакующего
		// Атакующий теряет больше, если был намного слабее
		defenderPowerGain := BasePowerGain * 0.4 * (1.0 + defenseMargin*0.3)           // от 4 до ~5.2
		attackerPowerLoss := BasePowerLoss * 0.5 * (1.0 + (1.0/strengthRatio-1.0)*0.5) // больше потерь если был слабее

		defender.Power = minFloat(defender.Power+defenderPowerGain, 100)
		attacker.Power = maxFloat(attacker.Power-attackerPowerLoss, 0)

		// Потери военной силы
		if defenseMargin < 0.2 { // очень близкая защита
			defender.MilitaryForce = maxFloat(defender.MilitaryForce-2, 0)
			attacker.MilitaryForce = maxFloat(attacker.MilitaryForce-4, 0)
		} else if defenseMargin < 0.5 { // средняя защита
			defender.MilitaryForce = maxFloat(defender.MilitaryForce-1, 0)
			attacker.MilitaryForce = maxFloat(attacker.MilitaryForce-3, 0)
		} else { // лёгкая защита
			defender.MilitaryForce = maxFloat(defender.MilitaryForce-0.5, 0)
			attacker.MilitaryForce = maxFloat(attacker.MilitaryForce-2, 0)
		}

		// Домен получает небольшой урон от попытки захвата
		domain.Stability = maxFloat(domain.Stability-5, 0)

		// Создаем событие
		defenseEvent := WorldEvent{
			ID:          generateID(),
			Tick:        sim.GlobalTick,
			Type:        "faction_war",
			Location:    domain.ID,
			Title:       fmt.Sprintf("%s defended %s", defender.Name, domain.Name),
			Description: fmt.Sprintf("%s successfully repelled the attack from %s.", defender.Name, attacker.Name),
			Consequence: fmt.Sprintf("Power: %s +%.1f, %s -%.1f", defender.Name, defenderPowerGain, attacker.Name, attackerPowerLoss),
			Factions:    []string{attacker.ID, defender.ID},
		}
		sim.EventLog = append(sim.EventLog, defenseEvent)

		log.Printf("EVENT=WAR_RESULT tick=%d result=DEFEAT attacker=%q domain=%q margin=%.3f power_gain=%.1f power_loss=%.1f",
			sim.GlobalTick, attacker.Name, domain.Name, defenseMargin, defenderPowerGain, attackerPowerLoss)
		return "defender_victory"
	}
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

	log.Printf("EVENT=TRADE_ROUTE tick=%d from=%q to=%q by=%q", sim.GlobalTick, domain1.Name, domain2.Name, faction.Name)
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
	// Получаем события в текущем тике
	factionsInWar := make([]string, 0)
	for _, event := range sim.EventLog {
		if event.Tick == sim.GlobalTick {
			if event.Type == "faction_war" {
				factionsInWar = append(factionsInWar, event.Factions...)
			}
		}
	}
	for _, faction := range sim.Factions {
		if slices.Contains(factionsInWar, faction.ID) {
			faction.MilitaryForce = minFloat(faction.MilitaryForce+0.1, MaxMilitaryForce)
		} else {
			faction.MilitaryForce = minFloat(faction.MilitaryForce+1, MaxMilitaryForce)
		}
	}
}
