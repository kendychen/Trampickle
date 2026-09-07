package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// dungHandlerThu dựng cả chồng lớp bọc, không phải mux trần — đúng thứ chạy
// trên máy chủ thật.
func dungHandlerThu(t *testing.T, vaiTro string) (http.Handler, *http.Cookie) {
	t.Helper()
	dungKhoTienThu(t)
	if err := InitTemplates(); err != nil {
		t.Fatal(err)
	}
	cuCongKhai := CongKhai
	t.Cleanup(func() { CongKhai = cuCongKhai })
	return NewHandler(true), phienThu(t, vaiTro)
}

func TestHeaderAnToanCoTrenMoiTrang(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)

	for _, duong := range []string{"/", "/qt", "/dang-nhap"} {
		r := httptest.NewRequest("GET", duong, nil)
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)

		hd := w.Header()
		for k, v := range map[string]string{
			"X-Content-Type-Options": "nosniff",
			"X-Frame-Options":        "DENY",
			"Referrer-Policy":        "strict-origin-when-cross-origin",
		} {
			if hd.Get(k) != v {
				t.Errorf("%s: %s = %q, đợi %q", duong, k, hd.Get(k), v)
			}
		}
		if !strings.Contains(hd.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
			t.Errorf("%s: CSP thiếu frame-ancestors: %q", duong, hd.Get("Content-Security-Policy"))
		}
	}
}

// Bật HSTS lúc còn chạy HTTP ở máy nhà là tự khoá mình khỏi localhost.
func TestHSTSChiKhiCoHTTPS(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	goi := func() string {
		r := httptest.NewRequest("GET", "/qt", nil)
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Header().Get("Strict-Transport-Security")
	}
	cu := HTTPSBat
	t.Cleanup(func() { HTTPSBat = cu })

	HTTPSBat = false
	if goi() != "" {
		t.Error("chưa có HTTPS mà đã gửi HSTS")
	}
	HTTPSBat = true
	if !strings.Contains(goi(), "max-age=") {
		t.Error("có HTTPS mà thiếu HSTS")
	}
}

// Nonce trong header và nonce trên thẻ <script> phải là một, không thì CSP
// chặn luôn script của chính mình.
func TestNonceTrongCSPKhopVoiTheScript(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	r := httptest.NewRequest("GET", "/", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	csp := w.Header().Get("Content-Security-Policy")
	i := strings.Index(csp, "'nonce-")
	if i < 0 {
		t.Fatalf("CSP không có nonce: %s", csp)
	}
	nonce := csp[i+len("'nonce-"):]
	nonce = nonce[:strings.Index(nonce, "'")]
	if len(nonce) != 32 {
		t.Fatalf("nonce dài %d, đợi 32 ký tự hex", len(nonce))
	}
	if !strings.Contains(w.Body.String(), `<script nonce="`+nonce+`">`) {
		t.Error("thẻ script không mang đúng nonce của lượt tải này")
	}
}

// Hai lượt tải phải khác nonce. Nonce cố định thì bằng không có nonce.
func TestNonceDoiMoiLuotTai(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	lay := func() string {
		r := httptest.NewRequest("GET", "/", nil)
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w.Header().Get("Content-Security-Policy")
	}
	if lay() == lay() {
		t.Error("hai lượt tải cùng một nonce")
	}
}

// --- Nhật ký -----------------------------------------------------------

func docNhatKyThu(t *testing.T) []MucNhatKy {
	t.Helper()
	ten := filepath.Join(thuMucNhatKy(), time.Now().Format("2006-01")+".jsonl")
	if _, err := os.Stat(ten); err != nil {
		return nil
	}
	return DocNhatKy("", 0)
}

func TestNhatKyGhiThaoTacGhi(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)

	// GET không vào nhật ký: xem thì không đổi gì, ghi vào chỉ chôn mất
	// những dòng đáng đọc.
	r := httptest.NewRequest("GET", "/qt", nil)
	r.AddCookie(ck)
	h.ServeHTTP(httptest.NewRecorder(), r)
	if n := len(docNhatKyThu(t)); n != 0 {
		t.Fatalf("GET mà ghi %d dòng nhật ký", n)
	}

	// POST bị chặn CSRF vẫn phải để lại dấu vết.
	r = httptest.NewRequest("POST", "/qt/kho/vat-tu", strings.NewReader("ten=abc"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Fatalf("POST thiếu token trả %d, đợi 403", w.Code)
	}
	ds := docNhatKyThu(t)
	if len(ds) == 0 || ds[0].KetQua != "csrf-hong" {
		t.Fatalf("không ghi lại lần bị chặn CSRF: %+v", ds)
	}
	if ds[0].Ai != "kendy" || ds[0].Duong != "/qt/kho/vat-tu" {
		t.Errorf("dòng nhật ký thiếu thông tin: %+v", ds[0])
	}
}

// Mật khẩu không được lọt vào nhật ký.
func TestNhatKyKhongGhiMatKhau(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)

	than := "_csrf=" + tokenTuPhien(ck.Value) + "&mat_khau_moi=sieu-bi-mat&mat_khau_cu=cu-bi-mat"
	r := httptest.NewRequest("POST", "/qt/mat-khau", strings.NewReader(than))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	h.ServeHTTP(httptest.NewRecorder(), r)

	b, err := os.ReadFile(filepath.Join(thuMucNhatKy(), time.Now().Format("2006-01")+".jsonl"))
	if err != nil {
		t.Fatalf("không có file nhật ký: %v", err)
	}
	for _, cam := range []string{"sieu-bi-mat", "cu-bi-mat", tokenTuPhien(ck.Value)} {
		if strings.Contains(string(b), cam) {
			t.Errorf("lọt bí mật %q vào nhật ký", cam)
		}
	}
}
