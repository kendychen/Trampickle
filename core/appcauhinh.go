package core

// Cấu hình màn hình app — phần KHÔNG phải chữ.
//
// Chữ của app vẫn nằm trong CayND như mọi chữ khác trên site (một kho chữ,
// một file yaml, một luật mặc-định-trong-code). File này giữ những thứ chữ
// không diễn tả được: chọn ảnh banner, che ảnh đậm nhạt bao nhiêu, khuyến mãi
// chạy từ ngày nào tới ngày nào, icon nhúc nhích kiểu gì.
//
// Đi theo đúng lối của data/lien-he.yaml: ghi ra data/app.yaml, mà data/
// không bao giờ bị đè khi deploy — Kendy chỉnh trên điện thoại lúc nào cũng
// được, không đợi ai build lại.
//
// LUẬT ZERO-VALUE: struct rỗng phải ra đúng hành vi đang chạy. Thêm trường
// mới thì đặt tên sao cho giá trị zero là "như cũ" (nên có AnBanner chứ không
// phải HienBanner). Có thế thì máy chủ đọc file cũ bằng binary mới không đổi
// mặt tiền sau lưng Kendy.

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Kiểu chạy của icon trong lưới việc.
const (
	DongTat       = "tat"
	DongLanLuot   = "lan-luot"
	DongNgauNhien = "ngau-nhien"
)

// NoiApp là giá trị Noi của tấm ảnh treo làm banner app. Dùng chung kho ảnh
// với trang chủ và trang Về chúng tôi: Kendy up ảnh ở đúng một chỗ đã quen,
// và chuyển một tấm sang chỗ khác chỉ là bấm một nút.
const NoiApp = "app"

type AppCauHinh struct {
	// --- Banner ---
	AnBanner  bool   `yaml:"an_banner"`
	BannerAnh string `yaml:"banner_anh"` // tên tệp trong kho ảnh; "" = bản vẽ vợt
	BannerToi int    `yaml:"banner_toi"` // 0..85, phần trăm lớp phủ tối trên ảnh

	// BannerVideo — tên tệp trong data/app-video/. Có video thì video thắng
	// ảnh, ảnh thắng bản vẽ; ba mức chứ không phải một công tắc ba trạng thái,
	// vì ảnh vẫn có việc riêng khi có video: nó làm khung đứng (poster) trong
	// lúc clip chưa tải xong, và làm bản thay thế khi máy khách từ chối tự
	// phát. Bỏ video đi là banner rơi về đúng tấm ảnh đang chọn, không phải
	// chọn lại từ đầu.
	BannerVideo string `yaml:"banner_video"`

	// --- Khuyến mãi ---
	// Khối vẫn tắt bằng cách xoá rỗng "Tên chương trình" như trước. Hai ngày
	// này là cái tắt thứ hai, tự động: hết đợt là khối tự biến mất, không phải
	// nhớ vào tắt tay. Để trống cả hai = chạy vô thời hạn.
	KMTu   string `yaml:"km_tu"`
	KMDen  string `yaml:"km_den"`
	KMLink string `yaml:"km_link"` // "" = /app/gui-anh

	// --- Icon trong lưới việc ---
	IconDong string `yaml:"icon_dong"`
	IconNhip int    `yaml:"icon_nhip"` // giây giữa hai lần nhúc nhích
}

// MacDinhApp là bản đang chạy khi chưa ai đụng vào trang /qt/app.
func MacDinhApp() AppCauHinh {
	return AppCauHinh{
		BannerToi: 45,
		KMLink:    "/app/gui-anh",
		IconDong:  DongLanLuot,
		IconNhip:  3,
	}
}

var (
	appCHMu sync.RWMutex
	appCH   = MacDinhApp()
)

func fileAppCH() string { return P("data/app.yaml") }

// NapAppCauHinh đọc lúc khởi động. Chưa có file không phải lỗi — màn app là
// thứ khách đang dùng, thà chạy bằng mặc định còn hơn không mở được.
func NapAppCauHinh() error {
	b, err := os.ReadFile(fileAppCH())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	c := MacDinhApp()
	if err := yaml.Unmarshal(b, &c); err != nil {
		return fmt.Errorf("%s hỏng: %w", fileAppCH(), err)
	}
	appCHMu.Lock()
	appCH = chuanAppCH(c)
	appCHMu.Unlock()
	return nil
}

// AppCauHinhHienTai là thứ duy nhất được phép đọc để dựng màn app.
func AppCauHinhHienTai() AppCauHinh {
	appCHMu.RLock()
	defer appCHMu.RUnlock()
	return appCH
}

// chuanAppCH kẹp mọi giá trị về khoảng dùng được. Chạy cả lúc nạp lẫn lúc
// lưu: file trên máy chủ sửa được bằng tay, mà một con số bậy ở đây làm hỏng
// màn hình khách chứ không hỏng trang admin.
func chuanAppCH(c AppCauHinh) AppCauHinh {
	if !tenAnhSach(c.BannerAnh) {
		c.BannerAnh = ""
	}
	if !tenAnhSach(c.BannerVideo) || !strings.HasSuffix(c.BannerVideo, ".mp4") {
		c.BannerVideo = ""
	}
	if c.BannerToi < 0 {
		c.BannerToi = 0
	}
	if c.BannerToi > 85 {
		c.BannerToi = 85
	}
	c.KMTu = chuanNgay(c.KMTu)
	c.KMDen = chuanNgay(c.KMDen)
	// Đường dẫn nội bộ thôi. Dán nguyên một địa chỉ http:// vào đây là khách
	// bấm khối khuyến mãi rồi rời khỏi app — không phải thứ Kendy muốn, mà
	// cũng là chỗ dán nhầm dễ nhất.
	c.KMLink = strings.TrimSpace(c.KMLink)
	if !strings.HasPrefix(c.KMLink, "/") || strings.HasPrefix(c.KMLink, "//") {
		c.KMLink = "/app/gui-anh"
	}
	switch c.IconDong {
	case DongTat, DongLanLuot, DongNgauNhien:
	default:
		c.IconDong = DongLanLuot
	}
	if c.IconNhip < 2 {
		c.IconNhip = 2
	}
	if c.IconNhip > 60 {
		c.IconNhip = 60
	}
	return c
}

// chuanNgay nhận đúng dạng yyyy-mm-dd, còn lại coi như để trống. Ô <input
// type=date> đã trả về đúng dạng này, nhưng file yaml thì gõ tay được.
func chuanNgay(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return ""
	}
	return s
}

// DatAppCauHinh kiểm rồi ghi, trả về bản đã chuẩn hoá để trang admin hiện
// đúng thứ vừa lưu chứ không phải thứ vừa gõ.
func DatAppCauHinh(c AppCauHinh) (AppCauHinh, error) {
	c = chuanAppCH(c)
	if c.KMTu != "" && c.KMDen != "" && c.KMDen < c.KMTu {
		return c, errors.New("ngày kết thúc khuyến mãi đứng trước ngày bắt đầu")
	}
	// Ảnh phải có thật trong kho. Tên ảnh đã xoá mà còn nằm trong file thì
	// banner ra một ô đen — thà quay về bản vẽ.
	if c.BannerAnh != "" && !coAnhHero(c.BannerAnh) {
		return c, errors.New("không còn ảnh " + c.BannerAnh + " trong kho")
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		return c, err
	}
	dau := []byte("# Cấu hình màn hình app — phần không phải chữ. Sửa ở /qt/app.\n" +
		"# Chữ của app nằm trong data/noi-dung.yaml, không phải ở đây.\n\n")
	if err := ghiAtomic(fileAppCH(), append(dau, b...)); err != nil {
		return c, err
	}
	appCHMu.Lock()
	appCH = c
	appCHMu.Unlock()
	return c, nil
}

// --- Dùng trong template --------------------------------------------------

// KMHien: khối khuyến mãi có được hiện lúc này không. Hai điều kiện, phải đủ
// cả hai — tên chương trình đã gõ, và hôm nay nằm trong đợt.
func (c AppCauHinh) KMHien() bool {
	if ND("app.km.ten") == "" {
		return false
	}
	hom := time.Now().Format("2006-01-02")
	if c.KMTu != "" && hom < c.KMTu {
		return false
	}
	if c.KMDen != "" && hom > c.KMDen {
		return false
	}
	return true
}

// BannerURL rỗng nghĩa là dùng bản vẽ vợt.
func (c AppCauHinh) BannerURL() string {
	if c.BannerAnh == "" || !coAnhHero(c.BannerAnh) {
		return ""
	}
	return "/anh-trang-chu/" + c.BannerAnh
}

// BannerVideoURL rỗng nghĩa là không có video — banner rơi về ảnh, rồi về bản
// vẽ. Kiểm cả tệp có thật trên đĩa: cấu hình còn ghi tên một clip đã bị xoá
// tay trên máy chủ thì thà về ảnh, chứ đừng treo thẻ <video> trỏ vào 404 rồi
// để khách nhìn một khung đen.
func (c AppCauHinh) BannerVideoURL() string {
	if !coVideoApp(c.BannerVideo) {
		return ""
	}
	return "/app-video/" + c.BannerVideo
}

// AnhBannerApp: tấm đang treo ở chỗ "app" trong kho ảnh. Trang /qt/app hiện
// danh sách này cho Kendy chọn.
func AnhBannerApp() []AnhHero { return DsAnhNoi(NoiApp) }
