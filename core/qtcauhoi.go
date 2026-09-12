// Trang /qt/cau-hoi — thêm bớt câu hỏi thường gặp hiện ở /cau-hoi.
//
// Chỉ chủ vào được, cùng luật với /qt/noi-dung và /qt/giao-dien: đây là chữ
// đứng tên trạm nói với khách, không phải ghi chú nội bộ của một đơn.
package core

import (
	"net/http"
	"net/url"
	"strconv"
)

func hQtCauHoi(w http.ResponseWriter, r *http.Request) {
	c := chung(r, "cau-hoi-qt")
	c.TieuDe = "Câu hỏi thường gặp"
	render(w, "qt-cauhoi.html", struct {
		dlQt
		CauHoi []CauHoi
	}{
		dlQt:   dlQt{Chung: c, OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		CauHoi: DanhSachCauHoi(),
	})
}

func hQtCauHoiLuu(w http.ResponseWriter, r *http.Request) {
	// 32 KB: câu trả lời tối đa 2000 ký tự, mà tiếng Việt có dấu đi qua form
	// urlencoded thì mỗi chữ phồng lên gấp mấy lần.
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Nội dung quá dài", http.StatusRequestEntityTooLarge)
		return
	}
	tt, _ := strconv.Atoi(r.FormValue("thu_tu"))
	_, err := LuuCauHoi(CauHoi{
		Ma:    r.FormValue("ma"),
		Hoi:   r.FormValue("hoi"),
		Dap:   r.FormValue("dap"),
		ThuTu: tt,
		An:    r.FormValue("an") == "1",
	})
	if err != nil {
		http.Redirect(w, r, "/qt/cau-hoi?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/cau-hoi?ok=Đã+lưu", http.StatusSeeOther)
}

func hQtCauHoiXoa(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	if err := XoaCauHoi(r.FormValue("ma")); err != nil {
		http.Redirect(w, r, "/qt/cau-hoi?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/cau-hoi?ok=Đã+xoá", http.StatusSeeOther)
}
