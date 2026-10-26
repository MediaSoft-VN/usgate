# Hướng dẫn vận hành nền tảng tài chính

Tài liệu này hướng dẫn từng bước vận hành các tính năng đã triển khai: tenant
workspace, membership và domain, customer wallet, ledger bất biến, provider
wallet, provider funding, provider credit threshold, cash-flow report và CSV
export.

> **Phạm vi hiện tại:** đây là MVP vận hành tài chính. Invoice, approval
> workflow và tự động đối soát nâng cao chưa được triển khai. Customer quota là
> số dư sử dụng; financial wallet và ledger là sổ cái nội bộ riêng biệt.

## Quy tắc an toàn bắt buộc

1. Chỉ Administrator được thực hiện funding, allocation, threshold và
   reconciliation.
2. Không ghi secret, access token, DSN, cookie phiên hoặc dữ liệu xác thực vào
   log, ticket, ảnh chụp màn hình hay repository.
3. Không sửa hoặc xoá transaction, ledger entry, funding hay allocation đã hoàn
   tất. Correction phải là transaction adjustment hoặc reversal mới.
4. Luôn lưu reference và CSV export của từng kỳ đối soát.
5. Không hạ threshold chỉ để che giấu lỗi số dư provider.
6. Customer wallet không hiển thị “Available balance”; không dùng giao diện
   đó để thay thế cho số dư quota.

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

## 9. Vận hành Partner Workspace

1. Đăng nhập bằng tài khoản có quyền quản lý tenant và mở
   `/partner-workspace`.
2. Chọn workspace cần thao tác.
3. Dùng các màn hình tương ứng:
   - `/partner-workspace/overview`: thống kê workspace.
   - `/partner-workspace/users`: quản lý thành viên và role.
   - `/partner-workspace/domains`: thêm, xác minh hoặc xoá domain.
   - `/partner-workspace/wallet`: số dư và giao dịch wallet workspace.
   - `/partner-workspace/transactions`: lịch sử transaction.
4. Khi thêm member, chọn đúng user và role, lưu, sau đó xác nhận user chỉ thấy
   dữ liệu thuộc workspace.
5. Khi thêm domain, nhập hostname, lưu, tạo DNS record theo giá trị xác minh,
   chờ DNS propagation rồi chọn **Verify**.
6. Khi xoá member hoặc domain, kiểm tra các liên kết đang sử dụng trước khi
   xác nhận.

## 10. Customer wallet và ledger

### 10.1 Phân biệt số dư

- **Customer quota:** số dư sử dụng của user cho relay/billing hiện hành.
- **Financial wallet:** số dư tiền nội bộ theo owner và currency.
- **Ledger:** các dòng debit/credit bất biến dùng để audit financial wallet.

Không dùng một loại số dư thay thế loại còn lại. Mọi operation wallet cần
owner, currency, amount ở minor unit, reference và idempotency key.

### 10.2 Thực hiện operation

1. Xác định owner, currency và amount minor unit.
2. Tạo reference/idempotency key duy nhất.
3. Kiểm tra wallet ở trạng thái active.
4. Với reserve, xác nhận available balance đủ.
5. Gọi service/model wallet; không cập nhật trực tiếp cột balance.
6. Xác nhận transaction completed và debit/credit ledger cân bằng.
7. Nếu retry cùng idempotency key, xác nhận kết quả cũ được trả lại và không
   phát sinh debit/credit thứ hai.
8. Hoàn tiền hoặc correction bằng release/refund/adjustment mới, không sửa
   transaction cũ.

### 10.3 Kiểm tra ledger

1. Tra transaction bằng transaction number hoặc external reference.
2. Kiểm tra status, currency và amount minor.
3. Đối chiếu tổng debit với tổng credit.
4. Nếu không cân bằng, dừng thao tác tài chính và mở incident; không sửa bảng
   trực tiếp.

## 11. Provider funding và provider wallet

### 11.1 Ghi nhận funding

1. Mở `/provider-funding`.
2. Chọn channel và currency.
3. Chọn **Create funding**.
4. Nhập amount minor unit, reference, provider reference và note.
5. Kiểm tra currency/amount rồi submit một lần.
6. Xác nhận record xuất hiện ở incoming cash flow.
7. Nếu timeout, tra reference trước khi tạo lại; không tạo reference mới chỉ vì
   chưa thấy response.

### 11.2 Allocation

1. Mở `/provider-wallet`.
2. Chọn provider wallet và currency.
3. Chọn user/tenant nhận allocation.
4. Nhập amount, quota, exchange rate và reference.
5. Xác nhận available provider funding đủ rồi submit.
6. Kiểm tra available provider funding giảm đúng amount.
7. Kiểm tra user/tenant nhận đúng quota và outgoing cash flow có allocation.

Nếu gặp insufficient funding hoặc quota limit, không retry vô hạn; kiểm tra
transaction gần nhất và số dư thực tế trước.

### 11.3 Đồng bộ

1. Mở `/provider-funding` và kiểm tra sync status.
2. Chỉ chọn **Start sync** khi không có job đang chạy.
3. Chờ job hoàn tất.
4. Đối chiếu kết quả với provider console.
5. Nếu sync lỗi, giữ nguyên dữ liệu, lưu error không chứa secret và xử lý theo
   mục incident.

## 12. Provider credit threshold

1. Mở trang quản trị channel/provider.
2. Chọn channel và nhập threshold theo currency/đơn vị cấu hình.
3. Lưu và xác nhận threshold đã cập nhật.
4. Khi balance thấp hơn threshold, channel chuyển sang auto-disabled.
5. Fund provider account theo quy trình provider.
6. Refresh hoặc sync balance.
7. Xác nhận channel chỉ auto-enable lại khi balance vượt threshold.
8. Nếu channel bị manual disable, threshold policy không được ghi đè trạng thái
   đó.

Khi channel bị auto-disable, kiểm tra provider console, balance, timestamp sync
và threshold trước khi thay đổi cấu hình. Không hạ threshold để che giấu lỗi
credit.

## 13. Cash-flow report và CSV export

1. Mở `/provider-funding`.
2. Chọn currency, khoảng thời gian `From`/`To` và direction.
3. Dùng **Incoming** cho funding vào provider pool, **Outgoing** cho allocation
   ra user/tenant.
4. Kiểm tra công thức:

   ```text
   net = incoming - outgoing
   ```

5. Đối chiếu summary với các dòng đang hiển thị.
6. Chọn **Export CSV** và lưu file theo kỳ, ví dụ
   `provider-cash-flow-YYYY-MM.csv`.
7. Giữ file gốc; nếu cần ghi chú, tạo bản sao.

API administrator tương ứng:

```text
GET  /api/financial/provider-funding
GET  /api/financial/provider-funding/export
POST /api/financial/provider-funding
POST /api/financial/provider-funding/allocate
GET  /api/financial/provider-funding/sync
POST /api/financial/provider-funding/sync
GET  /api/financial/wallets
```

## 14. Quy trình reconciliation định kỳ

1. Tạo restore point trước khi bắt đầu.
2. Export CSV theo từng currency và kỳ đối soát.
3. Đối chiếu incoming với provider payment records.
4. Đối chiếu outgoing với user/tenant allocation records.
5. Kiểm tra duplicate reference, currency lạ, amount âm và available balance âm.
6. Đối chiếu ledger, wallet transaction và cash-flow record.
7. Ghi lại chênh lệch, reference và người xử lý.
8. Không tự ý tạo adjustment nếu chưa có phê duyệt vận hành nội bộ.

## 15. Incident và rollback

### 15.1 5xx, timeout hoặc restart loop

1. Kiểm tra container/systemd status.
2. Đọc lỗi đầu tiên trong startup log.
3. Kiểm tra database, Redis, disk space và network.
4. Nếu là migration failure, dừng restart lặp và tạo database snapshot.
5. Giữ log, commit/image hiện tại và chỉ rollback sau khi kiểm tra schema.
6. Xác nhận `/api/status`, wallet và report read-only trước khi mở traffic.

### 15.2 Funding bị nghi ghi trùng

1. Tìm theo reference/idempotency key.
2. Kiểm tra financial transaction và cash-flow record.
3. Nếu completed, không submit lại.
4. Nếu pending, xử lý theo retry policy.
5. Chỉ tạo adjustment khi có bằng chứng giao dịch thực tế.

### 15.3 Số dư/report không khớp

1. Chốt currency, khoảng thời gian và direction filter.
2. Export CSV lại.
3. Tìm reference thiếu/trùng.
4. Đối chiếu ledger với wallet và cash-flow.
5. Không sửa trực tiếp dữ liệu production.

## 16. Vận hành custom domain của Partner Workspace

### 16.1 Trạng thái hiện tại

Tính năng domain hiện hỗ trợ:

1. Đối tác thêm hostname.
2. Hệ thống hiển thị DNS record cần cấu hình.
3. Hệ thống kiểm tra CNAME hoặc A record.
4. Domain chuyển từ `pending` sang `verified` khi DNS đúng.
5. Domain đã verified được dùng để resolve tenant theo HTTP `Host`.

Nginx/Certbot **chưa được tự động provision từ màn hình quản lý domain**. Domain
deploy chính của hệ thống có pipeline Nginx/Certbot riêng; pipeline đó không tự
áp dụng cho custom domain của partner.

### 16.2 Quy trình thêm và verify domain

1. Mở `/partner-workspace/domains`.
2. Chọn workspace cần cấu hình.
3. Chọn **Add domain** và nhập hostname, ví dụ `app.example.com`.
4. Ghi lại CNAME hoặc A record trong thẻ **DNS records to configure**.
5. Tạo record tại nhà cung cấp DNS của đối tác.
6. Chờ DNS propagation.
7. Chọn **Verify**.
8. Nếu thành công, kiểm tra trạng thái domain là `verified`.
9. Truy cập thử hostname và kiểm tra request được resolve đúng tenant.

### 16.3 Xử lý lỗi DNS

- `pending`: kiểm tra hostname có bị nhập sai, record có đúng loại và DNS đã
  propagate hay chưa.
- CNAME không khớp: kiểm tra trailing dot, proxy mode của DNS provider và record
  đang bị che bởi CNAME khác.
- A record không khớp: kiểm tra IP public hiện tại của gateway.
- Domain không hợp lệ: không dùng IP, wildcard, localhost, port, path hoặc ký tự
  đặc biệt.
- Domain đã thuộc workspace khác: không xoá hoặc sửa record của workspace kia;
  liên hệ administrator để xử lý ownership.

### 16.4 Quy trình khi triển khai tự động Nginx/SSL

Khi host-side domain provisioner được bật, luồng chuẩn sẽ là:

1. Chỉ provision domain có trạng thái `verified`.
2. Tạo file Nginx riêng theo `domain_id`, không dùng hostname làm tên file.
3. Tạo HTTP-only ACME challenge config.
4. Chạy `nginx -t` trước mỗi reload.
5. Dùng Certbot cấp certificate riêng cho hostname.
6. Tạo HTTPS reverse proxy về application port.
7. Giữ `Host`, `X-Forwarded-Proto`, WebSocket headers và streaming timeout.
8. Chỉ chuyển domain sang `ready` sau khi HTTPS health check thành công.

Không cho API container ghi trực tiếp `/etc/nginx` hoặc chạy `sudo`. Provisioner
phải chạy trên host qua root-owned wrapper với allowlist thao tác cố định.

### 16.5 Xử lý lỗi provisioner

- DNS chưa verified: không gọi Certbot.
- Certbot lỗi: giữ domain `verified`, ghi trạng thái `failed` và retry.
- `nginx -t` lỗi: không reload, giữ cấu hình đang chạy.
- Reload lỗi: rollback cấu hình trước đó.
- Wrapper hoặc sudo không khả dụng: không đánh dấu `ready`; báo operator xử lý.
- Xoá domain: ngừng resolve tenant ngay từ database, sau đó dọn Nginx config qua
  cleanup retry.
- Gia hạn certificate lỗi: giữ certificate hiện tại và retry ở lịch Certbot sau.

Không ghi certificate private key, token, cookie hoặc toàn bộ environment vào
log. Mỗi domain phải có cấu hình và trạng thái độc lập để lỗi một domain không
ảnh hưởng các domain khác.
