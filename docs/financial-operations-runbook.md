# Hướng dẫn vận hành nền tảng New API

Tài liệu này hướng dẫn triển khai, kiểm tra và xử lý sự cố cho các chức năng
không gian đối tác, thành viên và tên miền, ví khách hàng, sổ cái, ví nhà cung
cấp, nạp tiền, cấp hạn mức, ngưỡng tín dụng, báo cáo dòng tiền và xuất CSV.

Tài liệu dành cho quản trị viên và người vận hành. Mỗi mục được trình bày theo
thứ tự: việc cần làm, cách kiểm tra kết quả và cách xử lý nếu không đúng. Các
tên trong dấu `...` như `SQL_DSN`, `/api/status` và tên nút trên màn hình được
giữ nguyên để có thể đối chiếu trực tiếp với cấu hình hoặc giao diện.

### Từ ngữ dùng trong tài liệu

- **Tenant/không gian đối tác:** khu vực riêng của một đối tác trong hệ thống.
- **Quota/hạn mức sử dụng:** lượng dịch vụ người dùng còn được phép sử dụng;
  đây không phải tiền trong ví.
- **Ví tài chính:** nơi ghi nhận số tiền theo từng chủ sở hữu và loại tiền.
- **Sổ cái:** lịch sử ghi tăng/giảm tiền của ví; không sửa hoặc xóa lịch sử này.
- **Mã tham chiếu:** mã để tìm và đối chiếu một giao dịch.
- **Thử lại an toàn:** gửi lại cùng yêu cầu và cùng mã chống trùng để hệ thống
  không ghi nhận giao dịch lần thứ hai.
- **Host provisioner:** dịch vụ chạy trên máy chủ, thay ứng dụng tạo/xóa cấu hình
  Nginx và chứng chỉ cho tên miền.

> **Phạm vi hiện tại:** đây là phiên bản tối thiểu của chức năng tài chính. Hóa đơn,
> quy trình phê duyệt và đối soát tự động nâng cao chưa được triển khai. Quota
> khách hàng là hạn mức sử dụng dịch vụ; ví tài chính và sổ cái là các số liệu
> nội bộ riêng, không thay thế quota.

## Quy tắc an toàn bắt buộc

1. Chỉ quản trị viên được thực hiện nạp tiền, phân bổ, cấu hình ngưỡng và đối soát.
2. Không ghi bí mật, mã truy cập, DSN, cookie phiên hoặc dữ liệu xác thực vào
   nhật ký, phiếu hỗ trợ, ảnh chụp màn hình hay kho mã nguồn.
3. Không sửa hoặc xóa giao dịch, dòng sổ cái, khoản nạp hoặc khoản phân bổ đã
   hoàn tất. Mọi điều chỉnh phải được ghi nhận bằng giao dịch điều chỉnh/đảo
   ngược mới.
4. Luôn lưu mã tham chiếu và tệp CSV xuất ra của từng kỳ đối soát.
5. Không hạ ngưỡng chỉ để che giấu sai lệch số dư nhà cung cấp.
6. Không dùng số dư ví tài chính để thay thế số dư quota sử dụng dịch vụ.

## 1. Chuẩn bị trước khi triển khai

- Triển khai phiên bản đã gắn thẻ hoặc commit đã được duyệt; không triển khai
  thay đổi chưa được rà soát.
- Xác nhận `SQL_DSN` trỏ đến đúng cơ sở dữ liệu chính và đã tạo bản sao lưu có
  thể khôi phục.
- Nạp `REDIS_CONN_STRING`, `SESSION_SECRET` và `CRYPTO_SECRET` từ kho bí mật.
  Không ghi giá trị của chúng vào log, phiếu hỗ trợ hoặc kho mã nguồn.
- Khi dùng HTTPS, đặt `SESSION_COOKIE_SECURE=true` và khai báo chính xác các
  nguồn truy cập công khai trong `SESSION_COOKIE_TRUSTED_URL`.
- Cấu hình rõ `TRUSTED_PROXIES` hoặc đặt `none`; không bỏ qua cảnh báo tương
  thích mặc định trên môi trường thật.

## 2. Triển khai và kiểm tra migration

Với Docker Compose:

```bash
docker compose up -d
docker compose ps
curl -fsS http://127.0.0.1:3000/api/status
```

Với dịch vụ systemd, khởi động lại dịch vụ và kiểm tra endpoint tình trạng:

```bash
sudo systemctl restart new-api
sudo systemctl is-active new-api
curl -fsS https://<public-host>/api/status
```

Phản hồi phải có `"success":true`. Trước khi nhận lưu lượng, kiểm tra log khởi
động để phát hiện lỗi migration:

```bash
docker logs --since=10m <container> | grep -Ei 'fatal|panic|migration|error'
sudo journalctl -u new-api --since '-10 minutes' --no-pager
```

Migration tài chính phải chạy lặp lại an toàn. Khởi động lại lần thứ hai rồi
kiểm tra sức khỏe thêm lần nữa. Không xóa volume để xử lý lỗi migration.

## 3. Kiểm tra tài chính hằng ngày

1. Mở trang quản trị **Provider Wallet** (Ví nhà cung cấp).
2. Kiểm tra tổng số dư ví và số dư của từng kênh.
3. Rà soát các kênh có trạng thái **Below threshold** (Dưới ngưỡng) hoặc
   **Auto disabled** (Tự động tắt).
4. Đối chiếu các khoản nạp và phân bổ gần đây trong **Cash flow history**
   (Lịch sử dòng tiền).
5. Lọc theo ngày và chiều thu/chi; kiểm tra tổng báo cáo khớp các dòng hiển thị
   theo từng loại tiền.
6. Khi cần đối soát, xuất báo cáo đã lọc và lưu cùng hồ sơ vận hành.

Ngưỡng tín dụng nhà cung cấp là mức bảo vệ dựa trên số dư do từng kênh báo cáo,
không phải quota khách hàng. Kênh sẽ tự tắt khi số dư thấp hơn ngưỡng và chỉ
được tự bật lại khi số dư được khôi phục. Chính sách này không được ghi đè thao
tác tắt kênh thủ công.

## 4. Quy trình đối soát

Thực hiện riêng cho từng loại tiền và kỳ báo cáo:

1. Xuất CSV dòng tiền đã lọc từ trang quản trị.
2. Đối chiếu tiền vào với chứng từ thanh toán của nhà cung cấp.
3. Đối chiếu tiền ra với hồ sơ cấp tiền cho khách hàng hoặc tenant.
4. Kiểm tra công thức:

   ```text
   net = incoming - outgoing
   ```

5. Trước khi điều chỉnh, kiểm tra mã tham chiếu trùng, thiếu bên giao dịch, loại
   tiền bất thường hoặc số dư khả dụng âm.

Các dòng sổ cái và giao dịch tài chính đã hoàn tất chỉ được ghi nối tiếp; không
được sửa hoặc xóa trực tiếp. Sai lệch phải được xử lý bằng giao dịch điều chỉnh
hoặc đảo ngược mới, có thể kiểm toán và được người có thẩm quyền phê duyệt.

## 5. Xử lý khi tín dụng nhà cung cấp thấp

Khi một kênh bị hệ thống tự động tắt:

1. Kiểm tra số dư trực tiếp trên trang quản lý của nhà cung cấp.
2. Xác nhận ngưỡng đã cấu hình trong trang **Provider Wallet**.
3. Nạp tiền vào tài khoản nhà cung cấp theo quy trình thanh toán được duyệt.
4. Làm mới hoặc đồng bộ số dư của kênh.
5. Xác nhận kênh chỉ được bật lại sau khi số dư vượt ngưỡng.

Không hạ ngưỡng chỉ để che giấu vấn đề tín dụng với nhà cung cấp. Nếu kênh bị
tắt thủ công, hãy xử lý riêng thao tác của người vận hành.

## 6. Sự cố: lỗi 5xx, hết thời gian chờ hoặc khởi động lặp

1. Kiểm tra trạng thái container hoặc dịch vụ systemd.
2. Tìm lỗi nghiêm trọng đầu tiên khi khởi động, không chỉ xem thông báo khởi
   động lại mới nhất.
3. Kiểm tra kết nối cơ sở dữ liệu và dung lượng đĩa còn trống.
4. Nếu lỗi liên quan migration, dừng việc khởi động lặp và tạo ảnh chụp cơ sở
   dữ liệu trước khi khắc phục.
5. Chỉ khôi phục phiên bản hoạt động tốt gần nhất sau khi đã lưu log và bản sao
   cơ sở dữ liệu.
6. Trước khi mở lại lưu lượng, kiểm tra `/api/status` và xem báo cáo tài chính ở
   chế độ chỉ đọc.

Ví dụ lệnh kiểm tra Docker:

```bash
docker compose ps
docker logs --tail=200 <container>
docker inspect --format '{{.State.Status}}' <container>
```

Không để lộ DSN cơ sở dữ liệu, cookie phiên, khóa API hoặc lỗi xác thực thô
trong thông tin sự cố.

## 7. Sao lưu và quay lui

- Sao lưu cơ sở dữ liệu chính trước mỗi bản phát hành có thay đổi model hoặc
  migration.
- Lưu lại binary hoặc image của phiên bản trước cùng mã commit chính xác.
- Nếu schema vẫn tương thích, quay lui mã ứng dụng trước.
- Chỉ khôi phục bản sao cơ sở dữ liệu khi đã có quyết định khôi phục được duyệt;
  thao tác này có thể làm mất các giao dịch tài chính hợp lệ phát sinh sau thời
  điểm sao lưu.
- Sau khi quay lui hoặc khôi phục, kiểm tra migration chạy lặp an toàn, số dư ví,
  cân bằng sổ cái, tổng dòng tiền nhà cung cấp và `/api/status`.

## 8. Danh sách kiểm tra trước khi xác nhận phát hành

- [ ] Đã sao lưu và ghi nhận điểm có thể khôi phục.
- [ ] Ứng dụng khởi động không có lỗi migration.
- [ ] Endpoint kiểm tra sức khỏe trả về thành công sau hai lần khởi động lại.
- [ ] Tổng số dư ví nhà cung cấp tải được cho mọi loại tiền đã cấu hình.
- [ ] Ngưỡng tín dụng và trạng thái tự tắt của kênh chính xác.
- [ ] Tổng báo cáo dòng tiền đã lọc khớp CSV xuất ra.
- [ ] Log và bằng chứng phát hành không chứa bí mật hoặc lỗi nhạy cảm thô.
- [ ] Đã lưu kết quả kiểm tra SQLite, MySQL và PostgreSQL trước khi công bố khả
  năng tương thích cơ sở dữ liệu trên môi trường thật.

## 9. Vận hành không gian đối tác (Partner Workspace)

1. Đăng nhập bằng tài khoản có quyền quản lý tenant (không gian đối tác) và mở
   `/partner-workspace`.
2. Chọn không gian đối tác cần thao tác.
3. Dùng các màn hình tương ứng:
   - `/partner-workspace/overview`: số liệu tổng quan.
   - `/partner-workspace/users`: quản lý thành viên và vai trò.
   - `/partner-workspace/domains`: thêm, xác minh hoặc xóa tên miền.
   - `/partner-workspace/wallet`: số dư và giao dịch của ví không gian đối tác.
   - `/partner-workspace/transactions`: lịch sử giao dịch.
4. Khi thêm thành viên, chọn đúng người dùng và vai trò; sau khi lưu, xác nhận
   người dùng chỉ xem được dữ liệu thuộc không gian đối tác đó.
5. Khi thêm tên miền, nhập hostname, lưu, tạo bản ghi DNS theo giá trị xác minh,
   chờ DNS cập nhật rồi chọn **Verify** (Xác minh).
6. Trước khi xóa thành viên hoặc tên miền, kiểm tra các liên kết đang sử dụng.

## 10. Ví khách hàng và sổ cái

### 10.1 Phân biệt các loại số dư

- **Quota khách hàng:** hạn mức sử dụng dịch vụ của người dùng cho relay và tính
  phí hiện hành.
- **Ví tài chính:** số dư tiền nội bộ gắn với chủ sở hữu và loại tiền.
- **Sổ cái:** các dòng ghi nợ/ghi có bất biến dùng để kiểm toán ví tài chính.

Không dùng loại số dư này thay thế loại số dư khác. Mọi thao tác với ví cần có
chủ sở hữu, loại tiền, số tiền theo đơn vị nhỏ nhất, mã tham chiếu và khóa chống
ghi nhận trùng (`idempotency key`).

### 10.2 Thực hiện thao tác với ví

1. Xác định chủ sở hữu, loại tiền và số tiền theo đơn vị nhỏ nhất.
2. Tạo mã tham chiếu và khóa chống ghi nhận trùng duy nhất.
3. Kiểm tra ví đang ở trạng thái hoạt động.
4. Với thao tác giữ tiền (reserve), xác nhận số dư khả dụng đủ.
5. Gọi service/model của ví; không cập nhật trực tiếp cột số dư trong cơ sở dữ
   liệu.
6. Xác nhận giao dịch hoàn tất và tổng ghi nợ/ghi có trong sổ cái cân bằng.
7. Khi thử lại với cùng khóa chống trùng, xác nhận hệ thống trả kết quả cũ và
   không tạo thêm lần ghi nợ/ghi có.
8. Hoàn tiền hoặc điều chỉnh bằng thao tác giải phóng/hoàn tiền/điều chỉnh mới;
   không sửa giao dịch cũ.

### 10.3 Kiểm tra sổ cái

1. Tìm giao dịch theo số giao dịch hoặc mã tham chiếu bên ngoài.
2. Kiểm tra trạng thái, loại tiền và số tiền theo đơn vị nhỏ nhất.
3. Đối chiếu tổng ghi nợ với tổng ghi có.
4. Nếu không cân bằng, dừng thao tác tài chính và mở sự cố; không sửa trực tiếp
   dữ liệu trong bảng.

## 11. Nạp tiền và ví nhà cung cấp

### 11.1 Ghi nhận khoản nạp

1. Mở `/provider-funding` (Nạp tiền nhà cung cấp).
2. Chọn kênh và loại tiền.
3. Chọn **Create funding** (Tạo khoản nạp).
4. Nhập số tiền theo đơn vị nhỏ nhất, mã tham chiếu, mã giao dịch của nhà cung
   cấp và ghi chú nếu cần.
5. Kiểm tra loại tiền và số tiền, sau đó gửi một lần.
6. Xác nhận khoản nạp xuất hiện trong dòng tiền vào.
7. Nếu yêu cầu hết thời gian chờ, hãy tìm theo mã tham chiếu trước khi thử lại;
   không tạo mã mới chỉ vì chưa nhận được phản hồi.

### 11.2 Phân bổ tiền

1. Mở `/provider-wallet` (Ví nhà cung cấp).
2. Chọn ví nhà cung cấp và loại tiền.
3. Chọn người dùng hoặc tenant nhận khoản phân bổ.
4. Nhập số tiền, quota được cấp, tỷ giá và mã tham chiếu.
5. Xác nhận nguồn tiền khả dụng của nhà cung cấp đủ rồi mới gửi.
6. Kiểm tra nguồn tiền khả dụng giảm đúng số tiền phân bổ.
7. Kiểm tra người dùng/tenant nhận đúng quota và dòng tiền ra có khoản phân bổ.

Nếu nguồn tiền không đủ hoặc chạm giới hạn quota, không thử lại liên tục; trước
hết kiểm tra giao dịch gần nhất và số dư thực tế.

### 11.3 Đồng bộ số dư

1. Mở `/provider-funding` và kiểm tra trạng thái đồng bộ.
2. Chỉ chọn **Start sync** (Bắt đầu đồng bộ) khi không có tác vụ nào đang chạy.
3. Chờ tác vụ hoàn tất.
4. Đối chiếu kết quả với trang quản lý của nhà cung cấp.
5. Nếu đồng bộ lỗi, giữ nguyên dữ liệu, lưu thông tin lỗi không chứa bí mật và
   xử lý theo mục Sự cố.

## 12. Ngưỡng tín dụng nhà cung cấp

1. Mở trang quản trị kênh/nhà cung cấp.
2. Chọn kênh và nhập ngưỡng theo loại tiền/đơn vị cấu hình.
3. Lưu, sau đó xác nhận ngưỡng đã được cập nhật.
4. Khi số dư thấp hơn ngưỡng, kênh chuyển sang trạng thái tự động tắt.
5. Nạp tiền vào tài khoản nhà cung cấp theo quy trình của nhà cung cấp.
6. Làm mới hoặc đồng bộ số dư.
7. Xác nhận kênh chỉ tự bật lại khi số dư vượt ngưỡng.
8. Nếu kênh đã bị tắt thủ công, chính sách ngưỡng không được ghi đè trạng thái
   đó.

Khi kênh tự tắt, hãy kiểm tra trang nhà cung cấp, số dư, thời điểm đồng bộ và
ngưỡng trước khi đổi cấu hình. Không hạ ngưỡng để che giấu lỗi tín dụng.

## 13. Báo cáo dòng tiền và xuất CSV

1. Mở `/provider-funding`.
2. Chọn loại tiền, khoảng thời gian `From`/`To` (Từ/Đến) và chiều dòng tiền.
3. Chọn **Incoming** (Tiền vào) cho khoản nạp vào quỹ nhà cung cấp;
   **Outgoing** (Tiền ra) cho khoản phân bổ cho người dùng/tenant.
4. Kiểm tra công thức tính:

   ```text
   net = incoming - outgoing
   ```

5. Đối chiếu phần tổng hợp với các dòng đang hiển thị.
6. Chọn **Export CSV** (Xuất CSV) và lưu tệp theo kỳ, ví dụ
   `provider-cash-flow-YYYY-MM.csv`.
7. Giữ nguyên tệp gốc; nếu cần ghi chú, hãy tạo một bản sao.

Các API quản trị tương ứng:

```text
GET  /api/financial/provider-funding
GET  /api/financial/provider-funding/export
POST /api/financial/provider-funding
POST /api/financial/provider-funding/allocate
GET  /api/financial/provider-funding/sync
POST /api/financial/provider-funding/sync
GET  /api/financial/wallets
```

## 14. Quy trình đối soát định kỳ

1. Tạo điểm khôi phục trước khi bắt đầu.
2. Xuất CSV riêng theo từng loại tiền và kỳ đối soát.
3. Đối chiếu tiền vào với chứng từ thanh toán của nhà cung cấp.
4. Đối chiếu tiền ra với hồ sơ phân bổ cho người dùng/tenant.
5. Kiểm tra mã tham chiếu trùng, loại tiền lạ, số tiền âm và số dư khả dụng âm.
6. Đối chiếu sổ cái, giao dịch ví và bản ghi dòng tiền.
7. Ghi lại chênh lệch, mã tham chiếu và người xử lý.
8. Không tự tạo giao dịch điều chỉnh khi chưa được phê duyệt nội bộ.

## 15. Xử lý sự cố và quay lui

### 15.1 Lỗi 5xx, hết thời gian chờ hoặc khởi động lặp

1. Kiểm tra trạng thái container/dịch vụ systemd.
2. Đọc lỗi đầu tiên trong log khởi động.
3. Kiểm tra cơ sở dữ liệu, Redis, dung lượng đĩa và kết nối mạng.
4. Nếu lỗi migration, dừng khởi động lặp và tạo bản sao cơ sở dữ liệu.
5. Lưu log, mã commit/image hiện tại; chỉ quay lui sau khi kiểm tra tương thích
   schema.
6. Trước khi mở lại lưu lượng, xác nhận `/api/status`, số dư ví và báo cáo ở chế
   độ chỉ đọc.

### 15.2 Nghi ngờ khoản nạp bị ghi nhận trùng

1. Tìm theo mã tham chiếu/khóa chống ghi nhận trùng.
2. Kiểm tra giao dịch tài chính và bản ghi dòng tiền.
3. Nếu giao dịch đã hoàn tất, không gửi lại.
4. Nếu đang chờ, xử lý theo chính sách thử lại.
5. Chỉ tạo giao dịch điều chỉnh khi có bằng chứng về giao dịch thực tế.

### 15.3 Số dư hoặc báo cáo không khớp

1. Xác nhận loại tiền, khoảng thời gian và bộ lọc chiều dòng tiền.
2. Xuất lại CSV.
3. Tìm mã tham chiếu bị thiếu hoặc trùng.
4. Đối chiếu sổ cái với ví và dòng tiền.
5. Không sửa trực tiếp dữ liệu trên môi trường thật.

## 16. Vận hành tên miền riêng của Partner Workspace

### 16.1 Hiểu quy trình trước khi bắt đầu

Khi triển khai New API bằng GitHub Actions (systemd), workflow cài luôn dịch vụ
hỗ trợ tên miền lên máy chủ và bật kết nối giữa dịch vụ đó với New API. Không
cần SSH vào máy chủ để cài thủ công. Sau khi deployment thành công, đối tác có
thể thêm tên miền trong giao diện và cập nhật DNS. Khi bấm **Verify** (Xác minh),
New API kiểm tra DNS rồi tự yêu cầu dịch vụ trên máy chủ tạo cấu hình Nginx và
chứng chỉ HTTPS riêng.

Trước khi cài, hãy xác nhận với người quản trị máy chủ rằng:

- New API và Nginx chạy trên cùng máy chủ; Nginx kết nối đến New API qua cổng
  ứng dụng (mặc định `3000`).
- Tên miền đối tác đã trỏ về địa chỉ IP công khai của máy chủ.
- Người dùng Internet có thể kết nối đến máy chủ qua cổng 80 và 443.
- GitHub Actions đã deploy thành công và cài dịch vụ
  `new-api-tenant-domain-provisioner`.

Ứng dụng không được cấp quyền sửa cấu hình máy chủ trực tiếp. Thay vào đó, ứng
dụng gửi yêu cầu qua một kênh liên lạc nội bộ trên cùng máy chủ. Kênh này không
phải cổng mở ra Internet và chỉ dành cho tài khoản chạy New API.

### 16.2 Kiểm tra dịch vụ sau khi GitHub Actions deploy

Sau khi workflow deploy thành công, quản trị viên có thể kiểm tra dịch vụ và
socket trên máy chủ:

```bash
sudo systemctl status new-api-tenant-domain-provisioner --no-pager
sudo test -S /run/new-api-tenant-domain.sock
sudo nginx -t
```

Xác nhận ứng dụng được cấu hình để dùng socket:

```bash
sudo systemctl show new-api -p Environment
```

Nếu một lệnh báo lỗi hoặc socket chưa tồn tại, đừng tiếp tục thêm tên miền. Xem
log `new-api-tenant-domain-provisioner` và chạy lại GitHub Actions sau khi đã
khắc phục nguyên nhân.

### 16.3 Thêm và xác minh tên miền

1. Đăng nhập bằng tài khoản được cấp quyền quản lý domain của workspace và mở
   `/partner-workspace/domains`.
2. Chọn đúng workspace, bấm **Add domain** (Thêm tên miền), rồi nhập hostname,
   ví dụ `app.example.com`. Nhập hostname, không nhập URL, giao thức, cổng hay
   đường dẫn.
3. Lưu lại đúng loại bản ghi DNS, tên/host và giá trị mà giao diện hiển thị.
4. Tạo bản ghi tại nhà cung cấp DNS của đối tác. Không tự thay đổi giá trị hệ
   thống cung cấp.
5. Chờ DNS được cập nhật trên các máy chủ phân giải. Thời gian phụ thuộc TTL và
   nhà cung cấp DNS.
6. Quay lại giao diện và chọn **Verify** (Xác minh).
7. Khi DNS hợp lệ, hệ thống lưu trạng thái `verified` rồi tự động tạo vhost riêng,
   xin chứng chỉ Certbot và tải lại Nginx.
8. Mở `https://<hostname>/api/status` hoặc truy cập giao diện để kiểm tra HTTPS,
   chứng chỉ hợp lệ và nội dung được chuyển đến đúng tenant.

Tên miền gốc `usgate.ai` và các tên miền con của nó chỉ dành cho quản trị viên
hệ thống. Người dùng thông thường không thể thêm hoặc đổi sang các tên miền này.

### 16.4 Đổi hoặc xóa tên miền

- **Đổi tên miền:** ứng dụng dọn vhost/chứng chỉ của hostname cũ trước khi lưu
  hostname mới ở trạng thái `pending`. Sau đó đối tác phải tạo DNS và xác minh
  lại tên miền mới.
- **Xóa tên miền:** ứng dụng yêu cầu host provisioner gỡ đúng vhost, liên kết
  Nginx và chứng chỉ của hostname đó trước khi xóa bản ghi trong cơ sở dữ liệu.
- Nếu dọn cấu hình host thất bại, thao tác đổi/xóa không tiếp tục nhằm tránh
  mất dấu cấu hình cũ. Khắc phục host provisioner rồi thử lại thao tác.
- Mỗi hostname có tệp riêng
  `/etc/nginx/sites-available/partner-<hostname>.conf`, liên kết tương ứng trong
  `sites-enabled` và chứng chỉ riêng `partner-<hostname>`. Luồng này không sửa
  vhost chính `new-api.conf` hoặc cấu hình các đối tác khác.

### 16.5 Kiểm tra DNS và xử lý lỗi xác minh

- Trạng thái vẫn là `pending`: kiểm tra hostname, loại bản ghi, tên/host và giá
  trị DNS; chờ cập nhật rồi xác minh lại.
- Bản ghi CNAME không khớp: kiểm tra có nhập đúng đích mà giao diện cung cấp,
  cấu hình proxy của nhà cung cấp DNS và các bản ghi CNAME xung đột hay không.
- Bản ghi A không khớp: kiểm tra địa chỉ IP công khai hiện tại của gateway và
  bản ghi A tại DNS.
- Hostname không hợp lệ: không dùng địa chỉ IP, wildcard, `localhost`, cổng,
  đường dẫn hoặc ký tự đặc biệt.
- Tên miền đã được workspace khác sử dụng: không tự xóa bản ghi hoặc sửa dữ
  liệu của workspace kia; liên hệ quản trị viên để kiểm tra quyền sở hữu.
- Không gửi được yêu cầu host: xác nhận dịch vụ systemd đang hoạt động, socket
  tồn tại, UID của ứng dụng khớp UID được phép và cấu hình bật provisioner đúng
  cho kiểu triển khai đang dùng.

### 16.6 Kiểm tra log và khắc phục tự động cấu hình

Xem trạng thái và nhật ký của dịch vụ cấu hình tên miền:

```bash
sudo systemctl status new-api-tenant-domain-provisioner --no-pager
sudo journalctl -u new-api-tenant-domain-provisioner --since '-15 minutes' --no-pager
sudo journalctl -u new-api --since '-15 minutes' --no-pager
```

Các bước xử lý theo triệu chứng:

- **Certbot không cấp được chứng chỉ:** xác nhận DNS đã trỏ đúng IP, cổng 80
  truy cập được từ Internet, Nginx đang chạy và email vận hành hợp lệ. Không
  xóa thủ công thư mục Certbot; sửa nguyên nhân rồi bấm **Verify** để thử lại.
- **`nginx -t` thất bại:** không ép tải lại Nginx. Kiểm tra cấu hình lỗi và chỉ
  sửa tệp của hostname liên quan; không ghi đè cấu hình chính hoặc domain khác.
- **Tải lại Nginx thất bại:** kiểm tra trạng thái Nginx và cấu hình; thao tác host
  sẽ cố khôi phục cấu hình trước đó. Xác nhận lại bằng `sudo nginx -t`.
- **Ứng dụng báo cấu hình host thất bại:** kiểm tra socket và dịch vụ provisioner;
  sau khi khắc phục, thử lại **Verify**, đổi hoặc xóa theo đúng thao tác đang chờ.
- **Đã verified nhưng chưa truy cập được:** trạng thái DNS verified không đồng
  nghĩa provisioning đã thành công. Bấm **Verify** lần nữa để thử cấp cấu hình;
  kiểm tra DNS, cổng 80/443, Nginx và endpoint `/api/status`.
- **Gia hạn chứng chỉ lỗi:** chứng chỉ hiện tại được giữ lại; kiểm tra Certbot,
  hook tải lại Nginx và log dịch vụ, sau đó để Certbot thử lại theo lịch.

Không ghi private key chứng chỉ, token, cookie hoặc toàn bộ biến môi trường vào
log. Không chạy lệnh root tùy ý từ ứng dụng. Mỗi hostname phải được xử lý độc
lập để lỗi của một domain không ảnh hưởng domain chính hoặc các đối tác khác.
