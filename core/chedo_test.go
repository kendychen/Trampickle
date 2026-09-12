package core

import (
	"os"
	"path/filepath"
	"testing"
)

// gocTam đổi thư mục gốc dữ liệu sang thư mục tạm rồi trả lại như cũ.
func gocTam(t *testing.T) string {
	t.Helper()
	cu := Root
	Root = t.TempDir()
	if err := os.MkdirAll(P("data"), 0o755); err != nil {
		t.Fatal(err)
	}
	d := Root
	t.Cleanup(func() { Root = cu; _ = NapCheDo() })
	return d
}

func TestCheDoMacDinhLaTaiXuong(t *testing.T) {
	gocTam(t)
	if err := NapCheDo(); err != nil {
		t.Fatalf("NapCheDo: %v", err)
	}
	if got := CheDoHienTai(); got != CheDoTaiXuong {
		t.Fatalf("chưa có file thì phải là %q, nhận %q", CheDoTaiXuong, got)
	}
	if LaOnline() {
		t.Fatal("LaOnline phải false khi đang tại xưởng")
	}
}

func TestDatCheDoGhiRoiNapLai(t *testing.T) {
	gocTam(t)
	if err := DatCheDo(CheDoOnline); err != nil {
		t.Fatalf("DatCheDo: %v", err)
	}
	if !LaOnline() {
		t.Fatal("đặt online xong LaOnline phải true")
	}
	if err := NapCheDo(); err != nil {
		t.Fatalf("NapCheDo lần 2: %v", err)
	}
	if got := CheDoHienTai(); got != CheDoOnline {
		t.Fatalf("nạp lại phải ra %q, nhận %q", CheDoOnline, got)
	}
}

func TestDatCheDoLaBiTuChoi(t *testing.T) {
	gocTam(t)
	if err := DatCheDo("bay-gio"); err == nil {
		t.Fatal("chế độ lạ phải bị từ chối")
	}
	if got := CheDoHienTai(); got != CheDoTaiXuong {
		t.Fatalf("từ chối xong phải giữ %q, nhận %q", CheDoTaiXuong, got)
	}
}

// File cũ chỉ có khoá chết "theme: tpic" — đọc nó không được nổ, và phải ra
// tại xưởng.
func TestFileCuChiCoKhoaTheme(t *testing.T) {
	d := gocTam(t)
	if err := os.WriteFile(filepath.Join(d, "data", "giao-dien.yaml"), []byte("theme: tpic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := NapCheDo(); err != nil {
		t.Fatalf("file cũ không phải lỗi: %v", err)
	}
	if got := CheDoHienTai(); got != CheDoTaiXuong {
		t.Fatalf("phải ra %q, nhận %q", CheDoTaiXuong, got)
	}
}

// File hỏng trả lỗi để chỗ gọi ghi nhật ký, nhưng web vẫn lên ở tại xưởng.
func TestFileHongThiChayTaiXuong(t *testing.T) {
	d := gocTam(t)
	if err := os.WriteFile(filepath.Join(d, "data", "giao-dien.yaml"), []byte("che_do: [hong"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := NapCheDo(); err == nil {
		t.Fatal("file hỏng phải trả lỗi")
	}
	if got := CheDoHienTai(); got != CheDoTaiXuong {
		t.Fatalf("file hỏng vẫn phải chạy %q, nhận %q", CheDoTaiXuong, got)
	}
}

// --- Công tắc dịch vụ nổi bật ----------------------------------------

func TestNoiBatMacDinhBat(t *testing.T) {
	gocTam(t)
	if err := NapCheDo(); err != nil {
		t.Fatal(err)
	}
	if !DvNoiBatBat() {
		t.Fatal("chưa có file thì khung nổi bật phải bật")
	}
}

// Hai công tắc ở chung một file. Lưu cái này không được xoá cái kia — đó là
// cả lý do file được ghi lại trọn vẹn từ một chỗ.
func TestHaiCongTacKhongDeNhau(t *testing.T) {
	gocTam(t)
	if err := DatCheDo(CheDoOnline); err != nil {
		t.Fatal(err)
	}
	if err := DatDvNoiBat(false); err != nil {
		t.Fatal(err)
	}
	if err := NapCheDo(); err != nil {
		t.Fatal(err)
	}
	if !LaOnline() {
		t.Error("tắt nổi bật làm mất luôn chế độ online")
	}
	if DvNoiBatBat() {
		t.Error("tắt nổi bật xong nạp lại vẫn thấy bật")
	}

	if err := DatCheDo(CheDoTaiXuong); err != nil {
		t.Fatal(err)
	}
	if err := NapCheDo(); err != nil {
		t.Fatal(err)
	}
	if DvNoiBatBat() {
		t.Error("đổi chế độ làm khung nổi bật bật lại")
	}
	if LaOnline() {
		t.Error("chế độ không đổi được")
	}
}
