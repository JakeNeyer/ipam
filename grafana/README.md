# IPAM Grafana dashboard

Import **IPAM IP Space** (`ipam-ip-space.json`) into Grafana after Prometheus is scraping `GET /metrics`.

The dashboard asks for a Prometheus data source on first load (top-left **Data source**). Filters: organization, environment, address family.

Panels: blocks at ≥80% / ≥95% fill, block fill and pool carve-out over time, a per-block table (fill, used, remaining), and a collapsed inventory row.

## Grafana UI (any existing Grafana)

1. Scrape IPAM (`prometheus.yml` job against `/metrics`, or Helm `metrics.serviceMonitor.enabled=true`).
2. **Dashboards → New → Import**.
3. Upload `grafana/ipam-ip-space.json` (or paste the JSON).
4. Open the dashboard and choose your Prometheus data source if it is not already selected.

Direct link after import: `/d/ipam-ip-space`.

## Kubernetes (Grafana sidecar)

Label a ConfigMap so the Grafana dashboard sidecar (kube-prometheus-stack, grafana helm chart) loads it:

```bash
kubectl create configmap ipam-grafana-dashboard \
  --from-file=ipam-ip-space.json=grafana/ipam-ip-space.json \
  -n monitoring \
  --dry-run=client -o yaml \
| kubectl label --local -f - grafana_dashboard=1 -o yaml \
| kubectl apply -f -
```

Or with the IPAM Helm chart:

```bash
helm upgrade --install ipam ./helm/ipam \
  --set metrics.serviceMonitor.enabled=true \
  --set metrics.grafanaDashboard.enabled=true
```

The chart ConfigMap uses label `grafana_dashboard=1` by default (override with `metrics.grafanaDashboard.labels`). Put it in the same namespace as Grafana, or set `metrics.grafanaDashboard.namespace`.
