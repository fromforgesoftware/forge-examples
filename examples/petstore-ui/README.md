# petstore-ui

A Vue 3 single-page app — the reference frontend for the petstore example. It
signs in against an **aegis** realm via **OIDC authorization_code + PKCE** (a
public client, no secret), then drives the **catalog** and **adoptions** JSON:API
services with the access token: browse pets, view a pet, and place an adoption.

It is built on the forge frontend stack and consumes the **published** packages
(no source links, no Module Federation — this is a standalone app):

- [`@fromforgesoftware/vue-kit`](https://github.com/fromforgesoftware/vue-kit) —
  components (DataTable, Card, Dialog, Button, EmptyState, Badge, Toaster…) +
  `style.css` (Tailwind v4).
- [`@fromforgesoftware/ts-kit`](https://github.com/fromforgesoftware/ts-kit) —
  the JSON:API `ApiClient` (`./jsonapi-client`) and `SafeStorage`/`JsonStorage`
  (`./storage`).

```
src/
  auth/      pkce.ts (S256/state/nonce), oidc.ts (discovery + token endpoint)
  stores/    auth.ts (pinia: tokens, login/callback/refresh/logout)
  api/       client.ts (two ts-kit ApiClients), pets.ts, adoptions.ts
  router/    index.ts (routes + auth guard)
  views/     LoginView, CallbackView, PetsView, PetDetailView, AdoptionsView
  components/ NewPetDialog, StatusBadge
  config/    runtime.ts (env + runtime /config.js)
deploy/      Helm chart (deployment/service/configmap/sa/hpa)
nginx/       default.conf (SPA fallback + caching), docker-entrypoint.sh
Dockerfile   multi-stage: node build → nginx static serve
```

## The OIDC / PKCE flow

The kits ship **no** OIDC/PKCE helper, so the flow is hand-rolled — minimal,
auditable, and dependency-free on the Web Crypto API (`crypto.subtle` /
`crypto.getRandomValues`):

1. **Login** (`auth/store.ts → login`): fetch the realm's
   `/.well-known/openid-configuration` (`auth/oidc.ts → discover`) to find the
   `authorize`/`token`/`jwks` endpoints. Generate a CSPRNG `code_verifier`
   (96 chars, within RFC 7636's 43–128), derive the **S256** `code_challenge`,
   and a random **state** + **nonce**. Stash `{verifier, state, nonce}` in
   `sessionStorage`, then redirect to `authorize`.
2. **Callback** (`/callback` → `handleCallback`): validate the returned
   `state` (CSRF), exchange the `code` at the token endpoint with the
   `code_verifier` (public client: `client_id`, **no secret**), then decode the
   `id_token` and verify its **nonce** matches (replay protection). Replace the
   code-bearing URL out of history.
3. **Refresh**: the access token lives in memory with an expiry; before each API
   call (and in the router guard) `validToken()` transparently refreshes via the
   `refresh_token` grant when expired.
4. **Guard** (`router/index.ts`): non-public routes require a valid token; on a
   cold reload it silently restores from the persisted refresh token, otherwise
   it redirects to `/login?returnTo=…`.
5. **Logout**: clears local tokens and follows the realm's `end_session_endpoint`
   when advertised (RP-initiated logout).

### Token handling / security notes

- **Access token: memory only** — never persisted, so it can't be exfiltrated
  from `localStorage` via XSS.
- **Refresh token: `sessionStorage`** (per-tab, cleared on tab close). This is
  the standard public-SPA trade-off when there's no httpOnly-cookie backend.
  Configure the realm to issue **short-lived access tokens + rotating refresh
  tokens** to bound exposure. For the strongest posture, front the SPA with a
  BFF that keeps tokens in an httpOnly cookie — out of scope for this reference.
- The SPA **does not verify the id_token signature**; it only checks the nonce.
  Authorization is enforced server-side: the catalog/adoptions services verify
  the RS256 JWT against the realm JWKS on every request.
- The catalog/adoptions origins must allow this app's origin via **CORS**
  (preflight + `Authorization` header), since the browser calls them directly.

## Configuration

Five public values, resolved at runtime first (deploy-time `/config.js`), then
falling back to build-time `VITE_*` env:

| runtime (`/config.js`) | build-time env             | meaning                                   |
| ---------------------- | -------------------------- | ----------------------------------------- |
| `aegisIssuer`          | `VITE_AEGIS_ISSUER`        | realm issuer (`…/realms/petstore`)        |
| `oidcClientId`         | `VITE_OIDC_CLIENT_ID`      | public client id                          |
| `oidcRedirectUri`      | `VITE_OIDC_REDIRECT_URI`   | this app's `…/callback` URL               |
| `catalogUrl`           | `VITE_CATALOG_URL`         | catalog REST base                         |
| `adoptionsUrl`         | `VITE_ADOPTIONS_URL`       | adoptions REST base                       |

`src/config/runtime.ts` prefers `window.__PETSTORE_CONFIG__` (from `/config.js`)
and falls back to `import.meta.env.VITE_*`, so **one built image is portable**
across environments — the container entrypoint re-renders `/config.js` from env.

## Configure the aegis realm + public OIDC client

In your aegis realm (the same realm whose issuer the catalog/adoptions services
verify against — `AEGIS_ISSUER` in `../petstore/VERIFY.md`):

1. Create a **public** OIDC client (no client secret), e.g. `petstore-ui`.
2. Enable the **authorization code** grant with **PKCE (S256)** required.
3. Register the **redirect URI** exactly: `http://localhost:5173/callback` for
   local dev (or your deployed `https://<host>/callback`).
4. Allow the app origin as a **web origin / CORS** entry for the client.
5. Ensure standard scopes `openid profile email` (and `offline_access` if you
   want refresh tokens).
6. On the **catalog/adoptions** services, allow CORS from the SPA origin and the
   `Authorization` header.

## Run locally

```sh
cd examples/petstore-ui

# Install (kits are on GitHub Packages — authenticate with a GH token)
NODE_AUTH_TOKEN=$(gh auth token) npm install

# Point the app at your realm + services (dev fallback via Vite env)
cat > .env.local <<'EOF'
VITE_AEGIS_ISSUER=http://localhost:8081/realms/petstore
VITE_OIDC_CLIENT_ID=petstore-ui
VITE_OIDC_REDIRECT_URI=http://localhost:5173/callback
VITE_CATALOG_URL=http://localhost:8080
VITE_ADOPTIONS_URL=http://localhost:8083
EOF

npm run dev   # http://localhost:5173
```

Then **login → browse → adopt**:

1. Open the app; the guard sends you to **/login**. Click *Sign in with aegis* —
   you're redirected to the realm, authenticate, and bounce back to `/callback`.
2. **Pets** lists the catalog (filter by status). *New pet* POSTs a pet.
3. Open a pet, click **Adopt** → confirm. This POSTs `/v1/adoptions`, which
   server-side checks availability, vends a payment token via gleipnir, marks
   the pet `ADOPTED`, and records the order (`status: COMPLETED`).
4. **My adoptions** lists your orders.

## Verify (build only — no live aegis needed)

```sh
cd examples/petstore-ui
NODE_AUTH_TOKEN=$(gh auth token) npm install
npx vue-tsc --noEmit   # type-check (clean)
npx vite build         # production build → dist/ (+ dist/config.js)
```

## Build & deploy

```sh
# Image (kits need a token — pass it as a build secret, never an ARG):
NODE_AUTH_TOKEN=$(gh auth token) \
  docker build --secret id=npm_token,env=NODE_AUTH_TOKEN \
  -f Dockerfile -t petstore-ui .

# Helm
helm lint deploy
helm upgrade --install petstore-ui deploy \
  -f deploy/envs/local/values.yaml
```

The chart renders `/config.js` from a ConfigMap-backed env at pod start (nginx
entrypoint), serves the SPA with a history fallback, and rolls pods when the
config changes (`checksum/config` annotation).
