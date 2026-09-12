package core

import "testing"

func TestGiaVonDon(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t, VatTu{Ma: "vt-vien", Ten: "Viền vợt", DonVi: "cái"})
	phieuThu(t, PhieuNhap, "2026-09-01", DongPhieu{MaVatTu: "vt-vien", SoLuong: 10, DonGia: 20000})

	p := &Phieu{Loai: PhieuXuat, Ngay: "2026-09-05", Nguoi: "kendy", MaDon: "TV-2609-970",
		Dong: []DongPhieu{{MaVatTu: "vt-vien", SoLuong: 2}}}
	if err := LuuPhieu(p); err != nil {
		t.Fatal(err)
	}

	if got := GiaVonDon("TV-2609-970"); got != 40000 {
		t.Errorf("giá vốn = %d, muốn 40000", got)
	}
	if got := GiaVonDon("TV-2609-971"); got != 0 {
		t.Errorf("đơn không có phiếu xuất phải là 0, nhận %d", got)
	}
	if got := GiaVonDon(""); got != 0 {
		t.Errorf("mã rỗng phải là 0, nhận %d", got)
	}
}

func TestLaiSauVatTu(t *testing.T) {
	dungKhoTienThu(t)
	vatTuThu(t, VatTu{Ma: "vt-vien", Ten: "Viền vợt", DonVi: "cái"})
	phieuThu(t, PhieuNhap, "2026-09-01", DongPhieu{MaVatTu: "vt-vien", SoLuong: 10, DonGia: 20000})

	d := donThu(t, &Don{Ma: "TV-2609-972", TrangThai: TTDaGiao, TongTien: 500000,
		Ngay: "2026-09-02", VotHang: "Yonex",
		GuiDi:  GuiDi{TraDoiTac: 100000},
		LichSu: []Moc{{Luc: "2026-09-05 10:00", TrangThai: TTDaGiao}}})
	p := &Phieu{Loai: PhieuXuat, Ngay: "2026-09-05", Nguoi: "kendy", MaDon: d.Ma,
		Dong: []DongPhieu{{MaVatTu: "vt-vien", SoLuong: 2}}}
	if err := LuuPhieu(p); err != nil {
		t.Fatal(err)
	}

	tk := LayThongKeKy("", Ky{Tu: "2026-09-01", Den: "2026-09-30"})
	if tk.LaiGop != 400000 {
		t.Errorf("lãi gộp = %d, muốn 400000", tk.LaiGop)
	}
	if tk.VatTu != 40000 {
		t.Errorf("vật tư = %d, muốn 40000", tk.VatTu)
	}
	if tk.LaiSauVatTu != 360000 {
		t.Errorf("lãi sau vật tư = %d, muốn 360000", tk.LaiSauVatTu)
	}
}
