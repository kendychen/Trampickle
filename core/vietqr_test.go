package core

import (
	"strings"
	"testing"
)

// Nội dung chuyển khoản phải khớp được bằng CHÍNH cái regex saoke.go dùng.
// Đây là toàn bộ lý do QR tồn tại: tiền về là khớp được đơn.
func TestNoiDungCKKhopDuocBangReMaDon(t *testing.T) {
	for _, ma := range []string{"TV-2609-001", "TV-2612-045"} {
		nd := NoiDungCK(ma)
		if !reMaDon().MatchString(nd) {
			t.Fatalf("nội dung %q không khớp reMaDon — sao kê sẽ không nhận ra đơn %s", nd, ma)
		}
	}
}

// Ngân hàng nuốt dấu, viết hoa, chèn thêm chữ — nội dung vẫn phải khớp.
func TestNoiDungCKKhopKhiBiNganHangLamBien(t *testing.T) {
	nd := NoiDungCK("TV-2609-001")
	for _, bien := range []string{
		nd,
		strings.ReplaceAll(nd, "-", " "),
		strings.ReplaceAll(nd, "-", ""),
		"CK tu 0900000000 " + nd,
		"CHUYEN TIEN " + nd + " GD 123456",
	} {
		if !reMaDon().MatchString(bien) {
			t.Fatalf("không khớp: %q", bien)
		}
	}
}

// Vector chuẩn của CRC-16/CCITT-FALSE. Sai hàm này thì app ngân hàng báo "mã
// QR không hợp lệ" và không ai biết vì sao.
func TestCRC16CCITT(t *testing.T) {
	if got := crc16CCITT("123456789"); got != "29B1" {
		t.Fatalf(`crc16CCITT("123456789") = %q, chờ "29B1"`, got)
	}
}

func TestChuoiVietQRCoDuBaThu(t *testing.T) {
	nh := NganHang{Ma: "970436", SoTK: "1234567890", ChuTK: "NGUYEN VAN A"}
	s := ChuoiVietQR(nh, 250000, NoiDungCK("TV-2609-001"))
	if s == "" {
		t.Fatal("chuỗi QR rỗng")
	}
	for _, phai := range []string{"970436", "1234567890", "TV-2609-001", "704"} {
		if !strings.Contains(s, phai) {
			t.Fatalf("chuỗi QR thiếu %q:\n%s", phai, s)
		}
	}
	// 4 ký tự cuối là CRC của toàn chuỗi kể cả "6304".
	if len(s) < 8 || s[len(s)-8:len(s)-4] != "6304" {
		t.Fatalf("chuỗi QR không kết bằng 6304+CRC: %q", s[len(s)-8:])
	}
	if crc16CCITT(s[:len(s)-4]) != s[len(s)-4:] {
		t.Fatal("CRC cuối chuỗi không khớp phần đầu")
	}
}

func TestThieuTaiKhoanThiKhongCoQR(t *testing.T) {
	if s := ChuoiVietQR(NganHang{}, 250000, "TV-2609-001"); s != "" {
		t.Fatalf("thiếu tài khoản mà vẫn sinh QR: %q", s)
	}
}
