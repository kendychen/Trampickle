# Đơn giày và luồng gửi đi gia công

Chốt ngày 2026-09-08.

## Vì sao

Hai việc thật đang không có chỗ trong phần mềm:

1. Trạm nhận cả **giày** (thay đế — `THAY_DE_GIAY` đã nằm trong `bang-gia.yaml`
   từ giai đoạn 1) nhưng model `Don` thuần vợt: hãng vợt, cân trước, cân sau.
   Nhận một đôi giày phải nhét mọi thứ vào ô ghi chú.
2. Nhiều ca **gửi ra tiệm ngoài** rồi ăn phần chênh. Hiện không có chỗ ghi
   gửi cho ai, gửi ngày nào, trả tiệm bao nhiêu — nên không biết mỗi ca lãi
   thật bao nhiêu, và tiền trả tiệm không vào sổ.

## Tách ở đâu, chung ở đâu

Giao diện tách hẳn: URL riêng, mã đơn riêng, cột riêng, form riêng.

Kho dữ liệu **không** tách. `Don` đang bị bốn chỗ ngoài trang quản trị dùng
chung: tra cứu khách (`server.go`), đối chiếu sao kê (`saoke.go`), tổng quan
tiền (`sotien.go`), mail hằng ngày (`tudong.go`). Tách đôi kho thì mỗi chỗ đó
phải hỏi hai nơi rồi gộp, và ngày ai đó quên một chỗ thì khách gõ mã `TG-`
vào ô tra cứu sẽ nhận "không có đơn này".

→ `Don` thêm `Loai` (`vot` | `giay`), vẫn lưu ở `data/don-hang/`.
Đơn cũ không có trường này: rỗng đọc là `vot`.

## Dữ liệu

### Don — trường mới

| Trường | Ý nghĩa |
|---|---|
| `Loai` | `vot` hoặc `giay`. Rỗng = `vot` (đơn cũ). |
| `GiayHang` | Hãng / đời giày |
| `GiaySize` | Cỡ |
| `GiayKieu` | `chay_bo` \| `di_lai` — hai kiểu duy nhất nhận |
| `DeHienTai` | Đế hiện tại, mức mòn |
| `GuiDi` | Khối gửi đi gia công (dưới) |

Mã đơn đếm riêng theo tiền tố: `TV-2609-001` cho vợt, `TG-2609-001` cho giày.

Đơn giày không có ô cân. `VuotNguongCan()` chỉ áp cho loại vợt — cam kết
không tăng quá 3g là cam kết về vợt.

Form giày in thẳng điều kiện từ `bang-gia.yaml`: chỉ nhận giày chạy bộ và đi
lại, **từ chối** giày court / cầu lông / pickleball / tennis. Lý do in ngay
cạnh ô chọn kiểu để không nhận nhầm lúc khách đứng trước mặt.

### GuiDi — dùng cho cả hai loại đơn

```go
type GuiDi struct {
    MaDoiTac    string  // rỗng = đơn tự làm tại trạm
    NgayGui     string
    NgayHenVe   string
    NgayVe      string
    TraDoiTac   int     // tiền mình trả tiệm — GÕ TAY
    DaTra       bool
    MaKhoanChi  string  // khoản chi đã sinh trong sổ, để không sinh trùng
    GhiChu      string
}
```

Lãi thật của đơn = `TongTien - GuiDi.TraDoiTac`. Cả hai số đều do người gõ
vào, không có công thức nào — luật cứng #1 ở đầu `donhang.go` (chỉ
`src/quote.py` được sinh ra giá bán) vẫn nguyên vẹn. Đây là phép trừ hai con
số đã ghi vào sổ, cùng loại việc với `ConNo()`.

Vì sao ghi hai số thật thay vì gõ một tỷ lệ %: tiền trả tiệm đổi theo từng
đôi (loại đế, cỡ, mức mòn) — chính `bang-gia.yaml` đã ghi thế khi để
`bao_gia_rieng: true`. Gõ % rồi để máy suy ra tiền trả tiệm là bịa ra một con
số tiền không ai chốt với ai.

### Đối tác — data/doi-tac.yaml

Đi đúng lối `data/lien-he.yaml`: file trong `data/`, không bao giờ bị đè khi
deploy, chưa có file thì danh sách rỗng chứ không phải lỗi.

```yaml
doi_tac:
  - ma: tiem-de-abc
    ten: "Tiệm đế ABC"
    nghe: "Thay đế giày"
    lien_he: "0901234567"
    dia_chi: "..."
    ghi_chu: "..."
    ngung: false
```

`ngung: true` thì mất khỏi dropdown nhưng đơn cũ vẫn hiện đúng tên.

## Trạng thái

Chèn hai mã sau `dang_sua`:

| Mã | Tên trong quản trị | Câu khách đọc |
|---|---|---|
| `da_gui_di` | Đã gửi đi gia công | "Đang được xử lý" |
| `da_nhan_ve` | Đã nhận về, đang kiểm | "Đang được kiểm tra lần cuối" |

Câu cho khách trung tính — khách không cần biết trạm thuê ngoài.
Cả hai đều `DangChay: true`.

`QuaHenDoiTac()`: đang ở `da_gui_di` mà quá `NgayHenVe`. Khác với `QuaHan()`
(quá hẹn trả khách) — hai câu hỏi khác nhau: "tiệm trễ" và "mình trễ với
khách".

## Tiền trả đối tác vào sổ

Bấm "Đã trả đối tác" → sinh `Khoan{Loai: chi, Nhom: van_hanh, MaDon: ...}`,
đúng cơ chế phiếu nhập kho đang sinh khoản chi vật tư. Nhóm `van_hanh` đã có
sẵn mô tả "Ship, bao bì, **thuê ngoài**, quảng cáo, phí sàn".

Mã khoản lưu lại trong `GuiDi.MaKhoanChi`: bấm hai lần không sinh hai dòng,
bỏ tick thì xoá đúng khoản đó. Không có bước này thì lãi lỗ tháng nói dối
theo hướng nguy hiểm nhất — báo lãi cao hơn thật.

## Trang

| URL | Việc |
|---|---|
| `/qt/don` | Chuyển hướng `/qt/don-vot`. Link cũ, bookmark cũ không chết. |
| `/qt/don-vot` | Danh sách đơn vợt |
| `/qt/don-giay` | Danh sách đơn giày — cột "Giày / cỡ" và "Đối tác" thay cột "Vợt" |
| `/qt/don-moi?loai=giay` | Form tạo đơn, đổi khối giữa theo loại |
| `/qt/don/{ma}` | Chi tiết — giữ nguyên, mã đơn đã tự nói loại |
| `/qt/doi-tac` | Danh sách đối tác + số liệu theo từng tiệm (chỉ chủ) |

Ô số đầu mỗi trang danh sách đếm theo loại, thêm ô **"đang ở tiệm ngoài"**.

Trang đối tác hiện theo từng tiệm: bao nhiêu đơn, đã trả bao nhiêu, còn nợ
tiệm bao nhiêu, mấy đơn quá hẹn về.

Menu: "Đơn vợt", "Đơn giày", "+ Tạo đơn"; mục "Đối tác" nằm trong nhóm chủ
trạm — chọn tiệm gia công và biết đã trả tiệm bao nhiêu là việc của chủ.

## Chỗ khác phải sửa theo

- `LocDon` thêm bộ lọc `Loai`; ô tìm gộp thêm hãng giày.
- `LayThongKeDon(loai)` — rỗng là tất cả. Năm chỗ gọi hiện tại truyền `""`.
- `MaDonMoi(loai)` — hai chỗ gọi.
- `saoke.go` không đổi: đối chiếu tiền, không quan tâm loại đơn.
- Tra cứu khách không đổi: `TraCuuChoKhach` tìm theo mã nên `TG-` chạy sẵn.

## Kiểm chứng

- Mã `TG` đếm độc lập với `TV`; tạo xen kẽ không nhảy số của nhau.
- Lọc theo loại không lẫn; đơn cũ không có `Loai` nằm ở trang vợt.
- Đơn giày không hiện ô cân, không bao giờ báo "vượt ngưỡng cân".
- Bấm "đã trả đối tác" sinh đúng một khoản chi; bấm lại không sinh thêm; bỏ
  tick thì khoản biến mất khỏi sổ.
- Lãi thật = tổng đơn trừ tiền trả tiệm.
- Thợ mở được `/qt/don-giay`; thợ **không** mở được `/qt/doi-tac`.
- Chụp Chrome headless qua CDP xem bố cục trước khi deploy.
