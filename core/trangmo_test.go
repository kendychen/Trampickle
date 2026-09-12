package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Mọi trang có biểu mẫu đều dùng {{$.CSRF}} và {{$.Nonce}}. Trang nào dựng
// dữ liệu mà quên nhúng Chung sẽ hỏng lúc render — mà render hỏng thì trả
// 500, không phải lỗi biên dịch. Bài này quét hết trang tĩnh để chuyện đó
// lộ ra ở đây chứ không phải trên máy chủ.
func TestMoiTrangDeuDungDuoc(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)

	duong := []string{
		// Trang khách.
		"/", "/gioi-thieu", "/ve-chung-toi", "/dich-vu", "/quy-trinh",
		"/lien-he", "/chinh-sach", "/bai-viet", "/tra-cuu",
		"/app", "/app/kiem", "/app/quy-trinh", "/app/tra-cuu", "/app/gui-anh",
		"/dang-nhap",
		// Trang quản trị.
		"/qt", "/qt/don", "/qt/don-vot", "/qt/don-giay",
		"/qt/don-moi", "/qt/don-moi?loai=giay", "/qt/doi-tac",
		"/qt/khach", "/qt/khach?vang=6&con_no=1",
		"/qt/yeu-cau", "/qt/vot",
		"/qt/kho", "/qt/kho/phieu", "/qt/kho/phieu/moi", "/qt/kho/vat-tu",
		"/qt/tien", "/qt/tien/dinh-ky", "/qt/tien/sao-ke", "/qt/thong-ke",
		"/qt/bai-viet", "/qt/bai-viet/moi", "/qt/giao-trinh",
		"/qt/dich-vu", "/qt/noi-dung", "/qt/lien-he", "/qt/nguong",
		"/qt/giao-dien", "/qt/anh-trang-chu", "/qt/cai-dat",
		"/qt/nguoi-dung", "/qt/mat-khau",
		// Ba đường của app quản lý — mở được khi CHƯA đăng nhập (điện thoại đi
		// lấy manifest và icon lúc chưa có cookie phiên). Ở đây quét kèm cho
		// chắc chúng không hỏng lúc render.
		"/qt/manifest.webmanifest", "/qt/icon.svg", "/qt/icon.png", "/qt/mat-mang",
	}
	for _, d := range duong {
		r := httptest.NewRequest("GET", d, nil)
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		// /qt và /dang-nhap chuyển hướng khi đã đăng nhập — vẫn là bình thường.
		if w.Code != http.StatusOK && w.Code != http.StatusSeeOther {
			t.Errorf("%s trả %d: %s", d, w.Code, catBot(w.Body.String(), 200))
		}
	}

	// Trang đăng nhập lúc chưa đăng nhập: đây mới là lúc form hiện ra.
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/dang-nhap", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("/dang-nhap trả %d: %s", w.Code, catBot(w.Body.String(), 200))
	}
	if !strings.Contains(w.Body.String(), `name="_csrf"`) {
		t.Error("form đăng nhập không có token CSRF")
	}
}
