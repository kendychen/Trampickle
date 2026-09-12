// Chuyển khoản cho khách trả tiền đơn.
//
// CẢ TỆP NÀY TỒN TẠI VÌ MỘT CÂU: nội dung chuyển khoản phải mang mã đơn, vì
// saoke.go khớp tiền về đơn bằng đúng cái mã đó (reMaDon, saoke.go). Mất mã
// trong nội dung là tiền về mà không biết của ai — và người phải đi dò là
// Kendy.
//
// KHÔNG gọi dịch vụ sinh ảnh QR bên ngoài. Trang khách nạp ảnh từ máy chủ lạ
// là lộ mã đơn và số tiền của khách cho bên thứ ba, và ngày họ sập thì nút trả
// tiền trắng bóc. Sinh chuỗi tại chỗ; cho tới khi có thư viện vẽ QR trong
// binary thì trang hiện số tài khoản và nội dung dạng chữ để khách tự nhập.
package core

import (
	"fmt"
	"strings"
)

type NganHang struct {
	Ma    string // mã BIN VietQR, ví dụ 970436
	SoTK  string
	ChuTK string
}

// NoiDungCK — nội dung chuyển khoản. CHỈ mã đơn, không thêm chữ có dấu: nhiều
// app ngân hàng cắt dấu và cắt độ dài, chữ thừa đẩy mã ra khỏi khung.
func NoiDungCK(maDon string) string { return strings.ToUpper(strings.TrimSpace(maDon)) }

// NganHangNhan đọc tài khoản nhận tiền. Thiếu một trong ba thì coi như chưa
// khai — nút chuyển khoản không hiện, chỉ còn COD.
func NganHangNhan() (NganHang, bool) {
	l := LienHeHienTai()
	nh := NganHang{
		Ma:    strings.TrimSpace(l.NganHangMa),
		SoTK:  strings.TrimSpace(l.SoTaiKhoan),
		ChuTK: strings.TrimSpace(l.ChuTaiKhoan),
	}
	if nh.Ma == "" || nh.SoTK == "" || nh.ChuTK == "" {
		return NganHang{}, false
	}
	return nh, true
}

// tlv — một trường EMVCo: mã 2 ký tự, độ dài 2 chữ số, rồi giá trị.
func tlv(id, gia string) string { return fmt.Sprintf("%s%02d%s", id, len(gia), gia) }

// ChuoiVietQR — chuỗi nhúng vào mã QR, theo EMVCo/VietQR. Rỗng nếu thiếu dữ
// liệu: thà không có nút còn hơn có một mã quét ra lỗi.
func ChuoiVietQR(nh NganHang, soTien int, noiDung string) string {
	if nh.Ma == "" || nh.SoTK == "" || soTien <= 0 {
		return ""
	}
	thongTin := tlv("00", "A000000727") +
		tlv("01", tlv("00", nh.Ma)+tlv("01", nh.SoTK)) +
		tlv("02", "QRIBFTTA")
	s := tlv("00", "01") +
		tlv("01", "12") + // 12 = QR dùng một lần, có sẵn số tiền
		tlv("38", thongTin) +
		tlv("53", "704") + // VND
		tlv("54", fmt.Sprintf("%d", soTien)) +
		tlv("58", "VN") +
		tlv("62", tlv("08", noiDung))
	s += "6304" // mã và độ dài của chính trường CRC, tính vào phép CRC
	return s + crc16CCITT(s)
}

// crc16CCITT — CRC-16/CCITT-FALSE: đa thức 0x1021, khởi tạo 0xFFFF, không đảo
// bit, không XOR ra. Đúng biến thể EMVCo yêu cầu; mấy biến thể CRC-16 khác cho
// ra số khác và app ngân hàng sẽ từ chối mã.
func crc16CCITT(s string) string {
	crc := uint16(0xFFFF)
	for i := 0; i < len(s); i++ {
		crc ^= uint16(s[i]) << 8
		for j := 0; j < 8; j++ {
			if crc&0x8000 != 0 {
				crc = crc<<1 ^ 0x1021
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

// tenNganHang — mã BIN đọc thành tên người ta gọi hằng ngày.
//
// Vì sao có bảng này: hoá đơn đưa cho khách mà ghi "Ngân hàng 970436" thì
// khách phải đi tra, còn ghi "Vietcombank" là chuyển được ngay. Danh sách
// chỉ gồm mấy nhà khách hay dùng; BIN không có trong bảng thì trả rỗng và
// tờ hoá đơn hiện nguyên mã số — thiếu tên còn hơn gọi sai tên ngân hàng.
//
// BIN do NHNN cấp, không đổi theo thời gian. Đổi tên thương hiệu thì sửa ở
// đây một dòng.
var tenNganHang = map[string]string{
	"970436": "Vietcombank",
	"970415": "VietinBank",
	"970418": "BIDV",
	"970405": "Agribank",
	"970422": "MB Bank",
	"970407": "Techcombank",
	"970416": "ACB",
	"970432": "VPBank",
	"970423": "TPBank",
	"970403": "Sacombank",
	"970443": "SHB",
	"970431": "Eximbank",
	"970441": "VIB",
	"970426": "MSB",
	"970454": "VietCapital Bank",
	"970429": "SCB",
	"970448": "OCB",
	"970419": "NCB",
	"970437": "HDBank",
	"970400": "SaigonBank",
	"970409": "BacA Bank",
	"970412": "PVcomBank",
	"970414": "Oceanbank",
	"970425": "ABBANK",
	"970427": "VietABank",
	"970428": "Nam A Bank",
	"970430": "PGBank",
	"970433": "VietBank",
	"970438": "BaoViet Bank",
	"970440": "SeABank",
	"970449": "LPBank",
	"970452": "KienlongBank",
	"546034": "Cake by VPBank",
	"963388": "Timo",
}

func TenNganHang(bin string) string { return tenNganHang[strings.TrimSpace(bin)] }
