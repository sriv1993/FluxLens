// Command synth-source generates synthetic canonical events and publishes
// them to a Kafka topic. It exists to drive the end-to-end demo, smoke
// tests, and curation-algorithm benchmarking without requiring a real
// upstream system.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	kafka "github.com/segmentio/kafka-go"

	"github.com/sriharshav1/fluxlens/internal/canonical"
)

func main() {
	kafkaAddr := flag.String("kafka", "localhost:9092", "Kafka bootstrap address (host:port)")
	topic := flag.String("topic", "fluxlens.events.raw", "Kafka topic to publish events to")
	rate := flag.Int("rate", 100, "Events per second (across all sources combined)")
	sources := flag.Int("source-count", 20, "Number of distinct synthetic sources to simulate")
	burstPct := flag.Float64("burst-pct", 0.05, "Probability per second that a source emits a burst")
	gateway := flag.String(
		"gateway",
		"",
		"Optional API gateway base URL (e.g. http://localhost:8090); each event is POSTed to /api/v1/events so the dashboard shows live traffic while Kafka feeds curator/orchestrator",
	)
	noKafka := flag.Bool(
		"no-kafka",
		false,
		"Do not connect to Kafka (for UI-only demos); requires --gateway — use when Docker/Kafka is unavailable",
	)
	flag.Parse()

	if *noKafka && strings.TrimSpace(*gateway) == "" {
		fmt.Fprintln(os.Stderr, "when using -no-kafka, -gateway must be set (e.g. http://localhost:8090)")
		os.Exit(2)
	}

	if *rate <= 0 || *sources <= 0 {
		fmt.Fprintln(os.Stderr, "rate and source-count must be positive")
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	var w *kafka.Writer
	if !*noKafka {
		w = &kafka.Writer{
			Addr:                   kafka.TCP(*kafkaAddr),
			Topic:                  *topic,
			Balancer:               &kafka.Hash{},
			AllowAutoTopicCreation: true,
			BatchSize:              100,
			BatchTimeout:           20 * time.Millisecond,
		}
		defer w.Close()
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano())) //nolint:gosec
	httpCli := &http.Client{Timeout: 3 * time.Second}

	interval := time.Second / time.Duration(*rate)
	t := time.NewTicker(interval)
	defer t.Stop()

	if *gateway != "" {
		log.Printf("synth-source posting events to gateway %s", strings.TrimSuffix(*gateway, "/"))
	}
	if *noKafka {
		log.Printf("synth-source running: no-kafka mode rate=%d sources=%d", *rate, *sources)
	} else {
		log.Printf("synth-source running: kafka=%s topic=%s rate=%d sources=%d", *kafkaAddr, *topic, *rate, *sources)
	}
	var emitted uint64
	statusTicker := time.NewTicker(5 * time.Second)
	defer statusTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("synth-source shutting down (emitted=%d)", emitted)
			return
		case <-statusTicker.C:
			log.Printf("synth-source status: emitted=%d", emitted)
		case <-t.C:
			src := fmt.Sprintf("synth-src-%02d", rng.Intn(*sources)+1)
			severity := pickSeverity(rng)
			payload := map[string]any{
				"counter": emitted,
				"jitter":  rng.Float64(),
			}
			e, err := canonical.NewEvent(src, canonical.SourceSynthetic, "tick", severity, payload)
			if err != nil {
				log.Printf("error building event: %v", err)
				continue
			}
			if w != nil {
				eb, _ := json.Marshal(e)
				err = w.WriteMessages(ctx, kafka.Message{
					Key:   []byte(src),
					Value: eb,
					Time:  e.Timestamp,
				})
				if err != nil {
					if ctx.Err() != nil {
						return
					}
					log.Printf("kafka write error: %v", err)
					continue
				}
			}
			emitted++
			postEventToGateway(ctx, httpCli, *gateway, e)

			// occasional burst from a single source
			if rng.Float64() < *burstPct/float64(*rate) {
				for i := 0; i < 5+rng.Intn(15); i++ {
					be, _ := canonical.NewEvent(src, canonical.SourceSynthetic, "burst", canonical.SeverityWarn, map[string]int{"i": i})
					if w != nil {
						beb, _ := json.Marshal(be)
						_ = w.WriteMessages(ctx, kafka.Message{Key: []byte(src), Value: beb, Time: be.Timestamp})
					}
					emitted++
					postEventToGateway(ctx, httpCli, *gateway, be)
				}
			}
		}
	}
}

func postEventToGateway(ctx context.Context, cli *http.Client, baseURL string, e canonical.Event) {
	baseURL = strings.TrimSpace(strings.TrimSuffix(baseURL, "/"))
	if baseURL == "" || cli == nil {
		return
	}
	body, err := json.Marshal(e)
	if err != nil {
		return
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/api/v1/events", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("content-type", "application/json")
	resp, err := cli.Do(req)
	if err != nil {
		log.Printf("gateway post error (event=%s): %v", e.EventID, err)
		return
	}
	_ = resp.Body.Close()
	if resp.StatusCode >= 300 {
		log.Printf("gateway post reject: %s event=%s", resp.Status, e.EventID)
	}
}

func pickSeverity(r *rand.Rand) canonical.Severity {
	x := r.Float64()
	switch {
	case x < 0.80:
		return canonical.SeverityInfo
	case x < 0.95:
		return canonical.SeverityWarn
	case x < 0.99:
		return canonical.SeverityError
	default:
		return canonical.SeverityCritical
	}
}
