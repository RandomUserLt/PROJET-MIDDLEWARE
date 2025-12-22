package scheduler

import (
	"context"
	"log"
	"time"

	"middleware/scheduler/internal/configclient"
	"middleware/scheduler/internal/timetableclient"
	"middleware/scheduler/internal/natsbus"
)

type Job struct {
	cfg *configclient.Client
	tt  *timetableclient.Client
	pub *natsbus.Publisher

	// petite config optionnelle
	DefaultWeeks int    // ex: 8
	From         string // "YYYY-MM-DD" (facultatif)
	To           string // "YYYY-MM-DD" (facultatif)
}

func NewJob(cfg *configclient.Client, tt *timetableclient.Client, pub *natsbus.Publisher) *Job {
	return &Job{ cfg: cfg, tt: tt, pub: pub, DefaultWeeks: 8 }
}

func (j *Job) RunOnce(ctx context.Context) error {
	agendas, err := j.cfg.ListAgendas(ctx)
	if err != nil { return err }
	if len(agendas) == 0 {
		return ErrNoAgendas
	}

	ids := make([]string, 0, len(agendas))
	for _, a := range agendas { ids = append(ids, a.ID) }

	// fenêtre optionnelle: from/to — sinon laisse vide
	params := timetableclient.ListParams{
		AgendaIDs: ids,
		From:      j.From,
		To:        j.To,
	}
	evs, err := j.tt.ListEvents(ctx, params)
	if err != nil { return err }

	log.Printf("[scheduler] fetched %d events", len(evs))

	count := 0
	for _, ev := range evs {
		if err := j.pub.PublishJSON(ev); err != nil {
			log.Printf("[scheduler] publish err on %s: %v", ev.ID, err)
			continue
		}
		count++
	}
	log.Printf("[scheduler] published %d events", count)
	return nil
}

var ErrNoAgendas = noAgendasErr("no agendas returned by CONFIG")

type noAgendasErr string
func (e noAgendasErr) Error() string { return string(e) }

// Boucle périodique simple (si tu veux planifier)
func (j *Job) RunEvery(ctx context.Context, d time.Duration) {
	t := time.NewTicker(d)
	defer t.Stop()
	for {
		if err := j.RunOnce(ctx); err != nil {
			log.Printf("[scheduler] run error: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// next tick
		}
	}
}

