// Gọi model: gemini (mặc định) và ollama (dự phòng).
//
// LUẬT CỨNG: không hàm nào trong file này được dùng để sinh ra một con số
// tiền, một mốc thời gian cam kết, hay một điều khoản bảo hành. Số tiền do
// bảng giá YAML quyết định và do src/quote.py tính. Model chỉ diễn đạt lại.
//
// Bộ đếm quota dùng CHUNG file .quota.json với phần Python — nếu không thì
// hai bên mỗi bên đếm một nửa và cùng tin là mình còn hạn mức.
package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

var reThink = regexp.MustCompile(`(?is)<think>.*?</think>`)

type QuotaHetError struct{ Msg string }

func (e QuotaHetError) Error() string { return e.Msg }

// --- Quota ----------------------------------------------------------

type quotaFile struct {
	Ngay  string `json:"ngay"`
	SoLan int    `json:"so_lan"`
}

func quotaPath() string { return P(".quota.json") }

func quotaHomNay() int {
	b, err := os.ReadFile(quotaPath())
	if err != nil {
		return 0
	}
	var q quotaFile
	if json.Unmarshal(b, &q) != nil {
		return 0
	}
	if q.Ngay != time.Now().Format("2006-01-02") {
		return 0
	}
	return q.SoLan
}

func quotaTang() {
	q := quotaFile{Ngay: time.Now().Format("2006-01-02"), SoLan: quotaHomNay() + 1}
	if b, err := json.Marshal(q); err == nil {
		os.WriteFile(quotaPath(), b, 0o644)
	}
}

// QuotaStatus trả (đã dùng hôm nay, trần tự đặt).
func QuotaStatus() (int, int) { return quotaHomNay(), CFG.Model.Gemini.ReqMoiNgay }

// --- HTTP helper ----------------------------------------------------

func postJSON(url string, body any, timeout time.Duration, out any) error {
	b, err := json.Marshal(body)
	if err != nil {
		return err
	}
	cli := &http.Client{Timeout: timeout}
	resp, err := cli.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return QuotaHetError{Msg: fmt.Sprintf("429 hết hạn mức: %s", trim(string(data), 300))}
		}
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, trim(string(data), 400))
	}
	return json.Unmarshal(data, out)
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// --- Gemini ---------------------------------------------------------

const geminiBase = "https://generativelanguage.googleapis.com/v1beta/models/"

type gmPart struct {
	Text string `json:"text"`
}
type gmContent struct {
	Parts []gmPart `json:"parts"`
	Role  string   `json:"role,omitempty"`
}
type gmGenCfg struct {
	Temperature     float64 `json:"temperature"`
	MaxOutputTokens int     `json:"maxOutputTokens"`
}
type gmReq struct {
	Contents          []gmContent `json:"contents"`
	SystemInstruction *gmContent  `json:"systemInstruction,omitempty"`
	GenerationConfig  gmGenCfg    `json:"generationConfig"`
}
type gmResp struct {
	Candidates []struct {
		Content gmContent `json:"content"`
	} `json:"candidates"`
}

func askGemini(prompt, system string) (string, error) {
	key := GeminiKey()
	if key == "" {
		return "", fmt.Errorf("chưa có khóa Gemini — dán vào /qt/cai-dat hoặc đặt GEMINI_API_KEY")
	}
	daDung, tran := QuotaStatus()
	if daDung >= tran {
		return "", QuotaHetError{Msg: fmt.Sprintf("đã dùng %d/%d request Gemini hôm nay", daDung, tran)}
	}

	c := CFG.Model.Gemini
	req := gmReq{
		Contents:         []gmContent{{Parts: []gmPart{{Text: prompt}}, Role: "user"}},
		GenerationConfig: gmGenCfg{Temperature: c.NhietDo, MaxOutputTokens: c.TokenToiDa},
	}
	if system != "" {
		req.SystemInstruction = &gmContent{Parts: []gmPart{{Text: system}}}
	}

	var resp gmResp
	url := geminiBase + c.TenModel + ":generateContent?key=" + key
	if err := postJSON(url, req, 120*time.Second, &resp); err != nil {
		return "", err
	}
	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("Gemini trả rỗng (thường do bộ lọc an toàn)")
	}
	quotaTang()
	var sb strings.Builder
	for _, p := range resp.Candidates[0].Content.Parts {
		sb.WriteString(p.Text)
	}
	return strings.TrimSpace(sb.String()), nil
}

// --- Ollama ---------------------------------------------------------

type olMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type olOptions struct {
	NumCtx      int     `json:"num_ctx"`
	Temperature float64 `json:"temperature"`
}
type olReq struct {
	Model    string    `json:"model"`
	Messages []olMsg   `json:"messages"`
	Stream   bool      `json:"stream"`
	Options  olOptions `json:"options"`
}
type olResp struct {
	Message olMsg `json:"message"`
}

func askOllama(prompt, system string) (string, error) {
	c := CFG.Model.Ollama
	msgs := []olMsg{}
	if system != "" {
		msgs = append(msgs, olMsg{Role: "system", Content: system})
	}
	msgs = append(msgs, olMsg{Role: "user", Content: prompt})

	req := olReq{
		Model:    c.TenModel,
		Messages: msgs,
		Stream:   false,
		// num_ctx 4096 là điểm gãy trên Quadro T2000 4GB: 79 -> 40 -> 15 tok/s.
		Options: olOptions{NumCtx: c.NumCtx, Temperature: c.NhietDo},
	}
	var resp olResp
	url := OllamaHost() + "/api/chat"
	if err := postJSON(url, req, time.Duration(c.TimeoutGiay)*time.Second, &resp); err != nil {
		return "", fmt.Errorf("Ollama không phản hồi (%s): %w", OllamaHost(), err)
	}
	// qwen3 và các model reasoning nhả <think>...</think> ra ngoài
	return strings.TrimSpace(reThink.ReplaceAllString(resp.Message.Content, "")), nil
}

// --- Cửa vào chung --------------------------------------------------

// Ask gọi nhà cung cấp mặc định, tự chuyển sang dự phòng nếu hỏng.
// Trả thêm tên nhà cung cấp đã thực sự trả lời, để giao diện nói thật
// với người dùng thay vì im lặng đổi model.
func Ask(prompt, system string) (string, string, error) {
	chinh := Provider()
	var err error
	var out string

	switch chinh {
	case "gemini":
		out, err = askGemini(prompt, system)
	case "ollama":
		out, err = askOllama(prompt, system)
	default:
		return "", "", fmt.Errorf("không biết nhà cung cấp: %s", chinh)
	}
	if err == nil {
		return out, chinh, nil
	}

	loiChinh := err
	duPhong := DuPhongLLM()
	if duPhong == "" || duPhong == chinh {
		return "", chinh, loiChinh
	}
	fmt.Printf("[llm] %s hỏng (%v) -> chuyển sang %s\n", chinh, loiChinh, duPhong)

	switch duPhong {
	case "gemini":
		out, err = askGemini(prompt, system)
	case "ollama":
		out, err = askOllama(prompt, system)
	default:
		return "", chinh, fmt.Errorf("nhà cung cấp dự phòng lạ: %s", duPhong)
	}
	if err != nil {
		// Nói cả hai. Trước đây chỉ trả lỗi của dự phòng, nên màn hình báo
		// "Ollama connection refused" trong khi chuyện thật là thiếu khóa
		// Gemini — người đọc đi sửa đúng chỗ không liên quan.
		return "", duPhong, fmt.Errorf("%s hỏng: %v — dự phòng %s cũng hỏng: %v",
			chinh, loiChinh, duPhong, err)
	}
	return out, duPhong, nil
}
