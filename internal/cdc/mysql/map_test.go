package mysql

import (
	"testing"

	"github.com/go-mysql-org/go-mysql/replication"

	"github.com/sriharshav1/fluxlens/internal/canonical"
)

func TestMapRowsEventInsert(t *testing.T) {
	ev := &replication.BinlogEvent{Header: &replication.EventHeader{Timestamp: 1_700_000_000, LogPos: 99}}
	rows := &replication.RowsEvent{
		Table: &replication.TableMapEvent{Schema: []byte("db1"), Table: []byte("t1")},
		Rows:  [][]interface{}{{int64(1), "a"}},
	}
	got, err := mapRowsEvent("src", ev, rows, "insert", "bin.000001")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 event got %d", len(got))
	}
	if got[0].EventType != "mysql.row.insert" {
		t.Fatalf("event type %q", got[0].EventType)
	}
	if got[0].Severity != canonical.SeverityInfo {
		t.Fatalf("severity %q", got[0].Severity)
	}
}

func TestMapRowsEventDeleteSeverity(t *testing.T) {
	ev := &replication.BinlogEvent{Header: &replication.EventHeader{Timestamp: 1_700_000_000}}
	rows := &replication.RowsEvent{
		Table: &replication.TableMapEvent{Schema: []byte("db"), Table: []byte("t")},
		Rows:  [][]interface{}{{int64(2)}},
	}
	got, err := mapRowsEvent("src", ev, rows, "delete", "bin.000001")
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Severity != canonical.SeverityWarn {
		t.Fatalf("severity %q", got[0].Severity)
	}
}

func TestMapRowsEventUpdatePair(t *testing.T) {
	ev := &replication.BinlogEvent{Header: &replication.EventHeader{Timestamp: 1_700_000_000}}
	rows := &replication.RowsEvent{
		Table: &replication.TableMapEvent{Schema: []byte("db"), Table: []byte("t")},
		Rows: [][]interface{}{
			{int64(1), "old"},
			{int64(1), "new"},
		},
	}
	got, err := mapRowsEvent("src", ev, rows, "update", "bin.000001")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 got %d", len(got))
	}
	if got[0].EventType != "mysql.row.update" {
		t.Fatalf("type %q", got[0].EventType)
	}
}
