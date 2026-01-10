package nats

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"middleware/internal/helpers/configclient"
	"middleware/internal/helpers/mailer"
	"middleware/internal/helpers/render"
	 "middleware/internal/models"
)

type Runner struct {
	NatsURL     string
	Config      *configclient.Client
	Mailer      *mailer.Client
	Subject     string // "ALERTS.>" or "ALERTS.upsert"
	Stream      string // "ALERTS"
	Durable     string // "alerter_consumer"
}

func (r *Runner) Run(ctx context.Context) error {
	nc, err := nats.Connect(r.NatsURL)
	if err != nil { return err }
	defer nc.Drain()

	js, err := jetstream.New(nc)
	if err != nil { return err }


	_, _ = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     r.Stream,
		Subjects: []string{ r.Stream + ".>" },
	})

	stream, err := js.Stream(ctx, r.Stream)
	if err != nil { return err }

	cons, err := stream.Consumer(ctx, r.Durable)
	if err != nil {
		cons, err = stream.CreateConsumer(ctx, jetstream.ConsumerConfig{
			Durable: r.Durable,
			Name:    r.Durable,
			FilterSubject: r.Subject, 
			AckPolicy: jetstream.AckExplicitPolicy,
		})
		if err != nil { return err }
	}

	cc, err := cons.Consume(func(m jetstream.Msg) {
		defer func() {
			_ = m.Ack()
		}()

		var evt models.AlertEvent
		if err := json.Unmarshal(m.Data(), &evt); err != nil {
			log.Printf("[alerter] json err: %v", err)
			return
		}

		
		targets := make(map[string]struct{})

		
		for _, ag := range evt.AgendaIDs {
			subs, err := r.Config.ListAlerts(ctx, ag)
			if err != nil {
				log.Printf("[alerter] config alerts err for %s: %v", ag, err)
				continue
			}
			for _, s := range subs {
				
				if shouldNotify(s.Condition, evt.Changes, evt.Type) {
					targets[s.Target] = struct{}{}
				}
			}
		}

		
		if subsAll, err := r.Config.ListAlerts(ctx, ""); err == nil {
			for _, s := range subsAll {
				if s.AgendaID == "" || s.AgendaID == "all" {
					if shouldNotify(s.Condition, evt.Changes, evt.Type) {
						targets[s.Target] = struct{}{}
					}
				}
			}
		}

		if len(targets) == 0 {
			return 
		}

		
		
		subject, body, err := render.RenderMail(evt)
		if err != nil {
			subject = "[EDT] Notification"   
			body = evt.EmailText
		}

		for to := range targets {
		mail := models.OutgoingMail{
		Recipient: to,       
		Subject:   subject, 
		Content:   body,    
	}

	ctxSend, cancel := context.WithTimeout(ctx, 10*time.Second)
	if err := r.Mailer.Send(ctxSend, mail); err != nil {
		log.Printf("[alerter] send err to %s: %v", to, err)
	} else {
		log.Printf("[alerter] sent OK to %s", to)
	}
	cancel()
}

	})
	if err != nil { return err }

	<-ctx.Done()
	cc.Stop()
	return nil
}

func shouldNotify(cond string, changes []models.Change, typ string) bool {
	if cond == "" || cond == "always" {
		return true
	}
	if typ == "event_new" && cond == "new_event" {
		return true
	}
	if typ == "event_changed" { 
		return true
	}
	for _, c := range changes {
		switch cond {
		case "room_change":
			if c.Kind == "room_change" { return true }
		case "time_change":
			if c.Kind == "time_change" { return true }
		case "title_change":
			if c.Kind == "title_change" { return true }
		case "description_change":
			if c.Kind == "description_change" { return true }
		}
	}
	return false
}

