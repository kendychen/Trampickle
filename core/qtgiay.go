package core

import "net/http"

// Trang danh mục giày: /qt/giay
//
// Thợ vào được, không giới hạn cho chủ — cùng lý do như /qt/vot: đây là thứ
// phải tra lúc khách đang đứng trước mặt, chặn đúng chỗ thì hoá ra không ai
// dùng.
//
// Trang chỉ ĐỌC. Sửa danh mục vẫn là sửa data/giay.yaml rồi chạy
// `python scripts/print_shoes.py --ghi`, để bản cho thợ và bản cho agent
// không bao giờ lệch nhau.
func dangKyGiay(mux *http.ServeMux) {
	mux.HandleFunc("GET /qt/giay", canDangNhap(hQtGiay))
}

func hQtGiay(w http.ResponseWriter, r *http.Request) {
	type dl struct {
		dlQt
		Kho     *KhoGiay
		Bang    []HangGiay
		ChiTiet map[string]ChiTietGiay
		SoDoi   int
		SoNhan  int
		DuTin   bool
		// Nhãn nhận/từ chối cho bảng kiểu đế. Template không gọi được map ở
		// cấp gói, mà chép lại năm cái nhãn vào HTML thì sớm muộn lệch nhau.
		Nhan map[string]string
	}
	d := dl{dlQt: dlQt{Chung: chung(r, "giay")}, Nhan: NhanXuLy}
	k, err := NapGiay()
	if err != nil {
		d.Loi = "Không đọc được danh mục giày: " + err.Error()
		render(w, "qt-giay.html", d)
		return
	}
	d.Kho, d.Bang, d.ChiTiet = k, k.BangGiay(), k.ChiTiet()
	d.SoDoi, d.SoNhan, d.DuTin = k.SoDoi(), k.SoNhan(), k.DuTin()
	render(w, "qt-giay.html", d)
}
