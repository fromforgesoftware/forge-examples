// Package catalog wires the catalog service into an fx module.
package catalog

import (
	"context"
	"os"

	"go.uber.org/fx"

	kitaudit "github.com/fromforgesoftware/go-kit/audit"
	kitrest "github.com/fromforgesoftware/go-kit/transport/rest"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/app"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/db"
	cataloghttp "github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/transport/http"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/platform/audit"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/platform/auth"
)

const Version = "1.0.0"

func newVerifier() *auth.Verifier {
	return auth.NewVerifier(os.Getenv("AEGIS_ISSUER"))
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

func FxModule() fx.Option {
	return fx.Module("catalog",
		fx.Provide(
			newVerifier,
			auth.NewAuthenticator,
			newAuditSink,
			fx.Annotate(db.NewPetRepository, fx.As(new(app.PetRepository))),
			app.NewPetUsecase,
		),
		kitrest.NewFxController(cataloghttp.NewPetController),
	)
}
