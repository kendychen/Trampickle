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

// dungGiaoTrinhThu dựng hai bài giáo trình giả trong trạm thử và khai chúng
// vào tai_lieu_nap, đúng cách thật: màn giáo trình chỉ nhìn vào config.
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
	CFG.DuongDan.TaiLieuNap = []string{
		"vanhanh/checklist-tu-choi.md",
		"vanhanh/giao-trinh/03-ca-sua/3-2-tach-lop.md",
		// Khai một bài không có tệp: màn hình phải nói thiếu chứ không sập.
		"vanhanh/giao-trinh/03-ca-sua/3-9-chua-co.md",
	}
}

// Danh sách chỉ hiện bài giáo trình, không hiện tài liệu vận hành khác cũng
// nằm trong tai_lieu_nap.
func TestGiaoTrinhDanhSachChiHienBaiGiaoTrinh(t *testing.T) {
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
	if strings.Contains(s, "checklist-tu-choi") {
		t.Error("tài liệu vận hành lọt vào danh sách giáo trình")
	}
}

// Mở bài, sửa, lưu, mở lại thấy nội dung mới. Đây là cả vòng đời của màn này.
func TestGiaoTrinhSuaRoiLuu(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	dungGiaoTrinhThu(t)

	duong := "/qt/giao-trinh/bai/03-ca-sua/3-2-tach-lop.md"
	if s := moTrang(t, mux, ck, duong); !strings.Contains(s, "Gõ nghe bộp") {
		t.Fatal("màn sửa không hiện nội dung tệp")
	}

	f := url.Values{}
	f.Set("ma", "03-ca-sua/3-2-tach-lop.md")
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

// Mã bài không có trong config thì không mở được. Đây là hàng rào chặn
// ../../ đi ra ngoài thư mục giáo trình, nên phải có test riêng.
func TestGiaoTrinhKhongMoDuocTepNgoaiDanhSach(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	dungGiaoTrinhThu(t)

	for _, duong := range []string{
		"/qt/giao-trinh/bai/../../../config.yaml",
		"/qt/giao-trinh/bai/03-ca-sua/khong-co-bai-nay.md",
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
	f.Set("ma", "03-ca-sua/3-2-tach-lop.md")
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

// Thợ không phải chủ thì không vào được giáo trình.
func TestGiaoTrinhThoKhongVaoDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	dungGiaoTrinhThu(t)

	r := httptest.NewRequest("GET", "/qt/giao-trinh", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == http.StatusOK {
		t.Error("thợ mở được giáo trình")
	}
}
