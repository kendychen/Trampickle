package core

// Logo và favicon do trạm tự tải lên, nằm ở data/logo/. Không có tệp thì
// trang quay về SVG nhúng sẵn trong core/ui — chép binary sang máy mới mà
// quên chép data/ là mất logo RIÊNG, chứ không mất logo.
//
// Mỗi loại chỉ giữ đúng một tệp (logo.<đuôi>, bieu-tuong.<đuôi>): tải cái mới
// là cái cũ bị xoá. Giữ nhiều bản thì có ngày trang hiện nhầm bản, mà chỗ này
// không cần lịch sử — muốn quay lại logo cũ thì tải lại tệp cũ.

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const logoToiDaByte = 2 << 20 // 2 MB, logo nào nặng hơn thế là dùng sai tệp

// Hai loại, dùng luôn tên tệp làm mã: đường dẫn phục vụ, tên tệp trên đĩa và
// giá trị trong form đều là một chữ, không phải map ba chiều để dịch qua lại.
const (
	loaiLogo = "logo"
	loaiIcon = "bieu-tuong"
)

var (
	lgMu  sync.RWMutex
	lgTen = map[string]string{} // loại -> tên tệp trong data/logo
	lgKhi = map[string]int64{}  // loại -> giây sửa lần cuối, làm ?v= cho cache
)

func thuMucLogo() string { return P("data/logo") }

var mimeLogo = map[string]string{
	".svg":  "image/svg+xml",
	".png":  "image/png",
	".webp": "image/webp",
	".jpg":  "image/jpeg",
	".ico":  "image/x-icon",
}

// NapLogo quét thư mục một lần, kết quả nằm trong bộ nhớ. Mỗi lần tải hay xoá
// đều quét lại, nên không có chuyện trang hiện tệp đã xoá.
func NapLogo() error {
	ds, err := os.ReadDir(thuMucLogo())
	if os.IsNotExist(err) {
		lgMu.Lock()
		lgTen, lgKhi = map[string]string{}, map[string]int64{}
		lgMu.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	ten := map[string]string{}
	khi := map[string]int64{}
	for _, e := range ds {
		if e.IsDir() {
			continue
		}
		n := e.Name()
		duoi := strings.ToLower(filepath.Ext(n))
		goc := strings.TrimSuffix(n, filepath.Ext(n))
		if goc != loaiLogo && goc != loaiIcon {
			continue
		}
		if _, ok := mimeLogo[duoi]; !ok {
			continue
		}
		ten[goc] = n
		if in, err := e.Info(); err == nil {
			khi[goc] = in.ModTime().Unix()
		}
	}
	lgMu.Lock()
	lgTen, lgKhi = ten, khi
	lgMu.Unlock()
	return nil
}

func tepLogo(loai string) (string, int64) {
	lgMu.RLock()
	defer lgMu.RUnlock()
	return lgTen[loai], lgKhi[loai]
}

// --- Hàm cho template -----------------------------------------------

// logoURL rỗng nghĩa là chưa tải logo riêng — template dựng SVG mặc định.
func logoURL() string {
	ten, khi := tepLogo(loaiLogo)
	if ten == "" {
		return ""
	}
	return fmt.Sprintf("/logo?v=%d", khi)
}

func iconURL() string {
	ten, khi := tepLogo(loaiIcon)
	if ten == "" {
		return duongFavicon()
	}
	return fmt.Sprintf("/bieu-tuong?v=%d", khi)
}

func iconMIME() string {
	ten, _ := tepLogo(loaiIcon)
	if ten == "" {
		return "image/svg+xml"
	}
	return mimeLogo[strings.ToLower(filepath.Ext(ten))]
}

// --- Phục vụ cho khách ----------------------------------------------

func phucVuLogo(loai string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ten, _ := tepLogo(loai)
		if ten == "" {
			http.NotFound(w, r)
			return
		}
		f, err := os.Open(filepath.Join(thuMucLogo(), ten))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", mimeLogo[strings.ToLower(filepath.Ext(ten))])
		// SVG là tệp XML, mở thẳng bằng URL thì trình duyệt chạy script trong
		// đó. Chỉ chủ trạm mới tải lên được, nhưng hai dòng này rẻ hơn nhiều
		// so với việc đi giải thích vì sao trang có script lạ.
		w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		// Có ?v= đổi theo lần sửa nên cache lâu được: đổi logo là URL đổi.
		if r.URL.Query().Get("v") != "" {
			w.Header().Set("Cache-Control", "public, max-age=604800")
		}
		io.Copy(w, f)
	}
}

// --- Trang quản trị -------------------------------------------------

// duoiLogo đọc mấy byte đầu chứ không tin phần mở rộng của tên tệp: tên do
// máy khách gửi lên, đổi tên .exe thành .png là chuyện của một cú F2.
func duoiLogo(dau []byte) string {
	if d := duoiAnh(dau); d != "" {
		return d
	}
	s := strings.TrimSpace(strings.ToLower(string(dau)))
	if strings.HasPrefix(s, "<svg") || strings.HasPrefix(s, "<?xml") {
		return ".svg"
	}
	if len(dau) > 4 && dau[0] == 0x00 && dau[1] == 0x00 && dau[2] == 0x01 && dau[3] == 0x00 {
		return ".ico"
	}
	return ""
}

func veGiaoDien(w http.ResponseWriter, r *http.Request, ok, loi string) {
	duong := "/qt/giao-dien"
	switch {
	case loi != "":
		duong += "?loi=" + urlEsc(loi)
	case ok != "":
		duong += "?ok=" + urlEsc(ok)
	}
	http.Redirect(w, r, duong, http.StatusSeeOther)
}

func hQtLogoTai(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, logoToiDaByte+1<<20)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		veGiaoDien(w, r, "", "Tệp quá nặng — logo nên dưới 2 MB")
		return
	}

	// Biểu mẫu có tệp: lớp bọc không đọc được thân yêu cầu nên không kiểm
	// CSRF hộ được. Xem core/csrf.go.
	if !KiemCSRFMultipart(w, r) {
		return
	}
	loai := r.FormValue("loai")
	if loai != loaiLogo && loai != loaiIcon {
		veGiaoDien(w, r, "", "Không hiểu yêu cầu")
		return
	}
	ds := r.MultipartForm.File["tep"]
	if len(ds) == 0 {
		veGiaoDien(w, r, "", "Chưa chọn tệp")
		return
	}
	f, err := ds[0].Open()
	if err != nil {
		veGiaoDien(w, r, "", "Không mở được tệp")
		return
	}
	defer f.Close()

	dau := make([]byte, 512)
	n, _ := io.ReadFull(f, dau)
	duoi := duoiLogo(dau[:n])
	if duoi == "" {
		veGiaoDien(w, r, "", "Chỉ nhận SVG, PNG, WEBP, JPG hoặc ICO")
		return
	}

	if err := os.MkdirAll(thuMucLogo(), 0o755); err != nil {
		veGiaoDien(w, r, "", err.Error())
		return
	}
	// Ghi ra tệp tạm trước rồi mới xoá bản cũ và đổi tên: hỏng giữa chừng thì
	// logo cũ còn nguyên, chứ không rơi vào cảnh mất bản cũ mà bản mới dở dang.
	tam := filepath.Join(thuMucLogo(), loai+".tam")
	out, err := os.Create(tam)
	if err != nil {
		veGiaoDien(w, r, "", err.Error())
		return
	}
	if _, err := out.Write(dau[:n]); err == nil {
		_, err = io.Copy(out, io.LimitReader(f, logoToiDaByte))
	}
	if err := out.Close(); err != nil {
		os.Remove(tam)
		veGiaoDien(w, r, "", err.Error())
		return
	}
	xoaLogoCu(loai)
	if err := os.Rename(tam, filepath.Join(thuMucLogo(), loai+duoi)); err != nil {
		os.Remove(tam)
		veGiaoDien(w, r, "", err.Error())
		return
	}
	if err := NapLogo(); err != nil {
		veGiaoDien(w, r, "", err.Error())
		return
	}
	if loai == loaiLogo {
		veGiaoDien(w, r, "Đã đổi logo — tải lại trang khách là thấy", "")
	} else {
		veGiaoDien(w, r, "Đã đổi biểu tượng tab", "")
	}
}

func hQtLogoXoa(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	loai := r.FormValue("loai")
	if loai != loaiLogo && loai != loaiIcon {
		veGiaoDien(w, r, "", "Không hiểu yêu cầu")
		return
	}
	xoaLogoCu(loai)
	if err := NapLogo(); err != nil {
		veGiaoDien(w, r, "", err.Error())
		return
	}
	veGiaoDien(w, r, "Đã bỏ tệp tải lên — quay về bản mặc định", "")
}

// xoaLogoCu dọn mọi đuôi của một loại. Tải PNG đè lên bản SVG cũ mà không dọn
// thì thư mục còn hai tệp, và lần quét sau lấy cái nào là tuỳ thứ tự đọc.
func xoaLogoCu(loai string) {
	for duoi := range mimeLogo {
		os.Remove(filepath.Join(thuMucLogo(), loai+duoi))
	}
}
