// Command curator consumes raw canonical events from Kafka, applies the
// configured curation strategy, and publishes curated digests to a
// downstream Kafka topic. In Phase 1 it produces curated batches on a
// configurable cadence; in Phase 2 it will become a continuous streaming
// curator with windowed selection.
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

	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/curation"
)

func main() {
	kafkaAddr := flag.String("kafka", "localhost:9092", "Kafka bootstrap address (host:port)")
	inTopic := flag.String("in-topic", "fluxlens.events.raw", "Input topic of raw canonical events")
	outTopic := flag.String("out-topic", "fluxlens.events.curated", "Output topic for curated digests")
	group := flag.String("group", "fluxlens-curator", "Consumer group ID")
	strategy := flag.Int("strategy", int(curation.StrategyGuaranteedMinDiversity), "Curation strategy (1-6)")
	diversity := flag.Float64("diversity", 80, "Diversity percent for strategies 4-6")
	k := flag.Int("k", 40, "Digest size")
	windowSec := flag.Int("window-sec", 5, "Curation window in seconds")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{*kafkaAddr},
		Topic:   *inTopic,
		GroupID: *group,
		MaxWait: 200 * time.Millisecond,
	})
	defer reader.Close()

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(*kafkaAddr),
		Topic:                  *outTopic,
		AllowAutoTopicCreation: true,
		BatchSize:              50,
		BatchTimeout:           20 * time.Millisecond,
	}
	defer writer.Close()

	buf := make([]canonical.Event, 0, 1024)
	suppression := make(map[string]struct{})
	flush := time.NewTicker(time.Duration(*windowSec) * time.Second)
	defer flush.Stop()

	log.Printf("curator running: in=%s out=%s strategy=%d diversity=%.1f k=%d window=%ds",
		*inTopic, *outTopic, *strategy, *diversity, *k, *windowSec)

	for {
		select {
		case <-ctx.Done():
			log.Println("curator shutting down")
			return
		case <-flush.C:
			if len(buf) == 0 {
				continue
			}
			req := curation.Request{
				Strategy:         curation.Strategy(*strategy),
				Events:           buf,
				K:                *k,
				DiversityPercent: *diversity,
				SuppressionSet:   suppression,
				Now:              time.Now(),
			}
			res, err := curation.Select(req)
			if err != nil {
				log.Printf("curate error: %v", err)
				buf = buf[:0]
				continue
			}
			for _, e := range res.Selected {
				suppression[e.EventID] = struct{}{}
			}
			pruneSuppression(suppression, 10000)

			body, _ := json.Marshal(map[string]any{
				"strategy":        res.Strategy,
				"freshness":       res.FreshnessScore,
				"diversity":       res.DiversityScore,
				"redundancy":      res.RedundancyScore,
				"selected_events": res.Selected,
				"window_end":      time.Now().UTC().Format(time.RFC3339Nano),
			})
			if err := writer.WriteMessages(ctx, kafka.Message{Value: body}); err != nil {
				log.Printf("write curated digest: %v", err)
			} else {
				log.Printf("digest emitted: in=%d selected=%d freshness=%.3f diversity=%.3f redundancy=%.3f",
					len(buf), len(res.Selected), res.FreshnessScore, res.DiversityScore, res.RedundancyScore)
			}
			buf = buf[:0]

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
			var e canonical.Event
			if err := json.Unmarshal(m.Value, &e); err != nil {
				log.Printf("decode raw event: %v", err)
				continue
			}
			if err := e.Validate(); err != nil {
				log.Printf("invalid event %s: %v", e.EventID, err)
				continue
			}
			buf = append(buf, e)
		}
	}
}

// pruneSuppression bounds the suppression set so it does not grow
// unboundedly across long-running processes. When the set exceeds the
// soft limit, half of the entries are evicted.
func pruneSuppression(set map[string]struct{}, softLimit int) {
	if len(set) <= softLimit {
		return
	}
	drop := len(set) / 2
	for k := range set {
		if drop <= 0 {
			break
		}
		delete(set, k)
		drop--
	}
}
