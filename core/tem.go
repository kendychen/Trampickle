// Tem dán lên đồ. Trang HTML + Ctrl+P, cùng lối với core/hoadon.go: trình
// duyệt đã biết in, nhét thư viện PDF vào binary chỉ để làm lại việc ấy là
// đổi 30MB lấy một bố cục xấu hơn.
//
// Vì sao khổ 50×30mm: đó là cuộn tem nhiệt rẻ nhất và phổ biến nhất ở chợ
// máy in mã vạch, và là khổ dán vừa lên cán vợt mà không quấn quá nửa vòng.
// To hơn thì vướng tay cầm, nhỏ hơn thì mã đơn phải xuống dưới 9pt — máy in
// nhiệt in chữ nhỏ hơn thế là nhoè, dán lên cán vợt cầm mấy hôm thì mất chữ.
// Ai chưa mua máy in tem thì mở bản A4 (?kho=a4): 24 tem một tờ, cắt tay.
//
// Vì sao KHÔNG in QR: trạm chưa có máy quét, mà quét bằng điện thoại thì vẫn
// phải mở trình duyệt rồi đăng nhập mới ra đơn — chậm hơn đọc mã bằng mắt
// rồi gõ vào ô tìm. Bốn thứ trên tem là bốn thứ dùng để tra bằng mắt: mã đơn
// (to nhất), tên khách + 4 số cuối, hẹn trả, tên món.
//
// Số điện thoại chỉ lấy bốn số cuối. Tem dán ngoài đồ, cây vợt nằm trên giá
// cho cả tiệm nhìn; đủ để đối chiếu khi khách tới mà không phát tán danh bạ.
package core

import (
	"net/http"
	"strings"
)

type OTem struct {
	Ma     string
	Khach  string
	So     string // bốn số cuối điện thoại
	HenTra string // dd/mm, rỗng khi chưa hẹn
	Mon    string
}

type dlTem struct {
	Chung
	Tem  []OTem
	LaA4 bool
	// Duong + Truy: để hai nút đổi khổ quay lại đúng trang này với đúng
	// danh sách đơn, không phải đoán lại từ đầu.
	Duong string
	Truy  string
	VeDon string // chỉ có khi in tem một đơn
}

func hQtTem(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	veTem(w, r, []*Don{don}, "")
}

// hQtTemNhieu: /qt/tem?ma=A&ma=B — tick mấy dòng trên bảng đơn rồi in một
// lượt. Mã lạ bị bỏ im lặng chứ không báo lỗi cả trang: in bốn tem đúng vẫn
// hơn là không in được cái nào vì một mã gõ sai.
func hQtTemNhieu(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	var ds []*Don
	var truy []string
	daCo := map[string]bool{}
	for _, ma := range r.URL.Query()["ma"] {
		ma = strings.ToUpper(strings.TrimSpace(ma))
		if ma == "" || daCo[ma] {
			continue
		}
		don, ok := LayDon(ma)
		if !ok || !xemDuocDon(nd, don) {
			continue
		}
		daCo[ma] = true
		ds = append(ds, don)
		truy = append(truy, "ma="+ma)
	}
	if len(ds) == 0 {
		http.NotFound(w, r)
		return
	}
	veTem(w, r, ds, "&"+strings.Join(truy, "&"))
}

func veTem(w http.ResponseWriter, r *http.Request, ds []*Don, truy string) {
	c := chung(r, "tem")
	c.TieuDe = "Tem " + ds[0].Ma
	if len(ds) > 1 {
		c.TieuDe = "Tem " + itoa(len(ds)) + " đơn"
	}
	d := dlTem{
		Chung: c, LaA4: r.URL.Query().Get("kho") == "a4",
		Duong: r.URL.Path, Truy: truy,
	}
	if len(ds) == 1 {
		d.VeDon = "/qt/don/" + ds[0].Ma
	}
	for _, don := range ds {
		o := OTem{
			Ma:     don.Ma,
			Khach:  strings.TrimSpace(don.KhachTen),
			So:     bonSoCuoi(don.KhachLienHe),
			HenTra: ngayNganTem(don.HenTraNgay),
			Mon:    don.TenMon(),
		}
		if o.Khach == "" {
			o.Khach = "Khách lẻ"
		}
		d.Tem = append(d.Tem, o)
	}
	render(w, "qt-tem.html", d)
}

// bonSoCuoi: "0912 345 678" → "5678". Ô liên hệ có khi là tên Zalo chứ không
// phải số — lúc ấy lấy chữ cuối cùng, vẫn đối chiếu được bằng mắt.
func bonSoCuoi(lienHe string) string {
	s := strings.TrimSpace(lienHe)
	if s == "" {
		return ""
	}
	so := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
	if so == "" {
		f := strings.Fields(s)
		return f[len(f)-1]
	}
	if len(so) > 4 {
		return so[len(so)-4:]
	}
	return so
}

// ngayNganTem: "2026-09-20" → "20/09". Bỏ năm vì tem chỉ nằm trên đồ vài
// tuần, mà mỗi ký tự bỏ đi là một cỡ chữ to thêm.
func ngayNganTem(s string) string {
	if len(s) >= 10 {
		return s[8:10] + "/" + s[5:7]
	}
	return ""
}
