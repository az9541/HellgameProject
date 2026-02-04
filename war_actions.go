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

	// Базовые силы с учётом влияния на домене
	baseAttackerStrength := attacker.MilitaryForce * (1.0 + domain.Influence[attacker.ID])
	baseDefenderStrength := defender.MilitaryForce * (1.0 + domain.Influence[defender.ID])
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
			Momentum:       100,
			AttackerMorale: 100,
			DefenderMorale: 0,
			IsOver:         true,
			WinnersID:      map[string]string{attacker.ID: "auto_win_defender_zero_force"},
			LosersID:       map[string]string{defender.ID: "zero_force"},
		}
		sim.Wars[war.ID] = war
		sim.EventBus.Publish(GameEvent{
			Type: "WAR_ENDED",
			Tick: sim.GlobalTick,
			Data: map[string]any{
				"attacker": attacker.Name,
				"defender": defender.Name,
				"domain":   domain.Name,
				"reason":   "defender_zero_force",
			},
		})
		sim.FinishWar(war, attacker.ID, defender.ID, domain)
		return
	}
	// Проверка: атакующий должен иметь минимальное соотношение сил
	strengthRatio := baseAttackerStrength / baseDefenderStrength
	if strengthRatio < MinAttackStrengthRatio {
		// Атакующий слишком слаб - отказывается от атаки
		sim.EventBus.Publish(GameEvent{
			Type: "WAR_ABORTED",
			Tick: sim.GlobalTick,
			Data: map[string]any{
				"attacker": attacker.Name,
				"defender": defender.Name,
				"domain":   domain.Name,
				"reason":   "insufficient_strength",
				"ratio":    strengthRatio,
				"min":      MinAttackStrengthRatio,
			},
		})
		return
	}

	// Добавляем случайный фактор (10% вариация)
	randomFactor := 0.9 + rand.Float64()*0.2 // от 0.9 до 1.1
	attackerStrength := baseAttackerStrength * randomFactor
	defenderStrength := baseDefenderStrength * (0.9 + rand.Float64()*0.2)

	warID := fmt.Sprintf("war:%s:%s:%s:%d", domain.ID, attacker.ID, defender.ID, rand.Int())

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

		Momentum:       0,
		AttackerMorale: 100,
		DefenderMorale: 100,

		IsOver:    false,
		WinnersID: make(map[string]string),
		LosersID:  make(map[string]string),
	}

	sim.Wars[war.ID] = war

	sim.EventBus.Publish(GameEvent{
		Type: "WAR_STARTED",
		Tick: sim.GlobalTick,
		Data: map[string]any{
			"attacker": attacker.Name,
			"defender": defender.Name,
			"domain":   domain.Name,
			"a_str":    attackerStrength,
			"d_str":    defenderStrength,
			"ratio":    strengthRatio,
		},
	})
}

func (sim *WorldSimulator) UpdateWars() {
	for _, war := range sim.Wars {
		if war.IsOver {
			continue
		}
		// Собираем данные по учасникам
		attacker, okAttacker := sim.Factions[war.AttackerID]
		defender, okDefender := sim.Factions[war.DefenderID]
		domain, okDomain := sim.Domains[war.DomainID]
		if !okAttacker || !okDefender || !okDomain {
			war.IsOver = true
			war.WinnersID = map[string]string{}
			war.LosersID = map[string]string{}
			sim.EventBus.Publish(GameEvent{
				Type: "WAR_ENDED",
				Tick: sim.GlobalTick,
				Data: map[string]any{
					"attacker": war.AttackerID,
					"defender": war.DefenderID,
					"domain":   war.DomainID,
					"reason":   "invalid_war_state",
				},
			})
			continue
		}

		// Замороженные константы плотностей влияния
		frozenAttackerDensity := war.FrozenFactionsDenseties[attacker.ID]
		frozenDefenderDensity := war.FrozenFactionsDenseties[defender.ID]

		// Фактор разницы во влиянии
		influenceRatio := frozenAttackerDensity - frozenDefenderDensity

		// Фактор разницы сил
		baseAttackerStrength := attacker.MilitaryForce * (1.0 + frozenAttackerDensity)
		baseDefenderStrength := defender.MilitaryForce * (1.0 + frozenDefenderDensity)

		// Проверяем случай автоматической победы атакующего из-за нулевой силы защитника
		if baseDefenderStrength <= 0 {
			war.IsOver = true
			war.WinnersID = map[string]string{attacker.ID: "defender_zero_force"}
			war.LosersID = map[string]string{defender.ID: "zero_force"}
			sim.EventBus.Publish(GameEvent{
				Type: "WAR_ENDED",
				Tick: sim.GlobalTick,
				Data: map[string]any{
					"attacker": attacker.Name,
					"defender": defender.Name,
					"domain":   domain.Name,
					"reason":   "defender_zero_force",
				},
			})
			sim.FinishWar(war, attacker.ID, defender.ID, domain)
			continue
		}
		if baseAttackerStrength <= 0 {
			war.IsOver = true
			war.WinnersID = map[string]string{defender.ID: "attacker_zero_force"}
			war.LosersID = map[string]string{attacker.ID: "zero_force"}
			sim.EventBus.Publish(GameEvent{
				Type: "WAR_ENDED",
				Tick: sim.GlobalTick,
				Data: map[string]any{
					"attacker": attacker.Name,
					"defender": defender.Name,
					"domain":   domain.Name,
					"reason":   "attacker_zero_force",
				},
			})
			sim.FinishWar(war, defender.ID, attacker.ID, domain)
			continue
		}

		// ============ ЗАКОН ЛАНЧЕСТЕРА-ОСИПОВА (квадратичный) ============
		// Коэффициенты эффективности с учётом морали и влияния на домене
		// Мораль влияет на эффективность бойцов: низкая мораль = менее эффективный бой
		moraleFactor := func(morale float64) float64 {
			// Нелинейная зависимость: при морали 100 -> 1.0, при 50 -> 0.7, при 0 -> 0.3
			return 0.3 + 0.7*(morale/100.0)
		}

		// Штраф за опасность домена (влияет на обе стороны)
		dangerModifier := 1.0 - float64(domain.DangerLevel)/200.0 // от 0.95 до 0.5

		// Бонус/штраф за разницу влияния на домене (тактическое преимущество)
		// influenceRatio > 0 → атакующий знает территорию лучше
		influenceBonus := 1.0 + influenceRatio*0.2 // ±20% к эффективности

		// Коэффициенты эффективности (alpha для атакующего, beta для защитника)
		// Защитник имеет небольшое преимущество (+10%) за оборону
		alphaBase := 0.01 * (1.0 + frozenAttackerDensity) * moraleFactor(war.AttackerMorale) * dangerModifier * influenceBonus
		betaBase := 0.01 * (1.0 + frozenDefenderDensity) * moraleFactor(war.DefenderMorale) * dangerModifier * 1.1 / influenceBonus

		// Стохастический элемент (±10% вариация для каждой стороны)
		alphaRandom := alphaBase * (0.9 + rand.Float64()*0.2)
		betaRandom := betaBase * (0.9 + rand.Float64()*0.2)

		// Потери по квадратичному закону Ланчестера за один тик
		// dA/dt = -beta * B, dB/dt = -alpha * A
		attackerLosses := betaRandom * baseDefenderStrength
		defenderLosses := alphaRandom * baseAttackerStrength

		// Применяем потери к военной силе
		attacker.MilitaryForce = clamp(attacker.MilitaryForce-attackerLosses, 0, 100)
		defender.MilitaryForce = clamp(defender.MilitaryForce-defenderLosses, 0, 100)

		// Расход ресурсов (пропорционален интенсивности боя)
		const resourceCostFactor = 0.05
		attackerResourceCost := resourceCostFactor * (1.0 + attackerLosses)
		defenderResourceCost := resourceCostFactor * (1.0 + defenderLosses) * 0.8 // защитник тратит меньше ресурсов

		attacker.Resources = clamp(attacker.Resources-attackerResourceCost, 0, 100)
		defender.Resources = clamp(defender.Resources-defenderResourceCost, 0, 100)

		// Инвариант Ланчестера для определения momentum
		// L = alpha * A² - beta * B²
		// Если L > 0, преимущество у атакующего; если L < 0, у защитника
		lanchesterInvariant := alphaRandom*baseAttackerStrength*baseAttackerStrength -
			betaRandom*baseDefenderStrength*baseDefenderStrength

		// Нормализуем инвариант для momentum (стохастический элемент уже учтён в alpha/beta)
		momentumChange := lanchesterInvariant * 0.001 // масштабируем для разумных значений

		war.Momentum += momentumChange
		war.TicksDuration++
		war.LastUpdateTick = sim.GlobalTick

		// Обновление морали участников войны
		// Мораль зависит от соотношения потерь: больше потерь = падение морали
		lossRatio := 0.0
		if attackerLosses+defenderLosses > 0 {
			lossRatio = (defenderLosses - attackerLosses) / (attackerLosses + defenderLosses)
		}

		// Стохастический элемент в изменении морали (±5%)
		moraleRandomFactor := 0.95 + rand.Float64()*0.1

		// Базовое изменение морали
		baseMoraleChange := math.Abs(lossRatio) * 3.0 * moraleRandomFactor

		if lossRatio > 0 {
			// Защитник несёт больше потерь → мораль атакующего растёт, защитника падает
			war.AttackerMorale = clamp(war.AttackerMorale+baseMoraleChange, 0, 100)
			war.DefenderMorale = clamp(war.DefenderMorale-baseMoraleChange*1.2, 0, 100) // потери бьют по морали сильнее
		} else {
			// Атакующий несёт больше потерь
			war.DefenderMorale = clamp(war.DefenderMorale+baseMoraleChange, 0, 100)
			war.AttackerMorale = clamp(war.AttackerMorale-baseMoraleChange*1.2, 0, 100)
		}

		// Проверка условий окончания войны (Сдача защитником, отступление атакующего, истощение ресурсов)
		if war.DefenderMorale <= 0 || defender.MilitaryForce <= 0 || defender.Resources <= 0 {
			war.IsOver = true
			war.WinnersID = map[string]string{attacker.ID: "defender_defeated"}
			war.LosersID = map[string]string{defender.ID: "defeated_in_war"}
			sim.EventBus.Publish(GameEvent{
				Type: "WAR_ENDED",
				Tick: sim.GlobalTick,
				Data: map[string]any{
					"attacker":           attacker.Name,
					"defender":           defender.Name,
					"domain":             domain.Name,
					"reason":             "defender_defeated",
					"attacker_morale":    war.AttackerMorale,
					"defender_morale":    war.DefenderMorale,
					"attacker_force":     attacker.MilitaryForce,
					"defender_force":     defender.MilitaryForce,
					"attacker_resources": attacker.Resources,
					"defender_resources": defender.Resources,
				},
			})
			sim.FinishWar(war, attacker.ID, defender.ID, domain)
			continue
		}
		if war.AttackerMorale <= 0 || attacker.MilitaryForce <= 0 || attacker.Resources <= 0 {
			war.IsOver = true
			war.WinnersID = map[string]string{defender.ID: "attacker_retreat"}
			war.LosersID = map[string]string{attacker.ID: "retreated_from_war"}
			sim.EventBus.Publish(GameEvent{
				Type: "WAR_ENDED",
				Tick: sim.GlobalTick,
				Data: map[string]any{
					"attacker":           attacker.Name,
					"defender":           defender.Name,
					"domain":             domain.Name,
					"reason":             "attacker_retreat",
					"attacker_morale":    war.AttackerMorale,
					"defender_morale":    war.DefenderMorale,
					"attacker_force":     attacker.MilitaryForce,
					"defender_force":     defender.MilitaryForce,
					"attacker_resources": attacker.Resources,
					"defender_resources": defender.Resources,
				},
			})
			sim.FinishWar(war, defender.ID, attacker.ID, domain)
			continue
		}
		// Лог текущего состояния войны
		sim.EventBus.Publish(GameEvent{
			Type: "WAR_UPDATE",
			Tick: sim.GlobalTick,
			Data: map[string]any{
				"attacker":        attacker.Name,
				"defender":        defender.Name,
				"domain":          domain.Name,
				"momentum":        war.Momentum,
				"attacker_morale": war.AttackerMorale,
				"defender_morale": war.DefenderMorale,
				"attacker_force":  attacker.MilitaryForce,
				"defender_force":  defender.MilitaryForce,
				"attacker_res":    attacker.Resources,
				"defender_res":    defender.Resources,
			},
		})

	}

}

func (sim *WorldSimulator) FinishWar(war *WarState, winnerId, loserId string, domain *DomainState) {
	if winnerId != "" {
		domain.ControlledBy = winnerId
	}

	for factionID := range domain.Influence {
		if factionID == winnerId {
			domain.Influence[factionID] = 0.9
		} else if factionID == loserId {
			domain.Influence[factionID] = clamp((domain.Influence[factionID]-0.2)*0.5, 0, 1)
		} else {
			// Сторонние фракции получают небольшой прирост влияния
			domain.Influence[factionID] = clamp(domain.Influence[factionID]+0.05, 0, 1)
		}
	}
	war.IsOver = true
	war.WinnersID = map[string]string{winnerId: "victory"}
	war.LosersID = map[string]string{loserId: "defeat"}
}
