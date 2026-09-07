package core

// Xác định IP thật của khách. Đây là nền của mọi giới hạn tần suất: sai chỗ
// này thì rate limit, khoá đăng nhập và nhật ký đều ghi nhầm người.
//
// Bản cũ lấy phần tử ĐẦU của X-Forwarded-For. Phần tử đầu là do khách gửi —
// bot chỉ cần đổi header mỗi request là mọi giới hạn thành số 0, và mã đơn
// vốn theo số thứ tự thì dò ra được cả danh sách khách.
//
// Luật mới, theo thứ tự:
//  1. Request đến từ dải Cloudflare → tin CF-Connecting-IP.
//  2. Request đến từ loopback/mạng nội bộ và có -tin-proxy → tin X-Real-IP,
//     không có thì lấy phần tử CUỐI của X-Forwarded-For (phần nginx tự nối).
//  3. Còn lại → RemoteAddr.

import (
	"net"
	"net/http"
	"strings"
)

// TinProxy: chỉ bật khi thật sự có nginx đứng trước (cờ -tin-proxy).
var TinProxy bool

// Dải IP Cloudflare, nhúng cứng theo https://www.cloudflare.com/ips/
// (bản chép ngày 2026-09-07). Cố ý không tải qua mạng lúc khởi động: một lần
// Cloudflare đổi endpoint là dịch vụ không lên được, đắt hơn cái nó giải
// quyết. Cloudflare đổi dải rất hiếm; khi đổi thì sửa tay ở đây rồi build
// lại — nhớ sửa cả docs/van-hanh/nginx-trampickle.conf cho khớp.
var daiCloudflare = []string{
	"173.245.48.0/20", "103.21.244.0/22", "103.22.200.0/22", "103.31.4.0/22",
	"141.101.64.0/18", "108.162.192.0/18", "190.93.240.0/20", "188.114.96.0/20",
	"197.234.240.0/22", "198.41.128.0/17", "162.158.0.0/15", "104.16.0.0/13",
	"104.24.0.0/14", "172.64.0.0/13", "131.0.72.0/22",
	"2400:cb00::/32", "2606:4700::/32", "2803:f800::/32", "2405:b500::/32",
	"2405:8100::/32", "2a06:98c0::/29", "2c0f:f248::/32",
}

var mangCF []*net.IPNet

func init() {
	for _, s := range daiCloudflare {
		_, m, err := net.ParseCIDR(s)
		if err != nil {
			// Dải viết sai trong hằng số là lỗi lập trình, không phải lỗi vận
			// hành. Bỏ qua im lặng thì Cloudflare hoá ra không được tin mà
			// không ai biết.
			panic("dải Cloudflare sai cú pháp: " + s)
		}
		mangCF = append(mangCF, m)
	}
}

func laCloudflare(ip net.IP) bool {
	for _, m := range mangCF {
		if m.Contains(ip) {
			return true
		}
	}
	return false
}

func ipCua(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return host
	}

	if laCloudflare(ip) {
		if v := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); v != "" {
			return v
		}
	}

	if TinProxy && (ip.IsLoopback() || ip.IsPrivate()) {
		if v := strings.TrimSpace(r.Header.Get("X-Real-IP")); v != "" {
			return v
		}
		if h := r.Header.Get("X-Forwarded-For"); h != "" {
			phan := strings.Split(h, ",")
			return strings.TrimSpace(phan[len(phan)-1])
		}
	}

	return host
}
