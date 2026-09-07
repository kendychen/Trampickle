package core

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func dungNhatKyThu(t *testing.T) {
	t.Helper()
	cu := Root
	Root = t.TempDir()
	t.Cleanup(func() { Root = cu })
}

func TestGhiRoiDocLaiNhatKy(t *testing.T) {
	dungNhatKyThu(t)

	GhiNhatKy(MucNhatKy{Ai: "kendy", IP: "1.2.3.4", Viec: "dang-nhap", KetQua: "ok"})
	GhiNhatKy(MucNhatKy{Ai: "tho1", IP: "1.2.3.5", Viec: "sua-don", DoiTuong: "DH-9", KetQua: "ok"})

	ds := DocNhatKy("", 0)
	if len(ds) != 2 {
		t.Fatalf("đọc được %d mục, đợi 2", len(ds))
	}
	// Mới nhất trước.
	if ds[0].Ai != "tho1" {
		t.Errorf("thứ tự sai, mục đầu là %q", ds[0].Ai)
	}
	if ds[0].DoiTuong != "DH-9" {
		t.Errorf("mất trường DoiTuong: %+v", ds[0])
	}
	if ds[1].Luc == "" {
		t.Error("không tự điền thời điểm")
	}
}

// Nhật ký là file đọc được. Mật khẩu lọt vào đây là tự tạo thêm một chỗ rò.
func TestLocTruongCamBoMatKhau(t *testing.T) {
	v := url.Values{
		"ten":          {"kendy"},
		"mat_khau":     {"sieu-bi-mat"},
		"mat_khau_moi": {"cung-bi-mat"},
		"_csrf":        {"abc123"},
		"ma_totp":      {"050471"},
		"gemini":       {"AIzaXXXX"},
		"ghi_chu":      {"vợt gãy cán"},
	}
	s := locTruongCam(v)
	for _, cam := range []string{"sieu-bi-mat", "cung-bi-mat", "abc123", "050471", "AIzaXXXX"} {
		if strings.Contains(s, cam) {
			t.Errorf("lọt bí mật %q vào nhật ký: %s", cam, s)
		}
	}
	if !strings.Contains(s, "ghi_chu=vợt gãy cán") || !strings.Contains(s, "ten=kendy") {
		t.Errorf("cắt nhầm trường bình thường: %s", s)
	}
}

// thang đi thẳng từ query string vào tên file.
func TestDocNhatKyChanDuongDanBay(t *testing.T) {
	dungNhatKyThu(t)
	if err := os.WriteFile(filepath.Join(Root, "bi-mat.yaml"), []byte(`{"ai":"x"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, xau := range []string{"../bi-mat", "..", "2026-13", "abc", "2026-01/../../etc/passwd"} {
		if ds := DocNhatKy(xau, 0); ds != nil {
			t.Errorf("tháng %q lẽ ra bị chặn, lại đọc được %d mục", xau, len(ds))
		}
	}
}

func TestNhatKyGioiHanSoMuc(t *testing.T) {
	dungNhatKyThu(t)
	for i := 0; i < 10; i++ {
		GhiNhatKy(MucNhatKy{Ai: "kendy", Viec: "xem", KetQua: "ok"})
	}
	if ds := DocNhatKy("", 3); len(ds) != 3 {
		t.Errorf("giới hạn 3 mà trả %d mục", len(ds))
	}
}

func TestNhatKyQuyenFile(t *testing.T) {
	dungNhatKyThu(t)
	GhiNhatKy(MucNhatKy{Ai: "kendy", Viec: "xem", KetQua: "ok"})

	ten := filepath.Join(thuMucNhatKy(), time.Now().Format("2006-01")+".jsonl")
	st, err := os.Stat(ten)
	if err != nil {
		t.Fatalf("không thấy file nhật ký: %v", err)
	}
	// Windows không có quyền POSIX; chỉ kiểm trên máy chủ thật.
	if os.PathSeparator == '/' && st.Mode().Perm() != 0o600 {
		t.Errorf("quyền file là %o, phải là 600", st.Mode().Perm())
	}
}
