# Config Hub

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?logo=go)](https://go.dev/)

A lightweight, Spring Cloud Config Server-compatible configuration server written in Go, designed specifically for Cloud Foundry environments. Config Hub provides centralized configuration management with support for multiple backends including Git repositories and CredHub secret storage.

## Features

- **Spring Cloud Config Server Compatible**: Drop-in replacement supporting standard endpoints and response formats
- **Multiple Configuration Sources**:
  - Git repositories (HTTP/HTTPS/SSH) with advanced authentication options
  - CredHub integration for secure secret management
- **Advanced Git Authentication**:
  - Username/Password authentication
  - SSH private key authentication
  - Azure Service Principal (SPN) authentication
  - Azure Managed Identity with Workload Identity Federation (WIF)
- **Cloud Foundry Integration**:
  - Native CF environment support
  - UAA/OpenID authentication and authorization
  - Service instance-based access control
- **Flexible Configuration**:
  - Multiple profiles support
  - Label/branch selection
  - Search paths with wildcard support
  - Configuration file formats: YAML, Properties, JSON
- **Caching & Performance**:
  - Configurable fetch cache TTL
  - Cache invalidation endpoint
- **Git Credential Helper Mode**: Can act as a git credential helper for secure credential retrieval
- **Management Dashboard**: Web-based dashboard for monitoring configuration sources

## Table of Contents

- [Quick Start](#quick-start)
- [Installation](#installation)
- [Configuration](#configuration)
  - [Git Source Configuration](#git-source-configuration)
  - [CredHub Source Configuration](#credhub-source-configuration)
- [Usage](#usage)
  - [API Endpoints](#api-endpoints)
  - [Request Examples](#request-examples)
- [Authentication & Authorization](#authentication--authorization)
- [Git Credential Helper Mode](#git-credential-helper-mode)
- [Deployment](#deployment)
- [Development](#development)
- [License](#license)

## Quick Start

### Server Mode

```bash
# Set required environment variables
export CLIENT_ID="your-uaa-client"
export CLIENT_SECRET="your-uaa-secret"
export OPENID_URL="https://login.cf.example.com"
export SERVICE_INSTANCE_ID="your-service-instance-id"
export CH_SOURCES='[
  {
    "type": "git",
    "uri": "https://github.com/your-org/config-repo.git",
    "searchPaths": ["configs", "apps/{application}"]
  }
]'

# Run the server
./config-hub
```

The server will start on port 8080 (configurable via `PORT` environment variable).

### Git Credential Helper Mode

```bash
# Configure git to use config-hub as credential helper
git config credential.helper '!config-hub credentials <repo>'

# Git will now request credentials from config-hub
git clone https://github.com/your-org/private-repo.git
```

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/rubionic/config-hub.git
cd config-hub

# Build for Linux AMD64
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
  -o config-hub \
  -ldflags "-X github.com/rabobank/config-hub/cfg.Version=1.0.0 -X github.com/rabobank/config-hub/cfg.Commit=$(git rev-parse HEAD)"
```

### Using Pre-built Binaries

Download the latest release from the [releases page](https://github.com/rubionic/config-hub/releases).

## Configuration

Config Hub is configured via environment variables and JSON configuration.

### Required Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `CLIENT_ID` | UAA/OpenID client ID | `config-hub-client` |
| `CLIENT_SECRET` | UAA/OpenID client secret | `secret123` |
| `OPENID_URL` | OpenID provider URL | `https://login.cf.example.com` |
| `SERVICE_INSTANCE_ID` | CF service instance ID for authorization | `abc123-def456` |
| `CH_SOURCES` | JSON array of source configurations | See below |

### Optional Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `LOG_LEVEL` | `DEBUG` | Logging level (DEBUG, INFO, ERROR, CRITICAL) |
| `UAA_URL` | | UAA server URL (for user enrichment) |
| `CF_URL` | `https://api.cf.internal` | Cloud Foundry API URL |
| `CREDHUB-REF` | | CredHub reference path for retrieving configuration |

### Git Source Configuration

Git sources support fetching configuration from Git repositories with flexible authentication options.

#### Basic Git Configuration

```json
{
  "type": "git",
  "uri": "https://github.com/your-org/config-repo.git",
  "defaultLabel": "main",
  "searchPaths": ["config", "{application}", "{profile}"],
  "skipSslValidation": false,
  "failOnFetch": false,
  "fetchCacheTtl": 60
}
```

#### Configuration Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | ✓ | Must be `"git"` |
| `uri` | string | ✓ | Git repository URL (HTTP/HTTPS/SSH) |
| `defaultLabel` | string | | Default branch/tag (default: `"master"`) |
| `searchPaths` | []string | | Paths to search for config files |
| `deepClone` | bool | | Perform deep clone instead of shallow |
| `skipSslValidation` | bool | | Skip SSL certificate validation |
| `failOnFetch` | bool | | Fail requests if fetch fails (vs. using cached) |
| `fetchCacheTtl` | int | | Cache TTL in seconds (min: 60, default: 60) |

#### Search Paths

Search paths support placeholders and wildcards:

- `{application}`: Replaced with application name
- `{profile}`: Replaced with profile name
- `*`: Wildcard for directory matching

**Examples:**
```json
"searchPaths": [
  "apps/{application}/{profile}",
  "apps/{application}",
  "common/{profile}",
  "common",
  "*/configs"
]
```

#### Authentication Options

##### Username/Password

```json
{
  "type": "git",
  "uri": "https://github.com/your-org/config-repo.git",
  "username": "git-user",
  "password": "personal-access-token"
}
```

##### SSH Private Key

```json
{
  "type": "git",
  "uri": "git@github.com:your-org/config-repo.git",
  "privateKey": "-----BEGIN RSA PRIVATE KEY-----\n..."
}
```

##### Azure Service Principal (SPN)

```json
{
  "type": "git",
  "uri": "https://dev.azure.com/org/project/_git/repo",
  "azTenantId": "tenant-id",
  "azClient": "client-id",
  "azSecret": "client-secret"
}
```

**With CredHub Secret Reference:**
```json
{
  "type": "git",
  "uri": "https://dev.azure.com/org/project/_git/repo",
  "azTenantId": "tenant-id",
  "azClient": "client-id",
  "azSecret-credhub-ref": "/path/to/secret",
  "azSecret-credhub-client": "credhub-client",
  "azSecret-credhub-secret": "credhub-secret"
}
```

##### Azure Managed Identity with WIF

```json
{
  "type": "git",
  "uri": "https://dev.azure.com/org/project/_git/repo",
  "azTenantId": "tenant-id",
  "azMiId": "managed-identity-name",
  "azMiWifIssuer": "https://issuer.example.com",
  "username": "wif-token-username",
  "password": "wif-token-password"
}
```

### CredHub Source Configuration

CredHub sources provide secure secret storage with hierarchical organization.

```json
{
  "type": "credhub",
  "prefix": "/config-hub/",
  "client": "credhub-client",
  "secret": "credhub-secret"
}
```

#### Configuration Options

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | ✓ | Must be `"credhub"` |
| `prefix` | string | ✓ | Path prefix for secrets |
| `client` | string | | OAuth2 client ID (omit for mTLS) |
| `secret` | string | | OAuth2 client secret (omit for mTLS) |

**Note**: If neither `client` nor `secret` is provided, mTLS authentication is used.

#### Secret Hierarchy

Secrets are organized as: `{prefix}{app}/{profile}/{label}/secrets`

**Example:**
- Prefix: `/config-hub/`
- App: `myapp`
- Profile: `production`
- Label: `master`
- Path: `/config-hub/myapp/production/master/secrets`

### Complete Configuration Example

```bash
export CH_SOURCES='[
  {
    "type": "git",
    "uri": "https://github.com/your-org/config-repo.git",
    "defaultLabel": "main",
    "searchPaths": ["apps/{application}", "common"],
    "username": "git-user",
    "password": "ghp_token",
    "fetchCacheTtl": 120
  },
  {
    "type": "credhub",
    "prefix": "/config-hub/",
    "client": "config-hub-client",
    "secret": "secret"
  }
]'
```

## Usage

### API Endpoints

#### Configuration Endpoints (Spring Cloud Config Compatible)

**Get configuration:**
```
GET /{application}/{profiles}
GET /{application}/{profiles}/{label}
```

**Get formatted configuration:**
```
GET /{application}-{profiles}.json
GET /{application}-{profiles}.yml
GET /{application}-{profiles}.yaml
GET /{application}-{profiles}.properties
```

#### CredHub Management Endpoints

**Add secrets:**
```
POST /secrets
POST /secrets/add
```

**Delete secrets:**
```
DELETE /secrets
DELETE /secrets/delete
```

**List secrets:**
```
GET /secrets
GET /secrets/list
```

#### Management Endpoints

**Health check:**
```
GET /health
```

**Server info:**
```
GET /info
```

**Dashboard:**
```
GET /dashboard
```

**Cache management:**
```
DELETE /cache
```

**Git credentials (helper mode):**
```
POST /credentials
```

### Request Examples

#### Fetch Configuration

```bash
# Get configuration for 'myapp' with 'production' profile
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/myapp/production

# Get configuration from specific branch
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/myapp/production/feature-branch

# Get as YAML
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/myapp-production.yml

# Multiple profiles (priority: last to first in response)
curl -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/myapp/production,cloud
```

#### Response Format

```json
{
  "name": "myapp",
  "profiles": ["production"],
  "label": "main",
  "propertySources": [
    {
      "name": "credhub-myapp-production-main",
      "source": {
        "database.password": "secret123",
        "api.key": "abc-xyz"
      }
    },
    {
      "name": "/path/to/repo/apps/myapp/myapp-production.yml",
      "source": {
        "server": {
          "port": 8080
        },
        "database": {
          "url": "jdbc:postgresql://db:5432/myapp"
        }
      }
    }
  ]
}
```

#### Manage CredHub Secrets

```bash
# Add secrets
curl -X POST \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "app": "myapp",
    "profile": "production",
    "label": "main",
    "secrets": {
      "database.password": "secret123",
      "api.key": "abc-xyz"
    }
  }' \
  http://localhost:8080/secrets

# List secrets
curl -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/secrets?app=myapp&profile=production"

# Delete secrets
curl -X DELETE \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "app": "myapp",
    "profile": "production",
    "keys": ["api.key"]
  }' \
  http://localhost:8080/secrets
```

#### Clear Cache

```bash
curl -X DELETE \
  -H "Authorization: Bearer $TOKEN" \
  http://localhost:8080/cache
```

## Authentication & Authorization

Config Hub integrates with Cloud Foundry UAA/OpenID for authentication and authorization.

### Authentication Methods

1. **Bearer Token**: Standard OAuth2 bearer token in `Authorization` header
2. **SSO**: Browser-based single sign-on for dashboard access

### Authorization Rules

| Endpoint | Requirements |
|----------|--------------|
| `/health`, `/info` | Anonymous access |
| `/credentials` | Localhost only |
| `/secrets/*`, `/cache`, `/dashboard` | `cloud_controller.admin` scope OR space developer role |
| `/*/` (config endpoints) | `config_hub_{service-instance-id}.read` scope |

### Space Developer Check

The server verifies space developer permissions by checking CF API permissions for the service instance:

```
GET {CF_URL}/v3/service_instances/{SERVICE_INSTANCE_ID}/permissions
```

Users must have `manage` permission on the service instance to access management endpoints.

## Git Credential Helper Mode

Config Hub can act as a git credential helper, allowing Git operations to retrieve credentials securely.

### Setup

```bash
# Configure git globally
git config --global credential.helper '!config-hub credentials repo-path'

# Or for a specific repository
cd /path/to/repo
git config credential.helper '!config-hub credentials $(pwd)'
```

### Usage

```bash
# Git will automatically request credentials when needed
git clone https://github.com/your-org/private-repo.git
git pull
git fetch

# Credentials are retrieved from CredHub based on repository host
```

### How It Works

1. Git invokes: `config-hub credentials <repo> get`
2. Config Hub reads key-value pairs from stdin (host, protocol, etc.)
3. Credentials are fetched from CredHub or configured sources
4. Credentials are written to stdout in git credential helper format

**Note**: Only the `get` action is processed; `store` and `erase` are no-ops.

## Deployment

### Cloud Foundry

#### manifest.yml Example

```yaml
applications:
  - name: config-hub
    memory: 256M
    instances: 2
    buildpacks:
      - binary_buildpack
    command: ./config-hub
    env:
      CLIENT_ID: ((client-id))
      CLIENT_SECRET: ((client-secret))
      OPENID_URL: https://login.sys.cf.example.com
      SERVICE_INSTANCE_ID: ((service-instance-id))
      LOG_LEVEL: INFO
      CH_SOURCES: |
        [
          {
            "type": "git",
            "uri": "https://github.com/your-org/config-repo.git",
            "username": "git-user",
            "password": ((git-token))
          }
        ]
    services:
      - config-hub-credhub
```

#### Push to Cloud Foundry

```bash
# Build the binary
./.github/scripts/build.sh

# Push to CF
cf push

# Bind to CredHub service (optional)
cf bind-service config-hub config-hub-credhub
cf restage config-hub
```

### Docker

```dockerfile
FROM alpine:latest

RUN apk add --no-cache ca-certificates git

COPY config-hub /usr/local/bin/
RUN chmod +x /usr/local/bin/config-hub

ENV PORT=8080
EXPOSE 8080

CMD ["config-hub"]
```

```bash
# Build
docker build -t config-hub:latest .

# Run
docker run -p 8080:8080 \
  -e CLIENT_ID=your-client \
  -e CLIENT_SECRET=your-secret \
  -e OPENID_URL=https://login.cf.example.com \
  -e SERVICE_INSTANCE_ID=instance-id \
  -e CH_SOURCES='[...]' \
  config-hub:latest
```

## Development

### Prerequisites

- Go 1.24 or later
- Git

### Build

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Run linter
go vet ./...

# Build
go build -o config-hub

# Or use the build script
./.github/scripts/build.sh
```

### Project Structure

```
config-hub/
├── cfg/              # Configuration management
├── domain/           # Domain models and types
│   ├── configuration.go
│   ├── credentials.go
│   ├── credhub-config.go
│   ├── git-config.go
│   ├── response.go
│   └── secrets.go
├── server/           # HTTP server and handlers
│   ├── server.go
│   ├── authorization.go
│   └── health.go
├── sources/          # Configuration source implementations
│   ├── spi/          # Source interface
│   ├── git_source/   # Git repository source
│   └── credhub_source/ # CredHub source
├── util/             # Utility functions
├── main.go           # Application entry point
└── credentials.go    # Git credential helper
```

### Code Guidelines

- **Imports**: Standard library → third-party → internal packages
- **Naming**: PascalCase for public, camelCase for private
- **Error handling**: Use `e` for error variables, check immediately
- **Logging**: Use `github.com/gomatbase/go-log` with appropriate levels

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests for specific package
go test ./sources/git_source/

# Run with verbose output
go test -v ./...

# Run with coverage
go test -cover ./...
```

### Testing Script

```bash
# Run the complete test suite (vet + test)
./.github/scripts/runTests.sh
```

## Architecture

### Component Overview

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

### Configuration Resolution

1. **Request received** with app, profiles, and optional label
2. **Sources queried** in order of configuration
3. **Files searched** using search paths and naming conventions:
   - `{app}-{profile}.{ext}`
   - `{app}.{ext}`
   - `application-{profile}.{ext}`
   - `application.{ext}`
4. **Properties merged** with priority (last wins)
5. **Response formatted** according to requested format

### Caching Strategy

- **Git repositories**: Fetched on first request and cached for `fetchCacheTtl` seconds
- **Local checkouts**: Maintained in `{baseDir}/config-repo-{index}/`
- **Cache invalidation**: Explicit via DELETE `/cache` or TTL expiration
- **Failure handling**: Falls back to cached version if `failOnFetch` is false

## Troubleshooting

### Common Issues

#### Authentication Failures

**Problem**: `401 Unauthorized` responses

**Solution**: Verify bearer token and scopes:
```bash
# Decode JWT token
echo "$TOKEN" | cut -d. -f2 | base64 -d | jq .

# Check for required scope
echo "$TOKEN" | cut -d. -f2 | base64 -d | jq .scope
```

#### Git Clone/Fetch Failures

**Problem**: Unable to clone or fetch from Git repository

**Solutions**:
- Verify credentials are correct
- Check network connectivity to Git server
- For Azure: Verify tenant ID, client ID, and secret
- Enable debug logging: `LOG_LEVEL=DEBUG`
- Check dashboard for repository status: `GET /dashboard`

#### Configuration Not Found

**Problem**: `404 Not Found` for configuration requests

**Solutions**:
- Verify file naming matches pattern: `{app}-{profile}.yml` or `application.yml`
- Check search paths include the directory containing config files
- Verify label/branch exists in repository
- Check server logs for file search details

#### CredHub Connection Issues

**Problem**: Unable to connect to CredHub

**Solutions**:
- Verify CredHub service binding in Cloud Foundry
- Check mTLS certificates are valid
- For OAuth2: Verify client credentials
- Test CredHub connectivity: `curl $CREDHUB_URL/health`

## Performance Tuning

### Cache Settings

```json
{
  "type": "git",
  "uri": "...",
  "fetchCacheTtl": 300,
  "failOnFetch": false
}
```

- **Higher TTL**: Fewer fetches, stale data risk
- **Lower TTL**: More up-to-date, higher load
- **failOnFetch false**: More resilient, uses cache on failure

### Shallow vs Deep Clone

```json
{
  "type": "git",
  "uri": "...",
  "deepClone": false
}
```

- **Shallow clone** (default): Faster, uses less disk
- **Deep clone**: Required for git operations on checked-out repo

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes with tests
4. Run `go vet ./... && go test ./...`
5. Submit a pull request

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

## Support

For issues and questions:
- **Issues**: [GitHub Issues](https://github.com/rubionic/config-hub/issues)
- **Discussions**: [GitHub Discussions](https://github.com/rubionic/config-hub/discussions)

---

**Note**: This project is designed for Cloud Foundry environments but can be adapted for other platforms with appropriate authentication/authorization modifications.
