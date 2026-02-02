package main

import (
	"fmt"
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
