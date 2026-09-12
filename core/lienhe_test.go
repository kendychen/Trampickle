package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// Dựng một Root rỗng: chưa có data/lien-he.yaml, đúng trạng thái máy chủ hôm nay.
func dungLienHeTrong(t *testing.T) {
	t.Helper()
	cuRoot, cuCFG := Root, CFG
	cuMu := lienHeDaSua
	t.Cleanup(func() {
		Root, CFG = cuRoot, cuCFG
		lienHeMu.Lock()
		lienHeDaSua = cuMu
		lienHeMu.Unlock()
	})
	Root = t.TempDir()
	if err := os.MkdirAll(P("data"), 0o755); err != nil {
		t.Fatal(err)
	}
	lienHeMu.Lock()
	lienHeDaSua = nil
	lienHeMu.Unlock()
}

// Chưa có file thì lấy của config.yaml; lưu xong thì file đè lên.
func TestLienHeFileDeConfig(t *testing.T) {
	dungLienHeTrong(t)
	CFG.ThuongHieu.LienHe = LienHe{DienThoai: "0900000000", DiaChi: "Địa chỉ cũ trong config"}

	if got := LienHeHienTai().DiaChi; got != "Địa chỉ cũ trong config" {
		t.Fatalf("chưa có file mà không lấy config: %q", got)
	}

	if _, err := DatLienHe(LienHe{DiaChi: "12 Nguyễn Trãi, P.5, Gò Vấp, TP.HCM"}); err != nil {
		t.Fatal(err)
	}
	if got := LienHeHienTai().DiaChi; got != "12 Nguyễn Trãi, P.5, Gò Vấp, TP.HCM" {
		t.Fatalf("lưu xong vẫn ra %q", got)
	}
	// Ô bỏ trống phải thắng luôn: xoá số điện thoại trong admin mà config vẫn
	// còn số cũ thì web hiện lại số cũ — coi như xoá không được.
	if got := LienHeHienTai().DienThoai; got != "" {
		t.Fatalf("xoá điện thoại rồi mà vẫn ra %q", got)
	}

	// Nạp lại từ đĩa như lúc khởi động.
	lienHeMu.Lock()
	lienHeDaSua = nil
	lienHeMu.Unlock()
	if err := NapLienHe(); err != nil {
		t.Fatal(err)
	}
	if got := LienHeHienTai().DiaChi; got != "12 Nguyễn Trãi, P.5, Gò Vấp, TP.HCM" {
		t.Fatalf("nạp lại ra %q", got)
	}
}

func TestDatLienHeChanSaiSot(t *testing.T) {
	dungLienHeTrong(t)
	xau := []struct {
		ten string
		l   LienHe
	}{
		{"email không có @", LienHe{Email: "kendy.gmail.com"}},
		{"facebook thiếu https", LienHe{Facebook: "fb.com/trampickle"}},
		{"tiktok thiếu https", LienHe{TikTok: "tiktok.com/@trampickle"}},
		{"instagram thiếu https", LienHe{Instagram: "instagram.com/trampickle"}},
		{"youtube thiếu https", LienHe{YouTube: "youtube.com/@trampickle"}},
		{"địa chỉ dài quá", LienHe{DiaChi: strings.Repeat("x", 201)}},
	}
	for _, c := range xau {
		if _, err := DatLienHe(c.l); err == nil {
			t.Errorf("%s: lẽ ra phải báo lỗi", c.ten)
		}
	}
	if _, err := os.Stat(fileLienHe()); !os.IsNotExist(err) {
		t.Error("lưu hụt mà vẫn ghi ra file")
	}
}

func TestTrangLienHeDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, "chu")
	dungLienHeTrong(t)

	than := moTrang(t, mux, ck, "/qt/lien-he")
	if !strings.Contains(than, "dia_chi") || !strings.Contains(than, "Địa chỉ trạm") {
		t.Fatal("trang thiếu ô địa chỉ")
	}

	f := url.Values{
		"dien_thoai":   {"0901234567"},
		"dia_chi":      {"  12 Nguyễn Trãi,   P.5, Gò Vấp  "},
		"gio_lam_viec": {"8h – 20h"},
	}
	r := httptest.NewRequest("POST", "/qt/lien-he", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("POST trả %d", w.Code)
	}
	// Khoảng trắng thừa gom lại: dán từ Google Maps hay dính hai ba dấu cách.
	if got := LienHeHienTai().DiaChi; got != "12 Nguyễn Trãi, P.5, Gò Vấp" {
		t.Fatalf("địa chỉ lưu thành %q", got)
	}
}

// Địa chỉ phải ra tận trang chủ, và không có địa chỉ thì không để lại dòng rỗng.
func TestTrangChuHienDiaChi(t *testing.T) {
	mux, ck := dungTrangThu(t, "chu")
	dungLienHeTrong(t)

	than := moTrang(t, mux, ck, "/")
	if strings.Contains(than, `class="hero-lien-he"`) {
		t.Fatal("chưa điền địa chỉ mà trang chủ đã có dòng địa chỉ")
	}

	if _, err := DatLienHe(LienHe{DiaChi: "12 Nguyễn Trãi, Gò Vấp"}); err != nil {
		t.Fatal(err)
	}
	than = moTrang(t, mux, ck, "/")
	if !strings.Contains(than, "12 Nguyễn Trãi, Gò Vấp") {
		t.Fatal("trang chủ không hiện địa chỉ vừa lưu")
	}
}

// Trang Về chúng tôi phải trả lời được "trạm ở đâu" ngay cả khi chưa ai điền
// địa chỉ — chưa điền thì hiện câu thay thế, chứ không phải một khoảng trống
// và một nút chỉ đường dẫn tới bản đồ rỗng.
func TestTrangVeChungToi(t *testing.T) {
	mux, ck := dungTrangThu(t, "chu")
	dungLienHeTrong(t)

	than := moTrang(t, mux, ck, "/ve-chung-toi")
	if !strings.Contains(than, ND("vechungtoi.o.chua-co")) {
		t.Fatal("chưa có địa chỉ mà trang không hiện câu thay thế")
	}
	if strings.Contains(than, "google.com/maps") {
		t.Fatal("chưa có địa chỉ mà vẫn dựng nút chỉ đường")
	}

	if _, err := DatLienHe(LienHe{DiaChi: "69 Đại Linh, Đại Mỗ", GioLamVic: "8h–19h cả tuần"}); err != nil {
		t.Fatal(err)
	}
	than = moTrang(t, mux, ck, "/ve-chung-toi")
	for _, can := range []string{"69 Đại Linh, Đại Mỗ", "8h–19h cả tuần",
		"google.com/maps/search/?api=1&amp;query=69&#43;%C4%90%E1%BA%A1i&#43;Linh"} {
		if !strings.Contains(than, can) {
			t.Fatalf("trang Về chúng tôi thiếu %q", can)
		}
	}
}

// Ảnh chỉ hiện ở đúng nơi được treo. Một tấm ảnh chỗ làm việc lọt vào băng
// hero trang chủ giữa mấy tấm vá mặt vợt là hỏng cả hai trang.
func TestAnhTreoDungNoi(t *testing.T) {
	anhHeroMu.Lock()
	cu := anhHeroDS
	anhHeroDS = []AnhHero{{Ten: "a.webp"}, {Ten: "b.webp", Noi: NoiTram}}
	anhHeroMu.Unlock()
	t.Cleanup(func() {
		anhHeroMu.Lock()
		anhHeroDS = cu
		anhHeroMu.Unlock()
	})

	if ds := DsAnhNoi(""); len(ds) != 1 || ds[0].Ten != "a.webp" {
		t.Fatalf("băng ảnh trang chủ sai: %v", ds)
	}
	if ds := DsAnhNoi(NoiTram); len(ds) != 1 || ds[0].Ten != "b.webp" {
		t.Fatalf("dải ảnh trạm sai: %v", ds)
	}
}

// --- Mạng xã hội -----------------------------------------------------------

// Thứ tự hàng icon KHÔNG theo thứ tự điền: chân trang và bảng hiệu app phải
// giống nhau mọi lúc, người ta nhớ vị trí chứ không đọc lại từng cái.
func TestMangXaHoiChiTraCaiDaDien(t *testing.T) {
	l := LienHe{YouTube: "https://youtube.com/@t", Facebook: "https://fb.com/t"}
	got := l.MangXaHoi()
	if len(got) != 2 {
		t.Fatalf("điền 2 mà ra %d", len(got))
	}
	if got[0].Ma != "facebook" || got[1].Ma != "youtube" {
		t.Fatalf("sai thứ tự: %v", got)
	}
	if len((LienHe{}).MangXaHoi()) != 0 {
		t.Error("không điền gì mà vẫn có trang")
	}
}

// Chỉ có mạng xã hội, không điện thoại không địa chỉ, thì chân trang vẫn phải
// hiện chứ không rơi vào nhánh "Đang cập nhật".
func TestCoGiKhongTinhCaMangXaHoi(t *testing.T) {
	if !(LienHe{TikTok: "https://tiktok.com/@t"}).CoGiKhong() {
		t.Error("có TikTok mà bảo là chưa có gì")
	}
	if (LienHe{}).CoGiKhong() {
		t.Error("rỗng mà bảo là có")
	}
}

// Ba khoá mới phải đi trọn đường: form /qt/lien-he → đĩa → trang /lien-he.
func TestMangXaHoiTuFormRaTrang(t *testing.T) {
	mux, ck := dungTrangThu(t, "chu")
	dungLienHeTrong(t)

	f := url.Values{
		"facebook":  {"https://facebook.com/trampickle"},
		"tiktok":    {"https://www.tiktok.com/@trampickle"},
		"instagram": {"https://instagram.com/trampickle"},
		"youtube":   {"https://youtube.com/@trampickle"},
	}
	r := httptest.NewRequest("POST", "/qt/lien-he", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("POST trả %d", w.Code)
	}
	if n := len(LienHeHienTai().MangXaHoi()); n != 4 {
		t.Fatalf("lưu 4 trang mà còn %d", n)
	}

	than := moTrang(t, mux, ck, "/lien-he")
	for _, u := range []string{"https://www.tiktok.com/@trampickle", "https://instagram.com/trampickle", "https://youtube.com/@trampickle"} {
		if !strings.Contains(than, u) {
			t.Errorf("/lien-he thiếu %s", u)
		}
	}
}
