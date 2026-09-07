package core

// Màn đọc và sửa giáo trình dạy nghề trong khu quản trị.
//
// Giáo trình là tài liệu NỘI BỘ. Nó nằm trong vanhanh/, không route nào đưa
// ra web khách, và màn này bọc canLaChu. Cố tình KHÔNG đăng sang
// data/bai-viet/: kho đó công khai, mà quy trình sửa, vật tư và giá vốn là
// thứ đối thủ muốn nhất.
//
// Danh sách bài không quét thư mục mà lấy từ tai_lieu_nap trong config, lọc
// theo tiền tố vanhanh/giao-trinh/. Hai cái lợi: màn này hiện đúng những bài
// Trợ lý kỹ thuật thật sự đọc — quét thư mục thì màn hình sẽ khoe cả bài
// agent không thấy — và chính danh sách ấy là whitelist, nên đường dẫn gõ
// trên trình duyệt không đi ra ngoài thư mục được.
//
// Sửa xong KHÔNG cần khởi động lại: nhánh nhồi-thẳng của agent đọc tệp mỗi
// lần hỏi. Chỉ thêm bớt bài trong config mới phải restart.
//
// Ô soạn là <textarea> markdown trần, cùng lý do đã chốt ở qtbaiviet.go, và
// nút Xem thử cũng dựng ở server bằng đúng trình dựng của trang bài viết.

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
)

// Tiền tố cố định. Bài nào trong tai_lieu_nap không bắt đầu bằng chuỗi này
// thì không phải giáo trình và không hiện ở đây.
const giaoTrinhGoc = "vanhanh/giao-trinh/"

// Bài dài nhất hiện là 15 KB. 512 KB là chỗ thở, đồng thời chặn người dán
// nhầm cả cuốn sách vào một ô.
const giaoTrinhToiDaByte = 512 << 10

// Tên phần cho dễ đọc. Thư mục không có trong bảng thì hiện nguyên tên.
var tenPhanGT = map[string]string{
	"":             "Chung",
	"01-nen-tang":  "1 · Nền tảng",
	"02-quy-trinh": "2 · Quy trình",
	"03-ca-sua":    "3 · Các ca sửa",
	"04-lam-nghe":  "4 · Làm nghề",
}

type BaiGiaoTrinh struct {
	Tep       string // đường dẫn tương đối gốc repo
	Ma        string // phần sau giaoTrinhGoc, dùng trong URL
	Phan      string // thư mục con, "" nếu nằm ngay gốc
	TenPhan   string // tên phần đã làm đẹp
	Ten       string // dòng "# ..." đầu tệp
	TrangThai string // 🟨 NHÁP / ✅ XONG / ⬜ CHƯA CÓ, đọc từ đầu bài
	Byte      int64
	Thieu     bool // có trong config nhưng không có tệp
}

type NhomGiaoTrinh struct {
	Ten string
	Bai []BaiGiaoTrinh
}

type dlQtGT struct {
	dlQt
	Nhom  []NhomGiaoTrinh
	Bai   BaiGiaoTrinh
	Than  string
	SoBai int
}

// DsGiaoTrinh đọc danh sách bài từ config, giữ nguyên thứ tự đã khai.
func DsGiaoTrinh() []BaiGiaoTrinh {
	var ds []BaiGiaoTrinh
	for _, rel := range CFG.DuongDan.TaiLieuNap {
		if !strings.HasPrefix(rel, giaoTrinhGoc) {
			continue
		}
		ma := strings.TrimPrefix(rel, giaoTrinhGoc)
		b := BaiGiaoTrinh{Tep: rel, Ma: ma, Phan: path.Dir(ma)}
		if b.Phan == "." {
			b.Phan = ""
		}
		b.TenPhan = tenPhanGT[b.Phan]
		if b.TenPhan == "" {
			b.TenPhan = b.Phan
		}
		b.Ten = path.Base(ma)
		if st, err := os.Stat(P(rel)); err == nil {
			b.Byte = st.Size()
		} else {
			b.Thieu = true
		}
		if raw, err := os.ReadFile(P(rel)); err == nil {
			b.Ten, b.TrangThai = docDauBaiGT(string(raw), path.Base(ma))
		}
		ds = append(ds, b)
	}
	return ds
}

// docDauBaiGT lấy tiêu đề và trạng thái từ vài dòng đầu. Tiêu đề là dòng "# "
// đầu tiên; trạng thái là ký hiệu trong dòng trích dẫn ngay dưới nó.
func docDauBaiGT(raw, mac string) (ten, trangThai string) {
	ten = mac
	dong := strings.Split(raw, "\n")
	if len(dong) > 30 {
		dong = dong[:30]
	}
	coTen := false
	for _, d := range dong {
		d = strings.TrimSpace(d)
		if !coTen && strings.HasPrefix(d, "# ") {
			ten = strings.TrimSpace(strings.TrimPrefix(d, "# "))
			coTen = true
			continue
		}
		switch {
		case strings.Contains(d, "✅"):
			return ten, "✅ Xong"
		case strings.Contains(d, "🟨"):
			return ten, "🟨 Nháp"
		case strings.Contains(d, "⬜ CHƯA CÓ"):
			return ten, "⬜ Chưa có"
		}
	}
	return ten, ""
}

// nhomGiaoTrinh gom bài theo phần, giữ thứ tự phần xuất hiện lần đầu.
func nhomGiaoTrinh(ds []BaiGiaoTrinh) []NhomGiaoTrinh {
	var nhom []NhomGiaoTrinh
	viTri := map[string]int{}
	for _, b := range ds {
		i, co := viTri[b.Phan]
		if !co {
			viTri[b.Phan] = len(nhom)
			nhom = append(nhom, NhomGiaoTrinh{Ten: b.TenPhan})
			i = len(nhom) - 1
		}
		nhom[i].Bai = append(nhom[i].Bai, b)
	}
	for i := range nhom {
		sort.SliceStable(nhom[i].Bai, func(a, b int) bool {
			return nhom[i].Bai[a].Ma < nhom[i].Bai[b].Ma
		})
	}
	return nhom
}

// timBaiGiaoTrinh so mã người dùng gõ với danh sách trong config. So bằng
// chuỗi chứ không ghép đường dẫn rồi kiểm tra sau: ghép trước là mở cửa cho
// ../../ đi ra ngoài thư mục.
func timBaiGiaoTrinh(ma string) (BaiGiaoTrinh, bool) {
	for _, b := range DsGiaoTrinh() {
		if b.Ma == ma {
			return b, true
		}
	}
	return BaiGiaoTrinh{}, false
}

func veGiaoTrinh(w http.ResponseWriter, r *http.Request, duong, ok, loi string) {
	switch {
	case loi != "":
		duong += "?loi=" + urlEsc(loi)
	case ok != "":
		duong += "?ok=" + urlEsc(ok)
	}
	http.Redirect(w, r, duong, http.StatusSeeOther)
}

func hQtGTDs(w http.ResponseWriter, r *http.Request) {
	ds := DsGiaoTrinh()
	d := dlQtGT{dlQt: dlQt{Chung: chung(r, "giao-trinh")}}
	d.OK = r.URL.Query().Get("ok")
	d.Loi = r.URL.Query().Get("loi")
	d.Nhom = nhomGiaoTrinh(ds)
	d.SoBai = len(ds)
	render(w, "qt-giaotrinh.html", d)
}

func hQtGTSua(w http.ResponseWriter, r *http.Request) {
	bai, ok := timBaiGiaoTrinh(r.PathValue("ma"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	raw, err := os.ReadFile(P(bai.Tep))
	if err != nil {
		veGiaoTrinh(w, r, "/qt/giao-trinh", "", "Không đọc được tệp: "+err.Error())
		return
	}
	d := dlQtGT{dlQt: dlQt{Chung: chung(r, "giao-trinh")}}
	d.OK = r.URL.Query().Get("ok")
	d.Loi = r.URL.Query().Get("loi")
	d.Bai = bai
	d.Than = string(raw)
	render(w, "qt-giaotrinh-sua.html", d)
}

func hQtGTLuu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, giaoTrinhToiDaByte)
	if err := r.ParseForm(); err != nil {
		veGiaoTrinh(w, r, "/qt/giao-trinh", "", "Bài quá dài")
		return
	}
	bai, ok := timBaiGiaoTrinh(strings.TrimSpace(r.FormValue("ma")))
	if !ok {
		http.NotFound(w, r)
		return
	}
	ve := "/qt/giao-trinh/bai/" + bai.Ma

	than := strings.ReplaceAll(r.FormValue("than"), "\r\n", "\n")
	// Lưu bài rỗng là xoá trắng một bài giáo trình bằng một cú bấm nhầm.
	// Muốn bỏ bài thì bỏ khỏi tai_lieu_nap, không phải xoá ruột.
	if strings.TrimSpace(than) == "" {
		veGiaoTrinh(w, r, ve, "", "Bài trống — muốn bỏ bài thì bỏ khỏi tai_lieu_nap trong config")
		return
	}
	if !strings.HasSuffix(than, "\n") {
		than += "\n"
	}
	if err := os.WriteFile(P(bai.Tep), []byte(than), 0o644); err != nil {
		veGiaoTrinh(w, r, ve, "", "Không ghi được tệp: "+err.Error())
		return
	}
	veGiaoTrinh(w, r, ve, "Đã lưu — Trợ lý kỹ thuật dùng bản mới ngay, không cần khởi động lại", "")
}

func hQtGTXemThu(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, giaoTrinhToiDaByte)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bài quá dài", http.StatusRequestEntityTooLarge)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, MarkdownBai(r.FormValue("than")))
}
