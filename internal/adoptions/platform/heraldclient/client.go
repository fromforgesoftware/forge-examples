// Package heraldclient announces completed adoptions via Herald's REST API.
package heraldclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fromforgesoftware/go-kit/httpclient"

	"github.com/fromforgesoftware/forge-examples/internal/adoptions/domain"
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

func (c *Client) AdoptionCompleted(ctx context.Context, a domain.Adoption) error {
	payload, _ := json.Marshal(map[string]any{
		"data": map[string]any{
			"type": "notifications",
			"attributes": map[string]any{
				"recipient": a.Owner(),
				"channel":   "EMAIL",
				"subject":   "Your adoption is complete",
				"body":      fmt.Sprintf("Adoption %s for pet %s is complete.", a.ID(), a.PetID()),
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
		return fmt.Errorf("herald notify: status %d", resp.StatusCode)
	}
	return nil
}
