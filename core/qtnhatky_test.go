package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func moNhatKy(t *testing.T, h http.Handler, ck *http.Cookie, duong string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("GET", duong, nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestTrangNhatKyHienMucVuaGhi(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)

	GhiNhatKy(MucNhatKy{Ai: "kendy", IP: "203.0.113.9", Viec: "POST",
		Duong: "/qt/don/DH-001/sua", DoiTuong: "trang_thai=xong", KetQua: "ok"})

	w := moNhatKy(t, h, ck, "/qt/nhat-ky")
	if w.Code != http.StatusOK {
		t.Fatalf("mã %d", w.Code)
	}
	than := w.Body.String()
	for _, can := range []string{"203.0.113.9", "/qt/don/DH-001/sua", "trang_thai=xong"} {
		if !strings.Contains(than, can) {
			t.Errorf("trang thiếu %q", can)
		}
	}
}

// Thợ không được xem nhật ký: trong đó có đường đi của mọi người khác.
func TestTrangNhatKyChanTho(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroTho)
	if w := moNhatKy(t, h, ck, "/qt/nhat-ky"); w.Code != http.StatusForbidden {
		t.Fatalf("thợ mở được nhật ký: mã %d", w.Code)
	}
}

// ?thang= đi thẳng vào tên file. Chuỗi lạ phải rơi về tháng này chứ không
// được thành một đường dẫn.
func TestTrangNhatKyChanThangBay(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	nay := time.Now().Format("2006-01")
	for _, xau := range []string{"../../data/bi-mat", "2026-13", "abc", ""} {
		w := moNhatKy(t, h, ck, "/qt/nhat-ky?thang="+xau)
		if w.Code != http.StatusOK {
			t.Fatalf("thang=%q: mã %d", xau, w.Code)
		}
		if !strings.Contains(w.Body.String(), nay) {
			t.Errorf("thang=%q: không rơi về tháng này (%s)", xau, nay)
		}
	}
}
