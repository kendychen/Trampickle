package core

// Chặn dò mật khẩu ở /dang-nhap.
//
// Đếm theo CẢ IP lẫn tên tài khoản, vì mỗi cách một mình đều có lỗ:
//   - chỉ theo IP  : botnet nhiều IP dò một tài khoản vẫn lọt.
//   - chỉ theo tên : kẻ xấu cố tình gõ sai để khoá tài khoản Kendy (DoS).
//
// Cách thoát DoS: khoá theo tên chỉ chặn IP LẠ. IP đã từng đăng nhập thành
// công vào tài khoản đó vẫn vào được, nên Kendy ngồi ở tiệm không bao giờ bị
// người ngoài khoá ra ngoài.
//
// Giữ trong RAM như phiên đăng nhập: restart là quên hết. Chấp nhận — kẻ dò
// không làm cho tiến trình restart được, và đổi lại không có file trạng thái
// nào phải dọn.

import (
	"strings"
	"sync"
	"time"
)

const (
	saiToiDaIP  = 5
	saiToiDaTen = 10
	khoaDau     = 15 * time.Minute
	khoaTran    = 4 * time.Hour
	ipQuenToiDa = 20
	// khoaToiDa: quá số này thì dọn các mục đã nguội. Một lượt quét phân tán
	// đi qua Cloudflare có thể mang tới hàng vạn IP khác nhau; mỗi IP một mục
	// không bao giờ xoá là rò bộ nhớ, và rò bằng chính đòn mà bộ đếm này sinh
	// ra để chặn.
	khoaToiDa = 5000
)

type demSai struct {
	sai     int
	lanKhoa int
	moLuc   time.Time
	chamLuc time.Time // lần cuối chạm tới, để biết mục nào đã nguội
}

var (
	khoaMu  sync.Mutex
	khoaIP  = map[string]*demSai{}
	khoaTen = map[string]*demSai{}
	// ipQuen: tên tài khoản -> tập IP đã từng đăng nhập thành công.
	ipQuen = map[string]map[string]bool{}
)

// hanKhoa: 15 phút, rồi 30, 60, 120, trần 4 giờ.
func hanKhoa(lan int) time.Duration {
	d := khoaDau
	for i := 1; i < lan; i++ {
		d *= 2
		if d >= khoaTran {
			return khoaTran
		}
	}
	return d
}

// choPhepThu trả về false kèm thời gian còn phải chờ.
func choPhepThu(ip, ten string) (bool, time.Duration) {
	ten = strings.ToLower(strings.TrimSpace(ten))
	khoaMu.Lock()
	defer khoaMu.Unlock()
	now := time.Now()

	if d := khoaIP[ip]; d != nil && now.Before(d.moLuc) {
		return false, d.moLuc.Sub(now)
	}
	if d := khoaTen[ten]; d != nil && now.Before(d.moLuc) && !ipQuen[ten][ip] {
		return false, d.moLuc.Sub(now)
	}
	return true, 0
}

func ghiThuSai(ip, ten string) {
	ten = strings.ToLower(strings.TrimSpace(ten))
	khoaMu.Lock()
	defer khoaMu.Unlock()
	dem(khoaIP, ip, saiToiDaIP)
	dem(khoaTen, ten, saiToiDaTen)
	if len(khoaIP) > khoaToiDa {
		donNguoi(khoaIP)
	}
	if len(khoaTen) > khoaToiDa {
		donNguoi(khoaTen)
	}
}

// donNguoi xoá các mục hết khoá và không ai chạm tới trong khoaTran. Giữ
// nguyên mục đang khoá — đó mới là mục có việc. Gọi khi đang giữ khoaMu.
func donNguoi(m map[string]*demSai) {
	cat := time.Now().Add(-khoaTran)
	for k, d := range m {
		if d.moLuc.After(time.Now()) {
			continue // đang khoá, giữ
		}
		if d.chamLuc.Before(cat) {
			delete(m, k)
		}
	}
}

// dem tăng bộ đếm và khoá khi chạm ngưỡng. Gọi khi đang giữ khoaMu.
func dem(m map[string]*demSai, k string, nguong int) {
	d := m[k]
	if d == nil {
		d = &demSai{}
		m[k] = d
	}
	// Khoá cũ đã hết hạn thì đếm lại từ đầu, nhưng lanKhoa giữ nguyên — kẻ
	// quay lại dò tiếp phải chờ lâu hơn lần trước.
	if !d.moLuc.IsZero() && time.Now().After(d.moLuc) {
		d.sai = 0
		d.moLuc = time.Time{}
	}
	d.chamLuc = time.Now()
	d.sai++
	if d.sai >= nguong {
		d.lanKhoa++
		d.moLuc = time.Now().Add(hanKhoa(d.lanKhoa))
		d.sai = 0
	}
}

func ghiThuDung(ip, ten string) {
	ten = strings.ToLower(strings.TrimSpace(ten))
	khoaMu.Lock()
	defer khoaMu.Unlock()
	delete(khoaIP, ip)
	if khoaTen[ten] != nil {
		khoaTen[ten].sai = 0
	}
	if ipQuen[ten] == nil {
		ipQuen[ten] = map[string]bool{}
	}
	// Giới hạn số IP nhớ mỗi tài khoản, khỏi phình vô hạn nếu Kendy dùng 4G
	// đổi IP liên tục. Quá thì xoá sạch, lần sau nhớ lại từ đầu.
	if len(ipQuen[ten]) >= ipQuenToiDa {
		ipQuen[ten] = map[string]bool{}
	}
	ipQuen[ten][ip] = true
}

// ipDaQuen: IP này đã từng đăng nhập thành công vào tài khoản đó chưa. Dùng
// để biết có nên gửi mail cảnh báo không (core/mailcanhbao.go).
func ipDaQuen(ten, ip string) bool {
	ten = strings.ToLower(strings.TrimSpace(ten))
	khoaMu.Lock()
	defer khoaMu.Unlock()
	return ipQuen[ten][ip]
}
