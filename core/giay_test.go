package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const giayYamlThu = `version: 1
cap_nhat: "2026-09-09"
so_ca_lam_co_so: 0
nguong_tin_duoc: 100
thang_muc_gap:
  rat_pho_bien: "Gặp gần như hằng tuần"
  hiem: "Hoạ hoằn"
gan_de:
  - ma: dan_nguoi
    ten: "Dán nguội"
    nhan_ra: "Rãnh liền mạch quanh mép."
    boc_duoc: "Được."
    xuong_nhan: nhan
  - ma: luu_hoa
    ten: "Lưu hoá"
    nhan_ra: "Dải cao su phủ lên mép vải."
    boc_duoc: "Rất khó."
    xuong_nhan: tu_choi
    canh_bao_xuong: "TỪ CHỐI thay đế."
de_ngoai:
  - ma: cao_su_carbon
    ten: "Cao su carbon"
de_giua:
  - ma: eva
    ten: "EVA thường"
  - ma: peba
    ten: "PEBA"
    canh_bao: "KHÔNG gia nhiệt."
loi_hong:
  - ma: de_ngoai_mon
    ten: "Đế ngoài mòn trơ gai"
dong_giay:
  - ma: hang-thu
    hang: "Hãng Thử"
    nuoc: "Mỹ"
    dong: "Chạy bộ, đua"
    mon: "Chạy bộ"
    gan_de: [dan_nguoi]
    de_ngoai: [cao_su_carbon]
    de_giua: [eva, peba]
    muc_gap: rat_pho_bien
    co_so: "Bịa ra để test."
    hay_hong: [de_ngoai_mon]
    xu_ly: xet
    nhan_sua: "Xét từng model."
    do_tin_cay: trung_binh
    bien_the:
      - { ten: "Dòng chạy phổ thông", de_giua: eva, xu_ly: nhan }
      - ten: "Dòng đua"
        de_giua: peba
        xu_ly: tu_choi
        nhan_sua: "Bọt PEBA kỵ nhiệt."
  - ma: hang-vai-vn
    hang: "Hãng Vải Việt"
    nuoc: "Việt Nam"
    dong: "Giày vải"
    mon: "Đi lại"
    gan_de: [luu_hoa]
    de_giua: [eva]
    muc_gap: hiem
    co_so: "Bịa ra để test."
    hay_hong: [de_ngoai_mon]
    xu_ly: tu_choi
    nhan_sua: "TỪ CHỐI — giày lưu hoá."
    do_tin_cay: cao
`

func giayThu(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(Root, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(Root, "data", "giay.yaml"), []byte(giayYamlThu), 0o644); err != nil {
		t.Fatal(err)
	}
	cuMtime := giayMtime
	t.Cleanup(func() {
		giayMu.Lock()
		giayKho, giayMtime = nil, cuMtime
		giayMu.Unlock()
	})
}

// Trang phải dựng được, và phải hiện đủ thứ thợ cần: tên từng model, kiểu gắn
// đế, quyết định nhận hay từ chối, và cảnh báo chưa đủ ca thật.
func TestTrangGiayDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	giayThu(t)

	s := moTrang(t, mux, ck, "/qt/giay")
	for _, can := range []string{
		"Dòng chạy phổ thông", "Dòng đua", "Hãng Vải Việt",
		"Dán nguội", "Lưu hoá", "Chưa có ca thật chống lưng",
	} {
		if !strings.Contains(s, can) {
			t.Errorf("trang thiếu %q", can)
		}
	}
	// Mã trần lọt lên bảng nghĩa là tra ngược hỏng — thợ đọc "dan_nguoi".
	for _, ma := range []string{"dan_nguoi", "luu_hoa", "tu_choi"} {
		if strings.Contains(s, ">"+ma+"<") {
			t.Errorf("mã trần %q hiện ra thay vì tên", ma)
		}
	}
}

// Bảng kiểu đế phải nằm sẵn trên trang. Danh mục 57 dòng không bao giờ phủ
// hết cái khách mang tới; lúc search hụt thì đây là thứ duy nhất kết luận
// được, mà nó chỉ nằm trong file markdown của agent thì thợ không với tới.
func TestTrangGiayCoBangKieuDe(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	giayThu(t)

	s := moTrang(t, mux, ck, "/qt/giay")
	for _, can := range []string{
		"Các kiểu gắn đế",
		"Rãnh liền mạch quanh mép.", // nhan_ra
		"Rất khó.",                  // boc_duoc của đế lưu hoá
		"KHÔNG gia nhiệt.",          // cảnh báo của tầng bọt
		"tra theo kiểu đế",          // nút ở ô search hụt
	} {
		if !strings.Contains(s, can) {
			t.Errorf("bảng kiểu đế thiếu %q", can)
		}
	}
}

// Thợ vào được. Danh mục mà chặn cho riêng chủ thì lúc khách đứng trước mặt
// sẽ không ai tra — y hệt lý do ở /qt/vot.
func TestThoVaoDuocDanhMucGiay(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	giayThu(t)
	moTrang(t, mux, ck, "/qt/giay")
}

// Sửa file xong là F5 thấy ngay, không phải khởi động lại service.
func TestGiayNapLaiKhiFileDoi(t *testing.T) {
	dungKhoTienThu(t)
	giayThu(t)

	k, err := NapGiay()
	if err != nil {
		t.Fatal(err)
	}
	if k.SoDoi() != 3 {
		t.Fatalf("đếm được %d model, đáng lẽ 3", k.SoDoi())
	}

	p := filepath.Join(Root, "data", "giay.yaml")
	them := giayYamlThu + `    bien_the:
      - { ten: "Giày vải trơn", xu_ly: tu_choi }
      - { ten: "Giày vải cổ cao", xu_ly: tu_choi }
`
	if err := os.WriteFile(p, []byte(them), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(p, votMtimeXa(), votMtimeXa()); err != nil {
		t.Fatal(err)
	}
	k2, err := NapGiay()
	if err != nil {
		t.Fatal(err)
	}
	if k2.SoDoi() != 4 {
		t.Errorf("file đổi rồi mà vẫn đếm %d model — cache không nạp lại", k2.SoDoi())
	}
}

// Model thắng hãng. Hãng ghi "xét từng đôi" nhưng model đã chốt thì hàng trên
// bảng phải mang quyết định của model, không phải của hãng — đây là cả lý do
// bảng này tồn tại.
func TestModelThangHang(t *testing.T) {
	dungKhoTienThu(t)
	giayThu(t)

	k, err := NapGiay()
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]HangGiay{}
	for _, h := range k.BangGiay() {
		got[h.Ten] = h
	}
	if got["Dòng chạy phổ thông"].XuLy != "nhan" {
		t.Errorf("dòng chạy ra %q, đáng lẽ nhan", got["Dòng chạy phổ thông"].XuLy)
	}
	if got["Dòng đua"].XuLy != "tu_choi" {
		t.Errorf("dòng đua ra %q, đáng lẽ tu_choi", got["Dòng đua"].XuLy)
	}
	// Hãng không tách model thì vẫn phải có một hàng, mang tên hãng.
	if _, co := got["Hãng Vải Việt"]; !co {
		t.Error("hãng không có bien_the bị rơi khỏi bảng")
	}
	if !got["Hãng Vải Việt"].TuVN {
		t.Error("hãng Việt không được đánh dấu, bộ lọc xuất xứ sẽ sai")
	}
}

// Dòng cha có nhiều lựa chọn mà model không chốt thì KHÔNG được đoán bừa lấy
// cái đầu tiên: đoán sai đế giữa là hơ nóng nhầm bọt PEBA.
func TestKhongDoanKhiDongChaCoNhieuLuaChon(t *testing.T) {
	dungKhoTienThu(t)
	giayThu(t)

	k, err := NapGiay()
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range k.BangGiay() {
		// Hãng Vải Việt chỉ khai một loại đế giữa nên thừa kế được.
		if h.Hang == "Hãng Vải Việt" && h.DeGiua != "EVA thường" {
			t.Errorf("một lựa chọn duy nhất mà không thừa kế: %q", h.DeGiua)
		}
		// Hãng Vải Việt không khai de_ngoai — phải ra gạch ngang, không phải
		// tên vật liệu nào đó nhặt bừa.
		if h.Hang == "Hãng Vải Việt" && h.DeNgoai != "—" {
			t.Errorf("không khai đế ngoài mà vẫn ra %q", h.DeNgoai)
		}
	}
}

// Cảnh báo của kiểu gắn đế và của tầng bọt phải theo xuống tận hàng: cả hai
// đều là thứ làm hỏng đôi giày vĩnh viễn trong lúc sửa.
func TestCanhBaoXuongToiHang(t *testing.T) {
	dungKhoTienThu(t)
	giayThu(t)

	k, err := NapGiay()
	if err != nil {
		t.Fatal(err)
	}
	var thayNhiet, thayLuuHoa bool
	for _, h := range k.BangGiay() {
		if h.Ten == "Dòng đua" && strings.Contains(h.CanhBao, "KHÔNG gia nhiệt") {
			thayNhiet = true
		}
		if h.Hang == "Hãng Vải Việt" && strings.Contains(h.CanhBao, "TỪ CHỐI") {
			thayLuuHoa = true
		}
	}
	if !thayNhiet {
		t.Error("cảnh báo kỵ nhiệt của PEBA không xuống tới model")
	}
	if !thayLuuHoa {
		t.Error("cảnh báo của đế lưu hoá không xuống tới hàng")
	}
}

// Soát file thật. Sửa data/giay.yaml sai mã gắn đế, sai mã bọt hay sai xu_ly
// thì trang vẫn dựng được nhưng cột đó hiện ra mã trần — thợ đọc "dan_nguoi"
// thay vì "Dán nguội". Test này bắt ngay lúc sửa.
func TestDanhMucGiayThatDocDuoc(t *testing.T) {
	cu, cuKho, cuMtime := Root, giayKho, giayMtime
	Root = filepath.Join("..", "..")
	giayKho, giayMtime = nil, 0
	t.Cleanup(func() {
		giayMu.Lock()
		Root, giayKho, giayMtime = cu, cuKho, cuMtime
		giayMu.Unlock()
	})

	k, err := NapGiay()
	if err != nil {
		t.Fatal(err)
	}
	if len(k.DongGiay) < 30 {
		t.Errorf("chỉ có %d dòng hãng, đáng lẽ từ 30", len(k.DongGiay))
	}

	hopLe := map[string]bool{"nhan": true, "nhan_dk": true, "xet": true, "gui_di": true, "tu_choi": true}
	for _, d := range k.DongGiay {
		if !hopLe[d.XuLy] {
			t.Errorf("%s: xu_ly %q không hợp lệ", d.Ma, d.XuLy)
		}
		if strings.TrimSpace(d.NhanSua) == "" {
			t.Errorf("%s: không nói nhận sửa thế nào", d.Ma)
		}
		if strings.TrimSpace(d.CoSo) == "" {
			t.Errorf("%s: muc_gap là ước lượng mà không có co_so", d.Ma)
		}
		if _, co := NhanMucGap[d.MucGap]; !co {
			t.Errorf("%s: mức hay gặp %q không hợp lệ", d.Ma, d.MucGap)
		}
		if _, co := NhanTinCay[d.DoTinCay]; !co {
			t.Errorf("%s: do_tin_cay %q không hợp lệ", d.Ma, d.DoTinCay)
		}
		for _, m := range d.GanDe {
			if k.TenGanDe(m) == m {
				t.Errorf("%s: mã gắn đế %q không có trong mục gan_de:", d.Ma, m)
			}
		}
		for _, m := range d.DeNgoai {
			if k.TenDeNgoai(m) == m {
				t.Errorf("%s: mã đế ngoài %q không có trong mục de_ngoai:", d.Ma, m)
			}
		}
		for _, m := range d.DeGiua {
			if k.TenDeGiua(m) == m {
				t.Errorf("%s: mã đế giữa %q không có trong mục de_giua:", d.Ma, m)
			}
		}
		for _, m := range d.HayHong {
			if k.TenHongGiay(m) == m {
				t.Errorf("%s: mã hỏng %q không có trong mục loi_hong:", d.Ma, m)
			}
		}
		for _, b := range d.BienThe {
			if b.Ten == "" {
				t.Errorf("%s: có biến thể không tên", d.Ma)
			}
			if b.XuLy != "" && !hopLe[b.XuLy] {
				t.Errorf("%s / %s: xu_ly %q không hợp lệ", d.Ma, b.Ten, b.XuLy)
			}
			if b.GanDe != "" && k.TenGanDe(b.GanDe) == b.GanDe {
				t.Errorf("%s / %s: mã gắn đế %q không có trong mục gan_de:", d.Ma, b.Ten, b.GanDe)
			}
			if b.DeGiua != "" && k.TenDeGiua(b.DeGiua) == b.DeGiua {
				t.Errorf("%s / %s: mã đế giữa %q không có trong mục de_giua:", d.Ma, b.Ten, b.DeGiua)
			}
		}
	}

	// Mọi kiểu gắn đế phải tự khai xưởng nhận hay không: bảng này là bảng
	// quyết định, một ô trống ở đây là một ca báo giá sai.
	for _, g := range k.GanDe {
		if !hopLe[g.XuongNhan] {
			t.Errorf("gan_de %s: xuong_nhan %q không hợp lệ", g.Ma, g.XuongNhan)
		}
		if strings.TrimSpace(g.NhanRa) == "" {
			t.Errorf("gan_de %s: không nói nhận ra bằng gì", g.Ma)
		}
	}

	// Bảng phẳng phải đủ hàng, và không hàng nào lọt ra với ô trống hay trỏ
	// tới một dòng không mở được panel.
	bang := k.BangGiay()
	if len(bang) != k.SoDoi() {
		t.Errorf("bảng có %d hàng nhưng đếm được %d model", len(bang), k.SoDoi())
	}
	ct := k.ChiTiet()
	for _, h := range bang {
		if h.Ten == "" || h.GanDeNhan == "" || h.XuLyNhan == "" || h.MucGapNhan == "" {
			t.Errorf("hàng %q (%s) có ô trống", h.Ten, h.MaDong)
		}
		if _, co := ct[h.MaDong]; !co {
			t.Errorf("hàng %q trỏ tới dòng %q không có trong ChiTiet — bấm vào sẽ không mở được", h.Ten, h.MaDong)
		}
	}

	// Giày court là nhóm PHẢI từ chối. Có mặt trong danh mục để tra ra rồi
	// từ chối cho đúng — rơi khỏi bảng là thợ nhận nhầm.
	var yonex, kswiss bool
	for _, h := range bang {
		if h.Hang == "Yonex" {
			yonex = true
			if h.XuLy != "tu_choi" {
				t.Errorf("Yonex ra %q, đáng lẽ tu_choi", h.XuLy)
			}
		}
		if h.Hang == "K-Swiss" {
			kswiss = true
		}
	}
	if !yonex || !kswiss {
		t.Error("hãng giày court rơi khỏi bảng — thợ tra không ra để từ chối")
	}
}
