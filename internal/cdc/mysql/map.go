package mysql

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/go-mysql-org/go-mysql/replication"

	"github.com/sriharshav1/fluxlens/internal/canonical"
)

func eventTypeForAction(action string) string {
	switch action {
	case "insert":
		return "mysql.row.insert"
	case "update":
		return "mysql.row.update"
	case "delete":
		return "mysql.row.delete"
	default:
		return "mysql.row." + action
	}
}

func severityForAction(action string) canonical.Severity {
	if action == "delete" {
		return canonical.SeverityWarn
	}
	return canonical.SeverityInfo
}

func binlogTime(headerTimestamp uint32) time.Time {
	if headerTimestamp == 0 {
		return time.Now().UTC()
	}
	return time.Unix(int64(headerTimestamp), 0).UTC()
}

type rowChange struct {
	Before []any `json:"before,omitempty"`
	After  []any `json:"after,omitempty"`
	Row    []any `json:"row,omitempty"`
}

func actionFromEventType(t replication.EventType) string {
	switch t {
	case replication.WRITE_ROWS_EVENTv0, replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
		return "insert"
	case replication.UPDATE_ROWS_EVENTv0, replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
		return "update"
	case replication.DELETE_ROWS_EVENTv0, replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
		return "delete"
	default:
		return ""
	}
}

func mapRowsEvent(sourceID string, ev *replication.BinlogEvent, rows *replication.RowsEvent, action, binlogFile string) ([]canonical.Event, error) {
	schema := string(rows.Table.Schema)
	table := string(rows.Table.Table)
	ts := binlogTime(ev.Header.Timestamp)

	meta := map[string]any{
		"binlog_file":     binlogFile,
		"binlog_position": ev.Header.LogPos,
	}

	var changes []rowChange
	switch action {
	case "insert", "delete":
		for _, row := range rows.Rows {
			rc := rowChange{Row: rowValues(row)}
			if action == "delete" {
				rc.Before = rc.Row
				rc.Row = nil
			}
			changes = append(changes, rc)
		}
	case "update":
		if len(rows.Rows)%2 != 0 {
			return nil, fmt.Errorf("mysql cdc: update event has odd row count")
		}
		for i := 0; i < len(rows.Rows); i += 2 {
			changes = append(changes, rowChange{
				Before: rowValues(rows.Rows[i]),
				After:  rowValues(rows.Rows[i+1]),
			})
		}
	default:
		return nil, nil
	}

	out := make([]canonical.Event, 0, len(changes))
	for _, ch := range changes {
		payload := map[string]any{
			"database": schema,
			"table":    table,
			"action":   action,
			"binlog":   meta,
		}
		if ch.Row != nil {
			payload["row"] = ch.Row
		}
		if ch.Before != nil {
			payload["before"] = ch.Before
		}
		if ch.After != nil {
			payload["after"] = ch.After
		}
		e, err := canonical.NewEvent(sourceID, canonical.SourceMySQLCDC, eventTypeForAction(action), severityForAction(action), payload)
		if err != nil {
			return nil, err
		}
		e.Timestamp = ts
		out = append(out, e)
	}
	return out, nil
}

func rowValues(row []interface{}) []any {
	if len(row) == 0 {
		return nil
	}
	out := make([]any, len(row))
	for i, v := range row {
		out[i] = jsonValue(v)
	}
	return out
}

func jsonValue(v interface{}) any {
	switch x := v.(type) {
	case nil:
		return nil
	case []byte:
		// Prefer UTF-8 text; fall back to base64 for binary.
		if len(x) == 0 {
			return ""
		}
		for _, b := range x {
			if b < 0x20 && b != '\t' && b != '\n' && b != '\r' {
				return base64.StdEncoding.EncodeToString(x)
			}
		}
		return string(x)
	default:
		return x
	}
}
