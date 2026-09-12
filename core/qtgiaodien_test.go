package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// postForm dựng một POST dạng form thường. Dùng chung cho các test handler.
func postForm(duong string, v url.Values) *http.Request {
	r := httptest.NewRequest(http.MethodPost, duong, strings.NewReader(v.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return r
}

func TestDoiCheDoQuaAdmin(t *testing.T) {
	gocTam(t)
	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	hQtCheDo(w, postForm("/qt/giao-dien/che-do", url.Values{"che_do": {CheDoOnline}}))

	if w.Code != http.StatusSeeOther {
		t.Fatalf("phải redirect 303, nhận %d", w.Code)
	}
	if !LaOnline() {
		t.Fatal("chế độ chưa đổi sang online")
	}
}

func TestDoiCheDoLaBiTuChoiOAdmin(t *testing.T) {
	gocTam(t)
	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	hQtCheDo(w, postForm("/qt/giao-dien/che-do", url.Values{"che_do": {"linh-tinh"}}))

	if LaOnline() {
		t.Fatal("giá trị lạ mà vẫn đổi chế độ")
	}
	if w.Code != http.StatusSeeOther {
		t.Fatalf("vẫn phải quay về trang có lời báo, nhận %d", w.Code)
	}
}
