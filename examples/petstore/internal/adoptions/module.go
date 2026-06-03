// Package adoptions wires the adoptions service into an fx module.
package adoptions

import (
	"context"
	"os"
	"strconv"

	"go.uber.org/fx"

	kitaudit "github.com/fromforgesoftware/go-kit/audit"
	kitrest "github.com/fromforgesoftware/go-kit/transport/rest"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/app"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/db"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/platform/catalogclient"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/platform/gjallarhornclient"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/platform/gleipnirclient"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/platform/mockpayment"
	adoptionshttp "github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/transport/http"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/platform/audit"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/platform/auth"
)

const (
	Version         = "1.0.0"
	defaultFeeCents = 5000
)

func newVerifier() *auth.Verifier { return auth.NewVerifier(os.Getenv("AEGIS_ISSUER")) }

func newCatalog() app.Catalog { return catalogclient.New(os.Getenv("CATALOG_URL")) }

func newNotifier() app.Notifier { return gjallarhornclient.New(os.Getenv("GJALLARHORN_URL")) }

func newCharger() app.Charger { return mockpayment.New() }

func newVendor(lc fx.Lifecycle) (app.TokenVendor, error) {
	c, err := gleipnirclient.New(os.Getenv("GLEIPNIR_GRPC_ADDR"))
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return c.Close() }})
	return c, nil
}

// newAuditSink builds the kit audit.Sink from env (AUDIT_SINK) and registers a
// lifecycle hook to release any backing resources (e.g. the Talos connection).
func newAuditSink(lc fx.Lifecycle) (kitaudit.Sink, error) {
	sink, closer, err := audit.NewSink()
	if err != nil {
		return nil, err
	}
	lc.Append(fx.Hook{OnStop: func(context.Context) error { return closer() }})
	return sink, nil
}

func newAdoptionUsecase(orders app.AdoptionRepository, catalog app.Catalog, vendor app.TokenVendor, charger app.Charger, notifier app.Notifier, sink kitaudit.Sink) app.AdoptionUsecase {
	fee := defaultFeeCents
	if v, err := strconv.Atoi(os.Getenv("ADOPTION_FEE_CENTS")); err == nil && v > 0 {
		fee = v
	}
	return app.NewAdoptionUsecase(orders, catalog, vendor, charger, notifier, sink, os.Getenv("GLEIPNIR_PAYMENT_CONNECTION"), fee)
}

func FxModule() fx.Option {
	return fx.Module("adoptions",
		fx.Provide(
			newVerifier,
			auth.NewAuthenticator,
			fx.Annotate(db.NewAdoptionRepository, fx.As(new(app.AdoptionRepository))),
			newCatalog,
			newVendor,
			newCharger,
			newNotifier,
			newAuditSink,
			newAdoptionUsecase,
		),
		kitrest.NewFxController(adoptionshttp.NewAdoptionController),
	)
}
