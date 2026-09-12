package core

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Khu "Quản lý đơn": bốn trang cũ nay dùng chung một dải tab và một thanh
// chọn kỳ. Mẫu tab đọc .Ky, .GiuLai, .SoYeuCauMoi — trang nào quên truyền là
// 500 ngay, mà build vẫn xanh. Mấy test dưới dựng thật từng trang.

func TestKhuQuanLyDonDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	nay := time.Now().Format("2006-01-02")
	donThu(t, &Don{
		Ma: "TV-2609-901", Ngay: nay, TrangThai: TTDaGiao, KhachTen: "Anh Bảy",
		KhachLienHe: "0900000001", KhachEmail: "bay@vidu.vn", VotHang: "Joola",
		DongTien: []DongTien{{MaDichVu: "dan-day", Ten: "Đan dây", SoTien: 250000}},
		TongTien: 250000,
		LichSu:   []Moc{{Luc: nay + " 10:00", TrangThai: TTDaGiao, Nguoi: "kendy"}},
	})
	donThu(t, &Don{
		Ma: "TG-2609-901", Ngay: nay, Loai: LoaiGiay, TrangThai: TTMoi,
		KhachTen: "Chị Tám", KhachLienHe: "0900000002", GiayHang: "Asics",
	})

	for _, duong := range []string{
		"/qt/don-vot", "/qt/don-giay", "/qt/yeu-cau", "/qt/thong-ke",
		"/qt/don-vot?ky=ngay", "/qt/don-giay?ky=thang", "/qt/yeu-cau?ky=nam",
		"/qt/thong-ke?ky=nam", "/qt/thong-ke?ky=&loai=giay",
		"/qt/thong-ke?ky=ngay&moc=" + nay,
		// Mốc rác: DocKy phải chịu được chứ không được làm vỡ trang.
		"/qt/don-vot?ky=thang&moc=xyz", "/qt/thong-ke?ky=lung&tung=1",
	} {
		moTrang(t, mux, ck, duong)
	}

	// Cửa vào gộp chỉ là chuyển hướng — bốn URL cũ vẫn phải sống vì lối tắt
	// PWA và mail đã gửi còn trỏ vào đó.
	r := httptest.NewRequest("GET", "/qt/don-hang?ky=thang", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("/qt/don-hang trả %d, muốn 303", w.Code)
	}
	if d := w.Header().Get("Location"); !strings.HasPrefix(d, "/qt/don-vot?") || !strings.Contains(d, "ky=thang") {
		t.Errorf("/qt/don-hang đẩy sang %q, muốn /qt/don-vot mang theo kỳ", d)
	}
}

// Xuất CSV phải ra đúng bảng đang xem, và phải có BOM — thiếu BOM thì Excel
// mở ra tiếng Việt thành ký tự rác, coi như không xuất được.
func TestXuatCSVDonVaThongKe(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	nay := time.Now().Format("2006-01-02")
	donThu(t, &Don{
		Ma: "TV-2609-902", Ngay: nay, TrangThai: TTDaGiao, KhachTen: "Anh Chín",
		KhachLienHe: "0900000003", TongTien: 300000,
		DongTien: []DongTien{{MaDichVu: "dan-day", Ten: "Đan dây", SoTien: 300000}},
		LichSu:   []Moc{{Luc: nay + " 09:00", TrangThai: TTDaGiao, Nguoi: "kendy"}},
	})

	for _, duong := range []string{"/qt/don-vot.csv?ky=nam", "/qt/thong-ke.csv?ky=nam"} {
		r := httptest.NewRequest("GET", duong, nil)
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("%s trả %d", duong, w.Code)
		}
		than := w.Body.String()
		if !strings.HasPrefix(than, "\xef\xbb\xbf") {
			t.Errorf("%s thiếu BOM, Excel sẽ đọc ra ký tự rác", duong)
		}
		if !strings.Contains(w.Header().Get("Content-Disposition"), "attachment") {
			t.Errorf("%s không bảo trình duyệt tải về", duong)
		}
	}
}

// Ranh giới của thợ ở khu đơn: thấy tiền của TỪNG đơn (phải biết thu khách
// bao nhiêu lúc giao), nhưng không thấy tổng và không vào được thống kê.
// Tệp CSV phải chép đúng cái bảng thợ đang nhìn, không hơn.
func TestThoKhongThayTienOKhuDon(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	nay := time.Now().Format("2006-01-02")
	donThu(t, &Don{
		Ma: "TV-2609-903", Ngay: nay, TrangThai: TTDaGiao, KhachTen: "Anh Mười",
		KhachLienHe: "0900000004", TongTien: 777000,
		LichSu: []Moc{{Luc: nay + " 09:00", TrangThai: TTDaGiao, Nguoi: "kendy"}},
	})

	s := moTrang(t, mux, ck, "/qt/don-vot?ky=nam")
	if strings.Contains(s, "đơn đang hiện") {
		t.Error("thợ thấy hàng cộng tổng cuối bảng")
	}
	if strings.Contains(s, `href="/qt/thong-ke`) {
		t.Error("thợ thấy đường sang thống kê")
	}

	for _, duong := range []string{"/qt/thong-ke", "/qt/thong-ke.csv", "/qt/khach", "/qt/khach.csv"} {
		r := httptest.NewRequest("GET", duong, nil)
		r.AddCookie(ck)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		if w.Code == http.StatusOK {
			t.Errorf("%s mở được với tài khoản thợ", duong)
		}
	}

	r := httptest.NewRequest("GET", "/qt/don-vot.csv?ky=nam", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("thợ tải CSV đơn trả %d", w.Code)
	}
	csv := w.Body.String()
	if !strings.Contains(csv, "777000") {
		t.Error("CSV của thợ thiếu cột tiền mà bảng trên màn đang hiện")
	}
	if strings.Contains(csv, "Trả tiệm ngoài") {
		t.Error("CSV của thợ lộ giá vốn trả tiệm ngoài")
	}
}

// Hoá đơn: mở được từ trong admin lẫn từ link khách, và đơn chưa xong phải
// tự nói ra rằng nó mới là bảng tạm tính.
func TestHoaDonHaiCuaVao(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	nay := time.Now().Format("2006-01-02")
	xong := donThu(t, &Don{
		Ma: "TV-2609-904", Ngay: nay, TrangThai: TTDaGiao, KhachTen: "Anh Tư",
		KhachLienHe: "0900000005", Token: "tokenhoadon904xyz", TongTien: 250000,
		DongTien: []DongTien{{MaDichVu: "dan-day", Ten: "Đan dây Yonex", SoTien: 250000}},
		LichSu:   []Moc{{Luc: nay + " 10:00", TrangThai: TTDaGiao, Nguoi: "kendy"}},
	})
	dang := donThu(t, &Don{
		Ma: "TV-2609-905", Ngay: nay, TrangThai: TTDangSua, KhachTen: "Anh Năm",
		KhachLienHe: "0900000006", TongTien: 180000,
		DongTien: []DongTien{{MaDichVu: "dan-day", Ten: "Đan dây", SoTien: 180000}},
	})

	s := moTrang(t, mux, ck, "/qt/don/"+xong.Ma+"/hoa-don")
	if !strings.Contains(s, "Đan dây Yonex") || !strings.Contains(s, "250.000") {
		t.Error("hoá đơn thiếu dòng tiền")
	}
	if strings.Contains(s, "Phiếu tạm tính") {
		t.Error("đơn đã giao mà vẫn in là tạm tính")
	}

	s = moTrang(t, mux, ck, "/qt/don/"+dang.Ma+"/hoa-don")
	if !strings.Contains(s, "Phiếu tạm tính") {
		t.Error("đơn chưa xong mà in ra như hoá đơn thật")
	}

	// Cửa của khách: không cần đăng nhập, chỉ cần token.
	r := httptest.NewRequest("GET", "/hoa-don/"+xong.Token, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("/hoa-don/{token} trả %d", w.Code)
	}
	if strings.Contains(w.Body.String(), "Về đơn") {
		t.Error("tờ hoá đơn của khách có link ngược vào admin")
	}

	r = httptest.NewRequest("GET", "/hoa-don/khongphaitoken", nil)
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("token sai trả %d, muốn 404", w.Code)
	}
}

// Trang soạn thư: dựng được cả ba mẫu, và đơn không có email thì nói rõ vì
// sao chưa gửi được thay vì chìa ra một cái nút bấm vào là lỗi.
func TestTrangThuGuiKhach(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	nay := time.Now().Format("2006-01-02")
	d := donThu(t, &Don{
		Ma: "TV-2609-906", Ngay: nay, TrangThai: TTDangSua, KhachTen: "Anh Sáu",
		KhachLienHe: "0900000007", TongTien: 200000,
		DongTien: []DongTien{{MaDichVu: "dan-day", Ten: "Đan dây", SoTien: 200000}},
	})

	for _, mau := range []string{MailNhanDon, MailBaoGia, MailXong} {
		s := moTrang(t, mux, ck, "/qt/don/"+d.Ma+"/mail?mau="+mau)
		if !strings.Contains(s, d.Ma) {
			t.Errorf("thư mẫu %s không nhắc mã đơn", mau)
		}
		if !strings.Contains(s, "chưa có email") {
			t.Errorf("thư mẫu %s: đơn không có email mà trang không nói gì", mau)
		}
	}

	// Mẫu bịa ra thì quay về mẫu đầu, không phải 500.
	moTrang(t, mux, ck, "/qt/don/"+d.Ma+"/mail?mau=linhtinh")
}

// Duyệt yêu cầu web: nút nào ra đơn nấy, và email khách phải theo sang đơn —
// thiếu nó thì cả ba lá thư đều không gửi được.
func TestYeuCauWebTaoDungLoaiDon(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)
	nay := time.Now().Format("2006-01-02")
	if err := luuYeuCau(yeuCauKhach{
		Ma: "YC-909", Ngay: nay + " 08:00", Ten: "Chị Chín",
		LienHe: "0900000008", Email: "chin@vidu.vn", VotHang: "Selkirk",
		MoTa: "Vợt nứt mép",
	}); err != nil {
		t.Fatal(err)
	}

	s := moTrang(t, mux, ck, "/qt/yeu-cau")
	if !strings.Contains(s, `value="vot"`) || !strings.Contains(s, `value="giay"`) {
		t.Fatal("trang yêu cầu thiếu hai nút chọn loại đơn")
	}

	form := strings.NewReader("_csrf=x&loai=giay")
	r := httptest.NewRequest("POST", "/qt/yeu-cau/YC-909/tao-don", form)
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.AddCookie(ck)
	r.AddCookie(&http.Cookie{Name: tenCookieCSRF, Value: "x"})
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusSeeOther {
		t.Fatalf("tạo đơn từ yêu cầu trả %d, thân: %s", w.Code, catBot(w.Body.String(), 300))
	}

	var moi *Don
	for _, d := range LocDon(BoLoc{}) {
		if d.KhachLienHe == "0900000008" {
			moi = d
		}
	}
	if moi == nil {
		t.Fatal("bấm nhận khám mà không ra đơn nào")
	}
	if !moi.LaGiay() {
		t.Errorf("bấm nút giày mà ra đơn %s", moi.TenLoai())
	}
	if moi.TrangThai != TTKhamAnh {
		t.Errorf("đơn mở ra ở trạng thái %s, muốn %s", moi.TrangThai, TTKhamAnh)
	}
	if moi.KhachEmail != "chin@vidu.vn" {
		t.Errorf("email khách không theo sang đơn, nhận %q", moi.KhachEmail)
	}
	if moi.NguonDon != NguonWeb {
		t.Errorf("nguồn đơn = %q, muốn %q", moi.NguonDon, NguonWeb)
	}
}
