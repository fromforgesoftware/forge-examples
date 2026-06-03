// Package http holds the catalog JSON:API controllers.
package http

import (
	"encoding/json"
	"net/http"

	"github.com/fromforgesoftware/go-kit/application/repository"
	apierrors "github.com/fromforgesoftware/go-kit/errors"
	"github.com/fromforgesoftware/go-kit/openapi"
	"github.com/fromforgesoftware/go-kit/search/query"
	kitrest "github.com/fromforgesoftware/go-kit/transport/rest"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/api"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/app"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/domain"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/platform/auth"
)

// PetController exposes /v1/pets (JSON:API CRUD) plus a status action used by
// the adoptions service. Every route requires a valid aegis token.
type PetController struct {
	pets  app.PetUsecase
	authn *auth.Authenticator
}

func NewPetController(pets app.PetUsecase, authn *auth.Authenticator) kitrest.Controller {
	return &PetController{pets: pets, authn: authn}
}

func (c *PetController) Routes(r kitrest.Router) {
	r.Route("/v1/pets", func(r kitrest.Router) {
		r.Use(kitrest.NewAuthMiddleware(c.authn))
		r.Post("", kitrest.NewJsonApiCreateHandler(
			c.pets, api.PetFromDTO, api.PetToDTO,
			kitrest.HandlerWithOpenAPI(openapi.Summary("Add a pet"), openapi.Tags("pets"), openapi.Errors(400, 401)),
		))
		r.Get("", kitrest.NewJsonApiListHandler(
			c.pets, api.PetToDTO,
			kitrest.HandlerWithOpenAPI(openapi.Summary("List pets"), openapi.Description("Filter with filter[status]."), openapi.Tags("pets")),
		))
		r.Route("/{id}", func(r kitrest.Router) {
			r.Get("", kitrest.NewJsonApiGetHandler(
				c.pets, api.PetToDTO, []query.ParseOpt{},
				kitrest.HandlerWithOpenAPI(openapi.Summary("Get a pet"), openapi.Tags("pets"), openapi.Errors(404)),
			))
			r.Delete("", kitrest.NewJsonApiDeleteHandler(
				c.pets, repository.DeleteTypeHard,
				kitrest.HandlerWithOpenAPI(openapi.Summary("Remove a pet"), openapi.Tags("pets"), openapi.Errors(404)),
			))
			r.Post("/status", http.HandlerFunc(c.setStatus))
		})
	})
}

type statusRequest struct {
	Status string `json:"status"`
}

// setStatus flips a pet's availability; the adoptions service calls this
// (forwarding the caller's token) when an order is placed or settled.
func (c *PetController) setStatus(w http.ResponseWriter, r *http.Request) {
	var body statusRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		kitrest.JSONErrorEncoder(r.Context(), apierrors.InvalidArgument("invalid body"), w)
		return
	}
	if err := c.pets.SetStatus(r.Context(), r.PathValue("id"), domain.PetStatus(body.Status)); err != nil {
		kitrest.JSONErrorEncoder(r.Context(), err, w)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
