// Hồ sơ khách hàng: gộp mọi lần một người ghé trạm về một chỗ.
//
// Ghi ra data/khach-hang.yaml theo đúng lối data/doi-tac.yaml — data/ không
// bao giờ bị rsync đè khi deploy, nên danh sách gõ trên máy chủ sống qua mọi
// lần đẩy binary. Chưa có file không phải lỗi, chỉ là chưa có ai ghé.
//
// VÌ SAO SoChuan LÀ KHOÁ TRA, KHÔNG PHẢI LienHe. Khách đọc số kiểu
// "0846 161 368", "+84846161368", "084.616.1368" — ba cách gõ cùng một
// người. Tra theo nguyên văn thì mỗi lần thợ gõ khác dấu cách là đẻ ra một
// bác Minh mới, và câu hỏi thật — bác này đã tới mấy lần, còn nợ bao nhiêu —
// không bao giờ trả lời được. LienHe vẫn giữ nguyên văn để hiện đúng cái thợ
// đã gõ.
//
// VÌ SAO KHÁCH SINH RA TỪ ĐƠN. Thợ đang cầm cây vợt và một ông khách đứng
// đợi. Bắt mở thêm một trang "thêm khách mới" trước khi lập được đơn thì
// trang ấy sẽ bị bỏ qua, và sáu tháng sau danh sách khách rỗng trong khi sổ
// đơn đầy. Lưu đơn xong thì GhiNhanKhach tự dựng hồ sơ.
//
// VÌ SAO CÔNG NỢ VÀ TỔNG CHI TIÊU KHÔNG LƯU THÀNH TRƯỜNG. Đây là luật đã có
// ở đầu core/donhang.go: một con số gõ tay ở đây cộng một sổ tiền ở kia là
// hai nguồn sự thật, và ngày chúng lệch nhau thì bên sai luôn là bên người ta
// tin. Tính lúc đọc, cộng từ Don.DaThu()/Don.ConNo().
//
// VÌ SAO BA NHÓM KHAI CỨNG. Ba là đủ để lọc mà vẫn nhớ được. Cho tự thêm
// nhóm thì sáu tháng nữa có mười hai nhóm chồng nghĩa nhau và không lọc nổi
// cái gì.
package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type KhachHang struct {
	Ma      string `yaml:"ma" json:"ma"` // K0001
	Ten     string `yaml:"ten" json:"ten"`
	SoChuan string `yaml:"so_chuan" json:"so_chuan"` // chỉ chữ số, khoá tra
	LienHe  string `yaml:"lien_he" json:"lien_he"`   // nguyên văn thợ gõ
	Email   string `yaml:"email" json:"email"`
	DiaChi  string `yaml:"dia_chi" json:"dia_chi"`
	Nhom    string `yaml:"nhom" json:"nhom"` // le | than | clb
	GhiChu  string `yaml:"ghi_chu" json:"ghi_chu"`
	LanDau  string `yaml:"lan_dau" json:"lan_dau"` // ISO, ngày đơn đầu
	LanCuoi string `yaml:"lan_cuoi" json:"lan_cuoi"`
	// Ngung: ẩn khỏi ô gợi ý lúc lập đơn. Không xoá hẳn — lịch sử đơn trỏ vào
	// hồ sơ này.
	Ngung bool `yaml:"ngung" json:"ngung"`
}

const (
	NhomKhachLe   = "le"
	NhomKhachThan = "than"
	NhomKhachClb  = "clb"
)

type NhomKhach struct {
	Ma  string
	Ten string
	Mau string // chip-<mau> trong css.html
}

var CacNhomKhach = []NhomKhach{
	{NhomKhachLe, "Khách lẻ", "trung"},
	{NhomKhachThan, "Khách quen", "nhan"},
	{NhomKhachClb, "CLB / đội nhóm", "cho"},
}

func TimNhomKhach(ma string) (NhomKhach, bool) {
	for _, n := range CacNhomKhach {
		if n.Ma == ma {
			return n, true
		}
	}
	return NhomKhach{}, false
}

func (k KhachHang) TenNhom() string {
	if n, co := TimNhomKhach(k.Nhom); co {
		return n.Ten
	}
	return "Khách lẻ"
}

func (k KhachHang) MauNhom() string {
	if n, co := TimNhomKhach(k.Nhom); co {
		return n.Mau
	}
	return "trung"
}

type khoKhachFile struct {
	KhachHang []KhachHang `yaml:"khach_hang"`
}

var (
	khachMu sync.RWMutex
	khachDs []KhachHang
)

func fileKhach() string { return P("data/khach-hang.yaml") }

// ChuanSo bóc hết ký tự không phải chữ số, rồi 84 đầu chuỗi 11 số đổi về 0.
// Chỉ đổi khi đúng 11 số: "84" đứng đầu một số nội địa 10 chữ số là số thật
// của người ta, không phải mã nước.
func ChuanSo(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	so := b.String()
	if strings.HasPrefix(so, "84") && len(so) == 11 {
		so = "0" + so[2:]
	}
	return so
}

// NapKhach đọc danh sách. Chưa có file không phải lỗi.
func NapKhach() error {
	b, err := os.ReadFile(fileKhach())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var f khoKhachFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("%s hỏng: %w", fileKhach(), err)
	}
	khachMu.Lock()
	khachDs = f.KhachHang
	khachMu.Unlock()
	return nil
}

// ghiKhach — gọi khi ĐANG giữ khachMu.
func ghiKhach() error {
	b, err := yaml.Marshal(khoKhachFile{KhachHang: khachDs})
	if err != nil {
		return err
	}
	dau := []byte("# Hồ sơ khách của trạm. Sinh tự động từ đơn, sửa ở /qt/khach.\n\n")
	return ghiAtomic(fileKhach(), append(dau, b...))
}

// maKhachMoi — gọi khi ĐANG giữ khachMu.
func maKhachMoi() string {
	max := 0
	for _, k := range khachDs {
		var n int
		if _, err := fmt.Sscanf(k.Ma, "K%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("K%04d", max+1)
}

func KhachTheoSo(so string) (KhachHang, bool) {
	sc := ChuanSo(so)
	if sc == "" {
		return KhachHang{}, false
	}
	khachMu.RLock()
	defer khachMu.RUnlock()
	for _, k := range khachDs {
		if k.SoChuan == sc {
			return k, true
		}
	}
	return KhachHang{}, false
}

func KhachTheoMa(ma string) (KhachHang, bool) {
	if ma == "" {
		return KhachHang{}, false
	}
	khachMu.RLock()
	defer khachMu.RUnlock()
	for _, k := range khachDs {
		if k.Ma == ma {
			return k, true
		}
	}
	return KhachHang{}, false
}

// TenKhachHang trả về mã khi không tìm thấy: chỗ nào đã trỏ tới hồ sơ thì
// phải hiện ra một cái gì đọc được, không phải ô trống.
func TenKhachHang(ma string) string {
	if ma == "" {
		return ""
	}
	if k, co := KhachTheoMa(ma); co {
		return k.Ten
	}
	return ma
}

// GhiNhanKhach dựng hoặc cập nhật hồ sơ từ một đơn vừa lưu. Trả về mã khách,
// rỗng nếu không có số điện thoại — không có khoá tra thì hồ sơ ấy chỉ là một
// dòng rác không bao giờ gộp lại được với ai.
//
// Lần sau có thêm email/địa chỉ thì điền vào ô đang TRỐNG, không đè cái đã
// có: thợ sửa tay ở trang khách rồi mà một đơn cũ gõ vội lại ghi đè lên thì
// công sửa ấy mất.
func GhiNhanKhach(ten, lienHe, email, diaChi string) string {
	so := ChuanSo(lienHe)
	if so == "" {
		return ""
	}
	homNay := time.Now().Format("2006-01-02")
	ten = strings.TrimSpace(ten)
	email = strings.TrimSpace(email)
	diaChi = strings.TrimSpace(diaChi)

	khachMu.Lock()
	defer khachMu.Unlock()
	for i := range khachDs {
		if khachDs[i].SoChuan != so {
			continue
		}
		k := &khachDs[i]
		if k.Ten == "" {
			k.Ten = ten
		}
		if k.LienHe == "" {
			k.LienHe = strings.TrimSpace(lienHe)
		}
		if k.Email == "" {
			k.Email = email
		}
		if k.DiaChi == "" {
			k.DiaChi = diaChi
		}
		if k.LanDau == "" || homNay < k.LanDau {
			k.LanDau = homNay
		}
		if homNay > k.LanCuoi {
			k.LanCuoi = homNay
		}
		ma := k.Ma
		_ = ghiKhach()
		return ma
	}

	k := KhachHang{
		Ma:      maKhachMoi(),
		Ten:     ten,
		SoChuan: so,
		LienHe:  strings.TrimSpace(lienHe),
		Email:   email,
		DiaChi:  diaChi,
		Nhom:    NhomKhachLe,
		LanDau:  homNay,
		LanCuoi: homNay,
	}
	khachDs = append(khachDs, k)
	_ = ghiKhach()
	return k.Ma
}

// LuuKhach ghi đè một hồ sơ đã có (thợ sửa tay ở trang khách). Mã rỗng là
// thêm mới.
func LuuKhach(k KhachHang) (string, error) {
	k.Ten = strings.TrimSpace(k.Ten)
	k.LienHe = strings.TrimSpace(k.LienHe)
	k.SoChuan = ChuanSo(k.LienHe)
	if k.SoChuan == "" {
		return "", fmt.Errorf("phải có số điện thoại")
	}
	if _, co := TimNhomKhach(k.Nhom); !co {
		k.Nhom = NhomKhachLe
	}

	khachMu.Lock()
	defer khachMu.Unlock()
	// Đổi số sang trùng người khác thì chặn — gộp hai hồ sơ là việc phải làm
	// có ý thức, không phải hậu quả phụ của một lần sửa số.
	for _, cu := range khachDs {
		if cu.SoChuan == k.SoChuan && cu.Ma != k.Ma {
			return "", fmt.Errorf("số %s đã thuộc về %s (%s)", k.LienHe, cu.Ten, cu.Ma)
		}
	}
	if k.Ma != "" {
		for i := range khachDs {
			if khachDs[i].Ma != k.Ma {
				continue
			}
			k.LanDau = khachDs[i].LanDau
			k.LanCuoi = khachDs[i].LanCuoi
			khachDs[i] = k
			return k.Ma, ghiKhach()
		}
	}
	k.Ma = maKhachMoi()
	if k.LanDau == "" {
		k.LanDau = time.Now().Format("2006-01-02")
	}
	khachDs = append(khachDs, k)
	return k.Ma, ghiKhach()
}

// DatNhomKhach đổi nhóm một khách. Nhóm lạ thì thôi, không dựng nhóm mới.
func DatNhomKhach(ma, nhom string) error {
	if _, co := TimNhomKhach(nhom); !co {
		return fmt.Errorf("không có nhóm %q", nhom)
	}
	khachMu.Lock()
	defer khachMu.Unlock()
	for i := range khachDs {
		if khachDs[i].Ma == ma {
			khachDs[i].Nhom = nhom
			return ghiKhach()
		}
	}
	return fmt.Errorf("không thấy khách %s", ma)
}

// DanhSachKhach — bản sao cả danh sách, kể cả khách đã ngừng.
func DanhSachKhach() []KhachHang {
	khachMu.RLock()
	defer khachMu.RUnlock()
	return append([]KhachHang{}, khachDs...)
}

// KhachDangDung — dùng cho ô gợi ý lúc lập đơn.
func KhachDangDung() []KhachHang {
	out := []KhachHang{}
	for _, k := range DanhSachKhach() {
		if !k.Ngung {
			out = append(out, k)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].LanCuoi > out[j].LanCuoi })
	return out
}

// LichSuKhach — mọi đơn của một khách, mới nhất lên đầu.
//
// Đơn cũ chưa có MaKhach thì đối chiếu thêm bằng ChuanSo(KhachLienHe): hai
// đơn thật đang có trên server không mang trường mới, mà người ta vẫn phải
// thấy chúng trên trang khách.
func LichSuKhach(ma string) []*Don {
	k, co := KhachTheoMa(ma)
	if !co {
		return nil
	}
	out := []*Don{}
	for _, d := range LocDon(BoLoc{}) {
		if d.MaKhach == ma || (d.MaKhach == "" && ChuanSo(d.KhachLienHe) == k.SoChuan) {
			out = append(out, d)
		}
	}
	return out
}

type SoLieuKhach struct {
	SoDon     int
	SoDonDang int // đang chạy
	TongChi   int // tổng DaThu của mọi đơn
	ConNo     int // tổng ConNo của mọi đơn
	LanCuoi   string
	NgayVang  int // số ngày kể từ lần cuối
}

func SoLieuCuaKhach(ma string) SoLieuKhach {
	var s SoLieuKhach
	for _, d := range LichSuKhach(ma) {
		s.SoDon++
		if TrangThaiCua(d.TrangThai).DangChay {
			s.SoDonDang++
		}
		s.TongChi += d.DaThu()
		if no := d.ConNo(); no > 0 {
			s.ConNo += no
		}
		if d.Ngay > s.LanCuoi {
			s.LanCuoi = d.Ngay
		}
	}
	s.NgayVang = soNgayTuNgay(s.LanCuoi)
	return s
}

// soNgayTuNgay — số ngày từ một mốc ISO tới hôm nay. Mốc rỗng hoặc hỏng trả
// 0: "chưa biết" thì đừng bịa ra một con số để rồi lọt vào danh sách "vắng
// lâu quá".
func soNgayTuNgay(iso string) int {
	if iso == "" {
		return 0
	}
	t, err := time.ParseInLocation("2006-01-02", iso, time.Local)
	if err != nil {
		return 0
	}
	n := int(time.Since(t).Hours() / 24)
	if n < 0 {
		return 0
	}
	return n
}

// DongKhach — một dòng trên bảng /qt/khach: hồ sơ cộng số liệu tính tại chỗ.
type DongKhach struct {
	KhachHang
	SoLieuKhach
}

type BoLocKhach struct {
	Tim  string
	Nhom string
	// VangTuThang: chỉ giữ khách đã vắng quá ngần này tháng. 0 = không lọc.
	// Đây là lý do chính khiến trang khách đáng tồn tại — nó cho Kendy một
	// danh sách để nhắn Zalo.
	VangTuThang int
	ConNo       bool
}

func LocKhach(f BoLocKhach) []DongKhach {
	tim := strings.ToLower(strings.TrimSpace(f.Tim))
	timSo := ChuanSo(f.Tim)
	out := []DongKhach{}
	for _, k := range DanhSachKhach() {
		if f.Nhom != "" && k.Nhom != f.Nhom {
			continue
		}
		if tim != "" {
			gop := strings.ToLower(strings.Join([]string{k.Ma, k.Ten, k.LienHe, k.Email, k.GhiChu}, " "))
			khop := strings.Contains(gop, tim)
			if !khop && timSo != "" && strings.Contains(k.SoChuan, timSo) {
				khop = true
			}
			if !khop && strings.Contains(boDau(gop), boDau(tim)) {
				khop = true
			}
			if !khop {
				continue
			}
		}
		s := SoLieuCuaKhach(k.Ma)
		if f.ConNo && s.ConNo <= 0 {
			continue
		}
		if f.VangTuThang > 0 && s.NgayVang < f.VangTuThang*30 {
			continue
		}
		out = append(out, DongKhach{KhachHang: k, SoLieuKhach: s})
	}
	// Lần cuối gần nhất lên đầu; khách chưa có đơn nào xuống cuối.
	sort.Slice(out, func(i, j int) bool {
		if out[i].SoLieuKhach.LanCuoi != out[j].SoLieuKhach.LanCuoi {
			return out[i].SoLieuKhach.LanCuoi > out[j].SoLieuKhach.LanCuoi
		}
		return out[i].Ma < out[j].Ma
	})
	return out
}

// TongQuanKhach — bốn ô số đầu trang /qt/khach.
type TongQuanKhach struct {
	TongKhach     int
	MoiTrongThang int
	TongConNo     int
	SoKhachNo     int
}

func LayTongQuanKhach() TongQuanKhach {
	var t TongQuanKhach
	thang := time.Now().Format("2006-01")
	for _, k := range DanhSachKhach() {
		t.TongKhach++
		if strings.HasPrefix(k.LanDau, thang) {
			t.MoiTrongThang++
		}
		if s := SoLieuCuaKhach(k.Ma); s.ConNo > 0 {
			t.TongConNo += s.ConNo
			t.SoKhachNo++
		}
	}
	return t
}

// GoiYKhach là bản rút gọn của hồ sơ, vừa đủ cho ô gợi ý ở form lập đơn:
// những thứ điền được vào ô trống cộng ba con số thợ cần liếc qua. Ghi chú
// nội bộ không nằm ở đây — nó là chuyện của trang hồ sơ, không phải của một
// dòng chữ nhỏ dưới ô điện thoại.
type GoiYKhach struct {
	Ma      string `json:"ma"`
	So      string `json:"so"`
	Ten     string `json:"ten"`
	LienHe  string `json:"lien_he"`
	Email   string `json:"email"`
	DiaChi  string `json:"dia_chi"`
	Nhom    string `json:"nhom"`
	SoDon   int    `json:"so_don"`
	ConNo   int    `json:"con_no"`
	LanCuoi string `json:"lan_cuoi"`
}

// DanhSachGoiYKhach bỏ qua hồ sơ đã tắt: thợ vẫn gõ tay được số đó, chỉ là
// trạm thôi nhắc tới.
func DanhSachGoiYKhach() []GoiYKhach {
	ds := DanhSachKhach()
	ra := make([]GoiYKhach, 0, len(ds))
	for _, k := range ds {
		if k.Ngung {
			continue
		}
		s := SoLieuCuaKhach(k.Ma)
		g := GoiYKhach{
			Ma: k.Ma, So: k.SoChuan, Ten: k.Ten, LienHe: k.LienHe,
			Email: k.Email, DiaChi: k.DiaChi, Nhom: k.TenNhom(),
			SoDon: s.SoDon, ConNo: s.ConNo,
		}
		if len(s.LanCuoi) == 10 {
			g.LanCuoi = s.LanCuoi[8:10] + "/" + s.LanCuoi[5:7]
		}
		ra = append(ra, g)
	}
	return ra
}

// --- Giảm giá cho khách quen -----------------------------------------
//
// Không có cơ chế giảm giá mới nào ở đây. Ô giam_gia trên trang đơn đã đẩy
// một dòng âm mang mã MaGiamGia vào DongTien từ lâu; phần dưới chỉ đoán hộ
// con số để thợ khỏi phải nhớ và khỏi phải bấm máy tính.

// MucGiamCuaKhach trả phần trăm theo NHÓM khách. Gắn vào nhóm chứ không làm
// một luật riêng: sáu tháng nữa muốn cho CLB mức khác thì đổi một con số ở
// /qt/nguong, không phải sửa code.
func MucGiamCuaKhach(maKhach string) int {
	if maKhach == "" {
		return 0
	}
	k, co := KhachTheoMa(maKhach)
	if !co {
		return 0
	}
	n := NguongTramHienTai()
	switch k.Nhom {
	case NhomKhachThan:
		return n.GiamKhachQuenPhanTram
	case NhomKhachClb:
		return n.GiamClbPhanTram
	}
	return 0
}

// ThangNhomSauGiao đưa khách lẻ lên khách quen sau khi đã giao xong một đơn
// thật. Nhận cả đơn chứ không chỉ mã khách: luật "đơn nào tính, đơn nào
// không" phải nằm đúng một chỗ, chỗ gọi khỏi phải nhớ.
//
// Không tự hạ nhóm bao giờ. Khách vắng hai năm rồi quay lại vẫn là khách
// quen — hạ nhóm là cách nhanh nhất làm mất một người vừa mới quay về.
func ThangNhomSauGiao(d *Don) {
	switch {
	case d == nil, d.MaKhach == "":
		return
	case d.TrangThai != TTDaGiao:
		// Huỷ và từ chối không tính: khách huỷ không phải khách cũ.
		return
	case d.MaDonGoc != "":
		// Quay lại vì đồ hỏng lại không phải lần ghé thứ hai, đó là lần thứ
		// nhất chưa xong.
		return
	}
	k, co := KhachTheoMa(d.MaKhach)
	if !co || k.Nhom != NhomKhachLe {
		return
	}
	_ = DatNhomKhach(k.Ma, NhomKhachThan)
}

// GoiYGiamGia trả số tiền điền sẵn vào ô giảm giá, lý do, và cờ báo đã bị
// trần biên gộp kéo xuống.
//
// Trần biên gộp là chỗ nguy hiểm nhất: đơn thay đế thu khách 500k, trả tiệm
// ngoài 400k, trạm ăn 100k. Giảm 10% trên tổng là mất một nửa phần công. Vài
// đơn như thế là làm không công. Nên khi đơn có gửi tiệm, mức giảm bị hạ
// xuống vừa đủ chạm ngưỡng bien_gop_toi_thieu_dong — và nói ra chứ không
// lặng lẽ sửa số: thợ phải hiểu con số ở đâu ra, không thì lần sau họ gõ đè
// lên và cái trần thành vô dụng.
func GoiYGiamGia(d *Don) (soTien int, lyDo string, daHa bool) {
	if d == nil || d.MaDonGoc != "" {
		// Đơn bảo hành tổng 0. In một dòng "Giảm khách quen 0đ" lên hoá đơn
		// là chữ rác.
		return 0, "", false
	}
	pt := MucGiamCuaKhach(d.MaKhach)
	if pt <= 0 {
		return 0, "", false
	}
	goc := 0
	for _, dt := range d.DongTien {
		if dt.MaDichVu != MaGiamGia {
			goc += dt.SoTien
		}
	}
	if goc <= 0 {
		return 0, "", false
	}
	g := (goc*pt + 50) / 100
	if d.GuiDi.TraDoiTac > 0 {
		toiDa := goc - d.GuiDi.TraDoiTac - NguongHienTai().BienGopToiThieuDong
		if toiDa < 0 {
			toiDa = 0
		}
		if g > toiDa {
			g, daHa = toiDa, true
		}
	}
	if g <= 0 {
		return 0, "", daHa
	}
	return g, "Khách quen", daHa
}
