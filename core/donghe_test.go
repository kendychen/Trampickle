package core

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// dungDoNghe dựng một trạm rỗng có sẵn kho, sổ tiền và danh sách đồ nghề sạch.
// doNgheDs là biến toàn cục nên phải trả lại nguyên trạng, nếu không test chạy
// sau nhặt phải món của test chạy trước.
func dungDoNghe(t *testing.T) {
	t.Helper()
	dungKhoTienThu(t)
	doNgheMu.Lock()
	cu := doNgheDs
	doNgheDs = nil
	doNgheMu.Unlock()
	t.Cleanup(func() {
		doNgheMu.Lock()
		doNgheDs = cu
		doNgheMu.Unlock()
	})
	if err := NapDoNghe(); err != nil {
		t.Fatal(err)
	}
}

func themDoNghe(t *testing.T, d DoNghe) DoNghe {
	t.Helper()
	ra, err := LuuDoNghe(d)
	if err != nil {
		t.Fatal(err)
	}
	return ra
}

func TestDoNgheDaMuaSinhKhoanChoDuyet(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Cân điện tử", Nghe: NgheChung, Nhom: "do-chan-doan", Dot: 1, GiaDuKien: 250000})

	if _, err := DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "2026-09-12", 240000, "Shopee", "Kendy"); err != nil {
		t.Fatal(err)
	}
	sau, _ := TimDoNghe(d.Ma)
	if sau.MaKhoan == "" {
		t.Fatal("đã mua mà không sinh khoản chi")
	}
	k, co := LayKhoan(sau.MaKhoan)
	if !co {
		t.Fatalf("khoản %s không có trong sổ", sau.MaKhoan)
	}
	if k.Nhom != NhomDauTu || k.Loai != KhoanChi {
		t.Fatalf("khoản phải là chi nhóm đầu tư, đang là %s/%s", k.Loai, k.Nhom)
	}
	if !k.ChoDuyet {
		t.Fatal("khoản phải chờ duyệt, không được tự vào báo cáo")
	}
	if k.SoTien != 240000 {
		t.Fatalf("số tiền phải theo giá thực 240000, đang là %d", k.SoTien)
	}
	if k.MaDoNghe != d.Ma {
		t.Fatalf("khoản không trỏ ngược về món: %q", k.MaDoNghe)
	}
	// Chờ duyệt thì chưa được tính vào tiền đã chi.
	tq := TongQuanDoNgheHienGio()
	if tq.DaChi != 0 || tq.ChoDuyet != 240000 {
		t.Fatalf("chờ duyệt đã lọt vào đã chi: đã chi %d, chờ duyệt %d", tq.DaChi, tq.ChoDuyet)
	}
	if err := DuyetKhoan(k.Ma); err != nil {
		t.Fatal(err)
	}
	if tq := TongQuanDoNgheHienGio(); tq.DaChi != 240000 || tq.ChoDuyet != 0 {
		t.Fatalf("duyệt rồi mà chưa vào đã chi: đã chi %d, chờ duyệt %d", tq.DaChi, tq.ChoDuyet)
	}
}

// Ca hỏng thật: mạng chậm, Kendy bấm lại (hoặc F5 gửi lại form). Không có khoá
// thì sổ có hai khoản 240k cho một cái cân, và con số đầu tư sai gấp đôi.
func TestDoNgheBamHaiLanKhongDeHaiKhoan(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Máy khoan mini", Nghe: NgheVot, Nhom: "may-thiet-bi", Dot: 2, GiaDuKien: 700000})

	for i := 0; i < 3; i++ {
		if _, err := DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "2026-09-12", 690000, "Chợ Tốt", "Kendy"); err != nil {
			t.Fatal(err)
		}
	}
	if n := len(khoanDoNghe()); n != 1 {
		t.Fatalf("bấm 3 lần phải còn 1 khoản, đang có %d", n)
	}
	if tq := TongQuanDoNgheHienGio(); tq.ChoDuyet != 690000 {
		t.Fatalf("tiền chờ duyệt phải là 690000, đang là %d", tq.ChoDuyet)
	}
}

// Khoản còn chờ duyệt thì sửa giá được — gõ nhầm số là chuyện thường.
func TestDoNgheSuaGiaKhiKhoanConChoDuyet(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Quạt thông gió", Nghe: NgheChung, Nhom: "bao-ho", Dot: 1, GiaDuKien: 400000})

	DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "2026-09-12", 400000, "", "Kendy")
	if _, err := DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "2026-09-12", 430000, "Điện máy", "Kendy"); err != nil {
		t.Fatal(err)
	}
	sau, _ := TimDoNghe(d.Ma)
	k, _ := LayKhoan(sau.MaKhoan)
	if k.SoTien != 430000 {
		t.Fatalf("khoản chờ duyệt phải sửa theo giá mới, đang là %d", k.SoTien)
	}
	if len(khoanDoNghe()) != 1 {
		t.Fatal("sửa giá không được đẻ thêm khoản")
	}
}

// Đã duyệt rồi thì khoản là số đã vào báo cáo — không sửa sau lưng sổ tiền.
func TestDoNgheKhongSuaKhoanDaDuyet(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Máy hút ẩm", Nghe: NgheVot, Nhom: "may-thiet-bi", Dot: 2, GiaDuKien: 3000000})

	DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "2026-09-12", 2900000, "", "Kendy")
	sau, _ := TimDoNghe(d.Ma)
	if err := DuyetKhoan(sau.MaKhoan); err != nil {
		t.Fatal(err)
	}
	nhac, err := DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "2026-09-12", 3500000, "", "Kendy")
	if err != nil {
		t.Fatal(err)
	}
	k, _ := LayKhoan(sau.MaKhoan)
	if k.SoTien != 2900000 {
		t.Fatalf("khoản đã duyệt bị sửa: %d", k.SoTien)
	}
	if nhac == "" {
		t.Fatal("phải nhắc rằng khoản đã duyệt nên giữ nguyên số cũ")
	}
}

// Bỏ đánh dấu đã mua KHÔNG xoá tiền đã ghi. Tiền đã tiêu là đã tiêu; trả hàng
// là một khoản thu mới chứ không phải xoá lịch sử.
func TestDoNgheBoDanhDauKhongXoaKhoan(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Đèn hồng ngoại", Nghe: NgheVot, Nhom: "may-thiet-bi", Dot: 2, GiaDuKien: 500000})

	DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "2026-09-12", 500000, "", "Kendy")
	nhac, err := DoiTrangThaiDoNghe(d.Ma, DoNgheChuaMua, "", 0, "", "Kendy")
	if err != nil {
		t.Fatal(err)
	}
	if len(khoanDoNghe()) != 1 {
		t.Fatal("khoản chi biến mất khi bỏ đánh dấu")
	}
	if nhac == "" {
		t.Fatal("phải nhắc rằng khoản chi vẫn còn trong sổ")
	}
}

// "Đã có sẵn" là món không tốn tiền. Sinh khoản 0đ ở đây là rác trong sổ.
func TestDoNgheDaCoKhongSinhKhoan(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Kéo và tua vít", Nghe: NgheChung, Nhom: "thao-lam-sach", Dot: 1, GiaDuKien: 100000})

	if _, err := DoiTrangThaiDoNghe(d.Ma, DoNgheDaCo, "", 0, "", "Kendy"); err != nil {
		t.Fatal(err)
	}
	if n := len(khoanDoNghe()); n != 0 {
		t.Fatalf("đã có sẵn mà vẫn sinh %d khoản", n)
	}
	tq := TongQuanDoNgheHienGio()
	if tq.SoDaSam != 1 || tq.ConPhaiSam != 0 {
		t.Fatalf("đã có sẵn phải hết nằm trong 'còn phải sắm': đã sắm %d, còn phải sắm %d", tq.SoDaSam, tq.ConPhaiSam)
	}
}

// Không giá dự kiến, không gõ giá thực thì không đoán bừa một con số.
func TestDoNgheDaMuaThieuTienThiBaoLoi(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Món không rõ giá", Nghe: NgheVot, Nhom: "thao-lam-sach", Dot: 1})

	if _, err := DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "", 0, "", "Kendy"); err == nil {
		t.Fatal("thiếu tiền mà vẫn ghi được")
	}
	if len(khoanDoNghe()) != 0 {
		t.Fatal("lỗi rồi mà vẫn đẻ khoản")
	}
	if sau, _ := TimDoNghe(d.Ma); sau.TrangThai != DoNgheChuaMua {
		t.Fatalf("lỗi rồi mà trạng thái vẫn đổi: %s", sau.TrangThai)
	}
}

// Tạm hoãn không nằm trong "còn phải sắm" — đã quyết định là chưa cần. Toàn bộ
// đồ nghề nghề giày đang ở trạng thái này cho tới khi chốt nhánh A/B/C.
func TestDoNgheTamHoanKhongTinhVaoConPhaiSam(t *testing.T) {
	dungDoNghe(t)
	themDoNghe(t, DoNghe{Ten: "Máy khâu trụ da", Nghe: NgheGiay, Nhom: "khau-de", Dot: 2, GiaDuKien: 15000000})
	themDoNghe(t, DoNghe{Ten: "Cân điện tử", Nghe: NgheChung, Nhom: "do-chan-doan", Dot: 1, GiaDuKien: 250000})

	may, _ := TimDoNghe("may-khau-tru-da")
	if _, err := DoiTrangThaiDoNghe(may.Ma, DoNgheTamHoan, "", 0, "", "Kendy"); err != nil {
		t.Fatal(err)
	}
	tq := TongQuanDoNgheHienGio()
	if tq.ConPhaiSam != 250000 {
		t.Fatalf("tạm hoãn lọt vào còn phải sắm: %d", tq.ConPhaiSam)
	}
	if tq.SoConThieu != 1 {
		t.Fatalf("số món còn thiếu phải là 1, đang là %d", tq.SoConThieu)
	}
}

// Món đã có khoản chi mà xoá dòng đi thì khoản trong sổ mồ côi: nhìn "Sắm đồ
// nghề: ..." mà không tra ngược được là món gì.
func TestDoNgheKhongXoaDuocMonDaCoKhoan(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Kính lúp", Nghe: NgheChung, Nhom: "do-chan-doan", Dot: 1, GiaDuKien: 80000})
	if err := XoaDoNghe(d.Ma); err != nil {
		t.Fatalf("món chưa mua phải xoá được: %v", err)
	}

	d2 := themDoNghe(t, DoNghe{Ten: "Kẹp chữ C", Nghe: NgheVot, Nhom: "dan-ep", Dot: 1, GiaDuKien: 150000})
	DoiTrangThaiDoNghe(d2.Ma, DoNgheDaMua, "2026-09-12", 150000, "", "Kendy")
	if err := XoaDoNghe(d2.Ma); err == nil {
		t.Fatal("món đã có khoản chi mà vẫn xoá được")
	}
}

// Sửa khai báo (tên, giá dự kiến) không được xoá mất lịch sử mua.
func TestLuuDoNgheGiuNguyenTrangThaiMua(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Đèn bàn", Nghe: NgheChung, Nhom: "do-chan-doan", Dot: 1, GiaDuKien: 300000})
	DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "2026-09-12", 280000, "Tiệm điện", "Kendy")

	d.Ten = "Đèn bàn chiếu nghiêng"
	d.GiaDuKien = 320000
	if _, err := LuuDoNghe(d); err != nil {
		t.Fatal(err)
	}
	sau, _ := TimDoNghe(d.Ma)
	if sau.Ten != "Đèn bàn chiếu nghiêng" || sau.GiaDuKien != 320000 {
		t.Fatalf("không sửa được khai báo: %+v", sau)
	}
	if sau.TrangThai != DoNgheDaMua || sau.GiaThuc != 280000 || sau.MaKhoan == "" {
		t.Fatalf("sửa khai báo làm mất lịch sử mua: %+v", sau)
	}
}

// Mã là thứ khoản chi trong sổ trỏ vào: sinh một lần rồi giữ đời đời, và hai
// món trùng tên không được cùng mã.
func TestMaDoNgheKhongTrung(t *testing.T) {
	dungDoNghe(t)
	a := themDoNghe(t, DoNghe{Ten: "Kẹp chữ C", Nghe: NgheVot, Nhom: "dan-ep", Dot: 1})
	b := themDoNghe(t, DoNghe{Ten: "Kẹp chữ C", Nghe: NgheGiay, Nhom: "dan-ep", Dot: 2})
	if a.Ma == b.Ma {
		t.Fatalf("hai món cùng mã %q", a.Ma)
	}
	if a.Ma != "kep-chu-c" {
		t.Fatalf("mã phải sinh từ tên, đang là %q", a.Ma)
	}
}

func TestNapDoNgheDocLaiDuocSauKhiLuu(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Máy hút bụi mini", Nghe: NgheChung, Nhom: "ban-cho-lam", Dot: 3, GiaDuKien: 600000})
	DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "2026-09-12", 590000, "Điện máy", "Kendy")

	if _, err := os.Stat(fileDoNghe()); err != nil {
		t.Fatalf("chưa ghi ra file: %v", err)
	}
	doNgheMu.Lock()
	doNgheDs = nil
	doNgheMu.Unlock()
	if err := NapDoNghe(); err != nil {
		t.Fatal(err)
	}
	sau, co := TimDoNghe(d.Ma)
	if !co || sau.GiaThuc != 590000 || sau.NguonMua != "Điện máy" || sau.TrangThai != DoNgheDaMua {
		t.Fatalf("nạp lại mất dữ liệu: %+v", sau)
	}
}

// Lọc theo một nghề vẫn phải thấy món dùng chung — thiếu cái cân thì cả hai
// nghề đều đứng, giấu nó đi là nhìn bảng tưởng đủ đồ.
func TestBangDoNgheLocTheoNgheVanHienMonChung(t *testing.T) {
	dungDoNghe(t)
	themDoNghe(t, DoNghe{Ten: "Cân điện tử", Nghe: NgheChung, Nhom: "do-chan-doan", Dot: 1})
	themDoNghe(t, DoNghe{Ten: "Dao mổ", Nghe: NgheVot, Nhom: "thao-lam-sach", Dot: 1})
	themDoNghe(t, DoNghe{Ten: "Gôm tẩy đế", Nghe: NgheGiay, Nhom: "thao-lam-sach", Dot: 1})

	dem := func(b []NhomBangDoNghe) int {
		n := 0
		for _, nh := range b {
			n += len(nh.Mon)
		}
		return n
	}
	if n := dem(BangDoNghe(NgheVot, "")); n != 2 {
		t.Fatalf("lọc nghề vợt phải ra 2 món (vợt + chung), đang ra %d", n)
	}
	if n := dem(BangDoNghe(NgheGiay, "")); n != 2 {
		t.Fatalf("lọc nghề giày phải ra 2 món, đang ra %d", n)
	}
	if n := dem(BangDoNghe("", "")); n != 3 {
		t.Fatalf("không lọc phải ra 3 món, đang ra %d", n)
	}
}

// Ngày mua bỏ trống thì lấy hôm nay, không để khoản rơi vào tháng vô định.
func TestDoNgheThieuNgayThiLayHomNay(t *testing.T) {
	dungDoNghe(t)
	d := themDoNghe(t, DoNghe{Ten: "Nhíp đầu nhọn", Nghe: NgheVot, Nhom: "thao-lam-sach", Dot: 1, GiaDuKien: 60000})
	if _, err := DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "", 0, "", "Kendy"); err != nil {
		t.Fatal(err)
	}
	sau, _ := TimDoNghe(d.Ma)
	if sau.NgayMua != time.Now().Format("2006-01-02") {
		t.Fatalf("ngày mua phải là hôm nay, đang là %q", sau.NgayMua)
	}
	if sau.GiaThuc != 60000 {
		t.Fatalf("bỏ trống giá thực thì lấy giá dự kiến, đang là %d", sau.GiaThuc)
	}
}

// Template gọi sai tên trường thì build vẫn xanh, tới lúc bấm vào mới thấy
// trang lỗi. Dựng trang thật qua router để bắt ngay ở đây.
func TestTrangDoNgheDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroChu)

	s := moTrang(t, mux, ck, "/qt/do-nghe")
	if !strings.Contains(s, "Chưa khai món nào") {
		t.Error("danh sách rỗng mà không nói phải bắt đầu từ đâu")
	}

	d := themDoNghe(t, DoNghe{Ten: "Cân điện tử", Nghe: NgheChung, Nhom: "do-chan-doan", Dot: 1, BatBuoc: true, GiaDuKien: 250000})
	themDoNghe(t, DoNghe{Ten: "Máy khâu trụ da", Nghe: NgheGiay, Nhom: "khau-de", Dot: 2, GiaDuKien: 15000000})
	DoiTrangThaiDoNghe(d.Ma, DoNgheDaMua, "2026-09-12", 240000, "Shopee", "Kendy")

	s = moTrang(t, mux, ck, "/qt/do-nghe")
	for _, muon := range []string{"Cân điện tử", "Máy khâu trụ da", "240.000", "Shopee", "chờ duyệt"} {
		if !strings.Contains(s, muon) {
			t.Errorf("trang thiếu %q", muon)
		}
	}
	// Lọc: chọn nghề giày thì cái cân dùng chung vẫn phải còn.
	s = moTrang(t, mux, ck, "/qt/do-nghe?nghe=giay")
	if !strings.Contains(s, "Cân điện tử") || !strings.Contains(s, "Máy khâu trụ da") {
		t.Error("lọc nghề giày làm mất món dùng chung")
	}
	s = moTrang(t, mux, ck, "/qt/do-nghe?tt=da_mua")
	if strings.Contains(s, "Máy khâu trụ da") {
		t.Error("lọc trạng thái đã mua vẫn hiện món chưa mua")
	}
}

// Trang này lộ tổng vốn đã bỏ ra mở tiệm — thợ không được vào.
func TestTrangDoNgheThoKhongVaoDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	r := httptest.NewRequest("GET", "/qt/do-nghe", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusForbidden {
		t.Errorf("thợ mở trang đồ nghề trả %d, muốn 403", w.Code)
	}
}
