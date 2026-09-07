package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Route agent nằm trên internet từ bản này trở đi. Trước kia nó được bảo vệ
// bằng cách KHÔNG đăng ký ở chế độ public; giờ bảo vệ bằng canLaChu. Nếu ai
// đó lỡ tay bỏ canLaChu thì mọi khách vãng lai đều hỏi được Gemini bằng hạn
// mức của trạm — nên phân quyền này phải có test, không phải đọc code mà tin.
func TestRouteAgentChiChoChu(t *testing.T) {
	goi := func(mux *http.ServeMux, ck *http.Cookie, duong string) int {
		r := httptest.NewRequest("GET", duong, nil)
		if ck != nil {
			r.AddCookie(ck)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Code
	}

	muxChu, ckChu := dungTrangThu(t, VaiTroChu)
	if c := goi(muxChu, ckChu, "/noi-bo"); c != http.StatusOK {
		t.Errorf("chủ vào /noi-bo trên bản public trả %d, muốn 200", c)
	}
	// Chưa đăng nhập thì bị đá về trang đăng nhập, không phải 404.
	if c := goi(muxChu, nil, "/noi-bo"); c != http.StatusSeeOther {
		t.Errorf("khách lạ vào /noi-bo trả %d, muốn 303", c)
	}
	// Dựng index chỉ có ở máy nhà: một lần bấm là hàng trăm request nhúng.
	if c := goi(muxChu, ckChu, "/api/index"); c != http.StatusNotFound {
		t.Errorf("/api/index có mặt trên bản public (trả %d)", c)
	}

	muxTho, ckTho := dungTrangThu(t, VaiTroTho)
	for _, duong := range []string{"/noi-bo", "/api/hoi", "/api/tim?q=x", "/api/thong-ke"} {
		if c := goi(muxTho, ckTho, duong); c != http.StatusForbidden {
			t.Errorf("thợ gọi %s trả %d, muốn 403", duong, c)
		}
	}
}
