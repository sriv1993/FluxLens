// Package cdc defines the interface that all FluxLens Change Data Capture
// connectors implement. Concrete connectors live in sub-packages
// (cdc/mysql, cdc/postgres). The interface is intentionally small so that
// new source types can be added without touching the rest of the platform.
package cdc

import (
	"context"

	"github.com/sriharshav1/fluxlens/internal/canonical"
)

// Connector reads change events from a source system and emits them as
// canonical events on the returned channel.
//
// Connectors are responsible for:
//   - tracking their own position in the source system (e.g., binlog
//     offset for MySQL, replication slot for Postgres)
//   - implementing at-least-once delivery semantics
//   - exposing health, lag, and throughput metrics via Stats()
//   - returning from Run when ctx is canceled (graceful shutdown)
//
// Connectors do not write to Kafka directly; the caller is responsible
// for fanning out the event channel.
type Connector interface {
	// Name returns a stable identifier (e.g., "mysql-cdc").
	Name() string

	// Run starts the connector. It blocks until ctx is canceled or a
	// fatal error occurs.
	Run(ctx context.Context, out chan<- canonical.Event) error

	// Stats returns operational statistics for monitoring.
	Stats() Stats

	// Close releases any resources held by the connector.
	Close() error
}

// Stats reports operational characteristics of a connector.
type Stats struct {
	EventsEmitted  uint64
	ErrorsSeen     uint64
	LagSeconds     float64
	LastEventAtUTC string
	Healthy        bool
	Details        map[string]string
}
