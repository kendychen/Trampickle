package core

// --- Khóa API và lựa chọn model, sửa được trong admin ----------------------
//
// Trước đây khóa chỉ đọc từ biến môi trường, nghĩa là mỗi lần đổi khóa phải
// ssh vào máy chủ sửa .env rồi restart. Từ khi /noi-bo mở trên web thì cái
// vòng đó vô lý: Kendy đang ngồi trong admin, cần dán một chuỗi, mà phải mở
// terminal.
//
// Đi theo đúng lối data/lien-he.yaml: ghi ra data/bi-mat.yaml, file nằm trong
// data/ nên deploy không đụng tới và systemd cho ghi. Khác một điểm — file
// này là BÍ MẬT: chmod 600, có trong .gitignore, và lệnh kéo data/ về máy nhà
// trong README sẽ kéo luôn nó, biết mà đừng để bản sao lung tung.
//
// Thứ tự ưu tiên: biến môi trường THẮNG file. Máy nhà có .env thì .env nói
// thật; ai đặt biến môi trường là cố ý, đừng để một file ghi lúc nào không
// biết đè lên. Trang admin nói rõ khi nó đang bị đè, thay vì im lặng lưu một
// giá trị không có tác dụng.

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

// BiMat là toàn bộ nội dung data/bi-mat.yaml.
type BiMat struct {
	GeminiKey string `yaml:"gemini_api_key"`
	ResendKey string `yaml:"resend_api_key"`
	// NhaCungCap rỗng = theo model.mac_dinh trong config.yaml.
	NhaCungCap string `yaml:"nha_cung_cap"`
	// TatDuPhong: không rơi sang model dự phòng khi model chính hỏng. Trên
	// máy chủ không có Ollama, để nguyên dự phòng thì mọi lỗi Gemini đều
	// hiện ra dưới dạng "Ollama connection refused" — sai chỗ, khó hiểu.
	TatDuPhong bool `yaml:"tat_du_phong"`

	// Bot Telegram báo việc. Token xin ở @BotFather. ChatID là số của đúng
	// cuộc trò chuyện sẽ nhận tin — lấy bằng nút "Dò chat" ở /qt/cai-dat
	// sau khi đã nhắn cho bot một câu bất kỳ.
	TelegramToken string `yaml:"telegram_token"`
	TelegramChat  string `yaml:"telegram_chat_id"`

	// Cặp khóa VAPID của Web Push. Máy tự sinh lần đầu cần tới, không ai
	// phải gõ. Giữ nguyên một cặp cho mãi mãi: đổi khóa là mọi máy đã bật
	// thông báo phải bật lại từ đầu, mà không ai báo cho họ biết.
	VapidPub  string `yaml:"vapid_pub"`
	VapidPriv string `yaml:"vapid_priv"`
}

var (
	biMatMu sync.RWMutex
	biMat   BiMat
)

func fileBiMat() string { return P("data/bi-mat.yaml") }

// NapBiMat đọc file lúc khởi động. Chưa có file không phải lỗi.
func NapBiMat() error {
	b, err := os.ReadFile(fileBiMat())
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var m BiMat
	if err := yaml.Unmarshal(b, &m); err != nil {
		return fmt.Errorf("%s hỏng: %w", fileBiMat(), err)
	}
	biMatMu.Lock()
	biMat = m
	biMatMu.Unlock()
	return nil
}

// BiMatHienTai trả bản sao để chỗ gọi dùng ngoài khóa.
func BiMatHienTai() BiMat {
	biMatMu.RLock()
	defer biMatMu.RUnlock()
	return biMat
}

// DatBiMat kiểm rồi ghi. Khóa API không có khoảng trắng và không xuống dòng:
// dán từ trang web hay dính cả dấu cách đuôi, cắt sẵn còn hơn để nó đi vào
// header HTTP rồi nhận về lỗi 400 khó hiểu.
func DatBiMat(m BiMat) error {
	m.GeminiKey = strings.TrimSpace(m.GeminiKey)
	m.ResendKey = strings.TrimSpace(m.ResendKey)
	m.NhaCungCap = strings.ToLower(strings.TrimSpace(m.NhaCungCap))
	m.TelegramToken = strings.TrimSpace(m.TelegramToken)
	m.TelegramChat = strings.TrimSpace(m.TelegramChat)

	for _, k := range []struct{ ten, gia string }{
		{"Khóa Gemini", m.GeminiKey}, {"Khóa Resend", m.ResendKey},
		{"Token Telegram", m.TelegramToken}, {"Chat ID Telegram", m.TelegramChat},
	} {
		if strings.ContainsAny(k.gia, " \t\r\n") {
			return fmt.Errorf("%s có khoảng trắng ở giữa — dán thiếu hay thừa?", k.ten)
		}
		if len(k.gia) > 300 {
			return fmt.Errorf("%s dài bất thường (%d ký tự)", k.ten, len(k.gia))
		}
	}
	if m.NhaCungCap != "" && m.NhaCungCap != "gemini" && m.NhaCungCap != "ollama" {
		return fmt.Errorf("nhà cung cấp lạ: %s", m.NhaCungCap)
	}

	b, err := yaml.Marshal(m)
	if err != nil {
		return err
	}
	dau := []byte("# KHÓA API. File này là bí mật: đừng commit, đừng gửi qua chat.\n" +
		"# Sửa ở /qt/cai-dat. Biến môi trường cùng tên (nếu có) THẮNG file này.\n\n")
	if err := ghiAtomic(fileBiMat(), append(dau, b...)); err != nil {
		return err
	}
	// ghiAtomic tạo file mới bằng quyền mặc định; siết lại ngay sau đó.
	// Lỗi ở đây không đáng huỷ việc lưu (Windows không có khái niệm này),
	// nhưng trên máy chủ thì phải đúng 600.
	if err := os.Chmod(fileBiMat(), 0o600); err != nil && !errors.Is(err, fs.ErrNotExist) {
		fmt.Printf("[bí mật] không siết được quyền %s: %v\n", fileBiMat(), err)
	}
	biMatMu.Lock()
	biMat = m
	biMatMu.Unlock()
	return nil
}

// --- Nơi duy nhất được đọc khóa -------------------------------------------

// GeminiKey: biến môi trường trước, rồi tới file admin.
func GeminiKey() string {
	if k := strings.TrimSpace(os.Getenv("GEMINI_API_KEY")); k != "" {
		return k
	}
	return BiMatHienTai().GeminiKey
}

func ResendKey() string {
	if k := strings.TrimSpace(os.Getenv("RESEND_API_KEY")); k != "" {
		return k
	}
	return BiMatHienTai().ResendKey
}

func TelegramToken() string {
	if k := strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")); k != "" {
		return k
	}
	return BiMatHienTai().TelegramToken
}

func TelegramChat() string {
	if k := strings.TrimSpace(os.Getenv("TELEGRAM_CHAT_ID")); k != "" {
		return k
	}
	return BiMatHienTai().TelegramChat
}

// DuPhongLLM trả model dự phòng đang có hiệu lực. Rỗng = không rơi sang đâu.
func DuPhongLLM() string {
	if BiMatHienTai().TatDuPhong {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(CFG.Model.DuPhong))
}

// cheKhoa: đủ để nhận ra mình đã dán khóa nào, không đủ để dùng lại. Khóa
// ngắn thì che hết — thà không nhận ra còn hơn lộ gần hết một chuỗi ngắn.
func cheKhoa(s string) string {
	if s == "" {
		return ""
	}
	if len(s) < 12 {
		return strings.Repeat("•", len(s))
	}
	return s[:4] + strings.Repeat("•", 8) + s[len(s)-4:]
}

// --- Trang /qt/cai-dat ----------------------------------------------------

type dlCaiDat struct {
	dlQt
	Tep string
	// Che: khóa đang dùng, đã che bớt. Rỗng = chưa có khóa nào.
	CheGemini, CheResend string
	// Env: khóa đang bị biến môi trường quyết định. Lúc đó ô nhập vẫn lưu
	// được nhưng chưa có tác dụng — phải nói ra, không thì Kendy dán khóa
	// mới rồi ngồi đoán vì sao vẫn lỗi cũ.
	EnvGemini, EnvResend bool
	NhaCungCap           string
	TatDuPhong           bool
	MacDinhCfg           string // model.mac_dinh trong config.yaml
	DuPhongCfg           string // model.du_phong
	DangDung             string // nhà cung cấp thực sự đang có hiệu lực
	KetQua               string // kết quả nút kiểm tra

	// --- Thông báo đơn hàng ---
	CheTelegram  string // token đã che bớt, rỗng = chưa khai
	ChatTelegram string
	EnvTelegram  bool
	TenBot       string // "@ten_bot", chỉ có sau khi bấm kiểm tra
	LinkBot      string
	PushMay      []PushMay
	MailChu      string // địa chỉ đang nhận báo việc, rỗng = chưa khai
	MailBat      bool
}

// thuKhoaGemini gọi endpoint liệt kê model. Nó xác thực khóa mà KHÔNG sinh
// chữ, nên không tốn request nào trong hạn mức 900/ngày của bảng quota.
func thuKhoaGemini() error {
	key := GeminiKey()
	if key == "" {
		return errors.New("chưa có khóa nào để thử")
	}
	req, err := http.NewRequest("GET", strings.TrimSuffix(geminiBase, "/")+"?key="+key, nil)
	if err != nil {
		return err
	}
	resp, err := mailClient.Do(req)
	if err != nil {
		return fmt.Errorf("không gọi được Google: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusForbidden {
		return errors.New("Google từ chối khóa này (kiểm tra lại chuỗi vừa dán)")
	}
	return fmt.Errorf("Google trả mã %d", resp.StatusCode)
}

// thuKhoaResend hỏi danh sách tên miền — không gửi mail cho ai cả.
func thuKhoaResend() error {
	key := ResendKey()
	if key == "" {
		return errors.New("chưa có khóa nào để thử")
	}
	req, err := http.NewRequest("GET", "https://api.resend.com/domains", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+key)
	resp, err := mailClient.Do(req)
	if err != nil {
		return fmt.Errorf("không gọi được Resend: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return nil
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return errors.New("Resend từ chối khóa này")
	}
	return fmt.Errorf("Resend trả mã %d", resp.StatusCode)
}

func hQtCaiDat(w http.ResponseWriter, r *http.Request) {
	d := dlCaiDat{dlQt: dlQt{Chung: chung(r, "cai-dat")}, Tep: "data/bi-mat.yaml"}

	if r.Method == http.MethodPost {
		r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
		if err := r.ParseForm(); err != nil {
			d.Loi = "Không đọc được biểu mẫu."
		} else {
			m := BiMatHienTai()
			// capNhat: việc này có ghi lại data/bi-mat.yaml hay không. Vài nút
			// trên trang chỉ đụng tới danh sách máy nhận push.
			capNhat := true
			viec := r.FormValue("viec")
			// Nút "bỏ máy này" mang theo endpoint trong value của chính nó,
			// nên nó không gửi được trường viec. Xem ghi chú ở qt-caidat.html.
			if strings.TrimSpace(r.FormValue("xoa_push")) != "" {
				viec = "xoa-push"
			}
			switch viec {
			case "xoa-gemini":
				m.GeminiKey = ""
			case "xoa-resend":
				m.ResendKey = ""
			case "xoa-telegram":
				m.TelegramToken, m.TelegramChat = "", ""
			case "xoa-push":
				// Xóa một máy khỏi danh sách nhận push. Không đụng bi-mat.yaml
				// nên thoát sớm, đừng ghi lại file khóa vì một việc khác hẳn.
				if err := XoaPushMay(strings.TrimSpace(r.FormValue("xoa_push"))); err != nil {
					d.Loi = err.Error()
				} else {
					d.OK = "Đã bỏ máy đó khỏi danh sách nhận thông báo."
				}
				capNhat = false
			default:
				// Ô để trống nghĩa là GIỮ NGUYÊN khóa cũ, không phải xóa —
				// trang không hiện khóa đầy đủ nên không ai gõ lại được nó.
				// Muốn bỏ khóa thì bấm nút Xóa.
				if v := strings.TrimSpace(r.FormValue("gemini")); v != "" {
					m.GeminiKey = v
				}
				if v := strings.TrimSpace(r.FormValue("resend")); v != "" {
					m.ResendKey = v
				}
				if v := strings.TrimSpace(r.FormValue("telegram_token")); v != "" {
					m.TelegramToken = v
				}
				// Chat id thì ngược lại với khóa: nó KHÔNG bí mật, trang hiện
				// đủ số, nên ô trống đọc là "bỏ chat id" chứ không phải "giữ".
				//
				// Has() chứ không FormValue(): trang có hai biểu mẫu rời nhau,
				// và biểu mẫu Telegram không gửi kèm ô chọn model. Đọc thẳng
				// FormValue thì mỗi lần bấm nút bên này là model bên kia bị
				// đặt về mặc định, mà không có gì báo.
				if r.Form.Has("telegram_chat") {
					m.TelegramChat = strings.TrimSpace(r.FormValue("telegram_chat"))
				}
				if r.Form.Has("nha_cung_cap") {
					m.NhaCungCap = r.FormValue("nha_cung_cap")
					m.TatDuPhong = r.FormValue("tat_du_phong") == "1"
				}
			}
			if capNhat {
				if err := DatBiMat(m); err != nil {
					d.Loi = err.Error()
				} else {
					d.OK = "Đã lưu. Có hiệu lực ngay, không cần khởi động lại."
				}
			}
		}

		// Thử khóa sau khi đã lưu, để nút "lưu rồi thử" trong một lần bấm.
		if d.Loi == "" {
			switch r.FormValue("viec") {
			case "thu-gemini":
				if err := thuKhoaGemini(); err != nil {
					d.Loi = "Khóa Gemini: " + err.Error()
				} else {
					d.KetQua = "Khóa Gemini dùng được."
				}
			case "thu-resend":
				if err := thuKhoaResend(); err != nil {
					d.Loi = "Khóa Resend: " + err.Error()
				} else {
					d.KetQua = "Khóa Resend dùng được."
				}
			case "do-chat":
				// Đọc tin gần nhất ai nhắn cho bot rồi LƯU LUÔN chat id. Bắt
				// Kendy chép tay một dãy số chín chữ số là chỗ hay gõ nhầm
				// nhất trong cả trang này.
				id, ten, err := DoChatTelegram()
				if err != nil {
					d.Loi = "Telegram: " + err.Error()
				} else {
					m := BiMatHienTai()
					m.TelegramChat = id
					if err := DatBiMat(m); err != nil {
						d.Loi = err.Error()
					} else {
						d.KetQua = "Đã lấy chat id " + id
						if ten != "" {
							d.KetQua += " (" + ten + ")"
						}
					}
				}
			case "thu-telegram":
				if ten, err := TenBotTelegram(); err != nil {
					d.Loi = "Telegram: " + err.Error()
				} else {
					d.TenBot = ten
					d.LinkBot = linkBotTelegram(ten)
					d.KetQua = "Token đúng, bot là " + ten + "."
				}
			case "gui-thu":
				ke, di := BaoTinThu(SuKien{
					Loai:   TinThu,
					TieuDe: "Thử thông báo từ " + tenTram(),
					Than: "Đây là tin thử, không có đơn nào cả.\n" +
						"Nhận được tin này nghĩa là kênh đó đã thông.",
					Duong: "/qt/cai-dat",
				})
				if di {
					d.KetQua = ke
				} else {
					d.Loi = ke
				}
			}
		}
	}

	m := BiMatHienTai()
	d.CheGemini, d.CheResend = cheKhoa(GeminiKey()), cheKhoa(ResendKey())
	d.EnvGemini = strings.TrimSpace(os.Getenv("GEMINI_API_KEY")) != ""
	d.EnvResend = strings.TrimSpace(os.Getenv("RESEND_API_KEY")) != ""
	d.NhaCungCap, d.TatDuPhong = m.NhaCungCap, m.TatDuPhong
	d.CheTelegram, d.ChatTelegram = cheKhoa(TelegramToken()), TelegramChat()
	d.EnvTelegram = strings.TrimSpace(os.Getenv("TELEGRAM_BOT_TOKEN")) != ""
	d.PushMay = DanhSachPushMay()
	d.MailChu, d.MailBat = emailChu(), MailBat()
	d.MacDinhCfg = strings.ToLower(strings.TrimSpace(CFG.Model.MacDinh))
	d.DuPhongCfg = strings.ToLower(strings.TrimSpace(CFG.Model.DuPhong))
	d.DangDung = Provider()
	render(w, "qt-caidat.html", d)
}
