# IPAM Helm Chart

Deploy [IPAM](https://github.com/JakeNeyer/ipam) (IP Address Management) into Kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3+
- (Production) PostgreSQL – the app can run with an in-memory store when `DATABASE_URL` is not set (data is lost on restart).

## Install

Build and push the app image, then install the chart.

When using the optional PostgreSQL dependency, fetch dependencies first:

```bash
cd helm/ipam && helm dependency update && cd ../..
```

```bash
# From the repo root, build the image (example: local registry)
docker build -t ipam:latest .

# Install with default values (in-memory store, single replica)
helm install ipam ./helm/ipam

# Install with the optional PostgreSQL subchart (Bitnami PostgreSQL)
helm dependency update helm/ipam   # once, to fetch postgresql chart
helm install ipam ./helm/ipam \
  --set postgresql.enabled=true \
  --set postgresql.auth.postgresPassword=your-secure-password

# Install with existing PostgreSQL (create a secret with key database-url first)
kubectl create secret generic ipam-secrets --from-literal=database-url='postgresql://user:pass@host:5432/ipam?sslmode=disable'
helm install ipam ./helm/ipam --set existingSecret=ipam-secrets

# Install with ingress
helm install ipam ./helm/ipam \
  --set image.repository=your-registry/ipam \
  --set image.tag=1.0.0 \
  --set ingress.enabled=true \
  --set ingress.hosts[0].host=ipam.example.com \
  --set config.appOrigin=https://ipam.example.com
```

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
| `image.repository` | Image repository | `ipam` |
| `image.tag` | Image tag | `latest` |
| `service.port` | Service and container port | `8080` |
| `existingSecret` | Secret name for `database-url`, `initial-admin-password`, `oauth-<id>-client-secret` per provider | `""` |
| `oauth.providers` | Map of OAuth provider configs (see [OAuth providers](#oauth-providers)) | `{}` |
| `database.url` | PostgreSQL DSN (stored in a generated Secret; prefer `existingSecret` for production) | (none) |
| `postgresql.enabled` | Deploy Bitnami PostgreSQL as a subchart and set `DATABASE_URL` for IPAM | `false` |
| `postgresql.auth.postgresPassword` | PostgreSQL `postgres` user password (required when `postgresql.enabled`) | `""` |
| `postgresql.auth.database` | Database name to create | `ipam` |
| `config.appOrigin` | Public app URL (OAuth, signup links) | `""` |
| `config.initialAdminEmail` | First admin email (when no users exist) | `""` |
| `ingress.enabled` | Create an Ingress | `false` |
| `autoscaling.enabled` | Enable HPA | `false` |

When `postgresql.enabled` is true, run `helm dependency update helm/ipam` (or `helm dependency build`) before install. All Bitnami PostgreSQL [values](https://github.com/bitnami/charts/tree/main/bitnami/postgresql#parameters) can be set under `postgresql.*`.

## Uninstall

```bash
helm uninstall ipam
```
