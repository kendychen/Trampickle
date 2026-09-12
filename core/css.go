package core

// CSS đi ra thành một tệp riêng, tên mang mã băm của chính nội dung.
//
// Trước đây toàn bộ khối style nằm nội tuyến trong mỗi trang: 200 KB chữ CSS
// (phần lớn là phông Roboto nhúng base64) tải lại từ đầu ở MỌI lượt xem, vì
// HTML không cache được — mỗi trang mang một nonce CSP và một token CSRF
// riêng, cache HTML là phát nhầm token của người này cho người khác.
//
// Tách ra thì hai thứ tách theo: HTML vẫn no-cache và vẫn nhỏ, còn CSS nằm ở
// một URL bất biến, trình duyệt tải một lần rồi thôi. Tên tệp mang mã băm nên
// không cần đi xoá cache ở đâu cả: sửa CSS là mã băm đổi, URL đổi, trang gọi
// tệp mới. Cái URL cũ có ai giữ trong cache một năm cũng không sao — nội dung
// ở đó đúng là nội dung đã băm ra tên ấy.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
)

var (
	cssNoiDung []byte
	cssDuong   string // /tinh/tp-<băm>.css
)

// dungCSS chạy một lần trong InitTemplates, sau khi mẫu đã nạp. Dựng sẵn chứ
// không dựng theo yêu cầu vì mẫu "css" không đọc dữ liệu nào của trang — kiểm
// lại bằng: trong ui/css.html chỉ có đúng hai action, {{define}} và {{end}}.
// Ngày nào cần chèn dữ liệu vào CSS thì chỗ này sai, và nó sai to: mọi khách
// sẽ nhận CSS của khách đầu tiên.
func dungCSS() error {
	var b bytes.Buffer
	if err := tpl.ExecuteTemplate(&b, "css", nil); err != nil {
		return err
	}
	s := b.String()
	// Bỏ cặp <style> bọc ngoài. Trong HTML nó là bắt buộc, trong tệp .css nó
	// là lỗi cú pháp — trình duyệt gặp dòng đó thì bỏ luôn khối đầu tiên.
	i := strings.Index(s, "<style>")
	j := strings.LastIndex(s, "</style>")
	if i < 0 || j <= i {
		return fmt.Errorf("css: không thấy cặp <style> trong ui/css.html")
	}
	cssNoiDung = []byte(strings.TrimSpace(s[i+len("<style>") : j]))
	h := sha256.Sum256(cssNoiDung)
	// 16 chữ số hex là 64 bit — đủ để hai bản CSS khác nhau không bao giờ
	// trùng tên, mà vẫn đọc được bằng mắt lúc xem log.
	cssDuong = "/tinh/tp-" + hex.EncodeToString(h[:])[:16] + ".css"
	return nil
}

// cssURL cho mẫu. Rỗng thì trang ra không có CSS — để nó rỗng còn dễ thấy hơn
// là âm thầm quay về nội tuyến, vì lỗi kiểu đó lẫn vào trang trông vẫn chạy.
func cssURL() string { return cssDuong }

func hCSS(w http.ResponseWriter, r *http.Request) {
	// So cả đường dẫn: chỉ đúng mã băm hiện tại mới được trả nội dung. URL
	// mang mã băm cũ trả 404 chứ không trả bản mới — bản mới có nội dung khác
	// mà lại nằm ở một tên đã hứa là bất biến thì mới là sai.
	if cssDuong == "" || r.URL.Path != cssDuong {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(cssNoiDung)
}
