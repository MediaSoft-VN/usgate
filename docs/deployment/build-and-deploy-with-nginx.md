# Build và triển khai không dùng Docker

Guide này triển khai New API dưới dạng binary trên Linux server, chạy bằng
`systemd` và dùng Nginx làm reverse proxy HTTPS. Database và Redis chạy độc
lập trên server hoặc dùng dịch vụ managed.

## 1. Điều kiện chuẩn bị

Server cần có:

- Linux amd64 hoặc arm64.
- Go theo phiên bản trong `go.mod`.
- Bun theo phiên bản được dự án hỗ trợ.
- Nginx và systemd.
- PostgreSQL, MySQL hoặc SQLite.
- Redis nếu cần cache dùng chung.
- DNS trỏ domain về server.

Không dùng các giá trị mẫu như mật khẩu database `123456`. Không ghi secret
vào Git hoặc gửi secret qua chat.

## 2. Lấy source và tạo thư mục chạy

Nên build từ tag/release cụ thể thay vì branch đang phát triển:

```bash
sudo mkdir -p /opt/new-api/releases /opt/new-api/data /opt/new-api/logs
sudo chown -R "$USER":"$USER" /opt/new-api

git clone https://github.com/QuantumNous/new-api.git /opt/new-api/source
cd /opt/new-api/source
git checkout <version-or-commit>
```

Kiểm tra tag hoặc commit trước khi build:

```bash
git rev-parse HEAD
git status --short
```

Working tree phải sạch và đúng phiên bản muốn phát hành.

## 3. Build frontend và backend

Frontend được embed vào binary Go, vì vậy phải build frontend trước:

```bash
cd /opt/new-api/source/web
bun install --frozen-lockfile
bun run typecheck
bun run lint
bun run build

cd /opt/new-api/source
go mod download
go build -trimpath -ldflags "-s -w -X github.com/QuantumNous/new-api/common.Version=$(cat VERSION)" -o /opt/new-api/releases/new-api-<version> .
chmod 0755 /opt/new-api/releases/new-api-<version>
```

Kiểm tra binary:

```bash
/opt/new-api/releases/new-api-<version> --help
```

Sau khi kiểm tra, tạo symlink phiên bản đang chạy:

```bash
ln -sfn /opt/new-api/releases/new-api-<version> /opt/new-api/new-api
```

### Chuẩn bị cho GitHub Actions

Workflow `.github/workflows/deploy.yml` không build lại source. Nó yêu cầu
binary đã build sẵn được commit tại `deploy/new-api`:

```bash
mkdir -p deploy
cp /opt/new-api/releases/new-api-<version> deploy/new-api
chmod 0755 deploy/new-api
git add -f deploy/new-api
git commit -m "chore: publish native release <version>"
git push origin main
```

Không commit file môi trường, private key hoặc credential vào repository.
Workflow upload binary lên `/tmp`, kiểm tra SHA-256, cài vào
`/opt/new-api/releases`, đổi symlink, restart systemd và gọi health check. SSH
user cần quyền `sudo -n` cho `install`, `ln`, `systemctl` và `rm` trên máy chủ.

Tạo các GitHub Environment secrets trong environment `production`:

- `DEPLOY_HOST`: hostname hoặc IP máy chủ.
- `DEPLOY_PORT`: cổng SSH, thường là `22`.
- `DEPLOY_USER`: user SSH dùng để deploy.
- `DEPLOY_SSH_PRIVATE_KEY`: private key tương ứng với public key trên máy chủ.
- `DEPLOY_KNOWN_HOSTS`: output đã xác minh của `ssh-keyscan`, không lấy động
  trong workflow.

Tạo thêm Environment variables:

- `APP_USER`: user chạy `new-api`; nếu bỏ trống sẽ dùng `DEPLOY_USER`.
- `APP_DIR`: mặc định `/opt/new-api`.
- `APP_PORT`: port ưu tiên; nếu bỏ trống, workflow đọc port trước đó từ
  `/opt/new-api/.app-port`, mặc định bắt đầu từ `3000`, rồi tự chọn port trống
  trong phạm vi 20 port tiếp theo nếu port đang bị dịch vụ khác sử dụng.
- `PROD_DOMAIN`: domain live, ví dụ `api.example.com`.
- `OLD_DOMAIN`: domain cũ cần xóa certificate, có thể bỏ trống.
- `CERTBOT_EMAIL`: email nhận thông báo Let's Encrypt (Environment secret).

Workflow hiện tự deploy khi có push vào branch `main` có binary mới. Có thể
chạy thủ công bằng `workflow_dispatch`; production environment nên bật
required reviewers nếu cần phê duyệt trước deploy.

Mỗi lần deploy workflow sẽ cập nhật `/etc/systemd/system/new-api.service`
theo binary hiện tại, chạy `daemon-reload`, `enable`, `restart` và kiểm tra
`/api/status`. Nó cũng ghi lại Nginx server block cho domain mới, cấp hoặc
renew Let's Encrypt bằng Certbot, bật redirect HTTPS và xóa certificate của
`OLD_DOMAIN` nếu có.

Nếu `new-api` đang chạy, workflow đọc port từ `ExecStart` của service hiện tại
và tiếp tục dùng đúng port đó; service không bị dừng trong bước dò port. Nếu
service chưa chạy, workflow dùng `APP_PORT`, `/opt/new-api/.app-port` hoặc bắt
đầu từ `3000`, rồi tự chọn port trống trong phạm vi 20 port. Port được chọn sẽ được lưu tại
`/opt/new-api/.app-port` và được dùng đồng thời cho systemd, Nginx và health
check ở các lần deploy sau.

## 4. Cấu hình môi trường

Tạo file chỉ root được đọc:

```bash
sudo install -o root -g root -m 0600 /dev/null /etc/new-api.env
sudoedit /etc/new-api.env
```

Ví dụ tối thiểu:

```dotenv
PORT=3000
SQL_DSN=postgres://new_api:<database-password>@127.0.0.1:5432/new_api?sslmode=require
REDIS_CONN_STRING=redis://:<redis-password>@127.0.0.1:6379/0
SESSION_SECRET=<random-32-byte-secret>
CRYPTO_SECRET=<different-random-32-byte-secret>
SESSION_COOKIE_SECURE=true
SESSION_COOKIE_TRUSTED_URL=https://api.example.com
TRUSTED_PROXIES=127.0.0.1
DEFAULT_LANGUAGE=en
```

Tạo secret bằng:

```bash
openssl rand -hex 32
```

`SESSION_SECRET` và `CRYPTO_SECRET` phải giữ ổn định khi restart hoặc
upgrade. Với nhiều node, các node phải dùng cùng authentication secrets và
database chính.

Nếu dùng SQLite, thay `SQL_DSN` bằng `SQLITE_PATH=/opt/new-api/data/new-api.db`.
Không bật `DEBUG`, `ENABLE_PPROF` hoặc `TLS_INSECURE_SKIP_VERIFY` trên live.

## 5. Chạy bằng systemd

Tạo `/etc/systemd/system/new-api.service`:

```ini
[Unit]
Description=New API
Wants=network-online.target
After=network-online.target

[Service]
Type=simple
User=new-api
Group=new-api
WorkingDirectory=/opt/new-api
EnvironmentFile=/etc/new-api.env
ExecStart=/opt/new-api/new-api --port 3000 --log-dir /opt/new-api/logs
Restart=on-failure
RestartSec=5
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=true
ReadWritePaths=/opt/new-api/data /opt/new-api/logs

[Install]
WantedBy=multi-user.target
```

Tạo user và cấp quyền:

```bash
sudo useradd --system --home /opt/new-api --shell /usr/sbin/nologin new-api
sudo chown -R new-api:new-api /opt/new-api
sudo systemctl daemon-reload
sudo systemctl enable --now new-api
sudo systemctl status new-api
sudo journalctl -u new-api -f
```

Kiểm tra backend trước khi cấu hình Nginx:

```bash
curl --fail-with-body http://127.0.0.1:3000/api/status
```

## 6. Cấu hình Nginx

Cài Nginx rồi tạo `/etc/nginx/sites-available/new-api`:

```nginx
map $http_upgrade $connection_upgrade {
    default upgrade;
    ''      close;
}

server {
    listen 80;
    listen [::]:80;
    server_name api.example.com;

    location / {
        proxy_pass http://127.0.0.1:3000;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection $connection_upgrade;

        proxy_buffering off;
        proxy_read_timeout 1800s;
        proxy_send_timeout 1800s;
    }
}
```

Kích hoạt và kiểm tra cấu hình:

```bash
sudo ln -s /etc/nginx/sites-available/new-api /etc/nginx/sites-enabled/new-api
sudo nginx -t
sudo systemctl reload nginx
```

Sau đó cấp TLS bằng Certbot hoặc certificate manager của hệ thống. Chỉ bật
`SESSION_COOKIE_SECURE=true` sau khi domain đã phục vụ HTTPS ổn định.

Nginx phải giữ `proxy_buffering off` và timeout đủ dài cho streaming. Nếu
Nginx đứng sau một load balancer khác, đặt `TRUSTED_PROXIES` thành IP/CIDR
thực tế của proxy gần nhất, không dùng mạng tin cậy quá rộng.

## 7. Kiểm tra sau triển khai

```bash
curl --fail-with-body https://api.example.com/api/status
systemctl is-active --quiet new-api
sudo journalctl -u new-api --since "10 minutes ago" --no-pager
```

Kiểm tra thủ công thêm:

1. Mở console qua HTTPS và đăng nhập admin.
2. Tạo hoặc kiểm tra một API token.
3. Gửi một request relay không streaming.
4. Gửi một request streaming và xác nhận dữ liệu đến liên tục.
5. Kiểm tra log không chứa secret, token hoặc credential.

## 8. Upgrade và rollback

Trước mỗi upgrade:

1. Backup database và thư mục `/opt/new-api/data`.
2. Đọc release notes và xác nhận migration tương thích.
3. Build binary vào thư mục `releases` với version mới.
4. Đổi symlink `/opt/new-api/new-api`.
5. Restart service và kiểm tra `/api/status`.

```bash
sudo systemctl restart new-api
sudo systemctl status new-api
```

Rollback dùng binary release trước, không dùng tag `latest`:

```bash
ln -sfn /opt/new-api/releases/new-api-<previous-version> /opt/new-api/new-api
sudo systemctl restart new-api
curl --fail-with-body https://api.example.com/api/status
```

Không rollback database nếu chưa có backup và chưa xác nhận migration có thể
đảo ngược. Với migration không đảo ngược, cần khôi phục backup trên database
được cô lập trước khi quyết định rollback production.
