# syntax=docker/dockerfile:1

# --- Stage 1: build the SPA --------------------------------------------------
# The @fromforgesoftware/* packages are on GitHub Packages, so the build needs a
# token. Pass it as a build secret (never an ARG/ENV — that would bake it into a
# layer):
#
#   docker build --secret id=npm_token,env=NODE_AUTH_TOKEN \
#     -f examples/petstore-ui/Dockerfile -t petstore-ui examples/petstore-ui
#
FROM node:22-alpine AS build
WORKDIR /app

# Install deps first for layer caching. .npmrc references ${NODE_AUTH_TOKEN};
# the secret mount exposes it only for this RUN, not in the final image.
COPY package.json package-lock.json* .npmrc ./
RUN --mount=type=secret,id=npm_token \
	NODE_AUTH_TOKEN="$(cat /run/secrets/npm_token)" npm ci

COPY . .
RUN npm run build

# --- Stage 2: serve via nginx ------------------------------------------------
FROM nginx:1.27-alpine AS runtime

# SPA history fallback + asset caching config.
COPY nginx/default.conf /etc/nginx/conf.d/default.conf
# Entrypoint renders /usr/share/nginx/html/config.js from env at start.
COPY nginx/docker-entrypoint.sh /docker-entrypoint.d/40-render-config.sh
RUN chmod +x /docker-entrypoint.d/40-render-config.sh

COPY --from=build /app/dist /usr/share/nginx/html

EXPOSE 8080
# nginx:alpine's stock entrypoint runs /docker-entrypoint.d/* then execs nginx.
