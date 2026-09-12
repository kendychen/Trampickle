package core

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDuoiLogoTheoNoiDung(t *testing.T) {
	cac := []struct {
		ten  string
		dau  []byte
		muon string
	}{
		{"svg thẳng", []byte(`<svg xmlns="http://www.w3.org/2000/svg">`), ".svg"},
		{"svg có khai báo xml", []byte("<?xml version=\"1.0\"?>\n<svg>"), ".svg"},
		{"png", []byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0}, ".png"},
		{"ico", []byte{0x00, 0x00, 0x01, 0x00, 0x01, 0x00}, ".ico"},
		// Tên tệp nói .png cũng không cứu được: hàm chỉ đọc nội dung.
		{"tệp lạ", []byte("MZ\x90\x00 chuong trinh windows"), ""},
		{"rỗng", nil, ""},
	}
	for _, c := range cac {
		if got := duoiLogo(c.dau); got != c.muon {
			t.Errorf("%s: được %q, muốn %q", c.ten, got, c.muon)
		}
	}
}

func TestNapLogoVaXoa(t *testing.T) {
	cu := Root
	Root = t.TempDir()
	defer func() { Root = cu; NapLogo() }()

	// Chưa có thư mục: không lỗi, và trang quay về bản mặc định.
	if err := NapLogo(); err != nil {
		t.Fatalf("NapLogo khi chưa có thư mục: %v", err)
	}
	if logoURL() != "" {
		t.Errorf("chưa tải gì mà logoURL() = %q", logoURL())
	}
	if !strings.HasPrefix(iconURL(), "/favicon.svg?v=") || iconMIME() != "image/svg+xml" {
		t.Errorf("favicon mặc định sai: %q %q", iconURL(), iconMIME())
	}

	os.MkdirAll(thuMucLogo(), 0o755)
	os.WriteFile(filepath.Join(thuMucLogo(), "logo.png"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(thuMucLogo(), "bieu-tuong.ico"), []byte("x"), 0o644)
	// Tệp không phải logo lẫn trong thư mục thì bỏ qua, không làm hỏng gì.
	os.WriteFile(filepath.Join(thuMucLogo(), "ghi-chu.txt"), []byte("x"), 0o644)
	if err := NapLogo(); err != nil {
		t.Fatal(err)
	}
	if logoURL() == "" {
		t.Error("đã có logo.png mà logoURL() rỗng")
	}
	if iconMIME() != "image/x-icon" {
		t.Errorf("iconMIME() = %q, muốn image/x-icon", iconMIME())
	}

	// Đổi sang đuôi khác: bản cũ phải bị dọn, không thì lần quét sau lấy
	// nhầm tệp tuỳ thứ tự đọc thư mục.
	xoaLogoCu(loaiLogo)
	os.WriteFile(filepath.Join(thuMucLogo(), "logo.svg"), []byte("x"), 0o644)
	if err := NapLogo(); err != nil {
		t.Fatal(err)
	}
	ten, _ := tepLogo(loaiLogo)
	if ten != "logo.svg" {
		t.Errorf("sau khi đổi đuôi, tệp đang dùng là %q", ten)
	}

	xoaLogoCu(loaiIcon)
	if err := NapLogo(); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(iconURL(), "/favicon.svg?v=") {
		t.Errorf("bỏ tệp rồi mà iconURL() = %q", iconURL())
	}
}
