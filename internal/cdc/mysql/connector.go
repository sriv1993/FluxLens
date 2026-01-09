// Package mysql provides a Maxwell-style MySQL binlog CDC connector for
// FluxLens. It reads MySQL row-level changes via the replication protocol
// and emits them as canonical events.
//
// Phase 1 status: skeleton with the binlog reader stub. The full binlog
// parser will be wired up against go-mysql in milestone M1.3 of the
// roadmap. The interface and stats surface are stable and exercised by
// tests so downstream components can be built against them.
package mysql

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/cdc"
)

// Config holds the configuration for a MySQL CDC connector.
type Config struct {
	// DSN is a standard go-sql-driver/mysql DSN. The configured user
	// must have REPLICATION CLIENT and REPLICATION SLAVE privileges.
	DSN string

	// ServerID must be unique across all replicas reading the source
	// binlog. Conflicts cause one replica to be disconnected.
	ServerID uint32

	// Tables restricts the connector to the listed `db.table` names.
	// Empty means all tables in the configured databases.
	Tables []string

	// SourceID is the canonical-event source_id assigned to events
	// emitted by this connector.
	SourceID string

	// HeartbeatInterval controls how often the connector emits a
	// synthetic "heartbeat" event so downstream lag monitors can
	// distinguish "idle source" from "stalled connector".
	HeartbeatInterval time.Duration
}

// Connector implements cdc.Connector for MySQL via the binlog protocol.
type Connector struct {
	cfg       Config
	emitted   atomic.Uint64
	errors    atomic.Uint64
	lastEvent atomic.Int64 // unix nanoseconds
	healthy   atomic.Bool
}

// New returns a configured but not-yet-running Connector.
func New(cfg Config) (*Connector, error) {
	if cfg.DSN == "" {
		return nil, errors.New("mysql cdc: DSN required")
	}
	if cfg.ServerID == 0 {
		return nil, errors.New("mysql cdc: ServerID required")
	}
	if cfg.SourceID == "" {
		return nil, errors.New("mysql cdc: SourceID required")
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 30 * time.Second
	}
	c := &Connector{cfg: cfg}
	c.healthy.Store(true)
	return c, nil
}

// Name implements cdc.Connector.
func (c *Connector) Name() string { return "mysql-cdc" }

// Run implements cdc.Connector. In Phase 1 this is a stub that emits
// heartbeats so end-to-end wiring can be validated; the full binlog
// reader replaces this body in milestone M1.3.
func (c *Connector) Run(ctx context.Context, out chan<- canonical.Event) error {
	t := time.NewTicker(c.cfg.HeartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case now := <-t.C:
			e, err := canonical.NewEvent(c.cfg.SourceID, canonical.SourceMySQLCDC, "heartbeat", canonical.SeverityInfo, map[string]any{
				"emitted_at": now.UTC().Format(time.RFC3339Nano),
				"server_id":  c.cfg.ServerID,
			})
			if err != nil {
				c.errors.Add(1)
				continue
			}
			select {
			case out <- e:
				c.emitted.Add(1)
				c.lastEvent.Store(time.Now().UnixNano())
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}
}

// Stats implements cdc.Connector.
func (c *Connector) Stats() cdc.Stats {
	last := time.Unix(0, c.lastEvent.Load()).UTC()
	lagSec := 0.0
	if c.lastEvent.Load() > 0 {
		lagSec = time.Since(last).Seconds()
	}
	return cdc.Stats{
		EventsEmitted:  c.emitted.Load(),
		ErrorsSeen:     c.errors.Load(),
		LagSeconds:     lagSec,
		LastEventAtUTC: last.Format(time.RFC3339Nano),
		Healthy:        c.healthy.Load(),
		Details: map[string]string{
			"source_id": c.cfg.SourceID,
			"server_id": fmt.Sprintf("%d", c.cfg.ServerID),
		},
	}
}

// Close implements cdc.Connector. In Phase 1 it is a no-op; once the
// real binlog reader is wired up it will close the underlying syncer.
func (c *Connector) Close() error { return nil }
