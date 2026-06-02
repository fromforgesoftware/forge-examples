// Package catalog wires the catalog service into an fx module.
package catalog

import (
	"os"

	"go.uber.org/fx"

	kitrest "github.com/fromforgesoftware/go-kit/transport/rest"

	"github.com/fromforgesoftware/forge-examples/internal/catalog/app"
	"github.com/fromforgesoftware/forge-examples/internal/catalog/db"
	cataloghttp "github.com/fromforgesoftware/forge-examples/internal/catalog/transport/http"
	"github.com/fromforgesoftware/forge-examples/internal/platform/auth"
)

const Version = "1.0.0"

func newVerifier() *auth.Verifier {
	return auth.NewVerifier(os.Getenv("AEGIS_ISSUER"))
}

func FxModule() fx.Option {
	return fx.Module("catalog",
		fx.Provide(
			newVerifier,
			auth.NewAuthenticator,
			fx.Annotate(db.NewPetRepository, fx.As(new(app.PetRepository))),
			app.NewPetUsecase,
		),
		kitrest.NewFxController(cataloghttp.NewPetController),
	)
}
