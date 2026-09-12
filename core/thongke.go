// Thống kê theo kỳ: con số của một ngày, một tháng, một năm.
//
// Tách khỏi LayThongKeDon trong donhang.go, và cố ý không gộp: hàm cũ trả
// bảng điều khiển "ngay lúc này" (đang trên bàn, quá hẹn, còn nợ) mà sổ tiền
// với mail hằng ngày đang gọi; hàm ở đây trả sổ của MỘT KHOẢNG đã đóng.
// Nhập hai thứ làm một thì mọi chỗ gọi đều phải truyền thêm một kỳ mà chúng
// không quan tâm.
//
// Luật đếm — chỉ có hai, và mọi con số dưới đây theo đúng hai luật này:
//
//	Đơn NHẬN trong kỳ  → tính theo Don.Ngay.
//	Đơn GIAO trong kỳ  → tính theo Don.NgayGiao(), và CHỈ đơn giao mới
//	                     thành doanh thu. Vợt còn trên bàn chưa phải tiền.
//
// Hai con số ấy lệch nhau là bình thường, nên nhãn trên trang phải nói rõ
// cái nào là cái nào.
package core

import (
	"sort"
	"strings"
)

// DongKy — một dòng trong bảng chia nhỏ của kỳ. Kỳ năm chia theo tháng, kỳ
// tháng chia theo ngày, kỳ ngày chỉ một dòng.
type DongKy struct {
	Moc      string
	Ten      string
	NhanDon  int
	GiaoDon  int
	DoanhThu int
	DaThu    int
}

type ODichVuKy struct {
	Ten    string
	So     int
	SoTien int
}

type ThongKeKy struct {
	Ky Ky

	NhanDon int // đơn nhận trong kỳ
	GiaoDon int // đơn giao trong kỳ
	TuChoi  int // đơn nhận trong kỳ, kết cục là từ chối

	DoanhThu  int // tổng tiền của đơn giao trong kỳ
	DaThu     int
	ConNo     int // đơn đã giao trong kỳ mà chưa thu đủ
	TraDoiTac int // tiền trả tiệm ngoài của chính mấy đơn ấy
	LaiGop    int // DoanhThu - TraDoiTac
	// VatTu: giá vốn vật tư ƯỚC TÍNH theo giá nhập gần nhất — xem GiaVonDon
	// ở core/kho.go. Không phải tiền mặt đã chi; tiền mặt nằm ở Sổ tiền, ra
	// lúc mua hàng chứ không phải lúc dùng. Hai bảng lệch nhau là bình
	// thường, và trang thống kê phải in dòng nói rõ chuyện đó.
	VatTu       int
	LaiSauVatTu int // LaiGop - VatTu

	TyLeTuChoi int
	// TBMotDon: doanh thu trung bình một đơn đã giao, KHÔNG kể đơn bảo hành.
	// Đơn bảo hành là chi phí làm lại chứ không phải một lần bán; để nó vào
	// mẫu số thì trạm càng phải sửa lại nhiều, con số "trung bình một đơn"
	// càng đẹp lên — ngược hẳn sự thật.
	TBMotDon int

	// DonBaoHanh: đơn có MaDonGoc, nhận trong kỳ. TyLeBaoHanh so với số đơn
	// nhận. Chưa biết tỷ lệ phải làm lại là bao nhiêu thì không biết giá
	// đang đặt đúng hay sai.
	DonBaoHanh  int
	TyLeBaoHanh int

	TheoTrangThai []OThongKe
	TheoLoai      []OThongKe
	TheoDichVu    []ODichVuKy
	Dong          []DongKy
}

// LayThongKeKy — loai rỗng là gộp cả vợt lẫn giày.
func LayThongKeKy(loai string, k Ky) ThongKeKy {
	donMu.RLock()
	ds := make([]Don, 0, len(donDs))
	for _, d := range donDs {
		if loai != "" && LoaiHopLe(d.Loai) != LoaiHopLe(loai) {
			continue
		}
		ds = append(ds, *d)
	}
	donMu.RUnlock()

	tk := ThongKeKy{Ky: k}
	demTT := map[string]int{}
	demLoai := map[string]int{}
	demDV := map[string]*ODichVuKy{}
	dong := map[string]*DongKy{}
	// Đếm riêng phần "bán được": mẫu số của TBMotDon bỏ đơn bảo hành ra.
	giaoBan, doanhThuBan := 0, 0

	lay := func(moc string) *DongKy {
		if dong[moc] == nil {
			dong[moc] = &DongKy{Moc: moc, Ten: tenMocKy(moc)}
		}
		return dong[moc]
	}

	for _, d := range ds {
		if trongKhoang(d.Ngay, k.Tu, k.Den) {
			tk.NhanDon++
			demTT[d.TrangThai]++
			demLoai[LoaiHopLe(d.Loai)]++
			if d.TrangThai == TTTuChoi {
				tk.TuChoi++
			}
			if d.LaDonBaoHanh() {
				tk.DonBaoHanh++
			}
			lay(mocCuaKy(k, d.Ngay)).NhanDon++
		}

		ng := d.NgayGiao()
		if ng == "" || !trongKhoang(ng, k.Tu, k.Den) {
			continue
		}
		tk.GiaoDon++
		if !d.LaDonBaoHanh() {
			giaoBan++
			doanhThuBan += d.TongTien
		}
		tk.DoanhThu += d.TongTien
		tk.DaThu += d.DaThu()
		tk.ConNo += d.ConNo()
		tk.TraDoiTac += d.GuiDi.TraDoiTac
		tk.VatTu += GiaVonDon(d.Ma)

		r := lay(mocCuaKy(k, ng))
		r.GiaoDon++
		r.DoanhThu += d.TongTien
		r.DaThu += d.DaThu()

		for _, dt := range d.DongTien {
			ten := strings.TrimSpace(dt.Ten)
			if ten == "" {
				ten = "(không tên)"
			}
			// Dòng giảm giá gộp về một mục, đừng để mỗi lý do thành một hàng
			// riêng trong bảng — "Giảm giá — khách quen" và "Giảm giá — lễ"
			// là cùng một chuyện khi nhìn cả tháng.
			if dt.MaDichVu == MaGiamGia {
				ten = tenGiamGia
			}
			if demDV[ten] == nil {
				demDV[ten] = &ODichVuKy{Ten: ten}
			}
			demDV[ten].So++
			demDV[ten].SoTien += dt.SoTien
		}
	}

	tk.LaiGop = tk.DoanhThu - tk.TraDoiTac
	tk.LaiSauVatTu = tk.LaiGop - tk.VatTu
	if tk.NhanDon > 0 {
		tk.TyLeTuChoi = tk.TuChoi * 100 / tk.NhanDon
	}
	if tk.NhanDon > 0 {
		tk.TyLeBaoHanh = tk.DonBaoHanh * 100 / tk.NhanDon
	}
	if giaoBan > 0 {
		tk.TBMotDon = doanhThuBan / giaoBan
	}

	for _, t := range CacTrangThai {
		if demTT[t.Ma] > 0 || t.DangChay {
			tk.TheoTrangThai = append(tk.TheoTrangThai, OThongKe{t.Ma, t.Ten, demTT[t.Ma]})
		}
	}
	for _, l := range []string{LoaiVot, LoaiGiay} {
		tk.TheoLoai = append(tk.TheoLoai, OThongKe{l, "Đơn " + TenLoaiDon(l), demLoai[l]})
	}
	for _, v := range demDV {
		tk.TheoDichVu = append(tk.TheoDichVu, *v)
	}
	sort.Slice(tk.TheoDichVu, func(i, j int) bool {
		return tk.TheoDichVu[i].SoTien > tk.TheoDichVu[j].SoTien
	})
	for _, v := range dong {
		tk.Dong = append(tk.Dong, *v)
	}
	sort.Slice(tk.Dong, func(i, j int) bool { return tk.Dong[i].Moc > tk.Dong[j].Moc })
	// Kỳ "từ trước tới nay" chia theo tháng và có thể dài bằng tuổi trạm —
	// cắt còn hai năm gần nhất, dài hơn thì cột nào cũng bé tí không đọc nổi.
	if k.KhongGioiHan() && len(tk.Dong) > 24 {
		tk.Dong = tk.Dong[:24]
	}
	return tk
}

// mocCuaKy — một ngày ISO rơi vào dòng nào của bảng chia nhỏ.
func mocCuaKy(k Ky, ngay string) string {
	if len(ngay) < 10 {
		return ngay
	}
	switch k.Kieu {
	case KyNgay:
		return ngay
	case KyThang:
		return ngay // kỳ tháng chia theo ngày
	}
	return ngay[:7] // kỳ năm và kỳ "tất cả" chia theo tháng
}

func tenMocKy(moc string) string {
	if len(moc) == 7 {
		return thangDep(moc)
	}
	return ngayDep(moc)
}

// CaoNhatKy — cột cao nhất, để vẽ biểu đồ ngang. Không bao giờ trả 0: mẫu
// lấy con số này làm mẫu số.
func CaoNhatKy(ds []DongKy) int {
	m := 0
	for _, d := range ds {
		if d.DoanhThu > m {
			m = d.DoanhThu
		}
	}
	if m == 0 {
		return 1
	}
	return m
}
