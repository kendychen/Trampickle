package core

// Sửa danh sách dịch vụ ở /qt/dich-vu — tên, điều kiện nhận, giá ba giai
// đoạn, số ngày làm, tháng bảo hành, và mở bán từ giai đoạn nào.
//
// Ghi THẲNG vào vanhanh/bang-gia.yaml, cùng lý do đã ghi ở đầu nguong.go:
// file đó không chỉ Go đọc, src/quote.py cũng đọc đúng chỗ ấy để dựng báo
// giá. Để Go đọc một file còn Python đọc file khác là có ngày trang web niêm
// yết 250k mà báo giá gửi khách ghi 300k.
//
// Sửa theo DÒNG chứ không unmarshal rồi marshal lại, cũng cùng lý do:
// bang-gia.yaml là file người viết tay, mỗi con số có mấy dòng comment giải
// thích vì sao nó là con số đó — vì sao PHU_NHAM không niêm yết giá, vì sao
// QUAN_GRIP không được chạy quảng cáo riêng. yaml.Marshal xoá sạch chỗ đó.
//
// Go KHÔNG sinh ra con số tiền nào ở đây. Nó chép đúng cái người gõ vào ô
// nhập xuống file. Con số bán hàng vẫn do người quyết và do quote.py suy ra
// từ bảng giá — luật ấy không đổi vì có thêm cái form này.

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// dvMu giữ GIA.DichVu. Cả bảng GIA nạp một lần lúc khởi động rồi chỉ đọc,
// trừ sáu ngưỡng (đã có ngMu) và giờ thêm danh sách dịch vụ — nên mọi nơi
// đọc danh sách phải đi qua DichVuTatCa(), đừng chạm GIA.DichVu trực tiếp.
var dvMu sync.RWMutex

// DichVuTatCa trả BẢN SAO cả danh sách, kể cả việc đang ẩn với khách.
//
// Bản sao chứ không phải chính lát cắt: hDichVuMot cầm con trỏ &ds[i] rồi
// đưa thẳng vào template, mà lúc ấy trang /qt/dich-vu có thể đang ghi đè
// danh sách. Chép ra rồi thì cái template cầm là của riêng nó.
func DichVuTatCa() []DichVu {
	dvMu.RLock()
	defer dvMu.RUnlock()
	ra := make([]DichVu, len(GIA.DichVu))
	copy(ra, GIA.DichVu)
	// Sắp ngay tại đây chứ không ở từng trang: mọi nơi hiển thị dịch vụ đều đi
	// qua hàm này, nên xếp một lần là trang chủ, trang /dich-vu và bảng tích
	// việc lúc tạo phiếu cùng ăn theo. SliceStable là bắt buộc — nó giữ nguyên
	// thứ tự file cho đám chưa đánh số.
	sort.SliceStable(ra, func(i, j int) bool {
		a, b := ra[i].ThuTu, ra[j].ThuTu
		if (a == 0) != (b == 0) {
			return b == 0 // có số thì đứng trước đứa bỏ trống
		}
		return a < b
	})
	return ra
}

// --- Dữ liệu cho trang -----------------------------------------------

// ODichVu là một dịch vụ trên trang sửa.
type ODichVu struct {
	Ma           string
	Ten          string
	Nhom         string
	DoiTuong     string
	DieuKien     string
	Gia          [3]string // rỗng = chưa mở bán ở giai đoạn đó
	GiaDen       [3]string // rỗng = giá một con số, không phải một khoảng
	GiaDep       [3]string // cùng giá trị, dạng người đọc
	BaoGiaRieng  bool
	NoiBat       bool
	An           bool
	TuGiaiDoan   int
	BaoHanhThang string
	LeadTimeNgay string
	LeadTimeDen  string // rỗng = hẹn một con số, không phải một khoảng
	ThuTu        string

	// DangHien và ViSao là kết quả tính ra, không phải ô nhập. Có nó thì Kendy
	// biết ngay vì sao một việc không thấy trên web, khỏi phải đoán.
	DangHien bool
	ViSao    string
}

func oTuDichVu(d DichVu) ODichVu {
	o := ODichVu{
		Ma: d.Ma, Ten: d.Ten, Nhom: d.Nhom, DoiTuong: d.DoiTuongChuan(),
		DieuKien:    strings.TrimSpace(d.DieuKien),
		BaoGiaRieng: d.BaoGiaRieng,
		NoiBat:      d.NoiBat,
		An:          d.An,
		TuGiaiDoan:  d.TuGiaiDoan,
	}
	if o.TuGiaiDoan == 0 {
		o.TuGiaiDoan = 1
	}
	if d.BaoHanhThang > 0 {
		o.BaoHanhThang = strconv.Itoa(d.BaoHanhThang)
	}
	if d.LeadTimeNgay > 0 {
		o.LeadTimeNgay = strconv.Itoa(d.LeadTimeNgay)
	}
	if d.LeadTimeDen > 0 {
		o.LeadTimeDen = strconv.Itoa(d.LeadTimeDen)
	}
	if d.ThuTu > 0 {
		o.ThuTu = strconv.Itoa(d.ThuTu)
	}
	for i := 0; i < 3; i++ {
		if i >= len(d.Gia) || d.Gia[i] == nil {
			o.GiaDep[i] = "chưa mở"
			continue
		}
		o.Gia[i] = strconv.Itoa(*d.Gia[i])
		if *d.Gia[i] == 0 {
			o.GiaDep[i] = "tặng kèm"
		} else {
			o.GiaDep[i] = dinhDangTien(*d.Gia[i])
		}
		if den := d.GiaDenTheoGiaiDoan(i + 1); den != nil {
			o.GiaDen[i] = strconv.Itoa(*den)
			o.GiaDep[i] += " – " + dinhDangTien(*den)
		}
	}
	o.DangHien = !d.An && d.DaMo(GiaiDoan)
	switch {
	case o.DangHien:
		o.ViSao = "Đang hiện trên web."
	case d.An:
		// Nói rõ "thợ vẫn tích được": không có câu này thì tắt xong Kendy
		// tưởng việc biến mất khỏi cả trang lên phiếu, rồi bật lại cho chắc.
		o.ViSao = "Đang tắt: tạm ngừng nhận. Thợ vẫn tích được khi lên phiếu."
	case o.TuGiaiDoan > GiaiDoan:
		o.ViSao = fmt.Sprintf("Đang ẩn: mở từ giai đoạn %d, trạm đang ở giai đoạn %d.", o.TuGiaiDoan, GiaiDoan)
	default:
		o.ViSao = fmt.Sprintf("Đang ẩn: chưa có giá cho giai đoạn %d, mà cũng chưa bật ô báo giá riêng.", GiaiDoan)
	}
	return o
}

// phanSuaDuoc bỏ mấy trường tính ra (GiaDep, DangHien, ViSao) để so sánh.
//
// Phải bỏ: mở bán một việc là DangHien đổi theo, mà bản "định ghi" thì chép
// từ bản cũ nên vẫn mang giá trị cũ. So thẳng cả struct thì lần nào mở bán
// cũng báo "kiểm lại không khớp" rồi huỷ, dù file ghi ra hoàn toàn đúng.
func phanSuaDuoc(o ODichVu) ODichVu {
	o.GiaDep = [3]string{}
	o.DangHien, o.ViSao = false, ""
	// "0" và bỏ trống là một: DichVu.BaoHanhThang là int, không có khoá trong
	// file cũng ra 0. Không gộp thì mỗi lần lưu một việc có lead_time_ngay: 0
	// là chỗ kiểm lại thấy "0" một bên, "" một bên, rồi huỷ cả lượt lưu.
	if o.BaoHanhThang == "0" {
		o.BaoHanhThang = ""
	}
	if o.LeadTimeNgay == "0" {
		o.LeadTimeNgay = ""
	}
	if o.LeadTimeDen == "0" {
		o.LeadTimeDen = ""
	}
	if o.ThuTu == "0" {
		o.ThuTu = ""
	}
	return o
}

// DanhSachODichVu trả cả danh sách dạng để đổ vào form.
func DanhSachODichVu() []ODichVu {
	ds := DichVuTatCa()
	ra := make([]ODichVu, 0, len(ds))
	for _, d := range ds {
		ra = append(ra, oTuDichVu(d))
	}
	return ra
}

// --- Đọc chữ người gõ ------------------------------------------------

const tenDichVuToiDa = 80
const dieuKienToiDa = 600

// docODichVu đọc một khối ô nhập thành giá trị định ghi, kèm luật hợp lệ.
func docODichVu(cu ODichVu, f func(string) string) (ODichVu, error) {
	moi := cu
	loi := func(dang string, a ...any) error {
		return fmt.Errorf("%s — %s", cu.Ten, fmt.Sprintf(dang, a...))
	}

	moi.Ten = strings.TrimSpace(f("ten"))
	if moi.Ten == "" {
		return moi, loi("chưa điền tên việc")
	}
	if len([]rune(moi.Ten)) > tenDichVuToiDa {
		return moi, loi("tên dài quá %d ký tự", tenDichVuToiDa)
	}

	// Gộp mọi khoảng trắng về một dấu cách: khối `>` trong YAML vốn nối các
	// dòng lại bằng dấu cách, nên xuống dòng ở ô nhập không giữ được. Gộp
	// trước thì cái ghi xuống file đúng bằng cái đọc lên lần sau.
	moi.DieuKien = strings.Join(strings.Fields(f("dieu_kien")), " ")
	if len([]rune(moi.DieuKien)) > dieuKienToiDa {
		return moi, loi("điều kiện dài quá %d ký tự", dieuKienToiDa)
	}

	moi.DoiTuong = strings.TrimSpace(f("doi_tuong"))
	if moi.DoiTuong == "" {
		moi.DoiTuong = "vot"
	}
	if !LaDoiTuongHopLe(moi.DoiTuong) {
		return moi, loi("đối tượng phải là vot, giay hoặc ca_hai")
	}

	// Hai ô một giai đoạn: "từ" và "đến". Ô "đến" để trống là giá một con số,
	// y như trước khi có khoá gia_den — đó là trạng thái của mọi việc đang có.
	docGia := func(o, nhan string, i int) (string, error) {
		chu := strings.TrimSpace(f(fmt.Sprintf("%s%d", o, i+1)))
		if chu == "" {
			return "", nil
		}
		if !coChuSo(chu) {
			return "", loi("%s giai đoạn %d: %q không có chữ số nào", nhan, i+1, chu)
		}
		v := soTien(chu)
		if v < 0 || v > tienToiDa {
			return "", loi("%s giai đoạn %d phải trong khoảng 0 – %s", nhan, i+1, dinhDangTien(tienToiDa))
		}
		return strconv.Itoa(v), nil
	}
	for i := 0; i < 3; i++ {
		var err error
		if moi.Gia[i], err = docGia("gia", "giá", i); err != nil {
			return moi, err
		}
		if moi.GiaDen[i], err = docGia("giaden", "giá đến", i); err != nil {
			return moi, err
		}
		// Khoảng giá phải là khoảng thật, cùng luật với khoảng ngày làm:
		// "150.000 đến 150.000" in ra vẫn là một con số, "150.000 đến 100.000"
		// thì trang khách in ngược. Và đầu trên một mình thì không có nghĩa —
		// báo giá Python đọc khoá `gia`, không đọc khoá này.
		if moi.GiaDen[i] == "" {
			continue
		}
		if moi.Gia[i] == "" {
			return moi, loi("giai đoạn %d: điền ô giá \"đến\" thì phải điền cả ô giá \"từ\"", i+1)
		}
		tu, _ := strconv.Atoi(moi.Gia[i])
		den, _ := strconv.Atoi(moi.GiaDen[i])
		if den <= tu {
			return moi, loi("giá giai đoạn %d: số sau (%s) phải lớn hơn số trước (%s)",
				i+1, dinhDangTien(den), dinhDangTien(tu))
		}
	}

	moi.BaoGiaRieng = f("bao_gia_rieng") != ""
	moi.NoiBat = f("noi_bat") != ""
	moi.An = f("an") != ""

	tgd, err := strconv.Atoi(strings.TrimSpace(f("tu_giai_doan")))
	if err != nil || tgd < 1 || tgd > 3 {
		return moi, loi("giai đoạn mở bán phải là 1, 2 hoặc 3")
	}
	moi.TuGiaiDoan = tgd

	if moi.BaoHanhThang, err = docSoNho(f("bao_hanh_thang"), 120); err != nil {
		return moi, loi("bảo hành: %v", err)
	}
	if moi.LeadTimeNgay, err = docSoNho(f("lead_time_ngay"), 365); err != nil {
		return moi, loi("số ngày làm: %v", err)
	}
	if moi.LeadTimeDen, err = docSoNho(f("lead_time_ngay_den"), 365); err != nil {
		return moi, loi("số ngày làm (đến): %v", err)
	}
	// Khoảng ngày phải là một khoảng thật. "3 đến 3" in ra vẫn là "3 ngày",
	// giữ lại chỉ tổ để file có một khoá không làm gì; "5 đến 2" thì trang
	// khách in ngược. Điền ô sau mà bỏ trống ô trước cũng vô nghĩa: báo giá
	// Python đọc lead_time_ngay, không đọc khoá này.
	if moi.LeadTimeDen != "" {
		if moi.LeadTimeNgay == "" {
			return moi, loi("điền ô ngày làm \"đến\" thì phải điền cả ô ngày làm \"từ\"")
		}
		tu, _ := strconv.Atoi(moi.LeadTimeNgay)
		den, _ := strconv.Atoi(moi.LeadTimeDen)
		if den <= tu {
			return moi, loi("ngày làm: số sau (%d) phải lớn hơn số trước (%d)", den, tu)
		}
	}
	if moi.ThuTu, err = docSoNho(f("thu_tu"), 999); err != nil {
		return moi, loi("thứ tự: %v", err)
	}

	// Một việc không có giá ở giai đoạn nó mở bán, mà cũng không bật báo giá
	// riêng, thì bật lên xong vẫn không ai thấy. Nói ngay lúc lưu, đừng để
	// Kendy lưu thành công rồi ra trang chủ tìm mãi không thấy.
	if !moi.BaoGiaRieng && moi.Gia[moi.TuGiaiDoan-1] == "" {
		return moi, loi("mở từ giai đoạn %d thì phải điền giá giai đoạn %d, hoặc tích ô \"chưa niêm yết giá\"", moi.TuGiaiDoan, moi.TuGiaiDoan)
	}
	return moi, nil
}

func docSoNho(chu string, toiDa int) (string, error) {
	chu = strings.TrimSpace(chu)
	if chu == "" {
		return "", nil
	}
	v, err := strconv.Atoi(chu)
	if err != nil {
		return "", fmt.Errorf("%q không phải số", chu)
	}
	if v < 0 || v > toiDa {
		return "", fmt.Errorf("phải trong khoảng 0 – %d", toiDa)
	}
	if v == 0 {
		return "0", nil
	}
	return strconv.Itoa(v), nil
}

// --- Mổ dòng ---------------------------------------------------------

var (
	reMucDichVu = regexp.MustCompile(`^  - ma:\s*([A-Z0-9_]+)\s*(#.*)?$`)
	reKhoaMuc   = regexp.MustCompile(`^    ([a-z0-9_]+):`)
)

// khoiKhoaTrenCung khoanh vùng từ dòng "<ten>:" tới khoá cấp cao nhất kế
// tiếp. Khoanh vùng chứ không tìm trên cả file: tên khoá trùng nhau ở khối
// khác, mà sửa nhầm khối là hỏng đúng chỗ không ai ngờ tới.
func khoiKhoaTrenCung(dong []string, ten string) (int, int) {
	dau := -1
	for i, d := range dong {
		if dau < 0 {
			if strings.HasPrefix(d, ten+":") {
				dau = i
			}
			continue
		}
		if d != "" && !strings.HasPrefix(d, " ") && !strings.HasPrefix(d, "\t") &&
			!strings.HasPrefix(strings.TrimSpace(d), "#") {
			return dau, i
		}
	}
	if dau < 0 {
		return -1, -1
	}
	return dau, len(dong)
}

// vungMuc tìm khoảng dòng của một mã dịch vụ, tính cả comment đứng ngay
// trước nó thì KHÔNG — comment thuộc về mục nhưng ta không đụng tới nó, nên
// khoảng bắt đầu ngay ở dòng "- ma:".
func vungMuc(dong []string, dau, cuoi int, ma string) (int, int) {
	batDau := -1
	for i := dau + 1; i < cuoi && i < len(dong); i++ {
		m := reMucDichVu.FindStringSubmatch(dong[i])
		if m == nil {
			continue
		}
		if batDau >= 0 {
			// Lùi qua các dòng trống và comment đứng trước mục kế: chúng
			// thuộc về mục sau, cắt vào là đẩy comment sang nhầm chỗ.
			j := i
			for j > batDau && (strings.TrimSpace(dong[j-1]) == "" ||
				strings.HasPrefix(strings.TrimSpace(dong[j-1]), "#")) {
				j--
			}
			return batDau, j
		}
		if m[1] == ma {
			batDau = i
		}
	}
	if batDau < 0 {
		return -1, -1
	}
	// Mục cuối: sau nó là dòng trống rồi khối comment ngăn cách với khoá kế.
	// Cắt trước cả hai, không thì thêm khoá mới sẽ rơi xuống dưới comment ấy.
	j := cuoi
	for j > batDau && (strings.TrimSpace(dong[j-1]) == "" ||
		strings.HasPrefix(strings.TrimSpace(dong[j-1]), "#")) {
		j--
	}
	return batDau, j
}

// vungKhoa tìm dòng của một khoá trong mục và cả phần giá trị nhiều dòng của
// nó (khối `>` của dieu_kien trải qua mấy dòng thụt sâu hơn).
func vungKhoa(dong []string, ds, de int, khoa string) (int, int) {
	for i := ds; i < de; i++ {
		m := reKhoaMuc.FindStringSubmatch(dong[i])
		if m == nil || m[1] != khoa {
			continue
		}
		j := i + 1
		for j < de && strings.HasPrefix(dong[j], "      ") {
			j++
		}
		return i, j
	}
	return -1, -1
}

// dongFolded dựng lại khối `>` cho một chuỗi dài, gói dòng ở 76 cột.
func dongFolded(khoa, chu string) []string {
	ra := []string{"    " + khoa + ": >"}
	tu := strings.Fields(chu)
	d := ""
	for _, t := range tu {
		if d == "" {
			d = t
			continue
		}
		if len(d)+1+len(t) > 70 {
			ra = append(ra, "      "+d)
			d = t
			continue
		}
		d += " " + t
	}
	if d != "" {
		ra = append(ra, "      "+d)
	}
	return ra
}

func nhayKep(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

func dongGia(khoa string, g [3]string) string {
	p := make([]string, 3)
	for i, v := range g {
		if v == "" {
			p[i] = "null"
			continue
		}
		p[i] = v
	}
	return "    " + khoa + ": [" + strings.Join(p, ", ") + "]"
}

// coSo: mảng ba ô có ít nhất một ô điền. Mảng rỗng hoàn toàn thì không ghi
// `gia_den: [null, null, null]` xuống file — một dòng nói "không có gì" là
// một dòng thừa, và người sau đọc file sẽ tưởng khoá ấy có ý nghĩa gì đó.
func coSo(g [3]string) bool {
	for _, v := range g {
		if v != "" {
			return true
		}
	}
	return false
}

// suaMotMuc trả về các dòng mới cho vùng [ds,de) của một mục.
//
// Cách làm: gom mọi thay đổi thành danh sách "thay dòng x..y bằng z", rồi áp
// từ DƯỚI LÊN. Áp từ trên xuống thì mỗi lần chèn thêm dòng là mọi chỉ số
// phía dưới lệch đi, và cái lệch đó không ai nhìn thấy cho tới lúc file hỏng.
func suaMotMuc(dong []string, ds, de int, cu, moi ODichVu) []string {
	type sua struct {
		i, j int
		moi  []string
	}
	var ss []sua

	dat := func(khoa string, dongMoi []string, canCo bool) {
		i, j := vungKhoa(dong, ds, de, khoa)
		switch {
		case i >= 0 && canCo:
			ss = append(ss, sua{i, j, dongMoi})
		case i >= 0 && !canCo:
			ss = append(ss, sua{i, j, nil}) // xoá hẳn khoá
		case i < 0 && canCo:
			ss = append(ss, sua{de, de, dongMoi}) // chưa có thì thêm cuối mục
		}
	}

	if moi.Ten != cu.Ten {
		dat("ten", []string{"    ten: " + nhayKep(moi.Ten)}, true)
	}
	if moi.Gia != cu.Gia {
		dat("gia", []string{dongGia("gia", moi.Gia)}, true)
	}
	// gia_den đứng ngay sau gia trong file: hai khoá này đọc rời nhau thì vô
	// nghĩa. dat() thêm khoá mới vào CUỐI mục nên lần đầu điền trần giá nó sẽ
	// nằm cuối; lần lưu sau vungKhoa tìm thấy nên nó ở yên đó. Chấp nhận —
	// đổi chỗ khoá là phải dời cả khối chú thích viết tay quanh nó.
	if moi.GiaDen != cu.GiaDen {
		dat("gia_den", []string{dongGia("gia_den", moi.GiaDen)}, coSo(moi.GiaDen))
	}
	if moi.DieuKien != cu.DieuKien {
		dat("dieu_kien", dongFolded("dieu_kien", moi.DieuKien), moi.DieuKien != "")
	}
	if moi.BaoGiaRieng != cu.BaoGiaRieng {
		dat("bao_gia_rieng", []string{"    bao_gia_rieng: true"}, moi.BaoGiaRieng)
	}
	if moi.NoiBat != cu.NoiBat {
		dat("noi_bat", []string{"    noi_bat: true"}, moi.NoiBat)
	}
	// Bật lại thì XOÁ hẳn khoá chứ không ghi "an: false" — mặc định của cờ đã
	// là hiện, để lại một dòng nói đúng cái mặc định chỉ tổ làm file dài ra.
	if moi.An != cu.An {
		dat("an", []string{"    an: true"}, moi.An)
	}
	if moi.TuGiaiDoan != cu.TuGiaiDoan {
		// Giai đoạn 1 là mặc định của DaMo, ghi ra chỉ thêm dòng thừa.
		dat("tu_giai_doan", []string{"    tu_giai_doan: " + strconv.Itoa(moi.TuGiaiDoan)}, moi.TuGiaiDoan > 1)
	}
	if moi.BaoHanhThang != cu.BaoHanhThang {
		dat("bao_hanh_thang", []string{"    bao_hanh_thang: " + moi.BaoHanhThang}, moi.BaoHanhThang != "")
	}
	if moi.LeadTimeNgay != cu.LeadTimeNgay {
		dat("lead_time_ngay", []string{"    lead_time_ngay: " + moi.LeadTimeNgay}, moi.LeadTimeNgay != "")
	}
	if moi.LeadTimeDen != cu.LeadTimeDen {
		dat("lead_time_ngay_den", []string{"    lead_time_ngay_den: " + moi.LeadTimeDen}, moi.LeadTimeDen != "")
	}
	if moi.ThuTu != cu.ThuTu {
		dat("thu_tu", []string{"    thu_tu: " + moi.ThuTu}, moi.ThuTu != "")
	}
	if moi.DoiTuong != cu.DoiTuong {
		dat("doi_tuong", []string{"    doi_tuong: " + moi.DoiTuong}, moi.DoiTuong != "" && moi.DoiTuong != "vot")
	}
	if len(ss) == 0 {
		return dong
	}

	sort.Slice(ss, func(a, b int) bool { return ss[a].i > ss[b].i })
	for _, s := range ss {
		ra := make([]string, 0, len(dong)+len(s.moi))
		ra = append(ra, dong[:s.i]...)
		ra = append(ra, s.moi...)
		ra = append(ra, dong[s.j:]...)
		dong = ra
	}
	return dong
}

// SuaDichVu ghi những dịch vụ có đổi vào bang-gia.yaml. Trả tên các việc đã đổi.
func SuaDichVu(moi map[string]ODichVu) ([]string, error) {
	dvMu.Lock()
	defer dvMu.Unlock()

	b, err := os.ReadFile(fileBangGia())
	if err != nil {
		return nil, fmt.Errorf("đọc bảng giá: %w", err)
	}
	noi := string(b)
	crlf := strings.Contains(noi, "\r\n")
	dong := strings.Split(strings.ReplaceAll(noi, "\r\n", "\n"), "\n")

	// Sửa từ mục CUỐI ngược lên: mỗi lần thêm hay bớt dòng là các mục phía
	// dưới lệch chỉ số, mục phía trên thì không.
	var doi []string
	dinh := map[string]ODichVu{}
	cuTatCa := GIA.DichVu
	for k := len(cuTatCa) - 1; k >= 0; k-- {
		cu := oTuDichVu(cuTatCa[k])
		m, co := moi[cu.Ma]
		if !co {
			continue
		}
		if phanSuaDuoc(m) == phanSuaDuoc(cu) {
			continue
		}
		dau, cuoi := khoiKhoaTrenCung(dong, "dich_vu")
		if dau < 0 {
			return nil, fmt.Errorf("không tìm thấy khối dich_vu: trong %s", fileBangGia())
		}
		ds, de := vungMuc(dong, dau, cuoi, cu.Ma)
		if ds < 0 {
			return nil, fmt.Errorf("không thấy mục %s trong bảng giá", cu.Ma)
		}
		dong = suaMotMuc(dong, ds, de, cu, m)
		dinh[cu.Ma] = m
		doi = append(doi, m.Ten)
	}
	if len(doi) == 0 {
		return nil, nil
	}

	ra := strings.Join(dong, "\n")
	if crlf {
		ra = strings.ReplaceAll(ra, "\n", "\r\n")
	}

	// Đọc lại bằng yaml trước khi ghi: sửa dòng làm hỏng cấu trúc, hoặc giá
	// trị rơi không đúng mục định nhắm, thì file cũ còn nguyên.
	var thu BangGia
	if err := yaml.Unmarshal([]byte(ra), &thu); err != nil {
		return nil, fmt.Errorf("sửa xong bảng giá không đọc được nữa, đã huỷ: %w", err)
	}
	if len(thu.DichVu) != len(cuTatCa) {
		return nil, fmt.Errorf("sau khi sửa còn %d dịch vụ thay vì %d, đã huỷ", len(thu.DichVu), len(cuTatCa))
	}
	for _, d := range thu.DichVu {
		muon, co := dinh[d.Ma]
		if !co {
			continue
		}
		if co := phanSuaDuoc(oTuDichVu(d)); co != phanSuaDuoc(muon) {
			return nil, fmt.Errorf("kiểm lại %s không khớp, đã huỷ", d.Ma)
		}
	}
	if err := ghiAtomic(fileBangGia(), []byte(ra)); err != nil {
		return nil, err
	}
	GIA.DichVu = thu.DichVu
	sort.Strings(doi)
	return doi, nil
}

var reMaDV = regexp.MustCompile(`^[A-Z0-9_]{2,24}$`)

func ThemDichVu(o ODichVu) error {
	dvMu.Lock(); defer dvMu.Unlock()
	if !reMaDV.MatchString(o.Ma) { return fmt.Errorf("ma %q chi gom A-Z, 0-9, _, 2-24 ky tu", o.Ma) }
	for _, d := range GIA.DichVu { if d.Ma == o.Ma { return fmt.Errorf("ma %s da co", o.Ma) } }
	if o.Ten == "" { return fmt.Errorf("chua dien ten viec") }
	if o.DoiTuong == "" { o.DoiTuong = "vot" }
	if !LaDoiTuongHopLe(o.DoiTuong) { return fmt.Errorf("doi tuong phai la vot, giay hoac ca_hai") }
	b, err := os.ReadFile(fileBangGia())
	if err != nil { return err }
	noi := string(b)
	dong := strings.Split(strings.ReplaceAll(noi, "\r\n", "\n"), "\n")
	dau, cuoi := khoiKhoaTrenCung(dong, "dich_vu")
	if dau < 0 { return fmt.Errorf("khong tim thay khoi dich_vu") }
	chen := cuoi
	for chen > dau+1 && strings.TrimSpace(dong[chen-1]) == "" { chen-- }
	giaDong := dongGia("gia", o.Gia)
	var moiDong []string
	moiDong = append(moiDong, "  - ma: "+o.Ma)
	moiDong = append(moiDong, "    ten: "+nhayKep(o.Ten))
	moiDong = append(moiDong, "    nhom: "+o.Nhom)
	if o.DoiTuong != "" && o.DoiTuong != "vot" { moiDong = append(moiDong, "    doi_tuong: "+o.DoiTuong) }
	moiDong = append(moiDong, "    "+giaDong)
	has:=false; for _,v:=range o.GiaDen{ if v!=""{has=true;break}}
	if has { moiDong=append(moiDong,"    "+dongGia("gia_den",o.GiaDen)) }
	moiDong=append(moiDong,"    vat_tu: 0")
	moiDong=append(moiDong,"    gio_cong: 0")
	if o.BaoHanhThang!="" && o.BaoHanhThang!="0" { moiDong=append(moiDong,"    bao_hanh_thang: "+o.BaoHanhThang) }
	if o.LeadTimeNgay!="" && o.LeadTimeNgay!="0" { moiDong=append(moiDong,"    lead_time_ngay: "+o.LeadTimeNgay) }
	if o.LeadTimeDen!="" { moiDong=append(moiDong,"    lead_time_ngay_den: "+o.LeadTimeDen) }
	if o.ThuTu!="" && o.ThuTu!="0" { moiDong=append(moiDong,"    thu_tu: "+o.ThuTu) }
	if o.TuGiaiDoan>1 { moiDong=append(moiDong,"    tu_giai_doan: "+strconv.Itoa(o.TuGiaiDoan)) }
	if o.DieuKien!="" { moiDong=append(moiDong,dongFolded("dieu_kien",o.DieuKien)...) }
	if o.BaoGiaRieng { moiDong=append(moiDong,"    bao_gia_rieng: true") }
	if o.NoiBat { moiDong=append(moiDong,"    noi_bat: true") }
	if o.An { moiDong=append(moiDong,"    an: true") }
	ra:=make([]string,0,len(dong)+len(moiDong))
	ra=append(ra,dong[:chen]...);ra=append(ra,moiDong...);ra=append(ra,dong[chen:]...)
	out:=strings.Join(ra,"\n")
	var thu BangGia
	if err:=yaml.Unmarshal([]byte(out),&thu);err!=nil{return fmt.Errorf("them xong bang gia khong doc duoc: %w",err)}
	if err:=ghiAtomic(fileBangGia(),[]byte(out));err!=nil{return err}
	GIA.DichVu=thu.DichVu
	return nil
}

func XoaDichVu(ma string) error {
	dvMu.Lock();defer dvMu.Unlock()
	found:=false;for _,d:=range GIA.DichVu{if d.Ma==ma{found=true;break}}
	if !found{return fmt.Errorf("khong thay %s",ma)}
	b,err:=os.ReadFile(fileBangGia())
	if err!=nil{return err}
	noi:=string(b)
	dong:=strings.Split(strings.ReplaceAll(noi, "\r\n", "\n"),"\n")
	dau,cuoi:=khoiKhoaTrenCung(dong,"dich_vu")
	if dau<0{return fmt.Errorf("khong tim thay khoi dich_vu")}
	ds,de:=vungMuc(dong,dau,cuoi,ma)
	if ds<0{return fmt.Errorf("khong thay muc %s trong file",ma)}
	ra:=make([]string,0,len(dong)-(de-ds))
	ra=append(ra,dong[:ds]...);ra=append(ra,dong[de:]...)
	out:=strings.Join(ra,"\n")
	var thu BangGia
	if err:=yaml.Unmarshal([]byte(out),&thu);err!=nil{return fmt.Errorf("xoa xong bang gia hong: %w",err)}
	if len(thu.DichVu)!=len(GIA.DichVu)-1{return fmt.Errorf("sau khi xoa con %d thay vi %d",len(thu.DichVu),len(GIA.DichVu)-1)}
	if err:=ghiAtomic(fileBangGia(),[]byte(out));err!=nil{return err}
	GIA.DichVu=thu.DichVu
	return nil
}

// --- Trang quản trị --------------------------------------------------

type dlQtDichVu struct {
	dlQt
	DichVu   []ODichVu
	Tep      string
	GiaiDoan int
	SoHien   int

	// NoiBatTat: công tắc tổng ở /qt/giao-dien đang tắt. Không chặn tích ô,
	// chỉ nói ra — tích xong ra trang chủ không thấy gì là chỗ khó đoán nhất.
	NoiBatTat bool

	// KhongBan: hai ca trạm từ chối. Không sửa được gì ở biểu mẫu này (giá,
	// ngày làm đều không có), chỉ để mở đường sang trang soạn bài của chúng —
	// không có chỗ này thì hai bài ấy không có lối vào từ admin.
	KhongBan []KhongBan
}

func hQtDichVu(w http.ResponseWriter, r *http.Request) {
	d := dlQtDichVu{dlQt: dlQt{Chung: chung(r, "dich-vu-qt")}, KhongBan: GIA.KhongBan}
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 256*1024)
		if err := r.ParseForm(); err != nil {
			d.Loi = "Không đọc được biểu mẫu: " + err.Error()
		} else {
			moi := map[string]ODichVu{}
			var loi error
			for _, o := range DanhSachODichVu() {
				if _, co := r.Form["co."+o.Ma]; !co {
					continue // mục này không có trong biểu mẫu vừa gửi
				}
				m, err := docODichVu(o, func(k string) string {
					return r.Form.Get(k + "." + o.Ma)
				})
				if err != nil {
					loi = err
					break
				}
				moi[o.Ma] = m
			}
			switch {
			case loi != nil:
				d.Loi = loi.Error()
			default:
				doi, err := SuaDichVu(moi)
				switch {
				case err != nil:
					d.Loi = err.Error()
				case len(doi) == 0:
					d.OK = "Không có gì đổi."
				default:
					d.OK = "Đã đổi: " + strings.Join(doi, "; ") + ". Có hiệu lực ngay."
				}
			}
		}
	}
	// Đọc sau khi ghi: chung() chạy từ đầu hàm nên số trong đó vẫn là số cũ.
	d.DichVu = DanhSachODichVu()
	for _, o := range d.DichVu {
		if o.DangHien {
			d.SoHien++
		}
	}
	if r.URL.Query().Get("ok") != "" && d.OK == "" { d.OK = r.URL.Query().Get("ok") }
        if r.URL.Query().Get("loi") != "" && d.Loi == "" { d.Loi = r.URL.Query().Get("loi") }
        d.Tep = CFG.DuongDan.BangGia
	d.GiaiDoan = GiaiDoan
	d.NoiBatTat = !DvNoiBatBat()
	render(w, "qt-dich-vu.html", d)
}

func hQtDichVuThem(w http.ResponseWriter, r *http.Request) {
        r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
        if err := r.ParseForm(); err != nil { http.Error(w, "biểu mãu lỗi", 400); return }
        ma := strings.ToUpper(strings.TrimSpace(r.FormValue("ma")))
        ten := strings.TrimSpace(r.FormValue("ten"))
        nhom := strings.ToUpper(strings.TrimSpace(r.FormValue("nhom")))
        if nhom == "" { nhom = "A" }
        if nhom != "A" && nhom != "B" && nhom != "C" { nhom = "A" }
        doiTuong := strings.TrimSpace(r.FormValue("doi_tuong"))
        if doiTuong == "" { doiTuong = "vot" }
        var gia [3]string
        var giaDen [3]string
        for i:=0;i<3;i++ {
                gia[i] = strings.TrimSpace(r.FormValue("gia"+strconv.Itoa(i+1)))
                giaDen[i] = strings.TrimSpace(r.FormValue("giaden"+strconv.Itoa(i+1)))
        }
        o := ODichVu{Ma: ma, Ten: ten, Nhom: nhom, DoiTuong: doiTuong, Gia: gia, GiaDen: giaDen, TuGiaiDoan: 1}
        // Lay them cac truong tuy chon
        if v:=strings.TrimSpace(r.FormValue("thu_tu")); v!="" { o.ThuTu=v }
        if v:=strings.TrimSpace(r.FormValue("dieu_kien")); v!="" { o.DieuKien=v }
        // Validate qua docODichVu de dung chung luat gia
        base := ODichVu{Ma: ma, Ten: ten, Nhom: nhom, DoiTuong: doiTuong}
        // docODichVu can cu ODichVu de lay loi prefix, va f de doc field
        m, err := docODichVu(base, func(k string) string {
                switch k {
                case "ten": return ten
                case "dieu_kien": return o.DieuKien
                case "doi_tuong": return doiTuong
                case "tu_giai_doan": return "1"
                case "bao_hanh_thang", "lead_time_ngay", "lead_time_ngay_den", "thu_tu":
                        if k=="thu_tu" { return o.ThuTu }
                        return ""
                case "bao_gia_rieng", "noi_bat", "an": return ""
                default:
                        // gia/giaden
                        for i:=0;i<3;i++ {
                                if k=="gia"+strconv.Itoa(i+1) { return gia[i] }
                                if k=="giaden"+strconv.Itoa(i+1) { return giaDen[i] }
                        }
                        return ""
                }
        })
        if err != nil {
                http.Redirect(w, r, "/qt/dich-vu?loi="+urlQueryEscape(err.Error()), http.StatusSeeOther)
                return
        }
        // Giữ thông tin đã validate
        o.Ten = m.Ten; o.DieuKien=m.DieuKien; o.DoiTuong=m.DoiTuong; o.Gia=m.Gia; o.GiaDen=m.GiaDen; o.ThuTu=m.ThuTu
        o.Nhom = nhom
        if err := ThemDichVu(o); err != nil {
                http.Redirect(w, r, "/qt/dich-vu?loi="+urlQueryEscape(err.Error()), http.StatusSeeOther)
                return
        }
        http.Redirect(w, r, "/qt/dich-vu?ok="+urlQueryEscape("Đã thêm "+ma), http.StatusSeeOther)
}

func urlQueryEscape(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "%", "%25"), "&", "%26") }

func hQtDichVuXoa(w http.ResponseWriter, r *http.Request) {
        ma := strings.ToUpper(r.PathValue("ma"))
        if ma == "" { ma = strings.ToUpper(strings.TrimSpace(r.FormValue("ma"))) }
        if !reMaDV.MatchString(ma) { http.Error(w, "mã không hợp lệ", 400); return }
        if err := XoaDichVu(ma); err != nil {
                http.Redirect(w, r, "/qt/dich-vu?loi="+urlQueryEscape(err.Error()), http.StatusSeeOther)
                return
        }
        http.Redirect(w, r, "/qt/dich-vu?ok="+urlQueryEscape("Đã xóa "+ma), http.StatusSeeOther)
}

// --- Bài của từng việc -------------------------------------------------

// dlQtDVBai nuôi trang soạn bài cho một việc. Cố tình KHÔNG dùng lại trang
// sửa bài viết: bài viết có đường dẫn, tiêu đề SEO, ngày đăng, ảnh bìa — bài
// dịch vụ không có cái nào trong số đó, mã việc đã là địa chỉ rồi.
type dlQtDVBai struct {
	dlQt
	DV   DichVu
	Than string
	Hinh map[string]string
}

func hQtDichVuBai(w http.ResponseWriter, r *http.Request) {
	ma := strings.ToUpper(r.PathValue("ma"))
	dv, ok := TimDichVu(ma)
	if !ok {
		http.NotFound(w, r)
		return
	}
	d := dlQtDVBai{dlQt: dlQt{Chung: chung(r, "dich-vu-qt")}}
	d.DV = dv
	d.Hinh = hinhChoPhep
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
		switch err := r.ParseForm(); {
		case err != nil:
			d.Loi = "Bài quá dài"
		default:
			than := r.FormValue("than")
			if err := LuuBaiDichVu(ma, than); err != nil {
				// Giữ nguyên chữ vừa gõ trong ô: lưu hỏng mà ô trở về bản cũ
				// thì mất trắng công soạn.
				d.Loi = err.Error()
				d.Than = than
				render(w, "qt-dichvu-bai.html", d)
				return
			}
			d.OK = "Đã lưu. Có hiệu lực ngay."
		}
	}
	d.Than = BaiDichVu(ma)
	render(w, "qt-dichvu-bai.html", d)
}
