// Trang /qt/khach — hồ sơ khách của trạm.
//
// Chỉ chủ vào được: công nợ và tổng chi tiêu là chuyện tiền, cùng nhóm với
// sổ tiền và thống kê. Thợ vẫn thấy gợi ý khách quen lúc lập đơn — chỗ ấy
// chỉ cần tên với số, không có con số nào của sổ.
//
// Bộ lọc "chưa quay lại quá N tháng" là lý do chính khiến trang này đáng tồn
// tại: nó cho Kendy một danh sách để nhắn Zalo, chứ không phải một bảng đẹp
// để ngắm.
package core

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

// bolocKhachTuURL đọc bộ lọc từ query. Giữ nguyên trên mọi link (CSV, phân
// nhóm) để tệp tải về đúng bằng bảng đang nhìn.
func bolocKhachTuURL(r *http.Request) BoLocKhach {
	f := BoLocKhach{
		Tim:   strings.TrimSpace(r.URL.Query().Get("q")),
		ConNo: r.URL.Query().Get("con_no") == "1",
	}
	if n := r.URL.Query().Get("nhom"); n != "" {
		if _, co := TimNhomKhach(n); co {
			f.Nhom = n
		}
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("vang")); err == nil && v > 0 {
		f.VangTuThang = v
	}
	return f
}

func hQtKhach(w http.ResponseWriter, r *http.Request) {
	c := chung(r, "khach")
	c.TieuDe = "Khách hàng"
	f := bolocKhachTuURL(r)
	render(w, "qt-khach.html", struct {
		dlQt
		Khach    []DongKhach
		Loc      BoLocKhach
		Nhom     []NhomKhach
		TongQuan TongQuanKhach
	}{
		dlQt:     dlQt{Chung: c, OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		Khach:    LocKhach(f),
		Loc:      f,
		Nhom:     CacNhomKhach,
		TongQuan: LayTongQuanKhach(),
	})
}

func hQtKhachChiTiet(w http.ResponseWriter, r *http.Request) {
	k, ok := KhachTheoMa(r.PathValue("ma"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	c := chung(r, "khach")
	c.TieuDe = "Khách " + k.Ten
	render(w, "qt-khach-ct.html", struct {
		dlQt
		Khach  KhachHang
		SoLieu SoLieuKhach
		Don    []*Don
		Nhom   []NhomKhach
	}{
		dlQt:   dlQt{Chung: c, OK: r.URL.Query().Get("ok"), Loi: r.URL.Query().Get("loi")},
		Khach:  k,
		SoLieu: SoLieuCuaKhach(k.Ma),
		Don:    LichSuKhach(k.Ma),
		Nhom:   CacNhomKhach,
	})
}

func hQtKhachLuu(w http.ResponseWriter, r *http.Request) {
	ma := r.PathValue("ma")
	cu, ok := KhachTheoMa(ma)
	if !ok {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Nội dung quá dài", http.StatusRequestEntityTooLarge)
		return
	}
	cu.Ten = catBot(r.FormValue("ten"), 100)
	cu.LienHe = catBot(r.FormValue("lien_he"), 60)
	cu.Email = catBot(r.FormValue("email"), 254)
	cu.DiaChi = catBot(r.FormValue("dia_chi"), 200)
	cu.GhiChu = catBot(r.FormValue("ghi_chu"), 500)
	cu.Nhom = r.FormValue("nhom")
	cu.Ngung = r.FormValue("ngung") == "1"
	if _, err := LuuKhach(cu); err != nil {
		http.Redirect(w, r, "/qt/khach/"+ma+"?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/qt/khach/"+ma+"?ok=Đã+lưu", http.StatusSeeOther)
}

// hQtDonGanKhach — nút "Gắn vào hồ sơ khách" trên đơn cũ chưa có MaKhach.
func hQtDonGanKhach(w http.ResponseWriter, r *http.Request) {
	nd, _ := NguoiDangNhap(r)
	don, ok := LayDon(strings.ToUpper(r.PathValue("ma")))
	if !ok || !xemDuocDon(nd, don) {
		http.NotFound(w, r)
		return
	}
	ma := GhiNhanKhach(don.KhachTen, don.KhachLienHe, don.KhachEmail, don.KhachDiaChi)
	if ma == "" {
		http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok="+urlEsc("Đơn này chưa có số điện thoại nên chưa dựng được hồ sơ khách."), http.StatusSeeOther)
		return
	}
	don.MaKhach = ma
	if err := LuuDon(don); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/qt/don/"+don.Ma+"?ok=Đã+gắn+vào+hồ+sơ+khách", http.StatusSeeOther)
}

// --- Xuất CSV --------------------------------------------------------

// hQtXuatKhach — đúng bảng đang hiện, dưới dạng tệp. Cùng khuôn hQtXuatDon:
// BOM UTF-8 và ô bọc nháy nằm trong guiCSV/oCSV, tiền ghi số trần để Excel
// cộng được.
func hQtXuatKhach(w http.ResponseWriter, r *http.Request) {
	ds := LocKhach(bolocKhachTuURL(r))
	var b strings.Builder
	dongCSV(&b, "Mã", "Tên", "Điện thoại", "Email", "Địa chỉ", "Nhóm",
		"Số đơn", "Tổng chi", "Còn nợ", "Lần đầu", "Lần cuối", "Ngày vắng", "Ghi chú")
	for _, k := range ds {
		dongCSV(&b, k.Ma, k.Ten, k.LienHe, k.Email, k.DiaChi, k.TenNhom(),
			strconv.Itoa(k.SoDon), strconv.Itoa(k.TongChi), strconv.Itoa(k.SoLieuKhach.ConNo),
			k.LanDau, k.SoLieuKhach.LanCuoi, strconv.Itoa(k.NgayVang), k.GhiChu)
	}
	guiCSV(w, tenTepCSV("khach", Ky{Moc: "tat-ca"}), b.String())
}

// --- Nhập CSV --------------------------------------------------------

// KetQuaNhapKhach — bảng tổng kết hiện lại sau khi đổ tệp.
//
// Dòng bỏ qua phải kể ra bằng số dòng và lý do, không im lặng: Kendy đổ danh
// bạ CLB một trăm dòng vào, mất ba dòng mà không ai nói gì thì ba người ấy
// biến mất khỏi trạm và không bao giờ ai biết.
type KetQuaNhapKhach struct {
	ThemMoi int
	CapNhat int
	BoQua   []string
}

// NhapKhachCSV đọc tệp hai cột tối thiểu tên + số điện thoại. Cột thừa
// (email, địa chỉ, nhóm, ghi chú) nhận theo tên tiêu đề nếu có.
//
// Trùng SoChuan thì cập nhật chứ không tạo trùng, và chỉ điền vào ô đang
// TRỐNG — đổ lại tệp cũ lần thứ hai không được xoá công sửa tay.
func NhapKhachCSV(r io.Reader) (KetQuaNhapKhach, error) {
	var kq KetQuaNhapKhach
	c := csv.NewReader(r)
	c.FieldsPerRecord = -1
	c.TrimLeadingSpace = true
	dong, err := c.ReadAll()
	if err != nil {
		return kq, fmt.Errorf("tệp không đọc được: %w", err)
	}
	if len(dong) == 0 {
		return kq, fmt.Errorf("tệp rỗng")
	}

	// Sao lưu TRƯỚC khi ghi. Không có nút hoàn tác, nên phải cứu được bằng
	// tay khi Kendy đổ nhầm tệp.
	if b, err := os.ReadFile(fileKhach()); err == nil {
		_ = ghiAtomic(fileKhach()+".truoc-nhap", b)
	}

	cot := doCotKhach(dong[0])
	bat := 0
	if cot.coTieuDe {
		bat = 1
	}

	for i := bat; i < len(dong); i++ {
		o := dong[i]
		soDong := i + 1
		lay := func(n int) string {
			if n < 0 || n >= len(o) {
				return ""
			}
			return strings.TrimSpace(o[n])
		}
		ten := lay(cot.ten)
		lienHe := lay(cot.lienHe)
		if ChuanSo(lienHe) == "" {
			if ten == "" && lienHe == "" {
				continue // dòng trống cuối tệp, không phải lỗi
			}
			kq.BoQua = append(kq.BoQua, fmt.Sprintf("dòng %d (%s): thiếu số điện thoại", soDong, ten))
			continue
		}
		nhom := chuanNhomKhach(lay(cot.nhom))
		if cu, co := KhachTheoSo(lienHe); co {
			GhiNhanKhach(ten, lienHe, lay(cot.email), lay(cot.diaChi))
			// Ghi chú và nhóm chỉ điền khi hồ sơ đang trống — GhiNhanKhach
			// không đụng tới hai trường này.
			moi, _ := KhachTheoMa(cu.Ma)
			doi := false
			if moi.GhiChu == "" && lay(cot.ghiChu) != "" {
				moi.GhiChu = catBot(lay(cot.ghiChu), 500)
				doi = true
			}
			if nhom != "" && moi.Nhom == NhomKhachLe && nhom != NhomKhachLe {
				moi.Nhom = nhom
				doi = true
			}
			if doi {
				if _, err := LuuKhach(moi); err != nil {
					kq.BoQua = append(kq.BoQua, fmt.Sprintf("dòng %d (%s): %v", soDong, ten, err))
					continue
				}
			}
			kq.CapNhat++
			continue
		}
		if nhom == "" {
			nhom = NhomKhachLe
		}
		_, err := LuuKhach(KhachHang{
			Ten:    catBot(ten, 100),
			LienHe: catBot(lienHe, 60),
			Email:  catBot(lay(cot.email), 254),
			DiaChi: catBot(lay(cot.diaChi), 200),
			Nhom:   nhom,
			GhiChu: catBot(lay(cot.ghiChu), 500),
		})
		if err != nil {
			kq.BoQua = append(kq.BoQua, fmt.Sprintf("dòng %d (%s): %v", soDong, ten, err))
			continue
		}
		kq.ThemMoi++
	}
	return kq, nil
}

type cotKhach struct {
	coTieuDe                                 bool
	ten, lienHe, email, diaChi, nhom, ghiChu int
}

// doCotKhach đoán cột nào là gì từ dòng đầu. Không có tiêu đề nhận ra được
// thì mặc định cột 1 là tên, cột 2 là số — đúng thứ tự người ta xuất danh bạ
// ra, và cũng là thứ tự tệp chính trang này xuất ra.
func doCotKhach(dau []string) cotKhach {
	c := cotKhach{ten: 0, lienHe: 1, email: -1, diaChi: -1, nhom: -1, ghiChu: -1}
	nhanRa := false
	for i, o := range dau {
		s := boDau(strings.ToLower(strings.TrimSpace(o)))
		switch {
		case s == "ten" || s == "ho ten" || s == "ten khach" || s == "khach":
			c.ten, nhanRa = i, true
		case strings.Contains(s, "dien thoai") || s == "sdt" || s == "so" || s == "lien he" || s == "phone":
			c.lienHe, nhanRa = i, true
		case strings.Contains(s, "email") || strings.Contains(s, "mail"):
			c.email, nhanRa = i, true
		case strings.Contains(s, "dia chi"):
			c.diaChi, nhanRa = i, true
		case strings.Contains(s, "nhom"):
			c.nhom, nhanRa = i, true
		case strings.Contains(s, "ghi chu"):
			c.ghiChu, nhanRa = i, true
		}
	}
	c.coTieuDe = nhanRa
	return c
}

// chuanNhomKhach nhận cả mã lẫn tên tiếng Việt: tệp Kendy gõ tay sẽ ghi
// "khách quen" chứ không ghi "than".
func chuanNhomKhach(s string) string {
	s = boDau(strings.ToLower(strings.TrimSpace(s)))
	if s == "" {
		return ""
	}
	for _, n := range CacNhomKhach {
		if s == n.Ma || s == boDau(strings.ToLower(n.Ten)) {
			return n.Ma
		}
	}
	switch {
	case strings.Contains(s, "quen") || strings.Contains(s, "than"):
		return NhomKhachThan
	case strings.Contains(s, "clb") || strings.Contains(s, "doi") || strings.Contains(s, "nhom"):
		return NhomKhachClb
	case strings.Contains(s, "le"):
		return NhomKhachLe
	}
	return ""
}

func hQtKhachNhap(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(4 << 20); err != nil {
		http.Redirect(w, r, "/qt/khach?loi="+url.QueryEscape("Tệp quá lớn hoặc không đọc được."), http.StatusSeeOther)
		return
	}
	f, _, err := r.FormFile("tep")
	if err != nil {
		http.Redirect(w, r, "/qt/khach?loi=Chưa+chọn+tệp", http.StatusSeeOther)
		return
	}
	defer f.Close()
	kq, err := NhapKhachCSV(boBOM(f))
	if err != nil {
		http.Redirect(w, r, "/qt/khach?loi="+url.QueryEscape(err.Error()), http.StatusSeeOther)
		return
	}
	tin := fmt.Sprintf("Nhập xong: thêm mới %d, cập nhật %d, bỏ qua %d.", kq.ThemMoi, kq.CapNhat, len(kq.BoQua))
	if len(kq.BoQua) > 0 {
		tin += " — " + strings.Join(kq.BoQua, "; ")
	}
	http.Redirect(w, r, "/qt/khach?ok="+url.QueryEscape(tin), http.StatusSeeOther)
}

// boBOM bóc BOM UTF-8 ở đầu tệp. Excel bản Việt LUÔN ghi BOM khi lưu CSV, mà
// để nguyên thì tiêu đề cột đầu thành "<BOM>Tên" và không cột nào nhận ra.
func boBOM(r io.Reader) io.Reader {
	b, err := io.ReadAll(r)
	if err != nil {
		return strings.NewReader("")
	}
	return strings.NewReader(strings.TrimPrefix(string(b), "\ufeff"))
}
