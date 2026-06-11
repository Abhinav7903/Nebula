package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SearchesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "nebula_searches_total",
		Help: "Total searches processed",
	})
	SearchesActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "nebula_searches_active",
		Help: "Active searches",
	})
	QueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "nebula_queue_depth",
		Help: "Current queue depth",
	})
	WorkersBusy = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "nebula_workers_busy",
		Help: "Busy workers",
	})
	CollectorDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "nebula_collector_duration_seconds",
		Help:    "Collector duration in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"collector"})
	CollectorResults = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nebula_collector_results_total",
		Help: "Results per collector",
	}, []string{"collector"})
	AISummaryDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "nebula_ai_summary_duration_seconds",
		Help:    "AI summary duration",
		Buckets: prometheus.DefBuckets,
	})
	ErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "nebula_errors_total",
		Help: "Errors per collector",
	}, []string{"collector"})
)
