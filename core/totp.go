package core

// TOTP theo RFC 6238, tự cài bằng thư viện chuẩn.
//
// Vì sao không kéo thư viện OTP: cả app đang zero-dependency, mà phần cần
// dùng chỉ là HMAC-SHA1 rồi cắt 4 byte — dưới 80 dòng. Thư viện đầy đủ kéo
// theo QR encoder, nhiều thuật toán băm, nhiều độ dài mã, không dùng tới.
//
// Không vẽ mã QR: tự viết QR encoder là việc lớn cho một hai người dùng.
// Hiện chuỗi base32 để dán tay vào Google Authenticator, kèm URI otpauth://
// cho ai muốn tự dựng QR.

import (
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

const buocTOTP = 30 // giây, theo mặc định của Google Authenticator

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

func maTOTP(biMat []byte, buoc int64) string {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(buoc))

	m := hmac.New(sha1.New, biMat)
	m.Write(b[:])
	tong := m.Sum(nil)

	// Cắt động theo RFC 4226 mục 5.3: 4 bit cuối chỉ ra vị trí bắt đầu.
	vt := tong[len(tong)-1] & 0x0f
	so := (uint32(tong[vt])&0x7f)<<24 |
		uint32(tong[vt+1])<<16 |
		uint32(tong[vt+2])<<8 |
		uint32(tong[vt+3])
	return fmt.Sprintf("%06d", so%1000000)
}

// kiemTOTP trả về bước thời gian đã khớp, để chỗ gọi nhớ mà từ chối dùng lại.
// Cửa sổ ±1 bước: đồng hồ điện thoại lệch vài giây là chuyện thường.
func kiemTOTP(biMatB32, ma string, luc time.Time) (int64, bool) {
	ma = strings.TrimSpace(ma)
	if len(ma) != 6 {
		return 0, false
	}
	for _, c := range ma {
		if c < '0' || c > '9' {
			return 0, false
		}
	}

	biMat, err := b32.DecodeString(strings.ToUpper(strings.ReplaceAll(biMatB32, " ", "")))
	if err != nil || len(biMat) == 0 {
		return 0, false
	}

	hienTai := luc.Unix() / buocTOTP
	for d := int64(-1); d <= 1; d++ {
		if hmac.Equal([]byte(maTOTP(biMat, hienTai+d)), []byte(ma)) {
			return hienTai + d, true
		}
	}
	return 0, false
}

func sinhBiMatTOTP() string {
	// 20 byte = 160 bit, đúng độ dài RFC 4226 khuyến nghị cho SHA-1.
	return b32.EncodeToString(byteNgauNhien(20))
}

// byteNgauNhien: maNgauNhien trả chuỗi hex; ở đây cần byte thô.
func byteNgauNhien(n int) []byte {
	b, err := hex.DecodeString(maNgauNhien(n))
	if err != nil {
		// Chuỗi do maNgauNhien sinh ra, sai là lỗi lập trình.
		panic("hex từ maNgauNhien không giải mã được: " + err.Error())
	}
	return b
}

func uriOTPAuth(taiKhoan, biMatB32 string) string {
	pht := strings.TrimSpace(CFG.ThuongHieu.Ten)
	if pht == "" {
		pht = "TramVot"
	}
	return "otpauth://totp/" + url.PathEscape(pht+":"+taiKhoan) +
		"?secret=" + biMatB32 +
		"&issuer=" + url.QueryEscape(pht) +
		"&algorithm=SHA1&digits=6&period=30"
}

// Chống phát lại: TOTP sống 30 giây, kẻ đọc trộm mã trong khoảng đó dùng lại
// được. Nhớ bước vừa dùng của từng tài khoản là chặn được.
var (
	buocDaDungMu sync.Mutex
	buocDaDung   = map[string]int64{}
)

func daDungBuoc(ten string, buoc int64) bool {
	buocDaDungMu.Lock()
	defer buocDaDungMu.Unlock()
	return buocDaDung[strings.ToLower(ten)] == buoc
}

func nhoBuoc(ten string, buoc int64) {
	buocDaDungMu.Lock()
	defer buocDaDungMu.Unlock()
	buocDaDung[strings.ToLower(ten)] = buoc
}
