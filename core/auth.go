// Đăng nhập cho bàn thợ.
//
// Mật khẩu lưu bằng bcrypt, không bao giờ lưu bản rõ. File người dùng nằm
// trong data/ nên đã gitignore — hash lộ ra vẫn là hash, nhưng không có lý
// do gì để nó nằm trong lịch sử git.
//
// CHƯA CÓ HTTPS. Đang chạy demo trên IP:port, mật khẩu đi qua mạng dưới
// dạng bản rõ. Có tên miền thì chạy với -https để cookie bật cờ Secure và
// đặt nginx/Caddy đứng trước. Xem README mục "Bật HTTPS khi có tên miền".
package core

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

const (
	VaiTroChu = "chu" // xem được tiền, thống kê, quản lý người dùng
	VaiTroTho = "tho" // chỉ đơn được giao, không thấy doanh thu
)

type NguoiDung struct {
	Ten         string `yaml:"ten"` // tên đăng nhập
	HoTen       string `yaml:"ho_ten"`
	MatKhauHash string `yaml:"mat_khau_hash"`
	VaiTro      string `yaml:"vai_tro"`
	Tat         bool   `yaml:"tat"`
}

func (n NguoiDung) LaChu() bool { return n.VaiTro == VaiTroChu }

func (n NguoiDung) TenHienThi() string {
	if strings.TrimSpace(n.HoTen) != "" {
		return n.HoTen
	}
	return n.Ten
}

type khoNguoiDung struct {
	NguoiDung []NguoiDung `yaml:"nguoi_dung"`
}

var (
	nguoiDungMu sync.RWMutex
	nguoiDung   []NguoiDung
)

func fileNguoiDung() string { return P("data/nguoi-dung.yaml") }

func NapNguoiDung() error {
	b, err := os.ReadFile(fileNguoiDung())
	if os.IsNotExist(err) {
		nguoiDungMu.Lock()
		nguoiDung = nil
		nguoiDungMu.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	var kho khoNguoiDung
	if err := yaml.Unmarshal(b, &kho); err != nil {
		return fmt.Errorf("data/nguoi-dung.yaml hỏng: %w", err)
	}
	nguoiDungMu.Lock()
	nguoiDung = kho.NguoiDung
	nguoiDungMu.Unlock()
	return nil
}

func luuNguoiDung() error {
	nguoiDungMu.RLock()
	b, err := yaml.Marshal(khoNguoiDung{NguoiDung: nguoiDung})
	nguoiDungMu.RUnlock()
	if err != nil {
		return err
	}
	dau := []byte("# Tài khoản bàn thợ. Mật khẩu là bcrypt hash, không phải bản rõ.\n" +
		"# Thêm người: tramvot -them-nguoi-dung ten:vai_tro\n" +
		"# Đổi mật khẩu: tramvot -doi-mat-khau ten\n\n")
	return ghiAtomic(fileNguoiDung(), append(dau, b...))
}

func DanhSachNguoiDung() []NguoiDung {
	nguoiDungMu.RLock()
	defer nguoiDungMu.RUnlock()
	out := make([]NguoiDung, len(nguoiDung))
	copy(out, nguoiDung)
	return out
}

func TimNguoiDung(ten string) (NguoiDung, bool) {
	ten = strings.ToLower(strings.TrimSpace(ten))
	nguoiDungMu.RLock()
	defer nguoiDungMu.RUnlock()
	for _, n := range nguoiDung {
		if strings.ToLower(n.Ten) == ten {
			return n, true
		}
	}
	return NguoiDung{}, false
}

// ThemNguoiDung — dùng từ dòng lệnh. Người đầu tiên luôn là chủ, vì nếu
// người đầu tiên là thợ thì không ai vào được trang quản lý người dùng.
func ThemNguoiDung(ten, hoTen, matKhau, vaiTro string) error {
	ten = strings.ToLower(strings.TrimSpace(ten))
	if ten == "" {
		return fmt.Errorf("thiếu tên đăng nhập")
	}
	if len([]rune(matKhau)) < 8 {
		return fmt.Errorf("mật khẩu phải từ 8 ký tự trở lên")
	}
	if _, co := TimNguoiDung(ten); co {
		return fmt.Errorf("đã có tài khoản tên %q", ten)
	}
	if vaiTro != VaiTroChu && vaiTro != VaiTroTho {
		vaiTro = VaiTroTho
	}
	nguoiDungMu.RLock()
	trong := len(nguoiDung) == 0
	nguoiDungMu.RUnlock()
	if trong {
		vaiTro = VaiTroChu
	}

	h, err := bcrypt.GenerateFromPassword([]byte(matKhau), 12)
	if err != nil {
		return err
	}
	nguoiDungMu.Lock()
	nguoiDung = append(nguoiDung, NguoiDung{
		Ten: ten, HoTen: hoTen, MatKhauHash: string(h), VaiTro: vaiTro,
	})
	nguoiDungMu.Unlock()
	return luuNguoiDung()
}

func DoiMatKhau(ten, matKhauMoi string) error {
	if len([]rune(matKhauMoi)) < 8 {
		return fmt.Errorf("mật khẩu phải từ 8 ký tự trở lên")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(matKhauMoi), 12)
	if err != nil {
		return err
	}
	ten = strings.ToLower(strings.TrimSpace(ten))
	nguoiDungMu.Lock()
	thay := false
	for i := range nguoiDung {
		if strings.ToLower(nguoiDung[i].Ten) == ten {
			nguoiDung[i].MatKhauHash = string(h)
			thay = true
		}
	}
	nguoiDungMu.Unlock()
	if !thay {
		return fmt.Errorf("không có tài khoản tên %q", ten)
	}
	return luuNguoiDung()
}

func DoiVaiTro(ten, vaiTro string, tat bool) error {
	if vaiTro != VaiTroChu && vaiTro != VaiTroTho {
		return fmt.Errorf("vai trò phải là chu hoặc tho")
	}
	ten = strings.ToLower(strings.TrimSpace(ten))
	nguoiDungMu.Lock()
	vt := -1
	for i := range nguoiDung {
		if strings.ToLower(nguoiDung[i].Ten) == ten {
			vt = i
			break
		}
	}
	if vt < 0 {
		nguoiDungMu.Unlock()
		return fmt.Errorf("không có tài khoản tên %q", ten)
	}
	// Đếm chủ TRƯỚC khi sửa, tính trên kết quả giả định. Bản cũ sửa vào
	// danh sách rồi mới đếm, và khi từ chối thì không hoàn lại — file YAML
	// vẫn ghi "chu" nhưng bộ nhớ đã thành "tho", nên người vừa bấm mất
	// quyền quản trị cho tới lúc khởi động lại dịch vụ.
	conChu := 0
	for i := range nguoiDung {
		vtr, off := nguoiDung[i].VaiTro, nguoiDung[i].Tat
		if i == vt {
			vtr, off = vaiTro, tat
		}
		if vtr == VaiTroChu && !off {
			conChu++
		}
	}
	// Hạ nốt người chủ cuối cùng là tự nhốt mình ở ngoài cửa.
	if conChu == 0 {
		nguoiDungMu.Unlock()
		return fmt.Errorf("phải còn ít nhất một tài khoản chủ đang bật")
	}
	nguoiDung[vt].VaiTro = vaiTro
	nguoiDung[vt].Tat = tat
	nguoiDungMu.Unlock()
	// Nhả khoá rồi mới lưu: luuNguoiDung tự RLock, gọi khi đang giữ Lock
	// là treo cả tiến trình. Bản cũ mắc đúng lỗi đó, chỉ chưa lộ vì mọi
	// lần đổi vai trò đều bị chặn ở trên trước khi chạy tới.
	return luuNguoiDung()
}

// --- Phiên đăng nhập -------------------------------------------------
// Giữ trong RAM. Khởi động lại dịch vụ là mọi người phải đăng nhập lại —
// chấp nhận được, và đổi lại không có bảng phiên nào để rò rỉ.

type phien struct {
	Ten   string
	Het   time.Time
	IPTao string
}

var (
	phienMu sync.RWMutex
	phienDs = map[string]phien{}
)

const tenCookie = "tv_phien"
const hanPhien = 12 * time.Hour

// HTTPSBat — bật cờ Secure trên cookie. Chỉ bật khi thật sự có HTTPS,
// vì cookie Secure gửi qua HTTP sẽ bị trình duyệt bỏ, và không ai đăng
// nhập được nữa mà không hiểu vì sao.
var HTTPSBat bool

func maNgauNhien(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		// crypto/rand hỏng thì không có đường lui an toàn nào cả.
		panic("không lấy được số ngẫu nhiên: " + err.Error())
	}
	return hex.EncodeToString(b)
}

func taoPhien(w http.ResponseWriter, ten, ip string) {
	ma := maNgauNhien(32)
	phienMu.Lock()
	// Dọn phiên hết hạn nhân tiện, khỏi cần goroutine quét riêng.
	for k, v := range phienDs {
		if time.Now().After(v.Het) {
			delete(phienDs, k)
		}
	}
	phienDs[ma] = phien{Ten: ten, Het: time.Now().Add(hanPhien), IPTao: ip}
	phienMu.Unlock()

	http.SetCookie(w, &http.Cookie{
		Name:     tenCookie,
		Value:    ma,
		Path:     "/",
		HttpOnly: true,
		Secure:   HTTPSBat,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(hanPhien / time.Second),
	})
}

func xoaPhien(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(tenCookie); err == nil {
		phienMu.Lock()
		delete(phienDs, c.Value)
		phienMu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{
		Name: tenCookie, Value: "", Path: "/", HttpOnly: true,
		Secure: HTTPSBat, SameSite: http.SameSiteLaxMode, MaxAge: -1,
	})
}

// NguoiDangNhap trả về người dùng của request, hoặc false nếu chưa đăng nhập.
func NguoiDangNhap(r *http.Request) (NguoiDung, bool) {
	c, err := r.Cookie(tenCookie)
	if err != nil {
		return NguoiDung{}, false
	}
	phienMu.RLock()
	p, co := phienDs[c.Value]
	phienMu.RUnlock()
	if !co || time.Now().After(p.Het) {
		return NguoiDung{}, false
	}
	nd, co := TimNguoiDung(p.Ten)
	if !co || nd.Tat {
		return NguoiDung{}, false
	}
	return nd, true
}

// --- Chặn dò mật khẩu ------------------------------------------------

type demSai struct {
	mu  sync.Mutex
	lan map[string][]time.Time
}

var loginFail = &demSai{lan: map[string][]time.Time{}}

const (
	soLanSaiToiDa = 5
	cuaSoKhoa     = 15 * time.Minute
)

func (d *demSai) biKhoa(khoa string) (bool, time.Duration) {
	d.mu.Lock()
	defer d.mu.Unlock()
	now := time.Now()
	con := d.lan[khoa][:0]
	for _, t := range d.lan[khoa] {
		if now.Sub(t) < cuaSoKhoa {
			con = append(con, t)
		}
	}
	d.lan[khoa] = con
	if len(con) >= soLanSaiToiDa {
		return true, cuaSoKhoa - now.Sub(con[0])
	}
	return false, 0
}

func (d *demSai) ghiSai(khoa string) {
	d.mu.Lock()
	d.lan[khoa] = append(d.lan[khoa], time.Now())
	d.mu.Unlock()
}

func (d *demSai) xoa(khoa string) {
	d.mu.Lock()
	delete(d.lan, khoa)
	d.mu.Unlock()
}

// KiemTraDangNhap — so mật khẩu. Luôn chạy bcrypt kể cả khi không có tài
// khoản đó, để thời gian trả lời không tiết lộ tên nào tồn tại.
var hashGia = "$2a$12$C6UzMDM.H6dfI/f/IKcEeO3S9OQnRJPfN2xhBHDMBQ2lJ9EehLvBK"

func KiemTraDangNhap(ten, matKhau string) (NguoiDung, error) {
	nd, co := TimNguoiDung(ten)
	hash := hashGia
	if co {
		hash = nd.MatKhauHash
	}
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(matKhau))
	if !co || err != nil || nd.Tat {
		return NguoiDung{}, fmt.Errorf("sai tên đăng nhập hoặc mật khẩu")
	}
	return nd, nil
}

// --- Ghi file an toàn ------------------------------------------------
// Ghi tạm rồi đổi tên. Mất điện giữa chừng thì file cũ còn nguyên, thay vì
// còn lại một file JSON cụt đầu không đọc được.

func ghiAtomic(duong string, noiDung []byte) error {
	if err := os.MkdirAll(filepath.Dir(duong), 0o755); err != nil {
		return err
	}
	tam := duong + ".tmp"
	if err := os.WriteFile(tam, noiDung, 0o600); err != nil {
		return err
	}
	return os.Rename(tam, duong)
}
