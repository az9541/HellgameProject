package main

// ============ CONSTANTS ============

const (
	// Factions
	FactionCorporateConsortium = "corporate_consortium"
	FactionRepentantCommunes   = "repentant_communes"
	FactionNeoTormentors       = "neo_tormentors"
	FactionCaravanGuilds       = "caravan_guilds"
	FactionAncientRemnants     = "ancient_remnants"

	// Domains (9 Circles of Hell)
	DomainLimbo      = "limbo"
	DomainLust       = "lust"
	DomainGluttony   = "gluttony"
	DomainGreed      = "greed"
	DomainWrath      = "wrath"
	DomainHeresy     = "heresy"
	DomainViolence   = "violence"
	DomainFraud      = "fraud"
	DomainBetrayance = "betrayance"

	// Event types
	EventTypeWar               = "faction_war"
	EventTypeTradeRoute        = "trade_route"
	EventTypeRebellion         = "rebellion"
	EventTypeDiscovery         = "discovery"
	EventTypeMystery           = "mystery"
	EventTypeResourceDiscovery = "resource_discovery"
	EventTypeCultural          = "cultural"
	EventTypeDanger            = "danger"

	// Numerical constants
	InfluenceToTakeOver = 0.1 // Порог влияния фракции для попытки захвата домена
	MinInfluence        = 0.01

	// War constants
	MinAttackStrengthRatio = 0.65  // Минимальное соотношение силы атакующего к защитнику (65%)
	WarResourceCost        = 5.0   // Стоимость войны в ресурсах
	BasePowerGain          = 10.0  // Базовое изменение power при победе
	BasePowerLoss          = 8.0   // Базовое изменение power при поражении
	MaxMilitaryForce       = 100.0 // Максимальная военная сила фракции
	// Константы длительной войны
	DefenderSurrenderThreshold = 20.0
	AttackerRetreatThreshold   = 15.0
	MoraleChangeFactor         = 0.5
)
