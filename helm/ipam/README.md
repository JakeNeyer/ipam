# IPAM Helm Chart

Deploy [IPAM](https://github.com/JakeNeyer/ipam) (IP Address Management) into Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3+
- (Production) PostgreSQL – the app can run with an in-memory store when `DATABASE_URL` is not set (data is lost on restart).

## Install from a GitHub Release

Each [release](https://github.com/JakeNeyer/ipam/releases) attaches `ipam-<version>.tgz` and `ipam-<version>.tgz.sha256`. The chart pulls `ghcr.io/jakeneyer/ipam` by default (tag defaults to the chart `appVersion`).

```bash
VERSION=0.1.0   # or the latest release version

curl -fsSLO "https://github.com/JakeNeyer/ipam/releases/download/v${VERSION}/ipam-${VERSION}.tgz"
curl -fsSLO "https://github.com/JakeNeyer/ipam/releases/download/v${VERSION}/ipam-${VERSION}.tgz.sha256"
sha256sum -c "ipam-${VERSION}.tgz.sha256"

# Install with default values (in-memory store, single replica)
helm install ipam "./ipam-${VERSION}.tgz"

# Install with the optional PostgreSQL subchart (Bitnami PostgreSQL)
helm install ipam "./ipam-${VERSION}.tgz" \
  --set postgresql.enabled=true \
  --set postgresql.auth.postgresPassword=your-secure-password

# Install with existing PostgreSQL (create a secret with key database-url first)
kubectl create secret generic ipam-secrets --from-literal=database-url='postgresql://user:pass@host:5432/ipam?sslmode=disable'
helm install ipam "./ipam-${VERSION}.tgz" --set existingSecret=ipam-secrets

# Install with ingress
helm install ipam "./ipam-${VERSION}.tgz" \
  --set image.tag=0.1.0 \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=ipam.example.com \
  --set config.appOrigin=https://ipam.example.com
```

## Install from source

Clone the repo, fetch chart dependencies, then install from the chart path:

```bash
git clone https://github.com/JakeNeyer/ipam.git
cd ipam
cd helm/ipam && helm dependency update && cd ../..

# Uses ghcr.io/jakeneyer/ipam by default (Chart.AppVersion)
helm install ipam ./helm/ipam

# Or build and use a local image
docker build -t ipam:latest .
helm install ipam ./helm/ipam \
  --set image.repository=ipam \
  --set image.tag=latest \
  --set image.pullPolicy=IfNotPresent
```

When using the optional PostgreSQL dependency from source, fetch dependencies first (`helm dependency update helm/ipam`).

### OAuth providers

Configure providers under `oauth.providers` in `values.yaml` (or `--set` / `-f`). Each key is the provider id (e.g. `keycloak`, `github`). A provider is enabled when `clientId`, `authUrl`, `tokenUrl`, and `userInfoUrl` are set.

Store client secrets in `existingSecret` (recommended). Default secret key per provider: `oauth-<id>-client-secret` (override with `existingSecretKey` on the provider).

```bash
kubectl create secret generic ipam-secrets \
  --from-literal=database-url='postgresql://...' \
  --from-literal=oauth-keycloak-client-secret='your-keycloak-secret' \
  --from-literal=oauth-github-client-secret='your-github-secret'

helm install ipam ./helm/ipam -f my-values.yaml --set existingSecret=ipam-secrets
```

Example `values.yaml` fragment:

```yaml
config:
  appOrigin: https://ipam.example.com
  initialAdminEmail: admin@example.com
  initialOrganizationName: acme
  initialAdminAPITokenTTL: 90d

existingSecret: ipam-secrets

oauth:
  providers:
    keycloak:
      clientId: ipam
      authUrl: https://idp.example.com/realms/ipam/protocol/openid-connect/auth
      tokenUrl: https://idp.example.com/realms/ipam/protocol/openid-connect/token
      userInfoUrl: https://idp.example.com/realms/ipam/protocol/openid-connect/userinfo
      scopes: [openid, email, profile]
      displayName: Sign in with Keycloak
    github:
      clientId: "123456"
      authUrl: https://github.com/login/oauth/authorize
      tokenUrl: https://github.com/login/oauth/access_token
      userInfoUrl: https://api.github.com/user
      emailsUrl: https://api.github.com/user/emails
      scopes: [user:email]
      userIdClaim: id
      displayName: Sign in with GitHub
```

See the root [README.md](../../README.md#optional-oauth) for all `OAUTH_<ID>_*` variables (`emailsUrl`, `emailVerifiedClaim`, `allowEmailMatch`, etc.).

Validate rendered manifests:

```bash
helm template ipam ./helm/ipam -f helm/ipam/ci/oauth-values.yaml
```

## Configuration

| Key | Description | Default |
|-----|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Image repository | `ghcr.io/jakeneyer/ipam` |
| `image.tag` | Image tag (empty → `Chart.AppVersion`) | `""` |
| `service.port` | Service and container port | `8080` |
| `existingSecret` | Secret name for `database-url`, `initial-admin-password`, `initial-admin-token`, `oauth-<id>-client-secret` per provider | `""` |
| `oauth.providers` | Map of OAuth provider configs (see [OAuth providers](#oauth-providers)) | `{}` |
| `database.url` | PostgreSQL DSN (stored in a generated Secret; prefer `existingSecret` for production) | (none) |
| `postgresql.enabled` | Deploy Bitnami PostgreSQL as a subchart and set `DATABASE_URL` for IPAM | `false` |
| `postgresql.auth.postgresPassword` | PostgreSQL `postgres` user password (required when `postgresql.enabled`) | `""` |
| `postgresql.auth.database` | Database name to create | `ipam` |
| `config.appOrigin` | Public app URL (OAuth, signup links) | `""` |
| `config.initialAdminEmail` | First admin email (when no users exist) | `""` |
| `config.initialOrganizationName` | First organization name (created with initial admin) | `""` |
| `config.initialAdminAPITokenTTL` | TTL for bootstrap API token (`initial-admin-token` in `existingSecret`) | `""` |
| `ingress.enabled` | Create an Ingress | `false` |
| `autoscaling.enabled` | Enable HPA | `false` |

When `postgresql.enabled` is true, run `helm dependency update helm/ipam` (or `helm dependency build`) before install from source. All Bitnami PostgreSQL [values](https://github.com/bitnami/charts/tree/main/bitnami/postgresql#parameters) can be set under `postgresql.*`. Packaged release charts already include dependencies.

## Uninstall

```bash
helm uninstall ipam
```
