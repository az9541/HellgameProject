package main

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// WorldSimulator управляет симуляцией всего мира
type WorldSimulator struct {
	// State
	Factions   map[string]*FactionState
	Domains    map[string]*DomainState
	GlobalTick int64

	// Event tracking
	EventLog []WorldEvent
	mu       sync.RWMutex

	// Channels for goroutines
	stop chan bool
}

// FactionState отслеживает состояние фракции
type FactionState struct {
	ID             string
	Name           string
	Power          float64            // 0-100
	Territory      float64            // total size
	DomainsHeld    []string           // domain IDs
	Attitude       map[string]float64 // towards other factions: -100 to +100
	Resources      float64            // wealth/supplies
	MilitaryForce  float64            // strength 0-100
	LastActionTime int64
}

// DomainState отслеживает состояние домена
type DomainState struct {
	ID           string
	Name         string
	Stability    float64 // 0-100
	ControlledBy string  // faction ID
	DangerLevel  int     // 1-10
	Population   int
	Mood         string   // "stable", "unrest", "rebellion"
	Events       []string // event IDs that happened here
	Influence    map[string]float64
}

// WorldEvent представляет событие в мире
type WorldEvent struct {
	ID          string
	Hour        int64
	Type        string // "faction_war", "trade_route", "rebellion", "discovery"
	Location    string // domain ID
	Title       string
	Description string
	Consequence string
	Factions    []string // involved factions
}

// SimulationDelta - результат симуляции
type SimulationDelta struct {
	TicksSimulated int64
	Events         []WorldEvent
	FactionStates  map[string]*FactionState
	DomainStates   map[string]*DomainState
	GlobalTick     int64
}

// NewWorldSimulator создаёт новый симулятор
func NewWorldSimulator() *WorldSimulator {
	sim := &WorldSimulator{
		Factions:   createInitialFactions(),
		Domains:    createInitialDomains(),
		GlobalTick: 0,
		EventLog:   []WorldEvent{},
		stop:       make(chan bool),
	}
	sim.initializeFactionInfluence()
	return sim
}

// Start запускает фоновые горутины симуляции
func (sim *WorldSimulator) Start() {
	log.Println("🚀 Starting world simulation goroutines...")

	go sim.runTimeLoop()

	log.Printf("✅ Simulation goroutines started")
}

// Stop останавливает симуляцию
func (sim *WorldSimulator) Stop() {
	sim.stop <- true
	log.Println("Simulation stopped")
}

// ============ ОСНОВНАЯ СИМУЛЯЦИЯ ============

// Simulate запускает симуляцию на N часов
func (sim *WorldSimulator) Simulate(ticks int64) *SimulationDelta {
	sim.mu.Lock()
	defer sim.mu.Unlock()

	startTime := sim.GlobalTick
	endTime := startTime + ticks

	// Отслеживаем события, проходящие во время симуляции
	newEvents := []WorldEvent{}

	for tick := startTime; tick < endTime; tick++ {
		sim.GlobalTick = tick

		// Каждый игровой час есть фиксированное значение вероятности наступления события
		if rand.Float64() < 0.3 { // 30% chance per tick
			event := sim.generateTickEvent(tick)
			if event != nil {
				sim.EventLog = append(sim.EventLog, *event)
				newEvents = append(newEvents, *event)
			}
		}

		sim.executeFactionActions()

		keys := getSortedDomainKeys(sim.Domains) // детерминированный порядок
		domainsSlice := getDomainsList(keys, sim.Domains)

		for _, faction := range sim.Factions {
			SimulateFactionExpansion(faction, domainsSlice, 1)
		}

		// Синхронизируем списки доменов у фракций
		sim.syncFactionDomains()

		sim.updateDomainStability()
	}

	// Return delta (only changes)
	delta := &SimulationDelta{
		TicksSimulated: ticks,
		Events:         newEvents,
		FactionStates:  sim.copyFactionStates(),
		DomainStates:   sim.copyDomainStates(),
		GlobalTick:     sim.GlobalTick,
	}

	log.Printf("SIMULATE_COMPLETE ticks=%d global_tick=%d new_events=%d", ticks, sim.GlobalTick, len(newEvents))
	return delta
}

func (sim *WorldSimulator) runTimeLoop() {
	// 1 тик = 5 реальных секунд
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sim.stop:
			return
		case <-ticker.C:
			sim.mu.Lock()

			// 1. Сначала считаем физику (влияние распространяется)
			sim.runKPPSimulation()

			// 2. Раз в 6 тиков (30 сек) фракции принимают решения
			if sim.GlobalTick%6 == 0 {
				sim.executeFactionActions()
			}

			// 3. Раз в 9 тиков (45 сек) происходят случайные события
			if sim.GlobalTick%9 == 0 {
				sim.generateTickEvent(sim.GlobalTick) // или вызов runEventGenerator логики
			}

			// 4. Раз в 12 тиков (60 сек) обновляем стабильность доменов
			if sim.GlobalTick%12 == 0 {
				sim.updateDomainStability()
			}

			// 5. И только в конце обновляем время
			sim.GlobalTick++

			sim.mu.Unlock()
		}
	}
}

// ============ ГОРУТИНЫ: ДЕЙСТВИЯ ФРАКЦИЙ ============

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

func (sim *WorldSimulator) attemptDomainTakeover(attacker *FactionState, domain *DomainState, influence float64) {
	// Find a domain not controlled by this faction
	baseProbability := (attacker.MilitaryForce / 100) * (1 - float64(domain.DangerLevel)/20)
	probability := baseProbability * (1.0 + influence)
	if probability >= 0.6 {
		sim.transferDomainControl(domain, attacker)
		log.Printf("EVENT=DOMAIN_TAKEOVER tick=%d attacker=%q domain=%q", sim.GlobalTick, attacker.Name, domain.Name)
	} else {
		log.Printf("EVENT=TAKEOVER_FAILED tick=%d attacker=%q domain=%q probability=%.4f", sim.GlobalTick, attacker.Name, domain.Name, probability)
	}
}

func (sim *WorldSimulator) resolveFactionWar(attacker, defender *FactionState, domain *DomainState) string {

	// Сравниваем силу враждующих
	attackerStrength := attacker.MilitaryForce * (1 + rand.Float64()*domain.Influence[attacker.ID]) // добавляем рандома
	defenderStrength := defender.MilitaryForce * (1 + rand.Float64()*domain.Influence[defender.ID])

	// Логируем сам факт начала конфликта
	log.Printf("EVENT=WAR_STARTED tick=%d attacker=%q defender=%q domain=%q a_str=%.1f d_str=%.1f",
		sim.GlobalTick, attacker.Name, defender.Name, domain.Name, attackerStrength, defenderStrength)

	if attackerStrength > defenderStrength {
		// Если атакующий победил
		// Передаём ему домен
		domain.ControlledBy = attacker.ID
		attacker.DomainsHeld = append(attacker.DomainsHeld, domain.ID)

		// Убираем домен от защищующегося
		newDomains := []string{}
		for _, d := range defender.DomainsHeld {
			if d != domain.ID {
				newDomains = append(newDomains, d)
			}
		}
		defender.DomainsHeld = newDomains

		// Adjust power
		attacker.Power += 8
		defender.Power -= 5

		// Последствия для домена
		domain.Stability = maxFloat(domain.Stability-15, 0)
		domain.Mood = "conquered"

		// Создаем событие для фронтенда/истории
		warEvent := WorldEvent{
			ID:          generateID(),
			Hour:        sim.GlobalTick,
			Type:        "faction_war",
			Location:    domain.ID,
			Title:       fmt.Sprintf("%s conquered %s", attacker.Name, domain.Name),
			Description: fmt.Sprintf("After a fierce battle, %s seized control from %s.", attacker.Name, defender.Name),
			Consequence: "Owner changed",
			Factions:    []string{attacker.ID, defender.ID},
		}
		sim.EventLog = append(sim.EventLog, warEvent)

		log.Printf("EVENT=WAR_RESULT tick=%d result=VICTORY attacker=%q domain=%q", sim.GlobalTick, attacker.Name, domain.Name)
		return "attacker_victory"

	} else {
		// --- ЗАЩИТНИК ОТБИЛСЯ ---

		defender.Power += 2
		attacker.Power -= 2

		log.Printf("EVENT=WAR_RESULT tick=%d result=DEFEAT attacker=%q domain=%q", sim.GlobalTick, attacker.Name, domain.Name)
		return "defender_victory"
	}
}

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

// ============ ГОРУТИНЫ: СИМУЛЯЦИЯ ДОМЕНОВ ============

func (sim *WorldSimulator) initializeFactionInfluence() {
	// Каждая фракция имеет минимальное влияние везде
	baseInfluence := 0.1 // 10% везде по умолчанию

	for _, faction := range sim.Factions {
		for _, domain := range sim.Domains {
			if domain.Influence == nil {
				domain.Influence = make(map[string]float64)
			}

			// Стартовое влияние: выше в своих доменах, ниже в чужих
			if domain.ControlledBy == faction.ID {
				domain.Influence[faction.ID] = 0.8 // 80% в своих
			} else {
				domain.Influence[faction.ID] = baseInfluence // 10% везде
			}
		}
	}
}

// runKPPSimulation выполняет один шаг KPP (одно обновление влияния фракций).
// Это не должен быть блокирующий цикл с тикером — шаг вызывается из `runTimeLoop()`.
func (sim *WorldSimulator) runKPPSimulation() {
	// Пересчитываем физику для каждой фракции один раз
	keys := getSortedDomainKeys(sim.Domains)
	domainsSlice := getDomainsList(keys, sim.Domains)

	for _, faction := range sim.Factions {
		SimulateFactionExpansion(faction, domainsSlice, 1)
	}
}

func (sim *WorldSimulator) updateDomainStability() {
	for _, domain := range sim.Domains {
		controller := sim.Factions[domain.ControlledBy]
		if controller == nil {
			domain.Stability = maxFloat(domain.Stability-2, 0) // Контроля нет - уходим в хаос
			continue
		}

		// Стабильность доменов в зависимости от того, кто их контроллирует
		if controller.ID == FactionCorporateConsortium {
			// Corporate = stable but exploitative
			domain.Stability = minFloat(domain.Stability+1, 80)
		} else if controller.ID == FactionRepentantCommunes {
			// Communes = moderate stability, good morale
			domain.Stability = minFloat(domain.Stability+2, 90)
		} else if controller.ID == FactionNeoTormentors {
			// Neo-Tormentors = oppressive but effective
			domain.Stability = minFloat(domain.Stability+0.5, 70)
		}

		// Danger level decreases with stability
		if domain.Stability > 70 {
			domain.DangerLevel = maxInt(domain.DangerLevel-1, 1)
		} else if domain.Stability < 30 {
			domain.DangerLevel = minInt(domain.DangerLevel+1, 10)
		}
	}
}

// ============ ГОРУТИНЫ: ГЕНЕРАЦИЯ СОБЫТИЙ ============

func (sim *WorldSimulator) generateTickEvent(tick int64) *WorldEvent {
	eventType := rand.Intn(5)

	switch eventType {
	case 0:
		return sim.generateMysteryEvent(tick)
	case 1:
		return sim.generateResourceEvent(tick)
	case 2:
		return sim.generateCulturalEvent(tick)
	case 3:
		return sim.generateDangerEvent(tick)
	default:
		return nil
	}
}

func (sim *WorldSimulator) generateMysteryEvent(tick int64) *WorldEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		domains = append(domains, d)
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	titles := []string{
		"Ancient entity stirs in the shadows",
		"A mysterious figure appears in the mist",
		"Strange markings discovered on ancient stones",
		"Whispers of something forgotten echo through the domain",
	}

	return &WorldEvent{
		ID:          generateID(),
		Hour:        tick,
		Type:        "mystery",
		Location:    domain.ID,
		Title:       titles[rand.Intn(len(titles))],
		Description: "Something ancient and unknown has stirred...",
		Consequence: "heresy_danger_level +2",
	}
}

func (sim *WorldSimulator) generateResourceEvent(tick int64) *WorldEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		if d.ControlledBy == FactionCorporateConsortium {
			domains = append(domains, d)
		}
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	return &WorldEvent{
		ID:          generateID(),
		Hour:        tick,
		Type:        "resource_discovery",
		Location:    domain.ID,
		Title:       "New mineral deposits discovered",
		Description: "Corporate teams have found rich deposits of infernal ore.",
		Consequence: "corporate_consortium power +3",
	}
}

func (sim *WorldSimulator) generateCulturalEvent(tick int64) *WorldEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		if d.ControlledBy == FactionRepentantCommunes {
			domains = append(domains, d)
		}
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	return &WorldEvent{
		ID:          generateID(),
		Hour:        tick,
		Type:        "cultural",
		Location:    domain.ID,
		Title:       "Community gathering brings hope",
		Description: "The communes organize a gathering to celebrate survival and mutual aid.",
		Consequence: domain.Name + " stability +5",
	}
}

func (sim *WorldSimulator) generateDangerEvent(tick int64) *WorldEvent {
	domains := make([]*DomainState, 0)
	for _, d := range sim.Domains {
		if d.DangerLevel > 5 {
			domains = append(domains, d)
		}
	}

	if len(domains) == 0 {
		return nil
	}

	domain := domains[rand.Intn(len(domains))]

	return &WorldEvent{
		ID:          generateID(),
		Hour:        tick,
		Type:        "danger",
		Location:    domain.ID,
		Title:       "Dangerous creature sighting",
		Description: "Reports of a dangerous entity roaming the domain.",
		Consequence: domain.Name + " danger_level +1",
	}
}

// ============ HELPERS ============

func SimulateFactionExpansion(faction *FactionState, domains []*DomainState, ticks int) {
	n := len(domains)
	if n == 0 || ticks <= 0 {
		return
	}

	neighbors := buildNeighborsFromDomains(domains)

	// Начальное распределение: 1.0 в доменах, контролируемых фракцией
	u := make([]float64, n)
	for i := 0; i < n; i++ {
		if domains[i].ControlledBy == faction.ID {
			u[i] = 1.0
		} else {
			u[i] = MinInfluence
		}
	}

	// Параметры модели (настройте по вкусу или добавьте поля в FactionState)
	D := minFloat(1.0, 0.2+0.8*(faction.Power/100.0))
	r := minFloat(0.2, 0.01+0.09*(faction.Territory/5.0))
	dt := 1.0 // один час на внешний шаг

	// Оценка стабильности явной схемы — число субшагов внутри часа
	maxDeg := 0
	for _, nb := range neighbors {
		if len(nb) > maxDeg {
			maxDeg = len(nb)
		}
	}
	substeps := 1
	if D > 0 && maxDeg > 0 {
		substeps = int(math.Ceil(dt * D * float64(maxDeg) * 2.0))
		if substeps < 1 {
			substeps = 1
		}
		if substeps > 1000 {
			substeps = 1000
		}
	}

	// Интегрируем
	for h := 0; h < ticks; h++ {
		prev := make([]float64, n)
		copy(prev, u)
		u = SolveKPGraph(u, neighbors, D, r, dt, substeps)
		// 1) лог плотностей компактно
		row := ""
		for i := 0; i < n; i++ {
			row += fmt.Sprintf("%.3f", u[i])
			if i < n-1 {
				row += ", "
			}
		}
		log.Printf("EXPANSION_DENSITIES faction=%q step=%d densities=[%s]", faction.ID, h+1, row)

		// 2) события захвата (пересечение порога)
		/*for i := 0; i < n; i++ {
			if prev[i] <= 0.5 && u[i] > 0.5 {
				fmt.Printf("  TAKEOVER: faction=%s domain=%s idx=%d new_density=%.3f", faction.ID, domains[i].ID, i, u[i])
			}
		}*/

		// 3) агрегаты: count, max, center of mass
		count := 0
		maxv := 0.0
		sum := 0.0
		weightSum := 0.0
		for i, v := range u {
			if v > 0.5 {
				count++
			}
			if v > maxv {
				maxv = v
			}
			sum += v
			weightSum += float64(i) * v
		}
		com := 0.0
		if sum > 0 {
			com = weightSum / sum
		} // центр масс по индексам
		log.Printf("EXPANSION_METRICS=faction=%q step=%d controlled=%d max=%.3f mean=%.3f com=%.2f", faction.ID, h+1, count, maxv, sum/float64(n), com)
	}

	// Применяем результат к реальным доменам
	for i, density := range u {
		domains[i].Influence[faction.ID] = density // В домене влияние фракции меняется на величину density
		/*
			if density > 0.5 {
				// TODO!!! Добавить не простой переход при превышении порога плотности, а реакцию других фракций
				domains[i].ControlledBy = faction.ID
			}
		*/
	}

	type pair struct {
		id      string
		density float64
	}
	pairs := make([]pair, 0, len(domains))

	for i, d := range domains {
		pairs = append(pairs, pair{id: d.ID, density: u[i]})
	}

	sort.Slice(pairs, func(i, j int) bool { return pairs[i].density > pairs[j].density })

	top := minInt(3, len(pairs))
	for i := 0; i < top; i++ {
		log.Printf("EXPANSION_METRICS=TOP_TAKEOVER_CANDIDATE faction=%q rank=%d domain=%q density=%.3f", faction.ID, i+1, pairs[i].id, pairs[i].density)
	}
	log.Printf("\n_____________________________________")
}

func (sim *WorldSimulator) copyFactionStates() map[string]*FactionState {
	copy := make(map[string]*FactionState)
	for id, faction := range sim.Factions {
		copy[id] = &FactionState{
			ID:            faction.ID,
			Name:          faction.Name,
			Power:         faction.Power,
			Territory:     faction.Territory,
			DomainsHeld:   append([]string{}, faction.DomainsHeld...),
			Attitude:      faction.Attitude,
			Resources:     faction.Resources,
			MilitaryForce: faction.MilitaryForce,
		}
	}
	return copy
}

func (sim *WorldSimulator) copyDomainStates() map[string]*DomainState {
	copy := make(map[string]*DomainState)
	for id, domain := range sim.Domains {
		copy[id] = &DomainState{
			ID:           domain.ID,
			Name:         domain.Name,
			Stability:    domain.Stability,
			ControlledBy: domain.ControlledBy,
			DangerLevel:  domain.DangerLevel,
			Population:   domain.Population,
			Mood:         domain.Mood,
		}
	}
	return copy
}

func createInitialFactions() map[string]*FactionState {
	return map[string]*FactionState{
		FactionCorporateConsortium: {
			ID:             FactionCorporateConsortium,
			Name:           "Corporate Consortium",
			Power:          70,
			Territory:      3.5,
			DomainsHeld:    []string{DomainLust, DomainGreed},
			Attitude:       make(map[string]float64),
			Resources:      80,
			MilitaryForce:  75,
			LastActionTime: 0,
		},
		FactionRepentantCommunes: {
			ID:             FactionRepentantCommunes,
			Name:           "Repentant Communes",
			Power:          50,
			Territory:      1.8,
			DomainsHeld:    []string{DomainGluttony},
			Attitude:       make(map[string]float64),
			Resources:      40,
			MilitaryForce:  35,
			LastActionTime: 0,
		},
		FactionNeoTormentors: {
			ID:             FactionNeoTormentors,
			Name:           "Neo-Tormentors",
			Power:          65,
			Territory:      2.5,
			DomainsHeld:    []string{DomainWrath, DomainViolence},
			Attitude:       make(map[string]float64),
			Resources:      70,
			MilitaryForce:  85,
			LastActionTime: 0,
		},
		FactionCaravanGuilds: {
			ID:             FactionCaravanGuilds,
			Name:           "Caravan Guilds",
			Power:          45,
			Territory:      0.9,
			DomainsHeld:    []string{DomainLimbo},
			Attitude:       make(map[string]float64),
			Resources:      60,
			MilitaryForce:  40,
			LastActionTime: 0,
		},
		FactionAncientRemnants: {
			ID:             FactionAncientRemnants,
			Name:           "Ancient Remnants",
			Power:          30,
			Territory:      0.3,
			DomainsHeld:    []string{DomainHeresy},
			Attitude:       make(map[string]float64),
			Resources:      50,
			MilitaryForce:  60,
			LastActionTime: 0,
		},
	}
}

// Стартовые условия в доменах. Стабильность, кому принадлежит, уровень опасности.
func createInitialDomains() map[string]*DomainState {
	return map[string]*DomainState{
		DomainLimbo: {
			ID:           DomainLimbo,
			Name:         "Limbo",
			Stability:    60,
			ControlledBy: FactionCaravanGuilds,
			DangerLevel:  3,
			Population:   5000,
			Mood:         "stable",
			Influence:    make(map[string]float64),
		},
		DomainLust: {
			ID:           DomainLust,
			Name:         "Circle of Lust",
			Stability:    40,
			ControlledBy: FactionCorporateConsortium,
			DangerLevel:  5,
			Population:   3000,
			Mood:         "exploited",
			Influence:    make(map[string]float64),
		},
		DomainGluttony: {
			ID:           DomainGluttony,
			Name:         "Circle of Gluttony",
			Stability:    50,
			ControlledBy: FactionRepentantCommunes,
			DangerLevel:  4,
			Population:   2500,
			Mood:         "hopeful",
			Influence:    make(map[string]float64),
		},
		DomainGreed: {
			ID:           DomainGreed,
			Name:         "Circle of Greed",
			Stability:    35,
			ControlledBy: FactionCorporateConsortium,
			DangerLevel:  6,
			Population:   4000,
			Mood:         "unrest",
			Influence:    make(map[string]float64),
		},
		DomainWrath: {
			ID:           DomainWrath,
			Name:         "Circle of Wrath",
			Stability:    20,
			ControlledBy: FactionNeoTormentors,
			DangerLevel:  9,
			Population:   6000,
			Mood:         "terrified",
			Influence:    make(map[string]float64),
		},
		DomainHeresy: {
			ID:           DomainHeresy,
			Name:         "Circle of Heresy",
			Stability:    45,
			ControlledBy: FactionAncientRemnants,
			DangerLevel:  7,
			Population:   1000,
			Mood:         "mysterious",
			Influence:    make(map[string]float64),
		},
		DomainViolence: {
			ID:           DomainViolence,
			Name:         "Circle of Violence",
			Stability:    15,
			ControlledBy: FactionNeoTormentors,
			DangerLevel:  10,
			Population:   8000,
			Mood:         "chaotic",
			Influence:    make(map[string]float64),
		},
		DomainFraud: {
			ID:           DomainFraud,
			Name:         "Circle of Fraud",
			Stability:    30,
			ControlledBy: "none",
			DangerLevel:  8,
			Population:   2000,
			Mood:         "deceptive",
			Influence:    make(map[string]float64),
		},
		DomainBetrayance: {
			ID:           DomainBetrayance,
			Name:         "Ninth Circle",
			Stability:    10,
			ControlledBy: "none",
			DangerLevel:  10,
			Population:   500,
			Mood:         "despairing",
			Influence:    make(map[string]float64),
		},
	}
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Тупо для логов, никакой сакральной ценности не несёт
func generateID() string {
	chars := "abcdefghijklmnopqrstuvwxyz"
	id := "evt_"
	for i := 0; i < 3; i++ {
		id += string(chars[rand.Intn(len(chars))])
	}
	return id
}

// syncFactionDomains перестраивает DomainsHeld у всех фракций на основе current ControlledBy
func (sim *WorldSimulator) syncFactionDomains() {
	// очистить списки
	for _, f := range sim.Factions {
		f.DomainsHeld = f.DomainsHeld[:0]
	}
	// заполнить заново
	for _, d := range sim.Domains {
		if f := sim.Factions[d.ControlledBy]; f != nil {
			f.DomainsHeld = append(f.DomainsHeld, d.ID)
		}
	}
}

func (sim *WorldSimulator) transferDomainControl(domain *DomainState, newOwner *FactionState) {
	oldOwner := sim.Factions[domain.ControlledBy]

	if newOwner != nil && oldOwner != nil && oldOwner.ID == newOwner.ID {
		return // ничего не менять
	}

	if oldOwner != nil {
		oldOwner.removeDomain(domain.ID)
	}

	if newOwner == nil {
		domain.ControlledBy = "none"
		return
	}

	domain.ControlledBy = newOwner.ID
	newOwner.addDomain(domain.ID)
}

func (faction *FactionState) addDomain(id string) {
	if faction.hasDomain(id) {
		return
	}
	faction.DomainsHeld = append(faction.DomainsHeld, id)
}

func (faction *FactionState) removeDomain(id string) {
	out := faction.DomainsHeld[:0]
	for _, d := range faction.DomainsHeld {
		if d != id {
			out = append(out, d)
		}
	}
	faction.DomainsHeld = out
}

func (faction *FactionState) hasDomain(id string) bool {
	for _, d := range faction.DomainsHeld {
		if d == id {
			return true
		}
	}
	return false
}

func getSortedDomainKeys(domains map[string]*DomainState) []string {
	keys := make([]string, 0, len(domains))
	for k := range domains {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func getDomainsList(keys []string, domains map[string]*DomainState) []*DomainState {
	domainsSlice := make([]*DomainState, 0, len(keys))
	for _, k := range keys {
		domainsSlice = append(domainsSlice, domains[k])
	}
	return domainsSlice
}
