// Câu hỏi thường gặp: trang /cau-hoi và khối FAQPage cho Google.
//
// Khách hỏi lặp đúng mấy câu — bao lâu, bao nhiêu, có bảo hành không, ở tỉnh
// gửi được không. Câu trả lời vốn đã nằm rải trong /quy-trinh và từng trang
// dịch vụ, nhưng không ai đọc hết rồi tự ghép. Gom một chỗ để trả lời một
// lần, và để Google có cái mà nhặt vào ô "Mọi người cũng hỏi".
//
// Đi theo đúng lối data/doi-tac.yaml: ghi ra data/cau-hoi.yaml, mà data/
// không bao giờ bị đè khi deploy — câu Kendy gõ trên máy chủ sống qua mọi
// lần đẩy binary. Chưa có file không phải lỗi, chỉ là chưa có câu nào.
//
// Vì sao là danh sách sửa được chứ không phải mấy khoá cứng trong CayND:
// số câu hỏi thay đổi theo việc trạm nhận. Khoá cứng thì mỗi lần Kendy muốn
// thêm một câu lại phải sửa Go rồi deploy lại.
package core

import (
	"errors"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type CauHoi struct {
	Ma  string `yaml:"ma" json:"ma"`
	Hoi string `yaml:"hoi" json:"hoi"`
	// Dap viết bằng markdown gọn — xuống dòng, **đậm**, [link](/dich-vu).
	// Cho markdown vì câu trả lời hay phải trỏ sang trang khác, mà bắt khách
	// tự đi tìm "trang dịch vụ" thì mất nửa số người ngay tại đó.
	Dap string `yaml:"dap" json:"dap"`
	// ThuTu nhỏ đứng trước. Thứ tự do người đặt chứ không theo abc: câu
	// "sửa mất bao lâu" phải nằm trên cùng, còn abc thì nó rơi xuống giữa.
	ThuTu int `yaml:"thu_tu" json:"thu_tu"`
	// An: giấu khỏi trang khách mà vẫn giữ chữ. Mùa cao điểm trạm ngừng nhận
	// hàng tỉnh thì tắt câu đó đi, hết mùa bật lại, không phải gõ lại.
	An bool `yaml:"an" json:"an"`
}

// DapHTML dựng câu trả lời từ markdown gọn.
func (c CauHoi) DapHTML() template.HTML { return MarkdownGon(c.Dap) }

// DapTron là câu trả lời bóc hết markdown, dùng cho khối JSON-LD và cho thẻ
// mô tả. Google đọc answerText như văn bản; để nguyên dấu sao và ngoặc vuông
// thì ô "Mọi người cũng hỏi" hiện ra đúng mấy ký tự rác đó.
func (c CauHoi) DapTron() string {
	return chuTron(string(MarkdownGon(c.Dap)))
}

// theHTML khớp một thẻ HTML bất kỳ. Đủ dùng vì chuỗi đưa vào đây là đầu ra
// của MarkdownGon — HTML do mình dựng, không phải HTML người lạ gửi lên.
var theHTML = regexp.MustCompile(`<[^>]*>`)

// khoangThua khớp khoảng trắng đứng ngay trước một dấu câu.
var khoangThua = regexp.MustCompile(`\s+([,.;:!?\)\]…])`)

// chuTron bóc thẻ khỏi một đoạn HTML ngắn. Thay thẻ bằng khoảng trắng chứ
// không bằng rỗng: "<p>a</p><p>b</p>" mà nối trực tiếp thì thành "ab".
//
// Nhưng thẻ nội dòng thì ngược lại: "**1–3 ngày**." ra <b>1–3 ngày</b>. —
// chèn khoảng trắng vào chỗ </b> thì thành "1–3 ngày ." Rẻ hơn là cứ chèn
// hết rồi dọn khoảng trắng đứng trước dấu câu, vì phân biệt thẻ khối với thẻ
// nội dòng phải giữ thêm một danh sách tên thẻ đi kèm markdown.go.
func chuTron(h string) string {
	s := html.UnescapeString(theHTML.ReplaceAllString(h, " "))
	s = strings.Join(strings.Fields(s), " ")
	return khoangThua.ReplaceAllString(s, "$1")
}

type khoCauHoiFile struct {
	CauHoi []CauHoi `yaml:"cau_hoi"`
}

var (
	cauHoiMu sync.RWMutex
	cauHoiDs []CauHoi
)

func fileCauHoi() string { return P("data/cau-hoi.yaml") }

// NapCauHoi đọc danh sách. Chưa có file không phải lỗi.
func NapCauHoi() error {
	b, err := os.ReadFile(fileCauHoi())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var f khoCauHoiFile
	if err := yaml.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("%s hỏng: %w", fileCauHoi(), err)
	}
	cauHoiMu.Lock()
	cauHoiDs = f.CauHoi
	cauHoiMu.Unlock()
	return nil
}

func ghiCauHoi() error {
	b, err := yaml.Marshal(khoCauHoiFile{CauHoi: cauHoiDs})
	if err != nil {
		return err
	}
	dau := []byte("# Câu hỏi thường gặp, hiện ở /cau-hoi. Sửa ở /qt/cau-hoi.\n\n")
	return ghiAtomic(fileCauHoi(), append(dau, b...))
}

// DanhSachCauHoi — cả câu đang ẩn, dùng cho trang quản lý.
func DanhSachCauHoi() []CauHoi {
	cauHoiMu.RLock()
	defer cauHoiMu.RUnlock()
	out := append([]CauHoi{}, cauHoiDs...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ThuTu < out[j].ThuTu })
	return out
}

// CauHoiHien — chỉ câu còn bật, dùng cho trang khách.
func CauHoiHien() []CauHoi {
	out := []CauHoi{}
	for _, c := range DanhSachCauHoi() {
		if !c.An {
			out = append(out, c)
		}
	}
	return out
}

func TimCauHoi(ma string) (CauHoi, bool) {
	cauHoiMu.RLock()
	defer cauHoiMu.RUnlock()
	for _, c := range cauHoiDs {
		if c.Ma == ma {
			return c, true
		}
	}
	return CauHoi{}, false
}

// LuuCauHoi thêm mới hoặc sửa. Mã rỗng là thêm mới.
func LuuCauHoi(c CauHoi) (CauHoi, error) {
	c.Hoi = strings.Join(strings.Fields(c.Hoi), " ")
	if c.Hoi == "" {
		return c, errors.New("phải có câu hỏi")
	}
	if len([]rune(c.Hoi)) > 200 {
		return c, errors.New("câu hỏi dài quá 200 ký tự")
	}
	c.Dap = strings.TrimSpace(c.Dap)
	if c.Dap == "" {
		return c, errors.New("phải có câu trả lời")
	}
	c.Dap = catBot(c.Dap, 2000)

	cauHoiMu.Lock()
	defer cauHoiMu.Unlock()

	if c.Ma != "" {
		for i, cu := range cauHoiDs {
			if cu.Ma != c.Ma {
				continue
			}
			cauHoiDs[i] = c
			return c, ghiCauHoi()
		}
		return c, fmt.Errorf("không có câu hỏi %s", c.Ma)
	}

	// Mã lấy từ chính câu hỏi để đường dẫn neo /cau-hoi#sua-mat-bao-lau đọc
	// được, nhưng cắt ngắn: câu hỏi dài hai dòng mà làm mã thì cái neo dài
	// hơn cả tên miền. Cắt theo ký tự rồi bỏ dấu gạch thừa ở đuôi.
	goc := strings.Trim(catBot(maTuTen(c.Hoi), 40), "-")
	if goc == "" {
		goc = "cau-hoi"
	}
	ma := goc
	for i := 2; ; i++ {
		trung := false
		for _, cu := range cauHoiDs {
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
	c.Ma = ma
	// Câu mới xuống cuối danh sách chứ không chen lên đầu: Kendy gõ thêm một
	// câu phụ mà nó nhảy lên trên câu "sửa mất bao lâu" thì lần nào thêm
	// cũng phải đi sắp lại thứ tự.
	if c.ThuTu == 0 {
		max := 0
		for _, cu := range cauHoiDs {
			if cu.ThuTu > max {
				max = cu.ThuTu
			}
		}
		c.ThuTu = max + 10
	}
	cauHoiDs = append(cauHoiDs, c)
	return c, ghiCauHoi()
}

// XoaCauHoi bỏ hẳn một câu. Không như đối tác, câu hỏi không bị thứ gì trỏ
// vào nên xoá là xoá — muốn giữ chữ mà không hiện thì bấm ẩn.
func XoaCauHoi(ma string) error {
	cauHoiMu.Lock()
	defer cauHoiMu.Unlock()
	for i, c := range cauHoiDs {
		if c.Ma != ma {
			continue
		}
		cauHoiDs = append(cauHoiDs[:i:i], cauHoiDs[i+1:]...)
		return ghiCauHoi()
	}
	return fmt.Errorf("không có câu hỏi %s", ma)
}
