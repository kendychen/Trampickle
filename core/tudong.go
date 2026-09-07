// Những thứ hệ thống tự dựng sẵn — và tại sao chúng đều dừng lại trước một
// cái nút xác nhận.
//
// Ba việc ở đây: khoản chi lặp hàng tháng, khoản chi sinh từ phiếu nhập kho,
// và mail báo cáo tháng gửi chủ. Hai việc đầu chỉ DỰNG SẴN một khoản ở trạng
// thái chờ duyệt. Sổ tiền là căn cứ để nói "lời hay lỗ, bao giờ về vốn";
// tháng nào quên trả tiền nhà mà sổ vẫn tự ghi đã chi thì con số về vốn sai
// theo, và không ai biết nó sai. Máy đoán, người xác nhận.
package core

import (
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// --- Khoản định kỳ ---------------------------------------------------

type KhoanDinhKy struct {
	Ma             string `yaml:"ma" json:"ma"`
	Ten            string `yaml:"ten" json:"ten"`
	Nhom           string `yaml:"nhom" json:"nhom"`
	SoTien         int    `yaml:"so_tien" json:"so_tien"`
	NgayTrongThang int    `yaml:"ngay_trong_thang" json:"ngay_trong_thang"`
	PhuongThuc     string `yaml:"phuong_thuc" json:"phuong_thuc"`
	GhiChu         string `yaml:"ghi_chu" json:"ghi_chu"`
	Bat            bool   `yaml:"bat" json:"bat"`
}

func (k KhoanDinhKy) TenNhom() string { return TenNhomTien(k.Nhom) }

var (
	dinhKyMu sync.RWMutex
	dinhKyDs []KhoanDinhKy
)

func fileDinhKy() string       { return P("data/so-tien/dinh-ky.yaml") }
func fileNhatKyTuDong() string { return P("data/so-tien/tu-dong.json") }

type dinhKyFile struct {
	DinhKy []KhoanDinhKy `yaml:"dinh_ky"`
}

func NapDinhKy() error {
	ds := []KhoanDinhKy{}
	if b, err := os.ReadFile(fileDinhKy()); err == nil {
		var f dinhKyFile
		if err := yaml.Unmarshal(b, &f); err != nil {
			return fmt.Errorf("data/so-tien/dinh-ky.yaml hỏng: %w", err)
		}
		ds = f.DinhKy
	} else if !os.IsNotExist(err) {
		return err
	}
	dinhKyMu.Lock()
	dinhKyDs = ds
	dinhKyMu.Unlock()
	return nil
}

func DanhSachDinhKy() []KhoanDinhKy {
	dinhKyMu.RLock()
	defer dinhKyMu.RUnlock()
	return append([]KhoanDinhKy{}, dinhKyDs...)
}

func LuuDinhKy(ds []KhoanDinhKy) error {
	sach := make([]KhoanDinhKy, 0, len(ds))
	// Giữ trước mọi mã đang có: sinh mã mới mà đụng vào mã của một dòng chưa
	// duyệt tới thì dòng đó bị đổi mã, và nhật ký "đã dựng khoản dk-1 tháng
	// 2026-09" không còn khớp — tháng đó bị dựng khoản lần thứ hai.
	dung := map[string]bool{}
	for _, k := range ds {
		if m := strings.TrimSpace(k.Ma); m != "" {
			dung[m] = true
		}
	}
	daDung := map[string]bool{}
	for _, k := range ds {
		k.Ten = strings.TrimSpace(k.Ten)
		k.Ma = strings.TrimSpace(k.Ma)
		if k.Ten == "" || k.SoTien <= 0 {
			continue
		}
		if n, co := TimNhom(k.Nhom); !co || n.Loai != KhoanChi {
			k.Nhom = NhomChiKhac
		}
		if k.Ma == "" || daDung[k.Ma] {
			k.Ma = maDinhKyMoi(dung)
			dung[k.Ma] = true
		}
		if k.NgayTrongThang < 1 || k.NgayTrongThang > 28 {
			// 28 là ngày cuối cùng có mặt ở mọi tháng. Cho phép 31 thì tháng
			// hai khoản đó không bao giờ tới hạn.
			k.NgayTrongThang = 1
		}
		daDung[k.Ma] = true
		sach = append(sach, k)
	}
	b, err := yaml.Marshal(dinhKyFile{DinhKy: sach})
	if err != nil {
		return err
	}
	dau := []byte("# Các khoản chi lặp hàng tháng.\n" +
		"# Đầu tháng hệ thống dựng sẵn một khoản CHỜ DUYỆT theo bảng này;\n" +
		"# phải có người bấm xác nhận đã trả thì mới vào sổ.\n" +
		"# Sửa được ở /qt/tien/dinh-ky, không cần sửa tay file này.\n\n")
	if err := os.MkdirAll(filepath.Dir(fileDinhKy()), 0o755); err != nil {
		return err
	}
	if err := ghiAtomic(fileDinhKy(), append(dau, b...)); err != nil {
		return err
	}
	dinhKyMu.Lock()
	dinhKyDs = sach
	dinhKyMu.Unlock()
	return nil
}

// maDinhKyMoi — mã chỉ dùng để nhận ra "khoản này tháng nay đã dựng rồi",
// không ai đọc nó, nên đánh số là đủ. Quan trọng là ĐỪNG sinh mã từ tên: đổi
// tên khoản từ "Thuê nhà" thành "Tiền nhà" mà mã đổi theo thì tháng đó sinh
// thêm một khoản nữa vì mã cũ không còn khớp.
func maDinhKyMoi(dung map[string]bool) string {
	for i := 1; ; i++ {
		ma := fmt.Sprintf("dk-%d", i)
		if !dung[ma] {
			return ma
		}
	}
}

// SinhKhoanDinhKy dựng khoản chờ duyệt cho tháng đã cho. Chạy lại bao nhiêu
// lần cũng ra cùng kết quả: đã có khoản mang mã định kỳ đó trong tháng thì bỏ
// qua, kể cả khoản người ta đã duyệt hay đã sửa số. Nhờ vậy việc nền cứ 6
// tiếng chạy một lần cũng không đẻ ra bản sao.
func SinhKhoanDinhKy(thang string) (int, error) {
	if !thangHopLe(thang) {
		return 0, fmt.Errorf("tháng không hợp lệ: %q", thang)
	}
	daCo := map[string]bool{}
	for _, k := range KhoanTrongThang(thang) {
		if k.MaDinhKy != "" {
			daCo[k.MaDinhKy] = true
		}
	}
	dem := 0
	for _, dk := range DanhSachDinhKy() {
		if !dk.Bat || daCo[dk.Ma] {
			continue
		}
		ngay := fmt.Sprintf("%s-%02d", thang, dk.NgayTrongThang)
		err := LuuKhoan(Khoan{
			Ngay:       ngay,
			Loai:       KhoanChi,
			Nhom:       dk.Nhom,
			SoTien:     dk.SoTien,
			DienGiai:   dk.Ten,
			PhuongThuc: dk.PhuongThuc,
			MaDinhKy:   dk.Ma,
			ChoDuyet:   true,
			Nguoi:      "hệ thống",
		})
		if err != nil {
			return dem, err
		}
		dem++
	}
	return dem, nil
}

// --- Khoản chi sinh từ phiếu nhập kho --------------------------------

// GhiChiTuPhieu gắn một khoản chi vào phiếu nhập. Gọi lại với cùng mã phiếu
// thì SỬA khoản cũ chứ không thêm khoản mới — nhập kho sửa lại số lượng là
// chuyện thường, và mỗi lần sửa lại đẻ thêm một khoản chi thì sổ phồng lên
// gấp đôi mà không ai để ý.
//
// Vật tư tính chi ngay lúc trả tiền cho nhà cung cấp, KHÔNG tính lại lúc xuất
// dùng. Tính hai lần là trừ hai lần vào lãi.
func GhiChiTuPhieu(p *Phieu, daTra bool) error {
	if p == nil || p.Loai != PhieuNhap {
		return nil
	}
	cu, coCu := khoanTheoPhieu(p.Ma)
	if p.TongTien <= 0 {
		if coCu {
			return XoaKhoan(cu.Ma)
		}
		return nil
	}
	k := Khoan{
		Ngay:       p.Ngay,
		Loai:       KhoanChi,
		Nhom:       NhomVatTu,
		SoTien:     p.TongTien,
		DienGiai:   "Nhập kho " + p.Ma + moTaNhaCungCap(p),
		PhuongThuc: TraChuyenKhoan,
		MaPhieu:    p.Ma,
		ChoDuyet:   !daTra,
		Nguoi:      p.Nguoi,
	}
	if coCu {
		k.Ma = cu.Ma
		k.PhuongThuc = cu.PhuongThuc
		k.Tao = cu.Tao
	}
	return LuuKhoan(k)
}

func moTaNhaCungCap(p *Phieu) string {
	if strings.TrimSpace(p.NhaCungCap) == "" {
		return ""
	}
	return " — " + strings.TrimSpace(p.NhaCungCap)
}

func khoanTheoPhieu(maPhieu string) (Khoan, bool) {
	tienMu.RLock()
	defer tienMu.RUnlock()
	for _, ds := range tienTheoThang {
		for _, k := range ds {
			if k.MaPhieu == maPhieu {
				return k, true
			}
		}
	}
	return Khoan{}, false
}

// XoaChiTuPhieu gọi khi xoá phiếu nhập, để khoản chi không mồ côi lại trong sổ.
func XoaChiTuPhieu(maPhieu string) error {
	if k, co := khoanTheoPhieu(maPhieu); co {
		return XoaKhoan(k.Ma)
	}
	return nil
}

// --- Mail báo cáo tháng ----------------------------------------------

func emailChu() string {
	if s := strings.TrimSpace(CFG.Email.BaoCao); s != "" {
		return s
	}
	return strings.TrimSpace(CFG.Email.TraLoi)
}

// ThanBaoCaoThang dựng nội dung báo cáo. Tách khỏi việc gửi để trang quản trị
// xem thử được trước khi mail bay đi.
func ThanBaoCaoThang(thang string) (tieuDe, thanHTML, thanChu string) {
	var t ThangTien
	for _, x := range BaoCaoThang() {
		if x.Thang == thang {
			t = x
			break
		}
	}
	vv := TinhVeVon()
	tieuDe = fmt.Sprintf("[%s] Sổ tháng %s: %s", CFG.ThuongHieu.Ten, thang, tomTatLaiLo(t))

	dong := []string{
		fmt.Sprintf("Tháng %s", thang),
		fmt.Sprintf("Thu:            %s", dinhDangTien(t.Thu)),
		fmt.Sprintf("Chi vận hành:   %s", dinhDangTien(t.ChiVanHanh)),
		fmt.Sprintf("Lãi:            %s", dinhDangTien(t.Lai)),
		fmt.Sprintf("Đầu tư trong tháng: %s", dinhDangTien(t.DauTu)),
		"",
		cauVeVon(vv),
		"",
		fmt.Sprintf("Khách còn nợ:   %s", dinhDangTien(LayThongKeDon().ConNoTong)),
		fmt.Sprintf("Giá trị tồn kho: %s", dinhDangTien(GiaTriTonKho())),
	}
	if n := SoKhoanChoDuyet(); n > 0 {
		dong = append(dong, "", fmt.Sprintf("Còn %d khoản chờ xác nhận trong sổ.", n))
	}
	if n := SoVatTuCanMua(); n > 0 {
		dong = append(dong, fmt.Sprintf("Có %d vật tư dưới mức tồn tối thiểu.", n))
	}
	thanChu = strings.Join(dong, "\n")

	var b strings.Builder
	b.WriteString(`<div style="font:15px/1.6 system-ui,sans-serif;color:#1c1c1c">`)
	fmt.Fprintf(&b, `<h2 style="margin:0 0 4px">Sổ tháng %s</h2>`, html.EscapeString(thang))
	fmt.Fprintf(&b, `<p style="margin:0 0 16px;color:#666">%s</p>`, html.EscapeString(tomTatLaiLo(t)))
	b.WriteString(`<table cellpadding="6" style="border-collapse:collapse;font-size:15px">`)
	hangMail(&b, "Thu", dinhDangTien(t.Thu), false)
	hangMail(&b, "Chi vận hành", dinhDangTien(t.ChiVanHanh), false)
	hangMail(&b, "Lãi", dinhDangTien(t.Lai), true)
	hangMail(&b, "Đầu tư trong tháng", dinhDangTien(t.DauTu), false)
	hangMail(&b, "Khách còn nợ", dinhDangTien(LayThongKeDon().ConNoTong), false)
	hangMail(&b, "Giá trị tồn kho", dinhDangTien(GiaTriTonKho()), false)
	b.WriteString(`</table>`)
	fmt.Fprintf(&b, `<p style="margin:16px 0 0">%s</p>`, html.EscapeString(cauVeVon(vv)))
	if n := SoKhoanChoDuyet(); n > 0 {
		fmt.Fprintf(&b, `<p style="margin:8px 0 0;color:#a15c00">Còn %d khoản chờ xác nhận trong sổ.</p>`, n)
	}
	if n := SoVatTuCanMua(); n > 0 {
		fmt.Fprintf(&b, `<p style="margin:8px 0 0;color:#a15c00">Có %d vật tư dưới mức tồn tối thiểu.</p>`, n)
	}
	if goc := strings.TrimSpace(CFG.Email.GocWeb); goc != "" {
		fmt.Fprintf(&b, `<p style="margin:20px 0 0"><a href="%s/qt/tien">Mở sổ tiền</a></p>`,
			html.EscapeString(strings.TrimRight(goc, "/")))
	}
	b.WriteString(`</div>`)
	return tieuDe, b.String(), thanChu
}

func hangMail(b *strings.Builder, ten, gia string, dam bool) {
	w := "400"
	if dam {
		w = "700"
	}
	fmt.Fprintf(b, `<tr><td style="border-top:1px solid #eee;color:#666">%s</td>`+
		`<td style="border-top:1px solid #eee;text-align:right;font-weight:%s">%s</td></tr>`,
		html.EscapeString(ten), w, html.EscapeString(gia))
}

func tomTatLaiLo(t ThangTien) string {
	if t.SoKhoan == 0 {
		return "chưa có khoản nào được ghi"
	}
	if t.Lai >= 0 {
		return "lãi " + dinhDangTien(t.Lai)
	}
	return "lỗ " + dinhDangTien(-t.Lai)
}

func cauVeVon(vv KetQuaVeVon) string {
	switch {
	case !vv.CoSo || vv.TongDauTu == 0:
		return "Chưa khai khoản đầu tư nào nên chưa tính được điểm về vốn."
	case vv.DaVeVon:
		return fmt.Sprintf("Đã về vốn từ tháng %s. Tổng đầu tư %s, luỹ kế lãi %s.",
			vv.ThangVeVon, dinhDangTien(vv.TongDauTu), dinhDangTien(vv.LuyKeLai))
	case vv.DangLo:
		return fmt.Sprintf("Đang lỗ (%d tháng gần nhất), chưa có ngày về vốn. Còn phải bù %s.",
			vv.SoThangTinh, dinhDangTien(vv.ConThieu))
	default:
		s := fmt.Sprintf("Còn thiếu %s, với đà lãi %s/tháng thì khoảng %.1f tháng nữa về vốn",
			dinhDangTien(vv.ConThieu), dinhDangTien(vv.LaiTBThang), vv.ConMayThang)
		if vv.NgayDuKien != "" {
			s += " (quanh " + vv.NgayDuKien + ")"
		}
		return s + "."
	}
}

// --- Nhật ký việc nền ------------------------------------------------

// Ghi lại đã làm gì cho tháng nào, để khởi động lại máy chủ không gửi mail
// báo cáo lần thứ hai.
type nhatKyTuDong struct {
	DaGuiBaoCao  []string `json:"da_gui_bao_cao"`
	DaSinhDinhKy []string `json:"da_sinh_dinh_ky"`
}

var nhatKyMu sync.Mutex

func docNhatKy() nhatKyTuDong {
	var n nhatKyTuDong
	if b, err := os.ReadFile(fileNhatKyTuDong()); err == nil {
		json.Unmarshal(b, &n)
	}
	return n
}

func ghiNhatKy(n nhatKyTuDong) error {
	sort.Strings(n.DaGuiBaoCao)
	sort.Strings(n.DaSinhDinhKy)
	b, err := json.MarshalIndent(n, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(fileNhatKyTuDong()), 0o755); err != nil {
		return err
	}
	return ghiAtomic(fileNhatKyTuDong(), b)
}

func coTrong(ds []string, s string) bool {
	for _, x := range ds {
		if x == s {
			return true
		}
	}
	return false
}

// ChayViecNenMotLuot — sinh khoản định kỳ cho tháng này, và gửi báo cáo tháng
// trước nếu chưa gửi. Trả về mô tả những gì đã làm, rỗng nếu không có gì.
func ChayViecNenMotLuot(bayGio time.Time) []string {
	nhatKyMu.Lock()
	defer nhatKyMu.Unlock()

	nk := docNhatKy()
	lam := []string{}
	thangNay := bayGio.Format("2006-01")

	if !coTrong(nk.DaSinhDinhKy, thangNay) {
		n, err := SinhKhoanDinhKy(thangNay)
		if err != nil {
			lam = append(lam, "lỗi sinh khoản định kỳ: "+err.Error())
		} else {
			nk.DaSinhDinhKy = append(nk.DaSinhDinhKy, thangNay)
			if n > 0 {
				lam = append(lam, fmt.Sprintf("dựng %d khoản định kỳ chờ duyệt cho %s", n, thangNay))
			}
		}
	}

	// Báo cáo gửi từ ngày 1, và chỉ khi tháng trước có phát sinh. Trạm mới
	// mở chưa ghi gì mà tháng nào cũng nhận một cái mail toàn số 0 thì lần
	// sau người ta không mở mail nữa.
	thangTruoc := bayGio.AddDate(0, -1, 0).Format("2006-01")
	if !coTrong(nk.DaGuiBaoCao, thangTruoc) && len(KhoanTrongThang(thangTruoc)) > 0 {
		if den := emailChu(); MailBat() && HopLeEmail(den) {
			tieuDe, thanHTML, thanChu := ThanBaoCaoThang(thangTruoc)
			if err := guiMail(den, tieuDe, thanHTML, thanChu); err != nil {
				lam = append(lam, "lỗi gửi báo cáo: "+err.Error())
			} else {
				nk.DaGuiBaoCao = append(nk.DaGuiBaoCao, thangTruoc)
				lam = append(lam, "gửi báo cáo tháng "+thangTruoc+" cho "+den)
			}
		}
	}

	if err := ghiNhatKy(nk); err != nil {
		lam = append(lam, "lỗi ghi nhật ký tự động: "+err.Error())
	}
	return lam
}

// ChayViecNen chạy nền suốt đời tiến trình. Sáu tiếng một lượt: việc ở đây
// tính theo tháng, chậm nửa ngày không ai chết, mà lỡ có lỗi thì cũng không
// spam log mỗi phút một dòng.
func ChayViecNen(dung <-chan struct{}) {
	chay := func() {
		for _, s := range ChayViecNenMotLuot(time.Now()) {
			fmt.Println("[sổ tiền]", s)
		}
	}
	chay()
	tick := time.NewTicker(6 * time.Hour)
	defer tick.Stop()
	for {
		select {
		case <-tick.C:
			chay()
		case <-dung:
			return
		}
	}
}
