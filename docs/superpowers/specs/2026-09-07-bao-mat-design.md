# Thiết kế bảo mật TramVot — 2026-09-07

## Mục tiêu

Bịt các lỗ bảo mật đang khai thác được trên `https://trampickle.vn`, và thêm
khả năng truy vết + xác thực hai lớp cho tài khoản chủ. Giữ nguyên tinh thần
zero-dependency của dự án: không kéo thêm thư viện bên thứ ba nào.

## Bối cảnh hệ thống

- Ứng dụng Go một tiến trình, `tramvot -public -https -tin-proxy`, bind `0.0.0.0:8442`.
- Cloudflare → nginx → tiến trình Go. Cổng 8442 không mở ra internet.
- Dữ liệu là file phẳng trong `data/`, chỉ tồn tại trên VPS; deploy chỉ đẩy binary.
- Hai vai trò: `chu` (toàn quyền) và `tho` (chỉ đơn được giao).
- Phiên đăng nhập giữ trong RAM, mất khi restart — cố ý.

## Mô hình mối đe doạ

Xếp theo thứ tự thực tế cho một tiệm sửa vợt một người, không theo thứ tự
sách giáo khoa:

| # | Kẻ tấn công | Muốn gì | Hiện chặn được không |
|---|---|---|---|
| 1 | Script quét tự động | Dò mật khẩu admin | **Không** |
| 2 | Người tò mò / đối thủ | Moi danh sách khách (tên, SĐT) qua `/tra-cuu` | **Không** — bypass được rate limit |
| 3 | Bot spam | Nhồi form liên hệ đầy ổ đĩa | Có, nhưng bypass được |
| 4 | Trang web độc bên thứ ba | Lừa Kendy bấm link, thực hiện hành động trong admin | Một phần (SameSite=Lax) |
| 5 | Kẻ có được mật khẩu | Đăng nhập và sửa/xoá dữ liệu | **Không** — không có 2FA, không có nhật ký |
| 6 | Kẻ quét IP gốc VPS | Đi vòng qua Cloudflare | **Không** |
| 7 | Sự cố / xoá nhầm | Mất `data/` | **Không** — chưa có backup được kiểm chứng |

## Những gì đã đúng — không đụng vào

Đã kiểm tra và xác nhận trong đợt rà 2026-09-07. Ghi ra đây để lần sau không
ai "sửa" lại thành sai:

- **Mật khẩu**: bcrypt cost 12 (`auth.go:137`). `KiemTraDangNhap` luôn chạy
  bcrypt kể cả khi không có tài khoản — không lộ tài khoản nào tồn tại qua
  thời gian phản hồi.
- **Mã phiên**: `maNgauNhien(32)` = 256 bit từ `crypto/rand`, panic nếu
  `crypto/rand` hỏng thay vì rơi sang nguồn yếu.
- **Token đơn cho khách**: `maNgauNhien(8)` = 64 bit, hex 16 ký tự. Không dò
  được. (Comment trong `server.go:756` nói "8 ký tự" là nói nhầm, giá trị
  thật là 16 ký tự hex.)
- **Path traversal ảnh**: `tenAnhSach` + đối chiếu danh sách file của đơn.
  Không ghép đường dẫn thẳng từ tham số URL ở bất kỳ handler nào.
- **XSS qua markdown**: `markdown.go` escape HTML *trước* rồi mới chèn thẻ
  (`inline()`, dòng 378-391). Nguồn markdown là chữ admin, không phải chữ khách.
- **Quyền file**: `ghiAtomic` ghi `0o600` cho mọi file dữ liệu, kể cả
  `nguoi-dung.yaml` và `bi-mat.yaml`.
- **Timeout**: `ReadHeaderTimeout: 10s` đã chặn slowloris. `ReadTimeout` và
  `WriteTimeout` 300s là cố ý — khách upload video 40MB qua 4G, và agent
  Ollama chạy lâu trên máy yếu. Không rút ngắn.
- **Cookie phiên**: `HttpOnly`, `Secure` (theo cờ `-https`), `SameSite=Lax`.

## Quyết định thiết kế

### QĐ-1. Middleware chung, không vá từng handler

Có hơn 100 route đã đăng ký. Rải kiểm tra vào từng handler là chắc chắn sót,
và mỗi route thêm sau này lại phải nhớ. Thay vào đó: một chuỗi middleware bọc
ngoài `mux`.

`NewMux(public bool) *http.ServeMux` **giữ nguyên chữ ký** — hơn chục test
hiện có đang gọi nó và nhận `*http.ServeMux`. Thêm hàm mới:

```go
func NewHandler(public bool) http.Handler
```

trả về chuỗi middleware bọc `NewMux(public)`. `main.go` dùng `NewHandler`.
Test trang cũ tiếp tục dùng `NewMux`; test middleware mới dùng `NewHandler`.

### QĐ-2. Real IP đúng — làm trước mọi thứ khác

`ipCua` hiện lấy phần tử **đầu** của `X-Forwarded-For`. Phần tử đầu là do
khách gửi; bot đổi header mỗi request là mọi rate limit thành số 0. Đây là
lỗ đang khai thác được: comment ở `server.go:796` tự nói mã đơn theo số thứ
tự nên phải chặn dò, nhưng cái chặn đó vô hiệu.

Thứ tự mới:

1. Nếu `RemoteAddr` nằm trong dải IP Cloudflare → tin `CF-Connecting-IP`.
2. Ngược lại, nếu `TinProxy` bật và `RemoteAddr` là loopback (nginx cùng máy)
   → lấy phần tử **cuối** của `X-Forwarded-For` (phần nginx tự thêm).
3. Ngược lại → `RemoteAddr`.

Dải IP Cloudflare nhúng cứng trong binary, cập nhật tay khi cần. Không tải
từ mạng lúc khởi động: một lần Cloudflare đổi endpoint là dịch vụ không lên
được, đắt hơn cái nó giải quyết.

### QĐ-3. Khoá đăng nhập theo cả IP lẫn tên tài khoản

Chỉ đếm theo IP thì botnet nhiều IP dò một tài khoản vẫn lọt. Chỉ đếm theo
tên thì kẻ xấu khoá được tài khoản Kendy bằng cách cố tình gõ sai (DoS).
Đếm cả hai, khoá theo cái nào chạm ngưỡng trước:

- 5 lần sai/IP → khoá IP đó 15 phút, mỗi lần khoá tiếp theo nhân đôi, trần 4 giờ.
- 10 lần sai/tài khoản → khoá **đăng nhập từ IP mới** cho tài khoản đó 15 phút;
  IP đã từng đăng nhập thành công vẫn vào được. Đây là chỗ tránh DoS.

Trả HTTP 429 kèm số phút còn lại. Ghi nhật ký mọi lần khoá.

### QĐ-4. CSRF: token cho phiên, double-submit cho khách

`SameSite=Lax` giữ nguyên làm lớp hai. Thêm lớp một:

- **Route đã đăng nhập**: token = HMAC-SHA256(khoá server, mã phiên), hex.
  Ẩn trong form dưới tên `_csrf`. So sánh bằng `subtle.ConstantTimeCompare`.
- **Form khách** (`/gui-yeu-cau`, `/tra-cuu`, `/app/gui-anh`): khách không có
  phiên. Dùng double-submit: middleware đặt cookie `tv_csrf` (không HttpOnly,
  vì template cần đọc — thực ra template đọc từ context, cookie chỉ để đối
  chiếu), form gửi lại cùng giá trị, middleware so hai cái.
- Khoá HMAC sinh ngẫu nhiên lúc khởi động, giữ trong RAM. Restart là token cũ
  hết hiệu lực — cùng vòng đời với phiên, nên không lệch nhau.

Middleware kiểm mọi `POST`/`PUT`/`PATCH`/`DELETE`. Có danh sách miễn trừ
rỗng — không route nào được miễn. Sai token → 403, không render lại form.

### QĐ-5. CSP dùng nonce, `style-src` chấp nhận `unsafe-inline`

Đếm thực tế trong `core/ui/`: 14 khối `<script>` inline, và hàng trăm thuộc
tính `style="..."`.

- `script-src 'self' 'nonce-<ngẫu nhiên mỗi request>'` — thêm
  `nonce="{{.Nonce}}"` vào đúng 14 thẻ script.
- `style-src 'self' 'unsafe-inline'` — sửa hàng trăm thuộc tính `style=` là
  việc lớn, rủi ro vỡ giao diện cao, mà lợi ích thấp hơn hẳn `script-src`.
  Chấp nhận, ghi lại lý do ở đây.
- `default-src 'self'; img-src 'self' data:; object-src 'none';
  base-uri 'self'; frame-ancestors 'none'`.

CSP ở đây là lớp phòng thủ **phụ**, không phải lớp chính — đã xác nhận
markdown escape đúng và `html/template` tự escape. Nếu nonce làm vỡ trang
nào thì sửa trang đó, không nới CSP.

### QĐ-6. Nhật ký hành động: JSONL theo tháng

`data/nhat-ky/YYYY-MM.jsonl`, mỗi dòng một JSON. Ghi nối, không sửa, không
xoá. Trường: `luc`, `ai`, `ip`, `viec`, `duong`, `doi_tuong`, `ket_qua`.

Ghi cái gì:

- Đăng nhập thành công / thất bại / bị khoá.
- Đăng xuất.
- Đổi mật khẩu, đổi vai trò, bật/tắt tài khoản.
- Mọi `POST` vào `/qt/*` và `/api/*`.

**Không bao giờ ghi**: giá trị field tên `mat_khau`, `mat_khau_moi`, `_csrf`,
`gemini_api_key`, `resend_api_key`, `totp`, `ma_du_phong`. Danh sách này là
allowlist ngược, kiểm bằng test.

Trang `/qt/nhat-ky`, chỉ `chu` xem, lọc theo người và theo ngày, mặc định 200
dòng gần nhất.

### QĐ-7. 2FA TOTP tự cài, không QR

RFC 6238 bằng `crypto/hmac` + `crypto/sha1`, khoảng 80 dòng. Không vẽ mã QR:
tự viết QR encoder là nhiều việc cho 1-3 người dùng. Hiện chuỗi base32 để dán
tay vào Google Authenticator, kèm URI `otpauth://` để ai muốn thì tự dựng QR.

- Bí mật lưu trong `nguoi-dung.yaml` (đã `0o600`).
- Cửa sổ chấp nhận ±1 bước 30 giây, chống lệch đồng hồ.
- Chống dùng lại: nhớ bước thời gian vừa dùng, từ chối mã trùng.
- 8 mã dự phòng dùng-một-lần, lưu bcrypt, hiện đúng một lần lúc bật.
- Bắt buộc với `chu`, tuỳ chọn với `tho`.
- Bật 2FA phải nhập đúng một mã hiện hành trước — tránh tự nhốt mình ở ngoài
  vì quét sai bí mật.

### QĐ-8. Git trước tiên

Repo `d:\TramVot\tramvot` **chưa có commit nào** và **không có `.gitignore`**.
Hệ quả:

- Sửa bảo mật xuyên nhiều file mà không có đường lùi.
- `git add .` đầu tiên sẽ nuốt `data/bi-mat.yaml` chứa khoá Gemini + Resend.

Việc đầu tiên của kế hoạch là `.gitignore` rồi commit baseline.

## Hạ tầng

Phần này Kendy chạy trên VPS; kế hoạch cung cấp config đầy đủ và lệnh kiểm chứng.

- **Chặn đi vòng Cloudflare**: nginx chỉ nhận từ dải IP Cloudflare
  (`allow`/`deny all`) hoặc firewall tương đương. Không có bước này thì mọi
  luật ở biên chỉ là trang trí — ai quét ra IP gốc là bỏ qua hết.
- **nginx**: `set_real_ip_from` dải CF + `real_ip_header CF-Connecting-IP`;
  `proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for` (ghi đè, không
  cho khách tự đặt); `client_max_body_size 50m`; `limit_req` riêng cho
  `/dang-nhap`.
- **Cloudflare**: SSL mode Full (strict), Always Use HTTPS, WAF managed rules,
  rate limit rule cho `/dang-nhap`, bypass cache cho `/qt/*`.
- **VPS**: SSH chỉ dùng khoá, ufw chỉ mở 22/80/443, bật cập nhật bản vá tự
  động, systemd unit thêm `NoNewPrivileges=yes`, `ProtectSystem=strict`,
  `ReadWritePaths=` đúng thư mục data, `PrivateTmp=yes`.
- **Backup `data/`**: đây là rủi ro lớn hơn cả bị hack. `data/` chỉ tồn tại
  trên VPS, deploy không đụng tới. Backup mã hoá hàng ngày ra nơi khác, **và
  thử phục hồi một lần** — bản backup chưa từng phục hồi thử thì chưa phải
  backup.

## Ngoài phạm vi (YAGNI)

Cố ý không làm, ghi lại để khỏi bàn lại:

- WAF tự viết — Cloudflare đã có.
- Mã hoá dữ liệu ở tầng ứng dụng — VPS bị chiếm thì khoá cũng nằm trên đó.
- Phân quyền chi tiết hơn `chu`/`tho` — hiện có 1-3 người dùng.
- SIEM, quét phụ thuộc tự động — 1 dependency trực tiếp.
- Xoay khoá phiên định kỳ — phiên đã mất khi restart.
- Trang quản lý phiên đang mở — 1-3 người dùng, `hanPhien` 12 giờ là đủ.

## Kiểm chứng

Mỗi hạng mục code có test Go. Ngoài ra, checklist thử tay trên server thật
sau khi triển khai:

1. `curl -H "Host: trampickle.vn" https://<IP-gốc>/` → phải bị từ chối.
2. Sai mật khẩu 6 lần liên tiếp → lần 6 trả 429.
3. `curl -X POST https://trampickle.vn/qt/... -b "tv_phien=<mã thật>"` không
   kèm `_csrf` → 403.
4. Đổi `X-Forwarded-For` mỗi request khi dò `/tra-cuu` → vẫn bị chặn sau 12 lần.
5. Xem `curl -sI https://trampickle.vn/` → có đủ CSP, HSTS, nosniff,
   X-Frame-Options.
6. Bật 2FA, đăng xuất, đăng nhập lại → bị hỏi mã.
7. Mở `/qt/nhat-ky` → thấy đúng các việc vừa làm, không thấy mật khẩu nào.
