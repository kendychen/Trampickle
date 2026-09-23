// Đọc cấu hình. Nguồn sự thật là hai file YAML dùng chung với phần Python:
// config.yaml và vanhanh/bang-gia.yaml. Go KHÔNG có bảng giá riêng, không
// có tham số riêng — bất cứ thứ gì lệch giữa Go và Python đều là bug.
package core

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type ThuongHieu struct {
	Ten      string `yaml:"ten"`
	DongMoTa string `yaml:"dong_mo_ta"`
	TenDayDu string `yaml:"ten_day_du"`
	LienHe   LienHe `yaml:"lien_he"`
}

// LienHe để trống được. Trang web tự giấu dòng nào chưa điền thay vì hiện
// ra một ô rỗng — thà thiếu thông tin còn hơn hiện "Điện thoại: " cụt lủn.
type LienHe struct {
	DienThoai string `yaml:"dien_thoai"`
	Zalo      string `yaml:"zalo"`
	Email     string `yaml:"email"`
	DiaChi    string `yaml:"dia_chi"`
	GioLamVic string `yaml:"gio_lam_viec"`
	Facebook  string `yaml:"facebook"`

	// Mạng xã hội. Facebook nằm trên vì nó có trước; bốn cái này là một cụm,
	// hiện chung một hàng icon — xem MangXaHoi.
	TikTok    string `yaml:"tiktok"`
	Instagram string `yaml:"instagram"`
	YouTube   string `yaml:"youtube"`

	// Tài khoản nhận tiền, hiện trên trang tra cứu khi đơn đã xong. Để trống
	// thì trang chỉ còn COD.
	NganHangMa  string `yaml:"ngan_hang_ma"` // mã BIN VietQR, ví dụ 970436
	SoTaiKhoan  string `yaml:"so_tai_khoan"`
	ChuTaiKhoan string `yaml:"chu_tai_khoan"`
}

func (l LienHe) CoGiKhong() bool {
	return l.DienThoai != "" || l.Zalo != "" || l.Email != "" || l.DiaChi != "" || len(l.MangXaHoi()) > 0
}

// TrangMXH là một cái tên với một đường dẫn. Ma dùng để chọn hình trong
// {{template "bt-mxh"}} — xem core/ui/hinh.html.
type TrangMXH struct {
	Ma  string
	Ten string
	URL string
}

// MangXaHoi trả những trang đã điền, theo thứ tự cố định. Thứ tự cố định chứ
// không theo thứ tự điền: hàng icon ở chân trang và ở bảng hiệu app phải
// giống nhau mọi lúc, người ta nhớ vị trí chứ không đọc lại từng cái.
func (l LienHe) MangXaHoi() []TrangMXH {
	var ra []TrangMXH
	for _, t := range []TrangMXH{
		{"facebook", "Facebook", l.Facebook},
		{"tiktok", "TikTok", l.TikTok},
		{"instagram", "Instagram", l.Instagram},
		{"youtube", "YouTube", l.YouTube},
	} {
		if t.URL != "" {
			ra = append(ra, t)
		}
	}
	return ra
}

// BanDo — đường mở Google Maps tới địa chỉ trạm, rỗng nếu chưa điền địa chỉ.
//
// Dùng dạng tìm kiếm chứ không phải toạ độ: địa chỉ gõ tay không kèm lat/lng,
// mà bắt Kendy đi tra toạ độ mỗi lần đổi chỗ thì chắc chắn có ngày địa chỉ một
// nơi chấm bản đồ một nẻo. Maps tự khớp chuỗi, sai thì sai ở phía hiển thị chứ
// dữ liệu trong máy vẫn là đúng một chỗ.
func (l LienHe) BanDo() string {
	d := strings.TrimSpace(l.DiaChi)
	if d == "" {
		return ""
	}
	return "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(d)
}

type GeminiCfg struct {
	TenModel   string  `yaml:"ten_model"`
	NhietDo    float64 `yaml:"nhiet_do"`
	TokenToiDa int     `yaml:"token_toi_da"`
	ReqMoiNgay int     `yaml:"req_moi_ngay_toi_da"`
}

type OllamaCfg struct {
	Host        string  `yaml:"host"`
	TenModel    string  `yaml:"ten_model"`
	NumCtx      int     `yaml:"num_ctx"`
	NhietDo     float64 `yaml:"nhiet_do"`
	TimeoutGiay int     `yaml:"timeout_giay"`
}

type ModelCfg struct {
	MacDinh string    `yaml:"mac_dinh"`
	DuPhong string    `yaml:"du_phong"`
	Gemini  GeminiCfg `yaml:"gemini"`
	Ollama  OllamaCfg `yaml:"ollama"`
}

type RagGemini struct {
	TenModel string `yaml:"ten_model"`
	SoChieu  int    `yaml:"so_chieu"`
}

type RagOllama struct {
	TenModel    string `yaml:"ten_model"`
	TimeoutGiay int    `yaml:"timeout_giay"`
}

type RagCfg struct {
	NhaCungCap     string    `yaml:"nha_cung_cap"`
	MaxKyTuMoiDoan int       `yaml:"max_ky_tu_moi_doan"`
	ChongLanKyTu   int       `yaml:"chong_lan_ky_tu"`
	SoDoanLayVe    int       `yaml:"so_doan_lay_ve"`
	DiemToiThieu   float64   `yaml:"diem_toi_thieu"`
	Gemini         RagGemini `yaml:"gemini"`
	Ollama         RagOllama `yaml:"ollama"`
}

type AgentTechnical struct {
	Bat                 bool `yaml:"bat"`
	NguongChuyenSangRag int  `yaml:"nguong_chuyen_sang_rag"`
	SoCaToiThieuDeTin   int  `yaml:"so_ca_toi_thieu_de_tin"`
}

type AgentCfg struct {
	Technical AgentTechnical `yaml:"technical"`
}

type DuongDan struct {
	BangGia     string   `yaml:"bang_gia"`
	DanhMucVot  string   `yaml:"danh_muc_vot"`
	DanhMucGiay string   `yaml:"danh_muc_giay"`
	CaSua       string   `yaml:"ca_sua"`
	Voice       string   `yaml:"voice"`
	TinNhan     string   `yaml:"tin_nhan"`
	Anh         string   `yaml:"anh"`
	KhoTriThuc  string   `yaml:"kho_tri_thuc"`
	RagIndex    string   `yaml:"rag_index"`
	TaiLieuNap  []string `yaml:"tai_lieu_nap"`
}

// EmailCfg — gửi mail cho khách. Khóa API KHÔNG nằm ở đây: đọc từ biến môi
// trường RESEND_API_KEY. config.yaml có mặt trong git, khóa thì không.
type EmailCfg struct {
	Bat    bool   `yaml:"bat"`
	Tu     string `yaml:"tu"`
	TraLoi string `yaml:"tra_loi"`
	// GocWeb dùng để dựng link tra cứu trong mail. Để trống thì mail chỉ có
	// mã, không có link bấm được — vẫn dùng được, chỉ bất tiện hơn.
	GocWeb string `yaml:"goc_web"`
	// BaoCao — hòm thư nhận báo cáo sổ tháng. Để trống thì gửi về tra_loi.
	// Mail này chứa số liệu nội bộ, đừng đặt địa chỉ dùng chung với khách.
	BaoCao string `yaml:"bao_cao"`
}

type Config struct {
	ThuongHieu ThuongHieu `yaml:"thuong_hieu"`
	Model      ModelCfg   `yaml:"model"`
	Rag        RagCfg     `yaml:"rag"`
	Agent      AgentCfg   `yaml:"agent"`
	DuongDan   DuongDan   `yaml:"duong_dan"`
	Email      EmailCfg   `yaml:"email"`
}

// --- Bảng giá -------------------------------------------------------
// Đọc để HIỂN THỊ. Go không sinh ra GIÁ BÁN: báo giá, điểm hoà vốn theo giờ
// công, lãi gộp dự phóng — những thứ suy ra từ bảng giá này — đều ở
// src/quote.py. Hai nơi cùng tính một con số cam kết với khách là hai nơi có
// thể lệch nhau, và bên sai luôn là bên khách đang cầm.
//
// Go ĐƯỢC cộng trừ trên số đã ghi vào sổ (core/sotien.go): tiền đã thu, tiền
// đã chi, lãi lỗ, còn bao lâu về vốn. Đó là số liệu quá khứ có người gõ vào
// vì nó đã xảy ra, không phải giá sinh từ bảng này.

type DichVu struct {
	Ma                string  `yaml:"ma"`
	Slug              string  `yaml:"slug"`
	Ten               string  `yaml:"ten"`
	Nhom              string  `yaml:"nhom"`
	DoiTuong          string  `yaml:"doi_tuong"`
	Gia               []*int  `yaml:"gia"`
	GiaDen            []*int  `yaml:"gia_den"`
	VatTu             int     `yaml:"vat_tu"`
	GioCong           float64 `yaml:"gio_cong"`
	GioCongMoiVaoNghe float64 `yaml:"gio_cong_moi_vao_nghe"`
	BaoHanhThang      int     `yaml:"bao_hanh_thang"`
	LeadTimeNgay      int     `yaml:"lead_time_ngay"`

	// LeadTimeDen là đầu trên của khoảng ngày làm: điền 1 và 3 thì trang hiện
	// "1–3 ngày". Bỏ trống thì chỉ hiện một con số như trước.
	//
	// VÌ SAO HAI KHOÁ CHỨ KHÔNG MỘT CHUỖI "1-3". src/quote.py:146 đọc
	// lead_time_ngay bằng int(). Đổi khoá ấy thành chuỗi là báo giá Python vỡ
	// ngay, mà đó mới là chỗ chốt con số gửi khách. Thêm khoá mới thì Python
	// không thấy gì lạ và vẫn lấy đầu dưới — hẹn sớm hơn thực tế thì đằng nào
	// thợ cũng phải gọi báo, còn hẹn muộn hơn là mất khách.
	LeadTimeDen int    `yaml:"lead_time_ngay_den"`
	TuGiaiDoan  int    `yaml:"tu_giai_doan"`
	DieuKien    string `yaml:"dieu_kien"`

	// ThuTu quyết định chỗ đứng của việc trên mọi trang, nhỏ đứng trước. Bỏ
	// trống (0) nghĩa là "chưa xếp": việc ấy xuống sau các việc đã đánh số và
	// giữ nguyên thứ tự viết trong file. Nhờ vậy hôm chưa đánh số việc nào thì
	// trang không đổi gì so với trước khi có khoá này.
	ThuTu int `yaml:"thu_tu"`

	// BaoGiaRieng: có bán, nhưng không có giá niêm yết — phải xem vợt rồi
	// báo riêng. Khác hẳn gia null trơn (nghĩa là chưa mở bán ở giai đoạn
	// này). Đặt cờ này thì dịch vụ hiện ra cho khách dù cả ba ô giá đều null.
	// NoiBat tô nổi việc này trong lưới "Trạm làm được gì" ở trang chủ. Chỉ
	// có tác dụng khi công tắc dv_noi_bat đang bật — xem core/chedo.go.
	NoiBat bool `yaml:"noi_bat"`

	// An tắt một việc khỏi mọi chỗ khách nhìn thấy: lưới ở /app, trang chủ,
	// /dich-vu và cả trang giá riêng của nó (vào thẳng link cũ thì ra 404).
	//
	// Vì sao cần một cờ riêng khi đã có hai cách ẩn sẵn: xoá trống giá thì
	// mất luôn con số, bật lại phải nhớ mà gõ lại; còn tu_giai_doan là lịch
	// mở bán theo giai đoạn của trạm, mượn nó làm công tắc tạm thời là sau
	// này đọc file không ai biết số ấy nói lịch hay nói "đang nghỉ". Việc bật
	// bao_gia_rieng thì hai cách trên không tắt nổi.
	//
	// Tắt KHÔNG đụng tới bên trong: thợ vẫn tích được việc này khi lên phiếu
	// (DichVuLenPhieu), vì khách quen mang vợt tới tận nơi nhờ làm thì vẫn
	// phải ghi đúng việc, đúng giá — chứ không phải bật lên rồi tắt lại.
	//
	// Zero-value là "hiện", nên file bang-gia.yaml cũ không đổi hành vi.
	An bool `yaml:"an"`

	BaoGiaRieng bool `yaml:"bao_gia_rieng"`
}

type KhongBan struct {
	Ma   string `yaml:"ma"`
	Ten  string `yaml:"ten"`
	LyDo string `yaml:"ly_do"`
}

type Nguong struct {
	TangKhoiLuongToiDaG   float64 `yaml:"tang_khoi_luong_toi_da_g"`
	BienGopToiThieuDong   int     `yaml:"bien_gop_toi_thieu_dong"`
	CacQuaKenhTraTienDong int     `yaml:"cac_qua_kenh_tra_tien_dong"`
	ShipCaTuChoiDong      int     `yaml:"ship_ca_tu_choi_dong"`
	NguongMoKenhShipDong  int     `yaml:"nguong_mo_kenh_ship_dong"`

	// FreeShipVeTuDong — đơn từ mức này trở lên thì trạm chịu phí gửi trả.
	// Tách khỏi NguongMoKenhShipDong: cái kia là luật vận hành nội bộ (dưới
	// mức đó không mở kênh ship), cái này là lời hứa in trên web. Trộn hai
	// thứ vào một số là có ngày sửa luật vận hành xong lỡ đổi luôn lời hứa.
	FreeShipVeTuDong int `yaml:"free_ship_ve_tu_dong"`
}

type BangGia struct {
	GiaiDoanHienTai int        `yaml:"giai_doan_hien_tai"`
	DinhPhiThang    int        `yaml:"dinh_phi_thang"`
	Nguong          Nguong     `yaml:"nguong"`
	DichVu          []DichVu   `yaml:"dich_vu"`
	KhongBan        []KhongBan `yaml:"khong_ban"`
}

// --- Trạng thái toàn cục -------------------------------------------

var (
	Root     string
	CFG      Config
	GIA      BangGia
	GiaiDoan int
)

// timRoot đi ngược lên từ thư mục hiện tại để tìm nơi có config.yaml.
// Cho phép chạy exe từ bất cứ đâu, kể cả bấm đúp từ Explorer.
func timRoot() (string, error) {
	ung := []string{}
	if wd, err := os.Getwd(); err == nil {
		ung = append(ung, wd)
	}
	if exe, err := os.Executable(); err == nil {
		ung = append(ung, filepath.Dir(exe))
	}
	for _, start := range ung {
		d := start
		for i := 0; i < 6; i++ {
			if _, err := os.Stat(filepath.Join(d, "config.yaml")); err == nil {
				return d, nil
			}
			cha := filepath.Dir(d)
			if cha == d {
				break
			}
			d = cha
		}
	}
	return "", fmt.Errorf("không tìm thấy config.yaml — chạy exe trong thư mục dự án")
}

// napEnv đọc .env đơn giản. Không ghi đè biến môi trường đã có: biến
// môi trường thắng file, giống hành vi của python-dotenv bên Python.
func napEnv(root string) {
	f, err := os.Open(filepath.Join(root, ".env"))
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		dong := strings.TrimSpace(sc.Text())
		if dong == "" || strings.HasPrefix(dong, "#") {
			continue
		}
		k, v, ok := strings.Cut(dong, "=")
		if !ok {
			continue
		}
		k = strings.TrimSpace(k)
		v = strings.Trim(strings.TrimSpace(v), `"'`)
		if _, da := os.LookupEnv(k); !da {
			os.Setenv(k, v)
		}
	}
}

func LoadConfig() error {
	root, err := timRoot()
	if err != nil {
		return err
	}
	Root = root
	napEnv(root)

	b, err := os.ReadFile(filepath.Join(root, "config.yaml"))
	if err != nil {
		return fmt.Errorf("đọc config.yaml: %w", err)
	}
	if err := yaml.Unmarshal(b, &CFG); err != nil {
		return fmt.Errorf("config.yaml hỏng: %w", err)
	}

	b, err = os.ReadFile(filepath.Join(root, CFG.DuongDan.BangGia))
	if err != nil {
		return fmt.Errorf("đọc bảng giá: %w", err)
	}
	if err := yaml.Unmarshal(b, &GIA); err != nil {
		return fmt.Errorf("bảng giá hỏng: %w", err)
	}
	GiaiDoan = GIA.GiaiDoanHienTai
	NapBaoTri()
	return nil
}

func (d DichVu) DoiTuongChuan() string {
	switch d.DoiTuong {
	case "giay":
		return "giay"
	case "ca_hai":
		return "ca_hai"
	default:
		return "vot"
	}
}
func LaDoiTuongHopLe(s string) bool { return s == "vot" || s == "giay" || s == "ca_hai" }

func P(rel string) string { return filepath.Join(Root, rel) }

func OllamaHost() string {
	if h := strings.TrimSpace(os.Getenv("OLLAMA_HOST")); h != "" {
		return strings.TrimRight(h, "/")
	}
	return strings.TrimRight(CFG.Model.Ollama.Host, "/")
}

// Provider — biến môi trường thắng, rồi tới lựa chọn trong admin, cuối cùng
// mới tới config.yaml. Giống bên Python ở chỗ env luôn đứng đầu.
// GeminiKey và ResendKey nằm ở bimat.go, cùng chỗ với file admin ghi ra.
func Provider() string {
	if p := strings.TrimSpace(os.Getenv("LLM_PROVIDER")); p != "" {
		return strings.ToLower(p)
	}
	if p := BiMatHienTai().NhaCungCap; p != "" {
		return p
	}
	return strings.ToLower(CFG.Model.MacDinh)
}

// GiaTheoGiaiDoan trả giá của dịch vụ ở giai đoạn hiện tại.
// nil = chưa mở bán ở giai đoạn này (khác với 0 = tặng kèm).
func (d DichVu) GiaTheoGiaiDoan(gd int) *int {
	if gd < 1 || gd > len(d.Gia) {
		return nil
	}
	return d.Gia[gd-1]
}

// GiaDenTheoGiaiDoan trả đầu TRÊN của khoảng giá, nil = không có khoảng.
//
// Một ô cho mỗi giai đoạn, đúng hình dạng của `gia`, vì giá sàn giai đoạn 3
// có thể đã cao hơn trần của giai đoạn 1 — một trần dùng chung cho cả ba là
// có ngày trang khách in "từ 150.000đ đến 140.000đ".
//
// VÌ SAO KHOÁ MỚI chứ không đổi `gia` thành chuỗi "100000-150000": giống hệt
// chuyện lead_time_ngay_den — src/quote.py:126 đọc s["gia"][stage-1] rồi
// int() ngay, đổi kiểu là báo giá gửi khách vỡ. Thêm khoá thì Python không
// thấy gì lạ và vẫn chốt theo giá sàn, tức chốt thấp hơn trần — hụt thì thợ
// gọi báo, còn lỡ chốt cao hơn mới là mất khách.
func (d DichVu) GiaDenTheoGiaiDoan(gd int) *int {
	if gd < 1 || gd > len(d.GiaDen) {
		return nil
	}
	return d.GiaDen[gd-1]
}

func (d DichVu) DaMo(gd int) bool {
	tu := d.TuGiaiDoan
	if tu == 0 {
		tu = 1
	}
	if tu > gd {
		return false
	}
	return d.BaoGiaRieng || d.GiaTheoGiaiDoan(gd) != nil
}

// LeadSo trả con số ngày làm để in vào ô thống kê: "3" hoặc "1–3". Rỗng nghĩa
// là chưa khai, chỗ gọi tự quyết định in gạch ngang hay giấu cả dòng.
//
// Đầu trên nhỏ hơn hoặc bằng đầu dưới thì coi như không có khoảng — /qt/dich-vu
// đã chặn, nhưng file bang-gia.yaml sửa tay được nên ở đây không tin vào đó.
func (d DichVu) LeadSo() string {
	if d.LeadTimeNgay <= 0 {
		return ""
	}
	if d.LeadTimeDen > d.LeadTimeNgay {
		return strconv.Itoa(d.LeadTimeNgay) + "–" + strconv.Itoa(d.LeadTimeDen)
	}
	return strconv.Itoa(d.LeadTimeNgay)
}

// LeadChu là LeadSo kèm chữ "ngày", dùng ở chỗ không có sẵn nhãn đơn vị.
func (d DichVu) LeadChu() string {
	if s := d.LeadSo(); s != "" {
		return s + " ngày"
	}
	return ""
}

// SlugSEO: đường dẫn chuẩn cho khách và bot. Ma là khóa nội bộ (DAN_VIEN),
// Slug là đường dẫn thường chứa từ khóa (dan-vien-vot-pickleball). Rỗng thì
// rơi về Ma để không vỡ khi file cũ chưa điền slug.
func (d DichVu) SlugSEO() string {
	if s := strings.TrimSpace(d.Slug); s != "" {
		return s
	}
	return d.Ma
}

// DuongDanSEO: /dich-vu/<slug> để dùng thống nhất ở sitemap, JSON-LD, template.
func (d DichVu) DuongDanSEO() string { return "/dich-vu/" + d.SlugSEO() }
func (k KhongBan) DuongDanSEO() string { return "/dich-vu/" + strings.ToLower(k.Ma) }

// DichVuDangBan — danh sách hiển thị cho KHÁCH. Đã trừ việc tắt tay bằng ô
// "tạm ngừng nhận" ở /qt/dich-vu.
//
// Hàm này là cái lưới chắn: chỗ nào khách nhìn thấy thì gọi nó. Người trong
// trạm gọi DichVuLenPhieu. Chia theo hướng ấy chứ không ngược lại, vì quên
// một chỗ thì hậu quả lệch hẳn nhau — quên ở đây là việc đã ngừng biến mất
// khỏi một bảng nội bộ, thợ thấy ngay; quên theo chiều kia là trang khách
// vẫn chào bán thứ trạm không nhận, khách đặt xong mới phải gọi từ chối.
func DichVuDangBan(gd int) []DichVu {
	out := []DichVu{}
	for _, d := range DichVuLenPhieu(gd) {
		if !d.An {
			out = append(out, d)
		}
	}
	return out
}

// DichVuLenPhieu — danh sách cho người trong trạm: tạo/sửa phiếu, khai vật
// tư. Giữ cả việc đang tắt với khách, miễn là nó có giá ở giai đoạn này.
func DichVuLenPhieu(gd int) []DichVu {
	out := []DichVu{}
	for _, d := range DichVuTatCa() {
		if d.DaMo(gd) {
			out = append(out, d)
		}
	}
	return out
}
