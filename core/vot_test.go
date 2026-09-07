package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

const votYamlThu = `version: 1
cap_nhat: "2026-09-06"
so_ca_lam_co_so: 0
nguong_tin_duoc: 100
thang_muc_gap:
  rat_pho_bien: "Tuần nào cũng gặp."
  hiem: "Một năm vài cây."
loi:
  - ma: PP_HONEYCOMB
    ten: "Tổ ong polypropylene (PP)"
    cau_tao: "Nhựa PP ép đùn thành ô lục giác."
che_tao:
  - ma: EP_NONG
    ten: "Ép nóng liền khối (Gen 2)"
    canh_bao_xuong: "LUÔN gõ nghe tiếng trước khi báo giá."
  - ma: KHONG_RO
    ten: "Không rõ cách chế tạo"
loi_hong:
  - ma: CORE_CRUSH
    ten: "Sập lõi tổ ong"
dong_vot:
  - ma: THU_MOT
    hang: "Hãng Thử"
    nuoc: "Mỹ"
    dong: "Dòng Thử"
    loi: PP_HONEYCOMB
    che_tao: [EP_NONG]
    mat: "Carbon"
    vien: co
    gia_vn: [2000000, 4000000]
    muc_gap: rat_pho_bien
    co_so: "Bịa ra để test."
    hay_hong: [CORE_CRUSH]
    nhan_sua: "Nhận viền và mép."
    do_tin_cay: trung_binh
    bien_the:
      - { ten: "Cây Mỏng 14mm", nam: 2023, che_tao: EP_NONG, do_day_mm: 14, khoi_luong_g: [221, 229], nguon_so: hang_cong_bo }
      - ten: "Cây Dày 16mm"
        nam: 2024
        che_tao: KHONG_RO
        do_day_mm: 16
        khoi_luong_g: null
        nguon_so: chua_co
        ghi_chu: "Chưa tra được, phải gõ."
  - ma: THU_VN
    hang: "Hãng Việt Thử"
    nuoc: "Việt Nam"
    dong: "Dòng Việt"
    loi: PP_HONEYCOMB
    che_tao: [KHONG_RO]
    muc_gap: hiem
    nhan_sua: "Cân nhắc, giá vợt thấp."
    do_tin_cay: thap
    bien_the:
      - { ten: "Cây Việt 16mm", nam: 2025, che_tao: KHONG_RO, do_day_mm: 16, khoi_luong_g: 230, nguon_so: hang_cong_bo }
`

func votThu(t *testing.T) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(Root, "data"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(Root, "data", "vot.yaml"), []byte(votYamlThu), 0o644); err != nil {
		t.Fatal(err)
	}
	cuMtime := votMtime
	t.Cleanup(func() {
		votMu.Lock()
		votKho, votMtime = nil, cuMtime
		votMu.Unlock()
	})
}

// Trang phải dựng được, và phải hiện đủ ba thứ thợ cần: tên từng CÂY (không
// gộp thành khoảng), nguồn của con số, và cảnh báo chưa đủ ca thật.
func TestTrangVotDungDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	votThu(t)

	s := moTrang(t, mux, ck, "/qt/vot")
	for _, can := range []string{
		"Cây Mỏng 14mm", "Cây Dày 16mm", "Cây Việt 16mm",
		"Tổ ong polypropylene (PP)", "Ép nóng liền khối (Gen 2)",
		"chưa tra", "Chưa có ca thật chống lưng",
	} {
		if !strings.Contains(s, can) {
			t.Errorf("trang thiếu %q", can)
		}
	}
	// Cây chưa cân được thì phải là gạch ngang, không được bịa số.
	if strings.Contains(s, ">0g<") {
		t.Error("khối lượng null bị in thành 0g")
	}
	if gam(nil) != "—" || so(nil) != "—" {
		t.Error("số trống phải ra gạch ngang")
	}
	// Hãng công bố khoảng thì phải hiện khoảng, không được rút về một số.
	if !strings.Contains(s, "221–229g") {
		t.Error("khoảng cân nặng không lên bảng")
	}
	if gam(&Can{Tu: 230, Den: 230}) != "230g" {
		t.Error("một số phải in gọn một số, không thành 230–230g")
	}
}

// Thợ vào được. Danh mục vợt mà chặn cho riêng chủ thì lúc khách đứng trước
// mặt sẽ không ai tra.
func TestThoVaoDuocDanhMucVot(t *testing.T) {
	mux, ck := dungTrangThu(t, VaiTroTho)
	votThu(t)
	moTrang(t, mux, ck, "/qt/vot")
}

// Sửa file xong là F5 thấy ngay, không phải khởi động lại service.
func TestVotNapLaiKhiFileDoi(t *testing.T) {
	dungKhoTienThu(t)
	votThu(t)

	k, err := NapVot()
	if err != nil {
		t.Fatal(err)
	}
	if k.SoCay() != 3 {
		t.Fatalf("đếm được %d cây, đáng lẽ 3", k.SoCay())
	}

	p := filepath.Join(Root, "data", "vot.yaml")
	them := votYamlThu + `      - { ten: "Cây Việt 14mm", nam: 2025, che_tao: KHONG_RO, do_day_mm: 14, khoi_luong_g: 226, nguon_so: hang_cong_bo }
`
	if err := os.WriteFile(p, []byte(them), 0o644); err != nil {
		t.Fatal(err)
	}
	// ModTime trên Windows có thể trùng nếu ghi quá nhanh — ép cho khác hẳn.
	if err := os.Chtimes(p, votMtimeXa(), votMtimeXa()); err != nil {
		t.Fatal(err)
	}
	k2, err := NapVot()
	if err != nil {
		t.Fatal(err)
	}
	if k2.SoCay() != 4 {
		t.Errorf("file đổi rồi mà vẫn đếm %d cây — cache không nạp lại", k2.SoCay())
	}
}

// Thừa kế: biến thể không ghi lõi thì lấy lõi của dòng cha.
func TestBienTheThuaKeDongCha(t *testing.T) {
	dungKhoTienThu(t)
	votThu(t)

	k, err := NapVot()
	if err != nil {
		t.Fatal(err)
	}
	for _, h := range k.BangVot() {
		if h.Loi == "" || h.Loi == "PP_HONEYCOMB" {
			t.Errorf("%s không thừa kế được tên lõi từ dòng cha (được %q)", h.Ten, h.Loi)
		}
	}
	// Cảnh báo của đời chế tạo phải theo được xuống từng cây.
	var thay bool
	for _, h := range k.BangVot() {
		if h.Ten == "Cây Mỏng 14mm" && strings.Contains(h.CanhBao, "gõ nghe tiếng") {
			thay = true
		}
	}
	if !thay {
		t.Error("cảnh báo của đời ép nóng không xuống tới cây")
	}
}

// votMtimeXa — mốc thời gian lệch hẳn để ép ModTime đổi. Windows ghi hai lần
// trong cùng một nhịp đồng hồ thì ModTime y nguyên, và NapVot tưởng file chưa
// đổi nên trả bản nạp cũ.
func votMtimeXa() time.Time { return time.Now().Add(2 * time.Hour) }

// Soát file thật. Sửa data/vot.yaml sai mã lõi, sai mã chế tạo hay quên
// nguon_so thì trang vẫn dựng được nhưng cột đó hiện ra mã trần — thợ đọc
// "EP_NONG" thay vì "Ép nóng liền khối". Test này bắt ngay lúc sửa.
func TestDanhMucVotThatDocDuoc(t *testing.T) {
	cu, cuKho, cuMtime := Root, votKho, votMtime
	Root = filepath.Join("..", "..")
	votKho, votMtime = nil, 0
	t.Cleanup(func() {
		votMu.Lock()
		Root, votKho, votMtime = cu, cuKho, cuMtime
		votMu.Unlock()
	})

	k, err := NapVot()
	if err != nil {
		t.Fatal(err)
	}
	if len(k.DongVot) < 38 {
		t.Errorf("chỉ có %d dòng vợt, đáng lẽ từ 38", len(k.DongVot))
	}
	if k.SoCay() < 300 {
		t.Errorf("chỉ có %d cây, đáng lẽ từ 300", k.SoCay())
	}

	for _, d := range k.DongVot {
		if k.TenLoi(d.Loi) == d.Loi {
			t.Errorf("%s: mã lõi %q không có trong mục loi:", d.Ma, d.Loi)
		}
		if _, co := NhanMucGap[d.MucGap]; !co {
			t.Errorf("%s: mức hay gặp %q không hợp lệ", d.Ma, d.MucGap)
		}
		for _, h := range d.HayHong {
			if k.TenLoiHong(h) == h {
				t.Errorf("%s: mã hỏng %q không có trong mục loi_hong:", d.Ma, h)
			}
		}
		for _, b := range d.BienThe {
			if b.Ten == "" {
				t.Errorf("%s: có biến thể không tên", d.Ma)
			}
			if k.TenCheTao(b.CheTao) == b.CheTao {
				t.Errorf("%s / %s: mã chế tạo %q không có trong mục che_tao:", d.Ma, b.Ten, b.CheTao)
			}
			if _, co := NhanNguonSo[b.NguonSo]; !co {
				t.Errorf("%s / %s: nguon_so %q không hợp lệ", d.Ma, b.Ten, b.NguonSo)
			}
			// chua_co nói về CÂN NẶNG — con số quyết định báo giá. Độ dày
			// hãng nào cũng công bố nên có thể có sẵn; cân thì phải tự đo.
			if b.NguonSo == "chua_co" && b.KhoiLuongG != nil {
				t.Errorf("%s / %s: ghi chua_co nhưng vẫn đưa số cân", d.Ma, b.Ten)
			}
			if b.NguonSo != "chua_co" && b.DoDayMm == nil && b.KhoiLuongG == nil {
				t.Errorf("%s / %s: nhận nguồn %q nhưng không có số nào", d.Ma, b.Ten, b.NguonSo)
			}
		}
	}

	// Nhóm OEM không nhãn không có cây nào mang tên riêng, nhưng lại là
	// nhóm gặp nhiều nhất. Rơi khỏi bảng là thợ tra không ra.
	var coOEM bool
	for _, h := range k.BangVot() {
		if h.MaDong == "OEM_KHONG_NHAN" {
			coOEM = true
		}
	}
	if !coOEM {
		t.Error("nhóm OEM không nhãn không lên bảng")
	}

	// Bảng phẳng phải đủ cây, và không cây nào lọt ra với ô trống.
	bang := k.BangVot()
	if len(bang) != k.SoCay() {
		t.Errorf("bảng có %d hàng nhưng đếm được %d cây", len(bang), k.SoCay())
	}
	for _, h := range bang {
		if h.Ten == "" || h.Loi == "" || h.CheTao == "" || h.MucGapNhan == "" {
			t.Errorf("hàng %q (%s) có ô trống", h.Ten, h.MaDong)
		}
		if _, co := k.ChiTiet()[h.MaDong]; !co {
			t.Errorf("hàng %q trỏ tới dòng %q không có trong ChiTiet — bấm vào sẽ không mở được", h.Ten, h.MaDong)
		}
	}
}

// Gõ nhầm khoảng cân phải chặn ngay lúc nạp file, đừng để [235, 221] lên
// bảng thành "235–221g" rồi thợ đọc ngược.
func TestKhoangCanSaiThiBaoLoi(t *testing.T) {
	xau := map[string]string{
		"ngược đầu":  "khoi_luong_g: [235, 221]",
		"ba phần tử": "khoi_luong_g: [221, 229, 235]",
		"chữ":        "khoi_luong_g: nang",
	}
	for ten, dong := range xau {
		t.Run(ten, func(t *testing.T) {
			var c struct {
				K *Can `yaml:"khoi_luong_g"`
			}
			if err := yaml.Unmarshal([]byte(dong+"\n"), &c); err == nil {
				t.Errorf("%q nạp lọt, đáng lẽ phải báo lỗi", dong)
			}
		})
	}
	var c struct {
		K *Can `yaml:"khoi_luong_g"`
	}
	if err := yaml.Unmarshal([]byte("khoi_luong_g: null\n"), &c); err != nil || c.K != nil {
		t.Errorf("null phải ra nil, không phải lỗi (err=%v)", err)
	}
}
