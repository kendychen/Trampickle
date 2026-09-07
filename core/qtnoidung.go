package core

// Màn sửa chữ trên trang khách.
//
// Toàn bộ danh sách ô nhập dựng từ CayND, không gõ tay trong template. Thêm
// một khoá vào cây là ô nhập tự hiện ra đúng nhóm, đúng nhãn — quên cập nhật
// trang admin là chuyện không xảy ra được.
//
// Lưu theo TỪNG TRANG một, và chỉ nhận những khoá thuộc trang đang mở. Duyệt
// theo cây chứ không duyệt theo form: form là thứ trình duyệt gửi lên, cây là
// thứ mình biết chắc.

import (
	"net/http"
)

type ndMucQt struct {
	MucND
	GiaTri string
	DaSua  bool
}

type ndNhomQt struct {
	Ten string
	Muc []ndMucQt
}

type ndTabQt struct {
	Ma  string
	Ten string
	So  int
	Day bool
}

type dlQtND struct {
	dlQt
	Tab  []ndTabQt
	Hien TrangND
	Nhom []ndNhomQt
	// SoDoi: tổng số khoá đã đổi trên toàn site, hiện ở đầu trang.
	SoDoi int
}

func timTrangND(ma string) (TrangND, bool) {
	for _, t := range CayND {
		if t.Ma == ma {
			return t, true
		}
	}
	return TrangND{}, false
}

// veND quay lại đúng tab vừa làm việc. ma phải là mã đã kiểm, vì nó đi thẳng
// vào đường dẫn.
func veND(w http.ResponseWriter, r *http.Request, ma, ok, loi string) {
	duong := "/qt/noi-dung"
	if ma != "" {
		duong += "/" + ma
	}
	switch {
	case loi != "":
		duong += "?loi=" + urlEsc(loi)
	case ok != "":
		duong += "?ok=" + urlEsc(ok)
	}
	http.Redirect(w, r, duong, http.StatusSeeOther)
}

func hQtND(w http.ResponseWriter, r *http.Request) {
	ma := r.PathValue("ma")
	if ma == "" {
		ma = CayND[0].Ma
	}
	t, co := timTrangND(ma)
	if !co {
		http.NotFound(w, r)
		return
	}

	d := dlQtND{dlQt: dlQt{Chung: chung(r, "noi-dung")}}
	d.OK = r.URL.Query().Get("ok")
	d.Loi = r.URL.Query().Get("loi")
	d.Hien = t
	for _, x := range CayND {
		so := SoDaSuaND(x)
		d.SoDoi += so
		d.Tab = append(d.Tab, ndTabQt{Ma: x.Ma, Ten: x.Ten, So: so, Day: x.Ma == ma})
	}
	for _, n := range t.Nhom {
		nh := ndNhomQt{Ten: n.Ten}
		for _, m := range n.Muc {
			nh.Muc = append(nh.Muc, ndMucQt{MucND: m, GiaTri: ND(m.Khoa), DaSua: DaSuaND(m.Khoa)})
		}
		d.Nhom = append(d.Nhom, nh)
	}
	render(w, "qt-noidung.html", d)
}

func hQtNDLuu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		veND(w, r, "", "", "Nội dung quá dài")
		return
	}
	t, co := timTrangND(r.FormValue("ma"))
	if !co {
		http.NotFound(w, r)
		return
	}
	moi := map[string]string{}
	for _, n := range t.Nhom {
		for _, m := range n.Muc {
			if v, gui := r.Form["k."+m.Khoa]; gui {
				moi[m.Khoa] = v[0]
			}
		}
	}
	if err := DatND(moi); err != nil {
		veND(w, r, t.Ma, "", err.Error())
		return
	}
	veND(w, r, t.Ma, "Đã lưu — mở trang khách xem là thấy ngay", "")
}

func hQtNDKhoiPhuc(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	t, co := timTrangND(r.FormValue("ma"))
	if !co {
		http.NotFound(w, r)
		return
	}
	khoa := r.FormValue("khoa")
	if NDMac(khoa) == "" && !DaSuaND(khoa) {
		veND(w, r, t.Ma, "", "Không có khoá "+khoa)
		return
	}
	if err := KhoiPhucND(khoa); err != nil {
		veND(w, r, t.Ma, "", err.Error())
		return
	}
	veND(w, r, t.Ma, "Đã trả về chữ mặc định", "")
}
