// Đơn sửa xong mà không ai tới lấy.
//
// Đo bằng CapNhat chứ không phải HenTraNgay: quá nửa số đơn ở tiệm không có
// ngày hẹn nào cả (khách đứng chờ, hoặc "xong thì gọi"), còn CapNhat thì
// LuuDon ghi lại mỗi lần đổi trạng thái — bấm sang "Sửa xong" là có mốc.
//
// Rổ này đắt gấp đôi so với vẻ ngoài của nó: vừa là tiền chưa thu, vừa là chỗ
// trên kệ. Nhưng nó cũng là rổ dễ gây nhiễu nhất, nên ngưỡng để người khai ở
// /qt/nguong: tiệm đông thì 3 ngày đã chật kệ, tiệm vắng 14 ngày mới đáng gọi.
package core

import "time"

// XongBoQuen — đơn đã sửa xong mà nằm im quá nguongNgay ngày.
//
// nguongNgay <= 0 tắt hẳn rổ, không phải kêu mọi đơn: người khai 0 là người
// muốn im, đừng bắt họ phải khai một số thật to.
func (d Don) XongBoQuen(bayGio time.Time, nguongNgay int) bool {
	if nguongNgay <= 0 || d.TrangThai != TTXong {
		return false
	}
	moc, err := time.ParseInLocation("2006-01-02 15:04:05", d.CapNhat, time.Local)
	if err != nil {
		// Đơn cũ chưa có CapNhat thì im, không đoán. Đoán ở đây là bịa ra một
		// danh sách dài toàn đơn từ đời nào để Kendy phải đi dọn.
		return false
	}
	return bayGio.Sub(moc) >= time.Duration(nguongNgay)*24*time.Hour
}

// NgayNamIm — đơn đã nằm im bao nhiêu ngày, để in ra trong tin nhắc. Trả 0
// nếu không đọc được mốc.
func (d Don) NgayNamIm(bayGio time.Time) int {
	moc, err := time.ParseInLocation("2006-01-02 15:04:05", d.CapNhat, time.Local)
	if err != nil {
		return 0
	}
	return int(bayGio.Sub(moc).Hours() / 24)
}
