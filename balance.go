package main

// Константы для KPP
const (
	KPPDiffusionBaseRate     = 0.002
	KPPDiffusionPowerFactor  = 0.005
	KPPGrowthBaseRate        = 0.005
	KPPGrowthTerritoryFactor = 0.095
	KPPMaxDiffusion          = 1.0
	KPPMaxGrowth             = 0.1
)

type KPPParameters struct {
	Diffusion float64
	Growth    float64
}

func NewKPPParameters(faction *FactionState) KPPParameters {
	return KPPParameters{
		Diffusion: minFloat(KPPMaxDiffusion, KPPDiffusionBaseRate+KPPDiffusionPowerFactor*faction.WealthIndex),
		Growth:    minFloat(KPPMaxGrowth, KPPGrowthBaseRate+KPPGrowthTerritoryFactor*(faction.Territory/5.0)),
	}
}

// Константы для баланса войны
const (
	// --- Условия начала войны ---
	MinAttackStrengthRatio = 0.65 // Минимальное соотношение сил атакующего к защитнику
	WarAttackerForceBuffer = 10.0 // Запас сверх оценённой силы защитника при расчёте контингента

	// --- Ресурсы ---
	WarResourceCostBase         = 0.03 // Базовый расход ресурсов за тик
	WarResourceLossFactor       = 0.5  // Множитель потерь в расходе ресурсов (атакующий)
	WarResourceDefenderDiscount = 0.8  // Скидка защитнику на расход ресурсов
	WarResourceRetreatThreshold = 5.0  // Порог ресурсов, при котором фракция отступает

	// --- Пороги отступления и капитуляции ---
	WarRetreatLossThreshold         = 0.50 // Отступление при потере 50% контингента
	WarSurrenderLossThreshold       = 0.70 // Капитуляция при потере 70% контингента
	WarCriticalForceRatio           = 0.33 // Критическое соотношение сил 1:3
	WarDefenderRetreatLossThreshold = 0.45 // Стратегический отход: чуть раньше, чем атакующий отступит

	// --- Пороги морали ---
	WarSurrenderMoraleThreshold       = 10.0 // Мораль защитника → капитуляция
	WarDefenderRetreatMoraleThreshold = 25.0 // Мораль защитника → отступление
	WarAttackerRetreatMoraleThreshold = 20.0 // Мораль атакующего → отступление

	// --- Изменение морали ---
	WarMoraleRandomMin    = 0.95 // Минимум стохастики морали
	WarMoraleRandomRange  = 0.10 // Диапазон стохастики морали
	WarMoraleChangeFactor = 3.0  // Базовый множитель изменения морали
	WarMoraleLoserPenalty = 1.3  // Штраф проигрывающей стороне

	// --- Закон Ланчестера ---
	WarLanchesterBaseCoeff   = 0.008 // Базовый коэффициент потерь
	WarLanchesterRandomMin   = 0.9   // Минимум стохастики потерь
	WarLanchesterRandomRange = 0.2   // Диапазон стохастики потерь
	WarDefenderHomeBonus     = 1.15  // Бонус обороны для защитника (+15%)
	WarInfluenceBonusFactor  = 0.2   // Бонус за влияние на домене
	WarMultiFrontPenalty     = 0.25  // Штраф за каждую дополнительную войну (-25%)

	// --- Функции морали и истощения (нелинейность) ---
	WarMoraleBaseFloor   = 0.3 // Минимум морального фактора (при морали 0)
	WarMoraleBaseCeiling = 0.7 // Диапазон от floor до 1.0
	WarExhaustionFloor   = 0.3 // Минимум фактора истощения
	WarExhaustionCeiling = 0.7 // Диапазон от floor до 1.0

	// --- Импульс войны ---
	WarMomentumScaleFactor = 0.0005 // Масштаб изменения импульса за тик
	WarMomentumNormFactor  = 200.0  // Нормировка импульса при расчёте victoryScore

	// --- Влияние после войны ---
	WarVictoryScoreWeightLosses   = 0.5 // Вес потерь в victoryScore
	WarVictoryScoreWeightMorale   = 0.3 // Вес морали в victoryScore
	WarVictoryScoreWeightMomentum = 0.2 // Вес импульса в victoryScore
	WarWinnerInfluenceGain        = 0.5 // Насколько сильно победа прибавляет влияние
	WarLoserInfluenceDrop         = 0.2 // Вычитание влияния проигравшего
	WarLoserInfluenceDecay        = 0.5 // Затухание остатка влияния проигравшего

	// --- Общие ограничения ---
	MaxMilitaryForce   = 100.0 // Максимальная военная сила фракции
	WarDangerLevelNorm = 200.0 // Нормировка DangerLevel в расчёте боевого штрафа
)

type WarOutcome int

const (
	WarOutcomeContinues WarOutcome = iota
	WarOutcomeDefenderSurrenders
	WarOutcomeAttackerRetreats
	WarOutcomeDefenderRetreats
)

// Константы для модели Конвекции-Диффузии-Реакции. Динамика популяции
const (
	// r: Базовый коэффициент роста (Reaction)
	PopBaseGrowthRate = 0.0005
	// K: Базовая емкость домена (Reaction)
	PopBaseCapacity = 20000.0
	// D: Коэффициент диффузии (Diffusion) - случайное блуждание
	PopDiffusionCoeff = 0.0001 // Уменьшено в 10 раз
	// \mu: Подвижность (Convection) - скорость направленного бегства по градиенту
	PopConvectionMobility = 0.0001 // Уменьшено в 25 раз
	// Веса для расчета потенциала домена (U)
	PopPotentialStabilityWeight = 1.0
	PopPotentialDangerWeight    = 10.0
)

// Константы для вычисления значимости домена для фракции
const (
	DomainImportanceSurvivalWeight   = 0.7 // Вес выживаемости в значимости домена
	DomainImportanceResourcesWeight  = 0.5 // Вес ресурсов в значимости домена
	DomainImportancePopulationWeight = 0.3 // Вес населения в значимости домена
)
