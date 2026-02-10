package main

// === СТРУКТУРЫ ДЛЯ СОБЫТИЙ МИРА ===
// Метаданные события.
// Содержат общую информацию о событии, которая может быть полезна для отображения в UI или для логирования.
type WorldEventMeta struct {
	ID          string
	Location    string
	Title       string
	Description string
	Consequence string
	Factions    []string
}

// Создаём общий тип события, для которого не требуется отдельная категория
type GenericEventData struct {
	EventKind EventKind
	EventData map[string]any
	Meta      *WorldEventMeta
}

type WarEventData struct {
	Attacker              string  `json:"attacker,omitempty"`
	Defender              string  `json:"defender,omitempty"`
	Domain                string  `json:"domain,omitempty"`
	DomainID              string  `json:"domain_id,omitempty"`
	Reason                string  `json:"reason,omitempty"`
	ActualDefenderForce   float64 `json:"actual_defender_force,omitempty"`
	EstimateDefenderForce float64 `json:"estimate_defender_force,omitempty"`
	AttackerCommitted     float64 `json:"attacker_committed,omitempty"`
	DefenderCommitted     float64 `json:"defender_committed,omitempty"`
	AttackerRemaining     float64 `json:"attacker_remaining,omitempty"`
	DefenderRemaining     float64 `json:"defender_remaining,omitempty"`
	Ratio                 float64 `json:"ratio,omitempty"`
	MinStrengthRatio      float64 `json:"min_strength_ratio,omitempty"`
	AttackerLossesPct     float64 `json:"attacker_losses_pct,omitempty"`
	DefenderLossesPct     float64 `json:"defender_losses_pct,omitempty"`
	AttackerMorale        float64 `json:"attacker_morale,omitempty"`
	DefenderMorale        float64 `json:"defender_morale,omitempty"`
	AttackerForceRemain   float64 `json:"attacker_force_remain,omitempty"`
	DefenderForceRemain   float64 `json:"defender_force_remain,omitempty"`
	Momentum              float64 `json:"momentum,omitempty"`
	AttackerForce         float64 `json:"attacker_force,omitempty"`
	DefenderForce         float64 `json:"defender_force,omitempty"`
	ForceRatio            float64 `json:"force_ratio,omitempty"`
}

// Тип payload’а для событий мира. Он возвращает мета‑данные через EventMeta()
type WorldEventData struct {
	Meta WorldEventMeta
}

type WarStartData struct {
	Attacker              string  `json:"attacker,omitempty"`
	Defender              string  `json:"defender,omitempty"`
	Domain                string  `json:"domain,omitempty"`
	DomainID              string  `json:"domain_id,omitempty"`
	Reason                string  `json:"reason,omitempty"`
	ActualDefenderForce   float64 `json:"actual_defender_force,omitempty"`
	EstimateDefenderForce float64 `json:"estimate_defender_force,omitempty"`
	AttackerCommitted     float64 `json:"attacker_committed,omitempty"`
	DefenderCommitted     float64 `json:"defender_committed,omitempty"`
	Ratio                 float64 `json:"ratio,omitempty"`
	MinStrengthRatio      float64 `json:"min_strength_ratio,omitempty"`
}

type WarUpdateData struct {
	Attacker          string  `json:"attacker,omitempty"`
	Defender          string  `json:"defender,omitempty"`
	Domain            string  `json:"domain,omitempty"`
	Momentum          float64 `json:"momentum,omitempty"`
	AttackerMorale    float64 `json:"attacker_morale,omitempty"`
	DefenderMorale    float64 `json:"defender_morale,omitempty"`
	AttackerForce     float64 `json:"attacker_force,omitempty"`
	DefenderForce     float64 `json:"defender_force,omitempty"`
	AttackerLossesPct float64 `json:"attacker_losses_pct,omitempty"`
	DefenderLossesPct float64 `json:"defender_losses_pct,omitempty"`
	ForceRatio        float64 `json:"force_ratio,omitempty"`
}

// === ИНТЕРФЕЙСЫ ДЛЯ СОБЫТИЙ МИРА ===

// У нас может быть множество типов событий. Поэтому любой тип, который хочет считаться событием,
// должен реализовать интерфейс EventData. Это позволяет нам обрабатывать разные типы событий единообразно.
// Сам по себе интерфейс минималистичный - он содержит один метод Kind(),
// который возвращает EventKind(алиас), идентифицирующую тип события.
// Data - это внутренняя информация о событии, которая может быть полезна для логики игры, но не обязательно должна отображаться пользователю.
type EventData interface {
	Kind() EventKind
}

// Чтобы эмиттер мог узнать мета‑информацию (например, Location, Title, Description), ему нужна унифицированная
// точка доступа к этим данным. Поэтому мы вводим интерфейс WorldEventMetaProvider,
// который требует реализации метода EventMeta().
// Meta - информация, которая может быть отдана на внешний мир (например, для отображения в UI или в API).
type WorldEventMetaProvider interface {
	EventMeta() *WorldEventMeta
}

// Реализация интерфейса EventData для WorldEventData. Метод Kind() возвращает константное значение EventKindWorld
func (d WorldEventData) Kind() EventKind {
	return EventKindWorld
}

// Реализация интерфейса WorldEventMetaProvider для WorldEventData. Метод EventMeta() возвращает указатель на Meta
// Можно было бы и не использовать указатель, но так мы избегаем лишнего копирования данных при передаче мета‑информации
// и получаем возможность вернуть nil, если по каким‑то причинам мета‑информация недоступна
func (d WorldEventData) EventMeta() *WorldEventMeta {
	return &d.Meta
}

// === ИМПЛЕМЕНТАЦИЯ ИНТЕРФЕЙСОВ
// Имплементация внутренних данных для Generic-эвентов
func (d GenericEventData) Kind() EventKind {
	if d.EventKind != "" {
		return d.EventKind
	}
	return EventKindGeneric
}

// Имплементация метаданных для Generic-эвентов
func (d GenericEventData) EventMeta() *WorldEventMeta {
	return d.Meta
}

// Имплементация интерфейсов для WarEventData
// Внутренние данные для эвентов войны
func (d WarEventData) Kind() EventKind {
	return EventKindWar
}

// Метаданные для эвентов войны. В данном случае мы используем DomainID в качестве Location
func (d WarEventData) EventMeta() *WorldEventMeta {
	return &WorldEventMeta{
		Location: d.DomainID,
	}
}
func (d WarStartData) Kind() EventKind {
	return EventKindWar
}

func (d WarStartData) EventMeta() *WorldEventMeta {
	return &WorldEventMeta{
		Location: d.DomainID,
	}
}

func (d WarUpdateData) Kind() EventKind {
	return EventKindWar
}

func (d WarUpdateData) EventMeta() *WorldEventMeta {
	return &WorldEventMeta{
		Location: d.Domain,
	}
}
