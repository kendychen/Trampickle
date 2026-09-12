// Đồ nghề: cân, kẹp, máy khoan, súng nhiệt — thứ mua một lần rồi dùng mãi.
//
// VÌ SAO KHÔNG DÙNG CHUNG VỚI KHO VẬT TƯ. VatTu có tồn, có định mức ăn theo mã
// dịch vụ, có phiếu nhập/xuất/kiểm kê. Ba thứ đó vô nghĩa với cái kẹp chữ C.
// Trộn chung thì cây kẹp lọt vào form "khớp định mức" của một đơn sửa viền,
// thợ bấm một cú là kho trừ đi một cái kẹp, tháng sau tồn kẹp âm mà không phiếu
// nào giải thích được. Ranh giới: dùng hết thì mua tiếp là VẬT TƯ (keo, nhám,
// grip); mua một lần dùng mãi là ĐỒ NGHỀ.
//
// TRẠNG THÁI MUA LƯU THẲNG TRONG YAML, khác hẳn luật của kho. Tồn kho là số dẫn
// xuất từ phiếu nên không được lưu; còn "cái cân mua ngày 12/09 giá 240k" là dữ
// kiện gốc, không phát lại được từ đâu cả. Không có sổ thứ hai để lệch.
//
// Nối với sổ tiền chỉ bằng MaKhoan. Số tiền thật nằm ở sổ; file này giữ bản sao
// để dựng bảng mà không phải nạp cả sổ. Hai bên lệch thì SỔ TIỀN ĐÚNG — báo cáo
// về vốn đọc sổ, không đọc file này.
package core

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// --- Kiểu dữ liệu ----------------------------------------------------

const (
	NgheVot   = "vot"
	NgheGiay  = "giay"
	NgheChung = "chung" // dùng cho cả hai nghề, không đếm hai lần

	// DoNgheDaCo là trạng thái dễ quên nhất và cũng cần nhất. Kéo, tua vít,
	// điện thoại, đèn bàn — đã có sẵn ở nhà. Thiếu trạng thái này thì mấy món
	// ấy hoặc nằm mãi ở cột "còn thiếu" (nhìn vào tưởng chưa làm nghề được),
	// hoặc bị đánh "đã mua" rồi đẻ ra khoản chi 0đ rác trong sổ.
	DoNgheChuaMua = "chua_mua"
	DoNgheDaCo    = "da_co"
	DoNgheDaMua   = "da_mua"
	DoNgheTamHoan = "tam_hoan"
)

type MoTaNghe struct {
	Ma  string
	Ten string
}

var CacNghe = []MoTaNghe{
	{NgheVot, "Sửa vợt"},
	{NgheGiay, "Sửa giày"},
	{NgheChung, "Dùng chung"},
}

type MoTaTrangThaiDoNghe struct {
	Ma  string
	Ten string
	// SinhKhoan: đánh trạng thái này thì dựng khoản chi chờ duyệt.
	SinhKhoan bool
}

var CacTrangThaiDoNghe = []MoTaTrangThaiDoNghe{
	{DoNgheChuaMua, "Chưa mua", false},
	{DoNgheDaCo, "Đã có sẵn", false},
	{DoNgheDaMua, "Đã mua", true},
	{DoNgheTamHoan, "Tạm hoãn", false},
}

type MoTaNhomDoNghe struct {
	Ma  string
	Ten string
}

// Thứ tự bám theo bài 1.3 của giáo trình để bảng đọc giống thứ tự đã dạy.
var CacNhomDoNghe = []MoTaNhomDoNghe{
	{"do-chan-doan", "Đo và chẩn đoán"},
	{"thao-lam-sach", "Tháo và làm sạch"},
	{"dan-ep", "Dán và ép"},
	{"khau-de", "Khâu và đế"},
	{"hoan-thien-mau", "Hoàn thiện và màu"},
	{"may-thiet-bi", "Máy và thiết bị"},
	{"bao-ho", "Bảo hộ"},
	{"ban-cho-lam", "Bàn và chỗ làm"},
}

type DoNghe struct {
	Ma        string `yaml:"ma"`
	Ten       string `yaml:"ten"`
	Nghe      string `yaml:"nghe"`
	Nhom      string `yaml:"nhom"`
	Dot       int    `yaml:"dot"`      // 1 mở cửa được · 2 làm ca khó · 3 nhanh đẹp hơn
	BatBuoc   bool   `yaml:"bat_buoc"` // thiếu là không nhận hàng được
	QuyCach   string `yaml:"quy_cach,omitempty"`
	DungDe    string `yaml:"dung_de,omitempty"`
	GiaDuKien int    `yaml:"gia_du_kien"` // ước lượng để xếp thứ tự sắm, không phải cam kết

	TrangThai string `yaml:"trang_thai"`
	NgayMua   string `yaml:"ngay_mua,omitempty"`
	GiaThuc   int    `yaml:"gia_thuc,omitempty"`
	NguonMua  string `yaml:"nguon_mua,omitempty"`
	MaKhoan   string `yaml:"ma_khoan,omitempty"`
	GhiChu    string `yaml:"ghi_chu,omitempty"`
}

func (d DoNghe) TenNghe() string      { return tenTrongBang(d.Nghe, ngheTen) }
func (d DoNghe) TenNhom() string      { return tenTrongBang(d.Nhom, nhomDoNgheTen) }
func (d DoNghe) TenTrangThai() string { return tenTrongBang(d.TrangThai, trangThaiDoNgheTen) }

// DaSam: đã có cái mà dùng, bất kể tốn tiền hay không.
func (d DoNghe) DaSam() bool { return d.TrangThai == DoNgheDaMua || d.TrangThai == DoNgheDaCo }

// ConThieu: còn phải bỏ tiền ra. Tạm hoãn không tính — đã quyết định là chưa cần.
func (d DoNghe) ConThieu() bool { return d.TrangThai == DoNgheChuaMua }

// CanGap: món bắt buộc mà còn thiếu. Bảng tô đỏ dòng này.
func (d DoNghe) CanGap() bool { return d.BatBuoc && d.ConThieu() }

func tenTrongBang(ma string, bang map[string]string) string {
	if t, co := bang[ma]; co {
		return t
	}
	return ma
}

var (
	ngheTen            = map[string]string{}
	nhomDoNgheTen      = map[string]string{}
	trangThaiDoNgheTen = map[string]string{}
)

func init() {
	for _, n := range CacNghe {
		ngheTen[n.Ma] = n.Ten
	}
	for _, n := range CacNhomDoNghe {
		nhomDoNgheTen[n.Ma] = n.Ten
	}
	for _, t := range CacTrangThaiDoNghe {
		trangThaiDoNgheTen[t.Ma] = t.Ten
	}
}

func NgheHopLe(ma string) bool       { _, co := ngheTen[ma]; return co }
func NhomDoNgheHopLe(ma string) bool { _, co := nhomDoNgheTen[ma]; return co }
func TrangThaiDoNgheHopLe(ma string) bool {
	_, co := trangThaiDoNgheTen[ma]
	return co
}

// --- Nạp và lưu ------------------------------------------------------

var (
	doNgheMu sync.RWMutex
	doNgheDs []DoNghe
)

func fileDoNghe() string { return P("data/kho/do-nghe.yaml") }

type doNgheFile struct {
	DoNghe []DoNghe `yaml:"do_nghe"`
}

// NapDoNghe. Thiếu file thì danh sách rỗng, không phải lỗi — trạm chưa khai
// món nào vẫn phải vào được trang để khai.
func NapDoNghe() error {
	ds := []DoNghe{}
	if b, err := os.ReadFile(fileDoNghe()); err == nil {
		var f doNgheFile
		if err := yaml.Unmarshal(b, &f); err != nil {
			return fmt.Errorf("data/kho/do-nghe.yaml hỏng: %w", err)
		}
		ds = f.DoNghe
	} else if !os.IsNotExist(err) {
		return err
	}
	for i := range ds {
		if !TrangThaiDoNgheHopLe(ds[i].TrangThai) {
			ds[i].TrangThai = DoNgheChuaMua
		}
		if !NgheHopLe(ds[i].Nghe) {
			ds[i].Nghe = NgheChung
		}
		if ds[i].Dot < 1 || ds[i].Dot > 3 {
			ds[i].Dot = 1
		}
	}
	doNgheMu.Lock()
	doNgheDs = ds
	doNgheMu.Unlock()
	return nil
}

func luuDoNgheDaKhoa(ds []DoNghe) error {
	b, err := yaml.Marshal(doNgheFile{DoNghe: ds})
	if err != nil {
		return err
	}
	dau := []byte("# Đồ nghề hai nghề: sửa vợt và sửa giày. Sửa ở /qt/do-nghe.\n" +
		"# Đây KHÔNG phải vật tư tiêu hao — vật tư ở vat-tu.yaml. Xem đầu core/donghe.go.\n" +
		"# gia_du_kien là ước lượng để xếp thứ tự sắm; tiền thật nằm ở sổ tiền.\n\n")
	if err := ghiAtomic(fileDoNghe(), append(dau, b...)); err != nil {
		return err
	}
	doNgheDs = ds
	return nil
}

func DanhSachDoNghe() []DoNghe {
	doNgheMu.RLock()
	defer doNgheMu.RUnlock()
	return append([]DoNghe{}, doNgheDs...)
}

func TimDoNghe(ma string) (DoNghe, bool) {
	doNgheMu.RLock()
	defer doNgheMu.RUnlock()
	for _, d := range doNgheDs {
		if d.Ma == ma {
			return d, true
		}
	}
	return DoNghe{}, false
}

// LuuDoNghe thêm mới hoặc sửa phần khai báo của một món: tên, nghề, nhóm, đợt,
// giá dự kiến, ghi chú. KHÔNG đụng tới trạng thái mua và mã khoản — đổi trạng
// thái đi đường DoiTrangThaiDoNghe, nơi có luật sinh khoản chi.
func LuuDoNghe(d DoNghe) (DoNghe, error) {
	d.Ten = strings.Join(strings.Fields(d.Ten), " ")
	if d.Ten == "" {
		return d, fmt.Errorf("phải có tên món")
	}
	if !NgheHopLe(d.Nghe) {
		d.Nghe = NgheChung
	}
	if !NhomDoNgheHopLe(d.Nhom) {
		d.Nhom = "thao-lam-sach"
	}
	if d.Dot < 1 || d.Dot > 3 {
		d.Dot = 1
	}
	if d.GiaDuKien < 0 {
		d.GiaDuKien = 0
	}

	doNgheMu.Lock()
	defer doNgheMu.Unlock()
	ds := append([]DoNghe{}, doNgheDs...)
	for i, cu := range ds {
		if cu.Ma != d.Ma || d.Ma == "" {
			continue
		}
		// Giữ nguyên phần trạng thái: form khai báo không được phép ghi đè.
		d.TrangThai, d.NgayMua, d.GiaThuc, d.NguonMua, d.MaKhoan =
			cu.TrangThai, cu.NgayMua, cu.GiaThuc, cu.NguonMua, cu.MaKhoan
		ds[i] = d
		return d, luuDoNgheDaKhoa(ds)
	}

	if d.Ma == "" {
		d.Ma = maDoNgheMoi(d.Ten, ds)
	}
	if d.TrangThai == "" {
		d.TrangThai = DoNgheChuaMua
	}
	ds = append(ds, d)
	return d, luuDoNgheDaKhoa(ds)
}

// maDoNgheMoi sinh mã từ tên. Mã là thứ khoản chi trong sổ trỏ vào, nên sinh
// một lần rồi giữ đời đời — đổi tên món không được đổi mã.
func maDoNgheMoi(ten string, ds []DoNghe) string {
	goc := maTuTen(ten)
	if goc == "" {
		goc = "mon"
	}
	dung := map[string]bool{}
	for _, d := range ds {
		dung[d.Ma] = true
	}
	if !dung[goc] {
		return goc
	}
	for i := 2; ; i++ {
		thu := fmt.Sprintf("%s-%d", goc, i)
		if !dung[thu] {
			return thu
		}
	}
}

// XoaDoNghe. Món đã sinh khoản chi thì không xoá được: xoá dòng đi thì khoản
// trong sổ mồ côi, nhìn vào "Sắm đồ nghề: ..." mà không tra ngược được là món
// gì. Muốn giấu thì để Tạm hoãn.
func XoaDoNghe(ma string) error {
	doNgheMu.Lock()
	defer doNgheMu.Unlock()
	ds := append([]DoNghe{}, doNgheDs...)
	for i, d := range ds {
		if d.Ma != ma {
			continue
		}
		if d.MaKhoan != "" {
			return fmt.Errorf("món này đã có khoản chi %s trong sổ, không xoá được — để Tạm hoãn nếu không cần nữa", d.MaKhoan)
		}
		return luuDoNgheDaKhoa(append(ds[:i:i], ds[i+1:]...))
	}
	return fmt.Errorf("không có món %s", ma)
}

// --- Đổi trạng thái, nối sổ tiền -------------------------------------

// DoiTrangThaiDoNghe là đường duy nhất đổi trạng thái mua.
//
// Sang "đã mua" thì dựng một khoản chi nhóm dau_tu ở trạng thái CHỜ DUYỆT —
// chưa tính vào báo cáo cho tới khi chủ bấm duyệt ở /qt/tien. Không tự ghi lén,
// cùng luật với phiếu nhập kho.
//
// Ba ca phải chặn, đều đã gặp ở chỗ khác trong hệ:
//
//  1. Bấm hai lần (mạng chậm, F5 lại) — MaKhoan khác rỗng thì SỬA khoản cũ chứ
//     không đẻ khoản mới. Không có khoá này thì sổ có hai khoản 240k cho một
//     cái cân và con số đầu tư sai gấp đôi.
//  2. Khoản đã duyệt rồi mới sửa giá — chỉ sửa trong file này, không sờ vào
//     khoản đã vào báo cáo. Muốn sửa tiền thì sửa ở trang sổ tiền.
//  3. Bỏ trạng thái "đã mua" — KHÔNG xoá khoản trong sổ. Tiền đã tiêu là đã
//     tiêu; trả hàng là một khoản thu mới chứ không phải xoá lịch sử.
func DoiTrangThaiDoNghe(ma, trangThai, ngay string, giaThuc int, nguonMua, nguoi string) (string, error) {
	if !TrangThaiDoNgheHopLe(trangThai) {
		return "", fmt.Errorf("trạng thái không hợp lệ: %q", trangThai)
	}
	d, co := TimDoNghe(ma)
	if !co {
		return "", fmt.Errorf("không có món %s", ma)
	}

	nhac := ""
	if trangThai == DoNgheDaMua {
		if strings.TrimSpace(ngay) == "" {
			ngay = time.Now().Format("2006-01-02")
		}
		if giaThuc <= 0 {
			giaThuc = d.GiaDuKien
		}
		if giaThuc <= 0 {
			return "", fmt.Errorf("phải điền số tiền đã trả cho %q", d.Ten)
		}
		d.NgayMua, d.GiaThuc, d.NguonMua = ngay, giaThuc, strings.TrimSpace(nguonMua)
		maKhoan, tb, err := ghiChiDoNghe(d, nguoi)
		if err != nil {
			return "", err
		}
		d.MaKhoan, nhac = maKhoan, tb
	} else if d.MaKhoan != "" {
		nhac = "Khoản chi " + d.MaKhoan + " vẫn còn trong sổ — bỏ đánh dấu ở đây không xoá tiền đã ghi."
	}
	d.TrangThai = trangThai

	doNgheMu.Lock()
	defer doNgheMu.Unlock()
	ds := append([]DoNghe{}, doNgheDs...)
	for i, cu := range ds {
		if cu.Ma == d.Ma {
			ds[i] = d
			return nhac, luuDoNgheDaKhoa(ds)
		}
	}
	return nhac, fmt.Errorf("không có món %s", ma)
}

// ghiChiDoNghe trả về mã khoản và lời nhắc (nếu có chuyện đáng nói).
func ghiChiDoNghe(d DoNghe, nguoi string) (string, string, error) {
	if d.MaKhoan != "" {
		cu, co := LayKhoan(d.MaKhoan)
		switch {
		case !co:
			// Khoản bị xoá tay ở trang sổ tiền. Ghi lại một khoản mới.
		case !cu.ChoDuyet:
			return cu.Ma, "Khoản chi " + cu.Ma + " đã duyệt rồi nên giữ nguyên " +
				dinhDangTien(cu.SoTien) + " trong sổ. Muốn sửa tiền thì sửa ở trang sổ tiền.", nil
		default:
			cu.Ngay = d.NgayMua
			cu.SoTien = d.GiaThuc
			cu.DienGiai = dienGiaiDoNghe(d)
			return cu.Ma, "", LuuKhoan(cu)
		}
	}
	k := Khoan{
		Ngay:       d.NgayMua,
		Loai:       KhoanChi,
		Nhom:       NhomDauTu,
		SoTien:     d.GiaThuc,
		DienGiai:   dienGiaiDoNghe(d),
		PhuongThuc: TraTienMat,
		MaDoNghe:   d.Ma,
		ChoDuyet:   true,
		Nguoi:      nguoi,
	}
	k.Ma = MaKhoanMoi(k.Ngay)
	if err := LuuKhoan(k); err != nil {
		return "", "", err
	}
	return k.Ma, "Đã dựng khoản chi " + k.Ma + " chờ duyệt ở sổ tiền.", nil
}

func dienGiaiDoNghe(d DoNghe) string {
	s := "Sắm đồ nghề: " + d.Ten
	if n := strings.TrimSpace(d.NguonMua); n != "" {
		s += " — " + n
	}
	return s
}

// --- Bảng và tổng ----------------------------------------------------

type NhomBangDoNghe struct {
	Dot     int
	Nhom    string
	TenNhom string
	Mon     []DoNghe
}

// BangDoNghe dựng bảng cho trang: lọc theo nghề và trạng thái, gom theo đợt rồi
// theo nhóm. nghe hoặc trangThai rỗng là không lọc. Lọc theo nghề "vot" thì các
// món dùng chung vẫn hiện — thiếu cái cân thì cả hai nghề đều không làm được.
func BangDoNghe(nghe, trangThai string) []NhomBangDoNghe {
	thuTuNhom := map[string]int{}
	for i, n := range CacNhomDoNghe {
		thuTuNhom[n.Ma] = i
	}

	gom := map[string]*NhomBangDoNghe{}
	for _, d := range DanhSachDoNghe() {
		if nghe != "" && d.Nghe != nghe && d.Nghe != NgheChung {
			continue
		}
		if trangThai != "" && d.TrangThai != trangThai {
			continue
		}
		khoa := fmt.Sprintf("%d|%s", d.Dot, d.Nhom)
		if gom[khoa] == nil {
			gom[khoa] = &NhomBangDoNghe{Dot: d.Dot, Nhom: d.Nhom, TenNhom: tenTrongBang(d.Nhom, nhomDoNgheTen)}
		}
		gom[khoa].Mon = append(gom[khoa].Mon, d)
	}

	out := make([]NhomBangDoNghe, 0, len(gom))
	for _, n := range gom {
		sort.SliceStable(n.Mon, func(i, j int) bool { return n.Mon[i].Ten < n.Mon[j].Ten })
		out = append(out, *n)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Dot != out[j].Dot {
			return out[i].Dot < out[j].Dot
		}
		return thuTuNhom[out[i].Nhom] < thuTuNhom[out[j].Nhom]
	})
	return out
}

type TongQuanDoNghe struct {
	DaChi        int // tiền đã duyệt trong sổ, gắn với đồ nghề
	ChoDuyet     int // đã bấm mua nhưng chưa duyệt — chưa vào báo cáo
	ConPhaiSam   int // cộng giá dự kiến các món chưa mua
	Dot1ConThieu int
	SoConThieu   int
	SoCanGap     int // món bắt buộc mà chưa có
	SoDaSam      int
	SoMon        int
}

// TongQuanDoNghe cộng hai nguồn khác nhau, cố ý: tiền đã chi lấy từ SỔ TIỀN
// (nguồn sự thật của tiền), tiền còn phải sắm lấy từ giá dự kiến trong file này
// (sổ không biết gì về thứ chưa mua).
func TongQuanDoNgheHienGio() TongQuanDoNghe {
	t := TongQuanDoNghe{}
	for _, d := range DanhSachDoNghe() {
		t.SoMon++
		switch {
		case d.ConThieu():
			t.SoConThieu++
			t.ConPhaiSam += d.GiaDuKien
			if d.Dot == 1 {
				t.Dot1ConThieu += d.GiaDuKien
			}
			if d.BatBuoc {
				t.SoCanGap++
			}
		case d.DaSam():
			t.SoDaSam++
		}
	}
	for _, k := range khoanDoNghe() {
		if k.ChoDuyet {
			t.ChoDuyet += k.SoTien
		} else {
			t.DaChi += k.SoTien
		}
	}
	return t
}

func khoanDoNghe() []Khoan {
	tienMu.RLock()
	defer tienMu.RUnlock()
	out := []Khoan{}
	for _, ds := range tienTheoThang {
		for _, k := range ds {
			if k.MaDoNghe != "" {
				out = append(out, k)
			}
		}
	}
	return out
}
