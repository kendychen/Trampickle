package core

// Gửi mail cho khách sau khi họ gửi ảnh chỗ hỏng.
//
// Vì sao Resend chứ không phải SMTP thẳng từ VPS: một IP VPS mới toanh gửi
// SMTP tới Gmail thì gần như chắc chắn vào thư rác hoặc bị chặn thẳng. Việc
// dựng SPF/DKIM/DMARC cho tử tế tốn đúng bằng việc gọi một API, mà lại phải
// tự canh reputation. Resend lo phần đó.
//
// Khóa API đọc qua ResendKey(): biến môi trường RESEND_API_KEY trước, rồi
// tới ô nhập ở /qt/cai-dat. KHÔNG nằm trong
// config.yaml: file đó có mặt trong git.
//
// Nguyên tắc quan trọng nhất của file này: mail hỏng không được làm hỏng đơn.
// Khách đã thấy mã trên màn hình rồi; mail chỉ là tiện nghi thêm. Nên mọi lỗi
// ở đây đều nuốt vào log, không đẩy ra trang khách, và việc gửi chạy ở
// goroutine riêng để trang không phải đợi Resend.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	resendURL     = "https://api.resend.com/emails"
	mailTimeout   = 12 * time.Second
	mailThuLaiLan = 2
)

var mailClient = &http.Client{Timeout: mailTimeout}

// HopLeEmail — kiểm tra vừa đủ để khỏi gọi API với chuỗi rác. Không cố viết
// regex đúng RFC 5322: chuyện đó vô ích, địa chỉ có gõ đúng cú pháp vẫn có
// thể không tồn tại. Sai thì Resend trả lỗi, và ta đã quyết định nuốt lỗi.
func HopLeEmail(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 6 || len(s) > 254 || strings.ContainsAny(s, " \t\r\n,;<>\"") {
		return false
	}
	i := strings.LastIndex(s, "@")
	if i < 1 || i == len(s)-1 {
		return false
	}
	mien := s[i+1:]
	return strings.Contains(mien, ".") && !strings.HasPrefix(mien, ".") && !strings.HasSuffix(mien, ".")
}

type thuResend struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	ReplyTo string   `json:"reply_to,omitempty"`
	Subject string   `json:"subject"`
	Html    string   `json:"html"`
	Text    string   `json:"text"`
}

// guiMail gọi Resend. Thử lại đúng một lần: lỗi mạng chớp nhoáng thì lần hai
// qua, còn lỗi 4xx (sai khóa, sai địa chỉ gửi, chưa xác thực tên miền) thì
// thử bao nhiêu lần cũng thế — nên 4xx là dừng luôn.
func guiMail(den, tieuDe, thanHTML, thanChu string) error {
	if !MailBat() {
		return fmt.Errorf("chưa bật gửi mail")
	}
	if !HopLeEmail(den) {
		return fmt.Errorf("địa chỉ nhận không hợp lệ")
	}
	b, err := json.Marshal(thuResend{
		From:    MailTu(),
		To:      []string{strings.TrimSpace(den)},
		ReplyTo: MailTraLoi(),
		Subject: tieuDe,
		Html:    thanHTML,
		Text:    thanChu,
	})
	if err != nil {
		return err
	}

	var loiCuoi error
	for lan := 1; lan <= mailThuLaiLan; lan++ {
		ctx, huy := context.WithTimeout(context.Background(), mailTimeout)
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendURL, bytes.NewReader(b))
		if err != nil {
			huy()
			return err
		}
		req.Header.Set("Authorization", "Bearer "+ResendKey())
		req.Header.Set("Content-Type", "application/json")

		res, err := mailClient.Do(req)
		if err != nil {
			huy()
			loiCuoi = err
			continue
		}
		// Đọc hết body rồi mới đóng, để connection được tái dùng.
		var traLoi bytes.Buffer
		traLoi.ReadFrom(io.LimitReader(res.Body, 8<<10))
		res.Body.Close()
		huy()

		if res.StatusCode >= 200 && res.StatusCode < 300 {
			return nil
		}
		loiCuoi = fmt.Errorf("resend trả %d: %s", res.StatusCode, strings.TrimSpace(traLoi.String()))
		if res.StatusCode < 500 {
			return loiCuoi
		}
	}
	return loiCuoi
}

// MailXacNhanYeuCau gửi mã yêu cầu về hộp thư khách, chạy nền.
//
// Nội dung bám đúng lời hứa đang bán trên trang: báo giá trước, chốt rồi mới
// làm. Không hứa gì thêm ở đây — mail là thứ khách giữ lại và đem ra đối
// chiếu, nên câu chữ trong này phải trùng với câu chữ trên web.
func MailXacNhanYeuCau(yc yeuCauKhach) {
	den := strings.TrimSpace(yc.Email)
	if den == "" || !MailBat() || !HopLeEmail(den) {
		return
	}
	go func() {
		tieuDe := fmt.Sprintf("Đã nhận yêu cầu %s — %s", yc.Ma, TenTram())
		h, c := thanMailYeuCau(yc)
		if err := guiMail(den, tieuDe, h, c); err != nil {
			// Chỉ ghi log. Khách đã cầm mã trên màn hình, không mất gì.
			fmt.Fprintln(os.Stderr, "Gửi mail hụt cho", yc.Ma+":", err)
		}
	}()
}

func linkTraCuu() string {
	goc := GocWeb()
	if goc == "" {
		return ""
	}
	return goc + "/tra-cuu"
}

// thanMailYeuCau dựng cả bản HTML lẫn bản chữ trơn.
//
// Bản chữ trơn không phải cho đẹp: thiếu nó, bộ lọc thư rác chấm điểm mail
// xấu hẳn, mà đây lại đúng là loại mail không được phép rơi vào spam.
//
// Mọi thứ khách tự gõ đều đi qua html.EscapeString. Ô "mô tả tình trạng vợt"
// nhận 2000 ký tự tùy ý — nhét thẳng vào HTML là mời người ta chèn thẻ vào
// mail do mình đứng tên gửi.
func thanMailYeuCau(yc yeuCauKhach) (string, string) {
	e := html.EscapeString
	ten := strings.TrimSpace(yc.Ten)
	chao := "Chào anh/chị"
	if ten != "" {
		chao = "Chào anh/chị " + ten
	}
	link := linkTraCuu()

	var h strings.Builder
	h.WriteString(`<div style="font-family:-apple-system,Segoe UI,Roboto,Helvetica,Arial,sans-serif;font-size:15px;line-height:1.65;color:#1a1d18;max-width:560px">`)
	fmt.Fprintf(&h, `<p>%s,</p>`, e(chao))
	fmt.Fprintf(&h, `<p>%s đã nhận được ảnh anh/chị gửi. Thợ đang xem để báo giá, và sẽ nhắn lại trong ngày.</p>`, e(TenTram()))
	fmt.Fprintf(&h, `<p style="margin:22px 0;padding:14px 16px;border:1px dashed #999;border-radius:8px">`+
		`<span style="font-size:13px;color:#666">Mã yêu cầu của anh/chị</span><br>`+
		`<b style="font-size:19px;letter-spacing:.04em">%s</b></p>`, e(yc.Ma))

	if strings.TrimSpace(yc.VotHang) != "" || strings.TrimSpace(yc.MoTa) != "" {
		h.WriteString(`<p style="font-size:13px;color:#666;margin-bottom:6px">Anh/chị đã gửi:</p><ul style="margin-top:0;padding-left:20px">`)
		if v := strings.TrimSpace(yc.VotHang); v != "" {
			fmt.Fprintf(&h, `<li>Vợt: %s</li>`, e(v))
		}
		if m := strings.TrimSpace(yc.MoTa); m != "" {
			fmt.Fprintf(&h, `<li>Tình trạng: %s</li>`, e(m))
		}
		if n := len(yc.Tep); n > 0 {
			fmt.Fprintf(&h, `<li>%d tệp ảnh/video</li>`, n)
		}
		h.WriteString(`</ul>`)
	}

	if link != "" {
		fmt.Fprintf(&h, `<p>Xem yêu cầu đang tới đâu: <a href="%s">%s</a> — gõ mã trên cùng 4 số cuối điện thoại.</p>`, e(link), e(link))
	} else {
		h.WriteString(`<p>Vào mục <b>Tra cứu</b> trên web, gõ mã trên cùng 4 số cuối điện thoại là xem được yêu cầu đang tới đâu.</p>`)
	}
	h.WriteString(`<p><b>Chưa đồng ý giá thì trạm chưa động vào vợt.</b> Báo giá xong, anh/chị gật thì thợ mới làm.</p>`)
	fmt.Fprintf(&h, `<p style="font-size:13px;color:#666;margin-top:26px">%s`, e(TenTram()))
	if dt := strings.TrimSpace(LienHeHienTai().DienThoai); dt != "" {
		fmt.Fprintf(&h, ` · %s`, e(dt))
	}
	h.WriteString(`<br>Thư này gửi tự động, anh/chị trả lời thẳng vào đây cũng được.</p></div>`)

	var c strings.Builder
	fmt.Fprintf(&c, "%s,\n\n%s đã nhận được ảnh anh/chị gửi. Thợ đang xem để báo giá, và sẽ nhắn lại trong ngày.\n\n",
		chao, TenTram())
	fmt.Fprintf(&c, "MÃ YÊU CẦU: %s\n\n", yc.Ma)
	if v := strings.TrimSpace(yc.VotHang); v != "" {
		fmt.Fprintf(&c, "Vợt: %s\n", v)
	}
	if m := strings.TrimSpace(yc.MoTa); m != "" {
		fmt.Fprintf(&c, "Tình trạng: %s\n", m)
	}
	if n := len(yc.Tep); n > 0 {
		fmt.Fprintf(&c, "Đã gửi %d tệp ảnh/video\n", n)
	}
	c.WriteString("\n")
	if link != "" {
		fmt.Fprintf(&c, "Tra cứu: %s (mã trên + 4 số cuối điện thoại)\n\n", link)
	} else {
		c.WriteString("Vào mục Tra cứu trên web, gõ mã trên cùng 4 số cuối điện thoại.\n\n")
	}
	c.WriteString("Chưa đồng ý giá thì trạm chưa động vào vợt.\n\n")
	c.WriteString(TenTram())
	if dt := strings.TrimSpace(LienHeHienTai().DienThoai); dt != "" {
		c.WriteString(" · " + dt)
	}
	c.WriteString("\nThư này gửi tự động, anh/chị trả lời thẳng vào đây cũng được.\n")

	return h.String(), c.String()
}
