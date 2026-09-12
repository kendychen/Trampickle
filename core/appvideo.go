package core

// Video nền cho banner màn hình app.
//
// KHÔNG dùng chung kho với ảnh hero. Ba lý do, cái nào cũng đủ để tách:
//
//   - Kho ảnh phục vụ ba nơi (băng hero trang chủ, dải ảnh Về chúng tôi,
//     banner app) và mọi nơi ấy đều render bằng <img> hoặc background-image.
//     Nhét một tệp mp4 vào danh sách ấy là mỗi chỗ đọc kho phải tự nhớ lọc nó
//     ra, và chỗ nào quên thì ra một ô ảnh vỡ giữa trang.
//   - Trần dung lượng khác hẳn: ảnh 8MB là rộng rãi, video 8MB là một clip
//     ngắn chất lượng vừa. Một con số không phục vụ được cả hai.
//   - Kho ảnh giữ tối đa 14 tấm và Kendy sắp thứ tự trong đó. Video thì chỉ có
//     đúng một cái đang treo — không có thứ tự nào để sắp.
//
// Nên: một thư mục riêng, ĐÚNG MỘT tệp trong đó. Tải cái mới lên là cái cũ bị
// xoá ngay, không tích lại thành đống clip không ai nhớ cái nào đang chạy.
//
// Tên tệp có chuỗi ngẫu nhiên, nên đổi video là đổi luôn đường dẫn — Cloudflare
// và trình duyệt khách không thể phục vụ lại bản cũ. Đây không phải chuyện lý
// thuyết: bộ icon từng bị đúng lỗi ấy.

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Trần 10MB. Banner app là thứ khách tải lại mỗi lần mở app từ icon, và phần
// lớn mở bằng 4G ngoài sân — 10MB đã là nặng, để rộng hơn nữa thì cái giá
// không nằm ở máy chủ mà nằm ở gói cước của khách. Chỗ nhập ở /qt/app nói rõ
// nên để dưới 3MB; con số này chỉ là hàng rào cuối.
const videoAppToiDaByte = 10 << 20

func thuMucVideoApp() string { return P("data/app-video") }

// duoiVideo — nhận diện theo nội dung tệp, không theo đuôi tên. Đuôi tên là
// thứ người gửi đặt; hộp ftyp ở byte 4..8 là thứ tệp thật sự là.
//
// Chỉ MP4. WebM nhẹ hơn thật, nhưng Safari trên iPhone không chạy được nó, mà
// khách của trạm thì một nửa dùng iPhone — nhận vào rồi để một nửa số khách
// nhìn khung đen còn tệ hơn là từ chối ngay lúc tải lên.
func duoiVideo(dau []byte) string {
	if len(dau) >= 12 && string(dau[4:8]) == "ftyp" {
		return ".mp4"
	}
	return ""
}

// LuuVideoApp ghi tệp mới rồi trả tên. KHÔNG tự xoá tệp cũ: người gọi phải
// gắn được tên mới vào cấu hình trước đã. Xoá trước mà lưu cấu hình hỏng thì
// banner mất video mà Kendy không hiểu vì sao.
func LuuVideoApp(dau []byte, than io.Reader) (string, error) {
	duoi := duoiVideo(dau)
	if duoi == "" {
		return "", errors.New("chỉ nhận video MP4")
	}
	if err := os.MkdirAll(thuMucVideoApp(), 0o755); err != nil {
		return "", fmt.Errorf("không tạo được thư mục video: %w", err)
	}
	ten := fmt.Sprintf("app-%s-%s%s", time.Now().Format("150405"), maNgauNhien(3), duoi)
	duong := filepath.Join(thuMucVideoApp(), ten)
	out, err := os.Create(duong)
	if err != nil {
		return "", fmt.Errorf("không ghi được tệp: %w", err)
	}
	if _, err := out.Write(dau); err != nil {
		out.Close()
		os.Remove(duong)
		return "", err
	}
	// LimitReader chứ không tin vào MaxBytesReader ở ngoài: ở đây mới biết
	// chắc bao nhiêu byte thực sự chạm vào đĩa.
	if _, err := io.Copy(out, io.LimitReader(than, videoAppToiDaByte)); err != nil {
		out.Close()
		os.Remove(duong)
		return "", err
	}
	if err := out.Close(); err != nil {
		os.Remove(duong)
		return "", err
	}
	return ten, nil
}

// XoaVideoApp xoá đúng một tệp trong thư mục video. Tên không sạch thì không
// đụng vào đĩa — cùng luật với ảnh.
func XoaVideoApp(ten string) {
	if ten == "" || !tenAnhSach(ten) {
		return
	}
	os.Remove(filepath.Join(thuMucVideoApp(), ten))
}

// coVideoApp — tệp có thật trên đĩa không. Cấu hình ghi tên một tệp đã bị xoá
// tay trên máy chủ thì banner phải quay về ảnh/bản vẽ, chứ không phải treo một
// thẻ <video> trỏ vào 404.
func coVideoApp(ten string) bool {
	if ten == "" || !tenAnhSach(ten) {
		return false
	}
	st, err := os.Stat(filepath.Join(thuMucVideoApp(), ten))
	return err == nil && !st.IsDir()
}

// hAppVideo phục vụ video cho khách.
//
// Chỉ mở đúng cái tên đang ghi trong cấu hình, không ghép đường dẫn từ tham số
// URL — cùng luật với /anh-trang-chu/. Thư mục này chỉ có một tệp, nên "tên
// hợp lệ" và "tên đang treo" là một.
//
// http.ServeFile chứ không tự đọc: nó lo hộ Range request, mà video thì trình
// duyệt gần như luôn hỏi theo khúc.
func hAppVideo(w http.ResponseWriter, r *http.Request) {
	ten := r.PathValue("ten")
	if !tenAnhSach(ten) || !strings.HasSuffix(ten, ".mp4") {
		http.NotFound(w, r)
		return
	}
	if AppCauHinhHienTai().BannerVideo != ten {
		http.NotFound(w, r)
		return
	}
	duong := filepath.Join(thuMucVideoApp(), ten)
	if _, err := os.Stat(duong); errors.Is(err, fs.ErrNotExist) {
		http.NotFound(w, r)
		return
	}
	// Tên có chuỗi ngẫu nhiên, đổi video là đổi tên — cache dài thoải mái.
	w.Header().Set("Cache-Control", "public, max-age=604800")
	http.ServeFile(w, r, duong)
}
