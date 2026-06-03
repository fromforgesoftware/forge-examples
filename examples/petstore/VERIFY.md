# petstore — build, run & verify

How to build, migrate, run, and exercise the petstore example (catalog +
adoptions). All paths below are relative to `examples/petstore/`.

petstore consumes `go-kit v0.1.0`, `gleipnir v0.1.0`, and `talos v0.1.0` as
published modules (pinned in `go.mod`, no `replace`, no `go.work`). Build with
`GOWORK=off` so the
pinned versions resolve regardless of any ambient workspace.

## 0. Build, vet, test

```sh
cd examples/petstore
GOWORK=off go build ./...
GOWORK=off go vet ./...
GOWORK=off go test ./...
```

Integration tests (the `*_integration_test.go` files under `internal/*/db`) are
behind a build tag and spin up Postgres via gnomock/Docker; run them with:

```sh
GOWORK=off go test -tags integration ./...   # requires Docker
```

## 1. The binaries

Each service ships a server and a migrator:

| binary             | package                    | purpose                        |
| ------------------ | -------------------------- | ------------------------------ |
| catalog            | `./cmd/catalog`            | catalog REST/OpenAPI server    |
| catalog-migrator   | `./cmd/catalog-migrator`   | applies catalog SQL migrations |
| adoptions          | `./cmd/adoptions`          | adoptions REST/OpenAPI server  |
| adoptions-migrator | `./cmd/adoptions-migrator` | applies adoptions migrations   |

```sh
GOWORK=off go build -o ./bin/catalog            ./cmd/catalog
GOWORK=off go build -o ./bin/catalog-migrator   ./cmd/catalog-migrator
GOWORK=off go build -o ./bin/adoptions          ./cmd/adoptions
GOWORK=off go build -o ./bin/adoptions-migrator ./cmd/adoptions-migrator
```

The migrators embed their SQL (`cmd/*-migrator/migrations`) via `go-kit`'s
migrator. Each service owns its own Postgres schema (`catalog`, `adoptions`).

## 2. Configuration (environment)

Both servers and migrators read config from the environment (the same vars the
Helm charts inject — see `deploy/*/templates/_helpers.tpl`).

**Database (all four binaries):**

| var           | example     | notes                        |
| ------------- | ----------- | ---------------------------- |
| `DB_HOST`     | `localhost` |                              |
| `DB_PORT`     | `5432`      |                              |
| `DB_NAME`     | `petstore`  |                              |
| `DB_SCHEMA`   | `catalog`   | `adoptions` for that service |
| `DB_SSL`      | `disable`   | `require` in cluster         |
| `DB_USER`     | `postgres`  |                              |
| `DB_PASSWORD` | `postgres`  |                              |

**catalog server (additional):**

| var               | example                         | notes                                            |
| ----------------- | ------------------------------- | ------------------------------------------------ |
| `AEGIS_ISSUER`    | `https://aegis/realms/petstore` | realm whose JWKS verifies JWTs                   |
| `AUDIT_SINK`      | `stdout`                        | `stdout` (default) logs events; `talos` forwards |
| `TALOS_GRPC_ADDR` | `localhost:9091`                | required when `AUDIT_SINK=talos`                 |

**adoptions server (additional):**

| var                           | example                         | notes                                              |
| ----------------------------- | ------------------------------- | -------------------------------------------------- |
| `AEGIS_ISSUER`                | `https://aegis/realms/petstore` | realm whose JWKS verifies JWTs                     |
| `CATALOG_URL`                 | `http://localhost:8080`         | catalog REST base (S2S, token passthrough)         |
| `GLEIPNIR_GRPC_ADDR`          | `localhost:9090`                | gleipnir gRPC `TokenService` address               |
| `GLEIPNIR_PAYMENT_CONNECTION` | `<connection-id>`               | gleipnir connection id to vend a payment token     |
| `GJALLARHORN_URL`             | `http://localhost:8082`         | gjallarhorn notification REST base                 |
| `ADOPTION_FEE_CENTS`          | `5000`                          | optional; defaults to 5000                         |
| `AUDIT_SINK`                  | `stdout`                        | `stdout` (default) logs events; `talos` forwards   |
| `TALOS_GRPC_ADDR`             | `localhost:9091`                | required when `AUDIT_SINK=talos`                   |

The **catalog** server reads the same `AUDIT_SINK` / `TALOS_GRPC_ADDR` pair to
audit pet-create.

`AEGIS_ISSUER` **must equal** the `iss` aegis mints into its tokens — the
verifier rejects any other issuer and fetches the JWKS at
`$AEGIS_ISSUER/.well-known/jwks.json`.

## 3. Run locally

```sh
# 1. migrate, then run catalog
DB_HOST=localhost DB_NAME=petstore DB_SCHEMA=catalog DB_SSL=disable \
  DB_USER=postgres DB_PASSWORD=postgres ./bin/catalog-migrator
DB_HOST=localhost DB_NAME=petstore DB_SCHEMA=catalog DB_SSL=disable \
  DB_USER=postgres DB_PASSWORD=postgres \
  AEGIS_ISSUER=https://aegis/realms/petstore ./bin/catalog            # :8080

# 2. migrate, then run adoptions
DB_HOST=localhost DB_NAME=petstore DB_SCHEMA=adoptions DB_SSL=disable \
  DB_USER=postgres DB_PASSWORD=postgres ./bin/adoptions-migrator
DB_HOST=localhost DB_NAME=petstore DB_SCHEMA=adoptions DB_SSL=disable \
  DB_USER=postgres DB_PASSWORD=postgres \
  AEGIS_ISSUER=https://aegis/realms/petstore \
  CATALOG_URL=http://localhost:8080 \
  GLEIPNIR_GRPC_ADDR=localhost:9090 \
  GLEIPNIR_PAYMENT_CONNECTION=<connection-id> \
  GJALLARHORN_URL=http://localhost:8082 \
  AUDIT_SINK=stdout ./bin/adoptions                                   # :8080
```

Set `AUDIT_SINK=talos TALOS_GRPC_ADDR=localhost:9091` (on either server) to
forward audit events to Talos instead of logging them to stdout.

Every request needs an aegis-minted bearer token:
`Authorization: Bearer <jwt>`. No token → `401`. The token's owner (org claim if
present, else subject) is what adoptions passes to gleipnir as the vend owner.

### Exercise the API

```sh
TOKEN=<aegis RS256 access token>

# create a pet
curl -s -X POST http://localhost:8080/v1/pets \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/vnd.api+json' \
  -d '{"data":{"type":"pets","attributes":{"name":"Rex","species":"dog"}}}'

# place an adoption (catalog S2S + gleipnir vend + notify)
curl -s -X POST http://localhost:8080/v1/adoptions \
  -H "Authorization: Bearer $TOKEN" -H 'Content-Type: application/vnd.api+json' \
  -d '{"data":{"type":"adoptions","attributes":{"petId":"<pet-id>"}}}'
```

A successful adoption returns `status: COMPLETED`: adoptions confirmed
availability in catalog, vended a payment token from gleipnir and **charged the
fee with it** (a mock PSP charge in `platform/mockpayment` consumes the vended
token and returns a charge id), marked the pet `ADOPTED`, persisted the order,
**emitted a `adoption.placed` audit event** (Talos or stdout per `AUDIT_SINK`),
and fired a **richer adoption-confirmation notification** to gjallarhorn
carrying the pet name/species, owner, and fee. Catalog likewise emits a
`pet.create` audit event when a pet is added.

## 4. Container images

One distroless image per service bundles its server + migrator. Build context is
the repo root:

```sh
# from repo root
docker build -f examples/petstore/Dockerfile.catalog   -t catalog   .
docker build -f examples/petstore/Dockerfile.adoptions -t adoptions .
```

## 5. Helm charts

Per-service charts live under `deploy/`. They run the migrator as a
pre-install/pre-upgrade `Job`, then the server `Deployment`, and inject all the
env vars above (DB from a secret, `AEGIS_ISSUER` / downstream URLs from values).

```sh
# from repo root
helm lint examples/petstore/deploy/catalog
helm lint examples/petstore/deploy/adoptions
```

Key values (see each chart's `values.yaml` and `envs/<env>/values.yaml`):

- `database.*` — host/port/name/schema/ssl + credentials (or `existingSecret`).
- `aegisIssuer` — the realm issuer (both charts).
- `auditSink` (`stdout`|`talos`) + `talosGRPCAddr` — audit wiring (both charts).
- adoptions: `catalogURL`, `gleipnirGRPCAddr`, `gjallarhornURL`,
  `paymentConnection`, `feeCents`.
- `gatewaySecret` — shared HMAC gating any gateway-fronted admin routes.

Install (example, local env):

```sh
helm upgrade --install catalog   examples/petstore/deploy/catalog \
  -f examples/petstore/deploy/catalog/envs/local/values.yaml
helm upgrade --install adoptions examples/petstore/deploy/adoptions \
  -f examples/petstore/deploy/adoptions/envs/local/values.yaml \
  --set catalogURL=http://catalog --set paymentConnection=<connection-id>
```
