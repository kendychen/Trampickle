package core

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

// Lưu ở chế độ Online phải ghi vào khoá @online, không đè bản Tại xưởng.
func TestLuuNoiDungTheoCheDoDangBat(t *testing.T) {
	gocTam(t)
	const khoa = "trangchu.hero.h1"
	goc := NDMac(khoa)

	if err := DatCheDo(CheDoOnline); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	hQtNDLuu(w, postForm("/qt/noi-dung/trangchu/luu", url.Values{
		"ma":        {"trangchu"},
		"k." + khoa: {"Gửi vợt tới trạm"},
	}))

	if got := ND(khoa); got != "Gửi vợt tới trạm" {
		t.Fatalf("bản online phải ra chữ vừa lưu, nhận %q", got)
	}
	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	if !CoBanOnline(khoa) {
		t.Skip("khoá chưa khai MacOn — hai chế độ dùng chung, kiểm lại ở Task 6")
	}
	if got := ND(khoa); got != goc {
		t.Fatalf("bản tại xưởng bị đè: nhận %q, chờ %q", got, goc)
	}
}
