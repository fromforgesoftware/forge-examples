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
HELM    := helm --kube-context $(CTX) -n $(NAMESPACE)

.PHONY: dev up cluster deps postgres install wait status forward next down clean help

## dev: full local bring-up (cluster -> deps -> postgres -> install -> wait -> next)
dev up: cluster deps postgres install wait status next

## cluster: create the kind cluster if it doesn't exist
cluster:
	@kind get clusters 2>/dev/null | grep -qx '$(CLUSTER)' \
		&& echo "==> kind cluster '$(CLUSTER)' already exists" \
		|| { echo "==> creating kind cluster '$(CLUSTER)'"; kind create cluster --name '$(CLUSTER)'; }

## deps: pull the published forge-platform chart + build local subcharts
deps:
	@echo "==> helm dependency build (pulls forge-platform from ghcr)"
	@$(HELM) dependency build .

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

## forward: port-forward the UI (8080) and forge console (8081) for quick access
forward:
	@echo "UI:      http://localhost:8080   console: http://localhost:8081"
	@echo "(NOTE: the SPA's OIDC redirect uses the ingress hosts in envs/local;"
	@echo " for port-forward testing set petstore-ui.config.* to localhost URLs and"
	@echo " register a matching redirect URI on the aegis client.)"
	@$(KUBECTL) port-forward svc/petstore-ui 8080:80 & \
	 $(KUBECTL) port-forward svc/$(RELEASE)-forge 8081:8080 & wait

## next: print the steps that can't be automated
next:
	@echo ""
	@echo "==> petstore-platform is up. Manual steps to finish (see README.md):"
	@echo "  1. aegis realm + PUBLIC OIDC client:"
	@echo "       envs/local asks aegis to bootstrap realm 'petstore' + client 'petstore-ui'."
	@echo "       If your aegis only bootstraps one realm, create realm 'petstore' + a public"
	@echo "       PKCE client 'petstore-ui' (redirect http://petstore.localhost/callback) by hand."
	@echo "  2. gleipnir payment connection — create one, then:"
	@echo "       helm upgrade $(RELEASE) . -f $(VALUES) --set adoptions.paymentConnection=<id>"
	@echo "  3. Access: 'make forward' (port-forward) or set up ingress + *.localhost hosts."
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
