// Package adoptions wires the adoptions service into an fx module.
package adoptions

import (
	"context"
	"os"
	"strconv"

	"go.uber.org/fx"

	kitrest "github.com/fromforgesoftware/go-kit/transport/rest"

	"github.com/fromforgesoftware/forge-examples/internal/adoptions/app"
	"github.com/fromforgesoftware/forge-examples/internal/adoptions/db"
	"github.com/fromforgesoftware/forge-examples/internal/adoptions/platform/catalogclient"
	"github.com/fromforgesoftware/forge-examples/internal/adoptions/platform/gleipnirclient"
	"github.com/fromforgesoftware/forge-examples/internal/adoptions/platform/heraldclient"
	adoptionshttp "github.com/fromforgesoftware/forge-examples/internal/adoptions/transport/http"
	"github.com/fromforgesoftware/forge-examples/internal/platform/auth"
)

const (
	Version         = "1.0.0"
	defaultFeeCents = 5000
)

func newVerifier() *auth.Verifier { return auth.NewVerifier(os.Getenv("AEGIS_ISSUER")) }

func newCatalog() app.Catalog { return catalogclient.New(os.Getenv("CATALOG_URL")) }

func newNotifier() app.Notifier { return heraldclient.New(os.Getenv("HERALD_URL")) }

func newVendor(lc fx.Lifecycle) (app.TokenVendor, error) {
	c, err := gleipnirclient.New(os.Getenv("CONDUIT_GRPC_ADDR"))
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return c.Close() }})
	return c, nil
}

func newAdoptionUsecase(orders app.AdoptionRepository, catalog app.Catalog, vendor app.TokenVendor, notifier app.Notifier) app.AdoptionUsecase {
	fee := defaultFeeCents
	if v, err := strconv.Atoi(os.Getenv("ADOPTION_FEE_CENTS")); err == nil && v > 0 {
		fee = v
	}
	return app.NewAdoptionUsecase(orders, catalog, vendor, notifier, os.Getenv("CONDUIT_PAYMENT_CONNECTION"), fee)
}

func FxModule() fx.Option {
	return fx.Module("adoptions",
		fx.Provide(
			newVerifier,
			auth.NewAuthenticator,
			fx.Annotate(db.NewAdoptionRepository, fx.As(new(app.AdoptionRepository))),
			newCatalog,
			newVendor,
			newNotifier,
			newAdoptionUsecase,
		),
		kitrest.NewFxController(adoptionshttp.NewAdoptionController),
	)
}
