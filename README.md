# forge-examples

Reference microservices monorepo showing how to build apps on the forge stack
(go-kit, JSON:API, aegis-protected REST, gleipnir credential vend). Petstore:
a `catalog` service + an `adoptions` service.

> Local dev uses `replace` directives to the sibling `../go-kit` and `../gleipnir`
> repos. For CI / standalone use, drop the replaces and consume the published
> module versions.
