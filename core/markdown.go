package core

// Trình dựng markdown viết tay, dùng cho thân bài viết và cho những ô nội
// dung dài trong admin.
//
// Vì sao không kéo goldmark hay blackfriday: cả app đang zero-dependency
// ngoài yaml, mà thứ cần ở đây chỉ là bảy khối và bốn kiểu nhấn chữ. Một
// trình đủ dùng gói gọn trong một tệp thì đọc hết trong mười phút; một thư
// viện markdown đầy đủ kéo theo vài chục nghìn dòng cho đúng ngần ấy việc.
//
// Nguồn markdown là chữ Kendy gõ trong admin, KHÔNG phải chữ khách gõ —
// nhưng vẫn escape HTML mặc định. Lý do không phải sợ Kendy chèn <script>:
// lý do là gõ nhầm một dấu < trong bài viết không được phép làm vỡ trang.
//
// Những gì dựng được — đúng bằng những gì 13 bài viết cũ đang dùng, không
// hơn:
//
//	## ###           tiêu đề mục và tiểu mục
//	đoạn văn         cách nhau bằng dòng trống
//	- gạch đầu dòng  ra <ul class="gach">
//	1. đánh số       ra <ol>
//	> khung lưu ý    ra <div class="luu-y">
//	| bảng |         ra <table class="bang">
//	:::hinh <tên>    chèn một hình vẽ SVG có sẵn, kèm chú thích
//	![chú](đường)    ảnh, đứng một mình thì thành figure có chú thích
//	**đậm** *nghiêng* [chữ](đường) `mã`
//	---              vạch ngang
//	{khoá}           thay bằng số liệu thật, xem bangThe

import (
	"bytes"
	"html"
	"html/template"
	"regexp"
	"strconv"
	"strings"
)

// hinhChoPhep — chỉ những hình vẽ có sẵn trong ui/hinh.html mới gọi được từ
// markdown. Danh sách trắng chứ không cho gọi tên template bất kỳ: một ô
// nhập trong admin không nên là đường gọi tới mọi template của app.
var hinhChoPhep = map[string]string{
	"vot-bo":    "Bản vẽ cây vợt và các vị trí trên vợt",
	"cat-lop":   "Mặt cắt các lớp của vợt",
	"vung-hong": "Các vùng hay hỏng trên vợt",
	"can":       "Bản vẽ cán vợt và cổ vợt",
	"de-giay":   "Mặt cắt hai tầng đế giày",
	"vien":      "Mặt cắt mép vợt: nẹp viền, keo hai mặt, chỗ bong",
	"lead-tape": "Ba vị trí dán chì trên mặt vợt",
	"nham":      "Mặt cắt bề mặt: bên còn nhám, bên đã mòn lì",
	"ve-sinh":   "Mặt vợt nửa bẩn nửa sạch",
	"am-vao":    "Ba đường ẩm đi vào cây vợt",
	"bo-anh":    "Bốn khung ảnh cần chụp khi gửi ảnh vợt hỏng",
	"dong-goi":  "Mặt cắt thùng đóng gói vợt gửi đi tỉnh",
}

// bangThe dựng bảng thay thế cho các khoá {…}. Chỉ vài con số mà bài viết
// cần trích đúng, không phải một ngôn ngữ template thu nhỏ: chữ trong bài do
// người viết, còn con số thì phải đi ra từ bảng giá chứ không gõ tay — gõ tay
// là có ngày bài viết hứa 3 gram mà phiếu in 5 gram.
func bangThe() map[string]string {
	return map[string]string{
		"nguong_can": strconv.FormatFloat(NguongHienTai().TangKhoiLuongToiDaG, 'f', -1, 64),
		"free_ship":  dinhDangTien(NguongHienTai().FreeShipVeTuDong),
		"ten_xuong":  CFG.ThuongHieu.Ten,
	}
}

// MarkdownBai dựng thân một bài viết: đoạn đầu tiên thành đoạn dẫn khổ chữ
// lớn, đúng như <p class="dat"> của mấy bài viết bản cũ.
func MarkdownBai(src string) template.HTML { return template.HTML(dungMD(src, true)) }

// MarkdownGon dựng một ô nội dung dài của trang tĩnh. Không có đoạn dẫn —
// đoạn đầu của một ô mô tả chỉ là đoạn đầu, không phải lời mở bài.
func MarkdownGon(src string) template.HTML { return template.HTML(dungMD(src, false)) }

// MarkdownDong dựng MỘT dòng chữ: chỉ **đậm**, *nghiêng*, [chữ](/duong-dan)
// và mã, không có thẻ khối nào. Dùng cho nhãn nút, câu gợi ý dưới ô nhập,
// dòng bản quyền — mấy chỗ mà một cặp thẻ <p> bọc quanh là hỏng bố cục.
func MarkdownDong(src string) template.HTML {
	src = strings.ReplaceAll(src, "\r\n", " ")
	src = strings.ReplaceAll(src, "\n", " ")
	src = strings.ReplaceAll(src, "\x00", "")
	d := &mdDung{the: bangThe()}
	return template.HTML(d.inline(src))
}

func dungMD(src string, coDat bool) string {
	src = strings.ReplaceAll(src, "\r\n", "\n")
	src = strings.ReplaceAll(src, "\r", "\n")
	src = strings.ReplaceAll(src, "\x00", "")

	d := &mdDung{dong: strings.Split(src, "\n"), conDat: coDat, the: bangThe()}
	d.chay()
	return d.ra.String()
}

type mdDung struct {
	dong   []string
	i      int
	ra     strings.Builder
	conDat bool // đoạn tiếp theo có phải đoạn dẫn không
	the    map[string]string
}

func (d *mdDung) het() bool     { return d.i >= len(d.dong) }
func (d *mdDung) nay() string   { return d.dong[d.i] }
func (d *mdDung) trong() bool   { return strings.TrimSpace(d.nay()) == "" }
func (d *mdDung) tien()         { d.i++ }
func (d *mdDung) viet(s string) { d.ra.WriteString(s) }

func (d *mdDung) chay() {
	for !d.het() {
		if d.trong() {
			d.tien()
			continue
		}
		s := strings.TrimSpace(d.nay())
		switch {
		case d.laTieuDe():
			d.tieuDe()
		case strings.HasPrefix(s, ":::hinh"):
			d.hinh()
		case d.laVach():
			d.viet("<hr>\n")
			d.tien()
		case strings.HasPrefix(s, ">"):
			d.luuY()
		case d.laBang():
			d.bang()
		case d.laGach(s):
			d.danhSach(false)
		case d.laSo(s):
			d.danhSach(true)
		default:
			d.doan()
		}
	}
}

// --- Từng khối --------------------------------------------------------

var reTieuDe = regexp.MustCompile(`^(#{1,4})\s+(.*)$`)

func (d *mdDung) laTieuDe() bool { return reTieuDe.MatchString(strings.TrimSpace(d.nay())) }

// tieuDe: bài viết chỉ có H2 và H3, vì H1 đã là tiêu đề bài in ở tấm đầu
// trang. Ai gõ "#" một dấu thì vẫn ra H2 chứ không ra hai cái H1 trên một
// trang — đó là lỗi SEO chứ không phải sở thích trình bày.
func (d *mdDung) tieuDe() {
	m := reTieuDe.FindStringSubmatch(strings.TrimSpace(d.nay()))
	d.tien()
	muc := len(m[1])
	if muc < 2 {
		muc = 2
	}
	if muc > 4 {
		muc = 4
	}
	t := strconv.Itoa(muc)
	d.viet("<h" + t + ">" + d.inline(m[2]) + "</h" + t + ">\n")
}

func (d *mdDung) laVach() bool {
	s := strings.TrimSpace(d.nay())
	return s == "---" || s == "***"
}

// hinh: ":::hinh vot-bo" mở khối, chú thích nằm ở các dòng sau, ":::" đóng.
// Chú thích để trống được — có hình không chú thích vẫn hơn là không dựng
// được hình.
func (d *mdDung) hinh() {
	ten := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(d.nay()), ":::hinh"))
	d.tien()
	var chu []string
	for !d.het() && strings.TrimSpace(d.nay()) != ":::" {
		chu = append(chu, d.nay())
		d.tien()
	}
	if !d.het() {
		d.tien() // nuốt dòng ":::"
	}
	if _, ok := hinhChoPhep[ten]; !ok {
		// Tên hình sai thì hiện một dòng nhắc ngay trên trang, không im lặng
		// bỏ qua: im lặng thì Kendy gõ nhầm tên xong tưởng hình chưa vẽ.
		d.viet(`<p class="luu-y">Không có hình tên “` + html.EscapeString(ten) + `”.</p>` + "\n")
		return
	}
	var b bytes.Buffer
	if err := tpl.ExecuteTemplate(&b, "hinh-"+ten, nil); err != nil {
		return
	}
	d.viet(`<figure class="hinh-o">` + "\n" + b.String() + "\n")
	if s := strings.TrimSpace(strings.Join(chu, " ")); s != "" {
		d.viet("<figcaption>" + d.inline(s) + "</figcaption>\n")
	}
	d.viet("</figure>\n")
}

// luuY: khối trích dẫn của markdown dựng ra cái khung ".luu-y" chứ không ra
// <blockquote>. Bài viết ở đây không trích lời ai — chỗ nào dùng khung trích
// cũng là để nhấn một câu dặn dò, và CSS chỉ có sẵn khung đó.
func (d *mdDung) luuY() {
	var dong []string
	for !d.het() && !d.trong() {
		s := strings.TrimSpace(d.nay())
		if !strings.HasPrefix(s, ">") {
			break
		}
		dong = append(dong, strings.TrimSpace(strings.TrimPrefix(s, ">")))
		d.tien()
	}
	d.viet(`<div class="luu-y">` + d.inline(strings.Join(dong, " ")) + "</div>\n")
}

func (d *mdDung) laGach(s string) bool {
	t := strings.TrimSpace(s)
	return strings.HasPrefix(t, "- ") || strings.HasPrefix(t, "* ")
}

var reSo = regexp.MustCompile(`^\s*\d+[.)]\s+`)

func (d *mdDung) laSo(s string) bool { return reSo.MatchString(s) }

// danhSach gom cả những dòng xuống hàng giữa chừng vào cùng một mục: người
// viết ngắt dòng cho vừa màn hình chứ không phải để tách ý.
func (d *mdDung) danhSach(so bool) {
	var muc []string
	for !d.het() && !d.trong() {
		s := d.nay()
		switch {
		case so && d.laSo(s):
			muc = append(muc, reSo.ReplaceAllString(s, ""))
		case !so && d.laGach(s):
			muc = append(muc, strings.TrimSpace(strings.TrimSpace(s)[2:]))
		case len(muc) > 0:
			muc[len(muc)-1] += " " + strings.TrimSpace(s)
		default:
			return
		}
		d.tien()
	}
	mo, dong := `<ul class="gach">`, "</ul>"
	if so {
		mo, dong = "<ol>", "</ol>"
	}
	d.viet(mo + "\n")
	for _, m := range muc {
		d.viet("<li>" + d.inline(m) + "</li>\n")
	}
	d.viet(dong + "\n")
}

// laBang: một dòng bắt đầu bằng "|" và dòng ngay dưới là dòng gạch ngăn.
// Bắt buộc phải có dòng gạch, để một đoạn văn lỡ có dấu | ở đầu không biến
// thành bảng một cột.
func (d *mdDung) laBang() bool {
	if !strings.HasPrefix(strings.TrimSpace(d.nay()), "|") || d.i+1 >= len(d.dong) {
		return false
	}
	sau := strings.TrimSpace(d.dong[d.i+1])
	if !strings.HasPrefix(sau, "|") {
		return false
	}
	for _, c := range sau {
		if c != '|' && c != '-' && c != ':' && c != ' ' {
			return false
		}
	}
	return strings.Contains(sau, "-")
}

func oBang(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "|")
	s = strings.TrimSuffix(s, "|")
	o := strings.Split(s, "|")
	for i := range o {
		o[i] = strings.TrimSpace(o[i])
	}
	return o
}

func (d *mdDung) bang() {
	dau := oBang(d.nay())
	d.tien()
	d.tien() // dòng gạch ngăn
	var than [][]string
	for !d.het() && strings.HasPrefix(strings.TrimSpace(d.nay()), "|") {
		than = append(than, oBang(d.nay()))
		d.tien()
	}
	// Bảng rộng hơn cột chữ thì cho cuộn ngang, đừng để nó đẩy vỡ bố cục
	// trên điện thoại.
	d.viet(`<div style="overflow-x:auto"><table class="bang">` + "\n<thead><tr>")
	for _, o := range dau {
		d.viet("<th>" + d.inline(o) + "</th>")
	}
	d.viet("</tr></thead>\n<tbody>\n")
	for _, h := range than {
		d.viet("<tr>")
		for i := range dau {
			o := ""
			if i < len(h) {
				o = h[i]
			}
			d.viet("<td>" + d.inline(o) + "</td>")
		}
		d.viet("</tr>\n")
	}
	d.viet("</tbody></table></div>\n")
}

var reAnhMot = regexp.MustCompile(`^!\[([^\]]*)\]\(([^)\s]+)\)$`)

func (d *mdDung) doan() {
	var dong []string
	for !d.het() && !d.trong() {
		s := strings.TrimSpace(d.nay())
		if len(dong) > 0 && (d.laTieuDe() || d.laVach() || d.laGach(s) || d.laSo(s) ||
			strings.HasPrefix(s, ">") || strings.HasPrefix(s, ":::")) {
			break
		}
		dong = append(dong, s)
		d.tien()
	}
	noi := strings.Join(dong, " ")
	if strings.TrimSpace(noi) == "" {
		return
	}

	// Ảnh đứng một mình thành figure có chú thích, giống mấy hình vẽ. Ảnh
	// nằm giữa câu thì vẫn là ảnh chèn trong dòng.
	if m := reAnhMot.FindStringSubmatch(noi); m != nil {
		if u := duongDanSach(m[2]); u != "" {
			d.viet(`<figure class="hinh-o"><img src="` + html.EscapeString(u) +
				`" alt="` + html.EscapeString(m[1]) + `" loading="lazy">` + "\n")
			if m[1] != "" {
				d.viet("<figcaption>" + d.inline(m[1]) + "</figcaption>\n")
			}
			d.viet("</figure>\n")
			return
		}
	}

	lop := ""
	if d.conDat {
		lop = ` class="dat"`
		d.conDat = false
	}
	d.viet("<p" + lop + ">" + d.inline(noi) + "</p>\n")
}

// --- Nhấn chữ trong dòng ---------------------------------------------

var (
	reMa       = regexp.MustCompile("`([^`]+)`")
	reAnhTrong = regexp.MustCompile(`!\[([^\]]*)\]\(([^)\s]+)\)`)
	reLink     = regexp.MustCompile(`\[([^\]]+)\]\(([^)\s]+)\)`)
	reDam      = regexp.MustCompile(`\*\*([^*]+)\*\*`)
	reNghieng  = regexp.MustCompile(`\*([^*]+)\*`)
	reThe      = regexp.MustCompile(`\{([a-z0-9_]+)\}`)
)

// duongDanSach chặn javascript: và data: — chữ trong ô admin thì tin được,
// nhưng một đường dẫn dán nhầm từ chỗ khác thì không đáng để tin. Trả về
// chuỗi rỗng nghĩa là bỏ cái link đó đi.
func duongDanSach(u string) string {
	t := strings.TrimSpace(u)
	if t == "" {
		return ""
	}
	if strings.HasPrefix(t, "/") || strings.HasPrefix(t, "#") {
		return t
	}
	l := strings.ToLower(t)
	for _, ok := range []string{"http://", "https://", "mailto:", "tel:"} {
		if strings.HasPrefix(l, ok) {
			return t
		}
	}
	return ""
}

// inline chạy theo thứ tự: escape trước, rồi mới chèn thẻ. Sau bước escape
// thì trong chuỗi không còn dấu < nào của người viết, nên mọi thẻ chèn thêm
// đều là thẻ của mình.
//
// Mã trong dấu huyền được rút ra giữ chỗ trước tiên, để một dấu * nằm trong
// đoạn mã không bị hiểu thành in nghiêng.
func (d *mdDung) inline(s string) string {
	var ma []string
	s = reMa.ReplaceAllStringFunc(s, func(m string) string {
		ma = append(ma, reMa.FindStringSubmatch(m)[1])
		return "\x00" + strconv.Itoa(len(ma)-1) + "\x00"
	})

	s = html.EscapeString(s)

	s = reThe.ReplaceAllStringFunc(s, func(m string) string {
		k := reThe.FindStringSubmatch(m)[1]
		if v, ok := d.the[k]; ok {
			return html.EscapeString(v)
		}
		return m // khoá lạ thì để nguyên chữ, đừng nuốt mất
	})

	s = reAnhTrong.ReplaceAllStringFunc(s, func(m string) string {
		p := reAnhTrong.FindStringSubmatch(m)
		u := duongDanSach(html.UnescapeString(p[2]))
		if u == "" {
			return p[1]
		}
		return `<img src="` + html.EscapeString(u) + `" alt="` + p[1] + `" loading="lazy">`
	})

	s = reLink.ReplaceAllStringFunc(s, func(m string) string {
		p := reLink.FindStringSubmatch(m)
		u := duongDanSach(html.UnescapeString(p[2]))
		if u == "" {
			return p[1]
		}
		ngoai := ""
		if strings.HasPrefix(strings.ToLower(u), "http") {
			ngoai = ` rel="nofollow noopener" target="_blank"`
		}
		return `<a href="` + html.EscapeString(u) + `"` + ngoai + `>` + p[1] + `</a>`
	})

	s = reDam.ReplaceAllString(s, "<b>$1</b>")
	s = reNghieng.ReplaceAllString(s, "<em>$1</em>")

	for i, m := range ma {
		s = strings.ReplaceAll(s, "\x00"+strconv.Itoa(i)+"\x00", "<code>"+html.EscapeString(m)+"</code>")
	}
	return s
}
