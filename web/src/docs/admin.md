# Admin

The **Admin** page is available only to users with the **admin** role. From the sidebar, open **Admin** to manage users, organizations, signup links, and API tokens.

## Organizations (global admin only)

If you are the **global admin** (no organization assigned), you will see an **Organizations** section at the top of the Admin page.

- **Create** — Click **Create organization**, enter a name, and save. Organizations are tenants; users and environments belong to an organization.
- **Edit** — Click **Edit** next to an organization, change the name, and click **Save**.
- **Delete** — Click **Delete**. A confirmation modal explains that deleting an organization **permanently removes** all of its resources:
  - Environments and all network blocks and allocations in them
  - Reserved blocks
  - Users (and their API tokens and sessions)
  - Signup links  
  Confirm only when you are sure; this cannot be undone.

Org admins do not see the Organizations section; they manage users and signup links within their own organization.

## Users

The **Users** table lists all users you can manage (your organization’s users, or all users if you are global admin).

- **Add user** — Click **Add user**. Enter email, password, and role (`user` or `admin`). If you are global admin, you can also choose which organization the user belongs to; leave **None** to create another global admin.
- **Role** — Use the role dropdown to switch a user between `user` and `admin`. Admins can access the Admin page, manage reserved blocks, and (if global admin) manage organizations.
- **Organization** — Global admins can reassign a user to a different organization via the organization dropdown. Org admins do not see this column.
- **Delete** — Removes the user. Their API tokens and sessions are removed as well.

## Signup links

Create time-bound invite links so new users can sign up without being added manually.

- **Create signup link** — Click **Create signup link**, set an expiration (e.g. 7 days), choose the organization and role for the new user (global admin can pick any org), and create. Copy the link and share it; it can only be used once.
- **Revoke** — Use **Revoke** next to a link to invalidate it before it is used or expires.

Links are listed with status (Pending, Used, or Expired).

## API tokens

API tokens allow scripted or programmatic access to the IPAM API (e.g. Terraform, CI/CD).

- **Manage API tokens** — Click **API tokens** to open the tokens modal. Create a token with a name; the secret is shown once. Use it as `Authorization: Bearer <token>` for all `/api` requests.
- **Revoke** — Delete a token from the list when it is no longer needed. Existing requests using that token will fail after revocation.

See [Getting started](#docs/getting-started) for example API usage with a token.
