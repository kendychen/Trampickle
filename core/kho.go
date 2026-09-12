// Kho vật tư: viền, grip, keo, miếng vá, lead tape, đế giày gửi gia công.
//
// TỒN KHO KHÔNG ĐƯỢC LƯU Ở ĐÂU CẢ. Nó tính bằng cách phát lại toàn bộ phiếu,
// mỗi lần cần. Một con số tồn lưu sẵn cộng một sổ phiếu là hai nguồn sự thật;
// ngày chúng lệch nhau sẽ không ai biết bên nào đúng, và cách duy nhất để
// dựng lại vẫn là đọc sổ phiếu — nên thà đọc thẳng từ đầu. Ở mức vài nghìn
// phiếu một năm, phát lại hết mất vài mili giây.
//
// Về tiền: file này nhân số lượng với đơn giá GHI TRÊN HOÁ ĐƠN NHẬP và cộng
// lại. Đó là cộng sổ trên số đã có thật, không phải định giá — xem ranh giới
// ở đầu donhang.go. Nó không đọc bang-gia.yaml, không suy ra giá bán.
package core

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// --- Kiểu dữ liệu ----------------------------------------------------

type VatTu struct {
	Ma          string  `yaml:"ma"`
	Ten         string  `yaml:"ten"`
	DonVi       string  `yaml:"don_vi"` // mét, cuộn, tuýp, cái, hộp
	TonToiThieu float64 `yaml:"ton_toi_thieu"`
	NhaCungCap  string  `yaml:"nha_cung_cap"`
	GhiChu      string  `yaml:"ghi_chu"`
	Ngung       bool    `yaml:"ngung"` // không dùng nữa, ẩn khỏi form nhưng giữ lịch sử
}

type DongDinhMuc struct {
	MaVatTu string  `yaml:"ma_vat_tu"`
	SoLuong float64 `yaml:"so_luong"`
}

const (
	PhieuNhap   = "nhap"
	PhieuXuat   = "xuat"
	PhieuKiemKe = "kiem_ke"
	PhieuHaoHut = "hao_hut"
)

type MoTaPhieu struct {
	Loai string
	Ten  string
	Dau  string // tiền tố mã phiếu
}

var CacLoaiPhieu = []MoTaPhieu{
	{PhieuNhap, "Nhập kho", "NK"},
	{PhieuXuat, "Xuất dùng", "XK"},
	{PhieuKiemKe, "Kiểm kê", "KK"},
	{PhieuHaoHut, "Hao hụt / hỏng", "HH"},
}

func LoaiPhieuHopLe(loai string) bool {
	for _, l := range CacLoaiPhieu {
		if l.Loai == loai {
			return true
		}
	}
	return false
}

func TenLoaiPhieu(loai string) string {
	for _, l := range CacLoaiPhieu {
		if l.Loai == loai {
			return l.Ten
		}
	}
	return loai
}

type DongPhieu struct {
	MaVatTu string  `json:"ma_vat_tu"`
	SoLuong float64 `json:"so_luong"`
	// DonGia chỉ có ý nghĩa ở phiếu nhập: giá thật trên hoá đơn của lần nhập
	// đó. Phiếu xuất không mang giá — giá vốn của lần xuất là chuyện kế toán
	// kho, mà sổ tiền ở đây tính theo tiền mặt: tiền ra lúc mua, không phải
	// lúc dùng.
	DonGia int    `json:"don_gia"`
	GhiChu string `json:"ghi_chu"`
}

func (d DongPhieu) ThanhTien() int { return int(math.Round(d.SoLuong * float64(d.DonGia))) }

type Phieu struct {
	Ma         string      `json:"ma"`
	Loai       string      `json:"loai"`
	Ngay       string      `json:"ngay"`
	Nguoi      string      `json:"nguoi"`
	MaDon      string      `json:"ma_don"`       // phiếu xuất theo đơn sửa
	NhaCungCap string      `json:"nha_cung_cap"` // phiếu nhập
	GhiChu     string      `json:"ghi_chu"`
	Dong       []DongPhieu `json:"dong"`
	TongTien   int         `json:"tong_tien"`
	Tao        string      `json:"tao"`
}

func (p Phieu) TenLoai() string { return TenLoaiPhieu(p.Loai) }

func CongTienPhieu(dong []DongPhieu) int {
	t := 0
	for _, d := range dong {
		t += d.ThanhTien()
	}
	return t
}

// --- Kho trong bộ nhớ ------------------------------------------------

var (
	khoMu      sync.RWMutex
	khoVatTu   []VatTu
	khoDinhMuc = map[string][]DongDinhMuc{}
	khoPhieu   = map[string]*Phieu{}
)

func fileVatTu() string   { return P("data/kho/vat-tu.yaml") }
func fileDinhMuc() string { return P("data/kho/dinh-muc.yaml") }
func thuMucPhieu() string { return P("data/kho/phieu") }

type khoDanhMucFile struct {
	VatTu []VatTu `yaml:"vat_tu"`
}

type khoDinhMucFile struct {
	DinhMuc map[string][]DongDinhMuc `yaml:"dinh_muc"`
}

// NapKho đọc cả ba nguồn. Thiếu file thì kho rỗng, không phải lỗi: trạm
// chưa khai vật tư nào vẫn phải vào được trang quản lý để khai.
func NapKho() error {
	vt := []VatTu{}
	if b, err := os.ReadFile(fileVatTu()); err == nil {
		var f khoDanhMucFile
		if err := yaml.Unmarshal(b, &f); err != nil {
			return fmt.Errorf("data/kho/vat-tu.yaml hỏng: %w", err)
		}
		vt = f.VatTu
	} else if !os.IsNotExist(err) {
		return err
	}

	dm := map[string][]DongDinhMuc{}
	if b, err := os.ReadFile(fileDinhMuc()); err == nil {
		var f khoDinhMucFile
		if err := yaml.Unmarshal(b, &f); err != nil {
			return fmt.Errorf("data/kho/dinh-muc.yaml hỏng: %w", err)
		}
		if f.DinhMuc != nil {
			dm = f.DinhMuc
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	ph := map[string]*Phieu{}
	dir := thuMucPhieu()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var p Phieu
		if json.Unmarshal(b, &p) != nil || p.Ma == "" {
			continue
		}
		ph[p.Ma] = &p
	}

	khoMu.Lock()
	khoVatTu, khoDinhMuc, khoPhieu = vt, dm, ph
	khoMu.Unlock()
	return nil
}

func DanhSachVatTu(kesCaNgung bool) []VatTu {
	khoMu.RLock()
	defer khoMu.RUnlock()
	out := make([]VatTu, 0, len(khoVatTu))
	for _, v := range khoVatTu {
		if v.Ngung && !kesCaNgung {
			continue
		}
		out = append(out, v)
	}
	return out
}

func TimVatTu(ma string) (VatTu, bool) {
	khoMu.RLock()
	defer khoMu.RUnlock()
	for _, v := range khoVatTu {
		if v.Ma == ma {
			return v, true
		}
	}
	return VatTu{}, false
}

func TenVatTu(ma string) string {
	if v, co := TimVatTu(ma); co {
		return v.Ten
	}
	return ma
}

// LuuDanhMucVatTu ghi đè cả danh mục. Danh mục vài chục dòng, sửa bằng một
// bảng trên web rồi lưu một lượt — không đáng để làm từng dòng một.
func LuuDanhMucVatTu(ds []VatTu) error {
	b, err := yaml.Marshal(khoDanhMucFile{VatTu: ds})
	if err != nil {
		return err
	}
	dau := []byte("# Danh mục vật tư. Sửa ở /qt/kho/vat-tu.\n" +
		"# KHÔNG có số tồn ở đây: tồn tính bằng cách phát lại phiếu trong\n" +
		"# data/kho/phieu/. Xem đầu core/kho.go.\n\n")
	if err := ghiAtomic(fileVatTu(), append(dau, b...)); err != nil {
		return err
	}
	khoMu.Lock()
	khoVatTu = ds
	khoMu.Unlock()
	return nil
}

// maVatTuMoi sinh mã cho dòng vật tư mới: vt-1, vt-2… Không đặt theo tên vì
// mã là thứ phiếu kho trỏ vào — đổi tên "Dây 1.2mm" thành "Dây dù 1.2" mà mã
// đổi theo thì mọi phiếu cũ trỏ vào một vật tư không còn tồn tại.
func maVatTuMoi(dung map[string]bool) string {
	khoMu.RLock()
	for _, v := range khoVatTu {
		if dung[v.Ma] {
			continue
		}
		dung[v.Ma] = true
	}
	khoMu.RUnlock()
	for i := 1; ; i++ {
		m := "vt-" + strconv.Itoa(i)
		if !dung[m] {
			return m
		}
	}
}

func DinhMucCua(maDichVu string) []DongDinhMuc {
	khoMu.RLock()
	defer khoMu.RUnlock()
	return append([]DongDinhMuc{}, khoDinhMuc[maDichVu]...)
}

func LuuDinhMuc(dm map[string][]DongDinhMuc) error {
	b, err := yaml.Marshal(khoDinhMucFile{DinhMuc: dm})
	if err != nil {
		return err
	}
	dau := []byte("# Mỗi ca dịch vụ ăn hết bao nhiêu vật tư. Sửa ở /qt/kho/vat-tu.\n" +
		"# Dùng để dựng SẴN phiếu xuất khi đóng đơn — thợ vẫn phải sửa cho khớp\n" +
		"# thực tế rồi bấm xác nhận. Không có chuyện tự trừ theo định mức.\n\n")
	if err := ghiAtomic(fileDinhMuc(), append(dau, b...)); err != nil {
		return err
	}
	khoMu.Lock()
	khoDinhMuc = dm
	khoMu.Unlock()
	return nil
}

// --- Phiếu -----------------------------------------------------------

func MaPhieuMoi(loai string) string {
	dau := "XX"
	for _, l := range CacLoaiPhieu {
		if l.Loai == loai {
			dau = l.Dau
		}
	}
	tien := fmt.Sprintf("%s-%s-", dau, time.Now().Format("0601"))
	khoMu.RLock()
	max := 0
	for ma := range khoPhieu {
		if !strings.HasPrefix(ma, tien) {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(ma, tien)); err == nil && n > max {
			max = n
		}
	}
	khoMu.RUnlock()
	return fmt.Sprintf("%s%03d", tien, max+1)
}

func LuuPhieu(p *Phieu) error {
	// Phiếu không mã thì khoá map là chuỗi rỗng: phiếu thứ hai đè lên phiếu
	// thứ nhất và tồn kho im lặng sai. Tự cấp mã ở đây để không có đường nào
	// tạo ra chuyện đó, kể cả khi sau này có chỗ khác gọi LuuPhieu.
	if p.Ma == "" {
		p.Ma = MaPhieuMoi(p.Loai)
	}
	if p.Tao == "" {
		p.Tao = time.Now().Format("2006-01-02 15:04:05")
	}
	p.TongTien = CongTienPhieu(p.Dong)
	b, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}
	if err := ghiAtomic(filepath.Join(thuMucPhieu(), p.Ma+".json"), b); err != nil {
		return err
	}
	khoMu.Lock()
	khoPhieu[p.Ma] = p
	khoMu.Unlock()
	return nil
}

func LayPhieu(ma string) (*Phieu, bool) {
	khoMu.RLock()
	defer khoMu.RUnlock()
	p, co := khoPhieu[ma]
	if !co {
		return nil, false
	}
	ban := *p
	ban.Dong = append([]DongPhieu{}, p.Dong...)
	return &ban, true
}

func XoaPhieu(ma string) error {
	if _, co := LayPhieu(ma); !co {
		return fmt.Errorf("không có phiếu %s", ma)
	}
	if err := os.Remove(filepath.Join(thuMucPhieu(), ma+".json")); err != nil && !os.IsNotExist(err) {
		return err
	}
	khoMu.Lock()
	delete(khoPhieu, ma)
	khoMu.Unlock()
	return nil
}

// phieuTheoThuTu — thứ tự phát lại: theo ngày, cùng ngày thì theo mã. Mã có
// tiền tố loại nên cùng ngày sẽ xếp KK trước NK trước XK; điều đó không đổi
// kết quả cuối vì kiểm kê đặt lại tồn tại thời điểm nó được ghi, và người ta
// đếm kho xong mới nhập tiếp trong cùng ngày.
func phieuTheoThuTu() []*Phieu {
	khoMu.RLock()
	ds := make([]*Phieu, 0, len(khoPhieu))
	for _, p := range khoPhieu {
		ds = append(ds, p)
	}
	khoMu.RUnlock()
	sort.Slice(ds, func(i, j int) bool {
		if ds[i].Ngay != ds[j].Ngay {
			return ds[i].Ngay < ds[j].Ngay
		}
		return ds[i].Ma < ds[j].Ma
	})
	return ds
}

// TonKho phát lại toàn bộ phiếu. Kiểm kê ĐẶT LẠI tồn về số đếm được chứ
// không cộng trừ — vật tư nào không có tên trong phiếu kiểm kê thì giữ
// nguyên tồn cũ, vì người ta đếm một góc kho chứ không phải đếm cả kho.
func TonKho() map[string]float64 {
	ton := map[string]float64{}
	for _, p := range phieuTheoThuTu() {
		for _, d := range p.Dong {
			if d.MaVatTu == "" {
				continue
			}
			switch p.Loai {
			case PhieuNhap:
				ton[d.MaVatTu] += d.SoLuong
			case PhieuXuat, PhieuHaoHut:
				ton[d.MaVatTu] -= d.SoLuong
			case PhieuKiemKe:
				ton[d.MaVatTu] = d.SoLuong
			}
		}
	}
	return ton
}

// TieuThuMoiNgay — trung bình mỗi ngày đã dùng hết bao nhiêu, tính trên
// soNgay ngày gần nhất. Xuất dùng và hao hụt đều tính, vì cả hai đều làm
// vật tư biến khỏi kho và đều phải mua bù.
func TieuThuMoiNgay(soNgay int) map[string]float64 {
	if soNgay <= 0 {
		soNgay = 90
	}
	moc := time.Now().AddDate(0, 0, -soNgay).Format("2006-01-02")
	tong := map[string]float64{}
	for _, p := range phieuTheoThuTu() {
		if p.Loai != PhieuXuat && p.Loai != PhieuHaoHut {
			continue
		}
		if p.Ngay < moc {
			continue
		}
		for _, d := range p.Dong {
			tong[d.MaVatTu] += d.SoLuong
		}
	}
	for ma, v := range tong {
		tong[ma] = v / float64(soNgay)
	}
	return tong
}

// GiaNhapGanNhat — đơn giá của lần nhập gần nhất. Dùng để ước giá trị tồn.
// Cố tình KHÔNG dùng bình quân gia quyền: nó cần theo dõi từng lô còn lại
// bao nhiêu, mà giá trị tồn ở đây chỉ để nhìn cho biết, không vào sổ lãi.
func GiaNhapGanNhat(maVatTu string) int {
	gia := 0
	for _, p := range phieuTheoThuTu() {
		if p.Loai != PhieuNhap {
			continue
		}
		for _, d := range p.Dong {
			if d.MaVatTu == maVatTu && d.DonGia > 0 {
				gia = d.DonGia
			}
		}
	}
	return gia
}

// --- Bảng tồn cho trang quản lý --------------------------------------

type DongTonKho struct {
	VatTu
	Ton         float64
	TieuThuNgay float64
	ConDuNgay   int // -1 = chưa đủ dữ liệu để đoán
	DuoiNguong  bool
	Am          bool
	GiaNhap     int
	GiaTri      int
	DeXuatNhap  float64
}

// BangTonKho dựng bảng cho /qt/kho. Dòng nào cần chú ý (âm, dưới ngưỡng) lên
// đầu — trang này mở ra để biết phải đi mua gì, không phải để tra cứu.
func BangTonKho() []DongTonKho {
	ton := TonKho()
	tieuThu := TieuThuMoiNgay(90)
	out := []DongTonKho{}
	for _, v := range DanhSachVatTu(false) {
		d := DongTonKho{VatTu: v, Ton: ton[v.Ma], TieuThuNgay: tieuThu[v.Ma], ConDuNgay: -1}
		d.Am = d.Ton < 0
		d.DuoiNguong = v.TonToiThieu > 0 && d.Ton <= v.TonToiThieu
		if d.TieuThuNgay > 0 {
			d.ConDuNgay = int(d.Ton / d.TieuThuNgay)
			if d.ConDuNgay < 0 {
				d.ConDuNgay = 0
			}
		}
		d.GiaNhap = GiaNhapGanNhat(v.Ma)
		if d.Ton > 0 {
			d.GiaTri = int(math.Round(d.Ton * float64(d.GiaNhap)))
		}
		d.DeXuatNhap = deXuatNhap(d)
		out = append(out, d)
	}
	sort.Slice(out, func(i, j int) bool {
		ai, aj := out[i].Am || out[i].DuoiNguong, out[j].Am || out[j].DuoiNguong
		if ai != aj {
			return ai
		}
		return out[i].Ten < out[j].Ten
	})
	return out
}

// deXuatNhap — mua đủ dùng 30 ngày, và ít nhất là đủ chạm lại ngưỡng tối
// thiểu. Chỉ hiện khi đã chạm ngưỡng; chưa có phiếu xuất nào thì không đoán
// bừa, trả 0 và để người tự quyết.
func deXuatNhap(d DongTonKho) float64 {
	if !d.DuoiNguong && !d.Am {
		return 0
	}
	can := d.TonToiThieu - d.Ton
	if d.TieuThuNgay > 0 {
		if c := d.TieuThuNgay*30 - d.Ton; c > can {
			can = c
		}
	}
	if can <= 0 {
		return 0
	}
	return math.Ceil(can*10) / 10
}

func GiaTriTonKho() int {
	t := 0
	for _, d := range BangTonKho() {
		t += d.GiaTri
	}
	return t
}

func SoVatTuCanMua() int {
	n := 0
	for _, d := range BangTonKho() {
		if d.DuoiNguong || d.Am {
			n++
		}
	}
	return n
}

// --- Nối với đơn sửa -------------------------------------------------

// PhieuXuatCuaDon — đơn đã xuất vật tư chưa. Trả phiếu đầu tiên tìm được;
// một đơn chỉ nên có một phiếu xuất, và hQtXuatKhoDon chặn cái thứ hai.
func PhieuXuatCuaDon(maDon string) (*Phieu, bool) {
	if maDon == "" {
		return nil, false
	}
	khoMu.RLock()
	defer khoMu.RUnlock()
	for _, p := range khoPhieu {
		if p.Loai == PhieuXuat && p.MaDon == maDon {
			ban := *p
			ban.Dong = append([]DongPhieu{}, p.Dong...)
			return &ban, true
		}
	}
	return nil, false
}

// GiaVonDon — ước tính tiền vật tư đã dùng cho một đơn, theo GIÁ NHẬP GẦN
// NHẤT của từng loại.
//
// Chỗ này cố tình KHÔNG phải kế toán kho. Phiếu xuất không mang đơn giá (xem
// chú thích trên DongPhieu.DonGia) vì sổ tiền đếm theo tiền mặt: tiền ra lúc
// mua, không phải lúc dùng. Luật ấy giữ nguyên. Con số ở đây chỉ để so sánh
// giữa các dịch vụ và các kỳ — nhìn xem việc nào ăn vật tư nhiều — nên lấy
// giá nhập gần nhất là đủ, và trang thống kê phải nói rõ đây là ước tính.
//
// Cộng MỌI phiếu xuất mang mã đơn này, không chỉ phiếu đầu tiên: đơn làm dở
// rồi phải bù thêm vật tư thì thợ lập phiếu thứ hai.
func GiaVonDon(maDon string) int {
	if maDon == "" {
		return 0
	}
	// Gom số lượng trước rồi mới tra giá: GiaNhapGanNhat quét lại toàn bộ
	// phiếu mỗi lần gọi, một đơn mười dòng cùng loại vật tư thì quét mười lần.
	so := map[string]float64{}
	for _, p := range phieuTheoThuTu() {
		if p.Loai != PhieuXuat || p.MaDon != maDon {
			continue
		}
		for _, d := range p.Dong {
			so[d.MaVatTu] += d.SoLuong
		}
	}
	tong := 0.0
	for ma, sl := range so {
		tong += sl * float64(GiaNhapGanNhat(ma))
	}
	return int(math.Round(tong))
}

// DeXuatXuatChoDon dựng sẵn các dòng theo định mức của những dịch vụ đã chốt
// trong đơn. Dòng tiền gõ tay không có mã dịch vụ nên không có định mức —
// thợ tự thêm. Cộng dồn khi hai dịch vụ cùng ăn một loại vật tư.
func DeXuatXuatChoDon(d *Don) []DongPhieu {
	gop := map[string]float64{}
	thuTu := []string{}
	for _, dt := range d.DongTien {
		if dt.MaDichVu == "" {
			continue
		}
		for _, dm := range DinhMucCua(dt.MaDichVu) {
			if dm.MaVatTu == "" || dm.SoLuong <= 0 {
				continue
			}
			if _, co := gop[dm.MaVatTu]; !co {
				thuTu = append(thuTu, dm.MaVatTu)
			}
			gop[dm.MaVatTu] += dm.SoLuong
		}
	}
	out := make([]DongPhieu, 0, len(thuTu))
	for _, ma := range thuTu {
		out = append(out, DongPhieu{MaVatTu: ma, SoLuong: gop[ma]})
	}
	return out
}

// PhieuGanDay cho trang /qt/kho — mới nhất lên đầu.
func PhieuGanDay(gioiHan int) []*Phieu {
	ds := phieuTheoThuTu()
	for i, j := 0, len(ds)-1; i < j; i, j = i+1, j-1 {
		ds[i], ds[j] = ds[j], ds[i]
	}
	if gioiHan > 0 && len(ds) > gioiHan {
		ds = ds[:gioiHan]
	}
	return ds
}
