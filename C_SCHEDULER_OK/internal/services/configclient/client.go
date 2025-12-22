package configclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
		http: &http.Client{ Timeout: 15 * time.Second },
	}
}

func (c *Client) ListAgendas(ctx context.Context) ([]models.Agenda, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+"/agendas", nil)
	if err != nil { return nil, err }
	res, err := c.http.Do(req)
	if err != nil { return nil, err }
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("config %s: %s", res.Status, c.base)
	}
	var out []models.Agenda
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil { return nil, err }
	if out == nil { out = []models.Agenda{} }
	return out, nil
}
