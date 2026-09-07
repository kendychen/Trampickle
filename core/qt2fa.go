package core

// Trang bật/tắt xác thực hai bước cho chính tài khoản đang đăng nhập.
//
// Không cho chủ bật 2FA hộ người khác: bật hộ nghĩa là chủ giữ bí mật TOTP
// của thợ, và thợ mất khả năng đăng nhập cho tới khi được đưa mã. Ai muốn
// bật thì tự vào trang này quét mã của mình.

import (
	"net/http"
	"strings"
	"time"
)

type dl2FA struct {
	dlQt
	DangBat bool
	SoMaCon int
	// Lúc đang cài: bí mật vừa sinh, chưa lưu vào tài khoản cho tới khi
	// người dùng gõ đúng một mã từ nó.
	BiMat string
	URI   string
	// MaDuPhong hiện đúng một lần, ngay sau khi bật xong.
	MaDuPhong []string
}

func hQt2FA(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	d := dl2FA{dlQt: dlQt{Chung: chung(r, "2fa")}}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
		r.ParseForm()

		switch r.FormValue("viec") {
		case "bat":
			// Bí mật đi vòng qua ô ẩn trong biểu mẫu chứ không giữ trong RAM
			// máy chủ: giữ trong RAM phải quản lý hết hạn, dọn rác, và một
			// người mở hai tab là hỏng. Ô ẩn nằm trong trang chỉ chính chủ
			// mở được, và chỉ có tác dụng khi kèm một mã TOTP đúng.
			biMat := strings.TrimSpace(r.FormValue("bi_mat"))
			ma := strings.TrimSpace(r.FormValue("ma"))
			if biMat == "" {
				d.Loi = "Thiếu mã bí mật. Bấm lại nút Bắt đầu."
				break
			}
			if _, ok := kiemTOTP(biMat, ma, time.Now()); !ok {
				d.Loi = "Mã không đúng. Kiểm lại giờ trên điện thoại rồi gõ mã mới."
				d.BiMat, d.URI = biMat, uriOTPAuth(nd.Ten, biMat)
				break
			}
			maDP, err := DatTOTP(nd.Ten, biMat)
			if err != nil {
				d.Loi = err.Error()
				break
			}
			GhiNhatKy(MucNhatKy{Ai: nd.Ten, IP: ipCua(r), Viec: "bat-2fa", KetQua: "ok"})
			d.OK = "Đã bật xác thực hai bước."
			d.MaDuPhong = maDP

		case "tat":
			// Bắt gõ mã hiện tại mới cho tắt: nếu không, ai mượn được phiên
			// đang mở chỉ cần bấm một nút là gỡ sạch lớp bảo vệ.
			ma := strings.TrimSpace(r.FormValue("ma"))
			buoc, ok := kiemTOTP(nd.TotpBiMat, ma, time.Now())
			if ok && daDungBuoc(nd.Ten, buoc) {
				ok = false
			}
			if !ok && !DungMaDuPhong(nd.Ten, ma) {
				d.Loi = "Mã không đúng, chưa tắt gì cả."
				break
			}
			if ok {
				nhoBuoc(nd.Ten, buoc)
			}
			if err := TatTOTP(nd.Ten); err != nil {
				d.Loi = err.Error()
				break
			}
			GhiNhatKy(MucNhatKy{Ai: nd.Ten, IP: ipCua(r), Viec: "tat-2fa", KetQua: "ok"})
			d.OK = "Đã tắt xác thực hai bước."

		case "bat-dau":
			bm := sinhBiMatTOTP()
			d.BiMat, d.URI = bm, uriOTPAuth(nd.Ten, bm)
		}
		// Đọc lại: DatTOTP/TatTOTP vừa đổi tài khoản.
		nd, _ = TimNguoiDung(nd.Ten)
		d.NguoiDung = nd
	}

	d.DangBat = nd.TotpBat
	d.SoMaCon = len(nd.MaDuPhong)
	render(w, "qt-2fa.html", d)
}
