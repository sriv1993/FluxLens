// Command orchestrator consumes curated digests from Kafka, calls the
// configured LLM provider for each event in the digest, validates the
// response through Guardrails, writes a tamper-evident audit record,
// and publishes the resulting Decisions to a downstream Kafka topic
// for the API gateway / dashboard / paging integrations to consume.
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
	"github.com/sriharshav1/fluxlens/internal/canonical"
	"github.com/sriharshav1/fluxlens/internal/llm"
	"github.com/sriharshav1/fluxlens/internal/orchestrator"
)

type curatedDigest struct {
	Strategy   int                `json:"strategy"`
	Freshness  float64            `json:"freshness"`
	Diversity  float64            `json:"diversity"`
	Redundancy float64            `json:"redundancy"`
	Selected   []canonical.Event  `json:"selected_events"`
	WindowEnd  string             `json:"window_end"`
}

func main() {
	kafkaAddr := flag.String("kafka", "localhost:9092", "Kafka bootstrap address (host:port)")
	inTopic := flag.String("in-topic", "fluxlens.events.curated", "Input topic of curated digests")
	outTopic := flag.String("out-topic", "fluxlens.decisions", "Output topic for decisions")
	group := flag.String("group", "fluxlens-orchestrator", "Consumer group ID")
	llmBase := flag.String("llm-base", "http://localhost:8080", "OpenAI-compatible base URL (use http://localhost:8080 for local WireMock)")
	llmModel := flag.String("llm-model", "fluxlens-mock-llm", "Model ID to use")
	llmKey := flag.String("llm-key", "", "API key (leave empty for mock)")
	instruction := flag.String("instruction", "You are an industrial-operations decision-support model. Classify the event severity and suggest an operator action.", "Instruction template")
	useMock := flag.Bool("mock", false, "Use the in-process MockProvider instead of the OpenAI-compatible endpoint")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	var provider llm.Provider
	if *useMock {
		provider = llm.NewMockProvider(*llmModel)
	} else {
		provider = llm.NewOpenAIProvider(*llmBase, *llmKey, *llmModel)
	}
	defer provider.Close()

	chain := auditlog.NewChain()
	orc := orchestrator.New(provider, orchestrator.NewDefaultGuardrails(), chain, "default")

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
		BatchSize:              20,
		BatchTimeout:           20 * time.Millisecond,
	}
	defer writer.Close()

	log.Printf("orchestrator running: in=%s out=%s provider=%s model=%s mock=%v", *inTopic, *outTopic, provider.Name(), provider.ModelID(), *useMock)

	for {
		select {
		case <-ctx.Done():
			ok, err := chain.Verify()
			log.Printf("orchestrator shutting down: audit_records=%d verified=%v err=%v", chain.Len(), ok, err)
			return
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
			var digest curatedDigest
			if err := json.Unmarshal(m.Value, &digest); err != nil {
				log.Printf("decode digest: %v", err)
				continue
			}
			for _, ev := range digest.Selected {
				dec, err := orc.Decide(ctx, ev, *instruction)
				if err != nil {
					log.Printf("decide error for event %s: %v", ev.EventID, err)
					continue
				}
				body, _ := json.Marshal(dec)
				if err := writer.WriteMessages(ctx, kafka.Message{Key: []byte(ev.EventID), Value: body}); err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Printf("publish decision: %v", err)
				}
			}
			log.Printf("digest processed: events=%d freshness=%.3f diversity=%.3f", len(digest.Selected), digest.Freshness, digest.Diversity)
		}
	}
}
