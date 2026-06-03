# petstore-platform — the petstore "app on forge" deploy story

An umbrella Helm chart that deploys the **petstore example** (catalog +
adoptions + the Vue UI) on top of the **published forge platform**. It composes:

- `forge-platform` 0.1.0 (aliased `platform`) from
  `oci://ghcr.io/fromforgesoftware/charts` — aegis, talos, gjallarhorn,
  gleipnir, and the forge console. Gated by `platform.enabled`.
- the petstore's own charts, consumed locally from this repo:
  - `catalog` (`file://../catalog`)
  - `adoptions` (`file://../adoptions`)
  - `petstore-ui` (`file://../../../petstore-ui/deploy`)

The wiring follows in-cluster conventions: installed as release **`petstore`**,
the platform services resolve to `petstore-aegis`, `petstore-talos`,
`petstore-gjallarhorn`, `petstore-gleipnir`; the petstore services are pinned to
the stable names `catalog`, `adoptions`, `petstore-ui` (via `fullnameOverride`).
The services verify aegis-realm JWTs (`AEGIS_ISSUER =
http://petstore-aegis/realms/petstore`), adoptions calls gleipnir over gRPC
(`petstore-gleipnir:9090`), gjallarhorn over REST
(`http://petstore-gjallarhorn`), and talos for audit (`petstore-talos:9090`).

> Deploy against an **already-running** platform by setting
> `platform.enabled=false`; the petstore wiring still points at
> `petstore-<svc>` names, so install into the same release/namespace or override
> the `*URL`/`*Addr`/`aegisIssuer` values to match your platform's service names.

## What you must provide (be honest)

This chart renders cleanly, but a working install needs **you** to supply:

1. **Two key-encryption keys (KEKs)** — REQUIRED, no default. The services fail
   fast without them:
   - `platform.aegis.crypto.keyEncryptionKey` — base64 32 bytes; envelope-
     encrypts aegis realm signing keys.
   - `platform.gleipnir.vault.keyEncryptionKey` — base64 32 bytes; wraps the
     gleipnir credential vault data key. (Or set `platform.gleipnir.vault.kmsKey`
     to a Cloud KMS key in production.)
2. **Postgres.** Every backend service (aegis, talos, gjallarhorn, gleipnir,
   catalog, adoptions) needs Postgres. They each own a **schema** inside the
   target DB; the migrators (pre-install hooks) create their schema. The bundled
   values point all services at a single `postgres:5432` with DB `petstore`/
   `forge` and user `postgres/postgres` — **this chart does NOT deploy
   Postgres**. Stand one up (see below) or set `*.database.*` (and/or
   `*.database.existingSecret`) to your managed Postgres.
3. **An aegis `petstore` realm + a PUBLIC OIDC client** for the UI (see
   "Configure the realm").
4. **A gleipnir connection** for payments, and its id in
   `adoptions.paymentConnection` (see "Seed a payment connection").
5. **Ingress + CORS.** The UI runs in the browser and calls aegis/catalog/
   adoptions by their **ingress** hosts (not in-cluster names). Those services
   must allow the UI origin via CORS, and you need ingress routing those hosts.

## Prerequisites

- A cluster (e.g. `kind`) and `kubectl` context pointing at it.
- Helm 3.8+ (OCI registries on by default).
- Network access to `ghcr.io/fromforgesoftware` (the platform chart + all images
  are public).
- An ingress controller (e.g. ingress-nginx) if you want the UI reachable in a
  browser.

## 1. Build dependencies

From this directory:

```sh
helm dependency build
```

This pulls `forge-platform` 0.1.0 from the ghcr OCI registry and packages the
three local subcharts into `charts/`.

## 2. Generate the required secrets (KEKs)

```sh
# Two independent random 32-byte base64 keys:
head -c 32 /dev/urandom | base64   # -> platform.aegis.crypto.keyEncryptionKey
head -c 32 /dev/urandom | base64   # -> platform.gleipnir.vault.keyEncryptionKey
```

Pass them at install time (or commit them to a private env file / external
secret). The **`envs/local/values.yaml`** file ships well-known **DEV-ONLY**
keys so a throwaway local install works with no flags — never use those keys
elsewhere.

## 3. Stand up Postgres (dev)

A quick dev Postgres in the cluster (DEV ONLY — ephemeral, no persistence):

```sh
kubectl run postgres --image=postgres:16-alpine \
  --env=POSTGRES_USER=postgres --env=POSTGRES_PASSWORD=postgres \
  --env=POSTGRES_DB=petstore --port=5432
kubectl expose pod postgres --port=5432 --target-port=5432
# create the `forge` DB the platform services use (or set each *.database.name):
kubectl exec -it postgres -- psql -U postgres -c 'CREATE DATABASE forge;'
```

For anything real, point `*.database.host/name/username/password` (or
`existingSecret`) at managed Postgres, and use `ssl: require`.

## 4. Install

Local / kind (uses the DEV-ONLY KEKs and the dev Postgres above):

```sh
helm install petstore . -f envs/local/values.yaml
```

Any other environment — set the KEKs yourself:

```sh
helm install petstore . \
  --set platform.aegis.crypto.keyEncryptionKey="$AEGIS_KEK_B64" \
  --set platform.gleipnir.vault.keyEncryptionKey="$GLEIPNIR_KEK_B64" \
  # ... plus your database / issuer / ingress overrides
```

The migration jobs (pre-install hooks) run first and create each service's
schema; pods then come up.

## 5. Configure the aegis realm + public OIDC client

The UI signs in with **OIDC authorization_code + PKCE** as a **public client**.
The `envs/local` overrides ask the platform's aegis bootstrap to seed a
`petstore` realm and a `petstore-ui` public client with redirect
`http://petstore.localhost/callback`.

> The published aegis chart's bootstrap seeds a **single** realm. If your aegis
> only bootstraps the `master` realm (or you set `platform.enabled=false`), you
> must create the realm + client by hand. In the Foundry/aegis console:
>
> 1. Create realm **`petstore`** (issuer becomes
>    `http://petstore-aegis/realms/petstore` in-cluster, and
>    `http://aegis.localhost/realms/petstore` via ingress).
> 2. Create a **public** OIDC client `petstore-ui` (PKCE/S256, no client
>    secret).
> 3. Register the **redirect URI** exactly as the UI's
>    `petstore-ui.config.oidcRedirectUri` (default
>    `http://petstore.localhost/callback`).
> 4. Add the UI origin to the client's allowed **web origins / CORS**.
> 5. Create at least one user to log in with.

Keep these three in lockstep: `petstore-ui.config.aegisIssuer`,
`petstore-ui.config.oidcClientId`, `petstore-ui.config.oidcRedirectUri` and the
realm/client/redirect you registered. The **service** `aegisIssuer`
(`catalog.aegisIssuer` / `adoptions.aegisIssuer`) must name the **same realm**
so the JWTs the UI obtains validate against the JWKS the services fetch.

## 6. Seed a gleipnir payment connection

The adopt flow vends a payment-provider token from a **gleipnir connection** and
charges the fee with it. Create a connection in gleipnir (Foundry console or
API) for your (mock) payment provider, then set its id:

```sh
helm upgrade petstore . -f envs/local/values.yaml \
  --set adoptions.paymentConnection="<connection-id>"
```

With `adoptions.paymentConnection` empty the vend step has nothing to charge
against.

## 7. The login -> browse -> adopt flow

Open the UI (e.g. `http://petstore.localhost`) and:

1. **Login** — redirected to aegis, authenticate, return to `/callback` with a
   bearer token. *(exercises aegis OIDC + JWT)*
2. **Browse** — the SPA lists pets from `catalog` (`GET /v1/pets`), JWT attached.
   *(catalog verifies the aegis JWT against the realm JWKS)*
3. **View a pet** — `GET /v1/pets/{id}`.
4. **Adopt** — `POST /v1/adoptions` on `adoptions`, which orchestrates:
   - catalog (REST, S2S, token passthrough) — confirm `AVAILABLE`;
   - **gleipnir** (gRPC) — vend a payment token, then **charge** the fee;
   - catalog — mark `ADOPTED`; persist the order `COMPLETED`;
   - **talos** — emit an `adoption.placed` audit event (actor from the JWT);
   - **gjallarhorn** (REST) — fire an adoption-confirmation notification
     (best-effort).

   *(exercises aegis auth, gleipnir vend+charge, talos audit, gjallarhorn
   notify — the full platform path.)*

## Uninstall

```sh
helm uninstall petstore
# dev Postgres, if you created it:
kubectl delete pod/postgres svc/postgres
```

## Chart layout

```
petstore-platform/
  Chart.yaml            # deps: forge-platform (OCI, alias platform) + catalog/adoptions/petstore-ui (file://)
  values.yaml           # defaults + petstore->platform wiring (in-cluster names)
  envs/local/values.yaml# kind install: DEV-ONLY KEKs, dev Postgres, realm bootstrap, ingress hosts
  README.md             # this runbook
```
