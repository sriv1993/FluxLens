// Command webhook-gateway accepts HTTP webhook payloads and publishes
// canonical events to Kafka (and optionally mirrors to the API gateway).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"log"
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
	addr := flag.String("addr", ":8091", "HTTP listen address")
	kafkaAddr := flag.String("kafka", "localhost:9092", "Kafka bootstrap")
	topic := flag.String("topic", "fluxlens.events.raw", "Kafka topic")
	gateway := flag.String("gateway", "", "Optional API gateway base URL to mirror events")
	flag.Parse()

	writer := &kafka.Writer{
		Addr:                   kafka.TCP(*kafkaAddr),
		Topic:                  *topic,
		AllowAutoTopicCreation: true,
		BatchSize:              50,
	}
	defer writer.Close()

	httpCli := &http.Client{Timeout: 3 * time.Second}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/v1/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			SourceID  string          `json:"source_id"`
			EventType string          `json:"event_type"`
			Severity  string          `json:"severity"`
			Payload   json.RawMessage `json:"payload"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sev := canonical.Severity(body.Severity)
		if sev == "" {
			sev = canonical.SeverityInfo
		}
		payload := any(body.Payload)
		if len(body.Payload) == 0 {
			payload = map[string]string{}
		}
		ev, err := canonical.NewEvent(body.SourceID, canonical.SourceWebhook, body.EventType, sev, payload)
		if err != nil || ev.Validate() != nil {
			http.Error(w, "invalid event", http.StatusBadRequest)
			return
		}
		raw, _ := json.Marshal(ev)
		ctx := r.Context()
		if err := writer.WriteMessages(ctx, kafka.Message{Key: []byte(ev.SourceID), Value: raw}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if g := strings.TrimSpace(*gateway); g != "" {
			req, _ := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(g, "/")+"/api/v1/events", bytes.NewReader(raw))
			req.Header.Set("Content-Type", "application/json")
			if _, err := httpCli.Do(req); err != nil {
				log.Printf("gateway mirror: %v", err)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_ = json.NewEncoder(w).Encode(map[string]string{"event_id": ev.EventID})
	})

	srv := &http.Server{Addr: *addr, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		log.Printf("webhook-gateway listening on %s → kafka %s topic %s", *addr, *kafkaAddr, *topic)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
