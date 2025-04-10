package strategy

import (
	"balancer/internal/balancer/backend"
	"math"
	"sync/atomic"
	"time"
)

func NewLeastConnections(items []backend.Item) Strategy {
	return &leastConnections{items, &atomic.Uint64{}}
}

type leastConnections struct {
	items []backend.Item
	index *atomic.Uint64
}

func (l *leastConnections) Next() backend.Item {
	var (
		result   backend.Item
		minConns = math.MaxInt
	)

	for range l.items {
		index := l.index.Add(1) % uint64(len(l.items))
		item := l.items[index]

		status, updateTime := item.Status()
		if status == backend.StatusUnalive && time.Since(updateTime) > 5*time.Second {
			item.SetStatus(backend.StatusChecking)
			return item
		}
		if status == backend.StatusChecking || status == backend.StatusUnalive {
			continue
		}

		if conns := item.Connections(); conns < minConns {
			result = item
			minConns = conns
		}
	}

	return result
}
