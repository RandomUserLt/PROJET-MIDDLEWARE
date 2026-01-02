package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"middleware/internal/helpers/configclient"
	"middleware/internal/helpers/mailer"
	"middleware/internal/helpers/nats"
)

func env(k, d string) string {
	if v := os.Getenv(k); v != "" { return v }
	return d
}

func main() {
	natsURL   := env("NATS_URL", "nats://127.0.0.1:4222")
	configURL := env("CONFIG_URL", "http://localhost:8080")
	//mailAPI   := env("MAIL_API_URL", "https://mail.edu.forestier.re/api")
	mailAPI   := env("MAIL_API_URL", "https://mail-api.edu.forestier.re")
	//mailToken := env("MAIL_TOKEN", "") // depuis le portail GCC
	mailToken := "iWCUqAFlwQoQNakAvksXOTSUGospGVejjdbVOeXX" 

	cfg := configclient.New(configURL)
	mc  := mailer.New(mailAPI, mailToken)

	r := &nats.Runner{ 
		NatsURL: natsURL,
		Config:  cfg,
		Mailer:  mc,
		Stream:  "ALERTS",
		Subject: "ALERTS.upsert",
		Durable: "alerter_consumer",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := r.Run(ctx); err != nil {
			log.Fatalf("[alerter] run error: %v", err)
		}
	}()

	log.Println("[alerter] running. Ctrl+C to stop")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	cancel()
	log.Println("[alerter] stopped")
}

