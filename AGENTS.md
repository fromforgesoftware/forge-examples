# forge-examples — reference monorepo of examples

Shows how to build apps on the forge stack (go-kit, JSON:API, aegis-protected REST,
gleipnir credential vend). Examples live under `examples/`. `examples/petstore`:
a `catalog` service + an `adoptions` service.

## Commands (run inside the example, e.g. `cd examples/petstore`)
- Build: `GOWORK=off go build ./...`
- Test: `GOWORK=off go test ./...`

## Notes
- Each example is its own Go module, consuming published `go-kit` + `gleipnir`
  (pinned in go.mod; no `replace`, no `go.work` — build with `GOWORK=off`).

## Boundaries
- One-line conventional commits. NEVER commit secrets. NEVER hand-edit generated
  `*.pb.go` — use `buf generate`. No dependabot.
