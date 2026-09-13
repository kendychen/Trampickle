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
	// TTKhamAnh: yêu cầu từ web vừa được nhận, thợ đang soi ẢNH khách gửi để
	// báo giá — vợt vẫn còn ở nhà khách.
	//
	// Vì sao không dùng chung TTKiemTra: hai việc khác nhau hẳn. TTKiemTra là
	// cầm cây vợt thật trên tay, gõ nghe tiếng, cân lên; kết luận ở đó là
	// chốt. Khám qua ảnh chỉ ra được con số ƯỚC, và câu nói với khách phải
	// kèm "giá có thể đổi khi vợt tới tay". Trộn hai thứ vào một trạng thái
	// thì nhìn danh sách không biết cái nào đã sờ vào vợt, cái nào chưa.
	TTKhamAnh = "kham_anh"

	// TTChoHangVe: khách đã đặt trên web, kiện chưa tới tay trạm. Tách khỏi
	// "mới nhận" vì hai thứ này xử lý khác hẳn — cái này không có gì trên bàn
	// để làm, chỉ có một cái hẹn. Trộn chung thì danh sách việc đầy đơn không
	// làm được gì, và cái nào lâu không thấy hàng cũng không lộ ra.
	TTChoHangVe = "cho_hang_ve"

	TTMoi     = "moi"      // vừa nhận, chưa ai xem
	TTKiemTra = "kiem_tra" // đang soi vợt, chưa báo giá
	TTBaoGia  = "bao_gia"  // đã báo giá, chờ khách gật
	TTDangSua = "dang_sua"
	// Hai mốc của hàng gửi ra tiệm ngoài. Tách khỏi "đang sửa" vì nhìn danh
	// sách phải biết ngay cái nào còn trong tầm tay mình, cái nào đang nằm ở
	// chỗ người khác — hai thứ đó xử lý khác nhau khi khách gọi hỏi.
	TTDaGuiDi  = "da_gui_di"
	TTDaNhanVe = "da_nhan_ve"
	TTXong     = "xong"    // sửa xong, chờ giao
	TTDaGiao   = "da_giao" // hoàn tất
	TTTuChoi   = "tu_choi" // từ chối theo checklist
	TTHuy      = "huy"     // khách đổi ý
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
	{TTChoHangVe, "Chờ hàng về", "Đã nhận đơn, đang chờ hàng của anh/chị tới", "cho", true},
	// Khám qua ảnh đứng ngay sau: cũng là đơn chưa có đồ trong tay, nhưng
	// chờ hàng về vẫn phải xem trước — xem TestChoHangVeDungDauDanhSach.
	{TTKhamAnh, "Đang khám qua ảnh", "Thợ đang xem ảnh anh/chị gửi để báo giá", "trung", true},
	{TTMoi, "Mới nhận", "Đã nhận vợt, đang xếp lịch kiểm tra", "trung", true},
	{TTKiemTra, "Đang kiểm tra", "Thợ đang kiểm tra để xác định hư hỏng", "trung", true},
	{TTBaoGia, "Chờ khách duyệt giá", "Đã báo giá, đang chờ anh/chị xác nhận", "cho", true},
	{TTDangSua, "Đang sửa", "Vợt đang được sửa", "cho", true},
	// Câu cho khách cố tình trung tính: khách gửi đồ cho trạm, không cần biết
	// công đoạn nào trạm làm và công đoạn nào trạm thuê ngoài.
	{TTDaGuiDi, "Đã gửi đi gia công", "Đang được xử lý", "cho", true},
	{TTDaNhanVe, "Đã nhận về, đang kiểm", "Đang được kiểm tra lần cuối", "cho", true},
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

// --- Loại đơn --------------------------------------------------------
//
// Vợt và giày quản lý ở hai trang riêng, mã đơn riêng, form riêng. Nhưng vẫn
// một kho, một struct: Don đang bị bốn chỗ ngoài trang quản trị dùng chung —
// tra cứu khách (server.go), đối chiếu sao kê (saoke.go), tổng quan tiền
// (sotien.go), mail hằng ngày (tudong.go). Tách đôi kho thì mỗi chỗ ấy phải
// hỏi hai nơi rồi gộp, và ngày ai đó quên một chỗ thì khách gõ mã TG- vào ô
// tra cứu sẽ nhận "không có đơn này".
const (
	LoaiVot  = "vot"
	LoaiGiay = "giay"
)

// Hai cách khách trả tiền cho đơn gửi từ xa.
const (
	TraQR  = "qr"
	TraCOD = "cod"
)

// Đơn vào hệ bằng đường nào. Rỗng đọc là tay — mọi đơn có trước cửa hàng.
const (
	NguonWeb = "web"
	NguonTay = "tay"
)

// LoaiHopLe chuẩn hoá: rỗng đọc là vợt, vì đơn tạo trước ngày có giày không
// có trường này trong file JSON.
func LoaiHopLe(l string) string {
	if l == LoaiGiay {
		return LoaiGiay
	}
	return LoaiVot
}

func TenLoaiDon(l string) string {
	if LoaiHopLe(l) == LoaiGiay {
		return "giày"
	}
	return "vợt"
}

// Hai kiểu giày duy nhất trạm nhận. Xem dieu_kien của THAY_DE_GIAY trong
// bang-gia.yaml: giày court / cầu lông / pickleball / tennis bị TỪ CHỐI vì
// đế sai làm tăng hệ số ma sát xoay.
const (
	GiayChayBo = "chay_bo"
	GiayDiLai  = "di_lai"
)

var CacKieuGiay = []struct{ Ma, Ten string }{
	{GiayChayBo, "Giày chạy bộ"},
	{GiayDiLai, "Giày đi lại"},
}

func TenKieuGiay(ma string) string {
	for _, k := range CacKieuGiay {
		if k.Ma == ma {
			return k.Ten
		}
	}
	return ""
}

// GuiDi — ca gửi ra tiệm ngoài rồi ăn phần chênh. Dùng cho cả đơn vợt (phủ
// nhám, sơn lại) lẫn đơn giày (thay đế).
//
// TraDoiTac là số GÕ TAY, không phải số máy tính ra. Tiền trả tiệm đổi theo
// từng đôi — chính bang-gia.yaml đã ghi thế khi để bao_gia_rieng: true cho
// THAY_DE_GIAY. Gõ một tỷ lệ % rồi để máy suy ra tiền trả tiệm là bịa ra một
// con số không ai chốt với ai.
type GuiDi struct {
	MaDoiTac   string `json:"ma_doi_tac"` // rỗng = tự làm tại trạm
	NgayGui    string `json:"ngay_gui"`
	NgayHenVe  string `json:"ngay_hen_ve"`
	NgayVe     string `json:"ngay_ve"`
	TraDoiTac  int    `json:"tra_doi_tac"`
	DaTra      bool   `json:"da_tra"`
	MaKhoanChi string `json:"ma_khoan_chi"` // khoản chi đã sinh trong sổ tiền
	GhiChu     string `json:"ghi_chu"`
}

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

	// Loai rỗng đọc là vợt — đơn cũ không có trường này. Đừng so sánh thẳng
	// với LoaiVot ở đâu cả, dùng d.LaGiay().
	Loai string `json:"loai"`

	TrangThai   string `json:"trang_thai"`
	ThoPhuTrach string `json:"tho_phu_trach"`

	// MaKhach trỏ sang hồ sơ ở data/khach-hang.yaml. Rỗng với đơn cũ, và
	// KhachTen/KhachLienHe bên dưới KHÔNG bỏ đi: đơn là chứng từ, nó phải đọc
	// được đúng những gì đã ghi lúc nhận, kể cả sau này khách đổi tên đổi số.
	MaKhach string `json:"ma_khach"`

	// MaDonGoc: đơn này là đơn bảo hành của đơn nào. Rỗng với đơn thường.
	// Việc 3 dựng phần còn lại (nút mở đơn bảo hành, thống kê tỷ lệ); ở đây
	// nó đã cần rồi vì đơn bảo hành không tính là một lần ghé mới và không
	// được giảm giá — xem GoiYGiamGia.
	MaDonGoc    string `json:"ma_don_goc"`
	KhachTen    string `json:"khach_ten"`
	KhachLienHe string `json:"khach_lien_he"`
	// KhachEmail không bắt buộc, y như ô email ngoài biểu mẫu web: nhiều bác
	// chơi pickleball chỉ dùng Zalo. Có thì gửi được báo giá và hoá đơn về
	// hộp thư, không có thì thợ nhắn Zalo tay.
	KhachEmail string `json:"khach_email"`

	VotHang   string `json:"vot_hang"`
	VotGiaTri int    `json:"vot_gia_tri"` // khách khai, để quyết định có nhận ship không
	TinhTrang string `json:"tinh_trang"`  // khách mô tả
	ChanDoan  string `json:"chan_doan"`   // thợ ghi sau khi soi

	GiayHang  string `json:"giay_hang"`
	GiaySize  string `json:"giay_size"`
	GiayKieu  string `json:"giay_kieu"`   // chay_bo | di_lai
	DeHienTai string `json:"de_hien_tai"` // đế đang đi, mức mòn

	CanTruocG float64 `json:"can_truoc_g"`
	CanSauG   float64 `json:"can_sau_g"`

	GuiDi GuiDi `json:"gui_di"`

	DongTien []DongTien `json:"dong_tien"`
	TongTien int        `json:"tong_tien"`
	// Không có trường "đã thu" ở đây — xem DaThu() bên dưới.

	KenhNhan string `json:"kenh_nhan"` // truc_tiep | ship
	// Địa chỉ ship trả về. Rỗng với đơn khách tới lấy.
	KhachDiaChi string `json:"khach_dia_chi"`
	// qr | cod. Rỗng = khách chưa chọn.
	HinhThucTra string `json:"hinh_thuc_tra"`
	// Vận đơn hai chiều, GÕ TAY. Không nối API hãng vận chuyển — xem spec.
	MaVanDonDen string `json:"ma_van_don_den"`
	MaVanDonVe  string `json:"ma_van_don_ve"`
	// web | tay. Rỗng đọc là tay.
	NguonDon string `json:"nguon_don"`

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

// NgayGiao — ngày đơn này thành tiền, đọc ngược lịch sử tìm mốc "đã giao"
// gần nhất. Doanh thu tính theo ngày này chứ không theo ngày nhận đơn: đơn
// nhận cuối tháng 8, giao giữa tháng 9 thì tiền là của tháng 9.
//
// Rỗng nếu đơn chưa giao. Đơn cũ không có mốc nào (lịch sử trống) thì trả
// ngày nhận — thà tính vào tháng nhận còn hơn biến mất khỏi mọi bảng.
func (d Don) NgayGiao() string {
	if d.TrangThai != TTDaGiao {
		return ""
	}
	for i := len(d.LichSu) - 1; i >= 0; i-- {
		if d.LichSu[i].TrangThai != TTDaGiao {
			continue
		}
		if ng := ngayCuaMoc(d.LichSu[i].Luc); ng != "" {
			return ng
		}
	}
	return d.Ngay
}

// ngayCuaMoc cắt phần ngày khỏi dấu thời gian của một mốc. GhiMoc ghi bằng
// time.Now().Format(...) nên 10 ký tự đầu luôn là YYYY-MM-DD; kiểm lại độ
// dài cho chắc, file đơn là JSON và có thể bị sửa tay.
func ngayCuaMoc(luc string) string {
	if len(luc) < 10 {
		return ""
	}
	ng := luc[:10]
	if _, err := time.Parse("2006-01-02", ng); err != nil {
		return ""
	}
	return ng
}

// NguonDonHopLe — rỗng đọc là nhận tay, y như Loai rỗng đọc là vợt.
func (d Don) NguonDonHopLe() string {
	if d.NguonDon == NguonWeb {
		return NguonWeb
	}
	return NguonTay
}

func (d Don) ChoHangVe() bool { return d.TrangThai == TTChoHangVe }

func (d Don) LaGiay() bool    { return LoaiHopLe(d.Loai) == LoaiGiay }
func (d Don) TenLoai() string { return TenLoaiDon(d.Loai) }

// TenMon — thứ khách gửi tới, một dòng để hiện trong bảng và trong mail.
func (d Don) TenMon() string {
	if d.LaGiay() {
		s := strings.TrimSpace(d.GiayHang)
		if d.GiaySize != "" {
			s = strings.TrimSpace(s + " · cỡ " + d.GiaySize)
		}
		return s
	}
	return d.VotHang
}

func (d Don) TenKieuGiay() string { return TenKieuGiay(d.GiayKieu) }

// GuiRaNgoai — đơn này có nhờ tiệm ngoài làm hay không.
func (d Don) GuiRaNgoai() bool { return d.GuiDi.MaDoiTac != "" }

// LaiThat — tổng đơn trừ tiền trả tiệm ngoài. Là phép TRỪ hai con số đã ghi
// vào sổ, cùng loại việc với ConNo(); không phải định giá.
func (d Don) LaiThat() int { return d.TongTien - d.GuiDi.TraDoiTac }

// NoDoiTac — đã nhận việc của tiệm mà chưa trả tiền tiệm.
func (d Don) NoDoiTac() int {
	if !d.GuiRaNgoai() || d.GuiDi.DaTra {
		return 0
	}
	return d.GuiDi.TraDoiTac
}

// QuaHenDoiTac — tiệm trễ. Khác QuaHan(): đó là mình trễ với khách.
func (d Don) QuaHenDoiTac() bool {
	if d.TrangThai != TTDaGuiDi || d.GuiDi.NgayHenVe == "" {
		return false
	}
	// ParseInLocation chứ không phải Parse: Parse hiểu "2026-09-08" là nửa đêm
	// theo GIỜ UTC, tức 7 giờ sáng ở Việt Nam. Đơn hẹn hôm qua sẽ không bị
	// tính là trễ trong suốt buổi đêm tới 7h — đúng lúc Kendy mở máy xem
	// việc còn tồn.
	t, err := time.ParseInLocation("2006-01-02", d.GuiDi.NgayHenVe, time.Local)
	if err != nil {
		return false
	}
	return time.Now().After(t.AddDate(0, 0, 1))
}

func (d Don) TenDoiTac() string { return TenDoiTac(d.GuiDi.MaDoiTac) }

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
// Chỉ áp cho vợt: cam kết không tăng quá 3g là cam kết về vợt, đôi giày nặng
// thêm bao nhiêu sau khi thay đế không nằm trong lời hứa nào.
func (d Don) VuotNguongCan() bool {
	if d.LaGiay() {
		return false
	}
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
	// Giờ địa phương — xem ghi chú ở QuaHenDoiTac.
	t, err := time.ParseInLocation("2006-01-02", d.HenTraNgay, time.Local)
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

// MaDonMoi — TV-2609-001: tiền tố + năm 2 số + tháng 2 số + số thứ tự trong
// tháng. Đủ ngắn để khách đọc qua điện thoại, đủ dài để không trùng. Tiền tố
// sửa được ở /qt/tram — xem TienToDon trong core/tram.go. Đếm theo từng tiền
// tố nên đổi tiền tố giữa tháng cũng không đụng vào mã đã phát ra.
func MaDonMoi(loai string) string {
	now := time.Now()
	tien := fmt.Sprintf("%s%s-", TienToDon(loai), now.Format("0601"))
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
	Loai        string // rỗng = cả vợt lẫn giày
	TrangThai   string
	Tho         string
	DoiTac      string
	Tim         string
	ChiDangChay bool
	// ChuaThuDu: đơn còn nợ tiền. Tính từ SỔ TIỀN qua ConNo() — cố tình không
	// có trường "đã trả" trên đơn để đọc. Xem chú thích trên Don.DaThu.
	ChuaThuDu bool
	// ChiXongBoQuen: đơn sửa xong mà không ai tới lấy — xem core/boquen.go.
	ChiXongBoQuen bool
	// ConBaoHanh: đơn đã giao mà hạn bảo hành chưa qua. Rổ này để trả lời
	// "cây này còn bảo hành không" lúc khách cầm đồ tới càu nhàu.
	ConBaoHanh bool
	// Tu/Den: khoảng ngày NHẬN đơn, hai đầu đều tính. Rỗng cả hai là không
	// giới hạn. Lọc theo ngày nhận vì bảng này trả lời "kỳ này nhận bao
	// nhiêu việc"; doanh thu là câu hỏi khác và đi theo NgayGiao().
	Tu  string
	Den string
}

func LocDon(f BoLoc) []*Don {
	tim := strings.ToLower(strings.TrimSpace(f.Tim))
	donMu.RLock()
	out := []*Don{}
	for _, d := range donDs {
		if f.Loai != "" && LoaiHopLe(d.Loai) != LoaiHopLe(f.Loai) {
			continue
		}
		if f.DoiTac != "" && d.GuiDi.MaDoiTac != f.DoiTac {
			continue
		}
		if f.TrangThai != "" && d.TrangThai != f.TrangThai {
			continue
		}
		if f.Tho != "" && d.ThoPhuTrach != f.Tho {
			continue
		}
		if f.ChiDangChay && !TrangThaiCua(d.TrangThai).DangChay {
			continue
		}
		if f.ChuaThuDu && d.ConNo() <= 0 {
			continue
		}
		if f.ChiXongBoQuen && !d.XongBoQuen(time.Now(), NguongTramHienTai().XongBoQuenNgay) {
			continue
		}
		if f.ConBaoHanh && !d.ConBaoHanh() {
			continue
		}
		if !trongKhoang(d.Ngay, f.Tu, f.Den) {
			continue
		}
		if tim != "" {
			gop := strings.ToLower(strings.Join([]string{
				d.Ma, d.KhachTen, d.KhachLienHe, d.VotHang,
				d.GiayHang, d.GiaySize, d.TinhTrang, d.ChanDoan,
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
	// OTiemNgoai: đang nằm ở chỗ người khác, gọi điện mới biết bao giờ xong.
	OTiemNgoai   int `json:"o_tiem_ngoai"`
	QuaHenDoiTac int `json:"qua_hen_doi_tac"`
	NoDoiTacTong int `json:"no_doi_tac_tong"`
}

// LayThongKeDon — loai rỗng là đếm cả vợt lẫn giày.
func LayThongKeDon(loai string) ThongKeDon {
	donMu.RLock()
	ds := make([]*Don, 0, len(donDs))
	for _, d := range donDs {
		if loai != "" && LoaiHopLe(d.Loai) != LoaiHopLe(loai) {
			continue
		}
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
		if d.TrangThai == TTDaGuiDi {
			tk.OTiemNgoai++
		}
		if d.QuaHenDoiTac() {
			tk.QuaHenDoiTac++
		}
		tk.NoDoiTacTong += d.NoDoiTac()
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
