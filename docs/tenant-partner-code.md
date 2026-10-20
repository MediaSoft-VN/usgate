# Tenant partner code

The admin `POST /api/tenants` endpoint accepts an optional `partner_code`.

- When omitted or blank, the server creates a lowercase ASCII slug from the partner name.
- Supplied codes are trimmed, lowercased, and must contain only `a-z`, `0-9`, and `-`, with a maximum length of 64 characters.
- Codes are unique across tenants. A duplicate returns `partner code already exists`.
- Tenant list, detail, context selection, and upgrade responses include `partner_code`.

The tenant migration backfills codes for existing tenants, adding a numeric suffix when names produce duplicates.
