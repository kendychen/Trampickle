// Trang danh mục vợt: /qt/vot
//
// Thợ vào được, không giới hạn cho chủ: đây là thứ phải tra lúc khách đứng
// trước mặt, chặn đúng chỗ thì hoá ra không ai dùng.
//
// Trang chỉ ĐỌC. Sửa danh mục vẫn là sửa data/vot.yaml rồi chạy
// `python scripts/print_paddles.py --ghi`, để bản cho thợ và bản cho agent
// không bao giờ lệch nhau.
package core

import "net/http"

func dangKyVot(mux *http.ServeMux) {
	mux.HandleFunc("GET /qt/vot", canDangNhap(hQtVot))
}

func hQtVot(w http.ResponseWriter, r *http.Request) {
	type dl struct {
		dlQt
		Kho     *KhoVot
		Bang    []HangVot
		ChiTiet map[string]ChiTietDong
		DoiCT   map[string]CheTaoJS
		LoiVot  map[string]LoiJS
		SoCay   int
		DuTin   bool
	}
	d := dl{dlQt: dlQt{Chung: chung(r, "vot")}}

	k, err := NapVot()
	if err != nil {
		d.Loi = "Không đọc được danh mục vợt: " + err.Error()
		render(w, "qt-vot.html", d)
		return
	}
	d.Kho, d.Bang, d.ChiTiet, d.SoCay, d.DuTin = k, k.BangVot(), k.ChiTiet(), k.SoCay(), k.DuTin()
	d.DoiCT = k.CheTaoJSON()
	d.LoiVot = k.LoiJSON()
	render(w, "qt-vot.html", d)
}
