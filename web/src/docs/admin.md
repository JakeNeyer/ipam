# Admin

The Admin page is available only to users with the admin role. It is where you manage API tokens used to access the IPAM API from scripts, CI/CD, or other tools. Open it from the sidebar when signed in as an admin.

<p class="docs-screenshot">
<img src="/images/admin-light.png" alt="Admin page (light mode)" class="screenshot-light" />
<img src="/images/admin-dark.png" alt="Admin page (dark mode)" class="screenshot-dark" />
</p>

## API tokens

API tokens let external tools and scripts authenticate to the IPAM API. Create a token with a name (e.g. "CI pipeline" or "CLI") and an optional expiration. When you create a token, the full token value is shown once—copy it immediately and store it securely; it cannot be viewed again. Use the token in the `Authorization: Bearer <token>` header when calling the API. You can list existing tokens (names and expiry) and delete tokens you no longer need. Only admins can create or delete tokens.

## Who can access Admin

Only users with the admin role can open the Admin page. If you do not have admin access, the Admin item may not appear in the sidebar, or you may be redirected if you try to open it. Use an account that was granted admin during setup.
