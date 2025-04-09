package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewPrometheus(prefix string) *Meter {
	registry := prometheus.NewRegistry()
	registerer := prometheus.WrapRegistererWithPrefix(prefix, registry)

	registerer.MustRegister(
	// collectors.NewGoCollector(),
	// collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	return &Meter{
		registry:   registry,
		registerer: registerer,
	}
}

type Meter struct {
	registry   *prometheus.Registry
	registerer prometheus.Registerer
}

func (p *Meter) Register(collectors ...prometheus.Collector) {
	for _, collector := range collectors {
		p.registerer.MustRegister(collector)
	}
}

func (m *Meter) HttpHandler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{Registry: m.registerer})
}
