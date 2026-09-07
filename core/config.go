// Đọc cấu hình. Nguồn sự thật là hai file YAML dùng chung với phần Python:
// config.yaml và vanhanh/bang-gia.yaml. Go KHÔNG có bảng giá riêng, không
// có tham số riêng — bất cứ thứ gì lệch giữa Go và Python đều là bug.
package core

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
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
}

func (l LienHe) CoGiKhong() bool {
	return l.DienThoai != "" || l.Zalo != "" || l.Email != "" || l.DiaChi != "" || l.Facebook != ""
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
	BangGia    string   `yaml:"bang_gia"`
	DanhMucVot string   `yaml:"danh_muc_vot"`
	CaSua      string   `yaml:"ca_sua"`
	Voice      string   `yaml:"voice"`
	TinNhan    string   `yaml:"tin_nhan"`
	Anh        string   `yaml:"anh"`
	KhoTriThuc string   `yaml:"kho_tri_thuc"`
	RagIndex   string   `yaml:"rag_index"`
	TaiLieuNap []string `yaml:"tai_lieu_nap"`
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
	Ten               string  `yaml:"ten"`
	Nhom              string  `yaml:"nhom"`
	Gia               []*int  `yaml:"gia"`
	VatTu             int     `yaml:"vat_tu"`
	GioCong           float64 `yaml:"gio_cong"`
	GioCongMoiVaoNghe float64 `yaml:"gio_cong_moi_vao_nghe"`
	BaoHanhThang      int     `yaml:"bao_hanh_thang"`
	LeadTimeNgay      int     `yaml:"lead_time_ngay"`
	TuGiaiDoan        int     `yaml:"tu_giai_doan"`
	DieuKien          string  `yaml:"dieu_kien"`

	// ThuTu quyết định chỗ đứng của việc trên mọi trang, nhỏ đứng trước. Bỏ
	// trống (0) nghĩa là "chưa xếp": việc ấy xuống sau các việc đã đánh số và
	// giữ nguyên thứ tự viết trong file. Nhờ vậy hôm chưa đánh số việc nào thì
	// trang không đổi gì so với trước khi có khoá này.
	ThuTu int `yaml:"thu_tu"`

	// BaoGiaRieng: có bán, nhưng không có giá niêm yết — phải xem vợt rồi
	// báo riêng. Khác hẳn gia null trơn (nghĩa là chưa mở bán ở giai đoạn
	// này). Đặt cờ này thì dịch vụ hiện ra cho khách dù cả ba ô giá đều null.
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
	return nil
}

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

// DichVuDangBan — danh sách hiển thị cho khách.
func DichVuDangBan(gd int) []DichVu {
	out := []DichVu{}
	for _, d := range DichVuTatCa() {
		if d.DaMo(gd) {
			out = append(out, d)
		}
	}
	return out
}
