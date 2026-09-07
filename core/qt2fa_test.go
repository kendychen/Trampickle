package core

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Lấy giá trị một ô ẩn ra khỏi HTML. Test đi qua đúng đường người dùng đi —
// bấm nút, đọc trang, gõ lại — nên phải bóc được thứ trang vừa hiện.
func oAn(t *testing.T, than, ten string) string {
	t.Helper()
	re := regexp.MustCompile(`name="` + ten + `" value="([^"]*)"`)
	m := re.FindStringSubmatch(than)
	if m == nil {
		t.Fatalf("không thấy ô ẩn %q trong trang", ten)
	}
	return m[1]
}

func post2FA(t *testing.T, h http.Handler, ck *http.Cookie, than string) string {
	t.Helper()
	w := postThu(t, h, ck, "/qt/2fa", "_csrf="+tokenTuPhien(ck.Value)+"&"+than)
	if w.Code != http.StatusOK {
		t.Fatalf("POST /qt/2fa: mã %d", w.Code)
	}
	return w.Body.String()
}

func maBayGio(t *testing.T, biMat string) string {
	t.Helper()
	return maTOTP(giaiB32(t, biMat), time.Now().Unix()/buocTOTP)
}

func giaiB32(t *testing.T, s string) []byte {
	t.Helper()
	b, err := b32.DecodeString(strings.ToUpper(s))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// Vòng đời đầy đủ: bắt đầu → bật → tắt bằng mã dự phòng.
func TestBat2FARoiTat(t *testing.T) {
	h, ck := dungHandlerThu(t, VaiTroChu)

	than := post2FA(t, h, ck, "viec=bat-dau")
	biMat := oAn(t, than, "bi_mat")
	if len(biMat) != 32 {
		t.Fatalf("bí mật base32 dài %d, đợi 32", len(biMat))
	}

	// Gõ sai thì không bật, nhưng bí mật phải còn trên trang để gõ lại.
	than = post2FA(t, h, ck, "viec=bat&bi_mat="+biMat+"&ma=000000")
	if nd, _ := TimNguoiDung("kendy"); nd.TotpBat {
		t.Fatal("mã sai mà vẫn bật 2FA")
	}
	if oAn(t, than, "bi_mat") != biMat {
		t.Error("gõ sai xong mất bí mật, người dùng phải quét lại mã QR")
	}

	than = post2FA(t, h, ck, "viec=bat&bi_mat="+biMat+"&ma="+maBayGio(t, biMat))
	nd, _ := TimNguoiDung("kendy")
	if !nd.TotpBat || nd.TotpBiMat != biMat {
		t.Fatal("gõ đúng mã mà không bật được 2FA")
	}
	if len(nd.MaDuPhong) != soMaDuPhong {
		t.Fatalf("có %d mã dự phòng, đợi %d", len(nd.MaDuPhong), soMaDuPhong)
	}
	// Mã dự phòng phải hiện bản rõ đúng một lần, ngay tại đây.
	reMa := regexp.MustCompile(`\b[0-9a-f]{5}-[0-9a-f]{5}\b`)
	roDs := reMa.FindAllString(than, -1)
	if len(roDs) != soMaDuPhong {
		t.Fatalf("trang hiện %d mã dự phòng, đợi %d", len(roDs), soMaDuPhong)
	}
	// Và không được là bản rõ đang nằm trong file tài khoản.
	for _, h := range nd.MaDuPhong {
		if !strings.HasPrefix(h, "$2a$") && !strings.HasPrefix(h, "$2b$") {
			t.Fatalf("mã dự phòng lưu không phải bcrypt: %q", h)
		}
	}

	// Tắt bằng mã bịa: không được tắt.
	post2FA(t, h, ck, "viec=tat&ma=999999")
	if nd, _ := TimNguoiDung("kendy"); !nd.TotpBat {
		t.Fatal("mã sai mà tắt được 2FA")
	}

	// Tắt bằng mã dự phòng thật.
	post2FA(t, h, ck, "viec=tat&ma="+roDs[0])
	nd, _ = TimNguoiDung("kendy")
	if nd.TotpBat || nd.TotpBiMat != "" || len(nd.MaDuPhong) != 0 {
		t.Fatalf("tắt rồi mà còn sót: bat=%v biMat=%q con=%d",
			nd.TotpBat, nd.TotpBiMat, len(nd.MaDuPhong))
	}
}

// Mã dự phòng dùng một lần. Nếu dùng lại được thì nó chỉ là mật khẩu thứ hai
// ngắn hơn, tệ hơn cái nó thay thế.
func TestMaDuPhongChiDungMotLan(t *testing.T) {
	dungKhoTienThu(t)
	ro, hash, err := sinhMaDuPhong()
	if err != nil {
		t.Fatal(err)
	}
	nguoiDungMu.Lock()
	cu := nguoiDung
	nguoiDung = []NguoiDung{{Ten: "kendy", TotpBat: true, MaDuPhong: hash}}
	nguoiDungMu.Unlock()
	t.Cleanup(func() {
		nguoiDungMu.Lock()
		nguoiDung = cu
		nguoiDungMu.Unlock()
	})

	if !DungMaDuPhong("kendy", ro[0]) {
		t.Fatal("mã thật mà không nhận")
	}
	if DungMaDuPhong("kendy", ro[0]) {
		t.Fatal("mã dùng rồi vẫn nhận lần hai")
	}
	if nd, _ := TimNguoiDung("kendy"); len(nd.MaDuPhong) != soMaDuPhong-1 {
		t.Fatalf("còn %d mã, đợi %d", len(nd.MaDuPhong), soMaDuPhong-1)
	}
	// Gõ không gạch nối, gõ hoa — vẫn phải nhận.
	if !DungMaDuPhong("kendy", strings.ToUpper(strings.ReplaceAll(ro[1], "-", ""))) {
		t.Fatal("mã đúng nhưng gõ liền/gõ hoa thì không nhận")
	}
}

// Đăng nhập khi đã bật 2FA: mật khẩu đúng chưa đủ.
func TestDangNhapCanMaTOTP(t *testing.T) {
	h, _ := dungHandlerThu(t, VaiTroChu)
	xoaKhoaThu(t)

	hash, err := bcrypt.GenerateFromPassword([]byte("mat-khau-thu"), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	biMat := sinhBiMatTOTP()
	nguoiDungMu.Lock()
	cu := nguoiDung
	nguoiDung = []NguoiDung{{
		Ten: "kendy", VaiTro: VaiTroChu, MatKhauHash: string(hash),
		TotpBat: true, TotpBiMat: biMat,
	}}
	nguoiDungMu.Unlock()
	t.Cleanup(func() {
		nguoiDungMu.Lock()
		nguoiDung = cu
		nguoiDungMu.Unlock()
	})

	// Khách chưa đăng nhập dùng CSRF kiểu gửi-đôi: cookie tv_csrf phải khớp
	// ô _csrf. Lớp bọc tự đặt cookie ở lần đầu, ở đây đặt sẵn cho gọn.
	ckCSRF := &http.Cookie{Name: tenCookieCSRF, Value: maNgauNhien(32)}
	goiCSRF := func(them string) (int, string) {
		w := postThu(t, h, ckCSRF,
			"/dang-nhap", "_csrf="+ckCSRF.Value+"&ten=kendy&mat_khau=mat-khau-thu"+them)
		return w.Code, w.Body.String()
	}

	ma, than := goiCSRF("")
	if ma != http.StatusOK || !strings.Contains(than, `name="ma_totp"`) {
		t.Fatalf("mật khẩu đúng, chưa có mã: mã %d, trang không hỏi mã 2FA", ma)
	}
	if ma, _ := goiCSRF("&ma_totp=000000"); ma != http.StatusUnauthorized {
		t.Fatalf("mã sai: mã %d, đợi 401", ma)
	}

	dung := maBayGio(t, biMat)
	ma, _ = goiCSRF("&ma_totp=" + dung)
	if ma != http.StatusSeeOther {
		t.Fatalf("mã đúng: mã %d, đợi 303", ma)
	}

	// Phát lại đúng mã đó: phải bị từ chối.
	if ma, _ := goiCSRF("&ma_totp=" + dung); ma != http.StatusUnauthorized {
		t.Fatalf("phát lại mã cũ: mã %d, đợi 401", ma)
	}
}

// Bộ đếm khoá đăng nhập là biến toàn cục, test trước để lại rác thì test sau
// bị khoá oan.
func xoaKhoaThu(t *testing.T) {
	t.Helper()
	don := func() {
		khoaMu.Lock()
		khoaIP, khoaTen, ipQuen = map[string]*demSai{}, map[string]*demSai{}, map[string]map[string]bool{}
		khoaMu.Unlock()
		buocDaDungMu.Lock()
		buocDaDung = map[string]int64{}
		buocDaDungMu.Unlock()
	}
	don()
	t.Cleanup(don)
}
