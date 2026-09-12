package core

import (
	"crypto/sha256"
	"encoding/hex"
	"sync"
)

// Vân tay nội dung cho mấy tệp icon nhúng sẵn trong binary.
//
// Vì sao cần: icon nhúng được phục vụ ở một đường dẫn cố định
// (/favicon.svg, /qt/icon.svg, /qt/icon.png) kèm Cache-Control 7 ngày, mà
// site chạy sau Cloudflare. Đổi hình rồi deploy thì gốc trả bản mới ngay,
// nhưng CDN vẫn nhả bản cũ cho tới hết bảy ngày đó — không có token API để
// xoá cache bằng tay. Đổi lần thay logo 2026-09-10 dính đúng chuyện này.
//
// Cách chữa: gắn vân tay nội dung vào query. Đổi hình -> đổi đường dẫn ->
// CDN coi là tệp khác, phải hỏi lại gốc. Không đổi hình thì đường dẫn y
// nguyên, cache vẫn ăn. Đây cũng đúng mẹo mà logo Kendy tải lên đang dùng
// (`/logo?v=<mtime>`), chỉ khác nguồn: tệp nhúng không có mtime nên băm nội
// dung.
//
// Query không đụng tới định tuyến: mux khớp theo đường dẫn, các hàm phục vụ
// không đọc query.

var (
	vtMu  sync.Mutex
	vtNho = map[string]string{}
)

// vanTay trả "?v=xxxxxxxx" cho một tệp trong uiFS, rỗng nếu không đọc được.
func vanTay(ten string) string {
	vtMu.Lock()
	defer vtMu.Unlock()
	if v, co := vtNho[ten]; co {
		return v
	}
	v := ""
	if b, err := uiFS.ReadFile(ten); err == nil {
		s := sha256.Sum256(b)
		v = "?v=" + hex.EncodeToString(s[:4])
	}
	vtNho[ten] = v
	return v
}

// duongIcon* — đường dẫn kèm vân tay của từng icon nhúng. Dùng ở mọi chỗ
// trỏ tới chúng: <link rel="icon">, manifest, vỏ service worker.
func duongFavicon() string   { return "/favicon.svg" + vanTay("ui/favicon-4d.svg") }
func duongIconQt() string    { return "/qt/icon.svg" + vanTay("ui/icon-qt.svg") }
func duongIconQtPNG() string { return "/qt/icon.png" + vanTay("ui/icon-qt.png") }
func duongIconApp() string   { return "/icon.png" + vanTay("ui/icon-app.png") }
