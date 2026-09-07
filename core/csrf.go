package core

// Chống CSRF bằng token, không chỉ dựa vào SameSite.
//
// Trước đây cả app chỉ dựa vào cookie SameSite=Lax. Nó chặn được form POST
// từ tên miền khác, nhưng hở hai chỗ:
//   - Lax không phải Strict: một số luồng điều hướng cấp cao nhất vẫn gửi
//     cookie đi, và trình duyệt cũ thì coi như không có SameSite.
//   - Bất kỳ trang con nào của trampickle.vn cũng là "cùng site". Chỉ cần
//     một chỗ cho phép chèn nội dung là SameSite không còn nghĩa gì.
//
// Cách làm: token = HMAC-SHA256(khoá máy chủ, mã phiên). Không phải lưu thêm
// gì trong RAM, không hết hạn lệch với phiên, và đăng xuất là token cũ chết
// theo. Khách chưa đăng nhập dùng double-submit cookie — không có phiên để
// mà gắn HMAC vào.

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

const tenCookieCSRF = "tv_csrf"

// khoaCSRF sinh lại mỗi lần khởi động. Hệ quả: restart máy chủ thì form đang
// mở dở bị từ chối một lần, bấm lại là xong. Đổi lại không phải quản lý một
// bí mật lâu dài nữa — không có file để rò, không có khoá để xoay vòng.
var khoaCSRF = byteNgauNhien(32)

func tokenTuPhien(maPhien string) string {
	m := hmac.New(sha256.New, khoaCSRF)
	m.Write([]byte(maPhien))
	return hex.EncodeToString(m.Sum(nil))
}

// tokenCSRF trả token để nhét vào form. Đã đăng nhập thì gắn với phiên; chưa
// thì đọc cookie tv_csrf (boCSRF đã đặt sẵn từ đầu lượt).
func tokenCSRF(r *http.Request) string {
	if c, err := r.Cookie(tenCookie); err == nil && c.Value != "" {
		return tokenTuPhien(c.Value)
	}
	if s, _ := r.Context().Value(ctxCSRF).(string); s != "" {
		return s
	}
	if c, err := r.Cookie(tenCookieCSRF); err == nil {
		return c.Value
	}
	return ""
}

func khopHang(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// multipartChoPhep — những đường được nhận biểu mẫu có tệp đính kèm.
//
// Vì sao phải liệt kê tay: token của form multipart nằm trong THÂN yêu cầu,
// mà lớp bọc này không được đọc thân. Đọc ở đây là vô hiệu hóa giới hạn dung
// lượng riêng của từng handler — /gui-yeu-cau cho 100 MB, nếu lớp bọc đọc
// trước thì con số đó áp cho cả /dang-nhap.
//
// Nên: mặc định CHẶN mọi multipart. Đường nào thật sự nhận tệp thì ghi vào
// đây, và handler của nó phải tự gọi KiemCSRFMultipart ngay sau khi
// ParseMultipartForm. Quên ghi vào danh sách thì route mới bị chặn ngay lần
// thử đầu — hỏng lộ ra chứ không âm thầm mất bảo vệ.
func multipartChoPhep(duong string) bool {
	switch duong {
	case "/gui-yeu-cau", "/app/gui-anh",
		"/qt/giao-dien/logo", "/qt/anh-trang-chu/them", "/qt/bai-viet/anh":
		return true
	}
	// /qt/don/{ma}/anh — mã đơn nằm giữa nên không so bằng được.
	return strings.HasPrefix(duong, "/qt/don/") && strings.HasSuffix(duong, "/anh")
}

// KiemCSRFMultipart: handler nhận tệp gọi ngay sau ParseMultipartForm. Trả
// false nghĩa là đã trả lời 403 rồi, handler chỉ việc return.
func KiemCSRFMultipart(w http.ResponseWriter, r *http.Request) bool {
	gui := r.FormValue("_csrf")
	if gui == "" {
		gui = r.Header.Get("X-CSRF-Token")
	}
	if khopHang(gui, tokenMongDoi(r)) {
		return true
	}
	tuChoiCSRF(w, r)
	return false
}

// tokenMongDoi trả token đúng cho lượt yêu cầu này.
func tokenMongDoi(r *http.Request) string {
	if c, err := r.Cookie(tenCookie); err == nil && c.Value != "" {
		return tokenTuPhien(c.Value)
	}
	if c, err := r.Cookie(tenCookieCSRF); err == nil {
		return c.Value
	}
	return ""
}

func tuChoiCSRF(w http.ResponseWriter, r *http.Request) {
	if nd, ok := NguoiDangNhap(r); ok {
		GhiNhatKy(MucNhatKy{
			Ai: nd.Ten, IP: ipCua(r), Viec: r.Method,
			Duong: catBot(r.URL.Path, 200), KetQua: "csrf-hong",
		})
	}
	// Trả chữ trơn chứ không render template: lỗi này gần như luôn là phiên
	// đã cũ sau khi khởi động lại máy chủ, và câu cần nói là "tải lại trang".
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("Phiên làm việc đã cũ. Tải lại trang rồi gửi lại."))
}

// boCSRF chặn mọi POST/PUT/PATCH/DELETE không mang token đúng.
func boCSRF(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Khách chưa đăng nhập vẫn cần token cho form gửi yêu cầu. Đặt cookie
		// ngay từ lượt GET đầu tiên để trang render ra đã có token khớp.
		token := ""
		if c, err := r.Cookie(tenCookieCSRF); err == nil && len(c.Value) == 64 {
			token = c.Value
		} else {
			token = maNgauNhien(32)
			http.SetCookie(w, &http.Cookie{
				Name:     tenCookieCSRF,
				Value:    token,
				Path:     "/",
				HttpOnly: false, // để script đọc được nếu sau này cần gửi bằng fetch
				Secure:   HTTPSBat,
				SameSite: http.SameSiteLaxMode,
				MaxAge:   12 * 3600,
			})
		}
		r = r.WithContext(context.WithValue(r.Context(), ctxCSRF, token))

		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			h.ServeHTTP(w, r)
			return
		}
		if !hopLeCSRF(w, r, token) {
			return
		}
		h.ServeHTTP(w, r)
	})
}

func hopLeCSRF(w http.ResponseWriter, r *http.Request, tokenKhach string) bool {
	dung := tokenMongDoi(r)
	if dung == "" {
		dung = tokenKhach
	}

	// Biểu mẫu có tệp: token nằm trong thân, không đọc được ở đây. Chỉ những
	// đường trong danh sách mới được đi tiếp, và handler của chúng tự kiểm.
	if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
		if !multipartChoPhep(r.URL.Path) {
			tuChoiCSRF(w, r)
			return false
		}
		return true
	}

	// Header trước, rồi mới tới form. ParseForm ở đây an toàn với biểu mẫu
	// thường: handler gọi lại sẽ dùng kết quả đã có.
	gui := r.Header.Get("X-CSRF-Token")
	if gui == "" {
		r.ParseForm()
		gui = r.PostFormValue("_csrf")
	}
	if !khopHang(gui, dung) {
		tuChoiCSRF(w, r)
		return false
	}
	return true
}
