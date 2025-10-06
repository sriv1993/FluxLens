// Command audit-writer consumes decision and operator-action events from
// Kafka and appends them to the hash-chained audit log. In Phase 1 the
// chain is in-memory and periodically verified; Phase 2 persists to
// Postgres with optional S3 Object Lock WORM mirroring.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"github.com/sriharshav1/fluxlens/internal/auditlog"
)

func main() {
	kafkaAddr := flag.String("kafka", "localhost:9092", "Kafka bootstrap address (host:port)")
	topic := flag.String("topic", "fluxlens.audit.in", "Input topic for audit records")
	group := flag.String("group", "fluxlens-audit-writer", "Consumer group ID")
	verifyEvery := flag.Duration("verify-every", 60*time.Second, "How often to run chain self-verification")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	chain := auditlog.NewChain()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{*kafkaAddr},
		Topic:   *topic,
		GroupID: *group,
		MaxWait: 200 * time.Millisecond,
	})
	defer reader.Close()

	verifyTicker := time.NewTicker(*verifyEvery)
	defer verifyTicker.Stop()

	log.Printf("audit-writer running: topic=%s", *topic)

	for {
		select {
		case <-ctx.Done():
			ok, err := chain.Verify()
			log.Printf("audit-writer shutting down (records=%d verified=%v err=%v)", chain.Len(), ok, err)
			return
		case <-verifyTicker.C:
			ok, err := chain.Verify()
			if err != nil || !ok {
				log.Printf("AUDIT CHAIN VERIFICATION FAILED records=%d err=%v", chain.Len(), err)
				continue
			}
			log.Printf("audit chain healthy: records=%d head=%s", chain.Len(), chain.HeadHash())
		default:
			msgCtx, msgCancel := context.WithTimeout(ctx, 500*time.Millisecond)
			m, err := reader.ReadMessage(msgCtx)
			msgCancel()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			var rec map[string]any
			if err := json.Unmarshal(m.Value, &rec); err != nil {
				log.Printf("decode audit input: %v", err)
				continue
			}
			kind, _ := rec["kind"].(string)
			if kind == "" {
				kind = "unknown"
			}
			payload := rec["payload"]
			if _, err := chain.Append(kind, payload); err != nil {
				log.Printf("append audit: %v", err)
			}
		}
	}
}
