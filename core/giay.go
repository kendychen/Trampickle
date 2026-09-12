// Danh mục giày — đọc data/giay.yaml.
//
// Cùng lối với vot.go: file YAML là nguồn duy nhất, Go chỉ đọc. Sửa danh mục
// là sửa file rồi chạy `python scripts/print_shoes.py --ghi` để sinh lại
// vanhanh/danh-muc-giay.md cho agent. Hai đường phải cùng một nguồn, nếu
// không thợ tra một đằng agent trả lời một nẻo.
//
// Khác vot.go ở một chỗ đáng nói: với vợt thì cái quyết định là lõi, với giày
// là CÁCH GẮN ĐẾ. Nên GanDe được đẩy lên thành cột chính của bảng, và mọi
// hàng đều mang sẵn XuLy (nhận / từ chối) đã tính cả thừa kế — thợ nhìn một
// cái là biết có nhận đôi giày trên bàn hay không.
package core

import (
	"os"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// MoTaGanDe — cách đế dính vào mũ giày. Bảng quan trọng nhất của cả danh mục:
// hai đôi giày nhìn giống hệt nhau, gắn đế khác nhau thì một đôi nhận một đôi
// trả về.
type MoTaGanDe struct {
	Ma           string `yaml:"ma"`
	Ten          string `yaml:"ten"`
	TenEn        string `yaml:"ten_en"`
	NhanRa       string `yaml:"nhan_ra"`
	BocDuoc      string `yaml:"boc_duoc"`
	XuongNhan    string `yaml:"xuong_nhan"` // nhan | nhan_dk | xet | gui_di | tu_choi
	CanhBaoXuong string `yaml:"canh_bao_xuong"`
	GhiChu       string `yaml:"ghi_chu"`
}

type MoTaDeNgoai struct {
	Ma      string `yaml:"ma"`
	Ten     string `yaml:"ten"`
	DacDiem string `yaml:"dac_diem"`
	MucMon  string `yaml:"muc_mon"`
	CanhBao string `yaml:"canh_bao"`
	GhiChu  string `yaml:"ghi_chu"`
}

// MoTaDeGiua — tầng KHÔNG thay được. Bảng này không để sửa, mà để biết được
// phép làm gì: bóc đế cần nhiệt, mỗi loại bọt chịu nhiệt một khác.
type MoTaDeGiua struct {
	Ma         string `yaml:"ma"`
	Ten        string `yaml:"ten"`
	CoODau     string `yaml:"co_o_dau"`
	TuoiTho    string `yaml:"tuoi_tho"`
	ChiuNhiet  string `yaml:"chiu_nhiet"`
	LuatChoTho string `yaml:"luat_cho_tho"`
	CanhBao    string `yaml:"canh_bao"`
}

type MoTaHongGiay struct {
	Ma      string `yaml:"ma"`
	Ten     string `yaml:"ten"`
	DauHieu string `yaml:"dau_hieu"`
	PhepThu string `yaml:"phep_thu"`
	KetLuan string `yaml:"ket_luan"`
	DichVu  string `yaml:"dich_vu"` // mã dịch vụ trong bang-gia.yaml, rỗng = không nhận
	GhiChu  string `yaml:"ghi_chu"`
}

// BienTheGiay — MỘT model giày cụ thể. Trường nào để trống thì thừa kế của
// dòng cha. Ghi đè là chuyện thường xuyên chứ không phải ngoại lệ: Nike làm
// cả giày chạy dán nguội lẫn giày đua đế một khối dưới cùng một logo.
type BienTheGiay struct {
	Ten     string `yaml:"ten"`
	GanDe   string `yaml:"gan_de"`
	DeNgoai string `yaml:"de_ngoai"`
	DeGiua  string `yaml:"de_giua"`
	XuLy    string `yaml:"xu_ly"`
	NhanSua string `yaml:"nhan_sua"`
	GhiChu  string `yaml:"ghi_chu"`
}

type DongGiay struct {
	Ma       string        `yaml:"ma"`
	Hang     string        `yaml:"hang"`
	Nuoc     string        `yaml:"nuoc"`
	Dong     string        `yaml:"dong"`
	Mon      string        `yaml:"mon"`
	GanDe    []string      `yaml:"gan_de"`
	DeNgoai  []string      `yaml:"de_ngoai"`
	DeGiua   []string      `yaml:"de_giua"`
	MucGap   string        `yaml:"muc_gap"`
	CoSo     string        `yaml:"co_so"`
	HayHong  []string      `yaml:"hay_hong"`
	XuLy     string        `yaml:"xu_ly"`
	NhanSua  string        `yaml:"nhan_sua"`
	DoTinCay string        `yaml:"do_tin_cay"`
	GhiChu   string        `yaml:"ghi_chu"`
	Nguon    []string      `yaml:"nguon"`
	BienThe  []BienTheGiay `yaml:"bien_the"`
}

// NhomNguonGiay — nguồn gom theo mục nó chống lưng cho. Trang không hiện phần
// này; nó ở đây để bản markdown sinh ra cho agent có đủ dẫn nguồn, và để mục
// nào không có nguồn thì nhìn ra ngay.
type NhomNguonGiay struct {
	Nhom  string `yaml:"nhom"`
	ViSao string `yaml:"vi_sao"`
	Muc   []struct {
		Ten string `yaml:"ten"`
		Url string `yaml:"url"`
	} `yaml:"muc"`
}

type KhoGiay struct {
	Version       int               `yaml:"version"`
	CapNhat       string            `yaml:"cap_nhat"`
	SoCaLamCoSo   int               `yaml:"so_ca_lam_co_so"`
	NguongTinDuoc int               `yaml:"nguong_tin_duoc"`
	ThangMucGap   map[string]string `yaml:"thang_muc_gap"`
	GanDe         []MoTaGanDe       `yaml:"gan_de"`
	DeNgoai       []MoTaDeNgoai     `yaml:"de_ngoai"`
	DeGiua        []MoTaDeGiua      `yaml:"de_giua"`
	LoiHong       []MoTaHongGiay    `yaml:"loi_hong"`
	DongGiay      []DongGiay        `yaml:"dong_giay"`
	Nguon         []NhomNguonGiay   `yaml:"nguon"`
}

var (
	giayMu    sync.RWMutex
	giayKho   *KhoGiay
	giayMtime int64
)

// NapGiay đọc lại file nếu nó đã đổi kể từ lần đọc trước.
func NapGiay() (*KhoGiay, error) {
	p := CFG.DuongDan.DanhMucGiay
	if p == "" {
		p = "data/giay.yaml"
	}
	st, err := os.Stat(P(p))
	if err != nil {
		return nil, err
	}
	m := st.ModTime().UnixNano()

	giayMu.RLock()
	if giayKho != nil && giayMtime == m {
		k := giayKho
		giayMu.RUnlock()
		return k, nil
	}
	giayMu.RUnlock()

	b, err := os.ReadFile(P(p))
	if err != nil {
		return nil, err
	}
	var k KhoGiay
	if err := yaml.Unmarshal(b, &k); err != nil {
		return nil, err
	}

	giayMu.Lock()
	giayKho, giayMtime = &k, m
	giayMu.Unlock()
	return &k, nil
}

// --- Tra cứu ---------------------------------------------------------

func (k *KhoGiay) TenGanDe(ma string) string {
	for _, x := range k.GanDe {
		if x.Ma == ma {
			return x.Ten
		}
	}
	return ma
}

func (k *KhoGiay) TenDeNgoai(ma string) string {
	for _, x := range k.DeNgoai {
		if x.Ma == ma {
			return x.Ten
		}
	}
	return ma
}

func (k *KhoGiay) TenDeGiua(ma string) string {
	for _, x := range k.DeGiua {
		if x.Ma == ma {
			return x.Ten
		}
	}
	return ma
}

func (k *KhoGiay) TenHongGiay(ma string) string {
	for _, x := range k.LoiHong {
		if x.Ma == ma {
			return x.Ten
		}
	}
	return ma
}

func (k *KhoGiay) CanhBaoGanDe(ma string) string {
	for _, x := range k.GanDe {
		if x.Ma == ma {
			return strings.TrimSpace(x.CanhBaoXuong)
		}
	}
	return ""
}

// CanhBaoDeGiua — cảnh báo của tầng bọt: PU thuỷ phân, PEBA và túi khí kỵ
// nhiệt. Đây là thứ làm hỏng đôi giày TRONG LÚC sửa, nên phải nổi lên panel
// chứ không nằm im trong bảng vật liệu.
func (k *KhoGiay) CanhBaoDeGiua(ma string) string {
	for _, x := range k.DeGiua {
		if x.Ma == ma {
			return strings.TrimSpace(x.CanhBao)
		}
	}
	return ""
}

// SoDoi đếm số model giày, không phải số hãng.
func (k *KhoGiay) SoDoi() int {
	n := 0
	for _, d := range k.DongGiay {
		if len(d.BienThe) == 0 {
			n++
			continue
		}
		n += len(d.BienThe)
	}
	return n
}

// SoNhan đếm số model xưởng nhận được — con số thợ quan tâm nhất khi mở trang.
func (k *KhoGiay) SoNhan() int {
	n := 0
	for _, h := range k.BangGiay() {
		if h.XuLy == "nhan" || h.XuLy == "nhan_dk" {
			n++
		}
	}
	return n
}

// DuTin — dưới ngưỡng thì mọi trang phải nói rõ là chưa có ca thật chống lưng.
func (k *KhoGiay) DuTin() bool { return k.SoCaLamCoSo >= k.NguongTinDuoc }

// --- Bảng phẳng để hiển thị ------------------------------------------

// HangGiay — một dòng trong bảng: một MODEL giày, đã gộp sẵn thừa kế từ dòng
// cha để template khỏi phải tra ngược.
type HangGiay struct {
	MaDong     string // để mở khối chi tiết của dòng cha
	Hang       string
	Dong       string
	Ten        string // tên model
	Mon        string
	GanDe      string // mã, để JS lọc
	GanDeNhan  string
	DeNgoai    string
	DeGiua     string
	DeGiuaMa   string
	XuLy       string // mã: nhan | nhan_dk | xet | gui_di | tu_choi
	XuLyNhan   string
	CanhBao    string // cảnh báo gộp của gắn đế + đế giữa
	MucGap     string
	MucGapNhan string
	NhanSua    string // câu của model nếu model tự nói, không thì của dòng cha
	Nuoc       string
	TuVN       bool
	GhiChu     string
	Tim        string // chuỗi thường hoá để JS lọc
}

// thuTuXuLy xếp cái nhận được lên trên. Danh mục này dùng để quyết định nhận
// hay trả, nên thứ tự phải theo quyết định chứ không theo bảng chữ cái.
var thuTuXuLy = map[string]int{
	"nhan": 0, "nhan_dk": 1, "xet": 2, "gui_di": 3, "tu_choi": 4,
}

var NhanXuLy = map[string]string{
	"nhan":    "Nhận",
	"nhan_dk": "Nhận có điều kiện",
	"xet":     "Xét từng đôi",
	"gui_di":  "Gửi tiệm ngoài",
	"tu_choi": "Từ chối",
}

// BangGiay dựng bảng phẳng, sắp theo nhận/từ chối rồi tới mức hay gặp.
func (k *KhoGiay) BangGiay() []HangGiay {
	var out []HangGiay
	for _, d := range k.DongGiay {
		bt := d.BienThe
		if len(bt) == 0 {
			// Hãng không tách model — cả hãng cho một câu trả lời. Vẫn phải
			// lên bảng: Yonex, Converse, Biti's đều nằm ở nhóm này.
			bt = []BienTheGiay{{Ten: d.Hang}}
		}
		for _, b := range bt {
			gd := b.GanDe
			if gd == "" && len(d.GanDe) == 1 {
				gd = d.GanDe[0]
			}
			dn := b.DeNgoai
			if dn == "" && len(d.DeNgoai) == 1 {
				dn = d.DeNgoai[0]
			}
			dg := b.DeGiua
			if dg == "" && len(d.DeGiua) == 1 {
				dg = d.DeGiua[0]
			}
			xl := b.XuLy
			if xl == "" {
				xl = d.XuLy
			}
			h := HangGiay{
				MaDong:     d.Ma,
				Hang:       d.Hang,
				Dong:       d.Dong,
				Ten:        b.Ten,
				Mon:        d.Mon,
				GanDe:      gd,
				GanDeNhan:  nhanHoac(k.TenGanDe(gd), gd, "Tuỳ model"),
				DeNgoai:    nhanHoac(k.TenDeNgoai(dn), dn, "—"),
				DeGiua:     nhanHoac(k.TenDeGiua(dg), dg, "Tuỳ model"),
				DeGiuaMa:   dg,
				XuLy:       xl,
				XuLyNhan:   NhanXuLy[xl],
				MucGap:     d.MucGap,
				MucGapNhan: NhanMucGap[d.MucGap],
				Nuoc:       d.Nuoc,
				TuVN:       strings.HasPrefix(d.Nuoc, "Việt Nam"),
				GhiChu:     strings.TrimSpace(nhanChuoi(b.GhiChu, d.GhiChu)),
				// Model tự nói thì nghe model. Dòng Nike ghi "nhận dòng chạy
				// phổ thông, từ chối ZoomX" — dán nguyên câu ấy lên đúng hàng
				// ZoomX là mâu thuẫn với chính con chip "Từ chối" bên cạnh.
				NhanSua: strings.TrimSpace(nhanChuoi(b.NhanSua, d.NhanSua)),
			}
			// Gộp hai nguồn cảnh báo. Bóc nhầm đế lưu hoá thì rách vải; hơ
			// nhầm bọt PEBA thì méo phom. Cả hai đều hỏng vĩnh viễn, nên cả
			// hai phải hiện ngay trên hàng chứ không đợi mở panel.
			var cb []string
			if x := k.CanhBaoGanDe(gd); x != "" {
				cb = append(cb, x)
			}
			if x := k.CanhBaoDeGiua(dg); x != "" {
				cb = append(cb, x)
			}
			h.CanhBao = strings.Join(cb, " · ")
			h.Tim = strings.ToLower(strings.Join([]string{
				d.Hang, d.Dong, b.Ten, d.Nuoc, d.Mon,
				h.GanDeNhan, h.DeGiua, h.DeNgoai, h.XuLyNhan,
			}, " "))
			out = append(out, h)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := thuTuXuLy[out[i].XuLy], thuTuXuLy[out[j].XuLy]
		if a != b {
			return a < b
		}
		c, d := thuTuMucGap[out[i].MucGap], thuTuMucGap[out[j].MucGap]
		if c != d {
			return c < d
		}
		if out[i].Hang != out[j].Hang {
			return out[i].Hang < out[j].Hang
		}
		return out[i].Ten < out[j].Ten
	})
	return out
}

// nhanHoac: tra ra tên thì dùng tên; mã rỗng (dòng cha có nhiều lựa chọn,
// biến thể không chốt) thì dùng chữ thay thế.
func nhanHoac(ten, ma, thay string) string {
	if ma == "" {
		return thay
	}
	return ten
}

func nhanChuoi(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

// ChiTietGiay — khối chi tiết mở ra khi bấm vào một model.
type ChiTietGiay struct {
	Ma      string   `json:"ma"`
	Hang    string   `json:"hang"`
	Dong    string   `json:"dong"`
	Nuoc    string   `json:"nuoc"`
	Mon     string   `json:"mon"`
	GanDe   []string `json:"gan_de"`
	NhanRa  []string `json:"nhan_ra"`
	DeNgoai []string `json:"de_ngoai"`
	DeGiua  []string `json:"de_giua"`
	MucGap  string   `json:"muc_gap"`
	CoSo    string   `json:"co_so"`
	HayHong []string `json:"hay_hong"`
	XuLy    string   `json:"xu_ly"`
	NhanSua string   `json:"nhan_sua"`
	TinCay  string   `json:"tin_cay"`
	GhiChu  string   `json:"ghi_chu"`
	CanhBao []string `json:"canh_bao"`
	Nguon   []string `json:"nguon"`
}

func (k *KhoGiay) ChiTiet() map[string]ChiTietGiay {
	out := map[string]ChiTietGiay{}
	for _, d := range k.DongGiay {
		c := ChiTietGiay{
			Ma: d.Ma, Hang: d.Hang, Dong: d.Dong, Nuoc: d.Nuoc, Mon: d.Mon,
			MucGap:  NhanMucGap[d.MucGap],
			CoSo:    strings.TrimSpace(d.CoSo),
			XuLy:    NhanXuLy[d.XuLy],
			NhanSua: strings.TrimSpace(d.NhanSua),
			TinCay:  NhanTinCay[d.DoTinCay],
			GhiChu:  strings.TrimSpace(d.GhiChu),
			Nguon:   d.Nguon,
		}
		seen := map[string]bool{}
		for _, m := range d.GanDe {
			c.GanDe = append(c.GanDe, k.TenGanDe(m))
			// Cách nhận biết tại bàn — thứ duy nhất trong panel này thợ dùng
			// khi đang cầm đôi giày, nên đi kèm luôn tên kiểu gắn đế.
			for _, x := range k.GanDe {
				if x.Ma == m && strings.TrimSpace(x.NhanRa) != "" {
					c.NhanRa = append(c.NhanRa, x.Ten+": "+strings.TrimSpace(x.NhanRa))
				}
			}
			if cb := k.CanhBaoGanDe(m); cb != "" && !seen[cb] {
				seen[cb] = true
				c.CanhBao = append(c.CanhBao, cb)
			}
		}
		for _, m := range d.DeNgoai {
			c.DeNgoai = append(c.DeNgoai, k.TenDeNgoai(m))
		}
		for _, m := range d.DeGiua {
			c.DeGiua = append(c.DeGiua, k.TenDeGiua(m))
			if cb := k.CanhBaoDeGiua(m); cb != "" && !seen[cb] {
				seen[cb] = true
				c.CanhBao = append(c.CanhBao, cb)
			}
		}
		for _, m := range d.HayHong {
			c.HayHong = append(c.HayHong, k.TenHongGiay(m))
		}
		out[d.Ma] = c
	}
	return out
}
