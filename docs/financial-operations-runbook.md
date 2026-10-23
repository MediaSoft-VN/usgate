# Financial Operations Runbook

This runbook covers the financial wallet, provider funding, provider credit,
cash-flow reporting, and ledger migration paths in New API.

## 1. Before deployment

- Deploy a tagged release or a reviewed commit from a clean working tree.
- Confirm that `SQL_DSN` points to the intended primary database and that the
  database backup completed successfully.
- Confirm that `REDIS_CONN_STRING`, `SESSION_SECRET`, and `CRYPTO_SECRET` are
  configured from the secret store. Never commit or paste their values into
  logs, tickets, or this repository.
- For HTTPS deployments, set `SESSION_COOKIE_SECURE=true` and configure the
  exact public origins in `SESSION_COOKIE_TRUSTED_URL`.
- Set explicit `TRUSTED_PROXIES` values or `none`; do not rely on the default
  compatibility warning in production.

## 2. Deployment and migration check

For a Docker deployment:

```bash
docker compose up -d
docker compose ps
curl -fsS http://127.0.0.1:3000/api/status
```

For a systemd deployment, restart the service and verify the health endpoint:

```bash
sudo systemctl restart new-api
sudo systemctl is-active new-api
curl -fsS https://<public-host>/api/status
```

The response must contain `"success":true`. Check startup logs for migration
errors before sending traffic:

```bash
docker logs --since=10m <container> | grep -Ei 'fatal|panic|migration|error'
sudo journalctl -u new-api --since '-10 minutes' --no-pager
```

Financial migrations are expected to be idempotent. Run the health check again
after a second restart; do not delete volumes to resolve a migration failure.

## 3. Daily financial checks

1. Open the administrator **Provider Wallet** page.
2. Check provider wallet totals and per-channel balances.
3. Review channels marked **Below threshold** or **Auto disabled**.
4. Confirm recent deposits and allocations in **Cash flow history**.
5. Use the date and direction filters, then confirm that the report totals match
   the visible rows for each currency.
6. Export the filtered report and retain it with the operational record when a
   reconciliation is required.

Provider credit thresholds are safeguards for the provider balance reported by
each channel. They are not customer quotas. A channel below its configured
threshold is automatically disabled; it is re-enabled after the balance is
restored. A manual channel disable must not be overridden by this policy.

## 4. Reconciliation procedure

For each currency and reporting period:

1. Export the filtered cash-flow CSV from the administrator console.
2. Reconcile incoming funding against provider payment records.
3. Reconcile outgoing allocations against the customer or tenant funding
   records.
4. Verify:

   ```text
   net = incoming - outgoing
   ```

5. Investigate duplicate references, missing counterparties, unexpected
   currencies, or a negative available balance before making any adjustment.

Ledger entries and completed financial transactions are append-only. Do not
edit or delete them directly. Corrections must be represented by a new,
auditable adjustment or reversal flow approved by the operator.

## 5. Provider credit response

When a channel is automatically disabled:

1. Verify the provider balance directly in the provider console.
2. Confirm the configured threshold in Provider Wallet.
3. Fund the provider account through the approved payment process.
4. Refresh or synchronize the channel balance.
5. Confirm that the channel is enabled only after the balance is above the
   threshold.

Do not lower the threshold solely to hide an upstream credit problem. If the
channel was manually disabled, resolve that operator action separately.

## 6. Incident: 5xx, timeout, or restart loop

1. Check container or systemd status.
2. Inspect the first fatal startup error, not only the latest restart message.
3. Check database connectivity and available disk space.
4. If the error is a migration failure, stop repeated restarts and take a
   database snapshot before remediation.
5. Restore the last known-good release only after preserving logs and the
   database snapshot.
6. Verify `/api/status` and perform a read-only financial report check before
   reopening traffic.

Example Docker checks:

```bash
docker compose ps
docker logs --tail=200 <container>
docker inspect --format '{{.State.Status}}' <container>
```

Do not expose database DSNs, session cookies, API keys, or raw authentication
errors in incident output.

## 7. Backup and rollback

- Back up the primary database before every release that changes models or
  migrations.
- Keep the previous application binary or image and its exact commit ID.
- Roll back application code first when the schema remains compatible.
- Restore a database backup only with an approved recovery decision; restoring
  data can remove valid financial transactions created after the backup.
- After rollback or restore, verify migration idempotency, wallet balances,
  ledger balance, provider cash-flow totals, and `/api/status`.

## 8. Release sign-off

- [ ] Backup completed and restore point recorded.
- [ ] Application starts without migration errors.
- [ ] Health endpoint returns success after two restarts.
- [ ] Provider wallet totals load for every configured currency.
- [ ] Provider credit threshold and auto-disable state are correct.
- [ ] Filtered cash-flow report totals reconcile with the exported CSV.
- [ ] No secrets or raw sensitive errors appear in logs or release evidence.
- [ ] SQLite, MySQL, and PostgreSQL verification results are recorded before
  declaring database compatibility for production.
