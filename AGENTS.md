# forge-examples — reference microservices monorepo

Shows how to build apps on the forge stack (go-kit, JSON:API, aegis-protected REST,
gleipnir credential vend). Petstore: a `catalog` service + an `adoptions` service.

## Commands
- Build: `go build ./...`
- Test: `go test ./...`

## Notes
- Consumes published `go-kit` + `gleipnir` (earlier local `replace` directives dropped).

## Boundaries
- One-line conventional commits. NEVER commit secrets. NEVER hand-edit generated
  `*.pb.go` — use `buf generate`. No dependabot.
