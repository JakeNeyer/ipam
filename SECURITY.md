# Security

This document summarizes how the IPAM app and API address common risks (OWASP Top 10–oriented).

## API (Go)

- **Broken Access Control** — Admin routes (`/api/admin/*`) and token creation require `user.Role == admin`. Setup (`POST /api/setup`) only succeeds when no users exist. Token delete checks `user.ID` so users can only delete their own tokens.
- **Cryptographic Failures** — Passwords hashed with bcrypt. API tokens stored as SHA-256 hashes. Session cookie has `HttpOnly`, `SameSite=Lax`, and `Secure` when the request is TLS.
- **Injection** — All database access uses parameterized queries (`$1`, `$2`, etc.); no string concatenation into SQL.
- **Security Misconfiguration** — Response headers: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Referrer-Policy: strict-origin-when-cross-origin`, `Permissions-Policy` to disable geolocation/mic/camera. Request body limited to 1MB to reduce DoS from large payloads.
- **Identification and Authentication** — Email format and length validated; password minimum length 8, max 72 (bcrypt limit). Login/setup/admin user creation use the same validation.

## Web (Svelte)

- **Injection (XSS)** — User guide markdown is rendered only after sanitization with DOMPurify (allowlist of tags and attributes). No `innerHTML`/`eval` of unsanitized user input elsewhere.
- **Cross-Site Request Forgery** — API uses session cookies with `SameSite=Lax` and Bearer tokens; same-origin frontend sends `credentials: 'include'` for API calls.

## Recommendations

- Run the API behind HTTPS in production so the session cookie is sent with `Secure`.
- Keep dependencies updated (`go mod tidy`, `npm audit`).
- Use a reverse proxy or WAF for rate limiting (e.g. on `/api/auth/login`) if you need it.
