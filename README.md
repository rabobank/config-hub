# Config Hub

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go)](https://go.dev/)

A lightweight, Spring Cloud Config Server-compatible configuration server written in Go, designed specifically for Cloud Foundry environments. Config Hub provides centralized configuration management with support for multiple backends including Git repositories and CredHub secret storage.

## Features

- **Spring Cloud Config Server Compatible**: Drop-in replacement supporting standard endpoints and response formats
- **Multiple Configuration Sources**: Git repositories (HTTP/HTTPS/SSH) and CredHub integration for secure secret management
- **Advanced Git Authentication**: Username/password, SSH private key, Azure Service Principal (SPN), and Azure Managed Identity with Workload Identity Federation (WIF)
- **Cloud Foundry Integration**: Native CF environment support with UAA/OpenID authentication and service instance-based access control
- **Flexible Configuration**: Multiple profiles, label/branch selection, search paths with wildcard support
- **Caching & Performance**: Configurable fetch cache TTL with cache invalidation endpoint
- **Git Credential Helper Mode**: Can act as a git credential helper for secure credential retrieval
- **Management Dashboard**: Web-based dashboard for monitoring configuration sources

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

The server starts on port 8080 (configurable via `PORT` environment variable).

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
git clone https://github.com/rubionic/config-hub.git
cd config-hub
./.github/scripts/build.sh
```

### Pre-built Binaries

Download the latest release from the [releases page](https://github.com/rubionic/config-hub/releases).

### Docker

```bash
docker run -p 8080:8080 \
  -e CLIENT_ID=your-client \
  -e CLIENT_SECRET=your-secret \
  -e OPENID_URL=https://login.cf.example.com \
  -e SERVICE_INSTANCE_ID=instance-id \
  -e CH_SOURCES='[...]' \
  config-hub:latest
```

## Configuration

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

Basic Git source with username/password authentication:

```json
{
  "type": "git",
  "uri": "https://github.com/your-org/config-repo.git",
  "defaultLabel": "main",
  "searchPaths": ["apps/{application}", "common"],
  "username": "git-user",
  "password": "ghp_token",
  "fetchCacheTtl": 120
}
```

**Configuration Options:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `type` | string | ✓ | Must be `"git"` |
| `uri` | string | ✓ | Git repository URL (HTTP/HTTPS/SSH) |
| `defaultLabel` | string | | Default branch/tag (default: `"master"`) |
| `searchPaths` | []string | | Paths to search for config files (supports `{application}`, `{profile}`, and `*` wildcard) |
| `username` | string | | Username for authentication |
| `password` | string | | Password or personal access token |
| `fetchCacheTtl` | int | | Cache TTL in seconds (min: 60, default: 60) |
| `skipSslValidation` | bool | | Skip SSL certificate validation |
| `failOnFetch` | bool | | Fail requests if fetch fails (vs. using cached) |

### CredHub Source Configuration

```json
{
  "type": "credhub",
  "prefix": "/config-hub/",
  "client": "credhub-client",
  "secret": "credhub-secret"
}
```

Secrets are organized as: `{prefix}{app}/{profile}/{label}/secrets`

If neither `client` nor `secret` is provided, mTLS authentication is used.

### Complete Multi-Source Example

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

## API Endpoints

### Configuration Endpoints (Spring Cloud Config Compatible)

```
GET /{application}/{profiles}
GET /{application}/{profiles}/{label}
GET /{application}-{profiles}.json
GET /{application}-{profiles}.yml
GET /{application}-{profiles}.properties
```

**Note**: OpenAPI/Swagger documentation can be added in a future PR for detailed API specifications.

### Management Endpoints

```
GET /health                    # Health check (anonymous)
GET /info                      # Server info (anonymous)
GET /dashboard                 # Management dashboard (admin/developer)
DELETE /cache                  # Clear cache (admin/developer)
```

### CredHub Secret Management

```
POST /secrets                  # Add secrets (admin/developer)
GET /secrets                   # List secrets (admin/developer)
DELETE /secrets                # Delete secrets (admin/developer)
```

### Git Credentials

```
POST /credentials              # Git credential helper (localhost only)
```

## Advanced Features

### Git Credential Helper Mode

Config Hub can act as a git credential helper for secure credential retrieval from CredHub.

**Setup:**
```bash
# Configure git globally
git config --global credential.helper '!config-hub credentials repo-path'

# Or for a specific repository
cd /path/to/repo
git config credential.helper '!config-hub credentials $(pwd)'
```

**Usage:**
```bash
# Git will automatically request credentials when needed
git clone https://github.com/your-org/private-repo.git
git pull
```

**How it works**: Git invokes `config-hub credentials <repo> get`, Config Hub fetches credentials from CredHub based on repository host, and returns them in git credential helper format.

### Advanced Git Authentication

**SSH Private Key:**
```json
{
  "type": "git",
  "uri": "git@github.com:your-org/config-repo.git",
  "privateKey": "-----BEGIN RSA PRIVATE KEY-----\n..."
}
```

**Azure Service Principal (SPN):**
```json
{
  "type": "git",
  "uri": "https://dev.azure.com/org/project/_git/repo",
  "azTenantId": "tenant-id",
  "azClient": "client-id",
  "azSecret": "client-secret"
}
```

**Azure Managed Identity with WIF:**
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

## Authentication & Authorization

### Authentication Methods

1. **Bearer Token**: Standard OAuth2 bearer token in `Authorization` header
2. **SSO**: Browser-based single sign-on for dashboard access

### Authorization Rules

| Endpoint | Requirements |
|----------|--------------|
| `/health`, `/info` | Anonymous access |
| `/credentials` | Localhost only |
| `/secrets/*`, `/cache`, `/dashboard` | `cloud_controller.admin` scope OR space developer role |
| Configuration endpoints | `config_hub_{service-instance-id}.read` scope |

Space developer permissions are verified by checking CF API: `GET {CF_URL}/v3/service_instances/{SERVICE_INSTANCE_ID}/permissions`

## Deployment

### Cloud Foundry

**manifest.yml example:**

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

**Note**: Variables in `((variable))` format require a credential manager (e.g., CredHub via `cf push --vars-file` or Credhub Interpolation).

**Deployment:**
```bash
./.github/scripts/build.sh
cf push
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

Build and run:
```bash
docker build -t config-hub:latest .
docker run -p 8080:8080 -e CLIENT_ID=... config-hub:latest
```

## Development & Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for detailed development setup, project structure, code guidelines, and contribution workflow.

**Quick commands:**
```bash
# Run tests
./.github/scripts/runTests.sh

# Build
./.github/scripts/build.sh
```

## Troubleshooting

### Authentication Failures (401 Unauthorized)

Verify bearer token has required scopes:
```bash
echo "$TOKEN" | cut -d. -f2 | base64 -d | jq .scope
```

### Git Clone/Fetch Failures

- Verify credentials are correct
- Check network connectivity to Git server
- For Azure: Verify tenant ID, client ID, and secret
- Enable debug logging: `LOG_LEVEL=DEBUG`
- Check dashboard: `GET /dashboard`

### Configuration Not Found (404)

- Verify file naming: `{app}-{profile}.yml` or `application.yml`
- Check search paths include the directory containing config files
- Verify label/branch exists in repository

### CredHub Connection Issues

- Verify CredHub service binding in Cloud Foundry
- Check mTLS certificates are valid
- For OAuth2: Verify client credentials

## License

This project is licensed under the Apache License 2.0. See the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/rubionic/config-hub/issues)
- **Discussions**: [GitHub Discussions](https://github.com/rubionic/config-hub/discussions)

---

**Note**: This project is designed for Cloud Foundry environments but can be adapted for other platforms with appropriate authentication/authorization modifications.
