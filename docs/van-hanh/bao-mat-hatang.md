# Việc bảo mật hạ tầng — Kendy chạy trên VPS

Phần trong code đã xong và đã deploy. File này là phần **không nằm trong
code**: nginx, tường lửa, SSH, Cloudflare, backup. Agent không tự SSH vào
máy chủ, nên những việc dưới đây Kendy chạy tay.

Làm theo thứ tự. Mỗi mục có lệnh kiểm chứng; không qua được thì dừng ở đó,
đừng làm tiếp.

Đáng làm nhất là **mục 1** (chặn đi vòng qua Cloudflare) và **mục 7**
(backup). Mục 3 có bước dễ tự khoá mình ra khỏi VPS — đọc hết mục rồi mới gõ.

---

## 1. nginx chỉ nhận từ Cloudflare

```sh
sudo cp nginx-trampickle.conf /etc/nginx/sites-available/trampickle.vn
sudo nginx -t && sudo systemctl reload nginx
```

**Kiểm chứng** — chạy từ máy nhà, thay `<IP-gốc>` bằng IP thật của VPS:

```sh
curl -sS -o /dev/null -w "%{http_code}\n" \
  -H "Host: trampickle.vn" https://<IP-gốc>/ --insecure
```

Phải ra `403`. Ra `200` nghĩa là khối `allow/deny` chưa ăn — lúc đó mọi luật
WAF và rate limit bên Cloudflare đang là trang trí, ai biết IP gốc là vào
thẳng.

Rồi kiểm tên miền vẫn sống:

```sh
curl -sI https://trampickle.vn/ | head -1
```

**Bẫy chứng chỉ.** Khối `deny all` chặn cả certbot HTTP-01, vì Let's Encrypt
gọi thẳng IP gốc chứ không qua Cloudflare. File nginx đã chừa sẵn
`/.well-known/acme-challenge/` ở cổng 80 cho việc này. Thử gia hạn khan một
lần cho chắc, đừng đợi 90 ngày sau mới biết:

```sh
sudo certbot renew --dry-run
```

## 2. Tường lửa

```sh
sudo ufw default deny incoming
sudo ufw allow 22/tcp
sudo ufw allow 80,443/tcp
sudo ufw enable
sudo ufw status verbose
```

Kiểm chứng: **cổng 8442 không được có trong danh sách**. Go nghe ở 8442 và
tin header `X-Real-IP` khi request tới từ loopback — mở 8442 ra internet là
cho bất kỳ ai tự khai mình là IP nào cũng được, phá sạch cả khoá đăng nhập
lẫn nhật ký.

## 3. SSH chỉ dùng khoá

Trong `/etc/ssh/sshd_config`:

```
PasswordAuthentication no
PermitRootLogin no
KbdInteractiveAuthentication no
```

```sh
sudo sshd -t && sudo systemctl restart ssh
```

**Trước khi thoát phiên SSH hiện tại**, mở một cửa sổ thứ hai và SSH vào lại
để chắc chắn khoá dùng được. Phiên đang mở vẫn sống kể cả khi cấu hình sai —
thoát ra rồi mới biết là muộn.

## 4. systemd siết quyền

```sh
sudo cp tramvot.service /etc/systemd/system/tramvot.service
sudo systemctl daemon-reload
sudo systemctl restart tramvot
sudo systemctl status tramvot --no-pager
```

Nếu trước đây service chạy bằng `root` mà giờ đổi sang `tramvot`, phải trả
quyền sở hữu `data/` lại, không thì app chạy nhưng không ghi được gì:

```sh
sudo chown -R tramvot:tramvot /opt/tramvot
sudo chmod 600 /opt/tramvot/data/bi-mat.yaml /opt/tramvot/data/nguoi-dung.yaml
sudo chmod 700 /opt/tramvot/data/nhat-ky
sudo systemctl restart tramvot
```

Kiểm chứng: `curl -sI https://trampickle.vn/ | head -1` ra `HTTP/2 200`, rồi
vào `/qt` sửa thử một chữ và tải lại trang — sửa được nghĩa là quyền ghi ổn.

## 5. Cập nhật bản vá tự động

```sh
sudo apt install unattended-upgrades
sudo dpkg-reconfigure -plow unattended-upgrades
```

## 6. Cloudflare (bấm trên web)

- SSL/TLS → **Full (strict)**
- SSL/TLS → Edge Certificates → **Always Use HTTPS: bật**
- Security → WAF → **Managed rules: bật**
- Security → WAF → Rate limiting rules → tạo luật:
  `URI Path equals /dang-nhap` → 10 request / 1 phút / IP → Block 15 phút
- Caching → Cache Rules → `URI Path starts with /qt` → **Bypass cache**

Mục cuối quan trọng hơn vẻ ngoài của nó: trang quản trị mà bị cache ở biên
thì hai người dùng có thể nhìn thấy trang của nhau.

## 7. Backup `data/` — quan trọng hơn mọi mục trên

`data/` chỉ tồn tại trên VPS. Deploy chỉ đẩy binary, không đụng tới nó. Mất
là mất hết đơn hàng, sổ tiền, bài viết, nội dung trang.

```sh
sudo mkdir -p /opt/backup && sudo chmod 700 /opt/backup
sudo tee /opt/tramvot/backup.sh > /dev/null <<'EOF'
#!/bin/sh
set -e
NGAY=$(date +%F)
tar czf - -C /opt/tramvot data \
  | gpg --batch --yes --symmetric --cipher-algo AES256 \
        --passphrase-file /root/.backup-pass \
        -o /opt/backup/data-$NGAY.tar.gz.gpg
find /opt/backup -name 'data-*.gpg' -mtime +30 -delete
EOF
sudo chmod 700 /opt/tramvot/backup.sh
```

Mã hoá vì trong `data/` có `bi-mat.yaml` (khoá Gemini, khoá Resend),
`nguoi-dung.yaml` (hash mật khẩu, bí mật TOTP) và `nhat-ky/` (nhật ký thao
tác). Bản backup để trần là gói sẵn toàn bộ bí mật của trạm.

Đặt mật khẩu backup — gõ một chuỗi dài, **lưu ngay vào trình quản lý mật
khẩu**. Mất chuỗi này là mất luôn bản backup:

```sh
sudo sh -c 'printf "%s" "MAT-KHAU-DAI-O-DAY" > /root/.backup-pass'
sudo chmod 600 /root/.backup-pass
```

Chạy hàng ngày lúc 3h sáng:

```sh
sudo crontab -e
# thêm dòng:
0 3 * * * /opt/tramvot/backup.sh
```

**Kiểm chứng — bắt buộc làm một lần.** Bản backup chưa từng phục hồi thử thì
chưa phải backup:

```sh
sudo /opt/tramvot/backup.sh
cd /tmp && sudo gpg --batch --passphrase-file /root/.backup-pass \
    -d /opt/backup/data-$(date +%F).tar.gz.gpg | tar xzf -
ls /tmp/data/
```

Phải thấy `don`, `bai-viet`, `so-tien`, `nguoi-dung.yaml`... Xong thì dọn:
`sudo rm -rf /tmp/data`.

## 8. Kéo bản backup về máy nhà

Backup nằm cùng máy với dữ liệu thì cháy ổ là mất cả hai.

Chạy từ máy nhà, mỗi tuần một lần:

```sh
scp tramvot@<IP-gốc>:/opt/backup/data-$(date +%F).tar.gz.gpg "d:/TramVot-backup/"
```
