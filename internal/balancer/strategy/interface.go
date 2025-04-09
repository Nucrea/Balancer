package strategy

import "balancer/internal/balancer/backend"

type Type int

const (
	TypeRoundRobin = iota
	TypeLeastConnections
)

type Strategy interface {
	Next() backend.Item
}
