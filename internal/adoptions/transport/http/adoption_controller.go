// Package http holds the adoptions JSON:API controllers.
package http

import (
	"context"
	"net/http"

	"github.com/fromforgesoftware/go-kit/openapi"
	"github.com/fromforgesoftware/go-kit/search/query"
	kitrest "github.com/fromforgesoftware/go-kit/transport/rest"

	"github.com/fromforgesoftware/forge-examples/internal/adoptions/api"
	"github.com/fromforgesoftware/forge-examples/internal/adoptions/app"
	"github.com/fromforgesoftware/forge-examples/internal/adoptions/domain"
	"github.com/fromforgesoftware/forge-examples/internal/platform/auth"
)

// AdoptionController exposes /v1/adoptions: place an order (which calls catalog
// + Gleipnir + Herald) and read the caller's orders. Every route requires a
// valid aegis token; the owner is taken from the token, never the body.
type AdoptionController struct {
	adoptions app.AdoptionUsecase
	authn     *auth.Authenticator
}

func NewAdoptionController(adoptions app.AdoptionUsecase, authn *auth.Authenticator) kitrest.Controller {
	return &AdoptionController{adoptions: adoptions, authn: authn}
}

func (c *AdoptionController) Routes(r kitrest.Router) {
	r.Route("/v1/adoptions", func(r kitrest.Router) {
		r.Use(kitrest.NewAuthMiddleware(c.authn))
		r.Post("", kitrest.NewJsonApiCommandHandler(
			c.place, decodePlace, api.AdoptionToDTO,
			kitrest.HandlerWithOpenAPI(
				openapi.Summary("Place an adoption"),
				openapi.Description("Checks availability, charges via a Gleipnir-vended token, marks the pet adopted, and records the order."),
				openapi.Tags("adoptions"), openapi.Errors(400, 401, 409),
			),
		))
		r.Get("", kitrest.NewJsonApiListHandler(
			c.adoptions, api.AdoptionToDTO,
			kitrest.HandlerWithOpenAPI(openapi.Summary("List adoptions"), openapi.Description("Filter with filter[owner]."), openapi.Tags("adoptions")),
		))
		r.Get("/{id}", kitrest.NewJsonApiGetHandler(
			c.adoptions, api.AdoptionToDTO, []query.ParseOpt{},
			kitrest.HandlerWithOpenAPI(openapi.Summary("Get an adoption"), openapi.Tags("adoptions"), openapi.Errors(404)),
		))
	})
}

type placeCommand struct {
	Owner string
	PetID string
}

func (c *AdoptionController) place(ctx context.Context, cmd placeCommand) (domain.Adoption, error) {
	return c.adoptions.Place(ctx, cmd.Owner, cmd.PetID)
}

func decodePlace(req *http.Request) (placeCommand, error) {
	body, err := kitrest.UnmarshalPayloadFromRequest[*api.AdoptionDTO](req)
	if err != nil {
		return placeCommand{}, err
	}
	return placeCommand{
		Owner: auth.OwnerFromCtx(req.Context()),
		PetID: body.RPetID,
	}, nil
}
