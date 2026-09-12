// Trang /qt/doi-tac — các tiệm ngoài trạm gửi việc sang.
//
// Chỉ chủ vào được: chọn gửi việc cho tiệm nào và biết đã trả tiệm bao nhiêu
// là chuyện tiền, cùng nhóm với sổ tiền và thống kê. Thợ vẫn chọn được tiệm
// ở trang đơn — chỉ không sửa được danh sách.
package core

import (
	"net/http"
	"net/url"
)

func hQtDoiTac(w http.ResponseWriter, r *http.Request) {
	c := chung(r, "doi-tac")
	c.TieuDe = "Đối tác gia công"
	render(w, "qt-doitac.html", struct {
		dlQt
		DoiTac []SoLieuDoiTac
	}{
		dlQt:   dlQt{Chung: c, OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		DoiTac: SoLieuCacDoiTac(),
	})
}

func hQtDoiTacLuu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Nội dung quá dài", http.StatusRequestEntityTooLarge)
		return
	}
	_, err := LuuDoiTac(DoiTac{
		Ma:     r.FormValue("ma"),
		Ten:    r.FormValue("ten"),
		Nghe:   r.FormValue("nghe"),
		LienHe: r.FormValue("lien_he"),
		DiaChi: r.FormValue("dia_chi"),
		GhiChu: r.FormValue("ghi_chu"),
		Ngung:  r.FormValue("ngung") == "1",
	})
	if err != nil {
		http.Redirect(w, r, "/qt/doi-tac?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/doi-tac?ok=Đã+lưu", http.StatusSeeOther)
}

func hQtDoiTacXoa(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	if err := XoaDoiTac(r.FormValue("ma")); err != nil {
		http.Redirect(w, r, "/qt/doi-tac?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/doi-tac?ok=Đã+xoá", http.StatusSeeOther)
}
