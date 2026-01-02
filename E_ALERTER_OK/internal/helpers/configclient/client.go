package configclient

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"middleware/internal/models"
)

type Client struct {
	base string
	http *http.Client
}

func New(base string) *Client {
	return &Client{
		base: base,
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) ListAlerts(ctx context.Context, agendaID string) ([]models.AlertSubscription, error) {
	u := c.base + "/alerts"
	if agendaID != "" {
		v := url.Values{}
		v.Set("agendaId", agendaID)
		u += "?" + v.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, fmt.Errorf("config alerts %s: %s", res.Status, u)
	}

	ct := res.Header.Get("Content-Type")

	// 1) Si JSON, on garde ton comportement
	if strings.Contains(ct, "application/json") {
		var out []models.AlertSubscription
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			return nil, err
		}
		if out == nil {
			out = []models.AlertSubscription{}
		}
		return out, nil
	}

	// 2) Sinon, on parse le plain text
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	out, err := parseAlertsPlainText(string(b))
	if err != nil {
		return nil, fmt.Errorf("parse plain text alerts: %w", err)
	}
	if out == nil {
		out = []models.AlertSubscription{}
	}
	return out, nil
}

func parseAlertsPlainText(s string) ([]models.AlertSubscription, error) {
	sc := bufio.NewScanner(strings.NewReader(s))

	var out []models.AlertSubscription
	var cur *models.AlertSubscription

	flush := func() {
		if cur == nil {
			return
		}
		// Si tu veux être strict, tu peux valider ID/Target ici
		out = append(out, *cur)
		cur = nil
	}

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}

		// "Alerte n°1" démarre un bloc
		if strings.HasPrefix(line, "Alerte n°") {
			flush()
			cur = &models.AlertSubscription{}
			continue
		}

		// Ignore l’entête "Liste des alertes :"
		if strings.HasPrefix(line, "Liste des alertes") {
			continue
		}

		// Si on n'a pas encore ouvert de bloc, on ignore
		if cur == nil {
			continue
		}

		// Format attendu (après TrimSpace) :
		// "Identifiant : a1"
		// "Agenda ID   : 1"
		// "Cible       : you@example.com"
		// "Condition   : always"
		switch {
		case strings.HasPrefix(line, "Identifiant"):
			cur.ID = strings.TrimSpace(afterColon(line))
		case strings.HasPrefix(line, "Agenda ID"):
			cur.AgendaID = strings.TrimSpace(afterColon(line))
		case strings.HasPrefix(line, "Cible"):
			cur.Target = strings.TrimSpace(afterColon(line))
		case strings.HasPrefix(line, "Condition"):
			cur.Condition = strings.TrimSpace(afterColon(line))
		}
	}

	if err := sc.Err(); err != nil {
		return nil, err
	}
	flush()
	return out, nil
}

func afterColon(line string) string {
	i := strings.Index(line, ":")
	if i < 0 {
		return ""
	}
	return line[i+1:]
}

