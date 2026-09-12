// Kỳ xem số: một ngày, một tháng, hay một năm.
//
// Cả trang quản lý đơn dùng chung đúng một bộ tham số trên URL — ky=thang,
// moc=2026-09 — nên chuyển tab không mất chỗ đang xem, và nút xuất CSV chỉ
// việc chép nguyên query string. Mọi nơi khác trong core đọc kỳ qua Ky đã
// dựng sẵn ở đây, không tự cắt chuỗi ngày lần nữa.
//
// Mốc để rỗng nghĩa là "kỳ đang diễn ra" — hôm nay, tháng này, năm nay. Tiện
// cho cái bookmark: dán /qt/thong-ke?ky=thang vào thanh dấu trang thì tháng
// sau mở ra vẫn đúng tháng sau.
package core

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	KyNgay  = "ngay"
	KyThang = "thang"
	KyNam   = "nam"
	KyTatCa = "" // không giới hạn
)

// Ky — một khoảng ngày đã chốt hai đầu, cộng đủ thứ để vẽ lại thanh lọc.
type Ky struct {
	Kieu string // ngay | thang | nam | rỗng
	Moc  string // 2026-09-12 | 2026-09 | 2026
	// MocNgay — một ngày ISO đầy đủ nằm trong kỳ, cho ô <input type="date">.
	// Thanh lọc chỉ có ĐÚNG MỘT ô ngày cho cả ba kiểu kỳ: chọn "tháng" rồi
	// trỏ vào ngày nào trong tháng cũng ra cả tháng ấy. Làm ba ô date/month/
	// number đổi qua lại thì phải có JS, mà thanh lọc hỏng vì JS là thanh lọc
	// vô dụng.
	MocNgay string
	Tu      string // ISO, tính cả ngày này
	Den     string // ISO, tính cả ngày này
	Ten     string // câu hiện cho người đọc
}

// KhongGioiHan — kỳ rỗng, tức xem hết từ ngày mở trạm.
func (k Ky) KhongGioiHan() bool { return k.Tu == "" && k.Den == "" }

// Truy — query string để nút và link giữ nguyên kỳ đang xem. Có dấu & ở đầu
// khi không rỗng, để ghép thẳng vào sau một tham số khác.
func (k Ky) Truy() string {
	if k.Kieu == KyTatCa {
		return ""
	}
	return "&ky=" + urlEsc(k.Kieu) + "&moc=" + urlEsc(k.Moc)
}

// Q — y hệt Truy nhưng không có dấu & mở đầu, để ghép ngay sau dấu ?.
func (k Ky) Q() string { return strings.TrimPrefix(k.Truy(), "&") }

// DocKy dựng kỳ từ hai tham số URL. Mốc sai cú pháp KHÔNG báo lỗi mà rơi về
// kỳ đang diễn ra: đây là thanh lọc trên trang xem số, không phải biểu mẫu
// nhập liệu — gõ hỏng URL thì thấy tháng này, chứ không thấy trang lỗi.
func DocKy(kieu, moc string) Ky {
	nay := time.Now()
	moc = strings.TrimSpace(moc)
	// Cắt bớt cho vừa kiểu kỳ: ô ngày trên thanh lọc luôn gửi lên một ngày
	// đầy đủ, kể cả khi đang xem theo tháng hay theo năm.
	cat := func(n int) string {
		if len(moc) >= n {
			return moc[:n]
		}
		return moc
	}

	switch kieu {
	case KyNgay:
		t, err := time.Parse("2006-01-02", cat(10))
		if err != nil {
			t = nay
		}
		ng := t.Format("2006-01-02")
		return Ky{Kieu: KyNgay, Moc: ng, MocNgay: ng, Tu: ng, Den: ng, Ten: "ngày " + ngayDep(ng)}

	case KyThang:
		t, err := time.Parse("2006-01", cat(7))
		if err != nil {
			t = nay
		}
		dau := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
		cuoi := dau.AddDate(0, 1, -1)
		return Ky{
			Kieu: KyThang, Moc: dau.Format("2006-01"), MocNgay: dau.Format("2006-01-02"),
			Tu: dau.Format("2006-01-02"), Den: cuoi.Format("2006-01-02"),
			Ten: fmt.Sprintf("tháng %d/%d", int(dau.Month()), dau.Year()),
		}

	case KyNam:
		t, err := time.Parse("2006", cat(4))
		if err != nil {
			t = nay
		}
		return Ky{
			Kieu: KyNam, Moc: t.Format("2006"), MocNgay: fmt.Sprintf("%04d-01-01", t.Year()),
			Tu: fmt.Sprintf("%04d-01-01", t.Year()), Den: fmt.Sprintf("%04d-12-31", t.Year()),
			Ten: fmt.Sprintf("năm %d", t.Year()),
		}
	}
	return Ky{Kieu: KyTatCa, MocNgay: nay.Format("2006-01-02"), Ten: "từ trước tới nay"}
}

// DocKyTu — đọc kỳ thẳng từ query string. Mọi handler của khu quản lý đơn
// gọi đúng hàm này, để tên tham số chỉ được viết ra một lần.
func DocKyTu(r *http.Request) Ky {
	q := r.URL.Query()
	return DocKy(q.Get("ky"), q.Get("moc"))
}

// trongKhoang — ngày ISO nằm trong [tu, den], hai đầu đều tính. So bằng
// chuỗi: định dạng YYYY-MM-DD xếp theo thứ tự chữ cũng là xếp theo thời
// gian, khỏi parse hàng nghìn lần khi quét cả kho đơn.
//
// Ngày rỗng (đơn thiếu trường Ngay) rơi ra ngoài MỌI kỳ có giới hạn, nhưng
// vẫn có mặt khi không lọc — không kỳ nào nhận nó thì đúng hơn là nhét bừa
// vào kỳ hiện tại.
func trongKhoang(ngay, tu, den string) bool {
	if tu == "" && den == "" {
		return true
	}
	if len(ngay) < 10 {
		return false
	}
	ngay = ngay[:10]
	if tu != "" && ngay < tu {
		return false
	}
	if den != "" && ngay > den {
		return false
	}
	return true
}

// --- CSV --------------------------------------------------------------

// oCSV — một ô CSV đã bọc dấu nháy. Luôn bọc, kể cả ô không có dấu phẩy:
// số điện thoại 0912... mà để trần thì Excel đọc thành số rồi nuốt số 0 đầu.
func oCSV(s string) string {
	s = strings.ReplaceAll(s, "\r\n", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, `"`, `""`)
	return `"` + s + `"`
}

func dongCSV(b *strings.Builder, o ...string) {
	for i, s := range o {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(oCSV(s))
	}
	// CRLF: Excel trên Windows là chỗ tệp này sẽ được mở.
	b.WriteString("\r\n")
}
