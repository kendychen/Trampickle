// Ba lá thư gửi khách trong đời một cái đơn.
//
// Thợ bấm nút, xem thử, rồi mới gửi — KHÔNG tự động gửi. Lý do: mail ở đây
// mang số tiền và lời hẹn, mà số tiền thì đúng vào lúc thợ nhìn xong chứ
// không đúng vào lúc trạng thái đơn đổi. Còn Zalo thì Kendy tự nhắn tay:
// API Zalo OA đòi duyệt nội dung từng mẫu, không đáng cho một trạm nhỏ.
//
// Khác MailXacNhanYeuCau ở core/mail.go một chỗ quan trọng: thư đó gửi ngầm
// ở goroutine và nuốt lỗi, vì khách đã cầm mã trên màn hình rồi. Thư ở đây
// gửi THẲNG và trả lỗi ra màn hình — thợ vừa bấm gửi thì thợ phải biết ngay
// nó đi hay không, chứ không đứng đợi khách kêu "em không nhận được gì".
package core

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
	"strings"
)

// MauMail — một mẫu thư. Thứ tự trong CacMauMail là thứ tự đời một cái đơn,
// và cũng là thứ tự nút hiện trên trang đơn.
type MauMail struct {
	Ma  string
	Ten string
	// Khi: câu nhắc thợ nên bấm nút này lúc nào. Hiện ngay dưới nút.
	Khi string
}

const (
	MailNhanDon = "nhan"
	MailBaoGia  = "bao_gia"
	MailXong    = "xong"
)

var CacMauMail = []MauMail{
	{MailNhanDon, "Đã nhận đơn", "Ngay sau khi mở đơn, để khách có mã tra cứu."},
	{MailBaoGia, "Kết quả khám và báo giá", "Soi xong, đã điền chẩn đoán và các dòng tiền."},
	{MailXong, "Đã xong, kèm hoá đơn", "Sửa xong, trước khi giao hoặc gửi về."},
}

func mauMailCua(ma string) (MauMail, bool) {
	for _, m := range CacMauMail {
		if m.Ma == ma {
			return m, true
		}
	}
	return MauMail{}, false
}

// ThuDon — một lá thư đã dựng xong, chưa gửi.
type ThuDon struct {
	Mau     MauMail
	TieuDe  string
	HTML    string
	Chu     string
	Den     string
	GuiDuoc bool
	// VuongMac: lý do chưa gửi được, hiện thẳng cho thợ đọc. Rỗng là gửi
	// được. Ba lý do hay gặp: đơn chưa có email, chưa bật gửi mail trong cài
	// đặt, và mẫu này chưa đủ dữ liệu (báo giá mà chưa có dòng tiền nào).
	VuongMac string
}

// DungThuDon dựng lá thư cho một đơn. Không gửi, không ghi gì xuống đĩa.
func DungThuDon(kieu string, don *Don) ThuDon {
	mau, ok := mauMailCua(kieu)
	if !ok {
		return ThuDon{VuongMac: "Không có mẫu thư này"}
	}
	t := ThuDon{Mau: mau, Den: strings.TrimSpace(don.KhachEmail)}
	t.TieuDe, t.HTML, t.Chu = thanThuDon(mau.Ma, don)

	switch {
	case t.Den == "" || !HopLeEmail(t.Den):
		t.VuongMac = "Đơn này chưa có email khách. Điền ô Email trong phần Khách rồi quay lại."
	case !MailBat():
		t.VuongMac = "Chưa bật gửi mail. Vào /qt/cai-dat khai khóa Resend và địa chỉ gửi."
	case mau.Ma == MailBaoGia && len(don.DongTien) == 0:
		t.VuongMac = "Chưa có dòng tiền nào để báo giá. Điền bảng tiền trước đã."
	default:
		t.GuiDuoc = true
	}
	return t
}

// GuiThuDon dựng rồi gửi. Gửi xong ghi một mốc vào lịch sử đơn — ba tháng
// sau cãi nhau "đã báo giá chưa" thì cái mốc này là câu trả lời. KHÔNG gọi
// LuuDon: chỗ gọi tự ghi.
func GuiThuDon(kieu string, don *Don, nguoi string) error {
	t := DungThuDon(kieu, don)
	if !t.GuiDuoc {
		if t.VuongMac == "" {
			t.VuongMac = "không gửi được"
		}
		return fmt.Errorf("%s", t.VuongMac)
	}
	if err := guiMail(t.Den, t.TieuDe, t.HTML, t.Chu); err != nil {
		return err
	}
	GhiMoc(don, don.TrangThai, nguoi, "Gửi mail cho khách: "+t.Mau.Ten+" → "+t.Den)
	return nil
}

// --- Nội dung ---------------------------------------------------------

// thanThuDon trả tiêu đề, bản HTML và bản chữ trơn.
//
// Bản chữ trơn không phải cho đẹp: thiếu nó, bộ lọc thư rác chấm điểm mail
// xấu hẳn. Lý do dài hơn nằm ở core/mail.go.
//
// Mọi thứ người ta gõ vào đơn đều qua html.EscapeString. Ô chẩn đoán và ô
// tình trạng nhận chữ tự do; nhét thẳng vào HTML là mời người ta chèn thẻ
// vào lá thư do trạm đứng tên gửi.
func thanThuDon(kieu string, don *Don) (string, string, string) {
	e := html.EscapeString
	ten := CFG.ThuongHieu.Ten
	if ten == "" {
		ten = "Trạm"
	}
	mon := strings.ToLower(TenLoaiDon(don.Loai))
	chao := "Chào anh/chị"
	if t := strings.TrimSpace(don.KhachTen); t != "" {
		chao = "Chào anh/chị " + t
	}
	link := ""
	if g := strings.TrimRight(strings.TrimSpace(CFG.Email.GocWeb), "/"); g != "" && don.Token != "" {
		link = g + "/tra-cuu/" + don.Token
	}

	var h, c strings.Builder
	h.WriteString(`<div style="font-family:-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#1a1d18;max-width:560px">`)
	fmt.Fprintf(&h, `<p>%s,</p>`, e(chao))
	fmt.Fprintf(&c, "%s,\n\n", chao)

	var tieuDe string
	switch kieu {

	case MailNhanDon:
		tieuDe = fmt.Sprintf("Đã nhận %s %s — %s", mon, don.Ma, ten)
		fmt.Fprintf(&h, `<p>%s đã nhận %s của anh/chị. Thợ sẽ soi kỹ rồi báo giá trước khi làm.</p>`, e(ten), e(mon))
		fmt.Fprintf(&c, "%s đã nhận %s của anh/chị. Thợ sẽ soi kỹ rồi báo giá trước khi làm.\n\n", ten, mon)
		h.WriteString(oMaDon(don.Ma))
		fmt.Fprintf(&c, "MÃ ĐƠN: %s\n\n", don.Ma)
		if m := strings.TrimSpace(don.TenMon()); m != "" {
			fmt.Fprintf(&h, `<p style="font-size:13px;color:#666">Món đã nhận: %s</p>`, e(m))
			fmt.Fprintf(&c, "Món đã nhận: %s\n", m)
		}
		if s := strings.TrimSpace(don.TinhTrang); s != "" {
			fmt.Fprintf(&h, `<p style="font-size:13px;color:#666">Anh/chị mô tả: %s</p>`, e(s))
			fmt.Fprintf(&c, "Anh/chị mô tả: %s\n", s)
		}
		h.WriteString(`<p><b>Chưa đồng ý giá thì trạm chưa động vào.</b></p>`)
		c.WriteString("\nChưa đồng ý giá thì trạm chưa động vào.\n")

	case MailBaoGia:
		tieuDe = fmt.Sprintf("Báo giá đơn %s — %s", don.Ma, ten)
		h.WriteString(`<p>Thợ đã soi xong. Đây là kết quả và giá, anh/chị xem rồi trả lời giúp trạm nhé.</p>`)
		c.WriteString("Thợ đã soi xong. Đây là kết quả và giá, anh/chị xem rồi trả lời giúp trạm nhé.\n\n")
		if cd := strings.TrimSpace(don.ChanDoan); cd != "" {
			fmt.Fprintf(&h, `<p style="margin:18px 0;padding:12px 14px;background:#f4f5f2;border-radius:8px"><b>Thợ ghi:</b><br>%s</p>`,
				strings.ReplaceAll(e(cd), "\n", "<br>"))
			fmt.Fprintf(&c, "THỢ GHI:\n%s\n\n", cd)
		}
		h.WriteString(bangTienMail(don))
		c.WriteString(bangTienChu(don))
		if don.HenTraNgay != "" {
			fmt.Fprintf(&h, `<p style="font-size:13px;color:#666">Nếu anh/chị gật, trạm hẹn trả ngày %s.</p>`, e(ngayDep(don.HenTraNgay)))
			fmt.Fprintf(&c, "Nếu anh/chị gật, trạm hẹn trả ngày %s.\n", ngayDep(don.HenTraNgay))
		}
		h.WriteString(`<p><b>Trạm chờ anh/chị gật rồi mới làm.</b> Không đồng ý cũng không sao — trạm trả lại nguyên trạng, không tính công soi.</p>`)
		c.WriteString("\nTrạm chờ anh/chị gật rồi mới làm. Không đồng ý cũng không sao — trạm trả lại nguyên trạng, không tính công soi.\n")

	case MailXong:
		tieuDe = fmt.Sprintf("Đơn %s đã xong — %s", don.Ma, ten)
		h.WriteString(`<p>Đơn của anh/chị đã sửa xong.</p>`)
		c.WriteString("Đơn của anh/chị đã sửa xong.\n\n")
		h.WriteString(bangTienMail(don))
		c.WriteString(bangTienChu(don))
		if no := don.ConNo(); no > 0 {
			h.WriteString(khoiChuyenKhoan(don, no))
			c.WriteString(khoiChuyenKhoanChu(don, no))
		}
		if hd := LinkHoaDon(don); hd != "" {
			fmt.Fprintf(&h, `<p><a href="%s" style="display:inline-block;padding:10px 18px;background:#1a1d18;color:#fff;border-radius:8px;text-decoration:none">Xem và tải hoá đơn</a></p>`, e(hd))
			fmt.Fprintf(&c, "Hoá đơn: %s\n", hd)
		}
		if don.BaoHanhDen != "" {
			fmt.Fprintf(&h, `<p style="font-size:13px;color:#666">Bảo hành tới %s. Có gì bất thường thì nhắn trạm trước khi tự xử lý.</p>`, e(ngayDep(don.BaoHanhDen)))
			fmt.Fprintf(&c, "Bảo hành tới %s.\n", ngayDep(don.BaoHanhDen))
		}
	}

	if link != "" {
		fmt.Fprintf(&h, `<p style="font-size:13px;color:#666">Xem đơn đang tới đâu: <a href="%s">%s</a></p>`, e(link), e(link))
		fmt.Fprintf(&c, "\nXem đơn đang tới đâu: %s\n", link)
	}

	fmt.Fprintf(&h, `<p style="font-size:13px;color:#666;margin-top:26px">%s`, e(ten))
	c.WriteString("\n" + ten)
	if dt := strings.TrimSpace(CFG.ThuongHieu.LienHe.DienThoai); dt != "" {
		fmt.Fprintf(&h, ` · %s`, e(dt))
		c.WriteString(" · " + dt)
	}
	h.WriteString(`<br>Anh/chị trả lời thẳng vào thư này cũng được.</p></div>`)
	c.WriteString("\nAnh/chị trả lời thẳng vào thư này cũng được.\n")

	return tieuDe, h.String(), c.String()
}

func oMaDon(ma string) string {
	return fmt.Sprintf(`<p style="margin:22px 0;padding:14px 16px;border:1px dashed #999;border-radius:8px">`+
		`<span style="font-size:13px;color:#666">Mã đơn của anh/chị</span><br>`+
		`<b style="font-size:19px;letter-spacing:.04em">%s</b></p>`, html.EscapeString(ma))
}

// bangTienMail vẽ bảng dòng tiền cho thư. Chỉ CHÉP số đã có trên đơn rồi
// cộng lại — không dòng nào ở đây tự nghĩ ra giá. Kẻ bảng bằng style dán
// thẳng vào thẻ vì client mail nuốt sạch <style> ở đầu trang.
func bangTienMail(don *Don) string {
	if len(don.DongTien) == 0 {
		return ""
	}
	e := html.EscapeString
	var b strings.Builder
	b.WriteString(`<table style="width:100%;border-collapse:collapse;margin:18px 0;font-size:14px">`)
	for _, d := range don.DongTien {
		fmt.Fprintf(&b, `<tr><td style="padding:7px 0;border-bottom:1px solid #e6e6e6">%s</td>`+
			`<td style="padding:7px 0;border-bottom:1px solid #e6e6e6;text-align:right;white-space:nowrap">%s</td></tr>`,
			e(d.Ten), e(dinhDangTien(d.SoTien)))
	}
	fmt.Fprintf(&b, `<tr><td style="padding:9px 0"><b>Tổng</b></td>`+
		`<td style="padding:9px 0;text-align:right;white-space:nowrap"><b>%s</b></td></tr>`,
		e(dinhDangTien(don.TongTien)))
	if dt := don.DaThu(); dt > 0 {
		fmt.Fprintf(&b, `<tr><td style="padding:2px 0;color:#666">Đã nhận</td>`+
			`<td style="padding:2px 0;text-align:right;color:#666">%s</td></tr>`, e(dinhDangTien(dt)))
		fmt.Fprintf(&b, `<tr><td style="padding:2px 0"><b>Còn lại</b></td>`+
			`<td style="padding:2px 0;text-align:right"><b>%s</b></td></tr>`, e(dinhDangTien(don.ConNo())))
	}
	b.WriteString(`</table>`)
	return b.String()
}

func bangTienChu(don *Don) string {
	if len(don.DongTien) == 0 {
		return ""
	}
	var b strings.Builder
	for _, d := range don.DongTien {
		fmt.Fprintf(&b, "- %s: %s\n", d.Ten, dinhDangTien(d.SoTien))
	}
	fmt.Fprintf(&b, "TỔNG: %s\n", dinhDangTien(don.TongTien))
	if dt := don.DaThu(); dt > 0 {
		fmt.Fprintf(&b, "Đã nhận: %s\nCòn lại: %s\n", dinhDangTien(dt), dinhDangTien(don.ConNo()))
	}
	b.WriteString("\n")
	return b.String()
}

// khoiChuyenKhoan — số tài khoản dạng chữ, không có ảnh QR.
//
// Vì sao không có ảnh: binary không có thư viện vẽ QR, và core/vietqr.go cấm
// gọi dịch vụ sinh ảnh QR bên ngoài (lộ mã đơn và số tiền của khách cho bên
// thứ ba, mà họ sập thì thư trắng bóc). Nội dung chuyển khoản PHẢI là mã
// đơn — saoke.go khớp tiền về đơn bằng đúng chuỗi đó.
func khoiChuyenKhoan(don *Don, no int) string {
	nh, ok := NganHangNhan()
	if !ok {
		return ""
	}
	e := html.EscapeString
	return fmt.Sprintf(`<div style="margin:18px 0;padding:14px 16px;border:1px solid #d9d9d9;border-radius:8px">`+
		`<div style="font-size:13px;color:#666;margin-bottom:6px">Chuyển khoản cho trạm</div>`+
		`<div>%s<br>Số tài khoản: <b>%s</b><br>Chủ tài khoản: %s<br>`+
		`Số tiền: <b>%s</b><br>Nội dung: <b>%s</b></div>`+
		`<div style="font-size:12px;color:#666;margin-top:8px">Nội dung phải có mã đơn, không thì trạm không biết tiền của ai.</div></div>`,
		e(tenNHDoc(nh)), e(nh.SoTK), e(nh.ChuTK), e(dinhDangTien(no)), e(NoiDungCK(don.Ma)))
}

func khoiChuyenKhoanChu(don *Don, no int) string {
	nh, ok := NganHangNhan()
	if !ok {
		return ""
	}
	return fmt.Sprintf("CHUYỂN KHOẢN\n%s\nSố tài khoản: %s\nChủ tài khoản: %s\nSố tiền: %s\nNội dung: %s\n\n",
		tenNHDoc(nh), nh.SoTK, nh.ChuTK, dinhDangTien(no), NoiDungCK(don.Ma))
}

// tenNHDoc — tên ngân hàng để đọc. BIN lạ thì hiện nguyên mã: khách tra được
// mã số, chứ gọi sai tên ngân hàng là chuyển nhầm chỗ.
func tenNHDoc(nh NganHang) string {
	if t := TenNganHang(nh.Ma); t != "" {
		return t
	}
	return "Ngân hàng mã " + nh.Ma
}

// --- Trang xem thử và nút gửi -----------------------------------------

func hQtDonMail(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	kieu := r.URL.Query().Get("mau")
	if _, ok := mauMailCua(kieu); !ok {
		kieu = MailNhanDon
	}
	t := DungThuDon(kieu, don)
	c := chung(r, "don-mail")
	c.TieuDe = "Thư gửi khách — " + don.Ma
	render(w, "qt-mail.html", struct {
		dlQt
		Don  *Don
		Thu  ThuDon
		Than template.HTML
		Mau  []MauMail
	}{
		dlQt: dlQt{Chung: c, OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		Don:  don,
		Thu:  t,
		// Thân thư do chính tệp này dựng, mọi chữ của khách đã qua
		// html.EscapeString ở thanThuDon. Đây là chỗ DUY NHẤT trong admin
		// bơm HTML thô ra trang, nên nó phải nằm ngay cạnh hàm dựng.
		Than: template.HTML(t.HTML),
		Mau:  CacMauMail,
	})
}

func hQtDonGuiMail(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	ma := strings.ToUpper(r.PathValue("ma"))
	don, ok := LayDon(ma)
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	kieu := r.FormValue("mau")
	if err := GuiThuDon(kieu, don, nd.TenHienThi()); err != nil {
		http.Redirect(w, r, "/qt/don/"+ma+"/mail?mau="+urlEsc(kieu)+"&loi="+urlEsc("Không gửi được: "+err.Error()), http.StatusSeeOther)
		return
	}
	// Mốc lịch sử vừa ghi trong GuiThuDon, giờ mới ghi xuống đĩa. Mail đã đi
	// rồi mà lưu hụt thì mất dấu vết, nên lỗi lưu phải hiện ra chứ không nuốt.
	if err := LuuDon(don); err != nil {
		http.Redirect(w, r, "/qt/don/"+ma+"?loi="+urlEsc("Mail đã gửi nhưng ghi lịch sử hụt: "+err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/don/"+ma+"?ok="+urlEsc("Đã gửi thư cho khách."), http.StatusSeeOther)
}
