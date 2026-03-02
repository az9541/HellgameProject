package engine

// ============ CONSTANTS ============

const (
	// Factions
	FactionCorporateConsortium = "corporate_consortium"
	FactionRepentantCommunes   = "repentant_communes"
	FactionNeoTormentors       = "neo_tormentors"
	FactionCaravanGuilds       = "caravan_guilds"
	FactionAncientRemnants     = "ancient_remnants"
	FactionNone                = "none"

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
	InfluenceToTakeOver              = 0.15 // Порог влияния фракции для попытки захвата домена
	MinInfluence                     = 0.0
	MinAwareness                     = 0.2
	InfluenceToAwarenessFactor       = 0.8
	TEMPDomainAttractivnessThreshold = 0.4
	BaseOwnDomainInfluence           = 0.4

	// Диффузия и KPP-уравнение
	DiffusionThreshold = 0.25 // Минимальное влияние для диффузии

)
