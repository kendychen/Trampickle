package core

import (
	"strings"
	"testing"
	"time"
)

func khoanThu(t *testing.T, k Khoan) {
	t.Helper()
	if err := LuuKhoan(k); err != nil {
		t.Fatal(err)
	}
}

// Đầu tư và chi vận hành phải nằm hai cột khác nhau. Trộn lại thì tháng nào
// mua máy cũng thành tháng lỗ, và câu hỏi "bao giờ về vốn" không còn mẫu số.
func TestDauTuKhongTinhVaoLaiThang(t *testing.T) {
	dungKhoTienThu(t)
	khoanThu(t, Khoan{Ngay: "2026-09-05", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 5000000, DienGiai: "Sửa vợt"})
	khoanThu(t, Khoan{Ngay: "2026-09-05", Loai: KhoanChi, Nhom: NhomDinhPhi, SoTien: 2000000, DienGiai: "Thuê nhà"})
	khoanThu(t, Khoan{Ngay: "2026-09-06", Loai: KhoanChi, Nhom: NhomDauTu, SoTien: 30000000, DienGiai: "Máy ép"})

	bc := BaoCaoThang()
	if len(bc) != 1 {
		t.Fatalf("có %d tháng, muốn 1", len(bc))
	}
	th := bc[0]
	if th.Thu != 5000000 {
		t.Errorf("thu = %d", th.Thu)
	}
	if th.ChiVanHanh != 2000000 {
		t.Errorf("chi vận hành = %d — tiền mua máy không được nằm ở đây", th.ChiVanHanh)
	}
	if th.DauTu != 30000000 {
		t.Errorf("đầu tư = %d", th.DauTu)
	}
	if th.Lai != 3000000 {
		t.Errorf("lãi = %d, muốn 3.000.000 (thu trừ chi vận hành)", th.Lai)
	}
	if th.DongTienRong != 3000000-30000000 {
		t.Errorf("dòng tiền ròng = %d — tháng mua máy thì túi vẫn hụt tiền", th.DongTienRong)
	}
}

// Luỹ kế phải cộng theo chiều thời gian dù bảng hiện mới nhất trước. Cộng
// ngược chiều thì tháng đầu tiên mang luỹ kế của cả năm và mọi con số về vốn
// lệch theo.
func TestLuyKeCongTheoChieuThoiGian(t *testing.T) {
	dungKhoTienThu(t)
	khoanThu(t, Khoan{Ngay: "2026-07-10", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 1000000})
	khoanThu(t, Khoan{Ngay: "2026-08-10", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 2000000})
	khoanThu(t, Khoan{Ngay: "2026-09-10", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 3000000})

	bc := BaoCaoThang()
	if len(bc) != 3 {
		t.Fatalf("có %d tháng, muốn 3", len(bc))
	}
	if bc[0].Thang != "2026-09" {
		t.Fatalf("dòng đầu là %s, bảng phải mới nhất trước", bc[0].Thang)
	}
	if bc[0].LuyKeLai != 6000000 {
		t.Errorf("luỹ kế tháng mới nhất = %d, muốn 6.000.000", bc[0].LuyKeLai)
	}
	if bc[2].LuyKeLai != 1000000 {
		t.Errorf("luỹ kế tháng cũ nhất = %d, muốn 1.000.000", bc[2].LuyKeLai)
	}
}

// Khoản chờ duyệt là khoản MÁY đoán, chưa ai xác nhận đã trả. Nó vào báo cáo
// nghĩa là sổ nói tháng này đã trả tiền nhà trong khi thật ra chưa — và con
// số về vốn nói dối theo mà không ai biết.
func TestKhoanChoDuyetChuaVaoBaoCao(t *testing.T) {
	dungKhoTienThu(t)
	khoanThu(t, Khoan{Ngay: "2026-09-01", Loai: KhoanChi, Nhom: NhomDinhPhi, SoTien: 2000000, DienGiai: "Thuê nhà", ChoDuyet: true})
	khoanThu(t, Khoan{Ngay: "2026-09-02", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 500000})

	bc := BaoCaoThang()
	if bc[0].ChiVanHanh != 0 {
		t.Errorf("chi vận hành = %d — khoản chờ duyệt chưa được tính", bc[0].ChiVanHanh)
	}
	if bc[0].SoKhoan != 1 {
		t.Errorf("đếm %d khoản, muốn 1", bc[0].SoKhoan)
	}
	if SoKhoanChoDuyet() != 1 {
		t.Errorf("số khoản chờ duyệt = %d, muốn 1", SoKhoanChoDuyet())
	}

	// Duyệt xong thì nó vào sổ ngay, không cần nạp lại.
	ma := KhoanChoDuyet()[0].Ma
	if err := DuyetKhoan(ma); err != nil {
		t.Fatal(err)
	}
	if got := BaoCaoThang()[0].ChiVanHanh; got != 2000000 {
		t.Errorf("sau khi duyệt, chi vận hành = %d", got)
	}
	if SoKhoanChoDuyet() != 0 {
		t.Error("duyệt rồi mà vẫn còn trong hàng chờ")
	}
}

// Nhánh 1 của bài toán về vốn: chưa khai đồng đầu tư nào thì không có gì để
// bù, và không được bịa ra một cái hẹn.
func TestVeVonChuaKhaiDauTu(t *testing.T) {
	dungKhoTienThu(t)
	khoanThu(t, Khoan{Ngay: "2026-09-05", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 3000000})

	vv := TinhVeVon()
	if vv.TongDauTu != 0 {
		t.Errorf("tổng đầu tư = %d", vv.TongDauTu)
	}
	if vv.DaVeVon {
		t.Error("chưa bỏ vốn thì không thể 'đã về vốn'")
	}
	if vv.ConMayThang != 0 || vv.NgayDuKien != "" {
		t.Errorf("không có vốn mà vẫn hẹn ngày: %v tháng, %q", vv.ConMayThang, vv.NgayDuKien)
	}
}

// Nhánh 2: luỹ kế lãi đã bù hết vốn — phải chỉ đúng THÁNG bù xong, không phải
// tháng hiện tại.
func TestVeVonDaBuXongChiDungThang(t *testing.T) {
	dungKhoTienThu(t)
	khoanThu(t, Khoan{Ngay: "2026-06-01", Loai: KhoanChi, Nhom: NhomDauTu, SoTien: 10000000, DienGiai: "Máy ép"})
	khoanThu(t, Khoan{Ngay: "2026-06-20", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 4000000})
	khoanThu(t, Khoan{Ngay: "2026-07-20", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 4000000})
	// Tháng 8 luỹ kế chạm 12tr > 10tr: về vốn ở đây.
	khoanThu(t, Khoan{Ngay: "2026-08-20", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 4000000})
	khoanThu(t, Khoan{Ngay: "2026-09-20", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 4000000})

	vv := TinhVeVon()
	if !vv.DaVeVon {
		t.Fatalf("luỹ kế lãi %d ≥ vốn %d mà chưa báo về vốn", vv.LuyKeLai, vv.TongDauTu)
	}
	if vv.ThangVeVon != "2026-08" {
		t.Errorf("về vốn từ %q, muốn 2026-08", vv.ThangVeVon)
	}
	if vv.ConThieu != 0 {
		t.Errorf("còn thiếu = %d, đã về vốn thì phải là 0", vv.ConThieu)
	}
}

// Nhánh 3 — nhánh hay bị quên nhất. Đang lỗ thì phép chia ra số âm, in ra
// trông như một cái hẹn ("còn -4 tháng"). Phải từ chối chia và nói thẳng.
func TestVeVonDangLoThiKhongChia(t *testing.T) {
	dungKhoTienThu(t)
	khoanThu(t, Khoan{Ngay: "2026-08-01", Loai: KhoanChi, Nhom: NhomDauTu, SoTien: 30000000, DienGiai: "Máy ép"})
	khoanThu(t, Khoan{Ngay: "2026-08-10", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 1000000})
	khoanThu(t, Khoan{Ngay: "2026-08-11", Loai: KhoanChi, Nhom: NhomDinhPhi, SoTien: 3000000, DienGiai: "Thuê nhà"})
	khoanThu(t, Khoan{Ngay: "2026-09-10", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 1000000})
	khoanThu(t, Khoan{Ngay: "2026-09-11", Loai: KhoanChi, Nhom: NhomDinhPhi, SoTien: 3000000, DienGiai: "Thuê nhà"})

	vv := TinhVeVon()
	if !vv.DangLo {
		t.Fatalf("lãi trung bình %d mà không bật cờ đang lỗ", vv.LaiTBThang)
	}
	if vv.ConMayThang != 0 {
		t.Errorf("còn %v tháng — đang lỗ thì không được đưa ra con số nào", vv.ConMayThang)
	}
	if vv.NgayDuKien != "" {
		t.Errorf("hẹn ngày về vốn %q trong khi đang lỗ", vv.NgayDuKien)
	}
	if vv.ConThieu != 30000000-(-4000000) {
		t.Errorf("còn thiếu = %d, muốn 34.000.000 (vốn cộng phần lỗ)", vv.ConThieu)
	}
}

// Tiền khách trả là các LẦN thu ghi trong sổ, không phải một ô số gõ đè lên
// nhau. Xoá một lần thu thì công nợ phải quay lại đúng bằng chỗ đã xoá.
func TestDaThuConNoTinhTuSoTien(t *testing.T) {
	dungKhoTienThu(t)
	don := &Don{Ma: "TV-2609-001", TongTien: 500000}
	khoanThu(t, Khoan{Ngay: "2026-09-01", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 200000, MaDon: don.Ma, DienGiai: "Cọc"})
	khoanThu(t, Khoan{Ngay: "2026-09-03", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 150000, MaDon: don.Ma, DienGiai: "Trả thêm"})

	if got := don.DaThu(); got != 350000 {
		t.Fatalf("đã thu = %d, muốn 350.000", got)
	}
	if got := don.ConNo(); got != 150000 {
		t.Fatalf("còn nợ = %d, muốn 150.000", got)
	}
	if n := len(CacLanThuCuaDon(don.Ma)); n != 2 {
		t.Fatalf("có %d lần thu, muốn 2", n)
	}

	// Ghi nhầm rồi xoá: công nợ phải quay lại, không được kẹt ở số cũ.
	if err := XoaKhoan(CacLanThuCuaDon(don.Ma)[1].Ma); err != nil {
		t.Fatal(err)
	}
	if got := don.DaThu(); got != 200000 {
		t.Fatalf("sau khi xoá, đã thu = %d, muốn 200.000", got)
	}
}

// Khoản thu chờ duyệt (dán từ sao kê, chưa xác nhận) chưa được trừ vào công
// nợ khách — nếu không thì đơn hiện "đã trả đủ" trong khi tiền chưa chắc về.
func TestThuChoDuyetChuaTruCongNo(t *testing.T) {
	dungKhoTienThu(t)
	don := &Don{Ma: "TV-2609-002", TongTien: 500000}
	khoanThu(t, Khoan{Ngay: "2026-09-01", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 500000, MaDon: don.Ma, ChoDuyet: true})

	if got := don.DaThu(); got != 0 {
		t.Fatalf("đã thu = %d — khoản chờ duyệt chưa phải tiền đã về", got)
	}
	if got := don.ConNo(); got != 500000 {
		t.Fatalf("còn nợ = %d, muốn 500.000", got)
	}
}

// Khoản định kỳ dựng lại bao nhiêu lần cũng chỉ ra một khoản cho mỗi tháng.
// Việc nền chạy 6 tiếng một lần; đẻ thêm bản sao mỗi lần chạy thì sang tháng
// sau sổ có bốn lần tiền nhà.
func TestSinhKhoanDinhKyKhongDeSinhDoi(t *testing.T) {
	dungKhoTienThu(t)
	if err := LuuDinhKy([]KhoanDinhKy{
		{Ten: "Thuê nhà", Nhom: NhomDinhPhi, SoTien: 3000000, NgayTrongThang: 5, Bat: true},
		{Ten: "Internet", Nhom: NhomDinhPhi, SoTien: 300000, NgayTrongThang: 10, Bat: false},
	}); err != nil {
		t.Fatal(err)
	}

	n, err := SinhKhoanDinhKy("2026-09")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("dựng %d khoản, muốn 1 — khoản đang nghỉ không được dựng", n)
	}
	if n, err = SinhKhoanDinhKy("2026-09"); err != nil || n != 0 {
		t.Fatalf("chạy lần hai dựng thêm %d khoản (lỗi %v), phải là 0", n, err)
	}

	ds := KhoanTrongThang("2026-09")
	if len(ds) != 1 {
		t.Fatalf("tháng 9 có %d khoản, muốn 1", len(ds))
	}
	if !ds[0].ChoDuyet {
		t.Error("khoản máy dựng phải ở trạng thái chờ duyệt")
	}
	if ds[0].Ngay != "2026-09-05" {
		t.Errorf("ngày = %q, muốn 2026-09-05", ds[0].Ngay)
	}

	// Người đã duyệt và sửa số rồi thì lần chạy sau vẫn không được dựng lại.
	if err := DuyetKhoan(ds[0].Ma); err != nil {
		t.Fatal(err)
	}
	if n, err = SinhKhoanDinhKy("2026-09"); err != nil || n != 0 {
		t.Fatalf("dựng lại %d khoản sau khi đã duyệt (lỗi %v)", n, err)
	}
}

// Ngày trong tháng bị kẹp về 1..28. Khai 31 mà không kẹp thì tháng hai khoản
// đó không bao giờ tới hạn, và tiền nhà tháng hai biến mất khỏi sổ.
func TestDinhKyKepNgayVeTrongThang(t *testing.T) {
	dungKhoTienThu(t)
	if err := LuuDinhKy([]KhoanDinhKy{
		{Ten: "Thuê nhà", Nhom: NhomDinhPhi, SoTien: 3000000, NgayTrongThang: 31, Bat: true},
		{Ten: "VPS", Nhom: NhomDinhPhi, SoTien: 200000, NgayTrongThang: 0, Bat: true},
	}); err != nil {
		t.Fatal(err)
	}
	for _, k := range DanhSachDinhKy() {
		if k.NgayTrongThang < 1 || k.NgayTrongThang > 28 {
			t.Errorf("%s: ngày %d ngoài khoảng 1..28", k.Ten, k.NgayTrongThang)
		}
	}
	if _, err := SinhKhoanDinhKy("2026-02"); err != nil {
		t.Fatal(err)
	}
	if n := len(KhoanTrongThang("2026-02")); n != 2 {
		t.Fatalf("tháng 2 dựng được %d khoản, muốn 2", n)
	}
}

// Khoản định kỳ khai sai loại nhóm (chọn nhầm nhóm thu) phải bị nắn về chi
// khác, chứ không được lọt xuống LuuKhoan rồi hỏng cả lượt sinh.
func TestDinhKyNhomSaiLoaiBiNan(t *testing.T) {
	dungKhoTienThu(t)
	if err := LuuDinhKy([]KhoanDinhKy{
		{Ten: "Thuê nhà", Nhom: NhomSuaChua, SoTien: 3000000, NgayTrongThang: 5, Bat: true},
	}); err != nil {
		t.Fatal(err)
	}
	ds := DanhSachDinhKy()
	if len(ds) != 1 {
		t.Fatalf("có %d khoản định kỳ", len(ds))
	}
	if n, _ := TimNhom(ds[0].Nhom); n.Loai != KhoanChi {
		t.Fatalf("nhóm %q vẫn là nhóm thu", ds[0].Nhom)
	}
	if n, err := SinhKhoanDinhKy("2026-09"); err != nil || n != 1 {
		t.Fatalf("dựng %d khoản, lỗi %v", n, err)
	}
}

// Phiếu nhập kho sinh một khoản chi. Sửa lại phiếu thì SỬA khoản đó chứ
// không đẻ thêm — mỗi lần sửa lại thêm một khoản thì sổ phồng gấp đôi.
func TestPhieuNhapSinhMotKhoanChiDuySua(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})

	p := &Phieu{Ma: "NK-2609-001", Loai: PhieuNhap, Ngay: "2026-09-02", Nguoi: "kendy",
		Dong: []DongPhieu{{MaVatTu: "vt-1", SoLuong: 100, DonGia: 3000}}}
	if err := LuuPhieu(p); err != nil {
		t.Fatal(err)
	}
	if err := GhiChiTuPhieu(p, true); err != nil {
		t.Fatal(err)
	}
	ds := KhoanTrongThang("2026-09")
	if len(ds) != 1 || ds[0].SoTien != 300000 {
		t.Fatalf("khoản chi = %+v, muốn một khoản 300.000", ds)
	}
	maCu := ds[0].Ma

	// Ghi nhầm số lượng, sửa lại phiếu rồi ghi chi lần nữa.
	p.Dong[0].SoLuong = 50
	if err := LuuPhieu(p); err != nil {
		t.Fatal(err)
	}
	if err := GhiChiTuPhieu(p, true); err != nil {
		t.Fatal(err)
	}
	ds = KhoanTrongThang("2026-09")
	if len(ds) != 1 {
		t.Fatalf("sau khi sửa phiếu có %d khoản chi, muốn vẫn 1", len(ds))
	}
	if ds[0].Ma != maCu {
		t.Errorf("mã khoản đổi từ %s thành %s — phải sửa tại chỗ", maCu, ds[0].Ma)
	}
	if ds[0].SoTien != 150000 {
		t.Errorf("số tiền = %d, muốn 150.000", ds[0].SoTien)
	}

	// Xoá phiếu thì khoản chi không được ở lại mồ côi.
	if err := XoaChiTuPhieu(p.Ma); err != nil {
		t.Fatal(err)
	}
	if n := len(KhoanTrongThang("2026-09")); n != 0 {
		t.Errorf("còn %d khoản chi trỏ tới phiếu đã xoá", n)
	}
}

// Phiếu nhập chưa trả tiền nhà cung cấp thì khoản chi phải ở trạng thái chờ
// duyệt: hàng về trước, tiền chuyển sau là chuyện thường.
func TestPhieuNhapChuaTraThiChoDuyet(t *testing.T) {
	dungKhoTienThu(t)
	p := &Phieu{Ma: "NK-2609-002", Loai: PhieuNhap, Ngay: "2026-09-02", Nguoi: "kendy",
		Dong: []DongPhieu{{MaVatTu: "vt-1", SoLuong: 10, DonGia: 3000}}}
	if err := LuuPhieu(p); err != nil {
		t.Fatal(err)
	}
	if err := GhiChiTuPhieu(p, false); err != nil {
		t.Fatal(err)
	}
	ds := KhoanTrongThang("2026-09")
	if len(ds) != 1 || !ds[0].ChoDuyet {
		t.Fatalf("khoản = %+v, muốn một khoản chờ duyệt", ds)
	}
	if BaoCaoThang()[0].ChiVanHanh != 0 {
		t.Error("chưa trả tiền mà đã trừ vào lãi")
	}
}

// Định phí thật đứng CẠNH con số khai trong bảng giá, không sửa nó. File bảng
// giá là đầu vào định giá của quote.py — Go sửa nó là đổi giá bán sau lưng.
func TestDinhPhiThucTeChiLaSoDeSoSanh(t *testing.T) {
	dungKhoTienThu(t)
	khoanThu(t, Khoan{Ngay: "2026-08-05", Loai: KhoanChi, Nhom: NhomDinhPhi, SoTien: 3000000, DienGiai: "Thuê nhà"})
	khoanThu(t, Khoan{Ngay: "2026-09-05", Loai: KhoanChi, Nhom: NhomDinhPhi, SoTien: 5000000, DienGiai: "Thuê nhà"})

	tb, soThang := DinhPhiThucTe(3)
	if soThang != 2 {
		t.Fatalf("tính trên %d tháng, muốn 2", soThang)
	}
	if tb != 4000000 {
		t.Errorf("định phí trung bình = %d, muốn 4.000.000", tb)
	}
}

// Tồn kho và công nợ khách hiện ở ô riêng, KHÔNG được cộng vào lãi. Kế toán
// tiền mặt: lãi là tiền đã về trừ tiền đã ra.
func TestTonKhoVaCongNoKhongVaoLai(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})
	p := &Phieu{Ma: "NK-2609-003", Loai: PhieuNhap, Ngay: "2026-09-01", Nguoi: "kendy",
		Dong: []DongPhieu{{MaVatTu: "vt-1", SoLuong: 100, DonGia: 3000}}}
	if err := LuuPhieu(p); err != nil {
		t.Fatal(err)
	}
	if err := GhiChiTuPhieu(p, true); err != nil {
		t.Fatal(err)
	}
	khoanThu(t, Khoan{Ngay: "2026-09-05", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 1000000})

	tq := LayTongQuanTien()
	if tq.GiaTriTon != 300000 {
		t.Errorf("giá trị tồn = %d, muốn 300.000", tq.GiaTriTon)
	}
	nay := time.Now().Format("2006-01")
	if nay != "2026-09" {
		t.Skipf("test viết cho tháng 2026-09, hôm nay là %s", nay)
	}
	// Lãi = 1.000.000 thu - 300.000 chi vật tư. Tồn 300.000 KHÔNG được cộng lại.
	if tq.ThangNay.Lai != 700000 {
		t.Errorf("lãi tháng này = %d, muốn 700.000 — giá trị tồn không được cộng vào lãi", tq.ThangNay.Lai)
	}
}

// Việc nền chạy 6 tiếng một lượt và chạy lại mỗi lần khởi động máy chủ. Nó
// phải là việc làm lại được: dựng khoản định kỳ đúng một lần cho mỗi tháng,
// dù có gọi bao nhiêu lượt.
func TestViecNenChayLaiKhongDeThemGi(t *testing.T) {
	dungKhoTienThu(t)
	if err := LuuDinhKy([]KhoanDinhKy{
		{Ten: "Thuê nhà", Nhom: NhomDinhPhi, SoTien: 3000000, NgayTrongThang: 5, Bat: true},
	}); err != nil {
		t.Fatal(err)
	}
	bayGio := time.Date(2026, 9, 5, 8, 0, 0, 0, time.Local)

	lam := ChayViecNenMotLuot(bayGio)
	if len(lam) == 0 {
		t.Fatal("lượt đầu không làm gì cả")
	}
	if n := len(KhoanTrongThang("2026-09")); n != 1 {
		t.Fatalf("tháng 9 có %d khoản, muốn 1", n)
	}

	// Khởi động lại máy chủ trong cùng tháng: không được thêm gì.
	for i := 0; i < 3; i++ {
		if lam := ChayViecNenMotLuot(bayGio); len(lam) != 0 {
			t.Fatalf("lượt lặp lại còn làm: %v", lam)
		}
	}
	if n := len(KhoanTrongThang("2026-09")); n != 1 {
		t.Fatalf("chạy lại 3 lượt thành %d khoản", n)
	}

	// Sang tháng mới thì dựng khoản của tháng mới, không đụng tháng cũ.
	ChayViecNenMotLuot(bayGio.AddDate(0, 1, 0))
	if n := len(KhoanTrongThang("2026-10")); n != 1 {
		t.Errorf("tháng 10 có %d khoản, muốn 1", n)
	}
	if n := len(KhoanTrongThang("2026-09")); n != 1 {
		t.Errorf("tháng 9 bị đụng vào: %d khoản", n)
	}
}

// Câu trả lời về vốn viết ra chữ. Nhánh đang lỗ tuyệt đối không được chứa
// một con số tháng — người đọc mail chỉ liếc một dòng này.
func TestCauVeVonDangLoKhongHuaThang(t *testing.T) {
	s := cauVeVon(KetQuaVeVon{CoSo: true, TongDauTu: 30000000, ConThieu: 34000000, DangLo: true, SoThangTinh: 2})
	if !strings.Contains(s, "chưa có ngày về vốn") {
		t.Errorf("câu đang lỗ = %q", s)
	}
	if strings.Contains(s, "tháng nữa") {
		t.Errorf("câu đang lỗ vẫn hứa số tháng: %q", s)
	}

	if s := cauVeVon(KetQuaVeVon{}); !strings.Contains(s, "Chưa khai khoản đầu tư") {
		t.Errorf("chưa có số mà nói: %q", s)
	}
}

// Hoàn tiền là khoản CHI gắn mã đơn, và phải trừ ra khỏi số đã thu. Không trừ
// thì đơn huỷ vẫn hiện "đã thu đủ" trong khi tiền đã nằm lại túi khách.
func TestHoanTienTruVaoDaThu(t *testing.T) {
	dungKhoTienThu(t)
	don := &Don{Ma: "TV-2609-002", TongTien: 250000}
	khoanThu(t, Khoan{Ngay: "2026-09-01", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 250000, MaDon: don.Ma})
	if got := don.DaThu(); got != 250000 {
		t.Fatalf("đã thu = %d, muốn 250.000", got)
	}

	// Khách huỷ sau khi nhận, trạm trả lại 100k và giữ 150k tiền công.
	khoanThu(t, Khoan{Ngay: "2026-09-05", Loai: KhoanChi, Nhom: NhomHoanKhach, SoTien: 100000, MaDon: don.Ma})
	if got := don.DaThu(); got != 150000 {
		t.Fatalf("sau khi hoàn 100k, đã thu = %d, muốn 150.000", got)
	}
	if got := don.ConNo(); got != 100000 {
		t.Fatalf("còn lại = %d, muốn 100.000", got)
	}
	if n := len(CacLanHoanCuaDon(don.Ma)); n != 1 {
		t.Fatalf("có %d lần hoàn, muốn 1", n)
	}
	if got := DaHoanCuaDon(don.Ma); got != 100000 {
		t.Fatalf("tổng đã hoàn = %d, muốn 100.000", got)
	}

	// Khoản chi thường gắn mã đơn (ship chẳng hạn) KHÔNG được trừ vào đã thu.
	khoanThu(t, Khoan{Ngay: "2026-09-06", Loai: KhoanChi, Nhom: NhomVanHanh, SoTien: 30000, MaDon: don.Ma})
	if got := don.DaThu(); got != 150000 {
		t.Fatalf("chi vận hành làm lệch số đã thu: %d", got)
	}
}
