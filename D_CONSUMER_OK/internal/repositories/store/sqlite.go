package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	_ "modernc.org/sqlite"

	"middleware/consumer/internal/models"
)

type Store struct {
	DB *sql.DB
}

func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil { return nil, err }
	st := &Store{DB: db}
	if err := st.migrate(); err != nil { return nil, err }
	return st, nil
}

func (s *Store) Close() error { return s.DB.Close() }

func (s *Store) migrate() error {
	_, err := s.DB.Exec(`
CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  title TEXT,
  description TEXT,
  start TEXT,
  "end" TEXT,
  location TEXT,
  last_update TEXT,
  agenda_ids TEXT  -- JSON array
);
`)
	return err
}

func (s *Store) GetEvent(ctx context.Context, id string) (*models.Event, error) {
	row := s.DB.QueryRowContext(ctx, `SELECT id,title,description,start,"end",location,last_update,agenda_ids FROM events WHERE id=?`, id)
	var ev models.Event
	var agendas string
	if err := row.Scan(&ev.ID, &ev.Title, &ev.Description, &ev.Start, &ev.End, &ev.Location, &ev.LastUpdate, &agendas); err != nil {
		if errors.Is(err, sql.ErrNoRows) { return nil, nil }
		return nil, err
	}
	if agendas != "" {
		_ = json.Unmarshal([]byte(agendas), &ev.AgendaIDs)
	}
	return &ev, nil
}

func (s *Store) UpsertEvent(ctx context.Context, ev models.Event) error {
	ag, _ := json.Marshal(ev.AgendaIDs)
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO events (id,title,description,start,"end",location,last_update,agenda_ids)
VALUES (?,?,?,?,?,?,?,?)
ON CONFLICT(id) DO UPDATE SET
  title=excluded.title,
  description=excluded.description,
  start=excluded.start,
  "end"=excluded."end",
  location=excluded.location,
  last_update=excluded.last_update,
  agenda_ids=excluded.agenda_ids
`, ev.ID, ev.Title, ev.Description, ev.Start, ev.End, ev.Location, ev.LastUpdate, string(ag))
	return err
}

// Compare champ par champ et produit la liste des changements + "type" (room_change, time_change, etc.)
func Diff(old *models.Event, new models.Event) []models.Change {
	var changes []models.Change
	add := func(field, kind, o, n string) {
		if strings.TrimSpace(o) != strings.TrimSpace(n) {
			changes = append(changes, models.Change{Field: field, Kind: kind, Old: o, New: n})
		}
	}
	if old == nil {
		// Pas de diff pour un event nouveau (on le signalera côté caller)
		return nil
	}
	add("location", "room_change", old.Location, new.Location)
	add("start",    "time_change", old.Start,    new.Start)
	add("end",      "time_change", old.End,      new.End)
	add("title",    "title_change", old.Title,   new.Title)
	// Description souvent verbeuse — on peut l’ignorer si tu veux. Ici on la compare.
	add("description", "description_change", old.Description, new.Description)
	return changes
}

