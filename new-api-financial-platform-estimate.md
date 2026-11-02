# New API Financial AI Platform — Ước lượng effort và thời gian

## 1. Phạm vi ước lượng

Ước lượng cho việc mở rộng New API thành một nền tảng:

```text
AI Gateway
+ Customer/Organization
+ Customer Wallet
+ Provider Wallet
+ Financial Ledger
+ Payment/Top-up
+ Billing/Pricing
+ Usage Cost
+ Reconciliation
+ Whitelabel
+ Admin Console
```

Đây là ước lượng cho phiên bản production có kiểm thử, audit và vận hành; không chỉ là prototype.

## 2. Kết luận nhanh

| Phạm vi | Effort dự kiến | Team đề xuất | Thời gian lịch |
|---|---:|---|---:|
| Prototype kỹ thuật | 35–55 person-days | 2 backend | 3–5 tuần |
| MVP có wallet + provider credit | 145–190 person-days | 2 BE + 1 FE + QA part-time | 10–14 tuần |
| Production đầy đủ | 330–450 person-days | 3 BE + 2 FE + QA + DevOps/BA | 20–28 tuần |
| Enterprise hardening | 450–600 person-days | 6–8 người | 28–36 tuần |

**Khuyến nghị kế hoạch:** chọn MVP 10–14 tuần trước, sau đó production hardening thêm 8–14 tuần.

## 3. Phân rã module và PERT

Đơn vị: person-days, một ngày = 8 giờ.

| # | Work package | O | M | P | PERT expected | SD |
|---:|---|---:|---:|---:|---:|---:|
| 1 | Discovery, BA, scope và financial rules | 5 | 8 | 14 | 8.2 | 1.5 |
| 2 | Architecture, data model, migration strategy | 7 | 12 | 20 | 12.5 | 2.2 |
| 3 | Tenant, organization, membership, whitelabel | 10 | 18 | 30 | 18.7 | 3.3 |
| 4 | Wallet core: account, balance, reserve, lock | 18 | 30 | 48 | 31.0 | 5.0 |
| 5 | Double-entry financial ledger | 15 | 25 | 42 | 26.2 | 4.5 |
| 6 | Payment order, webhook, top-up, refund | 12 | 20 | 35 | 21.2 | 3.8 |
| 7 | Provider account, provider wallet, provider credit | 13 | 22 | 38 | 23.5 | 4.2 |
| 8 | Pricing, customer billing, markup, invoice | 16 | 28 | 48 | 29.3 | 5.3 |
| 9 | Allocation, budget, daily/monthly limits | 9 | 16 | 28 | 16.8 | 3.2 |
| 10 | AI Gateway integration with financial guard | 12 | 22 | 38 | 23.0 | 4.3 |
| 11 | Usage ingestion and provider cost calculation | 14 | 25 | 44 | 26.3 | 5.0 |
| 12 | Reconciliation and exception handling | 10 | 18 | 34 | 19.3 | 4.0 |
| 13 | Admin web: wallet/provider/billing screens | 24 | 40 | 70 | 42.3 | 7.7 |
| 14 | API integration, RBAC, audit, idempotency | 9 | 15 | 28 | 15.7 | 3.2 |
| 15 | Security: secrets, payment, financial controls | 8 | 15 | 30 | 16.3 | 3.7 |
| 16 | Automated tests, concurrency, failure scenarios | 22 | 35 | 60 | 36.8 | 6.3 |
| 17 | Ops, monitoring, backup, alerting, runbook | 7 | 12 | 22 | 12.8 | 2.5 |
| 18 | Data migration and compatibility with existing quota | 10 | 18 | 35 | 19.2 | 4.2 |
| 19 | Documentation, training, release checklist | 5 | 8 | 15 | 8.3 | 1.7 |
|  | **Tổng PERT** |  |  |  | **~387 person-days** |  |

PERT formula:

```text
Expected = (O + 4M + P) / 6
SD       = (P - O) / 6
```

## 4. Phạm vi MVP 10–14 tuần

MVP không làm toàn bộ enterprise scope. Chỉ triển khai:

### Backend

- Customer wallet.
- Provider wallet/credit.
- Financial transaction đơn giản nhưng có idempotency.
- Manual customer top-up.
- Manual provider top-up.
- Provider account và channel link.
- Usage log → provider cost → customer charge.
- Cấp quota cho user/organization.
- Low balance alert.
- Disable channel khi provider credit hết.

### Frontend

- Customer wallet.
- Provider wallet.
- Provider top-up record.
- Usage/billing list.
- Channel/provider status.
- Basic pricing configuration.

### Chưa đưa vào MVP

- Payment gateway tự động nhiều nhà cung cấp.
- Auto top-up provider.
- Multi-currency.
- Invoice thuế đầy đủ.
- Approval workflow nhiều cấp.
- ClickHouse financial ledger.
- Cross-tenant settlement.
- Mobile app.

MVP ước lượng: **145–190 person-days**.

## 5. Roadmap theo tuần

### Tuần 1–2: Foundation

- Chốt business rules.
- Chốt đơn vị tiền và quota.
- Chốt wallet ownership.
- Thiết kế schema.
- Thiết kế idempotency và audit.
- Tạo migration skeleton.

### Tuần 3–5: Wallet và Ledger

- Wallet account.
- Balance/reservation.
- Financial transaction.
- Debit/credit entries.
- Lock/concurrency control.
- Customer wallet API.
- Provider wallet API.

### Tuần 6–7: Provider Finance

- Provider accounts.
- Provider credit.
- Provider top-up manual.
- Balance snapshot.
- Channel threshold.
- Disable/re-enable channel.

### Tuần 8–9: Billing và Usage

- Usage event.
- Provider cost.
- Customer charge.
- Markup/pricing.
- Quota guard.
- Usage reconciliation cơ bản.

### Tuần 10–11: Frontend và Operations

- Wallet screens.
- Provider finance screens.
- Billing screens.
- Alerts.
- Audit view.
- Export CSV.

### Tuần 12–14: Test và release

- Payment/top-up idempotency.
- Concurrent wallet debit.
- Provider failure.
- Retry/fallback.
- Streaming.
- Negative balance prevention.
- Migration rehearsal.
- Staging release.

## 6. Production đầy đủ 20–28 tuần

Sau MVP cần thêm:

### Tài chính

- Double-entry hoàn chỉnh.
- Refund/adjustment approval.
- Invoice.
- Multi-currency.
- Tax/fee policy.
- Period close.
- Immutable audit.

### Provider operations

- Provider balance API integration.
- Reconciliation tự động.
- Provider invoice import.
- Auto channel draining.
- Provider failover.
- Cost anomaly detection.

### Enterprise

- Organization hierarchy.
- Cost center/project.
- Member budgets.
- Approval workflow.
- SSO/OIDC.
- Per-tenant branding.
- Custom domain.

### Reliability

- Multi-node wallet locking.
- Outbox/event processing.
- Dead-letter queue.
- Replay usage event.
- Disaster recovery.
- Financial report consistency checks.

## 7. Phân bổ theo vai trò

Với production scope khoảng 387 person-days:

| Vai trò | Effort dự kiến |
|---|---:|
| Backend Go/New API | 145–175 days |
| Frontend React | 65–85 days |
| QA/Test automation | 55–70 days |
| DevOps/SRE | 20–30 days |
| BA/PM/Design/Documentation | 35–50 days |
| Security/Code review | 15–25 days |
| **Tổng** | **335–435 days** |

Một team khả thi:

```text
3 Backend
2 Frontend
1 QA
0.5 DevOps
0.5 BA/PM
```

Với team này, 387 person-days không tương đương 12 tuần tuyệt đối vì còn dependency, review, test, bug fixing và gate approval. Lịch thực tế nên dự phòng **20–28 tuần**.

## 8. Các yếu tố có thể làm tăng thời gian

| Rủi ro | Tác động |
|---|---:|
| Không chốt rõ tiền khách hàng và provider credit | +2–4 tuần |
| Muốn tự động top-up từng provider | +3–6 tuần/provider |
| Multi-currency và tỷ giá | +2–4 tuần |
| Giữ tương thích đầy đủ với quota cũ | +2–3 tuần |
| Fork New API nhiều phần | +20–35% maintenance effort |
| Sửa cả frontend New API và USGate | +4–8 tuần |
| Financial audit/compliance nghiêm ngặt | +3–6 tuần |
| Realtime billing chính xác cho streaming | +2–4 tuần |

## 9. Giả định ước lượng

- New API tiếp tục dùng Go/Gin/GORM.
- PostgreSQL là database chính.
- Redis đã có thể dùng cho lock/cache.
- Provider top-up MVP là manual confirmation.
- Một currency ban đầu.
- Pricing theo token input/output.
- Provider cost lấy từ usage/log hoặc provider API nếu có.
- Không xây lại toàn bộ protocol conversion hiện có.
- Không thay đổi toàn bộ cơ chế quota cũ trong release đầu.
- Có người phụ trách chốt business rule và payment flow.

## 10. Khuyến nghị lịch triển khai

Không nên bắt đầu với toàn bộ 387 person-days ngay lập tức. Nên chốt milestone:

```text
Milestone 1 — 2 tuần: Schema + wallet prototype
Milestone 2 — 5 tuần: Wallet + ledger + provider credit
Milestone 3 — 9 tuần: Usage billing + gateway guard
Milestone 4 — 12 tuần: Admin UI + reconciliation cơ bản
Milestone 5 — 14 tuần: MVP staging/go-live
Milestone 6 — 20–28 tuần: Production hardening
```

## 11. Kết luận

Ước lượng hợp lý:

```text
Prototype:       3–5 tuần
MVP:             10–14 tuần
Production:      20–28 tuần
Enterprise:      28–36 tuần
```

Phần khó nhất không phải gọi OpenAI/Anthropic. Phần khó nhất là:

1. Financial ledger không sai số.
2. Không trừ tiền trùng khi retry/webhook.
3. Đồng bộ provider cost và customer charge.
4. Streaming nhưng vẫn ghi nhận usage đúng.
5. Concurrency khi nhiều request dùng chung một wallet.
6. Migration quota cũ sang mô hình wallet mới.

---

## 12. Ước lượng lại khi sử dụng AI để code và tự test

### 12.1. Giả định mới

- AI được dùng để tạo code, migration, test, fixture, mock provider và tài liệu.
- AI có quyền chạy build, unit test, integration test và E2E trong môi trường local/staging.
- AI tự đọc lỗi, sửa code và chạy regression test.
- Không tính việc AI tự phê duyệt business rule hoặc tự deploy production.
- Financial ledger, payment settlement và release production vẫn cần review độc lập.

### 12.2. Mức giảm effort thực tế

| Nhóm công việc | Mức giảm nhờ AI | Lý do |
|---|---:|---|
| Boilerplate backend/API | 35–50% | AI tạo model, route, DTO, CRUD, migration |
| Provider adapter | 30–45% | Có interface và test mock rõ ràng |
| Frontend admin | 35–50% | Tạo form, table, filter, API hook |
| Unit/integration test | 40–60% | AI sinh test và fixture |
| E2E/test evidence | 25–40% | AI viết flow và chạy browser |
| Financial design/review | 10–20% | Cần quyết định nghiệp vụ và kiểm soát độc lập |
| Security/concurrency/payment review | 10–25% | Không thể bỏ qua kiểm tra chuyên môn |
| Ops/production release | 15–25% | AI hỗ trợ script nhưng không thay thế approval |

AI không làm giảm tương ứng toàn bộ lịch vì các task vẫn phụ thuộc tuần tự và phải review.

### 12.3. Effort mới

| Phạm vi | Trước đây | Dùng AI code + AI test | Dự phòng 15% |
|---|---:|---:|---:|
| Prototype | 35–55 PD | 25–40 PD | 29–46 PD |
| MVP | 145–190 PD | 100–135 PD | 115–155 PD |
| Production đầy đủ | 330–450 PD | 255–340 PD | 293–391 PD |
| Enterprise hardening | 450–600 PD | 350–470 PD | 403–541 PD |

PD = person-day.

### 12.4. Thời gian lịch mới

| Phạm vi | Team tối thiểu | Team khuyến nghị | Thời gian thực tế |
|---|---|---|---:|
| Prototype | 1 BE + AI | 2 BE | 2–3 tuần |
| MVP | 2 BE + 1 FE + QA part-time | 2 BE + 1 FE + QA/DevOps part-time | 8–12 tuần |
| Production | 3 BE + 2 FE + QA + DevOps | 3 BE + 2 FE + 1 QA + 0.5 DevOps + 0.5 BA | 16–22 tuần |
| Enterprise | 3 BE + 2 FE + QA + DevOps + BA | 6–8 người | 24–32 tuần |

AI giúp rút ngắn khoảng 20–30% lịch production, không phải 50%, vì ledger, payment, reconciliation và release vẫn cần nhiều vòng kiểm tra.

## 13. Chi phí phát triển khi dùng AI

### 13.1. Chi phí nhân sự theo blended rate

Giả định blended rate đã bao gồm backend, frontend, QA, DevOps và PM/BA:

| Blended rate | Prototype | MVP | Production | Enterprise |
|---:|---:|---:|---:|---:|
| 4 triệu VND/PD | 100–160 triệu | 400–540 triệu | 1.02–1.36 tỷ | 1.40–1.88 tỷ |
| 6 triệu VND/PD | 150–240 triệu | 600–810 triệu | 1.53–2.04 tỷ | 2.10–2.82 tỷ |
| 8 triệu VND/PD | 200–320 triệu | 800 triệu–1.08 tỷ | 2.04–2.72 tỷ | 2.80–3.76 tỷ |

Các con số trên chưa bao gồm VAT, payment provider fee và chi phí vận hành production.

### 13.2. Chi phí hạ tầng và công cụ

| Hạng mục | Prototype/MVP | Production |
|---|---:|---:|
| PostgreSQL/Redis/object storage | 5–20 triệu VND/tháng | 20–80 triệu VND/tháng |
| Monitoring/log/backup | 3–15 triệu VND/tháng | 15–60 triệu VND/tháng |
| AI coding tools | 2–10 triệu VND/tháng/team | 5–25 triệu VND/tháng/team |
| Security scan/penetration test | 20–80 triệu/lần | 80–250 triệu/lần |
| Payment/provider sandbox | Theo provider | Theo provider |

### 13.3. Ngân sách khuyến nghị

| Milestone | Effort | Ngân sách development ở 6 triệu VND/PD |
|---|---:|---:|
| Prototype | 25–40 PD | 150–240 triệu VND |
| MVP staging | 100–135 PD | 600–810 triệu VND |
| Contingency 15% | 15–20 PD | 90–120 triệu VND |
| **MVP go-live budget** | **115–155 PD** | **690–930 triệu VND** |
| Production hardening sau MVP | 140–205 PD | 840 triệu–1.23 tỷ VND |

## 14. Kết luận cập nhật

Với AI làm developer và tester:

```text
Prototype:       2–3 tuần / 100–320 triệu VND
MVP go-live:     8–12 tuần / 690–930 triệu VND ở rate 6M/PD
Production:      16–22 tuần / 1.53–2.72 tỷ VND
Enterprise:      24–32 tuần / 2.10–3.76 tỷ VND
```

Khoản không nên cắt giảm dù dùng AI:

- Financial ledger review.
- Payment/webhook test độc lập.
- Concurrency và double-spend test.
- Security review API key/payment.
- Reconciliation với provider.
- Staging soak test.
- Backup/restore và production rollback.

---

## 15. Kịch bản AI thực hiện gần như toàn bộ

### 15.1. Giả định

AI agent có thể đọc source, tạo/sửa Go/GORM/React, tạo migration, chạy Docker, PostgreSQL, Redis, unit test, integration test, E2E test, đọc log, tự sửa lỗi và tạo test evidence. Con người chỉ cung cấp requirements, sandbox credentials và phê duyệt release.

### 15.2. Thời gian

| Phạm vi | AI execution | Human acceptance/security | Calendar time |
|---|---:|---:|---:|
| Prototype | 80–160 agent-hours | 2–4 person-days | 1–3 tuần |
| MVP staging | 300–600 agent-hours | 8–15 person-days | 4–8 tuần |
| MVP production-ready | 600–1,000 agent-hours | 15–25 person-days | 7–12 tuần |
| Production đầy đủ | 1,500–2,500 agent-hours | 30–50 person-days | 12–20 tuần |

### 15.3. Chi phí

| Hạng mục | Prototype | MVP | Production |
|---|---:|---:|---:|
| AI model/API/subscription | 2–10 triệu | 10–40 triệu | 30–120 triệu |
| CI runner, DB, browser, storage | 2–8 triệu | 8–30 triệu | 30–100 triệu |
| Human acceptance/security | 10–30 triệu | 40–100 triệu | 120–300 triệu |
| Staging/monitoring/backup | 2–8 triệu | 10–30 triệu | 30–100 triệu |
| **Tổng một lần** | **16–56 triệu** | **68–200 triệu** | **210–620 triệu** |

### 15.4. Ngân sách khuyến nghị

```text
Prototype:           1–3 tuần / 16–56 triệu VND
MVP staging:         4–8 tuần / 68–200 triệu VND
MVP production:      7–12 tuần / 120–300 triệu VND
Production đầy đủ:   12–20 tuần / 210–620 triệu VND
```

Đây là chi phí cho AI tooling, hạ tầng test và acceptance/security; chưa bao gồm payment fee, provider fee và production infrastructure theo traffic.

### 15.5. Giới hạn bắt buộc

AI có thể tự code và test, nhưng không nên tự quyết định:

- Số dư đúng khi provider lệch balance.
- Payment settlement thật.
- Chấp nhận rủi ro double-spend hoặc mất tiền.
- Security/compliance sign-off.
- Cấp production/payment credentials.
- Rollback khi ledger có sai lệch.

Vì vậy, mức thấp nhất phù hợp cho prototype/staging. Production tài chính vẫn cần human acceptance gate dù code và test được AI thực hiện.
