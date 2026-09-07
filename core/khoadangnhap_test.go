package core

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func dungKhoaThu(t *testing.T) {
	t.Helper()
	xoa := func() {
		khoaMu.Lock()
		khoaIP = map[string]*demSai{}
		khoaTen = map[string]*demSai{}
		ipQuen = map[string]map[string]bool{}
		khoaMu.Unlock()
	}
	xoa()
	t.Cleanup(xoa)
}

func TestKhoaSauNamLanSai(t *testing.T) {
	dungKhoaThu(t)
	for i := 0; i < 5; i++ {
		if ok, _ := choPhepThu("10.0.0.1", "kendy"); !ok {
			t.Fatalf("lần thử %d đã bị chặn, không được chặn trước lần 6", i+1)
		}
		ghiThuSai("10.0.0.1", "kendy")
	}
	ok, con := choPhepThu("10.0.0.1", "kendy")
	if ok {
		t.Fatal("sai 5 lần rồi mà vẫn cho thử tiếp")
	}
	if con < time.Minute {
		t.Fatalf("thời gian chờ quá ngắn: %v", con)
	}
}

func TestDangNhapDungThiXoaDem(t *testing.T) {
	dungKhoaThu(t)
	for i := 0; i < 3; i++ {
		ghiThuSai("10.0.0.2", "kendy")
	}
	ghiThuDung("10.0.0.2", "kendy")
	for i := 0; i < 5; i++ {
		if ok, _ := choPhepThu("10.0.0.2", "kendy"); !ok {
			t.Fatalf("đếm chưa được xoá sau khi đăng nhập đúng (lần %d)", i+1)
		}
		ghiThuSai("10.0.0.2", "kendy")
	}
}

// Kẻ xấu cố tình gõ sai để khoá tài khoản Kendy. IP Kendy đã từng đăng nhập
// thành công thì vẫn phải vào được.
func TestKhongDoSKhiIPDaQuen(t *testing.T) {
	dungKhoaThu(t)
	ghiThuDung("10.0.0.3", "kendy")

	for i := 0; i < 15; i++ {
		ghiThuSai("10.9.9.9", "kendy")
	}
	if ok, _ := choPhepThu("10.9.9.9", "kendy"); ok {
		t.Fatal("IP đang dò phải bị chặn")
	}
	if ok, con := choPhepThu("10.0.0.3", "kendy"); !ok {
		t.Fatalf("IP quen bị chặn oan, còn chờ %v", con)
	}
}

func TestKhoaTangDan(t *testing.T) {
	dungKhoaThu(t)
	for i := 0; i < 5; i++ {
		ghiThuSai("10.0.0.4", "kendy")
	}
	_, lan1 := choPhepThu("10.0.0.4", "kendy")

	// Giả lập hết hạn khoá lần một rồi lại sai đủ 5 lần.
	khoaMu.Lock()
	khoaIP["10.0.0.4"].moLuc = time.Now().Add(-time.Second)
	khoaMu.Unlock()
	for i := 0; i < 5; i++ {
		ghiThuSai("10.0.0.4", "kendy")
	}
	_, lan2 := choPhepThu("10.0.0.4", "kendy")

	if lan2 <= lan1 {
		t.Fatalf("khoá lần hai (%v) phải dài hơn lần một (%v)", lan2, lan1)
	}
}

func TestHTTPDangNhapBiKhoa(t *testing.T) {
	dungKhoaThu(t)
	if err := InitTemplates(); err != nil {
		t.Fatal(err)
	}
	mux := NewMux(true)

	goiSai := func() int {
		f := url.Values{"ten": {"kendy"}, "mat_khau": {"sai-bét-nhè"}}
		r := httptest.NewRequest("POST", "/dang-nhap", strings.NewReader(f.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.RemoteAddr = "10.0.0.77:1234"
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w.Code
	}

	for i := 0; i < 5; i++ {
		if got := goiSai(); got != http.StatusUnauthorized {
			t.Fatalf("lần %d trả %d, mong 401", i+1, got)
		}
	}
	if got := goiSai(); got != http.StatusTooManyRequests {
		t.Fatalf("lần 6 trả %d, mong 429", got)
	}
}

// Bộ đếm theo IP không được phình vô hạn: một lượt quét phân tán mang tới
// hàng vạn IP, mỗi IP một mục không xoá là rò bộ nhớ.
func TestDonMucKhoaDaNguoi(t *testing.T) {
	khoaMu.Lock()
	cuIP, cuTen := khoaIP, khoaTen
	khoaIP, khoaTen = map[string]*demSai{}, map[string]*demSai{}
	lau := time.Now().Add(-5 * time.Hour)
	for i := 0; i < khoaToiDa+10; i++ {
		khoaIP[fmt.Sprintf("10.0.%d.%d", i/256, i%256)] = &demSai{sai: 1, chamLuc: lau}
	}
	// Một mục ĐANG khoá, dù cũng nguội: phải sống sót, đó mới là mục có việc.
	khoaIP["203.0.113.7"] = &demSai{chamLuc: lau, moLuc: time.Now().Add(time.Hour)}
	khoaMu.Unlock()
	t.Cleanup(func() {
		khoaMu.Lock()
		khoaIP, khoaTen = cuIP, cuTen
		khoaMu.Unlock()
	})

	ghiThuSai("198.51.100.1", "kendy")

	khoaMu.Lock()
	defer khoaMu.Unlock()
	if len(khoaIP) > 10 {
		t.Fatalf("còn %d mục sau khi dọn, đợi vài mục", len(khoaIP))
	}
	if khoaIP["203.0.113.7"] == nil {
		t.Error("dọn mất mục đang khoá")
	}
	if khoaIP["198.51.100.1"] == nil {
		t.Error("dọn mất mục vừa ghi")
	}
}
