package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func donThu(t *testing.T, d *Don) *Don {
	t.Helper()
	if d.Ngay == "" {
		d.Ngay = time.Now().Format("2006-01-02")
	}
	if d.TrangThai == "" {
		d.TrangThai = TTMoi
	}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	return d
}

func doiTacThu(t *testing.T, ten string) DoiTac {
	t.Helper()
	dt, err := LuuDoiTac(DoiTac{Ten: ten})
	if err != nil {
		t.Fatal(err)
	}
	return dt
}

// Hai loại đơn đánh số riêng. Chung một dãy số thì nhìn mã TG-2609-014 không
// đoán được trạm đã nhận bao nhiêu đôi giày, mà đó là con số Kendy hỏi hằng tuần.
func TestSoDonGiayVaVotDemRiengNhau(t *testing.T) {
	dungKhoTienThu(t)

	v1 := MaDonMoi(LoaiVot)
	donThu(t, &Don{Ma: v1, Loai: LoaiVot})
	g1 := MaDonMoi(LoaiGiay)
	donThu(t, &Don{Ma: g1, Loai: LoaiGiay})
	v2 := MaDonMoi(LoaiVot)
	g2 := MaDonMoi(LoaiGiay)

	if !strings.HasPrefix(v1, "TV-") || !strings.HasPrefix(g1, "TG-") {
		t.Fatalf("tiền tố sai: vợt %s, giày %s", v1, g1)
	}
	if !strings.HasSuffix(v1, "001") || !strings.HasSuffix(g1, "001") {
		t.Errorf("đơn đầu tiên của mỗi loại phải là 001: %s / %s", v1, g1)
	}
	if !strings.HasSuffix(v2, "002") {
		t.Errorf("đơn vợt thứ hai = %s, muốn kết thúc 002", v2)
	}
	if !strings.HasSuffix(g2, "002") {
		t.Errorf("đơn giày thứ hai = %s — giày không được đếm nhờ đơn vợt", g2)
	}
}

// Đơn cũ lưu trước khi có trường Loai thì phải nằm ở trang vợt. Trôi sang
// trang giày là mấy trăm đơn cũ biến khỏi chỗ Kendy quen tìm.
func TestDonCuKhongCoLoaiTinhLaDonVot(t *testing.T) {
	dungKhoTienThu(t)
	cu := donThu(t, &Don{Ma: "TV-2508-001"}) // Loai rỗng: đơn thời chưa tách loại

	if cu.LaGiay() {
		t.Error("đơn không ghi loại mà bị tính là giày")
	}
	if got := len(LocDon(BoLoc{Loai: LoaiVot})); got != 1 {
		t.Errorf("trang vợt thấy %d đơn, muốn 1", got)
	}
	if got := len(LocDon(BoLoc{Loai: LoaiGiay})); got != 0 {
		t.Errorf("trang giày thấy %d đơn, muốn 0", got)
	}
	if LayThongKeDon(LoaiVot).TongDon != 1 {
		t.Error("thống kê vợt bỏ sót đơn cũ")
	}
}

// Lọc theo loại phải kín hai chiều: trang này không được lẫn đơn của trang kia.
func TestLocTheoLoaiKhongLanNhau(t *testing.T) {
	dungKhoTienThu(t)
	donThu(t, &Don{Ma: "TV-2609-001", Loai: LoaiVot, VotHang: "Joola Perseus"})
	donThu(t, &Don{Ma: "TG-2609-001", Loai: LoaiGiay, GiayHang: "Nike Vapor", GiaySize: "42"})
	donThu(t, &Don{Ma: "TG-2609-002", Loai: LoaiGiay, GiayHang: "Adidas"})

	vot := LocDon(BoLoc{Loai: LoaiVot})
	if len(vot) != 1 || vot[0].Ma != "TV-2609-001" {
		t.Errorf("lọc vợt ra %d đơn: %v", len(vot), vot)
	}
	if got := len(LocDon(BoLoc{Loai: LoaiGiay})); got != 2 {
		t.Errorf("lọc giày ra %d đơn, muốn 2", got)
	}
	if got := len(LocDon(BoLoc{})); got != 3 {
		t.Errorf("không lọc loại phải ra cả 3 đơn, ra %d", got)
	}
	// Tìm theo hãng giày và theo cỡ — hai thứ thợ gõ vào ô tìm nhiều nhất.
	if got := LocDon(BoLoc{Loai: LoaiGiay, Tim: "nike"}); len(got) != 1 {
		t.Errorf("tìm \"nike\" ra %d đơn, muốn 1", len(got))
	}
	if got := LocDon(BoLoc{Loai: LoaiGiay, Tim: "42"}); len(got) != 1 {
		t.Errorf("tìm cỡ \"42\" ra %d đơn, muốn 1", len(got))
	}
}

// Ngưỡng tăng khối lượng là luật của vợt. Đơn giày mà bị gắn cảnh báo vượt
// ngưỡng thì mỗi đôi giày nặng thêm đế lại đỏ một dòng vô nghĩa.
func TestDonGiayKhongDinhNguongCan(t *testing.T) {
	dungKhoTienThu(t)
	// Trạm thử chưa có bang-gia.yaml nên ngưỡng rơi về mặc định 3g trong code.
	nguong := NguongHienTai().TangKhoiLuongToiDaG
	if nguong <= 0 {
		nguong = 3.0
	}
	giay := &Don{Ma: "TG-2609-003", Loai: LoaiGiay, CanTruocG: 300, CanSauG: 300 + nguong + 50}
	if giay.VuotNguongCan() {
		t.Error("đơn giày không được vướng ngưỡng cân của vợt")
	}
	vot := &Don{Ma: "TV-2609-003", Loai: LoaiVot, CanTruocG: 220, CanSauG: 220 + nguong + 1}
	if !vot.VuotNguongCan() {
		t.Error("đơn vợt vượt ngưỡng mà không báo — luật cũ bị gãy")
	}
}

// Lãi thật là hiệu của hai con số người gõ: khách trả bao nhiêu, trạm trả
// tiệm bao nhiêu. Không có công thức %, không có giá nào do máy sinh ra.
func TestLaiThatLaHieuCuaHaiConSoGoTay(t *testing.T) {
	dungKhoTienThu(t)
	d := &Don{
		Ma: "TG-2609-004", Loai: LoaiGiay,
		DongTien: []DongTien{{Ten: "Dán đế", SoTien: 400000}},
		TongTien: 400000,
		GuiDi:    GuiDi{MaDoiTac: "tiem-a", TraDoiTac: 250000},
	}
	if d.TongTien != 400000 {
		t.Fatalf("tổng đơn = %d", d.TongTien)
	}
	if d.LaiThat() != 150000 {
		t.Errorf("trạm ăn = %d, muốn 150.000", d.LaiThat())
	}
	// Tự làm tại trạm: không trừ ai cả, trạm ăn trọn.
	tuLam := &Don{Ma: "TG-2609-005", Loai: LoaiGiay, TongTien: 400000,
		DongTien: []DongTien{{Ten: "Dán đế", SoTien: 400000}}}
	if tuLam.GuiRaNgoai() {
		t.Error("không chọn tiệm mà vẫn tính là gửi ra ngoài")
	}
	if tuLam.LaiThat() != 400000 {
		t.Errorf("đơn tự làm: trạm ăn = %d, muốn 400.000", tuLam.LaiThat())
	}
}

// Bấm "đã trả tiệm" phải sinh đúng MỘT khoản chi, bấm lại không sinh thêm,
// bỏ tick thì khoản ấy biến mất. Sai chỗ này là sổ tiền phình ra những khoản
// ma không ai đối chiếu được với ngân hàng.
func TestTraDoiTacGhiVaXoaDungMotKhoanChi(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	dt := doiTacThu(t, "Anh Tuấn giày")
	don := donThu(t, &Don{
		Ma: "TG-2609-010", Loai: LoaiGiay,
		DongTien: []DongTien{{Ten: "Dán đế", SoTien: 500000}},
		TongTien: 500000,
		GuiDi:    GuiDi{MaDoiTac: dt.Ma, TraDoiTac: 300000, NgayVe: "2026-09-08"},
	})
	tok := tokenTuPhien(ck.Value)

	postThu(t, h, ck, "/qt/don/"+don.Ma+"/tra-doi-tac", "_csrf="+tok+"&da_tra=1&phuong_thuc=tien_mat")
	postThu(t, h, ck, "/qt/don/"+don.Ma+"/tra-doi-tac", "_csrf="+tok+"&da_tra=1&phuong_thuc=tien_mat")

	sau, _ := LayDon(don.Ma)
	if !sau.GuiDi.DaTra || sau.GuiDi.MaKhoanChi == "" {
		t.Fatalf("chưa đánh dấu đã trả: %+v", sau.GuiDi)
	}
	chi := demKhoanChiCuaDon(t, don.Ma)
	if chi != 1 {
		t.Fatalf("sổ tiền có %d khoản chi cho đơn này, muốn đúng 1", chi)
	}
	// Khoản chi mang mã đơn nhưng là chi vận hành — không được đụng vào số
	// tiền khách đã trả cho đơn.
	if sau.DaThu() != 0 {
		t.Errorf("đã thu của khách = %d, khoản trả tiệm không được tính vào đây", sau.DaThu())
	}

	postThu(t, h, ck, "/qt/don/"+don.Ma+"/tra-doi-tac", "_csrf="+tok+"&da_tra=0")
	sau2, _ := LayDon(don.Ma)
	if sau2.GuiDi.DaTra || sau2.GuiDi.MaKhoanChi != "" {
		t.Errorf("bỏ tick rồi mà đơn vẫn giữ dấu đã trả: %+v", sau2.GuiDi)
	}
	if chi := demKhoanChiCuaDon(t, don.Ma); chi != 0 {
		t.Errorf("bỏ tick rồi mà sổ còn %d khoản chi", chi)
	}
}

func demKhoanChiCuaDon(t *testing.T, maDon string) int {
	t.Helper()
	n := 0
	for _, thang := range CacThangCoSo() {
		for _, k := range KhoanTrongThang(thang) {
			if k.MaDon == maDon && k.Loai == KhoanChi {
				n++
			}
		}
	}
	return n
}

// Đã trả tiền tiệm rồi thì không được bỏ chọn tiệm: khoản chi trong sổ đang
// trỏ vào đơn này, gỡ tiệm đi là còn lại một khoản chi không biết trả cho ai.
func TestKhongBoDuocTiemKhiDaTraTien(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)
	dt := doiTacThu(t, "Tiệm B")
	don := donThu(t, &Don{
		Ma: "TG-2609-011", Loai: LoaiGiay,
		DongTien: []DongTien{{Ten: "Khâu mũi", SoTien: 200000}},
		TongTien: 200000,
		GuiDi:    GuiDi{MaDoiTac: dt.Ma, TraDoiTac: 120000, DaTra: true, MaKhoanChi: "GIA-DINH"},
	})
	postThu(t, h, ck, "/qt/don/"+don.Ma+"/gui-di", "_csrf="+tokenTuPhien(ck.Value)+"&ma_doi_tac=")

	sau, _ := LayDon(don.Ma)
	if sau.GuiDi.MaDoiTac != dt.Ma {
		t.Errorf("tiệm bị gỡ mất dù đã trả tiền: %+v", sau.GuiDi)
	}
}

// Tiệm đã dính vào đơn thì không xoá hẳn được — xoá là lịch sử mấy chục đơn
// trỏ vào khoảng không. Đường thoát là đánh dấu ngừng hợp tác.
func TestKhongXoaDuocTiemDangDinhVaoDon(t *testing.T) {
	dungKhoTienThu(t)
	dt := doiTacThu(t, "Tiệm C")
	donThu(t, &Don{Ma: "TG-2609-012", Loai: LoaiGiay, GuiDi: GuiDi{MaDoiTac: dt.Ma}})

	if err := XoaDoiTac(dt.Ma); err == nil {
		t.Fatal("xoá được tiệm đang có đơn — lịch sử đơn sẽ mất tên tiệm")
	}
	dt.Ngung = true
	if _, err := LuuDoiTac(dt); err != nil {
		t.Fatal(err)
	}
	for _, d := range DoiTacDangDung() {
		if d.Ma == dt.Ma {
			t.Error("tiệm đã ngừng vẫn hiện trong ô chọn của đơn mới")
		}
	}
	if TenDoiTac(dt.Ma) != "Tiệm C" {
		t.Errorf("đơn cũ mất tên tiệm: %q", TenDoiTac(dt.Ma))
	}
}

// Thợ vẫn phải mở được cả hai trang đơn và chọn được tiệm cho đơn mình làm.
// Nhưng danh sách tiệm và tiền trả tiệm là chuyện của chủ.
func TestThoVaoDuocTrangDonNhungKhongVaoDuocTrangDoiTac(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroTho)

	for _, d := range []string{"/qt/don-vot", "/qt/don-giay", "/qt/don-moi?loai=giay"} {
		r := httptest.NewRequest("GET", d, nil)
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Errorf("thợ mở %s trả %d", d, w.Code)
		}
	}

	r := httptest.NewRequest("GET", "/qt/doi-tac", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("thợ mở /qt/doi-tac trả %d, muốn 403", w.Code)
	}
}

// Thống kê hai trang phải đếm riêng, và số đang nằm ở tiệm ngoài phải tách
// khỏi số đang trên bàn mình — hai thứ đó trả lời hai câu hỏi khác nhau.
func TestThongKeDemRiengVaTachHangODuoiTiem(t *testing.T) {
	dungKhoTienThu(t)
	hom_qua := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	donThu(t, &Don{Ma: "TV-2609-020", Loai: LoaiVot, TrangThai: TTDangSua})
	donThu(t, &Don{Ma: "TG-2609-020", Loai: LoaiGiay, TrangThai: TTDaGuiDi,
		GuiDi: GuiDi{MaDoiTac: "tiem-x", NgayHenVe: hom_qua}})
	donThu(t, &Don{Ma: "TG-2609-021", Loai: LoaiGiay, TrangThai: TTXong,
		DongTien: []DongTien{{Ten: "Dán đế", SoTien: 500000}}, TongTien: 500000,
		GuiDi: GuiDi{MaDoiTac: "tiem-x", TraDoiTac: 300000}})

	tkV := LayThongKeDon(LoaiVot)
	if tkV.TongDon != 1 || tkV.OTiemNgoai != 0 {
		t.Errorf("thống kê vợt: %d đơn, %d ở tiệm ngoài", tkV.TongDon, tkV.OTiemNgoai)
	}
	tkG := LayThongKeDon(LoaiGiay)
	if tkG.TongDon != 2 {
		t.Errorf("thống kê giày: %d đơn, muốn 2", tkG.TongDon)
	}
	if tkG.OTiemNgoai != 1 {
		t.Errorf("đang ở tiệm ngoài = %d, muốn 1", tkG.OTiemNgoai)
	}
	if tkG.QuaHenDoiTac != 1 {
		t.Errorf("tiệm trễ hẹn = %d, muốn 1", tkG.QuaHenDoiTac)
	}
	if tkG.NoDoiTacTong != 300000 {
		t.Errorf("chưa trả tiệm = %d, muốn 300.000", tkG.NoDoiTacTong)
	}
}

func TestTrangThaiChoHangVeHopLe(t *testing.T) {
	if !TrangThaiHopLe(TTChoHangVe) {
		t.Fatal("cho_hang_ve phải là trạng thái hợp lệ")
	}
	mt := TrangThaiCua(TTChoHangVe)
	if !mt.DangChay {
		t.Fatal("đơn chờ hàng về vẫn đang chạy — nó phải hiện trong danh sách việc")
	}
	if mt.TenKhach == "" || mt.TenKhach == TTChoHangVe {
		t.Fatal("thiếu câu nói cho khách")
	}
}

// cho_hang_ve đứng TRƯỚC moi: thứ tự trong CacTrangThai là thứ tự trên bảng
// điều khiển, và việc đầu tiên mỗi sáng là xem kiện nào đã về.
func TestChoHangVeDungDauDanhSach(t *testing.T) {
	if CacTrangThai[0].Ma != TTChoHangVe {
		t.Fatalf("trạng thái đầu phải là %s, nhận %s", TTChoHangVe, CacTrangThai[0].Ma)
	}
}

// Đơn cũ trong data/ không có trường mới nào — đọc vẫn phải ra nghĩa đúng.
func TestDonCuKhongCoTruongMoi(t *testing.T) {
	cu := []byte(`{"ma":"TV-2601-001","trang_thai":"xong","khach_ten":"A"}`)
	var d Don
	if err := json.Unmarshal(cu, &d); err != nil {
		t.Fatalf("đọc đơn cũ: %v", err)
	}
	if d.NguonDonHopLe() != NguonTay {
		t.Fatalf("đơn cũ phải hiểu là nhận tay, nhận %q", d.NguonDonHopLe())
	}
	if d.HinhThucTra != "" || d.KhachDiaChi != "" {
		t.Fatal("đơn cũ không có mấy trường này")
	}
	if d.ChoHangVe() {
		t.Fatal("đơn xong không phải đang chờ hàng về")
	}
}
