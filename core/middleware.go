package core

// Lớp bọc quanh mux: header an toàn, nonce cho CSP, kiểm CSRF, ghi nhật ký.
//
// Vì sao bọc ngoài mux chứ không gắn vào từng handler: có hơn tám mươi route.
// Gắn tay từng cái thì chỉ cần thêm một route mới mà quên bọc là thủng, và
// không có cách nào nhìn ra đã quên. Bọc ở một chỗ thì route mới tự có.

import (
	"context"
	"net/http"
	"strings"
)

// khoaCtx là kiểu riêng để khoá context không đụng khoá của gói khác.
type khoaCtx int

const (
	ctxNonce khoaCtx = iota
	ctxCSRF
)

// NewHandler trả về mux đã bọc đủ lớp. main.go dùng cái này, không dùng
// NewMux trực tiếp — NewMux vẫn xuất khẩu vì test dựng mux trần cho nhanh.
func NewHandler(public bool) http.Handler {
	return boHeader(boCSRF(boNhatKy(NewMux(public))))
}

// --- Header an toàn --------------------------------------------------

// boHeader gắn header cho MỌI phản hồi, kể cả file tĩnh và trang lỗi.
func boHeader(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nonce := maNgauNhien(16)

		hd := w.Header()
		// Trình duyệt không được tự đoán kiểu file. Thiếu cái này thì một
		// ảnh khách tải lên mà bên trong là HTML sẽ chạy như HTML.
		hd.Set("X-Content-Type-Options", "nosniff")
		// Không cho nhúng trang này vào iframe của ai: chặn clickjacking
		// kiểu phủ một nút trong suốt lên nút "Xoá đơn".
		hd.Set("X-Frame-Options", "DENY")
		// Bấm link ra ngoài thì đừng để lộ cả đường dẫn — /qt/don/DH-12 nói
		// cho trang kia biết cả cấu trúc admin lẫn mã đơn của khách.
		hd.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// Trang này không dùng camera, mic hay định vị. Tắt sẵn để một đoạn
		// script lọt vào cũng không hỏi quyền được.
		hd.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=(), payment=(), usb=()")
		hd.Set("Content-Security-Policy", chuoiCSP(nonce))

		// HSTS chỉ khi thật sự có HTTPS. Gửi qua HTTP thì trình duyệt bỏ
		// qua, nhưng bật nhầm lúc còn chạy HTTP ở máy nhà là tự khoá mình
		// khỏi localhost trong sáu tháng.
		if HTTPSBat {
			hd.Set("Strict-Transport-Security", "max-age=15768000; includeSubDomains")
		}

		h.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxNonce, nonce)))
	})
}

// chuoiCSP dựng chính sách. Mỗi lần tải trang một nonce khác nhau, nên một
// thẻ <script> do kẻ khác chèn vào sẽ không có nonce đúng và không chạy.
//
// style-src còn 'unsafe-inline': cả giao diện đang dùng thuộc tính style=""
// rải khắp template, và style="" KHÔNG nhận nonce. Bỏ nó phải viết lại toàn
// bộ phần trình bày — để lại, vì XSS qua CSS ở đây không dẫn tới đâu.
//
// img-src có data: cho ảnh SVG nhúng thẳng, blob: cho ảnh xem trước lúc
// khách chọn file. connect-src 'self' đủ vì mọi lệnh gọi đều về máy chủ này.
func chuoiCSP(nonce string) string {
	return strings.Join([]string{
		"default-src 'self'",
		"script-src 'self' 'nonce-" + nonce + "'",
		"style-src 'self' 'unsafe-inline'",
		"img-src 'self' data: blob:",
		"media-src 'self' blob:",
		"font-src 'self' data:",
		"connect-src 'self'",
		"form-action 'self'",
		"frame-ancestors 'none'",
		"base-uri 'none'",
		"object-src 'none'",
	}, "; ")
}

// nonceCua lấy nonce của lượt yêu cầu này để template gắn vào <script>.
// Rỗng nghĩa là handler chạy ngoài boHeader — trong test dựng mux trần.
func nonceCua(r *http.Request) string {
	s, _ := r.Context().Value(ctxNonce).(string)
	return s
}

// --- Ghi nhật ký ------------------------------------------------------

// boNhatKy ghi lại mọi thao tác GHI của người đã đăng nhập.
//
// Chỉ ghi POST/PUT/DELETE: GET là xem, mà xem thì không đổi gì. Ghi cả GET
// sẽ làm file phình lên vài nghìn dòng mỗi ngày và chôn mất những dòng đáng
// đọc — nhật ký không ai đọc nổi thì bằng không có.
func boNhatKy(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead {
			h.ServeHTTP(w, r)
			return
		}
		nd, daVao := NguoiDangNhap(r)
		if !daVao {
			// Form khách (gửi yêu cầu, tra cứu) không vào nhật ký quản trị.
			// /dang-nhap tự ghi lấy, vì chỉ nó biết đăng nhập trúng hay trượt.
			h.ServeHTTP(w, r)
			return
		}

		bm := &batMa{ResponseWriter: w, ma: http.StatusOK}
		h.ServeHTTP(bm, r)

		// Đọc form SAU khi handler chạy: handler đã gọi ParseForm/
		// ParseMultipartForm nên r.PostForm có sẵn, không phải đọc lại body
		// (đọc trước ở đây sẽ nuốt mất body của chính handler).
		mo := ""
		if r.PostForm != nil {
			mo = locTruongCam(r.PostForm)
		}
		GhiNhatKy(MucNhatKy{
			Ai:       nd.Ten,
			IP:       ipCua(r),
			Viec:     r.Method,
			Duong:    catBot(r.URL.Path, 200),
			DoiTuong: mo,
			KetQua:   ketQuaTheoMa(bm.ma),
		})
	})
}

func ketQuaTheoMa(ma int) string {
	switch {
	case ma >= 500:
		return "loi-may-chu"
	case ma == http.StatusForbidden:
		return "bi-tu-choi"
	case ma >= 400:
		return "loi-yeu-cau"
	default:
		return "ok"
	}
}

// batMa nhớ mã trạng thái để nhật ký phân biệt được làm xong với bị chặn.
type batMa struct {
	http.ResponseWriter
	ma      int
	daGhiMa bool
}

func (b *batMa) WriteHeader(ma int) {
	if !b.daGhiMa {
		b.ma, b.daGhiMa = ma, true
	}
	b.ResponseWriter.WriteHeader(ma)
}

func (b *batMa) Write(p []byte) (int, error) {
	b.daGhiMa = true // ghi thân mà chưa đặt mã = 200
	return b.ResponseWriter.Write(p)
}

// Unwrap để http.ResponseController (nếu sau này dùng) xuống được lớp gốc.
func (b *batMa) Unwrap() http.ResponseWriter { return b.ResponseWriter }
