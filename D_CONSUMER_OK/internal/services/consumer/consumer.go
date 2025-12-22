package consumer

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/nats-io/nats.go"
	"middleware/consumer/internal/models"
	"middleware/consumer/internal/store"
	"middleware/consumer/internal/alerts"
)

type Runner struct {
	JS     jetstream.JetStream
	Stream string         // "EVENTS"
	Subject string        // "EVENTS.>"
	ST     *store.Store
	APub   *alerts.Publisher
}

func NewRunner(natsURL string, st *store.Store, ap *alerts.Publisher) (*Runner, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil { return nil, err }
	js, err := jetstream.New(nc)
	if err != nil { return nil, err }

	// assure le stream EVENTS (au cas où)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	_, _ = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     "EVENTS",
		Subjects: []string{"EVENTS.>"},
	})

	return &Runner{
		JS: js, Stream: "EVENTS", Subject: "EVENTS.>", ST: st, APub: ap,
	}, nil
}

func (r *Runner) Run(ctx context.Context) error {
	// Consumer durable
	cons, err := r.ensureConsumer(ctx, "timetable_consumer")
	if err != nil { return err }

	// Callback sur chaque message
	cc, err := cons.Consume(func(m jetstream.Msg) {
		var ev models.Event
		if err := json.Unmarshal(m.Data(), &ev); err != nil {
			log.Printf("[consumer] json err: %v", err)
			_ = m.Ack() // ack pour éviter boucle
			return
		}

		old, err := r.ST.GetEvent(ctx, ev.ID)
		if err != nil {
			log.Printf("[consumer] db get err: %v", err)
			_ = m.Ack()
			return
		}

		changes := store.Diff(old, ev)
		if old == nil {
			// Nouveau cours → on peut envoyer une alerte “nouvel événement”
			msg := models.AlertMessage{
				Type:       "event_new",
				EventID:    ev.ID,
				AgendaIDs:  ev.AgendaIDs,
				Title:      ev.Title,
				Start:      ev.Start,
				End:        ev.End,
				Location:   ev.Location,
				Changes:    nil,
				EmailText:  alerts.BuildEmailText(ev, nil),
			}
			if err := r.APub.Publish(msg); err != nil {
				log.Printf("[consumer] alert publish err: %v", err)
			}
		} else if len(changes) > 0 {
			// Modifications détectées
			msg := models.AlertMessage{
				Type:       "event_changed",
				EventID:    ev.ID,
				AgendaIDs:  ev.AgendaIDs,
				Title:      ev.Title,
				Start:      ev.Start,
				End:        ev.End,
				Location:   ev.Location,
				Changes:    changes,
				EmailText:  alerts.BuildEmailText(ev, changes),
			}
			if err := r.APub.Publish(msg); err != nil {
				log.Printf("[consumer] alert publish err: %v", err)
			}
		}

		// Upsert en DB (à la fin pour garder l'ancien pour le diff)
		if err := r.ST.UpsertEvent(ctx, ev); err != nil {
			log.Printf("[consumer] db upsert err: %v", err)
		}

		_ = m.Ack()
	})
	if err != nil { return err }

	<-ctx.Done()
	cc.Stop()
	return nil
}

func (r *Runner) ensureConsumer(ctx context.Context, durable string) (jetstream.Consumer, error) {
	stream, err := r.JS.Stream(ctx, r.Stream)
	if err != nil { return nil, err }
	cons, err := stream.Consumer(ctx, durable)
	if err == nil { return cons, nil }
	return stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:     durable,
		Name:        durable,
		Description: "Timetable consumer",
	})
}

