package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// dungKhoTienThu dựng một trạm rỗng trong thư mục tạm: kho, sổ tiền, đơn,
// khoản định kỳ đều sạch. Mọi trạng thái ở đây là biến toàn cục nên phải trả
// lại nguyên trạng, nếu không test chạy sau nhặt phải phiếu của test chạy
// trước và hỏng theo thứ tự chạy — kiểu hỏng khó lần nhất.
func dungKhoTienThu(t *testing.T) {
	t.Helper()
	cuRoot := Root
	khoMu.Lock()
	cuVT, cuDM, cuP := khoVatTu, khoDinhMuc, khoPhieu
	khoMu.Unlock()
	tienMu.Lock()
	cuTien, cuDon := tienTheoThang, tienTheoDon
	tienMu.Unlock()
	dinhKyMu.Lock()
	cuDK := dinhKyDs
	dinhKyMu.Unlock()
	donMu.Lock()
	cuDonDs := donDs
	donMu.Unlock()

	t.Cleanup(func() {
		Root = cuRoot
		khoMu.Lock()
		khoVatTu, khoDinhMuc, khoPhieu = cuVT, cuDM, cuP
		khoMu.Unlock()
		tienMu.Lock()
		tienTheoThang, tienTheoDon = cuTien, cuDon
		tienMu.Unlock()
		dinhKyMu.Lock()
		dinhKyDs = cuDK
		dinhKyMu.Unlock()
		donMu.Lock()
		donDs = cuDonDs
		donMu.Unlock()
	})

	Root = t.TempDir()
	for _, d := range []string{"data/kho/phieu", "data/so-tien", "data/don-hang"} {
		if err := os.MkdirAll(filepath.Join(Root, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	donMu.Lock()
	donDs = map[string]*Don{}
	donMu.Unlock()
	if err := NapKho(); err != nil {
		t.Fatal(err)
	}
	if err := NapSoTien(); err != nil {
		t.Fatal(err)
	}
	if err := NapDinhKy(); err != nil {
		t.Fatal(err)
	}
}

func vatTuThu(t *testing.T, ds ...VatTu) {
	t.Helper()
	if err := LuuDanhMucVatTu(ds); err != nil {
		t.Fatal(err)
	}
}

func phieuThu(t *testing.T, loai, ngay string, dong ...DongPhieu) *Phieu {
	t.Helper()
	p := &Phieu{Loai: loai, Ngay: ngay, Nguoi: "kendy", Dong: dong}
	if err := LuuPhieu(p); err != nil {
		t.Fatal(err)
	}
	return p
}

// Tồn kho không được lưu ở đâu cả — nó là kết quả phát lại phiếu. Đây là lời
// hứa gốc của cả module: sửa một phiếu cũ thì tồn hôm nay phải đổi theo, chứ
// không phải giữ một con số đã cộng sai từ tháng trước.
func TestTonKhoPhatLaiTuPhieu(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})

	phieuThu(t, PhieuNhap, "2026-09-01", DongPhieu{MaVatTu: "vt-1", SoLuong: 100, DonGia: 3000})
	phieuThu(t, PhieuXuat, "2026-09-03", DongPhieu{MaVatTu: "vt-1", SoLuong: 12.5})
	phieuThu(t, PhieuHaoHut, "2026-09-04", DongPhieu{MaVatTu: "vt-1", SoLuong: 2.5})

	if got := TonKho()["vt-1"]; got != 85 {
		t.Fatalf("tồn = %v, muốn 85", got)
	}
}

// Kiểm kê ĐẶT LẠI tồn về số đếm được. Nếu nó cộng thêm như phiếu nhập thì
// mỗi lần đi đếm kho lại làm tồn phồng lên, và càng kiểm kê càng sai.
func TestKiemKeDatLaiChuKhongCong(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t,
		VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"},
		VatTu{Ma: "vt-2", Ten: "Keo epoxy", DonVi: "tuýp"},
	)

	phieuThu(t, PhieuNhap, "2026-09-01",
		DongPhieu{MaVatTu: "vt-1", SoLuong: 100},
		DongPhieu{MaVatTu: "vt-2", SoLuong: 10},
	)
	// Đếm một góc kho: chỉ đếm vt-1, không đụng vt-2.
	phieuThu(t, PhieuKiemKe, "2026-09-05", DongPhieu{MaVatTu: "vt-1", SoLuong: 40})
	phieuThu(t, PhieuXuat, "2026-09-06", DongPhieu{MaVatTu: "vt-1", SoLuong: 5})

	ton := TonKho()
	if ton["vt-1"] != 35 {
		t.Errorf("vt-1 = %v, muốn 35 (kiểm kê đặt về 40 rồi xuất 5)", ton["vt-1"])
	}
	if ton["vt-2"] != 10 {
		t.Errorf("vt-2 = %v, muốn 10 — thứ không có tên trong phiếu kiểm kê phải giữ nguyên", ton["vt-2"])
	}
}

// Phiếu kiểm kê ngày 05 phải thắng phiếu nhập ngày 01 dù được ghi sau. Thứ tự
// phát lại là theo NGÀY GHI TRÊN PHIẾU, không phải theo lúc bấm lưu — ghi bù
// phiếu nhập của tuần trước là chuyện xảy ra thật.
func TestPhatLaiTheoNgayKhongTheoLucLuu(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})

	phieuThu(t, PhieuKiemKe, "2026-09-05", DongPhieu{MaVatTu: "vt-1", SoLuong: 40})
	phieuThu(t, PhieuNhap, "2026-09-01", DongPhieu{MaVatTu: "vt-1", SoLuong: 100})

	if got := TonKho()["vt-1"]; got != 40 {
		t.Fatalf("tồn = %v, muốn 40 — kiểm kê ngày sau phải đè phiếu nhập ngày trước", got)
	}
}

// Tồn âm là chuyện có thật: thợ lấy đồ rồi mới ghi phiếu nhập. Máy phải HIỆN
// số âm chứ không kẹp về 0 — kẹp về 0 là giấu đúng cái dấu hiệu "sổ và kho
// đang lệch nhau".
func TestTonAmVanHien(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})
	phieuThu(t, PhieuXuat, "2026-09-02", DongPhieu{MaVatTu: "vt-1", SoLuong: 4})

	if got := TonKho()["vt-1"]; got != -4 {
		t.Fatalf("tồn = %v, muốn -4", got)
	}
	bang := BangTonKho()
	if len(bang) != 1 {
		t.Fatalf("bảng có %d dòng", len(bang))
	}
	if !bang[0].Am {
		t.Error("dòng tồn âm phải bật cờ Am để trang kho tô đỏ")
	}
	if bang[0].GiaTri != 0 {
		t.Errorf("giá trị tồn âm = %d, phải là 0 — nợ kho không phải là tài sản", bang[0].GiaTri)
	}
}

// Xoá phiếu là xoá luôn ảnh hưởng của nó lên tồn. Với sổ cái phát lại thì
// điều này tự đúng, nhưng nó là thứ Kendy sẽ dựa vào lúc lỡ tay ghi nhầm.
func TestXoaPhieuThiTonQuayLai(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})
	phieuThu(t, PhieuNhap, "2026-09-01", DongPhieu{MaVatTu: "vt-1", SoLuong: 100})
	p := phieuThu(t, PhieuXuat, "2026-09-03", DongPhieu{MaVatTu: "vt-1", SoLuong: 30})

	if err := XoaPhieu(p.Ma); err != nil {
		t.Fatal(err)
	}
	if got := TonKho()["vt-1"]; got != 100 {
		t.Fatalf("tồn sau khi xoá phiếu xuất = %v, muốn 100", got)
	}
	if _, co := LayPhieu(p.Ma); co {
		t.Error("phiếu đã xoá vẫn tra ra được")
	}
}

// Ngưỡng tồn tối thiểu và gợi ý mua. Gợi ý chỉ được hiện khi đã chạm ngưỡng —
// hiện lúc nào cũng có thì nó thành nhiễu và Kendy sẽ thôi nhìn nó.
func TestGoiYMuaChiHienKhiChamNguong(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t,
		VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét", TonToiThieu: 20},
		VatTu{Ma: "vt-2", Ten: "Keo epoxy", DonVi: "tuýp", TonToiThieu: 2},
	)
	phieuThu(t, PhieuNhap, "2026-09-01",
		DongPhieu{MaVatTu: "vt-1", SoLuong: 100, DonGia: 3000},
		DongPhieu{MaVatTu: "vt-2", SoLuong: 10, DonGia: 50000},
	)
	phieuThu(t, PhieuXuat, "2026-09-10", DongPhieu{MaVatTu: "vt-1", SoLuong: 85})

	m := map[string]DongTonKho{}
	for _, d := range BangTonKho() {
		m[d.Ma] = d
	}
	if !m["vt-1"].DuoiNguong {
		t.Error("vt-1 còn 15 dưới ngưỡng 20 mà không báo")
	}
	if m["vt-1"].DeXuatNhap <= 0 {
		t.Error("chạm ngưỡng thì phải đề xuất mua bao nhiêu")
	}
	if m["vt-2"].DuoiNguong {
		t.Error("vt-2 còn 10 trên ngưỡng 2 mà lại báo cần mua")
	}
	if m["vt-2"].DeXuatNhap != 0 {
		t.Errorf("vt-2 chưa chạm ngưỡng mà đề xuất mua %v", m["vt-2"].DeXuatNhap)
	}
	if SoVatTuCanMua() != 1 {
		t.Errorf("số vật tư cần mua = %d, muốn 1", SoVatTuCanMua())
	}
	// 15 mét x 3.000 + 10 tuýp x 50.000
	if got := GiaTriTonKho(); got != 15*3000+10*50000 {
		t.Errorf("giá trị tồn = %d", got)
	}
}

// Mã vật tư là thứ mọi phiếu cũ trỏ vào. Lưu danh mục mà sinh mã mới ĐÈ lên
// mã của một dòng phía dưới thì dòng đó im lặng mất sạch lịch sử phiếu — tồn
// của nó nhảy sang một vật tư khác mà không có thông báo nào.
func TestSinhMaVatTuKhongDeLenMaDangDung(t *testing.T) {
	dungKhoTienThu(t)
	dung := map[string]bool{"vt-1": true, "vt-3": true}
	ma := maVatTuMoi(dung)
	if ma == "vt-1" || ma == "vt-3" {
		t.Fatalf("mã mới %q đè lên mã đang dùng", ma)
	}
	if ma != "vt-2" {
		t.Fatalf("mã mới = %q, muốn vt-2 (lấp chỗ trống trước)", ma)
	}
	// Gọi lần nữa mà chưa nhả mã cũ ra thì không được lặp lại chính nó.
	dung[ma] = true
	if lai := maVatTuMoi(dung); lai == ma {
		t.Fatalf("sinh hai lần ra cùng một mã %q", lai)
	}
}

// Định mức là ĐỀ XUẤT, không phải lệnh trừ kho. Đóng đơn chỉ dựng sẵn dòng
// phiếu để thợ sửa và xác nhận — không có phiếu nào tự sinh ra.
func TestDeXuatXuatKhongTuTruKho(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})
	if err := LuuDinhMuc(map[string][]DongDinhMuc{
		"dan-day": {{MaVatTu: "vt-1", SoLuong: 6}},
	}); err != nil {
		t.Fatal(err)
	}
	phieuThu(t, PhieuNhap, "2026-09-01", DongPhieu{MaVatTu: "vt-1", SoLuong: 100})

	don := &Don{Ma: "TV-2609-001", DongTien: []DongTien{{MaDichVu: "dan-day", Ten: "Đan dây", SoTien: 150000}}}
	dx := DeXuatXuatChoDon(don)
	if len(dx) != 1 || dx[0].MaVatTu != "vt-1" || dx[0].SoLuong != 6 {
		t.Fatalf("đề xuất = %+v, muốn 6 mét vt-1", dx)
	}
	if got := TonKho()["vt-1"]; got != 100 {
		t.Fatalf("tồn = %v — đề xuất mà đã trừ kho là sai luật", got)
	}
	if _, co := PhieuXuatCuaDon(don.Ma); co {
		t.Error("chưa ai xác nhận mà đã có phiếu xuất cho đơn")
	}
}

// Tốc độ tiêu thụ chỉ đếm phiếu trong cửa sổ thời gian. Đếm cả phiếu năm
// ngoái thì "còn đủ dùng mấy ngày" luôn ra một con số vô nghĩa.
func TestTieuThuChiTinhTrongCuaSo(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t, VatTu{Ma: "vt-1", Ten: "Dây 1.2mm", DonVi: "mét"})
	nay := time.Now()
	phieuThu(t, PhieuXuat, nay.AddDate(0, 0, -5).Format("2006-01-02"), DongPhieu{MaVatTu: "vt-1", SoLuong: 30})
	phieuThu(t, PhieuXuat, nay.AddDate(0, 0, -200).Format("2006-01-02"), DongPhieu{MaVatTu: "vt-1", SoLuong: 900})

	tt := TieuThuMoiNgay(90)
	if got := tt["vt-1"]; got != 30.0/90.0 {
		t.Fatalf("tiêu thụ = %v, muốn %v — phiếu 200 ngày trước phải nằm ngoài cửa sổ", got, 30.0/90.0)
	}
}
