<div align="center">
  <img src="web/public/images/logo-light.svg" alt="IPAM logo" width="240" />
</div>


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

### Optional: OAuth

OAuth sign-in is optional. Users still need a signup invite or an account created by an admin. Set **`APP_ORIGIN`** when the UI is on a different host than the API.

Set **`OAUTH_PROVIDERS`** to a comma-separated list of provider ids (e.g. `sso`). Register each client with redirect URI `https://<host>/api/auth/oauth/<provider-id>/callback`.

For each id `<ID>` (uppercased in env names; hyphens become underscores):

| Variable | Required | Default | Notes |
|----------|----------|---------|-------|
| `OAUTH_<ID>_CLIENT_ID` | yes | — | |
| `OAUTH_<ID>_CLIENT_SECRET` | yes | — | |
| `OAUTH_<ID>_AUTH_URL` | yes | — | |
| `OAUTH_<ID>_TOKEN_URL` | yes | — | |
| `OAUTH_<ID>_USERINFO_URL` | yes | — | |
| `OAUTH_<ID>_SCOPES` | no | `openid,email,profile` | |
| `OAUTH_<ID>_DISPLAY_NAME` | no | derived from id | |
| `OAUTH_<ID>_USER_ID_CLAIM` | no | `sub` | |
| `OAUTH_<ID>_EMAIL_CLAIM` | no | `email` | |
| `OAUTH_<ID>_EMAILS_URL` | no | — | Second HTTP GET when email is not on userinfo (e.g. GitHub `/user/emails`). |
| `OAUTH_<ID>_EMAILS_PRIMARY_CLAIM` | no | `primary` | On the emails array: prefer the entry where this claim is true. |
| `OAUTH_<ID>_EMAIL_VERIFIED_CLAIM` | no | `email_verified` | On **userinfo**: if `EMAIL_CLAIM` is present, this claim must exist and be true or sign-in fails. Standard OIDC userinfo (Keycloak, etc.) exposes `email_verified`. Unset the env var to use an empty claim name and skip this check. |
| `OAUTH_<ID>_EMAILS_VERIFIED_CLAIM` | no | `verified` (only when `EMAILS_URL` is set) | On the **emails array**: ignore entries where this claim is missing or false. GitHub marks confirmed addresses with `"verified": true`. If every entry is unverified, sign-in fails. Unset to skip verification on the list. |
| `OAUTH_<ID>_ALLOW_EMAIL_MATCH` | no | `false` | Set `true` to sign in to an existing account by email alone (see below). |

**Email verification:** IPAM never trusts an email address from OAuth unless the provider marks it verified. For a single userinfo JSON object, that means `email_verified: true` (claim name configurable). For a separate emails list, only verified entries are considered; the primary verified address wins, otherwise the first verified address. This blocks IdP or GitHub responses that include an unverified email from taking over an existing account.

Set `OAUTH_<ID>_EMAILS_URL` when the provider returns email on a second JSON-array endpoint (for example GitHub `https://api.github.com/user/emails` after reading `https://api.github.com/user`).

```bash
export OAUTH_PROVIDERS=sso
export OAUTH_SSO_CLIENT_ID=ipam
export OAUTH_SSO_CLIENT_SECRET=secret
export OAUTH_SSO_AUTH_URL=https://idp.example.com/realms/app/protocol/openid-connect/auth
export OAUTH_SSO_TOKEN_URL=https://idp.example.com/realms/app/protocol/openid-connect/token
export OAUTH_SSO_USERINFO_URL=https://idp.example.com/realms/app/protocol/openid-connect/userinfo
```

GitHub example:

```bash
export OAUTH_PROVIDERS=github
export OAUTH_GITHUB_CLIENT_ID=...
export OAUTH_GITHUB_CLIENT_SECRET=...
export OAUTH_GITHUB_AUTH_URL=https://github.com/login/oauth/authorize
export OAUTH_GITHUB_TOKEN_URL=https://github.com/login/oauth/access_token
export OAUTH_GITHUB_USERINFO_URL=https://api.github.com/user
export OAUTH_GITHUB_EMAILS_URL=https://api.github.com/user/emails
export OAUTH_GITHUB_SCOPES=user:email
export OAUTH_GITHUB_USER_ID_CLAIM=id
export OAUTH_GITHUB_DISPLAY_NAME="Sign in with GitHub"
# EMAILS_VERIFIED_CLAIM defaults to "verified"; unverified GitHub emails are ignored.
```

Keycloak example (add to your SSO vars):

```bash
# Userinfo includes "email_verified": true; enforced by default (EMAIL_VERIFIED_CLAIM=email_verified).
export OAUTH_KEYCLOAK_EMAIL_VERIFIED_CLAIM=email_verified
```

When `OAUTH_PROVIDERS` is unset, login is email and password only.

## E2E tests (Playwright)

From the repo root, run the API with the built web UI, then run Playwright from `web/`:

1. Build the web UI: `cd web && npm run build`
2. From repo root: `STATIC_DIR=web/dist go run .` (leave this running)
3. In another terminal: `cd web && npx playwright install chromium && npm run e2e`

Tests cover auth (login, logout, setup), security (API 401 without session, protected routes), and basic flows (dashboard, nav). For login and post-login tests, set `E2E_LOGIN_EMAIL` and `E2E_LOGIN_PASSWORD`; otherwise those tests are skipped. Base URL defaults to `http://localhost:8011` (override with `BASE_URL`).


## License

Licensed under the MIT License. See [LICENSE](LICENSE) for details.
