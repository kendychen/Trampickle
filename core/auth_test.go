package core

import "testing"

// Hạ người chủ duy nhất phải bị từ chối MÀ KHÔNG đổi gì trong bộ nhớ. Bản cũ
// sửa trước rồi mới kiểm, nên tài khoản mất quyền quản trị dù đĩa vẫn ghi
// "chu" — đúng triệu chứng "ấn đổi sang thợ là không vào lại admin được".
func TestDoiVaiTroKhongHaChuCuoiCung(t *testing.T) {
	nguoiDungMu.Lock()
	cu := nguoiDung
	nguoiDung = []NguoiDung{{Ten: "kendy", VaiTro: VaiTroChu}}
	nguoiDungMu.Unlock()
	defer func() { nguoiDungMu.Lock(); nguoiDung = cu; nguoiDungMu.Unlock() }()

	if err := DoiVaiTro("kendy", VaiTroTho, false); err == nil {
		t.Fatal("phải báo lỗi khi hạ người chủ cuối cùng")
	}
	nguoiDungMu.RLock()
	vt := nguoiDung[0].VaiTro
	nguoiDungMu.RUnlock()
	if vt != VaiTroChu {
		t.Fatalf("bị từ chối rồi mà bộ nhớ vẫn đổi: vai trò = %q", vt)
	}
}

// Đổi vai trò thành công không được treo: luuNguoiDung tự RLock nên gọi nó
// khi còn giữ Lock là tự khoá chính mình.
func TestDoiVaiTroThanhCongKhongTreo(t *testing.T) {
	nguoiDungMu.Lock()
	cu := nguoiDung
	nguoiDung = []NguoiDung{{Ten: "kendy", VaiTro: VaiTroChu}, {Ten: "nam", VaiTro: VaiTroChu}}
	nguoiDungMu.Unlock()
	defer func() { nguoiDungMu.Lock(); nguoiDung = cu; nguoiDungMu.Unlock() }()

	if err := DoiVaiTro("nam", VaiTroTho, false); err != nil {
		t.Fatalf("đổi vai trò khi còn chủ khác phải được: %v", err)
	}
	nguoiDungMu.RLock()
	vt := nguoiDung[1].VaiTro
	nguoiDungMu.RUnlock()
	if vt != VaiTroTho {
		t.Fatalf("vai trò chưa đổi: %q", vt)
	}
}
