# Product Backlog

Phạm vi: MVP nền tảng AI Gateway có wallet, provider credit và usage billing.

Trạng thái: `todo` → `processing` → `done` → `reviewing` → `reviewed`.

## P0 — Foundation

- [ ] **FIN-001** Chốt business rules về tiền, quota, ownership và rounding.
- [ ] **FIN-002** Thiết kế schema wallet, transaction, ledger và usage event.
- [ ] **FIN-003** Tạo migration tương thích SQLite, MySQL và PostgreSQL.
- [ ] **FIN-004** Thiết kế idempotency key và financial audit event.
- [ ] **FIN-005** Xác định cơ chế tương thích với quota hiện tại.

## P0 — Customer wallet

- [ ] **FIN-006** Tạo customer wallet/account.
- [ ] **FIN-007** Implement balance, debit và credit.
- [ ] **FIN-008** Implement reserve/release funds.
- [ ] **FIN-009** Ngăn số dư âm.
- [ ] **FIN-010** Implement manual customer top-up.
- [ ] **FIN-011** Tạo API lịch sử giao dịch.
- [ ] **FIN-012** Implement refund/adjustment cơ bản.

## P0 — Ledger và giao dịch

- [ ] **FIN-013** Implement double-entry ledger cơ bản.
- [ ] **FIN-014** Implement locking và concurrency control.
- [ ] **FIN-015** Đảm bảo debit/credit idempotent.
- [ ] **FIN-016** Implement transaction consistency checks.
- [ ] **FIN-017** Tạo financial audit log không chứa dữ liệu nhạy cảm.

## P0 — Provider finance

- [ ] **FIN-018** Tạo provider account và provider wallet.
- [ ] **FIN-019** Liên kết provider account với channel.
- [ ] **FIN-020** Implement manual provider top-up.
- [ ] **FIN-021** Lưu provider balance snapshot.
- [ ] **FIN-022** Implement provider credit threshold.
- [ ] **FIN-023** Disable channel khi provider hết credit.
- [ ] **FIN-024** Re-enable channel khi credit được bổ sung.

## P0 — Usage và billing

- [ ] **FIN-025** Chuẩn hóa usage event từ request và task.
- [ ] **FIN-026** Tính provider cost.
- [ ] **FIN-027** Tính customer charge và markup.
- [ ] **FIN-028** Tích hợp financial guard trước khi relay.
- [ ] **FIN-029** Settlement sau request thành công.
- [ ] **FIN-030** Refund khi request/provider thất bại.
- [ ] **FIN-031** Hỗ trợ retry, fallback và idempotent settlement.
- [ ] **FIN-032** Hỗ trợ billing cho streaming.
- [ ] **FIN-033** Implement usage reconciliation cơ bản.

## P1 — Quota và cảnh báo

- [ ] **FIN-034** Cấp quota cho user.
- [ ] **FIN-035** Cấp quota cho organization.
- [ ] **FIN-036** Implement daily/monthly spending limit.
- [ ] **FIN-037** Implement low-balance alert.
- [ ] **FIN-038** Chặn request khi không đủ số dư/quota.

## P1 — Admin frontend

- [ ] **FIN-039** Customer wallet screen.
- [ ] **FIN-040** Provider wallet và credit screen.
- [ ] **FIN-041** Top-up records screen.
- [ ] **FIN-042** Usage/billing screen.
- [ ] **FIN-043** Channel/provider status screen.
- [ ] **FIN-044** Pricing configuration screen.
- [ ] **FIN-045** Audit log screen.
- [ ] **FIN-046** Export CSV.

## P0 — Kiểm thử và release

- [ ] **FIN-047** Unit/integration test cho wallet và ledger.
- [ ] **FIN-048** Test concurrent debit và negative balance prevention.
- [ ] **FIN-049** Test retry/webhook replay và idempotency.
- [ ] **FIN-050** Test provider failure, fallback và refund.
- [ ] **FIN-051** Test streaming settlement.
- [ ] **FIN-052** Test migration trên SQLite, MySQL và PostgreSQL.
- [ ] **FIN-053** Test migration fresh database và upgrade database cũ.
- [ ] **FIN-054** Bổ sung monitoring, alerting và release checklist.

## Deferred — Không thuộc MVP

- Payment gateway tự động nhiều nhà cung cấp.
- Auto top-up provider.
- Multi-currency và tỷ giá.
- Hóa đơn thuế đầy đủ.
- Approval workflow nhiều cấp.
- ClickHouse financial ledger.
- Cross-tenant settlement.
- Mobile app.

## P1 — AI Gateway và provider operations

- [ ] **FIN-055** Chuẩn hóa provider/channel health check.
- [ ] **FIN-056** Provider balance API integration.
- [ ] **FIN-057** Tự động reconcile provider usage và provider invoice.
- [ ] **FIN-058** Channel draining trước khi disable channel.
- [ ] **FIN-059** Provider failover theo health, credit và latency.
- [ ] **FIN-060** Phát hiện cost anomaly.
- [ ] **FIN-061** Quản lý model mapping, priority, weight và channel affinity.
- [ ] **FIN-062** Quản lý nhiều API key cho một provider/channel.
- [ ] **FIN-063** Chuẩn hóa retry policy và circuit breaker.
- [ ] **FIN-064** Theo dõi latency, error rate và availability theo provider.

## P1 — Tài chính production

- [ ] **FIN-065** Hoàn thiện double-entry ledger bất biến.
- [ ] **FIN-066** Refund/adjustment approval workflow.
- [ ] **FIN-067** Immutable financial audit trail.
- [ ] **FIN-068** Invoice và billing statement.
- [ ] **FIN-069** Multi-currency và exchange rate policy.
- [ ] **FIN-070** Tax/fee policy.
- [ ] **FIN-071** Period close và khóa kỳ kế toán.
- [ ] **FIN-072** Financial report consistency checks.
- [ ] **FIN-073** Reconciliation exception queue.
- [ ] **FIN-074** Export financial reports.

## P1 — Organization và enterprise

- [ ] **FIN-075** Organization hierarchy.
- [ ] **FIN-076** Membership, role và RBAC theo organization.
- [ ] **FIN-077** Cost center và project.
- [ ] **FIN-078** Member budget và spending limit.
- [ ] **FIN-079** Approval workflow cho top-up/adjustment.
- [ ] **FIN-080** SSO/OIDC theo tenant.
- [ ] **FIN-081** Per-tenant branding.
- [ ] **FIN-082** Custom domain.
- [ ] **FIN-083** Cross-tenant settlement policy.

## P1 — Reliability và security

- [ ] **FIN-084** Distributed wallet locking cho nhiều node.
- [ ] **FIN-085** Outbox/event processing.
- [ ] **FIN-086** Dead-letter queue và retry event.
- [ ] **FIN-087** Replay usage event an toàn.
- [ ] **FIN-088** Disaster recovery và backup/restore rehearsal.
- [ ] **FIN-089** Rate limit và quota guard theo tenant.
- [ ] **FIN-090** Secret rotation cho provider credentials.
- [ ] **FIN-091** Security audit cho payment, wallet và API credentials.
- [ ] **FIN-092** Monitoring dashboard và alert routing.
- [ ] **FIN-093** Runbook xử lý lỗi tài chính và provider.

## P2 — AI Gateway mở rộng

- [ ] **FIN-094** Hoàn thiện OpenAI Chat Completions/Responses relay.
- [ ] **FIN-095** Hoàn thiện Anthropic Messages relay.
- [ ] **FIN-096** Hoàn thiện Gemini relay.
- [ ] **FIN-097** Hỗ trợ realtime/WebSocket theo channel.
- [ ] **FIN-098** Hỗ trợ image, audio, embedding và rerank.
- [ ] **FIN-099** Hỗ trợ multimodal input và file handling.
- [ ] **FIN-100** Tool calling và reasoning compatibility tests.
- [ ] **FIN-101** JavaScript task plugin lifecycle và task polling.
- [ ] **FIN-102** Task artifact storage và download authorization.
- [ ] **FIN-103** Plugin marketplace/install/update management.
- [ ] **FIN-104** Playground cho debug request và response.
- [ ] **FIN-105** API documentation và OpenAPI contract verification.

## P2 — Console và vận hành

- [ ] **FIN-106** Dashboard request, cost, latency và error metrics.
- [ ] **FIN-107** Usage log search/filter/export.
- [ ] **FIN-108** Channel/model configuration console.
- [ ] **FIN-109** User, group, API key và access-token management.
- [ ] **FIN-110** Subscription/package management.
- [ ] **FIN-111** OAuth/OIDC, MFA, passkey và session management review.
- [ ] **FIN-112** System settings và feature flags.
- [ ] **FIN-113** Localization đầy đủ cho các màn hình mới.
- [ ] **FIN-114** Accessibility và responsive UI audit.
- [ ] **FIN-115** E2E test cho các luồng admin và customer.

## P2 — Data migration và compatibility

- [ ] **FIN-116** Migration quota cũ sang wallet mới.
- [ ] **FIN-117** Migration pricing/subscription cũ.
- [ ] **FIN-118** Backfill usage event từ usage logs.
- [ ] **FIN-119** Kiểm tra backward compatibility API.
- [ ] **FIN-120** Rollback migration và disaster recovery rehearsal.
- [ ] **FIN-121** Kiểm thử dữ liệu trên SQLite, MySQL và PostgreSQL.

## Phụ thuộc chính

```text
FIN-001..005
  → FIN-006..017
  → FIN-018..024
  → FIN-025..038
  → FIN-039..046
  → FIN-047..054
  → FIN-055..074
  → FIN-075..093
  → FIN-094..115
  → FIN-116..121
```
