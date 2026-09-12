package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Chạy trên chính vanhanh/bang-gia.yaml chứ không phải bản rút gọn: cái dễ
// hỏng ở đây là comment viết tay, khối `>` nhiều dòng, comment cuối dòng và
// mục cuối cùng sát khoá kế — bản rút gọn không có mấy thứ đó thì test xanh
// mà thật vẫn hỏng.
func dungDichVuThat(t *testing.T) string {
	t.Helper()
	goc, err := os.ReadFile(filepath.Join("..", "..", "vanhanh", "bang-gia.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cuRoot, cuCFG, cuGIA, cuGD := Root, CFG, GIA, GiaiDoan
	t.Cleanup(func() { Root, CFG, GIA, GiaiDoan = cuRoot, cuCFG, cuGIA, cuGD })

	Root = t.TempDir()
	CFG.DuongDan.BangGia = "vanhanh/bang-gia.yaml"
	if err := os.MkdirAll(filepath.Join(Root, "vanhanh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileBangGia(), goc, 0o644); err != nil {
		t.Fatal(err)
	}
	GIA = BangGia{}
	if err := yaml.Unmarshal(goc, &GIA); err != nil {
		t.Fatal(err)
	}
	GiaiDoan = GIA.GiaiDoanHienTai
	return string(goc)
}

func timO(t *testing.T, ma string) ODichVu {
	t.Helper()
	for _, o := range DanhSachODichVu() {
		if o.Ma == ma {
			return o
		}
	}
	t.Fatalf("không có dịch vụ %s", ma)
	return ODichVu{}
}

func docLaiFile(t *testing.T) (string, BangGia) {
	t.Helper()
	b, err := os.ReadFile(fileBangGia())
	if err != nil {
		t.Fatal(err)
	}
	var g BangGia
	if err := yaml.Unmarshal(b, &g); err != nil {
		t.Fatalf("file sau khi sửa không đọc được: %v", err)
	}
	return string(b), g
}

// Tra mục theo mã chứ không theo chỉ số: thứ tự trong bang-gia.yaml đổi mỗi
// lần trạm mở bán thêm một việc, test không được vỡ vì chuyện đó.
func timTrongFile(t *testing.T, g BangGia, ma string) DichVu {
	t.Helper()
	for _, d := range g.DichVu {
		if d.Ma == ma {
			return d
		}
	}
	t.Fatalf("file sau khi sửa không còn mục %s", ma)
	return DichVu{}
}

// Sửa một mục nằm giữa danh sách: mọi mục khác phải y nguyên từng dòng.
func TestSuaDichVuGiuNguyenPhanConLai(t *testing.T) {
	goc := dungDichVuThat(t)

	o := timO(t, "VA_MAT")
	o.Ten = "Vá mặt thủng nhỏ (đã đổi)"
	o.Gia[0] = "130000"
	o.BaoHanhThang = "6"
	doi, err := SuaDichVu(map[string]ODichVu{"VA_MAT": o})
	if err != nil {
		t.Fatal(err)
	}
	if len(doi) != 1 {
		t.Fatalf("báo đổi %v, muốn đúng một việc", doi)
	}

	moi, g := docLaiFile(t)
	if len(g.DichVu) != len(GIA.DichVu) {
		t.Fatalf("còn %d dịch vụ, trước có %d", len(g.DichVu), len(GIA.DichVu))
	}
	for _, d := range g.DichVu {
		if d.Ma != "VA_MAT" {
			continue
		}
		if d.Ten != o.Ten || d.BaoHanhThang != 6 || *d.Gia[0] != 130000 {
			t.Errorf("VA_MAT ghi sai: %+v", d)
		}
		if d.VatTu != 84000 || d.GioCong != 2.2 {
			t.Errorf("VA_MAT mất vat_tu/gio_cong: %+v", d)
		}
	}

	// Comment viết tay là thứ yaml.Marshal xoá sạch — đếm lại cho chắc.
	if a, b := strings.Count(goc, "#"), strings.Count(moi, "#"); a != b {
		t.Errorf("số dòng có # đổi từ %d thành %d — mất comment", a, b)
	}
	for _, mau := range []string{
		"# tiền trả tiệm gia công",
		"CHỈ bán kèm. Không bao giờ chạy quảng cáo riêng",
		"Trạm không có máy đo Rz/Rt",
		"# DỊCH VỤ KHÔNG BÁN",
	} {
		if !strings.Contains(moi, mau) {
			t.Errorf("mất đoạn %q", mau)
		}
	}
}

// Mở bán một việc đang ẩn: điền giá giai đoạn đang chạy là khách thấy ngay.
func TestSuaDichVuMoBan(t *testing.T) {
	dungDichVuThat(t)

	o := timO(t, "THAY_DE_GIAY")
	if o.DangHien != true {
		t.Fatalf("THAY_DE_GIAY đang phải hiện (bao_gia_rieng), ViSao=%q", o.ViSao)
	}
	o.Gia[0] = "500000"
	o.BaoGiaRieng = false
	if _, err := SuaDichVu(map[string]ODichVu{"THAY_DE_GIAY": o}); err != nil {
		t.Fatal(err)
	}

	moi, _ := docLaiFile(t)
	if strings.Contains(moi, "bao_gia_rieng: true\n    vat_tu: 300000") {
		t.Error("bỏ tick báo giá riêng mà khoá bao_gia_rieng vẫn còn")
	}
	sau := timO(t, "THAY_DE_GIAY")
	if !sau.DangHien || sau.Gia[0] != "500000" || sau.BaoGiaRieng {
		t.Errorf("sau khi sửa: %+v", sau)
	}
	// Nạp lại từ file phải ra đúng cái vừa ghi, không chỉ đúng trong bộ nhớ.
	if _, g := docLaiFile(t); *g.DichVu[7].Gia[0] != 500000 {
		t.Errorf("file ghi giá %v", g.DichVu[7].Gia[0])
	}
}

// Mục CUỐI danh sách, lại đang thiếu hai khoá: khoá thêm vào phải nằm trong
// mục, không được rơi xuống dưới khối comment ngăn cách với khong_ban.
func TestSuaDichVuThemKhoaOMucCuoi(t *testing.T) {
	dungDichVuThat(t)

	ma := GIA.DichVu[len(GIA.DichVu)-1].Ma
	o := timO(t, ma)
	if o.BaoHanhThang != "" {
		t.Fatalf("%s đáng ra chưa có khoá bao_hanh_thang: %+v", ma, o)
	}
	o.BaoHanhThang = "2"
	o.LeadTimeNgay = "4"
	o.DieuKien = "Trạm chưa có máy đo Rz/Rt. Anh chị có đánh giải thì đừng làm."
	if _, err := SuaDichVu(map[string]ODichVu{ma: o}); err != nil {
		t.Fatal(err)
	}

	moi, g := docLaiFile(t)
	cuoi := g.DichVu[len(g.DichVu)-1]
	if cuoi.Ma != ma || cuoi.BaoHanhThang != 2 || cuoi.LeadTimeNgay != 4 {
		t.Errorf("%s ghi sai: %+v", ma, cuoi)
	}
	if !strings.Contains(cuoi.DieuKien, "đánh giải thì đừng làm") {
		t.Errorf("điều kiện ghi sai: %q", cuoi.DieuKien)
	}
	if len(g.KhongBan) != len(GIA.KhongBan) {
		t.Errorf("khối khong_ban hỏng: còn %d mục", len(g.KhongBan))
	}
	if i, j := strings.Index(moi, "lead_time_ngay: 4"), strings.Index(moi, "khong_ban:"); i > j {
		t.Error("khoá mới rơi xuống sau khong_ban:")
	}
}

// Chặn ở chỗ đọc chữ, đừng để ghi ra file rồi mới biết sai.
func TestDocODichVuChanSaiSot(t *testing.T) {
	dungDichVuThat(t)
	cu := timO(t, "VA_MAT")

	dung := map[string]string{
		"ten": cu.Ten, "dieu_kien": "", "gia1": "120000", "gia2": "150000",
		"gia3": "200000", "tu_giai_doan": "1", "bao_hanh_thang": "3",
		"lead_time_ngay": "3", "bao_gia_rieng": "",
	}
	doc := func(sua map[string]string) error {
		f := map[string]string{}
		for k, v := range dung {
			f[k] = v
		}
		for k, v := range sua {
			f[k] = v
		}
		_, err := docODichVu(cu, func(k string) string { return f[k] })
		return err
	}

	if err := doc(nil); err != nil {
		t.Fatalf("bản đúng lại bị chặn: %v", err)
	}
	for ten, sua := range map[string]map[string]string{
		"tên rỗng":                 {"ten": "  "},
		"giá không có chữ số":      {"gia1": "gọi điện"},
		"giá quá to":               {"gia1": "9999999999"},
		"giai đoạn ngoài 1-3":      {"tu_giai_doan": "4"},
		"bảo hành không phải số":   {"bao_hanh_thang": "ba"},
		"mở bán mà không có giá":   {"gia1": "", "bao_gia_rieng": ""},
		"ngày làm chỉ có đầu trên": {"lead_time_ngay": "", "lead_time_ngay_den": "3"},
		"khoảng ngày bằng nhau":    {"lead_time_ngay": "5", "lead_time_ngay_den": "5"},
		"khoảng ngày ngược":        {"lead_time_ngay": "5", "lead_time_ngay_den": "2"},
	} {
		if err := doc(sua); err == nil {
			t.Errorf("%s: đáng ra phải báo lỗi", ten)
		}
	}

	// Bỏ giá giai đoạn 1 nhưng bật báo giá riêng thì hợp lệ.
	if err := doc(map[string]string{"gia1": "", "bao_gia_rieng": "1"}); err != nil {
		t.Errorf("chưa niêm yết giá mà vẫn bị chặn: %v", err)
	}
	// Gõ tiền kiểu người Việt.
	if err := doc(map[string]string{"gia1": "130.000đ"}); err != nil {
		t.Errorf("gõ 130.000đ bị chặn: %v", err)
	}
}

// Dựng trang thật qua router: template gọi sai tên trường thì build vẫn xanh,
// chỉ tới lúc bấm vào mới ra "Lỗi hiển thị trang".
func TestTrangDichVuDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, "chu")
	dungDichVuThat(t)

	than := moTrang(t, mux, ck, "/qt/dich-vu")
	for _, o := range DanhSachODichVu() {
		if !strings.Contains(than, o.Ma) {
			t.Errorf("trang thiếu %s", o.Ma)
		}
	}
	if !strings.Contains(than, "Thay đế giày") {
		t.Error("trang không hiện tên việc")
	}

	// Gửi biểu mẫu chỉ có một việc: những việc khác không có trong form thì
	// không được đụng tới, không thì mở trang con là xoá sạch phần còn lại.
	f := url.Values{
		"co.QUAN_GRIP":             {"1"},
		"ten.QUAN_GRIP":            {"Quấn grip"},
		"gia1.QUAN_GRIP":           {"60.000đ"},
		"gia2.QUAN_GRIP":           {"80000"},
		"gia3.QUAN_GRIP":           {"100000"},
		"tu_giai_doan.QUAN_GRIP":   {"1"},
		"bao_hanh_thang.QUAN_GRIP": {"1"},
		"lead_time_ngay.QUAN_GRIP": {"0"},
		"dieu_kien.QUAN_GRIP":      {""},
	}
	r := httptest.NewRequest("POST", "/qt/dich-vu", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("POST trả %d: %s", w.Code, catBot(w.Body.String(), 300))
	}
	if g := timO(t, "QUAN_GRIP"); g.Ten != "Quấn grip" || g.Gia[0] != "60000" {
		t.Errorf("QUAN_GRIP sau khi lưu: %+v", g)
	}
	if v := timO(t, "DAN_VIEN"); v.Ten != "Dán viền bong" || v.Gia[0] != "100000" {
		t.Errorf("việc không gửi lên mà bị đổi: %+v", v)
	}
}

// Thợ không được sửa bảng giá — cùng luật với /qt/nguong.
func TestThoKhongVaoDuocTrangDichVu(t *testing.T) {
	mux, ck := dungTrangThu(t, "tho")
	r := httptest.NewRequest("GET", "/qt/dich-vu", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code == http.StatusOK {
		t.Error("thợ mở được trang sửa bảng giá")
	}
}

// Đánh số thứ tự rồi xoá số ấy đi: khoá thu_tu phải xuất hiện rồi biến mất
// khỏi file, chứ không ở lại thành `thu_tu: 0`.
func TestSuaDichVuGhiThuTu(t *testing.T) {
	dungDichVuThat(t)

	o := timO(t, "QUAN_GRIP")
	o.ThuTu = "2"
	if _, err := SuaDichVu(map[string]ODichVu{"QUAN_GRIP": o}); err != nil {
		t.Fatal(err)
	}
	moi, g := docLaiFile(t)
	if !strings.Contains(moi, "thu_tu: 2") {
		t.Fatalf("file không có thu_tu: 2")
	}
	for _, d := range g.DichVu {
		if d.Ma == "QUAN_GRIP" && d.ThuTu != 2 {
			t.Fatalf("QUAN_GRIP thu_tu = %d, muốn 2", d.ThuTu)
		}
	}

	o = timO(t, "QUAN_GRIP")
	if o.ThuTu != "2" {
		t.Fatalf("đọc lên thu_tu = %q, muốn \"2\"", o.ThuTu)
	}
	o.ThuTu = ""
	if _, err := SuaDichVu(map[string]ODichVu{"QUAN_GRIP": o}); err != nil {
		t.Fatal(err)
	}
	if moi, _ = docLaiFile(t); strings.Contains(moi, "thu_tu:") {
		t.Fatalf("xoá số rồi mà khoá thu_tu vẫn còn trong file")
	}
}

// Luật xếp: có số đứng trước theo số tăng dần, bỏ trống xuống sau và giữ
// nguyên thứ tự file. Chưa đánh số việc nào thì danh sách không đổi.
func TestDichVuTatCaXepTheoThuTu(t *testing.T) {
	dungDichVuThat(t)

	truoc := DichVuTatCa()
	for i, d := range truoc {
		if d.Ma != GIA.DichVu[i].Ma {
			t.Fatalf("chưa đánh số mà thứ tự đã đổi ở vị trí %d", i)
		}
	}

	// Đẩy việc CUỐI lên đầu, việc áp chót lên thứ hai.
	n := len(GIA.DichVu)
	cuoi, apChot := GIA.DichVu[n-1].Ma, GIA.DichVu[n-2].Ma
	GIA.DichVu[n-1].ThuTu = 1
	GIA.DichVu[n-2].ThuTu = 7

	sau := DichVuTatCa()
	if sau[0].Ma != cuoi || sau[1].Ma != apChot {
		t.Fatalf("hai đầu danh sách là %s, %s — muốn %s, %s", sau[0].Ma, sau[1].Ma, cuoi, apChot)
	}
	for i := 2; i < len(sau); i++ {
		if sau[i].Ma != GIA.DichVu[i-2].Ma {
			t.Fatalf("đám chưa đánh số bị xáo ở vị trí %d: %s", i, sau[i].Ma)
		}
	}
}

// --- Khoảng ngày làm -------------------------------------------------

func TestLeadChuMotSoVaKhoang(t *testing.T) {
	for _, c := range []struct {
		tu, den int
		so, chu string
	}{
		{0, 0, "", ""},
		{0, 5, "", ""}, // chưa có đầu dưới thì đầu trên vô nghĩa
		{3, 0, "3", "3 ngày"},
		{3, 3, "3", "3 ngày"}, // khoảng rỗng, in như một con số
		{3, 1, "3", "3 ngày"}, // ngược đời, không in ngược
		{1, 3, "1–3", "1–3 ngày"},
	} {
		d := DichVu{LeadTimeNgay: c.tu, LeadTimeDen: c.den}
		if got := d.LeadSo(); got != c.so {
			t.Errorf("LeadSo(%d,%d) = %q, muốn %q", c.tu, c.den, got, c.so)
		}
		if got := d.LeadChu(); got != c.chu {
			t.Errorf("LeadChu(%d,%d) = %q, muốn %q", c.tu, c.den, got, c.chu)
		}
	}
}

func TestSuaDichVuGhiKhoangNgay(t *testing.T) {
	dungDichVuThat(t)

	o := timO(t, "PHU_NHAM")
	o.LeadTimeNgay = "2"
	o.LeadTimeDen = "5"
	if _, err := SuaDichVu(map[string]ODichVu{"PHU_NHAM": o}); err != nil {
		t.Fatal(err)
	}
	_, g := docLaiFile(t)
	d := timTrongFile(t, g, "PHU_NHAM")
	if d.LeadTimeNgay != 2 || d.LeadTimeDen != 5 {
		t.Fatalf("ghi sai: %+v", d)
	}
	if d.LeadChu() != "2–5 ngày" {
		t.Errorf("LeadChu = %q", d.LeadChu())
	}

	// Xoá đầu trên: khoá phải biến khỏi file, không được để lại dòng rỗng.
	sau := timO(t, "PHU_NHAM")
	sau.LeadTimeDen = ""
	if _, err := SuaDichVu(map[string]ODichVu{"PHU_NHAM": sau}); err != nil {
		t.Fatal(err)
	}
	chu, g2 := docLaiFile(t)
	if strings.Contains(chu, "lead_time_ngay_den") {
		t.Error("bỏ trống ô đến mà khoá lead_time_ngay_den vẫn còn trong file")
	}
	if timTrongFile(t, g2, "PHU_NHAM").LeadTimeNgay != 2 {
		t.Error("xoá đầu trên làm mất luôn đầu dưới")
	}
}

// Khoảng ngày hợp lệ đi qua được chỗ đọc chữ.
func TestDocODichVuNhanKhoangNgay(t *testing.T) {
	dungDichVuThat(t)
	cu := timO(t, "VA_MAT")
	f := map[string]string{
		"ten": cu.Ten, "gia1": "120000", "tu_giai_doan": "1",
		"lead_time_ngay": "1", "lead_time_ngay_den": "3",
	}
	moi, err := docODichVu(cu, func(k string) string { return f[k] })
	if err != nil {
		t.Fatal(err)
	}
	if moi.LeadTimeNgay != "1" || moi.LeadTimeDen != "3" {
		t.Fatalf("đọc sai: %q đến %q", moi.LeadTimeNgay, moi.LeadTimeDen)
	}
}

// Ô tích nổi bật ghi và xoá được khoá noi_bat, không đụng bao_gia_rieng.
func TestSuaDichVuNoiBat(t *testing.T) {
	dungDichVuThat(t)

	o := timO(t, "VA_MAT")
	if o.NoiBat {
		t.Fatal("VA_MAT đáng ra chưa nổi bật")
	}
	o.NoiBat = true
	if _, err := SuaDichVu(map[string]ODichVu{"VA_MAT": o}); err != nil {
		t.Fatal(err)
	}
	_, g := docLaiFile(t)
	var thay bool
	for _, d := range g.DichVu {
		if d.Ma == "VA_MAT" {
			thay = d.NoiBat
		}
	}
	if !thay {
		t.Fatal("khoá noi_bat không vào file")
	}

	sau := timO(t, "VA_MAT")
	sau.NoiBat = false
	if _, err := SuaDichVu(map[string]ODichVu{"VA_MAT": sau}); err != nil {
		t.Fatal(err)
	}
	if chu, _ := docLaiFile(t); strings.Contains(chu, "noi_bat") {
		t.Error("bỏ tích mà khoá noi_bat vẫn còn trong file")
	}
}

// --- Tắt tay từng việc -----------------------------------------------

// Tắt một việc: khoá an vào file, khách không thấy nữa, mà giá và mọi thứ
// khác trong mục vẫn nằm nguyên để bật lại là về y như cũ.
func TestSuaDichVuTatMotViec(t *testing.T) {
	goc := dungDichVuThat(t)

	o := timO(t, "VA_MAT")
	if o.An || !o.DangHien {
		t.Fatalf("VA_MAT đáng ra đang bán bình thường: %+v", o)
	}
	o.An = true
	if _, err := SuaDichVu(map[string]ODichVu{"VA_MAT": o}); err != nil {
		t.Fatal(err)
	}

	moi, g := docLaiFile(t)
	if !strings.Contains(moi, "\n    an: true") {
		t.Fatal("khoá an không vào file")
	}
	for _, d := range g.DichVu {
		if d.Ma != "VA_MAT" {
			continue
		}
		if !d.An {
			t.Error("đọc lại file mà an = false")
		}
		// Tắt KHÔNG phải là xoá giá: bật lại phải ra đúng bảng giá cũ.
		if d.Gia[0] == nil || *d.Gia[0] != 120000 || d.VatTu != 84000 {
			t.Errorf("tắt xong mất số của mục: %+v", d)
		}
	}
	if a, b := strings.Count(goc, "#"), strings.Count(moi, "#"); a != b {
		t.Errorf("số dòng có # đổi từ %d thành %d — mất comment", a, b)
	}

	// Hai danh sách phải rẽ đôi đúng ở đây: khách mất, thợ còn.
	if coDichVu(DichVuDangBan(GiaiDoan), "VA_MAT") {
		t.Error("việc đã tắt vẫn còn trong danh sách bán cho khách")
	}
	if !coDichVu(DichVuLenPhieu(GiaiDoan), "VA_MAT") {
		t.Error("việc đã tắt biến mất khỏi bảng tích việc lúc lên phiếu")
	}

	sau := timO(t, "VA_MAT")
	if sau.DangHien || !strings.Contains(sau.ViSao, "tích được") {
		t.Errorf("trang quản trị nói sai về việc đang tắt: %+v", sau)
	}

	// Bật lại: khoá phải biến hẳn khỏi file, không ở lại thành `an: false`.
	sau.An = false
	if _, err := SuaDichVu(map[string]ODichVu{"VA_MAT": sau}); err != nil {
		t.Fatal(err)
	}
	chu, _ := docLaiFile(t)
	// Tìm cả dòng chứ không phải "an:" trơn — "tu_giai_doan: 1" cũng chứa nó.
	if strings.Contains(chu, "\n    an:") {
		t.Error("bật lại mà khoá an vẫn còn trong file")
	}
	if !coDichVu(DichVuDangBan(GiaiDoan), "VA_MAT") {
		t.Error("bật lại rồi mà khách vẫn không thấy")
	}
}

func coDichVu(ds []DichVu, ma string) bool {
	for _, d := range ds {
		if d.Ma == ma {
			return true
		}
	}
	return false
}

// Tắt rồi mà trang giá riêng còn sống thì link cũ trong Google vẫn dẫn khách
// tới bảng giá của thứ trạm đang không nhận. Cả /dich-vu lẫn /app/dich-vu.
func TestTrangGiaViecDaTatRa404(t *testing.T) {
	mux, _ := dungTrangThu(t, VaiTroChu)
	dungDichVuThat(t)

	mo := func(duong string) int {
		r := httptest.NewRequest("GET", duong, nil)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Code
	}
	if c := mo("/dich-vu/VA_MAT"); c != http.StatusOK {
		t.Fatalf("chưa tắt mà trang giá đã trả %d", c)
	}

	o := timO(t, "VA_MAT")
	o.An = true
	if _, err := SuaDichVu(map[string]ODichVu{"VA_MAT": o}); err != nil {
		t.Fatal(err)
	}
	for _, duong := range []string{"/dich-vu/VA_MAT", "/app/dich-vu/VA_MAT"} {
		if c := mo(duong); c != http.StatusNotFound {
			t.Errorf("%s trả %d, muốn 404", duong, c)
		}
	}
	// Việc khác không bị vạ lây.
	if c := mo("/dich-vu/QUAN_GRIP"); c != http.StatusOK {
		t.Errorf("/dich-vu/QUAN_GRIP trả %d", c)
	}
}

// Ô tích ở form phải đi thẳng vào file: thiếu chỗ đọc f("an") thì bấm lưu
// xong trang vẫn báo đang hiện, mà không có lỗi nào để lần ra.
func TestFormTatViecDiDuocTuTrangQuanTri(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	dungDichVuThat(t)

	than := moTrang(t, mux, ck, "/qt/dich-vu")
	if !strings.Contains(than, `name="an.VA_MAT"`) {
		t.Fatal("trang quản trị không có ô tích tắt việc")
	}

	cu := timO(t, "VA_MAT")
	f := url.Values{
		"co.VA_MAT":             {"1"},
		"ten.VA_MAT":            {cu.Ten},
		"gia1.VA_MAT":           {cu.Gia[0]},
		"gia2.VA_MAT":           {cu.Gia[1]},
		"gia3.VA_MAT":           {cu.Gia[2]},
		"tu_giai_doan.VA_MAT":   {"1"},
		"bao_hanh_thang.VA_MAT": {cu.BaoHanhThang},
		"lead_time_ngay.VA_MAT": {cu.LeadTimeNgay},
		"dieu_kien.VA_MAT":      {cu.DieuKien},
		"an.VA_MAT":             {"1"},
	}
	r := httptest.NewRequest("POST", "/qt/dich-vu", strings.NewReader(f.Encode()))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("POST trả %d: %s", w.Code, catBot(w.Body.String(), 300))
	}
	if sau := timO(t, "VA_MAT"); !sau.An || sau.DangHien {
		t.Fatalf("tích ô tắt rồi mà vẫn hiện: %+v", sau)
	}
}

// --- Khoảng giá "từ X đến Y" -----------------------------------------
//
// Khoá gia_den là khoá MỚI, đứng cạnh gia chứ không thay nó: src/quote.py đọc
// s["gia"][stage-1] rồi int() ngay, đổi ô ấy thành chuỗi hay mảng là báo giá
// vỡ. Ba thứ phải đúng cùng lúc: chỗ đọc chữ chặn khoảng vô lý, file ghi ra
// vẫn đọc được, và bỏ trống ô "đến" thì khoá biến mất khỏi file.

func TestDocODichVuLuatKhoangGia(t *testing.T) {
	dungDichVuThat(t)
	cu := timO(t, "VA_MAT")
	nen := map[string]string{"ten": cu.Ten, "gia1": "120000", "tu_giai_doan": "1"}
	doc := func(sua map[string]string) (ODichVu, error) {
		f := map[string]string{}
		for k, v := range nen {
			f[k] = v
		}
		for k, v := range sua {
			f[k] = v
		}
		return docODichVu(cu, func(k string) string { return f[k] })
	}

	moi, err := doc(map[string]string{"giaden1": "180000"})
	if err != nil {
		t.Fatal(err)
	}
	if moi.Gia[0] != "120000" || moi.GiaDen[0] != "180000" {
		t.Fatalf("khoảng giá hợp lệ đọc ra sai: %q – %q", moi.Gia[0], moi.GiaDen[0])
	}

	for ten, sua := range map[string]map[string]string{
		"đầu trên nhỏ hơn đầu dưới": {"giaden1": "90000"},
		"hai đầu bằng nhau":         {"giaden1": "120000"},
		"có đầu trên, không có sàn": {"gia1": "", "giaden1": "180000", "tu_giai_doan": "2", "gia2": "90000"},
	} {
		if _, err := doc(sua); err == nil {
			t.Errorf("%s: lẽ ra phải báo lỗi", ten)
		}
	}
}

func TestSuaDichVuGhiRoiXoaKhoangGia(t *testing.T) {
	goc := dungDichVuThat(t)

	o := timO(t, "VA_MAT")
	o.GiaDen[0] = "180000"
	if _, err := SuaDichVu(map[string]ODichVu{"VA_MAT": o}); err != nil {
		t.Fatal(err)
	}
	chu, g := docLaiFile(t)
	if !strings.Contains(chu, "gia_den: [180000, null, null]") {
		t.Fatalf("không thấy dòng gia_den trong file")
	}
	for _, d := range g.DichVu {
		if d.Ma != "VA_MAT" {
			continue
		}
		den := d.GiaDenTheoGiaiDoan(1)
		if den == nil || *den != 180000 {
			t.Fatalf("gia_den đọc lại không ra: %+v", d.GiaDen)
		}
		if d.Gia[0] == nil || *d.Gia[0] != 120000 {
			t.Fatalf("ghi gia_den mà đụng vào gia: %+v", d.Gia)
		}
	}
	if a, b := strings.Count(goc, "#"), strings.Count(chu, "#"); a != b {
		t.Errorf("số dòng có # đổi từ %d thành %d — mất comment", a, b)
	}

	// Xoá ô "đến": khoá phải biến mất hẳn, không để lại [null, null, null].
	o = timO(t, "VA_MAT")
	if o.GiaDen[0] != "180000" {
		t.Fatalf("nạp lại không thấy khoảng giá vừa ghi: %+v", o.GiaDen)
	}
	o.GiaDen[0] = ""
	if _, err := SuaDichVu(map[string]ODichVu{"VA_MAT": o}); err != nil {
		t.Fatal(err)
	}
	chu, _ = docLaiFile(t)
	if strings.Contains(chu, "gia_den") {
		t.Fatalf("bỏ trống ô đến rồi mà khoá gia_den vẫn còn")
	}
}

// Dòng giá in cho khách: bản dài cho web, bản ngắn cho ô trong app.
func TestGiaKhachInKhoang(t *testing.T) {
	dungDichVuThat(t)
	GiaiDoan = 1
	so := func(n int) *int { return &n }

	co := DichVu{Ma: "X", Gia: []*int{so(120000)}, GiaDen: []*int{so(180000)}}
	if got, muon := giaKhach(co), "từ 120.000 đến 180.000đ"; got != muon {
		t.Errorf("giaKhach = %q, muốn %q", got, muon)
	}
	if got, muon := giaNgan(co), "120k–180k"; got != muon {
		t.Errorf("giaNgan = %q, muốn %q", got, muon)
	}

	// Không có đầu trên thì y như trước khi có khoá mới.
	khong := DichVu{Ma: "Y", Gia: []*int{so(120000)}}
	if got, muon := giaKhach(khong), "từ 120.000đ"; got != muon {
		t.Errorf("giaKhach không khoảng = %q, muốn %q", got, muon)
	}
	// Số lẻ không chia hết cho nghìn thì in đủ, không làm tròn.
	if got := giaNgan(DichVu{Ma: "Z", Gia: []*int{so(120500)}}); !strings.Contains(got, "120.500") {
		t.Errorf("giaNgan làm tròn số lẻ: %q", got)
	}

	// Câu chú thích dưới lưới chỉ hiện khi trong lưới có việc giá không cứng.
	if coGiaTheoTinhTrang([]DichVu{khong}) {
		t.Error("lưới toàn giá cứng mà vẫn treo câu giá tuỳ tình trạng")
	}
	if !coGiaTheoTinhTrang([]DichVu{khong, co}) {
		t.Error("lưới có khoảng giá mà không treo câu giá tuỳ tình trạng")
	}
	if !coGiaTheoTinhTrang([]DichVu{{Ma: "H", BaoGiaRieng: true}}) {
		t.Error("lưới có việc chưa niêm yết giá mà không treo câu")
	}
}
