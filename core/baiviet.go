// Mục bài viết: nội dung SEO cho trang khách.
//
// Bài viết nằm ở data/bai-viet/<slug>.md — front-matter YAML ở đầu, thân bài
// là markdown. Trước đây mỗi bài là một template nhúng trong binary; đổi một
// dấu phẩy phải build lại và rsync binary lên VPS, nên bài viết trên thực tế
// là thứ không ai sửa. Giờ sửa ở /qt/bai-viet.
//
// KHÔNG có bản dự phòng nhúng trong code, khác với data/noi-dung.yaml của
// trang tĩnh. Hai thứ khác nhau ở chỗ này: chữ của trang tĩnh mà thiếu thì
// trang thủng một lỗ, còn bài viết mà thiếu thì chỉ là chưa có bài — mục bài
// viết trống vẫn là một trang chạy được. Giữ 13 bài trong binary chỉ để
// phòng hờ là tự tạo ra hai nguồn sự thật lệch nhau.
//
// Đổi lại: data/bai-viet/ PHẢI đi cùng khi deploy. Xem KE-HOACH-CMS.md.
package core

import (
	"encoding/xml"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type BaiViet struct {
	Slug string `yaml:"-"` // tên tệp, cũng là đường dẫn /bai-viet/<slug>

	// ThuTu quyết định thứ tự trên trang danh sách, nhỏ đứng trước. Cách
	// nhau 10 để chèn một bài vào giữa mà không phải đánh số lại cả mục.
	ThuTu  int    `yaml:"thu_tu"`
	TieuDe string `yaml:"tieu_de"` // <h1> in trên đầu bài, viết cho người đọc
	// TieuDeSEO: bản rút gọn chỉ dùng cho <title> và thẻ chia sẻ. Google cắt
	// tiêu đề ở khoảng 60 ký tự, mà H1 hay dài hơn thế vì nó là một câu nói
	// với người đọc. Tách hai chỗ ra thì không phải bóp H1 cho vừa ô kết quả
	// tìm kiếm. Để trống thì lấy luôn TieuDe.
	TieuDeSEO string `yaml:"tieu_de_seo,omitempty"`
	MoTa      string `yaml:"mo_ta"`   // <meta name=description>, 150-160 ký tự
	Ngay      string `yaml:"ngay"`    // ngày đăng, ISO
	Sua       string `yaml:"sua"`     // ngày sửa gần nhất, rỗng nếu chưa sửa
	TomTat    string `yaml:"tom_tat"` // hiện ở trang danh sách
	Anh       string `yaml:"anh"`     // ảnh bìa, tên tệp trong data/bai-viet-anh
	Nhap      bool   `yaml:"nhap"`    // bản nháp: không hiện ra trang khách

	Than string `yaml:"-"` // thân bài, markdown
}

// TieuDeTab: chữ cho <title> và og:title. Xem TieuDeSEO ở trên.
func (b BaiViet) TieuDeTab() string {
	if s := strings.TrimSpace(b.TieuDeSEO); s != "" {
		return s
	}
	return b.TieuDe
}

func (b BaiViet) NgaySua() string {
	if b.Sua != "" {
		return b.Sua
	}
	return b.Ngay
}

var (
	baiMu sync.RWMutex
	baiDS []BaiViet
)

func thuMucBai() string    { return P("data/bai-viet") }
func thuMucAnhBai() string { return P("data/bai-viet-anh") }

// slugSach: tên tệp do người gõ vào ô nhập, nên phải soi trước khi ghép vào
// đường dẫn. Chỉ chữ thường, số và gạch nối — cũng là hình dạng của một
// đường dẫn tử tế cho máy tìm kiếm.
func slugSach(s string) bool {
	if s == "" || len(s) > 80 {
		return false
	}
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false
	}
	return !strings.HasPrefix(s, "-") && !strings.HasSuffix(s, "-")
}

// NapBaiViet đọc cả thư mục lúc khởi động. Một bài hỏng front-matter thì bỏ
// qua bài đó và kêu ra stderr, không làm sập server: mười hai bài còn lại
// vẫn phải lên được trang.
func NapBaiViet() error {
	ds, err := docThuMucBai()
	if err != nil {
		return err
	}
	baiMu.Lock()
	baiDS = ds
	baiMu.Unlock()
	return nil
}

func docThuMucBai() ([]BaiViet, error) {
	muc, err := os.ReadDir(thuMucBai())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ds []BaiViet
	for _, m := range muc {
		if m.IsDir() || !strings.HasSuffix(m.Name(), ".md") {
			continue
		}
		slug := strings.TrimSuffix(m.Name(), ".md")
		b, err := docMotBai(slug)
		if err != nil {
			fmt.Printf("[bài viết] bỏ qua %s: %v\n", m.Name(), err)
			continue
		}
		ds = append(ds, b)
	}
	xepBai(ds)
	return ds, nil
}

// xepBai: thứ tự tay trước, rồi tới ngày mới nhất. Bài nào chưa đặt thứ tự
// (thu_tu = 0) thì đứng đầu — bài mới viết xong nằm ngay trên cùng để còn
// nhìn thấy mà sắp chỗ cho nó.
func xepBai(ds []BaiViet) {
	sort.SliceStable(ds, func(i, j int) bool {
		if ds[i].ThuTu != ds[j].ThuTu {
			return ds[i].ThuTu < ds[j].ThuTu
		}
		return ds[i].Ngay > ds[j].Ngay
	})
}

func docMotBai(slug string) (BaiViet, error) {
	if !slugSach(slug) {
		return BaiViet{}, fmt.Errorf("tên tệp không hợp lệ")
	}
	raw, err := os.ReadFile(filepath.Join(thuMucBai(), slug+".md"))
	if err != nil {
		return BaiViet{}, err
	}
	dau, than, err := tachFrontMatter(string(raw))
	if err != nil {
		return BaiViet{}, err
	}
	var b BaiViet
	if err := yaml.Unmarshal([]byte(dau), &b); err != nil {
		return BaiViet{}, fmt.Errorf("front-matter hỏng: %w", err)
	}
	b.Slug = slug
	b.Than = than
	if strings.TrimSpace(b.TieuDe) == "" {
		return BaiViet{}, fmt.Errorf("thiếu tieu_de")
	}
	return b, nil
}

// tachFrontMatter cắt khối YAML giữa hai dòng "---" ở đầu tệp. Thiếu khối
// đó là lỗi chứ không phải "coi cả tệp là thân bài": một bài không tiêu đề
// thì lên trang cũng không dùng được, thà báo sớm.
func tachFrontMatter(s string) (dau, than string, err error) {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	if !strings.HasPrefix(s, "---\n") {
		return "", "", fmt.Errorf("thiếu front-matter (khối --- ở đầu tệp)")
	}
	con := s[4:]
	i := strings.Index(con, "\n---")
	if i < 0 {
		return "", "", fmt.Errorf("front-matter không đóng")
	}
	dau = con[:i]
	than = con[i+4:]
	than = strings.TrimPrefix(than, "\n")
	return dau, than, nil
}

// --- Đọc danh sách ----------------------------------------------------

// DsBaiViet — những bài khách xem được. Bản nháp không nằm trong này, nên
// cũng không lọt vào sitemap hay ô "đọc tiếp".
func DsBaiViet() []BaiViet {
	baiMu.RLock()
	defer baiMu.RUnlock()
	ds := make([]BaiViet, 0, len(baiDS))
	for _, b := range baiDS {
		if !b.Nhap {
			ds = append(ds, b)
		}
	}
	return ds
}

// DsBaiVietQt — cả nháp lẫn bài đã đăng, cho trang quản trị.
func DsBaiVietQt() []BaiViet {
	baiMu.RLock()
	defer baiMu.RUnlock()
	ds := make([]BaiViet, len(baiDS))
	copy(ds, baiDS)
	return ds
}

func TimBaiViet(slug string) (BaiViet, bool) {
	baiMu.RLock()
	defer baiMu.RUnlock()
	for _, b := range baiDS {
		if b.Slug == slug {
			return b, true
		}
	}
	return BaiViet{}, false
}

// --- Ghi ---------------------------------------------------------------

// LuuBaiViet ghi một bài ra đĩa rồi nạp lại cả thư mục. Nạp lại cả thư mục
// chứ không vá vào lát cắt đang có: chỉ tốn một lần đọc mười ba tệp nhỏ, đổi
// lại thứ tự và trạng thái trong bộ nhớ luôn đúng bằng thứ nằm trên đĩa.
func LuuBaiViet(b BaiViet) error {
	if !slugSach(b.Slug) {
		return fmt.Errorf("đường dẫn chỉ được có chữ thường không dấu, số và gạch nối")
	}
	if strings.TrimSpace(b.TieuDe) == "" {
		return fmt.Errorf("chưa có tiêu đề")
	}
	dau, err := yaml.Marshal(b)
	if err != nil {
		return err
	}
	than := strings.ReplaceAll(b.Than, "\r\n", "\n")
	noi := "---\n" + string(dau) + "---\n\n" + strings.TrimRight(than, "\n") + "\n"
	if err := os.MkdirAll(thuMucBai(), 0o755); err != nil {
		return err
	}
	if err := ghiAtomic(filepath.Join(thuMucBai(), b.Slug+".md"), []byte(noi)); err != nil {
		return err
	}
	return NapBaiViet()
}

func XoaBaiViet(slug string) error {
	if !slugSach(slug) {
		return fmt.Errorf("đường dẫn không hợp lệ")
	}
	if err := os.Remove(filepath.Join(thuMucBai(), slug+".md")); err != nil && !os.IsNotExist(err) {
		return err
	}
	return NapBaiViet()
}

// --- Trang khách -------------------------------------------------------

type dlBaiViet struct {
	Chung
	Bai    BaiViet
	DS     []BaiViet
	Khac   []BaiViet // vài bài khác, hiện ở cột bên
	Than   template.HTML
	DichVu []DichVu
	Nguong Nguong
}

// baiKhac lấy n bài kế tiếp trong danh sách, vòng lại từ đầu khi hết. Không
// chọn theo chủ đề vì chưa có trường phân loại — mà lấy các bài liền kề vẫn
// hơn lấy ngẫu nhiên: danh sách đang xếp theo thứ tự người mới nên đọc.
func baiKhac(ds []BaiViet, slug string, n int) []BaiViet {
	if len(ds) <= 1 {
		return nil
	}
	i := 0
	for k, b := range ds {
		if b.Slug == slug {
			i = k
			break
		}
	}
	var ra []BaiViet
	for b := 1; b < len(ds) && len(ra) < n; b++ {
		ra = append(ra, ds[(i+b)%len(ds)])
	}
	return ra
}

func hBaiVietList(w http.ResponseWriter, r *http.Request) {
	c := chung(r, "bai-viet")
	render(w, "baiviet.html", dlBaiViet{Chung: c, DS: DsBaiViet()})
}

func hBaiVietMot(w http.ResponseWriter, r *http.Request) {
	bai, ok := TimBaiViet(r.PathValue("slug"))
	if !ok || bai.Nhap {
		http.NotFound(w, r)
		return
	}
	ds := DsBaiViet()
	c := chung(r, "bai-viet")
	c.TieuDe = bai.TieuDeTab()
	c.MoTa = bai.MoTa
	c.LoaiOG = "article"
	c.NgayDang = bai.Ngay
	c.NgayCapNhat = bai.NgaySua()
	// Bài có ảnh bìa thì ảnh ấy đi theo link, không thì rơi về tấm mặc định.
	// Bỏ cỡ đi: ảnh bìa do người tải lên, cỡ nào cũng có, khai bừa 1200×630
	// thì Facebook cắt sai chỗ.
	if bai.Anh != "" {
		c.AnhChiaSe = goc(r) + "/bai-viet-anh/" + bai.Anh
		c.AnhRong, c.AnhCao = 0, 0
	}
	render(w, "baiviet-mot.html", dlBaiViet{
		Chung:  c,
		Bai:    bai,
		DS:     ds,
		Khac:   baiKhac(ds, bai.Slug, 4),
		Than:   MarkdownBai(bai.Than),
		DichVu: DichVuDangBan(GiaiDoan),
		Nguong: NguongHienTai(),
	})
}

// hAnhBaiViet phục vụ ảnh chèn trong bài. Chỉ mở tên có thật trong thư mục
// ảnh bài viết và không cho tên nào ghép được đường thoát ra ngoài — cùng
// luật với ảnh đơn hàng và ảnh trang chủ.
func hAnhBaiViet(w http.ResponseWriter, r *http.Request) {
	ten := r.PathValue("ten")
	if !tenAnhSach(ten) {
		http.NotFound(w, r)
		return
	}
	duong := filepath.Join(thuMucAnhBai(), ten)
	if _, err := os.Stat(duong); err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeFile(w, r, duong)
}

// --- Sitemap và robots ------------------------------------------------

// goc dựng gốc URL từ chính request. Không lấy từ config vì trạm chưa có
// tên miền: hôm nay chạy IP trần, mai gắn domain thì sitemap tự đúng theo,
// không phải nhớ sửa một dòng cấu hình.
func goc(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

type urlSitemap struct {
	XMLName xml.Name `xml:"url"`
	Loc     string   `xml:"loc"`
	LastMod string   `xml:"lastmod,omitempty"`
	Uu      string   `xml:"priority"`
}

type urlSet struct {
	XMLName xml.Name     `xml:"urlset"`
	NS      string       `xml:"xmlns,attr"`
	URL     []urlSitemap `xml:"url"`
}

func hSitemap(w http.ResponseWriter, r *http.Request) {
	g := goc(r)
	us := urlSet{NS: "http://www.sitemaps.org/schemas/sitemap/0.9"}
	them := func(duong, sua, uu string) {
		us.URL = append(us.URL, urlSitemap{Loc: g + duong, LastMod: sua, Uu: uu})
	}
	them("/", "", "1.0")
	them("/dich-vu", "", "0.9")
	them("/bai-viet", "", "0.8")
	them("/quy-trinh", "", "0.7")
	them("/cau-hoi", "", "0.7")
	them("/gioi-thieu", "", "0.6")
	them("/ve-chung-toi", "", "0.6")
	them("/lien-he", "", "0.6")
	them("/chinh-sach", "", "0.2")
	for _, dv := range DichVuDangBan(GiaiDoan) {
		them("/dich-vu/"+dv.Ma, "", "0.8")
	}
	for _, b := range DsBaiViet() {
		them("/bai-viet/"+b.Slug, b.NgaySua(), "0.9")
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	fmt.Fprint(w, xml.Header)
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	enc.Encode(us)
	fmt.Fprintln(w)
}

// robots.txt chặn khu quản trị. Không phải để bảo mật — /qt đã có đăng
// nhập — mà để trang đăng nhập và trang đơn không lọt vào kết quả tìm
// kiếm của khách.
func hRobots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprint(w, "User-agent: *\n")
	// Danh sách nằm ở core/jsonld.go, dùng chung với chỗ quyết định trang nào
	// được khai dữ liệu có cấu trúc — hai nơi lệch nhau thì thành ra vừa chặn
	// bot vừa mời bot vào cùng một trang.
	for _, d := range khongChoBot {
		fmt.Fprintf(w, "Disallow: %s\n", d)
	}
	fmt.Fprintf(w, "\nSitemap: %s/sitemap.xml\n", goc(r))
}
