// Ngưỡng chỉ Go dùng — ghi ra data/nguong-tram.yaml, sửa ở /qt/nguong cùng
// trang với sáu ngưỡng kia.
//
// Vì sao KHÔNG nhét chung vào vanhanh/bang-gia.yaml như core/nguong.go: file
// đó là file người viết tay, nằm ngoài data/, và lần deploy nào cũng chỉ đẩy
// mỗi binary lên VPS. Thêm một khoá mới vào bản trên máy Kendy thì bản trên
// máy chủ vẫn không có dòng ấy — SuaNguong đi tìm dòng để sửa sẽ báo "không
// thấy dòng", còn lúc khởi động yaml trả về 0 cho khoá thiếu, tức là mức
// giảm khách quen thành 0% mà không ai biết. Một file do chính Go tạo ra thì
// máy nào chạy cũng có, thiếu thì rơi về mặc định khai ngay dưới đây.
//
// Bang-gia.yaml vẫn là nơi duy nhất của những con số Python cũng đọc. Chỗ
// này chỉ chứa thứ Python không bao giờ nhìn tới.
package core

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type NguongTram struct {
	// GiamKhachQuenPhanTram: mức điền sẵn vào ô giảm giá khi khách ở nhóm
	// "than". Điền sẵn chứ không tự áp — xem GoiYGiamGia.
	GiamKhachQuenPhanTram int `yaml:"giam_khach_quen_phan_tram" json:"giam_khach_quen_phan_tram"`
	GiamClbPhanTram       int `yaml:"giam_clb_phan_tram" json:"giam_clb_phan_tram"`
	// XongBoQuenNgay: sửa xong bao nhiêu ngày mà chưa ai lấy thì tin nhắc
	// hằng ngày kể tên ra. 0 tắt hẳn — xem core/boquen.go.
	XongBoQuenNgay int `yaml:"xong_bo_quen_ngay" json:"xong_bo_quen_ngay"`
}

// Khách lẻ không có mặt ở đây: không khai tức là 0%.
var macDinhNguongTram = NguongTram{
	GiamKhachQuenPhanTram: 10,
	GiamClbPhanTram:       15,
	XongBoQuenNgay:        7,
}

var (
	ngTramMu sync.RWMutex
	ngTram   = macDinhNguongTram
)

func fileNguongTram() string { return P("data/nguong-tram.yaml") }

// NapNguongTram: chưa có file thì về mặc định, không phải lỗi.
func NapNguongTram() error {
	b, err := os.ReadFile(fileNguongTram())
	if errors.Is(err, fs.ErrNotExist) {
		ngTramMu.Lock()
		ngTram = macDinhNguongTram
		ngTramMu.Unlock()
		return nil
	}
	if err != nil {
		return err
	}
	// Unmarshal đè lên bản mặc định chứ không lên struct rỗng: khoá nào file
	// chưa có thì giữ mặc định thay vì tụt về 0.
	f := macDinhNguongTram
	if err := yaml.Unmarshal(b, &f); err != nil {
		return fmt.Errorf("%s hỏng: %w", fileNguongTram(), err)
	}
	ngTramMu.Lock()
	ngTram = f
	ngTramMu.Unlock()
	return nil
}

func ghiNguongTram(n NguongTram) error {
	b, err := yaml.Marshal(n)
	if err != nil {
		return err
	}
	dau := []byte("# Ngưỡng nội bộ của trạm. Sửa ở /qt/nguong.\n\n")
	return ghiAtomic(fileNguongTram(), append(dau, b...))
}

func NguongTramHienTai() NguongTram {
	ngTramMu.RLock()
	defer ngTramMu.RUnlock()
	return ngTram
}

// --- Ô nhập trên trang quản trị ---------------------------------------

// Cùng khuôn ONguong nhưng đơn vị là %, nên luật hợp lệ khác: 0–100 và phải
// là số nguyên.
var nguongTramCo = []ONguong{
	{
		Khoa: "giam_khach_quen_phan_tram", Don: "%",
		Nhan: "Giảm cho khách quen",
		MoTa: "Khách đã nhận đồ xong ít nhất một lần thì tự lên nhóm khách quen. Mức này được điền sẵn vào ô giảm giá lúc lập đơn — thợ vẫn xoá được.",
	},
	{
		Khoa: "giam_clb_phan_tram", Don: "%",
		Nhan: "Giảm cho CLB / đội nhóm",
		MoTa: "Nhóm này gán tay ở trang hồ sơ khách, máy không tự đẩy ai vào.",
	},
	{
		Khoa: "xong_bo_quen_ngay", Don: "ngày", Toi: 365,
		Nhan: "Xong bao lâu chưa ai lấy thì kêu",
		MoTa: "Đơn đã sửa xong mà nằm im quá số ngày này sẽ được kể tên trong tin nhắc hằng ngày, và gom vào rổ \"Xong, chưa ai lấy\" ở trang đơn. Điền 0 để tắt hẳn.",
	},
}

func layNguongTram(n NguongTram, khoa string) int {
	switch khoa {
	case "giam_khach_quen_phan_tram":
		return n.GiamKhachQuenPhanTram
	case "giam_clb_phan_tram":
		return n.GiamClbPhanTram
	case "xong_bo_quen_ngay":
		return n.XongBoQuenNgay
	}
	return 0
}

func datNguongTram(n *NguongTram, khoa string, v int) {
	switch khoa {
	case "giam_khach_quen_phan_tram":
		n.GiamKhachQuenPhanTram = v
	case "giam_clb_phan_tram":
		n.GiamClbPhanTram = v
	case "xong_bo_quen_ngay":
		n.XongBoQuenNgay = v
	}
}

func DanhSachNguongTram() []ONguong {
	n := NguongTramHienTai()
	ra := make([]ONguong, len(nguongTramCo))
	copy(ra, nguongTramCo)
	for i := range ra {
		v := layNguongTram(n, ra[i].Khoa)
		ra[i].So = strconv.Itoa(v)
		ra[i].Dep = ra[i].So + ra[i].Don
	}
	return ra
}

// SuaNguongTram ghi lại cả file. Khác SuaNguong ở chỗ file này do Go sinh ra
// nên không có comment của người để phải giữ.
func SuaNguongTram(moi map[string]string) ([]string, error) {
	ngTramMu.Lock()
	defer ngTramMu.Unlock()

	sau := ngTram
	var doi []string
	for _, o := range nguongTramCo {
		chu, co := moi[o.Khoa]
		if !co {
			continue
		}
		chu = strings.TrimSpace(chu)
		if chu == "" {
			return nil, fmt.Errorf("%s: chưa điền", o.Nhan)
		}
		v, err := strconv.Atoi(strings.TrimSpace(strings.TrimSuffix(chu, o.Don)))
		if err != nil {
			return nil, fmt.Errorf("%s: %q không phải số nguyên", o.Nhan, chu)
		}
		if v < 0 || v > o.ToiDa() {
			return nil, fmt.Errorf("%s: phải trong khoảng 0–%d%s", o.Nhan, o.ToiDa(), o.Don)
		}
		if v == layNguongTram(ngTram, o.Khoa) {
			continue
		}
		datNguongTram(&sau, o.Khoa, v)
		doi = append(doi, o.Nhan)
	}
	if len(doi) == 0 {
		return nil, nil
	}
	if err := ghiNguongTram(sau); err != nil {
		return nil, err
	}
	ngTram = sau
	return doi, nil
}

// hNguongTramPost gom phần POST để hQtNguong gọi — tách ra cho hàm kia khỏi
// phải biết file này ghi ở đâu.
func nhanFormNguongTram(r *http.Request) map[string]string {
	moi := map[string]string{}
	for _, o := range nguongTramCo {
		if v, co := r.Form[o.Khoa]; co && len(v) > 0 {
			moi[o.Khoa] = v[0]
		}
	}
	return moi
}
