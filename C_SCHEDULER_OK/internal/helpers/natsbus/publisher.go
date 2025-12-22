package natsbus

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

type Publisher struct {
	nc  *nats.Conn
	jsc nats.JetStreamContext
	subject string
}

type Options struct {
	URL     string // ex: nats://127.0.0.1:4222
	Stream  string // ex: "EVENTS"
	Subject string // ex: "EVENTS.new"
}

func New(ctx context.Context, opt Options) (*Publisher, error) {
	if opt.URL == "" { opt.URL = nats.DefaultURL }
	if opt.Stream == "" { opt.Stream = "EVENTS" }
	if opt.Subject == "" { opt.Subject = "EVENTS.new" }

	nc, err := nats.Connect(opt.URL)
	if err != nil { return nil, err }

	jsc, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil { nc.Close(); return nil, err }

	// idempotent: assurer le stream
	_, _ = jsc.AddStream(&nats.StreamConfig{
		Name:     opt.Stream,
		Subjects: []string{ opt.Stream + ".>" },
		MaxAge:   30 * 24 * time.Hour,
	})

	return &Publisher{ nc: nc, jsc: jsc, subject: opt.Subject }, nil
}

func (p *Publisher) PublishJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil { return err }
	pa, err := p.jsc.PublishAsync(p.subject, b)
	if err != nil { return err }
	select {
	case <-pa.Ok():
		return nil
	case err := <-pa.Err():
		return err
	case <-time.After(5 * time.Second):
		return fmt.Errorf("publish timeout")
	}
}

func (p *Publisher) Close() {
	if p.nc != nil { p.nc.Drain() }
}

