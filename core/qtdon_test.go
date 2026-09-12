package core

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestNhanHangChuyenSangMoi(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-888", Token: "tok12888", TrangThai: TTChoHangVe, KhachTen: "Chị Lan"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/qt/don/TV-2609-888/nhan-hang", url.Values{"ma_van_don": {"GHTK123456"}})
	r.SetPathValue("ma", "TV-2609-888")
	hQtNhanHang(httptest.NewRecorder(), r)

	sau, co := LayDon("TV-2609-888")
	if !co {
		t.Fatal("mất đơn")
	}
	if sau.TrangThai != TTMoi {
		t.Fatalf("nhận hàng xong phải sang %s, đang ở %s", TTMoi, sau.TrangThai)
	}
	if sau.MaVanDonDen != "GHTK123456" {
		t.Fatalf("chưa ghi mã vận đơn: %q", sau.MaVanDonDen)
	}
	if len(sau.LichSu) == 0 {
		t.Fatal("phải để lại một mốc trong lịch sử")
	}
}

// Bấm nhầm trên đơn đang sửa không được kéo nó ngược về "mới nhận".
func TestNhanHangChiChayOChoHangVe(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-889", TrangThai: TTDangSua, KhachTen: "Anh Nam"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/qt/don/TV-2609-889/nhan-hang", url.Values{})
	r.SetPathValue("ma", "TV-2609-889")
	hQtNhanHang(httptest.NewRecorder(), r)

	sau, _ := LayDon("TV-2609-889")
	if sau.TrangThai != TTDangSua {
		t.Fatalf("trạng thái bị kéo về %s", sau.TrangThai)
	}
}

// Lọc "sửa xong, chưa thu đủ" đọc SỔ TIỀN qua ConNo(), không đọc trường nào
// trên đơn.
func TestLocChuaThuDu(t *testing.T) {
	moCuaHang(t)
	if err := LuuDon(&Don{Ma: "TV-2609-890", TrangThai: TTXong, TongTien: 250000}); err != nil {
		t.Fatal(err)
	}
	if err := LuuDon(&Don{Ma: "TV-2609-891", TrangThai: TTXong, TongTien: 0}); err != nil {
		t.Fatal(err)
	}

	ds := LocDon(BoLoc{TrangThai: TTXong, ChuaThuDu: true})
	if len(ds) != 1 || ds[0].Ma != "TV-2609-890" {
		t.Fatalf("lọc sai: %v", ds)
	}
}

func TestKhachBaoVanDon(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-892", Token: "tok12892", TrangThai: TTChoHangVe}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/tra-cuu/tok12892/van-don", url.Values{"ma_van_don": {"GHTK999"}})
	r.SetPathValue("token", "tok12892")
	hKhachBaoVanDon(httptest.NewRecorder(), r)

	sau, _ := LayDon("TV-2609-892")
	if sau.MaVanDonDen != "GHTK999" {
		t.Fatalf("chưa ghi vận đơn khách báo: %q", sau.MaVanDonDen)
	}
}

// Đơn đã qua cho_hang_ve thì ô này đóng — không để người cầm link ghi đè.
func TestKhachKhongSuaVanDonSauKhiNhan(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-893", Token: "tok12893", TrangThai: TTDangSua, MaVanDonDen: "GHTK111"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/tra-cuu/tok12893/van-don", url.Values{"ma_van_don": {"GHTK222"}})
	r.SetPathValue("token", "tok12893")
	hKhachBaoVanDon(httptest.NewRecorder(), r)

	sau, _ := LayDon("TV-2609-893")
	if sau.MaVanDonDen != "GHTK111" {
		t.Fatalf("bị ghi đè thành %q", sau.MaVanDonDen)
	}
}

// Thợ gửi đồ về: ghi mã, KHÔNG đổi trạng thái.
func TestGhiVanDonVeKhongDoiTrangThai(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-894", TrangThai: TTXong, KhachDiaChi: "12 Lê Lợi, Q1"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/qt/don/TV-2609-894/van-don-ve", url.Values{"ma_van_don_ve": {"VTP777"}})
	r.SetPathValue("ma", "TV-2609-894")
	hQtVanDonVe(httptest.NewRecorder(), r)

	sau, _ := LayDon("TV-2609-894")
	if sau.MaVanDonVe != "VTP777" {
		t.Fatalf("chưa ghi vận đơn về: %q", sau.MaVanDonVe)
	}
	if sau.TrangThai != TTXong {
		t.Fatalf("không được đổi trạng thái, đang ở %s", sau.TrangThai)
	}
}
