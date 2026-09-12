package core

import (
	"os"
	"strings"
	"testing"
)

func TestChuanSo(t *testing.T) {
	cases := map[string]string{
		"0846161368":     "0846161368",
		"0846 161 368":   "0846161368",
		"084.616.1368":   "0846161368",
		"+84846161368":   "0846161368",
		"84846161368":    "0846161368",
		"  0846161368  ": "0846161368",
		"":               "",
		"zalo abc":       "",
	}
	for vao, mong := range cases {
		if ra := ChuanSo(vao); ra != mong {
			t.Errorf("ChuanSo(%q) = %q, mong %q", vao, ra, mong)
		}
	}
}

func TestGhiNhanKhachGopTheoSo(t *testing.T) {
	gocTam(t)
	if err := NapKhach(); err != nil {
		t.Fatal(err)
	}
	a := GhiNhanKhach("Anh Minh", "0846 161 368", "", "")
	b := GhiNhanKhach("Minh", "+84846161368", "minh@x.vn", "Hà Nội")
	if a == "" || a != b {
		t.Fatalf("hai cách gõ cùng một số phải ra một mã: %q vs %q", a, b)
	}
	k, ok := KhachTheoSo("0846161368")
	if !ok {
		t.Fatal("không tra được khách vừa ghi")
	}
	if k.Email != "minh@x.vn" || k.DiaChi != "Hà Nội" {
		t.Errorf("lần sau có thêm email/địa chỉ thì phải điền vào: %+v", k)
	}
	if k.Ten != "Anh Minh" {
		t.Errorf("tên đã có thì không ghi đè, nhận %q", k.Ten)
	}
	if k.Nhom != NhomKhachLe {
		t.Errorf("khách mới phải vào nhóm lẻ, nhận %q", k.Nhom)
	}
	if _, ok := KhachTheoMa(a); !ok {
		t.Errorf("tra theo mã %q không ra", a)
	}
}

func TestGhiNhanKhachKhongSoThiThoi(t *testing.T) {
	gocTam(t)
	if err := NapKhach(); err != nil {
		t.Fatal(err)
	}
	if ma := GhiNhanKhach("Khách vãng lai", "zalo abc", "", ""); ma != "" {
		t.Fatalf("không có số thì không dựng hồ sơ, nhận %q", ma)
	}
}

func TestKhachNapLaiTuFile(t *testing.T) {
	gocTam(t)
	if err := NapKhach(); err != nil {
		t.Fatal(err)
	}
	ma := GhiNhanKhach("Anh Minh", "0846161368", "", "")
	khachDs = nil
	if err := NapKhach(); err != nil {
		t.Fatal(err)
	}
	k, ok := KhachTheoMa(ma)
	if !ok || k.SoChuan != "0846161368" {
		t.Fatalf("nạp lại không thấy khách vừa ghi: %+v", k)
	}
}

func TestSoLieuKhachCongNo(t *testing.T) {
	moCuaHang(t)
	if err := NapKhach(); err != nil {
		t.Fatal(err)
	}
	if err := NapSoTien(); err != nil {
		t.Fatal(err)
	}
	ma := GhiNhanKhach("Anh Minh", "0846161368", "", "")

	a := &Don{Ma: "TV-2609-901", Ngay: "2026-09-01", MaKhach: ma, TrangThai: TTDaGiao,
		KhachTen: "Anh Minh", KhachLienHe: "0846161368", TongTien: 500000}
	b := &Don{Ma: "TV-2609-902", Ngay: "2026-09-05", MaKhach: ma, TrangThai: TTDangSua,
		KhachTen: "Anh Minh", KhachLienHe: "0846161368", TongTien: 300000}
	for _, d := range []*Don{a, b} {
		if err := LuuDon(d); err != nil {
			t.Fatal(err)
		}
	}
	for _, k := range []Khoan{
		{Ngay: "2026-09-02", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 500000, MaDon: a.Ma},
		{Ngay: "2026-09-06", Loai: KhoanThu, Nhom: NhomSuaChua, SoTien: 100000, MaDon: b.Ma},
	} {
		if err := LuuKhoan(k); err != nil {
			t.Fatal(err)
		}
	}

	s := SoLieuCuaKhach(ma)
	if s.SoDon != 2 {
		t.Errorf("SoDon = %d, mong 2", s.SoDon)
	}
	if s.TongChi != 600000 {
		t.Errorf("TongChi = %d, mong 600000", s.TongChi)
	}
	if s.ConNo != 200000 {
		t.Errorf("ConNo = %d, mong 200000", s.ConNo)
	}
	if s.SoDonDang != 1 {
		t.Errorf("SoDonDang = %d, mong 1", s.SoDonDang)
	}
}

func TestLichSuKhachBatCaDonCu(t *testing.T) {
	moCuaHang(t)
	if err := NapKhach(); err != nil {
		t.Fatal(err)
	}
	ma := GhiNhanKhach("Anh Minh", "0846161368", "", "")
	// Đơn cũ trên server không có MaKhach — vẫn phải hiện trên trang khách.
	cu := &Don{Ma: "TV-2608-001", Ngay: "2026-08-01", TrangThai: TTDaGiao,
		KhachTen: "Anh Minh", KhachLienHe: "084.616.1368", TongTien: 200000}
	if err := LuuDon(cu); err != nil {
		t.Fatal(err)
	}
	ds := LichSuKhach(ma)
	if len(ds) != 1 || ds[0].Ma != "TV-2608-001" {
		t.Fatalf("đơn cũ không có MaKhach phải khớp qua số điện thoại, nhận %d đơn", len(ds))
	}
}

func TestNhapKhachCSV(t *testing.T) {
	gocTam(t)
	khachDs = nil

	// Hồ sơ có sẵn: dòng thứ hai trong tệp trùng số này nên phải cập nhật
	// chứ không đẻ thêm một dòng nữa.
	GhiNhanKhach("Anh Ba", "0900000011", "", "")

	tep := "Tên,Điện thoại,Email,Địa chỉ,Nhóm,Ghi chú\n" +
		"Chị Tư,0900000022,tu@vd.vn,Quận 1,clb,thích căng 24 cân\n" +
		"Anh Ba,0900 000 011,,,than,\n" +
		"Người không số,,,,,\n"

	kq, err := NhapKhachCSV(strings.NewReader(tep))
	if err != nil {
		t.Fatalf("nhập CSV lỗi: %v", err)
	}
	if kq.ThemMoi != 1 || kq.CapNhat != 1 || len(kq.BoQua) != 1 {
		t.Fatalf("mong thêm 1 / cập nhật 1 / bỏ qua 1, nhận %d / %d / %d",
			kq.ThemMoi, kq.CapNhat, len(kq.BoQua))
	}
	if len(DanhSachKhach()) != 2 {
		t.Fatalf("trùng số phải gộp, sổ đang có %d hồ sơ", len(DanhSachKhach()))
	}

	tu, co := KhachTheoSo("0900000022")
	if !co || tu.Ten != "Chị Tư" || tu.Nhom != NhomKhachClb || tu.GhiChu == "" {
		t.Fatalf("hàng mới vào thiếu dữ liệu: %+v", tu)
	}
	ba, _ := KhachTheoSo("0900000011")
	if ba.Nhom != NhomKhachThan {
		t.Errorf("nhóm phải được điền vào hồ sơ đang để mặc định, nhận %q", ba.Nhom)
	}

	// Bản cũ phải nằm sẵn bên cạnh: không có nút hoàn tác.
	if _, err := os.ReadFile(fileKhach() + ".truoc-nhap"); err != nil {
		t.Errorf("thiếu bản sao lưu trước khi nhập: %v", err)
	}
}

// Bước 17: trang chi tiết đơn phải nói được đơn này thuộc hồ sơ nào, và đơn
// cũ chưa nối thì mọc ra nút gắn chứ không im lặng.
func TestTrangDonNoiVoiHoSoKhach(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	khachDs = nil

	ma := GhiNhanKhach("Chị Năm", "0900000055", "", "")
	coHoSo := donThu(t, &Don{Ma: "TV-2609-910", KhachTen: "Chị Năm",
		KhachLienHe: "0900000055", MaKhach: ma, TongTien: 200000})
	chuaNoi := donThu(t, &Don{Ma: "TV-2609-911", KhachTen: "Anh Sáu",
		KhachLienHe: "0900000066", TongTien: 100000})

	s := moTrang(t, mux, ck, "/qt/don/"+coHoSo.Ma)
	if !strings.Contains(s, `href="/qt/khach/`+ma+`"`) {
		t.Error("đơn đã nối mà không có đường sang hồ sơ khách")
	}
	if strings.Contains(s, "Gắn vào hồ sơ khách") {
		t.Error("đơn đã nối vẫn hiện nút gắn")
	}

	s = moTrang(t, mux, ck, "/qt/don/"+chuaNoi.Ma)
	if !strings.Contains(s, "Gắn vào hồ sơ khách") {
		t.Error("đơn chưa nối thiếu nút gắn")
	}
}

// Bước 12: form lập đơn phải mang theo kho gợi ý, và vào bằng ?khach= thì ô
// khách đã điền sẵn.
func TestFormDonMoiGoiYKhachQuen(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	khachDs = nil
	ma := GhiNhanKhach("Chị Bảy", "0900000077", "bay@vd.vn", "")

	s := moTrang(t, mux, ck, "/qt/don-moi")
	if !strings.Contains(s, `id="ds-khach"`) || !strings.Contains(s, "0900000077") {
		t.Error("form lập đơn thiếu kho gợi ý khách quen")
	}

	s = moTrang(t, mux, ck, "/qt/don-moi?khach="+ma)
	if !strings.Contains(s, `value="Chị Bảy"`) || !strings.Contains(s, `value="bay@vd.vn"`) {
		t.Error("vào từ hồ sơ khách mà ô khách không điền sẵn")
	}
}

// --- Việc 1b: khách cũ quay lại tự giảm ------------------------------

// Bước 21. Mức giảm gắn với nhóm, không phải một luật riêng — đổi con số ở
// /qt/nguong là xong, không phải sửa code.
func TestMucGiamTheoNhom(t *testing.T) {
	gocTam(t)
	khachDs = nil
	if err := NapNguongTram(); err != nil {
		t.Fatal(err)
	}

	ma := GhiNhanKhach("Anh Tám", "0900000088", "", "")
	if g := MucGiamCuaKhach(ma); g != 0 {
		t.Errorf("khách lẻ phải là 0%%, nhận %d", g)
	}
	if err := DatNhomKhach(ma, NhomKhachThan); err != nil {
		t.Fatal(err)
	}
	if g := MucGiamCuaKhach(ma); g != 10 {
		t.Errorf("khách quen phải là 10%%, nhận %d", g)
	}
	if err := DatNhomKhach(ma, NhomKhachClb); err != nil {
		t.Fatal(err)
	}
	if g := MucGiamCuaKhach(ma); g != 15 {
		t.Errorf("CLB phải là 15%%, nhận %d", g)
	}
	if g := MucGiamCuaKhach(""); g != 0 {
		t.Errorf("đơn chưa gắn hồ sơ phải là 0%%, nhận %d", g)
	}
}

// Bước 23. Thăng nhóm là chuyện một chiều.
func TestThangNhomSauKhiGiao(t *testing.T) {
	gocTam(t)
	khachDs = nil
	if err := NapNguongTram(); err != nil {
		t.Fatal(err)
	}

	nhomCua := func(ma string) string {
		k, _ := KhachTheoMa(ma)
		return k.Nhom
	}

	giao := GhiNhanKhach("Khách giao", "0900000101", "", "")
	ThangNhomSauGiao(&Don{MaKhach: giao, TrangThai: TTDaGiao})
	if nhomCua(giao) != NhomKhachThan {
		t.Errorf("giao xong phải lên khách quen, nhận %q", nhomCua(giao))
	}

	huy := GhiNhanKhach("Khách huỷ", "0900000102", "", "")
	ThangNhomSauGiao(&Don{MaKhach: huy, TrangThai: TTHuy})
	if nhomCua(huy) != NhomKhachLe {
		t.Errorf("đơn huỷ không phải một lần ghé, nhận %q", nhomCua(huy))
	}

	bh := GhiNhanKhach("Khách bảo hành", "0900000103", "", "")
	ThangNhomSauGiao(&Don{MaKhach: bh, TrangThai: TTDaGiao, MaDonGoc: "TV-2608-001"})
	if nhomCua(bh) != NhomKhachLe {
		t.Errorf("đơn bảo hành không tính là lần ghé mới, nhận %q", nhomCua(bh))
	}

	clb := GhiNhanKhach("Đội CLB", "0900000104", "", "")
	if err := DatNhomKhach(clb, NhomKhachClb); err != nil {
		t.Fatal(err)
	}
	ThangNhomSauGiao(&Don{MaKhach: clb, TrangThai: TTDaGiao})
	if nhomCua(clb) != NhomKhachClb {
		t.Errorf("không bao giờ được kéo tụt nhóm, nhận %q", nhomCua(clb))
	}
}

// Bước 25. Test quan trọng nhất của Việc 1b: giảm 10%% trên đơn gửi tiệm
// ngoài là ăn vào đúng phần công của trạm.
func TestGoiYGiamGiaBiTranBienGop(t *testing.T) {
	gocTam(t)
	khachDs = nil
	if err := NapNguongTram(); err != nil {
		t.Fatal(err)
	}
	cu := GIA.Nguong.BienGopToiThieuDong
	t.Cleanup(func() { GIA.Nguong.BienGopToiThieuDong = cu })
	GIA.Nguong.BienGopToiThieuDong = 80000

	ma := GhiNhanKhach("Chị Chín", "0900000109", "", "")
	if err := DatNhomKhach(ma, NhomKhachThan); err != nil {
		t.Fatal(err)
	}
	dong := []DongTien{{Ten: "Thay đế", SoTien: 500000}}

	guiDi := &Don{MaKhach: ma, DongTien: dong, TongTien: 500000}
	guiDi.GuiDi.TraDoiTac = 400000
	so, ly, daHa := GoiYGiamGia(guiDi)
	if so != 20000 || !daHa {
		t.Errorf("đơn gửi tiệm: mong 20000/hạ, nhận %d/%v", so, daHa)
	}
	if ly == "" {
		t.Error("thiếu lý do điền sẵn")
	}

	tuLam := &Don{MaKhach: ma, DongTien: dong, TongTien: 500000}
	so, _, daHa = GoiYGiamGia(tuLam)
	if so != 50000 || daHa {
		t.Errorf("đơn trạm tự làm: mong 50000/không hạ, nhận %d/%v", so, daHa)
	}

	baoHanh := &Don{MaKhach: ma, DongTien: dong, TongTien: 500000, MaDonGoc: "TV-2608-001"}
	if so, _, _ = GoiYGiamGia(baoHanh); so != 0 {
		t.Errorf("đơn bảo hành không giảm, nhận %d", so)
	}
}

// Bước 27: điền sẵn phải hiện ra màn, và khi bị trần biên gộp kéo xuống thì
// phải nói ra chứ không lặng lẽ sửa số.
func TestTrangDonDienSanGiamGia(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	khachDs = nil
	if err := NapNguongTram(); err != nil {
		t.Fatal(err)
	}
	cu := GIA.Nguong.BienGopToiThieuDong
	t.Cleanup(func() { GIA.Nguong.BienGopToiThieuDong = cu })
	GIA.Nguong.BienGopToiThieuDong = 80000

	ma := GhiNhanKhach("Chị Mười", "0900000110", "", "")
	if err := DatNhomKhach(ma, NhomKhachThan); err != nil {
		t.Fatal(err)
	}
	dong := []DongTien{{Ten: "Quấn cán", SoTien: 500000}}

	tuLam := donThu(t, &Don{Ma: "TV-2609-920", MaKhach: ma, KhachTen: "Chị Mười",
		KhachLienHe: "0900000110", DongTien: dong, TongTien: 500000})
	s := moTrang(t, mux, ck, "/qt/don/"+tuLam.Ma)
	if !strings.Contains(s, `name="giam_gia" inputmode="numeric" class="giam" value="50000"`) {
		t.Error("ô giảm giá không được điền sẵn 50000")
	}
	if strings.Contains(s, "Đã hạ từ") {
		t.Error("đơn không gửi tiệm mà vẫn báo bị hạ")
	}

	guiDi := &Don{Ma: "TV-2609-921", MaKhach: ma, KhachTen: "Chị Mười",
		KhachLienHe: "0900000110", DongTien: dong, TongTien: 500000}
	guiDi.GuiDi.TraDoiTac = 400000
	donThu(t, guiDi)
	s = moTrang(t, mux, ck, "/qt/don/"+guiDi.Ma)
	if !strings.Contains(s, `class="giam" value="20000"`) {
		t.Error("trần biên gộp không hạ mức giảm xuống 20000")
	}
	if !strings.Contains(s, "Đã hạ từ") {
		t.Error("hạ mức giảm mà không nói cho thợ biết vì sao")
	}
}
