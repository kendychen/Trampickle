// Cửa hàng: khách đặt sửa từ xa, không cần tới trạm.
//
// VÌ SAO ĐẺ RA ĐƠN THẬT NGAY, KHÔNG PHẢI MỘT "YÊU CẦU" NẰM CHỜ. Form gửi ảnh
// (hGuiYeuCau, server.go) tạo yeuCauKhach rồi để chủ tự tay dựng đơn — hợp lý
// khi khách đứng trước mặt. Nhưng khách ở xa cần MỘT MÃ để ghi lên kiện hàng
// trước khi ra bưu điện. Không có mã thì kiện tới nơi không ai biết của ai, và
// khách không có gì để tra.
//
// ĐƠN Ở ĐÂY CHƯA CÓ TIỀN. TongTien để 0, DongTien rỗng. Giá chốt sinh ở bước
// báo giá sau khi thợ khám — chỉ bảng giá mới được đẻ ra con số cam kết với
// khách. Dịch vụ khách chọn lúc đặt chỉ là NGUYỆN VỌNG, ghi vào TinhTrang cho
// thợ đọc.
package core

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CuaHangMo: cửa hàng chỉ mở khi site chạy bản Online VÀ đã có địa chỉ trạm.
// Thiếu địa chỉ mà vẫn nhận đơn là để khách đặt xong mới phát hiện không biết
// gửi đi đâu — hỏng nặng hơn nhiều so với đóng cửa một hôm.
func CuaHangMo() bool {
	return LaOnline() && strings.TrimSpace(LienHeHienTai().DiaChi) != ""
}

// OMonCuaHang — một dòng trong bảng chọn của khách. Gia là CHUỖI đã dựng sẵn, để
// template không có đường nào in ra một con số trần.
type OMonCuaHang struct {
	Ma       string
	Ten      string
	Gia      string
	DieuKien string
	LeadNgay int
}

// DichVuChoCuaHang lọc bảng giá theo món. Nhóm C là việc gia công ngoài (thay
// đế giày); A và B là việc trên vợt.
func DichVuChoCuaHang(loai string) []OMonCuaHang {
	giay := LoaiHopLe(loai) == LoaiGiay
	out := []OMonCuaHang{}
	for _, d := range DichVuDangBan(GiaiDoan) {
		if (d.Nhom == "C") != giay {
			continue
		}
		o := OMonCuaHang{Ma: d.Ma, Ten: d.Ten, DieuKien: d.DieuKien, LeadNgay: d.LeadTimeNgay}
		p := d.GiaTheoGiaiDoan(GiaiDoan)
		if d.BaoGiaRieng || p == nil {
			// Không có giá niêm yết. Nói thẳng, đừng bịa một con số.
			o.Gia = ND("cuahang.buoc.bao-gia-rieng")
		} else {
			o.Gia = ND("cuahang.buoc.gia-tu") + " " + dinhDangTien(*p)
		}
		out = append(out, o)
	}
	return out
}

// LyDoTuChoiCuaHang — chuỗi rỗng nghĩa là nhận. Chặn ở đây tiết kiệm cho khách
// một chiều ship, và cho trạm khoản ship-ca-từ-chối (nguong.ship_ca_tu_choi_dong
// trong bang-gia.yaml).
//
// CHỈ TỪ CHỐI KHI MÁY BIẾT CHẮC. Hãng lạ, đời lạ, mô tả mơ hồ đều KHÔNG phải
// lý do — thợ khám rồi mới biết, và từ chối nhầm là mất một khách thật. Vì thế
// ở đây không tra nhan_sua của dòng vợt: trường ấy là lời khuyên viết tay cho
// người đọc, không phải cờ đúng/sai cho máy quyết.
func LyDoTuChoiCuaHang(loai, kieuGiay string, giaTri int) string {
	if LoaiHopLe(loai) == LoaiGiay && kieuGiay != GiayChayBo && kieuGiay != GiayDiLai {
		return ND("cuahang.tuchoi.giay-kieu")
	}
	// giaTri == 0 nghĩa là khách không khai, không phải "món vô giá trị".
	if ng := NguongHienTai().NguongMoKenhShipDong; ng > 0 && giaTri > 0 && giaTri < ng {
		return ND("cuahang.tuchoi.duoi-nguong") + " " + dinhDangTien(ng) + "."
	}
	return ""
}

func timDichVuCuaHang(loai, ma string) (OMonCuaHang, bool) {
	for _, o := range DichVuChoCuaHang(loai) {
		if o.Ma == ma {
			return o, true
		}
	}
	return OMonCuaHang{}, false
}

type dlCuaHang struct {
	Chung

	Mo     bool
	DiaChi string
	GioLam string

	Loai   string
	DichVu []OMonCuaHang
	Loi    string

	// Sau khi đặt xong
	Don     *Don
	LinkTra string
}

func hCuaHang(w http.ResponseWriter, r *http.Request) {
	d := dlCuaHang{Chung: chung(r, "cua-hang"), Mo: CuaHangMo()}
	if !d.Mo {
		render(w, "cua-hang.html", d)
		return
	}
	lh := LienHeHienTai()
	d.DiaChi, d.GioLam = lh.DiaChi, lh.GioLamVic
	d.Loai = LoaiHopLe(r.URL.Query().Get("loai"))
	d.DichVu = DichVuChoCuaHang(d.Loai)
	render(w, "cua-hang.html", d)
}

func hCuaHangGui(w http.ResponseWriter, r *http.Request) {
	if !CuaHangMo() {
		render(w, "cua-hang.html", dlCuaHang{Chung: chung(r, "cua-hang"), Mo: false})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32<<20)
	// 8MB giữ trong RAM, phần còn lại multipart tự ghi ra tệp tạm rồi dọn khi
	// request đóng.
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		cuaHangLoi(w, r, "Ảnh nặng quá ạ. Anh/chị gửi ít tệp hơn giúp em.")
		return
	}
	defer r.MultipartForm.RemoveAll()

	// Biểu mẫu có tệp: lớp bọc không đọc được thân yêu cầu nên không kiểm CSRF
	// hộ được. Xem core/csrf.go.
	if !KiemCSRFMultipart(w, r) {
		return
	}

	if !rl.choPhep("cuahang:"+ipCua(r), 5, time.Hour) {
		cuaHangLoi(w, r, "Đặt hơi nhiều lần rồi ạ. Anh/chị nhắn trực tiếp giúp em.")
		return
	}

	loai := LoaiHopLe(r.FormValue("loai"))
	ten := catBot(r.FormValue("ten"), 100)
	lienHe := catBot(r.FormValue("lien_he"), 100)
	diaChi := catBot(r.FormValue("dia_chi"), 300)
	tinhTrang := catBot(r.FormValue("tinh_trang"), 1000)

	if ten == "" || lienHe == "" || diaChi == "" {
		cuaHangLoi(w, r, "Cần đủ tên, số liên lạc và địa chỉ nhận lại thì trạm mới gửi đồ về được ạ.")
		return
	}

	giaTri := 0
	if n, err := strconv.Atoi(strings.TrimSpace(r.FormValue("gia_tri"))); err == nil && n > 0 {
		giaTri = n
	}
	kieuGiay := strings.TrimSpace(r.FormValue("giay_kieu"))

	// Chặn TRƯỚC khi tạo đơn: khách phải biết trạm không nhận từ lúc còn ngồi
	// trước màn hình, chứ không phải sau khi đã ra bưu điện.
	if ly := LyDoTuChoiCuaHang(loai, kieuGiay, giaTri); ly != "" {
		cuaHangLoi(w, r, ly)
		return
	}

	// Nguyện vọng của khách, không phải chẩn đoán của thợ. Ghi vào phần khách
	// mô tả để thợ đọc, không đẻ ra dòng tiền nào.
	if ma := strings.TrimSpace(r.FormValue("dich_vu")); ma != "" {
		if dv, co := timDichVuCuaHang(loai, ma); co {
			tinhTrang = strings.TrimSpace("Khách chọn: " + dv.Ten + "\n" + tinhTrang)
		}
	}

	d := &Don{
		Ma:          MaDonMoi(loai),
		Token:       maNgauNhien(8),
		Ngay:        time.Now().Format("2006-01-02 15:04:05"),
		Loai:        loai,
		TrangThai:   TTChoHangVe,
		KhachTen:    ten,
		KhachLienHe: lienHe,
		KhachDiaChi: diaChi,
		TinhTrang:   tinhTrang,
		VotGiaTri:   giaTri,
		KenhNhan:    "ship",
		NguonDon:    NguonWeb,
	}
	if d.LaGiay() {
		d.GiayHang = catBot(r.FormValue("giay_hang"), 100)
		d.GiaySize = catBot(r.FormValue("giay_size"), 20)
		d.GiayKieu = kieuGiay
	} else {
		d.VotHang = catBot(r.FormValue("vot_hang"), 100)
	}
	d.LichSu = append(d.LichSu, Moc{
		Luc:       d.Ngay,
		TrangThai: TTChoHangVe,
		Nguoi:     "khách",
		GhiChu:    "Đặt trên web",
	})
	// Khách đặt ship cũng là khách — hồ sơ dựng ngay, không đợi thợ gõ lại.
	d.MaKhach = GhiNhanKhach(d.KhachTen, d.KhachLienHe, d.KhachEmail, d.KhachDiaChi)

	if err := LuuDon(d); err != nil {
		cuaHangLoi(w, r, "Trạm chưa lưu được đơn. Anh/chị nhắn Zalo giúp em.")
		return
	}

	// Ảnh lưu SAU khi đơn đã nằm an toàn trên đĩa: ảnh hỏng không được làm
	// mất đơn.
	if r.MultipartForm != nil && luuAnhVaoDon(d, r.MultipartForm.File["anh"], "truoc") > 0 {
		LuuDon(d)
	}

	// Báo cho Kendy. Gửi nền, lỗi nuốt vào nhật ký: Telegram chết không được
	// làm khách mất trang "đã nhận đơn".
	BaoDonWebMoi(d)

	lh := LienHeHienTai()
	render(w, "cua-hang-xong.html", dlCuaHang{
		Chung:   chung(r, "cua-hang"),
		Mo:      true,
		DiaChi:  lh.DiaChi,
		GioLam:  lh.GioLamVic,
		Don:     d,
		LinkTra: "/tra-cuu/" + d.Token,
	})
}

func cuaHangLoi(w http.ResponseWriter, r *http.Request, msg string) {
	lh := LienHeHienTai()
	loai := LoaiHopLe(r.FormValue("loai"))
	render(w, "cua-hang.html", dlCuaHang{
		Chung:  chung(r, "cua-hang"),
		Mo:     CuaHangMo(),
		DiaChi: lh.DiaChi,
		GioLam: lh.GioLamVic,
		Loai:   loai,
		DichVu: DichVuChoCuaHang(loai),
		Loi:    msg,
	})
}

// --- Thanh toán sau khi sửa xong -------------------------------------

// khoiTra là khối "trả tiền" trên trang tra cứu. Nhúng vào struct dữ liệu
// trang chứ không rải bảy trường rời, để nhánh token và nhánh POST của
// hTraCuu dựng cùng một cách.
type khoiTra struct {
	CanTra    bool
	SoTienTra int
	NganHang  NganHang
	CoQR      bool
	NoiDungCK string
	ChuoiQR   string
	FreeShip  bool
}

// dungKhoiTra dựng khối thanh toán cho một đơn. Đơn nil, chưa xong, hoặc đã
// thu đủ thì trả khối rỗng — trang không hiện gì.
//
// "Đã trả tiền chưa" hỏi SỔ TIỀN, không hỏi một trường trên đơn. Xem chú
// thích trên Don.DaThu.
func dungKhoiTra(don *Don) khoiTra {
	var k khoiTra
	if don == nil || don.TrangThai != TTXong || don.ConNo() <= 0 {
		return k
	}
	k.CanTra = true
	k.SoTienTra = don.ConNo()
	k.NoiDungCK = NoiDungCK(don.Ma)
	k.NganHang, k.CoQR = NganHangNhan()
	if k.CoQR {
		k.ChuoiQR = ChuoiVietQR(k.NganHang, k.SoTienTra, k.NoiDungCK)
	}
	if ng := NguongHienTai().FreeShipVeTuDong; ng > 0 && don.TongTien >= ng {
		k.FreeShip = true
	}
	return k
}

// hChonHinhThucTra — khách bấm "tôi sẽ chuyển khoản" hay "tôi trả khi nhận".
// Chỉ ghi Ý ĐỊNH, không ghi tiền: tiền vào sổ khi nó thật sự về, không phải
// khi khách bấm nút.
func hChonHinhThucTra(w http.ResponseWriter, r *http.Request) {
	tok := r.PathValue("token")
	don, ok := LayDonTheoToken(tok)
	if !ok {
		http.NotFound(w, r)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	r.ParseForm()
	switch r.FormValue("cach") {
	case TraQR:
		don.HinhThucTra = TraQR
	case TraCOD:
		don.HinhThucTra = TraCOD
	default:
		http.Redirect(w, r, "/tra-cuu/"+tok, http.StatusSeeOther)
		return
	}
	don.LichSu = append(don.LichSu, Moc{
		Luc:       time.Now().Format("2006-01-02 15:04:05"),
		TrangThai: don.TrangThai,
		Nguoi:     "khách",
		GhiChu:    "Chọn trả bằng " + don.HinhThucTra,
	})
	LuuDon(don)
	http.Redirect(w, r, "/tra-cuu/"+tok, http.StatusSeeOther)
}

// hKhachBaoVanDon — khách gửi hàng xong, dán mã vận đơn vào đơn. Chỉ nhận khi
// đơn còn đang chờ hàng về: kiện đã tới tay trạm rồi thì mã ấy không còn nói
// thêm được gì, mà vẫn là một ô cho người lạ ghi đè.
func hKhachBaoVanDon(w http.ResponseWriter, r *http.Request) {
	tok := r.PathValue("token")
	don, ok := LayDonTheoToken(tok)
	if !ok {
		http.NotFound(w, r)
		return
	}
	if !don.ChoHangVe() {
		http.Redirect(w, r, "/tra-cuu/"+tok, http.StatusSeeOther)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024)
	r.ParseForm()
	ma := catBot(strings.TrimSpace(r.FormValue("ma_van_don")), 60)
	if ma == "" {
		http.Redirect(w, r, "/tra-cuu/"+tok, http.StatusSeeOther)
		return
	}
	don.MaVanDonDen = ma
	don.LichSu = append(don.LichSu, Moc{
		Luc:       time.Now().Format("2006-01-02 15:04:05"),
		TrangThai: don.TrangThai,
		Nguoi:     "khách",
		GhiChu:    "Khách báo đã gửi, vận đơn " + ma,
	})
	LuuDon(don)
	http.Redirect(w, r, "/tra-cuu/"+tok, http.StatusSeeOther)
}
