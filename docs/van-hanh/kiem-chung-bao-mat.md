# Kiểm chứng bảo mật sau deploy

Chạy sau mỗi lần đẩy binary mới lên VPS. Không có test tự động nào thay được
bước này: cấu hình nginx, Cloudflare và binary phải khớp nhau, và chỉ chạy
thật mới biết là khớp.

Mục nào không đạt thì **ghi lại đúng output rồi dừng**, đừng bỏ qua.

## Deploy

```sh
cd d:/TramVot/tramvot
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o tramvot-linux .
scp tramvot-linux tramvot@<IP-gốc>:/tmp/tramvot-moi
ssh tramvot@<IP-gốc> 'sudo systemctl stop tramvot \
  && sudo mv /tmp/tramvot-moi /opt/tramvot/tramvot \
  && sudo chmod +x /opt/tramvot/tramvot \
  && sudo chown tramvot:tramvot /opt/tramvot/tramvot \
  && sudo systemctl start tramvot \
  && sudo systemctl status tramvot --no-pager | head -5'
```

`CGO_ENABLED=0` là bắt buộc: có CGO thì binary đòi glibc đúng phiên bản của
máy build, lên VPS là không chạy.

**Không bao giờ** rsync `data/` lên VPS. `data/` trên server là bản thật duy
nhất — `noi-dung.yaml`, đơn hàng, sổ tiền chỉ có ở đó.

## 7 phép kiểm

### 1. Không đi vòng qua Cloudflare được

```sh
curl -sk -o /dev/null -w "%{http_code}\n" -H "Host: trampickle.vn" https://<IP-gốc>/
```

Đúng: `403`. Ra `200` là khối `allow/deny` trong nginx chưa ăn.

### 2. Header bảo mật có mặt

```sh
curl -sI https://trampickle.vn/ | grep -iE "content-security|strict-transport|x-frame|x-content"
```

Đúng: đủ 4 dòng. Thiếu `strict-transport-security` nghĩa là service đang chạy
thiếu cờ `-https`.

### 3. Khoá đăng nhập

Gõ sai mật khẩu 6 lần ở `/dang-nhap`.

Đúng: tới lần thứ 6 hiện "Sai quá nhiều lần. Thử lại sau N phút." (mã HTTP
429). Sau đó vào `/qt/nhat-ky` phải thấy các mục `dang-nhap / sai-mat-khau`
rồi `dang-nhap / bi-khoa`.

Mở khoá sớm bằng cách khởi động lại service — bộ đếm nằm trong RAM.

### 4. Giả mạo IP không qua mặt được bộ đếm

```sh
for i in $(seq 1 15); do
  curl -s -X POST https://trampickle.vn/tra-cuu \
    -H "X-Forwarded-For: 1.2.3.$i" -d "ma=X" | grep -o "hơi nhiều lần" | head -1
done
```

Đúng: từ khoảng lần thứ 13 trở đi in ra `hơi nhiều lần`.

Đây là phép kiểm cho `core/realip.go`: nếu đổi `X-Forwarded-For` mỗi lần mà
KHÔNG bao giờ bị chặn, nghĩa là Go đang tin header khách tự đặt — mọi bộ đếm
theo IP trong app đều vô hiệu.

(Trang này trả lời bằng thông báo trong trang chứ không đổi mã HTTP, nên phải
tìm theo chữ chứ không theo `%{http_code}`.)

### 5. Thiếu token CSRF thì bị chặn

```sh
curl -s -o /dev/null -w "%{http_code}\n" -X POST https://trampickle.vn/tra-cuu -d "ma=X"
```

Đúng: `403`.

### 6. 2FA thật sự chặn

Mở `/qt/2fa` → Bắt đầu → quét mã bằng ứng dụng xác thực → gõ mã → Bật.
**Chép 8 mã dự phòng ra chỗ an toàn ngay lúc đó**, trang không hiện lại lần
thứ hai.

Đăng xuất, đăng nhập lại. Đúng: sau khi gõ đúng mật khẩu thì bị hỏi mã 6 số.

Thử luôn: gõ lại đúng mã vừa dùng lần nữa → phải bị từ chối (chống phát lại).

### 7. Nhật ký ghi việc, không ghi bí mật

Mở `/qt/nhat-ky`.

Đúng: thấy các việc vừa làm ở trên (đăng nhập, bật 2FA...), và **không** thấy
chuỗi mật khẩu, khoá API hay mã TOTP nào trong cột Chi tiết.

## Kiểm mắt thường giao diện

CSP nonce và ô CSRF chạm vào 25 file template. Mở tay và bấm thử, **có mở
Console trình duyệt (F12)** ở mỗi trang:

- `/` — ảnh hero chạy
- `/tra-cuu` — tra một mã đơn thật
- `/lien-he` — gửi thử một yêu cầu kèm ảnh
- `/qt/don` — mở một đơn, đổi trạng thái
- `/qt/giao-dien` — đổi theme, tải logo
- `/qt/bai-viet` — sửa và lưu một bài
- `/noi-bo` — hỏi agent một câu

Dòng `Refused to execute inline script` trong Console nghĩa là còn thẻ
`<script>` sót nonce. Dòng `403` khi bấm một nút nghĩa là form đó sót ô
`_csrf`.
