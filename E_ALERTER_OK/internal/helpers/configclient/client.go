package configclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"middleware/alerter/internal/models"
)

type Client struct {
	base string
	http *http.Client
}

func New(base string) *Client {
	return &Client{
		base: base,
		http: &http.Client{ Timeout: 10 * time.Second },
	}
}

// Récupère les abonnements, avec filtre éventuel par agendaId
func (c *Client) ListAlerts(ctx context.Context, agendaID string) ([]models.AlertSubscription, error) {
	u := c.base + "/alerts"
	if agendaID != "" {
		v := url.Values{}
		v.Set("agendaId", agendaID)
		u += "?" + v.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil { return nil, err }

	res, err := c.http.Do(req)
	if err != nil { return nil, err }
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("config alerts %s: %s", res.Status, u)
	}
	var out []models.AlertSubscription
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil { return nil, err }
	if out == nil { out = []models.AlertSubscription{} }
	return out, nil
}

