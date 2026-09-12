// Lưu ảnh vào thư mục của một đơn. Trước đây nằm trong hQtTaiAnh; tách ra
// khi cửa hàng cần đúng vòng lặp ấy cho ảnh khách gửi lúc đặt. Hai bản sao
// của một vòng lặp lọc tệp là hai chỗ để quên sửa.
package core

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// luuAnhVaoDon ghi các tệp hợp lệ vào thư mục ảnh của đơn và nối tên vào
// don.Anh. Trả số ảnh đã lưu. KHÔNG gọi LuuDon — chỗ gọi tự quyết lúc nào ghi
// đơn xuống đĩa.
//
// Tệp không nhận ra được là ảnh thì BỎ IM LẶNG. Người gửi thường kèm cả ảnh
// chụp màn hình lẫn video; báo lỗi cả biểu mẫu chỉ vì một tệp lạ là chặn mất
// một đơn thật.
func luuAnhVaoDon(don *Don, files []*multipart.FileHeader, nhan string) int {
	if nhan != "sau" {
		nhan = "truoc"
	}
	them := 0
	for _, fh := range files {
		if len(don.Anh) >= anhToiDaMoiDon {
			break
		}
		f, err := fh.Open()
		if err != nil {
			continue
		}
		dau := make([]byte, 16)
		n, _ := io.ReadFull(f, dau)
		duoi := duoiAnh(dau[:n])
		if duoi == "" {
			f.Close()
			continue
		}
		if err := os.MkdirAll(thuMucAnh(don.Ma), 0o755); err != nil {
			f.Close()
			continue
		}
		ten := fmt.Sprintf("%s-%s-%s%s", nhan, time.Now().Format("150405"), maNgauNhien(3), duoi)
		out, err := os.Create(filepath.Join(thuMucAnh(don.Ma), ten))
		if err != nil {
			f.Close()
			continue
		}
		out.Write(dau[:n])
		io.Copy(out, io.LimitReader(f, anhToiDaByte))
		out.Close()
		f.Close()
		don.Anh = append(don.Anh, ten)
		them++
	}
	return them
}

// chepTepYeuCauSangDon chép ảnh khách gửi lúc đặt trên web sang thư mục ảnh
// của đơn vừa mở, và nối tên vào don.Anh (nhãn "truoc").
//
// Vì sao chép hẳn chứ không trỏ sang thư mục yêu cầu: không chép thì thợ phải
// mở hai tab để vừa nhìn chỗ nứt vừa gõ chẩn đoán, và ngày nào đó dọn thư mục
// yêu cầu là đơn mất sạch ảnh gốc. Vài trăm KB nhân đôi rẻ hơn nhiều.
//
// Chỉ chép ảnh tĩnh .jpg/.png/.webp — đúng ba đuôi duoiAnh nhận. Video và
// .heic bỏ lại bên yêu cầu: khung ảnh của đơn vẽ bằng thẻ <img>, nhét .webm
// vào đó chỉ ra một ô vỡ.
//
// KHÔNG gọi LuuDon: chỗ gọi tự quyết lúc nào ghi xuống đĩa. Mọi lỗi đều bỏ
// qua im lặng — thiếu ảnh thì thợ hỏi lại khách, còn chặn việc mở đơn vì một
// tệp hỏng thì mất cả đơn.
func chepTepYeuCauSangDon(yc yeuCauKhach, don *Don) int {
	if len(yc.Tep) == 0 {
		return 0
	}
	nguon := thuMucTepYeuCau(yc.Ma)
	dich := thuMucAnh(don.Ma)
	them := 0
	for _, ten := range yc.Tep {
		if len(don.Anh) >= anhToiDaMoiDon {
			break
		}
		// filepath.Base chặn tên kiểu "../../config.yaml" lỡ lọt vào JSON.
		ten = filepath.Base(strings.TrimSpace(ten))
		if ten == "" || ten == "." || ten == ".." {
			continue
		}
		duoi := strings.ToLower(filepath.Ext(ten))
		if duoi != ".jpg" && duoi != ".png" && duoi != ".webp" {
			continue
		}
		src, err := os.Open(filepath.Join(nguon, ten))
		if err != nil {
			continue
		}
		if err := os.MkdirAll(dich, 0o755); err != nil {
			src.Close()
			continue
		}
		tenMoi := fmt.Sprintf("truoc-%s-%s%s", time.Now().Format("150405"), maNgauNhien(3), duoi)
		out, err := os.Create(filepath.Join(dich, tenMoi))
		if err != nil {
			src.Close()
			continue
		}
		_, err = io.Copy(out, io.LimitReader(src, anhToiDaByte))
		out.Close()
		src.Close()
		if err != nil {
			os.Remove(filepath.Join(dich, tenMoi))
			continue
		}
		don.Anh = append(don.Anh, tenMoi)
		them++
	}
	return them
}
