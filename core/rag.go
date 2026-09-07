// RAG — cắt đoạn, nhúng vector, tìm theo ngữ nghĩa.
//
// LUẬT CỨNG CỦA FILE NÀY
// ======================
// checklist-tu-choi.md KHÔNG BAO GIỜ đi qua retrieval. Nó luôn được nạp
// nguyên vẹn (xem LuonNap). Lý do: retrieval có thể trượt. Trượt một đoạn
// SOP thì câu trả lời thiếu chi tiết — khó chịu nhưng sửa được. Trượt luật
// từ chối thì nhận một cây vợt đáng lẽ phải trả lại: hỏng vợt của khách,
// mất tiền thật, mất uy tín thật.
//
// Vì sao không dùng thư viện vector store: với vài nghìn đoạn, cosine
// trong Go mất vài mili giây. Thêm một vector DB là thêm một thứ có thể
// hỏng, để giải quyết một vấn đề hiệu năng chưa tồn tại.
package core

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Tài liệu luôn nạp nguyên vẹn, không qua retrieval.
var LuonNapFiles = []string{"vanhanh/checklist-tu-choi.md"}

type Doan struct {
	Nguon  string    `json:"nguon"`
	TieuDe string    `json:"tieu_de"`
	Text   string    `json:"text"`
	Hash   string    `json:"hash"`
	Vec    []float32 `json:"vec"`
}

type Index struct {
	NhaCungCap string `json:"nha_cung_cap"`
	SoChieu    int    `json:"so_chieu"`
	NgayDung   string `json:"ngay_dung"`
	Doan       []Doan `json:"doan"`
}

type KetQuaTim struct {
	Nguon  string
	TieuDe string
	Text   string
	Diem   float64
}

// --- Cắt đoạn -------------------------------------------------------

var reHeading = regexp.MustCompile(`(?m)^#{2,3} `)
var reFirstHeading = regexp.MustCompile(`^#{1,6} +(.+)`)

// chunkMarkdown cắt theo tiêu đề, không cắt theo số ký tự cố định.
//
// Một SOP bị cắt đôi giữa chừng là một SOP vô dụng: nửa đầu nói chuẩn bị
// gì, nửa sau nói làm thế nào. Lấy về một nửa còn tệ hơn không lấy gì, vì
// nó trông như một câu trả lời đầy đủ.
func chunkMarkdown(text, nguon string) []Doan {
	maxKyTu := CFG.Rag.MaxKyTuMoiDoan
	chongLan := CFG.Rag.ChongLanKyTu

	// Tách trước mỗi tiêu đề ## / ###, giữ lại dòng tiêu đề
	loc := reHeading.FindAllStringIndex(text, -1)
	parts := []string{}
	truoc := 0
	for _, l := range loc {
		if l[0] > truoc {
			parts = append(parts, text[truoc:l[0]])
		}
		truoc = l[0]
	}
	parts = append(parts, text[truoc:])

	out := []Doan{}
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		tieuDe := nguon
		if m := reFirstHeading.FindStringSubmatch(part); m != nil {
			tieuDe = strings.TrimSpace(m[1])
		}

		if len([]rune(part)) <= maxKyTu {
			out = append(out, Doan{Nguon: nguon, TieuDe: tieuDe, Text: part})
			continue
		}

		// Đoạn quá dài: cắt theo đoạn văn, chồng lấn để không đứt mạch
		buf := []string{}
		cur := ""
		for _, para := range strings.Split(part, "\n\n") {
			if len([]rune(cur))+len([]rune(para))+2 > maxKyTu && cur != "" {
				buf = append(buf, strings.TrimSpace(cur))
				r := []rune(cur)
				if len(r) > chongLan {
					r = r[len(r)-chongLan:]
				}
				cur = string(r) + "\n\n" + para
			} else if cur == "" {
				cur = para
			} else {
				cur = cur + "\n\n" + para
			}
		}
		if strings.TrimSpace(cur) != "" {
			buf = append(buf, strings.TrimSpace(cur))
		}
		for i, b := range buf {
			out = append(out, Doan{
				Nguon:  nguon,
				TieuDe: fmt.Sprintf("%s (%d/%d)", tieuDe, i+1, len(buf)),
				Text:   b,
			})
		}
	}
	return out
}

// chunkCase — một ca sửa là một đoạn. Ca đã ngắn, cắt nhỏ hơn là mất mạch.
func chunkCase(c map[string]any) Doan {
	ma := str(c["ma_ca"])
	if ma == "" {
		ma = "?"
	}
	dong := []string{fmt.Sprintf("CA %s — %s", ma, str(c["ngay"]))}
	for _, k := range []string{
		"vot_hang", "vot_model", "trieu_chung", "chan_doan", "dich_vu",
		"ket_qua", "ly_do_tu_choi", "vat_tu_dung", "sai_lam", "lan_sau_lam_khac",
	} {
		if v, ok := c[k]; ok && v != nil {
			s := str(v)
			if s != "" && s != "[]" {
				dong = append(dong, fmt.Sprintf("%s: %s", k, s))
			}
		}
	}
	if t, ok := c["khoi_luong_truoc_g"].(float64); ok {
		if s, ok2 := c["khoi_luong_sau_g"].(float64); ok2 {
			dong = append(dong, fmt.Sprintf("khối lượng: %.2fg -> %.2fg (%+.2fg)", t, s, s-t))
		}
	}
	nhan := str(c["chan_doan"])
	if nhan == "" {
		nhan = str(c["trieu_chung"])
	}
	if nhan == "" {
		nhan = "?"
	}
	return Doan{
		Nguon:  "ca/" + ma,
		TieuDe: fmt.Sprintf("Ca %s: %s", ma, nhan),
		Text:   strings.Join(dong, "\n"),
	}
}

func str(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case float64:
		if t == math.Trunc(t) {
			return fmt.Sprintf("%d", int64(t))
		}
		return fmt.Sprintf("%g", t)
	case []any:
		s := []string{}
		for _, x := range t {
			s = append(s, str(x))
		}
		return strings.Join(s, ", ")
	default:
		return fmt.Sprintf("%v", t)
	}
}

// --- Nhúng vector ---------------------------------------------------

func EmbedProvider() string {
	want := strings.ToLower(strings.TrimSpace(CFG.Rag.NhaCungCap))
	if want == "tu_dong" || want == "" {
		if GeminiKey() != "" {
			return "gemini"
		}
		return "ollama"
	}
	return want
}

type gmEmbedReq struct {
	Model                string    `json:"model"`
	Content              gmContent `json:"content"`
	OutputDimensionality int       `json:"outputDimensionality,omitempty"`
}
type gmEmbedResp struct {
	Embedding struct {
		Values []float32 `json:"values"`
	} `json:"embedding"`
}

func embedGemini(texts []string) ([][]float32, error) {
	key := GeminiKey()
	if key == "" {
		return nil, fmt.Errorf("thiếu GEMINI_API_KEY trong .env")
	}
	c := CFG.Rag.Gemini
	url := geminiBase + c.TenModel + ":embedContent?key=" + key
	out := make([][]float32, 0, len(texts))
	for i, t := range texts {
		req := gmEmbedReq{
			Model:                "models/" + c.TenModel,
			Content:              gmContent{Parts: []gmPart{{Text: t}}},
			OutputDimensionality: c.SoChieu,
		}
		var resp gmEmbedResp
		if err := postJSON(url, req, 60*time.Second, &resp); err != nil {
			return nil, fmt.Errorf("Gemini embedding lỗi ở đoạn %d: %w", i+1, err)
		}
		if len(resp.Embedding.Values) == 0 {
			return nil, fmt.Errorf("Gemini trả vector rỗng ở đoạn %d", i+1)
		}
		out = append(out, chuanHoa(resp.Embedding.Values))
	}
	return out, nil
}

type olEmbedReq struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}
type olEmbedResp struct {
	Embedding []float32 `json:"embedding"`
}

func embedOllama(texts []string) ([][]float32, error) {
	c := CFG.Rag.Ollama
	url := OllamaHost() + "/api/embeddings"
	out := make([][]float32, 0, len(texts))
	for i, t := range texts {
		var resp olEmbedResp
		err := postJSON(url, olEmbedReq{Model: c.TenModel, Prompt: t},
			time.Duration(c.TimeoutGiay)*time.Second, &resp)
		if err != nil {
			return nil, fmt.Errorf(
				"Ollama embedding hỏng (%s). Đã tải model chưa? -> ollama pull %s\n%w",
				OllamaHost(), c.TenModel, err)
		}
		if len(resp.Embedding) == 0 {
			return nil, fmt.Errorf("Ollama trả vector rỗng ở đoạn %d", i+1)
		}
		out = append(out, chuanHoa(resp.Embedding))
	}
	return out, nil
}

// chuanHoa đưa vector về độ dài 1 ngay lúc nhúng, để lúc tìm chỉ còn phép
// nhân vô hướng. Rẻ hơn tính chuẩn lại cho từng đoạn ở mỗi câu hỏi.
func chuanHoa(v []float32) []float32 {
	var s float64
	for _, x := range v {
		s += float64(x) * float64(x)
	}
	if s == 0 {
		return v
	}
	n := float32(math.Sqrt(s))
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = x / n
	}
	return out
}

func Embed(texts []string, provider string) ([][]float32, error) {
	if provider == "" {
		provider = EmbedProvider()
	}
	switch provider {
	case "gemini":
		return embedGemini(texts)
	case "ollama":
		return embedOllama(texts)
	}
	return nil, fmt.Errorf("không biết nhà cung cấp embedding: %s", provider)
}

// --- Index ----------------------------------------------------------

func indexPath() string { return P(CFG.DuongDan.RagIndex) }

func bam(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])[:16]
}

func LoadIndex() *Index {
	b, err := os.ReadFile(indexPath())
	if err != nil {
		return nil
	}
	var idx Index
	if json.Unmarshal(b, &idx) != nil {
		fmt.Println("[rag] Index hỏng, coi như chưa có.")
		return nil
	}
	return &idx
}

// thuThap gom mọi thứ đáng đưa vào index.
func thuThap() ([]Doan, error) {
	doan := []Doan{}

	for _, rel := range CFG.DuongDan.TaiLieuNap {
		if laLuonNap(rel) {
			continue // nạp nguyên vẹn, đưa vào index chỉ tổ trùng
		}
		b, err := os.ReadFile(P(rel))
		if err != nil {
			continue
		}
		doan = append(doan, chunkMarkdown(string(b), rel)...)
	}

	kho := P(CFG.DuongDan.KhoTriThuc)
	filepath.Walk(kho, func(p string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(p))
		if ext != ".md" && ext != ".txt" {
			return nil
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return nil
		}
		rel, _ := filepath.Rel(kho, p)
		nguon := "kho-tri-thuc/" + filepath.ToSlash(rel)
		doan = append(doan, chunkMarkdown(string(b), nguon)...)
		return nil
	})

	for _, c := range LoadCases() {
		doan = append(doan, chunkCase(c))
	}
	return doan, nil
}

func laLuonNap(rel string) bool {
	for _, x := range LuonNapFiles {
		if x == rel {
			return true
		}
	}
	return false
}

type ThongKeBuild struct {
	SoDoan     int      `json:"so_doan"`
	SoDoanMoi  int      `json:"so_doan_moi"`
	SoChieu    int      `json:"so_chieu"`
	NhaCungCap string   `json:"nha_cung_cap"`
	Nguon      []string `json:"nguon"`
	DuongDan   string   `json:"duong_dan"`
}

// BuildIndex dựng lại index. Đoạn nào không đổi thì giữ vector cũ — nhúng
// lại toàn bộ mỗi lần là đốt quota vào việc đã làm rồi.
func BuildIndex(lamLai bool, log func(string)) (*ThongKeBuild, error) {
	if log == nil {
		log = func(string) {}
	}
	p := EmbedProvider()
	cu := map[string][]float32{}
	if !lamLai {
		if old := LoadIndex(); old != nil && old.NhaCungCap == p {
			for _, d := range old.Doan {
				if len(d.Vec) > 0 {
					cu[d.Hash] = d.Vec
				}
			}
		}
	}

	doan, err := thuThap()
	if err != nil {
		return nil, err
	}
	if len(doan) == 0 {
		return nil, fmt.Errorf("không có tài liệu nào để index")
	}
	for i := range doan {
		doan[i].Hash = bam(doan[i].Text)
	}

	canNhung := []int{}
	for i, d := range doan {
		if _, co := cu[d.Hash]; !co {
			canNhung = append(canNhung, i)
		}
	}
	if len(canNhung) > 0 {
		log(fmt.Sprintf("Nhúng %d đoạn mới qua %s...", len(canNhung), p))
		texts := make([]string, len(canNhung))
		for i, idx := range canNhung {
			texts[i] = doan[idx].Text
		}
		vecs, err := Embed(texts, p)
		if err != nil {
			return nil, err
		}
		for i, idx := range canNhung {
			cu[doan[idx].Hash] = vecs[i]
		}
	}
	for i := range doan {
		doan[i].Vec = cu[doan[i].Hash]
	}

	idx := Index{
		NhaCungCap: p,
		SoChieu:    len(doan[0].Vec),
		NgayDung:   time.Now().Format("2006-01-02 15:04"),
		Doan:       doan,
	}
	b, err := json.Marshal(idx)
	if err != nil {
		return nil, err
	}
	os.MkdirAll(filepath.Dir(indexPath()), 0o755)
	if err := os.WriteFile(indexPath(), b, 0o644); err != nil {
		return nil, err
	}

	set := map[string]bool{}
	for _, d := range doan {
		set[d.Nguon] = true
	}
	nguon := []string{}
	for k := range set {
		nguon = append(nguon, k)
	}
	sort.Strings(nguon)

	return &ThongKeBuild{
		SoDoan: len(doan), SoDoanMoi: len(canNhung), SoChieu: idx.SoChieu,
		NhaCungCap: p, Nguon: nguon, DuongDan: indexPath(),
	}, nil
}

// --- Tìm ------------------------------------------------------------

func nhanVoHuong(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}
	var s float64
	for i := range a {
		s += float64(a[i]) * float64(b[i])
	}
	return s
}

// Search trả các đoạn giống câu hỏi nhất.
//
// Chưa dựng index -> trả rỗng, KHÔNG lỗi. Tầng trên tự quay về nhồi thẳng
// tài liệu vào prompt. Hệ thống phải chạy được cả khi chưa ai bấm index.
func Search(cauHoi string, k int) []KetQuaTim {
	idx := LoadIndex()
	if idx == nil || len(idx.Doan) == 0 {
		return nil
	}
	if k <= 0 {
		k = CFG.Rag.SoDoanLayVe
	}
	qv, err := Embed([]string{cauHoi}, idx.NhaCungCap)
	if err != nil {
		fmt.Printf("[rag] Không nhúng được câu hỏi (%v) — bỏ qua retrieval.\n", err)
		return nil
	}
	out := []KetQuaTim{}
	for _, d := range idx.Doan {
		s := nhanVoHuong(qv[0], d.Vec)
		if s >= CFG.Rag.DiemToiThieu {
			out = append(out, KetQuaTim{d.Nguon, d.TieuDe, d.Text, s})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Diem > out[j].Diem })
	if len(out) > k {
		out = out[:k]
	}
	return out
}

// LuonNap đọc các tài liệu bắt buộc, không qua retrieval.
func LuonNap() []struct{ Ten, Text string } {
	out := []struct{ Ten, Text string }{}
	for _, rel := range LuonNapFiles {
		if b, err := os.ReadFile(P(rel)); err == nil {
			out = append(out, struct{ Ten, Text string }{rel, string(b)})
		}
	}
	return out
}

type RagStats struct {
	DaDungIndex bool           `json:"da_dung_index"`
	NhaCungCap  string         `json:"nha_cung_cap"`
	SoChieu     int            `json:"so_chieu"`
	SoDoan      int            `json:"so_doan"`
	NgayDung    string         `json:"ngay_dung"`
	TheoNguon   map[string]int `json:"theo_nguon"`
	LuonNap     []string       `json:"luon_nap"`
}

func RagStatus() RagStats {
	idx := LoadIndex()
	if idx == nil {
		return RagStats{DaDungIndex: false, NhaCungCap: EmbedProvider(), LuonNap: LuonNapFiles}
	}
	theo := map[string]int{}
	for _, d := range idx.Doan {
		theo[d.Nguon]++
	}
	return RagStats{
		DaDungIndex: true, NhaCungCap: idx.NhaCungCap, SoChieu: idx.SoChieu,
		SoDoan: len(idx.Doan), NgayDung: idx.NgayDung, TheoNguon: theo,
		LuonNap: LuonNapFiles,
	}
}
