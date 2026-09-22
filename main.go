// Trạm Pickle — web + quản lý đơn + RAG, một file chạy được.
//
//	tramvot                    máy nhà: web + quản trị + dashboard agent, chỉ localhost
//	tramvot -public            VPS: web khách + quản trị, mở ra internet, KHÔNG có agent
//	tramvot -index             dựng lại index RAG rồi thoát
//	tramvot -hoi "..."         hỏi agent từ dòng lệnh rồi thoát
//	tramvot -them-nguoi-dung   tạo tài khoản (hỏi mật khẩu trên terminal)
//	tramvot -doi-mat-khau      đổi mật khẩu một tài khoản
package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/term"

	"tramvot/core"
)

func main() {
	port := flag.Int("port", 8442, "cổng HTTP")
	public := flag.Bool("public", false, "mở ra internet: web khách + quản trị, không có route agent")
	lamIndex := flag.Bool("index", false, "dựng lại index RAG rồi thoát")
	lamLai := flag.Bool("lam-lai", false, "dùng với -index: nhúng lại từ đầu, bỏ vector cũ")
	hoi := flag.String("hoi", "", "hỏi Technical Agent rồi thoát")
	tinProxy := flag.Bool("tin-proxy", false, "tin header X-Forwarded-For (chỉ bật khi có nginx/cloudflare đứng trước)")
	https := flag.Bool("https", false, "đang chạy sau HTTPS: bật cờ Secure cho cookie phiên")
	themND := flag.String("them-nguoi-dung", "", "tạo tài khoản mới: -them-nguoi-dung tên")
	hoTen := flag.String("ho-ten", "", "dùng với -them-nguoi-dung: họ tên hiển thị")
	vaiTro := flag.String("vai-tro", "tho", "dùng với -them-nguoi-dung: chu | tho")
	doiMK := flag.String("doi-mat-khau", "", "đổi mật khẩu: -doi-mat-khau tên")
	xemND := flag.Bool("nguoi-dung", false, "liệt kê tài khoản rồi thoát")
	flag.Parse()

	core.TinProxy = *tinProxy
	core.HTTPSBat = *https

	if err := core.LoadConfig(); err != nil {
		fmt.Fprintln(os.Stderr, "Lỗi cấu hình:", err)
		os.Exit(1)
	}
	if err := core.NapNguoiDung(); err != nil {
		fmt.Fprintln(os.Stderr, "Lỗi đọc tài khoản:", err)
		os.Exit(1)
	}
	if err := core.NapDon(); err != nil {
		fmt.Fprintln(os.Stderr, "Lỗi đọc đơn hàng:", err)
		os.Exit(1)
	}
	// Sổ tiền phải nạp trước khi hiện đơn: "đã thu" của mỗi đơn cộng từ sổ.
	// Không nạp được mà vẫn chạy thì mọi đơn hiện đã thu 0đ, khách nào cũng
	// thành đang nợ — sai âm thầm, nên dừng hẳn.
	if err := core.NapSoTien(); err != nil {
		fmt.Fprintln(os.Stderr, "Lỗi đọc sổ tiền:", err)
		os.Exit(1)
	}
	// Kho hỏng thì trang kho rỗng và nhìn ra ngay, không đáng để cả web chết.
	if err := core.NapKho(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo kho:", err)
	}
	if err := core.NapDoNghe(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo đồ nghề:", err)
	}
	if err := core.NapDinhKy(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo khoản định kỳ:", err)
	}
	// Băng ảnh trang chủ cũng vậy: hỏng thì trang chủ quay về bản vẽ cây vợt.
	if err := core.NapAnhHero(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo ảnh trang chủ:", err)
	}
	// Bài viết nằm ở data/bai-viet/, KHÔNG nằm trong binary. Thiếu thư mục
	// thì mục bài viết rỗng chứ web vẫn chạy — nên chỉ cảnh báo.
	if err := core.NapBaiViet(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo bài viết:", err)
	}
	// Bài của từng việc ở data/dich-vu-bai/, cũng không nằm trong binary.
	// Thiếu thì trang dịch vụ mất phần chữ, còn nguyên giá — chỉ cảnh báo.
	if err := core.NapBaiDichVu(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo bài dịch vụ:", err)
	}
	// Chữ đã sửa ở /qt/noi-dung. Thiếu file thì mọi câu chữ lấy mặc định
	// trong code — trang vẫn đủ chữ, chỉ là chưa có phần Kendy đổi.
	if err := core.NapND(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo nội dung:", err)
	}
	// Công tắc Tại xưởng / Online ở data/giao-dien.yaml, đổi ở /qt/giao-dien.
	// Thiếu file hay file hỏng thì chạy bản Tại xưởng — bản cũ, an toàn nhất.
	if err := core.NapCheDo(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo chế độ:", err)
	}
	// Cụm liên hệ sửa ở /qt/lien-he. Thiếu file thì lấy khoá lien_he trong
	// config.yaml như trước khi có trang ấy.
	if err := core.NapBiMat(); err != nil {
		fmt.Fprintln(os.Stderr, "Lỗi đọc khóa API:", err)
		os.Exit(1)
	}
	if err := core.NapLienHe(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo liên hệ:", err)
	}
	// Tên trạm, tiền tố mã đơn, cụm email — sửa ở /qt/tram. Thiếu file thì
	// lấy khoá thuong_hieu và email trong config.yaml như trước.
	if err := core.NapTram(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo thông tin trạm:", err)
	}
	// Danh sách tiệm gia công ngoài ở data/doi-tac.yaml. Chưa có file thì
	// danh sách rỗng — đơn vẫn chạy, chỉ là chưa gửi đi đâu được.
	if err := core.NapDoiTac(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo đối tác:", err)
	}
	// Hồ sơ khách ở data/khach-hang.yaml. File hỏng cũng không được chặn khởi
	// động: trạm vẫn phải nhận đơn được, chỉ là mất phần gợi ý khách quen.
	// Ngưỡng nội bộ (mức giảm khách quen). Thiếu file thì dùng mặc định khai
	// trong core/nguongtram.go, không chặn khởi động.
	if err := core.NapNguongTram(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo ngưỡng trạm:", err)
	}
	if err := core.NapKhach(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo khách hàng:", err)
	}
	// Câu hỏi thường gặp ở data/cau-hoi.yaml. Chưa có file thì trang /cau-hoi
	// hiện câu "chưa có câu nào" — không phải lỗi, chỉ là Kendy chưa gõ.
	if err := core.NapCauHoi(); err != nil {
		fmt.Fprintln(os.Stderr, "Cảnh báo câu hỏi:", err)
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
	core.NapBaoTri()

	switch {
	case *themND != "":
		chayThemNguoiDung(*themND, *hoTen, *vaiTro)
		return
	case *doiMK != "":
		chayDoiMatKhau(*doiMK)
		return
	case *xemND:
		chayXemNguoiDung()
		return
	case *lamIndex:
		chayIndex(*lamLai)
		return
	case *hoi != "":
		chayHoi(*hoi)
		return
	}

	if err := core.InitTemplates(); err != nil {
		fmt.Fprintln(os.Stderr, "Lỗi giao diện:", err)
		os.Exit(1)
	}

	// Việc nền: dựng khoản định kỳ đầu tháng, gửi báo cáo sổ tháng trước.
	// Nhật ký nằm trong data/ nên mỗi bản chạy tự nhớ đã gửi gì; máy nhà và
	// VPS là hai thư mục data khác nhau, chạy song song thì mail đi hai lần.
	go core.ChayViecNen(nil)

	dia := "127.0.0.1"
	if *public {
		dia = "0.0.0.0"
	}
	addr := fmt.Sprintf("%s:%d", dia, *port)

	srv := &http.Server{
		Addr:              addr,
		Handler:           core.NewHandler(*public),
		ReadHeaderTimeout: 10 * time.Second,
		// Khách gửi kèm video quay bằng điện thoại: 40MB qua mạng 4G yếu
		// mất hơn một phút. 60s cắt đúng lúc sắp xong, và khách chỉ thấy
		// trang lỗi chứ không biết vì sao. Chặn nặng đã có MaxBytesReader.
		ReadTimeout: 300 * time.Second,
		// Hỏi agent qua Ollama có thể mất hàng phút trên máy yếu.
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Printf("\n%s\n", core.TenDayDuTram())
	fmt.Printf("Thư mục dự án: %s\n", core.Root)
	if *public {
		fmt.Printf("Chế độ CÔNG KHAI — web khách + trang quản trị (có đăng nhập).\n")
		fmt.Printf("   http://<ip-máy-chủ>:%d       và  /qt sau khi đăng nhập\n", *port)
		fmt.Printf("   Trợ lý kỹ thuật: /noi-bo — chỉ tài khoản vai trò chủ.\n")
		fmt.Printf("   /api/index KHÔNG đăng ký ở chế độ này; dựng index ở máy nhà.\n")
	} else {
		fmt.Printf("Trang khách:   http://localhost:%d\n", *port)
		fmt.Printf("Quản lý đơn:   http://localhost:%d/qt\n", *port)
		fmt.Printf("Dashboard AI:  http://localhost:%d/noi-bo\n", *port)
		fmt.Printf("Chỉ máy này vào được. Muốn mở ra ngoài phải chạy -public.\n")
	}
	if !*https {
		fmt.Printf("\nCHƯA CÓ HTTPS: mật khẩu đăng nhập đi qua mạng dạng bản rõ.\n")
		fmt.Printf("   Có tên miền rồi thì đặt nginx/Caddy đứng trước và chạy thêm -https -tin-proxy.\n")
	}
	if len(core.DanhSachNguoiDung()) == 0 {
		fmt.Printf("\nCHƯA CÓ TÀI KHOẢN NÀO. Tạo tài khoản chủ:\n")
		fmt.Printf("   tramvot -them-nguoi-dung kendy -ho-ten \"Kendy\" -vai-tro chu\n")
	}
	fmt.Printf("\nModel: %s | Embedding: %s\n", core.Provider(), core.EmbedProvider())
	if st := core.RagStatus(); st.DaDungIndex {
		fmt.Printf("RAG: %d đoạn, dựng lúc %s\n", st.SoDoan, st.NgayDung)
	} else {
		fmt.Printf("RAG: chưa dựng index — chạy `tramvot -index`\n")
	}
	fmt.Println("\nCtrl+C để dừng.")

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintln(os.Stderr, "Server chết:", err)
		os.Exit(1)
	}
}

// --- Tài khoản trên dòng lệnh ---------------------------------------

// hoiMatKhau đọc mật khẩu không hiện ra màn hình. Gõ mật khẩu vào tham số
// dòng lệnh thì nó nằm lại trong lịch sử shell và trong `ps`.
// Một reader dùng chung cho cả phiên: mỗi bufio.NewReader mới nuốt luôn
// phần input đã đệm, nên hỏi hai lần liên tiếp qua pipe sẽ mất dòng sau.
var stdin = bufio.NewReader(os.Stdin)

func hoiMatKhau(nhac string) string {
	fmt.Print(nhac)
	if term.IsTerminal(int(os.Stdin.Fd())) {
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		if err == nil {
			return string(b)
		}
	}
	s, _ := stdin.ReadString('\n')
	return strings.TrimRight(s, "\r\n")
}

func chayThemNguoiDung(ten, hoTen, vaiTro string) {
	mk := hoiMatKhau("Mật khẩu mới (ít nhất 8 ký tự): ")
	mk2 := hoiMatKhau("Gõ lại: ")
	if mk != mk2 {
		fmt.Fprintln(os.Stderr, "Hai lần gõ không giống nhau.")
		os.Exit(1)
	}
	if err := core.ThemNguoiDung(ten, hoTen, mk, vaiTro); err != nil {
		fmt.Fprintln(os.Stderr, "Không tạo được:", err)
		os.Exit(1)
	}
	nd, _ := core.TimNguoiDung(ten)
	fmt.Printf("Đã tạo tài khoản %s (%s), vai trò: %s\n", nd.Ten, nd.TenHienThi(), nd.VaiTro)
	nhacNapLai()
}

// nhacNapLai: lệnh này ghi vào data/nguoi-dung.yaml, còn tiến trình web đang
// chạy giữ danh sách tài khoản trong RAM và chỉ đọc file lúc khởi động.
// Không nhắc thì đăng nhập sẽ báo sai mật khẩu mà không hiểu vì sao.
func nhacNapLai() {
	fmt.Println("Server đang chạy chưa biết thay đổi này. Chạy lại nó —")
	fmt.Println("   trên VPS: systemctl restart tramvot")
}

func chayDoiMatKhau(ten string) {
	if _, co := core.TimNguoiDung(ten); !co {
		fmt.Fprintln(os.Stderr, "Không có tài khoản tên:", ten)
		os.Exit(1)
	}
	mk := hoiMatKhau("Mật khẩu mới (ít nhất 8 ký tự): ")
	mk2 := hoiMatKhau("Gõ lại: ")
	if mk != mk2 {
		fmt.Fprintln(os.Stderr, "Hai lần gõ không giống nhau.")
		os.Exit(1)
	}
	if err := core.DoiMatKhau(ten, mk); err != nil {
		fmt.Fprintln(os.Stderr, "Không đổi được:", err)
		os.Exit(1)
	}
	fmt.Println("Đã đổi mật khẩu cho", ten)
	nhacNapLai()
}

func chayXemNguoiDung() {
	ds := core.DanhSachNguoiDung()
	if len(ds) == 0 {
		fmt.Println("Chưa có tài khoản nào.")
		return
	}
	for _, n := range ds {
		trangThai := "bật"
		if n.Tat {
			trangThai = "TẮT"
		}
		fmt.Printf("  %-14s %-24s %-4s %s\n", n.Ten, n.TenHienThi(), n.VaiTro, trangThai)
	}
}

// --- RAG / agent trên dòng lệnh -------------------------------------

func chayIndex(lamLai bool) {
	tk, err := core.BuildIndex(lamLai, func(s string) { fmt.Println("[rag]", s) })
	if err != nil {
		fmt.Fprintln(os.Stderr, "Dựng index hỏng:", err)
		os.Exit(1)
	}
	fmt.Printf("\nXong. %d đoạn (%d đoạn mới nhúng), %d chiều, qua %s\n",
		tk.SoDoan, tk.SoDoanMoi, tk.SoChieu, tk.NhaCungCap)
	fmt.Printf("Ghi vào: %s\n\nNguồn:\n", tk.DuongDan)
	for _, n := range tk.Nguon {
		fmt.Printf("  - %s\n", n)
	}
	fmt.Printf("\nKhông đưa vào index (luôn nạp nguyên vẹn): %s\n",
		strings.Join(core.LuonNapFiles, ", "))
}

func chayHoi(cauHoi string) {
	kq, err := core.HoiKyThuat(cauHoi)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Lỗi:", err)
		os.Exit(1)
	}
	for _, c := range kq.CanhBao {
		fmt.Printf("[cảnh báo] %s\n", c)
	}
	fmt.Printf("\n%s\n\n", kq.CauTraLoi)
	if len(kq.DoanDaDung) > 0 {
		fmt.Println("--- Căn cứ đã dùng ---")
		for _, d := range kq.DoanDaDung {
			fmt.Printf("  [%.3f] %s (%s)\n", d.Diem, d.TieuDe, d.Nguon)
		}
	}
	fmt.Printf("(trả lời bởi %s)\n", kq.NhaCungCap)
}
