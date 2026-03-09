package metrics

import (
	"HellgameProject/internal/engine"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type PrometheusMetricsCollector struct {
	ticksSimulated             prometheus.Counter
	activeWars                 prometheus.Gauge
	tickDuration               prometheus.Histogram
	kolmogorovDuration         prometheus.Histogram
	lotkaVolterraDuration      prometheus.Histogram
	diffusionAdvectionDuration prometheus.Histogram
}

// NewPrometheusMetricsCollector - конструктор для PrometheusMetricsCollector
func NewPrometheusMetricsCollector() *PrometheusMetricsCollector {
	return &PrometheusMetricsCollector{
		ticksSimulated: promauto.NewCounter(prometheus.CounterOpts{
			Name: "hellgame_ticks_simulated_total",
			Help: "Общее количество тиков, прошедших в симуляции",
		}),
		activeWars: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "hellgame_active_wars",
			Help: "Текущее количество активных войн в симуляции",
		}),
		tickDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "hellgame_tick_duration_seconds",
			Help:    "Время выполнения одного тика симуляции в секундах",
			Buckets: prometheus.DefBuckets,
		}),
		kolmogorovDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "hellgame_kolmogorov_duration_seconds",
			Help:    "Время выполнения расчёта уравнений Колмогорова-Пискунова для всех доменов в секундах",
			Buckets: []float64{0.000007, 0.00001, 0.00002, 0.00003, 0.00005, 0.00008}, // Точные корзины для быстрых операций
		}),
		lotkaVolterraDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "hellgame_lotka_volterra_duration_seconds",
			Help:    "Время выполнения расчёта уравнений Лотки-Вольтерры для всех фракций в секундах",
			Buckets: []float64{0.000007, 0.00001, 0.00002, 0.00003, 0.00005, 0.00008}, // Точные корзины для быстрых операций
		}),
		diffusionAdvectionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "hellgame_diffusion_advection_duration_seconds",
			Help:    "Время выполнения расчёта уравнений диифузии-адвекции (переток населения) для всех доменов в секундах",
			Buckets: []float64{0.000007, 0.00001, 0.00002, 0.00003, 0.00005, 0.00008}, // Точные корзины для быстрых операций
		}),
	}
}

// Реализация методов MetricsCollector для интерфейса PrometheusMetricsCollector
func (p *PrometheusMetricsCollector) AddTicksSimulated(count int) {
	p.ticksSimulated.Add(float64(count))
}

func (p *PrometheusMetricsCollector) SetActiveWars(count int) {
	p.activeWars.Set(float64(count))
}

func (p *PrometheusMetricsCollector) SetTickDuration(duration time.Duration) {
	p.tickDuration.Observe(duration.Seconds())
}

func (p *PrometheusMetricsCollector) SetKolmogorovDuration(duration time.Duration) {
	p.kolmogorovDuration.Observe(duration.Seconds())
}

func (p *PrometheusMetricsCollector) SetLotkaVolterraDuration(duration time.Duration) {
	p.lotkaVolterraDuration.Observe(duration.Seconds())
}

func (p *PrometheusMetricsCollector) SetDiffusionAdvectionDuration(duration time.Duration) {
	p.diffusionAdvectionDuration.Observe(duration.Seconds())
}

var _ engine.MetricsCollector = (*PrometheusMetricsCollector)(nil) // Проверка интерфейса
