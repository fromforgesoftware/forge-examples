# forge-examples

A reference monorepo of example applications built on the forge stack. It is a
learning/reference resource (`.norelease` — not imported by any other module),
so each example lives under `examples/` and is free to evolve independently.

```
examples/
  petstore/      # a catalog service + an adoptions orchestrator (Go, this repo's example)
  petstore-ui/   # the petstore Vue SPA — aegis OIDC login over the catalog/adoptions APIs
```

## petstore

A two-service example showing how to build apps on the forge stack:

- **catalog** — a pet CRUD/availability service.
- **adoptions** — an order-placement service that orchestrates catalog and a
  couple of platform services to adopt a pet.

Both are Go services built on [`go-kit`](https://github.com/fromforgesoftware/go-kit):

- **JSON:API over REST** with an auto-generated **OpenAPI** spec (served by the
  kit REST gateway).
- **gorm / Postgres** persistence, each service owning its own schema, with a
  **migrator** binary per service (`go-kit`'s embedded-SQL migrator).
- **Authentication**: every request carries an **aegis-realm-minted RS256 JWT**.
  Services verify it locally against the realm's public **JWKS** (issuer +
  `/.well-known/jwks.json`) — there is **no dependency on the aegis module**,
  only its public keys. See `internal/platform/auth`.

### The adoption flow

`POST /v1/adoptions` (adoptions service) orchestrates:

1. **catalog (REST, S2S)** — confirm the pet is `AVAILABLE`. The caller's bearer
   token is forwarded (passthrough) so catalog authorizes the same identity.
2. **gleipnir (gRPC)** — vend a payment-provider token for the owner from a
   configured gleipnir connection (`TokenService.Vend`). No real charge is made;
   the example stops at vending the secret.
3. **catalog (REST, S2S)** — mark the pet `ADOPTED`.
4. persist the order (status `COMPLETED`).
5. **notification service (REST)** — fire a "your adoption is complete"
   notification (best-effort; a failure is logged, not fatal).

### Dependencies

petstore consumes `go-kit` and `gleipnir` as **published Go modules** — no
`replace` directives and no `go.work`. The versions are pinned in
`examples/petstore/go.mod`:

- `github.com/fromforgesoftware/go-kit v0.1.0`
- `github.com/fromforgesoftware/gleipnir v0.1.0`

CI builds against exactly these pinned versions (`GOWORK=off`).

## Build & run

See [`examples/petstore/VERIFY.md`](examples/petstore/VERIFY.md) for the full,
runnable build/run/migrate/verify steps. In short:

```sh
cd examples/petstore
GOWORK=off go build ./...
go test ./...
```

## petstore-ui

The reference **Vue 3 SPA** for petstore — a standalone app built on the
published `@fromforgesoftware/vue-kit` + `ts-kit` (components, the JSON:API
`ApiClient`, `SafeStorage`). It signs in against an **aegis** realm with **OIDC
authorization_code + PKCE** (public client, S256 + state + nonce), then drives
the catalog/adoptions JSON:API endpoints with the bearer token: browse pets,
view a pet, place an adoption.

See [`examples/petstore-ui/README.md`](examples/petstore-ui/README.md) for the
realm/client setup and the login → browse → adopt flow. In short:

```sh
cd examples/petstore-ui
NODE_AUTH_TOKEN=$(gh auth token) npm install   # kits are on GitHub Packages
npm run dev                                     # http://localhost:5173
```

## Roadmap

- ~~**examples/petstore-ui** (Phase 5b) — a Vue SPA on `vue-kit` / `ts-kit` with
  aegis OIDC login, driving the catalog/adoptions APIs.~~ ✅ done.
- Fuller **talos-audit** and **gjallarhorn** notification wiring (Phase 5c).
