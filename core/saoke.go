// Đọc sao kê ngân hàng dán vào ô văn bản, đề xuất khoản thu để người bấm duyệt.
//
// Không có định dạng sao kê chuẩn nào ở Việt Nam: app mỗi ngân hàng xuất một
// kiểu, CSV tải về lại một kiểu khác. Nên ở đây KHÔNG cố hiểu cấu trúc cột.
// Mỗi dòng chỉ đi tìm ba thứ — ngày, số tiền, mã đơn — bằng cách quét ký tự,
// rồi giao lại cho người xác nhận. Đoán sai một dòng thì người sửa ô đó rồi
// bấm; đoán sai mà tự ghi vào sổ thì phải đi dò sổ mới biết.
//
// Ba thứ file này TỪ CHỐI làm:
//   - Không tự ghi khoản nào. Kết quả chỉ là đề xuất.
//   - Không đoán bừa khi một dòng có nhiều số tiền (ghi nợ / ghi có / số dư
//     nằm cùng dòng). Đánh dấu cần kiểm tra và để người nhìn.
//   - Không nhận lại dòng đã ghi. Dán sao kê hai lần là chuyện sẽ xảy ra, và
//     một khoản thu vào sổ hai lần thì lãi tháng đó sai mà nhìn vẫn hợp lý.
package core

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// TV-2609-001, chấp nhận thiếu gạch hoặc thay bằng khoảng trắng.
var reMaDon = regexp.MustCompile(`(?i)TV[\s\-_.]?(\d{4})[\s\-_.]?(\d{3})`)

var (
	reNgayDMY = regexp.MustCompile(`\b(\d{1,2})[/\-.](\d{1,2})[/\-.](\d{4}|\d{2})\b`)
	reNgayYMD = regexp.MustCompile(`\b(\d{4})[/\-.](\d{1,2})[/\-.](\d{1,2})\b`)
	reSoTien  = regexp.MustCompile(`([+\-]?)\s?(\d{1,3}(?:[.,]\d{3})+|\d{4,})(?:[.,]\d{2})?\b`)
)

type DongSaoKe struct {
	NoiDung   string
	Ngay      string
	SoTien    int
	MaDon     string
	TenKhach  string // lấy từ đơn khớp được, để người nhìn cho chắc
	ConNo     int
	TrangThai string // "khop" | "chua_ro" | "da_ghi" | "bo_qua"
	LyDo      string
	Chon      bool // đề xuất tích sẵn hay không
}

func (d DongSaoKe) DaGhi() bool  { return d.TrangThai == "da_ghi" }
func (d DongSaoKe) ChuaRo() bool { return d.TrangThai == "chua_ro" }

// DocSaoKe tách từng dòng. Trả về theo đúng thứ tự đã dán để người đối chiếu
// bằng mắt với màn hình ngân hàng.
func DocSaoKe(text string) []DongSaoKe {
	out := []DongSaoKe{}
	for _, dong := range strings.Split(text, "\n") {
		s := strings.TrimSpace(strings.ReplaceAll(dong, "\t", " "))
		if s == "" {
			continue
		}
		d := docMotDongSaoKe(s)
		if d.SoTien <= 0 {
			continue // dòng tiêu đề, dòng tổng, dòng trống có chữ
		}
		out = append(out, d)
	}
	return out
}

func docMotDongSaoKe(s string) DongSaoKe {
	d := DongSaoKe{NoiDung: s}

	d.Ngay = timNgay(s)
	if d.Ngay == "" {
		d.Ngay = time.Now().Format("2006-01-02")
		d.LyDo = "không thấy ngày, tạm lấy hôm nay"
	}

	tien, nhieu := timSoTien(s, d.Ngay)
	d.SoTien = tien
	if nhieu {
		d.LyDo = them(d.LyDo, "dòng có nhiều số tiền")
	}

	if m := reMaDon.FindStringSubmatch(s); m != nil {
		d.MaDon = strings.ToUpper("TV-" + m[1] + "-" + m[2])
	}

	switch {
	case d.MaDon == "":
		d.TrangThai = "chua_ro"
		d.LyDo = them(d.LyDo, "không thấy mã đơn")
		if goiY := donTheoSoTien(d.SoTien); goiY != nil {
			d.MaDon, d.TenKhach, d.ConNo = goiY.Ma, goiY.KhachTen, goiY.ConNo()
			d.LyDo = them(d.LyDo, "đoán theo số tiền còn nợ khớp đúng")
		}
	default:
		don, co := LayDon(d.MaDon)
		if !co {
			d.TrangThai = "chua_ro"
			d.LyDo = them(d.LyDo, "không có đơn "+d.MaDon)
			d.MaDon = ""
			break
		}
		d.TenKhach, d.ConNo = don.KhachTen, don.ConNo()
		d.TrangThai = "khop"
		d.Chon = true
		if d.SoTien > don.ConNo() && don.ConNo() > 0 {
			d.LyDo = them(d.LyDo, "nhiều hơn số còn nợ")
		}
	}

	if daGhiRoi(d) {
		d.TrangThai = "da_ghi"
		d.Chon = false
		d.LyDo = "sổ đã có khoản này"
	}
	return d
}

func them(cu, moi string) string {
	if cu == "" {
		return moi
	}
	return cu + "; " + moi
}

func timNgay(s string) string {
	if m := reNgayYMD.FindStringSubmatch(s); m != nil {
		return ghepNgay(m[1], m[2], m[3])
	}
	if m := reNgayDMY.FindStringSubmatch(s); m != nil {
		nam := m[3]
		if len(nam) == 2 {
			nam = "20" + nam
		}
		return ghepNgay(nam, m[2], m[1])
	}
	return ""
}

func ghepNgay(nam, thang, ngay string) string {
	t, err := time.Parse("2006-1-2", nam+"-"+thang+"-"+ngay)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

// timSoTien trả về số tiền đoán được và cờ "dòng này có nhiều ứng viên".
//
// Bỏ qua chuỗi số nằm trong ngày (đã tìm được ở trên) và trong mã đơn. Khi
// còn nhiều ứng viên — sao kê dạng bảng có cả ghi nợ, ghi có, số dư — lấy số
// NHỎ NHẤT: số dư tài khoản gần như luôn lớn hơn từng giao dịch, còn đoán
// trúng số dư thì ghi vào sổ một khoản thu vài chục triệu không có thật.
// Dù sao dòng đó cũng đã bị đánh dấu để người kiểm.
func timSoTien(s string, ngay string) (int, bool) {
	sach := reMaDon.ReplaceAllString(s, " ")
	sach = reNgayYMD.ReplaceAllString(sach, " ")
	sach = reNgayDMY.ReplaceAllString(sach, " ")

	ungVien := []int{}
	coDauCong := 0
	for _, m := range reSoTien.FindAllStringSubmatch(sach, -1) {
		if m[1] == "-" {
			continue // tiền ra khỏi tài khoản, không phải khách trả
		}
		n, err := strconv.Atoi(strings.NewReplacer(".", "", ",", "").Replace(m[2]))
		if err != nil || n < 1000 {
			continue
		}
		if m[1] == "+" {
			if coDauCong == 0 {
				coDauCong = n
			}
			continue
		}
		ungVien = append(ungVien, n)
	}
	if coDauCong > 0 {
		// Dấu cộng là tín hiệu rõ ràng nhất của một khoản tiền vào.
		return coDauCong, false
	}
	if len(ungVien) == 0 {
		return 0, false
	}
	nho := ungVien[0]
	for _, n := range ungVien[1:] {
		if n < nho {
			nho = n
		}
	}
	return nho, len(ungVien) > 1
}

// donTheoSoTien — chỉ nhận khi đúng MỘT đơn đang chạy còn nợ đúng số đó.
// Hai đơn cùng nợ 200k thì không đoán, vì đoán sai là ghi tiền của khách này
// sang đơn của khách kia.
func donTheoSoTien(soTien int) *Don {
	if soTien <= 0 {
		return nil
	}
	var thay *Don
	for _, d := range LocDon(BoLoc{ChiDangChay: true}) {
		if d.ConNo() != soTien {
			continue
		}
		if thay != nil {
			return nil
		}
		thay = d
	}
	return thay
}

// daGhiRoi — cùng đơn, cùng số tiền, cùng ngày (lệch một ngày vẫn tính là
// một, vì ngày ghi sổ và ngày ngân hàng hạch toán hay lệch nhau).
func daGhiRoi(d DongSaoKe) bool {
	if d.MaDon == "" || d.SoTien <= 0 {
		return false
	}
	moc, err := time.Parse("2006-01-02", d.Ngay)
	if err != nil {
		return false
	}
	for _, k := range CacLanThuCuaDon(d.MaDon) {
		if k.SoTien != d.SoTien {
			continue
		}
		kt, err := time.Parse("2006-01-02", k.Ngay)
		if err != nil {
			continue
		}
		if lech := kt.Sub(moc); lech > -36*time.Hour && lech < 36*time.Hour {
			return true
		}
	}
	return false
}
