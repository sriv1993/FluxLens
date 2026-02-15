// Command ingest-postgres runs a Postgres logical-replication CDC connector
// and publishes canonical events to Kafka.
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
	cdcpostgres "github.com/sriharshav1/fluxlens/internal/cdc/postgres"
)

func main() {
	kafkaAddr := flag.String("kafka", "localhost:9092", "Kafka bootstrap address")
	topic := flag.String("topic", "fluxlens.events.raw", "Kafka topic")
	connString := flag.String("conn", "", "Postgres connection string (replication=database)")
	sourceID := flag.String("source-id", "postgres-source", "Canonical source_id")
	slot := flag.String("slot", "fluxlens_slot", "Replication slot name")
	tables := flag.String("tables", "", "Comma-separated schema.table list (required)")
	heartbeat := flag.Duration("heartbeat", 30*time.Second, "Heartbeat interval")
	flag.Parse()

	if *connString == "" {
		log.Fatal("--conn required")
	}
	var tableList []string
	for _, t := range strings.Split(*tables, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tableList = append(tableList, t)
		}
	}
	if len(tableList) == 0 {
		log.Fatal("--tables required (e.g. public.orders,public.inventory)")
	}

	connector, err := cdcpostgres.New(cdcpostgres.Config{
		ConnString:        *connString,
		SourceID:          *sourceID,
		SlotName:          *slot,
		Tables:            tableList,
		HeartbeatInterval: *heartbeat,
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

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-out:
			if !ok {
				return
			}
			body, err := json.Marshal(e)
			if err != nil {
				continue
			}
			if err := writer.WriteMessages(ctx, kafka.Message{
				Key:   []byte(e.SourceID),
				Value: body,
				Time:  e.Timestamp,
			}); err != nil && ctx.Err() == nil {
				log.Printf("kafka write: %v", err)
			}
		}
	}
}
