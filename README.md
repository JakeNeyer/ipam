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

## Deploy to Fly.io

1. Install [flyctl](https://fly.io/docs/hands-on/install-flyctl/) and log in: `fly auth login`.
2. Create the app (from repo root): `fly launch --no-deploy` (or copy `fly.toml` and run `fly apps create ipam`).
3. Create managed Postgres (smallest):  
   `fly postgres create --name ipam-db --vm-size shared-cpu-1x --volume-size 1`
4. Attach Postgres so the app gets `DATABASE_URL`:  
   `fly postgres attach ipam-db`
5. (Optional) Set initial admin so you skip the setup UI:  
   `fly secrets set INITIAL_ADMIN_EMAIL=you@example.com INITIAL_ADMIN_PASSWORD=your-secure-password`  
   Only used when the database has no users; ignored otherwise.
6. Deploy:  
   `fly deploy`

The app listens on `PORT` (8080) and serves the API plus the built web UI. Open the app URL and log in (or complete setup if you didn’t set initial admin).

### Update an existing Fly.io deployment

Use these steps whenever you want to roll out the newest version.

1. Pull latest code and verify your branch is up to date:
   - `git checkout main`
   - `git pull`
2. From the repo root, deploy the latest commit:
   - `fly deploy`
3. Watch deployment status:
   - `fly status`
   - `fly logs`
4. Verify the running release:
   - Open your app URL (`fly open`) and confirm UI/API behavior.
   - Optionally verify the image/release in Fly:
     - `fly releases`
5. If the deployment fails, roll back to the previous stable release:
   - `fly releases`
   - `fly deploy --image <previous-image-ref>`

## License

Licensed under the MIT License. See [LICENSE](LICENSE) for details.
