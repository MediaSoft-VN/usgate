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

## Phụ thuộc chính

```text
FIN-001..005
  → FIN-006..017
  → FIN-018..024
  → FIN-025..038
  → FIN-039..046
  → FIN-047..054
```
