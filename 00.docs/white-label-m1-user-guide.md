# White-label M1 — User and Operations Guide

## 1. What M1 provides

M1 lets an organization run an isolated White-label tenant with:

- Its own tenant identity.
- Members and tenant-specific roles.
- A verified custom domain.
- Server-side isolation from other tenants.

Payment channels, wallets and custom branding are planned for later milestones.

## 2. Roles

- **Owner:** full tenant control, including ownership transfer.
- **Admin:** manages members and domains.
- **Member:** uses permitted tenant functions.
- **Viewer:** read-only access.

Platform administrators remain separate from tenant roles.

## 3. Promote a user into a tenant partner

1. A platform administrator opens tenant management.
2. Select **Create tenant** and choose an existing enabled system user as the
   partner owner.
3. Enter a unique slug and display name.
4. Save.
5. The selected user becomes the tenant owner.

Regular users cannot create tenants themselves.

Tenant slugs are lowercase and cannot be reused by another tenant.

## 4. Add members

1. Open the tenant's **Members** page.
2. Select **Invite member**.
3. Enter the user's email and role.
4. Send the invitation.
5. Change or remove membership when required.

The final active owner cannot be removed. Ownership transfer is a protected
operation and may require re-authentication.

## 5. Add a custom domain

1. Open **Domains** for the tenant.
2. Add the full hostname, such as `console.example.com`.
3. Create the domain challenge.
4. Add the displayed DNS TXT record at the exact required hostname.
5. Select **Verify**.
6. Wait until the status changes from `pending` to `verified`.

Only verified domains can resolve to tenant data. Remove or disable a domain
immediately if ownership changes.

### Troubleshooting domain verification

- Confirm the TXT name and value exactly.
- Remove accidental quotes or extra whitespace.
- Wait for DNS propagation and retry.
- Confirm the hostname is not already assigned to another tenant.
- Do not share the verification token.

## 6. Tenant selection and access

The server determines the tenant from the verified hostname and the user's
membership. A client cannot switch tenant by changing a query parameter or
request body. If a user has memberships in multiple tenants, the dashboard may
offer an authenticated tenant selector where the hostname does not already
fix the tenant.

## 7. Security operations

- Use HTTPS for every White-label domain.
- Configure trusted reverse proxies before enabling forwarded-host handling.
- Remove users promptly when access is no longer required.
- Review audit events for membership, ownership and domain changes.
- Never place domain verification or invitation tokens in logs, tickets or
  screenshots.

## 8. Planned next milestones

The following partner capabilities are not implemented in this M1 flow and
remain M2 work:

1. Tenant verified-domain management UI and end-to-end domain operations.
2. Tenant member management UI and invitation lifecycle.
3. Tenant cash-flow views and ledger reporting.
4. Tenant wallet crediting and user wallet allocation.

After M2 is accepted:

1. Tenant payment channels and webhooks.
2. White-label branding and user-facing payment UI.
