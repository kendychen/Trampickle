# Năm việc còn thiếu để vận hành cửa hàng — Kế hoạch

> **Cho người thực hiện:** làm tuần tự Việc 1 → 5. Mỗi việc tự đứng được,
> test xanh rồi mới sang việc sau. Không dùng subagent, không dùng Workflow
> (Kendy cấm). Deploy sau mỗi việc, không cần hỏi lại.

**Mục tiêu:** bổ sung quản lý khách hàng (kèm tự giảm giá khách cũ), tem dán
đồ, bảo hành có gốc, rổ đơn xong bị bỏ quên, và giá vốn vật tư vào con số lãi.

**Bối cảnh:** hệ đã có đơn hai nghề (vợt/giày) với 12 trạng thái, kho vật tư
có định mức, sổ tiền, đối tác gia công, thống kê theo kỳ, thông báo 3 kênh.
Xem `core/donhang.go`, `core/kho.go`, `core/sotien.go`, `core/thongke.go`.

**Nguồn:** review hệ thống ngày 2026-09-12. Kendy chọn làm mục 1, 2, 3, 4, 6
(bỏ mục 5 — công thợ, chỉ cần khi có thợ thứ hai).

---

## Ràng buộc chung — áp cho mọi việc bên dưới

- Go, không thêm thư viện ngoài. Đã có `gopkg.in/yaml.v3`.
- Danh mục nhỏ ghi ra `data/<ten>.yaml` (theo `core/doitac.go`); dữ liệu
  nhiều bản ghi ghi mỗi bản một JSON trong `data/<thu-muc>/` (theo
  `core/donhang.go:466`).
- **Deploy chỉ đẩy binary.** `data/` trên VPS không bao giờ bị rsync đè.
  Build: `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /d/TramVot/build/tramvot-linux-amd64 .`
- Mọi chữ hiện ra cho khách phải là khoá CMS trong `core/noidung_cay.go`.
  Chữ trong trang quản trị thì gõ thẳng vào template — admin không qua CMS.
- Hai test `TestKhoaNDCoDu` và `TestKhoaNDKhongThua` bắt khoá khai và khoá
  dùng phải trùng khít hai chiều.
- CSS app phải có tiền tố `body.la-app`. `/app` gói trong đúng một màn không
  cuộn.
- Không dùng escape CSS (`\203A`) trong `css.html` — `html/template` nuốt.
- Chạy `go test ./core/` trước mỗi lần deploy.
- **Trên server hiện chỉ có 2 đơn thật.** Không phải lo vá dữ liệu cũ; nhưng
  code vẫn phải chịu được đơn không có trường mới (đơn cũ đọc ra là rỗng).

---

## Việc 1: Quản lý khách hàng (kiểu KiotViet, đã cắt phần thừa)

**Vì sao:** hiện mỗi đơn lưu tên/SĐT/email rời rạc. Tìm theo SĐT ra được đơn
cũ (`core/donhang.go:580`) nhưng tạo đơn mới vẫn gõ lại từ đầu — sai một chữ
là thành khách khác, và không bao giờ đếm được bác này đã tới mấy lần. Khách
quay lại là nguồn doanh thu chính của tiệm sửa chữa.

**Kendy chốt:** làm theo lối KiotViet — một trang khách hàng thật, có nhóm
khách, công nợ, tổng chi tiêu, lịch sử, nhập/xuất tệp; và lúc tạo đơn thì gõ
số là ra khách như màn bán hàng.

### Lấy gì của KiotViet, bỏ gì

**Lấy:** mã khách, nhóm khách, công nợ, tổng chi tiêu, số đơn, lịch sử giao
dịch, ghi chú, tìm nhanh lúc lập đơn, xuất/nhập CSV, lọc theo nhóm và theo
lần cuối ghé.

**Bỏ, và vì sao:**
- **Điểm tích luỹ.** Tiệm sửa chữa mỗi khách một năm vài lần; điểm không đủ
  tích để thành phần thưởng có nghĩa, mà thêm nó là thêm một cột số phải
  giải thích với khách. Thay bằng nhóm khách gán tay.
- **Ngày sinh, giới tính.** Không dùng vào việc gì ở đây. Thu thập dữ liệu
  cá nhân không dùng đến là gánh nợ, không phải tài sản.
- **Hạn mức nợ, chặn bán khi quá hạn mức.** KiotViet cần vì bán sỉ. Ở đây
  thợ nhìn mặt khách mà quyết, một cái ngưỡng cứng chỉ cản việc.
- **Nhiều chi nhánh, nhiều kho.** Một trạm.

### Thiết kế

`KhachHang` lưu trong `data/khach-hang.yaml`, một danh sách — cùng lối
`data/doi-tac.yaml`. Vài trăm khách là kịch trần của tiệm; ghi lại cả file
mỗi lần thêm vẫn dưới một phần nghìn giây.

```go
type KhachHang struct {
	Ma      string `yaml:"ma" json:"ma"`             // K0001
	Ten     string `yaml:"ten" json:"ten"`
	SoChuan string `yaml:"so_chuan" json:"so_chuan"` // chỉ chữ số, khoá tra
	LienHe  string `yaml:"lien_he" json:"lien_he"`   // nguyên văn thợ gõ
	Email   string `yaml:"email" json:"email"`
	DiaChi  string `yaml:"dia_chi" json:"dia_chi"`
	Nhom    string `yaml:"nhom" json:"nhom"`         // le | than | clb
	GhiChu  string `yaml:"ghi_chu" json:"ghi_chu"`
	LanDau  string `yaml:"lan_dau" json:"lan_dau"`   // ISO, ngày đơn đầu
	LanCuoi string `yaml:"lan_cuoi" json:"lan_cuoi"`
	Ngung   bool   `yaml:"ngung" json:"ngung"`       // ẩn khỏi ô chọn
}
```

Ba nhóm khai cứng trong code, không cho tự thêm:

```go
var CacNhomKhach = []NhomKhach{
	{"le",   "Khách lẻ",     "trung"},
	{"than", "Khách quen",   "nhan"},
	{"clb",  "CLB / đội nhóm", "cho"},
}
```

Ba là đủ để lọc mà vẫn nhớ được. Cho tự thêm nhóm thì sáu tháng nữa có mười
hai nhóm chồng nghĩa nhau và không lọc nổi cái gì.

**`SoChuan` là khoá tra, không phải `LienHe`.** Khách đọc số kiểu
`0846 161 368`, `+84846161368`, `084.616.1368` — ba cách gõ cùng một người.
`ChuanSo` bóc hết ký tự không phải chữ số, rồi `84` đầu chuỗi 11 số đổi về
`0`. Giữ nguyên `LienHe` để hiện đúng cái thợ đã gõ.

**Công nợ và tổng chi tiêu KHÔNG lưu thành trường.** Tính lúc đọc, cộng từ
các đơn của khách qua `Don.ConNo()` và `Don.DaThu()`. Đây là luật đã có ở
đầu `core/donhang.go`: một con số gõ tay ở đây cộng một sổ tiền ở kia là hai
nguồn sự thật, và ngày chúng lệch nhau thì bên sai luôn là bên người ta tin.

```go
type SoLieuKhach struct {
	SoDon      int
	SoDonDang  int // đang chạy
	TongChi    int // tổng DaThu của mọi đơn
	ConNo      int // tổng ConNo của mọi đơn
	LanCuoi    string
	NgayVang   int // số ngày kể từ lần cuối
}
func SoLieuCuaKhach(ma string) SoLieuKhach
```

**Khách sinh ra từ đơn, không bắt nhập tay trước.** Lưu đơn xong thì
`GhiNhanKhach` tra `SoChuan`: có rồi thì cập nhật `LanCuoi` và điền thêm
email/địa chỉ nếu trước bỏ trống (không ghi đè cái đã có); chưa có thì tạo
mới ở nhóm `le`. Thợ đang cầm cây vợt và một ông khách đứng đợi — không bắt
mở thêm một trang nữa trước khi lập được đơn.

**`Don.MaKhach`** là trường mới, rỗng với đơn cũ. `KhachTen`/`KhachLienHe`
trên đơn KHÔNG bỏ đi: đơn là chứng từ, nó phải đọc được đúng những gì đã ghi
lúc nhận, kể cả sau này khách đổi tên hay đổi số.

**Interfaces:**
- Sinh ra: `ChuanSo(s string) string`, `KhachTheoSo(so string) (*KhachHang, bool)`,
  `KhachTheoMa(ma string) (*KhachHang, bool)`,
  `GhiNhanKhach(ten, lienHe, email, diaChi string) string`,
  `LichSuKhach(ma string) []*Don`, `SoLieuCuaKhach(ma string) SoLieuKhach`,
  `LocKhach(f BoLocKhach) []DongKhach`.
- Việc 3 dùng `LichSuKhach` để hiện đơn bảo hành trên trang khách.

**Files:**
- Tạo: `core/khachhang.go`, `core/khachhang_test.go`
- Tạo: `core/qtkhachhang.go`
- Tạo: `core/ui/qt-khach.html`, `core/ui/qt-khach-ct.html`
- Sửa: `core/donhang.go` (thêm `MaKhach`)
- Sửa: `core/quantri.go` (gắn khách lúc lưu đơn, đăng ký route)
- Sửa: `core/xuatcsv.go` (xuất danh sách khách)
- Sửa: `core/ui/phan.html` (mục menu)
- Sửa: `core/ui/qt-don-moi.html`, `core/ui/qt-don-ct.html`

---

- [x] **Bước 1: Viết test cho `ChuanSo`**

```go
func TestChuanSo(t *testing.T) {
	cases := map[string]string{
		"0846161368":     "0846161368",
		"0846 161 368":   "0846161368",
		"084.616.1368":   "0846161368",
		"+84846161368":   "0846161368",
		"84846161368":    "0846161368",
		"  0846161368  ": "0846161368",
		"":               "",
		"zalo abc":       "",
	}
	for vao, mong := range cases {
		if ra := ChuanSo(vao); ra != mong {
			t.Errorf("ChuanSo(%q) = %q, mong %q", vao, ra, mong)
		}
	}
}
```

- [x] **Bước 2: Chạy để chắc là đỏ**

`go test ./core/ -run TestChuanSo` → FAIL, `undefined: ChuanSo`.

- [x] **Bước 3: Viết `core/khachhang.go` phần tối thiểu**

Đầu file phải có ghi chú giải thích: vì sao `SoChuan` là khoá tra chứ không
phải `LienHe`; vì sao khách sinh ra từ đơn; vì sao công nợ không lưu thành
trường; vì sao ba nhóm khai cứng.

```go
func ChuanSo(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	so := b.String()
	if strings.HasPrefix(so, "84") && len(so) == 11 {
		so = "0" + so[2:]
	}
	return so
}
```

- [x] **Bước 4: Chạy test, phải xanh**

- [x] **Bước 5: Viết test `GhiNhanKhach` gộp đúng người**

```go
func TestGhiNhanKhachGopTheoSo(t *testing.T) {
	t.Setenv("TRAMVOT_DATA", t.TempDir())
	NapKhach()
	a := GhiNhanKhach("Anh Minh", "0846 161 368", "", "")
	b := GhiNhanKhach("Minh", "+84846161368", "minh@x.vn", "Hà Nội")
	if a != b {
		t.Fatalf("hai cách gõ cùng một số phải ra một mã: %s vs %s", a, b)
	}
	k, ok := KhachTheoSo("0846161368")
	if !ok {
		t.Fatal("không tra được khách vừa ghi")
	}
	if k.Email != "minh@x.vn" || k.DiaChi != "Hà Nội" {
		t.Errorf("lần sau có thêm email/địa chỉ thì phải điền vào: %+v", k)
	}
	if k.Ten != "Anh Minh" {
		t.Errorf("tên đã có thì không ghi đè, nhận %q", k.Ten)
	}
	if k.Nhom != "le" {
		t.Errorf("khách mới phải vào nhóm lẻ, nhận %q", k.Nhom)
	}
}
```

- [x] **Bước 6: Chạy, phải đỏ. Rồi viết `GhiNhanKhach`, `KhachTheoSo`, `KhachTheoMa`, `NapKhach`, `LuuKhach`**

Theo đúng khuôn `core/doitac.go`: `sync.RWMutex`, nạp lúc khởi động, ghi
atomic ra `data/khach-hang.yaml`. Mã khách `K` + số thứ tự 4 chữ số.

- [x] **Bước 7: Chạy test, phải xanh. Commit.**

```bash
git add core/khachhang.go core/khachhang_test.go
git commit -m "feat(khach): hồ sơ khách hàng gộp theo số điện thoại chuẩn hoá

Co-Authored-By: Claude Opus 5 (1M context) <noreply@anthropic.com>"
```

- [x] **Bước 8: Test `SoLieuCuaKhach` cộng đúng công nợ**

```go
func TestSoLieuKhachCongNo(t *testing.T) {
	// khách có 2 đơn: đơn A 500k thu đủ, đơn B 300k thu 100k
	// → SoDon 2, TongChi 600000, ConNo 200000
}
```

- [x] **Bước 9: Chạy, phải đỏ. Viết `LichSuKhach` + `SoLieuCuaKhach`. Cho xanh.**

`LichSuKhach` lọc theo `MaKhach`; đơn cũ chưa có `MaKhach` thì đối chiếu
thêm bằng `ChuanSo(d.KhachLienHe)` — hai đơn thật đang có trên server không
mang trường mới, và người ta vẫn phải thấy chúng trên trang khách.

- [x] **Bước 10: Gắn vào luồng đơn**

Thêm `MaKhach string \`json:"ma_khach"\`` vào `Don` (`core/donhang.go:210`).
Trong `hQtDonMoi` và `hQtDonLuu` (`core/quantri.go`), sau khi đã có
`KhachTen`/`KhachLienHe`:

```go
don.MaKhach = GhiNhanKhach(don.KhachTen, don.KhachLienHe, don.KhachEmail, don.KhachDiaChi)
```

Đơn từ web (`hQtYeuCauTaoDon`) cũng phải gọi — khách gửi ảnh qua web cũng là
khách.

- [x] **Bước 11: Chạy `go test ./core/`, phải xanh. Commit.**

- [x] **Bước 12: Tra khách nhanh lúc lập đơn**

Trong `qt-don-moi.html`, nhúng `<datalist id="ds-khach">` dựng từ danh sách
khách chưa ngừng, mỗi option `value` là số, `label` là tên. Gắn vào ô
`khach_lien_he`.

Không dùng AJAX: vài trăm dòng nhúng thẳng vào HTML nhẹ hơn một vòng gọi
mạng, và nó chạy cả khi mạng ở tiệm chập chờn.

Thêm một đoạn JS ngắn (đặt trong `qt-don-moi.html`, không nhét vào tệp JS
chung): khi ô số mất focus, nếu số khớp một khách thì tự điền tên/email/địa
chỉ vào các ô còn TRỐNG — không đè lên thứ thợ đã gõ. Và hiện một dòng nhỏ
dưới ô:

`Khách quen · 4 đơn · còn nợ 200.000đ · lần cuối 12/07` + link sang trang khách.

Con số công nợ hiện ngay đây là thứ KiotViet làm đúng: người quyết định có
cho nợ tiếp hay không là thợ đang đứng đó, không phải cái báo cáo cuối tháng.

- [x] **Bước 13: Trang danh sách khách `/qt/khach`**

`canLaChu`. Route trong `core/qtkhachhang.go`.

Bốn ô số ở đầu trang: tổng khách, khách mới trong tháng, tổng còn nợ, số
khách đang nợ.

Bảng: mã, tên, số, nhóm (chip màu), số đơn, tổng chi, còn nợ, lần cuối, số
ngày vắng. Sắp mặc định theo lần cuối giảm dần.

Bộ lọc: ô tìm (tên/số/mã), chọn nhóm, và **"chưa quay lại quá N tháng"**
(mặc định 6). Cái lọc cuối là lý do chính khiến trang này đáng tồn tại — nó
cho Kendy một danh sách để nhắn Zalo.

- [x] **Bước 14: Xuất CSV danh sách khách**

Theo khuôn `core/xuatcsv.go`: `/qt/khach.csv` giữ nguyên bộ lọc đang xem.
Cột đúng như bảng. Có BOM UTF-8 để Excel không vỡ dấu (kiểm lại hàm
`guiCSV` đã làm chưa, nếu rồi thì dùng lại).

- [x] **Bước 15: Nhập CSV danh sách khách**

`POST /qt/khach/nhap`, nhận tệp CSV hai cột tối thiểu **tên, số điện thoại**
(thêm email/địa chỉ/nhóm/ghi chú nếu có). Đây là đường Kendy đổ danh bạ CLB
vào một lượt thay vì gõ tay trăm dòng.

Luật nhập:
- Trùng `SoChuan` thì **cập nhật, không tạo trùng** — điền vào ô đang trống,
  không đè ô đã có.
- Dòng thiếu số điện thoại thì bỏ qua và **đếm vào báo cáo**, không im lặng.
- Sau khi nhập hiện bảng tổng kết: thêm mới N, cập nhật M, bỏ qua K (kèm số
  dòng và lý do).
- **Không có nút hoàn tác** — nên trước khi ghi phải sao lưu
  `data/khach-hang.yaml` thành `khach-hang.yaml.truoc-nhap` để cứu được bằng
  tay nếu Kendy đổ nhầm tệp.

Test: một CSV 3 dòng trong đó 1 dòng trùng số và 1 dòng thiếu số → thêm 1,
cập nhật 1, bỏ qua 1.

- [x] **Bước 16: Trang một khách `/qt/khach/{ma}`**

Trên: tên, số, email, địa chỉ, nhóm, ghi chú — sửa được tại chỗ (form POST).
Đổi nhóm là một ô select lưu ngay.

Giữa: bốn ô số của riêng khách này — số đơn, tổng chi, còn nợ, ngày vắng.

Dưới: bảng toàn bộ đơn, mỗi dòng link sang đơn, có cột trạng thái và còn nợ.
Đơn còn bảo hành tô đậm (dùng ở Việc 3).

Nút **"Tạo đơn cho khách này"** → `/qt/don-moi?khach=<ma>` điền sẵn.

- [x] **Bước 17: Ô "Khách" trên trang đơn thành link**

Ở `qt-don-ct.html`, nếu đơn có `MaKhach` thì tên khách là link sang
`/qt/khach/{ma}`, kèm chip nhóm. Không có `MaKhach` thì hiện nút nhỏ "Gắn vào
hồ sơ khách" gọi `GhiNhanKhach` cho đơn ấy.

- [x] **Bước 18: Thêm mục menu**

`core/ui/phan.html`, nhóm "Chủ trạm", đặt ngay dưới "Quản lý đơn":

```html
<a href="/qt/khach" {{if batdau .Trang "khach"}}class="day"{{end}}>Khách hàng</a>
```

- [x] **Bước 19: Test trang mở được**

Thêm `/qt/khach`, `/qt/khach.csv` vào danh sách đường dẫn của
`core/trangmo_test.go` và `core/qtdonkhu_test.go` (trang chỉ-chủ, thợ vào
phải bị đuổi).

- [x] **Bước 20: `go test ./core/`, build, deploy, commit**

---

### Việc 1b: Khách cũ quay lại tự giảm giá

**Kendy chốt:** khách cũ quay lại thì tự động giảm 10%.

**Tin tốt: không phải làm cơ chế mới.** Giảm giá đã có đủ —
`core/quantri.go:584` đọc ô `giam_gia` + `giam_ly_do`, đẩy vào `DongTien`
một dòng âm mang mã `MaGiamGia = "@giam-gia"`, kẹp không cho vượt tổng
(`core/donhang.go:630`), và `core/thongke.go:122` đã biết bỏ qua dòng ấy khi
đếm doanh thu theo dịch vụ. Hoá đơn và mail báo giá cũng tự in ra vì nó là
một dòng tiền bình thường.

Việc còn lại chỉ là **tự điền sẵn hai ô đó**.

#### Thiết kế

**(a) Mức giảm gắn với NHÓM khách, không phải một luật riêng.** Việc 1 đã có
ba nhóm; giờ mỗi nhóm mang một mức phần trăm. Một cơ chế thay vì hai — sáu
tháng nữa Kendy muốn cho CLB mức khác thì đổi một con số, không sửa code.

Hai ngưỡng mới trong `core/nguong.go`, đơn vị `%`:

```go
{Khoa: "giam_khach_quen_phan_tram", Don: "%", Mac: 10}
{Khoa: "giam_clb_phan_tram",        Don: "%", Mac: 15}
```

Khách lẻ 0% — không khai ngưỡng, mặc định là không giảm.

**(b) Tự thăng nhóm, không tự hạ.** Sau khi một đơn chuyển sang `TTDaGiao`,
nếu khách đang ở nhóm `le` thì tự lên `than`. Điều kiện để tính là "đã từng
tới":

- Đơn phải ở `TTDaGiao`. `TTHuy` và `TTTuChoi` không tính — khách huỷ không
  phải khách cũ.
- **Đơn bảo hành (`MaDonGoc != ""`, xem Việc 3) không tính.** Khách quay lại
  vì đồ hỏng lại thì đó không phải lần ghé thứ hai, đó là lần thứ nhất chưa
  xong.

Không tự hạ nhóm bao giờ. Khách vắng hai năm rồi quay lại vẫn là khách quen;
hạ nhóm là cách nhanh nhất làm mất một người vừa mới quay về.

Không tự đẩy ai lên `clb` — nhóm đó gán tay.

**(c) Điền sẵn, không tự áp.** Khi mở trang đơn của một khách có mức giảm,
ô `giam_gia` điền sẵn `làm tròn(tổng dòng tiền × mức%)` và ô `giam_ly_do`
điền sẵn `Khách quen`. Thợ xoá được.

Đây vẫn đúng nghĩa "auto" — thợ không phải nhớ, không phải bấm máy tính —
nhưng cứu được ca thật: khách quay lại vì đơn trước trạm làm hỏng. Giảm 10%
cho ca ấy là nói sai chuyện. Và nó đi đúng luật đã ghi ở đầu
`core/tudong.go`: *máy đoán, người xác nhận*.

**(d) Trần theo biên gộp — chỗ nguy hiểm nhất.** Đơn thay đế gửi tiệm ngoài:
thu khách 500k, trả tiệm 400k, trạm ăn 100k. Giảm 10% trên tổng là mất 50k,
tức một nửa phần công. Vài đơn như thế là làm không công.

Nên: nếu đơn đã có `GuiDi.TraDoiTac > 0` và mức giảm làm
`TongTien − TraDoiTac` tụt dưới ngưỡng `bien_gop_toi_thieu_dong` (đã có sẵn
trong `core/nguong.go`), thì **hạ mức giảm xuống vừa đủ chạm ngưỡng**, và
hiện ngay cạnh ô một dòng nhỏ nói rõ vì sao:

`Đã hạ từ 50.000đ xuống 20.000đ — đơn này gửi tiệm ngoài, giảm thêm là dưới biên tối thiểu.`

Nói ra chứ không lặng lẽ sửa số: thợ phải hiểu con số ở đâu ra, nếu không
lần sau họ gõ đè lên và cái trần thành vô dụng.

**(e) Đơn bảo hành không giảm.** Tổng 0, giảm 10% của 0 vẫn là 0, nhưng in
một dòng "Giảm khách quen 0đ" lên hoá đơn là chữ rác. Bỏ qua hẳn.

**(f) Không tự quảng cáo trên web.** "Khách cũ giảm 10%" là quyết định
marketing, không phải hệ quả kỹ thuật. Để Kendy tự viết vào CMS nếu muốn —
đừng tự thêm khoá ND rồi bày lên trang chủ.

**Interfaces:**
- Sinh ra: `MucGiamCuaKhach(maKhach string) int` (phần trăm, 0 nếu không có),
  `GoiYGiamGia(d *Don) (soTien int, lyDo string, daHa bool)`,
  `ThangNhomSauGiao(maKhach string)`.

**Files:**
- Sửa: `core/khachhang.go`, `core/khachhang_test.go`
- Sửa: `core/nguong.go`
- Sửa: `core/quantri.go` (gọi `ThangNhomSauGiao` trong `hQtDoiTrangThai`)
- Sửa: `core/ui/qt-don-ct.html` (điền sẵn hai ô + dòng giải thích)

---

- [x] **Bước 21: Test `MucGiamCuaKhach` theo nhóm**

```go
func TestMucGiamTheoNhom(t *testing.T) {
	// nhóm le  → 0
	// nhóm than → 10
	// nhóm clb  → 15
	// mã khách rỗng → 0
}
```

- [x] **Bước 22: Chạy, phải đỏ. Thêm hai ngưỡng vào `core/nguong.go`, viết hàm. Cho xanh.**

- [x] **Bước 23: Test tự thăng nhóm**

```go
func TestThangNhomSauKhiGiao(t *testing.T) {
	// khách nhóm le, đơn chuyển sang TTDaGiao → lên than
	// khách nhóm le, đơn TTHuy → vẫn le
	// đơn bảo hành (MaDonGoc != "") đã giao → vẫn le
	// khách nhóm clb, đơn đã giao → vẫn clb, không bị kéo xuống than
}
```

- [x] **Bước 24: Chạy, phải đỏ. Viết `ThangNhomSauGiao`, gọi từ `hQtDoiTrangThai`. Cho xanh.**

- [x] **Bước 25: Test trần biên gộp — đây là test quan trọng nhất của Việc 1b**

```go
func TestGoiYGiamGiaBiTranBienGop(t *testing.T) {
	// ngưỡng biên tối thiểu 80k
	// đơn: tổng 500k, trả tiệm 400k, khách nhóm than (10%)
	// 10% = 50k → còn biên 50k, dưới ngưỡng
	// → soTien == 20000 (biên còn đúng 80k), daHa == true

	// đơn: tổng 500k, không gửi tiệm, nhóm than
	// → soTien == 50000, daHa == false

	// đơn bảo hành → soTien == 0
}
```

- [x] **Bước 26: Chạy, phải đỏ. Viết `GoiYGiamGia`. Cho xanh.**

- [x] **Bước 27: Điền sẵn hai ô trên trang đơn + dòng giải thích khi bị hạ**

Chỉ điền khi ô `giam_gia` đang trống — đơn đã lưu một mức giảm rồi thì không
đè lên.

- [x] **Bước 28: Kiểm bằng mắt một đơn thật**

Render trang đơn bằng Chrome headless (CDP `Emulation.setDeviceMetricsOverride`),
Read ảnh. Kiểm: số điền sẵn đúng, dòng giải thích không tràn, hoá đơn in ra
có dòng "Giảm khách quen".

- [x] **Bước 29: `go test ./core/`, build, deploy, commit**



## Việc 2: Tem dán lên đồ

**Vì sao:** hai chục cây vợt đen giống hệt nhau trên giá. Hoá đơn A4 không
dán lên cán vợt được. Đây là lỗi đắt nhất của tiệm sửa chữa: trả nhầm đồ.

**Files:**
- Tạo: `core/tem.go`
- Tạo: `core/ui/qt-tem.html`
- Sửa: `core/quantri.go` (route)
- Sửa: `core/ui/qt-don-ct.html` (nút in tem)
- Sửa: `core/ui/qt-don.html` (in tem hàng loạt)

### Thiết kế

Đi đúng lối `core/hoadon.go`: **trang HTML + Ctrl+P**, không nhét thư viện
PDF. Khác hoá đơn ở chỗ khổ giấy — dùng `@page { size: 50mm 30mm; margin: 0 }`
cho máy in tem nhiệt, và một bản dự phòng xếp 24 tem trên một tờ A4 để cắt
tay khi chưa mua máy in tem.

Trên tem chỉ bốn thứ, không hơn:
- **Mã đơn** — to nhất, đây là thứ dùng để tra.
- **Tên khách + 4 số cuối điện thoại** — để đối chiếu bằng mắt khi khách tới.
- **Hẹn trả** — để nhìn kệ là biết cây nào gấp.
- **Tên món** (`d.TenMon()`) — vợt Yonex / giày Nike size 42.

Không in QR: máy quét thì tiệm chưa có, mà điện thoại quét xong vẫn phải mở
trình duyệt đăng nhập — chậm hơn đọc mã bằng mắt.

- [x] **Bước 1: Viết test route trả 200 và có mã đơn trong thân**

Theo khuôn `core/qtdonkhu_test.go`.

```go
func TestTemCoMaDon(t *testing.T) {
	// dựng một đơn, gọi /qt/don/{ma}/tem, kiểm tra body chứa d.Ma
}
```

- [x] **Bước 2: Chạy, phải đỏ (404)**

- [x] **Bước 3: Viết `core/tem.go` + `qt-tem.html`, đăng ký route**

```go
mux.HandleFunc("GET /qt/don/{ma}/tem", canDangNhap(hQtTem))
mux.HandleFunc("GET /qt/tem", canDangNhap(hQtTemNhieu)) // ?ma=A&ma=B
```

Đầu `core/tem.go` ghi rõ vì sao khổ 50×30mm và vì sao không có QR.

- [x] **Bước 4: Chạy test, phải xanh**

- [x] **Bước 5: CSS in**

Trong `qt-tem.html` đặt `<style>` riêng, không đụng `css.html`. Phải có
`@media print { ... }` giấu mọi thứ trừ tem, và `@page`. Font tối thiểu 9pt —
nhỏ hơn thì máy in nhiệt nhoè.

- [x] **Bước 6: Nút "In tem" trên trang đơn**

Cạnh nút "In hoá đơn" ở `qt-don-ct.html:13`, `target="_blank"`.

- [x] **Bước 7: In hàng loạt**

Trên `qt-don.html`, thêm ô tick mỗi dòng + nút "In tem các đơn đã chọn" gửi
sang `/qt/tem?ma=...&ma=...`. Dùng cho buổi sáng nhận một lượt năm cây.

- [x] **Bước 8: Tự nhìn hình trước khi deploy**

Render bằng Chrome headless (CDP `Emulation.setDeviceMetricsOverride`, KHÔNG
dùng `--window-size` — trên Windows nó nói dối) rồi Read ảnh. Kiểm: mã đơn
không tràn, tên dài bị cắt gọn chứ không đẩy dòng.

- [x] **Bước 9: `go test ./core/`, build, deploy, commit**

---

## Việc 3: Bảo hành có gốc

**Vì sao:** `BaoHanhDen` hiện là một ô ngày gõ tay (`core/donhang.go:261`).
Không có mặc định theo dịch vụ, không có danh sách đang còn hạn, và khách
quay lại bảo hành thì phải mở đơn mới không gắn được vào đơn cũ — thống kê
đọc ra một đơn 0đ chứ không phải một khoản chi phí bảo hành. Chưa biết tỷ lệ
phải làm lại là bao nhiêu thì không biết giá đang đặt đúng hay sai.

**Files:**
- Sửa: `core/dichvu` — nơi khai dịch vụ (tìm bằng `grep -n "BaoHanhThang\|type DichVu" core/*.go`)
- Sửa: `core/donhang.go` (thêm `MaDonGoc`)
- Sửa: `core/quantri.go`
- Sửa: `core/thongke.go`
- Sửa: `core/ui/qt-dich-vu.html`, `qt-don-ct.html`, `qt-don-moi.html`
- Tạo: `core/baohanh_test.go`

### Thiết kế

Ba phần, làm theo thứ tự:

**(a) Mặc định theo dịch vụ.** Thêm `BaoHanhThang int` vào bản ghi dịch vụ,
sửa ở `/qt/dich-vu`. Lúc lưu đơn, nếu ô `bao_hanh_den` để trống thì tính:
ngày giao + số tháng LỚN NHẤT trong các dịch vụ đã chốt của đơn. Lớn nhất
chứ không nhỏ nhất: khách không phân biệt được món nào bảo hành mấy tháng,
mà hứa ngắn rồi từ chối sửa lại là mất khách.

Vẫn cho gõ đè bằng tay. Ô gõ tay thắng mặc định, luôn luôn.

**(b) Đơn bảo hành gắn với đơn gốc.** Thêm `MaDonGoc string` vào `Don`. Trên
trang một đơn, thêm nút **"Mở đơn bảo hành"** — dựng đơn mới đã điền sẵn
khách, món, và `MaDonGoc` trỏ về đơn cũ. Đơn bảo hành có `TongTien` 0.

**(c) Thống kê đọc được.** Trong `ThongKeKy` thêm:

```go
DonBaoHanh   int // đơn có MaDonGoc, nhận trong kỳ
TyLeBaoHanh  int // DonBaoHanh / NhanDon, phần trăm
```

và **loại đơn bảo hành khỏi `TBMotDon`** — một đơn 0đ kéo trung bình xuống
làm con số vô nghĩa.

Trên trang khách (Việc 1), đơn bảo hành hiện chip riêng.

- [x] **Bước 1: Test mặc định bảo hành lấy số tháng lớn nhất**

```go
func TestBaoHanhLayThangLonNhat(t *testing.T) {
	// đơn có 2 dịch vụ: 1 tháng và 3 tháng, giao 2026-09-12
	// → BaoHanhDen == "2026-12-12"
}
```

- [x] **Bước 2: Chạy, phải đỏ. Viết hàm, cho xanh.**

- [x] **Bước 3: Test ô gõ tay thắng mặc định. Cho xanh.**

- [x] **Bước 4: Thêm ô "Bảo hành (tháng)" vào `/qt/dich-vu`**

- [x] **Bước 5: Test đơn bảo hành không kéo `TBMotDon`. Cho xanh.**

- [x] **Bước 6: Nút "Mở đơn bảo hành" + trường `MaDonGoc`**

Trên đơn gốc hiện dòng "Đã mở N đơn bảo hành"; trên đơn bảo hành hiện link
ngược về gốc. Hai chiều — nhìn đơn nào cũng thấy được chuyện.

- [x] **Bước 7: Rổ "Còn bảo hành" trên danh sách đơn**

Thêm `ConBaoHanh bool` vào `BoLoc` (`core/donhang.go:538`) và một nút lọc
cạnh "Chờ hàng về" ở `qt-don.html:9`.

- [x] **Bước 8: `go test ./core/`, build, deploy, commit**

---

## Việc 4: Rổ đơn xong bị bỏ quên

**Vì sao:** tin nhắc hằng ngày (`core/thongbao.go:241`) chỉ nhìn
`HenTraNgay`. Khách hẹn "mai qua lấy" rồi biến mất ba tuần thì không có gì
kêu. Vừa là tiền chưa thu, vừa là chỗ trên kệ.

**Files:**
- Sửa: `core/donhang.go` (`BoLoc`, một hàm phụ trên `Don`)
- Sửa: `core/thongbao.go`
- Sửa: `core/nguong.go` (ngưỡng số ngày)
- Sửa: `core/ui/qt-don.html`
- Tạo/sửa: test tương ứng

### Thiết kế

Đơn ở trạng thái `TTXong` mà `CapNhat` đã quá N ngày. N khai ở `/qt/nguong`,
mặc định **7**. Đặt vào `nguong.go` chứ không hằng số cứng: tiệm đông thì 3
ngày đã chật kệ, tiệm vắng thì 14 ngày mới đáng gọi.

Đo bằng `CapNhat` chứ không phải `HenTraNgay`: nhiều đơn không có ngày hẹn,
mà lúc bấm sang "Sửa xong" thì `CapNhat` luôn được ghi.

Gộp vào **đúng cái tin nhắc hằng ngày đã có**, thành mục thứ ba sau "QUÁ
HẸN" và "Hẹn trả hôm nay". Không đẻ tin mới: `core/thongbao.go` đã ghi rõ lý
do — báo nhiều thì tới hôm thứ ba Kendy tắt thông báo và mất luôn cả ba việc.

- [x] **Bước 1: Test `XongBoQuen`**

```go
func TestXongBoQuen(t *testing.T) {
	// đơn TTXong, CapNhat 10 ngày trước, ngưỡng 7 → true
	// đơn TTXong, CapNhat 2 ngày trước → false
	// đơn TTDaGiao, CapNhat 30 ngày trước → false
}
```

- [x] **Bước 2: Chạy, phải đỏ. Viết `func (d Don) XongBoQuen(bayGio time.Time, nguongNgay int) bool`. Cho xanh.**

- [x] **Bước 3: Thêm ngưỡng `xong_bo_quen_ngay` vào `core/nguong.go`**

Mặc định 7, `Don: "ngày"`. Không phải `Web: true` — đây là chuyện nội bộ,
khách không cần biết.

- [x] **Bước 4: Test tin nhắc có mục thứ ba. Cho xanh.**

Sửa `BaoDonToiHen` gom thêm `boQuen []string`, tiêu đề cộng thêm
`"N đơn xong chưa ai lấy"`.

- [x] **Bước 5: Nút lọc trên danh sách đơn**

`ChiXongBoQuen bool` trong `BoLoc`, nút "Xong, chưa ai lấy" cạnh hai nút sẵn
có ở `qt-don.html:9-10`.

- [x] **Bước 6: `go test ./core/`, build, deploy, commit**

---

## Việc 5 (mục 6 trong review): Giá vốn vật tư vào con số lãi

**Vì sao:** `ThongKeKy.Lai = DoanhThu - TraDoiTac` (`core/thongke.go:52`)
không trừ vật tư đã xuất kho cho chính mấy đơn ấy, dù kho đã ghi đủ dữ liệu
(`PhieuXuatCuaDon` ở `core/kho.go:539`, `GiaNhapGanNhat` ở `core/kho.go:436`).
Sổ tiền mới ra con số thật. Hai bảng lệch nhau là chỗ dễ tin nhầm nhất.

**Files:**
- Sửa: `core/kho.go` (thêm một hàm)
- Sửa: `core/thongke.go`
- Sửa: `core/ui/qt-thongke.html`
- Sửa: `core/thongke_test.go` (hoặc tạo)

### Thiết kế

Phiếu xuất cố tình không mang `DonGia` (`core/kho.go:87-91`) — sổ tiền tính
theo tiền mặt, tiền ra lúc MUA chứ không phải lúc DÙNG. Giữ nguyên luật ấy;
chỗ này chỉ ước lượng để so sánh giữa các dịch vụ, nên lấy `GiaNhapGanNhat`.

Đổi tên trường cho hết mập mờ: `Lai` → **`LaiGop`**, và thêm
**`VatTu`** (giá vốn ước tính) với **`LaiSauVatTu`**.

```go
VatTu       int // ước tính theo giá nhập gần nhất
LaiGop      int // DoanhThu - TraDoiTac
LaiSauVatTu int // LaiGop - VatTu
```

Trên `qt-thongke.html` in cả ba, kèm một dòng nhỏ nói rõ: *"Giá vốn vật tư
là ước tính theo giá nhập gần nhất. Con số tiền mặt thật nằm ở Sổ tiền."*
Một dòng ấy là thứ ngăn Kendy tin nhầm hai bảng.

- [x] **Bước 1: Test `GiaVonDon`**

```go
func TestGiaVonDon(t *testing.T) {
	// nhập 10 cái viền giá 20k, xuất 2 cái cho đơn X
	// → GiaVonDon("X") == 40000
	// đơn không có phiếu xuất → 0
}
```

- [x] **Bước 2: Chạy, phải đỏ. Viết `func GiaVonDon(maDon string) int` trong `core/kho.go`. Cho xanh.**

Cộng `SoLuong × GiaNhapGanNhat(MaVatTu)` trên các dòng của phiếu xuất.

- [x] **Bước 3: Test `LaiSauVatTu`**

```go
func TestLaiSauVatTu(t *testing.T) {
	// đơn giao trong kỳ: 500k, trả tiệm 100k, vật tư 40k
	// → LaiGop 400k, VatTu 40k, LaiSauVatTu 360k
}
```

- [x] **Bước 4: Chạy, phải đỏ. Sửa `LayThongKeKy`. Cho xanh.**

Đổi tên `Lai` → `LaiGop` phải sửa cả `core/ui/qt-thongke.html` và
`core/xuatcsv.go` nếu có dùng — `grep -rn "\.Lai\b" core/` trước khi đổi.

- [x] **Bước 5: Sửa template + thêm dòng cảnh báo ước tính**

- [x] **Bước 6: Thêm cột vào CSV thống kê (`core/xuatcsv.go:133`)**

- [x] **Bước 7: `go test ./core/`, build, deploy, commit**

---

## Sau khi xong cả năm việc

- [x] `go test ./core/` toàn xanh
- [x] Render `/app` ở 390×844 và 360×500, kiểm `scrollHeight == innerHeight`
      — Việc 1 và 3 không đụng `/app` nhưng Việc 2 có sửa `qt-don.html`, và
      mọi thay đổi CSS đều phải đo lại
- [x] Báo Kendy danh sách những thứ cần gõ tay: số tháng bảo hành cho từng
      dịch vụ ở `/qt/dich-vu`, ngưỡng ngày bỏ quên ở `/qt/nguong`
