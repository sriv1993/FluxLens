// Package postgres provides a Postgres CDC connector for FluxLens using
// logical replication (pgoutput). The source database must have
// wal_level=logical and the FluxLens user needs REPLICATION privilege.
package postgres

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pglogrepl"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgproto3"

	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/cdc"
)

// Config configures the Postgres logical-replication connector.
type Config struct {
	ConnString string
	SourceID   string
	SlotName   string
	// Tables lists schema.table names included in the publication.
	Tables []string
	HeartbeatInterval time.Duration
}

// Connector implements cdc.Connector via Postgres logical decoding.
type Connector struct {
	cfg     Config
	filter  tableFilter
	emitted atomic.Uint64
	errors  atomic.Uint64
	lastEv  atomic.Int64
	healthy atomic.Bool

	relations map[uint32]relInfo
	relMu     sync.Mutex
}

type relInfo struct {
	Schema string
	Table  string
}

// New returns a configured Connector.
func New(cfg Config) (*Connector, error) {
	if cfg.ConnString == "" {
		return nil, errors.New("postgres cdc: ConnString required")
	}
	if cfg.SourceID == "" {
		return nil, errors.New("postgres cdc: SourceID required")
	}
	if cfg.SlotName == "" {
		cfg.SlotName = "fluxlens_slot"
	}
	if cfg.HeartbeatInterval == 0 {
		cfg.HeartbeatInterval = 30 * time.Second
	}
	c := &Connector{
		cfg:       cfg,
		filter:    newTableFilter(cfg.Tables),
		relations: make(map[uint32]relInfo),
	}
	c.healthy.Store(true)
	return c, nil
}

func (c *Connector) Name() string { return "postgres-cdc" }

func (c *Connector) Close() error { return nil }

func (c *Connector) Stats() cdc.Stats {
	last := ""
	if ns := c.lastEv.Load(); ns > 0 {
		last = time.Unix(0, ns).UTC().Format(time.RFC3339Nano)
	}
	return cdc.Stats{
		EventsEmitted:  c.emitted.Load(),
		ErrorsSeen:     c.errors.Load(),
		LastEventAtUTC: last,
		Healthy:        c.healthy.Load(),
		Details: map[string]string{
			"slot": c.cfg.SlotName,
		},
	}
}

// Run starts logical replication until ctx is canceled.
func (c *Connector) Run(ctx context.Context, out chan<- canonical.Event) error {
	cfg, err := pgconn.ParseConfig(c.cfg.ConnString)
	if err != nil {
		c.healthy.Store(false)
		return fmt.Errorf("postgres cdc: parse config: %w", err)
	}
	cfg.RuntimeParams["replication"] = "database"
	cfg.RuntimeParams["dbname"] = cfg.Database

	conn, err := pgconn.ConnectConfig(ctx, cfg)
	if err != nil {
		c.healthy.Store(false)
		return fmt.Errorf("postgres cdc: connect: %w", err)
	}
	defer conn.Close(ctx)

	if err := c.ensurePublication(ctx, conn); err != nil {
		c.healthy.Store(false)
		return err
	}

	sysident, err := pglogrepl.IdentifySystem(ctx, conn)
	if err != nil {
		c.healthy.Store(false)
		return fmt.Errorf("postgres cdc: identify system: %w", err)
	}
	_ = sysident

	_, err = pglogrepl.CreateReplicationSlot(ctx, conn, c.cfg.SlotName, "pgoutput", pglogrepl.CreateReplicationSlotOptions{Temporary: false})
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		c.healthy.Store(false)
		return fmt.Errorf("postgres cdc: create slot: %w", err)
	}

	if err := pglogrepl.StartReplication(ctx, conn, c.cfg.SlotName, 0, pglogrepl.StartReplicationOptions{
		PluginArgs: []string{"proto_version '1'", "publication_names 'fluxlens_pub'"},
	}); err != nil {
		c.healthy.Store(false)
		return fmt.Errorf("postgres cdc: start replication: %w", err)
	}

	hb := time.NewTicker(c.cfg.HeartbeatInterval)
	defer hb.Stop()

	var nextLSN pglogrepl.LSN
	inStream := false
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-hb.C:
			c.emitHeartbeat(ctx, out)
		default:
			ctx2, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
			raw, err := conn.ReceiveMessage(ctx2)
			cancel()
			if err != nil {
				if ctx.Err() != nil {
					return nil
				}
				if strings.Contains(err.Error(), "timeout") {
					continue
				}
				c.errors.Add(1)
				continue
			}
			switch msg := raw.(type) {
			case *pgproto3.CopyData:
				switch msg.Data[0] {
				case pglogrepl.XLogDataByteID:
					xld, err := pglogrepl.ParseXLogData(msg.Data[1:])
					if err != nil {
						c.errors.Add(1)
						continue
					}
					nextLSN = xld.WALStart + pglogrepl.LSN(len(xld.WALData))
					logicalMsg, err := pglogrepl.ParseV2(xld.WALData, inStream)
					if err != nil {
						c.errors.Add(1)
						continue
					}
					switch logicalMsg.(type) {
					case *pglogrepl.StreamStartMessageV2:
						inStream = true
					case *pglogrepl.StreamStopMessageV2:
						inStream = false
					}
					if ev, ok := c.mapMessage(logicalMsg); ok {
						select {
						case out <- ev:
							c.emitted.Add(1)
							c.lastEv.Store(time.Now().UnixNano())
						case <-ctx.Done():
							return nil
						}
					}
				case pglogrepl.PrimaryKeepaliveMessageByteID:
					pkm, err := pglogrepl.ParsePrimaryKeepaliveMessage(msg.Data[1:])
					if err != nil {
						continue
					}
					if pkm.ReplyRequested {
						_ = pglogrepl.SendStandbyStatusUpdate(ctx, conn, pglogrepl.StandbyStatusUpdate{WALWritePosition: nextLSN})
					}
				}
			}
		}
	}
}

func (c *Connector) ensurePublication(ctx context.Context, conn *pgconn.PgConn) error {
	tables := c.cfg.Tables
	if len(tables) == 0 {
		return errors.New("postgres cdc: at least one schema.table required")
	}
	var b strings.Builder
	b.WriteString("CREATE PUBLICATION fluxlens_pub FOR TABLE ")
	for i, t := range tables {
		if i > 0 {
			b.WriteString(", ")
		}
		parts := strings.SplitN(t, ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("postgres cdc: invalid table %q", t)
		}
		b.WriteString(quoteIdent(parts[0]))
		b.WriteString(".")
		b.WriteString(quoteIdent(parts[1]))
	}
	_, err := conn.Exec(ctx, b.String()).ReadAll()
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		// try alter publication for existing pub
		_, err2 := conn.Exec(ctx, "CREATE PUBLICATION fluxlens_pub").ReadAll()
		if err2 != nil && !strings.Contains(err2.Error(), "already exists") {
			log.Printf("postgres cdc: publication note: %v (may already exist)", err)
		}
	}
	return nil
}

func (c *Connector) mapMessage(msg pglogrepl.Message) (canonical.Event, bool) {
	switch m := msg.(type) {
	case *pglogrepl.RelationMessageV2:
		c.relMu.Lock()
		c.relations[m.RelationID] = relInfo{Schema: m.Namespace, Table: m.RelationName}
		c.relMu.Unlock()
		return canonical.Event{}, false
	case *pglogrepl.InsertMessageV2:
		return c.rowEvent(m.RelationID, "insert", m.Tuple)
	case *pglogrepl.UpdateMessageV2:
		tuple := m.NewTuple
		if tuple == nil {
			tuple = m.OldTuple
		}
		return c.rowEvent(m.RelationID, "update", tuple)
	case *pglogrepl.DeleteMessageV2:
		return c.rowEvent(m.RelationID, "delete", m.OldTuple)
	default:
		return canonical.Event{}, false
	}
}

func (c *Connector) rowEvent(relID uint32, op string, tuple *pglogrepl.TupleData) (canonical.Event, bool) {
	c.relMu.Lock()
	rel, ok := c.relations[relID]
	c.relMu.Unlock()
	if !ok {
		return canonical.Event{}, false
	}
	fqn := rel.Schema + "." + rel.Table
	if !c.filter.allow(fqn) {
		return canonical.Event{}, false
	}
	payload := map[string]any{"op": op, "schema": rel.Schema, "table": rel.Table}
	if tuple != nil {
		payload["columns"] = tuple.Columns
	}
	ev, err := canonical.NewEvent(
		c.cfg.SourceID,
		canonical.SourcePostgresCDC,
		"postgres.row_change",
		canonical.SeverityInfo,
		payload,
	)
	if err != nil {
		c.errors.Add(1)
		return canonical.Event{}, false
	}
	ev.Metadata.IngestionPod = "postgres-cdc"
	return ev, true
}

func (c *Connector) emitHeartbeat(ctx context.Context, out chan<- canonical.Event) {
	ev, err := canonical.NewEvent(c.cfg.SourceID, canonical.SourcePostgresCDC, "postgres.heartbeat", canonical.SeverityInfo, map[string]string{"status": "alive"})
	if err != nil {
		return
	}
	select {
	case out <- ev:
		c.emitted.Add(1)
	case <-ctx.Done():
	}
}

// tableFilter mirrors mysql package allowlist semantics.
type tableFilter struct {
	allowAll bool
	allowed  map[string]struct{}
}

func newTableFilter(tables []string) tableFilter {
	if len(tables) == 0 {
		return tableFilter{allowAll: true, allowed: map[string]struct{}{}}
	}
	m := make(map[string]struct{}, len(tables))
	for _, t := range tables {
		m[strings.ToLower(strings.TrimSpace(t))] = struct{}{}
	}
	return tableFilter{allowed: m}
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func (f tableFilter) allow(fqn string) bool {
	if f.allowAll {
		return true
	}
	_, ok := f.allowed[strings.ToLower(fqn)]
	return ok
}
