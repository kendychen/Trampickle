package core

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

// --- Cụm liên hệ -----------------------------------------------------------
//
// config.yaml có sẵn khoá lien_he, nhưng nó là file Kendy sửa bằng tay rồi
// đẩy lên máy chủ — mà đổi số điện thoại thì phải đổi được ngay từ điện
// thoại, không đợi ai deploy. Nên đi theo đúng lối của giao diện: cụm liên hệ
// sửa được ở /qt/lien-he ghi ra data/lien-he.yaml, và file ấy ĐÈ giá trị
// trong config.yaml. Chưa có file thì vẫn chạy bằng config như trước.
//
// data/ không bao giờ bị rsync đè khi deploy, nên bản Kendy gõ trên máy chủ
// an toàn qua mọi lần đẩy binary.

var (
	lienHeMu    sync.RWMutex
	lienHeDaSua *LienHe // nil = chưa ai sửa, dùng của config.yaml
)

func fileLienHe() string { return P("data/lien-he.yaml") }

// NapLienHe đọc bản đã sửa. Chưa có file không phải lỗi.
func NapLienHe() error {
	b, err := os.ReadFile(fileLienHe())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var l LienHe
	if err := yaml.Unmarshal(b, &l); err != nil {
		return fmt.Errorf("%s hỏng: %w", fileLienHe(), err)
	}
	lienHeMu.Lock()
	lienHeDaSua = &l
	lienHeMu.Unlock()
	return nil
}

// LienHeHienTai là thứ duy nhất được phép đọc để hiện ra trang. Đừng đọc
// thẳng CFG.ThuongHieu.LienHe nữa — làm thế là bỏ qua phần Kendy đã sửa.
func LienHeHienTai() LienHe {
	lienHeMu.RLock()
	defer lienHeMu.RUnlock()
	if lienHeDaSua != nil {
		return *lienHeDaSua
	}
	return CFG.ThuongHieu.LienHe
}

// Giới hạn độ dài: không phải để chặn tấn công (form này chỉ chủ vào được) mà
// để một cú dán nhầm cả trang web không đẩy chân trang vỡ bố cục.
var oLienHe = []struct {
	Khoa, Nhan, Goi string
	ToiDa           int
	Lay             func(*LienHe) *string
}{
	{"dien_thoai", "Điện thoại", "Số khách bấm gọi được, ví dụ 0901234567", 40, func(l *LienHe) *string { return &l.DienThoai }},
	{"zalo", "Zalo", "Số Zalo, để trống thì giấu dòng này", 40, func(l *LienHe) *string { return &l.Zalo }},
	{"email", "Email", "Hòm thư khách viết tới", 120, func(l *LienHe) *string { return &l.Email }},
	{"dia_chi", "Địa chỉ trạm", "Số nhà, đường, phường, quận, thành phố — dòng này hiện ngay ở đầu trang chủ", 200, func(l *LienHe) *string { return &l.DiaChi }},
	{"gio_lam_viec", "Giờ làm việc", "Ví dụ: 8h – 20h, cả thứ Bảy", 120, func(l *LienHe) *string { return &l.GioLamVic }},
	{"facebook", "Facebook", "Dán nguyên đường dẫn, phải bắt đầu bằng https://", 200, func(l *LienHe) *string { return &l.Facebook }},
	{"tiktok", "TikTok", "Đường dẫn trang TikTok, ví dụ https://www.tiktok.com/@trampickle", 200, func(l *LienHe) *string { return &l.TikTok }},
	{"instagram", "Instagram", "Đường dẫn trang Instagram, để trống thì giấu icon này", 200, func(l *LienHe) *string { return &l.Instagram }},
	{"youtube", "YouTube", "Đường dẫn kênh YouTube, để trống thì giấu icon này", 200, func(l *LienHe) *string { return &l.YouTube }},
	{"ngan_hang_ma", "Mã ngân hàng (BIN)", "6 số, ví dụ 970436 là Vietcombank — tra ở vietqr.io/danh-sach-ngan-hang", 10, func(l *LienHe) *string { return &l.NganHangMa }},
	{"so_tai_khoan", "Số tài khoản", "Số tài khoản nhận tiền của trạm", 30, func(l *LienHe) *string { return &l.SoTaiKhoan }},
	{"chu_tai_khoan", "Chủ tài khoản", "Tên không dấu, viết hoa, đúng như trên sổ", 60, func(l *LienHe) *string { return &l.ChuTaiKhoan }},
}

// DatLienHe kiểm rồi ghi. Trả về bản đã chuẩn hoá để trang hiện lại đúng thứ
// vừa lưu chứ không phải thứ vừa gõ.
func DatLienHe(l LienHe) (LienHe, error) {
	for _, o := range oLienHe {
		p := o.Lay(&l)
		*p = strings.Join(strings.Fields(*p), " ")
		if len([]rune(*p)) > o.ToiDa {
			return l, fmt.Errorf("%s dài quá %d ký tự", o.Nhan, o.ToiDa)
		}
	}
	if l.Email != "" && !strings.Contains(l.Email, "@") {
		return l, errors.New("email phải có dấu @")
	}
	// Thiếu https:// thì trình duyệt hiểu là đường dẫn trong site và dẫn khách
	// tới trang trắng của chính mình. Kiểm cả bốn trang mạng xã hội chứ không
	// riêng Facebook — chúng vào web theo đúng một đường.
	for _, t := range l.MangXaHoi() {
		if !strings.HasPrefix(t.URL, "http://") && !strings.HasPrefix(t.URL, "https://") {
			return l, fmt.Errorf("đường dẫn %s phải bắt đầu bằng https://", t.Ten)
		}
	}

	b, err := yaml.Marshal(l)
	if err != nil {
		return l, err
	}
	dau := []byte("# Cụm liên hệ hiện trên web. Sửa ở /qt/lien-he.\n" +
		"# File này ĐÈ khoá lien_he trong config.yaml.\n\n")
	if err := ghiAtomic(fileLienHe(), append(dau, b...)); err != nil {
		return l, err
	}
	lienHeMu.Lock()
	lienHeDaSua = &l
	lienHeMu.Unlock()
	return l, nil
}

// --- Trang /qt/lien-he -----------------------------------------------------

type oLienHeHien struct {
	Khoa, Nhan, Goi, Gia string
}

type dlLienHe struct {
	dlQt
	O       []oLienHeHien
	OK, Loi string
	Tep     string
}

func hQtLienHe(w http.ResponseWriter, r *http.Request) {
	d := dlLienHe{dlQt: dlQt{Chung: chung(r, "lien-he-qt")}, Tep: "data/lien-he.yaml"}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
		if err := r.ParseForm(); err != nil {
			d.Loi = "Không đọc được biểu mẫu."
		} else {
			var moi LienHe
			for _, o := range oLienHe {
				*o.Lay(&moi) = r.FormValue(o.Khoa)
			}
			if _, err := DatLienHe(moi); err != nil {
				d.Loi = err.Error()
			} else {
				d.OK = "Đã lưu. Mở trang khách ở tab khác để xem."
			}
		}
	}

	// Đọc lại sau khi ghi: hiện đúng thứ đang chạy, kể cả khi lưu hụt.
	hien := LienHeHienTai()
	for _, o := range oLienHe {
		d.O = append(d.O, oLienHeHien{o.Khoa, o.Nhan, o.Goi, *o.Lay(&hien)})
	}
	render(w, "qt-lienhe.html", d)
}
