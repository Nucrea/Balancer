package strategy

import (
	"balancer/internal/balancer/backend"
	"sync/atomic"
	"time"
)

func NewRoundRobin(items []backend.Item) Strategy {
	return &roundRobin{items, &atomic.Uint64{}}
}

type roundRobin struct {
	items []backend.Item
	index *atomic.Uint64
}

func (r *roundRobin) Next() backend.Item {
	for range r.items {
		index := r.index.Add(1) % uint64(len(r.items))
		item := r.items[index]

		status, updateTime := item.Status()
		if status == backend.StatusAlive {
			return item
		}
		if status == backend.StatusUnalive && time.Since(updateTime) > 5*time.Second {
			item.SetStatus(backend.StatusChecking)
			return item
		}
	}

	return nil
}
