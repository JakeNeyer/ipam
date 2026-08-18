# Helm charts

This directory contains [Helm](https://helm.sh) charts for deploying IPAM to Kubernetes.

## Chart

| Chart   | Description |
|---------|-------------|
| [ipam/](./ipam/) | IPAM – IP Address Management. Deploys the app with optional Ingress, HPA, and an optional **PostgreSQL** subchart (Bitnami). |

## Install from a GitHub Release

Release tags publish `ipam-<version>.tgz` and `ipam-<version>.tgz.sha256` as release assets. The chart uses the published container image `ghcr.io/jakeneyer/ipam`.

```bash
VERSION=0.1.0   # or the latest release version

curl -fsSLO "https://github.com/JakeNeyer/ipam/releases/download/v${VERSION}/ipam-${VERSION}.tgz"
curl -fsSLO "https://github.com/JakeNeyer/ipam/releases/download/v${VERSION}/ipam-${VERSION}.tgz.sha256"
sha256sum -c "ipam-${VERSION}.tgz.sha256"

helm install ipam "./ipam-${VERSION}.tgz"

# With the optional PostgreSQL subchart
helm install ipam "./ipam-${VERSION}.tgz" \
  --set postgresql.enabled=true \
  --set postgresql.auth.postgresPassword=your-secure-password
```

## Install from source

From the repo root. **Before any install**, fetch chart dependencies (required because the chart declares an optional PostgreSQL dependency):

```bash
cd helm/ipam && helm dependency update && cd ../..
```

```bash
# Uses ghcr.io/jakeneyer/ipam by default (Chart.AppVersion)
helm install ipam ./helm/ipam

# Or build a local image
docker build -t ipam:latest .
helm install ipam ./helm/ipam \
  --set image.repository=ipam \
  --set image.tag=latest

# Install IPAM with the optional PostgreSQL subchart
helm install ipam ./helm/ipam \
  --set postgresql.enabled=true \
  --set postgresql.auth.postgresPassword=your-secure-password
```

## Install with Kind and Ingress

Use [Kind](https://kind.sigs.k8s.io/) (Kubernetes in Docker) to run IPAM locally with an Ingress. You need **Docker**, **kubectl**, **Helm**, and **kind** installed.

**1. Create a Kind cluster** (with extra port mappings so the Ingress is reachable from the host):

```bash
kind create cluster --name ipam --config - <<EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
    extraPortMappings:
      - containerPort: 80
        hostPort: 80
        protocol: TCP
      - containerPort: 443
        hostPort: 443
        protocol: TCP
EOF
```

**2. Install the NGINX Ingress controller:**

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
kubectl wait -n ingress-nginx --for=condition=ready pod -l app.kubernetes.io/component=controller --timeout=90s
```

**3. Load a local IPAM image into Kind** (or pull from GHCR if the cluster can reach it):

```bash
# From the repo root — local build
docker build -t ipam:latest .
kind load docker-image ipam:latest --name ipam
```

Alternatively, skip the local build and use `ghcr.io/jakeneyer/ipam` in step 5 (omit the `image.repository` / `image.tag` / `image.pullPolicy` overrides so the chart defaults apply).

**4. Fetch chart dependencies** (required once; the chart declares an optional PostgreSQL dependency):

```bash
cd helm/ipam && helm dependency update && cd ../..
```

**5. Install IPAM with Ingress enabled:**

```bash
# From the repo root. Local image loaded into Kind above
helm install ipam ./helm/ipam \
  --set image.repository=ipam \
  --set image.tag=latest \
  --set image.pullPolicy=Never \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set 'ingress.hosts[0].host=ipam.local' \
  --set 'ingress.hosts[0].paths[0].path=/' \
  --set 'ingress.hosts[0].paths[0].pathType=Prefix'
```

**6. Resolve the host** (pick one):

- Add to `/etc/hosts`: `127.0.0.1 ipam.local`
- Or use a wildcard that points to localhost, e.g. `ipam.127.0.0.1.nip.io`, and set `--set 'ingress.hosts[0].host=ipam.127.0.0.1.nip.io'` in step 5.

Then open **http://ipam.local** (or http://ipam.127.0.0.1.nip.io) in your browser. Use **http** (not https); there is no TLS by default. API docs: **http://ipam.local/docs**.

**Troubleshooting "ipam.local not showing anything"**

1. **Test without DNS** – from your machine run:
   ```bash
   curl -v -H "Host: ipam.local" http://127.0.0.1/
   ```
   If this returns HTML but the browser does not, the issue is DNS or `/etc/hosts` (e.g. wrong file, need to flush DNS cache).

2. **Confirm Ingress is bound to port 80** – if port 80 is already in use on your machine, Kind may not map it. Check:
   ```bash
   kubectl get nodes -o wide
   docker ps  # find the Kind node container and check port mapping 0.0.0.0:80->80/tcp
   ```
   If 80 is not mapped, edit the Kind cluster config to use a different host port (e.g. `hostPort: 8080`) and use `http://ipam.local:8080` (and add that to `/etc/hosts` if needed).

3. **Check IPAM and Ingress controller**:
   ```bash
   kubectl get pods -A | grep -E 'ipam|ingress'
   kubectl get ingress
   kubectl get svc
   ```
   Ensure the `ipam-ipam-*` pod is Running, the Ingress exists with host `ipam.local`, and the ingress-nginx controller pod in `ingress-nginx` is Running.

4. **Check Ingress controller logs** (if requests never reach IPAM):
   ```bash
   kubectl logs -n ingress-nginx -l app.kubernetes.io/component=controller --tail=50
   ```

**Optional: use PostgreSQL in Kind**

```bash
cd helm/ipam && helm dependency update && cd ../..
helm install ipam ./helm/ipam \
  --set image.repository=ipam \
  --set image.tag=latest \
  --set image.pullPolicy=Never \
  --set postgresql.enabled=true \
  --set postgresql.auth.postgresPassword=ipam-demo \
  --set ingress.enabled=true \
  --set ingress.className=nginx \
  --set 'ingress.hosts[0].host=ipam.local' \
  --set 'ingress.hosts[0].paths[0].path=/' \
  --set 'ingress.hosts[0].paths[0].pathType=Prefix'
```

**Teardown**

```bash
kind delete cluster --name ipam
```

See [ipam/README.md](./ipam/README.md) for full configuration, existing-secret usage, and other options.
