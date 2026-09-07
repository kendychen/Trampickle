package core

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func reqTu(remote string, header map[string]string) *http.Request {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = remote
	for k, v := range header {
		r.Header.Set(k, v)
	}
	return r
}

func TestIPCuaTinCloudflare(t *testing.T) {
	// 172.68.0.1 nằm trong 172.64.0.0/13 của Cloudflare.
	r := reqTu("172.68.0.1:443", map[string]string{"CF-Connecting-IP": "203.0.113.9"})
	if got := ipCua(r); got != "203.0.113.9" {
		t.Fatalf("từ Cloudflare phải tin CF-Connecting-IP, được %q", got)
	}
}

func TestIPCuaKhongTinHeaderGia(t *testing.T) {
	// Kẻ tấn công gọi thẳng, tự đặt CF-Connecting-IP. Không được tin.
	r := reqTu("198.51.100.7:5555", map[string]string{
		"CF-Connecting-IP": "1.1.1.1",
		"X-Forwarded-For":  "1.1.1.1",
		"X-Real-IP":        "1.1.1.1",
	})
	if got := ipCua(r); got != "198.51.100.7" {
		t.Fatalf("không được tin header từ IP lạ, được %q", got)
	}
}

func TestIPCuaNginxNoiBo(t *testing.T) {
	cu := TinProxy
	TinProxy = true
	defer func() { TinProxy = cu }()

	r := reqTu("127.0.0.1:9999", map[string]string{"X-Real-IP": "203.0.113.20"})
	if got := ipCua(r); got != "203.0.113.20" {
		t.Fatalf("nginx cùng máy phải được tin, được %q", got)
	}
}

func TestIPCuaLayPhanTuCuoiXFF(t *testing.T) {
	cu := TinProxy
	TinProxy = true
	defer func() { TinProxy = cu }()

	// Khách tự gửi "1.1.1.1", nginx nối IP thật vào cuối.
	r := reqTu("127.0.0.1:9999", map[string]string{
		"X-Forwarded-For": "1.1.1.1, 203.0.113.30",
	})
	if got := ipCua(r); got != "203.0.113.30" {
		t.Fatalf("phải lấy phần tử CUỐI của XFF, được %q", got)
	}
}

func TestIPCuaTatTinProxy(t *testing.T) {
	cu := TinProxy
	TinProxy = false
	defer func() { TinProxy = cu }()

	r := reqTu("127.0.0.1:9999", map[string]string{"X-Real-IP": "1.1.1.1"})
	if got := ipCua(r); got != "127.0.0.1" {
		t.Fatalf("tắt tin-proxy thì không tin header nào, được %q", got)
	}
}
