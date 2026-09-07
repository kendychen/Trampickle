package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// Bản rút gọn của vanhanh/bang-gia.yaml, giữ đúng mấy thứ dễ hỏng: comment
// trên đầu khoá, comment cuối dòng, dòng trống giữa các khoá, và một khối
// khác nằm sau khối nguong.
const yamlThu = `# Bảng giá — người viết tay.
giai_doan_hien_tai: 1

nguong:
  # Vượt ngưỡng này -> TỪ CHỐI CA, không thương lượng.
  # Lý do: swing weight tỉ lệ m*r^2.
  tang_khoi_luong_toi_da_g: 3.0

  # Lãi gộp dưới mức này -> cảnh báo
  bien_gop_toi_thieu_dong: 50000

  cac_qua_kenh_tra_tien_dong: 143000
  ship_ca_tu_choi_dong: 30000
  nguong_mo_kenh_ship_dong: 800000
  free_ship_ve_tu_dong: 500000   # số này IN TRÊN WEB

# ---------------------------------------------------------------------
dich_vu: []
`

func dungBangGiaThu(t *testing.T) {
	t.Helper()
	cuRoot, cuCFG, cuGIA := Root, CFG, GIA
	t.Cleanup(func() { Root, CFG, GIA = cuRoot, cuCFG, cuGIA })

	Root = t.TempDir()
	CFG.DuongDan.BangGia = "vanhanh/bang-gia.yaml"
	if err := os.MkdirAll(filepath.Join(Root, "vanhanh"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fileBangGia(), []byte(yamlThu), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal([]byte(yamlThu), &GIA); err != nil {
		t.Fatal(err)
	}
}

// Khoá trong nguongCo phải khớp một-một với khoá trong yaml. Thiếu thì ô nhập
// ghi vào hư không; thừa thì có ngưỡng không ai sửa được mà cũng không ai biết.
func TestNguongDuKhoa(t *testing.T) {
	dungBangGiaThu(t)

	var kho struct {
		Nguong map[string]float64 `yaml:"nguong"`
	}
	if err := yaml.Unmarshal([]byte(yamlThu), &kho); err != nil {
		t.Fatal(err)
	}
	if len(kho.Nguong) != len(nguongCo) {
		t.Errorf("yaml có %d ngưỡng, trang quản trị khai %d", len(kho.Nguong), len(nguongCo))
	}
	for _, o := range nguongCo {
		v, co := kho.Nguong[o.Khoa]
		if !co {
			t.Errorf("khoá %s không có trong yaml", o.Khoa)
			continue
		}
		// layNguong là chỗ nối khoá yaml với trường Go — sai ở đây thì ô nhập
		// hiện số của ngưỡng khác, mà nhìn giao diện không thấy gì bất thường.
		if got := layNguong(GIA.Nguong, o.Khoa); got != v {
			t.Errorf("%s: layNguong ra %v, yaml ghi %v", o.Khoa, got, v)
		}
	}
}

func TestSuaNguongGiuComment(t *testing.T) {
	dungBangGiaThu(t)

	doi, err := SuaNguong(map[string]string{
		"tang_khoi_luong_toi_da_g": "3,5",      // dấu phẩy
		"free_ship_ve_tu_dong":     "600.000đ", // có dấu chấm và chữ đ
		"ship_ca_tu_choi_dong":     "30000",    // không đổi
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(doi) != 2 {
		t.Fatalf("báo đổi %d ô, muốn 2: %v", len(doi), doi)
	}

	b, err := os.ReadFile(fileBangGia())
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, phai := range []string{
		"# Lý do: swing weight tỉ lệ m*r^2.",
		"# Lãi gộp dưới mức này -> cảnh báo",
		"# ---------------------------------------------------------------------",
		"  tang_khoi_luong_toi_da_g: 3.5",
		"  free_ship_ve_tu_dong: 600000  # số này IN TRÊN WEB",
		"  ship_ca_tu_choi_dong: 30000",
	} {
		if !strings.Contains(s, phai) {
			t.Errorf("thiếu trong file sau khi ghi: %q", phai)
		}
	}
	if strings.Contains(s, "3.0") {
		t.Error("giá trị cũ 3.0 vẫn còn trong file")
	}
	// Sửa xong phải có hiệu lực ngay, không đợi khởi động lại.
	if NguongHienTai().TangKhoiLuongToiDaG != 3.5 || NguongHienTai().FreeShipVeTuDong != 600000 {
		t.Errorf("GIA chưa cập nhật: %+v", NguongHienTai())
	}
}

func TestSuaNguongChanSoBay(t *testing.T) {
	dungBangGiaThu(t)
	truoc, _ := os.ReadFile(fileBangGia())

	cac := []struct {
		ten string
		moi map[string]string
	}{
		{"chữ", map[string]string{"tang_khoi_luong_toi_da_g": "ba gam"}},
		{"âm", map[string]string{"tang_khoi_luong_toi_da_g": "-1"}},
		{"quá to", map[string]string{"tang_khoi_luong_toi_da_g": "500"}},
		{"tiền quá to", map[string]string{"free_ship_ve_tu_dong": "999000000"}},
		{"tiền không có số", map[string]string{"free_ship_ve_tu_dong": "miễn phí"}},
		{"bỏ trống", map[string]string{"free_ship_ve_tu_dong": "   "}},
	}
	for _, c := range cac {
		if _, err := SuaNguong(c.moi); err == nil {
			t.Errorf("%s: lẽ ra phải báo lỗi", c.ten)
		}
	}
	sau, _ := os.ReadFile(fileBangGia())
	if string(truoc) != string(sau) {
		t.Error("có lỗi mà file vẫn bị sửa")
	}
}

// Khối nguong: phải được khoanh đúng, không lấn sang khối khác — tên khoá
// trùng nhau ở hai khối là chuyện bình thường trong file cấu hình.
func TestKhoiNguongKhongLan(t *testing.T) {
	dong := strings.Split(yamlThu, "\n")
	dau, cuoi := khoiNguong(dong)
	if dau < 0 {
		t.Fatal("không thấy khối nguong")
	}
	if !strings.HasPrefix(dong[dau], "nguong:") {
		t.Errorf("dòng đầu khối là %q", dong[dau])
	}
	for i := dau + 1; i < cuoi; i++ {
		if strings.HasPrefix(dong[i], "dich_vu:") {
			t.Fatal("khối nguong lấn sang dich_vu")
		}
	}
	if timDongKhoa(dong, dau, cuoi, "giai_doan_hien_tai") >= 0 {
		t.Error("tìm thấy khoá ngoài khối, khoanh vùng hỏng")
	}
}

// Template gọi đúng tên trường không thì chỉ lúc dựng mới biết — mà lúc đó
// Kendy đã bấm vào trang rồi. Dựng thử ngay trong test.
func TestQtNguongDungDuoc(t *testing.T) {
	dungBangGiaThu(t)
	if err := InitTemplates(); err != nil {
		t.Fatal(err)
	}
	d := dlNguong{dlQt: dlQt{Chung: Chung{Trang: "nguong", CongKhai: true}}, Tep: "vanhanh/bang-gia.yaml"}
	for _, o := range DanhSachNguong() {
		if o.Web {
			d.Web = append(d.Web, o)
		} else {
			d.NoiBo = append(d.NoiBo, o)
		}
	}
	var b strings.Builder
	if err := tpl.ExecuteTemplate(&b, "qt-nguong.html", d); err != nil {
		t.Fatal(err)
	}
	s := b.String()
	for _, phai := range []string{
		`name="tang_khoi_luong_toi_da_g"`,
		`name="free_ship_ve_tu_dong"`,
		`value="3"`,      // 3.0 hiện thành 3
		`value="500000"`, // ô nhập là số trần, không dấu chấm
		"scp root@",      // nhắc kéo file về, vì CongKhai = true
	} {
		if !strings.Contains(s, phai) {
			t.Errorf("trang thiếu %q", phai)
		}
	}
	if strings.Count(s, `name="`) < 6 {
		t.Error("không đủ 6 ô nhập")
	}
}
