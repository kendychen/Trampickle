package core

// Dữ liệu có cấu trúc (JSON-LD). Hai việc nó làm được mà thẻ meta không
// làm được: nói cho Google biết đây là một CHỖ có thật — có tên, có số, có địa
// chỉ, để trạm còn cơ hội lọt vào khối bản đồ khi người ta tìm "sửa vợt gần
// đây"; và cho kết quả tìm kiếm hiện "Trang chủ › Bài viết › …" thay cho một
// dòng URL.
//
// KHÔNG khai FAQPage với HowTo. Google bỏ hai loại đó khỏi kết quả tìm kiếm
// của site thường từ 2023 — khai vào chỉ nặng trang, không đổi được gì.
//
// Mọi trường đều chỉ khai khi CÓ dữ liệu thật. Khai bừa một giờ mở cửa hay
// một khoảng giá không có trong cấu hình thì Google đối chiếu với trang thấy
// lệch, và đó là lý do bị bỏ khối dữ liệu chứ không phải lý do được thưởng.

import (
	"encoding/json"
	"html/template"
	"strings"
)

// khongChoBot: những nhánh không dành cho máy tìm kiếm. Một danh sách duy nhất
// cho cả robots.txt lẫn khối JSON-LD — hai chỗ lệch nhau thì hoá ra vừa chặn
// bot vừa mời bot vào cùng một trang.
var khongChoBot = []string{"/qt", "/noi-bo", "/api", "/dang-nhap", "/tra-cuu"}

func choBotDoc(duong string) bool {
	for _, p := range khongChoBot {
		if duong == p || strings.HasPrefix(duong, p+"/") {
			return false
		}
	}
	return true
}

// gocURL: lấy lại phần https://tên-miền từ Canonical. Không tự dựng lại từ
// cấu hình vì trạm chạy sau Cloudflare rồi nginx — tên miền thật chỉ biết qua
// yêu cầu, mà Canonical đã tính đúng việc đó một lần rồi (xem goc trong
// core/baiviet.go).
func (c Chung) gocURL() string {
	return strings.TrimSuffix(strings.TrimSuffix(c.Canonical, c.Duong), "/")
}

// DuLieuCoCau trả cả khối JSON cho <script type="application/ld+json">.
// Là method chứ không phải trường được handler điền: đến lúc mẫu chạy thì
// TieuDe, MoTa, ngày tháng mới là bản cuối, còn nếu dựng sẵn trong chung()
// thì mỗi handler sửa tiêu đề xong lại phải nhớ dựng lại khối này.
func (c Chung) DuLieuCoCau() template.JS {
	if c.Canonical == "" || !choBotDoc(c.Duong) {
		return ""
	}
	g := c.gocURL()
	if g == "" {
		return ""
	}
	nut := []any{c.nutTram(g)}
	if d := c.nutDuongDan(g); d != nil {
		nut = append(nut, d)
	}
	if c.LoaiOG == "article" {
		nut = append(nut, c.nutBaiViet(g))
	}
	if f := nutHoiDap(c.Duong); f != nil {
		nut = append(nut, f)
	}
	b, err := json.Marshal(map[string]any{
		"@context": "https://schema.org",
		"@graph":   nut,
	})
	if err != nil {
		return ""
	}
	// encoding/json mặc định đổi dấu nhỏ hơn thành \u003c, nên trong khối
	// này không thể mọc ra một thẻ đóng script để thoát ra ngoài — kể cả khi
	// ai đó gõ nguyên một đoạn HTML vào tiêu đề bài viết.
	return template.JS(b)
}

// nutTram — cái trạm. LocalBusiness khi đã có địa chỉ, còn chưa điền địa chỉ
// thì chỉ là Organization: LocalBusiness không có address là khối dữ liệu sai,
// Google bỏ cả khối chứ không bỏ riêng trường thiếu.
func (c Chung) nutTram(g string) map[string]any {
	l := c.LienHe
	diaChi := strings.TrimSpace(l.DiaChi)
	n := map[string]any{
		"@id":   g + "/#tram",
		"@type": "Organization",
		"name":  c.Brand.Ten,
		"url":   g + "/",
		// Ảnh mặc định chứ không phải .AnhChiaSe: ở trang bài viết .AnhChiaSe
		// là ảnh bìa bài đó, không phải ảnh của trạm.
		"image": g + AnhChiaSeMacDinh,
	}
	if s := strings.TrimSpace(c.Brand.DongMoTa); s != "" {
		n["description"] = s
	}
	if s := strings.TrimSpace(c.Brand.TenDayDu); s != "" && s != c.Brand.Ten {
		n["alternateName"] = s
	}
	if diaChi != "" {
		n["@type"] = "LocalBusiness"
		// Địa chỉ trạm là một dòng chữ gõ tay, không tách sẵn phường/quận.
		// Nhồi nguyên dòng vào streetAddress là bản khai đúng với những gì
		// đang có; bịa thêm addressLocality thì có ngày Google chấm bản đồ
		// một nơi mà trạm ở một nẻo.
		n["address"] = map[string]any{
			"@type":          "PostalAddress",
			"streetAddress":  diaChi,
			"addressCountry": "VN",
		}
		if u := l.BanDo(); u != "" {
			n["hasMap"] = u
		}
	}
	// Trạm nhận vợt gửi từ mọi tỉnh, phần lớn đơn đi đường chuyển phát chứ
	// không phải khách ghé. Có địa chỉ thật nên vẫn khai LocalBusiness — đó
	// là đường duy nhất lọt vào khối bản đồ khi người Hà Nội tìm "sửa vợt
	// gần đây" — nhưng khai thêm areaServed để chỗ này không bị đọc thành
	// một cửa hàng chỉ phục vụ mấy phường quanh nó. Khai đúng một Country,
	// không liệt kê tỉnh: danh sách tỉnh là thứ phải sửa lại mỗi lần đơn vị
	// hành chính đổi, mà sửa muộn thì thành bản khai sai.
	n["areaServed"] = map[string]any{"@type": "Country", "name": "Việt Nam"}
	if s := strings.TrimSpace(l.DienThoai); s != "" {
		n["telephone"] = s
	} else if s := strings.TrimSpace(l.Zalo); s != "" {
		n["telephone"] = s
	}
	if s := strings.TrimSpace(l.Email); s != "" {
		n["email"] = s
	}
	if u := logoURL(); u != "" {
		n["logo"] = g + u
	}
	var cungLa []string
	for _, t := range l.MangXaHoi() {
		cungLa = append(cungLa, t.URL)
	}
	if len(cungLa) > 0 {
		n["sameAs"] = cungLa
	}
	return n
}

// nutDuongDan — "Trang chủ › Bài viết › tên bài". Chỉ dựng được khi đoạn đầu
// của đường dẫn có tên trong tieuDeTrang; không có tên thì thà không khai còn
// hơn khai một nhãn máy móc lấy từ URL.
func (c Chung) nutDuongDan(g string) map[string]any {
	phan := strings.Split(strings.Trim(c.Duong, "/"), "/")
	if phan[0] == "" {
		return nil // trang chủ: không có đường dẫn nào để vẽ
	}
	tenMuc := strings.TrimSpace(tieuDeTrang[phan[0]])
	if tenMuc == "" {
		return nil
	}
	// "Trang chủ" gõ thẳng chứ không lấy từ mục nội dung: đây là nhãn đọc cho
	// máy tìm kiếm, không phải chữ hiện trên trang, nên không cần sửa được
	// trong admin.
	muc := []any{
		mocDuongDan(1, "Trang chủ", g+"/"),
		mocDuongDan(2, tenMuc, g+"/"+phan[0]),
	}
	if len(phan) > 1 && c.TieuDe != "" && c.TieuDe != tenMuc {
		muc = append(muc, mocDuongDan(3, c.TieuDe, c.Canonical))
	}
	return map[string]any{"@type": "BreadcrumbList", "itemListElement": muc}
}

func mocDuongDan(vt int, ten, u string) map[string]any {
	return map[string]any{"@type": "ListItem", "position": vt, "name": ten, "item": u}
}

func (c Chung) nutBaiViet(g string) map[string]any {
	n := map[string]any{
		"@type":            "Article",
		"@id":              c.Canonical + "#bai",
		"headline":         c.TieuDe,
		"mainEntityOfPage": map[string]any{"@type": "WebPage", "@id": c.Canonical},
		"inLanguage":       "vi-VN",
		// Trạm tự viết và tự đăng, nên tác giả và nhà xuất bản là cùng một nút
		// — trỏ bằng @id thay vì chép lại cả cụm tên/địa chỉ hai lần.
		"author":    map[string]any{"@id": g + "/#tram"},
		"publisher": map[string]any{"@id": g + "/#tram"},
	}
	if c.MoTa != "" {
		n["description"] = c.MoTa
	}
	if c.AnhChiaSe != "" {
		n["image"] = c.AnhChiaSe
	}
	if c.NgayDang != "" {
		n["datePublished"] = c.NgayDang
		// Bài chưa sửa lần nào thì ngày sửa là ngày đăng. Thiếu dateModified
		// không sao, nhưng có thì Google biết bài còn được trông, mà giá trị
		// đúng đang nằm ngay đây.
		ngaySua := c.NgayCapNhat
		if ngaySua == "" {
			ngaySua = c.NgayDang
		}
		n["dateModified"] = ngaySua
	}
	return n
}

// nutHoiDap — khối FAQPage của trang /cau-hoi, thứ Google nhặt vào ô "Mọi
// người cũng hỏi".
//
// Đọc thẳng CauHoiHien() chứ không nhận qua Chung: khối JSON-LD dựng lúc mẫu
// chạy, mà mọi trang đều đi qua chung() còn chỉ đúng một trang cần danh sách
// này — nhét vào Chung là bắt mọi trang mang theo thứ chúng không dùng.
//
// Chưa có câu nào thì KHÔNG phát khối rỗng: FAQPage không có mainEntity là
// dữ liệu sai, Google phạt cả trang chứ không bỏ riêng khối.
func nutHoiDap(duong string) map[string]any {
	if duong != "/cau-hoi" {
		return nil
	}
	ds := CauHoiHien()
	if len(ds) == 0 {
		return nil
	}
	muc := make([]any, 0, len(ds))
	for _, c := range ds {
		dap := c.DapTron()
		if dap == "" {
			continue
		}
		muc = append(muc, map[string]any{
			"@type": "Question",
			"name":  c.Hoi,
			"acceptedAnswer": map[string]any{
				"@type": "Answer",
				"text":  dap,
			},
		})
	}
	if len(muc) == 0 {
		return nil
	}
	return map[string]any{"@type": "FAQPage", "mainEntity": muc}
}
