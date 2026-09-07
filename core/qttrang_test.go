package core

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

// Template gọi sai tên trường thì Go build vẫn xanh — chỉ tới lúc Kendy bấm
// vào trang mới thấy "Lỗi hiển thị trang". Mấy test dưới đây dựng thật từng
// trang mới qua router, có phiên đăng nhập thật, và bắt lỗi ngay ở CI.
//
// render() nuốt lỗi template thành HTTP 500, nên cứ 200 là dựng được.

func phienThu(t *testing.T, vaiTro string) *http.Cookie {
	t.Helper()
	nguoiDungMu.Lock()
	cuND := nguoiDung
	nguoiDung = []NguoiDung{{Ten: "kendy", HoTen: "Kendy", VaiTro: vaiTro}}
	nguoiDungMu.Unlock()

	ma := maNgauNhien(8)
	phienMu.Lock()
	phienDs[ma] = phien{Ten: "kendy", Het: time.Now().Add(time.Hour), IPTao: "127.0.0.1"}
	phienMu.Unlock()

	t.Cleanup(func() {
		nguoiDungMu.Lock()
		nguoiDung = cuND
		nguoiDungMu.Unlock()
		phienMu.Lock()
		delete(phienDs, ma)
		phienMu.Unlock()
	})
	return &http.Cookie{Name: tenCookie, Value: ma}
}

func moTrang(t *testing.T, mux *http.ServeMux, ck *http.Cookie, duong string) string {
	t.Helper()
	r := httptest.NewRequest("GET", duong, nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("%s trả %d, thân: %s", duong, w.Code, catBot(w.Body.String(), 300))
	}
	return w.Body.String()
}

// dungTrangThu: trạm rỗng + template + phiên chủ. Trả về mux và cookie.
func dungTrangThu(t *testing.T, vaiTro string) (*http.ServeMux, *http.Cookie) {
	t.Helper()
	dungKhoTienThu(t)
	if err := InitTemplates(); err != nil {
		t.Fatal(err)
	}
	cuCongKhai := CongKhai
	t.Cleanup(func() { CongKhai = cuCongKhai })
	return NewMux(true), phienThu(t, vaiTro)
}

// Kho rỗng: trang phải mở được và nói cho người ta biết bắt đầu từ đâu, chứ
// không phải một cái bảng trắng không giải thích gì.
func TestTrangKhoRongDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	s := moTrang(t, mux, ck, "/qt/kho")
	if !strings.Contains(s, "/qt/kho/vat-tu") {
		t.Error("kho rỗng mà không chỉ đường sang khai danh mục vật tư")
	}
	moTrang(t, mux, ck, "/qt/kho/phieu")
	moTrang(t, mux, ck, "/qt/kho/vat-tu")
	for _, l := range []string{PhieuNhap, PhieuXuat, PhieuKiemKe, PhieuHaoHut} {
		moTrang(t, mux, ck, "/qt/kho/phieu/moi?loai="+l)
	}
}

// Kho có số: bảng tồn, phiếu, và trang sửa một phiếu đã lưu.
func TestTrangKhoCoSoDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	vatTuThu(t,
		VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét", TonToiThieu: 20, NhaCungCap: "Chợ Kim Biên"},
		VatTu{Ma: "vt-2", Ten: "Keo epoxy", DonVi: "tuýp", Ngung: true},
	)
	if err := LuuDinhMuc(map[string][]DongDinhMuc{"dan-day": {{MaVatTu: "vt-1", SoLuong: 6}}}); err != nil {
		t.Fatal(err)
	}
	p := phieuThu(t, PhieuNhap, "2026-09-01", DongPhieu{MaVatTu: "vt-1", SoLuong: 100, DonGia: 3000})
	phieuThu(t, PhieuXuat, "2026-09-10", DongPhieu{MaVatTu: "vt-1", SoLuong: 85})

	s := moTrang(t, mux, ck, "/qt/kho")
	if !strings.Contains(s, "Dây 1.2mm") {
		t.Error("bảng tồn không có tên vật tư")
	}
	if !strings.Contains(s, "45.000đ") {
		t.Error("giá trị tồn 15 mét × 3.000 không hiện cho chủ")
	}
	moTrang(t, mux, ck, "/qt/kho/phieu")
	moTrang(t, mux, ck, "/qt/kho/phieu/"+p.Ma)
	moTrang(t, mux, ck, "/qt/kho/vat-tu")
}

// Thợ vào kho được nhưng không thấy tiền, không vào được danh mục, và không
// mở được phiếu nhập. Đây là luật phân quyền, không phải trang trí.
func TestThoKhongThayTienVaKhongVaoDanhMuc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})
	phieuThu(t, PhieuNhap, "2026-09-01", DongPhieu{MaVatTu: "vt-1", SoLuong: 100, DonGia: 3000})

	s := moTrang(t, mux, ck, "/qt/kho")
	if strings.Contains(s, "300.000đ") || strings.Contains(s, "3.000đ") {
		t.Error("trang kho của thợ lộ giá vật tư")
	}
	if strings.Contains(s, `href="/qt/kho/phieu/moi?loai=nhap"`) {
		t.Error("thợ thấy nút tạo phiếu nhập")
	}

	for _, duong := range []string{"/qt/kho/vat-tu", "/qt/tien", "/qt/tien/dinh-ky", "/qt/tien/sao-ke"} {
		r := httptest.NewRequest("GET", duong, nil)
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			t.Errorf("%s mở được với tài khoản thợ", duong)
		}
	}
	// Phiếu nhập bị đá về trang kho kèm lý do, không phải trang trắng.
	r := httptest.NewRequest("GET", "/qt/kho/phieu/moi?loai=nhap", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Errorf("thợ mở phiếu nhập trả %d, muốn 303", w.Code)
	}
}

// Sổ tiền rỗng — nhánh "chưa có số" của trang về vốn.
func TestTrangTienRongDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	moTrang(t, mux, ck, "/qt/tien")
	moTrang(t, mux, ck, "/qt/tien/dinh-ky")
	moTrang(t, mux, ck, "/qt/tien/sao-ke")
}

// Sổ tiền đang lỗ: trang phải nói thẳng là chưa tính được ngày về vốn, và
// tuyệt đối không in ra một con số tháng âm.
func TestTrangTienDangLoKhongHenNgay(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	nay := time.Now().Format("2006-01")
	khoanThu(t, Khoan{Ngay: nay + "-01", Loai: KhoanChi, Nhom: NhomDauTu, SoTien: 30000000, DienGiai: "Máy ép"})
	khoanThu(t, Khoan{Ngay: nay + "-10", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 1000000, DienGiai: "Sửa vợt"})
	khoanThu(t, Khoan{Ngay: nay + "-11", Loai: KhoanChi, Nhom: NhomDinhPhi, SoTien: 3000000, DienGiai: "Thuê nhà"})

	s := moTrang(t, mux, ck, "/qt/tien?thang="+nay)
	if strings.Contains(s, "-2,0 tháng") || strings.Contains(s, "-15,0 tháng") {
		t.Error("trang in ra số tháng âm")
	}
	if !strings.Contains(s, "Máy ép") {
		t.Error("bảng khoản của tháng không có khoản vừa ghi")
	}
	if !strings.Contains(s, "30.000.000đ") {
		t.Error("tổng đầu tư không hiện")
	}
}

// Sổ tiền đã về vốn: trang phải nói rõ về vốn từ tháng nào.
func TestTrangTienDaVeVonNoiRoThang(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	khoanThu(t, Khoan{Ngay: "2026-06-01", Loai: KhoanChi, Nhom: NhomDauTu, SoTien: 5000000, DienGiai: "Đồ nghề"})
	khoanThu(t, Khoan{Ngay: "2026-06-20", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 6000000, DienGiai: "Sửa vợt"})

	s := moTrang(t, mux, ck, "/qt/tien?thang=2026-06")
	if !strings.Contains(s, "Tháng 6/2026") {
		t.Error("không nói về vốn từ tháng nào")
	}
}

// Trang chờ xác nhận: form phải nằm NGOÀI <table> và mỗi ô trỏ về form bằng
// thuộc tính form=. Đặt <form> giữa <tr> và <td> thì trình duyệt nhấc nó ra
// ngoài bảng, nút bấm gửi đi một form rỗng, và không ai duyệt được khoản nào.
func TestTrangTienChoDuyetFormNgoaiBang(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	nay := time.Now().Format("2006-01")
	khoanThu(t, Khoan{Ngay: nay + "-05", Loai: KhoanChi, Nhom: NhomDinhPhi, SoTien: 3000000, DienGiai: "Thuê nhà", ChoDuyet: true})
	ma := KhoanChoDuyet()[0].Ma

	s := moTrang(t, mux, ck, "/qt/tien?thang="+nay)
	if !strings.Contains(s, `id="duyet-`+ma+`"`) {
		t.Fatal("không có form duyệt cho khoản chờ")
	}
	if !strings.Contains(s, `form="duyet-`+ma+`"`) {
		t.Error("ô nhập không trỏ về form duyệt")
	}
	dauForm := strings.Index(s, `id="duyet-`+ma+`"`)
	dauBang := strings.Index(s, "<table")
	if dauBang >= 0 && dauForm > dauBang {
		t.Error("form duyệt nằm trong bảng — trình duyệt sẽ nhấc nó ra và nút bấm gửi form rỗng")
	}

	// Và bấm duyệt thì khoản vào sổ thật.
	r := httptest.NewRequest("POST", "/qt/tien/duyet", strings.NewReader("ma="+ma+"&ngay="+nay+"-05"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("duyệt trả %d", w.Code)
	}
	if SoKhoanChoDuyet() != 0 {
		t.Error("bấm duyệt rồi mà khoản vẫn nằm trong hàng chờ")
	}
}

// Trang định kỳ có số: bảng phải đổ đúng khoản đã khai ra ô nhập.
func TestTrangDinhKyCoSoDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	if err := LuuDinhKy([]KhoanDinhKy{
		{Ten: "Thuê nhà", Nhom: NhomDinhPhi, SoTien: 3000000, NgayTrongThang: 5, PhuongThuc: TraChuyenKhoan, Bat: true},
	}); err != nil {
		t.Fatal(err)
	}
	s := moTrang(t, mux, ck, "/qt/tien/dinh-ky")
	if !strings.Contains(s, `value="Thuê nhà"`) {
		t.Error("ô tên không đổ sẵn khoản đã khai")
	}
	if !strings.Contains(s, `value="3000000"`) {
		t.Error("ô số tiền phải là số trần, không dấu chấm")
	}
}

// Dán sao kê rồi xem bảng đề xuất. Ô tick phải mang chỉ số dòng, và dòng nào
// đã ghi rồi thì khoá tick lại chứ không khoá ô chữ — khoá ô chữ thì trình
// duyệt không gửi nó lên nữa và các dòng dưới lệch chỉ số hết.
func TestTrangSaoKeDeXuatKhop(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	don := &Don{Ma: "TV-2609-001", TongTien: 500000, KhachTen: "Anh Nam", KhachLienHe: "0901234567"}
	donMu.Lock()
	donDs[don.Ma] = don
	donMu.Unlock()

	than := "sao_ke=" + url.QueryEscape("05/09/2026  +500.000  CK TV2609001 chuyen tien sua vot")
	r := httptest.NewRequest("POST", "/qt/tien/sao-ke", strings.NewReader(than))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("dán sao kê trả %d: %s", w.Code, catBot(w.Body.String(), 300))
	}
	s := w.Body.String()
	if !strings.Contains(s, `name="chon" value="0"`) {
		t.Error("ô tick không mang chỉ số dòng — handler đọc chon theo chỉ số")
	}
	if !strings.Contains(s, `name="ma_don" maxlength="20" value="TV-2609-001"`) {
		t.Error("không nhận ra mã đơn nằm trong nội dung chuyển khoản")
	}
	if !strings.Contains(s, "Anh Nam") {
		t.Error("khớp được đơn thì phải hiện tên khách để người ta soát lại")
	}
	if strings.Contains(s, `name="ma_don" disabled`) {
		t.Error("khoá ô chữ sẽ làm mọi dòng dưới lệch chỉ số")
	}
}

// Trang chi tiết đơn có hai thẻ mới: thu tiền và vật tư đã dùng. Đây là chỗ
// dễ vỡ nhất vì handler phải bơm thêm 5 trường vào struct render.
func TestTrangChiTietDonCoTheThuTienVaVatTu(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})
	if err := LuuDinhMuc(map[string][]DongDinhMuc{"dan-day": {{MaVatTu: "vt-1", SoLuong: 6}}}); err != nil {
		t.Fatal(err)
	}
	don := &Don{
		Ma: "TV-2609-002", KhachTen: "Anh Nam", KhachLienHe: "0901234567",
		TrangThai: "dang_sua", TongTien: 500000,
		DongTien: []DongTien{{MaDichVu: "dan-day", Ten: "Đan dây", SoTien: 500000}},
	}
	donMu.Lock()
	donDs[don.Ma] = don
	donMu.Unlock()
	khoanThu(t, Khoan{Ngay: "2026-09-01", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 200000, MaDon: don.Ma, DienGiai: "Cọc"})

	s := moTrang(t, mux, ck, "/qt/don/"+don.Ma)
	if !strings.Contains(s, "/qt/don/"+don.Ma+"/thu") {
		t.Error("không có form ghi tiền khách trả")
	}
	if strings.Contains(s, `name="da_thu"`) {
		t.Error("ô đã thu cũ vẫn còn — tiền phải ghi thành từng lần thu")
	}
	if !strings.Contains(s, "Dây 1.2mm") {
		t.Error("không đề xuất vật tư theo định mức")
	}
	if !strings.Contains(s, "300.000đ") {
		t.Error("còn nợ 300.000 không hiện")
	}

	// Ghi thêm một lần thu qua đúng form đó.
	r := httptest.NewRequest("POST", "/qt/don/"+don.Ma+"/thu",
		strings.NewReader("so_tien=300000&phuong_thuc=tien_mat&ghi_chu=Tr%E1%BA%A3%20n%E1%BB%91t"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("ghi thu trả %d: %s", w.Code, catBot(w.Body.String(), 300))
	}
	if got := don.DaThu(); got != 500000 {
		t.Fatalf("đã thu = %d, muốn 500.000", got)
	}
	if don.ConNo() != 0 {
		t.Errorf("còn nợ = %d", don.ConNo())
	}
}

// Nút "khớp định mức" một cú bấm: dựng phiếu xuất từ định mức và trừ kho.
// Trước khi bấm thì kho phải chưa bị đụng vào.
func TestNutKhopDinhMucTruKhoMotLan(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})
	if err := LuuDinhMuc(map[string][]DongDinhMuc{"dan-day": {{MaVatTu: "vt-1", SoLuong: 6}}}); err != nil {
		t.Fatal(err)
	}
	phieuThu(t, PhieuNhap, "2026-09-01", DongPhieu{MaVatTu: "vt-1", SoLuong: 100, DonGia: 3000})
	don := &Don{
		Ma: "TV-2609-003", KhachTen: "Anh Nam", KhachLienHe: "0901234567", TrangThai: "dang_sua",
		DongTien: []DongTien{{MaDichVu: "dan-day", Ten: "Đan dây", SoTien: 500000}},
	}
	donMu.Lock()
	donDs[don.Ma] = don
	donMu.Unlock()

	if TonKho()["vt-1"] != 100 {
		t.Fatal("kho bị trừ trước khi ai bấm gì")
	}
	r := httptest.NewRequest("POST", "/qt/don/"+don.Ma+"/xuat-kho", strings.NewReader("nhanh=1"))
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("khớp định mức trả %d: %s", w.Code, catBot(w.Body.String(), 300))
	}
	if got := TonKho()["vt-1"]; got != 94 {
		t.Fatalf("tồn = %v, muốn 94", got)
	}
	p, co := PhieuXuatCuaDon(don.Ma)
	if !co {
		t.Fatal("không có phiếu xuất gắn với đơn")
	}
	if p.MaDon != don.Ma {
		t.Errorf("phiếu xuất trỏ về đơn %q", p.MaDon)
	}

	// Bấm lần hai không được trừ thêm lần nữa.
	r2 := httptest.NewRequest("POST", "/qt/don/"+don.Ma+"/xuat-kho", strings.NewReader("nhanh=1"))
	r2.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r2.AddCookie(ck)
	w2 := httptest.NewRecorder()
	mux.ServeHTTP(w2, r2)
	if got := TonKho()["vt-1"]; got != 94 {
		t.Fatalf("bấm hai lần trừ kho hai lần: tồn = %v", got)
	}
}

// Giảm giá: một dòng tiền âm chứ không phải trường riêng, và không bao giờ
// kéo tổng đơn xuống dưới 0 dù gõ thừa mấy số 0.
func TestGiamGiaTrenDon(t *testing.T) {
	don := &Don{Ma: "TV-2609-003"}
	dat := func(giam, ly string, tay ...string) {
		f := url.Values{}
		f.Set("khach_ten", "Khách thử")
		for i := 0; i+1 < len(tay); i += 2 {
			f.Add("dong_ten", tay[i])
			f.Add("dong_tien", tay[i+1])
		}
		f.Set("giam_gia", giam)
		f.Set("giam_ly_do", ly)
		r := httptest.NewRequest("POST", "/", strings.NewReader(f.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.ParseForm()
		apDungForm(don, r, NguoiDung{Ten: "tho"})
	}

	dat("50000", "khách quen", "Vá mặt vợt", "300000")
	if don.TongTien != 250000 {
		t.Fatalf("tổng = %d, muốn 250.000", don.TongTien)
	}
	if don.GiamGia() != 50000 {
		t.Fatalf("giảm giá = %d, muốn 50.000", don.GiamGia())
	}
	if don.LyDoGiam() != "khách quen" {
		t.Fatalf("lý do giảm = %q", don.LyDoGiam())
	}

	// Gõ thừa số 0: kẹp bằng tổng, không cho ra đơn âm.
	dat("3000000", "gõ nhầm", "Vá mặt vợt", "300000")
	if don.TongTien != 0 {
		t.Fatalf("giảm quá tổng ra %d, muốn kẹp về 0", don.TongTien)
	}

	// Xoá ô giảm giá là bỏ hẳn dòng ấy.
	dat("", "", "Vá mặt vợt", "300000")
	if don.TongTien != 300000 || don.GiamGia() != 0 {
		t.Fatalf("bỏ giảm giá không sạch: tổng %d, giảm %d", don.TongTien, don.GiamGia())
	}
	for _, dt := range don.DongTien {
		if dt.MaDichVu == MaGiamGia {
			t.Fatal("dòng giảm giá vẫn còn sau khi xoá ô")
		}
	}
}
