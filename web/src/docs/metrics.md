# Metrics (Prometheus / Grafana)

IPAM exposes current IP-space usage as Prometheus gauges at **`GET /metrics`**. Each scrape walks the live store (pools, blocks, allocations, reserved ranges), so Grafana and Alertmanager can watch capacity without polling the REST API. Results are cached for 5 seconds, so several scrapers (an HA Prometheus pair, an agent plus a sidecar) share one walk.

The endpoint does not use the session cookie, and it lists every organization, environment, pool, block, and CIDR. Keep it off public Ingress and scrape the Kubernetes Service (or localhost) instead.

| Variable | Default | Effect |
|----------|---------|--------|
| `METRICS_ENABLED` | `true` | Set to `false` to not serve `/metrics` at all. |
| `METRICS_TOKEN` | unset | When set, requests must send `Authorization: Bearer <token>`; anything else gets `401`. |

Without a token the server logs a warning at startup.

## Install the Grafana dashboard

The dashboard JSON is [grafana/ipam-ip-space.json](https://github.com/JakeNeyer/ipam/blob/main/grafana/ipam-ip-space.json). It uses a **Data source** dropdown so you can point it at any Prometheus that scrapes IPAM.

**Import into Grafana**

1. Scrape IPAM (`/metrics`) from Prometheus.
2. In Grafana: **Dashboards → New → Import**.
3. Upload `grafana/ipam-ip-space.json`.
4. Open **IPAM IP Space** and select your Prometheus data source if needed.

URL after import: `/d/ipam-ip-space`. Full steps: [grafana/README.md](https://github.com/JakeNeyer/ipam/blob/main/grafana/README.md).

**Helm (Grafana sidecar)** — creates a ConfigMap labeled `grafana_dashboard=1` for kube-prometheus-stack / Grafana sidecar:

```bash
helm upgrade --install ipam ./helm/ipam \
  --set metrics.serviceMonitor.enabled=true \
  --set metrics.grafanaDashboard.enabled=true
```

## Scrape

```bash
curl -s http://localhost:8011/metrics | head
```

With a token:

```bash
curl -s -H "Authorization: Bearer $METRICS_TOKEN" http://localhost:8011/metrics
```

Helm `metrics.enabled` (default `true`) sets `METRICS_ENABLED` and adds `prometheus.io/scrape` pod annotations. For prometheus-operator:

```bash
helm install ipam ./helm/ipam --set metrics.serviceMonitor.enabled=true
```

Set the scrape token with `metrics.token`, or store it in `existingSecret` under key `metrics-token` (override the key with `metrics.existingSecretKey`); `metrics.token` wins when both are set. Point the ServiceMonitor at that secret with `metrics.serviceMonitor.bearerTokenSecret`.

With `ingress.enabled=true` the chart refuses to render unless a token source is configured or `metrics.enabled=false`, because the default Ingress forwards every path. Set `metrics.allowUnauthenticatedIngress=true` only if `/metrics` is protected some other way.

## What is measured

| Metric | Meaning |
|--------|---------|
| `ipam_block_ips` | Addresses in the block CIDR |
| `ipam_block_used_ips` | Addresses covered by allocations in that block |
| `ipam_block_available_ips` | Remaining addresses in the block |
| `ipam_block_utilization_ratio` | Allocated fraction of the block (**0–1**) |
| `ipam_block_allocations` | Number of allocations in the block |
| `ipam_pool_ips` | Addresses in the pool CIDR |
| `ipam_pool_used_ips` | Addresses carved into **direct** child pools and blocks |
| `ipam_pool_available_ips` | Uncarved pool addresses |
| `ipam_pool_utilization_ratio` | Carved fraction of the pool (**0–1**) |
| `ipam_reserved_ips` | Addresses in a reserved (blacklisted) CIDR |
| `ipam_organizations` | Organization count |
| `ipam_environments` | Environments per organization |
| `ipam_pools` / `ipam_blocks` / `ipam_allocations` | Inventory counts by organization, environment, and address family |
| `ipam_metrics_collect_success` | `1` if the last scrape walk succeeded, `0` if a store query failed (the scrape itself still returns `200`) |
| `ipam_metrics_collect_duration_seconds` | Time to walk the store |

**Block utilization** is allocation fill (same idea as the Networks usage view). **Pool utilization** is how much of the pool CIDR is already assigned to child pools and blocks, not how full those blocks are.

**Labels.** Per-resource series (`ipam_block_*`, `ipam_pool_*`, `ipam_reserved_ips`) carry `id` (the resource UUID) plus `organization`, `environment`, `pool`, `block` or `name`, `cidr`, `family` (`ipv4` or `ipv6`), and `provider`. Names and CIDRs are not unique, so `id` is what keeps two same-named blocks apart; group by `block` or `cidr` in queries if you want them merged. Orphan blocks use an empty `environment` / `pool`. Inventory counts (`ipam_pools`, `ipam_blocks`, `ipam_allocations`) use only `organization`, `environment`, `family`.

Allocations reference their block by name. An allocation whose block cannot be resolved still counts toward `ipam_allocations`, under empty `organization` / `environment`, so `sum(ipam_allocations)` is always the true total.

IPv6 address counts are Prometheus `float64` values and can be approximate for very large prefixes; **`utilization_ratio` is the reliable signal**.

Go runtime and process metrics are included on the same endpoint.

## Alerts

Example Prometheus alerting rules (a `PrometheusRule` resource with prometheus-operator, or a file under `rule_files` in `prometheus.yml`):

```yaml
groups:
  - name: ipam
    rules:
      - alert: IPAMBlockHighUtilization
        expr: ipam_block_utilization_ratio > 0.8
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "IPAM block {{ $labels.block }} ({{ $labels.cidr }}) is {{ $value | humanizePercentage }} allocated"

      - alert: IPAMPoolExhausted
        expr: ipam_pool_utilization_ratio > 0.9
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "IPAM pool {{ $labels.pool }} in {{ $labels.environment }} is {{ $value | humanizePercentage }} carved"

      - alert: IPAMMetricsCollectFailing
        expr: ipam_metrics_collect_success == 0
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "IPAM metrics collection is failing on {{ $labels.instance }}"
```

Alertmanager handles routing and notification for these alerts; it does not evaluate the rules itself.
