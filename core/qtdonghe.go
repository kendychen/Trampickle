// Trang đồ nghề: /qt/do-nghe
//
// Chỉ chủ vào được. Trang này lộ toàn bộ số tiền đã bỏ ra để mở tiệm và số còn
// phải bỏ tiếp — không phải việc của thợ.
package core

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

func dangKyDoNghe(mux *http.ServeMux) {
	mux.HandleFunc("GET /qt/do-nghe", canLaChu(hQtDoNghe))
	mux.HandleFunc("POST /qt/do-nghe/trang-thai", canLaChu(hQtDoNgheTrangThai))
	mux.HandleFunc("POST /qt/do-nghe/luu", canLaChu(hQtDoNgheLuu))
	mux.HandleFunc("POST /qt/do-nghe/xoa", canLaChu(hQtDoNgheXoa))
}

func hQtDoNghe(w http.ResponseWriter, r *http.Request) {
	nghe := r.URL.Query().Get("nghe")
	if !NgheHopLe(nghe) {
		nghe = ""
	}
	tt := r.URL.Query().Get("tt")
	if !TrangThaiDoNgheHopLe(tt) {
		tt = ""
	}
	render(w, "qt-do-nghe.html", struct {
		dlQt
		Bang      []NhomBangDoNghe
		TongQuan  TongQuanDoNghe
		Nghe      string
		TrangThai string
		CacNghe   []MoTaNghe
		CacTT     []MoTaTrangThaiDoNghe
		CacNhom   []MoTaNhomDoNghe
		HomNay    string
		ChuaKhai  bool
	}{
		dlQt:      dlQt{Chung: chung(r, "do-nghe"), OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		Bang:      BangDoNghe(nghe, tt),
		TongQuan:  TongQuanDoNgheHienGio(),
		Nghe:      nghe,
		TrangThai: tt,
		CacNghe:   CacNghe,
		CacTT:     CacTrangThaiDoNghe,
		CacNhom:   CacNhomDoNghe,
		HomNay:    time.Now().Format("2006-01-02"),
		ChuaKhai:  len(DanhSachDoNghe()) == 0,
	})
}

// veDoNghe giữ nguyên bộ lọc đang xem. Đánh dấu xong mà bảng nhảy về "tất cả"
// thì mất chỗ đang làm dở, phải cuộn tìm lại.
func veDoNghe(r *http.Request, tham string) string {
	q := url.Values{}
	if n := r.FormValue("ve_nghe"); NgheHopLe(n) {
		q.Set("nghe", n)
	}
	if t := r.FormValue("ve_tt"); TrangThaiDoNgheHopLe(t) {
		q.Set("tt", t)
	}
	ve := "/qt/do-nghe"
	if len(q) > 0 {
		ve += "?" + q.Encode()
	}
	if tham == "" {
		return ve
	}
	if len(q) > 0 {
		return ve + "&" + tham
	}
	return ve + "?" + tham
}

func hQtDoNgheTrangThai(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	r.ParseForm()
	nd, _ := NguoiDangNhap(r)

	ma := strings.TrimSpace(r.FormValue("ma"))
	tt := r.FormValue("trang_thai")
	nhac, err := DoiTrangThaiDoNghe(ma, tt, r.FormValue("ngay_mua"),
		soTien(r.FormValue("gia_thuc")), r.FormValue("nguon_mua"), nd.TenHienThi())
	if err != nil {
		http.Redirect(w, r, veDoNghe(r, "loi="+url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}
	d, _ := TimDoNghe(ma)
	tb := d.Ten + ": " + tenTrongBang(tt, trangThaiDoNgheTen)
	if nhac != "" {
		tb += ". " + nhac
	}
	http.Redirect(w, r, veDoNghe(r, "ok="+url.QueryEscape(tb)), http.StatusSeeOther)
}

func hQtDoNgheLuu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	r.ParseForm()

	d := DoNghe{
		Ma:        strings.TrimSpace(r.FormValue("ma")),
		Ten:       catBot(r.FormValue("ten"), 120),
		Nghe:      r.FormValue("nghe"),
		Nhom:      r.FormValue("nhom"),
		Dot:       soTien(r.FormValue("dot")),
		BatBuoc:   r.FormValue("bat_buoc") == "1",
		QuyCach:   catBot(r.FormValue("quy_cach"), 200),
		DungDe:    catBot(r.FormValue("dung_de"), 200),
		GiaDuKien: soTien(r.FormValue("gia_du_kien")),
		GhiChu:    catBot(r.FormValue("ghi_chu"), 200),
	}
	moi := d.Ma == ""
	d, err := LuuDoNghe(d)
	if err != nil {
		http.Redirect(w, r, veDoNghe(r, "loi="+url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}
	tb := "Đã sửa " + d.Ten
	if moi {
		tb = "Đã thêm " + d.Ten
	}
	http.Redirect(w, r, veDoNghe(r, "ok="+url.QueryEscape(tb)), http.StatusSeeOther)
}

func hQtDoNgheXoa(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	if err := XoaDoNghe(strings.TrimSpace(r.FormValue("ma"))); err != nil {
		http.Redirect(w, r, veDoNghe(r, "loi="+url.QueryEscape(err.Error())), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, veDoNghe(r, "ok="+url.QueryEscape("Đã xoá khỏi danh sách")), http.StatusSeeOther)
}
