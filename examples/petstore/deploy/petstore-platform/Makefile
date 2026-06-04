# Local dev deploy of the petstore on the forge platform, in kind.
#
#   make dev     bring everything up (cluster, postgres, platform + petstore) and
#                wait for pods ready, then print the manual next steps
#   make status  show pods/jobs
#   make forward port-forward the UI + console for quick access
#   make down    helm uninstall + remove dev postgres (keep the cluster)
#   make clean   delete the whole kind cluster
#
# DEV ONLY. Uses envs/local/values.yaml (well-known dev KEKs + dev postgres).
# Some steps can't be automated and are printed by `make next`: creating the
# aegis `petstore` realm + public OIDC client, seeding a gleipnir payment
# connection, and ingress/hosts for browser access.

CLUSTER ?= petstore-dev
RELEASE ?= petstore
NAMESPACE ?= default
VALUES  ?= envs/local/values.yaml
TIMEOUT ?= 10m

CTX     := kind-$(CLUSTER)
KUBECTL := kubectl --context $(CTX) -n $(NAMESPACE)
KCTL    := kubectl --context $(CTX)
HELM    := helm --kube-context $(CTX) -n $(NAMESPACE)
INGRESS_NGINX ?= https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml

.PHONY: dev up cluster ingress deps images postgres install wait status forward next down clean help

# forge-examples repo root (4 levels up from this chart dir).
EXROOT := ../../../..
TAG    := 1.0.0

## dev: full local bring-up (cluster -> deps -> images -> postgres -> install -> ingress)
dev up: cluster deps images postgres install ingress wait status next

## cluster: create the kind cluster (with ingress host port-mappings) if absent
cluster:
	@kind get clusters 2>/dev/null | grep -qx '$(CLUSTER)' \
		&& echo "==> kind cluster '$(CLUSTER)' already exists" \
		|| { echo "==> creating kind cluster '$(CLUSTER)'"; kind create cluster --name '$(CLUSTER)' --config kind-config.yaml; }

## ingress: install ingress-nginx, route petstore.localhost, rewrite in-cluster DNS
ingress:
	@echo "==> installing ingress-nginx"
	@$(KCTL) apply -f $(INGRESS_NGINX)
	@echo "==> waiting for ingress-nginx controller"
	@$(KCTL) wait --namespace ingress-nginx --for=condition=Ready pod \
		-l app.kubernetes.io/component=controller --timeout=240s
	@echo "==> routing petstore.localhost + coredns rewrite"
	@$(KUBECTL) apply -f ingress.yaml
	@bash coredns-rewrite.sh $(CTX)

## deps: pull the published forge-platform chart + build local subcharts
deps:
	@echo "==> helm dependency build (pulls forge-platform from ghcr)"
	@$(HELM) dependency build .

## images: build the petstore example images and load them into kind
##   (the platform images are pulled from ghcr; they must be multi-arch for your
##    node's arch — on Apple Silicon that means linux/arm64. The example images
##    are built here for the host arch, so they always match.)
images:
	@echo "==> building petstore example images ($(TAG))"
	docker build -f $(EXROOT)/examples/petstore/Dockerfile.catalog   -t ghcr.io/fromforgesoftware/catalog:$(TAG)   $(EXROOT)
	docker build -f $(EXROOT)/examples/petstore/Dockerfile.adoptions -t ghcr.io/fromforgesoftware/adoptions:$(TAG) $(EXROOT)
	cd $(EXROOT)/examples/petstore-ui && DOCKER_BUILDKIT=1 NODE_AUTH_TOKEN="$$(gh auth token)" \
		docker build --secret id=npm_token,env=NODE_AUTH_TOKEN -t ghcr.io/fromforgesoftware/petstore-ui:$(TAG) .
	@echo "==> loading example images into kind"
	kind load docker-image ghcr.io/fromforgesoftware/catalog:$(TAG) ghcr.io/fromforgesoftware/adoptions:$(TAG) ghcr.io/fromforgesoftware/petstore-ui:$(TAG) --name $(CLUSTER)

## postgres: deploy the dev Postgres, wait for it, ensure the `forge` DB exists
postgres:
	@echo "==> deploying dev postgres"
	@$(KUBECTL) apply -f postgres-dev.yaml
	@echo "==> waiting for postgres to be ready"
	@$(KUBECTL) rollout status deploy/postgres --timeout=180s
	@$(KUBECTL) wait --for=condition=Available deploy/postgres --timeout=180s
	@echo "==> ensuring database 'forge' exists"
	@$(KUBECTL) exec deploy/postgres -- sh -c \
		'psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname='"'"'forge'"'"'" | grep -q 1 \
		 || psql -U postgres -c "CREATE DATABASE forge"'

## install: helm upgrade --install the umbrella and wait for resources ready
install:
	@echo "==> installing release '$(RELEASE)' (helm --wait, migrations run as pre-install hooks)"
	@$(HELM) upgrade --install '$(RELEASE)' . -f '$(VALUES)' --wait --timeout '$(TIMEOUT)'

## wait: belt-and-suspenders wait for all deployments to be Available
wait:
	@echo "==> waiting for all deployments to be Available"
	@$(KUBECTL) wait --for=condition=Available deploy --all --timeout=300s || \
		echo "!! some deployments not Available yet — check 'make status'"

## status: show pods and (migration) jobs
status:
	@echo "==> pods"; $(KUBECTL) get pods -o wide
	@echo "==> jobs (migrations)"; $(KUBECTL) get jobs

## forward: not needed — ingress serves everything at http://petstore.localhost
forward:
	@echo "Open http://petstore.localhost (ingress serves the SPA + APIs + aegis"
	@echo "on one origin via the host :80 port-mapping; *.localhost resolves to"
	@echo "127.0.0.1 on macOS, so no /etc/hosts edit is needed)."

## next: print the steps that can't be automated
next:
	@echo ""
	@echo "==> petstore is up. Open: http://petstore.localhost"
	@echo "  - login is aegis OIDC/PKCE; aegis bootstraps realm 'petstore' + the"
	@echo "    public 'petstore-ui' client (redirect http://petstore.localhost/callback)."
	@echo "  - to enable the adopt charge, seed a gleipnir payment connection, then:"
	@echo "      helm upgrade $(RELEASE) . -f $(VALUES) --set adoptions.paymentConnection=<id>"
	@echo "  Then: login -> browse -> adopt (aegis auth, gleipnir vend+charge, talos audit, gjallarhorn notify)."

## down: uninstall the release + dev postgres (keeps the cluster)
down:
	-$(HELM) uninstall '$(RELEASE)'
	-$(KUBECTL) delete -f postgres-dev.yaml

## clean: delete the whole kind cluster
clean:
	-kind delete cluster --name '$(CLUSTER)'

## help: list targets
help:
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/^## /  /'
