package core

// --- Thông tin trạm --------------------------------------------------------
//
// Đi theo đúng lối data/lien-he.yaml: sửa ở /qt/tram, ghi ra data/tram.yaml,
// và file ấy ĐÈ những khoá tương ứng trong config.yaml. Chưa có file thì chạy
// bằng config như trước.
//
// Vì sao phải làm: config.yaml là file nằm trên máy chủ, đổi một chữ trong đó
// nghĩa là mở SSH, sửa tay, khởi động lại dịch vụ. Đổi tên trạm, đổi dòng mô
// tả, đổi hòm thư gửi cho khách hay đổi tiền tố mã đơn đều là việc của chủ
// trạm chứ không phải việc của người deploy. data/ không bao giờ bị rsync đè
// khi đẩy binary, nên bản Kendy gõ ở đây sống qua mọi lần cập nhật.

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// Tiền tố mặc định khi chưa ai sửa. TV = trạm vợt, TG = trạm giày.
const (
	TienToVotMac  = "TV"
	TienToGiayMac = "TG"
)

// ToiDaTienToCu — giữ bao nhiêu tiền tố cũ. Đủ để vài lần đổi tên thương hiệu
// mà sao kê vẫn nhận ra mã đơn năm ngoái; không giữ vô hạn vì mỗi tiền tố là
// một nhánh nữa trong biểu thức quét sao kê.
const ToiDaTienToCu = 12

type ThongTinTram struct {
	Ten      string `yaml:"ten"`
	DongMoTa string `yaml:"dong_mo_ta"`
	TenDayDu string `yaml:"ten_day_du"`

	// Tiền tố mã đơn. Lưu trần chữ cái, không kèm gạch — gạch do MaDonMoi
	// ghép vào, để đổi tiền tố không phải nhớ gõ đúng dấu.
	TienToVot  string   `yaml:"tien_to_vot"`
	TienToGiay string   `yaml:"tien_to_giay"`
	TienToCu   []string `yaml:"tien_to_cu"`

	MailBat    bool   `yaml:"mail_bat"`
	MailTu     string `yaml:"mail_tu"`
	MailTraLoi string `yaml:"mail_tra_loi"`
	MailBaoCao string `yaml:"mail_bao_cao"`
	GocWeb     string `yaml:"goc_web"`
}

var (
	tramMu    sync.RWMutex
	tramDaSua *ThongTinTram // nil = chưa ai sửa, dùng của config.yaml
)

func fileTram() string { return P("data/tram.yaml") }

// tramGoc dựng bản mặc định từ config.yaml. Tiền tố không có trong config —
// trước đây nó nằm cứng trong code — nên lấy hằng số ở trên.
func tramGoc() ThongTinTram {
	return ThongTinTram{
		Ten:        CFG.ThuongHieu.Ten,
		DongMoTa:   CFG.ThuongHieu.DongMoTa,
		TenDayDu:   CFG.ThuongHieu.TenDayDu,
		TienToVot:  TienToVotMac,
		TienToGiay: TienToGiayMac,
		MailBat:    CFG.Email.Bat,
		MailTu:     CFG.Email.Tu,
		MailTraLoi: CFG.Email.TraLoi,
		MailBaoCao: CFG.Email.BaoCao,
		GocWeb:     CFG.Email.GocWeb,
	}
}

// NapTram đọc bản đã sửa. Chưa có file không phải lỗi.
func NapTram() error {
	b, err := os.ReadFile(fileTram())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var t ThongTinTram
	if err := yaml.Unmarshal(b, &t); err != nil {
		return fmt.Errorf("%s hỏng: %w", fileTram(), err)
	}
	// File có thể do tay người sửa. Tiền tố rỗng thì rơi về mặc định chứ
	// không sinh ra mã đơn cụt kiểu "-2609-001".
	if chuanTienTo(t.TienToVot) == "" {
		t.TienToVot = TienToVotMac
	}
	if chuanTienTo(t.TienToGiay) == "" {
		t.TienToGiay = TienToGiayMac
	}
	tramMu.Lock()
	tramDaSua = &t
	tramMu.Unlock()
	return nil
}

// TramHienTai là thứ duy nhất được phép đọc. Đừng đọc thẳng CFG.ThuongHieu
// hay CFG.Email nữa — làm thế là bỏ qua phần Kendy đã sửa trong admin.
func TramHienTai() ThongTinTram {
	tramMu.RLock()
	defer tramMu.RUnlock()
	if tramDaSua != nil {
		return *tramDaSua
	}
	return tramGoc()
}

// --- Đọc từng mảnh ---------------------------------------------------------

// TenTram không bao giờ rỗng: nó đi vào tiêu đề mail và tiêu đề tab, chỗ nào
// cũng cần một cái tên để đọc.
func TenTram() string {
	if t := strings.TrimSpace(TramHienTai().Ten); t != "" {
		return t
	}
	return "Trạm"
}

func DongMoTaTram() string { return strings.TrimSpace(TramHienTai().DongMoTa) }
func TenDayDuTram() string { return strings.TrimSpace(TramHienTai().TenDayDu) }

// ThuongHieuHienTai — bản ghép để đưa xuống template: ba dòng chữ lấy từ
// data/tram.yaml, cụm liên hệ lấy từ data/lien-he.yaml.
func ThuongHieuHienTai() ThuongHieu {
	t := TramHienTai()
	return ThuongHieu{
		Ten:      TenTram(),
		DongMoTa: strings.TrimSpace(t.DongMoTa),
		TenDayDu: strings.TrimSpace(t.TenDayDu),
		LienHe:   LienHeHienTai(),
	}
}

// TienToDon — tiền tố kèm gạch, dạng "TV-". Giày đếm riêng: nhìn mã là biết
// ngay đôi giày hay cây vợt, không phải mở đơn ra xem.
func TienToDon(loai string) string {
	t := TramHienTai()
	if LoaiHopLe(loai) == LoaiGiay {
		return chuanTienToHay(t.TienToGiay, TienToGiayMac) + "-"
	}
	return chuanTienToHay(t.TienToVot, TienToVotMac) + "-"
}

// TienToNhanDien — mọi tiền tố mà sao kê phải nhận ra: hai cái đang dùng cộng
// những cái đã từng dùng. Đổi tiền tố mà quên mã cũ thì mọi đơn năm ngoái
// biến mất khỏi màn đối chiếu sao kê.
func TienToNhanDien() []string {
	t := TramHienTai()
	ra := []string{}
	da := map[string]bool{}
	them := func(s string) {
		s = chuanTienTo(s)
		if s == "" || da[s] {
			return
		}
		da[s] = true
		ra = append(ra, s)
	}
	them(chuanTienToHay(t.TienToVot, TienToVotMac))
	them(chuanTienToHay(t.TienToGiay, TienToGiayMac))
	for _, s := range t.TienToCu {
		them(s)
	}
	return ra
}

func MailBat() bool {
	t := TramHienTai()
	return t.MailBat && strings.TrimSpace(t.MailTu) != "" && ResendKey() != ""
}

func MailTu() string     { return strings.TrimSpace(TramHienTai().MailTu) }
func MailTraLoi() string { return strings.TrimSpace(TramHienTai().MailTraLoi) }

// MailBaoCao — hòm thư nhận sổ tháng. Chưa khai thì gửi về hòm trả lời.
func MailBaoCao() string {
	if s := strings.TrimSpace(TramHienTai().MailBaoCao); s != "" {
		return s
	}
	return MailTraLoi()
}

// GocWeb — gốc đường dẫn tuyệt đối, không có gạch cuối. Rỗng thì mail chỉ có
// mã đơn, không có link bấm được.
func GocWeb() string {
	return strings.TrimRight(strings.TrimSpace(TramHienTai().GocWeb), "/")
}

// --- Chuẩn hoá tiền tố -----------------------------------------------------

// chuanTienTo giữ lại đúng chữ cái và chữ số, viết hoa. Gạch, khoảng trắng,
// dấu tiếng Việt đều bị bỏ: mã đơn phải đọc qua điện thoại được.
func chuanTienTo(s string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(strings.TrimSpace(s)) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func chuanTienToHay(s, mac string) string {
	if t := chuanTienTo(s); t != "" {
		return t
	}
	return mac
}

// kiemTienTo — 2 đến 5 ký tự, bắt đầu bằng chữ cái. Một ký tự thì biểu thức
// quét sao kê bắt nhầm mọi thứ có dạng chữ-cái + 7 chữ số; bắt đầu bằng số
// thì không phân biệt được với số tài khoản.
func kiemTienTo(s, nhan string) (string, error) {
	t := chuanTienTo(s)
	if len(t) < 2 || len(t) > 5 {
		return "", fmt.Errorf("%s phải dài 2–5 chữ cái hoặc chữ số", nhan)
	}
	if t[0] < 'A' || t[0] > 'Z' {
		return "", fmt.Errorf("%s phải bắt đầu bằng chữ cái", nhan)
	}
	return t, nil
}

// --- Ghi -------------------------------------------------------------------

// Giới hạn độ dài không phải để chặn tấn công (chỉ chủ trạm vào được form
// này) mà để một cú dán nhầm cả trang web không đẩy chân trang vỡ bố cục.
var oTram = []struct {
	Nhom, Khoa, Nhan, Goi string
	ToiDa                 int
	Lay                   func(*ThongTinTram) *string
}{
	{"Tên trạm", "ten", "Tên ngắn", "Tên hiện ở đầu trang, chân trang, tiêu đề tab và mọi mail gửi khách", 60,
		func(t *ThongTinTram) *string { return &t.Ten }},
	{"Tên trạm", "dong_mo_ta", "Dòng mô tả", "Câu ngắn đứng ngay sau tên, ví dụ: sửa vợt pickleball", 120,
		func(t *ThongTinTram) *string { return &t.DongMoTa }},
	{"Tên trạm", "ten_day_du", "Tên đầy đủ", "Tên pháp lý hoặc tên dài, dùng cho dữ liệu Google và tiêu đề cửa sổ máy tính", 160,
		func(t *ThongTinTram) *string { return &t.TenDayDu }},

	{"Mã đơn", "tien_to_vot", "Tiền tố đơn vợt", "2–5 chữ, không gõ gạch. TV cho ra mã TV-2609-001", 5,
		func(t *ThongTinTram) *string { return &t.TienToVot }},
	{"Mã đơn", "tien_to_giay", "Tiền tố đơn giày", "Phải khác tiền tố vợt, nhìn mã là biết ngay món gì", 5,
		func(t *ThongTinTram) *string { return &t.TienToGiay }},

	{"Email", "mail_tu", "Gửi đi từ", "Địa chỉ khách thấy ở ô From, ví dụ: Trạm Pickle <admin@trampickle.vn>. Tên miền phải đã xác thực ở Resend", 160,
		func(t *ThongTinTram) *string { return &t.MailTu }},
	{"Email", "mail_tra_loi", "Khách bấm trả lời thì về đâu", "Hòm thư thật của trạm, khác với địa chỉ gửi đi ở trên", 160,
		func(t *ThongTinTram) *string { return &t.MailTraLoi }},
	{"Email", "mail_bao_cao", "Nhận báo cáo sổ tháng", "Mail này có số liệu nội bộ. Để trống thì gửi về hòm trả lời", 160,
		func(t *ThongTinTram) *string { return &t.MailBaoCao }},
	{"Email", "goc_web", "Gốc web", "https://trampickle.vn — dùng để dựng link tra cứu trong mail. Sai gốc thì khách bấm vào link chết", 160,
		func(t *ThongTinTram) *string { return &t.GocWeb }},
}

// DatTram kiểm rồi ghi. Trả về bản đã chuẩn hoá để trang hiện lại đúng thứ
// vừa lưu chứ không phải thứ vừa gõ. TienToCu do đây tự tính — người gõ form
// không cần biết tới nó.
func DatTram(t ThongTinTram) (ThongTinTram, error) {
	for _, o := range oTram {
		p := o.Lay(&t)
		*p = strings.Join(strings.Fields(*p), " ")
		if len([]rune(*p)) > o.ToiDa {
			return t, fmt.Errorf("%s dài quá %d ký tự", o.Nhan, o.ToiDa)
		}
	}
	if t.Ten == "" {
		return t, errors.New("tên trạm không được để trống")
	}

	var err error
	if t.TienToVot, err = kiemTienTo(t.TienToVot, "Tiền tố đơn vợt"); err != nil {
		return t, err
	}
	if t.TienToGiay, err = kiemTienTo(t.TienToGiay, "Tiền tố đơn giày"); err != nil {
		return t, err
	}
	if t.TienToVot == t.TienToGiay {
		return t, errors.New("tiền tố vợt và giày phải khác nhau")
	}

	for _, c := range []struct{ nhan, gia string }{
		{"Địa chỉ gửi đi", t.MailTu}, {"Hòm thư trả lời", t.MailTraLoi}, {"Hòm thư báo cáo", t.MailBaoCao},
	} {
		if c.gia != "" && !strings.Contains(c.gia, "@") {
			return t, fmt.Errorf("%s phải có dấu @", c.nhan)
		}
	}
	// Thiếu https:// thì mọi link trong mail thành đường dẫn tương đối, dẫn
	// khách tới trang trắng của chính hòm thư họ đang mở.
	if t.GocWeb != "" && !strings.HasPrefix(t.GocWeb, "http://") && !strings.HasPrefix(t.GocWeb, "https://") {
		return t, errors.New("gốc web phải bắt đầu bằng https://")
	}
	t.GocWeb = strings.TrimRight(t.GocWeb, "/")

	// Lịch sử tiền tố: gộp cái cũ vào rồi bỏ hai cái đang dùng ra. Tính lại
	// từ đầu mỗi lần lưu nên bấm Lưu mười lần cũng ra một kết quả.
	cu, da := []string{}, map[string]bool{t.TienToVot: true, t.TienToGiay: true}
	truoc := TramHienTai()
	for _, s := range append([]string{truoc.TienToVot, truoc.TienToGiay}, truoc.TienToCu...) {
		s = chuanTienTo(s)
		if s == "" || da[s] {
			continue
		}
		da[s] = true
		cu = append(cu, s)
	}
	if len(cu) > ToiDaTienToCu {
		cu = cu[:ToiDaTienToCu]
	}
	t.TienToCu = cu

	b, err := yaml.Marshal(t)
	if err != nil {
		return t, err
	}
	dau := []byte("# Thông tin trạm. Sửa ở /qt/tram.\n" +
		"# File này ĐÈ khoá thuong_hieu và khoá email trong config.yaml.\n" +
		"# tien_to_cu do máy tự giữ: mã đơn cũ vẫn phải nhận ra khi đọc sao kê.\n\n")
	if err := ghiAtomic(fileTram(), append(dau, b...)); err != nil {
		return t, err
	}
	tramMu.Lock()
	tramDaSua = &t
	tramMu.Unlock()
	return t, nil
}

// --- Trang /qt/tram --------------------------------------------------------

type oTramHien struct {
	Nhom, Khoa, Nhan, Goi, Gia string
	DauNhom                    bool
}

type dlTram struct {
	dlQt
	O        []oTramHien
	MailBat  bool
	TienToCu string
	MauMa    string
	OK, Loi  string
	Tep      string
}

func hQtTram(w http.ResponseWriter, r *http.Request) {
	d := dlTram{dlQt: dlQt{Chung: chung(r, "tram")}, Tep: "data/tram.yaml"}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
		if err := r.ParseForm(); err != nil {
			d.Loi = "Không đọc được biểu mẫu."
		} else {
			moi := ThongTinTram{MailBat: r.FormValue("mail_bat") != ""}
			for _, o := range oTram {
				*o.Lay(&moi) = r.FormValue(o.Khoa)
			}
			if _, err := DatTram(moi); err != nil {
				d.Loi = err.Error()
			} else {
				d.OK = "Đã lưu. Mã đơn mới dùng tiền tố này ngay; đơn cũ giữ nguyên mã."
			}
		}
	}

	// Đọc lại sau khi ghi: hiện đúng thứ đang chạy, kể cả khi lưu hụt.
	hien := TramHienTai()
	nhom := ""
	for _, o := range oTram {
		d.O = append(d.O, oTramHien{o.Nhom, o.Khoa, o.Nhan, o.Goi, *o.Lay(&hien), o.Nhom != nhom})
		nhom = o.Nhom
	}
	d.MailBat = hien.MailBat
	d.TienToCu = strings.Join(hien.TienToCu, ", ")
	d.MauMa = TienToDon(LoaiVot) + "2609-001"
	render(w, "qt-tram.html", d)
}
