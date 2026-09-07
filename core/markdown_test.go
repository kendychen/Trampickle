package core

import (
	"strings"
	"testing"
)

// Test viết theo lối "gõ vào cái gì thì phải ra cái gì", vì trình dựng này
// là thứ duy nhất đứng giữa ô nhập của admin và trang khách: nó sai thì bài
// viết vỡ chứ không có lớp nào đỡ phía sau.

func co(t *testing.T, ten, vao, phai string) {
	t.Helper()
	ra := dungMD(vao, false)
	if !strings.Contains(ra, phai) {
		t.Errorf("%s: thiếu %q\ndựng ra:\n%s", ten, phai, ra)
	}
}

func khong(t *testing.T, ten, vao, cam string) {
	t.Helper()
	ra := dungMD(vao, false)
	if strings.Contains(ra, cam) {
		t.Errorf("%s: không được có %q\ndựng ra:\n%s", ten, cam, ra)
	}
}

func TestKhoi(t *testing.T) {
	co(t, "h2", "## Bốn vị trí", "<h2>Bốn vị trí</h2>")
	co(t, "h3", "### Đầu vợt", "<h3>Đầu vợt</h3>")
	co(t, "h1 hạ xuống h2", "# Một dấu thăng", "<h2>Một dấu thăng</h2>")
	co(t, "đoạn", "Cây vợt này\ncòn cứu được.", "<p>Cây vợt này còn cứu được.</p>")
	co(t, "hai đoạn", "Một.\n\nHai.", "<p>Một.</p>\n<p>Hai.</p>")
	co(t, "gạch đầu dòng", "- Lau sạch\n- Miết kỹ", `<ul class="gach">`)
	co(t, "gạch đầu dòng", "- Lau sạch\n- Miết kỹ", "<li>Lau sạch</li>")
	co(t, "đánh số", "1. Bắt đầu\n2. Đánh thử", "<ol>\n<li>Bắt đầu</li>")
	co(t, "mục xuống hàng", "- Lau sạch chỗ dán.\n  Bụi sân làm bong.", "<li>Lau sạch chỗ dán. Bụi sân làm bong.</li>")
	co(t, "lưu ý", "> Đừng dán quá 8 gram.", `<div class="luu-y">Đừng dán quá 8 gram.</div>`)
	co(t, "vạch", "---", "<hr>")
}

func TestDoanDan(t *testing.T) {
	ra := dungMD("Mở bài.\n\nĐoạn hai.", true)
	if !strings.Contains(ra, `<p class="dat">Mở bài.</p>`) {
		t.Errorf("đoạn đầu phải là đoạn dẫn, dựng ra:\n%s", ra)
	}
	if strings.Contains(ra, `<p class="dat">Đoạn hai.`) {
		t.Errorf("chỉ đoạn đầu mới là đoạn dẫn, dựng ra:\n%s", ra)
	}
}

func TestBang(t *testing.T) {
	vao := "| Kiểu hỏng | Sửa được |\n|---|---|\n| Bong viền | Được |\n| Sập lõi | Không |"
	co(t, "bảng", vao, `<table class="bang">`)
	co(t, "bảng", vao, "<th>Kiểu hỏng</th>")
	co(t, "bảng", vao, "<td>Bong viền</td>")

	// Đoạn văn lỡ mở đầu bằng dấu | thì vẫn là đoạn văn.
	khong(t, "không phải bảng", "| chỉ một dòng thôi", "<table")
}

func TestNhanChu(t *testing.T) {
	co(t, "đậm", "**Lau sạch** rồi dán.", "<b>Lau sạch</b> rồi dán.")
	co(t, "nghiêng", "dán *đối xứng* hai bên", "<em>đối xứng</em>")
	co(t, "mã", "sửa `config.yaml` là xong", "<code>config.yaml</code>")
	co(t, "link trong site", "xem [danh sách dịch vụ](/dich-vu) nhé", `<a href="/dich-vu">danh sách dịch vụ</a>`)
	co(t, "link ngoài", "xem [trang này](https://x.test)", `rel="nofollow noopener" target="_blank"`)

	// Dấu sao nằm trong đoạn mã không được hiểu thành in nghiêng.
	khong(t, "sao trong mã", "gõ `a * b * c` vào", "<em>")
}

func TestChanHTMLVaLinkBan(t *testing.T) {
	khong(t, "escape thẻ", "Gõ nhầm <script>alert(1)</script> giữa bài", "<script>")
	co(t, "escape thẻ", "Gõ nhầm <b>đậm</b> bằng HTML", "&lt;b&gt;")

	khong(t, "javascript:", "[bấm](javascript:alert(1))", "javascript:")
	co(t, "javascript: giữ chữ", "[bấm](javascript:alert(1))", "bấm")
	khong(t, "data:", "![x](data:text/html;base64,AAA)", "data:")
}

func TestThe(t *testing.T) {
	GIA.Nguong.TangKhoiLuongToiDaG = 3
	co(t, "ngưỡng cân", "chênh {nguong_can} gram", "chênh 3 gram")
	co(t, "khoá lạ để nguyên", "còn {khoa_khong_co} thì sao", "{khoa_khong_co}")
}

func TestHinh(t *testing.T) {
	if err := InitTemplates(); err != nil {
		t.Fatal(err)
	}
	ra := dungMD(":::hinh vot-bo\nCàng xa tay cầm, mỗi gram càng đắt.\n:::", false)
	if !strings.Contains(ra, `<figure class="hinh-o">`) || !strings.Contains(ra, "<svg") {
		t.Errorf("thiếu hình, dựng ra:\n%s", ra)
	}
	if !strings.Contains(ra, "<figcaption>Càng xa tay cầm, mỗi gram càng đắt.</figcaption>") {
		t.Errorf("thiếu chú thích, dựng ra:\n%s", ra)
	}

	// Tên hình bịa ra thì không được gọi tới template khác.
	ra = dungMD(":::hinh qt-dau\n:::", false)
	if strings.Contains(ra, "<svg") || strings.Contains(ra, "<aside") {
		t.Errorf("tên hình ngoài danh sách trắng vẫn dựng ra được:\n%s", ra)
	}
}

func TestAnh(t *testing.T) {
	co(t, "ảnh đứng riêng", "![Vợt bong viền](/anh-bai/a.jpg)", `<figure class="hinh-o"><img src="/anh-bai/a.jpg"`)
	co(t, "ảnh đứng riêng", "![Vợt bong viền](/anh-bai/a.jpg)", "<figcaption>Vợt bong viền</figcaption>")
	co(t, "ảnh giữa câu", "Nhìn ![vợt](/anh-bai/a.jpg) này", `Nhìn <img src="/anh-bai/a.jpg"`)
}

// Trình dựng không được rơi vào vòng lặp vô tận với đầu vào lạ. Mỗi ca ở đây
// từng là một cách làm treo bản nháp: dòng ":::" cụt, bảng thiếu thân, dấu
// gạch một mình.
func TestDauVaoLa(t *testing.T) {
	for _, s := range []string{":::", ":::hinh", "-", ">", "|", "|\n|-", "#", "```", "\x00"} {
		dungMD(s, true)
	}
}
