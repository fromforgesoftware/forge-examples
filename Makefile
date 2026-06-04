# forge-examples convenience Makefile.
# Delegates the petstore local-deploy targets to the umbrella chart's Makefile
# (examples/petstore/deploy/petstore-platform/Makefile), so you can run them from
# the repo root:
#
#   make dev      # bring the whole petstore-on-forge up in kind, wait for ready
#   make status   # pods + migration jobs
#   make forward  # port-forward the UI + console
#   make down     # uninstall + drop the dev postgres
#   make clean    # delete the kind cluster
#   make help     # list the underlying targets

PETSTORE := examples/petstore/deploy/petstore-platform

.PHONY: dev up cluster ingress deps images postgres install wait status forward next down clean help

dev up cluster ingress deps images postgres install wait status forward next down clean help:
	@$(MAKE) -C $(PETSTORE) $@
