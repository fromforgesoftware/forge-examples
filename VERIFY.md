# petstore — end-to-end verification runbook

Brings the petstore (catalog + adoptions) up on the `forge-dev` kind cluster
alongside aegis, gleipnir, and herald, and verifies the two auth paths:

1. **petstore services verify aegis JWTs** — a client with an aegis token calls
   catalog/adoptions directly; the token is verified against the realm JWKS.
2. **Foundry logs in via aegis** — OIDC authorization-code + PKCE against the
   aegis master realm.

> These are two distinct paths. Foundry proxies to apps with the **gateway
> HMAC** (`FORGE_GATEWAY_SECRET`), not the user's aegis JWT — so the direct
> client→service path below is what exercises the JWKS verification.

## 0. Prerequisites

- Docker running, the `forge-dev` kind cluster available.
- `forge`, `kubectl`, `helm`, `jq`, `curl`.
- go/kit `v0.6.0` and `services/gleipnir v0.6.0` are published (already done).

## 1. Bring everything up

```sh
cd examples/petstore
FORGE_SRC=$(git rev-parse --show-toplevel) forge dev up
```

`forge dev` resolves the `@forge/*` dependencies in `forge.json` to their
in-repo charts (aegis, hallmark, herald, gleipnir, foundry) and deploys the
`catalog` + `adoptions` projects. Confirm the services and their migration jobs:

```sh
kubectl get pods
kubectl get svc | grep -E 'forge-(aegis|gleipnir|herald|catalog|adoptions)'
```

> **Service names / issuer.** The local overlays assume `forge-` release names
> (`forge-catalog`, `forge-gleipnir:9090`, …) and `aegisIssuer:
http://forge-aegis/realms/master`. If your cluster names differ, set
> `catalogURL` / `gleipnirGRPCAddr` / `heraldURL` / `aegisIssuer` accordingly.
> The issuer **must equal** what aegis mints — confirm with:
> `kubectl exec deploy/forge-aegis -- wget -qO- http://localhost:8080/realms/master/.well-known/openid-configuration | jq .issuer`

## 2. Get an aegis token (direct-token path)

Two facts drive this:

- aegis derives a token's **`iss` from the request Host**, so we send
  `Host: forge-aegis` — making `iss = http://forge-aegis/realms/master`, exactly
  what the in-cluster services verify (and whose JWKS they can fetch).
- the bootstrapped `foundry` client is authorization-code only, so for a
  scripted token we create a **`client_credentials`** client. aegis's `/api` is
  gated by a gateway HMAC token (HS256 of `FORGE_GATEWAY_SECRET`, `sub` must be
  a UUID); the OAuth `/token` endpoint is exempt.

```sh
kubectl port-forward svc/forge-aegis 8081:80 &
A=http://localhost:8081
H='Host: forge-aegis'

# (a) mint a gateway HMAC token (sub must be a UUID)
b64(){ openssl base64 -A | tr '+/' '-_' | tr -d '='; }
H1=$(printf '{"alg":"HS256","typ":"JWT"}' | b64)
P1=$(printf '{"sub":"00000000-0000-0000-0000-000000000001","iss":"HMAC","exp":%d}' $(($(date +%s)+3600)) | b64)
S1=$(printf '%s.%s' "$H1" "$P1" | openssl dgst -sha256 -hmac dev-forge-gateway-secret -binary | b64)
GW="$H1.$P1.$S1"

# (b) create a client_credentials client (attribute is "type", not clientType)
RID=$(curl -s -H "Authorization: Bearer $GW" -H "$H" "$A/api/realms" | jq -r '.data[]|select(.attributes.name=="master").id')
SEC=$(curl -s -H "Authorization: Bearer $GW" -H "$H" -H 'Content-Type: application/vnd.api+json' -X POST "$A/api/clients" \
  -d "{\"data\":{\"type\":\"clients\",\"attributes\":{\"realmId\":\"$RID\",\"clientId\":\"petstore-verify\",\"type\":\"CONFIDENTIAL\",\"name\":\"verify\",\"grantTypes\":[\"client_credentials\"],\"scopes\":[\"openid\"]}}}" \
  | jq -r .data.attributes.clientSecret)

# (c) client_credentials grant → an aegis RS256 token (iss=http://forge-aegis/realms/master)
TOKEN=$(curl -s -H "$H" -X POST "$A/realms/master/token" \
  -d grant_type=client_credentials -d client_id=petstore-verify -d "client_secret=$SEC" -d scope=openid | jq -r .access_token)
OWNER=petstore-verify   # the token's sub; the owner Gleipnir scopes by
```

## 3. Verify catalog (aegis JWT verification)

```sh
kubectl port-forward svc/catalog 8082:80 &
CATALOG=http://localhost:8082

# No token → 401
curl -s -o /dev/null -w '%{http_code}\n' "$CATALOG/v1/pets"        # 401

# With token → 200, and create a pet
curl -s "$CATALOG/v1/pets" -H "Authorization: Bearer $TOKEN"        # 200 []
PET=$(curl -s -X POST "$CATALOG/v1/pets" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/vnd.api+json' \
  -d '{"data":{"type":"pets","attributes":{"name":"Rex","species":"dog"}}}' \
  | jq -r .data.id)
echo "pet=$PET"
```

## 4. Seed a Gleipnir payment connection

adoptions vends a payment token from a Gleipnir connection. Gleipnir's `/api` is
also gateway-gated, so create the connection with the **gateway token** (`$GW`),
scoped to `$OWNER` (the aegis token's sub), and add an API-key credential:

```sh
kubectl port-forward svc/forge-gleipnir 8083:80 &
CONDUIT=http://localhost:8083

CONN=$(curl -s -X POST "$CONDUIT/api/connections" -H "Authorization: Bearer $GW" \
  -H 'Content-Type: application/vnd.api+json' \
  -d "{\"data\":{\"type\":\"connections\",\"attributes\":{\"owner\":\"$OWNER\",\"connector\":\"binance\"}}}" \
  | jq -r .data.id)
curl -s -X POST "$CONDUIT/api/connections/$CONN/credentials" -H "Authorization: Bearer $GW" \
  -H 'Content-Type: application/vnd.api+json' \
  -d '{"data":{"type":"credentials","attributes":{"kind":"API_KEY","apiKey":"demo","apiSecret":"demo"}}}'
echo "connection=$CONN"
```

Set `paymentConnection` to `$CONN` and roll adoptions (note the project release
is `adoptions`, and `catalogURL` is `http://catalog`):

```sh
helm upgrade adoptions deploy/adoptions -f deploy/adoptions/envs/local/values.yaml \
  --set catalogURL=http://catalog --set paymentConnection="$CONN" --reuse-values
kubectl rollout status deploy/adoptions
```

## 5. Place an adoption (catalog S2S + Gleipnir vend + Herald)

```sh
kubectl port-forward svc/adoptions 8084:80 &
ADOPT=http://localhost:8084

curl -s -X POST "$ADOPT/v1/adoptions" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/vnd.api+json' \
  -d "{\"data\":{\"type\":\"adoptions\",\"attributes\":{\"petId\":\"$PET\"}}}" | jq .
```

Expect `status: COMPLETED`. Behind the scenes adoptions: checked availability
in catalog (S2S, token forwarded), vended a payment token from Gleipnir, marked
the pet `ADOPTED` in catalog, persisted the order, and notified Herald. Confirm:

```sh
curl -s "$CATALOG/v1/pets/$PET" -H "Authorization: Bearer $TOKEN" | jq -r .data.attributes.status  # ADOPTED
curl -s "$ADOPT/v1/adoptions" -H "Authorization: Bearer $TOKEN" | jq '.data[].attributes.status'   # COMPLETED
```

## 6. Foundry login via aegis

The OIDC issuer must resolve to aegis from **both** the foundry pod (server-side
discovery/token) and the **browser** (authorize redirect), and match the `iss`
aegis mints from the request Host. With no ingress, the only name that satisfies
all three is the in-cluster Service name `forge-aegis` — so the browser has to
resolve it to the cluster too. The overlay already sets the provider issuer to
`http://forge-aegis/realms/master`; the aegis `foundry` client's redirect URI
is `http://localhost:8080/api/auth/oidc/aegis/callback`. Then:

```sh
# 1) make the browser resolve the in-cluster name
echo '127.0.0.1 forge-aegis' | sudo tee -a /etc/hosts

# 2) expose aegis at forge-aegis:80 (privileged → sudo) and foundry at :8080
sudo kubectl port-forward svc/forge-aegis 80:80 &
kubectl port-forward svc/forge-foundry 8080:80 &

open http://localhost:8080   # → "Sign in with Aegis" → admin@forge.local / admin1234
```

The flow: browser → `http://forge-aegis/realms/master/authorize` (resolves to the
port-forward) → aegis login → redirect to `localhost:8080/api/auth/oidc/aegis/callback`
→ foundry exchanges the code at `http://forge-aegis/realms/master/token` in-cluster.
`iss` is `http://forge-aegis/realms/master` from both sides, so verification passes.

> The plain email/password form (`admin@forge.local / admin1234`) works without
> any of this — it's Foundry's own local login, not the aegis OIDC path.

## Teardown

```sh
forge dev down
```
