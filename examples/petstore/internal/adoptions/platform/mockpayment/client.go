// Package mockpayment is a MOCK payment charger. It stands in for a real
// payment-service-provider (PSP) integration: in production this would use the
// Gleipnir-vended token to authenticate to the PSP and submit a charge. Here it
// only validates that a token was vended and returns a synthetic charge id, so
// the adoptions flow demonstrates the token actually flowing downstream.
package mockpayment

import (
	"context"
	"errors"

	"github.com/google/uuid"

	"github.com/fromforgesoftware/go-kit/monitoring/logger"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/app"
)

// Client is a stand-in PSP client. DO NOT use in production — it does not talk
// to any real payment provider.
type Client struct {
	log logger.Logger
}

func New() *Client { return &Client{log: logger.New()} }

// Charge "charges" amountCents using the Gleipnir-vended token. A real PSP
// client would present token.AccessToken/APIKey/APISecret to the provider and
// create a charge; this mock only confirms a token is present and fabricates a
// charge id.
func (c *Client) Charge(ctx context.Context, token app.PaymentToken, amountCents int, reference string) (string, error) {
	if token.AccessToken == "" && token.APIKey == "" {
		return "", errors.New("mockpayment: no payment token vended")
	}
	chargeID := "mock_charge_" + uuid.NewString()
	// MOCK: a real implementation would submit amountCents to the PSP using the
	// vended credentials instead of logging.
	c.log.InfoContext(ctx, "MOCK payment charge",
		"charge", chargeID, "amountCents", amountCents, "reference", reference)
	return chargeID, nil
}
