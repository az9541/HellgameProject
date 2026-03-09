package engine

import (
	"time"
)

// MetricsCollector - интерфейс для сбора метрик симуляции
type MetricsCollector interface {
	// Количество тиков, прошедших в симуляции
	AddTicksSimulated(count int)
	// Фиксация текущих активных войн
	SetActiveWars(count int)

	// Время выполнения одного тика симуляции
	SetTickDuration(duration time.Duration)
	// Время выполнения расчёта уравнений Колмогорова-Пискунова для всех доменов
	SetKolmogorovDuration(duration time.Duration)
	// Время выполнения расчёта уравнений Лотки-Вольтерры для всех фракций
	SetLotkaVolterraDuration(duration time.Duration)
	// Время выполнения расчёта уравнений диифузии-адвекции (переток населения) для всех доменов
	SetDiffusionAdvectionDuration(duration time.Duration)
}

// NoopMetricsCollector - реализация MetricsCollector, которая ничего не делает (для отключения метрик)
type NoopMetricsCollector struct{}

func (n *NoopMetricsCollector) AddTicksSimulated(count int)                          {}
func (n *NoopMetricsCollector) SetActiveWars(count int)                              {}
func (n *NoopMetricsCollector) SetTickDuration(duration time.Duration)               {}
func (n *NoopMetricsCollector) SetKolmogorovDuration(duration time.Duration)         {}
func (n *NoopMetricsCollector) SetLotkaVolterraDuration(duration time.Duration)      {}
func (n *NoopMetricsCollector) SetDiffusionAdvectionDuration(duration time.Duration) {}

func MeasureTime(recorder func(duration time.Duration)) func() {
	start := time.Now()
	return func() { recorder(time.Since(start)) }
}
