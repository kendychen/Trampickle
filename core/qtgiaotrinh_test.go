package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dungGiaoTrinhThu dựng vài tệp giả trong trạm thử và khai chúng vào
// tai_lieu_nap, đúng cách thật: màn tài liệu nội bộ chỉ nhìn vào config.
func dungGiaoTrinhThu(t *testing.T) {
	t.Helper()
	cu := CFG.DuongDan.TaiLieuNap
	t.Cleanup(func() { CFG.DuongDan.TaiLieuNap = cu })

	thu := filepath.Join(Root, "vanhanh", "giao-trinh", "03-ca-sua")
	if err := os.MkdirAll(thu, 0o755); err != nil {
		t.Fatal(err)
	}
	bai := "# 3.2 Tách lớp\n\n> 🟨 NHÁP — chờ duyệt.\n\nGõ nghe bộp là tách lớp.\n"
	if err := os.WriteFile(filepath.Join(thu, "3-2-tach-lop.md"), []byte(bai), 0o644); err != nil {
		t.Fatal(err)
	}
	// Tệp phẳng ngay trong vanhanh/ — trước đây không hiện ở màn này.
	ck := "# Checklist từ chối\n\nGãy cán là trả về.\n"
	if err := os.WriteFile(filepath.Join(Root, "vanhanh", "checklist-tu-choi.md"), []byte(ck), 0o644); err != nil {
		t.Fatal(err)
	}
	// Tệp sinh từ YAML: hiện được, sửa không được.
	dm := "# Danh mục vợt\n\nSinh từ data/vot.yaml.\n"
	if err := os.WriteFile(filepath.Join(Root, "vanhanh", "danh-muc-vot.md"), []byte(dm), 0o644); err != nil {
		t.Fatal(err)
	}
	CFG.DuongDan.TaiLieuNap = []string{
		"vanhanh/checklist-tu-choi.md",
		"vanhanh/danh-muc-vot.md",
		"vanhanh/giao-trinh/03-ca-sua/3-2-tach-lop.md",
		// Khai một bài không có tệp: màn hình phải nói thiếu chứ không sập.
		"vanhanh/giao-trinh/03-ca-sua/3-9-chua-co.md",
		// Ngoài vanhanh/ thì không phải tài liệu nội bộ, không được lọt vào.
		"kho-tri-thuc/mot-tep-nao-do.md",
	}
}

// Danh sách hiện cả tệp phẳng trong vanhanh/ lẫn bài giáo trình, nhưng không
// hiện thứ nằm ngoài vanhanh/.
func TestGiaoTrinhDanhSachHienCaTaiLieuVanHanh(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	dungGiaoTrinhThu(t)

	s := moTrang(t, mux, ck, "/qt/giao-trinh")
	if !strings.Contains(s, "3.2 Tách lớp") {
		t.Error("không lấy tiêu đề từ dòng # đầu bài")
	}
	if !strings.Contains(s, "🟨 Nháp") {
		t.Error("không đọc được trạng thái nháp")
	}
	if !strings.Contains(s, "Thiếu tệp") {
		t.Error("bài khai trong config mà không có tệp thì phải báo thiếu")
	}
	if !strings.Contains(s, "checklist-tu-choi.md") {
		t.Error("tài liệu vận hành phẳng không hiện — đây là cả mục đích của màn này")
	}
	if !strings.Contains(s, "Chỉ đọc") {
		t.Error("danh-muc-vot.md phải mang nhãn chỉ đọc")
	}
	if strings.Contains(s, "kho-tri-thuc") {
		t.Error("tệp ngoài vanhanh/ lọt vào danh sách")
	}
}

// Mở bài, sửa, lưu, mở lại thấy nội dung mới. Đây là cả vòng đời của màn này.
func TestGiaoTrinhSuaRoiLuu(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	dungGiaoTrinhThu(t)

	duong := "/qt/giao-trinh/bai/giao-trinh/03-ca-sua/3-2-tach-lop.md"
	if s := moTrang(t, mux, ck, duong); !strings.Contains(s, "Gõ nghe bộp") {
		t.Fatal("màn sửa không hiện nội dung tệp")
	}

	f := url.Values{}
	f.Set("ma", "giao-trinh/03-ca-sua/3-2-tach-lop.md")
	f.Set("than", "# 3.2 Tách lớp\n\nĐã sửa lúc học về.\n")
	r := httptest.NewRequest("POST", "/qt/giao-trinh/luu", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("lưu trả %d, mong 303", w.Code)
	}

	if s := moTrang(t, mux, ck, duong); !strings.Contains(s, "Đã sửa lúc học về") {
		t.Error("lưu xong mở lại vẫn ra nội dung cũ")
	}
}

// Tệp phẳng trong vanhanh/ cũng sửa được, không chỉ bài giáo trình.
func TestGiaoTrinhSuaDuocTaiLieuPhang(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	dungGiaoTrinhThu(t)

	f := url.Values{}
	f.Set("ma", "checklist-tu-choi.md")
	f.Set("than", "# Checklist từ chối\n\nThêm luật mới.\n")
	r := httptest.NewRequest("POST", "/qt/giao-trinh/luu", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("lưu trả %d, mong 303", w.Code)
	}

	raw, err := os.ReadFile(filepath.Join(Root, "vanhanh", "checklist-tu-choi.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Thêm luật mới") {
		t.Error("không ghi được tài liệu vận hành phẳng")
	}
}

// danh-muc-vot.md sinh từ data/vot.yaml. Giấu nút Lưu chưa đủ — POST gõ tay
// vẫn tới được handler, nên handler phải tự chặn.
func TestGiaoTrinhKhongLuuDuocTepChiDoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	dungGiaoTrinhThu(t)

	if s := moTrang(t, mux, ck, "/qt/giao-trinh/bai/danh-muc-vot.md"); strings.Contains(s, "Lưu bài") {
		t.Error("màn sửa tệp chỉ đọc vẫn hiện nút Lưu bài")
	}

	f := url.Values{}
	f.Set("ma", "danh-muc-vot.md")
	f.Set("than", "# Danh mục vợt\n\nGõ tay đè lên.\n")
	r := httptest.NewRequest("POST", "/qt/giao-trinh/luu", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	raw, err := os.ReadFile(filepath.Join(Root, "vanhanh", "danh-muc-vot.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(raw), "Gõ tay đè lên") {
		t.Error("tệp chỉ đọc đã bị ghi đè")
	}
}

// Mã bài không có trong config thì không mở được. Đây là hàng rào chặn
// ../../ đi ra ngoài thư mục vanhanh/, nên phải có test riêng.
func TestGiaoTrinhKhongMoDuocTepNgoaiDanhSach(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	dungGiaoTrinhThu(t)

	for _, duong := range []string{
		"/qt/giao-trinh/bai/../../../config.yaml",
		"/qt/giao-trinh/bai/giao-trinh/03-ca-sua/khong-co-bai-nay.md",
		// Có trong tai_lieu_nap nhưng ngoài vanhanh/ — vẫn phải chặn.
		"/qt/giao-trinh/bai/kho-tri-thuc/mot-tep-nao-do.md",
	} {
		r := httptest.NewRequest("GET", duong, nil)
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			t.Errorf("%s mở được, lẽ ra phải chặn", duong)
		}
	}
}

// Lưu bài trống là xoá trắng một bài bằng một cú bấm nhầm. Phải chặn.
func TestGiaoTrinhKhongLuuBaiTrong(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	dungGiaoTrinhThu(t)

	f := url.Values{}
	f.Set("ma", "giao-trinh/03-ca-sua/3-2-tach-lop.md")
	f.Set("than", "   \n\n")
	r := httptest.NewRequest("POST", "/qt/giao-trinh/luu", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)

	raw, err := os.ReadFile(filepath.Join(Root, "vanhanh", "giao-trinh", "03-ca-sua", "3-2-tach-lop.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "Gõ nghe bộp") {
		t.Error("bài trống đã ghi đè lên nội dung cũ")
	}
}

// Thợ không phải chủ thì không vào được tài liệu nội bộ.
func TestGiaoTrinhThoKhongVaoDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	dungGiaoTrinhThu(t)

	r := httptest.NewRequest("GET", "/qt/giao-trinh", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == http.StatusOK {
		t.Error("thợ mở được tài liệu nội bộ")
	}
}
