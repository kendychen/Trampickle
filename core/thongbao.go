package core

// Báo việc về chỗ Kendy đang cầm, ngay lúc việc xảy ra.
//
// Ba việc đáng làm rung điện thoại, không thêm:
//
//	khách gửi yêu cầu từ web  — chậm một buổi là khách nhắn tiệm khác
//	đơn tới hẹn / quá hẹn     — thứ làm mất khách nhiều nhất mà không ai kêu
//	khách chuyển khoản khớp đơn — tiền vào thì nên biết ngay
//
// Cố tình KHÔNG báo khi thợ đổi trạng thái đơn: một ngày có vài chục lần đổi,
// báo hết thì tới hôm thứ ba Kendy tắt thông báo, và mất luôn ba việc trên.
//
// Ba kênh chạy song song chứ không thay nhau: Telegram tới nhanh nhất nhưng
// dễ bị tắt tiếng trong đống nhóm chat, push chỉ có trên máy đã bật, còn mail
// là thứ đọc lại được sau một tuần. Mỗi kênh tự kiểm "mình đã khai chưa" —
// chưa khai thì im lặng bỏ qua, không tính là lỗi.
//
// Nguyên tắc giống mail.go: báo tin hỏng KHÔNG được làm hỏng việc đang làm.
// Mọi lỗi nuốt vào nhật ký, và việc gửi chạy ở goroutine riêng để khách không
// phải ngồi đợi Telegram trả lời mới thấy trang "đã nhận đơn".

import (
	"errors"
	"fmt"
	"html"
	"os"
	"strings"
	"time"
)

// Loại sự kiện. Dùng làm "tag" của thông báo trên điện thoại: cùng tag thì
// thông báo mới đè lên thông báo cũ thay vì xếp thành một chồng mười cái.
const (
	TinDonWeb   = "don-web"
	TinHenTra   = "hen-tra"
	TinKhachTra = "khach-tra"
	TinThu      = "thu"
)

// SuKien là một tin cần báo, ở dạng chưa gắn với kênh nào. Than là văn bản
// thuần nhiều dòng — Telegram và push dùng nguyên, mail bọc lại thành HTML.
type SuKien struct {
	Loai   string
	TieuDe string
	Than   string
	// Duong là đường trong site, ví dụ "/qt/don/TV-2609-001". Rỗng thì thông
	// báo bấm vào mở trang /qt.
	Duong string
}

// errKenhTat: kênh chưa khai báo. Không phải lỗi — đừng ghi nhật ký.
var errKenhTat = errors.New("kênh chưa khai")

func gocWeb() string { return GocWeb() }

// LinkDayDu — đường tuyệt đối để dán vào tin nhắn. Không khai email.goc_web
// thì trả về đường tương đối: nó vô dụng trong Telegram, nhưng vẫn nói cho
// Kendy biết phải mở trang nào.
func (sk SuKien) LinkDayDu() string {
	if sk.Duong == "" {
		return ""
	}
	if g := gocWeb(); g != "" {
		return g + sk.Duong
	}
	return sk.Duong
}

// BaoTin gửi nền, không chờ, không trả lỗi. Đây là hàm mà chỗ móc sự kiện gọi.
func BaoTin(sk SuKien) {
	if strings.TrimSpace(sk.TieuDe) == "" {
		return
	}
	go func() {
		if loi := baoTinChoXong(sk); len(loi) > 0 {
			GhiNhatKy(MucNhatKy{
				Ai: "hệ thống", Viec: "bao-tin", DoiTuong: sk.Loai,
				KetQua: strings.Join(loi, "; "),
			})
			fmt.Fprintln(os.Stderr, "[báo tin]", strings.Join(loi, "; "))
		}
	}()
}

// kenh gom một kênh gửi lại thành cái tên của nó, để vòng lặp bên dưới không
// phải viết ba lần cùng một đoạn xử lý lỗi.
type kenh struct {
	ten string
	gui func(SuKien) (string, error)
}

func cacKenh() []kenh {
	return []kenh{
		{"Telegram", guiTelegramSuKien},
		{"Push", guiPushSuKien},
		{"Email", guiMailSuKien},
	}
}

// baoTinChoXong gửi lần lượt cả ba kênh, trả về danh sách lỗi thật. Kênh chưa
// khai không vào danh sách.
func baoTinChoXong(sk SuKien) []string {
	var loi []string
	for _, k := range cacKenh() {
		if _, err := k.gui(sk); err != nil && !errors.Is(err, errKenhTat) {
			loi = append(loi, k.ten+": "+err.Error())
		}
	}
	return loi
}

// BaoTinThu — bản cho nút "Gửi thử" ở /qt/cai-dat. Chạy đồng bộ và kể lại
// từng kênh ra sao, kể cả kênh chưa khai: mục đích của nút này là nói cho
// Kendy biết cái nào đang tắt, không phải chỉ báo "xong".
//
// Cờ thứ hai là "có ít nhất một kênh thật sự đi". Cả ba kênh đều tắt mà trang
// vẫn hiện khung xanh thì Kendy tưởng đã gửi được rồi ngồi đợi tin.
func BaoTinThu(sk SuKien) (string, bool) {
	var phan []string
	di := false
	for _, k := range cacKenh() {
		mo, err := k.gui(sk)
		switch {
		case errors.Is(err, errKenhTat):
			phan = append(phan, k.ten+": "+mo)
		case err != nil:
			phan = append(phan, k.ten+": LỖI — "+err.Error())
		default:
			phan = append(phan, k.ten+": "+mo)
			di = true
		}
	}
	return strings.Join(phan, " · "), di
}

// --- Mail báo việc -------------------------------------------------------

func guiMailSuKien(sk SuKien) (string, error) {
	den := emailChu()
	if !MailBat() {
		return "chưa bật gửi mail", errKenhTat
	}
	if den == "" || !HopLeEmail(den) {
		return "chưa khai địa chỉ chủ trạm", errKenhTat
	}
	h, c := thanMailSuKien(sk)
	if err := guiMail(den, "["+TenTram()+"] "+sk.TieuDe, h, c); err != nil {
		return "", err
	}
	return "đã gửi tới " + den, nil
}

// thanMailSuKien tách khỏi việc gửi để test được nội dung mà không gọi Resend.
// Mọi chuỗi đi qua html.EscapeString: tên khách và mô tả hỏng là chữ khách gõ.
func thanMailSuKien(sk SuKien) (string, string) {
	e := html.EscapeString
	link := sk.LinkDayDu()

	var h strings.Builder
	h.WriteString(`<div style="font:15px/1.65 -apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;color:#1a1d18;max-width:560px">`)
	fmt.Fprintf(&h, `<p style="margin:0 0 14px;font-size:17px;font-weight:600">%s</p>`, e(sk.TieuDe))
	for _, d := range strings.Split(sk.Than, "\n") {
		if strings.TrimSpace(d) == "" {
			continue
		}
		fmt.Fprintf(&h, `<p style="margin:0 0 6px">%s</p>`, e(d))
	}
	if strings.HasPrefix(link, "http") {
		fmt.Fprintf(&h, `<p style="margin:20px 0 0"><a href="%s">Mở trong trang quản lý</a></p>`, e(link))
	} else if link != "" {
		fmt.Fprintf(&h, `<p style="margin:20px 0 0;color:#666">Mở <code>%s</code> trong trang quản lý.</p>`, e(link))
	}
	h.WriteString(`</div>`)

	c := sk.TieuDe + "\n\n" + sk.Than
	if link != "" {
		c += "\n\n" + link
	}
	return h.String(), c + "\n"
}

// --- Nội dung cho từng việc ----------------------------------------------

// BaoDonWebMoi — khách vừa gửi yêu cầu từ web. Gọi SAU khi đơn đã nằm an toàn
// trên đĩa: báo một cái đơn chưa lưu được thì Kendy mở link ra 404.
func BaoDonWebMoi(d *Don) {
	if d == nil {
		return
	}
	dong := []string{
		"Món: " + d.TenMon(),
		"Khách: " + gonMotDong(d.KhachTen) + " — " + gonMotDong(d.KhachLienHe),
	}
	if s := gonMotDong(d.TinhTrang); s != "" {
		dong = append(dong, "Khách kể: "+s)
	}
	if d.KenhNhan == "ship" {
		dong = append(dong, "Gửi chuyển phát về trạm.")
	}
	BaoTin(SuKien{
		Loai:   TinDonWeb,
		TieuDe: "Đơn mới từ web: " + d.Ma,
		Than:   strings.Join(dong, "\n"),
		Duong:  "/qt/don/" + d.Ma,
	})
}

// BaoKhachChuyenKhoan — đã khớp sao kê vào đơn. Một tin cho cả mẻ chứ không
// mỗi dòng một tin: dán sao kê một lần khớp mười đơn là chuyện thường, và
// mười tiếng rung liền nhau thì không ai đọc cái nào.
func BaoKhachChuyenKhoan(dong []string, tong int) {
	if len(dong) == 0 {
		return
	}
	tieuDe := "Khách chuyển khoản: " + dinhDangTien(tong)
	if len(dong) > 1 {
		tieuDe = fmt.Sprintf("Khách chuyển khoản %d đơn: %s", len(dong), dinhDangTien(tong))
	}
	BaoTin(SuKien{
		Loai:   TinKhachTra,
		TieuDe: tieuDe,
		Than:   strings.Join(dong, "\n"),
		Duong:  "/qt/tien",
	})
}

// BaoDonToiHen — gom mọi đơn đang chạy đã tới hoặc đã quá ngày hẹn thành MỘT
// tin. Một tin cho cả danh sách chứ không phải một tin mỗi đơn: năm đơn quá hẹn
// mà rung năm lần thì tới lần thứ ba Kendy đã vuốt đi cho yên, lần sau có tin
// thật cũng thế.
//
// Trả về số đơn đã kể trong tin, 0 nghĩa là không gửi gì.
func BaoDonToiHen(bayGio time.Time) int {
	sk, n := dungTinHenTra(bayGio)
	if n == 0 {
		return 0
	}
	BaoTin(sk)
	return n
}

// dungTinHenTra dựng nội dung tin. Tách khỏi BaoDonToiHen để test đọc được
// chữ trong tin mà không phải gửi thật — BaoTin bắn goroutine rồi trả về ngay.
//
// Mục thứ ba "Xong, chưa ai lấy" gộp vào ĐÚNG tin này chứ không đẻ tin mới:
// xem đầu tệp — báo nhiều thì tới hôm thứ ba Kendy tắt thông báo, mất luôn cả
// ba việc.
func dungTinHenTra(bayGio time.Time) (SuKien, int) {
	homNay := bayGio.Format("2006-01-02")
	nguongQuen := NguongTramHienTai().XongBoQuenNgay
	var toi, qua, quen []string
	for _, d := range LocDon(BoLoc{ChiDangChay: true}) {
		mo := d.Ma + " · " + d.TenMon() + " · " + gonMotDong(d.KhachTen)
		if d.XongBoQuen(bayGio, nguongQuen) {
			// Đơn đã xong thì không kể vào mục hẹn nữa: cùng một mã hiện hai
			// lần trong một tin đọc ra thành hai đơn khác nhau.
			quen = append(quen, mo+fmt.Sprintf(" (xong %d ngày)", d.NgayNamIm(bayGio)))
			continue
		}
		// So chuỗi ISO chứ không parse: "2026-09-11" < "2026-09-12" đúng theo
		// thứ tự chữ, và khỏi phải nghĩ về múi giờ lần thứ hai.
		if d.HenTraNgay == "" || d.HenTraNgay > homNay {
			continue
		}
		if d.HenTraNgay == homNay {
			toi = append(toi, mo)
		} else {
			qua = append(qua, mo+" (hẹn "+ngayGonVN(d.HenTraNgay)+")")
		}
	}
	if len(toi)+len(qua)+len(quen) == 0 {
		return SuKien{}, 0
	}

	var than []string
	them := func(dau string, ds []string) {
		if len(ds) == 0 {
			return
		}
		if len(than) > 0 {
			than = append(than, "")
		}
		than = append(than, dau)
		than = append(than, ds...)
	}
	them("QUÁ HẸN:", qua)
	them("Hẹn trả hôm nay:", toi)
	them("Xong, chưa ai lấy:", quen)

	var phan []string
	if len(qua) > 0 {
		phan = append(phan, fmt.Sprintf("%d đơn quá hẹn", len(qua)))
	}
	if len(toi) > 0 {
		phan = append(phan, fmt.Sprintf("%d đơn hẹn hôm nay", len(toi)))
	}
	if len(quen) > 0 {
		phan = append(phan, fmt.Sprintf("%d đơn xong chưa ai lấy", len(quen)))
	}
	return SuKien{
		Loai:   TinHenTra,
		TieuDe: strings.Join(phan, ", "),
		Than:   strings.Join(than, "\n"),
		Duong:  "/qt/don",
	}, len(toi) + len(qua) + len(quen)
}

// ngayGonVN — "2026-09-11" thành "11/09". Bỏ năm: tin nhắc hẹn luôn nói về
// chuyện trong vòng vài tuần, in thêm năm chỉ làm dòng dài thêm.
func ngayGonVN(iso string) string {
	t, err := time.ParseInLocation("2006-01-02", iso, time.Local)
	if err != nil {
		return iso
	}
	return t.Format("02/01")
}

// gonMotDong — cắt chữ khách gõ về một dòng vừa đọc trên màn hình khóa. Xuống
// dòng trong tiêu đề thông báo push bị cắt ngang, không phải bị bỏ qua.
func gonMotDong(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	if len(s) > 120 {
		return s[:120] + "…"
	}
	return s
}
