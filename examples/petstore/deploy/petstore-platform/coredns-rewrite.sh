#!/usr/bin/env bash
# Make in-cluster petstore.localhost resolve to the aegis service, so the OIDC
# issuer (http://petstore.localhost/realms/petstore) is reachable from the
# services (which fetch JWKS at the issuer) exactly as it is from the browser.
# In-cluster only aegis is needed at that name (JWKS); the browser reaches it
# via the ingress. aegis serves /realms/... directly, so a direct hop works.
set -euo pipefail
CTX="${1:-kind-petstore-dev}"
NS="${2:-default}"
RELEASE="${3:-petstore}"
TARGET="${RELEASE}-aegis.${NS}.svc.cluster.local"

cf=$(kubectl --context "$CTX" -n kube-system get cm coredns -o jsonpath='{.data.Corefile}')
# Drop any prior petstore.localhost rewrite, then insert the current one after .:53 {.
new=$(printf '%s\n' "$cf" \
  | awk '!/rewrite name petstore.localhost/' \
  | awk -v t="$TARGET" '/^\.:53 \{/{print; print "    rewrite name petstore.localhost " t; next} {print}')
kubectl --context "$CTX" -n kube-system create cm coredns \
  --from-literal=Corefile="$new" --dry-run=client -o yaml | kubectl --context "$CTX" apply -f -
kubectl --context "$CTX" -n kube-system rollout restart deploy/coredns
kubectl --context "$CTX" -n kube-system rollout status deploy/coredns --timeout=90s >/dev/null 2>&1 || true
echo "coredns: petstore.localhost -> ${TARGET}"
