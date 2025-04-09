package backend

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type Status int

const (
	StatusAlive = iota
	StatusUnalive
	StatusChecking
)

func NewItem(
	id string,
	backend Backend,
	requestsCounter prometheus.Counter,
	errorsCounter prometheus.Counter,
	aliveGauge prometheus.Gauge,
	latencyHist prometheus.Observer,
) Item {
	return &item{
		id:              id,
		mut:             &sync.RWMutex{},
		backend:         backend,
		aliveGauge:      aliveGauge,
		requestsCounter: requestsCounter,
		errorsCounter:   errorsCounter,
		latencyHist:     latencyHist,
	}
}

type item struct {
	id          string
	mut         *sync.RWMutex
	backend     Backend
	status      Status
	updateTime  time.Time
	connections int64

	requestsCounter prometheus.Counter
	errorsCounter   prometheus.Counter
	aliveGauge      prometheus.Gauge
	latencyHist     prometheus.Observer
}

func (b *item) Id() string {
	return b.id
}

func (b *item) SetStatus(status Status) {
	b.mut.Lock()
	defer b.mut.Unlock()

	if status == StatusAlive {
		b.aliveGauge.Set(1)
	} else {
		b.aliveGauge.Set(0)
	}

	b.status = status
	b.updateTime = time.Now()
}

func (b *item) Status() (Status, time.Time) {
	b.mut.RLock()
	defer b.mut.RUnlock()

	return b.status, b.updateTime
}

func (b *item) Connections() int {
	return int(b.connections)
}

func (b *item) Invoke(ctx context.Context, req Request) (Response, error) {
	b.requestsCounter.Inc()
	atomic.AddInt64(&b.connections, 1)

	start := time.Now()
	defer func() {
		latency := time.Since(start)
		b.latencyHist.Observe(float64(latency.Milliseconds()))
		atomic.AddInt64(&b.connections, -1)
	}()

	resp, err := b.backend.Invoke(ctx, req)
	if err != nil && err != context.Canceled {
		b.errorsCounter.Inc()
	}
	return resp, err
}
