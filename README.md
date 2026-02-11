# IPAM

[![Test](https://img.shields.io/github/actions/workflow/status/JakeNeyer/ipam/test.yml?branch=main&style=for-the-badge)](https://github.com/JakeNeyer/ipam/actions/workflows/test.yml)
[![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)

**IPAM** is an IP Address Management application for tracking environments, network blocks (CIDR ranges), and allocations. It provides a REST API, a web UI, and a Terraform provider so you can manage IP space from the dashboard or from infrastructure-as-code.

## Quick start

1. Set `DATABASE_URL` and run the API (from repo root): `go run .`
2. Serve the web UI: `cd web && npm run dev`
3. Open the app, complete setup (create initial admin), then log in.

## E2E tests (Playwright)

From the repo root, run the API with the built web UI, then run Playwright from `web/`:

1. Build the web UI: `cd web && npm run build`
2. From repo root: `STATIC_DIR=web/dist go run .` (leave this running)
3. In another terminal: `cd web && npx playwright install chromium && npm run e2e`

Tests cover auth (login, logout, setup), security (API 401 without session, protected routes), and basic flows (dashboard, nav). For login and post-login tests, set `E2E_LOGIN_EMAIL` and `E2E_LOGIN_PASSWORD`; otherwise those tests are skipped. Base URL defaults to `http://localhost:8011` (override with `BASE_URL`).


## License

Licensed under the MIT License. See [LICENSE](LICENSE) for details.
