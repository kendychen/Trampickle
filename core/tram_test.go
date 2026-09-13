package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// Dựng một Root rỗng: chưa có data/tram.yaml, đúng trạng thái máy chủ hôm nay.
func dungTramTrong(t *testing.T) {
	t.Helper()
	cuRoot, cuCFG, cuSua := Root, CFG, tramDaSua
	t.Cleanup(func() {
		Root, CFG = cuRoot, cuCFG
		tramMu.Lock()
		tramDaSua = cuSua
		tramMu.Unlock()
	})
	Root = t.TempDir()
	if err := os.MkdirAll(P("data"), 0o755); err != nil {
		t.Fatal(err)
	}
	tramMu.Lock()
	tramDaSua = nil
	tramMu.Unlock()
}

// Chưa có file thì lấy của config.yaml; lưu xong thì file đè lên, và nạp lại
// từ đĩa như lúc khởi động vẫn ra bản đã sửa.
func TestTramFileDeConfig(t *testing.T) {
	dungTramTrong(t)
	CFG.ThuongHieu.Ten = "Tên cũ trong config"
	CFG.Email.TraLoi = "cu@example.com"

	if got := TenTram(); got != "Tên cũ trong config" {
		t.Fatalf("chưa có file mà không lấy config: %q", got)
	}
	if got := TienToDon(LoaiVot); got != "TV-" {
		t.Fatalf("tiền tố mặc định ra %q", got)
	}

	if _, err := DatTram(ThongTinTram{
		Ten: "Trạm Pickle", TienToVot: "PK", TienToGiay: "PG",
		MailTraLoi: "moi@example.com",
	}); err != nil {
		t.Fatal(err)
	}
	if got := TenTram(); got != "Trạm Pickle" {
		t.Fatalf("lưu xong vẫn ra %q", got)
	}
	if got := MailTraLoi(); got != "moi@example.com" {
		t.Fatalf("hòm thư trả lời ra %q", got)
	}

	tramMu.Lock()
	tramDaSua = nil
	tramMu.Unlock()
	if err := NapTram(); err != nil {
		t.Fatal(err)
	}
	if got := TenTram(); got != "Trạm Pickle" {
		t.Fatalf("nạp lại ra %q", got)
	}
}

// Đổi tiền tố thì mã MỚI đi theo tiền tố mới, còn mã CŨ vẫn phải nhận ra khi
// đọc sao kê — không thì mọi đơn trước ngày đổi biến mất khỏi màn đối chiếu.
func TestDoiTienToVanNhanMaCu(t *testing.T) {
	dungTramTrong(t)

	if _, err := DatTram(ThongTinTram{Ten: "Trạm", TienToVot: "PK", TienToGiay: "PG"}); err != nil {
		t.Fatal(err)
	}
	if got := TienToDon(LoaiVot); got != "PK-" {
		t.Fatalf("đơn vợt vẫn ra %q", got)
	}
	if got := TienToDon(LoaiGiay); got != "PG-" {
		t.Fatalf("đơn giày vẫn ra %q", got)
	}
	if got := MaDonMoi(LoaiVot); !strings.HasPrefix(got, "PK-") {
		t.Fatalf("mã đơn mới ra %q", got)
	}

	cu := TramHienTai().TienToCu
	if len(cu) != 2 || cu[0] != "TV" || cu[1] != "TG" {
		t.Fatalf("lịch sử tiền tố ra %v, phải giữ TV và TG", cu)
	}

	// Cả mã mới lẫn mã cũ đều phải qua được biểu thức quét sao kê.
	for _, c := range []struct{ dong, ma string }{
		{"CK PK2609001 chuyen tien 500000", "PK-2609-001"},
		{"CK TV 2609 001 chuyen tien 500000", "TV-2609-001"},
		{"CK TG-2609-007 tien giay", "TG-2609-007"},
	} {
		m := reMaDon().FindStringSubmatch(c.dong)
		if m == nil {
			t.Errorf("%q không khớp biểu thức mã đơn", c.dong)
			continue
		}
		if got := strings.ToUpper(m[1] + "-" + m[2] + "-" + m[3]); got != c.ma {
			t.Errorf("%q đọc ra %q, mong %q", c.dong, got, c.ma)
		}
	}

	// Bấm Lưu lần nữa với đúng thứ đang có: lịch sử không được sinh sôi.
	if _, err := DatTram(TramHienTai()); err != nil {
		t.Fatal(err)
	}
	if cu := TramHienTai().TienToCu; len(cu) != 2 {
		t.Fatalf("lưu lại lần hai làm lịch sử thành %v", cu)
	}
}

func TestDatTramChanSaiSot(t *testing.T) {
	dungTramTrong(t)
	ok := ThongTinTram{Ten: "Trạm", TienToVot: "TV", TienToGiay: "TG"}
	xau := []struct {
		ten string
		sua func(*ThongTinTram)
	}{
		{"thiếu tên", func(t *ThongTinTram) { t.Ten = "" }},
		{"tiền tố một chữ", func(t *ThongTinTram) { t.TienToVot = "T" }},
		{"tiền tố bắt đầu bằng số", func(t *ThongTinTram) { t.TienToVot = "1V" }},
		{"hai tiền tố trùng nhau", func(t *ThongTinTram) { t.TienToGiay = "TV" }},
		{"mail gửi đi không có @", func(t *ThongTinTram) { t.MailTu = "admin.trampickle.vn" }},
		{"gốc web thiếu https", func(t *ThongTinTram) { t.GocWeb = "trampickle.vn" }},
		{"tên dài quá", func(t *ThongTinTram) { t.Ten = strings.Repeat("x", 61) }},
	}
	for _, c := range xau {
		v := ok
		c.sua(&v)
		if _, err := DatTram(v); err == nil {
			t.Errorf("%s: lẽ ra phải báo lỗi", c.ten)
		}
	}
	if _, err := os.Stat(fileTram()); !os.IsNotExist(err) {
		t.Error("lưu hụt mà vẫn ghi ra file")
	}
}

func TestTrangTramDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, "chu")
	dungTramTrong(t)

	than := moTrang(t, mux, ck, "/qt/tram")
	for _, s := range []string{"tien_to_vot", "mail_bao_cao", "Thông tin trạm"} {
		if !strings.Contains(than, s) {
			t.Fatalf("trang thiếu %q", s)
		}
	}

	f := url.Values{
		"ten":          {"  Trạm   Pickle  "},
		"tien_to_vot":  {"pk-"}, // gõ thường, gõ kèm gạch: vẫn phải nhận
		"tien_to_giay": {"PG"},
		"goc_web":      {"https://trampickle.vn/"},
		"mail_bat":     {"1"},
	}
	r := httptest.NewRequest("POST", "/qt/tram", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("POST trả %d", w.Code)
	}
	if got := TenTram(); got != "Trạm Pickle" {
		t.Fatalf("tên lưu thành %q", got)
	}
	if got := TienToDon(LoaiVot); got != "PK-" {
		t.Fatalf("tiền tố lưu thành %q", got)
	}
	// Gạch cuối phải bị cắt, không thì mọi link trong mail thành hai gạch.
	if got := GocWeb(); got != "https://trampickle.vn" {
		t.Fatalf("gốc web lưu thành %q", got)
	}
}

// Tên trạm phải đi ra tận trang khách: sửa trong admin mà chân trang vẫn tên
// cũ thì coi như chưa sửa được.
func TestTenTramRaTrangKhach(t *testing.T) {
	mux, ck := dungTrangThu(t, "chu")
	dungTramTrong(t)

	if _, err := DatTram(ThongTinTram{
		Ten: "Trạm Kiểm Thử", DongMoTa: "sửa vợt pickleball",
		TienToVot: "TV", TienToGiay: "TG",
	}); err != nil {
		t.Fatal(err)
	}
	than := moTrang(t, mux, ck, "/")
	if !strings.Contains(than, "Trạm Kiểm Thử") {
		t.Fatal("trang chủ không hiện tên trạm vừa lưu")
	}
	if !strings.Contains(than, "sửa vợt pickleball") {
		t.Fatal("trang chủ không hiện dòng mô tả vừa lưu")
	}
}
