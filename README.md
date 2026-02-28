# go-framework

A way to structure your application modules.

<img width="573" height="435" alt="Screenshot 2025-09-26 at 10 14 57" src="https://github.com/user-attachments/assets/3e67e0f7-dd48-4ca0-98e5-80d216e86749" />

## Command line tool
There's a command line tool for running simple command modules (`framework.CommandModule`).

Install:

```sh
go install github.com/roboslone/go-framework/v2/cmd/fexec@latest
```

Or use as [devcontainers](https://containers.dev) feature:

```json
{
	"features": {
		"ghcr.io/roboslone/go-framework/fexec:1": {}
	}
}
```

Example config: [.fexec.yaml](https://github.com/roboslone/go-framework/blob/main/.fexec.yaml)

Run:

```sh
fexec --help

# Usage of fexec:
#   -c string
#         Path to config file (default ".fexec.yaml")
# 
# Available modules:
#         ci
#                 depends on lint, test
#         install
#                 $ go get
#         lint
#                 $ golangci-lint run --no-config .
#                 depends on install
#         test
#                 $ go test ./...
#                 depends on install
#         pre-commit
#                 depends on lint, test
```

## Module interfaces
Available interfaces can be found in `module.go`:

```go
Dependent
Preparable[State any]
Startable[State any]
Awaitable[State any]
Cleanable[State any]
```
