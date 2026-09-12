// Xuất CSV: danh sách đơn và sổ thống kê của một kỳ.
//
// Vì sao CSV chứ không phải xlsx: Excel mở CSV được, Google Sheets mở CSV
// được, mà sinh CSV không cần thêm một thư viện nào vào binary. Đổi lại phải
// cẩn thận hai chỗ Excel hay làm hỏng, và cả hai đều xử ở core/ky.go:
//
//	BOM UTF-8 — thiếu nó Excel bản Việt đọc "Vợt" thành "Vá»£t".
//	Ô bọc nháy — thiếu nó Excel nuốt số 0 đầu của số điện thoại.
//
// Tiền ghi bằng SỐ TRẦN, không kèm "đ" và không chấm ngăn nghìn: người tải
// về là để cộng, mà cột có chữ thì Excel không cộng được.
package core

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// guiCSV gắn header rồi đẩy chuỗi xuống. Tên tệp có sẵn kỳ và ngày xuất, để
// tải năm lần thì ra năm tệp khác tên chứ không phải "don (4).csv".
func guiCSV(w http.ResponseWriter, ten string, than string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, ten))
	w.Header().Set("Cache-Control", "no-store")
	w.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM
	w.Write([]byte(than))
}

func tenTepCSV(dau string, k Ky) string {
	moc := k.Moc
	if moc == "" {
		moc = "tat-ca"
	}
	return fmt.Sprintf("%s-%s-%s.csv", dau, moc, time.Now().Format("20060102-1504"))
}

// hQtXuatDon — đúng bảng đang hiện trên tab đơn, dưới dạng tệp.
//
// Thợ xuất được, nhưng chỉ ra mấy đơn thợ vốn đã nhìn thấy: donTheoQuyen là
// cùng một hàm trang danh sách gọi.
//
// Cột tiền cũng đúng bằng cột trên màn: tổng và còn nợ thì thợ đang nhìn
// thấy sẵn trong bảng — thợ giao hàng phải biết thu của khách bao nhiêu.
// Cắt hai cột ấy khỏi tệp chỉ làm tệp nói dối về cái bảng nó chép lại. Hai
// cột còn lại mới là chuyện của chủ: đã thu bao nhiêu nằm ở sổ tiền, còn
// trả tiệm ngoài là giá vốn, xem ghi chú đầu quantri.go.
func hQtXuatDon(loai string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		nd, _ := NguoiDangNhap(r)
		k := DocKyTu(r)
		ds := donTheoQuyen(nd, bolocTuURL(r, loai, nd))

		var b strings.Builder
		dau := []string{"Mã đơn", "Ngày nhận", "Ngày giao", "Loại", "Khách", "Liên hệ",
			"Email", "Món", "Trạng thái", "Thợ", "Tiệm ngoài", "Hẹn trả", "Nguồn",
			"Tổng tiền", "Còn nợ"}
		if nd.LaChu() {
			dau = append(dau, "Đã thu", "Trả tiệm ngoài")
		}
		dongCSV(&b, dau...)

		for _, d := range ds {
			o := []string{
				d.Ma, d.Ngay, d.NgayGiao(), TenLoaiDon(d.Loai),
				d.KhachTen, d.KhachLienHe, d.KhachEmail, d.TenMon(),
				TrangThaiCua(d.TrangThai).Ten, d.ThoPhuTrach, d.TenDoiTac(),
				d.HenTraNgay, d.NguonDonHopLe(),
				strconv.Itoa(d.TongTien), strconv.Itoa(d.ConNo()),
			}
			if nd.LaChu() {
				o = append(o, strconv.Itoa(d.DaThu()), strconv.Itoa(d.GuiDi.TraDoiTac))
			}
			dongCSV(&b, o...)
		}
		guiCSV(w, tenTepCSV("don-"+LoaiHopLe(loai), k), b.String())
	}
}

// hQtXuatThongKe — sổ của một kỳ, ba khối nối nhau trong cùng một tệp.
//
// Ba bảng trong một tệp chứ không phải ba tệp: người xuất ra là để dán vào
// một trang tính rồi nhìn cả cụm. Excel đọc dòng trống giữa hai khối là hết
// bảng và bắt đầu bảng mới, nên cách này không làm hỏng gì.
func hQtXuatThongKe(w http.ResponseWriter, r *http.Request) {
	k := DocKyTu(r)
	loai := locLoaiTuy(r.URL.Query().Get("loai"))
	tk := LayThongKeKy(loai, k)

	ten := "cả vợt lẫn giày"
	if loai != "" {
		ten = "đơn " + TenLoaiDon(loai)
	}

	var b strings.Builder
	dongCSV(&b, "Thống kê", k.Ten, ten)
	dongCSV(&b, "Xuất lúc", time.Now().Format("02/01/2006 15:04"))
	dongCSV(&b, "")

	dongCSV(&b, "Chỉ số", "Giá trị", "Đếm theo")
	dongCSV(&b, "Đơn nhận trong kỳ", strconv.Itoa(tk.NhanDon), "ngày nhận đơn")
	dongCSV(&b, "Đơn từ chối", strconv.Itoa(tk.TuChoi), "ngày nhận đơn")
	dongCSV(&b, "Tỷ lệ từ chối (%)", strconv.Itoa(tk.TyLeTuChoi), "ngày nhận đơn")
	dongCSV(&b, "Đơn giao trong kỳ", strconv.Itoa(tk.GiaoDon), "ngày giao")
	dongCSV(&b, "Doanh thu", strconv.Itoa(tk.DoanhThu), "ngày giao")
	dongCSV(&b, "Đã thu", strconv.Itoa(tk.DaThu), "ngày giao")
	dongCSV(&b, "Khách còn nợ", strconv.Itoa(tk.ConNo), "ngày giao")
	dongCSV(&b, "Trả tiệm ngoài", strconv.Itoa(tk.TraDoiTac), "ngày giao")
	dongCSV(&b, "Lãi gộp (sau tiền tiệm ngoài)", strconv.Itoa(tk.LaiGop), "ngày giao")
	dongCSV(&b, "Giá vốn vật tư (ước tính)", strconv.Itoa(tk.VatTu), "ngày giao")
	dongCSV(&b, "Lãi sau vật tư", strconv.Itoa(tk.LaiSauVatTu), "ngày giao")
	dongCSV(&b, "Trung bình một đơn", strconv.Itoa(tk.TBMotDon), "ngày giao")
	dongCSV(&b, "Đơn bảo hành nhận lại", strconv.Itoa(tk.DonBaoHanh), "ngày nhận đơn")
	dongCSV(&b, "Tỷ lệ làm lại (%)", strconv.Itoa(tk.TyLeBaoHanh), "ngày nhận đơn")
	dongCSV(&b, "")

	dongCSV(&b, "Mốc", "Đơn nhận", "Đơn giao", "Doanh thu", "Đã thu")
	for _, d := range tk.Dong {
		dongCSV(&b, d.Ten, strconv.Itoa(d.NhanDon), strconv.Itoa(d.GiaoDon),
			strconv.Itoa(d.DoanhThu), strconv.Itoa(d.DaThu))
	}
	dongCSV(&b, "")

	dongCSV(&b, "Dịch vụ", "Số lần", "Thành tiền")
	for _, v := range tk.TheoDichVu {
		dongCSV(&b, v.Ten, strconv.Itoa(v.So), strconv.Itoa(v.SoTien))
	}
	dongCSV(&b, "")

	dongCSV(&b, "Trạng thái (đơn nhận trong kỳ)", "Số đơn")
	for _, t := range tk.TheoTrangThai {
		dongCSV(&b, t.Ten, strconv.Itoa(t.So))
	}

	guiCSV(w, tenTepCSV("thong-ke", k), b.String())
}
