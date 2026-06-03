#!/bin/sh
# Render /config.js from runtime env so one image serves every environment.
# Runs from nginx:alpine's /docker-entrypoint.d/ before nginx starts.
#
# Recognised env vars (all public — a public OIDC client has no secret):
#   AEGIS_ISSUER, OIDC_CLIENT_ID, OIDC_REDIRECT_URI, CATALOG_URL, ADOPTIONS_URL
set -eu

TARGET=/usr/share/nginx/html/config.js

cat > "$TARGET" <<EOF
window.__PETSTORE_CONFIG__ = {
  aegisIssuer: "${AEGIS_ISSUER:-}",
  oidcClientId: "${OIDC_CLIENT_ID:-}",
  oidcRedirectUri: "${OIDC_REDIRECT_URI:-}",
  catalogUrl: "${CATALOG_URL:-}",
  adoptionsUrl: "${ADOPTIONS_URL:-}"
};
EOF

echo "rendered $TARGET from environment"
