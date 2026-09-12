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
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func dangKyQuanTri(mux *http.ServeMux) {
	mux.HandleFunc("GET /qt", canDangNhap(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/qt/don-vot", http.StatusSeeOther)
	}))
	// /qt/don là đường cũ — bookmark và link trong mail đã gửi vẫn còn trỏ tới
	// đây, đừng bỏ.
	mux.HandleFunc("GET /qt/don", canDangNhap(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/qt/don-vot", http.StatusSeeOther)
	}))
	// App quản lý cài lên màn hình chính — xem core/pwa.go. Ba đường này KHÔNG
	// bọc canDangNhap: điện thoại đi lấy manifest và icon bằng request riêng,
	// có lúc không kèm cookie phiên, và chặn lại thì máy im lặng không mời cài
	// mà không báo lỗi gì. Cả ba đều tĩnh, không có dữ liệu trạm trong đó.
	mux.HandleFunc("GET /qt/manifest.webmanifest", hManifestQt)
	mux.HandleFunc("GET /qt/icon.svg", hIconQt)
	mux.HandleFunc("GET /qt/icon.png", hIconQtPNG)
	mux.HandleFunc("GET /qt/mat-mang", hQtMatMang)

	// /qt/don-hang — một cửa vào cho cả khu quản lý đơn, để dải menu bên
	// trái chỉ còn một mục. Bốn tab bên trong VẪN giữ nguyên bốn đường cũ:
	// lối tắt trên màn hình chính của điện thoại trỏ /qt/don-vot, và mấy lá
	// mail đã gửi đi rồi thì không thu về sửa link được nữa.
	mux.HandleFunc("GET /qt/don-hang", canDangNhap(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/qt/don-vot?"+DocKyTu(r).Q(), http.StatusSeeOther)
	}))
	mux.HandleFunc("GET /qt/don-vot", canDangNhap(hQtDonDs(LoaiVot)))
	mux.HandleFunc("GET /qt/don-giay", canDangNhap(hQtDonDs(LoaiGiay)))
	mux.HandleFunc("GET /qt/don-vot.csv", canDangNhap(hQtXuatDon(LoaiVot)))
	mux.HandleFunc("GET /qt/don-giay.csv", canDangNhap(hQtXuatDon(LoaiGiay)))
	mux.HandleFunc("GET /qt/don-moi", canDangNhap(hQtDonMoi))
	mux.HandleFunc("POST /qt/don-moi", canDangNhap(hQtDonMoi))
	mux.HandleFunc("GET /qt/don/{ma}", canDangNhap(hQtDonChiTiet))
	mux.HandleFunc("POST /qt/don/{ma}/luu", canDangNhap(hQtDonLuu))
	mux.HandleFunc("POST /qt/don/{ma}/trang-thai", canDangNhap(hQtDoiTrangThai))
	mux.HandleFunc("POST /qt/don/{ma}/nhan-hang", canDangNhap(hQtNhanHang))
	mux.HandleFunc("POST /qt/don/{ma}/van-don-ve", canDangNhap(hQtVanDonVe))
	mux.HandleFunc("POST /qt/don/{ma}/gui-di", canDangNhap(hQtGuiDi))
	mux.HandleFunc("POST /qt/don/{ma}/tra-doi-tac", canLaChu(hQtTraDoiTac))
	mux.HandleFunc("POST /qt/don/{ma}/anh", canDangNhap(hQtTaiAnh))
	mux.HandleFunc("POST /qt/don/{ma}/xoa-anh", canDangNhap(hQtXoaAnh))
	mux.HandleFunc("GET /qt/don/{ma}/hoa-don", canDangNhap(hQtHoaDon))
	mux.HandleFunc("GET /qt/don/{ma}/tem", canDangNhap(hQtTem))
	mux.HandleFunc("GET /qt/tem", canDangNhap(hQtTemNhieu))
	mux.HandleFunc("GET /qt/don/{ma}/mail", canDangNhap(hQtDonMail))
	mux.HandleFunc("POST /qt/don/{ma}/gui-mail", canDangNhap(hQtDonGuiMail))
	mux.HandleFunc("GET /qt/anh/{ma}/{ten}", canDangNhap(hQtXemAnh))

	mux.HandleFunc("GET /qt/yeu-cau", canDangNhap(hQtYeuCau))
	mux.HandleFunc("POST /qt/yeu-cau/{ma}/tao-don", canDangNhap(hQtYeuCauTaoDon))
	mux.HandleFunc("POST /qt/yeu-cau/{ma}/bo-qua", canDangNhap(hQtYeuCauBoQua))
	mux.HandleFunc("GET /qt/yeu-cau/{ma}/tep/{ten}", canDangNhap(hQtTepYeuCau))

	mux.HandleFunc("GET /qt/mat-khau", canDangNhap(hQtMatKhau))
	mux.HandleFunc("POST /qt/mat-khau", canDangNhap(hQtMatKhau))

	mux.HandleFunc("GET /qt/thong-ke", canLaChu(hQtThongKe))
	mux.HandleFunc("GET /qt/thong-ke.csv", canLaChu(hQtXuatThongKe))
	mux.HandleFunc("GET /qt/nhat-ky", canLaChu(hQtNhatKy))
	mux.HandleFunc("GET /qt/2fa", canDangNhap(hQt2FA))
	mux.HandleFunc("POST /qt/2fa", canDangNhap(hQt2FA))
	mux.HandleFunc("GET /qt/nguoi-dung", canLaChu(hQtNguoiDung))
	mux.HandleFunc("POST /qt/nguoi-dung", canLaChu(hQtNguoiDungLuu))

	// Đổi giao diện là việc đổi mặt tiền cho cả trạm — thợ không nên bấm được.
	mux.HandleFunc("GET /qt/giao-dien", canLaChu(hQtGiaoDien))
	mux.HandleFunc("POST /qt/giao-dien/che-do", canLaChu(hQtCheDo))
	mux.HandleFunc("POST /qt/giao-dien/noi-bat", canLaChu(hQtNoiBat))
	mux.HandleFunc("POST /qt/giao-dien/logo", canLaChu(hQtLogoTai))
	mux.HandleFunc("POST /qt/giao-dien/logo-xoa", canLaChu(hQtLogoXoa))

	// Ngưỡng ghi thẳng vào vanhanh/bang-gia.yaml, mà file đó quyết định cả
	// con số báo giá bên Python — thợ không được đụng.
	mux.HandleFunc("GET /qt/nguong", canLaChu(hQtNguong))
	mux.HandleFunc("POST /qt/nguong", canLaChu(hQtNguong))

	// Danh sách dịch vụ cũng nằm trong bang-gia.yaml, cùng luật với ngưỡng.
	mux.HandleFunc("GET /qt/cai-dat", canLaChu(hQtCaiDat))
	mux.HandleFunc("POST /qt/cai-dat", canLaChu(hQtCaiDat))

	// Bật/tắt thông báo đẩy cho MỘT máy. Chỉ chủ trạm: người nhận thông báo
	// đã chốt là một mình Kendy, và thợ bật được thì mỗi đơn web lại rung máy
	// thợ lúc nửa đêm. JS gọi ba đường này, kèm X-CSRF-Token.
	mux.HandleFunc("GET /qt/push/khoa", canLaChu(hQtPushKhoa))
	mux.HandleFunc("POST /qt/push/dang-ky", canLaChu(hQtPushDangKy))
	mux.HandleFunc("POST /qt/push/bo", canLaChu(hQtPushBo))
	mux.HandleFunc("GET /qt/lien-he", canLaChu(hQtLienHe))
	mux.HandleFunc("POST /qt/lien-he", canLaChu(hQtLienHe))
	mux.HandleFunc("GET /qt/dich-vu", canLaChu(hQtDichVu))
	mux.HandleFunc("POST /qt/dich-vu", canLaChu(hQtDichVu))
	mux.HandleFunc("GET /qt/dich-vu/{ma}/bai", canLaChu(hQtDichVuBai))
	mux.HandleFunc("POST /qt/dich-vu/{ma}/bai", canLaChu(hQtDichVuBai))

	// Câu hỏi thường gặp: chữ đứng tên trạm nói với khách, cùng luật với
	// /qt/noi-dung — thợ không sửa.
	mux.HandleFunc("GET /qt/cau-hoi", canLaChu(hQtCauHoi))
	mux.HandleFunc("POST /qt/cau-hoi/luu", canLaChu(hQtCauHoiLuu))
	mux.HandleFunc("POST /qt/cau-hoi/xoa", canLaChu(hQtCauHoiXoa))

	// Chọn gửi việc cho tiệm nào, và biết đã trả tiệm bao nhiêu — việc của chủ.
	mux.HandleFunc("GET /qt/doi-tac", canLaChu(hQtDoiTac))
	mux.HandleFunc("POST /qt/doi-tac/luu", canLaChu(hQtDoiTacLuu))
	mux.HandleFunc("POST /qt/doi-tac/xoa", canLaChu(hQtDoiTacXoa))

	// Hồ sơ khách: công nợ và tổng chi tiêu là chuyện tiền, cùng nhóm với sổ
	// tiền — chỉ chủ vào được. Thợ vẫn thấy gợi ý khách quen lúc lập đơn.
	mux.HandleFunc("GET /qt/khach", canLaChu(hQtKhach))
	mux.HandleFunc("GET /qt/khach.csv", canLaChu(hQtXuatKhach))
	mux.HandleFunc("POST /qt/khach/nhap", canLaChu(hQtKhachNhap))
	mux.HandleFunc("GET /qt/khach/{ma}", canLaChu(hQtKhachChiTiet))
	mux.HandleFunc("POST /qt/khach/{ma}/luu", canLaChu(hQtKhachLuu))
	mux.HandleFunc("POST /qt/don/{ma}/gan-khach", canDangNhap(hQtDonGanKhach))

	// Màn app: ảnh, chữ, khuyến mãi, icon — gom cả về một trang. Mặt tiền,
	// nên cùng luật với giao diện: thợ không bấm được.
	mux.HandleFunc("GET /qt/app", canLaChu(hQtApp))
	mux.HandleFunc("POST /qt/app", canLaChu(hQtApp))
	mux.HandleFunc("POST /qt/app/anh", canLaChu(hQtAppAnh))
	mux.HandleFunc("POST /qt/app/anh-xoa", canLaChu(hQtAppAnhXoa))
	mux.HandleFunc("POST /qt/app/video", canLaChu(hQtAppVideo))
	mux.HandleFunc("POST /qt/app/video-xoa", canLaChu(hQtAppVideoXoa))
	mux.HandleFunc("POST /qt/app/khoi-phuc", canLaChu(hQtAppKhoiPhuc))

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
	dangKyDoNghe(mux)
	dangKyVot(mux)
	dangKyGiay(mux)
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

	// Ba trường dưới đây chỉ có nghĩa với bốn tab của khu "Quản lý đơn"
	// (đơn vợt · đơn giày · yêu cầu web · thống kê). Trang khác để trống,
	// mẫu dải tab không vẽ ra.
	//
	// Ky: kỳ đang xem, đọc từ ?ky=&moc= — xem core/ky.go.
	// SoYeuCauMoi: con số đỏ trên tab Yêu cầu web.
	// GiuLai: mấy tham số lọc khác đang có trên URL, dựng lại thành input
	// ẩn để đổi kỳ không làm rơi bộ lọc đang bật.
	Ky          Ky
	SoYeuCauMoi int
	GiuLai      []OThamSo
}

// OThamSo — một tham số URL cần mang theo qua form khác.
type OThamSo struct{ Ten, Gia string }

// giuLai chép mọi tham số đang có trên URL, trừ mấy cái gọi tên trong bo.
// Sắp theo tên để thứ tự input ẩn không nhảy lung tung giữa hai lần tải —
// diff của người xem mã nguồn trang sẽ sạch hơn, và test so chuỗi được.
func giuLai(r *http.Request, bo ...string) []OThamSo {
	var ra []OThamSo
	for ten, gia := range r.URL.Query() {
		if len(gia) == 0 || strings.TrimSpace(gia[0]) == "" {
			continue
		}
		if ten == "ok" || ten == "loi" {
			continue // thông báo một lần, đừng dính lại trên URL
		}
		boQua := false
		for _, b := range bo {
			if ten == b {
				boQua = true
				break
			}
		}
		if !boQua {
			ra = append(ra, OThamSo{ten, gia[0]})
		}
	}
	sort.Slice(ra, func(i, j int) bool { return ra[i].Ten < ra[j].Ten })
	return ra
}

// dlDon dựng phần chung cho cả bốn tab. Một chỗ duy nhất đọc kỳ và đếm yêu
// cầu chưa xử lý, để bốn trang không lệch nhau.
func dlDon(r *http.Request, trang string) dlQt {
	return dlQt{
		Chung:       chung(r, trang),
		OK:          r.URL.Query().Get("ok"),
		Loi:         r.URL.Query().Get("loi"),
		Ky:          DocKyTu(r),
		SoYeuCauMoi: demYeuCauChuaXuLy(),
		GiuLai:      giuLai(r, "ky", "moc"),
	}
}

// hQtMatMang — tấm chữ hiện ra khi app quản lý mở lúc không có mạng. Service
// worker giữ sẵn một bản trong kho (xem VO trong pwa.go), nên trang này phải
// tĩnh: không đơn, không số, không tên người đăng nhập. Bản trong kho có thể
// đã nằm đó từ tuần trước, mà chữ ở đây thì tuần nào cũng đúng.
func hQtMatMang(w http.ResponseWriter, r *http.Request) {
	render(w, "qt-matmang.html", chung(r, "mat-mang"))
}

// --- Danh sách đơn ---------------------------------------------------

// hQtDonDs sinh ra handler cho một loại đơn. Vợt và giày là hai trang riêng
// với hai đường dẫn riêng, nhưng việc lọc và đếm giống hệt nhau — khác nhau
// đúng ở mấy cột trong bảng, và chỗ ấy để template lo.
func hQtDonDs(loai string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nd, _ := NguoiDangNhap(r)
		f := bolocTuURL(r, loai, nd)
		ds := donTheoQuyen(nd, f)

		trang := "don-vot"
		if LoaiHopLe(loai) == LoaiGiay {
			trang = "don-giay"
		}
		d := dlDon(r, trang)
		d.TieuDe = "Đơn " + TenLoaiDon(loai)
		render(w, "qt-don.html", struct {
			dlQt
			Loai     string
			LaGiay   bool
			TenLoai  string
			Don      []*Don
			Loc      BoLoc
			DangChay bool
			ConNo    bool
			ConBH    bool
			BoQuen   bool
			ThongKe  ThongKeDon
			Tho      []NguoiDung
			DoiTac   []DoiTac
			TongTien int
			ConNoKy  int
		}{
			dlQt:     d,
			Loai:     LoaiHopLe(loai),
			LaGiay:   LoaiHopLe(loai) == LoaiGiay,
			TenLoai:  TenLoaiDon(loai),
			Don:      ds,
			Loc:      f,
			DangChay: r.URL.Query().Get("dang_chay") == "1",
			ConNo:    r.URL.Query().Get("con_no") == "1",
			ConBH:    r.URL.Query().Get("con_bao_hanh") == "1",
			BoQuen:   r.URL.Query().Get("bo_quen") == "1",
			ThongKe:  LayThongKeDon(loai),
			Tho:      DanhSachNguoiDung(),
			DoiTac:   DanhSachDoiTac(),
			TongTien: tongTienDon(ds),
			ConNoKy:  tongConNo(ds),
		})
	}
}

// bolocTuURL dựng bộ lọc từ query string. Trang danh sách và nút xuất CSV
// gọi chung đúng hàm này: tệp tải về phải khớp từng dòng với cái đang hiện
// trên màn hình, chứ không phải "gần giống".
func bolocTuURL(r *http.Request, loai string, nd NguoiDung) BoLoc {
	q := r.URL.Query()
	k := DocKyTu(r)
	f := BoLoc{
		Loai:      loai,
		TrangThai: q.Get("trang_thai"),
		Tho:       q.Get("tho"),
		DoiTac:    q.Get("doi_tac"),
		Tim:       q.Get("q"),
		// Danh sách lọc theo NGÀY NHẬN đơn. Không theo ngày giao: thợ mở
		// trang này để biết hôm nay nhận gì, còn tiền của ngày nào thì bên
		// tab Thống kê trả lời.
		Tu:  k.Tu,
		Den: k.Den,
	}
	if q.Get("dang_chay") == "1" {
		f.ChiDangChay = true
	}
	// "Sửa xong, chưa thu đủ" không phải một trạng thái, nên nó đi bằng
	// tham số riêng chứ không chen vào ô trạng thái.
	if q.Get("con_no") == "1" {
		f.ChuaThuDu = true
	}
	if q.Get("con_bao_hanh") == "1" {
		f.ConBaoHanh = true
	}
	if q.Get("bo_quen") == "1" {
		f.ChiXongBoQuen = true
	}
	if !nd.LaChu() {
		f.Tho = "" // thợ không được lọc theo thợ khác
	}
	return f
}

func donTheoQuyen(nd NguoiDung, f BoLoc) []*Don {
	ds := LocDon(f)
	if nd.LaChu() {
		return ds
	}
	loc := ds[:0]
	for _, d := range ds {
		if xemDuocDon(nd, d) {
			loc = append(loc, d)
		}
	}
	return loc
}

// tongTienDon / tongConNo — cộng đúng mấy đơn đang hiện trên bảng, để chân
// bảng nói được "mấy đơn này cộng lại bao nhiêu". Cộng chứ không tính giá:
// giá do src/quote.py sinh, ở đây chỉ gom số đã có sẵn trên từng đơn.
func tongTienDon(ds []*Don) int {
	t := 0
	for _, d := range ds {
		t += d.TongTien
	}
	return t
}

func tongConNo(ds []*Don) int {
	t := 0
	for _, d := range ds {
		t += d.ConNo()
	}
	return t
}

// duongDsDon — trang danh sách ứng với một đơn, để nút "← Danh sách" quay về
// đúng chỗ vừa đi ra.
func duongDsDon(d *Don) string {
	if d.LaGiay() {
		return "/qt/don-giay"
	}
	return "/qt/don-vot"
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
	loai := LoaiHopLe(r.FormValue("loai"))
	d := struct {
		dlQt
		Loai     string
		LaGiay   bool
		TenLoai  string
		KieuGiay []struct{ Ma, Ten string }
		DichVu   []DichVu
		Tho      []NguoiDung
		Nguong   Nguong
		Truoc    *Don
		GoiY     []GoiYKhach
	}{
		dlQt:     dlQt{Chung: chung(r, "don-moi")},
		Loai:     loai,
		LaGiay:   loai == LoaiGiay,
		TenLoai:  TenLoaiDon(loai),
		KieuGiay: CacKieuGiay,
		// LenPhieu chứ không DangBan: việc đang tắt với khách thì thợ vẫn
		// tích được — khách quen mang tới tận nơi nhờ làm là chuyện có thật.
		DichVu: DichVuLenPhieu(GiaiDoan),
		Tho:    DanhSachNguoiDung(),
		Nguong: NguongHienTai(),
		GoiY:   DanhSachGoiYKhach(),
	}

	// Vào từ trang hồ sơ khách: điền sẵn ô khách rồi để thợ gõ tiếp phần
	// việc. Dùng lại đường Truoc thay vì thêm một biến nữa — form vốn đã đọc
	// mọi ô khách từ đó khi lưu hụt.
	if ma := r.URL.Query().Get("khach"); ma != "" && r.Method != http.MethodPost {
		if k, co := KhachTheoMa(ma); co {
			d.Truoc = &Don{KhachTen: k.Ten, KhachLienHe: k.LienHe, KhachEmail: k.Email, MaKhach: k.Ma}
		}
	}

	// Mở đơn bảo hành từ một đơn cũ: chép sẵn khách và món, KHÔNG chép dòng
	// tiền — đơn bảo hành mặc định 0đ, có phải thu gì thì thợ tự thêm.
	if ma := r.URL.Query().Get("bao_hanh"); ma != "" && r.Method != http.MethodPost {
		if goc, co := LayDon(strings.ToUpper(ma)); co && !goc.LaDonBaoHanh() {
			d.Truoc = &Don{
				KhachTen: goc.KhachTen, KhachLienHe: goc.KhachLienHe,
				KhachEmail: goc.KhachEmail, MaKhach: goc.MaKhach,
				VotHang: goc.VotHang, VotGiaTri: goc.VotGiaTri,
				GiayHang: goc.GiayHang, GiaySize: goc.GiaySize, GiayKieu: goc.GiayKieu,
				MaDonGoc: goc.Ma,
			}
			d.Loai = LoaiHopLe(goc.Loai)
			d.LaGiay = d.Loai == LoaiGiay
			d.TenLoai = TenLoaiDon(d.Loai)
		}
	}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
		if err := r.ParseForm(); err != nil {
			d.Loi = "Nội dung quá dài."
			render(w, "qt-don-moi.html", d)
			return
		}
		don := &Don{
			Ma:          MaDonMoi(loai),
			Loai:        loai,
			Token:       maNgauNhien(8),
			Ngay:        time.Now().Format("2006-01-02"),
			TrangThai:   TTMoi,
			ThoPhuTrach: nd.Ten,
		}
		if nd.LaChu() && r.FormValue("tho") != "" {
			don.ThoPhuTrach = r.FormValue("tho")
		}
		// Mã gốc chỉ nhận lúc TẠO đơn, không nhận ở form sửa: đổi gốc của
		// một đơn đã nằm trong thống kê là làm lệch tỷ lệ bảo hành của kỳ đã
		// chốt. Gõ sai thì xoá đơn rồi mở lại.
		if g := strings.ToUpper(strings.TrimSpace(r.FormValue("ma_don_goc"))); g != "" {
			if goc, co := LayDon(g); co && !goc.LaDonBaoHanh() {
				don.MaDonGoc = goc.Ma
			}
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
			if err := ghiThuChoDon(don, coc, r.FormValue("phuong_thuc"), "Khách đưa lúc nhận "+don.TenLoai(), nd); err != nil {
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
	// Email gõ sai thì lưu chuỗi rỗng chứ không lưu rác: cả trang chi tiết
	// lẫn nút gửi mail đều đọc trường này để quyết định có gửi được hay
	// không, mà một chuỗi "abc" thì Resend nhận rồi trả lỗi lặng lẽ.
	if e := catBot(r.FormValue("khach_email"), 254); strings.TrimSpace(e) == "" || HopLeEmail(e) {
		don.KhachEmail = strings.TrimSpace(e)
	}
	don.TinhTrang = catBot(r.FormValue("tinh_trang"), 3000)
	don.ChanDoan = catBot(r.FormValue("chan_doan"), 3000)
	// Trường của loại kia không đọc tới: form giày không gửi ô cân, mà đọc một
	// ô không có trong form thì được chuỗi rỗng — tức là xoá trắng số cân của
	// đơn vợt nếu có ngày ai đó bấm nhầm.
	if don.LaGiay() {
		don.GiayHang = catBot(r.FormValue("giay_hang"), 100)
		don.GiaySize = catBot(r.FormValue("giay_size"), 20)
		if k := r.FormValue("giay_kieu"); TenKieuGiay(k) != "" {
			don.GiayKieu = k
		}
		don.DeHienTai = catBot(r.FormValue("de_hien_tai"), 200)
	} else {
		don.VotHang = catBot(r.FormValue("vot_hang"), 100)
		don.VotGiaTri = soTien(r.FormValue("vot_gia_tri"))
		don.CanTruocG = soThuc(r.FormValue("can_truoc"))
		don.CanSauG = soThuc(r.FormValue("can_sau"))
	}
	don.KenhNhan = r.FormValue("kenh_nhan")
	don.HenTraNgay = ngayISO(r.FormValue("hen_tra"))
	don.BaoHanhDen = ngayISO(r.FormValue("bao_hanh_den"))
	// Ô trống thì máy điền theo dịch vụ đã chốt; gõ tay thì thôi. Gọi ở đây
	// chứ không ở chỗ lưu đơn để cả tạo mới lẫn sửa đơn đi cùng một luật —
	// xem đầu core/baohanh.go.
	ApBaoHanhMacDinh(don)
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

	// Hồ sơ khách sinh ra từ đơn, không bắt thợ mở thêm một trang nữa — xem
	// đầu core/khachhang.go. Không có số điện thoại thì GhiNhanKhach trả rỗng
	// và đơn giữ nguyên mã cũ chứ không bị xoá trắng.
	if ma := GhiNhanKhach(don.KhachTen, don.KhachLienHe, don.KhachEmail, don.KhachDiaChi); ma != "" {
		don.MaKhach = ma
	}
}

// --- Chi tiết đơn ----------------------------------------------------

func hQtDonChiTiet(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	// Trang nav tô theo loại đơn: mở chi tiết một đơn giày thì mục "Đơn giày"
	// sáng, không phải không mục nào sáng cả.
	c := chung(r, "don-"+LoaiHopLe(don.Loai))
	c.TieuDe = "Đơn " + don.Ma
	phieuXuat, coPhieu := PhieuXuatCuaDon(don.Ma)
	// Hồ sơ khách chỉ để hiện một dòng liên kết. Đơn cũ chưa có MaKhach thì
	// hoSo rỗng và màn hình mọc ra nút gắn — không tự gắn ngầm, vì gộp nhầm
	// hai người trùng số là thứ phải do người quyết.
	var hoSo *KhachHang
	var soLieuKH SoLieuKhach
	if don.MaKhach != "" {
		if k, co := KhachTheoMa(don.MaKhach); co {
			hoSo = &k
			soLieuKH = SoLieuCuaKhach(k.Ma)
		}
	}

	// Điền sẵn ô giảm giá, không tự áp. Chỉ điền khi ô đang trống — đơn đã
	// lưu một mức giảm rồi thì không đè lên. Xem GoiYGiamGia.
	var goiGiam, goiGiamHa int
	var goiGiamLy string
	if don.GiamGia() == 0 {
		var daHa bool
		goiGiam, goiGiamLy, daHa = GoiYGiamGia(don)
		if daHa {
			// Mức đầy đủ trước khi trần biên gộp kéo xuống, để nói ra con số
			// đã bị cắt bớt bao nhiêu.
			goiGiamHa = (don.TongTien*MucGiamCuaKhach(don.MaKhach) + 50) / 100
		}
	}

	// Bảo hành hai chiều: đơn gốc kể ra mình đã phải làm lại mấy lần, đơn
	// bảo hành chỉ ngược về gốc. Nhìn đơn nào cũng thấy được chuyện.
	var donGoc *Don
	if don.LaDonBaoHanh() {
		if g, co := LayDon(don.MaDonGoc); co {
			donGoc = g
		}
	}

	render(w, "qt-don-ct.html", struct {
		dlQt
		Don       *Don
		DichVu    []DichVu
		Tho       []NguoiDung
		DoiTac    []DoiTac
		KieuGiay  []struct{ Ma, Ten string }
		DuongDs   string
		Nguong    Nguong
		CacTT     []MoTaTrangThai
		DaChonDV  map[string]bool
		DongThem  []DongTien
		LanThu    []Khoan
		LanHoan   []Khoan
		PhieuXuat *Phieu
		CoPhieu   bool
		DeXuatVT  []DongPhieu
		HoSo      *KhachHang
		SoLieuKH  SoLieuKhach
		GoiGiam   int
		GoiGiamLy string
		GoiGiamHa int
		DonGoc    *Don
		DonBH     []*Don
	}{
		dlQt:      dlQt{Chung: c, OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		Don:       don,
		DichVu:    DichVuLenPhieu(GiaiDoan), // xem chú thích ở hQtDonMoi
		Tho:       DanhSachNguoiDung(),
		DoiTac:    DoiTacDangDung(),
		KieuGiay:  CacKieuGiay,
		DuongDs:   duongDsDon(don),
		Nguong:    NguongHienTai(),
		CacTT:     CacTrangThai,
		DaChonDV:  chonDV(don),
		DongThem:  dongTay(don),
		LanThu:    CacLanThuCuaDon(don.Ma),
		LanHoan:   CacLanHoanCuaDon(don.Ma),
		PhieuXuat: phieuXuat,
		CoPhieu:   coPhieu,
		DeXuatVT:  DeXuatXuatChoDon(don),
		HoSo:      hoSo,
		SoLieuKH:  soLieuKH,
		DonGoc:    donGoc,
		DonBH:     DonBaoHanhCua(don.Ma),
		GoiGiam:   goiGiam,
		GoiGiamLy: goiGiamLy,
		GoiGiamHa: goiGiamHa,
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

// hQtGuiDi lưu khối "gửi đi gia công". Hai con số tiền của đơn — khách trả
// bao nhiêu, mình trả tiệm bao nhiêu — đều do người gõ; ở đây không nhân chia
// gì cả, xem ghi chú đầu donhang.go.
func hQtGuiDi(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Nội dung quá dài", http.StatusRequestEntityTooLarge)
		return
	}
	ma := r.FormValue("ma_doi_tac")
	if ma != "" {
		if _, co := TimDoiTac(ma); !co {
			http.Error(w, "không có đối tác này", http.StatusBadRequest)
			return
		}
	}
	// Bỏ chọn đối tác là huỷ cả khối. Nhưng đã trả tiền tiệm rồi thì không:
	// khoản chi đang nằm trong sổ, xoá mã đơn đi là khoản ấy mồ côi.
	if ma == "" && don.GuiDi.DaTra {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Đã trả tiền tiệm rồi thì không bỏ đối tác được — bỏ tick \"đã trả\" trước."), http.StatusSeeOther)
		return
	}
	g := don.GuiDi
	g.MaDoiTac = ma
	g.NgayGui = ngayISO(r.FormValue("ngay_gui"))
	g.NgayHenVe = ngayISO(r.FormValue("ngay_hen_ve"))
	g.NgayVe = ngayISO(r.FormValue("ngay_ve"))
	g.TraDoiTac = soTien(r.FormValue("tra_doi_tac"))
	g.GhiChu = catBot(r.FormValue("gui_ghi_chu"), 500)
	if ma == "" {
		g = GuiDi{}
	}
	don.GuiDi = g
	if err := LuuDon(don); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok=Đã+lưu+phần+gửi+đi", http.StatusSeeOther)
}

// hQtTraDoiTac bật/tắt "đã trả tiền tiệm" và ghi thẳng vào sổ.
//
// Không ghi vào sổ thì lãi lỗ tháng nói dối theo đúng hướng nguy hiểm nhất:
// báo lãi cao hơn thật. Mã khoản lưu lại trong đơn nên bấm hai lần không sinh
// hai dòng, và bỏ tick thì xoá đúng khoản ấy chứ không phải khoản gần giống.
func hQtTraDoiTac(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	bat := r.FormValue("da_tra") == "1"

	switch {
	case bat && !don.GuiDi.DaTra:
		if don.GuiDi.MaDoiTac == "" {
			http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Chọn tiệm và điền tiền trả tiệm trước đã."), http.StatusSeeOther)
			return
		}
		if don.GuiDi.TraDoiTac <= 0 {
			http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Chưa điền tiền trả tiệm — không ghi vào sổ được."), http.StatusSeeOther)
			return
		}
		ngay := don.GuiDi.NgayVe
		if ngay == "" {
			ngay = time.Now().Format("2006-01-02")
		}
		pt := r.FormValue("phuong_thuc")
		if pt != TraChuyenKhoan {
			pt = TraTienMat
		}
		maKhoan := MaKhoanMoi(ngay)
		err := LuuKhoan(Khoan{
			Ma:         maKhoan,
			Ngay:       ngay,
			Loai:       KhoanChi,
			Nhom:       NhomVanHanh,
			SoTien:     don.GuiDi.TraDoiTac,
			DienGiai:   "Trả tiệm gia công " + TenDoiTac(don.GuiDi.MaDoiTac) + " — đơn " + don.Ma,
			PhuongThuc: pt,
			MaDon:      don.Ma,
			Nguoi:      nd.TenHienThi(),
		})
		if err != nil {
			http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Chưa ghi được vào sổ: "+err.Error()), http.StatusSeeOther)
			return
		}
		don.GuiDi.DaTra = true
		don.GuiDi.MaKhoanChi = maKhoan

	case !bat && don.GuiDi.DaTra:
		if don.GuiDi.MaKhoanChi != "" {
			if err := XoaKhoan(don.GuiDi.MaKhoanChi); err != nil {
				// Khoản đã bị xoá tay ở sổ tiền: vẫn cho bỏ tick, nếu không đơn
				// kẹt vĩnh viễn ở trạng thái "đã trả" mà sổ không có dòng nào.
				GhiMoc(don, don.TrangThai, nd.TenHienThi(), "Bỏ đánh dấu đã trả tiệm; khoản chi "+don.GuiDi.MaKhoanChi+" không còn trong sổ")
			}
		}
		don.GuiDi.DaTra = false
		don.GuiDi.MaKhoanChi = ""

	default:
		http.Redirect(w, r, "/qt/don/"+don.Ma, http.StatusSeeOther)
		return
	}

	if err := LuuDon(don); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok=Đã+cập+nhật+tiền+trả+tiệm", http.StatusSeeOther)
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
	// Tới lúc này đơn mới có ngày giao để đếm bảo hành từ đó.
	ApBaoHanhMacDinh(don)
	if err := LuuDon(don); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Giao xong là một lần ghé đã trọn. Gọi sau LuuDon chứ không trước: hồ
	// sơ khách lên nhóm mà đơn không lưu được là hai sổ nói hai chuyện.
	ThangNhomSauGiao(don)
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape(nhacSauDoiTrangThai(don, tt)), http.StatusSeeOther)
}

// hQtNhanHang — kiện của đơn đặt online đã tới tay trạm. Một cú bấm, và từ
// đây đơn chảy đúng luồng cũ.
//
// Chỉ chạy ở cho_hang_ve. Bấm nhầm trên đơn đang sửa mà kéo được nó về "mới
// nhận" thì mất sạch mốc lịch sử của đoạn giữa.
func hQtNhanHang(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	if !don.ChoHangVe() {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?loi="+urlEsc("Đơn này không ở trạng thái chờ hàng về"), http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	// Mã vận đơn chỉ ghi đè khi thợ thật sự gõ: khách đã báo mã lúc gửi thì
	// ô trống trên form quản trị không được xoá mất nó.
	if mvd := catBot(strings.TrimSpace(r.FormValue("ma_van_don")), 60); mvd != "" {
		don.MaVanDonDen = mvd
	}
	don.TrangThai = TTMoi
	GhiMoc(don, TTMoi, nd.TenHienThi(), "Đã nhận kiện hàng")
	if err := LuuDon(don); err != nil {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?loi="+urlEsc("Không lưu được"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+urlEsc("Đã nhận hàng"), http.StatusSeeOther)
}

// hQtVanDonVe — thợ gửi đồ về, ghi lại mã để khách theo dõi. Không đổi trạng
// thái: gửi đi và giao xong là hai chuyện, máy trạng thái cũ đã có TTDaGiao.
func hQtVanDonVe(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	don.MaVanDonVe = catBot(strings.TrimSpace(r.FormValue("ma_van_don_ve")), 60)
	if err := LuuDon(don); err != nil {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?loi="+urlEsc("Không lưu được"), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+urlEsc("Đã ghi vận đơn về"), http.StatusSeeOther)
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

	// Vòng lặp lọc và ghi tệp nằm ở core/anhdon.go — cửa hàng dùng chung.
	luuAnhVaoDon(don, r.MultipartForm.File["anh"], r.FormValue("nhan"))
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
	k := DocKyTu(r)
	var ds []yeuCauKhach
	for _, y := range docYeuCau(300) {
		if !trongKhoang(y.Ngay, k.Tu, k.Den) {
			continue
		}
		ds = append(ds, y)
		if len(ds) >= 100 {
			break
		}
	}
	render(w, "qt-yeucau.html", struct {
		dlQt
		YeuCau []yeuCauKhach
	}{dlDon(r, "yeu-cau"), ds})
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
	// Biểu mẫu ngoài trang chỉ hỏi hãng vợt và tình trạng, nên máy không đoán
	// được khách gửi vợt hay giày — thợ nhìn ảnh rồi bấm đúng nút. Rỗng hoặc
	// bậy thì LoaiHopLe trả về vợt, đúng với phần lớn yêu cầu.
	loai := LoaiHopLe(r.FormValue("loai"))
	don := &Don{
		Ma:    MaDonMoi(loai),
		Loai:  loai,
		Token: maNgauNhien(8),
		Ngay:  time.Now().Format("2006-01-02"),
		// Khám qua ẢNH, vì vợt vẫn đang ở nhà khách — xem chú thích TTKhamAnh
		// trong donhang.go. Không phải TTMoi: "mới nhận" nghĩa là đồ đã nằm
		// trên bàn, mà ở đây chưa có gì trên bàn cả.
		TrangThai:   TTKhamAnh,
		ThoPhuTrach: nd.Ten,
		KhachTen:    yc.Ten,
		KhachLienHe: yc.LienHe,
		KhachEmail:  strings.TrimSpace(yc.Email),
		TinhTrang:   yc.MoTa,
		KenhNhan:    "web",
		NguonDon:    NguonWeb,
	}
	if loai == LoaiGiay {
		don.GiayHang = yc.VotHang // ô "hãng" ngoài web dùng chung cho cả hai
	} else {
		don.VotHang = yc.VotHang
	}
	// Khách gửi ảnh qua web cũng là khách — dựng hồ sơ ngay từ đây.
	don.MaKhach = GhiNhanKhach(don.KhachTen, don.KhachLienHe, don.KhachEmail, "")
	GhiMoc(don, TTKhamAnh, nd.TenHienThi(), "Tạo từ yêu cầu web "+yc.Ma)
	if err := LuuDon(don); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Chép ảnh khách gửi sang đơn. Không chép thì thợ phải mở hai tab để vừa
	// nhìn chỗ nứt vừa gõ chẩn đoán, và ngày nào đó dọn thư mục yêu cầu là
	// đơn mất sạch ảnh gốc.
	chepTepYeuCauSangDon(yc, don)
	yc.DaXuLy = true
	yc.MaDon = don.Ma
	luuYeuCau(yc)
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+urlEsc("Đã nhận khám. Ghi kết quả rồi gửi báo giá cho khách."), http.StatusSeeOther)
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
	k := DocKyTu(r)
	loai := locLoaiTuy(r.URL.Query().Get("loai"))
	tk := LayThongKeKy(loai, k)
	d := dlDon(r, "thong-ke")
	d.TieuDe = "Thống kê " + k.Ten
	render(w, "qt-thongke.html", struct {
		dlQt
		TK      ThongKeKy
		Loai    string
		Nay     ThongKeDon
		DinhPhi int
		CaoNhat int
	}{
		dlQt: d,
		TK:   tk,
		Loai: loai,
		// Nay là bảng "ngay lúc này" — đang trên bàn, quá hẹn trả. Mấy con số
		// ấy không thuộc kỳ nào cả: vợt đang trễ hẹn thì trễ bất kể đang xem
		// tháng nào, nên vẫn lấy từ hàm cũ.
		Nay:     LayThongKeDon(loai),
		DinhPhi: dinhPhiTheoKy(k),
		CaoNhat: CaoNhatKy(tk.Dong),
	})
}

// locLoaiTuy — khác LoaiHopLe ở chỗ chuỗi rỗng vẫn là rỗng, nghĩa là "cả vợt
// lẫn giày". LoaiHopLe đổi rỗng thành vợt, đúng cho trang đơn nhưng sai ở
// đây: thống kê mặc định phải gộp cả hai nghề.
func locLoaiTuy(l string) string {
	if l == LoaiVot || l == LoaiGiay {
		return l
	}
	return ""
}

// dinhPhiTheoKy — định phí khai trong bảng giá là con số MỘT THÁNG. Nhân/chia
// cho đúng kỳ đang xem, để dòng "tháng nào ăn vào vốn" không so nhầm doanh
// thu một ngày với chi phí cả tháng. Kỳ "từ trước tới nay" trả 0: không biết
// trạm mở bao lâu thì không nói bừa.
func dinhPhiTheoKy(k Ky) int {
	switch k.Kieu {
	case KyNgay:
		return GIA.DinhPhiThang / 30
	case KyThang:
		return GIA.DinhPhiThang
	case KyNam:
		return GIA.DinhPhiThang * 12
	}
	return 0
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

// --- Logo và biểu tượng ----------------------------------------------

// Trang này từng là trang chọn giao diện, kèm quản lý logo ở cuối. Bỏ hệ giao
// diện rồi thì phần logo vẫn phải ở đâu đó, và đường dẫn /qt/giao-dien/logo
// đang nằm trong danh sách miễn CSRF theo tên (core/csrf.go) — đổi tên đường
// dẫn là phải sửa cả chỗ đó, nên giữ nguyên tên cũ, chỉ đổi cái hiện ra.
type dlGiaoDien struct {
	dlQt
	// Tên tệp rỗng = đang dùng bản nhúng trong code; template lấy đó làm
	// mốc để hiện hay giấu nút bỏ tệp.
	LogoTen string
	LogoURL string
	IconTen string
	IconURL string

	// CheDo là công tắc NỘI DUNG, không phải giao diện màu. Nó ở trang này vì
	// đây là chỗ Kendy quen bấm khi muốn đổi bộ mặt của site.
	CheDo       string
	LaOnline    bool
	ThieuDiaChi bool

	NoiBatBat bool // khung nổi bật cho việc đã tích ở /qt/dich-vu
	SoNoiBat  int  // đã tích mấy việc — bật công tắc mà chưa tích thì không đổi gì
}

func hQtGiaoDien(w http.ResponseWriter, r *http.Request) {
	d := dlGiaoDien{dlQt: dlQt{Chung: chung(r, "giao-dien")}}
	// Việc tải logo quay về đây bằng redirect (POST xong không để lại form
	// trong lịch sử), nên lời báo tới qua thanh địa chỉ.
	d.OK = r.URL.Query().Get("ok")
	d.Loi = r.URL.Query().Get("loi")
	d.LogoTen, _ = tepLogo(loaiLogo)
	d.LogoURL = logoURL()
	d.IconTen, _ = tepLogo(loaiIcon)
	d.IconURL = iconURL()
	d.CheDo = CheDoHienTai()
	d.LaOnline = LaOnline()
	// Bật Online mà chưa có địa chỉ thì cửa hàng vẫn đóng — nói ngay ở đây,
	// đừng để khách phát hiện hộ.
	d.ThieuDiaChi = strings.TrimSpace(LienHeHienTai().DiaChi) == ""
	d.NoiBatBat = DvNoiBatBat()
	for _, dv := range DichVuTatCa() {
		if dv.NoiBat {
			d.SoNoiBat++
		}
	}
	render(w, "qt-giaodien.html", d)
}

// hQtCheDo đổi công tắc Tại xưởng / Online. Mọi lối ra đều là redirect kèm
// lời báo: POST xong không để lại form trong lịch sử trình duyệt.
func hQtCheDo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	ma := r.FormValue("che_do")
	if !CheDoHopLe(ma) {
		http.Redirect(w, r, "/qt/giao-dien?loi="+urlEsc("Chế độ không hợp lệ"), http.StatusSeeOther)
		return
	}
	if err := DatCheDo(ma); err != nil {
		http.Redirect(w, r, "/qt/giao-dien?loi="+urlEsc("Không ghi được: "+err.Error()), http.StatusSeeOther)
		return
	}
	ten := "Tại xưởng"
	if ma == CheDoOnline {
		ten = "Online"
	}
	http.Redirect(w, r, "/qt/giao-dien?ok="+urlEsc("Đã chuyển sang bản "+ten), http.StatusSeeOther)
}

// hQtNoiBat bật/tắt khung nổi bật. Ô tích không gửi gì khi bỏ tích, nên cứ
// không thấy khoá là tắt — không cần giá trị "0".
func hQtNoiBat(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	bat := r.FormValue("noi_bat") != ""
	if err := DatDvNoiBat(bat); err != nil {
		http.Redirect(w, r, "/qt/giao-dien?loi="+urlEsc("Không ghi được: "+err.Error()), http.StatusSeeOther)
		return
	}
	loi := "Đã tắt khung nổi bật"
	if bat {
		loi = "Đã bật khung nổi bật"
	}
	http.Redirect(w, r, "/qt/giao-dien?ok="+urlEsc(loi), http.StatusSeeOther)
}

func urlEsc(s string) string {
	return strings.NewReplacer(" ", "+", "&", "%26", "#", "%23", "?", "%3F").Replace(s)
}
