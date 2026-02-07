package main

import (
	"fmt"
	"math"
	"math/rand"
)

// Возвращает активную войну за домен, если есть.
func (sim *WorldSimulator) getActiveWarForDomain(domainID string) *WarState {
	for _, war := range sim.Wars {
		if !war.IsOver && war.DomainID == domainID {
			return war
		}
	}
	return nil
}

// Функция запускает триггер войны
func (sim *WorldSimulator) StartWarTrigger(attacker, defender *FactionState, domain *DomainState) {
	domain, ok := sim.Domains[domain.ID]
	if !ok {
		return
	}

	// Домен никто не контролирует - смысла воевать нет
	if domain.ControlledBy == "none" || domain.ControlledBy == "" {
		return
	}

	if defender.ID == attacker.ID {
		return
	}

	// Если война по домену уже идёт — ничего не делаем
	if sim.getActiveWarForDomain(domain.ID) != nil {
		return
	}

	// Рассчитываем осведомлённость акатующего о силе защитника на домене
	attackerAwareness := awarenessFromInfluence(domain.Influence[attacker.ID])

	// Базовые силы с учётом влияния на домене
	baseAttackerStrength := attacker.MilitaryForce * (1.0 + domain.Influence[attacker.ID])
	baseDefenderStrength := defender.MilitaryForce * (1.0 + domain.Influence[defender.ID])
	estimateDefenderStrength := estimateForceWithAwareness(baseDefenderStrength, attackerAwareness)

	// Проверка: атакующий должен иметь минимальное соотношение сил
	strengthRatio := 0.0
	if estimateDefenderStrength > 0 {
		strengthRatio = baseAttackerStrength / estimateDefenderStrength
	}

	// Расчёт выделяемого контингента (40-60% доступных сил)
	// Больше выделяем, если уверены в победе
	commitmentRatio := 0.4 + 0.2*(strengthRatio-MinAttackStrengthRatio)/(2.0-MinAttackStrengthRatio)
	commitmentRatio = clamp(commitmentRatio, 0.3, 0.7)

	attackerCommitted := attacker.MilitaryForce * commitmentRatio
	defenderCommitted := defender.MilitaryForce * 0.5 // защитник выделяет 50%

	if baseDefenderStrength <= 0 {
		warID := fmt.Sprintf("war:%s:%s:%s:%d", domain.ID, attacker.ID, defender.ID, rand.Int())
		war := &WarState{
			ID:             warID,
			AttackerID:     attacker.ID,
			DefenderID:     defender.ID,
			DomainID:       domain.ID,
			StartTick:      sim.GlobalTick,
			LastUpdateTick: sim.GlobalTick,
			FrozenFactionsDenseties: map[string]float64{
				attacker.ID: domain.Influence[attacker.ID],
				defender.ID: domain.Influence[defender.ID],
			},
			AttackerCommittedForce: attackerCommitted,
			DefenderCommittedForce: 0,
			AttackerCurrentForce:   attackerCommitted,
			DefenderCurrentForce:   0,
			Momentum:               100,
			AttackerMorale:         100,
			DefenderMorale:         0,
			IsOver:                 true,
			WinnersID:              map[string]string{attacker.ID: "auto_win_defender_zero_force"},
			LosersID:               map[string]string{defender.ID: "zero_force"},
		}
		sim.Wars[war.ID] = war
		gameEventBuillder := NewBuilderWarEvent()
		gameEventBuillder.SetType("WAR_STARTED").SetTick(sim.GlobalTick).SetData(map[string]any{
			"attacker":                attacker.Name,
			"defender":                defender.Name,
			"domain":                  domain.Name,
			"reason":                  "defender_zero_force",
			"actual_defender_force":   baseDefenderStrength,
			"estimate_defender_force": estimateDefenderStrength,
		})
		sim.EventBus.Publish(gameEventBuillder.Build())
		sim.FinishWar(war, attacker.ID, defender.ID, domain)
		return
	}

	// Проверка: атакующий должен иметь минимальное соотношение сил
	if strengthRatio < MinAttackStrengthRatio {
		// Атакующий слишком слаб - отказывается от атаки
		gameEventBuilder := NewBuilderWarEvent()
		gameEventBuilder.SetType("WAR_ABORTED").SetTick(sim.GlobalTick).SetData(map[string]any{
			"attacker":                attacker.Name,
			"defender":                defender.Name,
			"domain":                  domain.Name,
			"reason":                  "insufficient_strength",
			"ratio":                   strengthRatio,
			"min":                     MinAttackStrengthRatio,
			"actual_defender_force":   baseDefenderStrength,
			"estimate_defender_force": estimateDefenderStrength,
		})
		sim.EventBus.Publish(gameEventBuilder.Build())
		return
	}

	warID := fmt.Sprintf("war:%s:%s:%s:%d", domain.ID, attacker.ID, defender.ID, rand.Int())

	// Вычитаем контингент из общих сил фракций
	attacker.MilitaryForce -= attackerCommitted
	defender.MilitaryForce -= defenderCommitted

	war := &WarState{
		ID:         warID,
		AttackerID: attacker.ID,
		DefenderID: defender.ID,
		DomainID:   domain.ID,

		StartTick:      sim.GlobalTick,
		LastUpdateTick: sim.GlobalTick,

		FrozenFactionsDenseties: map[string]float64{
			attacker.ID: domain.Influence[attacker.ID],
			defender.ID: domain.Influence[defender.ID],
		},

		AttackerCommittedForce: attackerCommitted,
		DefenderCommittedForce: defenderCommitted,
		AttackerCurrentForce:   attackerCommitted,
		DefenderCurrentForce:   defenderCommitted,

		Momentum:       0,
		AttackerMorale: 100,
		DefenderMorale: 100,

		IsOver:    false,
		WinnersID: make(map[string]string),
		LosersID:  make(map[string]string),
	}

	sim.Wars[war.ID] = war

	gameEventBuilder := NewBuilderWarEvent()
	gameEventBuilder.SetType("WAR_STARTED").SetTick(sim.GlobalTick).SetData(map[string]any{
		"attacker":                attacker.Name,
		"defender":                defender.Name,
		"domain":                  domain.Name,
		"attacker_committed":      attackerCommitted,
		"defender_committed":      defenderCommitted,
		"attacker_remaining":      attacker.MilitaryForce,
		"defender_remaining":      defender.MilitaryForce,
		"ratio":                   strengthRatio,
		"actual_defender_force":   baseDefenderStrength,
		"estimate_defender_force": estimateDefenderStrength,
	})
	sim.EventBus.Publish(gameEventBuilder.Build())
}

// Подсчёт активных войн фракции
func (sim *WorldSimulator) countActiveWars(factionID string) int {
	count := 0
	for _, war := range sim.Wars {
		if !war.IsOver && (war.AttackerID == factionID || war.DefenderID == factionID) {
			count++
		}
	}
	return count
}

func (sim *WorldSimulator) UpdateWars() {
	for _, war := range sim.Wars {
		if war.IsOver {
			continue
		}
		// Собираем данные по участникам
		attacker, okAttacker := sim.Factions[war.AttackerID]
		defender, okDefender := sim.Factions[war.DefenderID]
		domain, okDomain := sim.Domains[war.DomainID]
		if !okAttacker || !okDefender || !okDomain {
			war.IsOver = true
			war.WinnersID = map[string]string{}
			war.LosersID = map[string]string{}
			gameEventBuilder := NewBuilderWarEvent()
			gameEventBuilder.SetType("WAR_ENDED").SetTick(sim.GlobalTick).SetData(map[string]any{
				"attacker": war.AttackerID,
				"defender": war.DefenderID,
				"domain":   war.DomainID,
				"reason":   "invalid_war_state",
			})
			sim.EventBus.Publish(gameEventBuilder.Build())
			continue
		}

		// Замороженные константы плотностей влияния
		frozenAttackerDensity := war.FrozenFactionsDenseties[attacker.ID]
		frozenDefenderDensity := war.FrozenFactionsDenseties[defender.ID]

		// Фактор разницы во влиянии
		influenceRatio := frozenAttackerDensity - frozenDefenderDensity

		// Текущие силы контингентов с учётом влияния на домене
		effectiveAttackerForce := war.AttackerCurrentForce * (1.0 + frozenAttackerDensity)
		effectiveDefenderForce := war.DefenderCurrentForce * (1.0 + frozenDefenderDensity)

		// Проверяем случай автоматической победы из-за нулевой силы контингента
		if war.DefenderCurrentForce <= 0 {
			war.IsOver = true
			war.WinnersID = map[string]string{attacker.ID: "defender_zero_force"}
			war.LosersID = map[string]string{defender.ID: "zero_force"}
			gameEventBuilder := NewBuilderWarEvent()
			gameEventBuilder.SetType("WAR_ENDED").SetTick(sim.GlobalTick).SetData(map[string]any{
				"attacker": attacker.Name,
				"defender": defender.Name,
				"domain":   domain.Name,
				"reason":   "defender_annihilated",
			})
			sim.EventBus.Publish(gameEventBuilder.Build())
			sim.FinishWar(war, attacker.ID, defender.ID, domain)
			continue
		}
		if war.AttackerCurrentForce <= 0 {
			war.IsOver = true
			war.WinnersID = map[string]string{defender.ID: "attacker_zero_force"}
			war.LosersID = map[string]string{attacker.ID: "zero_force"}
			gameEventBuilder := NewBuilderWarEvent()
			gameEventBuilder.SetType("WAR_ENDED").SetTick(sim.GlobalTick).SetData(map[string]any{
				"attacker": attacker.Name,
				"defender": defender.Name,
				"domain":   domain.Name,
				"reason":   "attacker_annihilated",
			})
			sim.EventBus.Publish(gameEventBuilder.Build())
			sim.FinishWar(war, defender.ID, attacker.ID, domain)
			continue
		}

		// ============ ЗАКОН ЛАНЧЕСТЕРА-ОСИПОВА (квадратичный) с затуханием ============

		// Функция морали: влияет на эффективность бойцов
		moraleFactor := func(morale float64) float64 {
			// Нелинейная зависимость: при морали 100 -> 1.0, при 50 -> 0.7, при 0 -> 0.3
			return 0.3 + 0.7*(morale/100.0)
		}

		// Функция истощения: чем меньше осталось войск от начального, тем менее эффективны
		// Это создаёт нелинейность: истощённые армии наносят меньше урона
		exhaustionFactor := func(current, initial float64) float64 {
			if initial <= 0 {
				return 0.3
			}
			ratio := current / initial
			// При 100% сил -> 1.0, при 50% -> 0.75, при 25% -> 0.5
			return 0.3 + 0.7*math.Sqrt(ratio)
		}

		// Штраф за многофронтовую войну
		attackerWars := sim.countActiveWars(attacker.ID)
		defenderWars := sim.countActiveWars(defender.ID)
		multiFrontPenalty := func(warsCount int) float64 {
			if warsCount <= 1 {
				return 1.0
			}
			return 1.0 / (1.0 + float64(warsCount-1)*0.25) // -25% за каждую дополнительную войну
		}

		// Штраф за опасность домена
		dangerModifier := 1.0 - float64(domain.DangerLevel)/200.0

		// Бонус за влияние на территории
		influenceBonus := 1.0 + influenceRatio*0.2

		// Коэффициенты эффективности (alpha для атакующего, beta для защитника)
		alphaBase := 0.008 * (1.0 + frozenAttackerDensity) *
			moraleFactor(war.AttackerMorale) *
			exhaustionFactor(war.AttackerCurrentForce, war.AttackerCommittedForce) *
			multiFrontPenalty(attackerWars) *
			dangerModifier *
			influenceBonus

		betaBase := 0.008 * (1.0 + frozenDefenderDensity) *
			moraleFactor(war.DefenderMorale) *
			exhaustionFactor(war.DefenderCurrentForce, war.DefenderCommittedForce) *
			multiFrontPenalty(defenderWars) *
			dangerModifier *
			1.15 / influenceBonus // защитник +15% за оборону

		// Стохастический элемент (±10%)
		alphaRandom := alphaBase * (0.9 + rand.Float64()*0.2)
		betaRandom := betaBase * (0.9 + rand.Float64()*0.2)

		// Потери по квадратичному закону Ланчестера
		// dA/dt = -beta * B, dB/dt = -alpha * A
		attackerLosses := betaRandom * effectiveDefenderForce
		defenderLosses := alphaRandom * effectiveAttackerForce

		// Применяем потери к контингентам
		war.AttackerCurrentForce = math.Max(0, war.AttackerCurrentForce-attackerLosses)
		war.DefenderCurrentForce = math.Max(0, war.DefenderCurrentForce-defenderLosses)

		// Расход ресурсов (пропорционален интенсивности боя)
		const resourceCostFactor = 0.03
		attackerResourceCost := resourceCostFactor * (1.0 + attackerLosses*0.5)
		defenderResourceCost := resourceCostFactor * (1.0 + defenderLosses*0.5) * 0.8

		attacker.Resources = clamp(attacker.Resources-attackerResourceCost, 0, 100)
		defender.Resources = clamp(defender.Resources-defenderResourceCost, 0, 100)

		// Инвариант Ланчестера для momentum
		lanchesterInvariant := alphaRandom*effectiveAttackerForce*effectiveAttackerForce -
			betaRandom*effectiveDefenderForce*effectiveDefenderForce

		momentumChange := lanchesterInvariant * 0.0005
		war.Momentum += momentumChange
		war.TicksDuration++
		war.LastUpdateTick = sim.GlobalTick

		// ============ РАСЧЁТ ПОТЕРЬ И МОРАЛИ ============

		// Процент потерь от начального контингента
		attackerLossPercent := 1.0 - war.AttackerCurrentForce/war.AttackerCommittedForce
		defenderLossPercent := 1.0 - war.DefenderCurrentForce/war.DefenderCommittedForce

		// Текущее соотношение сил
		currentForceRatio := 0.0
		if war.DefenderCurrentForce > 0 {
			currentForceRatio = war.AttackerCurrentForce / war.DefenderCurrentForce
		}

		// Изменение морали на основе потерь за этот тик
		lossRatio := 0.0
		if attackerLosses+defenderLosses > 0 {
			lossRatio = (defenderLosses - attackerLosses) / (attackerLosses + defenderLosses)
		}

		moraleRandomFactor := 0.95 + rand.Float64()*0.1

		// Базовое изменение морали
		baseMoraleChange := math.Abs(lossRatio) * 3.0 * moraleRandomFactor

		if lossRatio > 0 {
			// Защитник несёт больше потерь → мораль атакующего растёт, защитника падает
			war.AttackerMorale = clamp(war.AttackerMorale+baseMoraleChange, 0, 100)
			war.DefenderMorale = clamp(war.DefenderMorale-baseMoraleChange*1.3, 0, 100)
		} else {
			// Атакующий несёт больше потерь
			war.DefenderMorale = clamp(war.DefenderMorale+baseMoraleChange, 0, 100)
			war.AttackerMorale = clamp(war.AttackerMorale-baseMoraleChange*1.3, 0, 100)
		}

		// ============ УСЛОВИЯ ОТСТУПЛЕНИЯ И СДАЧИ ============

		const (
			retreatLossThreshold   = 0.50 // Отступление при 50% потерь контингента
			surrenderLossThreshold = 0.70 // Сдача при 70% потерь
			criticalForceRatio     = 0.33 // Критическое соотношение сил 1:3
		)

		// Проверка условий для защитника
		defenderShouldSurrender := defenderLossPercent >= surrenderLossThreshold ||
			war.DefenderMorale <= 10 ||
			war.DefenderCurrentForce <= 0 ||
			defender.Resources <= 5

		defenderShouldRetreat := defenderLossPercent >= retreatLossThreshold ||
			war.DefenderMorale <= 25 ||
			(currentForceRatio > 1.0/criticalForceRatio) // атакующий превосходит в 3+ раза

		// Проверка условий для атакующего
		attackerShouldRetreat := attackerLossPercent >= retreatLossThreshold ||
			war.AttackerMorale <= 20 ||
			war.AttackerCurrentForce <= 0 ||
			attacker.Resources <= 5 ||
			(currentForceRatio < criticalForceRatio) // защитник превосходит в 3+ раза

		// Обработка окончания войны
		if defenderShouldSurrender {
			war.IsOver = true
			war.WinnersID = map[string]string{attacker.ID: "defender_surrendered"}
			war.LosersID = map[string]string{defender.ID: "surrendered"}
			gameEventBuilder := NewBuilderWarEvent()
			gameEventBuilder.SetType("WAR_ENDED").SetTick(sim.GlobalTick).SetData(map[string]any{
				"attacker":              attacker.Name,
				"defender":              defender.Name,
				"domain":                domain.Name,
				"reason":                "defender_surrendered",
				"attacker_losses_pct":   attackerLossPercent * 100,
				"defender_losses_pct":   defenderLossPercent * 100,
				"attacker_morale":       war.AttackerMorale,
				"defender_morale":       war.DefenderMorale,
				"attacker_force_remain": war.AttackerCurrentForce,
				"defender_force_remain": war.DefenderCurrentForce,
			})
			sim.EventBus.Publish(gameEventBuilder.Build())
			sim.FinishWar(war, attacker.ID, defender.ID, domain)
			continue
		}

		if attackerShouldRetreat {
			war.IsOver = true
			war.WinnersID = map[string]string{defender.ID: "attacker_retreated"}
			war.LosersID = map[string]string{attacker.ID: "retreated"}
			gameEventBuilder := NewBuilderWarEvent()
			gameEventBuilder.SetType("WAR_ENDED").SetTick(sim.GlobalTick).SetData(map[string]any{
				"attacker":              attacker.Name,
				"defender":              defender.Name,
				"domain":                domain.Name,
				"reason":                "attacker_retreated",
				"attacker_losses_pct":   attackerLossPercent * 100,
				"defender_losses_pct":   defenderLossPercent * 100,
				"attacker_morale":       war.AttackerMorale,
				"defender_morale":       war.DefenderMorale,
				"attacker_force_remain": war.AttackerCurrentForce,
				"defender_force_remain": war.DefenderCurrentForce,
			})
			sim.EventBus.Publish(gameEventBuilder.Build())
			sim.FinishWar(war, defender.ID, attacker.ID, domain)
			continue
		}

		if defenderShouldRetreat && !defenderShouldSurrender {
			// Защитник может отступить, сохранив часть войск, но теряет домен
			war.IsOver = true
			war.WinnersID = map[string]string{attacker.ID: "defender_retreated"}
			war.LosersID = map[string]string{defender.ID: "strategic_retreat"}
			gameEventBuilder := NewBuilderWarEvent()
			gameEventBuilder.SetType("WAR_ENDED").SetTick(sim.GlobalTick).SetData(map[string]any{
				"attacker":              attacker.Name,
				"defender":              defender.Name,
				"domain":                domain.Name,
				"reason":                "defender_strategic_retreat",
				"attacker_losses_pct":   attackerLossPercent * 100,
				"defender_losses_pct":   defenderLossPercent * 100,
				"attacker_morale":       war.AttackerMorale,
				"defender_morale":       war.DefenderMorale,
				"attacker_force_remain": war.AttackerCurrentForce,
				"defender_force_remain": war.DefenderCurrentForce,
			})
			sim.EventBus.Publish(gameEventBuilder.Build())
			sim.FinishWar(war, attacker.ID, defender.ID, domain)
			continue
		}

		// Лог текущего состояния войны
		warLogBuilder := NewBuilderWarEvent()
		warLogBuilder.SetType("WAR_UPDATE").SetTick(sim.GlobalTick).SetData(map[string]any{
			"attacker":            attacker.Name,
			"defender":            defender.Name,
			"domain":              domain.Name,
			"momentum":            war.Momentum,
			"attacker_morale":     war.AttackerMorale,
			"defender_morale":     war.DefenderMorale,
			"attacker_force":      war.AttackerCurrentForce,
			"defender_force":      war.DefenderCurrentForce,
			"attacker_losses_pct": attackerLossPercent * 100,
			"defender_losses_pct": defenderLossPercent * 100,
			"force_ratio":         currentForceRatio,
		})
		sim.EventBus.Publish(warLogBuilder.Build())
	}
}

func (sim *WorldSimulator) FinishWar(war *WarState, winnerId, loserId string, domain *DomainState) {
	if winnerId != "" {
		domain.ControlledBy = winnerId
	}

	// Возвращаем выживших бойцов обратно в резервы фракций
	if attacker, ok := sim.Factions[war.AttackerID]; ok {
		attacker.MilitaryForce = clamp(attacker.MilitaryForce+war.AttackerCurrentForce, 0, 100)
	}
	if defender, ok := sim.Factions[war.DefenderID]; ok {
		defender.MilitaryForce = clamp(defender.MilitaryForce+war.DefenderCurrentForce, 0, 100)
	}

	for factionID := range domain.Influence {
		switch factionID {
		case winnerId:
			domain.Influence[factionID] = 0.9
		case loserId:
			domain.Influence[factionID] = clamp((domain.Influence[factionID]-0.2)*0.5, 0, 1)
		default:
			// Сторонние фракции получают небольшой прирост влияния
			domain.Influence[factionID] = clamp(domain.Influence[factionID]+0.05, 0, 1)
		}
	}
	war.IsOver = true
	war.WinnersID = map[string]string{winnerId: "victory"}
	war.LosersID = map[string]string{loserId: "defeat"}
}
