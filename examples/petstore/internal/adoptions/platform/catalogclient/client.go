// Package catalogclient is the adoptions service's S2S client for catalog,
// forwarding the caller's bearer token (passthrough) so catalog authorizes the
// same identity.
package catalogclient

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
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/platform/auth"
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

type petDoc struct {
	Data struct {
		ID         string `json:"id"`
		Attributes struct {
			Status string `json:"status"`
		} `json:"attributes"`
	} `json:"data"`
}

func (c *Client) GetPet(ctx context.Context, petID string) (app.PetInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/pets/"+petID, nil)
	if err != nil {
		return app.PetInfo{}, err
	}
	c.authorize(ctx, req)
	req.Header.Set("Accept", "application/vnd.api+json")

	resp, err := c.http.Do(ctx, req)
	if err != nil {
		return app.PetInfo{}, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return app.PetInfo{}, fmt.Errorf("catalog get pet: status %d", resp.StatusCode)
	}
	var doc petDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return app.PetInfo{}, err
	}
	return app.PetInfo{ID: doc.Data.ID, Status: doc.Data.Attributes.Status}, nil
}

func (c *Client) SetPetStatus(ctx context.Context, petID, status string) error {
	payload, _ := json.Marshal(map[string]string{"status": status})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.baseURL+"/v1/pets/"+petID+"/status", bytes.NewReader(payload))
	if err != nil {
		return err
	}
	c.authorize(ctx, req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(ctx, req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("catalog set status: status %d", resp.StatusCode)
	}
	return nil
}

func (c *Client) authorize(ctx context.Context, req *http.Request) {
	if tok, ok := auth.RawTokenFromCtx(ctx); ok {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
}
