package mysql

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sriharshav1/fluxlens/internal/canonical"
)

func TestConnectorIntegration_MySQL(t *testing.T) {
	dsn := os.Getenv("FLUXLENS_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("FLUXLENS_TEST_MYSQL_DSN not set")
	}
	c, err := New(Config{
		DSN:               dsn,
		ServerID:          9901,
		SourceID:          "integration-test",
		HeartbeatInterval: time.Minute,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	out := make(chan canonical.Event, 32)
	go func() {
		_ = c.Run(ctx, out)
	}()

	select {
	case <-out:
		return
	case <-ctx.Done():
		if c.Stats().EventsEmitted == 0 {
			t.Fatal("no events before timeout")
		}
	}
}
