package core

// Màn soạn bài viết trong khu quản trị.
//
// Ô soạn là một <textarea> markdown trần, không có thanh công cụ. Kendy đã
// chốt vậy: một trình soạn thảo WYSIWYG là thêm một thư viện ngoài, thêm một
// tầng có thể dựng ra HTML mà mình không kiểm được, đổi lấy tiện lợi cho
// đúng mười ba bài mỗi năm. Bù lại phải có nút Xem thử — gõ markdown mà
// không thấy trước thì đúng là bắt người ta viết mù.
//
// Xem thử dựng ở SERVER chứ không dựng bằng JavaScript trên trang. Dựng hai
// nơi là có hai trình dựng, và cái nhìn thấy lúc xem thử sẽ dần lệch khỏi
// cái lên trang thật — mà lệch chỗ đó là chỗ tệ nhất để lệch.

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

const anhBaiToiDaByte = 8 << 20

type dlQtBai struct {
	dlQt
	DS  []BaiViet
	Bai BaiViet
	Moi bool
	// Anh: ảnh đã tải lên, để chèn vào bài bằng cách sao đường dẫn.
	Anh []string
	// Hinh: tên các hình vẽ có sẵn gọi được bằng :::hinh
	Hinh map[string]string
}

func veBaiViet(w http.ResponseWriter, r *http.Request, duong, ok, loi string) {
	switch {
	case loi != "":
		duong += "?loi=" + urlEsc(loi)
	case ok != "":
		duong += "?ok=" + urlEsc(ok)
	}
	http.Redirect(w, r, duong, http.StatusSeeOther)
}

func hQtBaiDs(w http.ResponseWriter, r *http.Request) {
	loai := r.URL.Query().Get("loai")
	trang := "bai-viet-qt"
	if loai == "ve-tinh" {
		trang = "bai-viet-qt-ve-tinh"
	}
	d := dlQtBai{dlQt: dlQt{Chung: chung(r, trang)}}
	d.OK = r.URL.Query().Get("ok")
	d.Loi = r.URL.Query().Get("loi")
	if loai == "ve-tinh" {
		d.DS = DsBaiVietVeTinh()
	} else {
		d.DS = DsBaiVietQt()
	}
	render(w, "qt-baiviet.html", d)
}

func hQtBaiMoi(w http.ResponseWriter, r *http.Request) {
	d := dlQtBai{dlQt: dlQt{Chung: chung(r, "bai-viet-qt")}}
	d.Moi = true
	d.Loi = r.URL.Query().Get("loi")
	d.Anh = dsAnhBai()
	d.Hinh = hinhChoPhep
	// Bài mới mặc định là nháp và xếp cuối: viết dở nửa chừng mà đã nằm trên
	// trang khách thì không sửa lại được ấn tượng đầu tiên.
	d.Bai = BaiViet{
		Ngay:  time.Now().Format("2006-01-02"),
		ThuTu: thuTuKeTiep(),
		Nhap:  true,
	}
	render(w, "qt-baiviet-sua.html", d)
}

func thuTuKeTiep() int {
	cao := 0
	for _, b := range DsBaiVietQt() {
		if b.ThuTu > cao {
			cao = b.ThuTu
		}
	}
	return cao + 10
}

func hQtBaiSua(w http.ResponseWriter, r *http.Request) {
	bai, ok := TimBaiViet(r.PathValue("slug"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	d := dlQtBai{dlQt: dlQt{Chung: chung(r, "bai-viet-qt")}}
	d.OK = r.URL.Query().Get("ok")
	d.Loi = r.URL.Query().Get("loi")
	d.Bai = bai
	d.Anh = dsAnhBai()
	d.Hinh = hinhChoPhep
	render(w, "qt-baiviet-sua.html", d)
}

// hQtBaiLuu nhận cả bài mới lẫn bài sửa. Đường dẫn cũ đi trong ô ẩn "cu":
// đổi đường dẫn của một bài đã đăng là đổi luôn địa chỉ khách đã lưu, nên
// phải biết bài cũ tên gì mà xoá tệp cũ đi.
func hQtBaiLuu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		veBaiViet(w, r, "/qt/bai-viet", "", "Bài quá dài")
		return
	}
	b := baiTuForm(r)
	cu := strings.TrimSpace(r.FormValue("cu"))
	ve := "/qt/bai-viet/" + b.Slug
	if cu != "" {
		ve = "/qt/bai-viet/" + cu
	}

	if !slugSach(b.Slug) {
		veBaiViet(w, r, ve, "", "Đường dẫn chỉ được có chữ thường không dấu, số và gạch nối")
		return
	}
	if cu != b.Slug {
		if _, trung := TimBaiViet(b.Slug); trung {
			veBaiViet(w, r, ve, "", "Đã có bài dùng đường dẫn "+b.Slug)
			return
		}
	}
	if err := LuuBaiViet(b); err != nil {
		veBaiViet(w, r, ve, "", err.Error())
		return
	}
	// Đổi đường dẫn: ghi tệp mới xong mới xoá tệp cũ. Ngược lại thì lỡ ghi
	// hỏng là mất bài.
	if cu != "" && cu != b.Slug {
		if err := XoaBaiViet(cu); err != nil {
			veBaiViet(w, r, "/qt/bai-viet/"+b.Slug, "", "Đã lưu bài mới nhưng chưa xoá được bài cũ: "+err.Error())
			return
		}
	}
	veBaiViet(w, r, "/qt/bai-viet/"+b.Slug, "Đã lưu", "")
}

func baiTuForm(r *http.Request) BaiViet {
	thu, _ := strconv.Atoi(strings.TrimSpace(r.FormValue("thu_tu")))
	loai := strings.TrimSpace(r.FormValue("loai"))
	if loai != "ve-tinh" {
		loai = ""
	}
	return BaiViet{
		Slug:      strings.TrimSpace(r.FormValue("slug")),
		ThuTu:     thu,
		TieuDe:    strings.TrimSpace(r.FormValue("tieu_de")),
		TieuDeSEO: strings.TrimSpace(r.FormValue("tieu_de_seo")),
		MoTa:      strings.TrimSpace(r.FormValue("mo_ta")),
		Ngay:      ngayISO(r.FormValue("ngay")),
		Sua:       ngayISO(r.FormValue("sua")),
		TomTat:    strings.TrimSpace(r.FormValue("tom_tat")),
		Anh:       strings.TrimSpace(r.FormValue("anh")),
		Nhap:      r.FormValue("nhap") != "",
		HenGio:    strings.TrimSpace(r.FormValue("hen_gio")),
		Loai:      loai,
		Than:      r.FormValue("than"),
	}
}

func hQtBaiXoa(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if err := XoaBaiViet(slug); err != nil {
		veBaiViet(w, r, "/qt/bai-viet", "", err.Error())
		return
	}
	veBaiViet(w, r, "/qt/bai-viet", "Đã xoá bài "+slug, "")
}

// hQtBaiXemThu trả về một mẩu HTML, không phải cả trang: trang soạn gọi
// bằng fetch rồi nhét vào khung xem thử. Dùng đúng MarkdownBai như lúc lên
// trang thật, kể cả class="dat" của đoạn dẫn.
func hQtBaiXemThu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bài quá dài", http.StatusRequestEntityTooLarge)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, MarkdownBai(r.FormValue("than")))
}

// --- Ảnh trong bài -----------------------------------------------------

func dsAnhBai() []string {
	muc, err := os.ReadDir(thuMucAnhBai())
	if err != nil {
		return nil
	}
	var ds []string
	for _, m := range muc {
		if !m.IsDir() && tenAnhSach(m.Name()) {
			ds = append(ds, m.Name())
		}
	}
	return ds
}

func hQtBaiAnhThem(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, anhBaiToiDaByte+1<<20)
	ve := "/qt/bai-viet/" + r.FormValue("ve")
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		veBaiViet(w, r, ve, "", "Ảnh quá nặng")
		return
	}

	// Biểu mẫu có tệp: lớp bọc không đọc được thân yêu cầu nên không kiểm
	// CSRF hộ được. Xem core/csrf.go.
	if !KiemCSRFMultipart(w, r) {
		return
	}
	if v := strings.TrimSpace(r.FormValue("ve")); slugSach(v) {
		ve = "/qt/bai-viet/" + v
	} else {
		ve = "/qt/bai-viet/moi"
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
		os.MkdirAll(thuMucAnhBai(), 0o755)
		ten := fmt.Sprintf("bai-%s-%s%s", time.Now().Format("150405"), maNgauNhien(3), duoi)
		out, err := os.Create(filepath.Join(thuMucAnhBai(), ten))
		if err != nil {
			f.Close()
			continue
		}
		out.Write(dau[:n])
		io.Copy(out, io.LimitReader(f, anhBaiToiDaByte))
		out.Close()
		f.Close()
		them++
	}
	if them == 0 {
		veBaiViet(w, r, ve, "", "Không nhận được ảnh nào — chỉ nhận JPG, PNG, WEBP")
		return
	}
	veBaiViet(w, r, ve, fmt.Sprintf("Đã tải lên %d ảnh", them), "")
}

// hQtBaiAnhXoa xoá hẳn tệp ảnh. Không dò xem bài nào đang dùng: mười ba bài
// thì Kendy nhìn là biết, mà dò tự động lại đẻ ra chuyện "xoá không được vì
// còn một bài nháp nào đó tham chiếu tới".
func hQtBaiAnhXoa(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	r.ParseForm()
	ten := r.FormValue("ten")
	ve := "/qt/bai-viet/moi"
	if v := strings.TrimSpace(r.FormValue("ve")); slugSach(v) {
		ve = "/qt/bai-viet/" + v
	}
	if !tenAnhSach(ten) {
		veBaiViet(w, r, ve, "", "Tên ảnh không hợp lệ")
		return
	}
	if err := os.Remove(filepath.Join(thuMucAnhBai(), ten)); err != nil && !os.IsNotExist(err) {
		veBaiViet(w, r, ve, "", err.Error())
		return
	}
	veBaiViet(w, r, ve, "Đã xoá ảnh", "")
}
