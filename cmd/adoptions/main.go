// Command adoptions boots the petstore adoptions service: aegis-protected
// order placement that calls catalog (S2S), Gleipnir (vend), and Herald.
package main

import (
	"github.com/fromforgesoftware/go-kit/app"
	"github.com/fromforgesoftware/go-kit/openapi"
	"github.com/fromforgesoftware/go-kit/persistence/gormdb/gormpg"

	"github.com/fromforgesoftware/forge-examples/internal/adoptions"
)

func main() {
	app.Run(
		app.WithName("adoptions"),
		app.WithVersion(adoptions.Version),
		app.WithOpenAPI(
			openapi.SpecTitle("Petstore Adoptions"),
			openapi.SpecVersion(adoptions.Version),
			openapi.SpecDescription("Adoption orders — catalog S2S, Gleipnir-vended payment, Herald notifications."),
			openapi.SpecSecurityScheme("bearerAuth", openapi.BearerJWT()),
			openapi.DefaultSecurity("bearerAuth"),
		),
		gormpg.FxModule(),
		adoptions.FxModule(),
	)
}
