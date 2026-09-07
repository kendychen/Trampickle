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
	DieuKien     string
	Gia          [3]string // rỗng = chưa mở bán ở giai đoạn đó
	GiaDep       [3]string // cùng giá trị, dạng người đọc
	BaoGiaRieng  bool
	TuGiaiDoan   int
	BaoHanhThang string
	LeadTimeNgay string
	ThuTu        string

	// DangHien và ViSao là kết quả tính ra, không phải ô nhập. Có nó thì Kendy
	// biết ngay vì sao một việc không thấy trên web, khỏi phải đoán.
	DangHien bool
	ViSao    string
}

func oTuDichVu(d DichVu) ODichVu {
	o := ODichVu{
		Ma: d.Ma, Ten: d.Ten, Nhom: d.Nhom,
		DieuKien:    strings.TrimSpace(d.DieuKien),
		BaoGiaRieng: d.BaoGiaRieng,
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
	}
	o.DangHien = d.DaMo(GiaiDoan)
	switch {
	case o.DangHien:
		o.ViSao = "Đang hiện trên web."
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

	for i := 0; i < 3; i++ {
		chu := strings.TrimSpace(f(fmt.Sprintf("gia%d", i+1)))
		if chu == "" {
			moi.Gia[i] = ""
			continue
		}
		if !coChuSo(chu) {
			return moi, loi("giá giai đoạn %d: %q không có chữ số nào", i+1, chu)
		}
		v := soTien(chu)
		if v < 0 || v > tienToiDa {
			return moi, loi("giá giai đoạn %d phải trong khoảng 0 – %s", i+1, dinhDangTien(tienToiDa))
		}
		moi.Gia[i] = strconv.Itoa(v)
	}

	moi.BaoGiaRieng = f("bao_gia_rieng") != ""

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

func dongGia(g [3]string) string {
	p := make([]string, 3)
	for i, v := range g {
		if v == "" {
			p[i] = "null"
			continue
		}
		p[i] = v
	}
	return "    gia: [" + strings.Join(p, ", ") + "]"
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
		dat("gia", []string{dongGia(moi.Gia)}, true)
	}
	if moi.DieuKien != cu.DieuKien {
		dat("dieu_kien", dongFolded("dieu_kien", moi.DieuKien), moi.DieuKien != "")
	}
	if moi.BaoGiaRieng != cu.BaoGiaRieng {
		dat("bao_gia_rieng", []string{"    bao_gia_rieng: true"}, moi.BaoGiaRieng)
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
	if moi.ThuTu != cu.ThuTu {
		dat("thu_tu", []string{"    thu_tu: " + moi.ThuTu}, moi.ThuTu != "")
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

// --- Trang quản trị --------------------------------------------------

type dlQtDichVu struct {
	dlQt
	DichVu   []ODichVu
	Tep      string
	GiaiDoan int
	SoHien   int
}

func hQtDichVu(w http.ResponseWriter, r *http.Request) {
	d := dlQtDichVu{dlQt: dlQt{Chung: chung(r, "dich-vu-qt")}}
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
	d.Tep = CFG.DuongDan.BangGia
	d.GiaiDoan = GiaiDoan
	render(w, "qt-dich-vu.html", d)
}
