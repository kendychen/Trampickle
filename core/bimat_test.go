package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func dangKhoa(t *testing.T, mux *http.ServeMux, ck *http.Cookie, f url.Values) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("POST", "/qt/cai-dat", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

// Khóa dán trong admin phải có hiệu lực NGAY, không đợi khởi động lại — đó là
// toàn bộ lý do làm trang này thay vì tiếp tục ssh sửa .env.
func TestKhoaDanTrongAdminCoHieuLucNgay(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	t.Cleanup(func() {
		biMatMu.Lock()
		biMat = BiMat{}
		biMatMu.Unlock()
	})

	const khoa = "AIzaSyTEST-khoa-gia-dinh-1234"
	if w := dangKhoa(t, mux, ck, url.Values{"viec": {"luu"}, "gemini": {khoa}}); w.Code != http.StatusOK {
		t.Fatalf("lưu khóa trả %d", w.Code)
	}
	if GeminiKey() != khoa {
		t.Fatalf("GeminiKey() = %q, muốn khóa vừa lưu", GeminiKey())
	}

	// Ô để trống là GIỮ NGUYÊN. Trang không hiện khóa đầy đủ nên nếu trống
	// mà bị hiểu thành xóa thì mỗi lần đổi model là mất khóa.
	dangKhoa(t, mux, ck, url.Values{"viec": {"luu"}, "gemini": {""}, "nha_cung_cap": {"ollama"}})
	if GeminiKey() != khoa {
		t.Error("lưu form với ô khóa trống đã xóa mất khóa cũ")
	}
	if Provider() != "ollama" {
		t.Errorf("Provider() = %q, muốn ollama theo lựa chọn trong admin", Provider())
	}

	// Trang không được in khóa đầy đủ ra HTML: người ngồi cạnh, ảnh chụp
	// màn hình gửi cho tôi, đều đọc được.
	r := httptest.NewRequest("GET", "/qt/cai-dat", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if strings.Contains(w.Body.String(), khoa) {
		t.Error("trang cài đặt in nguyên khóa ra HTML")
	}

	// Biến môi trường thắng file: ai đặt env là cố ý.
	t.Setenv("GEMINI_API_KEY", "khoa-tu-moi-truong")
	if GeminiKey() != "khoa-tu-moi-truong" {
		t.Errorf("GeminiKey() = %q, muốn giá trị từ biến môi trường", GeminiKey())
	}
}

func TestXoaKhoaVaTatDuPhong(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	t.Cleanup(func() {
		biMatMu.Lock()
		biMat = BiMat{}
		biMatMu.Unlock()
	})

	dangKhoa(t, mux, ck, url.Values{"viec": {"luu"}, "gemini": {"khoa-cu-can-bo"}, "tat_du_phong": {"1"}})
	if DuPhongLLM() != "" {
		t.Errorf("DuPhongLLM() = %q, muốn rỗng khi đã tắt dự phòng", DuPhongLLM())
	}
	dangKhoa(t, mux, ck, url.Values{"viec": {"xoa-gemini"}})
	if GeminiKey() != "" {
		t.Error("bấm Xóa mà khóa vẫn còn")
	}
}

// Khóa API là thứ đắt nhất trong trạm: ai cầm được là tiêu tiền của Kendy.
func TestThoKhongVaoDuocTrangKhoa(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	r := httptest.NewRequest("GET", "/qt/cai-dat", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("thợ mở /qt/cai-dat trả %d, muốn 403", w.Code)
	}
	if w := dangKhoa(t, mux, ck, url.Values{"viec": {"luu"}, "gemini": {"x"}}); w.Code != http.StatusForbidden {
		t.Errorf("thợ POST /qt/cai-dat trả %d, muốn 403", w.Code)
	}
}
