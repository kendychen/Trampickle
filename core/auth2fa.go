package core

// Mã dự phòng cho xác thực hai bước.
//
// VÁ ĐỂ BIÊN DỊCH LẠI: core/auth.go trỏ tới file này và core/server.go gọi
// DungMaDuPhong, nhưng file chưa kịp ra đời — cả app đứng ở lỗi "undefined:
// DungMaDuPhong", không build được bản nào. Ở đây chỉ có đúng phần luồng
// đăng nhập cần. Việc bật/tắt 2FA, sinh mã dự phòng và trang quản trị vẫn
// còn để trống cho người đang làm dở phần đó viết tiếp.
//
// Vì sao 2FA phải có mã dự phòng: TOTP nằm trong điện thoại. Mất máy, đổi
// máy, xoá nhầm app — không còn đường nào vào lại, mà đây là tài khoản chủ
// trạm. Mỗi mã dùng đúng một lần rồi xoá hẳn khỏi danh sách.

import (
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// DungMaDuPhong so mã người dùng gõ với từng hash còn lại của tài khoản.
// Khớp thì xoá hash ấy đi rồi lưu file, trả về true.
//
// Xoá chứ không đánh dấu đã dùng: một hash đã dùng còn nằm lại trong file
// nghĩa là bản sao lưu nào của file đó cũng còn một mã vào được.
func DungMaDuPhong(ten, ma string) bool {
	ma = chuanMaDuPhong(ma)
	if ma == "" {
		return false
	}
	ten = strings.ToLower(strings.TrimSpace(ten))

	// Chụp danh sách ra rồi so bcrypt NGOÀI khoá. So trong lúc giữ khoá ghi
	// thì tám lần cost 12 nối đuôi nhau, mọi request khác của trạm xếp hàng
	// chờ gần một giây — một cái khoá đăng nhập tự tạo.
	nguoiDungMu.RLock()
	var ds []string
	for _, n := range nguoiDung {
		if strings.ToLower(n.Ten) == ten {
			ds = append(ds, n.MaDuPhong...)
			break
		}
	}
	nguoiDungMu.RUnlock()

	khop := -1
	for i, h := range ds {
		if bcrypt.CompareHashAndPassword([]byte(h), []byte(ma)) == nil {
			khop = i
			break
		}
	}
	if khop < 0 {
		return false
	}

	nguoiDungMu.Lock()
	for i := range nguoiDung {
		if strings.ToLower(nguoiDung[i].Ten) != ten {
			continue
		}
		// Tìm lại theo nội dung hash chứ không theo chỉ số cũ: trong lúc so
		// bcrypt ở trên, một lần đăng nhập khác có thể đã xoá mất một mã và
		// chỉ số cũ giờ trỏ sang mã còn nguyên giá trị.
		for j, h := range nguoiDung[i].MaDuPhong {
			if h == ds[khop] {
				con := make([]string, 0, len(nguoiDung[i].MaDuPhong)-1)
				con = append(con, nguoiDung[i].MaDuPhong[:j]...)
				con = append(con, nguoiDung[i].MaDuPhong[j+1:]...)
				nguoiDung[i].MaDuPhong = con
				break
			}
		}
		break
	}
	nguoiDungMu.Unlock()

	luuNguoiDung()
	return true
}

// chuanMaDuPhong bỏ gạch nối, khoảng trắng và hạ chữ thường. Mã chép tay từ
// tờ giấy thì người gõ hay bỏ gạch hoặc gõ hoa; bắt gõ đúng từng ký tự chỉ
// làm hỏng nốt lần vào cuối cùng còn lại.
func chuanMaDuPhong(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, "-", "")
	s = strings.ReplaceAll(s, " ", "")
	return s
}
