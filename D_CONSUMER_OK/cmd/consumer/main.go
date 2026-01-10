package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"middleware/consumer/internal/services/alerts"
	"middleware/consumer/internal/services/consumer"
	"middleware/consumer/internal/repositories/store"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}

func main() {
	natsURL := env("NATS_URL", "nats://127.0.0.1:4222")
	dbPath  := env("DB_PATH", "./timetable_consumer.db")

	st, err := store.Open(dbPath)
	if err != nil { log.Fatalf("db open: %v", err) }
	defer st.Close()

	ap, err := alerts.New(context.Background(), alerts.Options{
		URL: natsURL,
		Stream: "ALERTS",
		Subject: "ALERTS.upsert",
	})
	if err != nil { log.Fatalf("alerts nats: %v", err) }
	defer ap.Close()

	run, err := consumer.NewRunner(natsURL, st, ap)
	if err != nil { log.Fatalf("consumer init: %v", err) }

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := run.Run(ctx); err != nil {
			log.Fatalf("consumer run: %v", err)
		}
	}()

	log.Println("[consumer] running. Ctrl+C to stop.")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	cancel()
	log.Println("[consumer] stopped")
}
