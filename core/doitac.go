// Đối tác gia công: các tiệm ngoài trạm gửi việc sang rồi ăn phần chênh.
//
// Đi theo đúng lối data/lien-he.yaml: ghi ra data/doi-tac.yaml, mà data/
// không bao giờ bị rsync đè khi deploy — danh sách Kendy gõ trên máy chủ sống
// qua mọi lần đẩy binary. Chưa có file không phải lỗi, chỉ là chưa có tiệm
// nào.
//
// Vì sao là danh sách chứ không phải một ô gõ tay ở từng đơn: gõ tay thì
// "Tiệm đế ABC" và "tiem de abc" thành hai tiệm khác nhau, và câu hỏi thật —
// tháng này đã trả tiệm nào bao nhiêu, tiệm nào hay trễ — không bao giờ trả
// lời được.
package core

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type DoiTac struct {
	Ma     string `yaml:"ma" json:"ma"`
	Ten    string `yaml:"ten" json:"ten"`
	Nghe   string `yaml:"nghe" json:"nghe"`
	LienHe string `yaml:"lien_he" json:"lien_he"`
	DiaChi string `yaml:"dia_chi" json:"dia_chi"`
	GhiChu string `yaml:"ghi_chu" json:"ghi_chu"`
	// Ngung: ngừng hợp tác. Ẩn khỏi ô chọn của đơn mới, nhưng đơn cũ vẫn hiện
	// đúng tên tiệm — xoá hẳn thì lịch sử mấy chục đơn trỏ vào khoảng không.
	Ngung bool `yaml:"ngung" json:"ngung"`
}

type khoDoiTacFile struct {
	DoiTac []DoiTac `yaml:"doi_tac"`
}

var (
	doiTacMu sync.RWMutex
	doiTacDs []DoiTac
)

func fileDoiTac() string { return P("data/doi-tac.yaml") }

// NapDoiTac đọc danh sách. Chưa có file không phải lỗi.
func NapDoiTac() error {
	b, err := os.ReadFile(fileDoiTac())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var f khoDoiTacFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("%s hỏng: %w", fileDoiTac(), err)
	}
	doiTacMu.Lock()
	doiTacDs = f.DoiTac
	doiTacMu.Unlock()
	return nil
}

func ghiDoiTac() error {
	b, err := yaml.Marshal(khoDoiTacFile{DoiTac: doiTacDs})
	if err != nil {
		return err
	}
	dau := []byte("# Các tiệm gia công trạm gửi việc sang. Sửa ở /qt/doi-tac.\n\n")
	return ghiAtomic(fileDoiTac(), append(dau, b...))
}

// DanhSachDoiTac — cả tiệm đã ngừng, dùng cho trang quản lý.
func DanhSachDoiTac() []DoiTac {
	doiTacMu.RLock()
	defer doiTacMu.RUnlock()
	out := append([]DoiTac{}, doiTacDs...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].Ngung != out[j].Ngung {
			return !out[i].Ngung
		}
		return strings.ToLower(out[i].Ten) < strings.ToLower(out[j].Ten)
	})
	return out
}

// DoiTacDangDung — chỉ tiệm còn hợp tác, dùng cho ô chọn ở đơn.
func DoiTacDangDung() []DoiTac {
	out := []DoiTac{}
	for _, d := range DanhSachDoiTac() {
		if !d.Ngung {
			out = append(out, d)
		}
	}
	return out
}

func TimDoiTac(ma string) (DoiTac, bool) {
	doiTacMu.RLock()
	defer doiTacMu.RUnlock()
	for _, d := range doiTacDs {
		if d.Ma == ma {
			return d, true
		}
	}
	return DoiTac{}, false
}

// TenDoiTac trả về mã khi không tìm thấy: đơn cũ trỏ tới tiệm đã bị xoá vẫn
// phải hiện ra một cái gì đó đọc được, không phải ô trống.
func TenDoiTac(ma string) string {
	if ma == "" {
		return ""
	}
	if d, co := TimDoiTac(ma); co {
		return d.Ten
	}
	return ma
}

// maTuTen sinh mã từ tên tiệm: "Tiệm đế ABC" -> "tiem-de-abc".
func maTuTen(ten string) string {
	s := boDau(strings.ToLower(strings.TrimSpace(ten)))
	var b strings.Builder
	truoc := false
	for _, c := range s {
		switch {
		case (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9'):
			b.WriteRune(c)
			truoc = false
		default:
			if !truoc && b.Len() > 0 {
				b.WriteRune('-')
				truoc = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// LuuDoiTac thêm mới hoặc sửa. Mã rỗng là thêm mới.
func LuuDoiTac(d DoiTac) (DoiTac, error) {
	d.Ten = strings.Join(strings.Fields(d.Ten), " ")
	if d.Ten == "" {
		return d, errors.New("phải có tên tiệm")
	}
	if len([]rune(d.Ten)) > 100 {
		return d, errors.New("tên tiệm dài quá 100 ký tự")
	}
	d.Nghe = catBot(strings.TrimSpace(d.Nghe), 100)
	d.LienHe = catBot(strings.TrimSpace(d.LienHe), 60)
	d.DiaChi = catBot(strings.TrimSpace(d.DiaChi), 200)
	d.GhiChu = catBot(strings.TrimSpace(d.GhiChu), 500)

	doiTacMu.Lock()
	defer doiTacMu.Unlock()

	if d.Ma != "" {
		for i, cu := range doiTacDs {
			if cu.Ma != d.Ma {
				continue
			}
			doiTacDs[i] = d
			return d, ghiDoiTac()
		}
		return d, fmt.Errorf("không có đối tác %s", d.Ma)
	}

	goc := maTuTen(d.Ten)
	if goc == "" {
		goc = "doi-tac"
	}
	ma := goc
	for i := 2; ; i++ {
		trung := false
		for _, cu := range doiTacDs {
			if cu.Ma == ma {
				trung = true
				break
			}
		}
		if !trung {
			break
		}
		ma = fmt.Sprintf("%s-%d", goc, i)
	}
	d.Ma = ma
	doiTacDs = append(doiTacDs, d)
	return d, ghiDoiTac()
}

// XoaDoiTac chỉ cho xoá tiệm chưa dính đơn nào. Còn đơn thì bấm "ngừng hợp
// tác": xoá đi là mấy chục đơn cũ trỏ vào khoảng không.
func XoaDoiTac(ma string) error {
	if n := len(LocDon(BoLoc{DoiTac: ma})); n > 0 {
		return fmt.Errorf("tiệm này đang dính %d đơn — dùng \"ngừng hợp tác\" thay vì xoá", n)
	}
	doiTacMu.Lock()
	defer doiTacMu.Unlock()
	for i, d := range doiTacDs {
		if d.Ma != ma {
			continue
		}
		doiTacDs = append(doiTacDs[:i:i], doiTacDs[i+1:]...)
		return ghiDoiTac()
	}
	return fmt.Errorf("không có đối tác %s", ma)
}

// --- Số liệu theo từng tiệm ------------------------------------------

type SoLieuDoiTac struct {
	DoiTac
	SoDon     int
	DangGui   int // đang nằm ở tiệm
	QuaHen    int // tiệm trễ
	DaTra     int // tổng đã trả tiệm
	ConNo     int // đã nhận việc mà chưa trả tiền
	TongChenh int // tổng đơn trừ tiền trả tiệm, trên các đơn của tiệm này
}

func SoLieuCacDoiTac() []SoLieuDoiTac {
	ds := DanhSachDoiTac()
	m := map[string]*SoLieuDoiTac{}
	out := make([]SoLieuDoiTac, 0, len(ds))
	for _, d := range ds {
		out = append(out, SoLieuDoiTac{DoiTac: d})
	}
	for i := range out {
		m[out[i].Ma] = &out[i]
	}
	for _, don := range LocDon(BoLoc{}) {
		s := m[don.GuiDi.MaDoiTac]
		if s == nil {
			continue
		}
		s.SoDon++
		s.TongChenh += don.LaiThat()
		if don.GuiDi.DaTra {
			s.DaTra += don.GuiDi.TraDoiTac
		} else {
			s.ConNo += don.GuiDi.TraDoiTac
		}
		if don.TrangThai == TTDaGuiDi {
			s.DangGui++
		}
		if don.QuaHenDoiTac() {
			s.QuaHen++
		}
	}
	return out
}

// boDau bỏ dấu tiếng Việt để sinh mã. Bảng gõ tay thay vì kéo thêm
// golang.org/x/text: bảng chữ tiếng Việt đóng, không thêm chữ mới bao giờ, và
// dự án đang chỉ có đúng một phụ thuộc ngoài (yaml).
var bangBoDau = map[rune]rune{
	'à': 'a', 'á': 'a', 'ạ': 'a', 'ả': 'a', 'ã': 'a',
	'â': 'a', 'ầ': 'a', 'ấ': 'a', 'ậ': 'a', 'ẩ': 'a', 'ẫ': 'a',
	'ă': 'a', 'ằ': 'a', 'ắ': 'a', 'ặ': 'a', 'ẳ': 'a', 'ẵ': 'a',
	'è': 'e', 'é': 'e', 'ẹ': 'e', 'ẻ': 'e', 'ẽ': 'e',
	'ê': 'e', 'ề': 'e', 'ế': 'e', 'ệ': 'e', 'ể': 'e', 'ễ': 'e',
	'ì': 'i', 'í': 'i', 'ị': 'i', 'ỉ': 'i', 'ĩ': 'i',
	'ò': 'o', 'ó': 'o', 'ọ': 'o', 'ỏ': 'o', 'õ': 'o',
	'ô': 'o', 'ồ': 'o', 'ố': 'o', 'ộ': 'o', 'ổ': 'o', 'ỗ': 'o',
	'ơ': 'o', 'ờ': 'o', 'ớ': 'o', 'ợ': 'o', 'ở': 'o', 'ỡ': 'o',
	'ù': 'u', 'ú': 'u', 'ụ': 'u', 'ủ': 'u', 'ũ': 'u',
	'ư': 'u', 'ừ': 'u', 'ứ': 'u', 'ự': 'u', 'ử': 'u', 'ữ': 'u',
	'ỳ': 'y', 'ý': 'y', 'ỵ': 'y', 'ỷ': 'y', 'ỹ': 'y',
	'đ': 'd',
}

func boDau(s string) string {
	var b strings.Builder
	for _, c := range s {
		if t, co := bangBoDau[c]; co {
			b.WriteRune(t)
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}
