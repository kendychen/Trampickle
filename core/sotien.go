// Sổ thu chi của trạm, và câu trả lời cho "lời hay lỗ, bao giờ về vốn".
//
// RANH GIỚI VỚI src/quote.py — đọc kỹ trước khi thêm phép tính vào đây.
// Python độc quyền GIÁ BÁN: con số cam kết với khách, sinh từ bang-gia.yaml.
// File này chỉ được cộng trừ chia trên SỐ ĐÃ GHI VÀO SỔ — tiền đã thu, tiền
// đã chi, những con số có người gõ vào vì nó đã xảy ra thật. Nó KHÔNG đọc
// bảng giá để suy ra một con số tiền mới. Chỗ duy nhất bảng giá xuất hiện là
// dòng đối chiếu định phí, và ở đó hai con số chỉ nằm CẠNH nhau cho người
// nhìn, không cái nào đẻ ra cái nào.
//
// Cơ sở tính: TIỀN MẶT. Thu là tiền khách đã trả, không phải tiền đã báo
// giá. Chi là tiền đã bỏ ra, kể cả mua máy — không khấu hao. Chọn vậy vì
// "bao giờ về vốn" là câu hỏi về dòng tiền: bao giờ tiền vào bù hết tiền đã
// bỏ ra. Công nợ khách và giá trị tồn kho hiện riêng, không trộn vào lãi.
//
// Lưu mỗi tháng một file JSON. Vài chục khoản một tháng, ghi lại cả file khi
// sửa một dòng vẫn rẻ, và mở ra đọc được bằng mắt khi có sự cố.
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
)

// --- Loại và nhóm ----------------------------------------------------

const (
	KhoanThu = "thu"
	KhoanChi = "chi"
)

const (
	NhomSuaChua = "sua_chua"
	NhomThuKhac = "thu_khac"
	NhomDauTu   = "dau_tu"
	NhomDinhPhi = "dinh_phi"
	NhomVatTu   = "vat_tu"
	NhomVanHanh = "van_hanh"
	NhomChiKhac = "chi_khac"
	// NhomHoanKhach: tiền trả ngược lại cho khách. Là khoản CHI chứ không phải
	// xoá khoản thu cũ — sao kê ngân hàng có hai giao dịch thì sổ cũng phải có
	// hai dòng, ngày đối chiếu mới khớp. Gắn mã đơn thì tự trừ vào số đã thu.
	NhomHoanKhach = "hoan_khach"
)

type MoTaNhom struct {
	Ma   string
	Ten  string
	Loai string
	// DauTu: tiền bỏ ra một lần để có cái làm ăn. Đây là MẪU SỐ của bài toán
	// về vốn nên phải tách khỏi chi phí vận hành — trộn vào thì tháng nào mua
	// máy cũng thành tháng lỗ, và không bao giờ tính được đã bù được bao nhiêu.
	DauTu bool
	MoTa  string
}

var CacNhomTien = []MoTaNhom{
	{NhomSuaChua, "Tiền sửa vợt", KhoanThu, false, "Khách trả cho đơn sửa. Ghi từ trang đơn hoặc khớp sao kê."},
	{NhomThuKhac, "Thu khác", KhoanThu, false, "Bán vật tư lẻ, hoàn tiền nhà cung cấp, thu nhập phụ."},
	{NhomDauTu, "Đầu tư", KhoanChi, true, "Máy móc, đồ nghề, bàn ghế, cọc thuê chỗ, sửa sang chỗ làm. Cả tiền học nghề — học đan dây, khoá kỹ thuật: cũng là tiền bỏ ra một lần để có cái làm ăn."},
	{NhomDinhPhi, "Định phí", KhoanChi, false, "Thuê nhà, điện nước, internet, VPS, tên miền — tháng nào cũng trả."},
	{NhomVatTu, "Nhập vật tư", KhoanChi, false, "Sinh tự động từ phiếu nhập kho, không gõ lại."},
	{NhomVanHanh, "Vận hành", KhoanChi, false, "Ship, bao bì, thuê ngoài, quảng cáo, phí sàn."},
	{NhomChiKhac, "Chi khác", KhoanChi, false, "Chi vặt không xếp được vào đâu — cà phê tiếp khách, xăng xe, đồ lặt vặt."},
	{NhomHoanKhach, "Hoàn tiền khách", KhoanChi, false, "Trả lại tiền cho khách khi đơn huỷ hoặc làm lại. Ghi từ trang đơn thì tự trừ vào số đã thu của đơn ấy."},
}

func TimNhom(ma string) (MoTaNhom, bool) {
	for _, n := range CacNhomTien {
		if n.Ma == ma {
			return n, true
		}
	}
	return MoTaNhom{}, false
}

func TenNhomTien(ma string) string {
	if n, co := TimNhom(ma); co {
		return n.Ten
	}
	return ma
}

func NhomTheoLoai(loai string) []MoTaNhom {
	out := []MoTaNhom{}
	for _, n := range CacNhomTien {
		if n.Loai == loai {
			out = append(out, n)
		}
	}
	return out
}

const (
	TraTienMat     = "tien_mat"
	TraChuyenKhoan = "chuyen_khoan"
)

// --- Khoản -----------------------------------------------------------

type Khoan struct {
	Ma         string `json:"ma"`
	Ngay       string `json:"ngay"`
	Loai       string `json:"loai"`
	Nhom       string `json:"nhom"`
	SoTien     int    `json:"so_tien"`
	DienGiai   string `json:"dien_giai"`
	PhuongThuc string `json:"phuong_thuc"`
	MaDon      string `json:"ma_don"`     // khoản thu gắn với đơn sửa
	MaPhieu    string `json:"ma_phieu"`   // khoản chi sinh từ phiếu nhập kho
	MaDinhKy   string `json:"ma_dinh_ky"` // sinh từ khai báo định kỳ
	// ChoDuyet: khoản định kỳ hệ thống dựng sẵn đầu tháng, CHƯA tính vào báo
	// cáo. Tháng nào quên trả tiền nhà mà sổ vẫn ghi đã chi thì sổ nói dối,
	// nên phải có người bấm xác nhận đã trả rồi mới vào sổ.
	ChoDuyet bool   `json:"cho_duyet"`
	Nguoi    string `json:"nguoi"`
	Tao      string `json:"tao"`
}

func (k Khoan) TenNhom() string { return TenNhomTien(k.Nhom) }

func (k Khoan) LaThu() bool { return k.Loai == KhoanThu }

func (k Khoan) Thang() string {
	if len(k.Ngay) >= 7 {
		return k.Ngay[:7]
	}
	return ""
}

func (k Khoan) TenPhuongThuc() string {
	switch k.PhuongThuc {
	case TraTienMat:
		return "tiền mặt"
	case TraChuyenKhoan:
		return "chuyển khoản"
	}
	return ""
}

// --- Kho trong bộ nhớ ------------------------------------------------

var (
	tienMu        sync.RWMutex
	tienTheoThang = map[string][]Khoan{}
	tienTheoDon   = map[string]int{}
)

func thuMucSoTien() string { return P("data/so-tien") }

func fileThang(thang string) string {
	return filepath.Join(thuMucSoTien(), thang+".json")
}

func thangHopLe(t string) bool {
	_, err := time.Parse("2006-01", t)
	return err == nil
}

func NapSoTien() error {
	dir := thuMucSoTien()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	moi := map[string][]Khoan{}
	for _, e := range ents {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		thang := strings.TrimSuffix(e.Name(), ".json")
		if !thangHopLe(thang) {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var ds []Khoan
		if json.Unmarshal(b, &ds) != nil {
			continue
		}
		moi[thang] = ds
	}
	tienMu.Lock()
	tienTheoThang = moi
	dungLaiChiSoDon()
	tienMu.Unlock()
	return nil
}

// dungLaiChiSoDon — gọi khi đang giữ tienMu. Dựng lại từ đầu thay vì cộng
// trừ theo từng thay đổi: sửa một khoản phải nhớ trừ giá trị cũ, quên một
// lần là chỉ số lệch mãi mãi. Vài trăm khoản dựng lại hết mất vài micro giây.
func dungLaiChiSoDon() {
	m := map[string]int{}
	for _, ds := range tienTheoThang {
		for _, k := range ds {
			if k.MaDon == "" || k.ChoDuyet {
				continue
			}
			// Hoàn tiền trừ ra: không trừ thì đơn đã trả lại tiền vẫn hiện
			// "đã thu đủ, không nợ", và doanh thu tháng đếm cả phần đã nhả.
			switch {
			case k.Loai == KhoanThu:
				m[k.MaDon] += k.SoTien
			case k.Nhom == NhomHoanKhach:
				m[k.MaDon] -= k.SoTien
			}
		}
	}
	tienTheoDon = m
}

func ghiThang(thang string) error {
	ds := tienTheoThang[thang]
	sort.Slice(ds, func(i, j int) bool {
		if ds[i].Ngay != ds[j].Ngay {
			return ds[i].Ngay < ds[j].Ngay
		}
		return ds[i].Ma < ds[j].Ma
	})
	b, err := json.MarshalIndent(ds, "", "  ")
	if err != nil {
		return err
	}
	return ghiAtomic(fileThang(thang), b)
}

// MaKhoanMoi — TC-2609-001. Duy nhất trong tháng là đủ vì mã đã mang tháng.
func MaKhoanMoi(ngay string) string {
	thang := "2006-01"
	if len(ngay) >= 7 {
		thang = ngay[:7]
	}
	t, err := time.Parse("2006-01", thang)
	if err != nil {
		t = time.Now()
	}
	tien := fmt.Sprintf("TC-%s-", t.Format("0601"))
	tienMu.RLock()
	max := 0
	for _, k := range tienTheoThang[t.Format("2006-01")] {
		if !strings.HasPrefix(k.Ma, tien) {
			continue
		}
		if n, err := strconv.Atoi(strings.TrimPrefix(k.Ma, tien)); err == nil && n > max {
			max = n
		}
	}
	tienMu.RUnlock()
	return fmt.Sprintf("%s%03d", tien, max+1)
}

func LuuKhoan(k Khoan) error {
	if !thangHopLe(k.Thang()) {
		return fmt.Errorf("ngày không hợp lệ: %q", k.Ngay)
	}
	if k.Loai != KhoanThu && k.Loai != KhoanChi {
		return fmt.Errorf("loại không hợp lệ: %q", k.Loai)
	}
	n, co := TimNhom(k.Nhom)
	if !co || n.Loai != k.Loai {
		return fmt.Errorf("nhóm %q không thuộc loại %q", k.Nhom, k.Loai)
	}
	if k.SoTien <= 0 {
		return fmt.Errorf("số tiền phải lớn hơn 0")
	}
	if k.Ma == "" {
		k.Ma = MaKhoanMoi(k.Ngay)
	}
	if k.Tao == "" {
		k.Tao = time.Now().Format("2006-01-02 15:04:05")
	}

	tienMu.Lock()
	defer tienMu.Unlock()
	// Khoản có thể đổi ngày sang tháng khác: gỡ khỏi mọi tháng trước khi thêm.
	for thang, ds := range tienTheoThang {
		for i, cu := range ds {
			if cu.Ma != k.Ma {
				continue
			}
			tienTheoThang[thang] = append(ds[:i:i], ds[i+1:]...)
			if thang != k.Thang() {
				if err := ghiThang(thang); err != nil {
					return err
				}
			}
			break
		}
	}
	tienTheoThang[k.Thang()] = append(tienTheoThang[k.Thang()], k)
	if err := ghiThang(k.Thang()); err != nil {
		return err
	}
	dungLaiChiSoDon()
	return nil
}

func LayKhoan(ma string) (Khoan, bool) {
	tienMu.RLock()
	defer tienMu.RUnlock()
	for _, ds := range tienTheoThang {
		for _, k := range ds {
			if k.Ma == ma {
				return k, true
			}
		}
	}
	return Khoan{}, false
}

func XoaKhoan(ma string) error {
	tienMu.Lock()
	defer tienMu.Unlock()
	for thang, ds := range tienTheoThang {
		for i, k := range ds {
			if k.Ma != ma {
				continue
			}
			tienTheoThang[thang] = append(ds[:i:i], ds[i+1:]...)
			if err := ghiThang(thang); err != nil {
				return err
			}
			dungLaiChiSoDon()
			return nil
		}
	}
	return fmt.Errorf("không có khoản %s", ma)
}

// DuyetKhoan bật một khoản chờ duyệt thành khoản thật.
func DuyetKhoan(ma string) error {
	k, co := LayKhoan(ma)
	if !co {
		return fmt.Errorf("không có khoản %s", ma)
	}
	if !k.ChoDuyet {
		return nil
	}
	k.ChoDuyet = false
	return LuuKhoan(k)
}

func KhoanTrongThang(thang string) []Khoan {
	tienMu.RLock()
	defer tienMu.RUnlock()
	return append([]Khoan{}, tienTheoThang[thang]...)
}

// CacThangCoSo — mọi tháng có khoản, cũ đến mới.
func CacThangCoSo() []string {
	tienMu.RLock()
	out := make([]string, 0, len(tienTheoThang))
	for t, ds := range tienTheoThang {
		if len(ds) > 0 {
			out = append(out, t)
		}
	}
	tienMu.RUnlock()
	sort.Strings(out)
	return out
}

func KhoanChoDuyet() []Khoan {
	tienMu.RLock()
	out := []Khoan{}
	for _, ds := range tienTheoThang {
		for _, k := range ds {
			if k.ChoDuyet {
				out = append(out, k)
			}
		}
	}
	tienMu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Ngay < out[j].Ngay })
	return out
}

func SoKhoanChoDuyet() int { return len(KhoanChoDuyet()) }

// ThuCuaDon — tổng tiền khách đã trả cho đơn này. Đây là nguồn duy nhất cho
// Don.DaThu(); đơn không giữ riêng một con số nào.
func ThuCuaDon(maDon string) int {
	if maDon == "" {
		return 0
	}
	tienMu.RLock()
	defer tienMu.RUnlock()
	return tienTheoDon[maDon]
}

func CacLanThuCuaDon(maDon string) []Khoan {
	tienMu.RLock()
	out := []Khoan{}
	for _, ds := range tienTheoThang {
		for _, k := range ds {
			if k.MaDon == maDon && k.Loai == KhoanThu {
				out = append(out, k)
			}
		}
	}
	tienMu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Ngay < out[j].Ngay })
	return out
}

// CacLanHoanCuaDon — các lần trả tiền lại cho khách của đơn này.
func CacLanHoanCuaDon(maDon string) []Khoan {
	tienMu.RLock()
	out := []Khoan{}
	for _, ds := range tienTheoThang {
		for _, k := range ds {
			if k.MaDon == maDon && k.Loai == KhoanChi && k.Nhom == NhomHoanKhach {
				out = append(out, k)
			}
		}
	}
	tienMu.RUnlock()
	sort.Slice(out, func(i, j int) bool { return out[i].Ngay < out[j].Ngay })
	return out
}

// DaHoanCuaDon — tổng đã trả lại khách.
func DaHoanCuaDon(maDon string) int {
	t := 0
	for _, k := range CacLanHoanCuaDon(maDon) {
		t += k.SoTien
	}
	return t
}

// --- Báo cáo ---------------------------------------------------------

type ThangTien struct {
	Thang        string
	Thu          int
	ChiVanHanh   int // mọi khoản chi trừ đầu tư
	DauTu        int
	Lai          int // Thu - ChiVanHanh. Đây là con số bù dần vào vốn.
	DongTienRong int // Lai - DauTu. Tiền thực sự vào hay ra khỏi túi tháng đó.
	LuyKeLai     int
	LuyKeDauTu   int
	TheoNhom     map[string]int
	SoKhoan      int
}

func (t ThangTien) Lo() bool { return t.Lai < 0 }

// BaoCaoThang trả về mới nhất trước. Luỹ kế đã tính theo chiều thời gian.
func BaoCaoThang() []ThangTien {
	thangs := CacThangCoSo()
	out := make([]ThangTien, 0, len(thangs))
	luyLai, luyDauTu := 0, 0
	for _, th := range thangs {
		t := ThangTien{Thang: th, TheoNhom: map[string]int{}}
		for _, k := range KhoanTrongThang(th) {
			if k.ChoDuyet {
				continue
			}
			t.SoKhoan++
			t.TheoNhom[k.Nhom] += k.SoTien
			if k.Loai == KhoanThu {
				t.Thu += k.SoTien
				continue
			}
			if n, co := TimNhom(k.Nhom); co && n.DauTu {
				t.DauTu += k.SoTien
			} else {
				t.ChiVanHanh += k.SoTien
			}
		}
		t.Lai = t.Thu - t.ChiVanHanh
		t.DongTienRong = t.Lai - t.DauTu
		luyLai += t.Lai
		luyDauTu += t.DauTu
		t.LuyKeLai, t.LuyKeDauTu = luyLai, luyDauTu
		out = append(out, t)
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

type KetQuaVeVon struct {
	CoSo        bool // đã có số liệu chưa
	TongDauTu   int
	LuyKeLai    int
	DaVeVon     bool
	ThangVeVon  string
	ConThieu    int
	LaiTBThang  int
	SoThangTinh int
	DangLo      bool
	ConMayThang float64
	NgayDuKien  string
}

// TinhVeVon — ba nhánh, và nhánh thứ ba là nhánh hay bị quên nhất.
//
//  1. Luỹ kế lãi đã bù hết đầu tư → nói rõ về vốn từ tháng nào.
//  2. Chưa bù hết mà đang có lãi → ước tính còn mấy tháng.
//  3. Đang lỗ → KHÔNG chia. Chia số dương cho số âm ra một con số âm trông
//     như một cái hẹn, mà thực tế là "không bao giờ, nếu cứ thế này". Năm
//     đầu của một trạm mới nhánh này chạy là chuyện bình thường.
func TinhVeVon() KetQuaVeVon {
	bc := BaoCaoThang()
	if len(bc) == 0 {
		return KetQuaVeVon{}
	}
	kq := KetQuaVeVon{CoSo: true}
	// bc đang là mới nhất trước; duyệt ngược để đi theo chiều thời gian.
	for i := len(bc) - 1; i >= 0; i-- {
		t := bc[i]
		if !kq.DaVeVon && t.LuyKeDauTu > 0 && t.LuyKeLai >= t.LuyKeDauTu {
			kq.DaVeVon = true
			kq.ThangVeVon = t.Thang
		}
	}
	moi := bc[0]
	kq.TongDauTu, kq.LuyKeLai = moi.LuyKeDauTu, moi.LuyKeLai
	kq.ConThieu = kq.TongDauTu - kq.LuyKeLai
	if kq.ConThieu < 0 {
		kq.ConThieu = 0
	}

	// Tốc độ lấy 3 tháng gần nhất, bỏ tháng chưa có khoản nào.
	tong, dem := 0, 0
	for _, t := range bc {
		if t.SoKhoan == 0 {
			continue
		}
		tong += t.Lai
		dem++
		if dem == 3 {
			break
		}
	}
	kq.SoThangTinh = dem
	if dem > 0 {
		kq.LaiTBThang = tong / dem
	}
	if kq.DaVeVon || kq.TongDauTu == 0 {
		return kq
	}
	if kq.LaiTBThang <= 0 {
		kq.DangLo = true
		return kq
	}
	kq.ConMayThang = math.Ceil(float64(kq.ConThieu)/float64(kq.LaiTBThang)*10) / 10
	if kq.ConMayThang > 0 && kq.ConMayThang < 600 {
		kq.NgayDuKien = time.Now().AddDate(0, int(math.Ceil(kq.ConMayThang)), 0).Format("2006-01")
	}
	return kq
}

// DinhPhiThucTe — trung bình chi định phí mấy tháng gần nhất. Chỉ để đặt
// CẠNH dinh_phi_thang trong bang-gia.yaml cho người nhìn thấy lệch. Không
// sửa file bảng giá: file đó là đầu vào định giá của quote.py, sửa nó là đổi
// giá bán sau lưng.
func DinhPhiThucTe(soThang int) (int, int) {
	if soThang <= 0 {
		soThang = 3
	}
	tong, dem := 0, 0
	for _, t := range BaoCaoThang() {
		if t.SoKhoan == 0 {
			continue
		}
		tong += t.TheoNhom[NhomDinhPhi]
		dem++
		if dem == soThang {
			break
		}
	}
	if dem == 0 {
		return 0, 0
	}
	return tong / dem, dem
}

type TongQuanTien struct {
	ThangNay    ThangTien
	VeVon       KetQuaVeVon
	CongNoKhach int
	GiaTriTon   int
	ChoDuyet    int
	DinhPhiTB   int
	DinhPhiBang int
}

func LayTongQuanTien() TongQuanTien {
	tq := TongQuanTien{
		CongNoKhach: LayThongKeDon().ConNoTong,
		GiaTriTon:   GiaTriTonKho(),
		ChoDuyet:    SoKhoanChoDuyet(),
		DinhPhiBang: GIA.DinhPhiThang,
		VeVon:       TinhVeVon(),
	}
	tq.DinhPhiTB, _ = DinhPhiThucTe(3)
	nay := time.Now().Format("2006-01")
	for _, t := range BaoCaoThang() {
		if t.Thang == nay {
			tq.ThangNay = t
			break
		}
	}
	if tq.ThangNay.Thang == "" {
		tq.ThangNay = ThangTien{Thang: nay, TheoNhom: map[string]int{}}
	}
	return tq
}
