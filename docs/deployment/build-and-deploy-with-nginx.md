# Build và triển khai với GitHub Actions

Tài liệu này hướng dẫn đưa New API lên máy chủ live bằng repository
`https://github.com/MediaSoft-VN/usgate`.

**Luồng thực hiện:** build gói ứng dụng trên máy local → đẩy gói lên GitHub →
GitHub Actions gửi gói lên máy chủ và triển khai. GitHub Actions **không build**
mã nguồn. Máy chủ chạy binary New API bằng systemd, đặt Nginx phía trước để
phục vụ HTTPS; không chạy ứng dụng trong Docker.

## 1. Chuẩn bị một lần

### 1.1 Máy chủ

Máy chủ live cần:

- Ubuntu/Debian Linux có systemd.
- Một tài khoản SSH để GitHub Actions đăng nhập. Tài khoản này phải có quyền
  `sudo` không hỏi mật khẩu cho các thao tác triển khai.
- DNS của `newapi.usgate.ai` trỏ đến IP công khai của máy chủ.
- Cổng 80 và 443 mở từ Internet. Cổng SSH phải cho phép GitHub Actions kết nối.
- Cơ sở dữ liệu và Redis đã sẵn sàng (trên máy chủ hoặc dịch vụ quản lý riêng).

Workflow tự cài Nginx, Certbot và các gói hệ thống cần thiết. Nó tạo hoặc cập
nhật dịch vụ `new-api`, cấu hình Nginx và HTTPS cho domain chính. Workflow cũng
cài dịch vụ host provisioner cho domain tenant, bật kết nối socket của New API
và xác nhận dịch vụ provisioner đã chạy.

### 1.2 Cấu hình GitHub

Trong repository `MediaSoft-VN/usgate`, vào **Settings → Environments** và tạo
Environment tên `production`. Thêm các giá trị sau trong Environment này.

**Secrets** — giá trị bí mật, không đưa vào mã nguồn hoặc gửi trong chat:

| Tên | Giá trị cần điền |
| --- | --- |
| `DEPLOY_HOST` | IP hoặc hostname của máy chủ live |
| `DEPLOY_PORT` | Cổng SSH; thường là `22` |
| `DEPLOY_USER` | Tên tài khoản SSH dùng để triển khai |
| `DEPLOY_SSH_PRIVATE_KEY` | Khóa riêng SSH của tài khoản triển khai |
| `DEPLOY_KNOWN_HOSTS` | Khóa nhận diện SSH của máy chủ đã được xác minh |
| `CERTBOT_EMAIL` | Email người phụ trách chứng chỉ HTTPS |
| `NEW_API_ENV` | Toàn bộ nội dung file cấu hình `.env` của New API; giữ nguyên nhiều dòng |

**Variables** — cấu hình không bí mật:

| Tên | Giá trị cần điền |
| --- | --- |
| `PROD_DOMAIN` | `newapi.usgate.ai` |
| `APP_USER` | Tài khoản hệ điều hành chạy New API; nếu bỏ trống sẽ dùng `DEPLOY_USER` |
| `APP_DIR` | Thư mục cài ứng dụng; thường dùng `/opt/new-api` |
| `APP_PORT` | Cổng nội bộ của ứng dụng; thường dùng `3000` |
| `OLD_DOMAIN` | Tên miền cũ nếu cần xóa chứng chỉ; không có thì để trống |

Tài khoản triển khai trên máy chủ phải tồn tại và có thể dùng `sudo -n` cho các
lệnh trong `.github/scripts/deploy-native.sh`. Không cấp quyền rộng hơn cần
thiết. Nếu không rõ cách cấu hình quyền này, nhờ quản trị viên máy chủ thực hiện.

## 2. Phát hành mỗi lần có phiên bản mới

Thực hiện lần lượt các bước dưới đây trên máy local, từ thư mục repository.

### Bước 1: cập nhật mã nguồn

Đảm bảo mã nguồn muốn phát hành đã được commit và đẩy lên branch `main`. Nếu có
thay đổi mã nguồn mới, đẩy chúng lên trước. Push mã nguồn **chưa tự triển khai**;
workflow chỉ chạy khi có gói binary mới.

Nếu máy local có thay đổi chưa commit, lưu chúng an toàn trước khi chuyển
branch. Không dùng `git reset --hard` để xử lý thay đổi chưa lưu.

```bash
git switch main
git pull origin main
```

### Bước 2: build gói Linux ở local

Trên máy local cần Docker đang chạy và Docker Compose đã cài. Mở Terminal tại
thư mục gốc repository `MediaSoft-VN/usgate`, rồi chạy lệnh build này:

```bash
./scripts/package-native.sh
```

Đây là lệnh build đầy đủ của dự án; không chạy `go build` riêng. Script build cả
frontend và backend rồi đóng gói binary Linux. Khi hoàn tất, gói server được
tạo tại `deploy/new-api.gz`. Tệp
`deploy/new-api-client.tar.gz` là gói frontend riêng, không cần đưa lên để chạy
server.

### Bước 3: kiểm tra gói

```bash
gzip -t deploy/new-api.gz
sha256sum deploy/new-api.gz
```

Nếu lệnh `gzip -t` báo lỗi hoặc không có `deploy/new-api.gz`, dừng lại và build
lại. Chưa push gói lỗi lên GitHub.

### Bước 4: commit riêng gói binary

Thêm đúng một tệp vào commit phát hành:

```bash
git add deploy/new-api.gz
git diff --cached --name-only
```

Trước khi tiếp tục, kết quả lệnh thứ hai phải chỉ có:

```text
deploy/new-api.gz
```

Nếu còn tệp khác, đừng commit vội; bỏ các tệp không thuộc gói phát hành khỏi
staging rồi kiểm tra lại. Sau đó tạo commit và push:

```bash
git commit -m "chore: publish native release $(cat VERSION)"
git push origin main
```

Không commit `.env`, khóa SSH, mật khẩu hoặc thông tin xác thực.

### Bước 5: theo dõi GitHub Actions

1. Mở repository `MediaSoft-VN/usgate` trên GitHub.
2. Chọn tab **Actions**.
3. Mở workflow **Deploy native release** mới nhất.
4. Chờ trạng thái chuyển sang thành công. Nếu thất bại, mở bước màu đỏ và xem
   thông báo lỗi; không đăng tải log có thể chứa thông tin nhạy cảm.

Push gói mới lên `main` sẽ tự chạy workflow. Có thể chạy thủ công bằng
**Actions → Deploy native release → Run workflow**; commit được chọn phải có
sẵn `deploy/new-api.gz`.

### Bước 6: xác nhận dịch vụ live

Mở các địa chỉ sau:

- `https://newapi.usgate.ai/api/status` — phải trả về trạng thái thành công.
- `https://newapi.usgate.ai` — trang phải mở được và trình duyệt không báo lỗi
  chứng chỉ.

## 3. Workflow thực hiện những gì?

Khi nhận commit có thay đổi `deploy/new-api.gz`, workflow:

1. Kiểm tra gói binary có tồn tại, đọc được và khớp checksum.
2. Kết nối đến máy chủ qua SSH và chuyển gói lên.
3. Cài binary vào thư mục release, cập nhật liên kết tới phiên bản mới và khởi
   động lại `new-api` bằng systemd.
4. Kiểm tra ứng dụng trên cổng nội bộ.
5. Cài/cập nhật Nginx và cấu hình chuyển yêu cầu đến ứng dụng.
6. Dùng Certbot xin hoặc gia hạn chứng chỉ domain chính, bật HTTPS và chuyển
   HTTP sang HTTPS.
7. Cài ba script host provisioner, đối chiếu checksum, tạo dịch vụ systemd
   root-owned và socket `/run/new-api-tenant-domain.sock`.
8. Bật `TENANT_DOMAIN_PROVISIONER_ENABLED=true` cho `new-api`; xác nhận dịch vụ
   provisioner và socket đang hoạt động.
9. Kiểm tra endpoint HTTPS công khai.

Sau khi deploy thành công, khi đối tác xác minh domain, New API sẽ gửi yêu cầu
qua socket. Host provisioner tự tạo cấu hình Nginx và xin chứng chỉ riêng cho
hostname đó. Luồng này không cần chạy GitHub Actions thêm lần nữa.

Nếu health check nội bộ sau khi khởi động ứng dụng thất bại, workflow sẽ cố
khôi phục binary trước đó. Nếu lỗi xảy ra ở bước Nginx hoặc cấp chứng chỉ, xem
log Actions và xử lý lỗi trên máy chủ trước khi chạy lại.

## 4. Kiểm tra lỗi trên máy chủ

Người quản trị máy chủ có thể kiểm tra trạng thái dịch vụ và các thông báo gần
đây bằng:

```bash
sudo systemctl status new-api --no-pager
sudo systemctl status nginx --no-pager
sudo systemctl status new-api-tenant-domain-provisioner --no-pager
sudo test -S /run/new-api-tenant-domain.sock
sudo journalctl -u new-api --since '-10 minutes' --no-pager
sudo nginx -t
```

Nếu chứng chỉ không được cấp, kiểm tra DNS đã trỏ đúng IP và cổng 80 có thể truy
cập từ Internet. Không xóa thủ công dữ liệu Certbot hoặc cấu hình Nginx chính
để thử khắc phục.

## 5. Cấu hình ứng dụng và dữ liệu

`NEW_API_ENV` là nội dung cấu hình `.env` của New API được lưu dưới dạng GitHub
Secret. Workflow ghi cấu hình đó lên máy chủ với quyền hạn chế và giữ lại khi
triển khai phiên bản mới. Cấu hình phải có thông tin kết nối cơ sở dữ liệu và
các giá trị bí mật cần thiết cho môi trường của bạn.

Không đưa mật khẩu mẫu vào môi trường live. Không thay đổi `SESSION_SECRET` hoặc
`CRYPTO_SECRET` khi chỉ nâng cấp binary; thay đổi các khóa này có thể làm mất
hiệu lực phiên hoặc dữ liệu đã mã hóa. Sao lưu cơ sở dữ liệu trước khi nâng cấp
có thay đổi migration.
