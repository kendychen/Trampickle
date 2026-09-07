package core

import (
	"strings"
	"testing"
	"time"
)

// Bí mật mẫu của RFC 6238 phụ lục B: chuỗi ASCII "12345678901234567890"
// viết dưới dạng base32.
const biMatRFC = "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ"

// TestTOTPTheoRFC6238 — nếu bài này hỏng thì mã sinh ra không khớp với
// Google Authenticator, và không ai đăng nhập được nữa.
func TestTOTPTheoRFC6238(t *testing.T) {
	// RFC in mã 8 chữ số; bản 6 chữ số là 6 số cuối.
	bang := []struct {
		giay int64
		ma   string
	}{
		{59, "287082"},
		{1111111109, "081804"},
		{1111111111, "050471"},
		{1234567890, "005924"},
		{2000000000, "279037"},
		{20000000000, "353130"},
	}
	biMat := []byte("12345678901234567890")
	for _, c := range bang {
		got := maTOTP(biMat, c.giay/buocTOTP)
		if got != c.ma {
			t.Errorf("lúc %d: được %s, đợi %s", c.giay, got, c.ma)
		}
	}
}

func TestKiemTOTPNhanMaDungVaTraBuoc(t *testing.T) {
	luc := time.Unix(1111111111, 0)
	buoc, ok := kiemTOTP(biMatRFC, "050471", luc)
	if !ok {
		t.Fatal("mã đúng mà bị từ chối")
	}
	if buoc != 1111111111/buocTOTP {
		t.Errorf("bước trả về sai: %d", buoc)
	}
}

// Đồng hồ điện thoại lệch vài giây là chuyện thường; cửa sổ ±1 bước phải
// nhận được mã của bước trước và bước sau.
func TestKiemTOTPChoLechMotBuoc(t *testing.T) {
	goc := time.Unix(1111111111, 0)
	for _, d := range []time.Duration{-buocTOTP * time.Second, 0, buocTOTP * time.Second} {
		if _, ok := kiemTOTP(biMatRFC, "050471", goc.Add(d)); !ok {
			t.Errorf("lệch %v: mã bị từ chối", d)
		}
	}
	// Lệch hai bước thì thôi — quá xa, coi như sai.
	if _, ok := kiemTOTP(biMatRFC, "050471", goc.Add(2*buocTOTP*time.Second)); ok {
		t.Error("lệch hai bước mà vẫn nhận")
	}
}

func TestKiemTOTPTuChoiRac(t *testing.T) {
	luc := time.Unix(1111111111, 0)
	for _, ma := range []string{"", "12345", "1234567", "05047a", "abcdef", "000000"} {
		if _, ok := kiemTOTP(biMatRFC, ma, luc); ok {
			t.Errorf("nhận nhầm mã rác %q", ma)
		}
	}
	// Bí mật hỏng thì từ chối chứ không panic.
	if _, ok := kiemTOTP("khong-phai-base32!!", "050471", luc); ok {
		t.Error("bí mật hỏng mà vẫn nhận")
	}
}

// Mã sống 30 giây. Ai nhìn trộm được màn hình trong khoảng đó dùng lại được,
// nên bước đã dùng phải bị nhớ và từ chối.
func TestKhongDungLaiBuocCu(t *testing.T) {
	buocDaDungMu.Lock()
	cu := buocDaDung
	buocDaDung = map[string]int64{}
	buocDaDungMu.Unlock()
	t.Cleanup(func() {
		buocDaDungMu.Lock()
		buocDaDung = cu
		buocDaDungMu.Unlock()
	})

	if daDungBuoc("kendy", 100) {
		t.Fatal("chưa dùng lần nào mà đã báo đã dùng")
	}
	nhoBuoc("kendy", 100)
	if !daDungBuoc("kendy", 100) {
		t.Error("vừa dùng xong mà không nhớ")
	}
	if daDungBuoc("kendy", 101) {
		t.Error("bước sau bị chặn oan")
	}
	// Tên viết hoa viết thường phải cùng một tài khoản.
	if !daDungBuoc("Kendy", 100) {
		t.Error("KENDY và kendy phải là một")
	}
}

func TestSinhBiMatDungDoDaiVaKhacNhau(t *testing.T) {
	a, b := sinhBiMatTOTP(), sinhBiMatTOTP()
	if a == b {
		t.Fatal("hai lần sinh ra cùng một bí mật")
	}
	// 20 byte base32 không đệm = 32 ký tự.
	if len(a) != 32 {
		t.Errorf("độ dài %d, đợi 32", len(a))
	}
	if _, ok := kiemTOTP(a, maTOTP(byteNgauNhien(0), 0), time.Now()); ok {
		t.Error("mã của bí mật rỗng lại khớp")
	}
}

func TestURIOTPAuthCoDuThamSo(t *testing.T) {
	u := uriOTPAuth("kendy", biMatRFC)
	for _, phai := range []string{"otpauth://totp/", "secret=" + biMatRFC, "issuer=", "digits=6", "period=30"} {
		if !strings.Contains(u, phai) {
			t.Errorf("URI thiếu %q: %s", phai, u)
		}
	}
}
