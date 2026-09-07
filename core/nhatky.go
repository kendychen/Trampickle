package core

// Nhật ký hành động: ai làm gì, lúc nào, từ đâu.
//
// Vì sao cần: mật khẩu lộ là chuyện có thể xảy ra. Khi xảy ra, câu hỏi đầu
// tiên không phải "làm sao nó vào được" mà "nó đã sửa những gì". Không có
// nhật ký thì không trả lời được, và phải coi như toàn bộ dữ liệu đáng ngờ.
//
// JSONL ghi nối, một file một tháng. Không sửa, không xoá. Dùng JSONL chứ
// không phải một file JSON lớn vì ghi nối không cần đọc-sửa-ghi lại cả file,
// và file hỏng giữa chừng chỉ mất dòng cuối.

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type MucNhatKy struct {
	Luc      string `json:"luc"`
	Ai       string `json:"ai"`
	IP       string `json:"ip"`
	Viec     string `json:"viec"`
	Duong    string `json:"duong,omitempty"`
	DoiTuong string `json:"doi_tuong,omitempty"`
	KetQua   string `json:"ket_qua"`
}

// truongCam: giá trị của những trường này KHÔNG BAO GIỜ vào nhật ký. Nhật ký
// là file đọc được; để lộ mật khẩu ở đây là tự tạo ra một chỗ rò mới.
var truongCam = map[string]bool{
	"mat_khau":       true,
	"mat_khau_moi":   true,
	"mat_khau_cu":    true,
	"_csrf":          true,
	"gemini":         true,
	"resend":         true,
	"gemini_api_key": true,
	"resend_api_key": true,
	"bi_mat":         true,
	"totp":           true,
	"ma_totp":        true,
	"ma_du_phong":    true,
}

// Tên khác nhatKyMu: khoá ấy đã thuộc về nhật ký chạy tự động bên
// core/tudong.go, hai thứ không dùng chung file nên không dùng chung khoá.
var nhatKyGhiMu sync.Mutex

func thuMucNhatKy() string { return P("data/nhat-ky") }

func GhiNhatKy(m MucNhatKy) {
	if m.Luc == "" {
		m.Luc = time.Now().Format(time.RFC3339)
	}
	b, err := json.Marshal(m)
	if err != nil {
		return
	}

	nhatKyGhiMu.Lock()
	defer nhatKyGhiMu.Unlock()

	if err := os.MkdirAll(thuMucNhatKy(), 0o700); err != nil {
		return
	}
	ten := filepath.Join(thuMucNhatKy(), time.Now().Format("2006-01")+".jsonl")
	f, err := os.OpenFile(ten, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	defer f.Close()
	f.Write(append(b, '\n'))
}

// DocNhatKy trả về mới nhất trước. thang rỗng = tháng hiện tại.
func DocNhatKy(thang string, gioiHan int) []MucNhatKy {
	if thang == "" {
		thang = time.Now().Format("2006-01")
	}
	// thang đi thẳng từ query string vào tên file: chặn ../ và mọi thứ không
	// phải YYYY-MM.
	if !thangHopLe(thang) {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(thuMucNhatKy(), thang+".jsonl"))
	if err != nil {
		return nil
	}

	var ds []MucNhatKy
	for _, dong := range strings.Split(string(b), "\n") {
		if strings.TrimSpace(dong) == "" {
			continue
		}
		var m MucNhatKy
		if json.Unmarshal([]byte(dong), &m) == nil {
			ds = append(ds, m)
		}
	}
	// Đảo trước rồi mới sắp: Luc chỉ chính xác tới giây, nhiều mục trong
	// cùng một giây sẽ hoà nhau. Sort ổn định giữ nguyên thứ tự đầu vào, nên
	// phải đưa mục ghi sau lên trước ngay từ đầu thì hoà mới ra đúng chiều.
	for i, j := 0, len(ds)-1; i < j; i, j = i+1, j-1 {
		ds[i], ds[j] = ds[j], ds[i]
	}
	// Luc là RFC3339 nên so chuỗi cũng ra đúng thứ tự thời gian.
	sort.SliceStable(ds, func(i, j int) bool { return ds[i].Luc > ds[j].Luc })
	if gioiHan > 0 && len(ds) > gioiHan {
		ds = ds[:gioiHan]
	}
	return ds
}

// locTruongCam dựng chuỗi mô tả form, bỏ mọi trường bí mật.
func locTruongCam(v url.Values) string {
	khoa := make([]string, 0, len(v))
	for k := range v {
		khoa = append(khoa, k)
	}
	sort.Strings(khoa)

	var phan []string
	for _, k := range khoa {
		if truongCam[k] {
			continue
		}
		phan = append(phan, k+"="+catBot(v.Get(k), 60))
	}
	return catBot(strings.Join(phan, " "), 400)
}
