// Trang sổ tiền: /qt/tien
//
// Chỉ chủ vào được. Thợ nhận tiền của khách thì ghi ngay ở trang đơn
// (/qt/don/{ma}/thu) — đó là việc của thợ; còn nhìn tổng lãi lỗ và đầu tư thì
// không.
package core

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

func itoa(n int) string { return strconv.Itoa(n) }

func dangKySoTien(mux *http.ServeMux) {
	mux.HandleFunc("GET /qt/tien", canLaChu(hQtTien))
	mux.HandleFunc("POST /qt/tien/them", canLaChu(hQtTienThem))
	mux.HandleFunc("POST /qt/tien/xoa", canLaChu(hQtTienXoa))
	mux.HandleFunc("POST /qt/tien/duyet", canLaChu(hQtTienDuyet))
	mux.HandleFunc("GET /qt/tien/dinh-ky", canLaChu(hQtDinhKy))
	mux.HandleFunc("POST /qt/tien/dinh-ky", canLaChu(hQtDinhKyLuu))
	mux.HandleFunc("GET /qt/tien/sao-ke", canLaChu(hQtSaoKe))
	mux.HandleFunc("POST /qt/tien/sao-ke", canLaChu(hQtSaoKe))
	mux.HandleFunc("POST /qt/tien/sao-ke/ghi", canLaChu(hQtSaoKeGhi))

	// Ghi tiền khách trả: thợ cũng làm được, trên chính đơn mình đang cầm.
	mux.HandleFunc("POST /qt/don/{ma}/thu", canDangNhap(hQtGhiThu))
	mux.HandleFunc("POST /qt/don/{ma}/hoan", canDangNhap(hQtHoanTien))
	mux.HandleFunc("POST /qt/don/{ma}/thu-xoa", canDangNhap(hQtXoaThu))
}

// --- Trang chính ------------------------------------------------------

func hQtTien(w http.ResponseWriter, r *http.Request) {
	thang := r.URL.Query().Get("thang")
	if !thangHopLe(thang) {
		thang = time.Now().Format("2006-01")
	}
	ds := KhoanTrongThang(thang)
	// Mới nhất lên đầu: cái vừa gõ xong phải thấy ngay, không phải cuộn.
	for i, j := 0, len(ds)-1; i < j; i, j = i+1, j-1 {
		ds[i], ds[j] = ds[j], ds[i]
	}

	c := chung(r, "tien")
	render(w, "qt-tien.html", struct {
		dlQt
		Thang    string
		DsThang  []string
		Khoan    []Khoan
		TongQuan TongQuanTien
		BaoCao   []ThangTien
		NhomThu  []MoTaNhom
		NhomChi  []MoTaNhom
		ChoDuyet []Khoan
		HomNay   string
	}{
		dlQt:     dlQt{Chung: c, OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		Thang:    thang,
		DsThang:  thangGanDay(thang),
		Khoan:    ds,
		TongQuan: LayTongQuanTien(),
		BaoCao:   BaoCaoThang(),
		NhomThu:  NhomTheoLoai(KhoanThu),
		NhomChi:  NhomTheoLoai(KhoanChi),
		ChoDuyet: KhoanChoDuyet(),
		HomNay:   time.Now().Format("2006-01-02"),
	})
}

// thangGanDay — các tháng đã có số, cộng thêm tháng này và tháng đang xem dù
// còn rỗng, để chọn được tháng mới mà chưa ghi gì.
func thangGanDay(dangXem string) []string {
	co := map[string]bool{}
	out := []string{}
	them := func(t string) {
		if t != "" && !co[t] {
			co[t] = true
			out = append(out, t)
		}
	}
	them(time.Now().Format("2006-01"))
	them(dangXem)
	for i := len(CacThangCoSo()) - 1; i >= 0; i-- {
		them(CacThangCoSo()[i])
	}
	return out
}

func veTien(thang, thongBao string, loi bool) string {
	k := "ok"
	if loi {
		k = "loi"
	}
	return "/qt/tien?thang=" + url.QueryEscape(thang) + "&" + k + "=" + url.QueryEscape(thongBao)
}

func hQtTienThem(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	r.ParseForm()

	k := Khoan{
		Ma:         strings.TrimSpace(r.FormValue("ma")), // rỗng = thêm mới
		Ngay:       ngayISO(r.FormValue("ngay")),
		Loai:       r.FormValue("loai"),
		Nhom:       r.FormValue("nhom"),
		SoTien:     soTien(r.FormValue("so_tien")),
		DienGiai:   catBot(r.FormValue("dien_giai"), 200),
		PhuongThuc: r.FormValue("phuong_thuc"),
		MaDon:      strings.ToUpper(strings.TrimSpace(r.FormValue("ma_don"))),
		Nguoi:      nd.TenHienThi(),
	}
	if k.Ngay == "" {
		k.Ngay = time.Now().Format("2006-01-02")
	}
	if k.MaDon != "" {
		if _, co := LayDon(k.MaDon); !co {
			http.Redirect(w, r, veTien(k.Thang(), "Không có đơn "+k.MaDon, true), http.StatusSeeOther)
			return
		}
	}
	if err := LuuKhoan(k); err != nil {
		http.Redirect(w, r, veTien(k.Thang(), "Không ghi được: "+err.Error(), true), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, veTien(k.Thang(), "Đã ghi vào sổ", false), http.StatusSeeOther)
}

func hQtTienXoa(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ma := r.FormValue("ma")
	k, co := LayKhoan(ma)
	if !co {
		http.NotFound(w, r)
		return
	}
	// Khoản sinh từ phiếu nhập kho phải xoá ở phiếu, không xoá ở đây: xoá
	// riêng khoản chi thì phiếu nhập còn đó, kho vẫn cộng hàng vào mà tiền
	// mua hàng biến mất khỏi sổ — lãi tháng đó tự nhiên đẹp lên.
	if k.MaPhieu != "" {
		http.Redirect(w, r, veTien(k.Thang(), "Khoản này thuộc phiếu "+k.MaPhieu+", sửa ở trang kho", true), http.StatusSeeOther)
		return
	}
	if err := XoaKhoan(ma); err != nil {
		http.Redirect(w, r, veTien(k.Thang(), err.Error(), true), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, veTien(k.Thang(), "Đã xoá khoản "+ma, false), http.StatusSeeOther)
}

func hQtTienDuyet(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	ma := r.FormValue("ma")
	k, co := LayKhoan(ma)
	if !co {
		http.NotFound(w, r)
		return
	}
	// Cho sửa lại số tiền ngay lúc duyệt: hoá đơn điện tháng này 412k chứ
	// không phải 400k như khai định kỳ là chuyện bình thường.
	if n := soTien(r.FormValue("so_tien")); n > 0 {
		k.SoTien = n
	}
	if ng := ngayISO(r.FormValue("ngay")); ng != "" {
		k.Ngay = ng
	}
	k.ChoDuyet = false
	if err := LuuKhoan(k); err != nil {
		http.Redirect(w, r, veTien(k.Thang(), err.Error(), true), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, veTien(k.Thang(), "Đã xác nhận "+k.DienGiai, false), http.StatusSeeOther)
}

// --- Khoản định kỳ ----------------------------------------------------

func hQtDinhKy(w http.ResponseWriter, r *http.Request) {
	render(w, "qt-tien-dinhky.html", struct {
		dlQt
		DinhKy   []KhoanDinhKy
		NhomChi  []MoTaNhom
		ThangNay string
	}{
		dlQt:     dlQt{Chung: chung(r, "tien-dinh-ky"), OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		DinhKy:   DanhSachDinhKy(),
		NhomChi:  NhomTheoLoai(KhoanChi),
		ThangNay: time.Now().Format("2006-01"),
	})
}

func hQtDinhKyLuu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	r.ParseForm()

	ten := r.Form["ten"]
	ds := make([]KhoanDinhKy, 0, len(ten))
	for i := range ten {
		ds = append(ds, KhoanDinhKy{
			Ma:             lay(r.Form["ma"], i),
			Ten:            catBot(ten[i], 120),
			Nhom:           lay(r.Form["nhom"], i),
			SoTien:         soTien(lay(r.Form["so_tien"], i)),
			NgayTrongThang: soTien(lay(r.Form["ngay_trong_thang"], i)),
			PhuongThuc:     lay(r.Form["phuong_thuc"], i),
			GhiChu:         catBot(lay(r.Form["ghi_chu"], i), 200),
			Bat:            lay(r.Form["bat"], i) == "1",
		})
	}
	if err := LuuDinhKy(ds); err != nil {
		http.Redirect(w, r, "/qt/tien/dinh-ky?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}

	tb := "Đã lưu"
	if r.FormValue("sinh_ngay") == "1" {
		n, err := SinhKhoanDinhKy(time.Now().Format("2006-01"))
		switch {
		case err != nil:
			tb = "Đã lưu, nhưng không dựng được khoản: " + err.Error()
		case n > 0:
			tb = "Đã lưu và dựng " + itoa(n) + " khoản chờ xác nhận cho tháng này"
		default:
			tb = "Đã lưu. Tháng này đã có đủ khoản định kỳ rồi"
		}
	}
	http.Redirect(w, r, "/qt/tien/dinh-ky?ok="+url.QueryEscape(tb), http.StatusSeeOther)
}

func lay(ds []string, i int) string {
	if i < len(ds) {
		return ds[i]
	}
	return ""
}

// --- Sao kê ------------------------------------------------------------

func hQtSaoKe(w http.ResponseWriter, r *http.Request) {
	d := struct {
		dlQt
		Text string
		Dong []DongSaoKe
		SoDe int
	}{dlQt: dlQt{Chung: chung(r, "tien-sao-ke"), OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")}}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 512*1024)
		if err := r.ParseForm(); err != nil {
			d.Loi = "Đoạn dán quá dài, cắt bớt rồi dán làm nhiều lần."
			render(w, "qt-tien-saoke.html", d)
			return
		}
		d.Text = r.FormValue("sao_ke")
		d.Dong = DocSaoKe(d.Text)
		for _, x := range d.Dong {
			if x.Chon {
				d.SoDe++
			}
		}
		if len(d.Dong) == 0 && strings.TrimSpace(d.Text) != "" {
			d.Loi = "Không đọc được dòng nào có số tiền. Dán cả cột ngày và số tiền vào nhé."
		}
	}
	render(w, "qt-tien-saoke.html", d)
}

func hQtSaoKeGhi(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	r.Body = http.MaxBytesReader(w, r.Body, 512*1024)
	r.ParseForm()

	chon := map[string]bool{}
	for _, i := range r.Form["chon"] {
		chon[i] = true
	}
	ghi, bo := 0, 0
	var loiCuoi string
	var daKhop []string
	tongKhop := 0
	for i := range r.Form["ma_don"] {
		if !chon[itoa(i)] {
			continue
		}
		maDon := strings.ToUpper(strings.TrimSpace(lay(r.Form["ma_don"], i)))
		st := soTien(lay(r.Form["so_tien"], i))
		ngay := ngayISO(lay(r.Form["ngay"], i))
		don, co := LayDon(maDon)
		if !co || st <= 0 {
			bo++
			continue
		}
		if ngay == "" {
			ngay = time.Now().Format("2006-01-02")
		}
		err := LuuKhoan(Khoan{
			Ngay:       ngay,
			Loai:       KhoanThu,
			Nhom:       NhomSuaChua,
			SoTien:     st,
			DienGiai:   "Khách trả — đơn " + don.Ma + " " + don.KhachTen,
			PhuongThuc: TraChuyenKhoan,
			MaDon:      don.Ma,
			Nguoi:      nd.TenHienThi(),
		})
		if err != nil {
			loiCuoi = err.Error()
			bo++
			continue
		}
		ghi++
		daKhop = append(daKhop, don.Ma+" · "+dinhDangTien(st)+" · "+gonMotDong(don.KhachTen))
		tongKhop += st
	}
	// Báo sau khi đã ghi xong cả lô: một tin cho cả lần đối soát, không phải
	// một tin cho mỗi dòng sao kê.
	BaoKhachChuyenKhoan(daKhop, tongKhop)

	tb := "Đã ghi " + itoa(ghi) + " khoản thu"
	if bo > 0 {
		tb += ", bỏ qua " + itoa(bo) + " dòng"
		if loiCuoi != "" {
			tb += " (" + loiCuoi + ")"
		}
	}
	http.Redirect(w, r, "/qt/tien/sao-ke?ok="+url.QueryEscape(tb), http.StatusSeeOther)
}

// --- Ghi thu ngay trên đơn --------------------------------------------

func hQtGhiThu(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()

	st := soTien(r.FormValue("so_tien"))
	if st <= 0 {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok=Chưa+nhập+số+tiền", http.StatusSeeOther)
		return
	}
	err := ghiThuChoDon(don, st, r.FormValue("phuong_thuc"), catBot(r.FormValue("ghi_chu"), 120), nd)
	if err != nil {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok=Không+ghi+được:+"+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Đã ghi thu "+dinhDangTien(st)), http.StatusSeeOther)
}

func ghiThuChoDon(don *Don, st int, phuongThuc, ghiChu string, nd NguoiDung) error {
	if phuongThuc != TraChuyenKhoan {
		phuongThuc = TraTienMat
	}
	dg := "Khách trả — đơn " + don.Ma
	if strings.TrimSpace(don.KhachTen) != "" {
		dg += " " + strings.TrimSpace(don.KhachTen)
	}
	if strings.TrimSpace(ghiChu) != "" {
		dg += " (" + strings.TrimSpace(ghiChu) + ")"
	}
	return LuuKhoan(Khoan{
		Ngay:       time.Now().Format("2006-01-02"),
		Loai:       KhoanThu,
		Nhom:       NhomSuaChua,
		SoTien:     st,
		DienGiai:   dg,
		PhuongThuc: phuongThuc,
		MaDon:      don.Ma,
		Nguoi:      nd.TenHienThi(),
	})
}

// hQtHoanTien — khách huỷ sau khi đã trả tiền, hoặc trạm làm hỏng phải đền.
//
// Không giới hạn theo trạng thái đơn: đơn đã giao vẫn hoàn được, vì đời thật
// khách gọi lại sau khi cầm vợt về là chuyện thường. Chỉ chặn đúng một thứ —
// hoàn quá số đã thu, vì đó chắc chắn là gõ nhầm chứ không phải ý định.
func hQtHoanTien(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()

	st := soTien(r.FormValue("so_tien"))
	if st <= 0 {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok=Chưa+nhập+số+tiền+hoàn", http.StatusSeeOther)
		return
	}
	if con := don.DaThu(); st > con {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Đơn này mới thu "+dinhDangTien(con)+", không hoàn nhiều hơn được"), http.StatusSeeOther)
		return
	}

	pt := r.FormValue("phuong_thuc")
	if pt != TraChuyenKhoan {
		pt = TraTienMat
	}
	dg := "Hoàn tiền khách — đơn " + don.Ma
	if t := strings.TrimSpace(don.KhachTen); t != "" {
		dg += " " + t
	}
	if g := strings.TrimSpace(catBot(r.FormValue("ghi_chu"), 120)); g != "" {
		dg += " (" + g + ")"
	}
	err := LuuKhoan(Khoan{
		Ngay:       time.Now().Format("2006-01-02"),
		Loai:       KhoanChi,
		Nhom:       NhomHoanKhach,
		SoTien:     st,
		DienGiai:   dg,
		PhuongThuc: pt,
		MaDon:      don.Ma,
		Nguoi:      nd.TenHienThi(),
	})
	if err != nil {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok=Không+ghi+được:+"+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Đã hoàn "+dinhDangTien(st)+" cho khách"), http.StatusSeeOther)
}

func hQtXoaThu(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	r.ParseForm()
	ma := r.FormValue("ma")
	k, co := LayKhoan(ma)
	if !co || k.MaDon != don.Ma {
		http.NotFound(w, r)
		return
	}
	if err := XoaKhoan(ma); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Đã xoá khoản thu "+dinhDangTien(k.SoTien)), http.StatusSeeOther)
}
