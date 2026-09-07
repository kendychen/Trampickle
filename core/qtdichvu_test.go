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

	o := timO(t, "PHU_NHAM")
	if o.BaoHanhThang != "" || o.LeadTimeNgay != "" {
		t.Fatalf("PHU_NHAM đáng ra chưa có hai khoá này: %+v", o)
	}
	o.BaoHanhThang = "2"
	o.LeadTimeNgay = "4"
	o.DieuKien = "Trạm chưa có máy đo Rz/Rt. Anh chị có đánh giải thì đừng làm."
	if _, err := SuaDichVu(map[string]ODichVu{"PHU_NHAM": o}); err != nil {
		t.Fatal(err)
	}

	moi, g := docLaiFile(t)
	cuoi := g.DichVu[len(g.DichVu)-1]
	if cuoi.Ma != "PHU_NHAM" || cuoi.BaoHanhThang != 2 || cuoi.LeadTimeNgay != 4 {
		t.Errorf("PHU_NHAM ghi sai: %+v", cuoi)
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
		"tên rỗng":               {"ten": "  "},
		"giá không có chữ số":    {"gia1": "gọi điện"},
		"giá quá to":             {"gia1": "9999999999"},
		"giai đoạn ngoài 1-3":    {"tu_giai_doan": "4"},
		"bảo hành không phải số": {"bao_hanh_thang": "ba"},
		"mở bán mà không có giá": {"gia1": "", "bao_gia_rieng": ""},
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
