package core

// Băng ảnh chạy ở tấm hero trang chủ.
//
// Ảnh nằm trong data/anh-trang-chu/, thứ tự và chú thích nằm trong
// data/anh-trang-chu.yaml. Tách đôi như vậy vì hai thứ này đổi theo hai nhịp
// khác nhau: ảnh thì thỉnh thoảng mới thêm, còn thứ tự và chú thích thì sửa
// vặt liên tục — nhét chú thích vào tên file là kiểu gì cũng có ngày phải đổi
// tên file chỉ để sửa một dấu phẩy.
//
// Chưa up ảnh nào thì trang chủ vẫn chạy bình thường bằng bản vẽ cây vợt.
// Đây là thứ trang trí, không phải nội dung — không có nó trang vẫn bán hàng
// được, nên mọi lỗi ở đây đều đi theo hướng "bỏ qua rồi chạy tiếp".

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

type AnhHero struct {
	Ten      string `yaml:"ten"`
	ChuThich string `yaml:"chu_thich"`
	// Noi: chỗ treo tấm ảnh. "" là băng ảnh hero trang chủ, "tram" là dải ảnh
	// trang Về chúng tôi. Để trống làm mặc định vì mọi ảnh có từ trước khi
	// tách hai nơi đều là ảnh trang chủ — không phải sửa lại file yaml cũ.
	//
	// Một kho ảnh chung cho cả hai nơi thay vì hai thư mục riêng: Kendy up
	// ảnh ở đúng một chỗ đã quen, và chuyển một tấm sang nơi khác chỉ là bấm
	// một nút chứ không phải tải lên lại.
	Noi string `yaml:"noi,omitempty"`
}

// NoiTram là giá trị Noi của ảnh treo ở trang Về chúng tôi.
const NoiTram = "tram"

type khoAnhHero struct {
	Anh []AnhHero `yaml:"anh"`
}

// Tám ảnh là quá đủ cho một băng chạy: khách xem tấm hero chừng vài giây rồi
// cuộn, ảnh thứ chín không ai nhìn thấy nhưng vẫn phải tải về. Con số thành
// mười bốn khi kho phải gánh thêm dải ảnh trang Về chúng tôi — vẫn là tám cho
// mỗi nơi, chỉ là hai nơi dùng chung một kho.
const anhHeroToiDa = 14
const anhHeroToiDaByte = 8 << 20

var (
	anhHeroMu sync.RWMutex
	anhHeroDS []AnhHero
)

func thuMucAnhHero() string { return P("data/anh-trang-chu") }
func fileAnhHero() string   { return P("data/anh-trang-chu.yaml") }

// NapAnhHero đọc danh sách lúc khởi động. Ảnh ghi trong yaml mà file đã mất
// thì loại luôn khỏi danh sách — để lại chỉ tổ ra một ô trống trong băng ảnh.
func NapAnhHero() error {
	b, err := os.ReadFile(fileAnhHero())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var kho khoAnhHero
	if err := yaml.Unmarshal(b, &kho); err != nil {
		return fmt.Errorf("data/anh-trang-chu.yaml hỏng: %w", err)
	}
	con := make([]AnhHero, 0, len(kho.Anh))
	for _, a := range kho.Anh {
		if a.Ten == "" || !tenAnhSach(a.Ten) {
			continue
		}
		if _, err := os.Stat(filepath.Join(thuMucAnhHero(), a.Ten)); err != nil {
			continue
		}
		con = append(con, a)
	}
	anhHeroMu.Lock()
	anhHeroDS = con
	anhHeroMu.Unlock()
	return nil
}

// tenAnhSach: tên file do máy đặt chứ không do người gửi đặt, nhưng yaml thì
// người sửa được bằng tay. Chặn ở đây để một dòng "ten: ../../config.yaml"
// không đi được tới os.Open.
func tenAnhSach(ten string) bool {
	if ten == "" || len(ten) > 80 {
		return false
	}
	if strings.ContainsAny(ten, `/\`) || strings.Contains(ten, "..") {
		return false
	}
	return true
}

// DsAnhHero trả bản sao — người gọi cầm về đem đi render, đừng để nó nắm tay
// vào lát cắt mà mấy hàm sửa bên dưới đang ghi.
func DsAnhHero() []AnhHero {
	anhHeroMu.RLock()
	defer anhHeroMu.RUnlock()
	ds := make([]AnhHero, len(anhHeroDS))
	copy(ds, anhHeroDS)
	return ds
}

// DsAnhNoi lọc theo chỗ treo. Trang chủ và trang Về chúng tôi gọi hàm này,
// còn trang quản trị vẫn xem cả kho bằng DsAnhHero.
func DsAnhNoi(noi string) []AnhHero {
	anhHeroMu.RLock()
	defer anhHeroMu.RUnlock()
	var ds []AnhHero
	for _, a := range anhHeroDS {
		if a.Noi == noi {
			ds = append(ds, a)
		}
	}
	return ds
}

func SoAnhHero() int {
	anhHeroMu.RLock()
	defer anhHeroMu.RUnlock()
	return len(anhHeroDS)
}

// coAnhHero — chỉ phục vụ tên có trong danh sách, không ghép đường dẫn từ
// tham số URL. Cùng luật với ảnh đơn hàng.
func coAnhHero(ten string) bool {
	anhHeroMu.RLock()
	defer anhHeroMu.RUnlock()
	for _, a := range anhHeroDS {
		if a.Ten == ten {
			return true
		}
	}
	return false
}

func luuAnhHero() error {
	anhHeroMu.RLock()
	kho := khoAnhHero{Anh: append([]AnhHero(nil), anhHeroDS...)}
	anhHeroMu.RUnlock()
	b, err := yaml.Marshal(kho)
	if err != nil {
		return err
	}
	dau := []byte("# Băng ảnh chạy ở hero trang chủ. Sửa ở /qt/anh-trang-chu.\n" +
		"# Thứ tự trong danh sách là thứ tự chạy.\n\n")
	return ghiAtomic(fileAnhHero(), append(dau, b...))
}

func ThemAnhHero(ten, chuThich string) error {
	anhHeroMu.Lock()
	if len(anhHeroDS) >= anhHeroToiDa {
		anhHeroMu.Unlock()
		return fmt.Errorf("đã đủ %d ảnh, xoá bớt rồi thêm", anhHeroToiDa)
	}
	anhHeroDS = append(anhHeroDS, AnhHero{Ten: ten, ChuThich: chuThich})
	anhHeroMu.Unlock()
	return luuAnhHero()
}

// DatNoiAnhHero chuyển một tấm sang nơi treo khác. Nơi lạ thì coi như trang
// chủ, để một tham số hỏng không làm ảnh biến mất khỏi cả hai trang.
func DatNoiAnhHero(ten, noi string) error {
	if noi != NoiTram {
		noi = ""
	}
	anhHeroMu.Lock()
	thay := false
	for i := range anhHeroDS {
		if anhHeroDS[i].Ten == ten {
			anhHeroDS[i].Noi = noi
			thay = true
			break
		}
	}
	anhHeroMu.Unlock()
	if !thay {
		return fmt.Errorf("không có ảnh %s", ten)
	}
	return luuAnhHero()
}

func XoaAnhHero(ten string) error {
	anhHeroMu.Lock()
	con := anhHeroDS[:0]
	thay := false
	for _, a := range anhHeroDS {
		if a.Ten == ten {
			thay = true
			continue
		}
		con = append(con, a)
	}
	anhHeroDS = con
	anhHeroMu.Unlock()
	if !thay {
		return nil
	}
	os.Remove(filepath.Join(thuMucAnhHero(), ten))
	return luuAnhHero()
}

func DatChuThichAnhHero(ten, chuThich string) error {
	anhHeroMu.Lock()
	for i := range anhHeroDS {
		if anhHeroDS[i].Ten == ten {
			anhHeroDS[i].ChuThich = chuThich
		}
	}
	anhHeroMu.Unlock()
	return luuAnhHero()
}

// ChuyenAnhHero đổi chỗ một ảnh với ảnh liền kề. Đủ để sắp thứ tự mà không
// cần kéo thả — kéo thả trên điện thoại giữa sân thì lóng ngóng hơn hai nút.
func ChuyenAnhHero(ten string, len_ bool) error {
	anhHeroMu.Lock()
	for i := range anhHeroDS {
		if anhHeroDS[i].Ten != ten {
			continue
		}
		j := i + 1
		if len_ {
			j = i - 1
		}
		if j >= 0 && j < len(anhHeroDS) {
			anhHeroDS[i], anhHeroDS[j] = anhHeroDS[j], anhHeroDS[i]
		}
		break
	}
	anhHeroMu.Unlock()
	return luuAnhHero()
}

// --- Handler ---------------------------------------------------------

// hAnhTrangChu phục vụ ảnh của băng hero cho khách. Chỉ mở tên CÓ TRONG danh
// sách; không ghép đường dẫn từ tham số URL, cùng luật với ảnh đơn hàng.
func hAnhTrangChu(w http.ResponseWriter, r *http.Request) {
	ten := r.PathValue("ten")
	if !tenAnhSach(ten) || !coAnhHero(ten) {
		http.NotFound(w, r)
		return
	}
	// Tên file có chuỗi ngẫu nhiên, đổi ảnh là đổi tên — nên cache dài được.
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeFile(w, r, filepath.Join(thuMucAnhHero(), ten))
}

type dlAnhHero struct {
	dlQt
	Anh   []AnhHero
	ToiDa int
}

func hQtAnhTrangChu(w http.ResponseWriter, r *http.Request) {
	d := dlAnhHero{dlQt: dlQt{Chung: chung(r, "anh-trang-chu")}}
	d.OK = r.URL.Query().Get("ok")
	d.Loi = r.URL.Query().Get("loi")
	d.Anh = DsAnhHero()
	d.ToiDa = anhHeroToiDa
	render(w, "qt-anhtrangchu.html", d)
}

func veAnhTrangChu(w http.ResponseWriter, r *http.Request, ok, loi string) {
	u := "/qt/anh-trang-chu"
	switch {
	case loi != "":
		u += "?loi=" + urlEsc(loi)
	case ok != "":
		u += "?ok=" + urlEsc(ok)
	}
	http.Redirect(w, r, u, http.StatusSeeOther)
}

func hQtAnhTrangChuThem(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, anhHeroToiDaByte+1<<20)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		veAnhTrangChu(w, r, "", "Ảnh quá nặng")
		return
	}
	chuThich := strings.TrimSpace(r.FormValue("chu_thich"))
	if len(chuThich) > 120 {
		chuThich = chuThich[:120]
	}
	them := 0
	for _, fh := range r.MultipartForm.File["anh"] {
		f, err := fh.Open()
		if err != nil {
			continue
		}
		dau := make([]byte, 16)
		n, _ := io.ReadFull(f, dau)
		duoi := duoiAnh(dau[:n])
		if duoi == "" {
			f.Close()
			continue
		}
		os.MkdirAll(thuMucAnhHero(), 0o755)
		ten := fmt.Sprintf("hero-%s-%s%s", time.Now().Format("150405"), maNgauNhien(3), duoi)
		out, err := os.Create(filepath.Join(thuMucAnhHero(), ten))
		if err != nil {
			f.Close()
			continue
		}
		out.Write(dau[:n])
		io.Copy(out, io.LimitReader(f, anhHeroToiDaByte))
		out.Close()
		f.Close()
		if err := ThemAnhHero(ten, chuThich); err != nil {
			os.Remove(filepath.Join(thuMucAnhHero(), ten))
			veAnhTrangChu(w, r, "", err.Error())
			return
		}
		them++
	}
	if them == 0 {
		veAnhTrangChu(w, r, "", "Không nhận được ảnh nào — chỉ nhận JPG, PNG, WEBP")
		return
	}
	veAnhTrangChu(w, r, fmt.Sprintf("Đã thêm %d ảnh", them), "")
}

func hQtAnhTrangChuSua(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	ten := r.FormValue("ten")
	if !tenAnhSach(ten) {
		veAnhTrangChu(w, r, "", "Tên ảnh không hợp lệ")
		return
	}
	var err error
	ok := ""
	switch r.FormValue("viec") {
	case "xoa":
		err, ok = XoaAnhHero(ten), "Đã xoá ảnh"
	case "len":
		err, ok = ChuyenAnhHero(ten, true), "Đã đổi thứ tự"
	case "xuong":
		err, ok = ChuyenAnhHero(ten, false), "Đã đổi thứ tự"
	case "noi":
		err, ok = DatNoiAnhHero(ten, r.FormValue("noi")), "Đã chuyển chỗ treo ảnh"
	case "chu-thich":
		ct := strings.TrimSpace(r.FormValue("chu_thich"))
		if len(ct) > 120 {
			ct = ct[:120]
		}
		err, ok = DatChuThichAnhHero(ten, ct), "Đã lưu chú thích"
	default:
		veAnhTrangChu(w, r, "", "Không hiểu yêu cầu")
		return
	}
	if err != nil {
		veAnhTrangChu(w, r, "", err.Error())
		return
	}
	veAnhTrangChu(w, r, ok, "")
}
