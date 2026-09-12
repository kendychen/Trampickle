# Bản Online: cửa hàng dịch vụ — kế hoạch thi công

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Site có công tắc chế độ Tại xưởng ⇄ Online, và một trang `/cua-hang` cho khách đặt dịch vụ sửa từ xa, đẻ ra đơn thật ngay khi đặt.

**Architecture:** Cửa hàng là *mặt tiền mới của luồng đơn đã có*, không phải hệ thứ hai. Đặt online tạo `Don` ở trạng thái mới `cho_hang_ve`; từ khi kiện tới, đơn chảy nguyên máy trạng thái cũ. Chế độ nội dung đi qua hệ CMS đã có bằng hậu tố khoá `@online` đăng ký trong chính `ndMac`, không tạo cây khoá thứ hai.

**Tech Stack:** Go 1.26.1, `gopkg.in/yaml.v3`, `html/template`, `net/http` (`http.ServeMux` với pattern method), test bằng `go test` chuẩn thư viện.

**Spec:** `docs/superpowers/specs/2026-09-10-cua-hang-online-design.md`

## Global Constraints

- **Không thêm dependency.** Chỉ `gopkg.in/yaml.v3` như hiện tại.
- **Build cho VPS:** `CGO_ENABLED=0`. Không đưa gì cần cgo vào.
- **Một khoá nội dung khai MỘT lần** ở `CayND`. Bản Online khai bằng trường `MacOn` trên chính mục đó, không gõ khoá `@online` bằng tay ở đâu cả — `init()` sinh nó.
- **Không thêm trường "đã thu" hay trạng thái "chờ thanh toán".** Tiền đọc từ sổ qua `Don.DaThu()` / `Don.ConNo()`. Sổ tiền là nguồn sự thật duy nhất cho tiền mặt.
- **Giá:** trang khách chỉ hiện **"giá từ …"**. Số lấy từ `DichVu.GiaTheoGiaiDoan(GiaiDoan)` đọc từ `vanhanh/bang-gia.yaml`; `BaoGiaRieng` thì không in số nào. Không thêm trường giá mới ở đâu khác.
- **Trường mới trên `Don` phải chịu được đơn cũ thiếu trường**: giá trị rỗng luôn có nghĩa mặc định (`NguonDon` rỗng = `tay`).
- **CSS:** không sửa bảng màu, không thêm biến màu gốc. Style riêng của cửa hàng đặt dưới tiền tố `body.la-cua-hang`, theo đúng tiền lệ `body.la-app`.
- **Không dùng escape CSS** kiểu `\203A` trong `css.html` — `html/template` nuốt thành ký tự rác. Dán ký tự thẳng hoặc vẽ bằng viền.
- **Mọi chữ hiện cho khách phải đi qua `{{nd}}` / `{{ndm}}`**, không hardcode trong template. `TestKhoaNDCoDu` bắt khoá gọi mà cây không có.
- **Ghi tệp dữ liệu dùng `ghiAtomic`** (đã dùng ở `DatLienHe`, `LuuDon`, `ghiND`). Không gọi `os.WriteFile` thẳng.
- Thư mục gốc dữ liệu là biến gói `Root` (`core/config.go:192`), đường dẫn ghép bằng `P(rel)` (`core/config.go:277`). Test đổi gốc bằng `Root = t.TempDir()` rồi trả lại.
- Chạy toàn bộ test: `go test ./...` trong `d:/TramVot/tramvot`.

---

## Bản đồ tệp

| Tệp | Trách nhiệm | Việc |
|---|---|---|
| `core/chedo.go` | Đọc/ghi chế độ site trong `data/giao-dien.yaml` | **Tạo** |
| `core/chedo_test.go` | Test chế độ | **Tạo** |
| `core/noidung.go` | `MacOn`, `ND()` chọn theo chế độ | Sửa |
| `core/noidung_cay.go` | Khai `MacOn`; thêm trang nội dung `cua-hang` | Sửa |
| `core/noidung_test.go` | Test hai chế độ | Sửa |
| `core/quantri.go` | Công tắc ở `/qt/giao-dien`; nút nhận hàng; lọc đơn | Sửa |
| `core/qtnoidung.go` | Admin nội dung sửa đúng bản đang bật | Sửa |
| `core/ui/qt-giaodien.html` | Ô chọn chế độ | Sửa |
| `core/ui/qt-noidung.html` | Nhãn chế độ đang sửa | Sửa |
| `core/donhang.go` | Trạng thái `cho_hang_ve`, 5 trường mới, lọc còn nợ | Sửa |
| `core/cuahang.go` | Luồng đặt hàng công khai, khách báo vận đơn | **Tạo** |
| `core/cuahang_test.go` | Test luồng đặt hàng | **Tạo** |
| `core/ui/cua-hang.html` | Trang đặt hàng | **Tạo** |
| `core/ui/cua-hang-xong.html` | Màn kết sau khi đặt | **Tạo** |
| `core/anhdon.go` | `luuAnhVaoDon` tách từ `hQtThemAnh` | **Tạo** |
| `core/vietqr.go` | Chuỗi VietQR mang mã đơn | **Tạo** |
| `core/vietqr_test.go` | Test QR khớp `reMaDon` | **Tạo** |
| `core/server.go` | Route `/cua-hang`, `/tra-cuu/{token}`, tiêu đề trang | Sửa |
| `core/ui/tracuu.html` | Khối thanh toán, ô báo vận đơn | Sửa |
| `core/ui/css.html` | Style `body.la-cua-hang` | Sửa |

---

## Task 1: Chế độ site — đọc và ghi

**Files:**
- Create: `core/chedo.go`, `core/chedo_test.go`
- Modify: `core/server.go` (chỗ khởi động gọi `NapND()`, `NapLienHe()`)

**Interfaces:**
- Consumes: `Root`, `P(rel)`, `ghiAtomic`
- Produces: `const CheDoTaiXuong = "tai_xuong"`, `const CheDoOnline = "online"`, `func NapCheDo() error`, `func CheDoHienTai() string`, `func DatCheDo(ma string) error`, `func CheDoHopLe(ma string) bool`, `func LaOnline() bool`

- [ ] **Step 1: Viết test thất bại**

Tạo `core/chedo_test.go`:

```go
package core

import (
	"os"
	"path/filepath"
	"testing"
)

// gocTam đổi thư mục gốc dữ liệu sang thư mục tạm rồi trả lại như cũ.
func gocTam(t *testing.T) string {
	t.Helper()
	cu := Root
	Root = t.TempDir()
	if err := os.MkdirAll(P("data"), 0o755); err != nil {
		t.Fatal(err)
	}
	d := Root
	t.Cleanup(func() { Root = cu; _ = NapCheDo() })
	return d
}

func TestCheDoMacDinhLaTaiXuong(t *testing.T) {
	gocTam(t)
	if err := NapCheDo(); err != nil {
		t.Fatalf("NapCheDo: %v", err)
	}
	if got := CheDoHienTai(); got != CheDoTaiXuong {
		t.Fatalf("chưa có file thì phải là %q, nhận %q", CheDoTaiXuong, got)
	}
	if LaOnline() {
		t.Fatal("LaOnline phải false khi đang tại xưởng")
	}
}

func TestDatCheDoGhiRoiNapLai(t *testing.T) {
	gocTam(t)
	if err := DatCheDo(CheDoOnline); err != nil {
		t.Fatalf("DatCheDo: %v", err)
	}
	if !LaOnline() {
		t.Fatal("đặt online xong LaOnline phải true")
	}
	if err := NapCheDo(); err != nil {
		t.Fatalf("NapCheDo lần 2: %v", err)
	}
	if got := CheDoHienTai(); got != CheDoOnline {
		t.Fatalf("nạp lại phải ra %q, nhận %q", CheDoOnline, got)
	}
}

func TestDatCheDoLaBiTuChoi(t *testing.T) {
	gocTam(t)
	if err := DatCheDo("bay-gio"); err == nil {
		t.Fatal("chế độ lạ phải bị từ chối")
	}
	if got := CheDoHienTai(); got != CheDoTaiXuong {
		t.Fatalf("từ chối xong phải giữ %q, nhận %q", CheDoTaiXuong, got)
	}
}

// File cũ chỉ có khoá chết "theme: tpic" — đọc nó không được nổ, và phải ra
// tại xưởng.
func TestFileCuChiCoKhoaTheme(t *testing.T) {
	d := gocTam(t)
	if err := os.WriteFile(filepath.Join(d, "data", "giao-dien.yaml"), []byte("theme: tpic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := NapCheDo(); err != nil {
		t.Fatalf("file cũ không phải lỗi: %v", err)
	}
	if got := CheDoHienTai(); got != CheDoTaiXuong {
		t.Fatalf("phải ra %q, nhận %q", CheDoTaiXuong, got)
	}
}

// File hỏng trả lỗi để chỗ gọi ghi nhật ký, nhưng web vẫn lên ở tại xưởng.
func TestFileHongThiChayTaiXuong(t *testing.T) {
	d := gocTam(t)
	if err := os.WriteFile(filepath.Join(d, "data", "giao-dien.yaml"), []byte("che_do: [hong"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := NapCheDo(); err == nil {
		t.Fatal("file hỏng phải trả lỗi")
	}
	if got := CheDoHienTai(); got != CheDoTaiXuong {
		t.Fatalf("file hỏng vẫn phải chạy %q, nhận %q", CheDoTaiXuong, got)
	}
}
```

- [ ] **Step 2: Chạy test cho thấy nó fail**

Run: `go test ./core/ -run TestCheDo -v`
Expected: FAIL — `undefined: NapCheDo`

- [ ] **Step 3: Viết `core/chedo.go`**

```go
// Chế độ site: khách mang đồ tới xưởng, hay khách gửi đồ tới.
//
// Một công tắc, đổi ở /qt/giao-dien. Nó KHÔNG đổi màu, không đổi logo, không
// đổi bố cục — chỉ đổi giọng của chữ và bật lối vào cửa hàng. Gộp chuyện màu
// vào đây là dựng lại hệ theme đã cố tình xoá hồi làm Material 3.
//
// data/giao-dien.yaml từng giữ khoá "theme" của hệ giao diện cũ. Khoá ấy chết
// từ khi bỏ hệ theme; đọc file này phải bước qua nó mà không vấp.
package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

const (
	CheDoTaiXuong = "tai_xuong"
	CheDoOnline   = "online"
)

type tepGiaoDien struct {
	CheDo string `yaml:"che_do"`
}

var (
	cheDoMu  sync.RWMutex
	cheDoNay = CheDoTaiXuong
)

func fileGiaoDien() string { return P("data/giao-dien.yaml") }

func CheDoHopLe(ma string) bool { return ma == CheDoTaiXuong || ma == CheDoOnline }

// NapCheDo đọc công tắc. Chưa có file không phải lỗi. File hỏng TRẢ lỗi nhưng
// vẫn để chế độ ở tại xưởng — chỗ gọi ghi nhật ký, web vẫn lên.
func NapCheDo() error {
	cheDoMu.Lock()
	cheDoNay = CheDoTaiXuong
	cheDoMu.Unlock()

	b, err := os.ReadFile(fileGiaoDien())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var t tepGiaoDien
	if err := yaml.Unmarshal(b, &t); err != nil {
		return fmt.Errorf("%s hỏng: %w", fileGiaoDien(), err)
	}
	// Khoá trống hoặc giá trị lạ (kể cả file cũ chỉ có "theme"): chạy tại
	// xưởng. Không đoán.
	if !CheDoHopLe(t.CheDo) {
		return nil
	}
	cheDoMu.Lock()
	cheDoNay = t.CheDo
	cheDoMu.Unlock()
	return nil
}

func CheDoHienTai() string {
	cheDoMu.RLock()
	defer cheDoMu.RUnlock()
	return cheDoNay
}

func LaOnline() bool { return CheDoHienTai() == CheDoOnline }

// DatCheDo ghi ra đĩa TRƯỚC rồi mới đổi trong bộ nhớ. Ghi hỏng thì chế độ
// đang chạy không đổi — tránh cảnh admin thấy báo đã đổi mà lần khởi động sau
// quay về cũ.
func DatCheDo(ma string) error {
	if !CheDoHopLe(ma) {
		return fmt.Errorf("chế độ không hợp lệ: %q", ma)
	}
	noiDung := "# Chế độ site, đổi ở /qt/giao-dien.\n" +
		"# tai_xuong: khách mang đồ tới. online: khách gửi đồ tới.\n\n" +
		"che_do: " + ma + "\n"
	if err := ghiAtomic(fileGiaoDien(), []byte(noiDung)); err != nil {
		return err
	}
	cheDoMu.Lock()
	cheDoNay = ma
	cheDoMu.Unlock()
	return nil
}
```

- [ ] **Step 4: Chạy test cho thấy pass**

Run: `go test ./core/ -run TestCheDo -v`
Expected: PASS cả 5 test

- [ ] **Step 5: Gọi `NapCheDo()` lúc khởi động**

Run: `grep -n "NapND()\|NapLienHe()" core/server.go`

Thêm `NapCheDo()` cạnh chúng, theo đúng lối xử lý lỗi của hàm bên cạnh (ghi nhật ký, không dừng server).

- [ ] **Step 6: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add core/chedo.go core/chedo_test.go core/server.go
git commit -m "feat: cong tac che do tai xuong / online"
```

---

## Task 2: `MacOn` — bản chữ Online trong cây nội dung

**Files:**
- Modify: `core/noidung.go` — `MucND`, `init()`, `ND`, `DaSuaND`, `SoDaSuaND`
- Test: `core/noidung_test.go`

**Interfaces:**
- Consumes: `LaOnline()` (Task 1)
- Produces: `MucND.MacOn string`; `func KhoaTheoCheDo(khoa string) string`; `func CoBanOnline(khoa string) bool`; `ND(khoa)` trả bản Online khi đang bật

**Ý tưởng:** không thêm map thứ hai. Mục có `MacOn` thì `init()` đăng ký **thêm** khoá `trangchu.hero.h1@online` vào chính `ndMac`. `NapND`, `DatND`, `KhoiPhucND`, `ghiND`, `ndLa` chạy trên khoá chuỗi nên không phải đổi một dòng nào.

- [ ] **Step 1: Viết test thất bại**

Thêm vào `core/noidung_test.go`:

```go
func TestKhoaTheoCheDoGiuNguyenKhiTaiXuong(t *testing.T) {
	gocTam(t)
	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	if got := KhoaTheoCheDo("trangchu.hero.h1"); got != "trangchu.hero.h1" {
		t.Fatalf("tại xưởng phải giữ nguyên khoá, nhận %q", got)
	}
}

// Khoá không khai MacOn thì hai chế độ dùng chung một chữ, và không rỗng.
func TestKhoaKhongCoMacOnDungChung(t *testing.T) {
	gocTam(t)
	const khoa = "trangchu.hero.h1"
	if CoBanOnline(khoa) {
		t.Skip("khoá này đã khai MacOn — nhánh dùng chung kiểm ở khoá khác")
	}
	if err := DatCheDo(CheDoOnline); err != nil {
		t.Fatal(err)
	}
	if got := KhoaTheoCheDo(khoa); got != khoa {
		t.Fatalf("khoá không có MacOn phải giữ nguyên, nhận %q", got)
	}
	if ND(khoa) == "" {
		t.Fatalf("%s rỗng ở chế độ online", khoa)
	}
}

// Sửa bản Online không được đụng bản Tại xưởng. Đây là toàn bộ lý do có hậu
// tố @online.
func TestSuaBanOnlineKhongDeBanTaiXuong(t *testing.T) {
	gocTam(t)
	const khoa = "trangchu.hero.h1"
	goc := NDMac(khoa)

	if err := DatCheDo(CheDoOnline); err != nil {
		t.Fatal(err)
	}
	if err := DatND(map[string]string{KhoaTheoCheDo(khoa): "Gửi vợt tới trạm"}); err != nil {
		t.Fatal(err)
	}
	if got := ND(khoa); got != "Gửi vợt tới trạm" {
		t.Fatalf("online phải ra chữ vừa sửa, nhận %q", got)
	}
	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	if CoBanOnline(khoa) {
		if got := ND(khoa); got != goc {
			t.Fatalf("sửa bản online không được đụng bản tại xưởng: nhận %q, chờ %q", got, goc)
		}
	} else {
		t.Log("khoá chưa khai MacOn — hai chế độ dùng chung, đúng thiết kế")
	}
	_ = KhoiPhucND(KhoaTheoCheDo(khoa))
}
```

Tại Task 2 khoá `trangchu.hero.h1` chưa khai `MacOn` (Task 6 mới khai), nên nhánh `else` chạy. Task 6 Step 5 chạy lại đúng ba test này, khi đó nhánh `if` mới là nhánh thật.

Tên khoá `trangchu.hero.h1` phải là khoá có thật — lấy bằng `grep -n "trangchu.hero" core/noidung_cay.go`; không có thì thay bằng khoá đầu tiên của trang chủ.

- [ ] **Step 2: Chạy test cho thấy fail**

Run: `go test ./core/ -run "TestKhoaTheoCheDo|TestKhoaKhongCoMacOn|TestSuaBanOnline" -v`
Expected: FAIL — `undefined: KhoaTheoCheDo`

- [ ] **Step 3: Thêm trường vào `MucND`**

```go
type MucND struct {
	Khoa string
	Nhan string
	Mac  string
	// MacOn: chữ mặc định khi site chạy chế độ Online. Rỗng = hai chế độ dùng
	// chung Mac. Đừng khai nó cho khoá mà hai chế độ nói y hệt nhau — mỗi ô
	// khai thêm là một ô nữa phải giữ cho khỏi cũ.
	MacOn string
	Dai   bool
}
```

- [ ] **Step 4: Đăng ký khoá `@online` trong `init()`**

```go
// hauToOnline nối vào khoá gốc để thành khoá của bản Online. Ký tự "@" chọn
// vì khoá thật đặt theo <trang>.<khối>.<chỗ>, không bao giờ có "@" — nên
// không đụng khoá nào.
const hauToOnline = "@online"

func init() {
	for _, t := range CayND {
		for _, n := range t.Nhom {
			for _, m := range n.Muc {
				if _, trung := ndMac[m.Khoa]; trung {
					panic("noi-dung: khoá trùng " + m.Khoa)
				}
				if strings.Contains(m.Khoa, hauToOnline) {
					panic("noi-dung: khoá tự chứa hậu tố " + m.Khoa)
				}
				ndMac[m.Khoa] = m.Mac
				ndDai[m.Khoa] = m.Dai
				ndNhan[m.Khoa] = m.Nhan
				if m.MacOn != "" {
					k := m.Khoa + hauToOnline
					ndMac[k] = m.MacOn
					ndDai[k] = m.Dai
					ndNhan[k] = m.Nhan
				}
			}
		}
	}
}
```

Giữ nguyên phần còn lại của `init()` nếu nó làm thêm việc gì khác. `strings` đã có trong import của `noidung.go`.

- [ ] **Step 5: Thêm hai hàm tra khoá**

```go
// KhoaTheoCheDo trả khoá thật sự đang dùng để tra chữ. Chế độ Online mà khoá
// không khai MacOn thì vẫn dùng khoá gốc — hai chế độ nói chung một câu.
//
// KHÔNG khoá ndMu ở đây: ndMac chỉ ghi trong init(), và ND gọi hàm này khi
// đang giữ RLock. RWMutex của Go không đệ quy — khoá thêm lần nữa là kẹt cứng.
func KhoaTheoCheDo(khoa string) string {
	if !LaOnline() {
		return khoa
	}
	if _, co := ndMac[khoa+hauToOnline]; !co {
		return khoa
	}
	return khoa + hauToOnline
}

// CoBanOnline: khoá này có khai riêng chữ cho bản Online không.
func CoBanOnline(khoa string) bool {
	_, co := ndMac[khoa+hauToOnline]
	return co
}
```

- [ ] **Step 6: Sửa `ND`, `DaSuaND`, `SoDaSuaND`**

```go
func ND(khoa string) string {
	k := KhoaTheoCheDo(khoa)
	ndMu.RLock()
	defer ndMu.RUnlock()
	if v, co := ndSua[k]; co {
		return v
	}
	return ndMac[k]
}

// DaSuaND hỏi về BẢN ĐANG BẬT: khoá này Kendy đã đổi khác mặc định chưa.
func DaSuaND(khoa string) bool {
	k := KhoaTheoCheDo(khoa)
	ndMu.RLock()
	defer ndMu.RUnlock()
	_, co := ndSua[k]
	return co
}

func SoDaSuaND(t TrangND) int {
	ks := make([]string, 0, 32)
	for _, nh := range t.Nhom {
		for _, m := range nh.Muc {
			ks = append(ks, KhoaTheoCheDo(m.Khoa))
		}
	}
	ndMu.RLock()
	defer ndMu.RUnlock()
	n := 0
	for _, k := range ks {
		if _, co := ndSua[k]; co {
			n++
		}
	}
	return n
}
```

`NDMac`, `DatND`, `KhoiPhucND`, `ghiND`, `NapND` **không đổi**: khoá `@online` đã nằm trong `ndMac` nên chúng nhận nó như mọi khoá khác. Riêng `NDMac` vẫn trả bản Tại xưởng theo đúng tên nó — chỗ nào cần bản đang bật thì gọi `NDMac(KhoaTheoCheDo(khoa))`.

- [ ] **Step 7: Chạy test cho thấy pass**

Run: `go test ./core/ -run "TestKhoaTheoCheDo|TestKhoaKhongCoMacOn|TestSuaBanOnline|TestND" -v`
Expected: PASS

- [ ] **Step 8: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add core/noidung.go core/noidung_test.go
git commit -m "feat: khoa noi dung co ban Online qua MacOn"
```

---

## Task 3: Test bảo vệ cây nội dung hai chế độ

**Files:**
- Modify: `core/noidung_test.go` (`TestKhoaNDCoDu`)

**Interfaces:**
- Consumes: `CayND`, `MucND.MacOn`, `ND`, `DatCheDo`, `hauToOnline` (Task 1, 2)
- Produces: không có API mới

- [ ] **Step 1: Sửa dòng đếm cuối `TestKhoaNDCoDu`**

Test này soi khoá *template gọi mà cây không có*, nên khoá `@online` thừa trong `ndMac` không làm nó gãy. Nhưng dòng log cuối đếm `len(ndMac)` giờ đếm cả bản Online — sửa cho khỏi hiểu nhầm:

```go
	goc := 0
	for k := range ndMac {
		if !strings.HasSuffix(k, hauToOnline) {
			goc++
		}
	}
	t.Logf("%d lời gọi, %d khoá gốc, %d khoá có bản Online", dung, goc, len(ndMac)-goc)
```

Tên biến `dung` lấy theo biến đếm đang có trong hàm.

- [ ] **Step 2: Thêm ba test mới**

```go
// Khoá khai MacOn thì cả hai bản phải có chữ. Một bản rỗng nghĩa là trang
// trống ở đúng chế độ ít người xem — và không ai phát hiện ra.
func TestMacOnKhongRong(t *testing.T) {
	for _, tr := range CayND {
		for _, nh := range tr.Nhom {
			for _, m := range nh.Muc {
				if m.MacOn == "" {
					continue
				}
				if strings.TrimSpace(m.Mac) == "" {
					t.Errorf("%s: có MacOn nhưng Mac rỗng", m.Khoa)
				}
				if strings.TrimSpace(m.MacOn) == "" {
					t.Errorf("%s: MacOn chỉ có khoảng trắng", m.Khoa)
				}
				if m.Mac == m.MacOn {
					t.Errorf("%s: MacOn giống hệt Mac — bỏ MacOn đi", m.Khoa)
				}
			}
		}
	}
}

// Đổi chế độ không được làm mất chữ ở bất kỳ khoá nào.
func TestChuyenCheDoKhongMatChu(t *testing.T) {
	gocTam(t)
	for _, cd := range []string{CheDoTaiXuong, CheDoOnline} {
		if err := DatCheDo(cd); err != nil {
			t.Fatal(err)
		}
		for _, tr := range CayND {
			for _, nh := range tr.Nhom {
				for _, m := range nh.Muc {
					if strings.TrimSpace(ND(m.Khoa)) == "" {
						t.Errorf("chế độ %s: khoá %s ra chuỗi rỗng", cd, m.Khoa)
					}
				}
			}
		}
	}
}

// Khoá tự chứa "@" sẽ đẻ ra "a@online@online". init() đã panic, test này nói
// rõ lý do trước khi ai đó phải đi đọc stack.
func TestKhoaKhongChuaKyTuAt(t *testing.T) {
	for _, tr := range CayND {
		for _, nh := range tr.Nhom {
			for _, m := range nh.Muc {
				if strings.Contains(m.Khoa, "@") {
					t.Errorf("khoá %s chứa @ — hậu tố %s sẽ chồng lên nhau", m.Khoa, hauToOnline)
				}
			}
		}
	}
}
```

- [ ] **Step 3: Chạy test**

Run: `go test ./core/ -run "TestMacOn|TestChuyenCheDo|TestKhoaKhongChua|TestKhoaNDCoDu" -v`
Expected: PASS

- [ ] **Step 4: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add core/noidung_test.go
git commit -m "test: bao ve cay noi dung hai che do"
```

---

## Task 4: Công tắc trên `/qt/giao-dien`

**Files:**
- Modify: `core/quantri.go:1023-1044` (`dlGiaoDien`, `hQtGiaoDien`) và bảng route ở đầu tệp
- Modify: `core/ui/qt-giaodien.html`
- Test: `core/qtgiaodien_test.go` (tạo mới)

**Interfaces:**
- Consumes: `CheDoHienTai()`, `DatCheDo()`, `CheDoHopLe()`, `LaOnline()` (Task 1); `LienHeHienTai()` (`core/lienhe.go`); `canLaChu`, `render`, `urlEsc` (`core/quantri.go:1047`)
- Produces: `func hQtCheDo(w http.ResponseWriter, r *http.Request)`; route `POST /qt/giao-dien/che-do`

Test gọi thẳng handler nên không đi qua lớp bọc `boCSRF` — không cần dựng token.

- [ ] **Step 1: Viết test thất bại**

Tạo `core/qtgiaodien_test.go`:

```go
package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// postForm dựng một POST dạng form thường. Dùng chung cho các test handler.
func postForm(duong string, v url.Values) *http.Request {
	r := httptest.NewRequest(http.MethodPost, duong, strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestDoiCheDoQuaAdmin(t *testing.T) {
	gocTam(t)
	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	hQtCheDo(w, postForm("/qt/giao-dien/che-do", url.Values{"che_do": {CheDoOnline}}))

	if w.Code != http.StatusSeeOther {
		t.Fatalf("phải redirect 303, nhận %d", w.Code)
	}
	if !LaOnline() {
		t.Fatal("chế độ chưa đổi sang online")
	}
}

func TestDoiCheDoLaBiTuChoiOAdmin(t *testing.T) {
	gocTam(t)
	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	hQtCheDo(w, postForm("/qt/giao-dien/che-do", url.Values{"che_do": {"linh-tinh"}}))

	if LaOnline() {
		t.Fatal("giá trị lạ mà vẫn đổi chế độ")
	}
	if w.Code != http.StatusSeeOther {
		t.Fatalf("vẫn phải quay về trang có lời báo, nhận %d", w.Code)
	}
}
```

- [ ] **Step 2: Chạy test cho thấy fail**

Run: `go test ./core/ -run TestDoiCheDo -v`
Expected: FAIL — `undefined: hQtCheDo`

- [ ] **Step 3: Thêm trường vào `dlGiaoDien`**

```go
type dlGiaoDien struct {
	dlQt
	LogoTen string
	LogoURL string
	IconTen string
	IconURL string

	// CheDo là công tắc NỘI DUNG, không phải giao diện màu. Nó ở trang này vì
	// đây là chỗ Kendy quen bấm khi muốn đổi bộ mặt của site.
	CheDo       string
	LaOnline    bool
	ThieuDiaChi bool
}
```

Giữ nguyên các trường đang có; chỉ nối thêm ba trường cuối.

- [ ] **Step 4: Điền chúng trong `hQtGiaoDien`, ngay trước `render`**

```go
	d.CheDo = CheDoHienTai()
	d.LaOnline = LaOnline()
	// Bật Online mà chưa có địa chỉ thì cửa hàng vẫn đóng — nói ngay ở đây,
	// đừng để khách phát hiện hộ.
	d.ThieuDiaChi = strings.TrimSpace(LienHeHienTai().DiaChi) == ""
```

- [ ] **Step 5: Thêm `hQtCheDo` cạnh `hQtGiaoDien`**

```go
func hQtCheDo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	ma := r.FormValue("che_do")
	if !CheDoHopLe(ma) {
		http.Redirect(w, r, "/qt/giao-dien?loi="+urlEsc("Chế độ không hợp lệ"), http.StatusSeeOther)
		return
	}
	if err := DatCheDo(ma); err != nil {
		http.Redirect(w, r, "/qt/giao-dien?loi="+urlEsc("Không ghi được: "+err.Error()), http.StatusSeeOther)
		return
	}
	ten := "Tại xưởng"
	if ma == CheDoOnline {
		ten = "Online"
	}
	http.Redirect(w, r, "/qt/giao-dien?ok="+urlEsc("Đã chuyển sang bản "+ten), http.StatusSeeOther)
}
```

- [ ] **Step 6: Đăng ký route**

Run: `grep -n "/qt/giao-dien" core/quantri.go`

Thêm dòng cạnh route `/qt/giao-dien` đang có, dùng đúng lớp bọc quyền mà dòng bên cạnh đang dùng:

```go
	mux.HandleFunc("POST /qt/giao-dien/che-do", canLaChu(hQtCheDo))
```

- [ ] **Step 7: Thêm ô chọn vào `core/ui/qt-giaodien.html`**

Đặt **trên** khối logo — chế độ là quyết định lớn hơn. Lấy tên lớp CSS thật từ chính tệp này (mở ra xem khối logo dùng lớp gì), đừng bịa lớp mới:

```html
<section class="the">
  <h2>Bản đang chạy</h2>
  <p class="mo">Đổi giọng chữ trên trang khách. Không đổi màu, không đổi logo.</p>
  <form method="post" action="/qt/giao-dien/che-do">
    <input type="hidden" name="_csrf" value="{{.CSRF}}">
    <label>
      <input type="radio" name="che_do" value="tai_xuong" {{if not .LaOnline}}checked{{end}}>
      <strong>Tại xưởng</strong> — khách mang đồ tới tận nơi.
    </label>
    <label>
      <input type="radio" name="che_do" value="online" {{if .LaOnline}}checked{{end}}>
      <strong>Online</strong> — khách gửi đồ tới, nhận lại tận nhà. Bật lối vào cửa hàng.
    </label>
    {{if .ThieuDiaChi}}
      <p class="loi">Chưa có địa chỉ trạm. Bật Online bây giờ thì cửa hàng vẫn đóng —
        <a href="/qt/lien-he">điền địa chỉ</a> trước.</p>
    {{end}}
    <button type="submit">Lưu</button>
  </form>
</section>
```

Tên trường CSRF (`_csrf`) phải khớp tên mà `boCSRF` đọc — `grep -n "FormValue" core/csrf.go`.

- [ ] **Step 8: Chạy test cho thấy pass**

Run: `go test ./core/ -run TestDoiCheDo -v`
Expected: PASS

- [ ] **Step 9: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add core/quantri.go core/ui/qt-giaodien.html core/qtgiaodien_test.go
git commit -m "feat: cong tac che do o /qt/giao-dien"
```

---

## Task 5: `/qt/noi-dung` sửa đúng bản đang bật

**Files:**
- Modify: `core/qtnoidung.go` — `ndMucQt`, `dlQtND`, `hQtND`, `hQtNDLuu`, `hQtNDKhoiPhuc`
- Modify: `core/ui/qt-noidung.html`
- Test: `core/qtnoidung_test.go` (tạo mới)

**Interfaces:**
- Consumes: `KhoaTheoCheDo`, `CoBanOnline`, `DaSuaND`, `ND`, `NDMac`, `LaOnline` (Task 1, 2)
- Produces: `ndMucQt.CoBanRieng bool`, `dlQtND.LaOnline bool`

**Nguyên tắc:** tên ô trong form giữ nguyên `k.<khoá gốc>`. Chỉ chỗ *ghi* mới đổi khoá — form không cần biết có hai chế độ, y như template không cần biết.

- [ ] **Step 1: Viết test thất bại**

Tạo `core/qtnoidung_test.go`:

```go
package core

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

// Lưu ở chế độ Online phải ghi vào khoá @online, không đè bản Tại xưởng.
func TestLuuNoiDungTheoCheDoDangBat(t *testing.T) {
	gocTam(t)
	const khoa = "trangchu.hero.h1"
	goc := NDMac(khoa)

	if err := DatCheDo(CheDoOnline); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	hQtNDLuu(w, postForm("/qt/noi-dung/trang-chu/luu", url.Values{
		"k." + khoa: {"Gửi vợt tới trạm"},
	}))

	if got := ND(khoa); got != "Gửi vợt tới trạm" {
		t.Fatalf("bản online phải ra chữ vừa lưu, nhận %q", got)
	}
	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	if !CoBanOnline(khoa) {
		t.Skip("khoá chưa khai MacOn — hai chế độ dùng chung, kiểm lại ở Task 6")
	}
	if got := ND(khoa); got != goc {
		t.Fatalf("bản tại xưởng bị đè: nhận %q, chờ %q", got, goc)
	}
}
```

`hQtNDLuu` lấy mã trang từ đâu thì test phải cấp đúng chỗ đó — `grep -n "func hQtNDLuu" -A 12 core/qtnoidung.go`. Nếu nó đọc `r.PathValue("ma")` thì thêm `r.SetPathValue("ma", "trang-chu")`; nếu đọc `r.FormValue("ma")` thì thêm `"ma": {"trang-chu"}` vào `url.Values`. Mã trang và khoá phải là mã/khoá có thật trong `CayND`.

- [ ] **Step 2: Chạy test cho thấy fail**

Run: `go test ./core/ -run TestLuuNoiDungTheoCheDo -v`
Expected: FAIL — bản Tại xưởng bị đè, hoặc SKIP (khoá chưa có `MacOn` tới Task 6)

- [ ] **Step 3: Sửa `hQtNDLuu`**

```go
	moi := map[string]string{}
	for _, n := range t.Nhom {
		for _, m := range n.Muc {
			if v, gui := r.Form["k."+m.Khoa]; gui {
				// Tên ô trong form là khoá GỐC; khoá được ghi là khoá của bản
				// đang bật. Form không cần biết có hai chế độ.
				moi[KhoaTheoCheDo(m.Khoa)] = v[0]
			}
		}
	}
```

- [ ] **Step 4: Sửa `hQtNDKhoiPhuc`**

```go
	khoa := r.FormValue("khoa")
	if err := KhoiPhucND(KhoaTheoCheDo(khoa)); err != nil {
```

Giữ nguyên phần kiểm khoá có thật ở phía trên — nó soi khoá gốc, đúng ý.

- [ ] **Step 5: Sửa `ndMucQt`, `dlQtND` và `hQtND`**

```go
type ndMucQt struct {
	MucND
	GiaTri string
	DaSua  bool
	// CoBanRieng: khoá này có chữ riêng cho bản Online. Sai thì sửa ô này là
	// sửa cả hai bản — phải nói trước.
	CoBanRieng bool
}
```

Trong `dlQtND` thêm `LaOnline bool`. Trong `hQtND`:

```go
	d.LaOnline = LaOnline()

	// (bên trong vòng lặp dựng nhóm/mục đang có — chỉ thêm dòng CoBanRieng)
			nh.Muc = append(nh.Muc, ndMucQt{
				MucND:      m,
				GiaTri:     ND(m.Khoa),
				DaSua:      DaSuaND(m.Khoa),
				CoBanRieng: CoBanOnline(m.Khoa),
			})
```

`ND` và `DaSuaND` đã tự chọn bản đúng sau Task 2 — không đổi gì thêm.

- [ ] **Step 6: Sửa `core/ui/qt-noidung.html`**

Trên đầu danh sách ô:

```html
<p class="mo">
  {{if .LaOnline}}Đang sửa <strong>bản Online</strong>.{{else}}Đang sửa <strong>bản Tại xưởng</strong>.{{end}}
  <a href="/qt/giao-dien">Đổi bản đang chạy</a>
</p>
```

Trong vòng lặp mỗi ô, khi đang chạy Online mà khoá không có bản riêng:

```html
{{if and $.LaOnline (not .CoBanRieng)}}<span class="nhan-nho">Dùng chung cả hai bản</span>{{end}}
```

Biến trang trong vòng lặp lồng có thể không phải `$` — mở tệp xem `{{range}}` đang lồng thế nào rồi dùng đúng biến.

- [ ] **Step 7: Chạy test cho thấy pass**

Run: `go test ./core/ -run TestLuuNoiDungTheoCheDo -v`
Expected: PASS (hoặc SKIP tới Task 6)

- [ ] **Step 8: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 9: Commit**

```bash
git add core/qtnoidung.go core/qtnoidung_test.go core/ui/qt-noidung.html
git commit -m "feat: admin noi dung sua dung ban dang bat"
```

---

## Task 6: Viết bản chữ Online và cụm khoá cửa hàng

**Files:**
- Modify: `core/noidung_cay.go`
- Modify: `core/server.go` (`tieuDeTrang`, khoảng dòng 395)

**Interfaces:**
- Consumes: `MucND.MacOn` (Task 2)
- Produces: các khoá `cuahang.*` dùng ở Task 8, 10, 12

- [ ] **Step 1: Liệt kê khoá cần đổi giọng**

Run: `grep -n "tới\|ghé\|mang\|đợi\|tại xưởng\|tại chỗ\|đến trạm" core/noidung_cay.go`

Đánh dấu khoá nào nói tới việc khách *có mặt ở xưởng*. Chỉ những khoá đó mới khai `MacOn`.

- [ ] **Step 2: Khai `MacOn`**

Mẫu — áp lên đúng khoá tìm được ở Step 1, giữ nguyên `Khoa` và `Nhan`:

```go
{Khoa: "trangchu.hero.h1", Nhan: "Câu lớn nhất trang chủ",
	Mac:   "Mang vợt tới, lấy về như mới",
	MacOn: "Gửi vợt tới, nhận lại như mới"},

{Khoa: "chung.chan.gioi-thieu", Nhan: "Câu giới thiệu ở chân trang", Dai: true,
	Mac:   "Khám vợt xong báo giá ngay — anh/chị chốt rồi thợ mới làm.",
	MacOn: "Gửi vợt tới, khám xong trạm báo giá — anh/chị chốt rồi thợ mới làm, sửa xong ship về tận nhà."},
```

Giọng bản Online: khách **gửi đồ tới**, nhận lại **tận nhà**. Không hứa số ngày trong chữ mặc định — lead time nằm ở `DichVu.LeadTimeNgay` trong `bang-gia.yaml`.

Giữ nguyên cam kết cân vợt ở **cả hai bản** — cam kết kỹ thuật không đổi theo kênh bán.

- [ ] **Step 3: Thêm trang nội dung `cua-hang` vào `CayND`**

Đặt sau trang dùng chung. Trang này **không** khai `MacOn` — chữ cửa hàng vốn đã là giọng online, và cửa hàng chỉ mở ở chế độ Online.

```go
{
	Ma:  "cua-hang",
	Ten: "Cửa hàng",
	Nhom: []NhomND{
		{
			Ten: "Mở đầu",
			Muc: []MucND{
				{Khoa: "cuahang.mo.tieu-de", Nhan: "Tiêu đề trang", Mac: "Đặt sửa online"},
				{Khoa: "cuahang.mo.dan", Nhan: "Câu dẫn", Dai: true,
					Mac: "Chọn món và việc cần làm, rồi gửi đồ tới. Khám xong trạm báo giá, anh/chị chốt thì thợ mới làm."},
			},
		},
		{
			Ten: "Các bước",
			Muc: []MucND{
				{Khoa: "cuahang.buoc.mon", Nhan: "Bước 1 — tên", Mac: "Anh/chị gửi gì?"},
				{Khoa: "cuahang.buoc.goi", Nhan: "Bước 2 — tên", Mac: "Cần làm gì?"},
				{Khoa: "cuahang.buoc.tinh-trang", Nhan: "Bước 3 — tên", Mac: "Tình trạng hiện tại"},
				{Khoa: "cuahang.buoc.nguoi-nhan", Nhan: "Bước 4 — tên", Mac: "Nhận lại ở đâu"},
				{Khoa: "cuahang.buoc.gia-tu", Nhan: "Nhãn đứng trước giá", Mac: "Giá từ"},
				{Khoa: "cuahang.buoc.bao-gia-rieng", Nhan: "Nhãn cho việc phải xem mới báo giá", Mac: "Xem món rồi báo giá"},
				{Khoa: "cuahang.buoc.gia-nhac", Nhan: "Câu nhắc dưới bảng giá", Dai: true,
					Mac: "Đây là giá khởi điểm. Giá chốt báo sau khi thợ khám, và anh/chị duyệt rồi thợ mới làm."},
				{Khoa: "cuahang.buoc.anh-nhac", Nhan: "Câu nhắc gửi ảnh", Dai: true,
					Mac: "Chụp giúp trạm chỗ hỏng, chụp gần và đủ sáng. Có ảnh thì thợ đoán được việc trước khi hàng tới."},
			},
		},
		{
			Ten: "Màn kết — sau khi đặt xong",
			Muc: []MucND{
				{Khoa: "cuahang.xong.tieu-de", Nhan: "Tiêu đề", Mac: "Đã nhận đơn. Giờ gửi đồ tới nhé"},
				{Khoa: "cuahang.xong.ghi-ma", Nhan: "Câu nhắc ghi mã lên kiện", Dai: true,
					Mac: "**Ghi mã đơn lên kiện hàng** — hoặc kẹp một mẩu giấy có mã vào trong. Không có mã thì kiện tới nơi trạm không biết của ai."},
				{Khoa: "cuahang.xong.dong-goi", Nhan: "Hướng dẫn đóng gói", Dai: true,
					Mac: "Bọc vợt bằng xốp hơi hoặc khăn dày, kỹ nhất ở cán và viền. Cho vào hộp cứng, chèn kín để món không xê dịch trong hộp. Giày thì buộc dây lại, nhét giấy vào mũi cho giữ phom."},
				{Khoa: "cuahang.xong.tra-cuu", Nhan: "Câu nhắc lưu link tra cứu", Dai: true,
					Mac: "Lưu lại đường dẫn này. Mọi cập nhật của đơn hiện ở đó, không cần đăng nhập."},
			},
		},
		{
			Ten: "Thanh toán",
			Muc: []MucND{
				{Khoa: "cuahang.tra.tieu-de", Nhan: "Tiêu đề khối thanh toán", Mac: "Sửa xong rồi"},
				{Khoa: "cuahang.tra.qr", Nhan: "Nhãn cách chuyển khoản", Mac: "Chuyển khoản"},
				{Khoa: "cuahang.tra.qr-nhac", Nhan: "Câu nhắc nội dung chuyển khoản", Dai: true,
					Mac: "Giữ nguyên nội dung chuyển khoản có sẵn mã đơn. Xoá mã đi thì trạm không biết tiền của đơn nào."},
				{Khoa: "cuahang.tra.cod", Nhan: "Nhãn cách trả khi nhận", Mac: "Trả khi nhận hàng (COD)"},
				{Khoa: "cuahang.tra.cod-nhac", Nhan: "Câu nhắc phí COD", Dai: true,
					Mac: "Trả khi nhận thì có thêm phí thu hộ của bên vận chuyển, và không được miễn phí gửi về."},
				{Khoa: "cuahang.tra.free-ship", Nhan: "Câu báo miễn phí gửi về", Dai: true,
					Mac: "Đơn này đạt mức trạm chịu phí gửi về — anh/chị không phải trả tiền ship chiều về."},
			},
		},
		{
			Ten: "Khi cửa hàng đóng",
			Muc: []MucND{
				{Khoa: "cuahang.dong.tieu-de", Nhan: "Tiêu đề", Mac: "Tạm chưa nhận đơn online"},
				{Khoa: "cuahang.dong.than", Nhan: "Nội dung", Dai: true,
					Mac: "Anh/chị nhắn Zalo giúp trạm, hoặc gọi trực tiếp. Trạm vẫn nhận sửa bình thường."},
			},
		},
		{
			Ten: "Khi không nhận ca",
			Muc: []MucND{
				{Khoa: "cuahang.tuchoi.giay-kieu", Nhan: "Lý do — giày sai kiểu", Dai: true,
					Mac: "Trạm chỉ thay đế giày chạy bộ và giày đi lại. Giày court, cầu lông, tennis, pickleball trạm chưa nhận — đế sai làm tăng ma sát xoay, hại chân."},
				{Khoa: "cuahang.tuchoi.duoi-nguong", Nhan: "Lý do — món dưới ngưỡng mở kênh ship", Dai: true,
					Mac: "Món dưới mức này thì hai chiều ship đã đắt hơn tiền sửa. Anh/chị mang tới trực tiếp, hoặc nhắn Zalo để trạm xem có cách nào rẻ hơn không."},
			},
		},
	},
},
```

Kiểm đúng tên trường của `TrangND` trước khi dán: `grep -n "type TrangND struct" -A 10 core/noidung.go`. Có trường `Rieng` thì để mặc định `false` — trang này phải hiện trong admin.

- [ ] **Step 4: Thêm tiêu đề tab cho trang cửa hàng**

Trong map `tieuDeTrang` (`core/server.go:395`):

```go
	"cua-hang": "Đặt sửa online",
```

- [ ] **Step 5: Chạy test**

Run: `go test ./core/ -run "TestKhoaND|TestMacOn|TestChuyenCheDo|TestSuaBanOnline|TestLuuNoiDungTheoCheDo" -v`
Expected: PASS. Hai test từng đi nhánh SKIP/`else` giờ chạy nhánh thật vì `trangchu.hero.h1` đã có `MacOn`. `TestKhoaNDCoDu` vẫn PASS — nó soi chiều template→cây, khoá `cuahang.*` chưa ai gọi là bình thường ở bước này.

- [ ] **Step 6: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add core/noidung_cay.go core/server.go
git commit -m "feat: ban chu Online va cum khoa cua hang"
```

---

## Task 7: Trạng thái `cho_hang_ve` và trường mới trên `Don`

**Files:**
- Modify: `core/donhang.go` — khối hằng trạng thái, `CacTrangThai` (dòng 61), `Don` (dòng 178), cạnh `LaGiay()` (dòng 231)
- Test: `core/donloai_test.go`

**Interfaces:**
- Consumes: không
- Produces: `const TTChoHangVe = "cho_hang_ve"`; `const TraQR = "qr"`, `TraCOD = "cod"`, `NguonWeb = "web"`, `NguonTay = "tay"`; `Don.KhachDiaChi`, `Don.HinhThucTra`, `Don.MaVanDonDen`, `Don.MaVanDonVe`, `Don.NguonDon` (đều `string`); `func (d Don) NguonDonHopLe() string`; `func (d Don) ChoHangVe() bool`

- [ ] **Step 1: Viết test thất bại**

Thêm vào `core/donloai_test.go`:

```go
func TestTrangThaiChoHangVeHopLe(t *testing.T) {
	if !TrangThaiHopLe(TTChoHangVe) {
		t.Fatal("cho_hang_ve phải là trạng thái hợp lệ")
	}
	mt := TrangThaiCua(TTChoHangVe)
	if !mt.DangChay {
		t.Fatal("đơn chờ hàng về vẫn đang chạy — nó phải hiện trong danh sách việc")
	}
	if mt.TenKhach == "" || mt.TenKhach == TTChoHangVe {
		t.Fatal("thiếu câu nói cho khách")
	}
}

// cho_hang_ve đứng TRƯỚC moi: thứ tự trong CacTrangThai là thứ tự trên bảng
// điều khiển, và việc đầu tiên mỗi sáng là xem kiện nào đã về.
func TestChoHangVeDungDauDanhSach(t *testing.T) {
	if CacTrangThai[0].Ma != TTChoHangVe {
		t.Fatalf("trạng thái đầu phải là %s, nhận %s", TTChoHangVe, CacTrangThai[0].Ma)
	}
}

// Đơn cũ trong data/ không có trường mới nào — đọc vẫn phải ra nghĩa đúng.
func TestDonCuKhongCoTruongMoi(t *testing.T) {
	cu := []byte(`{"ma":"TV-2601-001","trang_thai":"xong","khach_ten":"A"}`)
	var d Don
	if err := json.Unmarshal(cu, &d); err != nil {
		t.Fatalf("đọc đơn cũ: %v", err)
	}
	if d.NguonDonHopLe() != NguonTay {
		t.Fatalf("đơn cũ phải hiểu là nhận tay, nhận %q", d.NguonDonHopLe())
	}
	if d.HinhThucTra != "" || d.KhachDiaChi != "" {
		t.Fatal("đơn cũ không có mấy trường này")
	}
	if d.ChoHangVe() {
		t.Fatal("đơn xong không phải đang chờ hàng về")
	}
}
```

Thêm `encoding/json` vào import của tệp test nếu chưa có. Tên khoá JSON trong chuỗi `cu` phải khớp thẻ `json:` thật của `Don` — `grep -n "json:\"ma\"\|json:\"trang_thai\"\|json:\"khach_ten\"" core/donhang.go`.

- [ ] **Step 2: Chạy test cho thấy fail**

Run: `go test ./core/ -run "TestTrangThaiChoHangVe|TestChoHangVeDungDau|TestDonCuKhong" -v`
Expected: FAIL — `undefined: TTChoHangVe`

- [ ] **Step 3: Thêm hằng trạng thái**

Trong khối `const` cùng `TTMoi`:

```go
	// TTChoHangVe: khách đã đặt trên web, kiện chưa tới tay trạm. Tách khỏi
	// "mới nhận" vì hai thứ này xử lý khác hẳn — cái này không có gì trên bàn
	// để làm, chỉ có một cái hẹn. Trộn chung thì danh sách việc đầy đơn không
	// làm được gì, và cái nào lâu không thấy hàng cũng không lộ ra.
	TTChoHangVe = "cho_hang_ve"
```

- [ ] **Step 4: Thêm dòng đầu `CacTrangThai`**

Thứ tự trường là `{Ma, Ten, TenKhach, Mau, DangChay}`:

```go
var CacTrangThai = []MoTaTrangThai{
	{TTChoHangVe, "Chờ hàng về", "Đã nhận đơn, đang chờ hàng của anh/chị tới", "cho", true},
	{TTMoi, "Mới nhận", "Đã nhận vợt, đang xếp lịch kiểm tra", "trung", true},
	// ... giữ nguyên phần còn lại
}
```

Giá trị `"cho"` ở cột `Mau` phải là một tên màu CSS đã có — `grep -n "\.tt-" core/ui/css.html | head -20`. Không có tên phù hợp thì dùng lại `"trung"`; **không thêm biến màu mới**.

- [ ] **Step 5: Thêm hằng và trường trên `Don`**

Cạnh khối `LoaiVot`/`LoaiGiay`:

```go
// Hai cách khách trả tiền cho đơn gửi từ xa.
const (
	TraQR  = "qr"
	TraCOD = "cod"
)

// Đơn vào hệ bằng đường nào. Rỗng đọc là tay — mọi đơn có trước cửa hàng.
const (
	NguonWeb = "web"
	NguonTay = "tay"
)
```

Trong `Don`, ngay dưới `KenhNhan`:

```go
	// Địa chỉ ship trả về. Rỗng với đơn khách tới lấy.
	KhachDiaChi string `json:"khach_dia_chi"`
	// qr | cod. Rỗng = khách chưa chọn.
	HinhThucTra string `json:"hinh_thuc_tra"`
	// Vận đơn hai chiều, GÕ TAY. Không nối API hãng vận chuyển — xem spec.
	MaVanDonDen string `json:"ma_van_don_den"`
	MaVanDonVe  string `json:"ma_van_don_ve"`
	// web | tay. Rỗng đọc là tay.
	NguonDon string `json:"nguon_don"`
```

- [ ] **Step 6: Thêm hai hàm cạnh `LaGiay()`**

```go
// NguonDonHopLe — rỗng đọc là nhận tay, y như Loai rỗng đọc là vợt.
func (d Don) NguonDonHopLe() string {
	if d.NguonDon == NguonWeb {
		return NguonWeb
	}
	return NguonTay
}

func (d Don) ChoHangVe() bool { return d.TrangThai == TTChoHangVe }
```

- [ ] **Step 7: Chạy test cho thấy pass**

Run: `go test ./core/ -run "TestTrangThaiChoHangVe|TestChoHangVeDungDau|TestDonCuKhong" -v`
Expected: PASS

- [ ] **Step 8: Chạy toàn bộ test và sửa chỗ gãy**

Run: `go test ./...`
Expected: có thể FAIL ở test đếm trạng thái hoặc test bảng điều khiển vì danh sách dài thêm một mục. Sửa **test**, không sửa danh sách — trạng thái mới là đúng ý.

- [ ] **Step 9: Commit**

```bash
git add core/donhang.go core/donloai_test.go
git commit -m "feat: trang thai cho_hang_ve va truong don cho ban Online"
```

---

## Task 8: Trang `/cua-hang` — chọn dịch vụ và tạo đơn

**Files:**
- Create: `core/cuahang.go`, `core/cuahang_test.go`, `core/ui/cua-hang.html`, `core/ui/cua-hang-xong.html`
- Modify: `core/server.go` (bảng route, khoảng dòng 1155), `core/ui/css.html`

**Interfaces:**
- Consumes: `LaOnline()` (T1); khoá `cuahang.*` (T6); `TTChoHangVe`, `NguonWeb` (T7); `Don`, `Moc`, `MaDonMoi(loai)` (`core/donhang.go:431`), `LuuDon` (`:375`), `LoaiHopLe` (`:111`), `LoaiVot`/`LoaiGiay`, `GiayChayBo`/`GiayDiLai` (`:125`); `maNgauNhien(n int) string` (`core/auth.go:250`); `LienHeHienTai()` (`core/lienhe.go:53`); `Chung` + `chung(r, trang)` (`core/server.go:375`), `render` (`:1095`), `catBot` (`:779`), `dinhDangTien` (`:208`), `rl.choPhep(ip string, soLan int, trong time.Duration) bool` (`:350`), `ipCua`; `DichVuDangBan(GiaiDoan)`, `DichVu.GiaTheoGiaiDoan`, `.BaoGiaRieng`, `.DieuKien`, `.LeadTimeNgay`, `.Nhom` (`core/config.go`)
- Produces: `func CuaHangMo() bool`; `func hCuaHang(w,r)`; `func hCuaHangGui(w,r)`; `type ODichVu struct{Ma, Ten, Gia, DieuKien string; LeadNgay int}`; `func DichVuChoCuaHang(loai string) []ODichVu`

**Nhóm dịch vụ:** `Nhom` `"C"` là việc gia công ngoài (giày — hiện chỉ `THAY_DE_GIAY`), `"A"`/`"B"` là việc trên vợt. Xem `tenNhom` (`core/server.go:298`).

- [ ] **Step 1: Viết test thất bại**

Tạo `core/cuahang_test.go`:

```go
package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// datDiaChi đặt địa chỉ trạm cho test rồi trả lại như cũ.
func datDiaChi(t *testing.T, dc string) {
	t.Helper()
	truoc := LienHeHienTai()
	moi := truoc
	moi.DiaChi = dc
	if _, err := DatLienHe(moi); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = DatLienHe(truoc) })
}

// moCuaHang: gốc dữ liệu tạm, chế độ Online, có địa chỉ, kho đơn rỗng.
func moCuaHang(t *testing.T) {
	t.Helper()
	gocTam(t)
	if err := NapDon(); err != nil {
		t.Fatal(err)
	}
	datDiaChi(t, "12 Nguyễn Trãi, Thanh Xuân, Hà Nội")
	if err := DatCheDo(CheDoOnline); err != nil {
		t.Fatal(err)
	}
}

func TestCuaHangDongKhiChuaCoDiaChi(t *testing.T) {
	moCuaHang(t)
	datDiaChi(t, "")
	if CuaHangMo() {
		t.Fatal("chưa có địa chỉ thì cửa hàng phải đóng")
	}
}

func TestCuaHangDongKhiTaiXuong(t *testing.T) {
	moCuaHang(t)
	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	if CuaHangMo() {
		t.Fatal("bản Tại xưởng thì cửa hàng phải đóng")
	}
}

func TestDatHangTaoDonChoHangVe(t *testing.T) {
	moCuaHang(t)
	w := httptest.NewRecorder()
	hCuaHangGui(w, postForm("/cua-hang", url.Values{
		"loai":       {LoaiVot},
		"dich_vu":    {"DAN_VIEN"},
		"vot_hang":   {"Joola Ben Johns"},
		"gia_tri":    {"3000000"},
		"tinh_trang": {"Viền bong một đoạn 10cm"},
		"ten":        {"Chị Lan"},
		"lien_he":    {"0900000000"},
		"dia_chi":    {"45 Lê Lợi, Q1, TP.HCM"},
	}))

	ds := LocDon(BoLoc{TrangThai: TTChoHangVe})
	if len(ds) != 1 {
		t.Fatalf("phải có đúng 1 đơn chờ hàng về, có %d", len(ds))
	}
	d := ds[0]
	if !strings.HasPrefix(d.Ma, "TV-") {
		t.Fatalf("đơn vợt phải mang tiền tố TV-, nhận %s", d.Ma)
	}
	if d.Token == "" {
		t.Fatal("thiếu token tra cứu")
	}
	if d.NguonDonHopLe() != NguonWeb {
		t.Fatal("đơn đặt trên web phải ghi nguồn web")
	}
	if d.KenhNhan != "ship" {
		t.Fatalf("kênh nhận phải là ship, nhận %q", d.KenhNhan)
	}
	if d.KhachDiaChi == "" {
		t.Fatal("thiếu địa chỉ trả về")
	}
	if d.VotGiaTri != 3000000 {
		t.Fatalf("giá trị món phải là 3000000, nhận %d", d.VotGiaTri)
	}
	// Chưa khám thì chưa có tiền. Giá chốt sinh ở bước báo giá.
	if d.TongTien != 0 || len(d.DongTien) != 0 {
		t.Fatalf("đơn mới đặt không được có tiền: %d / %d dòng", d.TongTien, len(d.DongTien))
	}
	if len(d.LichSu) == 0 {
		t.Fatal("phải để lại một mốc trong lịch sử")
	}
	// Màn kết phải in mã đơn và địa chỉ — đó là toàn bộ việc của nó.
	than := w.Body.String()
	if !strings.Contains(than, d.Ma) {
		t.Fatal("màn kết không in mã đơn")
	}
	if !strings.Contains(than, "Nguyễn Trãi") {
		t.Fatal("màn kết không in địa chỉ gửi tới")
	}
}

func TestDatHangGiayMangTienToTG(t *testing.T) {
	moCuaHang(t)
	w := httptest.NewRecorder()
	hCuaHangGui(w, postForm("/cua-hang", url.Values{
		"loai":       {LoaiGiay},
		"dich_vu":    {"THAY_DE_GIAY"},
		"giay_hang":  {"Nike Pegasus 40"},
		"giay_size":  {"42"},
		"giay_kieu":  {GiayChayBo},
		"gia_tri":    {"3200000"},
		"tinh_trang": {"Đế mòn hết gai"},
		"ten":        {"Anh Nam"},
		"lien_he":    {"0900000001"},
		"dia_chi":    {"45 Lê Lợi, Q1"},
	}))
	_ = w

	ds := LocDon(BoLoc{TrangThai: TTChoHangVe, Loai: LoaiGiay})
	if len(ds) != 1 {
		t.Fatalf("phải có 1 đơn giày, có %d", len(ds))
	}
	if !strings.HasPrefix(ds[0].Ma, "TG-") {
		t.Fatalf("đơn giày phải mang tiền tố TG-, nhận %s", ds[0].Ma)
	}
	if ds[0].GiayKieu != GiayChayBo {
		t.Fatalf("mất kiểu giày: %q", ds[0].GiayKieu)
	}
}

func TestThieuLienHeThiKhongTaoDon(t *testing.T) {
	moCuaHang(t)
	w := httptest.NewRecorder()
	hCuaHangGui(w, postForm("/cua-hang", url.Values{
		"loai":    {LoaiVot},
		"dich_vu": {"DAN_VIEN"},
		"ten":     {"Chị Lan"},
		// thiếu lien_he và dia_chi
	}))

	if n := len(LocDon(BoLoc{TrangThai: TTChoHangVe})); n != 0 {
		t.Fatalf("thiếu liên hệ mà vẫn tạo %d đơn", n)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("phải trả lại form kèm lời báo, nhận %d", w.Code)
	}
}

// Trang khách không bao giờ được in một con số giá trần trụi: mọi số phải đi
// kèm nhãn "giá từ", còn việc bao_gia_rieng thì không có số nào.
func TestGiaLuonLaGiaTu(t *testing.T) {
	moCuaHang(t)
	for _, o := range DichVuChoCuaHang(LoaiVot) {
		if o.Gia == "" {
			t.Errorf("%s: ô giá rỗng — phải có chữ, kể cả khi là báo giá riêng", o.Ma)
		}
	}
	w := httptest.NewRecorder()
	hCuaHang(w, httptest.NewRequest(http.MethodGet, "/cua-hang", nil))
	if !strings.Contains(w.Body.String(), ND("cuahang.buoc.gia-tu")) {
		t.Fatal("trang cửa hàng không thấy nhãn giá từ")
	}
}
```

Mã dịch vụ `DAN_VIEN` phải có thật trong `vanhanh/bang-gia.yaml` và đang mở ở `GiaiDoan` hiện tại — `grep -n "^  - ma:" vanhanh/bang-gia.yaml`. Không có thì thay bằng mã vợt đầu tiên đang bán.

- [ ] **Step 2: Chạy test cho thấy fail**

Run: `go test ./core/ -run "TestCuaHang|TestDatHang|TestThieuLienHe|TestGiaLuon" -v`
Expected: FAIL — `undefined: CuaHangMo`

- [ ] **Step 3: Viết `core/cuahang.go`**

```go
// Cửa hàng: khách đặt sửa từ xa, không cần tới trạm.
//
// VÌ SAO ĐẺ RA ĐƠN THẬT NGAY, KHÔNG PHẢI MỘT "YÊU CẦU" NẰM CHỜ. Form gửi ảnh
// (hGuiYeuCau, server.go) tạo yeuCauKhach rồi để chủ tự tay dựng đơn — hợp lý
// khi khách đứng trước mặt. Nhưng khách ở xa cần MỘT MÃ để ghi lên kiện hàng
// trước khi ra bưu điện. Không có mã thì kiện tới nơi không ai biết của ai, và
// khách không có gì để tra.
//
// ĐƠN Ở ĐÂY CHƯA CÓ TIỀN. TongTien để 0, DongTien rỗng. Giá chốt sinh ở bước
// báo giá sau khi thợ khám — chỉ bảng giá mới được đẻ ra con số cam kết với
// khách. Dịch vụ khách chọn lúc đặt chỉ là NGUYỆN VỌNG, ghi vào TinhTrang cho
// thợ đọc.
package core

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CuaHangMo: cửa hàng chỉ mở khi site chạy bản Online VÀ đã có địa chỉ trạm.
// Thiếu địa chỉ mà vẫn nhận đơn là để khách đặt xong mới phát hiện không biết
// gửi đi đâu — hỏng nặng hơn nhiều so với đóng cửa một hôm.
func CuaHangMo() bool {
	return LaOnline() && strings.TrimSpace(LienHeHienTai().DiaChi) != ""
}

// ODichVu — một dòng trong bảng chọn của khách. Gia là CHUỖI đã dựng sẵn, để
// template không có đường nào in ra một con số trần.
type ODichVu struct {
	Ma       string
	Ten      string
	Gia      string
	DieuKien string
	LeadNgay int
}

// DichVuChoCuaHang lọc bảng giá theo món. Nhóm C là việc gia công ngoài (thay
// đế giày); A và B là việc trên vợt.
func DichVuChoCuaHang(loai string) []ODichVu {
	giay := LoaiHopLe(loai) == LoaiGiay
	out := []ODichVu{}
	for _, d := range DichVuDangBan(GiaiDoan) {
		if (d.Nhom == "C") != giay {
			continue
		}
		o := ODichVu{Ma: d.Ma, Ten: d.Ten, DieuKien: d.DieuKien, LeadNgay: d.LeadTimeNgay}
		p := d.GiaTheoGiaiDoan(GiaiDoan)
		if d.BaoGiaRieng || p == nil {
			// Không có giá niêm yết. Nói thẳng, đừng bịa một con số.
			o.Gia = ND("cuahang.buoc.bao-gia-rieng")
		} else {
			o.Gia = ND("cuahang.buoc.gia-tu") + " " + dinhDangTien(*p)
		}
		out = append(out, o)
	}
	return out
}

func timDichVuCuaHang(loai, ma string) (ODichVu, bool) {
	for _, o := range DichVuChoCuaHang(loai) {
		if o.Ma == ma {
			return o, true
		}
	}
	return ODichVu{}, false
}

type dlCuaHang struct {
	Chung

	Mo     bool
	DiaChi string
	GioLam string

	Loai   string
	DichVu []ODichVu
	Loi    string

	// Sau khi đặt xong
	Don     *Don
	LinkTra string
}

func hCuaHang(w http.ResponseWriter, r *http.Request) {
	d := dlCuaHang{Chung: chung(r, "cua-hang"), Mo: CuaHangMo()}
	if !d.Mo {
		render(w, "cua-hang.html", d)
		return
	}
	lh := LienHeHienTai()
	d.DiaChi, d.GioLam = lh.DiaChi, lh.GioLamVic
	d.Loai = LoaiHopLe(r.URL.Query().Get("loai"))
	d.DichVu = DichVuChoCuaHang(d.Loai)
	render(w, "cua-hang.html", d)
}

func hCuaHangGui(w http.ResponseWriter, r *http.Request) {
	if !CuaHangMo() {
		render(w, "cua-hang.html", dlCuaHang{Chung: chung(r, "cua-hang"), Mo: false})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	r.ParseForm()

	if !rl.choPhep("cuahang:"+ipCua(r), 5, time.Hour) {
		cuaHangLoi(w, r, "Đặt hơi nhiều lần rồi ạ. Anh/chị nhắn trực tiếp giúp em.")
		return
	}

	loai := LoaiHopLe(r.FormValue("loai"))
	ten := catBot(r.FormValue("ten"), 100)
	lienHe := catBot(r.FormValue("lien_he"), 100)
	diaChi := catBot(r.FormValue("dia_chi"), 300)
	tinhTrang := catBot(r.FormValue("tinh_trang"), 1000)

	if ten == "" || lienHe == "" || diaChi == "" {
		cuaHangLoi(w, r, "Cần đủ tên, số liên lạc và địa chỉ nhận lại thì trạm mới gửi đồ về được ạ.")
		return
	}

	giaTri := 0
	if n, err := strconv.Atoi(strings.TrimSpace(r.FormValue("gia_tri"))); err == nil && n > 0 {
		giaTri = n
	}
	kieuGiay := strings.TrimSpace(r.FormValue("giay_kieu"))

	// Nguyện vọng của khách, không phải chẩn đoán của thợ. Ghi vào phần khách
	// mô tả để thợ đọc, không đẻ ra dòng tiền nào.
	if ma := strings.TrimSpace(r.FormValue("dich_vu")); ma != "" {
		if dv, co := timDichVuCuaHang(loai, ma); co {
			tinhTrang = strings.TrimSpace("Khách chọn: " + dv.Ten + "\n" + tinhTrang)
		}
	}

	d := &Don{
		Ma:          MaDonMoi(loai),
		Token:       maNgauNhien(8),
		Ngay:        time.Now().Format("2006-01-02 15:04:05"),
		Loai:        loai,
		TrangThai:   TTChoHangVe,
		KhachTen:    ten,
		KhachLienHe: lienHe,
		KhachDiaChi: diaChi,
		TinhTrang:   tinhTrang,
		VotGiaTri:   giaTri,
		KenhNhan:    "ship",
		NguonDon:    NguonWeb,
	}
	if d.LaGiay() {
		d.GiayHang = catBot(r.FormValue("giay_hang"), 100)
		d.GiaySize = catBot(r.FormValue("giay_size"), 20)
		d.GiayKieu = kieuGiay
	} else {
		d.VotHang = catBot(r.FormValue("vot_hang"), 100)
	}
	d.LichSu = append(d.LichSu, Moc{
		Luc:       d.Ngay,
		TrangThai: TTChoHangVe,
		Nguoi:     "khách",
		GhiChu:    "Đặt trên web",
	})

	if err := LuuDon(d); err != nil {
		cuaHangLoi(w, r, "Trạm chưa lưu được đơn. Anh/chị nhắn Zalo giúp em.")
		return
	}

	lh := LienHeHienTai()
	render(w, "cua-hang-xong.html", dlCuaHang{
		Chung:   chung(r, "cua-hang"),
		Mo:      true,
		DiaChi:  lh.DiaChi,
		GioLam:  lh.GioLamVic,
		Don:     d,
		LinkTra: "/tra-cuu/" + d.Token,
	})
}

func cuaHangLoi(w http.ResponseWriter, r *http.Request, msg string) {
	lh := LienHeHienTai()
	loai := LoaiHopLe(r.FormValue("loai"))
	render(w, "cua-hang.html", dlCuaHang{
		Chung:  chung(r, "cua-hang"),
		Mo:     CuaHangMo(),
		DiaChi: lh.DiaChi,
		GioLam: lh.GioLamVic,
		Loai:   loai,
		DichVu: DichVuChoCuaHang(loai),
		Loi:    msg,
	})
}
```

Trường giờ làm việc trên `LienHe` tên thật là `GioLamVic` — thiếu chữ, nhưng thẻ YAML vẫn là `gio_lam_viec` (`core/config.go:30`). Đừng "sửa lỗi chính tả" ở đây; đổi tên trường là việc riêng, không gộp vào task này.

`/tra-cuu/{token}` chưa có — Task 11 làm. Tới lúc đó link mới sống; ở bước này nó chỉ là một chuỗi trong màn kết.

- [ ] **Step 4: Đăng ký route trong `core/server.go`**

Cạnh các route công khai khác (khoảng dòng 1155):

```go
	mux.HandleFunc("GET /cua-hang", hCuaHang)
	mux.HandleFunc("POST /cua-hang", hCuaHangGui)
```

- [ ] **Step 5: Viết `core/ui/cua-hang.html`**

Một trang, một form, POST một lần — không nhiều bước bằng JavaScript. Mở `core/ui/quytrinh.html` lấy đúng khung `{{define}}` và tên khối của bộ template rồi bọc nội dung này vào:

```html
<main class="la-cua-hang">
{{if not .Mo}}
  <h1>{{nd "cuahang.dong.tieu-de"}}</h1>
  {{ndm "cuahang.dong.than"}}
{{else}}
  <h1>{{nd "cuahang.mo.tieu-de"}}</h1>
  {{ndm "cuahang.mo.dan"}}
  {{if .Loi}}<p class="loi">{{.Loi}}</p>{{end}}

  <form method="post" action="/cua-hang" enctype="multipart/form-data">
    <input type="hidden" name="_csrf" value="{{.CSRF}}">

    <fieldset><legend>{{nd "cuahang.buoc.mon"}}</legend>
      <label><input type="radio" name="loai" value="vot" checked> Vợt pickleball</label>
      <label><input type="radio" name="loai" value="giay"> Giày</label>
    </fieldset>

    <fieldset><legend>{{nd "cuahang.buoc.goi"}}</legend>
      {{range .DichVu}}
        <label class="goi">
          <input type="radio" name="dich_vu" value="{{.Ma}}">
          <span class="goi-ten">{{.Ten}}</span>
          <span class="goi-gia">{{.Gia}}</span>
          {{if .DieuKien}}<span class="goi-dk">{{.DieuKien}}</span>{{end}}
        </label>
      {{end}}
      {{ndm "cuahang.buoc.gia-nhac"}}
    </fieldset>

    <fieldset><legend>{{nd "cuahang.buoc.tinh-trang"}}</legend>
      <label>Hãng / đời vợt <input name="vot_hang" maxlength="100"></label>
      <label>Hãng / đời giày <input name="giay_hang" maxlength="100"></label>
      <label>Size giày <input name="giay_size" maxlength="20"></label>
      <label>Kiểu giày
        <select name="giay_kieu">
          <option value="">— chọn —</option>
          <option value="chay_bo">Giày chạy bộ</option>
          <option value="di_lai">Giày đi lại</option>
        </select>
      </label>
      <label>Giá trị món (đ) <input name="gia_tri" inputmode="numeric"></label>
      <label>Mô tả chỗ hỏng <textarea name="tinh_trang" rows="4"></textarea></label>
      <label>Ảnh chỗ hỏng <input type="file" name="anh" accept="image/*" multiple></label>
      {{ndm "cuahang.buoc.anh-nhac"}}
    </fieldset>

    <fieldset><legend>{{nd "cuahang.buoc.nguoi-nhan"}}</legend>
      <label>Tên <input name="ten" maxlength="100" required></label>
      <label>Số điện thoại / Zalo <input name="lien_he" maxlength="100" required></label>
      <label>Địa chỉ nhận lại <textarea name="dia_chi" rows="2" required></textarea></label>
    </fieldset>

    <button type="submit">Đặt sửa</button>
  </form>
{{end}}
</main>
```

Ô vợt và ô giày ẩn/hiện theo lựa chọn bước 1 bằng một đoạn JS ngắn mang `nonce="{{.Nonce}}"` (CSP chặn script không có nonce). **Server không tin vào việc ẩn**: `hCuaHangGui` đã chỉ đọc ô của đúng loại.

Ô `anh` chưa được xử lý ở task này — Task 9 làm.

- [ ] **Step 6: Viết `core/ui/cua-hang-xong.html`**

```html
<main class="la-cua-hang">
  <h1>{{nd "cuahang.xong.tieu-de"}}</h1>

  <section class="ma-don">
    <p class="nhan">Mã đơn</p>
    <p class="ma">{{.Don.Ma}}</p>
    {{ndm "cuahang.xong.ghi-ma"}}
  </section>

  <section>
    <h2>Gửi tới</h2>
    <p class="dia-chi">{{.DiaChi}}</p>
    <p>{{.Don.KhachTen}} — {{.Don.KhachLienHe}}</p>
    {{if .GioLam}}<p>{{.GioLam}}</p>{{end}}
  </section>

  <section>
    <h2>Đóng gói</h2>
    {{ndm "cuahang.xong.dong-goi"}}
  </section>

  <section>
    <h2>Theo dõi đơn</h2>
    <p><a href="{{.LinkTra}}">{{.LinkTra}}</a></p>
    {{ndm "cuahang.xong.tra-cuu"}}
  </section>
</main>
```

Nút sao chép mã đơn và địa chỉ: một `<button type="button" data-chep="...">` cộng một đoạn JS mang `nonce="{{.Nonce}}"` gọi `navigator.clipboard.writeText`. Trình duyệt không có `clipboard` thì nút tự ẩn — chữ vẫn chọn tay được, nên không mất gì.

- [ ] **Step 7: Thêm style `body.la-cua-hang` vào `core/ui/css.html`**

Chỉ bố cục và khoảng cách, dùng token `--md-*` đã có. Mã đơn hiện to, dễ chép. Không khai biến màu mới, không dùng escape CSS.

- [ ] **Step 8: Chạy test cho thấy pass**

Run: `go test ./core/ -run "TestCuaHang|TestDatHang|TestThieuLienHe|TestGiaLuon" -v`
Expected: PASS

- [ ] **Step 9: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS. `TestKhoaNDCoDu` giờ soi cả `cua-hang.html` — khoá nào gõ sai sẽ lộ ra ở đây.

- [ ] **Step 10: Commit**

```bash
git add core/cuahang.go core/cuahang_test.go core/ui/cua-hang.html core/ui/cua-hang-xong.html core/ui/css.html core/server.go
git commit -m "feat: trang /cua-hang dat sua online"
```

---

## Task 9: Ảnh khách gửi kèm lúc đặt

**Files:**
- Create: `core/anhdon.go`
- Modify: `core/quantri.go:745-790` (`hQtThemAnh` — rút ruột vòng lặp), `core/cuahang.go`
- Modify: `core/csrf.go` (danh sách miễn CSRF theo tên đường dẫn)
- Test: `core/cuahang_test.go`

**Interfaces:**
- Consumes: `thuMucAnh(ma)` (`core/quantri.go:717`), `duoiAnh(dau []byte) string` (`:721`), `anhToiDaMoiDon = 16` (`:714`), `anhToiDaByte = 8 << 20` (`:715`), `maNgauNhien`, `KiemCSRFMultipart` (`core/csrf.go`)
- Produces: `func luuAnhVaoDon(don *Don, files []*multipart.FileHeader, nhan string) int`

Tách hàm vì cùng một vòng lặp giờ có hai chỗ gọi. Để nguyên hai bản sao là để dành ngày một bản sửa mà bản kia thì không.

- [ ] **Step 1: Viết test thất bại**

Thêm vào `core/cuahang_test.go`:

```go
// tepGia dựng một tệp tải lên giả để test không phải đi qua HTTP thật.
func tepGia(t *testing.T, truong, ten string, than []byte) *multipart.FileHeader {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, err := w.CreateFormFile(truong, ten)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(than); err != nil {
		t.Fatal(err)
	}
	w.Close()
	r := multipart.NewReader(&b, w.Boundary())
	form, err := r.ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { form.RemoveAll() })
	return form.File[truong][0]
}

func TestLuuAnhVaoDon(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-900", TrangThai: TTChoHangVe}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	// PNG tối thiểu: 8 byte chữ ký là đủ cho duoiAnh nhận ra.
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 32)...)
	fh := tepGia(t, "anh", "hong.png", png)

	if n := luuAnhVaoDon(d, []*multipart.FileHeader{fh}, "truoc"); n != 1 {
		t.Fatalf("phải lưu 1 ảnh, lưu %d", n)
	}
	if len(d.Anh) != 1 {
		t.Fatalf("đơn phải có 1 ảnh, có %d", len(d.Anh))
	}
	if !strings.HasPrefix(d.Anh[0], "truoc-") {
		t.Fatalf("tên ảnh phải mang nhãn truoc-, nhận %q", d.Anh[0])
	}
}

// Tệp không phải ảnh bị bỏ im lặng — không nổ, không lưu.
func TestBoQuaTepKhongPhaiAnh(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-901", TrangThai: TTChoHangVe}
	fh := tepGia(t, "anh", "virus.exe", []byte("MZ khong phai anh"))
	if n := luuAnhVaoDon(d, []*multipart.FileHeader{fh}, "truoc"); n != 0 {
		t.Fatalf("không được lưu tệp lạ, lưu %d", n)
	}
	if len(d.Anh) != 0 {
		t.Fatal("đơn không được dính tệp lạ")
	}
}
```

Thêm `bytes`, `mime/multipart` vào import của tệp test.

- [ ] **Step 2: Chạy test cho thấy fail**

Run: `go test ./core/ -run "TestLuuAnhVaoDon|TestBoQuaTep" -v`
Expected: FAIL — `undefined: luuAnhVaoDon`

- [ ] **Step 3: Viết `core/anhdon.go`**

Chuyển nguyên vòng lặp từ `hQtThemAnh` vào đây, không đổi logic:

```go
// Lưu ảnh vào thư mục của một đơn. Trước đây nằm trong hQtThemAnh; tách ra
// khi cửa hàng cần đúng vòng lặp ấy cho ảnh khách gửi lúc đặt. Hai bản sao
// của một vòng lặp lọc tệp là hai chỗ để quên sửa.
package core

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"
)

// luuAnhVaoDon ghi các tệp hợp lệ vào thư mục ảnh của đơn và nối tên vào
// don.Anh. Trả số ảnh đã lưu. KHÔNG gọi LuuDon — chỗ gọi tự quyết lúc nào ghi
// đơn xuống đĩa.
//
// Tệp không nhận ra được là ảnh thì BỎ IM LẶNG. Người gửi thường kèm cả ảnh
// chụp màn hình lẫn video; báo lỗi cả biểu mẫu chỉ vì một tệp lạ là chặn mất
// một đơn thật.
func luuAnhVaoDon(don *Don, files []*multipart.FileHeader, nhan string) int {
	if nhan != "sau" {
		nhan = "truoc"
	}
	them := 0
	for _, fh := range files {
		if len(don.Anh) >= anhToiDaMoiDon {
			break
		}
		f, err := fh.Open()
		if err != nil {
			continue
		}
		dau := make([]byte, 16)
		n, _ := io.ReadFull(f, dau)
		duoi := duoiAnh(dau[:n])
		if duoi == "" {
			f.Close()
			continue
		}
		if err := os.MkdirAll(thuMucAnh(don.Ma), 0o755); err != nil {
			f.Close()
			continue
		}
		ten := fmt.Sprintf("%s-%s-%s%s", nhan, time.Now().Format("150405"), maNgauNhien(3), duoi)
		out, err := os.Create(filepath.Join(thuMucAnh(don.Ma), ten))
		if err != nil {
			f.Close()
			continue
		}
		out.Write(dau[:n])
		io.Copy(out, io.LimitReader(f, anhToiDaByte))
		out.Close()
		f.Close()
		don.Anh = append(don.Anh, ten)
		them++
	}
	return them
}
```

Bản gốc đếm `len(don.Anh)+them >= anhToiDaMoiDon`; ở đây `don.Anh` được nối ngay trong vòng lặp nên `len(don.Anh) >= anhToiDaMoiDon` cho cùng kết quả.

- [ ] **Step 4: Gọi nó trong `hQtThemAnh`**

Thay cả vòng lặp bằng:

```go
	luuAnhVaoDon(don, r.MultipartForm.File["anh"], r.FormValue("nhan"))
	LuuDon(don)
```

- [ ] **Step 5: Nhận ảnh trong `hCuaHangGui`**

Biểu mẫu giờ có tệp nên phải đổi cách đọc thân yêu cầu. Thay khối `MaxBytesReader` + `ParseForm` bằng:

```go
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
	// 8MB giữ trong RAM, phần còn lại multipart tự ghi ra tệp tạm rồi dọn khi
	// request đóng.
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		cuaHangLoi(w, r, "Ảnh nặng quá ạ. Anh/chị gửi ít tệp hơn giúp em.")
		return
	}
	defer r.MultipartForm.RemoveAll()

	// Biểu mẫu có tệp: lớp bọc không đọc được thân yêu cầu nên không kiểm CSRF
	// hộ được. Xem core/csrf.go.
	if !KiemCSRFMultipart(w, r) {
		return
	}
```

Và sau `LuuDon(d)` thành công:

```go
	if r.MultipartForm != nil && luuAnhVaoDon(d, r.MultipartForm.File["anh"], "truoc") > 0 {
		LuuDon(d)
	}
```

Ảnh lưu **sau** khi đơn đã nằm an toàn trên đĩa: ảnh hỏng không được làm mất đơn.

- [ ] **Step 6: Thêm `/cua-hang` vào danh sách miễn CSRF theo tên**

Run: `grep -n "qt/giao-dien/logo" core/csrf.go`

Nối `"/cua-hang"` vào đúng nhánh `case` đó. `KiemCSRFMultipart` vẫn kiểm sau khi `ParseMultipartForm` chạy — miễn ở lớp bọc không phải là bỏ kiểm.

- [ ] **Step 7: Sửa test POST của Task 8**

`postForm` gửi `application/x-www-form-urlencoded`, mà handler giờ đòi multipart. Thêm helper và đổi các test cửa hàng sang dùng nó:

```go
// postDat gửi form đặt hàng dạng multipart, đúng như trình duyệt gửi.
func postDat(t *testing.T, v url.Values) *http.Request {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for k, vs := range v {
		for _, s := range vs {
			if err := w.WriteField(k, s); err != nil {
				t.Fatal(err)
			}
		}
	}
	w.Close()
	r := httptest.NewRequest(http.MethodPost, "/cua-hang", &b)
	r.Header.Set("Content-Type", w.FormDataContentType())
	return r
}
```

Đổi mọi `hCuaHangGui(w, postForm(...))` thành `hCuaHangGui(w, postDat(t, url.Values{...}))`.

`KiemCSRFMultipart` sẽ từ chối request test không có token. Xem cách `core/csrf_test.go` dựng cookie + trường `_csrf` hợp lệ rồi gắn đúng cách đó vào `postDat`.

- [ ] **Step 8: Chạy test cho thấy pass**

Run: `go test ./core/ -run "TestLuuAnh|TestBoQuaTep|TestDatHang|TestCuaHang|TestThieuLienHe|TestGiaLuon" -v`
Expected: PASS

- [ ] **Step 9: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 10: Commit**

```bash
git add core/anhdon.go core/cuahang.go core/cuahang_test.go core/quantri.go core/csrf.go
git commit -m "feat: khach gui anh cho hong ngay luc dat"
```

---

## Task 10: Chặn ca không nhận trước khi khách gửi hàng

**Files:**
- Modify: `core/cuahang.go`, `core/cuahang_test.go`

**Interfaces:**
- Consumes: `GiayChayBo`, `GiayDiLai` (`core/donhang.go:128`); `NguongHienTai()` (`core/nguong.go:39`) → `.NguongMoKenhShipDong` (`core/config.go:172`); khoá `cuahang.tuchoi.*` (T6); `dinhDangTien`
- Produces: `func LyDoTuChoiCuaHang(loai, kieuGiay string, giaTri int) string` — chuỗi rỗng = nhận

**Lệch với spec, cố ý.** Spec nói tra `vot.yaml → dong_vot[].nhan_sua`. Đọc code thì `NhanSua` là **chuỗi lời khuyên tự do** (`core/vot.go:119`, `core/giay.go:77`), không phải cờ đúng/sai — không có gì để máy quyết. Nên chặn bằng hai điều kiện máy biết chắc: kiểu giày, và ngưỡng mở kênh ship. Lời khuyên theo dòng vợt vẫn hiện cho khách đọc, nhưng không tự từ chối: từ chối nhầm là mất một khách thật.

- [ ] **Step 1: Viết test thất bại**

```go
func TestTuChoiGiaySaiKieu(t *testing.T) {
	moCuaHang(t)
	if LyDoTuChoiCuaHang(LoaiGiay, "court", 3000000) == "" {
		t.Fatal("giày court phải bị từ chối")
	}
	if got := LyDoTuChoiCuaHang(LoaiGiay, GiayChayBo, 3000000); got != "" {
		t.Fatalf("giày chạy bộ phải nhận, bị từ chối vì %q", got)
	}
	if got := LyDoTuChoiCuaHang(LoaiGiay, GiayDiLai, 3000000); got != "" {
		t.Fatalf("giày đi lại phải nhận, bị từ chối vì %q", got)
	}
	if LyDoTuChoiCuaHang(LoaiGiay, "", 3000000) == "" {
		t.Fatal("chưa chọn kiểu giày thì chưa nhận được")
	}
}

func TestTuChoiMonDuoiNguongShip(t *testing.T) {
	moCuaHang(t)
	ng := NguongHienTai().NguongMoKenhShipDong
	if ng <= 0 {
		t.Skip("chưa đặt ngưỡng mở kênh ship")
	}
	if LyDoTuChoiCuaHang(LoaiVot, "", ng-1) == "" {
		t.Fatalf("món dưới %d phải bị chặn", ng)
	}
	if got := LyDoTuChoiCuaHang(LoaiVot, "", ng); got != "" {
		t.Fatalf("đúng ngưỡng phải nhận, bị từ chối vì %q", got)
	}
}

// Không khai giá trị món thì KHÔNG chặn — thợ khám rồi mới biết, và bắt khách
// tự định giá cây vợt của họ là một cái cửa đóng vô cớ.
func TestKhongKhaiGiaTriThiVanNhan(t *testing.T) {
	moCuaHang(t)
	if got := LyDoTuChoiCuaHang(LoaiVot, "", 0); got != "" {
		t.Fatalf("không khai giá trị mà bị chặn: %q", got)
	}
}

func TestTuChoiThiKhongTaoDon(t *testing.T) {
	moCuaHang(t)
	w := httptest.NewRecorder()
	hCuaHangGui(w, postDat(t, url.Values{
		"loai":       {LoaiGiay},
		"dich_vu":    {"THAY_DE_GIAY"},
		"giay_hang":  {"Nike Court Vision"},
		"giay_kieu":  {"court"},
		"gia_tri":    {"3000000"},
		"tinh_trang": {"Mòn đế"},
		"ten":        {"Anh Nam"},
		"lien_he":    {"0900000002"},
		"dia_chi":    {"45 Lê Lợi"},
	}))

	if n := len(LocDon(BoLoc{TrangThai: TTChoHangVe})); n != 0 {
		t.Fatalf("ca từ chối mà vẫn tạo %d đơn", n)
	}
	if !strings.Contains(w.Body.String(), "chạy bộ") {
		t.Fatal("phải nói rõ lý do ngay trên trang, trước khi khách ra bưu điện")
	}
}
```

- [ ] **Step 2: Chạy test cho thấy fail**

Run: `go test ./core/ -run "TestTuChoi|TestKhongKhaiGiaTri" -v`
Expected: FAIL — `undefined: LyDoTuChoiCuaHang`

- [ ] **Step 3: Viết hàm trong `core/cuahang.go`**

```go
// LyDoTuChoiCuaHang — chuỗi rỗng nghĩa là nhận. Chặn ở đây tiết kiệm cho khách
// một chiều ship, và cho trạm khoản ship-ca-từ-chối (nguong.ship_ca_tu_choi_dong
// trong bang-gia.yaml).
//
// CHỈ TỪ CHỐI KHI MÁY BIẾT CHẮC. Hãng lạ, đời lạ, mô tả mơ hồ đều KHÔNG phải
// lý do — thợ khám rồi mới biết, và từ chối nhầm là mất một khách thật.
func LyDoTuChoiCuaHang(loai, kieuGiay string, giaTri int) string {
	if LoaiHopLe(loai) == LoaiGiay && kieuGiay != GiayChayBo && kieuGiay != GiayDiLai {
		return ND("cuahang.tuchoi.giay-kieu")
	}
	// giaTri == 0 nghĩa là khách không khai, không phải "món vô giá trị".
	if ng := NguongHienTai().NguongMoKenhShipDong; ng > 0 && giaTri > 0 && giaTri < ng {
		return ND("cuahang.tuchoi.duoi-nguong") + " (" + dinhDangTien(ng) + ")"
	}
	return ""
}
```

- [ ] **Step 4: Gọi nó trong `hCuaHangGui`, TRƯỚC khi tạo đơn**

Ngay sau khi đọc xong `giaTri` và `kieuGiay`, trước khi dựng `&Don{...}`:

```go
	if ly := LyDoTuChoiCuaHang(loai, kieuGiay, giaTri); ly != "" {
		cuaHangLoi(w, r, ly)
		return
	}
```

- [ ] **Step 5: Chạy test cho thấy pass**

Run: `go test ./core/ -run "TestTuChoi|TestKhongKhaiGiaTri|TestDatHang" -v`
Expected: PASS. `TestDatHangGiayMangTienToTG` đã gửi `giay_kieu: chay_bo` nên vẫn qua.

- [ ] **Step 6: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add core/cuahang.go core/cuahang_test.go
git commit -m "feat: chan ca khong nhan truoc khi khach gui hang"
```

---

## Task 11: Tra cứu bằng token

**Files:**
- Modify: `core/server.go:848-881` (`hTraCuu`), bảng route
- Test: `core/cuahang_test.go`

**Interfaces:**
- Consumes: `LayDonTheoToken` (`core/donhang.go:405`), `Don.Token`
- Produces: route `GET /tra-cuu/{token}`

Màn kết hứa một đường dẫn. Hôm nay tra cứu chỉ có POST mã + 4 số cuối điện thoại (`TraCuuChoKhach`, `core/donhang.go:497`) — không mở được bằng một cú bấm. Token đã có sẵn trên đơn và đã đủ tin để phục vụ ảnh (`hAnhDonChoKhach`, `core/server.go:825`), nên dùng đúng nó.

- [ ] **Step 1: Viết test thất bại**

```go
func TestTraCuuBangToken(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-910", Token: "abc12345", TrangThai: TTChoHangVe, KhachTen: "Chị Lan"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tra-cuu/abc12345", nil)
	r.SetPathValue("token", "abc12345")
	hTraCuu(w, r)

	if !strings.Contains(w.Body.String(), "TV-2609-910") {
		t.Fatal("mở bằng token phải ra thẳng đơn")
	}
}

func TestTokenSaiKhongLoDon(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-911", Token: "abc12345", TrangThai: TTChoHangVe, KhachTen: "Chị Lan"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tra-cuu/sai00000", nil)
	r.SetPathValue("token", "sai00000")
	hTraCuu(w, r)

	if strings.Contains(w.Body.String(), "TV-2609-911") {
		t.Fatal("token sai mà vẫn lộ đơn")
	}
}
```

- [ ] **Step 2: Chạy test cho thấy fail**

Run: `go test ./core/ -run "TestTraCuuBangToken|TestTokenSai" -v`
Expected: FAIL — trang không chứa mã đơn

- [ ] **Step 3: Sửa `hTraCuu`**

Ngay sau khi dựng biến dữ liệu trang và chọn mẫu, trước nhánh POST:

```go
	// Mở bằng token: đường dẫn màn kết của cửa hàng đưa cho khách. Token 8 ký
	// tự ngẫu nhiên, cùng cửa đã dùng cho ảnh đơn (hAnhDonChoKhach). Không
	// giới hạn tần suất ở đây: token không dò được như mã đơn theo số thứ tự.
	if tok := r.PathValue("token"); tok != "" {
		if don, ok := LayDonTheoToken(tok); ok {
			d.Don = don
			d.Ma = don.Ma
		} else {
			d.Loi = "Đường dẫn này không còn đúng. Anh/chị tra bằng mã đơn và 4 số cuối điện thoại giúp em."
		}
		render(w, mau, d)
		return
	}
```

Tên biến (`d`, `mau`) và tên trường (`Don`, `Ma`, `Loi`) lấy theo bản thật của `hTraCuu` — `sed -n '848,881p' core/server.go`. Kiểu trả về của `LayDonTheoToken` phải khớp kiểu trường `Don` trong struct dữ liệu trang.

- [ ] **Step 4: Đăng ký route**

```go
	mux.HandleFunc("GET /tra-cuu/{token}", hTraCuu)
```

Có bản `/app/tra-cuu` thì thêm dòng song song cho nó — `grep -n "tra-cuu" core/server.go`.

- [ ] **Step 5: Chạy test cho thấy pass**

Run: `go test ./core/ -run "TestTraCuuBangToken|TestTokenSai" -v`
Expected: PASS

- [ ] **Step 6: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add core/server.go core/cuahang_test.go
git commit -m "feat: mo trang tra cuu bang token tren duong dan"
```

---

## Task 12: Thanh toán — VietQR và COD

**Files:**
- Create: `core/vietqr.go`, `core/vietqr_test.go`
- Modify: `core/config.go:25-36` (`LienHe`), `core/lienhe.go:64-75` (`oLienHe`), `core/server.go` (`hTraCuu`, route), `core/cuahang.go`, `core/ui/tracuu.html`

**Interfaces:**
- Consumes: `reMaDon` (`core/saoke.go:25`); `Don.ConNo()`, `Don.TongTien`, `TTXong`, `TraQR`, `TraCOD` (T7); khoá `cuahang.tra.*` (T6); `LienHeHienTai()`, `LayDonTheoToken`, `NguongHienTai().FreeShipVeTuDong`
- Produces: `LienHe.NganHangMa`, `.SoTaiKhoan`, `.ChuTaiKhoan`; `type NganHang struct{ Ma, SoTK, ChuTK string }`; `func NganHangNhan() (NganHang, bool)`; `func NoiDungCK(maDon string) string`; `func ChuoiVietQR(nh NganHang, soTien int, noiDung string) string`; `func crc16CCITT(s string) string`; `func hChonHinhThucTra(w,r)`; route `POST /tra-cuu/{token}/hinh-thuc-tra`

- [ ] **Step 1: Kiểm chưa có chỗ khai tài khoản**

Run: `grep -rn "so_tk\|SoTaiKhoan\|ngan_hang\|NganHang" core/*.go config.yaml | head`

Đã có thì dùng cái có sẵn và bỏ Step 3.

- [ ] **Step 2: Viết test thất bại**

Tạo `core/vietqr_test.go`:

```go
package core

import (
	"strings"
	"testing"
)

// Nội dung chuyển khoản phải khớp được bằng CHÍNH cái regex saoke.go dùng.
// Đây là toàn bộ lý do QR tồn tại: tiền về là khớp được đơn.
func TestNoiDungCKKhopDuocBangReMaDon(t *testing.T) {
	for _, ma := range []string{"TV-2609-001", "TV-2612-045"} {
		nd := NoiDungCK(ma)
		if !reMaDon.MatchString(nd) {
			t.Fatalf("nội dung %q không khớp reMaDon — sao kê sẽ không nhận ra đơn %s", nd, ma)
		}
	}
}

// Ngân hàng nuốt dấu, viết hoa, chèn thêm chữ — nội dung vẫn phải khớp.
func TestNoiDungCKKhopKhiBiNganHangLamBien(t *testing.T) {
	nd := NoiDungCK("TV-2609-001")
	for _, bien := range []string{
		nd,
		strings.ReplaceAll(nd, "-", " "),
		strings.ReplaceAll(nd, "-", ""),
		"CK tu 0900000000 " + nd,
		"CHUYEN TIEN " + nd + " GD 123456",
	} {
		if !reMaDon.MatchString(bien) {
			t.Fatalf("không khớp: %q", bien)
		}
	}
}

// Vector chuẩn của CRC-16/CCITT-FALSE. Sai hàm này thì app ngân hàng báo "mã
// QR không hợp lệ" và không ai biết vì sao.
func TestCRC16CCITT(t *testing.T) {
	if got := crc16CCITT("123456789"); got != "29B1" {
		t.Fatalf(`crc16CCITT("123456789") = %q, chờ "29B1"`, got)
	}
}

func TestChuoiVietQRCoDuBaThu(t *testing.T) {
	nh := NganHang{Ma: "970436", SoTK: "1234567890", ChuTK: "NGUYEN VAN A"}
	s := ChuoiVietQR(nh, 250000, NoiDungCK("TV-2609-001"))
	if s == "" {
		t.Fatal("chuỗi QR rỗng")
	}
	for _, phai := range []string{"970436", "1234567890", "TV-2609-001", "704"} {
		if !strings.Contains(s, phai) {
			t.Fatalf("chuỗi QR thiếu %q:\n%s", phai, s)
		}
	}
	// 4 ký tự cuối là CRC của toàn chuỗi kể cả "6304".
	if len(s) < 8 || s[len(s)-8:len(s)-4] != "6304" {
		t.Fatalf("chuỗi QR không kết bằng 6304+CRC: %q", s[len(s)-8:])
	}
	if crc16CCITT(s[:len(s)-4]) != s[len(s)-4:] {
		t.Fatal("CRC cuối chuỗi không khớp phần đầu")
	}
}

func TestThieuTaiKhoanThiKhongCoQR(t *testing.T) {
	if s := ChuoiVietQR(NganHang{}, 250000, "TV-2609-001"); s != "" {
		t.Fatalf("thiếu tài khoản mà vẫn sinh QR: %q", s)
	}
}
```

- [ ] **Step 3: Thêm ba trường tài khoản vào `LienHe`**

Trong `core/config.go`:

```go
	// Tài khoản nhận tiền, hiện trên trang tra cứu khi đơn đã xong. Để trống
	// thì trang chỉ còn COD.
	NganHangMa  string `yaml:"ngan_hang_ma"` // mã BIN VietQR, ví dụ 970436
	SoTaiKhoan  string `yaml:"so_tai_khoan"`
	ChuTaiKhoan string `yaml:"chu_tai_khoan"`
```

Và ba ô nhập trong `oLienHe` (`core/lienhe.go:64`), theo đúng thứ tự trường của các dòng đang có:

```go
	{"ngan_hang_ma", "Mã ngân hàng (BIN)", "6 số, ví dụ 970436 là Vietcombank — tra ở vietqr.io/danh-sach-ngan-hang", 10, func(l *LienHe) *string { return &l.NganHangMa }},
	{"so_tai_khoan", "Số tài khoản", "Số tài khoản nhận tiền của trạm", 30, func(l *LienHe) *string { return &l.SoTaiKhoan }},
	{"chu_tai_khoan", "Chủ tài khoản", "Tên không dấu, viết hoa, đúng như trên sổ", 60, func(l *LienHe) *string { return &l.ChuTaiKhoan }},
```

Ba ô này tự hiện ở `/qt/lien-he` — trang ấy dựng từ `oLienHe`, không gõ tay trong template. Kiểm bằng `grep -n "oLienHe" core/lienhe.go core/ui/qt-lienhe.html`.

- [ ] **Step 4: Viết `core/vietqr.go`**

```go
// Chuyển khoản cho khách trả tiền đơn.
//
// CẢ TỆP NÀY TỒN TẠI VÌ MỘT CÂU: nội dung chuyển khoản phải mang mã đơn, vì
// saoke.go khớp tiền về đơn bằng đúng cái mã đó (reMaDon, saoke.go:25). Mất mã
// trong nội dung là tiền về mà không biết của ai — và người phải đi dò là
// Kendy.
//
// KHÔNG gọi dịch vụ sinh ảnh QR bên ngoài. Trang khách nạp ảnh từ máy chủ lạ
// là lộ mã đơn và số tiền của khách cho bên thứ ba, và ngày họ sập thì nút trả
// tiền trắng bóc. Sinh chuỗi tại chỗ; cho tới khi có thư viện vẽ QR trong
// binary thì trang hiện số tài khoản và nội dung dạng chữ để khách tự nhập.
package core

import (
	"fmt"
	"strings"
)

type NganHang struct {
	Ma    string // mã BIN VietQR, ví dụ 970436
	SoTK  string
	ChuTK string
}

// NoiDungCK — nội dung chuyển khoản. CHỈ mã đơn, không thêm chữ có dấu: nhiều
// app ngân hàng cắt dấu và cắt độ dài, chữ thừa đẩy mã ra khỏi khung.
func NoiDungCK(maDon string) string { return strings.ToUpper(strings.TrimSpace(maDon)) }

// NganHangNhan đọc tài khoản nhận tiền. Thiếu một trong ba thì coi như chưa
// khai — nút chuyển khoản không hiện, chỉ còn COD.
func NganHangNhan() (NganHang, bool) {
	l := LienHeHienTai()
	nh := NganHang{
		Ma:    strings.TrimSpace(l.NganHangMa),
		SoTK:  strings.TrimSpace(l.SoTaiKhoan),
		ChuTK: strings.TrimSpace(l.ChuTaiKhoan),
	}
	if nh.Ma == "" || nh.SoTK == "" || nh.ChuTK == "" {
		return NganHang{}, false
	}
	return nh, true
}

// tlv — một trường EMVCo: mã 2 ký tự, độ dài 2 chữ số, rồi giá trị.
func tlv(id, gia string) string { return fmt.Sprintf("%s%02d%s", id, len(gia), gia) }

// ChuoiVietQR — chuỗi nhúng vào mã QR, theo EMVCo/VietQR. Rỗng nếu thiếu dữ
// liệu: thà không có nút còn hơn có một mã quét ra lỗi.
func ChuoiVietQR(nh NganHang, soTien int, noiDung string) string {
	if nh.Ma == "" || nh.SoTK == "" || soTien <= 0 {
		return ""
	}
	thongTin := tlv("00", "A000000727") +
		tlv("01", tlv("00", nh.Ma)+tlv("01", nh.SoTK)) +
		tlv("02", "QRIBFTTA")
	s := tlv("00", "01") +
		tlv("01", "12") + // 12 = QR dùng một lần, có sẵn số tiền
		tlv("38", thongTin) +
		tlv("53", "704") + // VND
		tlv("54", fmt.Sprintf("%d", soTien)) +
		tlv("58", "VN") +
		tlv("62", tlv("08", noiDung))
	s += "6304" // mã và độ dài của chính trường CRC, tính vào phép CRC
	return s + crc16CCITT(s)
}

// crc16CCITT — CRC-16/CCITT-FALSE: đa thức 0x1021, khởi tạo 0xFFFF, không đảo
// bit, không XOR ra. Đúng biến thể EMVCo yêu cầu; mấy biến thể CRC-16 khác cho
// ra số khác và app ngân hàng sẽ từ chối mã.
func crc16CCITT(s string) string {
	crc := uint16(0xFFFF)
	for i := 0; i < len(s); i++ {
		crc ^= uint16(s[i]) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}
```

- [ ] **Step 5: Chạy test cho thấy pass**

Run: `go test ./core/ -run "TestNoiDungCK|TestCRC16|TestChuoiVietQR|TestThieuTaiKhoan" -v`
Expected: PASS

- [ ] **Step 6: Đưa khối thanh toán vào `hTraCuu`**

Thêm vào struct dữ liệu trang của `hTraCuu`:

```go
		// Khối thanh toán, chỉ dựng khi đơn đã xong mà sổ tiền chưa đủ.
		CanTra    bool
		SoTienTra int
		NganHang  NganHang
		CoQR      bool
		NoiDungCK string
		ChuoiQR   string
		FreeShip  bool
```

Ngay trước mỗi `render` cuối hàm (gọn nhất là gom vào một hàm nhỏ rồi gọi ở cả nhánh token lẫn nhánh POST):

```go
	// "Đã trả tiền chưa" hỏi SỔ TIỀN, không hỏi một trường trên đơn. Xem chú
	// thích trên Don.DaThu.
	if d.Don != nil && d.Don.TrangThai == TTXong && d.Don.ConNo() > 0 {
		d.CanTra = true
		d.SoTienTra = d.Don.ConNo()
		d.NoiDungCK = NoiDungCK(d.Don.Ma)
		d.NganHang, d.CoQR = NganHangNhan()
		if d.CoQR {
			d.ChuoiQR = ChuoiVietQR(d.NganHang, d.SoTienTra, d.NoiDungCK)
		}
		if ng := NguongHienTai().FreeShipVeTuDong; ng > 0 && d.Don.TongTien >= ng {
			d.FreeShip = true
		}
	}
```

`NguongHienTai()` ở `core/nguong.go:39`; `FreeShipVeTuDong` ở `core/config.go:178`.

- [ ] **Step 7: Ghi lựa chọn hình thức trả**

Spec đòi đơn phải đánh dấu `HinhThucTra`. Thêm vào `core/cuahang.go`:

```go
// hChonHinhThucTra — khách bấm "tôi sẽ chuyển khoản" hay "tôi trả khi nhận".
// Chỉ ghi Ý ĐỊNH, không ghi tiền: tiền vào sổ khi nó thật sự về, không phải
// khi khách bấm nút.
func hChonHinhThucTra(w http.ResponseWriter, r *http.Request) {
	tok := r.PathValue("token")
	don, ok := LayDonTheoToken(tok)
	if !ok {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	r.ParseForm()
	switch r.FormValue("cach") {
	case TraQR:
		don.HinhThucTra = TraQR
	case TraCOD:
		don.HinhThucTra = TraCOD
	default:
		http.Redirect(w, r, "/tra-cuu/"+tok, http.StatusSeeOther)
		return
	}
	don.LichSu = append(don.LichSu, Moc{
		Luc:       time.Now().Format("2006-01-02 15:04:05"),
		TrangThai: don.TrangThai,
		Nguoi:     "khách",
		GhiChu:    "Chọn trả bằng " + don.HinhThucTra,
	})
	LuuDon(don)
	http.Redirect(w, r, "/tra-cuu/"+tok, http.StatusSeeOther)
}
```

`LayDonTheoToken` trả bản sao hay con trỏ vào cache thì `LuuDon` đều ghi đúng — nhưng kiểm chữ ký để dùng đúng kiểu: `grep -n "func LayDonTheoToken" -A 8 core/donhang.go`.

Route cạnh các route tra cứu:

```go
	mux.HandleFunc("POST /tra-cuu/{token}/hinh-thuc-tra", hChonHinhThucTra)
```

- [ ] **Step 8: Sửa `core/ui/tracuu.html`**

```html
{{if .CanTra}}
<section class="tra-tien">
  <h2>{{nd "cuahang.tra.tieu-de"}}</h2>
  <p class="so-tien">{{tien .SoTienTra}}</p>
  {{if .FreeShip}}{{ndm "cuahang.tra.free-ship"}}{{end}}

  {{if .CoQR}}
    <h3>{{nd "cuahang.tra.qr"}}</h3>
    <dl>
      <dt>Ngân hàng</dt><dd>{{.NganHang.Ma}}</dd>
      <dt>Số tài khoản</dt><dd>{{.NganHang.SoTK}}</dd>
      <dt>Chủ tài khoản</dt><dd>{{.NganHang.ChuTK}}</dd>
      <dt>Số tiền</dt><dd>{{tien .SoTienTra}}</dd>
      <dt>Nội dung</dt><dd><strong>{{.NoiDungCK}}</strong></dd>
    </dl>
    {{ndm "cuahang.tra.qr-nhac"}}
    <form method="post" action="/tra-cuu/{{.Don.Token}}/hinh-thuc-tra">
      <input type="hidden" name="_csrf" value="{{.CSRF}}">
      <input type="hidden" name="cach" value="qr">
      <button type="submit">Tôi sẽ chuyển khoản</button>
    </form>
  {{end}}

  <h3>{{nd "cuahang.tra.cod"}}</h3>
  {{ndm "cuahang.tra.cod-nhac"}}
  <form method="post" action="/tra-cuu/{{.Don.Token}}/hinh-thuc-tra">
    <input type="hidden" name="_csrf" value="{{.CSRF}}">
    <input type="hidden" name="cach" value="cod">
    <button type="submit">Tôi trả khi nhận hàng</button>
  </form>
</section>
{{end}}
```

`{{tien .SoTienTra}}` phải là tên hàm định dạng tiền thật đã đăng ký trong `FuncMap` — `grep -n "dinhDangTien" core/server.go`. Không in số thô.

- [ ] **Step 9: Test khối thanh toán**

```go
func TestTraCuuHienTienKhiXongVaConNo(t *testing.T) {
	moCuaHang(t)
	d := &Don{
		Ma: "TV-2609-777", Token: "abc12777", TrangThai: TTXong,
		KhachTen: "Chị Lan", KhachLienHe: "0900000000", TongTien: 250000,
	}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tra-cuu/abc12777", nil)
	r.SetPathValue("token", "abc12777")
	hTraCuu(w, r)

	than := w.Body.String()
	if !strings.Contains(than, "TV-2609-777") {
		t.Fatal("nội dung chuyển khoản phải hiện mã đơn để khách chép đúng")
	}
	if !strings.Contains(than, ND("cuahang.tra.cod")) {
		t.Fatal("thiếu lựa chọn COD")
	}
}

// Đơn chưa xong thì không được đòi tiền.
func TestChuaXongThiKhongDoiTien(t *testing.T) {
	moCuaHang(t)
	d := &Don{
		Ma: "TV-2609-778", Token: "abc12778", TrangThai: TTDangSua,
		KhachTen: "Chị Lan", TongTien: 250000,
	}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tra-cuu/abc12778", nil)
	r.SetPathValue("token", "abc12778")
	hTraCuu(w, r)

	if strings.Contains(w.Body.String(), ND("cuahang.tra.tieu-de")) {
		t.Fatal("đơn đang sửa mà đã hiện khối thanh toán")
	}
}

// Bấm "tôi sẽ chuyển khoản" phải ghi vào đơn — spec đòi đơn nhớ ý định này.
func TestGhiHinhThucTra(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-779", Token: "abc12779", TrangThai: TTXong, TongTien: 250000}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/tra-cuu/abc12779/hinh-thuc-tra", url.Values{"cach": {TraQR}})
	r.SetPathValue("token", "abc12779")
	hChonHinhThucTra(httptest.NewRecorder(), r)

	sau, co := LayDon("TV-2609-779")
	if !co {
		t.Fatal("mất đơn")
	}
	if sau.HinhThucTra != TraQR {
		t.Fatalf("chưa ghi hình thức trả: %q", sau.HinhThucTra)
	}
}
```

Tên hằng trạng thái đang sửa (`TTDangSua`) lấy theo bản thật — `grep -n "TT[A-Z]" core/donhang.go | head -20`.

Run: `go test ./core/ -run "TestTraCuuHienTien|TestChuaXongThiKhong|TestGhiHinhThucTra" -v`
Expected: PASS

- [ ] **Step 10: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 11: Commit**

```bash
git add core/vietqr.go core/vietqr_test.go core/config.go core/lienhe.go core/cuahang.go core/cuahang_test.go core/server.go core/ui/tracuu.html
git commit -m "feat: khoi thanh toan QR va COD tren trang tra cuu"
```

---

## Task 13: Bấm nhận hàng, và hai bộ lọc mới trên bảng đơn

**Files:**
- Modify: `core/quantri.go` (handler đơn, bảng route), `core/donhang.go:448-457` (`BoLoc`, `LocDon`), `core/cuahang.go`, `core/server.go` (route tra cứu)
- Modify: `core/ui/qt-don.html`, `core/ui/qt-don-ct.html`, `core/ui/tracuu.html`
- Test: `core/qtdon_test.go` (tạo mới)

**Interfaces:**
- Consumes: `TTChoHangVe`, `Don.MaVanDonDen`, `Don.ChoHangVe()`, `NguonWeb` (T7); `LayDon` (`core/donhang.go:390`), `LuuDon`, `NguoiDangNhap(r) (NguoiDung, bool)` (`core/auth.go`), `canLaChu`, `urlEsc`, `catBot`
- Produces: `func hQtNhanHang(w,r)` + route `POST /qt/don/{ma}/nhan-hang`; `func hKhachBaoVanDon(w,r)` + route `POST /tra-cuu/{token}/van-don`; `func hQtVanDonVe(w,r)` + route `POST /qt/don/{ma}/van-don-ve`; `BoLoc.ChuaThuDu bool`

Không có bước này thì đơn đặt online kẹt mãi ở `cho_hang_ve`: cửa hàng chạy được nhưng trạm không dùng được.

- [ ] **Step 1: Viết test thất bại**

Tạo `core/qtdon_test.go`:

```go
package core

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNhanHangChuyenSangMoi(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-888", Token: "tok12888", TrangThai: TTChoHangVe, KhachTen: "Chị Lan"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/qt/don/TV-2609-888/nhan-hang", url.Values{"ma_van_don": {"GHTK123456"}})
	r.SetPathValue("ma", "TV-2609-888")
	hQtNhanHang(httptest.NewRecorder(), r)

	sau, co := LayDon("TV-2609-888")
	if !co {
		t.Fatal("mất đơn")
	}
	if sau.TrangThai != TTMoi {
		t.Fatalf("nhận hàng xong phải sang %s, đang ở %s", TTMoi, sau.TrangThai)
	}
	if sau.MaVanDonDen != "GHTK123456" {
		t.Fatalf("chưa ghi mã vận đơn: %q", sau.MaVanDonDen)
	}
	if len(sau.LichSu) == 0 {
		t.Fatal("phải để lại một mốc trong lịch sử")
	}
}

// Bấm nhầm trên đơn đang sửa không được kéo nó ngược về "mới nhận".
func TestNhanHangChiChayOChoHangVe(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-889", TrangThai: TTDangSua, KhachTen: "Anh Nam"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/qt/don/TV-2609-889/nhan-hang", url.Values{})
	r.SetPathValue("ma", "TV-2609-889")
	hQtNhanHang(httptest.NewRecorder(), r)

	sau, _ := LayDon("TV-2609-889")
	if sau.TrangThai != TTDangSua {
		t.Fatalf("trạng thái bị kéo về %s", sau.TrangThai)
	}
}

// Lọc "sửa xong, chưa thu đủ" đọc SỔ TIỀN qua ConNo(), không đọc trường nào
// trên đơn.
func TestLocChuaThuDu(t *testing.T) {
	moCuaHang(t)
	if err := LuuDon(&Don{Ma: "TV-2609-890", TrangThai: TTXong, TongTien: 250000}); err != nil {
		t.Fatal(err)
	}
	if err := LuuDon(&Don{Ma: "TV-2609-891", TrangThai: TTXong, TongTien: 0}); err != nil {
		t.Fatal(err)
	}

	ds := LocDon(BoLoc{TrangThai: TTXong, ChuaThuDu: true})
	if len(ds) != 1 || ds[0].Ma != "TV-2609-890" {
		t.Fatalf("lọc sai: %v", ds)
	}
}
```

Handler chạy dưới `canLaChu` nên nó gọi `NguoiDangNhap(r)`. Gọi thẳng handler thì `NguoiDangNhap` trả `ok == false` và `NguoiDung` rỗng — kiểm xem `hQtNhanHang` có chịu được không, hoặc dựng cookie phiên theo cách `core/auth_test.go` đang làm.

- [ ] **Step 2: Chạy test cho thấy fail**

Run: `go test ./core/ -run "TestNhanHang|TestLocChuaThuDu" -v`
Expected: FAIL — `undefined: hQtNhanHang`

- [ ] **Step 3: Thêm `ChuaThuDu` vào `BoLoc` và `LocDon`**

```go
type BoLoc struct {
	Loai        string
	TrangThai   string
	Tho         string
	DoiTac      string
	Tim         string
	ChiDangChay bool
	// ChuaThuDu: đơn còn nợ tiền. Tính từ SỔ TIỀN qua ConNo() — cố tình không
	// có trường "đã trả" trên đơn để đọc. Xem chú thích trên Don.DaThu.
	ChuaThuDu bool
}
```

Trong `LocDon`, cạnh mấy nhánh `continue` khác:

```go
		if f.ChuaThuDu && d.ConNo() <= 0 {
			continue
		}
```

- [ ] **Step 4: Viết `hQtNhanHang` trong `core/quantri.go`**

Đặt cạnh handler đổi trạng thái đang có (`grep -n "trang-thai" core/quantri.go`) và dùng đúng lối kiểm quyền của nó:

```go
// hQtNhanHang — kiện của đơn đặt online đã tới tay trạm. Một cú bấm, và từ đây
// đơn chảy đúng luồng cũ.
func hQtNhanHang(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	ma := strings.ToUpper(r.PathValue("ma"))
	don, ok := LayDon(ma)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if don.TrangThai != TTChoHangVe {
		http.Redirect(w, r, "/qt/don/"+ma+"?loi="+urlEsc("Đơn này không ở trạng thái chờ hàng về"), http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	don.MaVanDonDen = catBot(r.FormValue("ma_van_don"), 60)
	don.TrangThai = TTMoi
	don.LichSu = append(don.LichSu, Moc{
		Luc:       time.Now().Format("2006-01-02 15:04:05"),
		TrangThai: TTMoi,
		Nguoi:     nd.Ten,
		GhiChu:    "Đã nhận kiện hàng",
	})
	if err := LuuDon(don); err != nil {
		http.Redirect(w, r, "/qt/don/"+ma+"?loi="+urlEsc("Không lưu được"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/don/"+ma+"?ok="+urlEsc("Đã nhận hàng"), http.StatusSeeOther)
}
```

Hai chỗ phải kiểm trước khi chạy: tên trường tên người trên `NguoiDung` (`grep -n "type NguoiDung struct" -A 8 core/auth.go`), và handler đơn khác có lọc quyền xem đơn theo thợ không — có thì thêm đúng lời gọi ấy vào sau `LayDon`.

- [ ] **Step 5: Đăng ký route cạnh các route đơn**

```go
	mux.HandleFunc("POST /qt/don/{ma}/nhan-hang", canLaChu(hQtNhanHang))
```

- [ ] **Step 6: Thêm nút vào `core/ui/qt-don-ct.html`**

Chỉ hiện khi đơn đang `cho_hang_ve`:

```html
{{if .Don.ChoHangVe}}
<form method="post" action="/qt/don/{{.Don.Ma}}/nhan-hang" class="nhan-hang">
  <input type="hidden" name="_csrf" value="{{$.CSRF}}">
  <p>Địa chỉ trả về: {{.Don.KhachDiaChi}}</p>
  <label>Mã vận đơn (nếu có) <input name="ma_van_don" maxlength="60"></label>
  <button type="submit">Đã nhận kiện hàng</button>
</form>
{{end}}
```

Biến trang có thể không phải `$` — mở tệp xem `{{range}}`/`{{with}}` đang lồng thế nào.

- [ ] **Step 7: Thêm hai bộ lọc vào bảng đơn**

- Tab **"Chờ hàng về"** — `BoLoc{TrangThai: TTChoHangVe}`. Tự hiện nếu danh sách tab dựng từ `CacTrangThai`; kiểm bằng `grep -n "CacTrangThai" core/quantri.go core/ui/qt-don.html`. Không tự hiện thì thêm tay.
- Tab **"Sửa xong, chưa thu đủ"** — `BoLoc{TrangThai: TTXong, ChuaThuDu: true}`, kèm tham số truy vấn riêng (ví dụ `?loc=con-no`) vì nó không phải một trạng thái. Đọc tham số ấy trong handler danh sách đơn rồi bật `ChuaThuDu`.
- Cột nguồn đơn: `{{if eq .NguonDonHopLe "web"}}<span class="nhan-nho">web</span>{{end}}` để nhìn ra ngay đơn nào đặt trên web.

- [ ] **Step 8: Khách tự điền mã vận đơn gửi tới**

Spec cho hai đường vào `MaVanDonDen`: thợ ghi khi nhận (Step 4), hoặc khách tự
điền. Đường thứ hai mới là đường có ích: lúc khách vừa ở bưu điện về, trạm chưa
thấy kiện đâu — mã ấy là thứ duy nhất trả lời được "hàng em đến đâu rồi".

Thêm vào `core/cuahang.go`:

```go
// hKhachBaoVanDon — khách gửi hàng xong, dán mã vận đơn vào đơn. Chỉ nhận khi
// đơn còn đang chờ hàng về: kiện đã tới tay trạm rồi thì mã ấy không còn nói
// thêm được gì, mà vẫn là một ô cho người lạ ghi đè.
func hKhachBaoVanDon(w http.ResponseWriter, r *http.Request) {
	tok := r.PathValue("token")
	don, ok := LayDonTheoToken(tok)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !don.ChoHangVe() {
		http.Redirect(w, r, "/tra-cuu/"+tok, http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	r.ParseForm()
	ma := catBot(strings.TrimSpace(r.FormValue("ma_van_don")), 60)
	if ma == "" {
		http.Redirect(w, r, "/tra-cuu/"+tok, http.StatusSeeOther)
		return
	}
	don.MaVanDonDen = ma
	don.LichSu = append(don.LichSu, Moc{
		Luc:       time.Now().Format("2006-01-02 15:04:05"),
		TrangThai: don.TrangThai,
		Nguoi:     "khách",
		GhiChu:    "Khách báo đã gửi, vận đơn " + ma,
	})
	LuuDon(don)
	http.Redirect(w, r, "/tra-cuu/"+tok, http.StatusSeeOther)
}
```

Route cạnh `hChonHinhThucTra`:

```go
	mux.HandleFunc("POST /tra-cuu/{token}/van-don", hKhachBaoVanDon)
```

Trong `core/ui/tracuu.html`, trên khối thanh toán:

```html
{{if and .Don .Don.ChoHangVe}}
<section class="bao-van-don">
  <h2>Đã gửi hàng rồi?</h2>
  <p>Dán mã vận đơn vào đây để trạm biết kiện đang trên đường.</p>
  <form method="post" action="/tra-cuu/{{.Don.Token}}/van-don">
    <input type="hidden" name="_csrf" value="{{.CSRF}}">
    <input name="ma_van_don" maxlength="60" value="{{.Don.MaVanDonDen}}" placeholder="VD: GHTK123456">
    <button type="submit">Lưu mã vận đơn</button>
  </form>
</section>
{{end}}
```

Test trong `core/qtdon_test.go`:

```go
func TestKhachBaoVanDon(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-892", Token: "tok12892", TrangThai: TTChoHangVe}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/tra-cuu/tok12892/van-don", url.Values{"ma_van_don": {"GHTK999"}})
	r.SetPathValue("token", "tok12892")
	hKhachBaoVanDon(httptest.NewRecorder(), r)

	sau, _ := LayDon("TV-2609-892")
	if sau.MaVanDonDen != "GHTK999" {
		t.Fatalf("chưa ghi vận đơn khách báo: %q", sau.MaVanDonDen)
	}
}

// Đơn đã qua cho_hang_ve thì ô này đóng — không để người cầm link ghi đè.
func TestKhachKhongSuaVanDonSauKhiNhan(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-893", Token: "tok12893", TrangThai: TTDangSua, MaVanDonDen: "GHTK111"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/tra-cuu/tok12893/van-don", url.Values{"ma_van_don": {"GHTK222"}})
	r.SetPathValue("token", "tok12893")
	hKhachBaoVanDon(httptest.NewRecorder(), r)

	sau, _ := LayDon("TV-2609-893")
	if sau.MaVanDonDen != "GHTK111" {
		t.Fatalf("bị ghi đè thành %q", sau.MaVanDonDen)
	}
}
```

- [ ] **Step 9: Ghi mã vận đơn chiều về**

`MaVanDonVe` khai ở Task 7 mà chưa chỗ nào ghi. Một trường không ai điền là một
trường sẽ mục ra rồi có ngày ai đó tin nó.

Trong `core/quantri.go`, cạnh `hQtNhanHang`:

```go
// hQtVanDonVe — thợ gửi đồ về, ghi lại mã để khách theo dõi. Không đổi trạng
// thái: gửi đi và giao xong là hai chuyện, máy trạng thái cũ đã có TTDaGiao.
func hQtVanDonVe(w http.ResponseWriter, r *http.Request) {
	ma := strings.ToUpper(r.PathValue("ma"))
	don, ok := LayDon(ma)
	if !ok {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	don.MaVanDonVe = catBot(strings.TrimSpace(r.FormValue("ma_van_don_ve")), 60)
	if err := LuuDon(don); err != nil {
		http.Redirect(w, r, "/qt/don/"+ma+"?loi="+urlEsc("Không lưu được"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/don/"+ma+"?ok="+urlEsc("Đã ghi vận đơn về"), http.StatusSeeOther)
}
```

Route:

```go
	mux.HandleFunc("POST /qt/don/{ma}/van-don-ve", canLaChu(hQtVanDonVe))
```

Ô nhập trong `core/ui/qt-don-ct.html`, chỉ hiện với đơn có địa chỉ trả về:

```html
{{if .Don.KhachDiaChi}}
<form method="post" action="/qt/don/{{.Don.Ma}}/van-don-ve">
  <input type="hidden" name="_csrf" value="{{$.CSRF}}">
  <label>Mã vận đơn gửi về <input name="ma_van_don_ve" maxlength="60" value="{{.Don.MaVanDonVe}}"></label>
  <button type="submit">Lưu</button>
</form>
{{end}}
```

Và hiện lại cho khách ở `core/ui/tracuu.html`:

```html
{{if .Don.MaVanDonVe}}<p>Mã vận đơn gửi về: <strong>{{.Don.MaVanDonVe}}</strong></p>{{end}}
```

- [ ] **Step 10: Chạy test cho thấy pass**

Run: `go test ./core/ -run "TestNhanHang|TestLocChuaThuDu|TestKhachBaoVanDon|TestKhachKhongSuaVanDon" -v`
Expected: PASS

- [ ] **Step 11: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 12: Commit**

```bash
git add core/quantri.go core/cuahang.go core/donhang.go core/server.go core/qtdon_test.go core/ui/qt-don-ct.html core/ui/qt-don.html core/ui/tracuu.html
git commit -m "feat: van don hai chieu va loc don cho hang ve / con no"
```

---

## Task 14: Chạy thử đầu-cuối và deploy

**Files:** không sửa code

- [ ] **Step 1: Chạy toàn bộ test**

Run: `go test ./...`
Expected: PASS, không SKIP nào ngoài dự tính

- [ ] **Step 2: Đi hết một vòng bằng tay trên máy dev**

1. `/qt/lien-he` — điền địa chỉ, giờ làm, mã BIN + số tài khoản + chủ tài khoản.
2. `/qt/giao-dien` — bật Online, xác nhận hết cảnh báo thiếu địa chỉ.
3. Xem trang chủ, quy trình, chân trang: chữ đã đổi giọng chưa.
4. `/qt/noi-dung` — sửa một khoá có `MacOn`, xác nhận dòng "Đang sửa bản Online" và `data/noi-dung.yaml` mọc đúng một dòng khoá hậu tố `@online`. Tắt Online, kiểm chữ cũ còn nguyên.
5. `/cua-hang` — thử đặt một đơn giày kiểu `court`: phải bị chặn ngay, không tạo đơn.
6. Đặt một đơn vợt thật, kèm ảnh. Chép mã đơn và đường dẫn tra cứu.
7. Mở đường dẫn tra cứu, dán một mã vận đơn giả vào ô "Đã gửi hàng rồi?". Kiểm `ma_van_don_den` trong tệp đơn.
8. `/qt/don` — thấy đơn ở tab "Chờ hàng về", có nhãn `web`, có ảnh khách gửi, có mã vận đơn khách vừa báo.
9. Bấm "Đã nhận kiện hàng" → đơn sang "Mới nhận". Quay lại trang tra cứu: ô báo vận đơn đã đóng.
10. Báo giá → duyệt → sửa → `xong`. Ghi mã vận đơn gửi về ở trang chi tiết đơn.
11. Mở đường dẫn tra cứu: thấy số tiền, nội dung chuyển khoản có mã đơn, mã vận đơn về, hai lựa chọn trả tiền. Bấm "Tôi sẽ chuyển khoản", kiểm `hinh_thuc_tra: qr` trong tệp đơn.
12. `/qt/tien` — dán một dòng sao kê giả mang mã đơn đó, xác nhận `saoke.go` đề xuất đúng đơn. Duyệt khoản thu → khối thanh toán trên trang tra cứu biến mất.
13. `/qt/giao-dien` — tắt Online. `/cua-hang` trả trang "tạm chưa nhận đơn online".

- [ ] **Step 3: Chụp màn hình kiểm bố cục**

Chrome headless **qua CDP Emulation** — `--window-size=390` trên Windows nói dối (dựng ở 504). Bề ngang 390 và 1280, cho `/cua-hang`, màn kết, và trang tra cứu có khối thanh toán. Đọc ảnh, không đoán.

- [ ] **Step 4: Build cho VPS**

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ../tramvot-linux .
```

- [ ] **Step 5: Deploy**

Chỉ đẩy binary. **Không** rsync `data/` lên VPS — `noi-dung.yaml`, `lien-he.yaml`, `giao-dien.yaml` trên server là bản Kendy đã gõ; đè lên là mất hết. Restart service sau khi chép.

- [ ] **Step 6: Kiểm trên bản thật**

Địa chỉ và tài khoản ngân hàng trên VPS phải điền lại — chúng nằm trong `data/`, không đi theo binary. Đặt một đơn thật rồi xoá.

Icon hoặc logo không đổi như mong đợi thì đừng vội sửa code: Cloudflare cache 7 ngày, và tệp tải lên ở `data/logo/` đè bản trong binary. Purge cache trước khi kết luận.

- [ ] **Step 7: Commit cuối**

```bash
git add -A
git commit -m "chore: ban Online chay duoc dau-cuoi"
```
