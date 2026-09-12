package core

// Màn chỉnh app trên điện thoại — /qt/app.
//
// Gom về MỘT chỗ mọi thứ quyết định màn hình khách thấy khi mở app từ icon:
// ảnh banner, chữ, khối khuyến mãi và đợt chạy của nó, icon nhúc nhích kiểu
// gì. Trước đây chữ nằm ở /qt/noi-dung còn ảnh thì không sửa được — Kendy
// phải nhớ hai chỗ, mà một nửa việc thì không có chỗ nào làm được.
//
// Chữ vẫn ĐI QUA CayND như cũ: cùng kho chữ, cùng file yaml, cùng luật
// mặc-định-nằm-trong-code. Trang này chỉ dựng ô nhập từ nhánh "app" của cây
// rồi gọi DatND — không có đường chữ thứ hai. Đổi lại tab "App trên điện
// thoại" ở /qt/noi-dung ẩn đi (TrangND.Rieng), để không có hai chỗ sửa cùng
// một câu rồi đá nhau.
//
// Phần không phải chữ nằm trong AppCauHinh, ghi ra data/app.yaml.

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type dlQtApp struct {
	dlQt
	// Hien chứ không phải Trang: Chung đã có Trang là mã trang (chuỗi) cho
	// thanh bên tô đậm mục đang mở. Đặt trùng tên là trường ngoài che trường
	// nhúng, phan.html so chuỗi với TrangND rồi nổ giữa lúc render.
	Hien       TrangND
	Nhom       []ndNhomQt
	CH         AppCauHinh
	Anh        []AnhHero // cả kho, để chọn tấm nào làm banner
	AnhApp     []AnhHero // riêng mấy tấm đang treo ở chỗ "app"
	KMDangChay bool
	HomNay     string
}

func veApp(w http.ResponseWriter, r *http.Request, ok, loi string) {
	u := "/qt/app"
	switch {
	case loi != "":
		u += "?loi=" + urlEsc(loi)
	case ok != "":
		u += "?ok=" + urlEsc(ok)
	}
	http.Redirect(w, r, u, http.StatusSeeOther)
}

// trangApp là nhánh "app" của cây nội dung. Không có nhánh ấy thì trang này
// vô nghĩa — nhưng đừng sập, cứ hiện phần cấu hình.
func trangApp() TrangND {
	t, _ := timTrangND("app")
	return t
}

func hQtApp(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		hQtAppLuu(w, r)
		return
	}
	t := trangApp()
	d := dlQtApp{dlQt: dlQt{Chung: chung(r, "app-qt")}, Hien: t}
	d.OK = r.URL.Query().Get("ok")
	d.Loi = r.URL.Query().Get("loi")
	d.CH = AppCauHinhHienTai()
	d.Anh = DsAnhHero()
	d.AnhApp = AnhBannerApp()
	d.KMDangChay = d.CH.KMHien()
	d.HomNay = time.Now().Format("2006-01-02")
	for _, n := range t.Nhom {
		nh := ndNhomQt{Ten: n.Ten}
		for _, m := range n.Muc {
			nh.Muc = append(nh.Muc, ndMucQt{MucND: m, GiaTri: ND(m.Khoa), DaSua: DaSuaND(m.Khoa)})
		}
		d.Nhom = append(d.Nhom, nh)
	}
	render(w, "qt-app.html", d)
}

// hQtAppLuu nhận cả chữ lẫn cấu hình trong một lần bấm. Hai kho khác nhau
// nhưng với Kendy đây là một màn hình — bắt bấm hai nút Lưu thì kiểu gì cũng
// có lần gõ xong nửa trên rồi lưu nửa dưới.
func hQtAppLuu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		veApp(w, r, "", "Nội dung quá dài")
		return
	}

	// Chữ: duyệt theo cây chứ không duyệt theo form. Form là thứ trình duyệt
	// gửi lên, cây là thứ mình biết chắc.
	moi := map[string]string{}
	for _, n := range trangApp().Nhom {
		for _, m := range n.Muc {
			if v, gui := r.Form["k."+m.Khoa]; gui {
				moi[m.Khoa] = v[0]
			}
		}
	}
	if err := DatND(moi); err != nil {
		veApp(w, r, "", err.Error())
		return
	}

	c := AppCauHinhHienTai()
	c.AnBanner = r.FormValue("an_banner") != ""
	c.BannerAnh = r.FormValue("banner_anh")
	c.BannerToi, _ = strconv.Atoi(r.FormValue("banner_toi"))
	c.KMTu = r.FormValue("km_tu")
	c.KMDen = r.FormValue("km_den")
	c.KMLink = r.FormValue("km_link")
	c.IconDong = r.FormValue("icon_dong")
	c.IconNhip, _ = strconv.Atoi(r.FormValue("icon_nhip"))
	if _, err := DatAppCauHinh(c); err != nil {
		// Chữ đã lưu rồi — nói rõ để Kendy khỏi gõ lại cả trang.
		veApp(w, r, "", "Chữ đã lưu, phần cài đặt thì chưa: "+err.Error())
		return
	}
	veApp(w, r, "Đã lưu — mở /app trên điện thoại xem là thấy ngay", "")
}

// hQtAppAnh nhận ảnh banner. Ảnh vào chung kho với ảnh trang chủ nhưng đánh
// dấu noi="app", và chọn luôn làm banner: up ảnh xong mà còn phải đi tick một
// ô nữa thì đằng nào cũng có lần quên.
func hQtAppAnh(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, anhHeroToiDaByte+1<<20)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		veApp(w, r, "", "Ảnh quá nặng")
		return
	}
	if !KiemCSRFMultipart(w, r) {
		return
	}
	fhs := r.MultipartForm.File["anh"]
	if len(fhs) == 0 {
		veApp(w, r, "", "Chưa chọn ảnh nào")
		return
	}
	fh := fhs[0]
	f, err := fh.Open()
	if err != nil {
		veApp(w, r, "", "Không mở được tệp")
		return
	}
	defer f.Close()

	dau := make([]byte, 16)
	n, _ := io.ReadFull(f, dau)
	duoi := duoiAnh(dau[:n])
	if duoi == "" {
		veApp(w, r, "", "Chỉ nhận JPG, PNG, WEBP")
		return
	}
	os.MkdirAll(thuMucAnhHero(), 0o755)
	ten := fmt.Sprintf("app-%s-%s%s", time.Now().Format("150405"), maNgauNhien(3), duoi)
	duong := filepath.Join(thuMucAnhHero(), ten)
	out, err := os.Create(duong)
	if err != nil {
		veApp(w, r, "", "Không ghi được tệp")
		return
	}
	out.Write(dau[:n])
	io.Copy(out, io.LimitReader(f, anhHeroToiDaByte))
	out.Close()

	if err := ThemAnhHero(ten, "Banner app"); err != nil {
		os.Remove(duong)
		veApp(w, r, "", err.Error())
		return
	}
	if err := DatNoiAnhHero(ten, NoiApp); err != nil {
		veApp(w, r, "", err.Error())
		return
	}
	c := AppCauHinhHienTai()
	c.BannerAnh = ten
	if _, err := DatAppCauHinh(c); err != nil {
		veApp(w, r, "", err.Error())
		return
	}
	veApp(w, r, "Đã tải ảnh lên và đặt làm banner", "")
}

// hQtAppVideo nhận video nền cho banner.
//
// Đường riêng chứ không gộp vào hQtAppAnh: hai thứ khác trần dung lượng, khác
// cách nhận dạng, và khác chỗ lưu. Gộp lại thì mỗi lần đọc phải tự nhớ nhánh
// nào đang chạy.
//
// Tải cái mới lên là thay thẳng cái cũ — thư mục chỉ giữ đúng một tệp. Xoá tệp
// cũ SAU khi cấu hình đã trỏ sang tệp mới: ngược lại thì một lần lưu hỏng là
// mất cả hai.
func hQtAppVideo(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, videoAppToiDaByte+1<<20)
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		veApp(w, r, "", fmt.Sprintf("Video quá nặng — tối đa %dMB", videoAppToiDaByte>>20))
		return
	}
	if !KiemCSRFMultipart(w, r) {
		return
	}
	fhs := r.MultipartForm.File["video"]
	if len(fhs) == 0 {
		veApp(w, r, "", "Chưa chọn video nào")
		return
	}
	f, err := fhs[0].Open()
	if err != nil {
		veApp(w, r, "", "Không mở được tệp")
		return
	}
	defer f.Close()

	// 12 byte đầu là chỗ hộp ftyp nằm. Đọc riêng rồi ghi lại vào tệp, giống
	// hệt lối làm ở ảnh — đọc trước thì mất khúc đầu của luồng.
	dau := make([]byte, 12)
	n, _ := io.ReadFull(f, dau)
	ten, err := LuuVideoApp(dau[:n], f)
	if err != nil {
		veApp(w, r, "", err.Error())
		return
	}

	c := AppCauHinhHienTai()
	cu := c.BannerVideo
	c.BannerVideo = ten
	if _, err := DatAppCauHinh(c); err != nil {
		XoaVideoApp(ten)
		veApp(w, r, "", err.Error())
		return
	}
	if cu != "" && cu != ten {
		XoaVideoApp(cu)
	}
	veApp(w, r, "Đã tải video lên và đặt làm nền banner", "")
}

// hQtAppVideoXoa gỡ video. Một mức thôi, không như ảnh: kho video chỉ có đúng
// một tệp đang treo, "thôi dùng nhưng vẫn giữ" thì giữ để làm gì — không có
// danh sách nào để chọn lại từ đó.
func hQtAppVideoXoa(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	c := AppCauHinhHienTai()
	cu := c.BannerVideo
	c.BannerVideo = ""
	if _, err := DatAppCauHinh(c); err != nil {
		veApp(w, r, "", err.Error())
		return
	}
	XoaVideoApp(cu)
	veApp(w, r, "Đã bỏ video — banner quay về ảnh đang chọn", "")
}

// hQtAppAnhXoa gỡ ảnh khỏi banner. Hai mức: "bo" chỉ thôi dùng (ảnh vẫn nằm
// trong kho, chọn lại được), "xoa" mới xoá hẳn tệp.
func hQtAppAnhXoa(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	ten := strings.TrimSpace(r.FormValue("ten"))
	c := AppCauHinhHienTai()

	if r.FormValue("viec") == "xoa" {
		if !tenAnhSach(ten) {
			veApp(w, r, "", "Tên ảnh không hợp lệ")
			return
		}
		if err := XoaAnhHero(ten); err != nil {
			veApp(w, r, "", err.Error())
			return
		}
		if c.BannerAnh == ten {
			c.BannerAnh = ""
			DatAppCauHinh(c)
		}
		veApp(w, r, "Đã xoá ảnh khỏi kho", "")
		return
	}

	c.BannerAnh = ""
	if _, err := DatAppCauHinh(c); err != nil {
		veApp(w, r, "", err.Error())
		return
	}
	veApp(w, r, "Banner quay về bản vẽ vợt", "")
}

// hQtAppKhoiPhuc trả một ô chữ về mặc định trong code. Dùng lại KhoiPhucND
// nhưng có đường riêng để quay về /qt/app — mượn /qt/noi-dung/khoi-phuc thì
// lưu xong nó đá sang tab "chung" của trang kia.
func hQtAppKhoiPhuc(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	khoa := r.FormValue("khoa")
	thuoc := false
	for _, n := range trangApp().Nhom {
		for _, m := range n.Muc {
			if m.Khoa == khoa {
				thuoc = true
			}
		}
	}
	if !thuoc {
		veApp(w, r, "", "Khoá "+khoa+" không thuộc màn app")
		return
	}
	if err := KhoiPhucND(khoa); err != nil {
		veApp(w, r, "", err.Error())
		return
	}
	veApp(w, r, "Đã trả về chữ mặc định", "")
}
