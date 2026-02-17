<div align="center">
  <img src="web/public/images/logo.svg" alt="IPAM logo" width="120" />
</div>

# IPAM

[![Test](https://img.shields.io/github/actions/workflow/status/JakeNeyer/ipam/test.yml?branch=main&style=for-the-badge)](https://github.com/JakeNeyer/ipam/actions/workflows/test.yml)
[![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=for-the-badge)](https://opensource.org/licenses/MIT)

**IPAM** is an IP Address Management application. It provides a REST API, a web UI, and a Terraform provider so you can manage IP space from the dashboard or from infrastructure-as-code.

This project is under active development. APIs are subject to change.

## Quick start

1. Set `DATABASE_URL` and run the API (from repo root): `go run .`
2. Serve the web UI: `cd web && npm run dev`
3. Open the app (e.g. `http://localhost:5173`), complete setup (create initial admin), then log in.

When the UI runs on a different origin (e.g. Vite on 5173), set **`APP_ORIGIN`** to that URL (e.g. `http://localhost:5173`). The API will then return 401 Unauthorized with a short message for non-API requests (so visiting the API URL directly shows “use the app at …” instead of the login page), and signup links and OAuth redirects will use the app origin.

### Optional: OAuth (GitHub and future providers)

OAuth is generic: add providers via config (GitHub is built-in). When enabled, users can sign in or sign up with a provider; OAuth still requires either a signup invite link or a pre-created user (by an admin or global admin). The implementation uses [golang.org/x/oauth2](https://github.com/golang/oauth2).

**GitHub:** Set `ENABLE_GITHUB_OAUTH=true` (or `1`), `GITHUB_CLIENT_ID`, and `GITHUB_CLIENT_SECRET`. Use **Authorization callback URL**: `https://<your-host>/api/auth/oauth/github/callback`.

To add more providers later, extend the server config and register an endpoint and user-info fetcher in the OAuth provider registry. If no OAuth providers are configured, only email/password login is used.

## E2E tests (Playwright)

From the repo root, run the API with the built web UI, then run Playwright from `web/`:

1. Build the web UI: `cd web && npm run build`
2. From repo root: `STATIC_DIR=web/dist go run .` (leave this running)
3. In another terminal: `cd web && npx playwright install chromium && npm run e2e`

Tests cover auth (login, logout, setup), security (API 401 without session, protected routes), and basic flows (dashboard, nav). For login and post-login tests, set `E2E_LOGIN_EMAIL` and `E2E_LOGIN_PASSWORD`; otherwise those tests are skipped. Base URL defaults to `http://localhost:8011` (override with `BASE_URL`).


## License

Licensed under the MIT License. See [LICENSE](LICENSE) for details.
