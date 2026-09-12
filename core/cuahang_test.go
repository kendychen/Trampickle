package core

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// napGiaThat nạp bảng giá thật vào GIA. Trang cửa hàng in ra danh sách việc
// và giá khởi điểm; chạy trên bảng rỗng thì test xanh mà chẳng soi gì.
func napGiaThat(t *testing.T) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "vanhanh", "bang-gia.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cuGIA, cuGD := GIA, GiaiDoan
	t.Cleanup(func() { GIA, GiaiDoan = cuGIA, cuGD })
	GIA = BangGia{}
	if err := yaml.Unmarshal(b, &GIA); err != nil {
		t.Fatal(err)
	}
	GiaiDoan = GIA.GiaiDoanHienTai
}

// datDiaChi đặt địa chỉ trạm cho test rồi trả lại như cũ.
func datDiaChi(t *testing.T, dc string) {
	t.Helper()
	truoc := LienHeHienTai()
	moi := truoc
	moi.DiaChi = dc
	if _, err := DatLienHe(moi); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = DatLienHe(truoc) })
}

// moCuaHang: gốc dữ liệu tạm, chế độ Online, có địa chỉ, kho đơn rỗng.
func moCuaHang(t *testing.T) {
	t.Helper()
	gocTam(t)
	// Bộ template chỉ nạp ở InitTemplates, và test gọi thẳng handler nên
	// không đi qua chỗ khởi động. Không nạp thì render bắn nil pointer.
	if err := InitTemplates(); err != nil {
		t.Fatal(err)
	}
	napGiaThat(t)
	if err := NapDon(); err != nil {
		t.Fatal(err)
	}
	datDiaChi(t, "12 Nguyễn Trãi, Thanh Xuân, Hà Nội")
	if err := DatCheDo(CheDoOnline); err != nil {
		t.Fatal(err)
	}
}

// postDat gửi form đặt hàng dạng multipart, đúng như trình duyệt gửi, kèm
// cặp cookie/_csrf khớp nhau — hCuaHangGui tự kiểm CSRF chứ không nhờ lớp bọc.
func postDat(t *testing.T, v url.Values) *http.Request {
	t.Helper()
	const tok = "tokentestcuahang0123456789abcdef0123456789abcdef0123456789abcdef"
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	if err := w.WriteField("_csrf", tok); err != nil {
		t.Fatal(err)
	}
	for k, vs := range v {
		for _, s := range vs {
			if err := w.WriteField(k, s); err != nil {
				t.Fatal(err)
			}
		}
	}
	w.Close()
	r := httptest.NewRequest(http.MethodPost, "/cua-hang", &b)
	r.Header.Set("Content-Type", w.FormDataContentType())
	r.AddCookie(&http.Cookie{Name: tenCookieCSRF, Value: tok})
	return r
}

func TestCuaHangDongKhiChuaCoDiaChi(t *testing.T) {
	moCuaHang(t)
	datDiaChi(t, "")
	if CuaHangMo() {
		t.Fatal("chưa có địa chỉ thì cửa hàng phải đóng")
	}
}

func TestCuaHangDongKhiTaiXuong(t *testing.T) {
	moCuaHang(t)
	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	if CuaHangMo() {
		t.Fatal("bản Tại xưởng thì cửa hàng phải đóng")
	}
}

func TestDatHangTaoDonChoHangVe(t *testing.T) {
	moCuaHang(t)
	w := httptest.NewRecorder()
	hCuaHangGui(w, postDat(t, url.Values{
		"loai":       {LoaiVot},
		"dich_vu":    {"DAN_VIEN"},
		"vot_hang":   {"Joola Ben Johns"},
		"gia_tri":    {"3000000"},
		"tinh_trang": {"Viền bong một đoạn 10cm"},
		"ten":        {"Chị Lan"},
		"lien_he":    {"0900000000"},
		"dia_chi":    {"45 Lê Lợi, Q1, TP.HCM"},
	}))

	ds := LocDon(BoLoc{TrangThai: TTChoHangVe})
	if len(ds) != 1 {
		t.Fatalf("phải có đúng 1 đơn chờ hàng về, có %d", len(ds))
	}
	d := ds[0]
	if !strings.HasPrefix(d.Ma, "TV-") {
		t.Fatalf("đơn vợt phải mang tiền tố TV-, nhận %s", d.Ma)
	}
	if d.Token == "" {
		t.Fatal("thiếu token tra cứu")
	}
	if d.NguonDonHopLe() != NguonWeb {
		t.Fatal("đơn đặt trên web phải ghi nguồn web")
	}
	if d.KenhNhan != "ship" {
		t.Fatalf("kênh nhận phải là ship, nhận %q", d.KenhNhan)
	}
	if d.KhachDiaChi == "" {
		t.Fatal("thiếu địa chỉ trả về")
	}
	if d.VotGiaTri != 3000000 {
		t.Fatalf("giá trị món phải là 3000000, nhận %d", d.VotGiaTri)
	}
	// Chưa khám thì chưa có tiền. Giá chốt sinh ở bước báo giá.
	if d.TongTien != 0 || len(d.DongTien) != 0 {
		t.Fatalf("đơn mới đặt không được có tiền: %d / %d dòng", d.TongTien, len(d.DongTien))
	}
	if len(d.LichSu) == 0 {
		t.Fatal("phải để lại một mốc trong lịch sử")
	}
	// Màn kết phải in mã đơn và địa chỉ — đó là toàn bộ việc của nó.
	than := w.Body.String()
	if !strings.Contains(than, d.Ma) {
		t.Fatal("màn kết không in mã đơn")
	}
	if !strings.Contains(than, "Nguyễn Trãi") {
		t.Fatal("màn kết không in địa chỉ gửi tới")
	}
}

func TestDatHangGiayMangTienToTG(t *testing.T) {
	moCuaHang(t)
	w := httptest.NewRecorder()
	hCuaHangGui(w, postDat(t, url.Values{
		"loai":       {LoaiGiay},
		"dich_vu":    {"THAY_DE_GIAY"},
		"giay_hang":  {"Nike Pegasus 40"},
		"giay_size":  {"42"},
		"giay_kieu":  {GiayChayBo},
		"gia_tri":    {"3200000"},
		"tinh_trang": {"Đế mòn hết gai"},
		"ten":        {"Anh Nam"},
		"lien_he":    {"0900000001"},
		"dia_chi":    {"45 Lê Lợi, Q1"},
	}))
	_ = w

	ds := LocDon(BoLoc{TrangThai: TTChoHangVe, Loai: LoaiGiay})
	if len(ds) != 1 {
		t.Fatalf("phải có 1 đơn giày, có %d", len(ds))
	}
	if !strings.HasPrefix(ds[0].Ma, "TG-") {
		t.Fatalf("đơn giày phải mang tiền tố TG-, nhận %s", ds[0].Ma)
	}
	if ds[0].GiayKieu != GiayChayBo {
		t.Fatalf("mất kiểu giày: %q", ds[0].GiayKieu)
	}
}

func TestThieuLienHeThiKhongTaoDon(t *testing.T) {
	moCuaHang(t)
	w := httptest.NewRecorder()
	hCuaHangGui(w, postDat(t, url.Values{
		"loai":    {LoaiVot},
		"dich_vu": {"DAN_VIEN"},
		"ten":     {"Chị Lan"},
		// thiếu lien_he và dia_chi
	}))

	if n := len(LocDon(BoLoc{TrangThai: TTChoHangVe})); n != 0 {
		t.Fatalf("thiếu liên hệ mà vẫn tạo %d đơn", n)
	}
	if w.Code != http.StatusOK {
		t.Fatalf("phải trả lại form kèm lời báo, nhận %d", w.Code)
	}
}

// Trang khách không bao giờ được in một con số giá trần trụi: mọi số phải đi
// kèm nhãn "giá từ", còn việc bao_gia_rieng thì không có số nào.
func TestGiaLuonLaGiaTu(t *testing.T) {
	moCuaHang(t)
	for _, o := range DichVuChoCuaHang(LoaiVot) {
		if o.Gia == "" {
			t.Errorf("%s: ô giá rỗng — phải có chữ, kể cả khi là báo giá riêng", o.Ma)
		}
	}
	w := httptest.NewRecorder()
	hCuaHang(w, httptest.NewRequest(http.MethodGet, "/cua-hang", nil))
	if !strings.Contains(w.Body.String(), ND("cuahang.buoc.gia-tu")) {
		t.Fatal("trang cửa hàng không thấy nhãn giá từ")
	}
}

// tepGia dựng một tệp tải lên giả để test không phải đi qua HTTP thật.
func tepGia(t *testing.T, truong, ten string, than []byte) *multipart.FileHeader {
	t.Helper()
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	f, err := w.CreateFormFile(truong, ten)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(than); err != nil {
		t.Fatal(err)
	}
	w.Close()
	r := multipart.NewReader(&b, w.Boundary())
	form, err := r.ReadForm(1 << 20)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { form.RemoveAll() })
	return form.File[truong][0]
}

func TestLuuAnhVaoDon(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-900", TrangThai: TTChoHangVe}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	// PNG tối thiểu: 8 byte chữ ký là đủ cho duoiAnh nhận ra.
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 32)...)
	fh := tepGia(t, "anh", "hong.png", png)

	if n := luuAnhVaoDon(d, []*multipart.FileHeader{fh}, "truoc"); n != 1 {
		t.Fatalf("phải lưu 1 ảnh, lưu %d", n)
	}
	if len(d.Anh) != 1 {
		t.Fatalf("đơn phải có 1 ảnh, có %d", len(d.Anh))
	}
	if !strings.HasPrefix(d.Anh[0], "truoc-") {
		t.Fatalf("tên ảnh phải mang nhãn truoc-, nhận %q", d.Anh[0])
	}
}

// Tệp không phải ảnh bị bỏ im lặng — không nổ, không lưu.
func TestBoQuaTepKhongPhaiAnh(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-901", TrangThai: TTChoHangVe}
	fh := tepGia(t, "anh", "virus.exe", []byte("MZ khong phai anh"))
	if n := luuAnhVaoDon(d, []*multipart.FileHeader{fh}, "truoc"); n != 0 {
		t.Fatalf("không được lưu tệp lạ, lưu %d", n)
	}
	if len(d.Anh) != 0 {
		t.Fatal("đơn không được dính tệp lạ")
	}
}

// Không có token thì không có đơn. Đường /cua-hang nằm trong danh sách miễn
// của lớp bọc, nên nếu handler quên gọi KiemCSRFMultipart thì mất bảo vệ mà
// không ai biết — test này là chốt chặn.
func TestDatHangKhongTokenBiTuChoi(t *testing.T) {
	moCuaHang(t)
	truoc := len(LocDon(BoLoc{}))

	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	for k, v := range map[string]string{"loai": "vot", "ten": "A", "lien_he": "0900", "dia_chi": "X"} {
		w.WriteField(k, v)
	}
	w.Close()
	r := httptest.NewRequest(http.MethodPost, "/cua-hang", &b)
	r.Header.Set("Content-Type", w.FormDataContentType())

	rec := httptest.NewRecorder()
	hCuaHangGui(rec, r)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("thiếu token phải là 403, nhận %d", rec.Code)
	}
	if sau := len(LocDon(BoLoc{})); sau != truoc {
		t.Fatalf("không được tạo đơn: %d -> %d", truoc, sau)
	}
}

func TestTuChoiGiaySaiKieu(t *testing.T) {
	moCuaHang(t)
	if LyDoTuChoiCuaHang(LoaiGiay, "court", 3000000) == "" {
		t.Fatal("giày court phải bị từ chối")
	}
	if got := LyDoTuChoiCuaHang(LoaiGiay, GiayChayBo, 3000000); got != "" {
		t.Fatalf("giày chạy bộ phải nhận, bị từ chối vì %q", got)
	}
	if got := LyDoTuChoiCuaHang(LoaiGiay, GiayDiLai, 3000000); got != "" {
		t.Fatalf("giày đi lại phải nhận, bị từ chối vì %q", got)
	}
	if LyDoTuChoiCuaHang(LoaiGiay, "", 3000000) == "" {
		t.Fatal("chưa chọn kiểu giày thì chưa nhận được")
	}
}

func TestTuChoiMonDuoiNguongShip(t *testing.T) {
	moCuaHang(t)
	ng := NguongHienTai().NguongMoKenhShipDong
	if ng <= 0 {
		t.Skip("chưa đặt ngưỡng mở kênh ship")
	}
	if LyDoTuChoiCuaHang(LoaiVot, "", ng-1) == "" {
		t.Fatalf("món dưới %d phải bị chặn", ng)
	}
	if got := LyDoTuChoiCuaHang(LoaiVot, "", ng); got != "" {
		t.Fatalf("đúng ngưỡng phải nhận, bị từ chối vì %q", got)
	}
}

// Không khai giá trị món thì KHÔNG chặn — thợ khám rồi mới biết, và bắt khách
// tự định giá cây vợt của họ là một cái cửa đóng vô cớ.
func TestKhongKhaiGiaTriThiVanNhan(t *testing.T) {
	moCuaHang(t)
	if got := LyDoTuChoiCuaHang(LoaiVot, "", 0); got != "" {
		t.Fatalf("không khai giá trị mà bị chặn: %q", got)
	}
}

func TestTuChoiThiKhongTaoDon(t *testing.T) {
	moCuaHang(t)
	w := httptest.NewRecorder()
	hCuaHangGui(w, postDat(t, url.Values{
		"loai":       {LoaiGiay},
		"dich_vu":    {"THAY_DE_GIAY"},
		"giay_hang":  {"Nike Court Vision"},
		"giay_kieu":  {"court"},
		"gia_tri":    {"3000000"},
		"tinh_trang": {"Mòn đế"},
		"ten":        {"Anh Nam"},
		"lien_he":    {"0900000002"},
		"dia_chi":    {"45 Lê Lợi"},
	}))

	if n := len(LocDon(BoLoc{TrangThai: TTChoHangVe})); n != 0 {
		t.Fatalf("ca từ chối mà vẫn tạo %d đơn", n)
	}
	if !strings.Contains(w.Body.String(), "chạy bộ") {
		t.Fatal("phải nói rõ lý do ngay trên trang, trước khi khách ra bưu điện")
	}
}

func TestTraCuuBangToken(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-910", Token: "abc12345", TrangThai: TTChoHangVe, KhachTen: "Chị Lan"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tra-cuu/abc12345", nil)
	r.SetPathValue("token", "abc12345")
	hTraCuu(w, r)

	if !strings.Contains(w.Body.String(), "TV-2609-910") {
		t.Fatal("mở bằng token phải ra thẳng đơn")
	}
}

func TestTokenSaiKhongLoDon(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-911", Token: "abc12345", TrangThai: TTChoHangVe, KhachTen: "Chị Lan"}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tra-cuu/sai00000", nil)
	r.SetPathValue("token", "sai00000")
	hTraCuu(w, r)

	if strings.Contains(w.Body.String(), "TV-2609-911") {
		t.Fatal("token sai mà vẫn lộ đơn")
	}
}

// --- Khối thanh toán trên trang tra cứu ------------------------------

func TestTraCuuHienTienKhiXongVaConNo(t *testing.T) {
	moCuaHang(t)
	d := &Don{
		Ma: "TV-2609-777", Token: "abc12777", TrangThai: TTXong,
		KhachTen: "Chị Lan", KhachLienHe: "0900000000", TongTien: 250000,
	}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tra-cuu/abc12777", nil)
	r.SetPathValue("token", "abc12777")
	hTraCuu(w, r)

	than := w.Body.String()
	if !strings.Contains(than, ND("cuahang.tra.tieu-de")) {
		t.Fatal("đơn xong còn nợ mà không thấy khối thanh toán")
	}
	if !strings.Contains(than, "TV-2609-777") {
		t.Fatal("nội dung chuyển khoản phải hiện mã đơn để khách chép đúng")
	}
	if !strings.Contains(than, ND("cuahang.tra.cod")) {
		t.Fatal("thiếu lựa chọn COD")
	}
}

// Đơn chưa xong thì không được đòi tiền.
func TestChuaXongThiKhongDoiTien(t *testing.T) {
	moCuaHang(t)
	d := &Don{
		Ma: "TV-2609-778", Token: "abc12778", TrangThai: TTDangSua,
		KhachTen: "Chị Lan", TongTien: 250000,
	}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/tra-cuu/abc12778", nil)
	r.SetPathValue("token", "abc12778")
	hTraCuu(w, r)

	if strings.Contains(w.Body.String(), ND("cuahang.tra.tieu-de")) {
		t.Fatal("đơn đang sửa mà đã hiện khối thanh toán")
	}
}

// Bấm "tôi sẽ chuyển khoản" phải ghi vào đơn — spec đòi đơn nhớ ý định này.
func TestGhiHinhThucTra(t *testing.T) {
	moCuaHang(t)
	d := &Don{Ma: "TV-2609-779", Token: "abc12779", TrangThai: TTXong, TongTien: 250000}
	if err := LuuDon(d); err != nil {
		t.Fatal(err)
	}
	r := postForm("/tra-cuu/abc12779/hinh-thuc-tra", url.Values{"cach": {TraQR}})
	r.SetPathValue("token", "abc12779")
	hChonHinhThucTra(httptest.NewRecorder(), r)

	sau, co := LayDon("TV-2609-779")
	if !co {
		t.Fatal("mất đơn")
	}
	if sau.HinhThucTra != TraQR {
		t.Fatalf("chưa ghi hình thức trả: %q", sau.HinhThucTra)
	}
}
