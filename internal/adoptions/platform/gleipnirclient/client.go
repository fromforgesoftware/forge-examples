// Package gleipnirclient vends payment-provider secrets from Gleipnir over its
// S2S gRPC TokenService.
package gleipnirclient

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	gleipnirv1 "github.com/fromforgesoftware/gleipnir/pkg/api/gleipnir/v1"

	"github.com/fromforgesoftware/forge-examples/internal/adoptions/app"
)

type Client struct {
	conn *grpc.ClientConn
	svc  gleipnirv1.TokenServiceClient
}

// New dials Gleipnir's gRPC endpoint (e.g. gleipnir:9090).
func New(addr string) (*Client, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, svc: gleipnirv1.NewTokenServiceClient(conn)}, nil
}

func (c *Client) Vend(ctx context.Context, owner, connectionID string) (app.PaymentToken, error) {
	resp, err := c.svc.Vend(ctx, &gleipnirv1.VendRequest{Owner: owner, ConnectionId: connectionID})
	if err != nil {
		return app.PaymentToken{}, err
	}
	return app.PaymentToken{
		AccessToken: resp.GetAccessToken(),
		APIKey:      resp.GetApiKey(),
		APISecret:   resp.GetApiSecret(),
	}, nil
}

func (c *Client) Close() error { return c.conn.Close() }
