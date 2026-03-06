# Contributing to gflow

Thanks for your interest in contributing to gflow! This document covers how to get started.

## Development Setup

```bash
# Clone the repo
git clone https://github.com/abhinavcdev/gflow.git
cd gflow

# Build
make build

# Run tests
make test

# Install locally
make install
```

**Requirements:**
- Go 1.22+
- Git 2.x+

## Project Structure

```
gflow/
├── main.go                      # Entry point
├── cmd/                         # CLI commands (one file per command)
│   ├── root.go                  # Root command, flag registration
│   ├── init.go                  # gflow init
│   ├── start.go                 # gflow start
│   ├── pr.go                    # gflow pr (flagship)
│   ├── commit.go                # gflow commit
│   ├── finish.go                # gflow finish
│   ├── sync.go                  # gflow sync
│   ├── release.go               # gflow release
│   ├── status.go                # gflow status
│   ├── config.go                # gflow config
│   ├── checkout.go              # gflow checkout
│   ├── log.go                   # gflow log
│   ├── clean.go                 # gflow clean
│   ├── diff.go                  # gflow diff
│   ├── reopen.go                # gflow reopen
│   ├── helpers.go               # Shared helpers
│   └── helpers_test.go          # Command-level tests
├── internal/
│   ├── config/                  # Configuration (.gflow.yml)
│   │   ├── config.go
│   │   └── config_test.go
│   ├── git/                     # Git CLI wrapper
│   │   ├── git.go
│   │   └── git_test.go
│   ├── provider/                # Provider abstraction
│   │   ├── provider.go          # Interface
│   │   ├── github.go            # GitHub implementation
│   │   ├── github_test.go       # Mock HTTP tests
│   │   ├── gitlab.go            # GitLab implementation
│   │   └── bitbucket.go         # Bitbucket implementation
│   └── ui/                      # Terminal UI (colors, prompts, spinners)
│       └── ui.go
├── examples/
│   └── .gflow.yml               # Example configuration
├── .github/workflows/ci.yml     # GitHub Actions CI
├── .goreleaser.yml              # Release automation
├── Makefile                     # Build automation
└── README.md                    # Documentation
```

## How to Contribute

### Bug Reports

Open an issue with:
- gflow version (`gflow version`)
- OS and shell
- Steps to reproduce
- Expected vs actual behavior

### Feature Requests

Open an issue describing:
- The use case
- Proposed CLI interface (command name, flags)
- How it fits into existing workflow

### Pull Requests

1. Fork the repo
2. Create a feature branch: `gflow start feature your-feature` (or `git checkout -b feature/your-feature`)
3. Make changes
4. Add/update tests
5. Run `make test` and `make lint`
6. Submit a PR

#### Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use `internal/` packages for non-exported code
- One command per file in `cmd/`
- Tests live next to the code they test (`_test.go`)
- Use the `ui` package for all terminal output (colors, spinners, prompts)

#### Adding a New Command

1. Create `cmd/yourcommand.go`
2. Define a `var yourCmd = &cobra.Command{...}`
3. Register in `cmd/root.go`: `rootCmd.AddCommand(yourCmd)`
4. Add tests in `cmd/helpers_test.go` or a new test file
5. Update `README.md`

#### Adding a New Provider

1. Create `internal/provider/yourprovider.go`
2. Implement the `Provider` interface
3. Add the case to `provider.New()` in `provider.go`
4. Add mock HTTP tests
5. Update README provider table

### Commit Convention

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add new command
fix: handle edge case in branch parsing
docs: update README examples
test: add provider mock tests
refactor: extract shared helpers
```

### Testing

- Unit tests: `go test ./...`
- Race detection: `go test -race ./...`
- Coverage: `make test-cover`
- Provider tests use `httptest.NewServer` for mock HTTP

## License

By contributing, you agree that your contributions will be licensed under the MIT License.
