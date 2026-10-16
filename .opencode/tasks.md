# New API Task Backlog

## Execution rules

- Work on one task at a time.
- Move a task through `todo` → `in_progress` → `tested` → `committed` → `done`.
- Every implementation task must define observable acceptance criteria and an
  E2E scenario when it crosses an HTTP, payment, relay, or UI boundary.
- Do not start the next task until the current task's required checks pass and
  its changes are committed.
- Keep unrelated existing working-tree changes out of the task commit.
- Update this file when a task changes status; include the commit hash for
  completed implementation tasks.

## Current task

- None selected.

## P0 — MVP financial platform

- [ ] **FIN-001** Implement customer financial wallet.
- [ ] **FIN-002** Implement wallet debit, credit, reserve, release, and refund.
- [ ] **FIN-003** Integrate the financial wallet with the existing billing flow.
- [ ] **FIN-004** Integrate the financial ledger with SePay top-up settlement.
- [ ] **FIN-005** Complete payment webhook idempotency and replay handling.
- [ ] **FIN-006** Add provider accounts.
- [ ] **FIN-007** Add provider wallets and provider credit limits.
- [ ] **FIN-008** Link providers to channels.
- [ ] **FIN-009** Disable and re-enable channels based on provider credit.
- [ ] **FIN-010** Record usage events.
- [ ] **FIN-011** Calculate provider cost from usage.
- [ ] **FIN-012** Calculate customer charge and markup.
- [ ] **FIN-013** Reconcile usage, provider cost, and customer charge.
- [ ] **FIN-014** Add quota allocation for users and organizations.
- [ ] **FIN-015** Add low-balance alerts.
- [ ] **FIN-016** Add customer wallet and transaction APIs.
- [ ] **FIN-017** Add provider wallet and provider transaction APIs.
- [ ] **FIN-018** Add customer wallet frontend screens.
- [ ] **FIN-019** Add provider wallet frontend screens.
- [ ] **FIN-020** Add billing and usage frontend screens.
- [ ] **FIN-021** Add basic pricing configuration frontend.
- [ ] **FIN-022** Add provider and channel status frontend.
- [ ] **FIN-023** Add financial audit view.
- [ ] **FIN-024** Add CSV export for financial records.
- [ ] **FIN-025** Test payment retry and webhook replay.
- [ ] **FIN-026** Test concurrent debit and double-spend prevention.
- [ ] **FIN-027** Verify migrations on SQLite, MySQL, and PostgreSQL.

## P1 — Missing API and relay functionality

- [ ] **API-001** Implement `POST /v1/images/variations`.
- [ ] **API-002** Implement the Files API.
- [ ] **API-003** Implement the Fine-tuning API.
- [ ] **API-004** Implement model deletion.
- [ ] **API-005** Implement Gemini `countTokens`.
- [ ] **RELAY-001** Complete Ali adapter conversions.
- [ ] **RELAY-002** Complete AWS adapter conversions.
- [ ] **RELAY-003** Complete Baidu and Baidu V2 adapter conversions.
- [ ] **RELAY-004** Complete Claude adapter conversions.
- [ ] **RELAY-005** Complete Cloudflare adapter conversions.
- [ ] **RELAY-006** Complete Cohere adapter conversions.
- [ ] **RELAY-007** Complete Coze adapter conversions.
- [ ] **RELAY-008** Complete DeepSeek adapter conversions.
- [ ] **RELAY-009** Complete Dify adapter conversions.
- [ ] **RELAY-010** Complete Gemini adapter conversions.
- [ ] **RELAY-011** Complete Jimeng and Jina adapter conversions.
- [ ] **RELAY-012** Complete Mistral, Minimax, and MokaAI adapter conversions.
- [ ] **RELAY-013** Complete Moonshot, Ollama, and Palm adapter conversions.
- [ ] **RELAY-014** Complete Perplexity and Replicate adapter conversions.
- [ ] **RELAY-015** Complete SiliconFlow, Tencent, and Vertex adapter conversions.
- [ ] **RELAY-016** Complete Volcengine, XAI, and Xunfei adapter conversions.
- [ ] **RELAY-017** Complete Zhipu and Zhipu 4V adapter conversions.
- [ ] **RELAY-018** Classify unsupported adapter methods explicitly and remove
  misleading `not implemented` stubs where unsupported behavior is intentional.

## P2 — Production hardening

- [ ] **PROD-001** Complete multi-account double-entry ledger.
- [ ] **PROD-002** Add refund and adjustment approval.
- [ ] **PROD-003** Add invoices and tax/fee policy.
- [ ] **PROD-004** Complete multi-currency support.
- [ ] **PROD-005** Add accounting period close.
- [ ] **PROD-006** Integrate provider balance APIs.
- [ ] **PROD-007** Automate provider reconciliation.
- [ ] **PROD-008** Import provider invoices.
- [ ] **PROD-009** Add channel draining and provider failover.
- [ ] **PROD-010** Add cost anomaly detection.
- [ ] **PROD-011** Add organization hierarchy, cost centers, and projects.
- [ ] **PROD-012** Add member budgets and approval workflow.
- [ ] **PROD-013** Add SSO/OIDC, tenant branding, and custom domains.
- [ ] **PROD-014** Add multi-node wallet locking.
- [ ] **PROD-015** Add outbox, retry, and dead-letter processing.
- [ ] **PROD-016** Add usage-event replay.
- [ ] **PROD-017** Add backup, restore, and disaster recovery procedures.
- [ ] **PROD-018** Add financial consistency checks.
- [ ] **PROD-019** Implement S3 task artifact storage.
- [ ] **PROD-020** Complete Azure billing and asynchronous billing work.
