// HTTP server. Hai chế độ, và chúng KHÁC NHAU về route chứ không chỉ khác
// nhau về hiển thị:
//
//	-public : trang khách + quản trị + /noi-bo, bind 0.0.0.0.
//	mặc định: thêm /api/index, bind 127.0.0.1.
//
// Route agent từng KHÔNG được đăng ký ở chế độ public, vì một endpoint hỏi
// agent không mật khẩu nằm trên internet là cách nhanh nhất để cháy quota
// Gemini. Nay nó có mật khẩu: canLaChu chặn cả khách lẫn thợ. Đổi lại,
// GEMINI_API_KEY phải nằm trên máy chủ — chấp nhận, vì máy chủ này chỉ
// Kendy vào.
//
// Riêng /api/index vẫn chỉ ở máy nhà: nó gọi nhúng vector cho toàn bộ tài
// liệu, và ở quy mô 141 KB thì nhồi thẳng vào prompt đã đủ, index không
// thêm được gì để đáng đánh đổi.
//
// Chống CSRF: token trong mọi biểu mẫu, kiểm ở lớp bọc quanh mux. Cookie
// SameSite=Lax vẫn giữ làm lớp thứ hai. Xem core/csrf.go và core/middleware.go.
package core

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

//go:embed ui
var uiFS embed.FS

var tpl *template.Template

// CongKhai để template biết đang chạy chế độ nào mà ẩn/hiện link nội bộ.
var CongKhai bool

func InitTemplates() error {
	fm := template.FuncMap{
		"tien":        dinhDangTien,
		"gia":         giaHienThi,
		"giaso":       giaSo,
		"oso":         oSo,
		"sl":          soLuongDep,
		"slo":         soLuongO,
		"ptram":       phanTram,
		"batdau":      strings.HasPrefix,
		"nhom":        tenNhom,
		"tenvt":       TenVatTu,
		"tennhomtien": TenNhomTien,
		"json":        toJSON,
		"cong":        func(a, b int) int { return a + b },
		// lap n: n dòng trống trong form. range cần một slice, và {{range 4}}
		// thì template không hiểu.
		"lap":      func(n int) []int { return make([]int, n) },
		"tt":       TrangThaiCua,
		"ngaydep":  ngayDep,
		"canDep":   canDep,
		"thangDep": thangDep,
		"cat":      catBot,
		"cocach":   func(s string) bool { return strings.TrimSpace(s) != "" },
		"xemTep":   xemTep,
		// Chữ sửa được ở /qt/noi-dung. nd = chữ trơn (dùng được cả trong
		// thuộc tính HTML), ndd = một dòng có **đậm** và [link](/x),
		// ndm = ô nhiều dòng dựng qua markdown gọn.
		"nd":  ND,
		"ndd": ndDong,
		"ndm": ndKhoi,
		"ndx": ndx,
		// Logo/biểu tượng tab: rỗng nghĩa là chưa tải bản riêng, template
		// dựng SVG nhúng sẵn. Để hàm chứ không nhét vào Chung, vì mọi trang
		// đều cần mà không phải handler nào cũng đi qua chung().
		"logoURL":  logoURL,
		"iconURL":  iconURL,
		"iconMIME": iconMIME,
		// Màu nhấn của giao diện đang bật, cho <meta name="theme-color"> —
		// thanh trạng thái của điện thoại lấy màu từ đấy.
		"mauNhan": mauNhanTheme,
		// Số điện thoại bỏ dấu cách, để nhét vào href="tel:".
		"soGoi": soGoi,
	}
	t, err := template.New("").Funcs(fm).ParseFS(uiFS, "ui/*.html")
	if err != nil {
		return err
	}
	tpl = t
	return nil
}

// hFavicon trả thẳng tệp trong ui đã nhúng. Bản favicon có sẵn nền than —
// tab trình duyệt sáng hay tối là tuỳ máy người xem, để nền trong suốt thì
// bộ đồ nghề màu steel biến mất trên tab nền sáng.
func hFavicon(w http.ResponseWriter, r *http.Request) {
	b, err := uiFS.ReadFile("ui/favicon.svg")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Write(b)
}

// hAnhNen trả ảnh trang trí của giao diện: ảnh nền tấm đầu trang, dải ảnh
// giữa trang. Khác ảnh ở /anh-trang-chu/ — mấy tấm kia là nội dung Kendy tự
// up, ghi trong data/anh-trang-chu.yaml, đổi lúc nào cũng được. Mấy tấm này
// đi kèm bộ giao diện: chọn bộ nào thì CSS gọi đúng ảnh của bộ đó, nên nhúng
// thẳng vào binary cho khỏi lệch pha giữa CSS và thư mục data.
//
// Đọc từ FS nhúng nên không có đường thoát ra ngoài thư mục, nhưng vẫn lọc
// tên qua tenAnhSach để người đọc sau không phải dừng lại tự chứng minh điều
// đó. Chỉ nhận .jpg vì cả bộ ảnh là JPEG.
func hAnhNen(w http.ResponseWriter, r *http.Request) {
	ten := r.PathValue("ten")
	if !tenAnhSach(ten) || !strings.HasSuffix(ten, ".jpg") {
		http.NotFound(w, r)
		return
	}
	b, err := uiFS.ReadFile("ui/anh/" + ten)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/jpeg")
	// Ảnh nằm trong binary, muốn đổi là phải build lại — cache một tuần thoải mái.
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Write(b)
}

// dinhDangTien: 150000 -> "150.000đ". Dấu chấm ngăn nghìn, kiểu Việt Nam.
func dinhDangTien(n int) string {
	s := fmt.Sprintf("%d", n)
	am := ""
	if strings.HasPrefix(s, "-") {
		am, s = "-", s[1:]
	}
	var b strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(c)
	}
	return am + b.String() + "đ"
}

// soLuongDep: 1.5 -> "1,5", 2 -> "2". Kho đo bằng mét và tuýp nên số lẻ là
// bình thường, nhưng "2" đọc dễ hơn "2.0000".
func soLuongDep(f float64) string {
	s := strconv.FormatFloat(f, 'f', -1, 64)
	return strings.ReplaceAll(s, ".", ",")
}

// soLuongO: giá trị cho ô nhập. Số âm nghĩa là "chưa đếm/chưa điền" — trả về
// rỗng để ô trống, vì 0 trong phiếu kiểm kê có nghĩa là "đếm rồi, hết sạch".
func soLuongO(f float64) string {
	if f < 0 {
		return ""
	}
	return strconv.FormatFloat(f, 'f', -1, 64)
}

// giaHienThi đọc giá từ bảng giá. KHÔNG tính toán gì — Go không sinh ra
// con số tiền, chỉ đọc con số đã có trong YAML.
func giaHienThi(d DichVu) string {
	p := d.GiaTheoGiaiDoan(GiaiDoan)
	if p == nil {
		if d.BaoGiaRieng {
			// Không nói "xem vợt": mục Thay đế giày nhận GIÀY, câu kia đọc
			// vào là sai việc. Một câu dùng chung cho mọi mã bao_gia_rieng.
			return "Gửi ảnh, em báo giá"
		}
		return "—"
	}
	if *p == 0 {
		return "Tặng kèm"
	}
	return dinhDangTien(*p)
}

// oSo đổ số vào ô input: 0 thì để ô trống thay vì bắt người ta xoá số 0.
func oSo(v any) string {
	switch n := v.(type) {
	case int:
		if n == 0 {
			return ""
		}
		return strconv.Itoa(n)
	case float64:
		if n == 0 {
			return ""
		}
		return strconv.FormatFloat(n, 'f', -1, 64)
	}
	return ""
}

// phanTram cho biểu đồ cột: bao nhiêu phần trăm của cột cao nhất.
// Đây là vẽ hình, không phải định giá.
func phanTram(v, caoNhat int) int {
	if caoNhat <= 0 || v <= 0 {
		return 0
	}
	p := v * 100 / caoNhat
	if p < 1 {
		p = 1
	}
	return p
}

// giaSo trả về con số thô trong bảng giá để form tick chọn cộng lại cho
// người nhập nhìn. Vẫn là đọc, không phải tính.
func giaSo(d DichVu) int {
	p := d.GiaTheoGiaiDoan(GiaiDoan)
	if p == nil {
		return 0
	}
	return *p
}

func tenNhom(n string) string {
	switch n {
	case "A":
		return "Sửa cơ bản"
	case "B":
		return "Sửa kết cấu"
	case "C":
		return "Gia công ngoài"
	}
	return n
}

func toJSON(v any) template.JS {
	b, _ := json.Marshal(v)
	return template.JS(b)
}

// ngayDep: "2026-09-03" -> "03/09/2026". Chuỗi rỗng trả về "—".
func ngayDep(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	if len(s) >= 10 {
		return s[8:10] + "/" + s[5:7] + "/" + s[0:4]
	}
	return s
}

func thangDep(s string) string {
	if len(s) == 7 {
		return "Tháng " + strings.TrimPrefix(s[5:7], "0") + "/" + s[0:4]
	}
	return s
}

func canDep(g float64) string {
	if g <= 0 {
		return "—"
	}
	return strconv.FormatFloat(g, 'f', 1, 64) + "g"
}

// --- Chống spam -----------------------------------------------------
// Rate limit thô theo IP cho form khách. Không phải tường lửa, chỉ để một
// con bot rảnh rỗi không ghi đầy ổ đĩa.

type gioiHan struct {
	mu  sync.Mutex
	lan map[string][]time.Time
}

var rl = &gioiHan{lan: map[string][]time.Time{}}

func (g *gioiHan) choPhep(ip string, soLan int, trong time.Duration) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	now := time.Now()
	con := []time.Time{}
	for _, t := range g.lan[ip] {
		if now.Sub(t) < trong {
			con = append(con, t)
		}
	}
	if len(con) >= soLan {
		g.lan[ip] = con
		return false
	}
	g.lan[ip] = append(con, now)
	return true
}

// TinProxy và ipCua chuyển sang core/realip.go — xác định IP thật là nền của
// mọi giới hạn tần suất nên đáng có file riêng kèm test.

// --- Khung dữ liệu chung cho mọi trang -------------------------------

type Chung struct {
	Brand      ThuongHieu
	LienHe     LienHe
	Trang      string // để nav biết đang ở đâu
	TieuDe     string
	MoTa       string // <meta name=description>
	Canonical  string // URL chuẩn, để Google không coi IP và tên miền là hai trang
	NguoiDung  NguoiDung
	DaDangNhap bool
	CongKhai   bool
	GiaiDoan   int
	Theme      string // giao diện đang bật, chọn ở /qt/giao-dien
	LopThan    string // lớp cho <body>: theme + nền tối nếu trang đó tối
	// Nonce cho <script> nội tuyến. CSP chặn mọi script không mang nonce
	// đúng của lượt tải trang đó — xem core/middleware.go.
	Nonce string
	// CSRF: token cho <input type="hidden" name="_csrf">. Xem core/csrf.go.
	CSRF string
}

// Tiêu đề tab gom về một chỗ, thay vì rải mỗi handler một chuỗi rồi lệch
// nhau lúc sửa.
var tieuDeTrang = map[string]string{
	"chu":           "Sửa vợt Pickleball",
	"vot":           "Danh mục vợt",
	"cai-dat":       "Khóa API & model",
	"nhat-ky":       "Nhật ký",
	"2fa":           "Xác thực hai bước",
	"gioi-thieu":    "Giới thiệu",
	"ve-chung-toi":  "Về chúng tôi",
	"app":           "App",
	"dich-vu":       "Dịch vụ",
	"dich-vu-qt":    "Bảng dịch vụ",
	"lien-he-qt":    "Liên hệ",
	"bai-viet":      "Bài viết",
	"quy-trinh":     "Quy trình nhận và trả vợt",
	"lien-he":       "Liên hệ",
	"chinh-sach":    "Chính sách bảo mật",
	"tra-cuu":       "Tra cứu đơn sửa",
	"dang-nhap":     "Đăng nhập",
	"don":           "Đơn sửa",
	"don-moi":       "Tạo đơn mới",
	"yeu-cau":       "Yêu cầu từ web",
	"thong-ke":      "Thống kê",
	"nguoi-dung":    "Tài khoản",
	"mat-khau":      "Đổi mật khẩu",
	"giao-dien":     "Giao diện",
	"anh-trang-chu": "Ảnh trang chủ",
	"bai-viet-qt":   "Bài viết",
	"giao-trinh":    "Giáo trình",
	"noi-dung":      "Chữ trên trang",
	"agent":         "Trợ lý kỹ thuật",
	"kho":           "Kho vật tư",
	"kho-phieu":     "Phiếu kho",
	"kho-vat-tu":    "Danh mục vật tư",
	"tien":          "Sổ thu chi",
	"tien-dinh-ky":  "Khoản định kỳ",
	"tien-sao-ke":   "Dán sao kê",
}

// Mô tả cho ô snippet của Google. Trang nào không có ở đây thì để trống —
// Google tự trích một đoạn trong bài, còn hơn là nhét đại một câu chung
// chung giống hệt nhau ở mọi trang.
var moTaTrang = map[string]string{
	"chu":          "Trạm sửa vợt Pickleball: dán viền, vá mặt, hàn cán, sửa tách lớp, thay đế giày. Gửi ảnh video để nghe nhận xét trước, gửi vợt sau — khám xong báo giá ngay, chốt rồi thợ mới làm.",
	"dich-vu":      "Danh sách việc trạm đang nhận: dán viền, thay viền, vá mặt thủng, hàn cán carbon, sửa tách lớp, quấn grip, thay đế giày. Mỗi việc có quy trình và thời gian riêng.",
	"quy-trinh":    "Bốn bước từ lúc nhắn tin đến lúc nhận vợt: gửi ảnh nghe nhận xét, gửi vợt và khám để ra giá, chốt giá rồi mới làm, nghiệm thu và bàn giao. Kèm luật phí vận chuyển.",
	"gioi-thieu":   "Trạm nhỏ, làm ít việc nhưng mỗi việc có quy trình viết sẵn. Khám xong báo giá, chốt rồi mới mở keo.",
	"ve-chung-toi": "Địa chỉ trạm, giờ mở cửa và ảnh chỗ làm việc — ghé xem trực tiếp hoặc gửi chuyển phát đều được.",
	"lien-he":      "Gửi ảnh hoặc video chỗ hỏng, trạm xem rồi nhận xét trong ngày là làm được hay không và mất mấy ngày. Vợt Pickleball và giày đều nhận.",
	"bai-viet":     "Bài viết về sửa vợt Pickleball: các kiểu hỏng thường gặp, cách xử lý, và những thứ nên biết trước khi mang vợt đi sửa.",
	"tra-cuu":      "Tra tình trạng đơn sửa bằng mã đơn trên phiếu, hoặc mã yêu cầu sau khi gửi ảnh, cùng 4 số cuối điện thoại.",
	"app":          "Các việc trạm nhận làm với vợt Pickleball, gọn trong một màn hình. Cài lên màn hình chính rồi mở như một app.",
}

func chung(r *http.Request, trang string) Chung {
	nd, ok := NguoiDangNhap(r)
	theme := ThemeHienTai()
	return Chung{
		Brand:      CFG.ThuongHieu,
		LienHe:     LienHeHienTai(),
		Trang:      trang,
		TieuDe:     tieuDeTrang[trang],
		MoTa:       moTaTrang[trang],
		Canonical:  goc(r) + r.URL.Path,
		NguoiDung:  nd,
		DaDangNhap: ok,
		CongKhai:   CongKhai,
		GiaiDoan:   GiaiDoan,
		Theme:      theme,
		LopThan:    lopThan(trang, theme),
		Nonce:      nonceCua(r),
		CSRF:       tokenCSRF(r),
	}
}

// --- Trang khách ----------------------------------------------------

type dlTrangChu struct {
	Chung
	DichVu   []DichVu
	KhongBan []KhongBan
	Nguong   Nguong
	// Anh: băng ảnh chạy ở hero. Rỗng thì hero quay về bản vẽ cây vợt.
	Anh   []AnhHero
	DaGui bool
	MaDon string
	// MaYeuCau: mã của yêu cầu vừa gửi. Không hiện ra thì khách gửi ảnh xong
	// không cầm được gì trên tay, và ô tra cứu ở trang kia thành vô dụng với
	// họ — chưa có đơn thì chưa có mã đơn nào để gõ vào.
	MaYeuCau string
	// MailToi: địa chỉ vừa gửi thư xác nhận tới. Rỗng khi khách không để
	// email, hoặc khi máy chủ chưa có khóa gửi mail — lúc đó không được hứa
	// trên trang là "đã gửi mail".
	MailToi string
	Loi     string
}

func layTrangChu(r *http.Request) dlTrangChu {
	return dlTrangChu{
		Chung:    chung(r, "chu"),
		DichVu:   DichVuDangBan(GiaiDoan),
		KhongBan: GIA.KhongBan,
		Nguong:   NguongHienTai(),
		Anh:      DsAnhNoi(""),
	}
}

func hTrangChu(w http.ResponseWriter, r *http.Request) {
	render(w, "trangchu.html", layTrangChu(r))
}

func hGioiThieu(w http.ResponseWriter, r *http.Request) {
	render(w, "gioithieu.html", struct {
		Chung
		KhongBan []KhongBan
		Nguong   Nguong
	}{chung(r, "gioi-thieu"), GIA.KhongBan, NguongHienTai()})
}

// hVeChungToi — trang "trạm có thật, ở đây, trông thế này".
//
// Tách khỏi /gioi-thieu chứ không nhập làm một: trang giới thiệu trả lời
// "trạm làm việc ra sao" bằng chữ, trang này trả lời "trạm có thật không"
// bằng địa chỉ và ảnh. Khách đang phân vân gửi vợt đi hay không thì hỏi câu
// thứ hai trước, và câu ấy phải trả lời được trong ba giây.
func hVeChungToi(w http.ResponseWriter, r *http.Request) {
	c := chung(r, "ve-chung-toi")
	bando := ""
	if d := strings.TrimSpace(c.LienHe.DiaChi); d != "" {
		bando = "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(d)
	}
	render(w, "vechungtoi.html", struct {
		Chung
		Anh   []AnhHero
		BanDo string
	}{c, DsAnhNoi(NoiTram), bando})
}

func hQuyTrinh(w http.ResponseWriter, r *http.Request) {
	render(w, mauTheoVo(r, "quytrinh.html", "app-quytrinh.html"), struct {
		Chung
		Nguong Nguong
	}{chung(r, "quy-trinh"), NguongHienTai()})
}

func hDichVuList(w http.ResponseWriter, r *http.Request) {
	render(w, "dichvu.html", struct {
		Chung
		DichVu   []DichVu
		KhongBan []KhongBan
	}{chung(r, "dich-vu"), DichVuDangBan(GiaiDoan), GIA.KhongBan})
}

func hDichVuMot(w http.ResponseWriter, r *http.Request) {
	ma := strings.ToUpper(r.PathValue("ma"))
	var dv *DichVu
	// Bản sao, không phải con trỏ vào lát cắt gốc: trang /qt/dich-vu thay cả
	// danh sách khi lưu, mà template bên dưới còn đang đọc dv.
	ds := DichVuTatCa()
	for i := range ds {
		if ds[i].Ma == ma && ds[i].DaMo(GiaiDoan) {
			dv = &ds[i]
			break
		}
	}
	if dv == nil {
		http.NotFound(w, r)
		return
	}
	c := chung(r, "dich-vu")
	c.TieuDe = dv.Ten
	render(w, mauTheoVo(r, "dichvu-mot.html", "app-dichvu.html"), struct {
		Chung
		DV     DichVu
		Khac   []DichVu
		Nguong Nguong
	}{c, *dv, DichVuDangBan(GiaiDoan), NguongHienTai()})
}

func hLienHe(w http.ResponseWriter, r *http.Request) {
	render(w, "lienhe.html", struct {
		Chung
		Nguong Nguong
	}{chung(r, "lien-he"), NguongHienTai()})
}

func hChinhSach(w http.ResponseWriter, r *http.Request) {
	render(w, "chinhsach.html", chung(r, "chinh-sach"))
}

// --- Yêu cầu từ khách ------------------------------------------------

type yeuCauKhach struct {
	Ma     string `json:"ma"`
	Ngay   string `json:"ngay"`
	Ten    string `json:"ten"`
	LienHe string `json:"lien_he"`
	// Email không bắt buộc. Bắt buộc thì mất khách — nhiều bác chơi pickleball
	// chỉ dùng Zalo. Có thì gửi xác nhận về, không có thì thôi.
	Email   string   `json:"email"`
	VotHang string   `json:"vot_hang"`
	MoTa    string   `json:"mo_ta"`
	IP      string   `json:"ip"`
	Tep     []string `json:"tep"`
	DaXuLy  bool     `json:"da_xu_ly"`
	MaDon   string   `json:"ma_don"`
}

// Tệp khách gửi kèm. Ba con số này quyết định ổ đĩa VPS đầy nhanh hay
// chậm, nên để cạnh nhau cho dễ chỉnh:
//
//	5 tệp  — đủ chụp bốn góc cây vợt cộng một clip.
//	12MB   — ảnh điện thoại đời mới nặng nhất khoảng 8MB.
//	40MB   — khoảng 10-20 giây quay 1080p. Dài hơn thì bảo khách gửi Zalo.
const (
	tepToiDaMoiYeuCau    = 5
	anhYeuCauToiDaByte   = 12 << 20
	videoYeuCauToiDaByte = 40 << 20
	tongYeuCauToiDaByte  = 100 << 20
)

func thuMucTepYeuCau(ma string) string { return filepath.Join(P("data/tep-yeu-cau"), ma) }

// duoiTep đọc mấy byte đầu chứ không tin phần mở rộng trong tên file —
// tên do máy khách đặt, chỉ nội dung mới nói thật. Trả về đuôi chuẩn hoá
// và cờ "đây là video" để áp đúng ngưỡng dung lượng.
func duoiTep(dau []byte) (string, bool) {
	switch {
	case len(dau) > 3 && dau[0] == 0xFF && dau[1] == 0xD8 && dau[2] == 0xFF:
		return ".jpg", false
	case len(dau) > 8 && string(dau[1:4]) == "PNG":
		return ".png", false
	case len(dau) > 12 && string(dau[0:4]) == "RIFF" && string(dau[8:12]) == "WEBP":
		return ".webp", false
	case len(dau) > 3 && dau[0] == 0x1A && dau[1] == 0x45 && dau[2] == 0xDF && dau[3] == 0xA3:
		return ".webm", true
	case len(dau) > 12 && string(dau[4:8]) == "ftyp":
		// Cùng một vỏ ISO-BMFF: iPhone chụp ảnh ra HEIC, quay phim ra MOV,
		// Android quay ra MP4. Phân biệt bằng brand ở byte 8-12.
		switch string(dau[8:12]) {
		case "heic", "heix", "hevc", "hevx", "mif1", "msf1":
			return ".heic", false
		case "qt  ":
			return ".mov", true
		}
		return ".mp4", true
	}
	return "", false
}

// xemTep nói cho template biết vẽ tệp này bằng thẻ gì. HEIC của iPhone là
// ảnh thật nhưng Chrome trên Windows không mở được — vẽ <img> thì thợ chỉ
// thấy ô vỡ, nên trả về "tai" để đưa link tải về.
func xemTep(ten string) string {
	switch kieu := kieuTep[strings.ToLower(filepath.Ext(ten))]; {
	case strings.HasPrefix(kieu, "video/"):
		return "video"
	case kieu == "image/heic" || kieu == "":
		return "tai"
	default:
		return "anh"
	}
}

var kieuTep = map[string]string{
	".jpg":  "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".heic": "image/heic",
	".mp4":  "video/mp4",
	".mov":  "video/quicktime",
	".webm": "video/webm",
}

// luuTepYeuCau bỏ qua tệp hỏng thay vì trả lỗi cả biểu mẫu: khách đã gõ
// xong mô tả rồi, đá cả form về chỉ vì một tấm ảnh sai định dạng là mất
// luôn cái tin nhắn đáng giá nhất.
func luuTepYeuCau(ma string, files []*multipart.FileHeader) []string {
	var ten []string
	for _, fh := range files {
		if len(ten) >= tepToiDaMoiYeuCau {
			break
		}
		f, err := fh.Open()
		if err != nil {
			continue
		}
		dau := make([]byte, 16)
		n, _ := io.ReadFull(f, dau)
		duoi, video := duoiTep(dau[:n])
		gioiHan := int64(anhYeuCauToiDaByte)
		if video {
			gioiHan = videoYeuCauToiDaByte
		}
		if duoi == "" || fh.Size > gioiHan {
			f.Close()
			continue
		}
		if err := os.MkdirAll(thuMucTepYeuCau(ma), 0o755); err != nil {
			f.Close()
			break
		}
		t := fmt.Sprintf("%02d-%s%s", len(ten)+1, maNgauNhien(3), duoi)
		out, err := os.Create(filepath.Join(thuMucTepYeuCau(ma), t))
		if err != nil {
			f.Close()
			continue
		}
		out.Write(dau[:n])
		io.Copy(out, io.LimitReader(f, gioiHan))
		out.Close()
		f.Close()
		ten = append(ten, t)
	}
	return ten
}

func hGuiYeuCau(w http.ResponseWriter, r *http.Request) {
	mau := mauTheoVo(r, "trangchu.html", "app-guianh.html")
	loi := func(msg string) {
		d := layGuiAnh(r)
		d.Loi = msg
		render(w, mau, d)
	}
	if !rl.choPhep(ipCua(r), 5, time.Hour) {
		loi("Gửi hơi nhiều lần rồi ạ. Anh/chị nhắn trực tiếp giúp em.")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, tongYeuCauToiDaByte)
	// 8MB giữ trong RAM, phần còn lại multipart tự ghi ra tệp tạm rồi
	// dọn khi request đóng — không cần tự quản.
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		loi("Ảnh hoặc video nặng quá ạ. Anh/chị gửi ít tệp hơn, hoặc quay ngắn lại giúp em.")
		return
	}
	defer r.MultipartForm.RemoveAll()

	// Biểu mẫu có tệp: lớp bọc không đọc được thân yêu cầu nên không kiểm
	// CSRF hộ được. Xem core/csrf.go.
	if !KiemCSRFMultipart(w, r) {
		return
	}

	yc := yeuCauKhach{
		Ma:      time.Now().Format("2006-01-02-150405") + "-" + maNgauNhien(3),
		Ngay:    time.Now().Format("2006-01-02 15:04:05"),
		Ten:     catBot(r.FormValue("ten"), 100),
		LienHe:  catBot(r.FormValue("lien_he"), 100),
		Email:   catBot(strings.TrimSpace(r.FormValue("email")), 254),
		VotHang: catBot(r.FormValue("vot_hang"), 100),
		MoTa:    catBot(r.FormValue("mo_ta"), 2000),
		IP:      ipCua(r),
	}
	if strings.TrimSpace(yc.LienHe) == "" || strings.TrimSpace(yc.MoTa) == "" {
		loi("Cần số điện thoại/Zalo và mô tả tình trạng vợt ạ.")
		return
	}
	// Gõ sai email thì nói ngay, đừng nuốt. Người ta để email là vì muốn nhận
	// mã — im lặng bỏ qua rồi họ ngồi đợi một lá thư không bao giờ tới.
	if yc.Email != "" && !HopLeEmail(yc.Email) {
		loi("Email hình như gõ thiếu ạ. Anh/chị xem lại, hoặc bỏ trống cũng được.")
		return
	}

	yc.Tep = luuTepYeuCau(yc.Ma, r.MultipartForm.File["tep"])

	dir := P(CFG.DuongDan.TinNhan)
	os.MkdirAll(dir, 0o755)
	if b, err := json.MarshalIndent(yc, "", "  "); err == nil {
		os.WriteFile(filepath.Join(dir, yc.Ma+".json"), b, 0o644)
	}

	// Gửi sau khi đã ghi xuống đĩa: yêu cầu nằm an toàn rồi mới lo báo tin.
	// Hàm này tự chạy nền, không giữ chân trang trả về.
	MailXacNhanYeuCau(yc)

	d := layGuiAnh(r)
	d.DaGui = true
	d.MaYeuCau = yc.Ma
	if yc.Email != "" && MailBat() {
		d.MailToi = yc.Email
	}
	render(w, mau, d)
}

func catBot(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > n {
		return string(r[:n])
	}
	return s
}

// --- Tra cứu đơn cho khách -------------------------------------------

// traCuuYeuCau tìm theo mã của yêu cầu gửi từ web, cùng luật bảo vệ như đơn:
// phải khớp thêm 4 số cuối điện thoại. Có hàm này vì khách gửi ảnh xong chỉ
// cầm mã yêu cầu, chưa có mã đơn — mà đó lại đúng là lúc họ sốt ruột nhất.
//
// Yêu cầu đã được dựng thành đơn thì trả luôn đơn: khách không phải biết
// trong trạm có hai loại mã khác nhau.
func traCuuYeuCau(ma, bonSoCuoi string) (*Don, *yeuCauKhach, bool) {
	ma = strings.TrimSpace(ma)
	var yc yeuCauKhach
	found := false
	for _, y := range docYeuCau(0) {
		if strings.EqualFold(y.Ma, ma) {
			yc, found = y, true
			break
		}
	}
	if !found {
		return nil, nil, false
	}
	so, nhap := chiSo(yc.LienHe), chiSo(bonSoCuoi)
	if len(so) < 4 || len(nhap) < 4 || !strings.HasSuffix(so, nhap[len(nhap)-4:]) {
		return nil, nil, false
	}
	if yc.MaDon != "" {
		if d, co := LayDon(yc.MaDon); co {
			return d, &yc, true
		}
	}
	return nil, &yc, true
}

// hAnhDonChoKhach phục vụ ảnh trước/sau của một đơn cho khách, không cần
// đăng nhập. Cửa duy nhất là token 8 ký tự trong URL. Cũng như hQtXemAnh:
// chỉ trả tên file CÓ TRONG danh sách của đơn, không bao giờ ghép đường dẫn
// thẳng từ tham số URL.
func hAnhDonChoKhach(w http.ResponseWriter, r *http.Request) {
	don, ok := LayDonTheoToken(r.PathValue("token"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	ten := r.PathValue("ten")
	found := false
	for _, a := range don.Anh {
		if a == ten {
			found = true
			break
		}
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Robots-Tag", "noindex")
	http.ServeFile(w, r, filepath.Join(thuMucAnh(don.Ma), ten))
}

func hTraCuu(w http.ResponseWriter, r *http.Request) {
	type dl struct {
		Chung
		Ma string
		// DienThoai: trả lại đúng số khách vừa gõ. Bản app đọc nó để nhớ đơn
		// vào máy; bản web thì đơn giản là không bắt gõ lại khi tra hụt.
		DienThoai string
		Don       *Don
		YeuCau    *yeuCauKhach
		Loi       string
	}
	mau := mauTheoVo(r, "tracuu.html", "app-tracuu.html")
	d := dl{Chung: chung(r, "tra-cuu")}
	if r.Method == http.MethodPost {
		// Tra cứu cũng phải chặn: mã đơn theo số thứ tự, không giới hạn thì
		// dò được. 12 lần/giờ đủ cho người thật, không đủ cho script.
		if !rl.choPhep("tracuu:"+ipCua(r), 12, time.Hour) {
			d.Loi = "Tra cứu hơi nhiều lần rồi. Anh/chị thử lại sau ạ."
			render(w, mau, d)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
		r.ParseForm()
		d.Ma = catBot(r.FormValue("ma"), 40)
		d.DienThoai = catBot(r.FormValue("dien_thoai"), 12)
		if don, ok := TraCuuChoKhach(d.Ma, r.FormValue("dien_thoai")); ok {
			d.Don = don
		} else if don, yc, ok := traCuuYeuCau(d.Ma, r.FormValue("dien_thoai")); ok {
			d.Don, d.YeuCau = don, yc
		} else {
			d.Loi = "Không tìm thấy đơn hay yêu cầu nào khớp mã và số điện thoại này."
		}
	}
	render(w, mau, d)
}

// --- Đăng nhập -------------------------------------------------------

func hDangNhap(w http.ResponseWriter, r *http.Request) {
	type dl struct {
		Chung
		Ten string
		Loi string
		// Hoi2FA: mật khẩu đúng rồi, giờ hỏi thêm mã 6 số.
		Hoi2FA bool
	}
	d := dl{Chung: chung(r, "dang-nhap")}
	if d.DaDangNhap {
		http.Redirect(w, r, "/qt", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
		r.ParseForm()
		d.Ten = catBot(r.FormValue("ten"), 50)
		ip := ipCua(r)

		// Chặn TRƯỚC khi chạy bcrypt: mỗi lần so mật khẩu là một lần cost 12,
		// để kẻ dò kích hoạt tuỳ ý là biếu không CPU của máy chủ.
		if ok, con := choPhepThu(ip, d.Ten); !ok {
			GhiNhatKy(MucNhatKy{Ai: d.Ten, IP: ip, Viec: "dang-nhap", KetQua: "bi-khoa"})
			d.Loi = fmt.Sprintf("Sai quá nhiều lần. Thử lại sau %d phút.",
				int(con.Minutes())+1)
			w.WriteHeader(http.StatusTooManyRequests)
			render(w, "dangnhap.html", d)
			return
		}

		nd, err := KiemTraDangNhap(d.Ten, r.FormValue("mat_khau"))
		if err != nil {
			ghiThuSai(ip, d.Ten)
			GhiNhatKy(MucNhatKy{Ai: d.Ten, IP: ip, Viec: "dang-nhap", KetQua: "sai-mat-khau"})
			d.Loi = err.Error()
			w.WriteHeader(http.StatusUnauthorized)
			render(w, "dangnhap.html", d)
			return
		}

		if nd.TotpBat {
			maNhap := strings.TrimSpace(r.FormValue("ma_totp"))
			if maNhap == "" {
				// Mật khẩu đúng nhưng chưa có mã: hiện ô nhập mã, CHƯA cấp
				// phiên. Không cấp phiên tạm nào cả — mật khẩu gõ lại cùng mã
				// đơn giản hơn, và không có phiên nửa vời nào để rò.
				d.Hoi2FA = true
				render(w, "dangnhap.html", d)
				return
			}
			buoc, ok := kiemTOTP(nd.TotpBiMat, maNhap, time.Now())
			if ok && daDungBuoc(nd.Ten, buoc) {
				ok = false // chống phát lại: mã này vừa dùng xong
			}
			if !ok && !DungMaDuPhong(nd.Ten, maNhap) {
				ghiThuSai(ip, d.Ten)
				GhiNhatKy(MucNhatKy{Ai: nd.Ten, IP: ip, Viec: "dang-nhap", KetQua: "sai-2fa"})
				d.Loi = "Mã xác thực không đúng."
				d.Hoi2FA = true
				w.WriteHeader(http.StatusUnauthorized)
				render(w, "dangnhap.html", d)
				return
			}
			if ok {
				nhoBuoc(nd.Ten, buoc)
			}
		}

		// Hỏi "IP này quen chưa" TRƯỚC khi ghiThuDung, vì ghiThuDung chính là
		// thứ làm cho nó thành quen.
		la := !ipDaQuen(nd.Ten, ip)
		ghiThuDung(ip, nd.Ten)
		GhiNhatKy(MucNhatKy{Ai: nd.Ten, IP: ip, Viec: "dang-nhap", KetQua: "ok"})
		if la {
			canhBaoDangNhapLa(nd.Ten, ip)
		}
		taoPhien(w, nd.Ten, ip)
		http.Redirect(w, r, "/qt", http.StatusSeeOther)
		return
	}
	render(w, "dangnhap.html", d)
}

func hDangXuat(w http.ResponseWriter, r *http.Request) {
	if nd, ok := NguoiDangNhap(r); ok {
		GhiNhatKy(MucNhatKy{Ai: nd.Ten, IP: ipCua(r), Viec: "dang-xuat", KetQua: "ok"})
	}
	xoaPhien(w, r)
	http.Redirect(w, r, "/dang-nhap", http.StatusSeeOther)
}

// canDangNhap chặn mọi route quản trị. canLaChu chặn thêm những trang
// chỉ chủ được xem — thợ không cần thấy doanh thu tháng.
func canDangNhap(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := NguoiDangNhap(r); !ok {
			http.Redirect(w, r, "/dang-nhap", http.StatusSeeOther)
			return
		}
		h(w, r)
	}
}

func canLaChu(h http.HandlerFunc) http.HandlerFunc {
	return canDangNhap(func(w http.ResponseWriter, r *http.Request) {
		nd, _ := NguoiDangNhap(r)
		if !nd.LaChu() {
			w.WriteHeader(http.StatusForbidden)
			render(w, "cam.html", chung(r, ""))
			return
		}
		h(w, r)
	})
}

// --- Dashboard agent (chỉ chủ trạm) ----------------------------------

func hDashboard(w http.ResponseWriter, r *http.Request) {
	render(w, "noibo.html", struct {
		Chung
		ThongKe  ThongKe
		DichVu   []DichVu
		KhongBan []KhongBan
		YeuCau   []yeuCauKhach
	}{chung(r, "agent"), LayThongKe(), DichVuDangBan(GiaiDoan), GIA.KhongBan, docYeuCau(20)})
}

func docYeuCau(n int) []yeuCauKhach {
	dir := P(CFG.DuongDan.TinNhan)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := []yeuCauKhach{}
	for i := len(ents) - 1; i >= 0 && (n <= 0 || len(out) < n); i-- {
		e := ents[i]
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var yc yeuCauKhach
		if json.Unmarshal(b, &yc) == nil {
			if yc.Ma == "" {
				yc.Ma = strings.TrimSuffix(e.Name(), ".json")
			}
			out = append(out, yc)
		}
	}
	return out
}

func luuYeuCau(yc yeuCauKhach) error {
	b, err := json.MarshalIndent(yc, "", "  ")
	if err != nil {
		return err
	}
	return ghiAtomic(filepath.Join(P(CFG.DuongDan.TinNhan), yc.Ma+".json"), b)
}

func hApiHoi(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonLoi(w, http.StatusMethodNotAllowed, "chỉ nhận POST")
		return
	}
	var req struct {
		CauHoi string `json:"cau_hoi"`
	}
	if json.NewDecoder(http.MaxBytesReader(w, r.Body, 16*1024)).Decode(&req) != nil {
		jsonLoi(w, http.StatusBadRequest, "JSON hỏng")
		return
	}
	kq, err := HoiKyThuat(req.CauHoi)
	if err != nil {
		jsonLoi(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, kq)
}

func hApiIndex(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonLoi(w, http.StatusMethodNotAllowed, "chỉ nhận POST")
		return
	}
	lamLai := r.URL.Query().Get("lam_lai") == "1"
	tk, err := BuildIndex(lamLai, func(s string) { fmt.Println("[rag]", s) })
	if err != nil {
		jsonLoi(w, http.StatusInternalServerError, err.Error())
		return
	}
	jsonOK(w, tk)
}

func hApiThongKe(w http.ResponseWriter, r *http.Request) { jsonOK(w, LayThongKe()) }

func hApiTim(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if strings.TrimSpace(q) == "" {
		jsonLoi(w, http.StatusBadRequest, "thiếu tham số q")
		return
	}
	jsonOK(w, Search(q, 0))
}

// --- Tiện ích -------------------------------------------------------

func render(w http.ResponseWriter, ten string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	// Mọi trang ở đây đều là trang động: trạng thái đơn, nội dung Kendy vừa sửa
	// trong admin, thẻ mời cài app. Không nói gì thì trình duyệt tự đoán lấy một
	// hạn cache — và khách sẽ đọc trạng thái đơn của hôm qua. no-cache không cấm
	// giữ bản sao, chỉ bắt hỏi lại máy chủ trước khi dùng.
	w.Header().Set("Cache-Control", "no-cache")
	if err := tpl.ExecuteTemplate(w, ten, data); err != nil {
		fmt.Printf("[web] lỗi render %s: %v\n", ten, err)
		http.Error(w, "Lỗi hiển thị trang", http.StatusInternalServerError)
	}
}

func jsonOK(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(v)
}

func jsonLoi(w http.ResponseWriter, code int, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"loi": msg})
}

// NewMux dựng router.
//
//	public=true  : khách + quản trị + agent, agent khóa sau canLaChu.
//	public=false : thêm /api/index. Cả hai chế độ đều cần GEMINI_API_KEY.
func NewMux(public bool) *http.ServeMux {
	CongKhai = public
	mux := http.NewServeMux()

	// Trang khách.
	mux.HandleFunc("GET /{$}", hTrangChu)
	mux.HandleFunc("GET /gioi-thieu", hGioiThieu)
	mux.HandleFunc("GET /ve-chung-toi", hVeChungToi)
	mux.HandleFunc("GET /dich-vu", hDichVuList)
	mux.HandleFunc("GET /dich-vu/{ma}", hDichVuMot)
	mux.HandleFunc("GET /quy-trinh", hQuyTrinh)
	mux.HandleFunc("GET /bai-viet", hBaiVietList)
	mux.HandleFunc("GET /bai-viet/{slug}", hBaiVietMot)
	mux.HandleFunc("GET /anh-trang-chu/{ten}", hAnhTrangChu)
	mux.HandleFunc("GET /bai-viet-anh/{ten}", hAnhBaiViet)
	mux.HandleFunc("GET /sitemap.xml", hSitemap)
	mux.HandleFunc("GET /robots.txt", hRobots)
	mux.HandleFunc("GET /favicon.svg", hFavicon)
	// App cài lên màn hình chính. Xem core/pwa.go — cả ba đường này chỉ có
	// tác dụng khi site chạy https.
	mux.HandleFunc("GET /app", hApp)
	mux.HandleFunc("GET /manifest.webmanifest", hManifest)
	mux.HandleFunc("GET /sw.js", hServiceWorker)
	mux.HandleFunc("GET /logo", phucVuLogo(loaiLogo))
	mux.HandleFunc("GET /bieu-tuong", phucVuLogo(loaiIcon))
	mux.HandleFunc("GET /anh-gd/{ten}", hAnhNen)
	mux.HandleFunc("GET /lien-he", hLienHe)
	mux.HandleFunc("GET /chinh-sach", hChinhSach)
	mux.HandleFunc("POST /gui-yeu-cau", hGuiYeuCau)
	mux.HandleFunc("GET /tra-cuu", hTraCuu)
	mux.HandleFunc("POST /tra-cuu", hTraCuu)

	// Bản app của ba trang khách hay dùng nhất. Cùng handler, cùng dữ liệu,
	// khác cái vỏ — xem mauTheoVo trong pwa.go.
	mux.HandleFunc("GET /app/tra-cuu", hTraCuu)
	mux.HandleFunc("POST /app/tra-cuu", hTraCuu)
	mux.HandleFunc("GET /app/quy-trinh", hQuyTrinh)
	mux.HandleFunc("GET /app/gui-anh", hAppGuiAnh)
	// Trang soi máy, không có đường dẫn nào trỏ tới. Mở tay khi cần biết máy
	// khách đang khai báo gì với trang.
	mux.HandleFunc("GET /app/kiem", hAppKiem)
	mux.HandleFunc("GET /app/dich-vu/{ma}", hDichVuMot)
	mux.HandleFunc("POST /app/gui-anh", hGuiYeuCau)
	mux.HandleFunc("GET /don-anh/{token}/{ten}", hAnhDonChoKhach)

	// Đăng nhập.
	mux.HandleFunc("GET /dang-nhap", hDangNhap)
	mux.HandleFunc("POST /dang-nhap", hDangNhap)
	mux.HandleFunc("POST /dang-xuat", hDangXuat)
	mux.HandleFunc("GET /dang-xuat", hDangXuat)

	// Quản trị — có ở CẢ HAI chế độ, vì thợ dùng qua internet.
	dangKyQuanTri(mux)

	// Agent — có ở CẢ HAI chế độ, nhưng canLaChu. Thợ đăng nhập vào /qt
	// bình thường vẫn bị chặn 403 ở đây: một câu hỏi là một request Gemini,
	// và bộ đếm quota 900/ngày dùng chung với phần Python.
	mux.HandleFunc("/noi-bo", canLaChu(hDashboard))
	mux.HandleFunc("/api/hoi", canLaChu(hApiHoi))
	mux.HandleFunc("/api/thong-ke", canLaChu(hApiThongKe))
	mux.HandleFunc("/api/tim", canLaChu(hApiTim))
	// Dựng index thì khác: một lần bấm là hàng trăm request nhúng vector.
	// Chỉ để ở máy nhà, nơi có người ngồi nhìn nó chạy và biết dừng.
	if !public {
		mux.HandleFunc("/api/index", canLaChu(hApiIndex))
	}
	return mux
}
