package core

import (
	"strings"
	"testing"
	"time"
)

// capNhatCachDay đặt lại dấu thời gian "sửa lần cuối" vì LuuDon luôn ghi đè
// bằng giờ hiện tại — không có cách nào tạo một đơn cũ ngoài việc sửa sau.
func capNhatCachDay(d *Don, ngay int) {
	d.CapNhat = time.Now().AddDate(0, 0, -ngay).Format("2006-01-02 15:04:05")
}

// khoDonTam: gốc tạm + kho đơn nạp lại từ đó. Riêng cái thứ hai mới là điều
// quan trọng — donDs là biến gói, không nạp lại thì đơn của test trước còn
// nằm nguyên trong bộ nhớ và mọi phép đếm đều lệch.
func khoDonTam(t *testing.T) {
	t.Helper()
	gocTam(t)
	if err := NapDon(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = NapDon() })
}

func TestXongBoQuen(t *testing.T) {
	bayGio := time.Now()

	cu := &Don{Ma: "TV-2609-960", TrangThai: TTXong}
	capNhatCachDay(cu, 10)
	if !cu.XongBoQuen(bayGio, 7) {
		t.Error("đơn xong 10 ngày trước, ngưỡng 7 — phải là bỏ quên")
	}

	moi := &Don{Ma: "TV-2609-961", TrangThai: TTXong}
	capNhatCachDay(moi, 2)
	if moi.XongBoQuen(bayGio, 7) {
		t.Error("đơn xong 2 ngày trước không phải bỏ quên")
	}

	daGiao := &Don{Ma: "TV-2609-962", TrangThai: TTDaGiao}
	capNhatCachDay(daGiao, 30)
	if daGiao.XongBoQuen(bayGio, 7) {
		t.Error("đơn đã giao thì không còn nằm trên kệ")
	}

	// Ngưỡng 0 tắt hẳn rổ này chứ không phải kêu mọi đơn.
	if cu.XongBoQuen(bayGio, 0) {
		t.Error("ngưỡng 0 phải tắt rổ bỏ quên")
	}
}

func TestTinNhacCoMucBoQuen(t *testing.T) {
	khoDonTam(t)
	homNay := time.Now().Format("2006-01-02")
	donThu(t, &Don{Ma: "TV-2609-963", KhachTen: "Anh Bảy", TrangThai: TTDangSua,
		HenTraNgay: homNay, VotHang: "Yonex"})
	quen := donThu(t, &Don{Ma: "TV-2609-964", KhachTen: "Chị Tám", TrangThai: TTXong,
		VotHang: "Lining"})
	capNhatCachDay(quen, 12)

	sk, n := dungTinHenTra(time.Now())
	if n != 2 {
		t.Fatalf("tin kể %d đơn, muốn 2", n)
	}
	if !strings.Contains(sk.Than, quen.Ma) {
		t.Error("tin nhắc thiếu đơn xong bị bỏ quên")
	}
	if !strings.Contains(sk.TieuDe, "chưa ai lấy") {
		t.Errorf("tiêu đề không nhắc đơn bỏ quên: %q", sk.TieuDe)
	}
}

func TestTinNhacKhongCoGiThiKhongGui(t *testing.T) {
	khoDonTam(t)
	donThu(t, &Don{Ma: "TV-2609-965", TrangThai: TTDangSua, VotHang: "Yonex"})
	if _, n := dungTinHenTra(time.Now()); n != 0 {
		t.Errorf("không có đơn tới hẹn mà vẫn kể %d đơn", n)
	}
}

func TestLocXongBoQuen(t *testing.T) {
	khoDonTam(t)
	quen := donThu(t, &Don{Ma: "TV-2609-966", TrangThai: TTXong, VotHang: "Yonex"})
	capNhatCachDay(quen, 20)
	donThu(t, &Don{Ma: "TV-2609-967", TrangThai: TTXong, VotHang: "Lining"})

	ds := LocDon(BoLoc{ChiXongBoQuen: true})
	if len(ds) != 1 || ds[0].Ma != quen.Ma {
		t.Fatalf("rổ bỏ quên trả %d đơn, muốn đúng %s", len(ds), quen.Ma)
	}
}

func TestNguongBoQuenSuaDuoc(t *testing.T) {
	gocTam(t)
	if NguongTramHienTai().XongBoQuenNgay != 7 {
		t.Fatalf("mặc định phải là 7, đang là %d", NguongTramHienTai().XongBoQuenNgay)
	}
	if _, err := SuaNguongTram(map[string]string{"xong_bo_quen_ngay": "14"}); err != nil {
		t.Fatal(err)
	}
	if NguongTramHienTai().XongBoQuenNgay != 14 {
		t.Error("sửa ngưỡng bỏ quên không ăn")
	}
	if _, err := SuaNguongTram(map[string]string{"giam_khach_quen_phan_tram": "150"}); err == nil {
		t.Error("giảm giá 150% phải bị chặn")
	}
}
