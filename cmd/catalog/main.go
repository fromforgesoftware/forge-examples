// Command catalog boots the petstore catalog service: aegis-protected pet CRUD
// over the kit REST gateway, backed by Postgres.
package main

import (
	"github.com/fromforgesoftware/go-kit/app"
	"github.com/fromforgesoftware/go-kit/openapi"
	"github.com/fromforgesoftware/go-kit/persistence/gormdb/gormpg"

	"github.com/fromforgesoftware/forge-examples/internal/catalog"
)

func main() {
	app.Run(
		app.WithName("catalog"),
		app.WithVersion(catalog.Version),
		app.WithOpenAPI(
			openapi.SpecTitle("Petstore Catalog"),
			openapi.SpecVersion(catalog.Version),
			openapi.SpecDescription("Pet catalog — aegis-protected CRUD and availability."),
			openapi.SpecSecurityScheme("bearerAuth", openapi.BearerJWT()),
			openapi.DefaultSecurity("bearerAuth"),
		),
		gormpg.FxModule(),
		catalog.FxModule(),
	)
}
