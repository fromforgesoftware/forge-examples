// Package gjallarhornclient announces completed adoptions via Gjallarhorn's
// REST notification API. It is a fire-and-forget POST to /api/notifications
// (JSON:API); a failure is logged by the caller, never fatal to the adoption.
package gjallarhornclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fromforgesoftware/go-kit/httpclient"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/app"
)

type Client struct {
	http    *httpclient.Client
	baseURL string
}

func New(baseURL string) *Client {
	return &Client{
		http:    httpclient.New(httpclient.WithTimeout(10 * time.Second)),
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// AdoptionCompleted sends an adoption-confirmation notification carrying the
// pet name/species, owner, and fee, so the recipient gets a meaningful message
// rather than a bare id.
func (c *Client) AdoptionCompleted(ctx context.Context, n app.AdoptionNotification) error {
	subject := fmt.Sprintf("Your adoption of %s is confirmed", petLabel(n))
	body := fmt.Sprintf(
		"Congratulations! Your adoption of %s (%s) is complete. "+
			"Adoption %s was placed for %s; an adoption fee of %s was charged.",
		n.PetName, n.PetSpecies, n.AdoptionID, n.Owner, formatFee(n.FeeCents),
	)

	payload, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"type": "notifications",
			"attributes": map[string]any{
				"recipient": n.Owner,
				"channel":   "EMAIL",
				"subject":   subject,
				"body":      body,
			},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/api/notifications", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/vnd.api+json")

	resp, err := c.http.Do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("gjallarhorn notify: status %d", resp.StatusCode)
	}
	return nil
}

func petLabel(n app.AdoptionNotification) string {
	if n.PetName != "" {
		return n.PetName
	}
	return "your pet"
}

func formatFee(cents int) string {
	return fmt.Sprintf("$%d.%02d", cents/100, cents%100)
}
