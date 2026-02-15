// Package mysql provides a Maxwell-style MySQL binlog CDC connector for
// FluxLens. It reads MySQL row-level changes via the replication protocol
// and emits them as canonical events.
package mysql

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"

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

	// BinlogFile and BinlogPos optionally set the replication start
	// position. When BinlogFile is empty the connector starts at the
	// current master position.
	BinlogFile string
	BinlogPos  uint32
}

// Connector implements cdc.Connector for MySQL via the binlog protocol.
type Connector struct {
	cfg       Config
	filter    tableFilter
	emitted   atomic.Uint64
	errors    atomic.Uint64
	lastEvent atomic.Int64 // unix nanoseconds
	healthy   atomic.Bool

	mu         sync.Mutex
	syncer     *replication.BinlogSyncer
	binlogFile string
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
	c := &Connector{
		cfg:    cfg,
		filter: newTableFilter(cfg.Tables),
	}
	c.healthy.Store(true)
	return c, nil
}

// Name implements cdc.Connector.
func (c *Connector) Name() string { return "mysql-cdc" }

// Run implements cdc.Connector.
func (c *Connector) Run(ctx context.Context, out chan<- canonical.Event) error {
	info, err := parseDSN(c.cfg.DSN)
	if err != nil {
		c.healthy.Store(false)
		return err
	}

	syncerCfg := replication.BinlogSyncerConfig{
		ServerID:        c.cfg.ServerID,
		Flavor:          "mysql",
		Host:            info.Host,
		Port:            info.Port,
		User:            info.User,
		Password:        info.Password,
		HeartbeatPeriod: time.Second,
		ReadTimeout:     30 * time.Second,
	}
	syncer := replication.NewBinlogSyncer(syncerCfg)
	c.mu.Lock()
	c.syncer = syncer
	c.mu.Unlock()

	var pos mysql.Position
	if c.cfg.BinlogFile != "" {
		pos = mysql.Position{Name: c.cfg.BinlogFile, Pos: c.cfg.BinlogPos}
	} else {
		pos, err = masterPosition(c.cfg.DSN)
		if err != nil {
			c.healthy.Store(false)
			return fmt.Errorf("mysql cdc: get master position: %w", err)
		}
	}
	c.binlogFile = pos.Name

	streamer, err := syncer.StartSync(pos)
	if err != nil {
		c.healthy.Store(false)
		return fmt.Errorf("mysql cdc: start sync: %w", err)
	}

	hbCtx, hbCancel := context.WithCancel(ctx)
	defer hbCancel()
	go c.runHeartbeats(hbCtx, out)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		ev, err := streamer.GetEvent(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			c.errors.Add(1)
			c.healthy.Store(false)
			continue
		}
		c.healthy.Store(true)

		switch ev.Header.EventType {
		case replication.ROTATE_EVENT:
			if rotate, ok := ev.Event.(*replication.RotateEvent); ok {
				c.binlogFile = string(rotate.NextLogName)
			}
			continue
		case replication.WRITE_ROWS_EVENTv0, replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2,
			replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2,
			replication.DELETE_ROWS_EVENTv0, replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
			rows, ok := ev.Event.(*replication.RowsEvent)
			if !ok {
				continue
			}
			if !c.filter.allow(string(rows.Table.Schema), string(rows.Table.Table)) {
				continue
			}
			action := actionFromEventType(ev.Header.EventType)
			events, err := mapRowsEvent(c.cfg.SourceID, ev, rows, action, c.binlogFile)
			if err != nil {
				c.errors.Add(1)
				continue
			}
			for _, e := range events {
				if err := c.emit(ctx, out, e); err != nil {
					return err
				}
			}
		}
	}
}

func (c *Connector) emit(ctx context.Context, out chan<- canonical.Event, e canonical.Event) error {
	select {
	case out <- e:
		c.emitted.Add(1)
		c.lastEvent.Store(time.Now().UnixNano())
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Connector) runHeartbeats(ctx context.Context, out chan<- canonical.Event) {
	t := time.NewTicker(c.cfg.HeartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			e, err := canonical.NewEvent(c.cfg.SourceID, canonical.SourceMySQLCDC, "heartbeat", canonical.SeverityInfo, map[string]any{
				"emitted_at":  now.UTC().Format(time.RFC3339Nano),
				"server_id":   c.cfg.ServerID,
				"binlog_file": c.binlogFile,
			})
			if err != nil {
				c.errors.Add(1)
				continue
			}
			_ = c.emit(ctx, out, e)
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
			"source_id":   c.cfg.SourceID,
			"server_id":   fmt.Sprintf("%d", c.cfg.ServerID),
			"binlog_file": c.binlogFile,
		},
	}
}

// Close implements cdc.Connector.
func (c *Connector) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.syncer != nil {
		c.syncer.Close()
		c.syncer = nil
	}
	return nil
}
