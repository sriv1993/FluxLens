// Package kafkabridge consumes Kafka topics and invokes callbacks for the API gateway.
package kafkabridge

import (
	"context"
	"encoding/json"
	"log"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/curation"
	"github.com/sriharshav1/fluxlens/internal/orchestrator"
)

// CuratedDigest matches the JSON published by fluxlens-curator.
type CuratedDigest struct {
	Strategy   int               `json:"strategy"`
	Freshness  float64           `json:"freshness"`
	Diversity  float64           `json:"diversity"`
	Redundancy float64           `json:"redundancy"`
	Selected   []canonical.Event `json:"selected_events"`
	WindowEnd  string            `json:"window_end"`
}

// Config holds Kafka consumer settings for the gateway bridge.
type Config struct {
	Brokers        string
	GroupID        string
	DecisionsTopic string
	CuratedTopic   string
	RawTopic       string
}

// Handlers receive decoded Kafka messages.
type Handlers struct {
	OnDecision func(orchestrator.Decision)
	OnCurated  func(curation.Result)
	OnRawEvent func(canonical.Event)
}

// RunDecisionsConsumer blocks until ctx is canceled, reading decision records.
func RunDecisionsConsumer(ctx context.Context, cfg Config, h Handlers) {
	if cfg.DecisionsTopic == "" || h.OnDecision == nil {
		return
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{cfg.Brokers},
		Topic:   cfg.DecisionsTopic,
		GroupID: cfg.GroupID + "-decisions",
		MaxWait: 200,
	})
	defer r.Close()
	log.Printf("kafkabridge: consuming decisions topic %s", cfg.DecisionsTopic)
	loop(ctx, r, func(data []byte) {
		var dec orchestrator.Decision
		if err := json.Unmarshal(data, &dec); err != nil {
			log.Printf("kafkabridge: decode decision: %v", err)
			return
		}
		h.OnDecision(dec)
	})
}

// RunCuratedConsumer reads curated digests from Kafka.
func RunCuratedConsumer(ctx context.Context, cfg Config, h Handlers) {
	if cfg.CuratedTopic == "" || h.OnCurated == nil {
		return
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{cfg.Brokers},
		Topic:   cfg.CuratedTopic,
		GroupID: cfg.GroupID + "-curated",
		MaxWait: 200,
	})
	defer r.Close()
	log.Printf("kafkabridge: consuming curated topic %s", cfg.CuratedTopic)
	loop(ctx, r, func(data []byte) {
		var digest CuratedDigest
		if err := json.Unmarshal(data, &digest); err != nil {
			log.Printf("kafkabridge: decode curated: %v", err)
			return
		}
		h.OnCurated(curation.Result{
			Selected:         digest.Selected,
			Strategy:         curation.Strategy(digest.Strategy),
			FreshnessScore:   digest.Freshness,
			DiversityScore:   digest.Diversity,
			RedundancyScore:  digest.Redundancy,
		})
	})
}

// RunRawConsumer reads canonical events from the raw events topic.
func RunRawConsumer(ctx context.Context, cfg Config, h Handlers) {
	if cfg.RawTopic == "" || h.OnRawEvent == nil {
		return
	}
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{cfg.Brokers},
		Topic:   cfg.RawTopic,
		GroupID: cfg.GroupID + "-raw",
		MaxWait: 200,
	})
	defer r.Close()
	log.Printf("kafkabridge: consuming raw topic %s", cfg.RawTopic)
	loop(ctx, r, func(data []byte) {
		var ev canonical.Event
		if err := json.Unmarshal(data, &ev); err != nil {
			log.Printf("kafkabridge: decode event: %v", err)
			return
		}
		if err := ev.Validate(); err != nil {
			return
		}
		h.OnRawEvent(ev)
	})
}

func loop(ctx context.Context, r *kafka.Reader, handle func([]byte)) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			msgCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			m, err := r.ReadMessage(msgCtx)
			cancel()
			if err != nil {
				if ctx.Err() != nil {
					return
				}
				continue
			}
			handle(m.Value)
		}
	}
}
