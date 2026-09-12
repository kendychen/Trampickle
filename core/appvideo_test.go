package core

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Mấy byte đầu của một tệp mp4 thật: kích thước hộp rồi tới "ftyp". Phần thân
// sau đó không cần đúng chuẩn — chỗ này chỉ kiểm luật nhận dạng và đường phục
// vụ, không kiểm bộ giải mã của trình duyệt.
var dauMP4 = []byte{0, 0, 0, 0x20, 'f', 't', 'y', 'p', 'i', 's', 'o', 'm'}

func dungVideoApp(t *testing.T) {
	t.Helper()
	cuRoot := Root
	cuCH := AppCauHinhHienTai()
	t.Cleanup(func() {
		Root = cuRoot
		appCHMu.Lock()
		appCH = cuCH
		appCHMu.Unlock()
	})
	Root = t.TempDir()
	appCHMu.Lock()
	appCH = MacDinhApp()
	appCHMu.Unlock()
}

func TestDuoiVideoChiNhanMP4(t *testing.T) {
	if duoiVideo(dauMP4) != ".mp4" {
		t.Error("tệp mp4 thật mà không nhận")
	}
	// WebM bị từ chối là CỐ Ý, không phải sót: Safari trên iPhone không mở
	// được nó. Test này giữ lời hứa ấy khỏi bị ai đó nới ra cho tiện.
	webm := []byte{0x1A, 0x45, 0xDF, 0xA3, 1, 2, 3, 4, 5, 6, 7, 8}
	if duoiVideo(webm) != "" {
		t.Error("webm lọt qua")
	}
	if duoiVideo([]byte("GIF89a")) != "" {
		t.Error("tệp ngắn/không phải video lọt qua")
	}
}

func TestLuuVideoAppRoiXoa(t *testing.T) {
	dungVideoApp(t)

	ten, err := LuuVideoApp(dauMP4, bytes.NewReader([]byte("phần thân")))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(ten, ".mp4") {
		t.Fatalf("tên tệp không có đuôi mp4: %s", ten)
	}
	if !coVideoApp(ten) {
		t.Fatal("lưu xong mà coVideoApp nói không có")
	}
	b, err := os.ReadFile(filepath.Join(thuMucVideoApp(), ten))
	if err != nil {
		t.Fatal(err)
	}
	// 12 byte đầu đọc riêng để nhận dạng rồi phải ghi lại vào tệp — quên chỗ
	// này là mọi clip tải lên đều mất khúc đầu và không máy nào mở được.
	if !bytes.Equal(b, append(append([]byte{}, dauMP4...), []byte("phần thân")...)) {
		t.Errorf("nội dung tệp sai: %q", b)
	}

	XoaVideoApp(ten)
	if coVideoApp(ten) {
		t.Error("xoá rồi mà vẫn còn")
	}
}

func TestLuuVideoAppTuChoiTepLa(t *testing.T) {
	dungVideoApp(t)
	if _, err := LuuVideoApp([]byte("khong phai video"), bytes.NewReader(nil)); err == nil {
		t.Error("nhận cả tệp không phải mp4")
	}
}

// Cấu hình sửa tay trên máy chủ là chuyện có thật. Tên bậy phải rơi về rỗng
// chứ không được đi tiếp thành đường dẫn.
func TestChuanAppCHLocTenVideo(t *testing.T) {
	for _, ten := range []string{"../../etc/passwd", "clip.mp4.exe", "anh.jpg", "a/b.mp4"} {
		c := chuanAppCH(AppCauHinh{BannerVideo: ten})
		if c.BannerVideo != "" {
			t.Errorf("%q lọt qua thành %q", ten, c.BannerVideo)
		}
	}
}

// Tệp còn trên đĩa nhưng không phải tệp đang treo thì cũng không được phục vụ:
// /app-video/ không phải một thư mục mở.
func TestAppVideoChiPhucVuTepDangTreo(t *testing.T) {
	dungVideoApp(t)

	dung, err := LuuVideoApp(dauMP4, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	bo, err := LuuVideoApp(dauMP4, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	appCHMu.Lock()
	appCH.BannerVideo = dung
	appCHMu.Unlock()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /app-video/{ten}", hAppVideo)

	goi := func(ten string) int {
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, httptest.NewRequest("GET", "/app-video/"+ten, nil))
		return w.Code
	}
	if got := goi(dung); got != 200 {
		t.Errorf("tệp đang treo trả %d", got)
	}
	if got := goi(bo); got != 404 {
		t.Errorf("tệp không treo trả %d, đáng ra 404", got)
	}
	if got := goi("khong-co.mp4"); got != 404 {
		t.Errorf("tệp không tồn tại trả %d", got)
	}
}

// BannerVideoURL là thứ app.html hỏi. Cấu hình trỏ vào tệp đã bị xoá tay thì
// nó phải trả rỗng, để banner rơi về ảnh chứ không treo thẻ <video> vào 404.
func TestBannerVideoURLTheoTepThat(t *testing.T) {
	dungVideoApp(t)

	ten, err := LuuVideoApp(dauMP4, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	c := AppCauHinh{BannerVideo: ten}
	if got := c.BannerVideoURL(); got != "/app-video/"+ten {
		t.Errorf("URL sai: %q", got)
	}
	XoaVideoApp(ten)
	if got := c.BannerVideoURL(); got != "" {
		t.Errorf("tệp mất rồi mà URL vẫn là %q", got)
	}
}

// Đường tải video là multipart — token CSRF nằm trong thân, nên lớp bọc chung
// phải được cho phép đi qua. Quên ghi vào danh sách là nút tải lên chết 403.
func TestDuongVideoDuocPhepMultipart(t *testing.T) {
	if !multipartChoPhep("/qt/app/video") {
		t.Error("/qt/app/video chưa nằm trong danh sách multipart")
	}
}

// Trang /qt/app phải dựng HẾT. html/template ghi ra tới đâu lỗi tới đó rồi mới
// dừng, nên một tên trường viết sai trong khối video không trả 500 — nó trả một
// trang cụt nửa chừng, và mắt thường nhìn qua vẫn thấy "trang vẫn lên".
func TestTrangQtAppDungHetKhoiVideo(t *testing.T) {
	dungVideoApp(t)
	if err := InitTemplates(); err != nil {
		t.Fatal(err)
	}
	ten, err := LuuVideoApp(dauMP4, bytes.NewReader(nil))
	if err != nil {
		t.Fatal(err)
	}
	appCHMu.Lock()
	appCH.BannerVideo = ten
	appCHMu.Unlock()

	w := httptest.NewRecorder()
	hQtApp(w, httptest.NewRequest("GET", "/qt/app", nil))
	if w.Code != 200 {
		t.Fatalf("code %d", w.Code)
	}
	b := w.Body.String()
	for _, can := range []string{"/app-video/" + ten, "/qt/app/video-xoa", "</html>"} {
		if !strings.Contains(b, can) {
			t.Errorf("trang thiếu %q (dài %d byte)", can, len(b))
		}
	}
}
