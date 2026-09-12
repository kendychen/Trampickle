package core

import (
	"os"
	"path/filepath"
	"testing"
)

// Đổ một trang quản lý ra file HTML tĩnh để soi bằng Chrome headless — luật
// "tự nhìn hình trước khi deploy" cần cái này. Dữ liệu lấy từ data/vot.yaml
// thật, không phải fixture, nên hình ra đúng cây thật.
//
//	DUMP=/tmp/qtvot.html go test ./core/ -run TestXemTrangVot
//
// Bỏ qua khi không có DUMP, để CI không ghi rác.
func TestXemTrangVot(t *testing.T) {
	ra := os.Getenv("DUMP")
	if ra == "" {
		t.Skip("đặt DUMP=<đường dẫn> để đổ trang ra file")
	}
	mux, ck := dungTrangThu(t, VaiTroTho)

	cu, cuKho, cuMtime := Root, votKho, votMtime
	Root = filepath.Join("..", "..")
	votKho, votMtime = nil, 0
	t.Cleanup(func() {
		votMu.Lock()
		Root, votKho, votMtime = cu, cuKho, cuMtime
		votMu.Unlock()
	})

	s := moTrang(t, mux, ck, "/qt/vot")
	if err := os.WriteFile(ra, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("viết %d byte vào %s", len(s), ra)
}
