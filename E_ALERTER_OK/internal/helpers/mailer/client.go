package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	 "middleware/internal/models"
)

type Client struct {
	base  string
	token string
	http  *http.Client
}

func New(baseURL, token string) *Client {
	return &Client{
		base:  strings.TrimRight(baseURL, "/"), 
		token: strings.TrimSpace(token),       
		http:  &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Send(ctx context.Context, m models.OutgoingMail) error {
	u := c.base + "/mail" 

	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false) 
	if err := enc.Encode(m); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(buf.Bytes()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		
		req.Header.Set("Authorization", c.token)
	}

	
	if os.Getenv("ALERTER_DEBUG") == "1" {
		log.Printf("[alerter] POST %s auth=%t payload=%s", u, c.token != "", buf.String())
	}

	res, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(res.Body)
		return fmt.Errorf("mail API %s: %s", res.Status, strings.TrimSpace(string(body)))
	}

	return nil
}
