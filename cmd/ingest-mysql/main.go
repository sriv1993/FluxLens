// Command ingest-mysql runs a Maxwell-style MySQL binlog CDC connector
// and publishes the resulting canonical events to a Kafka topic. The
// underlying connector is in internal/cdc/mysql; this command is the
// thin process wrapper.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"github.com/sriharshav1/fluxlens/internal/canonical"
	cdcmysql "github.com/sriharshav1/fluxlens/internal/cdc/mysql"
)

func main() {
	kafkaAddr := flag.String("kafka", "localhost:9092", "Kafka bootstrap address (host:port)")
	topic := flag.String("topic", "fluxlens.events.raw", "Kafka topic to publish events to")
	dsn := flag.String("dsn", "", "MySQL DSN (user:password@tcp(host:port)/db)")
	serverID := flag.Uint("server-id", 1000, "Unique MySQL replication server-id")
	sourceID := flag.String("source-id", "mysql-source", "Canonical source_id for emitted events")
	heartbeat := flag.Duration("heartbeat", 30*time.Second, "Heartbeat interval")
	tables := flag.String("tables", "", "Comma-separated db.table allowlist (empty = all)")
	binlogFile := flag.String("binlog-file", "", "Optional binlog file to start from")
	binlogPos := flag.Uint("binlog-pos", 0, "Optional binlog position when -binlog-file is set")
	flag.Parse()

	if *dsn == "" {
		log.Fatal("--dsn required")
	}

	var tableList []string
	if strings.TrimSpace(*tables) != "" {
		for _, t := range strings.Split(*tables, ",") {
			t = strings.TrimSpace(t)
			if t != "" {
				tableList = append(tableList, t)
			}
		}
	}
	connector, err := cdcmysql.New(cdcmysql.Config{
		DSN:               *dsn,
		ServerID:          uint32(*serverID),
		SourceID:          *sourceID,
		HeartbeatInterval: *heartbeat,
		Tables:            tableList,
		BinlogFile:        *binlogFile,
		BinlogPos:         uint32(*binlogPos),
	})
	if err != nil {
		log.Fatalf("new connector: %v", err)
	}
	defer connector.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(*kafkaAddr),
		Topic:                  *topic,
		Balancer:               &kafka.Hash{},
		AllowAutoTopicCreation: true,
		BatchSize:              100,
		BatchTimeout:           20 * time.Millisecond,
	}
	defer writer.Close()

	out := make(chan canonical.Event, 1024)

	go func() {
		if err := connector.Run(ctx, out); err != nil && ctx.Err() == nil {
			log.Printf("connector exited: %v", err)
			cancel()
		}
		close(out)
	}()

	statusTicker := time.NewTicker(5 * time.Second)
	defer statusTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			stats := connector.Stats()
			log.Printf("ingest-mysql shutting down: emitted=%d errors=%d", stats.EventsEmitted, stats.ErrorsSeen)
			return
		case <-statusTicker.C:
			s := connector.Stats()
			log.Printf("status: emitted=%d errors=%d lag_sec=%.2f healthy=%v",
				s.EventsEmitted, s.ErrorsSeen, s.LagSeconds, s.Healthy)
		case e, ok := <-out:
			if !ok {
				return
			}
			body, err := json.Marshal(e)
			if err != nil {
				log.Printf("marshal event: %v", err)
				continue
			}
			if err := writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(e.SourceID),
				Value: body,
				Time:  e.Timestamp,
			}); err != nil {
				if ctx.Err() != nil {
					return
				}
				log.Printf("kafka write: %v", err)
			}
		}
	}
}
