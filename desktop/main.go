// App desktop cho Kendy — cùng một web server, đóng trong cửa sổ riêng.
//
// Không có API riêng, không có template riêng, không có bảng giá riêng.
// Cửa sổ này chỉ là một trình duyệt trỏ vào core.NewMux(false) — đúng cái
// mux mà bản chạy trên máy nhà dùng. Sửa web là app đổi theo, không có
// chuyện hai bên lệch nhau.
//
// Vì sao là module riêng: Wails kéo theo cả đống thư viện GUI. Bản chạy
// trên VPS không cần biết chúng tồn tại, và `go build ./...` bên module
// tramvot phải nhẹ như cũ.
//
//	cd tramvot/desktop && wails dev     chạy thử, sửa là nạp lại
//	cd tramvot/desktop && wails build   ra build/bin/TramVot.exe
package main

import (
	"fmt"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/options/windows"

	"tramvot/core"
)

func main() {
	// Nạp đúng thứ tự như bản CLI: cấu hình → tài khoản → đơn → template.
	if err := core.LoadConfig(); err != nil {
		thoat("Lỗi cấu hình", err)
	}
	if err := core.NapNguoiDung(); err != nil {
		thoat("Lỗi đọc tài khoản", err)
	}
	if err := core.NapDon(); err != nil {
		thoat("Lỗi đọc đơn hàng", err)
	}
	// Trước đơn một bước: "đã thu" của đơn cộng từ sổ tiền.
	if err := core.NapSoTien(); err != nil {
		thoat("Lỗi đọc sổ tiền", err)
	}
	if err := core.NapKho(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo kho:", err)
	}
	if err := core.NapDinhKy(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo khoản định kỳ:", err)
	}
	if err := core.NapBaiViet(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo bài viết:", err)
	}
	// Chữ đã sửa ở /qt/noi-dung. Thiếu file thì mọi câu chữ lấy mặc định
	// trong code — trang vẫn đủ chữ, chỉ là chưa có phần Kendy đổi.
	if err := core.NapND(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo nội dung:", err)
	}
	// Cụm liên hệ sửa ở /qt/lien-he. Thiếu file thì lấy khoá lien_he trong
	// config.yaml như trước khi có trang ấy.
	if err := core.NapBiMat(); err != nil {
		log.Fatal(err)
	}
	if err := core.NapLienHe(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo liên hệ:", err)
	}
	if err := core.NapTram(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo thông tin trạm:", err)
	}
	// Danh sách tiệm gia công ngoài ở data/doi-tac.yaml. Chưa có file thì
	// danh sách rỗng — đơn vẫn chạy, chỉ là chưa gửi đi đâu được.
	if err := core.NapDoiTac(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo đối tác:", err)
	}
	// Cấu hình màn app (ảnh banner, đợt khuyến mãi, icon động) ở data/app.yaml.
	// Chưa có file thì chạy bằng mặc định trong code, nên chỉ cảnh báo.
	if err := core.NapAppCauHinh(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo cấu hình app:", err)
	}
	// Logo/biểu tượng tab tải lên nằm ở data/logo/. Không có thì đầu trang
	// dùng SVG nhúng trong binary, nên đây cũng chỉ là cảnh báo.
	if err := core.NapLogo(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo logo:", err)
	}
	if err := core.InitTemplates(); err != nil {
		thoat("Lỗi giao diện", err)
	}

	// false = chế độ máy nhà: có cả /qt lẫn /noi-bo và /api/*.
	// Cửa sổ này chỉ mở trên máy Kendy nên GEMINI_API_KEY vẫn ở đúng chỗ.
	mux := core.NewMux(false)

	err := wails.Run(&options.App{
		Title:            core.TenDayDuTram(),
		Width:            1280,
		Height:           860,
		MinWidth:         900,
		MinHeight:        600,
		AssetServer:      &assetserver.Options{Handler: mux},
		BackgroundColour: &options.RGBA{R: 250, G: 249, B: 247, A: 1},
		Windows: &windows.Options{
			WebviewIsTransparent: false,
			WindowIsTranslucent:  false,
		},
	})
	if err != nil {
		thoat("Không mở được cửa sổ", err)
	}
}

func thoat(nhac string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", nhac, err)
	os.Exit(1)
}
