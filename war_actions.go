package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
)

// Возвращает активную войну за домен, если есть.
func (sim *WorldSimulator) getActiveWarForDomain(domainID string) *WarState {
	for _, war := range sim.State.Wars {
		if !war.IsOver && war.DomainID == domainID {
			return war
		}
	}
	return nil
}

// Функция запускает триггер войны
func (sim *WorldSimulator) StartWarTrigger(attacker, defender *FactionState, domain *DomainState) bool {
	domain, ok := sim.State.Domains[domain.ID]
	if !ok {
		return false
	}

	// Домен никто не контролирует - смысла воевать нет
	if domain.ControlledBy == FactionNone || domain.ControlledBy == "" {
		return false
	}

	if defender.ID == attacker.ID {
		return false
	}

	// Если война по домену уже идёт — ничего не делаем
	if sim.getActiveWarForDomain(domain.ID) != nil {
		return false
	}

	// Рассчитываем осведомлённость акатующего о силе защитника на домене
	attackerAwareness := awarenessFromInfluence(domain.Influence[attacker.ID])

	// Базовые силы с учётом влияния на домене

	estimateDefenderStrength := estimateForceWithAwareness(defender.MilitaryForce, attackerAwareness)

	// Проверяем отношение силы атакующего к оценённой силе защитника
	strengthRatio := 0.0
	if estimateDefenderStrength > 0 {
		strengthRatio = attacker.MilitaryForce / estimateDefenderStrength
	}

	// Считаем, сколько атакующий ХОЧЕТ выделить
	attackerWantedMilitaryPower := estimateDefenderStrength + WarAttackerForceBuffer // Ожидаемая сила защитника + небольшой запас для уверенности

	// Считаем, сколько атакующий МОЖЕТ выделить
	attackerCommitted := clamp(attackerWantedMilitaryPower*attacker.WealthIndex, 0, attacker.MilitaryForce)
	defenderCommitted := clamp(defender.MilitaryForce*defender.WealthIndex, 0, defender.MilitaryForce)
	log.Printf("faction wealth attacker: %f", attacker.WealthIndex)

	if defender.MilitaryForce <= 0 {
		// В auto-win ветке контингент тоже должен считаться выделенным,
		// иначе FinishWar вернёт бойцов, которых мы не списывали, и надует MilitaryForce.
		attacker.MilitaryForce = math.Max(0, attacker.MilitaryForce-attackerCommitted)

		warID := fmt.Sprintf("war:%s:%s:%s:%d", domain.ID, attacker.ID, defender.ID, rand.Int())
		war := &WarState{
			ID:             warID,
			AttackerID:     attacker.ID,
			DefenderID:     defender.ID,
			DomainID:       domain.ID,
			StartTick:      sim.State.GlobalTick,
			LastUpdateTick: sim.State.GlobalTick,
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
		sim.State.Wars[war.ID] = war
		gameEventBuillder := NewBuilderWarEvent()

		gameEventBuillder.SetType("WAR_STARTED").SetTick(sim.State.GlobalTick).SetData(WarStartData{
			Attacker:              attacker.ID,
			Defender:              defender.ID,
			Domain:                domain.Name,
			DomainID:              domain.ID,
			Reason:                "defender_zero_force",
			ActualDefenderForce:   defender.MilitaryForce,
			EstimateDefenderForce: estimateDefenderStrength,
			AttackerCommitted:     attackerCommitted,
			DefenderCommitted:     defenderCommitted,
			Ratio:                 strengthRatio,
			MinStrengthRatio:      MinAttackStrengthRatio,
		})

		sim.emitEventLocked(gameEventBuillder.Build())
		sim.FinishWar(war, attacker, defender, domain)
		return true
	}

	// Проверка: атакующий должен иметь минимальное соотношение сил
	if strengthRatio < MinAttackStrengthRatio {
		// Атакующий слишком слаб - отказывается от атаки
		gameEventBuilder := NewBuilderWarEvent()
		gameEventBuilder.SetType("WAR_ABORTED").SetTick(sim.State.GlobalTick).SetData(WarAbortedData{
			Attacker:              attacker.ID,
			Defender:              defender.ID,
			Domain:                domain.Name,
			DomainID:              domain.ID,
			Reason:                "insufficient_strength",
			ActualDefenderForce:   defender.MilitaryForce,
			EstimateDefenderForce: estimateDefenderStrength,
			AttackerCommitted:     attackerCommitted,
			DefenderCommitted:     defenderCommitted,
			Ratio:                 strengthRatio,
			MinStrengthRatio:      MinAttackStrengthRatio,
		})
		sim.emitEventLocked(gameEventBuilder.Build())
		return false
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

		StartTick:      sim.State.GlobalTick,
		LastUpdateTick: sim.State.GlobalTick,

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

	sim.State.Wars[war.ID] = war

	gameEventBuilder := NewBuilderWarEvent()

	gameEventBuilder.SetType("WAR_STARTED").SetTick(sim.State.GlobalTick).SetData(WarStartData{
		Attacker:              attacker.ID,
		Defender:              defender.ID,
		Domain:                domain.Name,
		DomainID:              domain.ID,
		Reason:                "acceptable_strength_ratio",
		ActualDefenderForce:   defender.MilitaryForce,
		EstimateDefenderForce: estimateDefenderStrength,
		AttackerCommitted:     attackerCommitted,
		DefenderCommitted:     defenderCommitted,
		Ratio:                 strengthRatio,
		MinStrengthRatio:      MinAttackStrengthRatio,
	})
	sim.emitEventLocked(gameEventBuilder.Build())
	return true
}

// Подсчёт активных войн фракции
func (sim *WorldSimulator) countActiveWars(factionID string) int {
	count := 0
	for _, war := range sim.State.Wars {
		if !war.IsOver && (war.AttackerID == factionID || war.DefenderID == factionID) {
			count++
		}
	}
	return count
}

func (sim *WorldSimulator) UpdateWars() {
	for _, war := range sim.State.Wars {
		if war.IsOver {
			continue
		}
		attacker, defender, domain, validWar := sim.validateWarParticipants(war)
		// Собираем данные по участникам
		if !validWar {
			gameEventBuilder := NewBuilderWarEvent()
			gameEventBuilder.SetType("WAR_ENDED").SetTick(sim.State.GlobalTick).SetData(WarEndedData{
				Attacker: war.AttackerID,
				Defender: war.DefenderID,
				Domain:   war.DomainID,
				Reason:   "invalid_war_state",
			})
			sim.emitEventLocked(gameEventBuilder.Build())
			continue
		}

		// Проверяем случай автоматической победы из-за нулевой силы контингента
		autoEndWar, reason, winner, loser := sim.checkAutoEndWar(war, attacker, defender, domain)
		if autoEndWar {
			warEventBuilder := sim.buildWarEndedEvent(war, attacker, defender, domain, reason,
				(1.0-war.AttackerCurrentForce/war.AttackerCommittedForce)*100,
				(1.0-war.DefenderCurrentForce/war.DefenderCommittedForce)*100,
				winner.ID, loser.ID)
			sim.emitEventLocked(warEventBuilder.Build())
			sim.FinishWar(war, winner, loser, domain)
			continue
		}

		// ============ ЗАКОН ЛАНЧЕСТЕРА-ОСИПОВА (квадратичный) с затуханием ============

		attackerLossPercent, defenderLossPercent, lossRatio := sim.applyBattleTick(war, attacker, defender, domain)

		// ============ Обновляем мораль ============
		currentForceRatio := 0.0
		if war.DefenderCurrentForce > 0 {
			currentForceRatio = war.AttackerCurrentForce / war.DefenderCurrentForce
		}
		updateMorale(war, lossRatio)

		// ============ УСЛОВИЯ ОТСТУПЛЕНИЯ И СДАЧИ ============
		warOutcome := evaluateWarOutcome(attacker, defender, war, attackerLossPercent, defenderLossPercent, currentForceRatio)

		// Обработка окончания войны
		sim.resolveWarOutcome(war, attacker, defender, domain, warOutcome, attackerLossPercent, defenderLossPercent, currentForceRatio)

		// Лог текущего состояния войны

	}
}

func (sim *WorldSimulator) FinishWar(war *WarState, winnerId, loserId *FactionState, domain *DomainState) {
	if winnerId != nil { // Передаём контроль над доменом победителю
		domain.ControlledBy = winnerId.ID
	}

	// ---- РАСЧЕТ ВРЕМЕННОГО ЭФФЕКТА ПОСЛЕ ВОЙНЫ ----
	var lossesRatio float64
	if winnerId.ID == war.AttackerID {
		lossesRatio = 1.0 - war.AttackerCurrentForce/war.AttackerCommittedForce
	} else {
		lossesRatio = 1.0 - war.DefenderCurrentForce/war.DefenderCommittedForce
	}
	// Чем выше потери и чем дольше война — тем сильнее дебафф на стабильность
	baseDuration := int64(30) + (war.TicksDuration / 2) // Минимум 30 тиков
	basePenalty := clamp(0.1+lossesRatio, 0.1, 0.9)     // От 10% до 90% порезки стабильности
	decayRate := 0.5
	// Индивидуальные особенности фракции-победителя
	switch winnerId.ID {
	case FactionNeoTormentors:
		basePenalty += 0.2 // Торменторы еще сильнее глушат
		decayRate = 0.8    // Но эффект спадает быстрее (очень резкий дебафф)
	case FactionCorporateConsortium:
		basePenalty -= 0.1 // Корпорации быстрее восстанавливают порядок
		decayRate = 0.2    // Долгий, но мелкий осадок
	case FactionRepentantCommunes:
		baseDuration += 10
	case FactionAncientRemnants:
		basePenalty += 0.1
		decayRate = 0.3
	case FactionCaravanGuilds:
		baseDuration += 5
		basePenalty -= 0.05
		decayRate = 0.4
	}

	basePenalty = clamp(basePenalty, 0.1, 1.0)

	// Синхронно добавляем в State эффекты
	newEffect := &DomainTimedEffect{
		DomainID:    domain.ID,
		FactionID:   winnerId.ID,
		StartTick:   sim.State.GlobalTick,
		Duration:    baseDuration,
		BasePenalty: basePenalty,
		DecayRate:   decayRate,
	}
	sim.State.TimedEffects[domain.ID] = append(sim.State.TimedEffects[domain.ID], newEffect)

	// Возвращаем выживших бойцов обратно в резервы фракций
	if attacker, ok := sim.State.Factions[war.AttackerID]; ok {
		attacker.MilitaryForce = clamp(attacker.MilitaryForce+war.AttackerCurrentForce, 0, MaxMilitaryForce)
	}
	if defender, ok := sim.State.Factions[war.DefenderID]; ok {
		defender.MilitaryForce = clamp(defender.MilitaryForce+war.DefenderCurrentForce, 0, MaxMilitaryForce)
	}

	for factionID := range domain.Influence {
		switch factionID {
		case winnerId.ID:
			momentumRatio := war.Momentum / WarMomentumNormFactor
			if winnerId.ID == war.AttackerID {
				lossesRatio := 1.0 - war.AttackerCurrentForce/war.AttackerCommittedForce
				moraleRatio := war.AttackerMorale / 100.0
				victoryScore := WarVictoryScoreWeightLosses*lossesRatio + WarVictoryScoreWeightMorale*moraleRatio + WarVictoryScoreWeightMomentum*momentumRatio
				domain.Influence[factionID] = clamp(domain.Influence[factionID]+victoryScore*WarWinnerInfluenceGain, 0, 1)
			} else {
				lossesRatio := 1.0 - war.DefenderCurrentForce/war.DefenderCommittedForce
				moraleRatio := war.DefenderMorale / 100.0
				victoryScore := WarVictoryScoreWeightLosses*lossesRatio + WarVictoryScoreWeightMorale*moraleRatio + WarVictoryScoreWeightMomentum*momentumRatio
				domain.Influence[factionID] = clamp(domain.Influence[factionID]+victoryScore*WarWinnerInfluenceGain, 0, 1)
			}
		case loserId.ID:
			domain.Influence[factionID] = clamp((domain.Influence[factionID]-WarLoserInfluenceDrop)*WarLoserInfluenceDecay, 0, 1)
		default:
			domain.Influence[factionID] = domain.Influence[factionID]
		}
	}
	sim.State.TimedEffects[domain.ID] = append(sim.State.TimedEffects[domain.ID], &DomainTimedEffect{
		DomainID:    domain.ID,
		FactionID:   winnerId.ID,
		StartTick:   sim.State.GlobalTick,
		Duration:    baseDuration,
		BasePenalty: basePenalty,
		DecayRate:   decayRate,
	})
	capDomainInfluence(domain.Influence)
	sim.syncFactionDomains()
}

func (sim *WorldSimulator) validateWarParticipants(war *WarState) (attacker *FactionState, defender *FactionState, domain *DomainState, ok bool) {
	attacker, okAttacker := sim.State.Factions[war.AttackerID]
	defender, okDefender := sim.State.Factions[war.DefenderID]
	domain, okDomain := sim.State.Domains[war.DomainID]
	okParticipants := okAttacker && okDefender && okDomain
	notNilParticipants := attacker != nil && defender != nil && domain != nil
	ok = okParticipants && notNilParticipants
	if !ok {
		war.IsOver = true
		war.WinnersID = map[string]string{}
		war.LosersID = map[string]string{}
		return nil, nil, nil, false
	}
	return attacker, defender, domain, true
}

func (sim *WorldSimulator) buildWarEndedEvent(war *WarState, attacker *FactionState, defender *FactionState,
	domain *DomainState, reason string, attackerLossPercent, defenderLossPercent float64, winnerID, loserID string) *BuilderWarEvent {
	warEventBuilder := NewBuilderWarEvent()
	warEventBuilder.SetType("WAR_ENDED").SetTick(sim.State.GlobalTick).SetData(WarEndedData{
		Attacker:            attacker.ID,
		Defender:            defender.ID,
		Domain:              domain.ID,
		Reason:              reason,
		WinnerID:            winnerID,
		LoserID:             loserID,
		AttackerLossesPct:   attackerLossPercent,
		DefenderLossesPct:   defenderLossPercent,
		AttackerMorale:      war.AttackerMorale,
		DefenderMorale:      war.DefenderMorale,
		AttackerForceRemain: war.AttackerCurrentForce,
		DefenderForceRemain: war.DefenderCurrentForce,
	})
	return warEventBuilder
}

func (sim *WorldSimulator) checkAutoEndWar(war *WarState, attacker *FactionState, defender *FactionState, domain *DomainState) (ended bool, reason string, winner, loser *FactionState) {
	if war.DefenderCurrentForce <= 0 {
		war.IsOver = true
		war.WinnersID = map[string]string{attacker.ID: "defender_zero_force"}
		war.LosersID = map[string]string{defender.ID: "zero_force"}
		return true, "defender_annihilated", attacker, defender
	}
	if war.AttackerCurrentForce <= 0 {
		war.IsOver = true
		war.WinnersID = map[string]string{defender.ID: "attacker_zero_force"}
		war.LosersID = map[string]string{attacker.ID: "zero_force"}
		return true, "attacker_annihilated", defender, attacker
	}
	return false, "", nil, nil
}

func (sim *WorldSimulator) computeBattleCoefficients(attacker *FactionState, defender *FactionState,
	domain *DomainState, war *WarState) (alpha, beta, lanchesterInvariant float64) {
	frozenAttackerDensity := war.FrozenFactionsDenseties[attacker.ID]
	frozenDefenderDensity := war.FrozenFactionsDenseties[defender.ID]

	// Фактор разницы во влиянии
	influenceRatio := frozenAttackerDensity - frozenDefenderDensity

	// Текущие силы контингентов с учётом влияния на домене
	effectiveAttackerForce := war.AttackerCurrentForce * (1.0 + frozenAttackerDensity)
	effectiveDefenderForce := war.DefenderCurrentForce * (1.0 + frozenDefenderDensity)
	// Функция морали: влияет на эффективность бойцов
	moraleFactor := func(morale float64) float64 {
		// Нелинейная зависимость: при морали 100 -> 1.0, при 50 -> 0.7, при 0 -> 0.3
		return WarMoraleBaseFloor + WarMoraleBaseCeiling*(morale/100.0)
	}

	// Функция истощения: чем меньше осталось войск от начального, тем менее эффективны
	// Это создаёт нелинейность: истощённые армии наносят меньше урона
	exhaustionFactor := func(current, initial float64) float64 {
		if initial <= 0 {
			return WarExhaustionFloor
		}
		ratio := current / initial
		// При 100% сил -> 1.0, при 50% -> 0.75, при 25% -> 0.5
		return WarExhaustionFloor + WarExhaustionCeiling*math.Sqrt(ratio)
	}

	// Штраф за многофронтовую войну
	attackerWars := sim.countActiveWars(attacker.ID)
	defenderWars := sim.countActiveWars(defender.ID)
	multiFrontPenalty := func(warsCount int) float64 {
		if warsCount <= 1 {
			return 1.0
		}
		return 1.0 / (1.0 + float64(warsCount-1)*WarMultiFrontPenalty) // -25% за каждую дополнительную войну
	}

	// Штраф за опасность домена
	dangerModifier := 1.0 - float64(domain.DangerLevel)/WarDangerLevelNorm

	// Бонус за влияние на территории
	influenceBonus := 1.0 + influenceRatio*WarInfluenceBonusFactor

	// Коэффициенты эффективности (alpha для атакующего, beta для защитника)
	alphaBase := WarLanchesterBaseCoeff * (1.0 + frozenAttackerDensity) *
		moraleFactor(war.AttackerMorale) *
		exhaustionFactor(war.AttackerCurrentForce, war.AttackerCommittedForce) *
		multiFrontPenalty(attackerWars) *
		dangerModifier *
		influenceBonus

	betaBase := WarLanchesterBaseCoeff * (1.0 + frozenDefenderDensity) *
		moraleFactor(war.DefenderMorale) *
		exhaustionFactor(war.DefenderCurrentForce, war.DefenderCommittedForce) *
		multiFrontPenalty(defenderWars) *
		dangerModifier *
		WarDefenderHomeBonus * influenceBonus // защитник +15% за оборону

	// Стохастический элемент (±10%)
	alphaRandom := alphaBase * (WarLanchesterRandomMin + rand.Float64()*WarLanchesterRandomRange)
	betaRandom := betaBase * (WarLanchesterRandomMin + rand.Float64()*WarLanchesterRandomRange)

	// Потери по квадратичному закону Ланчестера
	// dA/dt = -beta * B, dB/dt = -alpha * A
	alpha = alphaRandom * effectiveAttackerForce
	beta = betaRandom * effectiveDefenderForce
	lanchesterInvariant = alphaRandom*effectiveAttackerForce*effectiveAttackerForce -
		betaRandom*effectiveDefenderForce*effectiveDefenderForce
	return alpha, beta, lanchesterInvariant
}

func (sim *WorldSimulator) applyBattleTick(war *WarState, attacker, defender *FactionState, domain *DomainState) (attLossPercent, defLossPercent, lossRatio float64) {
	attackerLosses, defenderLosses, lanchesterInvariant := sim.computeBattleCoefficients(attacker, defender, domain, war)

	// Применяем потери к контингентам
	war.AttackerCurrentForce = math.Max(0, war.AttackerCurrentForce-attackerLosses)
	war.DefenderCurrentForce = math.Max(0, war.DefenderCurrentForce-defenderLosses)

	// Расход ресурсов (пропорционален интенсивности боя)
	attackerResourceCost := WarResourceCostBase * (1.0 + attackerLosses*WarResourceLossFactor)
	defenderResourceCost := WarResourceCostBase * (1.0 + defenderLosses*WarResourceLossFactor) * WarResourceDefenderDiscount

	attacker.Resources = clamp(attacker.Resources-attackerResourceCost, 0, 100)
	defender.Resources = clamp(defender.Resources-defenderResourceCost, 0, 100)

	momentumChange := lanchesterInvariant * WarMomentumScaleFactor
	war.Momentum += momentumChange
	war.TicksDuration++
	war.LastUpdateTick = sim.State.GlobalTick

	// Процент потерь от начального контингента
	attackerLossPercent := 1.0 - war.AttackerCurrentForce/war.AttackerCommittedForce
	defenderLossPercent := 1.0 - war.DefenderCurrentForce/war.DefenderCommittedForce

	if attackerLosses+defenderLosses > 0 {
		lossRatio = (defenderLosses - attackerLosses) / (attackerLosses + defenderLosses)
	}

	return attackerLossPercent, defenderLossPercent, lossRatio
}

func updateMorale(war *WarState, lossRatio float64) {

	moraleRandomFactor := WarMoraleRandomMin + rand.Float64()*WarMoraleRandomRange

	// Базовое изменение морали
	baseMoraleChange := math.Abs(lossRatio) * WarMoraleChangeFactor * moraleRandomFactor

	if lossRatio > 0 {
		// Защитник несёт больше потерь → мораль атакующего растёт, защитника падает
		war.AttackerMorale = clamp(war.AttackerMorale+baseMoraleChange, 0, 100)
		war.DefenderMorale = clamp(war.DefenderMorale-baseMoraleChange*WarMoraleLoserPenalty, 0, 100)
	} else {
		// Атакующий несёт больше потерь
		war.DefenderMorale = clamp(war.DefenderMorale+baseMoraleChange, 0, 100)
		war.AttackerMorale = clamp(war.AttackerMorale-baseMoraleChange*WarMoraleLoserPenalty, 0, 100)
	}

}

func evaluateWarOutcome(attacker, defender *FactionState, war *WarState, attackerLossPercent, defenderLossPercent, currentForceRatio float64) WarOutcome {

	defenderSurrenders := defenderLossPercent >= WarSurrenderLossThreshold ||
		war.DefenderMorale <= WarSurrenderMoraleThreshold ||
		war.DefenderCurrentForce <= 0 ||
		defender.Resources <= WarResourceRetreatThreshold

	if defenderSurrenders {
		return WarOutcomeDefenderSurrenders
	}

	attackerRetreats := attackerLossPercent >= WarRetreatLossThreshold ||
		war.AttackerMorale <= WarAttackerRetreatMoraleThreshold ||
		war.AttackerCurrentForce <= 0 ||
		attacker.Resources <= WarResourceRetreatThreshold ||
		currentForceRatio < WarCriticalForceRatio

	if attackerRetreats {
		return WarOutcomeAttackerRetreats
	}

	defenderRetreats := defenderLossPercent >= WarRetreatLossThreshold ||
		war.DefenderMorale <= WarDefenderRetreatMoraleThreshold ||
		currentForceRatio > 1.0/WarCriticalForceRatio

	if defenderRetreats {
		return WarOutcomeDefenderRetreats
	}

	return WarOutcomeContinues
}

func (sim *WorldSimulator) resolveWarOutcome(war *WarState, attacker, defender *FactionState,
	domain *DomainState, outcome WarOutcome,
	attackerLossPercent, defenderLossPercent float64, currentForceRatio float64) {
	switch outcome {
	case WarOutcomeDefenderSurrenders:
		war.IsOver = true
		war.WinnersID = map[string]string{attacker.ID: "defender_surrendered"}
		war.LosersID = map[string]string{defender.ID: "surrendered"}
		gameEventBuilder := NewBuilderWarEvent()
		gameEventBuilder.SetType("WAR_ENDED").SetTick(sim.State.GlobalTick).SetData(WarEndedData{
			Attacker:            attacker.ID,
			Defender:            defender.ID,
			Domain:              domain.ID,
			Reason:              "defender_surrendered",
			WinnerID:            attacker.ID,
			LoserID:             defender.ID,
			AttackerLossesPct:   attackerLossPercent * 100,
			DefenderLossesPct:   defenderLossPercent * 100,
			AttackerMorale:      war.AttackerMorale,
			DefenderMorale:      war.DefenderMorale,
			AttackerForceRemain: war.AttackerCurrentForce,
			DefenderForceRemain: war.DefenderCurrentForce,
		})
		sim.emitEventLocked(gameEventBuilder.Build())
		sim.FinishWar(war, attacker, defender, domain)

	case WarOutcomeAttackerRetreats:
		war.IsOver = true
		war.WinnersID = map[string]string{defender.ID: "attacker_retreated"}
		war.LosersID = map[string]string{attacker.ID: "retreated"}
		gameEventBuilder := NewBuilderWarEvent()
		gameEventBuilder.SetType("WAR_ENDED").SetTick(sim.State.GlobalTick).SetData(WarEndedData{
			Attacker:            attacker.ID,
			Defender:            defender.ID,
			Domain:              domain.ID,
			Reason:              "attacker_retreated",
			WinnerID:            defender.ID,
			LoserID:             attacker.ID,
			AttackerLossesPct:   attackerLossPercent * 100,
			DefenderLossesPct:   defenderLossPercent * 100,
			AttackerMorale:      war.AttackerMorale,
			DefenderMorale:      war.DefenderMorale,
			AttackerForceRemain: war.AttackerCurrentForce,
			DefenderForceRemain: war.DefenderCurrentForce,
		})
		sim.emitEventLocked(gameEventBuilder.Build())
		sim.FinishWar(war, defender, attacker, domain)

	case WarOutcomeDefenderRetreats:
		// Защитник может отступить, сохранив часть войск, но теряет домен
		war.IsOver = true
		war.WinnersID = map[string]string{attacker.ID: "defender_retreated"}
		war.LosersID = map[string]string{defender.ID: "strategic_retreat"}
		gameEventBuilder := NewBuilderWarEvent()
		gameEventBuilder.SetType("WAR_ENDED").SetTick(sim.State.GlobalTick).SetData(WarEndedData{
			Attacker:            attacker.ID,
			Defender:            defender.ID,
			Domain:              domain.ID,
			Reason:              "defender_strategic_retreat",
			WinnerID:            attacker.ID,
			LoserID:             defender.ID,
			AttackerLossesPct:   attackerLossPercent * 100,
			DefenderLossesPct:   defenderLossPercent * 100,
			AttackerMorale:      war.AttackerMorale,
			DefenderMorale:      war.DefenderMorale,
			AttackerForceRemain: war.AttackerCurrentForce,
			DefenderForceRemain: war.DefenderCurrentForce,
		})
		sim.emitEventLocked(gameEventBuilder.Build())
		sim.FinishWar(war, attacker, defender, domain)

	case WarOutcomeContinues:
		warLogBuilder := NewBuilderWarEvent()
		warLogBuilder.SetType("WAR_UPDATE").SetTick(sim.State.GlobalTick).SetData(WarUpdateData{
			Attacker:          attacker.ID,
			Defender:          defender.ID,
			Domain:            domain.ID,
			Momentum:          war.Momentum,
			AttackerMorale:    war.AttackerMorale,
			DefenderMorale:    war.DefenderMorale,
			AttackerForce:     war.AttackerCurrentForce,
			DefenderForce:     war.DefenderCurrentForce,
			AttackerLossesPct: attackerLossPercent * 100,
			DefenderLossesPct: defenderLossPercent * 100,
			ForceRatio:        currentForceRatio,
		})
		sim.emitEventLocked(warLogBuilder.Build())
	}

}
