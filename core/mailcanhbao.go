package core

// Báo về hộp thư chủ khi có đăng nhập từ IP chưa từng thấy.
//
// Vì sao đáng làm: khoá đăng nhập chặn được dò mật khẩu, 2FA chặn được mật
// khẩu lộ. Nhưng nếu cả hai đều thủng, thứ duy nhất còn lại là Kendy BIẾT có
// người vừa vào. Một cái mail là đường báo động rẻ nhất.
//
// Cố tình không chặn gì cả — chỉ báo. Đăng nhập từ quán cà phê hay từ 4G là
// chuyện thường ngày; chặn IP lạ sẽ khoá chính chủ ra ngoài nhiều hơn là
// chặn được kẻ trộm.

import (
	"fmt"
	"html"
	"os"
	"strings"
	"time"
)

// canhBaoDangNhapLa chạy nền, nuốt lỗi. Không có mail thì không sao — đăng
// nhập vẫn thành công, và nhật ký ở data/nhat-ky vẫn ghi đủ.
func canhBaoDangNhapLa(ten, ip string) {
	den := emailChu()
	if den == "" || !MailBat() || !HopLeEmail(den) {
		return
	}
	luc := time.Now().Format("15:04 02/01/2006")
	th := strings.TrimSpace(TenTram())
	if th == "" {
		th = "Trạm"
	}
	// Gửi nền: người vừa đăng nhập không việc gì phải ngồi chờ Resend trả
	// lời mới thấy trang quản trị.
	go func() {
		h, c := thanMailDangNhapLa(ten, ip, luc)
		if err := guiMail(den, "["+th+"] Đăng nhập từ máy lạ: "+ten, h, c); err != nil {
			// Ghi vào nhật ký chứ không chỉ stderr: mail cảnh báo hụt là
			// thứ phải thấy được từ trang quản trị, không phải từ journalctl.
			GhiNhatKy(MucNhatKy{
				Ai: ten, IP: ip, Viec: "canh-bao-dang-nhap",
				KetQua: "loi-mail: " + err.Error(),
			})
			fmt.Fprintln(os.Stderr, "Gửi mail cảnh báo hụt:", err)
		}
	}()
}

// thanMailDangNhapLa tách khỏi việc gửi để test được nội dung mà không gọi
// Resend. ten và ip đều đi qua html.EscapeString: ten là chuỗi người dùng gõ
// ở ô đăng nhập, ip có thể đến từ header do khách gửi.
func thanMailDangNhapLa(ten, ip, luc string) (string, string) {
	e := html.EscapeString

	var h strings.Builder
	h.WriteString(`<div style="font-family:-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#1a1d18;max-width:560px">`)
	h.WriteString(`<p>Vừa có một lần đăng nhập vào trang quản lý từ địa chỉ chưa từng dùng trước đây.</p>`)
	fmt.Fprintf(&h, `<p style="margin:22px 0;padding:14px 16px;border:1px dashed #999;border-radius:8px">`+
		`Tài khoản: <b>%s</b><br>Địa chỉ IP: <b>%s</b><br>Lúc: %s</p>`, e(ten), e(ip), e(luc))
	h.WriteString(`<p><b>Nếu đây là anh</b> thì bỏ qua mail này. Lần sau đăng nhập từ chính máy đó sẽ không báo nữa.</p>`)
	h.WriteString(`<p><b>Nếu không phải anh</b>: đổi mật khẩu ngay ở <code>/qt/nguoi-dung</code>, rồi mở <code>/qt/nhat-ky</code> xem người đó đã làm gì.</p>`)
	h.WriteString(`</div>`)

	c := "Vừa có một lần đăng nhập vào trang quản lý từ địa chỉ chưa từng dùng trước đây.\n\n" +
		"Tài khoản: " + ten + "\nĐịa chỉ IP: " + ip + "\nLúc: " + luc + "\n\n" +
		"Nếu đây là anh thì bỏ qua mail này.\n" +
		"Nếu không phải: đổi mật khẩu ngay, rồi mở Nhật ký xem người đó đã làm gì.\n"

	return h.String(), c
}
