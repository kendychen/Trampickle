// Bảo hành: hạn mặc định theo dịch vụ, và đơn bảo hành gắn được với đơn gốc.
//
// Hai luật đứng sau mọi hàm dưới đây:
//
//  1. Ô gõ tay THẮNG. Máy chỉ điền khi ô đang trống. Thợ đứng trước cây vợt
//     biết chuyện mà máy không biết — dây khách tự mua, đế giày đã mòn sẵn —
//     nên số của thợ luôn đè số của máy. Xem "máy đoán, người xác nhận" ở
//     core/tudong.go.
//
//  2. Hạn đếm từ NGÀY GIAO chứ không từ ngày nhận. Cây vợt nằm ở trạm ba
//     tuần chờ dây về thì ba tuần ấy không được ăn vào bảo hành của khách.
//     Đơn chưa giao thì chưa có hạn — không đoán trước.
//
// Số tháng lấy LỚN NHẤT trong các dịch vụ đã chốt của đơn, không phải nhỏ
// nhất: khách không phân biệt được món nào bảo hành mấy tháng, mà hứa ngắn
// rồi từ chối sửa lại là mất khách — đắt hơn mấy lần sửa lại.
package core

import (
	"sort"
	"strings"
	"time"
)

// BaoHanhThangCuaDon — số tháng bảo hành của đơn: lớn nhất trong các dịch vụ
// đã chốt. Dòng gõ tay không có mã dịch vụ nên không mang bảo hành.
func BaoHanhThangCuaDon(d *Don) int {
	if d == nil {
		return 0
	}
	max := 0
	for _, dt := range d.DongTien {
		if dt.MaDichVu == "" || dt.MaDichVu == MaGiamGia {
			continue
		}
		dv, co := TimDichVu(dt.MaDichVu)
		if co && dv.BaoHanhThang > max {
			max = dv.BaoHanhThang
		}
	}
	return max
}

// ApBaoHanhMacDinh điền hạn bảo hành khi ô đang trống và đơn đã giao. Trả về
// true nếu có đổi — chỗ gọi biết mình cần lưu lại hay không.
func ApBaoHanhMacDinh(d *Don) bool {
	if d == nil || strings.TrimSpace(d.BaoHanhDen) != "" {
		return false
	}
	ngGiao := d.NgayGiao()
	if ngGiao == "" {
		return false
	}
	thang := BaoHanhThangCuaDon(d)
	if thang <= 0 {
		return false
	}
	den := congThang(ngGiao, thang)
	if den == "" {
		return false
	}
	d.BaoHanhDen = den
	return true
}

// congThang cộng n tháng vào một ngày ISO. Ngày không tồn tại ở tháng đích
// thì lùi về ngày cuối tháng — 31/12 cộng hai tháng là 28/02, không phải
// 03/03 như AddDate trả về. Khách đọc "bảo hành tới 03/03" của một đơn nhận
// ngày 31 thì tưởng trạm tính nhầm.
func congThang(ngay string, n int) string {
	t, err := time.Parse("2006-01-02", strings.TrimSpace(ngay))
	if err != nil {
		return ""
	}
	ra := t.AddDate(0, n, 0)
	if ra.Day() != t.Day() {
		ra = time.Date(ra.Year(), ra.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, -1)
	}
	return ra.Format("2006-01-02")
}

// LaDonBaoHanh — đơn này mở ra để làm lại một đơn cũ.
func (d Don) LaDonBaoHanh() bool { return strings.TrimSpace(d.MaDonGoc) != "" }

// ConBaoHanh — đã giao và hạn chưa qua. Ngày hết hạn tính là còn: khách tới
// đúng ngày cuối vẫn được nhận.
func (d Don) ConBaoHanh() bool {
	if d.BaoHanhDen == "" || d.TrangThai != TTDaGiao {
		return false
	}
	return d.BaoHanhDen >= time.Now().Format("2006-01-02")
}

// DonBaoHanhCua — các đơn bảo hành đã mở từ một đơn gốc, mới nhất trước.
func DonBaoHanhCua(maGoc string) []*Don {
	maGoc = strings.ToUpper(strings.TrimSpace(maGoc))
	if maGoc == "" {
		return nil
	}
	donMu.RLock()
	defer donMu.RUnlock()
	var ra []*Don
	for _, d := range donDs {
		if strings.EqualFold(d.MaDonGoc, maGoc) {
			ra = append(ra, d)
		}
	}
	sort.Slice(ra, func(i, j int) bool { return ra[i].Ma > ra[j].Ma })
	return ra
}
