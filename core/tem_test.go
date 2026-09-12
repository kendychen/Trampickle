package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBonSoCuoi(t *testing.T) {
	cac := []struct{ vao, ra string }{
		{"0912345678", "5678"},
		{"091 234 5678", "5678"},
		{"+84912345678", "5678"},
		{"123", "123"},
		{"", ""},
		{"zalo: kendy", "kendy"},
	}
	for _, c := range cac {
		if got := bonSoCuoi(c.vao); got != c.ra {
			t.Errorf("bonSoCuoi(%q) = %q, muốn %q", c.vao, got, c.ra)
		}
	}
}

// Tem một đơn: đủ bốn thứ cần để tra cây vợt trên giá, và KHÔNG có số điện
// thoại đầy đủ — tem dán ngoài đồ, ai cầm cây vợt cũng đọc được.
func TestTemCoMaDon(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	nay := time.Now().Format("2006-01-02")
	d := donThu(t, &Don{
		Ma: "TV-2609-910", Ngay: nay, TrangThai: TTDangSua,
		KhachTen: "Anh Sáu", KhachLienHe: "0912345678",
		HenTraNgay: "2026-09-20", VotHang: "Yonex Astrox 99",
	})

	s := moTrang(t, mux, ck, "/qt/don/"+d.Ma+"/tem")
	for _, muon := range []string{d.Ma, "Anh Sáu", "5678", "20/09", "Yonex Astrox 99"} {
		if !strings.Contains(s, muon) {
			t.Errorf("tem thiếu %q", muon)
		}
	}
	if strings.Contains(s, "0912345678") {
		t.Error("tem in nguyên số điện thoại khách")
	}
	if !strings.Contains(s, "@page") || !strings.Contains(s, "@media print") {
		t.Error("tem thiếu CSS in")
	}
}

// Đơn không hẹn trả, không tên khách: tem vẫn ra, chỗ trống nói rõ là trống
// chứ không để mấy chữ "<no value>" của template.
func TestTemDonTrongVanIn(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	d := donThu(t, &Don{Ma: "TV-2609-911", TrangThai: TTMoi})
	s := moTrang(t, mux, ck, "/qt/don/"+d.Ma+"/tem")
	if !strings.Contains(s, d.Ma) {
		t.Error("tem thiếu mã đơn")
	}
	if strings.Contains(s, "<no value>") || strings.Contains(s, "0001-01-01") {
		t.Error("tem để lộ giá trị rỗng thô")
	}
}

// In hàng loạt: buổi sáng nhận năm cây thì tick năm dòng rồi in một lượt.
func TestTemNhieuDon(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	a := donThu(t, &Don{Ma: "TV-2609-912", KhachTen: "Chị Bảy", VotHang: "Lining"})
	b := donThu(t, &Don{Ma: "TG-2609-913", KhachTen: "Anh Tám", GiayHang: "Nike", GiaySize: "42"})

	s := moTrang(t, mux, ck, "/qt/tem?ma="+a.Ma+"&ma="+b.Ma+"&ma=TV-KHONG-CO")
	if !strings.Contains(s, a.Ma) || !strings.Contains(s, b.Ma) {
		t.Error("tem hàng loạt thiếu đơn")
	}
	if strings.Contains(s, "TV-KHONG-CO") {
		t.Error("mã không tồn tại vẫn ra một cái tem rỗng")
	}

	// Không tick dòng nào: đừng đưa ra tờ giấy trắng.
	r := httptest.NewRequest("GET", "/qt/tem", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("/qt/tem không mã trả %d, muốn 404", w.Code)
	}
}
