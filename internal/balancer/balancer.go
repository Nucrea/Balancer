package balancer

import (
	"balancer/internal/balancer/backend"
	"balancer/internal/balancer/strategy"
	"balancer/internal/metrics"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
)

func NewBalancer(meter *metrics.Meter, logger zerolog.Logger, addrs ...string) (*Balancer, error) {
	requestsCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "requests",
		Help: "balancer requests counter",
	})
	latencyHist := prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "latency",
		Help:    "balancer requests latency",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 200},
	})
	errorsCounter := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "errors",
		Help: "balancer errors counter",
	})

	serviceRequestsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "service_requests",
		Help: "service requests counter",
	}, []string{"service"})
	serviceErrorsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "service_errors",
		Help: "service errors counter",
	}, []string{"service"})
	serviceLatencyHist := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "service_latency",
		Help:    "service requests latency",
		Buckets: []float64{1, 5, 10, 25, 50, 100, 200},
	}, []string{"service"})
	serviceAliveGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "service_alive",
		Help: "services status",
	}, []string{"service"})

	meter.Register(requestsCounter, errorsCounter, latencyHist,
		serviceRequestsCounter, serviceErrorsCounter, serviceAliveGauge, serviceLatencyHist)

	items := []backend.Item{}
	for _, addr := range addrs {
		bck := backend.NewHttpBackend(addr)
		items = append(items, backend.NewItem(
			addr, bck,
			serviceRequestsCounter.WithLabelValues(addr),
			serviceErrorsCounter.WithLabelValues(addr),
			serviceAliveGauge.WithLabelValues(addr),
			serviceLatencyHist.WithLabelValues(addr),
		))
	}
	for _, item := range items {
		item.SetStatus(backend.StatusAlive)
	}

	return &Balancer{
		items:           items,
		strategy:        strategy.NewRoundRobin(items),
		requestsCounter: requestsCounter,
		errorsCounter:   errorsCounter,
		latencyHist:     latencyHist,
		logger:          logger,
	}, nil
}

type Balancer struct {
	items []backend.Item

	strategy        strategy.Strategy
	logger          zerolog.Logger
	requestsCounter prometheus.Counter
	errorsCounter   prometheus.Counter
	latencyHist     prometheus.Histogram
}

func (b *Balancer) Routine(ctx context.Context) {
	b.logger.Info().Msg("started balancer routine")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	unhealty := []backend.Item{}

	for {
		select {
		case <-ctx.Done():
			b.logger.Info().Msg("stopped balancer routine")
			return
		case <-ticker.C:
		}

		unhealty = unhealty[:0]
		for _, item := range b.items {
			status, _ := item.Status()
			if status != backend.StatusUnalive {
				continue
			}

			ctx, _ := context.WithTimeout(ctx, 100*time.Millisecond)
			if err := item.Health(ctx); err != nil {
				unhealty = append(unhealty, item)
				continue
			}

			item.SetStatus(backend.StatusAlive)
			b.logger.Info().Msgf("now healthy (%s)", item.Id())
		}

		if len(unhealty) > 0 {
			sb := strings.Builder{}
			sb.WriteString("unhealthy nodes: [")
			for _, item := range unhealty {
				sb.WriteString(item.Id())
				sb.WriteRune(' ')
			}
			sb.WriteRune(']')
			b.logger.Warn().Msg(sb.String())
		}

		ticker.Reset(time.Second)
	}
}

func (b *Balancer) SetStrategy(strat strategy.Type) {
	switch strat {
	case strategy.TypeRoundRobin:
		b.strategy = strategy.NewRoundRobin(b.items)
	case strategy.TypeLeastConnections:
		b.strategy = strategy.NewLeastConnections(b.items)
	}
}

func (b *Balancer) Invoke(ctx context.Context, req backend.Request) (backend.Response, error) {
	b.requestsCounter.Inc()

	start := time.Now()
	defer func() {
		latency := time.Since(start)
		b.latencyHist.Observe(float64(latency.Milliseconds()))
	}()

	var (
		resp backend.Response
		err  error
	)

	retries := 2
	for range retries {
		item := b.strategy.Next()
		if item == nil {
			return backend.Response{}, fmt.Errorf("no backend")
		}

		status, _ := item.Status()
		if status == backend.StatusChecking {
			b.logger.Info().Msgf("try check inalive (%s)", item.Id())
		}

		resp, err := item.Invoke(ctx, req)
		if errors.Is(err, context.Canceled) {
			return backend.Response{}, context.Canceled
		}
		if err != nil {
			item.SetStatus(backend.StatusUnalive)
			b.logger.Warn().Msgf("%s not alive (%s)", item.Id(), err.Error())
			continue
		}

		if status == backend.StatusChecking {
			b.logger.Info().Msgf("now is alive (%s)", item.Id())
			item.SetStatus(backend.StatusAlive)
		}

		return resp, nil
	}

	b.errorsCounter.Inc()

	return resp, err
}
