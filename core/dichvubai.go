package core

// Bài nội dung của từng việc: "vì sao hay hỏng kiểu này", "tự kiểm tra thế
// nào", "làm gì cho đỡ bị". Mỗi việc một tệp data/dich-vu-bai/<MA>.md, chữ
// markdown trần, KHÔNG có phần đầu YAML — tiêu đề, giá, bảo hành, điều kiện
// nhận đều đã nằm trong bang-gia.yaml rồi, chép lại lần nữa vào đây là mở ra
// hai nguồn sự thật cho cùng một con số.
//
// VÌ SAO NẰM Ở TRANG DỊCH VỤ CHỨ KHÔNG PHẢI MỘT BÀI VIẾT RIÊNG. Nội dung này
// chỉ hiện ở /dich-vu/<MA>, không có URL thứ hai. Nếu vừa đăng ở /bai-viet/x
// vừa nhúng vào /dich-vu/Y thì Google thấy hai trang cùng một bài và tự chọn
// một cái để giữ — mà cái nó bỏ thường lại là trang mình cần lên.
//
// Tệp thiếu thì trang dịch vụ chạy y như trước khi có phần này: mất phần
// chữ, còn nguyên giá và quy trình. Nên không việc nào bắt buộc phải có bài.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

func thuMucBaiDichVu() string { return P("data/dich-vu-bai") }

var (
	dvBaiMu sync.RWMutex
	dvBai   = map[string]string{}
)

// NapBaiDichVu đọc cả thư mục vào bộ nhớ. Gọi lúc khởi động và sau mỗi lần
// lưu ở trang quản trị.
func NapBaiDichVu() error {
	muc, err := os.ReadDir(thuMucBaiDichVu())
	if os.IsNotExist(err) {
		dvBaiMu.Lock()
		dvBai = map[string]string{}
		dvBaiMu.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	m := map[string]string{}
	for _, e := range muc {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		ma := strings.TrimSuffix(e.Name(), ".md")
		if !maDichVuSach(ma) {
			fmt.Printf("[bài dịch vụ] bỏ qua tên lạ: %s\n", e.Name())
			continue
		}
		b, err := os.ReadFile(filepath.Join(thuMucBaiDichVu(), e.Name()))
		if err != nil {
			fmt.Printf("[bài dịch vụ] bỏ qua %s: %v\n", e.Name(), err)
			continue
		}
		m[ma] = strings.ReplaceAll(string(b), "\r\n", "\n")
	}
	dvBaiMu.Lock()
	dvBai = m
	dvBaiMu.Unlock()
	return nil
}

// BaiDichVu trả chữ markdown của một việc, rỗng nếu việc ấy chưa có bài.
func BaiDichVu(ma string) string {
	dvBaiMu.RLock()
	defer dvBaiMu.RUnlock()
	return dvBai[strings.ToUpper(ma)]
}

// LuuBaiDichVu ghi bài của một việc. Chữ rỗng thì xoá tệp đi chứ không để
// lại một tệp trắng — thư mục còn phản ánh đúng "việc nào đã có bài".
func LuuBaiDichVu(ma, than string) error {
	ma = strings.ToUpper(strings.TrimSpace(ma))
	if !maDichVuSach(ma) {
		return fmt.Errorf("mã việc không hợp lệ")
	}
	if _, ok := TimDichVu(ma); !ok {
		return fmt.Errorf("không có việc nào mã %s", ma)
	}
	duong := filepath.Join(thuMucBaiDichVu(), ma+".md")
	than = strings.TrimRight(strings.ReplaceAll(than, "\r\n", "\n"), "\n")
	if strings.TrimSpace(than) == "" {
		if err := os.Remove(duong); err != nil && !os.IsNotExist(err) {
			return err
		}
		return NapBaiDichVu()
	}
	if err := os.MkdirAll(thuMucBaiDichVu(), 0o755); err != nil {
		return err
	}
	if err := ghiAtomic(duong, []byte(than+"\n")); err != nil {
		return err
	}
	return NapBaiDichVu()
}

// maDichVuSach: mã việc chỉ có chữ HOA và gạch dưới (DAN_VIEN, THAY_DE_GIAY).
// Tên tệp ghép từ đây nên không nhận dấu chấm — không có ".." nào lọt qua.
func maDichVuSach(ma string) bool {
	if ma == "" || len(ma) > 40 {
		return false
	}
	for _, c := range ma {
		if !(c >= 'A' && c <= 'Z' || c == '_') {
			return false
		}
	}
	return true
}

// TimDichVu tra một việc theo mã, kể cả việc đang ẩn — trang quản trị cần
// sửa được bài của việc tạm tắt.
//
// Hai ca trạm không nhận (khong_ban) cũng trả về ở đây, dựng thành DichVu chỉ
// có mã và tên. Chúng không nằm trong bảng giá nên mọi ô tiền bằng không —
// đúng ý, vì nơi duy nhất gọi hàm này là trang soạn bài, mà bài thì cả hai ca
// ấy đều phải có: khách hỏi "sửa được không" nhiều hơn hỏi giá.
func TimDichVu(ma string) (DichVu, bool) {
	ma = strings.ToUpper(ma)
	for _, dv := range DichVuTatCa() {
		if dv.Ma == ma {
			return dv, true
		}
	}
	if kb, ok := TimKhongBan(ma); ok {
		return DichVu{Ma: kb.Ma, Ten: kb.Ten}, true
	}
	return DichVu{}, false
}

// TimKhongBan tra một ca trạm từ chối theo mã.
func TimKhongBan(ma string) (KhongBan, bool) {
	ma = strings.ToUpper(ma)
	for _, kb := range GIA.KhongBan {
		if strings.ToUpper(kb.Ma) == ma {
			return kb, true
		}
	}
	return KhongBan{}, false
}
