package timetableclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"middleware/scheduler/internal/models"
)

type Client struct {
	base string
	http *http.Client
}

func New(base string) *Client {
	return &Client{
		base: base,
		http: &http.Client{ Timeout: 20 * time.Second },
	}
}

type ListParams struct {
	AgendaIDs []string // obligatoire côté Timetable
	From      string   // "YYYY-MM-DD", optionnel
	To        string   // "YYYY-MM-DD", optionnel
}

func (c *Client) ListEvents(ctx context.Context, p ListParams) ([]models.Event, error) {
	if len(p.AgendaIDs) == 0 {
		return []models.Event{}, nil
	}
	v := url.Values{}
	v.Set("agendaIds", strings.Join(p.AgendaIDs, ","))
	if strings.TrimSpace(p.From) != "" { v.Set("from", p.From) }
	if strings.TrimSpace(p.To) != "" { v.Set("to", p.To) }

	u := fmt.Sprintf("%s/events?%s", c.base, v.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil { return nil, err }
	res, err := c.http.Do(req)
	if err != nil { return nil, err }
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("timetable %s: %s", res.Status, u)
	}
	var out []models.Event
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil { return nil, err }
	if out == nil { out = []models.Event{} }
	return out, nil
}

