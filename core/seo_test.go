package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// docLD dựng khối JSON-LD của một trang rồi mở ra thành dữ liệu Go.
func docLD(t *testing.T, c Chung) map[string]any {
	t.Helper()
	s := string(c.DuLieuCoCau())
	if s == "" {
		return nil
	}
	var ra map[string]any
	if err := json.Unmarshal([]byte(s), &ra); err != nil {
		t.Fatalf("JSON-LD hỏng: %v\n%s", err, s)
	}
	return ra
}

func chungThu(duong string) Chung {
	return Chung{
		Brand:     ThuongHieu{Ten: "Trạm Pickle", DongMoTa: "Chuyên sửa chữa"},
		LienHe:    LienHe{DienThoai: "0846161368", DiaChi: "Số 69, Đại Mỗ, Hà Nội"},
		Canonical: "https://trampickle.vn" + duong,
		Duong:     duong,
	}
}

func nutTheoLoai(g []any, loai string) map[string]any {
	for _, n := range g {
		if m, ok := n.(map[string]any); ok && m["@type"] == loai {
			return m
		}
	}
	return nil
}

// Trạm nhận hàng gửi từ cả nước, nên bản khai phải nói cả hai điều cùng lúc:
// có chỗ thật ở Hà Nội (LocalBusiness + address) và phục vụ ngoài Hà Nội
// (areaServed). Mất một trong hai là mất một nửa số khách.
func TestLDKhaiPhucVuCaNuoc(t *testing.T) {
	d := docLD(t, chungThu("/"))
	g, _ := d["@graph"].([]any)
	tram := nutTheoLoai(g, "LocalBusiness")
	if tram == nil {
		t.Fatal("có địa chỉ mà không khai LocalBusiness")
	}
	vung, _ := tram["areaServed"].(map[string]any)
	if vung == nil {
		t.Fatal("không khai areaServed — Google đọc thành cửa hàng chỉ phục vụ quanh nhà")
	}
	if vung["@type"] != "Country" || vung["name"] != "Việt Nam" {
		t.Errorf("areaServed sai: %v", vung)
	}
	// Chưa có địa chỉ thì vẫn phải nói được là nhận hàng cả nước.
	c := chungThu("/")
	c.LienHe.DiaChi = ""
	d2 := docLD(t, c)
	g2, _ := d2["@graph"].([]any)
	if to := nutTheoLoai(g2, "Organization"); to == nil || to["areaServed"] == nil {
		t.Error("chưa điền địa chỉ thì mất luôn areaServed")
	}
}

// Trang chủ: chỉ có cái trạm, không có đường dẫn — "Trang chủ › Trang chủ" là
// một dòng vô nghĩa.
func TestLDTrangChuChiCoTram(t *testing.T) {
	d := docLD(t, chungThu("/"))
	g, _ := d["@graph"].([]any)
	if len(g) != 1 {
		t.Fatalf("trang chủ đáng ra chỉ khai một nút, có %d", len(g))
	}
	tram := nutTheoLoai(g, "LocalBusiness")
	if tram == nil {
		t.Fatal("có địa chỉ mà không khai LocalBusiness")
	}
	if tram["telephone"] != "0846161368" {
		t.Errorf("thiếu số điện thoại: %v", tram["telephone"])
	}
}

// Chưa điền địa chỉ thì hạ xuống Organization: LocalBusiness thiếu address là
// khối sai, Google bỏ cả khối chứ không bỏ riêng trường thiếu.
func TestLDChuaCoDiaChiThiKhongPhaiLocalBusiness(t *testing.T) {
	c := chungThu("/")
	c.LienHe.DiaChi = ""
	g, _ := docLD(t, c)["@graph"].([]any)
	if nutTheoLoai(g, "LocalBusiness") != nil {
		t.Error("không có địa chỉ mà vẫn khai LocalBusiness")
	}
	if nutTheoLoai(g, "Organization") == nil {
		t.Error("mất luôn nút trạm")
	}
}

func TestLDBaiVietCoDuongDanVaNgay(t *testing.T) {
	c := chungThu("/bai-viet/vot-pickleball-bi-nut")
	c.TieuDe = "Vợt Pickleball bị nứt: chỗ nào vá được"
	c.MoTa = "Bản đồ vùng đỏ và lý do."
	c.LoaiOG = "article"
	c.NgayDang = "2026-09-03"
	g, _ := docLD(t, c)["@graph"].([]any)

	dd := nutTheoLoai(g, "BreadcrumbList")
	if dd == nil {
		t.Fatal("thiếu đường dẫn")
	}
	muc, _ := dd["itemListElement"].([]any)
	if len(muc) != 3 {
		t.Fatalf("đường dẫn phải ba bậc, có %d", len(muc))
	}
	if m := muc[1].(map[string]any); m["name"] != "Bài viết" {
		t.Errorf("bậc giữa sai: %v", m["name"])
	}
	if m := muc[2].(map[string]any); m["name"] != c.TieuDe {
		t.Errorf("bậc cuối phải là tên bài: %v", m["name"])
	}

	bai := nutTheoLoai(g, "Article")
	if bai == nil {
		t.Fatal("thiếu nút bài viết")
	}
	// Bài chưa sửa lần nào: ngày sửa lấy theo ngày đăng chứ không để trống.
	if bai["dateModified"] != "2026-09-03" {
		t.Errorf("dateModified sai: %v", bai["dateModified"])
	}
	if bai["headline"] != c.TieuDe {
		t.Errorf("headline sai: %v", bai["headline"])
	}
}

// Trang quản trị và trang tra cứu đã chặn trong robots.txt thì cũng không khai
// dữ liệu — mời bot vào đúng chỗ vừa cấm nó vào là tự mâu thuẫn.
func TestLDBoQuaTrangDaChanBot(t *testing.T) {
	for _, d := range []string{"/qt", "/qt/don/123", "/tra-cuu", "/dang-nhap"} {
		if s := chungThu(d).DuLieuCoCau(); s != "" {
			t.Errorf("%s vẫn khai dữ liệu có cấu trúc", d)
		}
	}
	if s := chungThu("/tra-cuu-gi-do").DuLieuCoCau(); s == "" {
		t.Error("chặn nhầm trang chỉ trùng tiền tố")
	}
}

// Tiêu đề có ký tự HTML không được phép đóng sớm khối script.
func TestLDKhongThoatRaKhoiScript(t *testing.T) {
	c := chungThu("/bai-viet/x")
	c.TieuDe = `</script><img src=x>`
	c.LoaiOG = "article"
	if strings.Contains(string(c.DuLieuCoCau()), "</script>") {
		t.Error("chuỗi </script> lọt nguyên vào khối dữ liệu")
	}
}

func TestCSSRaTepRiengCoMaBam(t *testing.T) {
	if err := InitTemplates(); err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^/tinh/tp-[0-9a-f]{16}\.css$`).MatchString(cssDuong) {
		t.Fatalf("tên tệp CSS sai: %q", cssDuong)
	}
	if strings.Contains(string(cssNoiDung), "<style") {
		t.Error("còn nguyên thẻ <style> trong tệp .css")
	}
	if len(cssNoiDung) < 10000 {
		t.Fatalf("tệp CSS chỉ %d byte — nhiều khả năng cắt hụt", len(cssNoiDung))
	}

	w := httptest.NewRecorder()
	hCSS(w, httptest.NewRequest("GET", cssDuong, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("mã trả về %d", w.Code)
	}
	if h := w.Header().Get("Cache-Control"); !strings.Contains(h, "immutable") {
		t.Errorf("thiếu cache dài hạn: %q", h)
	}

	// Mã băm cũ phải 404 chứ không được trả nội dung mới: cái tên đã hứa là
	// bất biến.
	w = httptest.NewRecorder()
	hCSS(w, httptest.NewRequest("GET", "/tinh/tp-0000000000000000.css", nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("mã băm sai vẫn được phục vụ: %d", w.Code)
	}
}
