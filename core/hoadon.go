// Hoá đơn in được, gắn vào một đơn.
//
// Trang HTML chứ không phải PDF: Ctrl+P của trình duyệt in ra giấy được, mà
// "Lưu thành PDF" trong hộp thoại in cũng ra đúng tệp gửi khách được. Nhét
// một thư viện PDF vào binary chỉ để làm lại việc trình duyệt đã làm sẵn là
// đổi một tệp 30MB lấy một bố cục xấu hơn.
//
// Hai cửa vào cùng một tờ:
//
//	/qt/don/{ma}/hoa-don   — thợ và chủ, sau đăng nhập, để in tại quầy.
//	/hoa-don/{token}       — khách, mở bằng token của đơn, y hệt trang
//	                         tra cứu. Link này dán được vào mail.
//
// KHÔNG có con số nào sinh ra ở đây. Tổng tiền đã chốt sẵn trên đơn, đã thu
// hỏi sổ tiền, còn nợ là hiệu của hai số ấy — xem luật ở đầu core/donhang.go.
package core

import (
	"net/http"
	"strings"
	"time"
)

type dlHoaDon struct {
	Chung
	Don       *Don
	DaThu     int
	ConNo     int
	TamTinh   bool // đơn chưa xong: tờ này là bảng tạm tính, không phải hoá đơn
	NganHang  NganHang
	CoCK      bool
	TenNH     string
	NoiDungCK string
	LinkTra   string
	InLuc     string
	// LaKhach: đang xem qua đường /hoa-don/{token}. Giấu mấy dòng nội bộ —
	// khách không cần biết đơn đang nằm ở tiệm ngoài nào.
	LaKhach bool
}

func hQtHoaDon(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	veHoaDon(w, r, don, false)
}

// hHoaDonChoKhach — cùng tờ hoá đơn, mở bằng token. Không đăng nhập, giống
// hệt /tra-cuu/{token}: token dài và ngẫu nhiên chính là mật khẩu của đơn.
func hHoaDonChoKhach(w http.ResponseWriter, r *http.Request) {
	don, ok := LayDonTheoToken(r.PathValue("token"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	veHoaDon(w, r, don, true)
}

func veHoaDon(w http.ResponseWriter, r *http.Request, don *Don, laKhach bool) {
	c := chung(r, "hoa-don")
	c.TieuDe = "Hoá đơn " + don.Ma
	d := dlHoaDon{
		Chung:     c,
		Don:       don,
		DaThu:     don.DaThu(),
		ConNo:     don.ConNo(),
		TamTinh:   don.TrangThai != TTXong && don.TrangThai != TTDaGiao,
		NoiDungCK: NoiDungCK(don.Ma),
		InLuc:     time.Now().Format("02/01/2006 15:04"),
		LaKhach:   laKhach,
	}
	d.NganHang, d.CoCK = NganHangNhan()
	if d.CoCK {
		d.TenNH = TenNganHang(d.NganHang.Ma)
	}
	if don.Token != "" {
		d.LinkTra = goc(r) + "/tra-cuu/" + don.Token
	}
	render(w, "qt-hoadon.html", d)
}

// LinkHoaDon — đường dẫn tuyệt đối tới hoá đơn của một đơn, để dán vào mail.
// Rỗng khi chưa khai gốc web ở /qt/cai-dat: thà mail không có link còn hơn
// có một link http://localhost gửi cho khách.
func LinkHoaDon(don *Don) string {
	goc := strings.TrimRight(strings.TrimSpace(CFG.Email.GocWeb), "/")
	if goc == "" || don == nil || don.Token == "" {
		return ""
	}
	return goc + "/hoa-don/" + don.Token
}
