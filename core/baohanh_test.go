package core

import (
	"strings"
	"testing"
	"time"
)

// dvBaoHanhThu: thay bảng giá bằng vài dịch vụ có số tháng bảo hành khác
// nhau, trả lại như cũ sau test.
func dvBaoHanhThu(t *testing.T) {
	t.Helper()
	cu, cuGD := GIA, GiaiDoan
	t.Cleanup(func() { GIA, GiaiDoan = cu, cuGD })
	g := func(n int) []*int { return []*int{&n} }
	GIA = BangGia{DichVu: []DichVu{
		{Ma: "DAN-DAY", Ten: "Đan dây", Gia: g(150000), BaoHanhThang: 1},
		{Ma: "THAY-CAN", Ten: "Thay cán", Gia: g(80000), BaoHanhThang: 3},
		{Ma: "VE-SINH", Ten: "Vệ sinh", Gia: g(50000)},
	}}
	GiaiDoan = 0
}

func TestCongThang(t *testing.T) {
	cac := []struct {
		ngay string
		n    int
		ra   string
	}{
		{"2026-09-12", 3, "2026-12-12"},
		{"2026-09-12", 0, "2026-09-12"},
		{"2026-12-31", 2, "2027-02-28"}, // 31/02 không có: lùi về cuối tháng
		{"2026-01-31", 1, "2026-02-28"},
		{"", 3, ""},
		{"linh tinh", 3, ""},
	}
	for _, c := range cac {
		if got := congThang(c.ngay, c.n); got != c.ra {
			t.Errorf("congThang(%q,%d) = %q, muốn %q", c.ngay, c.n, got, c.ra)
		}
	}
}

// Hai dịch vụ, một tháng và ba tháng: lấy số LỚN NHẤT. Khách không phân biệt
// được món nào bảo hành mấy tháng.
func TestBaoHanhLayThangLonNhat(t *testing.T) {
	gocTam(t)
	dvBaoHanhThu(t)
	d := &Don{
		Ma: "TV-2609-930", Ngay: "2026-09-01", TrangThai: TTDaGiao,
		DongTien: []DongTien{
			{MaDichVu: "DAN-DAY", Ten: "Đan dây", SoTien: 150000},
			{MaDichVu: "THAY-CAN", Ten: "Thay cán", SoTien: 80000},
		},
		LichSu: []Moc{{Luc: "2026-09-12 10:00", TrangThai: TTDaGiao, Nguoi: "kendy"}},
	}
	if got := BaoHanhThangCuaDon(d); got != 3 {
		t.Fatalf("số tháng = %d, muốn 3", got)
	}
	if !ApBaoHanhMacDinh(d) || d.BaoHanhDen != "2026-12-12" {
		t.Fatalf("BaoHanhDen = %q, muốn 2026-12-12", d.BaoHanhDen)
	}
}

// Chưa giao thì chưa tính: bảo hành đếm từ ngày trao đồ, không phải từ ngày
// nhận. Và dịch vụ không khai tháng nào thì không tự sinh ra hạn.
func TestBaoHanhChuaGiaoThiChuaTinh(t *testing.T) {
	gocTam(t)
	dvBaoHanhThu(t)
	dang := &Don{Ma: "TV-2609-931", Ngay: "2026-09-01", TrangThai: TTDangSua,
		DongTien: []DongTien{{MaDichVu: "THAY-CAN", Ten: "Thay cán", SoTien: 80000}}}
	if ApBaoHanhMacDinh(dang) || dang.BaoHanhDen != "" {
		t.Errorf("đơn chưa giao đã có hạn bảo hành %q", dang.BaoHanhDen)
	}
	khong := &Don{Ma: "TV-2609-932", Ngay: "2026-09-01", TrangThai: TTDaGiao,
		DongTien: []DongTien{{MaDichVu: "VE-SINH", Ten: "Vệ sinh", SoTien: 50000}},
		LichSu:   []Moc{{Luc: "2026-09-12 10:00", TrangThai: TTDaGiao}}}
	if ApBaoHanhMacDinh(khong) || khong.BaoHanhDen != "" {
		t.Errorf("dịch vụ không khai bảo hành mà vẫn ra hạn %q", khong.BaoHanhDen)
	}
}

// Ô gõ tay thắng mặc định, luôn luôn.
func TestBaoHanhGoTayThangMacDinh(t *testing.T) {
	gocTam(t)
	dvBaoHanhThu(t)
	d := &Don{
		Ma: "TV-2609-933", Ngay: "2026-09-01", TrangThai: TTDaGiao,
		BaoHanhDen: "2027-01-01",
		DongTien:   []DongTien{{MaDichVu: "THAY-CAN", Ten: "Thay cán", SoTien: 80000}},
		LichSu:     []Moc{{Luc: "2026-09-12 10:00", TrangThai: TTDaGiao}},
	}
	if ApBaoHanhMacDinh(d) || d.BaoHanhDen != "2027-01-01" {
		t.Fatalf("máy đè lên ngày gõ tay: %q", d.BaoHanhDen)
	}
}

// Đơn bảo hành 0đ không được kéo trung bình một đơn xuống — nó là chi phí,
// không phải một lần bán.
func TestDonBaoHanhKhongKeoTrungBinh(t *testing.T) {
	dungKhoTienThu(t)
	k := Ky{Kieu: KyThang, Tu: "2026-09-01", Den: "2026-09-30", Moc: "2026-09"}
	donThu(t, &Don{Ma: "TV-2609-940", Ngay: "2026-09-02", TrangThai: TTDaGiao, TongTien: 300000,
		LichSu: []Moc{{Luc: "2026-09-03 10:00", TrangThai: TTDaGiao}}})
	donThu(t, &Don{Ma: "TV-2609-941", Ngay: "2026-09-04", TrangThai: TTDaGiao, TongTien: 100000,
		LichSu: []Moc{{Luc: "2026-09-05 10:00", TrangThai: TTDaGiao}}})
	donThu(t, &Don{Ma: "TV-2609-942", Ngay: "2026-09-06", TrangThai: TTDaGiao, TongTien: 0,
		MaDonGoc: "TV-2609-940",
		LichSu:   []Moc{{Luc: "2026-09-07 10:00", TrangThai: TTDaGiao}}})

	tk := LayThongKeKy("", k)
	if tk.GiaoDon != 3 {
		t.Fatalf("giao %d đơn, muốn 3", tk.GiaoDon)
	}
	if tk.TBMotDon != 200000 {
		t.Errorf("TB một đơn = %d, muốn 200000 (bỏ đơn bảo hành)", tk.TBMotDon)
	}
	if tk.DonBaoHanh != 1 {
		t.Errorf("đếm %d đơn bảo hành, muốn 1", tk.DonBaoHanh)
	}
	if tk.TyLeBaoHanh != 33 {
		t.Errorf("tỷ lệ bảo hành = %d%%, muốn 33", tk.TyLeBaoHanh)
	}
}

// Hai chiều: đơn gốc kể ra đã mở mấy đơn bảo hành, đơn bảo hành chỉ ngược về
// gốc. Nhìn đơn nào cũng thấy được chuyện.
func TestTrangDonNoiHaiChieuBaoHanh(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	goc := donThu(t, &Don{Ma: "TV-2609-950", KhachTen: "Anh Chín", KhachLienHe: "0900000009",
		TrangThai: TTDaGiao, TongTien: 250000, VotHang: "Yonex",
		LichSu: []Moc{{Luc: "2026-09-05 10:00", TrangThai: TTDaGiao}}})
	bh := donThu(t, &Don{Ma: "TV-2609-951", KhachTen: "Anh Chín", KhachLienHe: "0900000009",
		MaDonGoc: goc.Ma, VotHang: "Yonex"})

	s := moTrang(t, mux, ck, "/qt/don/"+goc.Ma)
	if !strings.Contains(s, `href="/qt/don/`+bh.Ma+`"`) {
		t.Error("đơn gốc không kể ra đơn bảo hành đã mở")
	}
	if !strings.Contains(s, "/qt/don-moi?bao_hanh="+goc.Ma) {
		t.Error("đơn đã giao mà không có nút mở đơn bảo hành")
	}
	s = moTrang(t, mux, ck, "/qt/don/"+bh.Ma)
	if !strings.Contains(s, `href="/qt/don/`+goc.Ma+`"`) {
		t.Error("đơn bảo hành không chỉ ngược về đơn gốc")
	}
	if strings.Contains(s, "/qt/don-moi?bao_hanh="+bh.Ma) {
		t.Error("đơn bảo hành lại mở được đơn bảo hành của chính nó")
	}
}

// Form đơn mới mở từ nút bảo hành: điền sẵn khách và món, mang theo mã gốc.
func TestFormDonMoiTuBaoHanh(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	goc := donThu(t, &Don{Ma: "TV-2609-952", KhachTen: "Chị Mười", KhachLienHe: "0900000010",
		TrangThai: TTDaGiao, VotHang: "Lining Axforce", TongTien: 200000,
		LichSu: []Moc{{Luc: "2026-09-05 10:00", TrangThai: TTDaGiao}}})
	s := moTrang(t, mux, ck, "/qt/don-moi?bao_hanh="+goc.Ma)
	for _, muon := range []string{`value="Chị Mười"`, `value="0900000010"`, `value="Lining Axforce"`,
		`name="ma_don_goc" value="` + goc.Ma + `"`} {
		if !strings.Contains(s, muon) {
			t.Errorf("form bảo hành thiếu %s", muon)
		}
	}
}

// Rổ "Còn bảo hành": đơn đã giao, hạn chưa qua.
func TestLocConBaoHanh(t *testing.T) {
	dungKhoTienThu(t)
	mai := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	qua := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	donThu(t, &Don{Ma: "TV-2609-960", TrangThai: TTDaGiao, BaoHanhDen: mai})
	donThu(t, &Don{Ma: "TV-2609-961", TrangThai: TTDaGiao, BaoHanhDen: qua})
	donThu(t, &Don{Ma: "TV-2609-962", TrangThai: TTDangSua})

	ds := LocDon(BoLoc{ConBaoHanh: true})
	if len(ds) != 1 || ds[0].Ma != "TV-2609-960" {
		t.Fatalf("lọc còn bảo hành ra %d đơn: %v", len(ds), ds)
	}
}
