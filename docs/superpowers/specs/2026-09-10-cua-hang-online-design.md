# Bản Online: cửa hàng dịch vụ và công tắc chế độ

Chốt ngày 2026-09-10.

## Vì sao

Site hiện viết cho khách **mang vợt tới xưởng**. Nhưng phần lớn khách không ở
gần, và hai thứ họ cần để dám gửi đồ đi thì site chưa có:

1. **Không có chỗ đặt.** Form gửi ảnh ở `server.go:711` chỉ đẻ ra một
   `yeuCauKhach` nằm chờ trong `data/tep-yeu-cau`. Chủ xem ở `/qt/yeu-cau` rồi
   *tự tay* tạo đơn. Khách gửi xong không có mã nào để ghi lên kiện hàng,
   không có gì để tra cứu, không biết bao giờ được trả lời.
2. **Không có địa chỉ.** `data/lien-he.yaml` mới chỉ có điện thoại và Zalo.
   Gửi một cây vợt 5 triệu cho một số điện thoại là việc không ai làm.

Bản Online sửa đúng hai chỗ đó: một luồng đặt hàng đẻ ra đơn thật ngay, và
một địa chỉ xưởng in rõ trên trang.

## Phạm vi — và cái cố ý không làm

Làm: công tắc chế độ nội dung, trang `/cua-hang`, một trạng thái đơn mới,
thu tiền qua VietQR hoặc COD, địa chỉ xưởng.

**Không** làm, và không phải vì hết thời gian:

| Không làm | Vì sao |
|---|---|
| Bộ theme màu mới | Màu và logo giữ nguyên. Chữ "giao diện" trong yêu cầu gốc nghĩa là **nội dung** chuyển sang giọng online, không phải bảng màu. Hệ theme đã xoá hẳn hồi làm Material 3, không dựng lại. |
| Tài khoản khách | `Don.Token` đã cho tra cứu không đoán được. Thêm tài khoản là thêm mật khẩu để quên, để lộ, để đi khôi phục. |
| Giỏ hàng nhiều đơn | Một lần gửi là một kiện là một đơn. Kiện chứa hai cây vợt thì hai dòng dịch vụ trong cùng đơn, không phải hai đơn. |
| Cổng thanh toán | Cần đăng ký hộ kinh doanh, mất 1–2% mỗi giao dịch. VietQR khớp sao kê đã đủ và miễn phí. Thiết kế chừa chỗ cắm sau. |
| API hãng vận chuyển | Khách tự gửi. Nối API là việc của ngày có đủ đơn để việc đặt shipper thành gánh nặng. |
| Bán hàng hoá | Cửa hàng này bán **dịch vụ**: sửa vợt pickleball và thay đế giày. Không bán vợt, không bán grip lẻ. |

## Công tắc chế độ

`data/giao-dien.yaml` hiện chỉ còn khoá `theme: tpic` — khoá chết, không code
nào đọc từ khi hệ theme bị xoá. Dọn nó đi, thay bằng:

```yaml
che_do: tai_xuong   # tai_xuong | online
```

Chọn ở `/qt/giao-dien` — đúng chỗ Kendy đã quen bấm.

### Vì sao không tạo bộ khoá nội dung thứ hai

`noidung.go` có một luật viết sẵn ở đầu file: *một khoá chỉ khai MỘT lần, ở
`CayND`*. Khai hai chỗ thì có ngày nhãn nói một đằng chữ chạy một nẻo.

Cách dễ nghĩ nhất — đặt `hero.title.tai_xuong` và `hero.title.online` thành
hai khoá — vi phạm đúng luật đó, và nhân đôi cây khoá cho một site mà đa số
chữ hai chế độ dùng chung.

→ `MucND` thêm **một trường**: `MacOn` — chữ mặc định của bản Online.

```go
type MucND struct {
    Khoa string
    Nhan string
    Mac  string  // bản Tại xưởng, cũng là bản dùng chung
    MacOn string // bản Online. Rỗng = hai chế độ dùng chung Mac.
    Dai  bool
}
```

`{{nd "khoa"}}` không đổi cách gọi — nó tự trả bản đúng theo chế độ đang bật.
Template không biết có hai chế độ, và không được biết.

Khoá Kendy sửa tay lưu tách trong `data/noi-dung.yaml`:

```yaml
hero.title: "Mang vợt tới, lấy về như mới"
hero.title@online: "Gửi vợt tới, nhận lại như mới"
```

Hậu tố `@online` chỉ xuất hiện với khoá Kendy đã sửa ở chế độ Online. Khoá
chưa sửa vẫn lấy `MacOn` trong code — đúng luật cũ: chữ mới lên trang ngay khi
deploy, không chờ ai vào admin gõ lại.

### Admin

`/qt/noi-dung` hiện ô của **chế độ đang bật**, kèm một dòng nhắc rõ đang sửa
bản nào và nút nhảy sang bản kia. Không hiện hai ô cạnh nhau: hai ô cạnh nhau
là hai thứ để so sánh, mà việc ở đây là viết một bản cho xong.

Khoá không có `MacOn` hiện thêm nhãn "dùng chung cả hai chế độ" — sửa nó là
sửa cả hai bản, phải nói trước.

### Test

`TestKhoaNDCoDu` mở rộng: khoá nào khai `MacOn` thì cả `Mac` lẫn `MacOn` phải
khác rỗng. Một bản rỗng nghĩa là trang trống ở đúng chế độ ít ai xem — và
không ai phát hiện ra.

Thêm `TestChuyenCheDoKhongMatChu`: bật lần lượt hai chế độ, dựng mọi khoá
trong cây, không khoá nào ra chuỗi rỗng.

## Cửa hàng `/cua-hang`

Bốn bước, một trang, không đăng nhập:

1. **Món** — vợt hay giày.
2. **Gói dịch vụ** — đọc thẳng từ `bang-gia.yaml`, lọc theo món và theo
   `giai_doan_hien_tai`. Hiện **"giá từ …"**, không hiện một con số cứng.
   Dịch vụ giày chỉ có `THAY_DE_GIAY`, và in ngay điều kiện nhận: chỉ giày
   chạy bộ và đi lại.
3. **Tình trạng** — hãng/đời, mô tả hỏng, ảnh (dùng lại đường tải ảnh đã có
   của form gửi ảnh), và **giá trị món** — `Don.VotGiaTri` đã có sẵn để quyết
   định có nhận ship hay không, theo `nguong.nguong_mo_kenh_ship_dong`.
4. **Người nhận** — tên, số điện thoại, địa chỉ nhận lại.

Gửi xong tạo `Don` thật: `TrangThai = cho_hang_ve`, `KenhNhan = ship`,
`NguonDon = web`. Mã đơn sinh theo tiền tố đã có (`TV-` vợt, `TG-` giày).

### Màn kết — thứ quan trọng nhất của cả luồng

Không phải "cảm ơn, chúng tôi sẽ liên hệ". Màn này phải trả lời đúng câu khách
đang hỏi: *giờ tôi làm gì với cây vợt đang cầm?*

- Địa chỉ xưởng, in to, có nút sao chép.
- **Mã đơn, kèm câu "ghi mã này lên kiện hàng"** — không có nó thì kiện tới
  nơi không ai biết của ai.
- Hướng dẫn đóng gói, 4–5 dòng, sửa được ở admin.
- Link tra cứu theo `Token`, và nhắc lưu lại.

Cùng nội dung đó gửi vào SMS/Zalo thì tính sau — mail đang vướng tên miền xác
thực, không chặn việc này.

### Từ chối trước khi khách gửi

`checklist-tu-choi.md` và `vot.yaml → dong_vot[].nhan_sua` đã biết ca nào
không nhận. Bước 3 tra ngay: món thuộc dòng không nhận thì nói **trước** khi
khách bỏ tiền ship. Ngưỡng `ship_ca_tu_choi_dong: 30000` là số tiền mất mỗi
lần chặn hụt.

## Đổi trên `Don` — cố ý ít nhất có thể

### Một trạng thái mới, đúng một

```go
TTChoHangVe = "cho_hang_ve"  // đã đặt online, kiện chưa tới
```

Đứng trước `TTMoi` trong `CacTrangThai`, `DangChay: true`. Câu cho khách:
"Đã nhận đơn, đang chờ hàng về". Kiện tới, thợ bấm một cú sang `moi` — từ đó
chảy nguyên luồng cũ: `kiem_tra` → `bao_gia` → `dang_sua` → `xong` → `da_giao`.

### Vì sao KHÔNG có trạng thái "chờ thanh toán"

Nghe thì cần: sửa xong, phải biết đơn nào đã trả tiền mới đóng gói. Nhưng
`Don.ConNo()` đã trả lời được câu đó, và nó cộng từ **sổ tiền** — nguồn sự
thật duy nhất cho tiền mặt.

Thêm một trạng thái `cho_tra` là dựng nguồn sự thật thứ hai: ngày đơn ở trạng
thái `cho_tra` mà sổ đã ghi đủ tiền, hoặc ngược lại, sẽ không ai biết bên nào
đúng. Đúng cái lỗi `KE-HOACH-KHO-TIEN.md` đã cấm với tồn kho và với
`Don.DaThu`.

→ Bảng đơn lọc "sửa xong, chưa thu đủ" bằng `TrangThai == xong && ConNo() > 0`.

### Trường mới

| Trường | Ý nghĩa |
|---|---|
| `KhachDiaChi` | Địa chỉ ship trả về. Rỗng với đơn nhận trực tiếp. |
| `HinhThucTra` | `qr` \| `cod`. Rỗng = chưa chọn. |
| `MaVanDonDen` | Vận đơn khách gửi tới. Khách tự điền ở trang tra cứu, hoặc thợ ghi khi nhận. |
| `MaVanDonVe` | Vận đơn trả về. |
| `NguonDon` | `web` \| `tay`. Rỗng đọc là `tay` — đơn cũ. |

Đơn cũ thiếu hết mấy trường này và vẫn phải đọc được, y như cách `Loai` rỗng
đọc là `vot`.

## Thu tiền

Khi đơn sang `xong`, trang tra cứu của khách hiện: ảnh kết quả, số tiền chốt,
hai nút.

**QR.** Sinh VietQR với nội dung điền sẵn **đúng mã đơn**. `saoke.go` đã bắt
mã dạng `TV-2609-001` ngay trong nội dung chuyển khoản (`reMaDon`,
`saoke.go:25`), chấp nhận cả khi ngân hàng nuốt dấu gạch. Dán sao kê là khớp,
không phải viết thêm gì. Đạt `nguong.free_ship_ve_tu_dong: 500000` thì miễn
ship về.

**COD.** Cộng phí thu hộ, không miễn ship. Đơn đánh dấu `HinhThucTra = cod`.

### Đánh đổi: COD ghi tay ở giai đoạn 1

Tiền COD hãng vận chuyển trả về **gộp lô** sau 2–5 ngày, một khoản cho nhiều
đơn, không mang mã đơn nào. `saoke.go` cố tình không đoán bừa những dòng như
vậy — nó sẽ đánh dấu cần kiểm tra và để người nhìn, đúng như thiết kế.

Giai đoạn 1: nhận tiền COD thì gõ tay từng khoản thu vào sổ, gắn mã đơn. Sổ
tiền đã có đường nhập tay.

**Quay lại làm màn đối soát COD khi** COD vượt ~20 đơn/tháng — lúc đó gõ tay
thành gánh nặng thật và sai sót bắt đầu đắt hơn công viết.

## Địa chỉ xưởng

Hạ tầng **đã có đủ**, chỉ chưa có dữ liệu. `LienHe.DiaChi` và
`LienHe.GioLamVic` có sẵn trong `config.go:25`; `/qt/lien-he` đã có ô nhập
(`lienhe.go:73`); chân trang (`phan.html:73`) và trang liên hệ
(`lienhe.html:31-32`) đã hiện và đã ẩn đúng khi rỗng.

→ Việc còn lại chỉ là hai thứ:

1. **Kendy điền địa chỉ thật** ở `/qt/lien-he`. Không có nó thì cửa hàng
   không mở được — khách không biết gửi đi đâu.
2. Màn kết của cửa hàng đọc `LienHeHienTai().DiaChi`. Địa chỉ rỗng thì
   **`/cua-hang` trả về trang "tạm chưa nhận đơn online"** chứ không hiện màn
   kết thiếu địa chỉ. Để khách gửi đơn xong mới phát hiện không có chỗ gửi là
   hỏng nặng hơn nhiều so với đóng cửa hàng một hôm.

## Kiểm thử

Mở rộng bộ test đã có, không dựng bộ mới:

- `TestKhoaNDCoDu` — khoá có `MacOn` thì hai bản đều khác rỗng.
- `TestChuyenCheDoKhongMatChu` — hai chế độ, không khoá nào ra rỗng.
- `TestDatHangTaoDon` — POST `/cua-hang` đẻ ra `Don` ở `cho_hang_ve`, có
  `Token`, có mã đúng tiền tố theo món.
- `TestDonCuKhongCoTruongMoi` — nạp một đơn JSON không có `NguonDon`,
  `HinhThucTra`: đọc được, `NguonDon` rỗng hiểu là `tay`.
- `TestTuChoiTruocKhiGui` — chọn dòng vợt `nhan_sua: false` thì bước 3 chặn.
- `TestQRMangMaDon` — nội dung QR sinh ra khớp được bằng chính `reMaDon`.
- Test giá: trang cửa hàng không bao giờ in một con số cứng cho dịch vụ có
  khoảng giá — chỉ "giá từ".

## Thứ tự làm

1. Công tắc chế độ + `MacOn` + admin + test. Chưa đụng cửa hàng.
2. Địa chỉ trong `lien-he.yaml` + chân trang.
3. Viết bản nội dung Online cho các trang đang có.
4. Trạng thái `cho_hang_ve` + trường mới trên `Don`.
5. Trang `/cua-hang` bốn bước + màn kết.
6. VietQR ở trang tra cứu + COD ghi tay.

Bước 1–3 lên được ngay mà chưa cần cửa hàng chạy. Bước 4 là chỗ duy nhất đụng
code đang chạy, và `data/don-hang` còn rỗng nên đổi cấu trúc chưa đau.
