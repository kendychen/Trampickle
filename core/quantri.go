// Trang quản trị: đơn hàng, tiền đã chốt, thống kê, tài khoản thợ.
//
// Phân quyền chỉ có hai mức và cố ý giữ nó đơn giản:
//
//	chu : thấy hết, kể cả doanh thu tháng và trang tài khoản.
//	tho : thấy đơn mình đang cầm và đơn chưa ai nhận. Không thấy thống kê.
//
// Vì sao thợ không thấy thống kê: doanh thu tháng và tỷ lệ từ chối là
// chuyện của chủ. Thợ cần biết vợt nào đang trên bàn mình, hẹn trả ngày
// nào, khách đã đưa bao nhiêu tiền — chỉ vậy.
package core

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func dangKyQuanTri(mux *http.ServeMux) {
	mux.HandleFunc("GET /qt", canDangNhap(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/qt/don", http.StatusSeeOther)
	}))
	mux.HandleFunc("GET /qt/don", canDangNhap(hQtDonDs))
	mux.HandleFunc("GET /qt/don-moi", canDangNhap(hQtDonMoi))
	mux.HandleFunc("POST /qt/don-moi", canDangNhap(hQtDonMoi))
	mux.HandleFunc("GET /qt/don/{ma}", canDangNhap(hQtDonChiTiet))
	mux.HandleFunc("POST /qt/don/{ma}/luu", canDangNhap(hQtDonLuu))
	mux.HandleFunc("POST /qt/don/{ma}/trang-thai", canDangNhap(hQtDoiTrangThai))
	mux.HandleFunc("POST /qt/don/{ma}/anh", canDangNhap(hQtTaiAnh))
	mux.HandleFunc("POST /qt/don/{ma}/xoa-anh", canDangNhap(hQtXoaAnh))
	mux.HandleFunc("GET /qt/anh/{ma}/{ten}", canDangNhap(hQtXemAnh))

	mux.HandleFunc("GET /qt/yeu-cau", canDangNhap(hQtYeuCau))
	mux.HandleFunc("POST /qt/yeu-cau/{ma}/tao-don", canDangNhap(hQtYeuCauTaoDon))
	mux.HandleFunc("POST /qt/yeu-cau/{ma}/bo-qua", canDangNhap(hQtYeuCauBoQua))
	mux.HandleFunc("GET /qt/yeu-cau/{ma}/tep/{ten}", canDangNhap(hQtTepYeuCau))

	mux.HandleFunc("GET /qt/mat-khau", canDangNhap(hQtMatKhau))
	mux.HandleFunc("POST /qt/mat-khau", canDangNhap(hQtMatKhau))

	mux.HandleFunc("GET /qt/thong-ke", canLaChu(hQtThongKe))
	mux.HandleFunc("GET /qt/nguoi-dung", canLaChu(hQtNguoiDung))
	mux.HandleFunc("POST /qt/nguoi-dung", canLaChu(hQtNguoiDungLuu))

	// Đổi giao diện là việc đổi mặt tiền cho cả trạm — thợ không nên bấm được.
	mux.HandleFunc("GET /qt/giao-dien", canLaChu(hQtGiaoDien))
	mux.HandleFunc("POST /qt/giao-dien", canLaChu(hQtGiaoDien))
	mux.HandleFunc("POST /qt/giao-dien/logo", canLaChu(hQtLogoTai))
	mux.HandleFunc("POST /qt/giao-dien/logo-xoa", canLaChu(hQtLogoXoa))

	// Ngưỡng ghi thẳng vào vanhanh/bang-gia.yaml, mà file đó quyết định cả
	// con số báo giá bên Python — thợ không được đụng.
	mux.HandleFunc("GET /qt/nguong", canLaChu(hQtNguong))
	mux.HandleFunc("POST /qt/nguong", canLaChu(hQtNguong))

	// Danh sách dịch vụ cũng nằm trong bang-gia.yaml, cùng luật với ngưỡng.
	mux.HandleFunc("GET /qt/cai-dat", canLaChu(hQtCaiDat))
	mux.HandleFunc("POST /qt/cai-dat", canLaChu(hQtCaiDat))
	mux.HandleFunc("GET /qt/lien-he", canLaChu(hQtLienHe))
	mux.HandleFunc("POST /qt/lien-he", canLaChu(hQtLienHe))
	mux.HandleFunc("GET /qt/dich-vu", canLaChu(hQtDichVu))
	mux.HandleFunc("POST /qt/dich-vu", canLaChu(hQtDichVu))

	// Ảnh trang chủ cũng là mặt tiền, cùng luật với giao diện.
	mux.HandleFunc("GET /qt/anh-trang-chu", canLaChu(hQtAnhTrangChu))
	mux.HandleFunc("POST /qt/anh-trang-chu/them", canLaChu(hQtAnhTrangChuThem))
	mux.HandleFunc("POST /qt/anh-trang-chu/sua", canLaChu(hQtAnhTrangChuSua))

	// Bài viết cũng là mặt tiền: chữ ở đây là thứ Google đọc và khách đọc
	// trước khi gửi vợt. Thợ sửa đơn thì được, sửa lời trạm nói thì không.
	// "moi" là đoạn đường chữ sẵn nên nó thắng {slug} — luật ưu tiên của
	// ServeMux Go 1.22, không phải nhờ thứ tự đăng ký ở đây.
	mux.HandleFunc("GET /qt/bai-viet", canLaChu(hQtBaiDs))
	mux.HandleFunc("GET /qt/bai-viet/moi", canLaChu(hQtBaiMoi))
	mux.HandleFunc("POST /qt/bai-viet/luu", canLaChu(hQtBaiLuu))
	mux.HandleFunc("POST /qt/bai-viet/xem-thu", canLaChu(hQtBaiXemThu))
	mux.HandleFunc("POST /qt/bai-viet/anh", canLaChu(hQtBaiAnhThem))
	mux.HandleFunc("POST /qt/bai-viet/anh-xoa", canLaChu(hQtBaiAnhXoa))
	mux.HandleFunc("GET /qt/bai-viet/{slug}", canLaChu(hQtBaiSua))
	mux.HandleFunc("POST /qt/bai-viet/{slug}/xoa", canLaChu(hQtBaiXoa))

	// Chữ trên trang: cùng lý do với bài viết. Đây là lời trạm nói với khách.
	// Giáo trình dạy nghề — nội bộ, xem qtgiaotrinh.go.
	mux.HandleFunc("GET /qt/giao-trinh", canLaChu(hQtGTDs))
	mux.HandleFunc("POST /qt/giao-trinh/luu", canLaChu(hQtGTLuu))
	mux.HandleFunc("POST /qt/giao-trinh/xem-thu", canLaChu(hQtGTXemThu))
	// {ma...} chứ không {ma}: mã bài có dấu gạch chéo (03-ca-sua/3-2-tach-lop.md).
	mux.HandleFunc("GET /qt/giao-trinh/bai/{ma...}", canLaChu(hQtGTSua))

	mux.HandleFunc("GET /qt/noi-dung", canLaChu(hQtND))
	mux.HandleFunc("POST /qt/noi-dung/luu", canLaChu(hQtNDLuu))
	mux.HandleFunc("POST /qt/noi-dung/khoi-phuc", canLaChu(hQtNDKhoiPhuc))
	mux.HandleFunc("GET /qt/noi-dung/{ma}", canLaChu(hQtND))

	dangKyKho(mux)
	dangKyVot(mux)
	dangKySoTien(mux)
}

// --- Tiện ích đọc form ----------------------------------------------

// soTien nhận "150000", "150.000", "150,000đ" — người nhập tay hay gõ dấu
// chấm. Chỉ giữ chữ số. Đây là ĐỌC một con số người ta gõ vào, không phải
// tính ra một con số.
func soTien(s string) int {
	var b strings.Builder
	am := strings.HasPrefix(strings.TrimSpace(s), "-")
	for _, c := range s {
		if c >= '0' && c <= '9' {
			b.WriteRune(c)
		}
	}
	if b.Len() == 0 {
		return 0
	}
	n, err := strconv.Atoi(b.String())
	if err != nil {
		return 0
	}
	if am {
		return -n
	}
	return n
}

func soThuc(s string) float64 {
	s = strings.TrimSpace(strings.ReplaceAll(s, ",", "."))
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

// ngayISO nhận "2026-09-03" từ <input type=date>, trả về rỗng nếu sai.
func ngayISO(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	if _, err := time.Parse("2006-01-02", s); err != nil {
		return ""
	}
	return s
}

// xemDuocDon: thợ chỉ mở được đơn mình cầm hoặc đơn chưa ai nhận.
// Không phải để chống nội gián — để một cái link gửi nhầm không mở ra
// thông tin khách của người khác.
func xemDuocDon(nd NguoiDung, d *Don) bool {
	if nd.LaChu() {
		return true
	}
	return d.ThoPhuTrach == "" || d.ThoPhuTrach == nd.Ten
}

type dlQt struct {
	Chung
	Loi string
	OK  string
}

// --- Danh sách đơn ---------------------------------------------------

func hQtDonDs(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	f := BoLoc{
		TrangThai: r.URL.Query().Get("trang_thai"),
		Tho:       r.URL.Query().Get("tho"),
		Tim:       r.URL.Query().Get("q"),
	}
	if r.URL.Query().Get("dang_chay") == "1" {
		f.ChiDangChay = true
	}
	if !nd.LaChu() {
		f.Tho = "" // thợ không được lọc theo thợ khác
	}
	ds := LocDon(f)
	if !nd.LaChu() {
		loc := ds[:0]
		for _, d := range ds {
			if xemDuocDon(nd, d) {
				loc = append(loc, d)
			}
		}
		ds = loc
	}

	tk := LayThongKeDon()
	render(w, "qt-don.html", struct {
		dlQt
		Don      []*Don
		Loc      BoLoc
		DangChay bool
		ThongKe  ThongKeDon
		Tho      []NguoiDung
		SoYeuCau int
	}{
		dlQt:     dlQt{Chung: chung(r, "don")},
		Don:      ds,
		Loc:      f,
		DangChay: r.URL.Query().Get("dang_chay") == "1",
		ThongKe:  tk,
		Tho:      DanhSachNguoiDung(),
		SoYeuCau: demYeuCauChuaXuLy(),
	})
}

func demYeuCauChuaXuLy() int {
	n := 0
	for _, y := range docYeuCau(0) {
		if !y.DaXuLy {
			n++
		}
	}
	return n
}

// --- Tạo đơn ---------------------------------------------------------

func hQtDonMoi(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	d := struct {
		dlQt
		DichVu []DichVu
		Tho    []NguoiDung
		Nguong Nguong
		Truoc  *Don
	}{
		dlQt:   dlQt{Chung: chung(r, "don-moi")},
		DichVu: DichVuDangBan(GiaiDoan),
		Tho:    DanhSachNguoiDung(),
		Nguong: NguongHienTai(),
	}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
		if err := r.ParseForm(); err != nil {
			d.Loi = "Nội dung quá dài."
			render(w, "qt-don-moi.html", d)
			return
		}
		don := &Don{
			Ma:          MaDonMoi(),
			Token:       maNgauNhien(8),
			Ngay:        time.Now().Format("2006-01-02"),
			TrangThai:   TTMoi,
			ThoPhuTrach: nd.Ten,
		}
		if nd.LaChu() && r.FormValue("tho") != "" {
			don.ThoPhuTrach = r.FormValue("tho")
		}
		apDungForm(don, r, nd)
		if strings.TrimSpace(don.KhachLienHe) == "" {
			d.Loi = "Phải có số điện thoại/Zalo của khách — không có thì sau này không tra cứu được."
			d.Truoc = don
			render(w, "qt-don-moi.html", d)
			return
		}
		GhiMoc(don, don.TrangThai, nd.TenHienThi(), "Tạo đơn")
		if err := LuuDon(don); err != nil {
			d.Loi = "Không lưu được: " + err.Error()
			d.Truoc = don
			render(w, "qt-don-moi.html", d)
			return
		}
		// Tiền cọc lúc nhận vợt: ghi thẳng thành khoản thu trong sổ. Lỗi ở
		// đây không được làm hỏng việc tạo đơn — vợt đã nằm trên bàn rồi.
		if coc := soTien(r.FormValue("da_thu")); coc > 0 {
			if err := ghiThuChoDon(don, coc, r.FormValue("phuong_thuc"), "Khách đưa lúc nhận vợt", nd); err != nil {
				http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok=Đã+tạo+đơn,+nhưng+chưa+ghi+được+tiền+cọc:+"+err.Error(), http.StatusSeeOther)
				return
			}
		}
		http.Redirect(w, r, "/qt/don/"+don.Ma, http.StatusSeeOther)
		return
	}
	render(w, "qt-don-moi.html", d)
}

// apDungForm chép các trường từ form vào đơn. Dòng tiền lấy nguyên con số
// người dùng gõ hoặc con số đọc từ bảng giá — không nhân chia gì.
func apDungForm(don *Don, r *http.Request, nd NguoiDung) {
	don.KhachTen = catBot(r.FormValue("khach_ten"), 100)
	don.KhachLienHe = catBot(r.FormValue("khach_lien_he"), 60)
	don.VotHang = catBot(r.FormValue("vot_hang"), 100)
	don.VotGiaTri = soTien(r.FormValue("vot_gia_tri"))
	don.TinhTrang = catBot(r.FormValue("tinh_trang"), 3000)
	don.ChanDoan = catBot(r.FormValue("chan_doan"), 3000)
	don.CanTruocG = soThuc(r.FormValue("can_truoc"))
	don.CanSauG = soThuc(r.FormValue("can_sau"))
	don.KenhNhan = r.FormValue("kenh_nhan")
	don.HenTraNgay = ngayISO(r.FormValue("hen_tra"))
	don.BaoHanhDen = ngayISO(r.FormValue("bao_hanh_den"))
	don.GhiChu = catBot(r.FormValue("ghi_chu"), 3000)
	// Tiền khách đưa KHÔNG lưu ở đơn nữa — nó là một khoản thu trong sổ tiền.
	// Xem Don.DaThu() và hQtGhiThu.
	if nd.LaChu() {
		if t := r.FormValue("tho"); t != "" {
			if _, co := TimNguoiDung(t); co {
				don.ThoPhuTrach = t
			}
		}
	}

	// Dòng tiền: các dịch vụ tick chọn từ bảng giá + các dòng gõ tay.
	dong := []DongTien{}
	dsDichVu := DichVuTatCa()
	for _, ma := range r.Form["dv"] {
		for _, dv := range dsDichVu {
			if dv.Ma != ma {
				continue
			}
			p := dv.GiaTheoGiaiDoan(GiaiDoan)
			if p == nil {
				continue
			}
			dong = append(dong, DongTien{MaDichVu: dv.Ma, Ten: dv.Ten, SoTien: *p})
		}
	}
	tenTay := r.Form["dong_ten"]
	tienTay := r.Form["dong_tien"]
	for i := range tenTay {
		ten := catBot(tenTay[i], 120)
		if ten == "" {
			continue
		}
		st := 0
		if i < len(tienTay) {
			st = soTien(tienTay[i])
		}
		dong = append(dong, DongTien{Ten: ten, SoTien: st})
	}
	// Giảm giá đứng cuối và là số âm. Kẹp không cho vượt tổng phía trên: một
	// đơn có tổng âm thì mọi thứ đọc số ấy đều sai theo — sổ nợ, thống kê,
	// phiếu in. Gõ thừa một số 0 là chuyện xảy ra thật.
	if g := soTien(r.FormValue("giam_gia")); g > 0 {
		if t := CongDongTien(dong); g > t {
			g = t
		}
		if g > 0 {
			ten := tenGiamGia
			if ly := catBot(r.FormValue("giam_ly_do"), 80); strings.TrimSpace(ly) != "" {
				ten += " — " + strings.TrimSpace(ly)
			}
			dong = append(dong, DongTien{MaDichVu: MaGiamGia, Ten: ten, SoTien: -g})
		}
	}
	don.DongTien = dong
	don.TongTien = CongDongTien(dong)
}

// --- Chi tiết đơn ----------------------------------------------------

func hQtDonChiTiet(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	c := chung(r, "don")
	c.TieuDe = "Đơn " + don.Ma
	phieuXuat, coPhieu := PhieuXuatCuaDon(don.Ma)
	render(w, "qt-don-ct.html", struct {
		dlQt
		Don       *Don
		DichVu    []DichVu
		Tho       []NguoiDung
		Nguong    Nguong
		CacTT     []MoTaTrangThai
		DaChonDV  map[string]bool
		DongThem  []DongTien
		LanThu    []Khoan
		LanHoan   []Khoan
		PhieuXuat *Phieu
		CoPhieu   bool
		DeXuatVT  []DongPhieu
	}{
		dlQt:      dlQt{Chung: c, OK: r.URL.Query().Get("ok")},
		Don:       don,
		DichVu:    DichVuDangBan(GiaiDoan),
		Tho:       DanhSachNguoiDung(),
		Nguong:    NguongHienTai(),
		CacTT:     CacTrangThai,
		DaChonDV:  chonDV(don),
		DongThem:  dongTay(don),
		LanThu:    CacLanThuCuaDon(don.Ma),
		LanHoan:   CacLanHoanCuaDon(don.Ma),
		PhieuXuat: phieuXuat,
		CoPhieu:   coPhieu,
		DeXuatVT:  DeXuatXuatChoDon(don),
	})
}

func chonDV(d *Don) map[string]bool {
	m := map[string]bool{}
	for _, dt := range d.DongTien {
		if dt.MaDichVu != "" {
			m[dt.MaDichVu] = true
		}
	}
	return m
}

func dongTay(d *Don) []DongTien {
	out := []DongTien{}
	for _, dt := range d.DongTien {
		if dt.MaDichVu == "" {
			out = append(out, dt)
		}
	}
	return out
}

func hQtDonLuu(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Nội dung quá dài", http.StatusRequestEntityTooLarge)
		return
	}
	apDungForm(don, r, nd)
	if err := LuuDon(don); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok=Đã+lưu", http.StatusSeeOther)
}

func hQtDoiTrangThai(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	tt := r.FormValue("trang_thai")
	if !TrangThaiHopLe(tt) {
		http.Error(w, "trạng thái không hợp lệ", http.StatusBadRequest)
		return
	}
	// Thợ nhận đơn chưa có ai cầm thì mặc nhiên đơn về tay thợ đó.
	if don.ThoPhuTrach == "" {
		don.ThoPhuTrach = nd.Ten
	}
	don.TrangThai = tt
	GhiMoc(don, tt, nd.TenHienThi(), catBot(r.FormValue("ghi_chu"), 300))
	if err := LuuDon(don); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape(nhacSauDoiTrangThai(don, tt)), http.StatusSeeOther)
}

// nhacSauDoiTrangThai — nhắc chứ không tự làm. Đơn đóng thì hai thứ hay bị
// quên là trừ kho và ghi tiền; cả hai đều phải có người xác nhận, nên chỗ này
// chỉ đẩy lời nhắc lên đầu trang.
func nhacSauDoiTrangThai(don *Don, tt string) string {
	s := "Đã đổi trạng thái"
	if tt != TTXong && tt != TTDaGiao {
		return s
	}
	if _, coPhieu := PhieuXuatCuaDon(don.Ma); !coPhieu && len(DeXuatXuatChoDon(don)) > 0 {
		s += ". Chưa xuất kho cho đơn này"
	}
	if n := don.ConNo(); n > 0 {
		s += ". Khách còn nợ " + dinhDangTien(n)
	}
	return s
}

// --- Ảnh -------------------------------------------------------------

const anhToiDaMoiDon = 16
const anhToiDaByte = 8 << 20

func thuMucAnh(ma string) string { return filepath.Join(P("data/anh-don"), ma) }

// duoiAnh nhận diện theo nội dung file, không theo tên. Tên file do người
// gửi đặt; chỉ có mấy byte đầu là nói thật về nó.
func duoiAnh(dau []byte) string {
	switch {
	case len(dau) > 3 && dau[0] == 0xFF && dau[1] == 0xD8 && dau[2] == 0xFF:
		return ".jpg"
	case len(dau) > 8 && string(dau[1:4]) == "PNG":
		return ".png"
	case len(dau) > 12 && string(dau[0:4]) == "RIFF" && string(dau[8:12]) == "WEBP":
		return ".webp"
	}
	return ""
}

func hQtTaiAnh(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	ma := strings.ToUpper(r.PathValue("ma"))
	don, ok := LayDon(ma)
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, anhToiDaByte+1<<20)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		http.Redirect(w, r, "/qt/don/"+ma+"?ok=Ảnh+quá+nặng", http.StatusSeeOther)
		return
	}

	// Biểu mẫu có tệp: lớp bọc không đọc được thân yêu cầu nên không kiểm
	// CSRF hộ được. Xem core/csrf.go.
	if !KiemCSRFMultipart(w, r) {
		return
	}

	nhan := r.FormValue("nhan") // truoc | sau
	if nhan != "sau" {
		nhan = "truoc"
	}
	files := r.MultipartForm.File["anh"]
	them := 0
	for _, fh := range files {
		if len(don.Anh)+them >= anhToiDaMoiDon {
			break
		}
		f, err := fh.Open()
		if err != nil {
			continue
		}
		dau := make([]byte, 16)
		n, _ := io.ReadFull(f, dau)
		duoi := duoiAnh(dau[:n])
		if duoi == "" {
			f.Close()
			continue
		}
		os.MkdirAll(thuMucAnh(ma), 0o755)
		ten := fmt.Sprintf("%s-%s-%s%s", nhan, time.Now().Format("150405"), maNgauNhien(3), duoi)
		out, err := os.Create(filepath.Join(thuMucAnh(ma), ten))
		if err != nil {
			f.Close()
			continue
		}
		out.Write(dau[:n])
		io.Copy(out, io.LimitReader(f, anhToiDaByte))
		out.Close()
		f.Close()
		don.Anh = append(don.Anh, ten)
		them++
	}
	LuuDon(don)
	http.Redirect(w, r, "/qt/don/"+ma+"?ok=Đã+thêm+ảnh", http.StatusSeeOther)
}

func hQtXoaAnh(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	ma := strings.ToUpper(r.PathValue("ma"))
	don, ok := LayDon(ma)
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	r.ParseForm()
	ten := r.FormValue("ten")
	con := don.Anh[:0]
	for _, a := range don.Anh {
		if a == ten {
			os.Remove(filepath.Join(thuMucAnh(ma), a))
			continue
		}
		con = append(con, a)
	}
	don.Anh = con
	LuuDon(don)
	http.Redirect(w, r, "/qt/don/"+ma+"?ok=Đã+xoá+ảnh", http.StatusSeeOther)
}

// hQtXemAnh chỉ phục vụ tên file CÓ TRONG danh sách của đơn. Không ghép
// đường dẫn từ tham số URL — đó là cách ../../etc/passwd đi ra ngoài.
func hQtXemAnh(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	ma := strings.ToUpper(r.PathValue("ma"))
	don, ok := LayDon(ma)
	if !ok || !xemDuocDon(nd, don) {
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
	http.ServeFile(w, r, filepath.Join(thuMucAnh(ma), ten))
}

// --- Yêu cầu từ web --------------------------------------------------

func hQtYeuCau(w http.ResponseWriter, r *http.Request) {
	render(w, "qt-yeucau.html", struct {
		dlQt
		YeuCau []yeuCauKhach
	}{dlQt{Chung: chung(r, "yeu-cau"), OK: r.URL.Query().Get("ok")}, docYeuCau(100)})
}

// hQtTepYeuCau chỉ phục vụ tên tệp CÓ TRONG yêu cầu, giống hQtXemAnh:
// không ghép đường dẫn từ tham số URL. Ảnh khách gửi nằm sau đăng nhập vì
// đó là vợt của người ta, không phải thư viện ảnh công khai.
func hQtTepYeuCau(w http.ResponseWriter, r *http.Request) {
	yc, ok := timYeuCau(r.PathValue("ma"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	ten := r.PathValue("ten")
	found := false
	for _, t := range yc.Tep {
		if t == ten {
			found = true
			break
		}
	}
	if !found {
		http.NotFound(w, r)
		return
	}
	if ct := kieuTep[strings.ToLower(filepath.Ext(ten))]; ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Cache-Control", "private, max-age=3600")
	http.ServeFile(w, r, filepath.Join(thuMucTepYeuCau(yc.Ma), ten))
}

func timYeuCau(ma string) (yeuCauKhach, bool) {
	for _, y := range docYeuCau(0) {
		if y.Ma == ma {
			return y, true
		}
	}
	return yeuCauKhach{}, false
}

// hQtYeuCauTaoDon biến một tin nhắn từ web thành đơn thật, chép sẵn tên,
// liên hệ, hãng vợt và mô tả. Đỡ gõ lại, và quan trọng hơn là đỡ gõ sai
// số điện thoại — số sai thì khách không tra cứu được đơn của mình.
func hQtYeuCauTaoDon(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	yc, ok := timYeuCau(r.PathValue("ma"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	if yc.MaDon != "" {
		http.Redirect(w, r, "/qt/don/"+yc.MaDon, http.StatusSeeOther)
		return
	}
	don := &Don{
		Ma:          MaDonMoi(),
		Token:       maNgauNhien(8),
		Ngay:        time.Now().Format("2006-01-02"),
		TrangThai:   TTMoi,
		ThoPhuTrach: nd.Ten,
		KhachTen:    yc.Ten,
		KhachLienHe: yc.LienHe,
		VotHang:     yc.VotHang,
		TinhTrang:   yc.MoTa,
		KenhNhan:    "web",
	}
	GhiMoc(don, TTMoi, nd.TenHienThi(), "Tạo từ yêu cầu web "+yc.Ma)
	if err := LuuDon(don); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	yc.DaXuLy = true
	yc.MaDon = don.Ma
	luuYeuCau(yc)
	http.Redirect(w, r, "/qt/don/"+don.Ma, http.StatusSeeOther)
}

func hQtYeuCauBoQua(w http.ResponseWriter, r *http.Request) {
	yc, ok := timYeuCau(r.PathValue("ma"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	yc.DaXuLy = true
	luuYeuCau(yc)
	http.Redirect(w, r, "/qt/yeu-cau?ok=Đã+đánh+dấu+xong", http.StatusSeeOther)
}

// --- Thống kê --------------------------------------------------------

func hQtThongKe(w http.ResponseWriter, r *http.Request) {
	tk := LayThongKeDon()
	render(w, "qt-thongke.html", struct {
		dlQt
		TK      ThongKeDon
		DinhPhi int
		CaoNhat int
	}{
		dlQt:    dlQt{Chung: chung(r, "thong-ke")},
		TK:      tk,
		DinhPhi: GIA.DinhPhiThang,
		CaoNhat: caoNhat(tk.DoanhThuThang),
	})
}

func caoNhat(ds []ThangDoanhThu) int {
	m := 0
	for _, t := range ds {
		if t.DoanhThu > m {
			m = t.DoanhThu
		}
	}
	if m == 0 {
		return 1
	}
	return m
}

// --- Tài khoản -------------------------------------------------------

func hQtNguoiDung(w http.ResponseWriter, r *http.Request) {
	render(w, "qt-nguoidung.html", struct {
		dlQt
		DS []NguoiDung
	}{dlQt{Chung: chung(r, "nguoi-dung"), OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")}, DanhSachNguoiDung()})
}

func hQtNguoiDungLuu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	var err error
	switch r.FormValue("viec") {
	case "them":
		err = ThemNguoiDung(r.FormValue("ten"), r.FormValue("ho_ten"),
			r.FormValue("mat_khau"), r.FormValue("vai_tro"))
	case "vai_tro":
		err = DoiVaiTro(r.FormValue("ten"), r.FormValue("vai_tro"), r.FormValue("tat") == "1")
	case "mat_khau":
		err = DoiMatKhau(r.FormValue("ten"), r.FormValue("mat_khau"))
	default:
		err = fmt.Errorf("không hiểu yêu cầu")
	}
	if err != nil {
		http.Redirect(w, r, "/qt/nguoi-dung?loi="+urlEsc(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/nguoi-dung?ok=Xong", http.StatusSeeOther)
}

func hQtMatKhau(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	d := dlQt{Chung: chung(r, "mat-khau")}
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
		r.ParseForm()
		if _, err := KiemTraDangNhap(nd.Ten, r.FormValue("cu")); err != nil {
			d.Loi = "Mật khẩu cũ không đúng."
		} else if r.FormValue("moi") != r.FormValue("moi2") {
			d.Loi = "Hai ô mật khẩu mới không giống nhau."
		} else if err := DoiMatKhau(nd.Ten, r.FormValue("moi")); err != nil {
			d.Loi = err.Error()
		} else {
			d.OK = "Đã đổi mật khẩu."
		}
	}
	render(w, "qt-matkhau.html", d)
}

// --- Chọn giao diện --------------------------------------------------

// Template không biết gì về từng giao diện: nó chỉ lặp DanhSach và so mã với
// DangDung. Thêm giao diện thứ ba chỉ phải sửa ThemeCo trong giaodien.go.
type dlGiaoDien struct {
	dlQt
	DanhSach []Theme
	DangDung string
	// Tên tệp rỗng = đang dùng bản nhúng trong code; template lấy đó làm
	// mốc để hiện hay giấu nút bỏ tệp.
	LogoTen string
	LogoURL string
	IconTen string
	IconURL string
}

func hQtGiaoDien(w http.ResponseWriter, r *http.Request) {
	d := dlGiaoDien{dlQt: dlQt{Chung: chung(r, "giao-dien")}}
	// Việc tải logo quay về đây bằng redirect (POST xong không để lại form
	// trong lịch sử), nên lời báo tới qua thanh địa chỉ.
	d.OK = r.URL.Query().Get("ok")
	d.Loi = r.URL.Query().Get("loi")
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
		r.ParseForm()
		if err := DatTheme(r.FormValue("theme")); err != nil {
			d.Loi = err.Error()
		} else {
			d.OK = "Đã đổi giao diện. Mở trang khách ở tab khác để xem."
		}
	}
	// Đọc sau khi ghi, không dùng lại giá trị trong d.Chung: chung() chạy từ
	// đầu hàm nên nó vẫn giữ giao diện cũ, hiện ra sẽ như là lưu hụt.
	d.DanhSach = ThemeCo
	d.DangDung = ThemeHienTai()
	d.LogoTen, _ = tepLogo(loaiLogo)
	d.LogoURL = logoURL()
	d.IconTen, _ = tepLogo(loaiIcon)
	d.IconURL = iconURL()
	render(w, "qt-giaodien.html", d)
}

func urlEsc(s string) string {
	return strings.NewReplacer(" ", "+", "&", "%26", "#", "%23", "?", "%3F").Replace(s)
}
