package core

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func postThu(t *testing.T, h http.Handler, ck *http.Cookie, duong, than string) *httptest.ResponseRecorder {
	t.Helper()
	r := httptest.NewRequest("POST", duong, strings.NewReader(than))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if ck != nil {
		r.AddCookie(ck)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestPOSTThieuTokenBiChan(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	for _, than := range []string{"", "ten=abc", "_csrf=&ten=abc", "_csrf=lung-tung&ten=abc"} {
		if w := postThu(t, h, ck, "/qt/kho/vat-tu", than); w.Code != http.StatusForbidden {
			t.Errorf("thân %q trả %d, đợi 403", than, w.Code)
		}
	}
}

func TestPOSTCoTokenDungThiQua(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	w := postThu(t, h, ck, "/qt/kho/vat-tu", "_csrf="+tokenTuPhien(ck.Value)+"&ten=Cán+vợt&dvt=cái")
	if w.Code == http.StatusForbidden {
		t.Fatalf("token đúng mà bị chặn: %s", catBot(w.Body.String(), 200))
	}
}

// Token của phiên khác không dùng được — nếu dùng được thì token vô nghĩa,
// vì kẻ tấn công chỉ cần tự đăng nhập một tài khoản để lấy một cái.
func TestTokenCuaPhienKhacKhongDung(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	la := tokenTuPhien("ma-phien-cua-nguoi-khac")
	if w := postThu(t, h, ck, "/qt/kho/vat-tu", "_csrf="+la+"&ten=abc"); w.Code != http.StatusForbidden {
		t.Errorf("token phiên khác lại qua được: %d", w.Code)
	}
}

// Khách chưa đăng nhập vẫn phải gửi được biểu mẫu tra cứu: cookie đặt ở lượt
// GET, token in vào form, POST lại thì khớp.
func TestKhachChuaDangNhapVanGuiDuocForm(t *testing.T) {
	h, _ := dungHandlerThu(t, VaiTroChu)

	r := httptest.NewRequest("GET", "/tra-cuu", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)

	var ckCSRF *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == tenCookieCSRF {
			ckCSRF = c
		}
	}
	if ckCSRF == nil {
		t.Fatal("lượt GET không đặt cookie CSRF cho khách")
	}
	if !strings.Contains(w.Body.String(), `name="_csrf" value="`+ckCSRF.Value+`"`) {
		t.Fatal("form không mang token khớp với cookie vừa đặt")
	}

	r2 := httptest.NewRequest("POST", "/tra-cuu", strings.NewReader("_csrf="+ckCSRF.Value+"&ma=X&sdt=1234"))
	r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r2.AddCookie(ckCSRF)
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, r2)
	if w2.Code == http.StatusForbidden {
		t.Errorf("khách gửi đúng token vẫn bị chặn: %s", catBot(w2.Body.String(), 200))
	}
}

func dungMultipart(t *testing.T, truong map[string]string) (string, *bytes.Buffer) {
	t.Helper()
	var b bytes.Buffer
	mw := multipart.NewWriter(&b)
	for k, v := range truong {
		mw.WriteField(k, v)
	}
	fw, err := mw.CreateFormFile("anh", "a.png")
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte("\x89PNG\r\n\x1a\n"))
	mw.Close()
	return mw.FormDataContentType(), &b
}

// Lớp bọc không đọc được thân multipart nên mặc định phải CHẶN. Đường nào
// thật sự nhận tệp mới được ghi vào danh sách, và tự kiểm lấy.
func TestMultipartNgoaiDanhSachBiChan(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	ct, b := dungMultipart(t, map[string]string{"_csrf": tokenTuPhien(ck.Value)})

	r := httptest.NewRequest("POST", "/qt/kho/vat-tu", b)
	r.Header.Set("Content-Type", ct)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("multipart tới đường không nhận tệp lại qua được: %d", w.Code)
	}
}

// Đường có nhận tệp: token sai thì chính handler phải chặn.
func TestMultipartTrongDanhSachVanKiemToken(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	ct, b := dungMultipart(t, map[string]string{"_csrf": "sai-be-bet", "loai": "logo"})

	r := httptest.NewRequest("POST", "/qt/giao-dien/logo", b)
	r.Header.Set("Content-Type", ct)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("tải logo với token sai trả %d, đợi 403", w.Code)
	}
}

// Danh sách cho phép multipart phải khớp với những handler thật sự gọi
// KiemCSRFMultipart. Bài này chốt danh sách để không ai thêm vào cho vui.
func TestDanhSachMultipartDungNhuMongDoi(t *testing.T) {
	cho := []string{
		"/gui-yeu-cau", "/app/gui-anh", "/qt/giao-dien/logo",
		"/qt/anh-trang-chu/them", "/qt/bai-viet/anh", "/qt/don/DH-1/anh",
	}
	for _, d := range cho {
		if !multipartChoPhep(d) {
			t.Errorf("%s phải được nhận tệp", d)
		}
	}
	for _, d := range []string{"/qt/kho/vat-tu", "/dang-nhap", "/qt/don/DH-1", "/qt/anh-trang-chu", "/anh"} {
		if multipartChoPhep(d) {
			t.Errorf("%s không nhận tệp mà lại được cho qua", d)
		}
	}
}

func TestGETKhongBiChan(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	for _, d := range []string{"/", "/qt", "/qt/kho", "/dang-nhap"} {
		r := httptest.NewRequest("GET", d, nil)
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code == http.StatusForbidden {
			t.Errorf("GET %s bị chặn CSRF", d)
		}
	}
}
