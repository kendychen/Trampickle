package core

// Sáu ngưỡng trong vanhanh/bang-gia.yaml, sửa được ở /qt/nguong.
//
// Vì sao ghi THẲNG vào bang-gia.yaml chứ không làm file override kiểu
// data/noi-dung.yaml: file đó KHÔNG chỉ Go đọc. Python đọc cùng chỗ ở
// src/quote.py, src/intake.py, scripts/print_prices.py — và src/llm.py nói rõ
// Python là nơi độc quyền tính con số cam kết với khách. Để Go đọc một file
// còn Python đọc file khác là có ngày trang web hứa 3.5g mà báo giá vẫn tính
// 3.0g. Một nguồn, không hai.
//
// Vì sao sửa theo DÒNG chứ không unmarshal rồi marshal lại: bang-gia.yaml là
// file người viết tay, mỗi con số có mấy dòng comment giải thích vì sao nó là
// con số đó. yaml.Marshal xoá sạch comment và dồn hết dòng trống. Đổi đúng
// phần số trên đúng một dòng thì phần còn lại của file giữ nguyên từng byte.
//
// Trước khi ghi, nội dung mới được parse lại bằng yaml và đối chiếu với con số
// định đặt. Không khớp là không ghi — thà báo lỗi còn hơn để lại một
// bang-gia.yaml hỏng mà cả Go lẫn Python đều không đọc nổi.

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// ngMu giữ GIA.Nguong. Cả bảng GIA nạp một lần lúc khởi động rồi chỉ đọc, trừ
// sáu số này giờ đổi được lúc đang chạy — nên chỉ chúng cần khoá, và mọi nơi
// đọc phải đi qua NguongHienTai().
var ngMu sync.RWMutex

// NguongHienTai trả bản sao, để chỗ gọi cầm về dùng thoải mái ngoài khoá.
func NguongHienTai() Nguong {
	ngMu.RLock()
	defer ngMu.RUnlock()
	return GIA.Nguong
}

func fileBangGia() string { return P(CFG.DuongDan.BangGia) }

// ONguong là một ô nhập trên trang quản trị.
type ONguong struct {
	Khoa string
	Nhan string
	MoTa string
	Don  string // "g" hoặc "đ" — hiện sau ô nhập
	// Web: con số này khách đọc được, trên trang hoặc trong báo giá. Đổi nó
	// là đổi lời hứa, không phải chỉnh một tham số nội bộ.
	Web bool
	So  string // giá trị đang dùng, dạng để đổ vào ô nhập
	Dep string // cùng giá trị, dạng người đọc
	// Toi: trần hợp lệ, chỉ nhóm ngưỡng nội bộ (core/nguongtram.go) dùng tới.
	// 0 đọc là 100 — phần lớn ô ở đó là phần trăm.
	Toi int
}

func (o ONguong) ToiDa() int {
	if o.Toi <= 0 {
		return 100
	}
	return o.Toi
}

// Thứ tự ở đây là thứ tự hiện trên trang. Nhóm khách đọc được lên trước.
var nguongCo = []ONguong{
	{
		Khoa: "tang_khoi_luong_toi_da_g", Don: "g", Web: true,
		Nhan: "Vượt quá thì không tính tiền công",
		MoTa: "Vợt sửa xong nặng thêm quá mức này là trạm chịu, không thu tiền công. In trên trang giới thiệu, trang quy trình và trong bài viết; cũng là ngưỡng máy dùng để đánh dấu đơn ở mục Đơn sửa.",
	},
	{
		Khoa: "free_ship_ve_tu_dong", Don: "đ", Web: true,
		Nhan: "Đơn từ mức này thì trạm chịu phí gửi trả",
		MoTa: "In trên trang quy trình. Dưới mức đó khách chịu cả hai chiều.",
	},
	{
		Khoa: "nguong_mo_kenh_ship_dong", Don: "đ", Web: true,
		Nhan: "Đơn dưới mức này thì không nhận ship",
		MoTa: "Luật vận hành, nhưng phần báo giá nói câu này với khách nên vẫn tính là khách đọc được.",
	},
	{
		Khoa: "bien_gop_toi_thieu_dong", Don: "đ",
		Nhan: "Lãi gộp tối thiểu một ca",
		MoTa: "Dưới mức này thì cảnh báo để xem lại có nên nhận không. Không hiện cho khách.",
	},
	{
		Khoa: "cac_qua_kenh_tra_tien_dong", Don: "đ",
		Nhan: "Chi phí kéo được một khách trả tiền",
		MoTa: "Tính trên ca CÓ doanh thu, đã trừ tỷ lệ từ chối. Chỉ dùng khi chạy quảng cáo. Không hiện cho khách.",
	},
	{
		Khoa: "ship_ca_tu_choi_dong", Don: "đ",
		Nhan: "Phí gửi trả ca từ chối",
		MoTa: "Ca khám xong không nhận thì trạm chịu chiều về, tính vào chi phí marketing. Không hiện cho khách.",
	},
}

// DanhSachNguong trả sáu ô kèm giá trị đang chạy.
func DanhSachNguong() []ONguong {
	n := NguongHienTai()
	ra := make([]ONguong, len(nguongCo))
	copy(ra, nguongCo)
	for i := range ra {
		v := layNguong(n, ra[i].Khoa)
		if ra[i].Don == "g" {
			ra[i].So = strconv.FormatFloat(v, 'f', -1, 64)
			ra[i].Dep = ra[i].So + " g"
			continue
		}
		ra[i].So = strconv.Itoa(int(v))
		ra[i].Dep = dinhDangTien(int(v))
	}
	return ra
}

// layNguong là chỗ duy nhất buộc khoá yaml phải khớp với trường Go. Thêm
// ngưỡng mới mà quên khai ở đây thì compiler không bắt được, nên
// TestNguongDuKhoa soi lại bằng chính file bảng giá.
func layNguong(n Nguong, khoa string) float64 {
	switch khoa {
	case "tang_khoi_luong_toi_da_g":
		return n.TangKhoiLuongToiDaG
	case "free_ship_ve_tu_dong":
		return float64(n.FreeShipVeTuDong)
	case "nguong_mo_kenh_ship_dong":
		return float64(n.NguongMoKenhShipDong)
	case "bien_gop_toi_thieu_dong":
		return float64(n.BienGopToiThieuDong)
	case "cac_qua_kenh_tra_tien_dong":
		return float64(n.CacQuaKenhTraTienDong)
	case "ship_ca_tu_choi_dong":
		return float64(n.ShipCaTuChoiDong)
	}
	return 0
}

const tienToiDa = 100_000_000 // 100 triệu: quá mức này chắc chắn gõ nhầm

// docNguong đọc chữ người gõ thành số, kèm luật hợp lệ riêng của từng ô.
func docNguong(o ONguong, chu string) (string, error) {
	chu = strings.TrimSpace(chu)
	if chu == "" {
		return "", fmt.Errorf("%s: chưa điền", o.Nhan)
	}
	if o.Don == "g" {
		// Cho gõ cả "3,5" — bàn phím tiếng Việt hay ra dấu phẩy.
		f, err := strconv.ParseFloat(strings.ReplaceAll(chu, ",", "."), 64)
		if err != nil {
			return "", fmt.Errorf("%s: %q không phải số", o.Nhan, chu)
		}
		if f <= 0 || f > 100 {
			return "", fmt.Errorf("%s: phải trong khoảng 0–100 g", o.Nhan)
		}
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	}
	// Tiền: soTien bỏ hết dấu chấm, dấu phẩy và chữ đ — người ta gõ 500.000đ.
	if !coChuSo(chu) {
		return "", fmt.Errorf("%s: %q không có chữ số nào", o.Nhan, chu)
	}
	v := soTien(chu)
	if v < 0 || v > tienToiDa {
		return "", fmt.Errorf("%s: phải trong khoảng 0 – %s", o.Nhan, dinhDangTien(tienToiDa))
	}
	return strconv.Itoa(v), nil
}

func coChuSo(s string) bool {
	for _, r := range s {
		if r >= '0' && r <= '9' {
			return true
		}
	}
	return false
}

// SuaNguong ghi những ô có đổi vào bang-gia.yaml. Trả về nhãn các ô đã đổi.
func SuaNguong(moi map[string]string) ([]string, error) {
	ngMu.Lock()
	defer ngMu.Unlock()

	b, err := os.ReadFile(fileBangGia())
	if err != nil {
		return nil, fmt.Errorf("đọc bảng giá: %w", err)
	}
	noi := string(b)
	crlf := strings.Contains(noi, "\r\n")
	dong := strings.Split(strings.ReplaceAll(noi, "\r\n", "\n"), "\n")

	dau, cuoi := khoiNguong(dong)
	if dau < 0 {
		return nil, fmt.Errorf("không tìm thấy khối nguong: trong %s", fileBangGia())
	}

	var doi []string
	dinh := map[string]string{} // khoá -> chuỗi số định đặt, để đối chiếu sau
	for _, o := range danhSachNguongDaKhoa() {
		chu, co := moi[o.Khoa]
		if !co {
			continue
		}
		so, err := docNguong(o, chu)
		if err != nil {
			return nil, err
		}
		if so == o.So {
			continue
		}
		i := timDongKhoa(dong, dau, cuoi, o.Khoa)
		if i < 0 {
			return nil, fmt.Errorf("không thấy dòng %s trong bảng giá", o.Khoa)
		}
		dong[i] = thayGiaTri(dong[i], so)
		dinh[o.Khoa] = so
		doi = append(doi, o.Nhan)
	}
	if len(doi) == 0 {
		return nil, nil
	}

	ra := strings.Join(dong, "\n")
	if crlf {
		ra = strings.ReplaceAll(ra, "\n", "\r\n")
	}

	// Đọc lại bằng yaml trước khi ghi: nếu sửa dòng làm hỏng cấu trúc, hoặc
	// con số rơi không đúng chỗ định nhắm, thì file cũ còn nguyên.
	var thu BangGia
	if err := yaml.Unmarshal([]byte(ra), &thu); err != nil {
		return nil, fmt.Errorf("sửa xong bảng giá không đọc được nữa, đã huỷ: %w", err)
	}
	for k, muon := range dinh {
		if layNguong(thu.Nguong, k) != phaiLaSo(muon) {
			return nil, fmt.Errorf("kiểm lại %s không khớp, đã huỷ", k)
		}
	}
	if err := ghiAtomic(fileBangGia(), []byte(ra)); err != nil {
		return nil, err
	}
	GIA.Nguong = thu.Nguong
	return doi, nil
}

// danhSachNguongDaKhoa như DanhSachNguong nhưng dùng khi ĐANG giữ ngMu.
func danhSachNguongDaKhoa() []ONguong {
	ra := make([]ONguong, len(nguongCo))
	copy(ra, nguongCo)
	for i := range ra {
		v := layNguong(GIA.Nguong, ra[i].Khoa)
		if ra[i].Don == "g" {
			ra[i].So = strconv.FormatFloat(v, 'f', -1, 64)
			continue
		}
		ra[i].So = strconv.Itoa(int(v))
	}
	return ra
}

func phaiLaSo(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// khoiNguong khoanh vùng từ dòng "nguong:" tới khoá cấp cao nhất kế tiếp.
// Khoanh vùng chứ không tìm khoá trên cả file: tên khoá có thể trùng ở khối
// khác, mà sửa nhầm khối là hỏng đúng chỗ không ai ngờ tới.
func khoiNguong(dong []string) (int, int) {
	dau := -1
	for i, d := range dong {
		if dau < 0 {
			if strings.HasPrefix(d, "nguong:") {
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

var reKhoaYaml = regexp.MustCompile(`^(\s+)([a-z0-9_]+):[ \t]*(.*)$`)

func timDongKhoa(dong []string, dau, cuoi int, khoa string) int {
	for i := dau + 1; i < cuoi && i < len(dong); i++ {
		if m := reKhoaYaml.FindStringSubmatch(dong[i]); m != nil && m[2] == khoa {
			return i
		}
	}
	return -1
}

// thayGiaTri đổi phần số, giữ nguyên thụt đầu dòng và comment cuối dòng.
func thayGiaTri(d, so string) string {
	m := reKhoaYaml.FindStringSubmatch(d)
	if m == nil {
		return d
	}
	ghi := ""
	if i := strings.Index(m[3], "#"); i >= 0 {
		ghi = "  " + strings.TrimSpace(m[3][i:])
	}
	return m[1] + m[2] + ": " + so + ghi
}

// --- Trang quản trị --------------------------------------------------

type dlNguong struct {
	dlQt
	Web   []ONguong
	NoiBo []ONguong
	Tep   string
}

func hQtNguong(w http.ResponseWriter, r *http.Request) {
	d := dlNguong{dlQt: dlQt{Chung: chung(r, "nguong")}}
	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
		r.ParseForm()
		moi := map[string]string{}
		for _, o := range nguongCo {
			if v, co := r.Form[o.Khoa]; co && len(v) > 0 {
				moi[o.Khoa] = v[0]
			}
		}
		doi, err := SuaNguong(moi)
		// Hai kho ngưỡng, hai file: sáu số kia đi vào bảng giá vì Python
		// cũng đọc, mấy số này ở data/ vì chỉ Go dùng — xem đầu
		// core/nguongtram.go. Với người dùng vẫn là một cái form.
		doiTram, errTram := SuaNguongTram(nhanFormNguongTram(r))
		doi = append(doi, doiTram...)
		if err == nil {
			err = errTram
		}
		switch {
		case err != nil:
			d.Loi = err.Error()
		case len(doi) == 0:
			d.OK = "Không có gì đổi."
		default:
			d.OK = "Đã đổi: " + strings.Join(doi, "; ") + ". Có hiệu lực ngay."
		}
	}
	// Đọc sau khi ghi, giống trang giao diện: chung() chạy từ đầu hàm nên
	// giá trị nằm trong đó vẫn là giá trị cũ.
	for _, o := range DanhSachNguong() {
		if o.Web {
			d.Web = append(d.Web, o)
		} else {
			d.NoiBo = append(d.NoiBo, o)
		}
	}
	d.NoiBo = append(d.NoiBo, DanhSachNguongTram()...)
	d.Tep = CFG.DuongDan.BangGia
	render(w, "qt-nguong.html", d)
}
