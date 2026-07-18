# Petstore Expanded Example

This example demonstrates [oapi-codegen](https://github.com/oapi-codegen/oapi-codegen) generating server stubs for several different Go HTTP frameworks from a single [OpenAPI 3.0 spec](petstore-expanded.yaml).

Each server variant is fully self-contained, in that it has its own generated types, its own
copy of the trivial in-memory database (a Go map), and its own handler implementation.
The backend is simple enough that copying it into each variant keeps every example 
readable on its own. A single shared test client (`common/client`) can exercise any of the variants over real HTTP.

## Directory Structure

```
petstore-expanded/
├── petstore-expanded.yaml          # Shared OpenAPI spec
├── common/
│   └── client/                     # Single test client, works against any variant
│       ├── main.go                 # CLI test client
│       ├── testclient/
│       │   └── testclient.go       # Reusable CRUD test sequence
│       └── openapi/
│           ├── generate.go         # go:generate for the client
│           ├── client.cfg.yaml     # Codegen config: client + models
│           └── client.gen.go       # Generated HTTP client (+ model types)
├── chi/                            # Chi (net/http compatible)
├── gorilla/                        # Gorilla/mux (net/http compatible)
├── stdhttp/                        # stdlib net/http
├── echo/                           # Echo v4
├── echo-v5/                        # Echo v5
├── gin/                            # Gin
├── fiber/                          # Fiber
├── iris/                           # Iris
└── strict/                         # Strict server (Chi + typed request/response objects)
```

Each server variant follows the same self-contained pattern:
- `api/server.cfg.yaml` — codegen config generating the server interface, model types, and embedded spec
- `api/generate.go` — `//go:generate` directive for the server code
- `api/petstore-server.gen.go` — generated server boilerplate and model types
- `server/store.go` — the in-memory CRUD store (copied into each variant)
- `server/server.go` — hand-written `ServerInterface` implementation backed by the local store
- `server/setup.go` — factory function that creates a fully configured server/app
- `petstore.go` — `main()` wiring (thin wrapper around `setup.go`)

## Generating Code

From the repository root, `make generate` regenerates everything. To regenerate a
single piece from the `examples/` directory:

```sh
# Generate a specific server variant (server + models)
cd examples/petstore-expanded/chi/api && go generate ./...

# Generate the shared test client
cd examples/petstore-expanded/common/client/openapi && go generate ./...
```

## Running a Server

```sh
cd examples/petstore-expanded/chi
go run . --port 8080
```

Replace `chi` with any variant: `gorilla`, `stdhttp`, `echo`, `echo-v5`, `gin`, `fiber`, `iris`, `strict`.

## Test Client

A single client executable verifies the behavior of any variant over real HTTP.
Start a server in one terminal, then point the client at it from another:

```sh
# Terminal 1: start any server variant
cd examples/petstore-expanded/chi && go run . --port 8080

# Terminal 2: run the test client against it
cd examples && go run ./petstore-expanded/common/client/ --port 8080
```

The client verifies: add pets, find by ID, 404 on missing pet, list/filter by tag, delete, and empty list after deletion.

## Contract Testing with Specmatic

The `stdhttp` variant has an opt-in [Specmatic](https://specmatic.io/) contract test for `petstore-expanded.yaml`. The normal test suite does not invoke Specmatic.

### Prerequisites

- Docker
- Go and GNU Make

The test runs the pinned `specmatic/specmatic:2.50.1` image, following Specmatic's current [Docker installation](https://docs.specmatic.io/download) and [CLI quick-start](https://docs.specmatic.io/getting_started/cli_quick_start) guidance. It does not use a host-installed CLI, alias, JDK, Homebrew, or npm package.

### Running Locally

```sh
make specmatic-test
```

The Go test starts `server.NewServer()`, seeds pets through its public `POST /pets` API, and makes the host service reachable from the container. `schemaResiliencyTests: all` enables generative negative scenarios that mutate required fields, types, and parameters. Specmatic also runs positive, 400, and 404 scenarios for all four OpenAPI operations. The checked-in overlay makes the broad `default` responses explicit for testing, so the report contains real HTTP statuses rather than Specmatic's internal `1000` marker. The configured gate requires 100% API coverage and zero missed operations.

The HTML report is written to `examples/petstore-expanded/stdhttp/build/reports/specmatic/test/html/index.html`. The generated report directory is ignored by Git, and the dedicated GitHub Actions workflow uploads it as the `specmatic-report` artifact for seven days.

This test verifies externally observable HTTP behavior against the OpenAPI contract; it does not replace Go tests for internal business rules. The same pattern works for another service by pointing the Config V3 source at its OpenAPI document, starting that service in the harness, and providing deterministic state through examples or a dictionary.

Spring Actuator does not apply to this Go `net/http` server. For actual implementation coverage, the harness instead exposes the generated server's embedded OpenAPI document at a test-only URL and configures Specmatic's supported `swaggerUrl`. No actuator or documentation route is added to the example server itself.

The dedicated GitHub Actions workflow runs the same Make target. It is path-filtered to the target, workflow, examples module metadata, and Petstore stdhttp contract-test files.
