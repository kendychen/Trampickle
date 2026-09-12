// Trang kho: /qt/kho
//
// Thợ vào được — xuất vật tư, kiểm kê, báo hao hụt là việc hằng ngày của thợ.
// Hai thứ thợ không đụng: danh mục vật tư kèm định mức (đổi định mức là đổi
// cách cả trạm tính hao phí), và phiếu NHẬP (phiếu nhập kéo theo một khoản
// chi trong sổ tiền). Cột tiền trong bảng tồn cũng chỉ hiện cho chủ.
package core

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

func dangKyKho(mux *http.ServeMux) {
	mux.HandleFunc("GET /qt/kho", canDangNhap(hQtKho))
	mux.HandleFunc("GET /qt/kho/phieu", canDangNhap(hQtPhieuDs))
	mux.HandleFunc("GET /qt/kho/phieu/moi", canDangNhap(hQtPhieuMoi))
	mux.HandleFunc("POST /qt/kho/phieu/luu", canDangNhap(hQtPhieuLuu))
	mux.HandleFunc("POST /qt/kho/phieu/xoa", canDangNhap(hQtPhieuXoa))
	mux.HandleFunc("GET /qt/kho/phieu/{ma}", canDangNhap(hQtPhieuSua))

	mux.HandleFunc("GET /qt/kho/vat-tu", canLaChu(hQtVatTu))
	mux.HandleFunc("POST /qt/kho/vat-tu", canLaChu(hQtVatTuLuu))
	mux.HandleFunc("POST /qt/kho/dinh-muc", canLaChu(hQtDinhMucLuu))

	mux.HandleFunc("POST /qt/don/{ma}/xuat-kho", canDangNhap(hQtXuatKhoDon))
}

// --- Bảng tồn ---------------------------------------------------------

func hQtKho(w http.ResponseWriter, r *http.Request) {
	render(w, "qt-kho.html", struct {
		dlQt
		Bang      []DongTonKho
		Phieu     []*Phieu
		LoaiPhieu []MoTaPhieu
		CanMua    int
		GiaTri    int
		ChuaKhai  bool
	}{
		dlQt:      dlQt{Chung: chung(r, "kho"), OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		Bang:      BangTonKho(),
		Phieu:     PhieuGanDay(15),
		LoaiPhieu: CacLoaiPhieu,
		CanMua:    SoVatTuCanMua(),
		GiaTri:    GiaTriTonKho(),
		ChuaKhai:  len(DanhSachVatTu(false)) == 0,
	})
}

func hQtPhieuDs(w http.ResponseWriter, r *http.Request) {
	render(w, "qt-kho-phieu.html", struct {
		dlQt
		Phieu     []*Phieu
		LoaiPhieu []MoTaPhieu
	}{
		dlQt:      dlQt{Chung: chung(r, "kho-phieu"), OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		Phieu:     PhieuGanDay(200),
		LoaiPhieu: CacLoaiPhieu,
	})
}

// --- Một phiếu --------------------------------------------------------

type dlPhieu struct {
	dlQt
	Phieu     *Phieu
	VatTu     []VatTu
	LoaiPhieu []MoTaPhieu
	Ton       map[string]float64
	Moi       bool
	SuaDuoc   bool
}

func hQtPhieuMoi(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	loai := r.URL.Query().Get("loai")
	if !LoaiPhieuHopLe(loai) {
		loai = PhieuXuat
	}
	if loai == PhieuNhap && !nd.LaChu() {
		http.Redirect(w, r, "/qt/kho?loi="+url.QueryEscape("Phiếu nhập do chủ ghi, vì nó kéo theo một khoản chi"), http.StatusSeeOther)
		return
	}
	p := &Phieu{
		Loai:  loai,
		Ngay:  time.Now().Format("2006-01-02"),
		Nguoi: nd.TenHienThi(),
		MaDon: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("don"))),
		Dong:  []DongPhieu{},
	}
	// Phiếu kiểm kê dựng sẵn cả danh mục: người ta cầm sổ đi đếm từng thứ,
	// bỏ trống dòng nào thì dòng đó không vào phiếu.
	if loai == PhieuKiemKe {
		for _, v := range DanhSachVatTu(false) {
			p.Dong = append(p.Dong, DongPhieu{MaVatTu: v.Ma, SoLuong: -1})
		}
	}
	if p.MaDon != "" {
		if don, co := LayDon(p.MaDon); co {
			p.Dong = append(p.Dong, DeXuatXuatChoDon(don)...)
			p.GhiChu = "Vật tư dùng cho đơn " + don.Ma
		}
	}
	renderPhieu(w, r, p, true)
}

func hQtPhieuSua(w http.ResponseWriter, r *http.Request) {
	p, co := LayPhieu(strings.ToUpper(r.PathValue("ma")))
	if !co {
		http.NotFound(w, r)
		return
	}
	renderPhieu(w, r, p, false)
}

func renderPhieu(w http.ResponseWriter, r *http.Request, p *Phieu, moi bool) {
	nd, _ := NguoiDangNhap(r)
	c := chung(r, "kho-phieu")
	if !moi {
		c.TieuDe = "Phiếu " + p.Ma
	}
	render(w, "qt-kho-mot-phieu.html", dlPhieu{
		dlQt:      dlQt{Chung: c, OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		Phieu:     p,
		VatTu:     DanhSachVatTu(false),
		LoaiPhieu: CacLoaiPhieu,
		Ton:       TonKho(),
		Moi:       moi,
		SuaDuoc:   p.Loai != PhieuNhap || nd.LaChu(),
	})
}

func hQtPhieuLuu(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
	r.ParseForm()

	loai := r.FormValue("loai")
	if !LoaiPhieuHopLe(loai) {
		http.Error(w, "loại phiếu không hợp lệ", http.StatusBadRequest)
		return
	}
	if loai == PhieuNhap && !nd.LaChu() {
		http.Error(w, "chỉ chủ ghi được phiếu nhập", http.StatusForbidden)
		return
	}

	ma := strings.ToUpper(strings.TrimSpace(r.FormValue("ma")))
	var p *Phieu
	if ma != "" {
		cu, co := LayPhieu(ma)
		if !co {
			http.NotFound(w, r)
			return
		}
		p = cu
	} else {
		p = &Phieu{Ma: MaPhieuMoi(loai), Nguoi: nd.TenHienThi()}
	}
	p.Loai = loai
	p.Ngay = ngayISO(r.FormValue("ngay"))
	if p.Ngay == "" {
		p.Ngay = time.Now().Format("2006-01-02")
	}
	p.MaDon = strings.ToUpper(strings.TrimSpace(r.FormValue("ma_don")))
	p.NhaCungCap = catBot(r.FormValue("nha_cung_cap"), 120)
	p.GhiChu = catBot(r.FormValue("ghi_chu"), 300)
	p.Dong = docDongPhieu(r, loai)

	if len(p.Dong) == 0 {
		http.Redirect(w, r, "/qt/kho?loi="+url.QueryEscape("Phiếu không có dòng nào, chưa lưu"), http.StatusSeeOther)
		return
	}
	if err := LuuPhieu(p); err != nil {
		http.Redirect(w, r, "/qt/kho?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}

	tb := "Đã lưu phiếu " + p.Ma
	if loai == PhieuNhap {
		// Nhập kho là tiền đã ra khỏi túi (hoặc sắp ra, nếu còn nợ nhà cung
		// cấp) — ghi thẳng vào sổ để khỏi phải nhớ gõ lại lần thứ hai.
		if err := GhiChiTuPhieu(p, r.FormValue("da_tra") == "1"); err != nil {
			tb += ", nhưng chưa ghi được khoản chi: " + err.Error()
		} else if r.FormValue("da_tra") == "1" {
			tb += " và ghi chi " + dinhDangTien(p.TongTien)
		} else {
			tb += ". Khoản chi " + dinhDangTien(p.TongTien) + " đang chờ xác nhận trong sổ tiền"
		}
	}
	ve := "/qt/kho?ok=" + url.QueryEscape(tb)
	if p.MaDon != "" {
		ve = "/qt/don/" + p.MaDon + "?ok=" + url.QueryEscape(tb)
	}
	http.Redirect(w, r, ve, http.StatusSeeOther)
}

// docDongPhieu đọc các dòng từ form. Dòng không chọn vật tư, hoặc số lượng ≤ 0
// (kiểm kê thì < 0), coi như người ta bỏ trống — bỏ qua chứ không báo lỗi.
// Riêng kiểm kê nhận số 0: "đếm xong, hết sạch" là một câu trả lời hợp lệ.
func docDongPhieu(r *http.Request, loai string) []DongPhieu {
	ma := r.Form["ma_vat_tu"]
	out := []DongPhieu{}
	for i := range ma {
		mv := strings.TrimSpace(ma[i])
		if mv == "" {
			continue
		}
		if _, co := TimVatTu(mv); !co {
			continue
		}
		sl := soThuc(lay(r.Form["so_luong"], i))
		if sl < 0 || (sl == 0 && loai != PhieuKiemKe) {
			continue
		}
		if lay(r.Form["so_luong"], i) == "" {
			continue // kiểm kê: bỏ trống = không đếm thứ này, giữ nguyên tồn
		}
		d := DongPhieu{MaVatTu: mv, SoLuong: sl, GhiChu: catBot(lay(r.Form["ghi_chu_dong"], i), 120)}
		if loai == PhieuNhap {
			d.DonGia = soTien(lay(r.Form["don_gia"], i))
		}
		out = append(out, d)
	}
	return out
}

func hQtPhieuXoa(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	r.ParseForm()
	ma := strings.ToUpper(strings.TrimSpace(r.FormValue("ma")))
	p, co := LayPhieu(ma)
	if !co {
		http.NotFound(w, r)
		return
	}
	if p.Loai == PhieuNhap && !nd.LaChu() {
		http.Error(w, "chỉ chủ xoá được phiếu nhập", http.StatusForbidden)
		return
	}
	// Xoá phiếu nhập thì xoá luôn khoản chi kèm theo, nếu không sổ còn lại một
	// khoản chi trỏ tới phiếu không tồn tại.
	if p.Loai == PhieuNhap {
		if err := XoaChiTuPhieu(p.Ma); err != nil {
			http.Redirect(w, r, "/qt/kho?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
			return
		}
	}
	if err := XoaPhieu(ma); err != nil {
		http.Redirect(w, r, "/qt/kho?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/kho?ok="+url.QueryEscape("Đã xoá phiếu "+ma), http.StatusSeeOther)
}

// --- Danh mục vật tư và định mức --------------------------------------

func hQtVatTu(w http.ResponseWriter, r *http.Request) {
	dm := map[string][]DongDinhMuc{}
	// Định mức vật tư là chuyện trong xưởng: việc đang tắt với khách vẫn phải
	// khai được, vì thợ vẫn nhận làm nó khi khách mang tới tận nơi.
	for _, dv := range DichVuLenPhieu(GiaiDoan) {
		dm[dv.Ma] = DinhMucCua(dv.Ma)
	}
	render(w, "qt-kho-vattu.html", struct {
		dlQt
		VatTu   []VatTu
		DichVu  []DichVu
		DinhMuc map[string][]DongDinhMuc
	}{
		dlQt:    dlQt{Chung: chung(r, "kho-vat-tu"), OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		VatTu:   DanhSachVatTu(true),
		DichVu:  DichVuLenPhieu(GiaiDoan),
		DinhMuc: dm,
	})
}

func hQtVatTuLuu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	r.ParseForm()

	ten := r.Form["ten"]
	ds := make([]VatTu, 0, len(ten))
	// Gom trước mọi mã đang được gửi lên, kể cả của dòng phía dưới: sinh mã mới
	// mà đụng vào mã một dòng chưa duyệt tới thì dòng đó mất lịch sử phiếu.
	dung := map[string]bool{}
	for _, m := range r.Form["ma"] {
		if m = strings.TrimSpace(m); m != "" {
			dung[m] = true
		}
	}
	daDung := map[string]bool{}
	for i := range ten {
		v := VatTu{
			Ma:          strings.TrimSpace(lay(r.Form["ma"], i)),
			Ten:         catBot(ten[i], 120),
			DonVi:       catBot(lay(r.Form["don_vi"], i), 20),
			TonToiThieu: soThuc(lay(r.Form["ton_toi_thieu"], i)),
			NhaCungCap:  catBot(lay(r.Form["nha_cung_cap"], i), 120),
			GhiChu:      catBot(lay(r.Form["ghi_chu"], i), 200),
			Ngung:       lay(r.Form["ngung"], i) == "1",
		}
		// Dòng trống cuối bảng là chỗ để thêm thứ mới; bỏ trống thì bỏ qua.
		if strings.TrimSpace(v.Ten) == "" {
			continue
		}
		if v.DonVi == "" {
			v.DonVi = "cái"
		}
		// Mã là thứ phiếu cũ trỏ vào. Sinh một lần rồi giữ nguyên đời đời —
		// đặt theo tên thì đổi tên vật tư là mọi phiếu cũ trỏ vào hư không.
		if v.Ma == "" || daDung[v.Ma] {
			v.Ma = maVatTuMoi(dung)
			dung[v.Ma] = true
		}
		daDung[v.Ma] = true
		ds = append(ds, v)
	}
	if err := LuuDanhMucVatTu(ds); err != nil {
		http.Redirect(w, r, "/qt/kho/vat-tu?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/kho/vat-tu?ok="+url.QueryEscape("Đã lưu danh mục vật tư"), http.StatusSeeOther)
}

func hQtDinhMucLuu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	r.ParseForm()

	// Form gửi theo cặp song song: dv[] là mã dịch vụ của dòng, vt[] là vật
	// tư, sl[] là số lượng. Một dịch vụ có thể xuất hiện nhiều dòng.
	dv := r.Form["dv"]
	dm := map[string][]DongDinhMuc{}
	for i := range dv {
		maDV := strings.TrimSpace(dv[i])
		maVT := strings.TrimSpace(lay(r.Form["vt"], i))
		sl := soThuc(lay(r.Form["sl"], i))
		if maDV == "" || maVT == "" || sl <= 0 {
			continue
		}
		if _, co := TimVatTu(maVT); !co {
			continue
		}
		dm[maDV] = append(dm[maDV], DongDinhMuc{MaVatTu: maVT, SoLuong: sl})
	}
	if err := LuuDinhMuc(dm); err != nil {
		http.Redirect(w, r, "/qt/kho/vat-tu?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/kho/vat-tu?ok="+url.QueryEscape("Đã lưu định mức"), http.StatusSeeOther)
}

// --- Xuất kho cho một đơn ---------------------------------------------

// hQtXuatKhoDon — nút trên trang đơn. Hai đường:
//
//	nhanh=1 : đúng định mức, ghi luôn. Vẫn là người bấm, không phải máy tự trừ.
//	mặc định: mở form phiếu đã điền sẵn để thợ sửa cho khớp thực tế.
//
// Chặn phiếu thứ hai cho cùng một đơn: bấm hai lần (hoặc F5 lại trang) mà đẻ
// ra hai phiếu xuất thì kho hụt gấp đôi và không ai nhìn ra vì cả hai phiếu
// đều hợp lệ.
func hQtXuatKhoDon(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	if p, co := PhieuXuatCuaDon(don.Ma); co {
		http.Redirect(w, r, "/qt/kho/phieu/"+p.Ma+"?ok="+url.QueryEscape("Đơn này đã có phiếu xuất, sửa ở đây"), http.StatusSeeOther)
		return
	}
	r.ParseForm()
	if r.FormValue("nhanh") != "1" {
		http.Redirect(w, r, "/qt/kho/phieu/moi?loai=xuat&don="+url.QueryEscape(don.Ma), http.StatusSeeOther)
		return
	}

	dong := DeXuatXuatChoDon(don)
	if len(dong) == 0 {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Chưa khai định mức cho dịch vụ nào trong đơn"), http.StatusSeeOther)
		return
	}
	p := &Phieu{
		Ma:     MaPhieuMoi(PhieuXuat),
		Loai:   PhieuXuat,
		Ngay:   time.Now().Format("2006-01-02"),
		Nguoi:  nd.TenHienThi(),
		MaDon:  don.Ma,
		GhiChu: "Đúng định mức, xác nhận nhanh",
		Dong:   dong,
	}
	if err := LuuPhieu(p); err != nil {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Không lưu được phiếu: "+err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+url.QueryEscape("Đã xuất kho theo định mức, phiếu "+p.Ma), http.StatusSeeOther)
}
