package core

// Trang xem nhật ký. Chỉ chủ — nhật ký cho biết ai làm gì lúc nào, thợ không
// cần và không nên xem lịch sử của người khác.

import (
	"net/http"
	"strings"
	"time"
)

// Số dòng nạp một lần. Nhật ký một tháng của trạm này cỡ vài trăm dòng; 500
// đủ xem hết mà không dựng ra một trang HTML nặng vài megabyte.
const nhatKyMoiTrang = 500

type dlNhatKy struct {
	dlQt
	Thang    string
	CacThang []string
	Muc      []MucNhatKy
	BiCat    bool
}

// thangCoNhatKy liệt kê 12 tháng gần nhất, không dò thư mục: file tháng nào
// chưa có thì DocNhatKy trả rỗng, và một danh sách ổn định dễ hiểu hơn một
// danh sách nhảy theo file.
func thangCoNhatKy() []string {
	var ds []string
	m := time.Now()
	for i := 0; i < 12; i++ {
		ds = append(ds, m.Format("2006-01"))
		m = m.AddDate(0, -1, 0)
	}
	return ds
}

func hQtNhatKy(w http.ResponseWriter, r *http.Request) {
	thang := strings.TrimSpace(r.URL.Query().Get("thang"))
	if thang == "" {
		thang = time.Now().Format("2006-01")
	}

	// Lấy dư một dòng để biết còn nữa hay không.
	muc := DocNhatKy(thang, nhatKyMoiTrang+1)
	biCat := len(muc) > nhatKyMoiTrang
	if biCat {
		muc = muc[:nhatKyMoiTrang]
	}

	render(w, "qt-nhatky.html", dlNhatKy{
		dlQt:     dlQt{Chung: chung(r, "nhat-ky")},
		Thang:    thang,
		CacThang: thangCoNhatKy(),
		Muc:      muc,
		BiCat:    biCat,
	})
}
