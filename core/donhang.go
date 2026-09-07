// Quản lý đơn sửa.
//
// LƯU Ý VỀ LUẬT CỨNG #1 — chỉ src/quote.py được sinh ra GIÁ BÁN.
// File này KHÔNG định giá: không nhân giờ công với đơn giá, không suy ra
// mức giá nào từ bang-gia.yaml. Nó chỉ làm hai việc với tiền:
//  1. lưu lại con số người ta đã chốt với khách (gõ tay, hoặc lấy từ
//     bảng giá bằng cách đọc, không phải bằng cách tính),
//  2. cộng các dòng đã chốt để ra tổng đơn, và cộng tổng đơn để ra doanh
//     thu tháng.
//
// Cộng sổ không phải định giá. Ranh giới nằm đúng ở đó và đừng bước qua:
// nếu có ngày cần "gợi ý giá" trong web, gọi sang Python, đừng viết công
// thức thứ hai ở đây. Lãi lỗ và điểm về vốn tính trên số đã ghi vào sổ thì
// được — chỗ đó ở core/sotien.go, và nó cũng không đọc bảng giá.
//
// Lưu bằng JSON mỗi đơn một file, không dùng cơ sở dữ liệu. Ở quy mô vài
// trăm đơn một năm, đọc hết thư mục vẫn nhanh hơn nhiều so với công sức
// dựng và sao lưu một cái database — và mở file ra vẫn đọc được bằng mắt
// khi có sự cố.
package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// --- Trạng thái ------------------------------------------------------

const (
	TTMoi     = "moi"      // vừa nhận, chưa ai xem
	TTKiemTra = "kiem_tra" // đang soi vợt, chưa báo giá
	TTBaoGia  = "bao_gia"  // đã báo giá, chờ khách gật
	TTDangSua = "dang_sua"
	TTXong    = "xong"    // sửa xong, chờ giao
	TTDaGiao  = "da_giao" // hoàn tất
	TTTuChoi  = "tu_choi" // từ chối theo checklist
	TTHuy     = "huy"     // khách đổi ý
)

type MoTaTrangThai struct {
	Ma       string
	Ten      string
	TenKhach string // câu hiện cho khách khi tra cứu
	Mau      string // nhan | cho | tuchoi | trung
	DangChay bool   // còn nằm trên bàn thợ
}

// Thứ tự ở đây là thứ tự hiện trên bảng điều khiển.
var CacTrangThai = []MoTaTrangThai{
	{TTMoi, "Mới nhận", "Đã nhận vợt, đang xếp lịch kiểm tra", "trung", true},
	{TTKiemTra, "Đang kiểm tra", "Thợ đang kiểm tra để xác định hư hỏng", "trung", true},
	{TTBaoGia, "Chờ khách duyệt giá", "Đã báo giá, đang chờ anh/chị xác nhận", "cho", true},
	{TTDangSua, "Đang sửa", "Vợt đang được sửa", "cho", true},
	{TTXong, "Sửa xong, chờ giao", "Đã sửa xong, chuẩn bị bàn giao", "nhan", true},
	{TTDaGiao, "Đã giao", "Đã bàn giao. Cảm ơn anh/chị", "nhan", false},
	{TTTuChoi, "Từ chối nhận", "Ca này chúng tôi không nhận sửa", "tuchoi", false},
	{TTHuy, "Khách huỷ", "Đơn đã huỷ", "tuchoi", false},
}

func TrangThaiCua(ma string) MoTaTrangThai {
	for _, t := range CacTrangThai {
		if t.Ma == ma {
			return t
		}
	}
	return MoTaTrangThai{Ma: ma, Ten: ma, TenKhach: ma, Mau: "trung"}
}

func TrangThaiHopLe(ma string) bool {
	for _, t := range CacTrangThai {
		if t.Ma == ma {
			return true
		}
	}
	return false
}

// --- Kiểu dữ liệu ----------------------------------------------------

type DongTien struct {
	MaDichVu string `json:"ma_dich_vu"`
	Ten      string `json:"ten"`
	SoTien   int    `json:"so_tien"`
}

type Moc struct {
	Luc       string `json:"luc"`
	TrangThai string `json:"trang_thai"`
	Nguoi     string `json:"nguoi"`
	GhiChu    string `json:"ghi_chu"`
}

type Don struct {
	Ma      string `json:"ma"`
	Token   string `json:"token"` // để khách tra cứu, không đoán được
	Ngay    string `json:"ngay"`
	CapNhat string `json:"cap_nhat"`

	TrangThai   string `json:"trang_thai"`
	ThoPhuTrach string `json:"tho_phu_trach"`

	KhachTen    string `json:"khach_ten"`
	KhachLienHe string `json:"khach_lien_he"`

	VotHang   string `json:"vot_hang"`
	VotGiaTri int    `json:"vot_gia_tri"` // khách khai, để quyết định có nhận ship không
	TinhTrang string `json:"tinh_trang"`  // khách mô tả
	ChanDoan  string `json:"chan_doan"`   // thợ ghi sau khi soi

	CanTruocG float64 `json:"can_truoc_g"`
	CanSauG   float64 `json:"can_sau_g"`

	DongTien []DongTien `json:"dong_tien"`
	TongTien int        `json:"tong_tien"`
	// Không có trường "đã thu" ở đây — xem DaThu() bên dưới.

	KenhNhan   string `json:"kenh_nhan"` // truc_tiep | ship
	HenTraNgay string `json:"hen_tra_ngay"`
	BaoHanhDen string `json:"bao_hanh_den"`

	Anh    []string `json:"anh"`
	GhiChu string   `json:"ghi_chu"`
	LichSu []Moc    `json:"lich_su"`
}

// DaThu — cộng các khoản thu trong sổ tiền có gắn mã đơn này. Cố tình KHÔNG
// lưu thành trường trong file đơn: một con số gõ tay ở đây cộng một sổ tiền ở
// kia là hai nguồn sự thật, và ngày chúng lệch nhau thì bên sai luôn là bên
// đang tính lãi lỗ. Ghi tiền khách trả ở trang đơn thực chất là tạo một khoản
// thu trong sổ.
func (d Don) DaThu() int { return ThuCuaDon(d.Ma) }

func (d Don) ConNo() int { return d.TongTien - d.DaThu() }

// AnhTruoc / AnhSau — nhãn nằm ngay trong tên file, do hQtTaiAnh đặt lúc
// tải lên ("truoc-150405-abc.webp"). Không có trường riêng trong JSON nên
// đọc lại từ tên; đơn cũ tải trước khi có nhãn thì rơi vào nhóm "trước",
// đúng như trang quản trị đang hiện.
func (d Don) AnhTruoc() []string { return d.locAnh(false) }
func (d Don) AnhSau() []string   { return d.locAnh(true) }

func (d Don) locAnh(sau bool) []string {
	out := []string{}
	for _, a := range d.Anh {
		if strings.HasPrefix(a, "sau-") == sau {
			out = append(out, a)
		}
	}
	return out
}

// ChenhCan — dương là vợt nặng lên sau khi sửa. Đây là cân, không phải
// tiền: Go được phép trừ.
func (d Don) ChenhCan() float64 {
	if d.CanTruocG <= 0 || d.CanSauG <= 0 {
		return 0
	}
	return d.CanSauG - d.CanTruocG
}

// VuotNguongCan — quá ngưỡng thì theo cam kết là không tính tiền công.
func (d Don) VuotNguongCan() bool {
	nguong := NguongHienTai().TangKhoiLuongToiDaG
	if nguong <= 0 {
		nguong = 3.0
	}
	return d.ChenhCan() > nguong
}

func (d Don) TrangThaiMoTa() MoTaTrangThai { return TrangThaiCua(d.TrangThai) }

// QuaHan — đã hẹn trả mà chưa giao xong.
func (d Don) QuaHan() bool {
	if d.HenTraNgay == "" || !TrangThaiCua(d.TrangThai).DangChay {
		return false
	}
	t, err := time.Parse("2006-01-02", d.HenTraNgay)
	if err != nil {
		return false
	}
	return time.Now().After(t.AddDate(0, 0, 1))
}

// --- Kho đơn ---------------------------------------------------------

var (
	donMu sync.RWMutex
	donDs = map[string]*Don{}
)

func thuMucDon() string { return P("data/don-hang") }

func NapDon() error {
	dir := thuMucDon()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	moi := map[string]*Don{}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var d Don
		if json.Unmarshal(b, &d) != nil || d.Ma == "" {
			continue
		}
		moi[d.Ma] = &d
	}
	donMu.Lock()
	donDs = moi
	donMu.Unlock()
	return nil
}

func LuuDon(d *Don) error {
	d.CapNhat = time.Now().Format("2006-01-02 15:04:05")
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}
	if err := ghiAtomic(filepath.Join(thuMucDon(), d.Ma+".json"), b); err != nil {
		return err
	}
	donMu.Lock()
	donDs[d.Ma] = d
	donMu.Unlock()
	return nil
}

func LayDon(ma string) (*Don, bool) {
	donMu.RLock()
	defer donMu.RUnlock()
	d, co := donDs[ma]
	if !co {
		return nil, false
	}
	ban := *d
	return &ban, true
}

// LayDonTheoToken tìm đơn bằng token ngẫu nhiên sinh lúc tạo đơn. Dùng cho
// link ảnh gửi cho khách: khách không đăng nhập được, mà bắt gõ lại mã + số
// điện thoại cho từng thẻ <img> thì không có cách nào. Token rỗng luôn
// không khớp — nếu không chặn, đơn cũ chưa có token sẽ mở ra cho mọi người.
func LayDonTheoToken(token string) (*Don, bool) {
	if token == "" {
		return nil, false
	}
	donMu.RLock()
	defer donMu.RUnlock()
	for _, d := range donDs {
		if d.Token == token {
			ban := *d
			return &ban, true
		}
	}
	return nil, false
}

// MaDonMoi — TV-2609-001: TV + năm 2 số + tháng 2 số + số thứ tự trong
// tháng. Đủ ngắn để khách đọc qua điện thoại, đủ dài để không trùng.
func MaDonMoi() string {
	now := time.Now()
	tien := fmt.Sprintf("TV-%s-", now.Format("0601"))
	donMu.RLock()
	max := 0
	for ma := range donDs {
		if !strings.HasPrefix(ma, tien) {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(ma, tien)); err == nil && n > max {
			max = n
		}
	}
	donMu.RUnlock()
	return fmt.Sprintf("%s%03d", tien, max+1)
}

type BoLoc struct {
	TrangThai   string
	Tho         string
	Tim         string
	ChiDangChay bool
}

func LocDon(f BoLoc) []*Don {
	tim := strings.ToLower(strings.TrimSpace(f.Tim))
	donMu.RLock()
	out := []*Don{}
	for _, d := range donDs {
		if f.TrangThai != "" && d.TrangThai != f.TrangThai {
			continue
		}
		if f.Tho != "" && d.ThoPhuTrach != f.Tho {
			continue
		}
		if f.ChiDangChay && !TrangThaiCua(d.TrangThai).DangChay {
			continue
		}
		if tim != "" {
			gop := strings.ToLower(strings.Join([]string{
				d.Ma, d.KhachTen, d.KhachLienHe, d.VotHang, d.TinhTrang, d.ChanDoan,
			}, " "))
			if !strings.Contains(gop, tim) {
				continue
			}
		}
		ban := *d
		out = append(out, &ban)
	}
	donMu.RUnlock()
	// Mới nhất lên đầu.
	sort.Slice(out, func(i, j int) bool { return out[i].Ma > out[j].Ma })
	return out
}

// TraCuuChoKhach — mã đơn cộng 4 số cuối điện thoại. Chỉ mã đơn thôi thì
// đoán được: TV-2609-001, TV-2609-002... và đơn của người khác mở ra hết.
func TraCuuChoKhach(ma, bonSoCuoi string) (*Don, bool) {
	d, co := LayDon(strings.ToUpper(strings.TrimSpace(ma)))
	if !co {
		return nil, false
	}
	so := chiSo(d.KhachLienHe)
	nhap := chiSo(bonSoCuoi)
	if len(so) < 4 || len(nhap) < 4 || !strings.HasSuffix(so, nhap[len(nhap)-4:]) {
		return nil, false
	}
	return d, true
}

func chiSo(s string) string {
	var b strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			b.WriteRune(c)
		}
	}
	return b.String()
}

// CongDongTien — cộng sổ, không định giá. Xem ghi chú đầu file.
// MaGiamGia đánh dấu dòng giảm giá trong DongTien. Để nó nằm chung với các
// dòng tiền khác chứ không tách thành một trường riêng: tổng đơn khi ấy vẫn
// là phép cộng của đúng một danh sách, và phiếu in ra cho khách thấy được
// giảm bao nhiêu, giảm vì cái gì. Dấu @ để không đụng mã dịch vụ thật.
const MaGiamGia = "@giam-gia"

// GiamGia trả số dương đang giảm (0 nếu không giảm).
func (d Don) GiamGia() int {
	for _, dt := range d.DongTien {
		if dt.MaDichVu == MaGiamGia {
			return -dt.SoTien
		}
	}
	return 0
}

// LyDoGiam — phần chữ sau "Giảm giá — " trong tên dòng.
func (d Don) LyDoGiam() string {
	for _, dt := range d.DongTien {
		if dt.MaDichVu == MaGiamGia {
			return strings.TrimPrefix(strings.TrimPrefix(dt.Ten, tenGiamGia), " — ")
		}
	}
	return ""
}

const tenGiamGia = "Giảm giá"

func CongDongTien(dong []DongTien) int {
	t := 0
	for _, d := range dong {
		t += d.SoTien
	}
	return t
}

func GhiMoc(d *Don, trangThai, nguoi, ghiChu string) {
	d.LichSu = append(d.LichSu, Moc{
		Luc:       time.Now().Format("2006-01-02 15:04"),
		TrangThai: trangThai,
		Nguoi:     nguoi,
		GhiChu:    ghiChu,
	})
}

// --- Thống kê --------------------------------------------------------

type OThongKe struct {
	Ma  string
	Ten string
	So  int
}

type ThangDoanhThu struct {
	Thang    string `json:"thang"`
	SoDon    int    `json:"so_don"`
	DoanhThu int    `json:"doanh_thu"`
	DaThu    int    `json:"da_thu"`
}

type ThongKeDon struct {
	TongDon       int             `json:"tong_don"`
	DangChay      int             `json:"dang_chay"`
	QuaHan        int             `json:"qua_han"`
	TheoTrangThai []OThongKe      `json:"-"`
	DoanhThuThang []ThangDoanhThu `json:"doanh_thu_thang"`
	ConNoTong     int             `json:"con_no_tong"`
	TyLeTuChoi    int             `json:"ty_le_tu_choi"` // phần trăm
}

func LayThongKeDon() ThongKeDon {
	donMu.RLock()
	ds := make([]*Don, 0, len(donDs))
	for _, d := range donDs {
		ds = append(ds, d)
	}
	donMu.RUnlock()

	tk := ThongKeDon{TongDon: len(ds)}
	dem := map[string]int{}
	theoThang := map[string]*ThangDoanhThu{}
	tuChoi := 0

	for _, d := range ds {
		dem[d.TrangThai]++
		mt := TrangThaiCua(d.TrangThai)
		if mt.DangChay {
			tk.DangChay++
			tk.ConNoTong += d.ConNo()
		}
		if d.QuaHan() {
			tk.QuaHan++
		}
		if d.TrangThai == TTTuChoi {
			tuChoi++
		}
		// Doanh thu tính trên đơn ĐÃ GIAO. Đơn đang sửa chưa phải tiền.
		if d.TrangThai == TTDaGiao && len(d.Ngay) >= 7 {
			th := d.Ngay[:7]
			if theoThang[th] == nil {
				theoThang[th] = &ThangDoanhThu{Thang: th}
			}
			theoThang[th].SoDon++
			theoThang[th].DoanhThu += d.TongTien
			theoThang[th].DaThu += d.DaThu()
		}
	}

	for _, t := range CacTrangThai {
		tk.TheoTrangThai = append(tk.TheoTrangThai, OThongKe{t.Ma, t.Ten, dem[t.Ma]})
	}
	for _, v := range theoThang {
		tk.DoanhThuThang = append(tk.DoanhThuThang, *v)
	}
	sort.Slice(tk.DoanhThuThang, func(i, j int) bool {
		return tk.DoanhThuThang[i].Thang > tk.DoanhThuThang[j].Thang
	})
	if len(tk.DoanhThuThang) > 12 {
		tk.DoanhThuThang = tk.DoanhThuThang[:12]
	}
	if tk.TongDon > 0 {
		tk.TyLeTuChoi = tuChoi * 100 / tk.TongDon
	}
	return tk
}
