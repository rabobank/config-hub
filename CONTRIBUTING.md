# Contributing to Config Hub

Thank you for your interest in contributing to Config Hub! This guide provides all the information you need to set up a development environment, understand the codebase, and contribute effectively.

## Welcome

Config Hub is a Spring Cloud Config Server-compatible configuration server written in Go for Cloud Foundry environments. We welcome contributions in the form of bug reports, feature requests, documentation improvements, and code contributions.

## Development Setup

### Prerequisites

- **Go 1.23 or later**: [Download Go](https://go.dev/dl/)
- **Git**: For version control and repository access
- **Optional**: Docker for container testing

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/rubionic/config-hub.git
cd config-hub

# Install dependencies
go mod download

# Build the binary
./.github/scripts/build.sh

# Run tests
./.github/scripts/runTests.sh
```

### Running Locally

```bash
# Set required environment variables
export CLIENT_ID="test-client"
export CLIENT_SECRET="test-secret"
export OPENID_URL="https://login.cf.example.com"
export SERVICE_INSTANCE_ID="test-instance-id"
export CH_SOURCES='[{"type":"git","uri":"https://github.com/your-org/config-repo.git"}]'

# Run the server
go run main.go credentials.go

# Or run the built binary
./config-hub
```

The server will start on `http://localhost:8080`.

## Project Structure

```
config-hub/
├── cfg/                      # Configuration management
│   └── config.go            # App configuration and version info
│
├── domain/                   # Domain models and types
│   ├── configuration.go     # Configuration request/response types
│   ├── credentials.go       # Git credential types
│   ├── credhub-config.go    # CredHub configuration types
│   ├── git-config.go        # Git source configuration types
│   ├── response.go          # API response types
│   ├── secrets.go           # Secret management types
│   └── util.go              # Domain utility functions
│
├── server/                   # HTTP server and handlers
│   ├── server.go            # Main server setup and endpoint handlers
│   ├── authorization.go     # UAA/OpenID authorization logic
│   └── health.go            # Health check endpoint
│
├── sources/                  # Configuration source implementations
│   ├── spi/                 # Source interface definition
│   │   └── source.go        # Source SPI interface
│   │
│   ├── git_source/          # Git repository source
│   │   ├── source.go        # Git source implementation
│   │   ├── source_test.go   # Git source tests
│   │   ├── repository.go    # Git operations (clone, fetch, checkout)
│   │   ├── credentials.go   # Git credential management
│   │   ├── spnCredentials.go    # Azure SPN authentication
│   │   └── miWifCredentials.go  # Azure MI WIF authentication
│   │
│   ├── credhub_source/      # CredHub source
│   │   ├── source.go        # CredHub source implementation
│   │   └── secrets.go       # CredHub secret operations
│   │
│   ├── source.go            # Source factory and management
│   ├── properties.go        # Properties file parsing
│   └── dashboard.go         # Dashboard HTML generation
│
├── util/                     # Utility functions
│   ├── Http.go              # HTTP utilities
│   ├── Map.go               # Map manipulation utilities
│   ├── Map_test.go          # Map utilities tests
│   └── Util.go              # General utilities
│
├── .github/
│   └── scripts/
│       ├── build.sh         # Build script (creates binaries)
│       └── runTests.sh      # Test script (vet + test)
│
├── main.go                   # Application entry point (server mode)
├── credentials.go            # Git credential helper mode
├── go.mod                    # Go module dependencies
└── go.sum                    # Dependency checksums
```

### Architecture Overview

```
┌─────────────────────────────────────────┐
│         HTTP Server (go-we)              │
│  ┌────────────────────────────────────┐ │
│  │  Security Filter                    │ │
│  │  - OpenID/UAA Authentication       │ │
│  │  - Bearer Token Validation         │ │
│  │  - Authorization Rules             │ │
│  └────────────────────────────────────┘ │
│  ┌────────────────────────────────────┐ │
│  │  Endpoint Handlers                  │ │
│  │  - Config Retrieval                │ │
│  │  - Secret Management               │ │
│  │  - Cache Management                │ │
│  │  - Dashboard                       │ │
│  └────────────────────────────────────┘ │
└──────────────┬──────────────────────────┘
               │
        ┌──────▼──────┐
        │   Sources   │
        └──────┬──────┘
               │
        ┏━━━━━━┻━━━━━━━┓
        ┃               ┃
  ┌─────▼──────┐  ┌────▼─────┐
  │ Git Source │  │ CredHub  │
  │            │  │  Source  │
  │ - Clone    │  │          │
  │ - Fetch    │  │ - mTLS   │
  │ - Checkout │  │ - OAuth2 │
  │ - Parse    │  │ - CRUD   │
  └────────────┘  └──────────┘
```

## How Configuration Resolution Works

### Request Flow

1. **HTTP Request Received**: Client requests configuration via `GET /{app}/{profiles}` or `GET /{app}-{profiles}.yml`
2. **Authentication**: Bearer token validated against UAA/OpenID provider
3. **Authorization**: Scopes and permissions checked (see [README.md](README.md) for authorization rules)
4. **Source Query**: Each configured source queried in order
5. **File Search**: Files searched using naming conventions and search paths
6. **Property Merging**: Properties merged with priority (last source wins)
7. **Response Formatting**: Response formatted as JSON, YAML, or Properties
8. **Caching**: Git repositories cached according to `fetchCacheTtl`

### File Search Patterns

For each search path, the following files are searched (in order):
- `{app}-{profile}.{ext}` (e.g., `myapp-production.yml`)
- `{app}.{ext}` (e.g., `myapp.yml`)
- `application-{profile}.{ext}` (e.g., `application-production.yml`)
- `application.{ext}` (e.g., `application.yml`)

Supported extensions: `.yml`, `.yaml`, `.properties`, `.json`

### Search Path Placeholders

- `{application}`: Replaced with the application name from the request
- `{profile}`: Replaced with each profile name (e.g., `production`, `cloud`)
- `*`: Wildcard for directory matching (e.g., `*/configs` matches `app1/configs`, `app2/configs`)

### Caching Strategy

- **Git Repositories**: Cloned on first request, fetched on subsequent requests if cache expired
- **Cache Location**: `{baseDir}/config-repo-{index}/`
- **Cache TTL**: Configurable via `fetchCacheTtl` (minimum 60 seconds)
- **Cache Invalidation**: Explicit via `DELETE /cache` or automatic TTL expiration
- **Failure Handling**: If `failOnFetch: false`, falls back to cached version on fetch failure

## Code Guidelines

Config Hub follows Go best practices and conventions. Please adhere to these guidelines when contributing.

### Import Organization

Organize imports in three groups with blank lines between:
```go
import (
    // Standard library
    "encoding/json"
    "fmt"
    "os"
    
    // Third-party packages
    "github.com/gomatbase/go-log"
    "gopkg.in/yaml.v2"
    
    // Internal packages
    "github.com/rabobank/config-hub/domain"
    "github.com/rabobank/config-hub/util"
)
```

### Naming Conventions

- **Public exports**: `PascalCase` (e.g., `GitConfig`, `FetchCache`)
- **Private/internal**: `camelCase` (e.g., `fetchRepo`, `parseYaml`)
- **Error variables**: `e` (short, conventional)
- **Acronyms**: All caps in type names (e.g., `UAA`, `URL`), lowercase in variable names (e.g., `uaaUrl`)

### Error Handling

Check errors immediately after function calls:
```go
// Good
data, e := os.ReadFile(path)
if e != nil {
    return e
}

// Bad - don't shadow error variables
data, err := os.ReadFile(path)
if err2 := someFunc(); err2 != nil {
    return err2
}
```

### Types & Formatting

- Use `any` instead of `interface{}` for generic types
- Run `go fmt` before committing (automatically formats code)
- Use struct field tags for JSON serialization:
  ```go
  type Config struct {
      ClientID     string `json:"clientId"`
      ClientSecret string `json:"clientSecret"`
  }
  ```

### Logging

Use the `github.com/gomatbase/go-log` package with appropriate levels:
```go
import log "github.com/gomatbase/go-log"

log.DEBUG("Fetching repository: %s", uri)
log.INFO("Server started on port %s", port)
log.ERROR("Failed to parse config: %v", e)
log.CRITICAL("Unable to connect to UAA: %v", e)
```

## Testing

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./sources/git_source/
go test ./util/

# Run specific test
go test ./sources/git_source/ -run TestReadYamlFile2
go test ./util/ -run TestGet

# Verbose output
go test -v ./...

# With coverage
go test -cover ./...

# Complete test suite (vet + test)
./.github/scripts/runTests.sh
```

### Writing Tests

Place test files in the same package as the code being tested (`*_test.go`).

**Example test structure:**
```go
package git_source

import "testing"

func TestParseYaml(t *testing.T) {
    // Table-driven test
    tests := []struct {
        name     string
        input    string
        expected map[string]any
    }{
        {"simple", "key: value", map[string]any{"key": "value"}},
        {"nested", "outer:\n  inner: value", map[string]any{"outer": map[string]any{"inner": "value"}}},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, e := parseYaml(tt.input)
            if e != nil {
                t.Errorf("unexpected error: %v", e)
            }
            // Assert result matches expected
        })
    }
}
```

**Best practices:**
- Use table-driven tests for multiple scenarios
- Test both success and error paths
- Use descriptive test names
- Keep tests focused and isolated

## Building & Release

### Build Commands

```bash
# Development build
go build -o config-hub

# Production build with version info
./.github/scripts/build.sh

# Cross-platform builds
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o config-hub-linux-amd64
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o config-hub-darwin-amd64
```

### Build Script

The `.github/scripts/build.sh` script creates production builds with version information:

```bash
VERSION=1.0.0 COMMIT=$(git rev-parse HEAD) ./.github/scripts/build.sh
```

This script builds Linux AMD64 binaries and packages them into tar archives with version information embedded.

### Version Information

Version and commit information are embedded during build:
```bash
-ldflags "-X github.com/rabobank/config-hub/cfg.Version=${VERSION} \
          -X github.com/rabobank/config-hub/cfg.Commit=${COMMIT}"
```

## Contributing Workflow

### 1. Fork and Clone

```bash
# Fork the repository on GitHub
# Clone your fork
git clone https://github.com/YOUR_USERNAME/config-hub.git
cd config-hub
```

### 2. Create a Feature Branch

```bash
git checkout -b feature/your-feature-name
```

### 3. Make Changes

- Write code following the code guidelines above
- Add tests for new functionality
- Update documentation if needed
- Run tests and linting: `./.github/scripts/runTests.sh`

### 4. Commit Changes

Use conventional commit format:
```bash
git commit -m "feat: add support for new authentication method"
git commit -m "fix: resolve cache invalidation issue"
git commit -m "docs: update configuration examples"
git commit -m "refactor: simplify git credential handling"
```

### 5. Run Quality Checks

```bash
# Run vet (static analysis)
go vet ./...

# Run tests
go test ./...

# Or run both with the test script
./.github/scripts/runTests.sh
```

### 6. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a pull request on GitHub with:
- Clear title describing the change
- Description explaining the motivation and approach
- Link to related issues (e.g., "Fixes #123")

## Troubleshooting Development

### Build Issues

**Problem**: `cannot find package`
```bash
# Solution: Ensure dependencies are downloaded
go mod download
go mod tidy
```

**Problem**: Version/commit not showing in binary
```bash
# Solution: Use build script or set ldflags
VERSION=1.0.0 COMMIT=$(git rev-parse HEAD) ./.github/scripts/build.sh
```

### Test Failures

**Problem**: Tests fail with `connection refused`
```bash
# Solution: Some tests may require external services
# Check test setup and mock requirements
```

**Problem**: `undefined: someFunc`
```bash
# Solution: Import missing packages
go get package-name
go mod tidy
```

### Git Credential Helper Testing

To test git credential helper mode locally:
```bash
# Configure git to use local binary
git config --global credential.helper '!/path/to/config-hub credentials test-repo'

# Test credential retrieval
echo -e "host=github.com\nprotocol=https" | ./config-hub credentials test-repo get
```

## Additional Resources

- **Go Documentation**: [https://go.dev/doc/](https://go.dev/doc/)
- **Spring Cloud Config**: [https://spring.io/projects/spring-cloud-config](https://spring.io/projects/spring-cloud-config)
- **Cloud Foundry Docs**: [https://docs.cloudfoundry.org/](https://docs.cloudfoundry.org/)
- **CredHub Documentation**: [https://docs.cloudfoundry.org/credhub/](https://docs.cloudfoundry.org/credhub/)

---

Thank you for contributing to Config Hub! 🚀
