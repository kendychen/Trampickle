package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
)

// dungCauHoiTrong: Root tạm, danh sách rỗng — đúng trạng thái máy chủ trước
// khi Kendy gõ câu đầu tiên.
func dungCauHoiTrong(t *testing.T) {
	t.Helper()
	cuRoot := Root
	cauHoiMu.Lock()
	cu := cauHoiDs
	cauHoiDs = nil
	cauHoiMu.Unlock()
	t.Cleanup(func() {
		Root = cuRoot
		cauHoiMu.Lock()
		cauHoiDs = cu
		cauHoiMu.Unlock()
	})
	Root = t.TempDir()
	if err := os.MkdirAll(P("data"), 0o755); err != nil {
		t.Fatal(err)
	}
}

// Chưa có file không phải lỗi: máy chủ mới deploy lên chưa có data/cau-hoi.yaml,
// nổ ở đây là cả trang web không lên.
func TestNapCauHoiChuaCoFile(t *testing.T) {
	dungCauHoiTrong(t)
	if err := NapCauHoi(); err != nil {
		t.Fatalf("chưa có file mà báo lỗi: %v", err)
	}
	if len(DanhSachCauHoi()) != 0 {
		t.Fatal("chưa có file mà đã có câu")
	}
}

func TestLuuCauHoiRoiNapLai(t *testing.T) {
	dungCauHoiTrong(t)
	c, err := LuuCauHoi(CauHoi{Hoi: "  Sửa vợt   mất bao lâu?  ", Dap: "1–3 ngày."})
	if err != nil {
		t.Fatal(err)
	}
	// Khoảng trắng thừa gom lại: câu hỏi dán từ tin nhắn Zalo hay dính hai
	// ba dấu cách, mà nó đi thẳng vào <h3> và vào khối JSON-LD.
	if c.Hoi != "Sửa vợt mất bao lâu?" {
		t.Fatalf("câu hỏi lưu thành %q", c.Hoi)
	}
	if c.Ma != "sua-vot-mat-bao-lau" {
		t.Fatalf("mã neo ra %q", c.Ma)
	}

	cauHoiMu.Lock()
	cauHoiDs = nil
	cauHoiMu.Unlock()
	if err := NapCauHoi(); err != nil {
		t.Fatal(err)
	}
	ds := DanhSachCauHoi()
	if len(ds) != 1 || ds[0].Dap != "1–3 ngày." {
		t.Fatalf("nạp lại ra %v", ds)
	}
}

// Mã sinh từ chính câu hỏi, nên hai câu na ná nhau là đụng mã. Đụng mà đè lên
// nhau thì câu cũ biến mất không báo gì.
func TestMaCauHoiKhongTrung(t *testing.T) {
	dungCauHoiTrong(t)
	for i := 0; i < 3; i++ {
		if _, err := LuuCauHoi(CauHoi{Hoi: "Có bảo hành không?", Dap: "Có."}); err != nil {
			t.Fatal(err)
		}
	}
	ma := map[string]bool{}
	for _, c := range DanhSachCauHoi() {
		if ma[c.Ma] {
			t.Fatalf("mã %q lặp", c.Ma)
		}
		ma[c.Ma] = true
	}
	if !ma["co-bao-hanh-khong"] || !ma["co-bao-hanh-khong-2"] || !ma["co-bao-hanh-khong-3"] {
		t.Fatalf("mã sinh ra: %v", ma)
	}
}

// Câu mới xuống cuối. Chen lên đầu thì thêm câu phụ nào cũng phải đi sắp lại
// thứ tự cả trang.
func TestCauMoiXuongCuoi(t *testing.T) {
	dungCauHoiTrong(t)
	for _, h := range []string{"Câu một", "Câu hai", "Câu ba"} {
		if _, err := LuuCauHoi(CauHoi{Hoi: h, Dap: "x"}); err != nil {
			t.Fatal(err)
		}
	}
	ds := DanhSachCauHoi()
	for i, muon := range []string{"Câu một", "Câu hai", "Câu ba"} {
		if ds[i].Hoi != muon {
			t.Fatalf("vị trí %d là %q, muốn %q", i, ds[i].Hoi, muon)
		}
	}

	// Đặt tay số nhỏ thì nhảy lên đầu.
	cuoi := ds[2]
	cuoi.ThuTu = 1
	if _, err := LuuCauHoi(cuoi); err != nil {
		t.Fatal(err)
	}
	if got := DanhSachCauHoi()[0].Hoi; got != "Câu ba" {
		t.Fatalf("đặt thứ tự 1 mà đầu danh sách vẫn là %q", got)
	}
}

func TestLuuCauHoiChanSaiSot(t *testing.T) {
	dungCauHoiTrong(t)
	xau := []struct {
		ten string
		c   CauHoi
	}{
		{"không có câu hỏi", CauHoi{Hoi: "   ", Dap: "có"}},
		{"không có câu trả lời", CauHoi{Hoi: "Hỏi gì đó?", Dap: "  "}},
		{"câu hỏi dài quá", CauHoi{Hoi: strings.Repeat("à", 201), Dap: "có"}},
		{"sửa câu không tồn tại", CauHoi{Ma: "khong-co", Hoi: "Hỏi gì đó?", Dap: "có"}},
	}
	for _, x := range xau {
		if _, err := LuuCauHoi(x.c); err == nil {
			t.Errorf("%s: lẽ ra phải báo lỗi", x.ten)
		}
	}
	if _, err := os.Stat(fileCauHoi()); !os.IsNotExist(err) {
		t.Error("lưu hụt mà vẫn ghi ra file")
	}
	// 200 ký tự có dấu vẫn phải lọt: giới hạn đếm rune, không đếm byte.
	if _, err := LuuCauHoi(CauHoi{Hoi: strings.Repeat("à", 200), Dap: "có"}); err != nil {
		t.Errorf("200 ký tự có dấu bị chặn: %v", err)
	}
}

// Ẩn là giấu khỏi trang khách mà vẫn giữ chữ cho trang quản lý.
func TestCauHoiAnKhongRaTrangKhach(t *testing.T) {
	dungCauHoiTrong(t)
	a, err := LuuCauHoi(CauHoi{Hoi: "Nhận vợt ở tỉnh không?", Dap: "Có."})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LuuCauHoi(CauHoi{Hoi: "Câu còn hiện", Dap: "x"}); err != nil {
		t.Fatal(err)
	}
	a.An = true
	if _, err := LuuCauHoi(a); err != nil {
		t.Fatal(err)
	}
	if len(DanhSachCauHoi()) != 2 {
		t.Fatal("ẩn xong mất luôn khỏi trang quản lý")
	}
	hien := CauHoiHien()
	if len(hien) != 1 || hien[0].Hoi != "Câu còn hiện" {
		t.Fatalf("trang khách vẫn thấy câu ẩn: %v", hien)
	}
}

func TestXoaCauHoi(t *testing.T) {
	dungCauHoiTrong(t)
	c, err := LuuCauHoi(CauHoi{Hoi: "Xoá thử", Dap: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if err := XoaCauHoi("khong-co"); err == nil {
		t.Error("xoá mã không có mà im lặng")
	}
	if err := XoaCauHoi(c.Ma); err != nil {
		t.Fatal(err)
	}
	if _, co := TimCauHoi(c.Ma); co {
		t.Error("xoá rồi vẫn tìm thấy")
	}
}

// DapTron đi thẳng vào khối JSON-LD. Còn sót thẻ HTML thì ô "Mọi người cũng
// hỏi" của Google hiện ra đúng mấy ký tự rác đó.
func TestDapTronBocHetThe(t *testing.T) {
	c := CauHoi{Dap: "Phần lớn **1–3 ngày**.\n\nXem [bảng dịch vụ](/dich-vu) để biết từng việc."}
	got := c.DapTron()
	if strings.ContainsAny(got, "<>") {
		t.Fatalf("còn thẻ: %q", got)
	}
	// Thẻ thay bằng khoảng trắng chứ không bằng rỗng: "<p>a</p><p>b</p>" nối
	// thẳng thì thành "ab".
	if !strings.Contains(got, "ngày. Xem") {
		t.Fatalf("hai đoạn dính vào nhau: %q", got)
	}
	if strings.Contains(got, "**") || strings.Contains(got, "](") {
		t.Fatalf("còn dấu markdown: %q", got)
	}
}

// --- Khối FAQPage ----------------------------------------------------------

// Chưa có câu nào thì KHÔNG phát khối rỗng: FAQPage không có mainEntity là dữ
// liệu sai, Google phạt cả trang chứ không bỏ riêng khối.
func TestFAQPageChiCoKhiCoCau(t *testing.T) {
	dungCauHoiTrong(t)
	g, _ := docLD(t, chungThu("/cau-hoi"))["@graph"].([]any)
	if nutTheoLoai(g, "FAQPage") != nil {
		t.Fatal("chưa có câu nào mà đã phát FAQPage")
	}

	if _, err := LuuCauHoi(CauHoi{Hoi: "Sửa mất bao lâu?", Dap: "**1–3 ngày.**"}); err != nil {
		t.Fatal(err)
	}
	an, err := LuuCauHoi(CauHoi{Hoi: "Câu đang ẩn", Dap: "x"})
	if err != nil {
		t.Fatal(err)
	}
	an.An = true
	if _, err := LuuCauHoi(an); err != nil {
		t.Fatal(err)
	}

	g, _ = docLD(t, chungThu("/cau-hoi"))["@graph"].([]any)
	f := nutTheoLoai(g, "FAQPage")
	if f == nil {
		t.Fatal("có câu rồi mà không phát FAQPage")
	}
	muc, _ := f["mainEntity"].([]any)
	if len(muc) != 1 {
		t.Fatalf("có %d câu trong khối, muốn 1 (câu ẩn phải bị loại)", len(muc))
	}
	q, _ := muc[0].(map[string]any)
	if q["name"] != "Sửa mất bao lâu?" {
		t.Fatalf("tên câu hỏi: %v", q["name"])
	}
	tl, _ := q["acceptedAnswer"].(map[string]any)
	if tl["text"] != "1–3 ngày." {
		t.Fatalf("câu trả lời trong khối: %v", tl["text"])
	}
}

// Khối FAQPage chỉ được ra ở đúng trang /cau-hoi. Đặt ở mọi trang thì Google
// coi cả site là một trang hỏi đáp.
func TestFAQPageChiOTrangCauHoi(t *testing.T) {
	dungCauHoiTrong(t)
	if _, err := LuuCauHoi(CauHoi{Hoi: "Sửa mất bao lâu?", Dap: "1–3 ngày."}); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{"/", "/dich-vu", "/quy-trinh"} {
		g, _ := docLD(t, chungThu(d))["@graph"].([]any)
		if nutTheoLoai(g, "FAQPage") != nil {
			t.Errorf("%s cũng phát FAQPage", d)
		}
	}
}

// --- Trang ------------------------------------------------------------------

func TestTrangCauHoiDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)

	// Chưa có câu nào: trang vẫn phải lên, và nói rõ là chưa có.
	than := moTrang(t, mux, ck, "/cau-hoi")
	if !strings.Contains(than, ND("cauhoi.trong")) {
		t.Error("trang rỗng không hiện câu thay thế")
	}

	f := url.Values{
		"hoi": {"Sửa vợt mất bao lâu?"},
		"dap": {"Phần lớn **1–3 ngày**."},
	}
	r := httptest.NewRequest("POST", "/qt/cau-hoi/luu", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("POST lưu trả %d, thân: %s", w.Code, catBot(w.Body.String(), 300))
	}

	than = moTrang(t, mux, ck, "/cau-hoi")
	if !strings.Contains(than, "Sửa vợt mất bao lâu?") {
		t.Error("câu vừa thêm không ra trang khách")
	}
	// Markdown phải được dựng thành HTML chứ không hiện nguyên dấu sao.
	if !strings.Contains(than, "<b>1–3 ngày</b>") {
		t.Error("câu trả lời không dựng markdown")
	}
	if !strings.Contains(than, `id="sua-vot-mat-bao-lau"`) {
		t.Error("thiếu neo để gửi riêng một câu cho khách")
	}

	// Trang quản lý dựng được và thấy câu vừa thêm.
	qt := moTrang(t, mux, ck, "/qt/cau-hoi")
	if !strings.Contains(qt, "Sửa vợt mất bao lâu?") {
		t.Error("trang quản lý không thấy câu vừa thêm")
	}

	// Và trang phải nằm trong sitemap, nếu không thì làm SEO cho ai.
	sm := moTrang(t, mux, ck, "/sitemap.xml")
	if !strings.Contains(sm, "/cau-hoi<") {
		t.Error("sitemap thiếu /cau-hoi")
	}
}

// Thợ không được sửa chữ đứng tên trạm nói với khách.
func TestThoKhongVaoDuocQtCauHoi(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	r := httptest.NewRequest("GET", "/qt/cau-hoi", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == http.StatusOK {
		t.Fatal("thợ vào được /qt/cau-hoi")
	}
}
