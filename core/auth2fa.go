package core

// Bật/tắt xác thực hai bước, và mã dự phòng.
//
// Vì sao 2FA phải có mã dự phòng: TOTP nằm trong điện thoại. Mất máy, đổi
// máy, xoá nhầm app — không còn đường nào vào lại, mà đây là tài khoản chủ
// trạm. Mỗi mã dùng đúng một lần rồi xoá hẳn khỏi danh sách.
//
// Lưu bcrypt hash chứ không phải bản rõ: mã dự phòng có sức mạnh ngang mật
// khẩu, để bản rõ trong tai-khoan.yaml là biến một file thành hai bí mật.

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const soMaDuPhong = 8

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

	if err := luuNguoiDung(); err != nil {
		// Mã đã bị xoá khỏi RAM nên lần này vẫn tính là dùng rồi. Lưu hụt
		// nghĩa là sau khi khởi động lại nó sống dậy — phải thấy được.
		fmt.Printf("[2fa] không lưu được sau khi dùng mã dự phòng: %v\n", err)
	}
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

// sinhMaDuPhong trả (bản rõ để hiện một lần, hash để lưu). Bản rõ có gạch
// nối cho dễ chép tay; hash băm bản ĐÃ chuẩn hoá, để lúc gõ lại có gạch hay
// không đều khớp.
func sinhMaDuPhong() (ro []string, hash []string, err error) {
	for i := 0; i < soMaDuPhong; i++ {
		// 5 byte = 10 ký tự hex.
		m := maNgauNhien(5)
		h, e := bcrypt.GenerateFromPassword([]byte(chuanMaDuPhong(m)), 12)
		if e != nil {
			return nil, nil, e
		}
		ro = append(ro, m[:5]+"-"+m[5:])
		hash = append(hash, string(h))
	}
	return ro, hash, nil
}

// DatTOTP bật 2FA sau khi người dùng đã chứng minh quét đúng mã. Trả về danh
// sách mã dự phòng bản rõ — hiện đúng một lần rồi không lấy lại được.
func DatTOTP(ten, biMat string) ([]string, error) {
	// Sinh mã TRƯỚC khi giữ khoá: tám lần bcrypt cost 12 mất khoảng hai
	// giây, giữ khoá ghi suốt quãng đó là chặn mọi request khác của trạm.
	ro, hash, err := sinhMaDuPhong()
	if err != nil {
		return nil, err
	}

	ten = strings.ToLower(strings.TrimSpace(ten))
	nguoiDungMu.Lock()
	thay := false
	for i := range nguoiDung {
		if strings.ToLower(nguoiDung[i].Ten) == ten {
			nguoiDung[i].TotpBiMat = biMat
			nguoiDung[i].TotpBat = true
			nguoiDung[i].MaDuPhong = hash
			thay = true
		}
	}
	nguoiDungMu.Unlock()
	if !thay {
		return nil, fmt.Errorf("không có tài khoản tên %q", ten)
	}
	// luuNguoiDung tự lấy RLock nên phải gọi sau khi đã nhả Lock.
	if err := luuNguoiDung(); err != nil {
		return nil, err
	}
	return ro, nil
}

func TatTOTP(ten string) error {
	ten = strings.ToLower(strings.TrimSpace(ten))
	nguoiDungMu.Lock()
	thay := false
	for i := range nguoiDung {
		if strings.ToLower(nguoiDung[i].Ten) == ten {
			nguoiDung[i].TotpBiMat = ""
			nguoiDung[i].TotpBat = false
			nguoiDung[i].MaDuPhong = nil
			thay = true
		}
	}
	nguoiDungMu.Unlock()
	if !thay {
		return fmt.Errorf("không có tài khoản tên %q", ten)
	}
	return luuNguoiDung()
}
