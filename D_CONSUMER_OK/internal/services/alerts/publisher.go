package alerts

import (
	"encoding/json" 
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"middleware/consumer/internal/models"
)

type Publisher struct {
	nc      *nats.Conn
	jsc     nats.JetStreamContext
	subject string // ex: "ALERTS.upsert"
}

type Options struct {
	URL     string
	Stream  string // "ALERTS"
	Subject string // "ALERTS.upsert"
}

func New(ctx context.Context, opt Options) (*Publisher, error) {
	if opt.URL == "" { opt.URL = nats.DefaultURL }
	if opt.Stream == "" { opt.Stream = "ALERTS" }
	if opt.Subject == "" { opt.Subject = "ALERTS.upsert" }

	nc, err := nats.Connect(opt.URL)
	if err != nil { return nil, err }

	jsc, err := nc.JetStream(nats.PublishAsyncMaxPending(256))
	if err != nil { nc.Close(); return nil, err }

	// Assure le stream (idempotent)
	_, _ = jsc.AddStream(&nats.StreamConfig{
		Name:     opt.Stream,
		Subjects: []string{ opt.Stream + ".>" },
		MaxAge:   30 * 24 * time.Hour,
	})

	return &Publisher{ nc: nc, jsc: jsc, subject: opt.Subject }, nil
}

func (p *Publisher) Close() { if p.nc != nil { _ = p.nc.Drain() } }

func (p *Publisher) Publish(msg models.AlertMessage) error {
	b, err := json.Marshal(msg) 
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

// Construit le texte “prêt pour email” avec détail des modifications.
func BuildEmailText(ev models.Event, changes []models.Change) string {
	if len(changes) == 0 {
		return fmt.Sprintf("Nouveau cours: %s\nQuand: %s → %s\nOù: %s\n", ev.Title, ev.Start, ev.End, ev.Location)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Mise à jour du cours: %s\nQuand: %s → %s\nOù: %s\n\n", ev.Title, ev.Start, ev.End, ev.Location)
	fmt.Fprintf(&b, "Modifications:\n")
	for _, c := range changes {
		label := map[string]string{
			"location":    "Salle",
			"start":       "Début",
			"end":         "Fin",
			"title":       "Titre",
			"description": "Description",
		}[c.Field]
		if label == "" { label = c.Field }
		fmt.Fprintf(&b, "- [%s] %s -> %s (%s)\n", label, safe(c.Old), safe(c.New), c.Kind)
	}
	return b.String()
}

func safe(s string) string {
	if strings.TrimSpace(s) == "" { return "—" }
	return s
}
