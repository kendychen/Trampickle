package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Gõ sai một khoá thì chỗ đó trên trang thành khoảng trắng — không lỗi, không
// báo gì, chỉ là câu chữ biến mất và không ai biết cho tới khi khách hỏi. Nên
// khoá phải kiểm bằng máy.
var reGoiND = regexp.MustCompile(`\{\{\s*nd[dmx]?\s+"([^"]+)"`)

func TestKhoaNDCoDu(t *testing.T) {
	tep, err := fs.Glob(uiFS, "ui/*.html")
	if err != nil {
		t.Fatal(err)
	}
	dung := 0
	for _, p := range tep {
		b, err := fs.ReadFile(uiFS, p)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range reGoiND.FindAllStringSubmatch(string(b), -1) {
			dung++
			if _, co := ndMac[m[1]]; !co {
				t.Errorf("%s gọi khoá %q mà CayND không có", p, m[1])
			}
		}
	}
	if dung == 0 {
		t.Fatal("không thấy lời gọi {{nd}} nào — regex hỏng hoặc template chưa chuyển")
	}
	t.Logf("%d lời gọi, %d khoá khai trong cây", dung, len(ndMac))
}

// Khoá khai ra mà không template nào gọi thì ô nhập trong admin sửa xong
// chẳng thấy gì đổi trên trang. Đây là lỗi lặng y như trên, chỉ ngược chiều.
func TestKhoaNDKhongThua(t *testing.T) {
	tep, _ := fs.Glob(uiFS, "ui/*.html")
	goi := map[string]bool{}
	for _, p := range tep {
		b, _ := fs.ReadFile(uiFS, p)
		for _, m := range reGoiND.FindAllStringSubmatch(string(b), -1) {
			goi[m[1]] = true
		}
	}
	// Vài khoá do Go đọc thẳng chứ không qua template — lối tắt trong bản kê
	// khai app chẳng hạn, thứ đi vào JSON chứ không vào HTML. Không quét ở đây
	// thì chúng bị coi là thừa và test đỏ oan.
	goGoi := regexp.MustCompile(`ND\("([a-z0-9.\-]+)"\)`)
	tepGo, _ := filepath.Glob("*.go")
	for _, p := range tepGo {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, m := range goGoi.FindAllStringSubmatch(string(b), -1) {
			goi[m[1]] = true
		}
	}
	for _, tr := range CayND {
		for _, n := range tr.Nhom {
			for _, m := range n.Muc {
				if !goi[m.Khoa] {
					t.Errorf("khoá %q khai trong CayND mà không template nào gọi", m.Khoa)
				}
			}
		}
	}
}

func TestDatNDXoaKhiBangMacDinh(t *testing.T) {
	khoa := CayND[0].Nhom[0].Muc[0].Khoa
	mac := NDMac(khoa)

	ndMu.Lock()
	ndSua[khoa] = "chữ khác"
	ndMu.Unlock()
	if !DaSuaND(khoa) || ND(khoa) != "chữ khác" {
		t.Fatal("đặt tay không ăn")
	}

	// Gõ lại đúng chữ mặc định thì phải coi như chưa đổi, không phải lưu một
	// bản sao y hệt: bản sao đó sẽ đè lên mặc định mới sau này.
	ndMu.Lock()
	if _, co := ndSua[khoa]; co {
		delete(ndSua, khoa)
	}
	ndSua[khoa] = mac + "  "
	ndMu.Unlock()

	ndMu.Lock()
	v := chuanND(ndSua[khoa])
	if v == ndMac[khoa] {
		delete(ndSua, khoa)
	}
	ndMu.Unlock()

	if DaSuaND(khoa) {
		t.Errorf("chữ bằng mặc định (thừa khoảng trắng) vẫn bị coi là đã đổi")
	}
	if ND(khoa) != mac {
		t.Errorf("ND(%q) = %q, muốn %q", khoa, ND(khoa), mac)
	}
}

// Vòng tròn thật: sửa -> ghi ra yaml -> quên hết -> nạp lại. Chỗ này chưa
// hỏng bao giờ nhưng nếu hỏng thì hỏng lặng: Kendy lưu xong thấy trang đổi,
// tới lúc khởi động lại service mới phát hiện chữ quay về mặc định.
func TestNDGhiRoiNapLai(t *testing.T) {
	cu := Root
	Root = t.TempDir()
	defer func() {
		Root = cu
		ndMu.Lock()
		ndSua, ndLa = map[string]string{}, map[string]string{}
		ndMu.Unlock()
	}()
	if err := os.MkdirAll(P("data"), 0o755); err != nil {
		t.Fatal(err)
	}

	const khoa = "trangchu.hero.h1"
	if err := DatND(map[string]string{khoa: "Chữ Kendy tự gõ"}); err != nil {
		t.Fatal(err)
	}

	// Một khoá của bản sau: bản này chưa biết nó, nhưng ghi đè lên file thì
	// không được phép nuốt mất.
	b, err := os.ReadFile(P("data/noi-dung.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(P("data/noi-dung.yaml"), append(b, []byte("trang.moi.chua-co: chữ của bản sau\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	ndMu.Lock()
	ndSua = map[string]string{}
	ndMu.Unlock()
	if err := NapND(); err != nil {
		t.Fatal(err)
	}
	if ND(khoa) != "Chữ Kendy tự gõ" {
		t.Errorf("nạp lại ra %q", ND(khoa))
	}

	if err := KhoiPhucND(khoa); err != nil {
		t.Fatal(err)
	}
	if ND(khoa) != NDMac(khoa) {
		t.Errorf("khôi phục xong vẫn ra %q", ND(khoa))
	}
	b, err = os.ReadFile(P("data/noi-dung.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), khoa) {
		t.Errorf("khoá đã khôi phục vẫn còn trong file")
	}
	if !strings.Contains(string(b), "trang.moi.chua-co") {
		t.Errorf("khoá lạ bị nuốt mất khi ghi lại file")
	}
}
