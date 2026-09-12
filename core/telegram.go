package core

// Kênh Telegram: một con bot nhắn riêng cho Kendy.
//
// Vì sao Telegram chứ không Zalo: Zalo OA muốn doanh nghiệp xác thực, muốn
// duyệt từng mẫu tin, và chỉ cho nhắn trong 24 giờ sau khi khách nhắn trước.
// Cái ta cần là báo việc cho ĐÚNG MỘT người — chủ trạm. Bot Telegram làm được
// việc đó bằng một token và một chat id, không đơn từ, không hạn mức.
//
// Hai thứ phải khai ở /qt/cai-dat:
//
//	token    xin ở @BotFather, dạng 123456789:AA...
//	chat id  số của cuộc trò chuyện. Kendy nhắn cho bot một câu bất kỳ rồi
//	         bấm "Dò chat id" — nút đó gọi getUpdates và đọc ra số.
//
// Vì sao phải nhắn trước: bot KHÔNG được phép mở đầu cuộc trò chuyện. Chưa ai
// nhắn cho nó thì getUpdates trả về danh sách rỗng, và sendMessage trả 403 —
// đúng như thiết kế của Telegram, không phải lỗi cấu hình.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const telegramGoc = "https://api.telegram.org/bot"

// tgGoi gọi một method của Bot API. Dùng chung mailClient: cùng một hạn 12
// giây, cùng một lý do — không để một dịch vụ ngoài treo goroutine của ta.
func tgGoi(token, method string, than any, ra any) error {
	b, err := json.Marshal(than)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", telegramGoc+token+"/"+method, bytes.NewReader(b))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := mailClient.Do(req)
	if err != nil {
		return fmt.Errorf("không gọi được Telegram: %w", err)
	}
	defer resp.Body.Close()
	// Đọc có giới hạn: getUpdates của một bot bị spam có thể trả về rất dài.
	thanRa, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return err
	}
	var bao struct {
		OK      bool            `json:"ok"`
		MoTa    string          `json:"description"`
		KetQua  json.RawMessage `json:"result"`
		ErrCode int             `json:"error_code"`
	}
	if err := json.Unmarshal(thanRa, &bao); err != nil {
		return fmt.Errorf("Telegram trả mã %d, nội dung không đọc được", resp.StatusCode)
	}
	if !bao.OK {
		mo := strings.TrimSpace(bao.MoTa)
		if mo == "" {
			mo = fmt.Sprintf("mã %d", bao.ErrCode)
		}
		// Dịch hai lỗi hay gặp nhất, vì nguyên văn tiếng Anh của Telegram
		// không nói ra phải làm gì.
		switch bao.ErrCode {
		case 401:
			mo = "token sai hoặc đã bị thu hồi ở @BotFather"
		case 403:
			mo = "bot chưa được phép nhắn — mở Telegram, tìm bot rồi nhắn cho nó một câu trước"
		case 400:
			if strings.Contains(strings.ToLower(mo), "chat not found") {
				mo = "không có chat id này — bấm “Dò chat id” để lấy số đúng"
			}
		}
		return errors.New(mo)
	}
	if ra != nil && len(bao.KetQua) > 0 {
		return json.Unmarshal(bao.KetQua, ra)
	}
	return nil
}

// guiTelegramSuKien — kênh Telegram của BaoTin.
func guiTelegramSuKien(sk SuKien) (string, error) {
	token, chat := TelegramToken(), TelegramChat()
	if token == "" || chat == "" {
		return "chưa khai token hoặc chat id", errKenhTat
	}
	chu := sk.TieuDe
	if strings.TrimSpace(sk.Than) != "" {
		chu += "\n\n" + sk.Than
	}
	if l := sk.LinkDayDu(); strings.HasPrefix(l, "http") {
		chu += "\n\n" + l
	}
	// Không dùng parse_mode: chữ trong tin là tên khách và mô tả hỏng do khách
	// gõ. Bật Markdown là một dấu * lạc trong tên khách làm Telegram từ chối
	// cả tin, mà lỗi hiện ra là "can't parse entities" — không ai lần ra.
	than := map[string]any{
		"chat_id":              chat,
		"text":                 catBot(chu, 3500),
		"link_preview_options": map[string]any{"is_disabled": true},
		"disable_notification": false,
	}
	if err := tgGoi(token, "sendMessage", than, nil); err != nil {
		return "", err
	}
	return "đã gửi tới chat " + chat, nil
}

// DoChatTelegram đọc tin gần nhất ai đã nhắn cho bot, trả về chat id. Đây là
// cách duy nhất lấy được số đó mà không bắt Kendy đi cài thêm bot thứ ba
// (@userinfobot) chỉ để đọc một con số.
func DoChatTelegram() (string, string, error) {
	token := TelegramToken()
	if token == "" {
		return "", "", errors.New("chưa khai token")
	}
	var ds []struct {
		Tin struct {
			Chat struct {
				ID    int64  `json:"id"`
				Ten   string `json:"first_name"`
				Ho    string `json:"last_name"`
				Nick  string `json:"username"`
				TenNh string `json:"title"`
			} `json:"chat"`
		} `json:"message"`
	}
	// limit nhỏ và chỉ xin loại message: ta cần một con số, không cần lịch sử.
	than := map[string]any{"limit": 20, "allowed_updates": []string{"message"}}
	if err := tgGoi(token, "getUpdates", than, &ds); err != nil {
		return "", "", err
	}
	for i := len(ds) - 1; i >= 0; i-- {
		c := ds[i].Tin.Chat
		if c.ID == 0 {
			continue
		}
		ten := strings.TrimSpace(strings.TrimSpace(c.Ten+" "+c.Ho) + " " + c.TenNh)
		if ten == "" {
			ten = c.Nick
		}
		return fmt.Sprint(c.ID), strings.TrimSpace(ten), nil
	}
	return "", "", errors.New("bot chưa nhận tin nào — mở Telegram, tìm bot rồi nhắn cho nó một câu, xong bấm lại nút này")
}

// TenBotTelegram gọi getMe, dùng cho nút kiểm tra token: nó nói ra bot nào
// đang giữ token, để Kendy biết mình vừa dán token của đúng con bot hay không.
func TenBotTelegram() (string, error) {
	token := TelegramToken()
	if token == "" {
		return "", errors.New("chưa khai token")
	}
	var me struct {
		Nick string `json:"username"`
		Ten  string `json:"first_name"`
	}
	if err := tgGoi(token, "getMe", map[string]any{}, &me); err != nil {
		return "", err
	}
	if me.Nick != "" {
		return "@" + me.Nick, nil
	}
	return me.Ten, nil
}

// linkBotTelegram — đường mở thẳng cuộc trò chuyện với bot, để Kendy bấm một
// lần rồi nhắn. Rỗng khi chưa lấy được tên bot.
func linkBotTelegram(nick string) string {
	nick = strings.TrimPrefix(strings.TrimSpace(nick), "@")
	if nick == "" {
		return ""
	}
	return "https://t.me/" + url.PathEscape(nick)
}
