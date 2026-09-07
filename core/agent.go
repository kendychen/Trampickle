// Technical Agent — trợ lý chẩn đoán, dùng NỘI BỘ, không nói chuyện với khách.
//
// Thứ tự ưu tiên khi ngữ cảnh chật (Ollama num_ctx 4096 chỉ chứa ~5.200 ký
// tự tiếng Việt):
//  1. checklist từ chối — NGUYÊN VẸN, không bao giờ bị cắt
//  2. các đoạn RAG lấy về, điểm cao trước
//  3. hết chỗ thì dừng, và NÓI RA đã bỏ gì
//
// Cắt âm thầm rồi trả lời tự tin dựa trên nửa kho là dạng lỗi tệ nhất, vì
// nhìn không ra. Ở đây cắt tường minh và báo ra màn hình.
package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const heThongKyThuat = `Bạn là trợ lý kỹ thuật của một trạm sửa vợt pickleball, nói
chuyện với NGƯỜI THỢ chứ không phải với khách.

Luật:
- Chỉ dựa vào tài liệu và các ca đã ghi ở dưới. Không có căn cứ thì nói
  "chưa có ca nào tương tự trong kho" — đó là câu trả lời hợp lệ.
- Khi checklist từ chối áp dụng được, nói TỪ CHỐI trước mọi thứ khác.
- TUYỆT ĐỐI không đưa ra con số tiền. Giá do phần mềm tính, không phải bạn.
- Dẫn ra mã ca cụ thể khi viện dẫn kinh nghiệm cũ, ví dụ (2026-09-03-002).
- Nói ngắn. Người thợ đang cầm cây vợt trên tay.`

// Tiếng Việt có dấu tốn token hơn tiếng Anh. 2 ký tự/token là ước lượng
// thận trọng — thà tự cắt sớm còn hơn để nhà cung cấp cắt hộ mà không nói.
const kyTuMoiToken = 2
const tokenChuaLai = 1500

func LoadCases() []map[string]any {
	dir := P(CFG.DuongDan.CaSua)
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	names := []string{}
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	out := []map[string]any{}
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(dir, n))
		if err != nil {
			continue
		}
		var m map[string]any
		if json.Unmarshal(b, &m) != nil {
			fmt.Printf("[cảnh báo] Bỏ qua file ca hỏng: %s\n", n)
			continue
		}
		out = append(out, m)
	}
	return out
}

// nganSachKyTu — chỗ chứa được, theo nhà cung cấp đang dùng.
func nganSachKyTu() int {
	if Provider() == "ollama" {
		n := CFG.Model.Ollama.NumCtx - tokenChuaLai
		if n < 0 {
			n = 0
		}
		return n * kyTuMoiToken
	}
	return 800000 // Gemini: cửa sổ triệu token, thực tế không chạm trần
}

type TraLoiAgent struct {
	CauTraLoi   string      `json:"cau_tra_loi"`
	NhaCungCap  string      `json:"nha_cung_cap"`
	DoanDaDung  []KetQuaTim `json:"doan_da_dung"`
	DaBoBotDoan int         `json:"da_bo_bot_doan"`
	CanhBao     []string    `json:"canh_bao"`
	DungRag     bool        `json:"dung_rag"`
}

// HoiKyThuat trả lời câu hỏi kỹ thuật, kèm đúng những đoạn đã dùng để trả
// lời. Người thợ phải kiểm tra được agent lấy căn cứ từ đâu.
func HoiKyThuat(cauHoi string) (*TraLoiAgent, error) {
	if !CFG.Agent.Technical.Bat {
		return nil, fmt.Errorf("Technical Agent đang tắt trong config.yaml")
	}
	if strings.TrimSpace(cauHoi) == "" {
		return nil, fmt.Errorf("chưa nhập câu hỏi")
	}

	kq := &TraLoiAgent{CanhBao: []string{}}
	nganSach := nganSachKyTu()
	daDung := 0
	phan := []string{}

	// 1. Checklist từ chối — nguyên vẹn, không thương lượng.
	for _, d := range LuonNap() {
		t := fmt.Sprintf("===== TÀI LIỆU BẮT BUỘC: %s =====\n%s", d.Ten, d.Text)
		phan = append(phan, t)
		daDung += len([]rune(t))
	}

	// 2. Các đoạn RAG. Index chưa dựng thì Search trả rỗng và ta rơi về
	//    nhồi thẳng tài liệu — vẫn chạy được, chỉ kém chính xác hơn.
	doan := Search(cauHoi, 0)
	kq.DungRag = len(doan) > 0
	if !kq.DungRag {
		kq.CanhBao = append(kq.CanhBao,
			"Chưa dựng index RAG — đang nhồi thẳng tài liệu. Bấm \"Dựng lại index\" để chính xác hơn.")
		for _, rel := range CFG.DuongDan.TaiLieuNap {
			if laLuonNap(rel) {
				continue
			}
			b, err := os.ReadFile(P(rel))
			if err != nil {
				continue
			}
			doan = append(doan, KetQuaTim{Nguon: rel, TieuDe: rel, Text: string(b)})
		}
	}

	// Cắt theo đúng thứ tự điểm, gặp đoạn không vừa là DỪNG — không nhảy
	// qua để nhét đoạn ngắn hơn ở dưới. Nhét-vừa-thì-lấy nghe như tận dụng
	// được chỗ trống, nhưng nó đổi độ liên quan lấy độ ngắn: đoạn đúng nhất
	// bị bỏ vì dài, thứ lọt vào là thứ tình cờ ngắn. Agent khi đó dẫn ra một
	// căn cứ không liên quan gì tới câu hỏi.
	boBot := 0
	for i, d := range doan {
		t := fmt.Sprintf("===== %s (%s) =====\n%s", d.TieuDe, d.Nguon, d.Text)
		if daDung+len([]rune(t)) > nganSach {
			boBot = len(doan) - i
			break
		}
		phan = append(phan, t)
		daDung += len([]rune(t))
		kq.DoanDaDung = append(kq.DoanDaDung, d)
	}
	kq.DaBoBotDoan = boBot
	if boBot > 0 {
		kq.CanhBao = append(kq.CanhBao, fmt.Sprintf(
			"Ngữ cảnh chật (%s, trần %d ký tự) — đã bỏ %d đoạn. Chạy Gemini để agent nhìn thấy cả kho.",
			Provider(), nganSach, boBot))
	}

	// 3. Cảnh báo kho còn mỏng. Đây là cảnh báo quan trọng nhất trong hệ
	//    thống lúc này: dưới ngưỡng, agent chỉ là gợi ý để nghĩ tiếp.
	soCa := len(LoadCases())
	nguong := CFG.Agent.Technical.SoCaToiThieuDeTin
	if soCa < nguong {
		kq.CanhBao = append(kq.CanhBao, fmt.Sprintf(
			"Kho mới có %d ca, dưới ngưỡng %d. Câu trả lời là gợi ý để nghĩ tiếp, KHÔNG phải căn cứ để quyết định.",
			soCa, nguong))
	}

	prompt := strings.Join(phan, "\n\n") + "\n\n===== CÂU HỎI CỦA THỢ =====\n" + cauHoi
	traLoi, nhaCC, err := Ask(prompt, heThongKyThuat)
	if err != nil {
		return nil, err
	}
	kq.CauTraLoi = traLoi
	kq.NhaCungCap = nhaCC
	return kq, nil
}

// --- Thống kê cho dashboard ----------------------------------------

type ThongKe struct {
	SoCa          int            `json:"so_ca"`
	TheoKetQua    map[string]int `json:"theo_ket_qua"`
	NguongTinDuoc int            `json:"nguong_tin_duoc"`
	ConThieuCa    int            `json:"con_thieu_ca"`
	NhaCungCapLLM string         `json:"nha_cung_cap_llm"`
	QuotaDaDung   int            `json:"quota_da_dung"`
	QuotaTran     int            `json:"quota_tran"`
	TranNguCanh   int            `json:"tran_ngu_canh_ky_tu"`
	Rag           RagStats       `json:"rag"`
	GiaiDoan      int            `json:"giai_doan"`
}

func LayThongKe() ThongKe {
	cases := LoadCases()
	theo := map[string]int{}
	for _, c := range cases {
		k := str(c["ket_qua"])
		if k == "" {
			k = "khong_ro"
		}
		theo[k]++
	}
	daDung, tran := QuotaStatus()
	thieu := CFG.Agent.Technical.SoCaToiThieuDeTin - len(cases)
	if thieu < 0 {
		thieu = 0
	}
	return ThongKe{
		SoCa: len(cases), TheoKetQua: theo,
		NguongTinDuoc: CFG.Agent.Technical.SoCaToiThieuDeTin, ConThieuCa: thieu,
		NhaCungCapLLM: Provider(), QuotaDaDung: daDung, QuotaTran: tran,
		TranNguCanh: nganSachKyTu(), Rag: RagStatus(), GiaiDoan: GiaiDoan,
	}
}
