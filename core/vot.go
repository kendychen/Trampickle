// Danh mục vợt — đọc data/vot.yaml.
//
// File YAML là nguồn duy nhất. Go chỉ đọc, không ghi: sửa danh mục là sửa
// file rồi chạy `python scripts/print_paddles.py --ghi` để sinh lại bản
// markdown cho agent. Hai đường đó phải cùng một nguồn, nếu không thợ đọc
// một đằng agent trả lời một nẻo.
//
// Nạp lại theo ModTime, không nạp một lần rồi thôi: sửa file trên máy là
// F5 thấy ngay, không phải khởi động lại service.
package core

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type MoTaLoi struct {
	Ma           string   `yaml:"ma"`
	Ten          string   `yaml:"ten"`
	CauTao       string   `yaml:"cau_tao"`
	DacDiem      string   `yaml:"dac_diem"`
	HongDacTrung []string `yaml:"hong_dac_trung"`
	XuongNhan    string   `yaml:"xuong_nhan"`
	GhiChu       string   `yaml:"ghi_chu"`
}

type MoTaCheTao struct {
	Ma           string   `yaml:"ma"`
	Ten          string   `yaml:"ten"`
	CachLam      string   `yaml:"cach_lam"`
	TuoiThoThang []*int   `yaml:"tuoi_tho_thang"`
	HongDacTrung []string `yaml:"hong_dac_trung"`
	ViSao        string   `yaml:"vi_sao"`
	CanhBaoXuong string   `yaml:"canh_bao_xuong"`
	Nguon        []string `yaml:"nguon"`
}

type MoTaLoiHong struct {
	Ma          string `yaml:"ma"`
	Ten         string `yaml:"ten"`
	DauHieu     string `yaml:"dau_hieu"`
	PhepThu     string `yaml:"phep_thu"`
	KetLuan     string `yaml:"ket_luan"`
	DichVu      string `yaml:"dich_vu"` // mã dịch vụ trong bang-gia.yaml, rỗng = không nhận
	TyLeUocTinh string `yaml:"ty_le_uoc_tinh"`
	GhiChu      string `yaml:"ghi_chu"`
}

// Can — cân nặng công bố. Hãng có khi cho một số, có khi cho khoảng
// (7.8–8.3 oz). Giữ nguyên dạng hãng nói: quy khoảng về một số thì thợ cân
// ra lệch vài gam lại tưởng vợt ngấm nước hoặc đã bị sửa.
//
// YAML nhận cả hai: `khoi_luong_g: 225` hoặc `khoi_luong_g: [221, 235]`.
type Can struct{ Tu, Den int }

func (c *Can) UnmarshalYAML(n *yaml.Node) error {
	switch n.Kind {
	case yaml.ScalarNode:
		if n.ShortTag() == "!!null" {
			return nil
		}
		var v int
		if err := n.Decode(&v); err != nil {
			return err
		}
		c.Tu, c.Den = v, v
	case yaml.SequenceNode:
		var v []int
		if err := n.Decode(&v); err != nil {
			return err
		}
		if len(v) != 2 {
			return fmt.Errorf("dòng %d: khoi_luong_g phải là một số hoặc [nhẹ, nặng], nhận %d phần tử", n.Line, len(v))
		}
		if v[0] > v[1] {
			return fmt.Errorf("dòng %d: khoi_luong_g [%d, %d] — đầu nhẹ phải nhỏ hơn đầu nặng", n.Line, v[0], v[1])
		}
		c.Tu, c.Den = v[0], v[1]
	default:
		return fmt.Errorf("dòng %d: khoi_luong_g không đọc được", n.Line)
	}
	return nil
}

// BienThe — MỘT cây vợt. Trường nào để trống thì thừa kế của dòng cha.
type BienThe struct {
	Ten string `yaml:"ten"`
	Nam int    `yaml:"nam"`
	// Loi để trống thì lấy của dòng cha. Ghi đè khi trong cùng một hãng có
	// cây tổ ong lẫn cây lõi bọt — Kamito RX1 và Kamito Dominus chẳng hạn.
	// Lõi sai thì thợ gõ nghe tiếng ra kết luận ngược.
	Loi        string   `yaml:"loi"`
	CheTao     string   `yaml:"che_tao"`
	DoDayMm    *float64 `yaml:"do_day_mm"`
	KhoiLuongG *Can     `yaml:"khoi_luong_g"`
	NguonSo    string   `yaml:"nguon_so"`
	GhiChu     string   `yaml:"ghi_chu"`
}

type DongVot struct {
	Ma       string    `yaml:"ma"`
	Hang     string    `yaml:"hang"`
	Nuoc     string    `yaml:"nuoc"`
	Dong     string    `yaml:"dong"`
	Loi      string    `yaml:"loi"`
	CheTao   []string  `yaml:"che_tao"`
	Mat      string    `yaml:"mat"`
	Vien     string    `yaml:"vien"`
	GiaVN    []*int    `yaml:"gia_vn"`
	MucGap   string    `yaml:"muc_gap"`
	CoSo     string    `yaml:"co_so"`
	HayHong  []string  `yaml:"hay_hong"`
	NhanSua  string    `yaml:"nhan_sua"`
	DoTinCay string    `yaml:"do_tin_cay"`
	GhiChu   string    `yaml:"ghi_chu"`
	Nguon    []string  `yaml:"nguon"`
	BienThe  []BienThe `yaml:"bien_the"`
}

type KhoVot struct {
	Version       int               `yaml:"version"`
	CapNhat       string            `yaml:"cap_nhat"`
	SoCaLamCoSo   int               `yaml:"so_ca_lam_co_so"`
	NguongTinDuoc int               `yaml:"nguong_tin_duoc"`
	ThangMucGap   map[string]string `yaml:"thang_muc_gap"`
	Loi           []MoTaLoi         `yaml:"loi"`
	CheTao        []MoTaCheTao      `yaml:"che_tao"`
	LoiHong       []MoTaLoiHong     `yaml:"loi_hong"`
	DongVot       []DongVot         `yaml:"dong_vot"`
}

var (
	votMu    sync.RWMutex
	votKho   *KhoVot
	votMtime int64
)

// NapVot đọc lại file nếu nó đã đổi kể từ lần đọc trước.
func NapVot() (*KhoVot, error) {
	p := CFG.DuongDan.DanhMucVot
	if p == "" {
		p = "data/vot.yaml"
	}
	st, err := os.Stat(P(p))
	if err != nil {
		return nil, err
	}
	m := st.ModTime().UnixNano()

	votMu.RLock()
	if votKho != nil && votMtime == m {
		k := votKho
		votMu.RUnlock()
		return k, nil
	}
	votMu.RUnlock()

	b, err := os.ReadFile(P(p))
	if err != nil {
		return nil, err
	}
	var k KhoVot
	if err := yaml.Unmarshal(b, &k); err != nil {
		return nil, err
	}

	votMu.Lock()
	votKho, votMtime = &k, m
	votMu.Unlock()
	return &k, nil
}

// --- Tra cứu ---------------------------------------------------------

func (k *KhoVot) TenLoi(ma string) string {
	for _, x := range k.Loi {
		if x.Ma == ma {
			return x.Ten
		}
	}
	return ma
}

func (k *KhoVot) TenCheTao(ma string) string {
	for _, x := range k.CheTao {
		if x.Ma == ma {
			return x.Ten
		}
	}
	return ma
}

func (k *KhoVot) TenLoiHong(ma string) string {
	for _, x := range k.LoiHong {
		if x.Ma == ma {
			return x.Ten
		}
	}
	return ma
}

func (k *KhoVot) CanhBaoCheTao(ma string) string {
	for _, x := range k.CheTao {
		if x.Ma == ma {
			return strings.TrimSpace(x.CanhBaoXuong)
		}
	}
	return ""
}

// SoCay đếm tổng số cây vợt, không phải số dòng.
func (k *KhoVot) SoCay() int {
	n := 0
	for _, d := range k.DongVot {
		if len(d.BienThe) == 0 {
			n++ // dòng không có cây nào mang tên riêng vẫn là một hàng trên bảng
			continue
		}
		n += len(d.BienThe)
	}
	return n
}

// DuTin — dưới ngưỡng thì mọi trang phải nói rõ là chưa có ca thật chống lưng.
func (k *KhoVot) DuTin() bool { return k.SoCaLamCoSo >= k.NguongTinDuoc }

// --- Bảng phẳng để hiển thị ------------------------------------------

// HangVot — một dòng trong bảng: một CÂY vợt, đã gộp sẵn thông tin thừa kế
// từ dòng cha để template khỏi phải tra ngược.
type HangVot struct {
	MaDong     string // để mở khối chi tiết của dòng cha
	Hang       string
	Dong       string
	Ten        string // tên cây
	Nam        int
	Loi        string
	CheTao     string
	CanhBao    string // cảnh báo của đời chế tạo, nếu có
	DoDay      string
	KhoiLuong  string
	NguonSo    string
	MucGap     string
	MucGapNhan string
	Nuoc       string
	TuVN       bool
	GhiChu     string
	Tim        string // chuỗi thường hoá để JS lọc
}

var thuTuMucGap = map[string]int{
	"rat_pho_bien": 0, "pho_bien": 1, "thinh_thoang": 2, "hiem": 3, "rat_hiem": 4,
}

var NhanMucGap = map[string]string{
	"rat_pho_bien": "Rất phổ biến", "pho_bien": "Phổ biến",
	"thinh_thoang": "Thỉnh thoảng", "hiem": "Hiếm", "rat_hiem": "Rất hiếm",
}

var NhanNguonSo = map[string]string{
	"do_lab": "đo", "hang_cong_bo": "hãng nói", "chua_co": "chưa tra",
}

var NhanVien = map[string]string{
	"co": "Có viền", "khong": "Không viền", "tuy_model": "Tuỳ cây",
}

var NhanTinCay = map[string]string{
	"cao": "cao", "trung_binh": "trung bình", "thap": "thấp",
}

func so(f *float64) string {
	if f == nil {
		return "—"
	}
	return strconv.FormatFloat(*f, 'f', -1, 64) + "mm"
}

func gam(c *Can) string {
	if c == nil {
		return "—"
	}
	if c.Tu == c.Den {
		return strconv.Itoa(c.Tu) + "g"
	}
	return strconv.Itoa(c.Tu) + "–" + strconv.Itoa(c.Den) + "g"
}

// BangVot dựng bảng phẳng, sắp theo mức hay gặp rồi tới tên hãng.
func (k *KhoVot) BangVot() []HangVot {
	var out []HangVot
	for _, d := range k.DongVot {
		bt := d.BienThe
		if len(bt) == 0 {
			// Dòng không có cây nào mang tên riêng — vợt OEM không nhãn.
			// Vẫn phải lên bảng: đây lại đúng là nhóm thợ tra nhiều nhất.
			bt = []BienThe{{Ten: d.Hang, NguonSo: "chua_co"}}
		}
		for _, b := range bt {
			ct := b.CheTao
			if ct == "" && len(d.CheTao) == 1 {
				ct = d.CheTao[0]
			}
			lo := b.Loi
			if lo == "" {
				lo = d.Loi
			}
			h := HangVot{
				MaDong:     d.Ma,
				Hang:       d.Hang,
				Dong:       d.Dong,
				Ten:        b.Ten,
				Nam:        b.Nam,
				Loi:        k.TenLoi(lo),
				CheTao:     k.TenCheTao(ct),
				CanhBao:    k.CanhBaoCheTao(ct),
				DoDay:      so(b.DoDayMm),
				KhoiLuong:  gam(b.KhoiLuongG),
				NguonSo:    NhanNguonSo[b.NguonSo],
				MucGap:     d.MucGap,
				MucGapNhan: NhanMucGap[d.MucGap],
				Nuoc:       d.Nuoc,
				TuVN:       strings.HasPrefix(d.Nuoc, "Việt Nam"),
				GhiChu:     strings.TrimSpace(b.GhiChu),
			}
			h.Tim = strings.ToLower(strings.Join([]string{
				d.Hang, d.Dong, b.Ten, d.Nuoc, h.Loi, h.CheTao,
			}, " "))
			out = append(out, h)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := thuTuMucGap[out[i].MucGap], thuTuMucGap[out[j].MucGap]
		if a != b {
			return a < b
		}
		if out[i].Hang != out[j].Hang {
			return out[i].Hang < out[j].Hang
		}
		return out[i].Ten < out[j].Ten
	})
	return out
}

// ChiTietDong — khối chi tiết mở ra khi bấm vào một cây.
type ChiTietDong struct {
	Ma        string   `json:"ma"`
	Hang      string   `json:"hang"`
	Dong      string   `json:"dong"`
	Nuoc      string   `json:"nuoc"`
	Loi       string   `json:"loi"`
	LoiCauTao string   `json:"loi_cau_tao"`
	Mat       string   `json:"mat"`
	Vien      string   `json:"vien"`
	Gia       string   `json:"gia"`
	MucGap    string   `json:"muc_gap"`
	CoSo      string   `json:"co_so"`
	HayHong   []string `json:"hay_hong"`
	NhanSua   string   `json:"nhan_sua"`
	TinCay    string   `json:"tin_cay"`
	GhiChu    string   `json:"ghi_chu"`
	CanhBao   []string `json:"canh_bao"`
	Nguon     []string `json:"nguon"`
}

func (k *KhoVot) ChiTiet() map[string]ChiTietDong {
	out := map[string]ChiTietDong{}
	for _, d := range k.DongVot {
		c := ChiTietDong{
			Ma: d.Ma, Hang: d.Hang, Dong: d.Dong, Nuoc: d.Nuoc,
			Loi:     k.TenLoi(d.Loi),
			Mat:     d.Mat,
			Vien:    NhanVien[d.Vien],
			Gia:     khoangGia(d.GiaVN),
			MucGap:  NhanMucGap[d.MucGap],
			CoSo:    strings.TrimSpace(d.CoSo),
			NhanSua: strings.TrimSpace(d.NhanSua),
			TinCay:  NhanTinCay[d.DoTinCay],
			GhiChu:  strings.TrimSpace(d.GhiChu),
			Nguon:   d.Nguon,
		}
		for _, x := range k.Loi {
			if x.Ma == d.Loi {
				c.LoiCauTao = strings.TrimSpace(x.CauTao)
			}
		}
		for _, m := range d.HayHong {
			c.HayHong = append(c.HayHong, k.TenLoiHong(m))
		}
		seen := map[string]bool{}
		for _, m := range d.CheTao {
			if cb := k.CanhBaoCheTao(m); cb != "" && !seen[cb] {
				seen[cb] = true
				c.CanhBao = append(c.CanhBao, cb)
			}
		}
		out[d.Ma] = c
	}
	return out
}

func khoangGia(g []*int) string {
	if len(g) < 2 || g[0] == nil || g[1] == nil {
		return "—"
	}
	return dinhDangTien(*g[0]) + " – " + dinhDangTien(*g[1])
}
