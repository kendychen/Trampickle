package core

// Chữ trên các trang tĩnh — sửa được ở /qt/noi-dung, không phải build lại.
//
// Cách xếp: MẶC ĐỊNH NẰM TRONG CODE (cây CayND dưới đây), còn
// data/noi-dung.yaml chỉ giữ đúng những khoá Kendy đã sửa. Hai lý do, cả hai
// đều là bài học từ lần trước:
//
//  1. Thêm chữ mới vào template mà mặc định nằm trong yaml thì trên VPS chỗ
//     đó hiện ra TRỐNG cho tới khi ai đó nhớ vào admin gõ lại. Mặc định trong
//     code thì chữ mới lên trang ngay khi deploy.
//  2. Nâng cấp không phải trộn tay: yaml chỉ có mấy dòng Kendy đổi, không
//     phải bản sao của toàn bộ chữ trên site.
//
// Một khoá chỉ khai MỘT lần, ở CayND: khoá, nhãn hiện trong admin, chữ mặc
// định, và có phải ô nhiều dòng không. Map tra cứu dựng từ cây lúc khởi động.
// Khai hai chỗ là kiểu gì cũng có ngày nhãn nói một đằng chữ chạy một nẻo.
//
// Template gọi {{nd "khoa"}} cho chữ một dòng, {{ndm "khoa"}} cho ô nhiều
// dòng (qua markdown gọn: **đậm**, *nghiêng*, [chữ](/duong-dan), xuống dòng).
// Khoá nào có trong template mà không có trong cây thì TestKhoaNDCoDu bắt
// được — gõ sai một chữ là chữ biến mất khỏi trang, mà không ai thấy lỗi.

import (
	"fmt"
	"html/template"
	"os"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type MucND struct {
	Khoa string
	Nhan string
	Mac  string
	// MacOn: chữ mặc định khi site chạy chế độ Online. Rỗng = hai chế độ dùng
	// chung Mac. Đừng khai nó cho khoá mà hai chế độ nói y hệt nhau — mỗi ô
	// khai thêm là một ô nữa phải giữ cho khỏi cũ.
	MacOn string
	// Dai: ô nhập nhiều dòng, và dựng qua markdown gọn lúc lên trang.
	Dai bool
}

type NhomND struct {
	Ten string
	Muc []MucND
}

type TrangND struct {
	Ma   string
	Ten  string
	MoTa string
	Nhom []NhomND
	// Rieng: trang này đã có màn quản trị riêng, đừng hiện tab ở /qt/noi-dung.
	// Khoá vẫn khai ở đây như mọi khoá khác — cây là nguồn duy nhất của chữ
	// mặc định, và TestKhoaND* vẫn soi được. Chỉ khác chỗ Kendy ngồi gõ.
	//
	// Có nó vì một màn hình như app không chỉ có chữ: nó còn có ảnh banner,
	// đợt khuyến mãi, nhịp icon. Bắt Kendy sửa chữ ở trang này rồi sang trang
	// kia sửa ảnh là kiểu gì cũng có lần sửa nửa vời.
	Rieng bool
}

var (
	ndMu   sync.RWMutex
	ndSua  = map[string]string{} // chỉ những khoá Kendy đã đổi
	ndLa   = map[string]string{} // khoá trong file mà bản này không biết
	ndMac  = map[string]string{} // dựng từ CayND
	ndDai  = map[string]bool{}
	ndNhan = map[string]string{}
)

func fileND() string { return P("data/noi-dung.yaml") }

// hauToOnline nối vào khoá gốc để thành khoá của bản Online. Ký tự "@" chọn
// vì khoá thật đặt theo <trang>.<khối>.<chỗ>, không bao giờ có "@" — nên
// không đụng khoá nào.
const hauToOnline = "@online"

func init() {
	for _, t := range CayND {
		for _, n := range t.Nhom {
			for _, m := range n.Muc {
				if _, trung := ndMac[m.Khoa]; trung {
					panic("noi-dung: khoá trùng " + m.Khoa)
				}
				if strings.Contains(m.Khoa, hauToOnline) {
					panic("noi-dung: khoá tự chứa hậu tố " + m.Khoa)
				}
				ndMac[m.Khoa] = m.Mac
				ndDai[m.Khoa] = m.Dai
				ndNhan[m.Khoa] = m.Nhan
				if m.MacOn != "" {
					k := m.Khoa + hauToOnline
					ndMac[k] = m.MacOn
					ndDai[k] = m.Dai
					ndNhan[k] = m.Nhan
				}
			}
		}
	}
}

// KhoaTheoCheDo trả khoá thật sự đang dùng để tra chữ. Chế độ Online mà khoá
// không khai MacOn thì vẫn dùng khoá gốc — hai chế độ nói chung một câu.
//
// KHÔNG khoá ndMu ở đây: ndMac chỉ ghi trong init(), và ND gọi hàm này khi
// đang giữ RLock. RWMutex của Go không đệ quy — khoá thêm lần nữa là kẹt cứng.
func KhoaTheoCheDo(khoa string) string {
	if !LaOnline() {
		return khoa
	}
	if _, co := ndMac[khoa+hauToOnline]; !co {
		return khoa
	}
	return khoa + hauToOnline
}

// CoBanOnline: khoá này có khai riêng chữ cho bản Online không.
func CoBanOnline(khoa string) bool {
	_, co := ndMac[khoa+hauToOnline]
	return co
}

// NapND đọc phần Kendy đã sửa. File hỏng hoặc chưa có thì chạy bằng mặc
// định — mất mấy câu chữ đã sửa còn hơn mất cả web.
func NapND() error {
	ndMu.Lock()
	defer ndMu.Unlock()
	ndSua = map[string]string{}
	ndLa = map[string]string{}

	b, err := os.ReadFile(fileND())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var kho map[string]string
	if err := yaml.Unmarshal(b, &kho); err != nil {
		return fmt.Errorf("đọc %s: %w", fileND(), err)
	}
	for k, v := range kho {
		// Khoá lạ: không nạp, nhưng nhớ lại để lần ghi sau vẫn còn trong
		// file. Đây là dấu hiệu chạy binary cũ hơn file data (hay quên chép
		// binary mới lên VPS) — nuốt mất chữ Kendy đã sửa thì cay hơn nhiều
		// so với việc để thừa vài dòng không ai đọc.
		if _, co := ndMac[k]; !co {
			ndLa[k] = v
			continue
		}
		ndSua[k] = v
	}
	return nil
}

// ND trả chữ đang hiện cho một khoá — theo chế độ site đang chạy.
//
// Đổi khoá TRƯỚC khi khoá ndMu: KhoaTheoCheDo không được giữ ndMu, mà gọi nó
// trong vùng đã RLock cũng vẫn sai kiểu khác — cứ tách hẳn ra cho khỏi nghĩ.
func ND(khoa string) string {
	khoa = KhoaTheoCheDo(khoa)
	ndMu.RLock()
	defer ndMu.RUnlock()
	if v, co := ndSua[khoa]; co {
		return v
	}
	return ndMac[khoa]
}

// NDMac trả chữ mặc định trong code — trang quản trị cần nó để hiện nút
// "khôi phục" và để biết ô nào đang khác mặc định.
func NDMac(khoa string) string { return ndMac[khoa] }

// DaSuaND: khoá này Kendy đã đổi khác mặc định chưa. Theo chế độ đang chạy —
// admin chỉ hiện đúng ô đang sửa được.
func DaSuaND(khoa string) bool {
	khoa = KhoaTheoCheDo(khoa)
	ndMu.RLock()
	defer ndMu.RUnlock()
	_, co := ndSua[khoa]
	return co
}

// SoDaSuaND đếm số khoá đã đổi trong một trang — hiện trên tab của admin.
func SoDaSuaND(t TrangND) int {
	ndMu.RLock()
	defer ndMu.RUnlock()
	n := 0
	for _, nh := range t.Nhom {
		for _, m := range nh.Muc {
			if _, co := ndSua[KhoaTheoCheDo(m.Khoa)]; co {
				n++
			}
		}
	}
	return n
}

// DatND ghi một loạt khoá cùng lúc (một trang admin là một lần ghi). Giá trị
// bằng đúng mặc định thì XOÁ khỏi file thay vì ghi lại: file chỉ nên chứa
// những chỗ Kendy thật sự đổi, để sau này sửa mặc định trong code là chữ
// trên trang đổi theo, không bị bản sao cũ đè lên.
func DatND(moi map[string]string) error {
	ndMu.Lock()
	for k, v := range moi {
		if _, co := ndMac[k]; !co {
			continue
		}
		v = chuanND(v)
		if v == ndMac[k] {
			delete(ndSua, k)
			continue
		}
		ndSua[k] = v
	}
	err := ghiND()
	ndMu.Unlock()
	return err
}

// KhoiPhucND trả một khoá về mặc định trong code.
func KhoiPhucND(khoa string) error {
	ndMu.Lock()
	delete(ndSua, khoa)
	err := ghiND()
	ndMu.Unlock()
	return err
}

// chuanND: bỏ khoảng trắng thừa hai đầu và đổi CRLF về LF. Ô textarea trên
// trình duyệt trả về CRLF, mà mặc định trong code là LF — không chuẩn hoá thì
// một ô không đổi gì cũng bị coi là đã sửa.
func chuanND(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\r\n", "\n"))
}

// ghiND — gọi khi đang giữ ndMu.
func ghiND() error {
	// Sắp khoá cho file có thứ tự ổn định: đọc diff giữa hai lần sửa mới có
	// nghĩa, và mở file bằng tay cũng dò ra chỗ cần tìm.
	ra := make(map[string]string, len(ndSua)+len(ndLa))
	for k, v := range ndLa {
		ra[k] = v
	}
	for k, v := range ndSua {
		ra[k] = v
	}
	ks := make([]string, 0, len(ra))
	for k := range ra {
		ks = append(ks, k)
	}
	sort.Strings(ks)

	var b strings.Builder
	b.WriteString("# Chữ trên trang đã sửa ở /qt/noi-dung.\n")
	b.WriteString("# Chỉ những khoá Kendy đổi mới nằm ở đây; phần còn lại lấy mặc định\n")
	b.WriteString("# trong code (core/noidung.go). Xoá một dòng = trả khoá đó về mặc định.\n\n")
	for _, k := range ks {
		dong, err := yaml.Marshal(map[string]string{k: ra[k]})
		if err != nil {
			return err
		}
		b.Write(dong)
	}
	return ghiAtomic(fileND(), []byte(b.String()))
}

// --- Dùng trong template ------------------------------------------------

// ndDong: một dòng chữ có **đậm**, *nghiêng*, [chữ](/duong-dan). Không sinh
// thẻ khối nào nên đặt được trong <span>, <li>, <button>.
func ndDong(khoa string) template.HTML { return MarkdownDong(ND(khoa)) }

// ndKhoi: ô nhiều dòng. Markdown gọn — đoạn, danh sách, khung lưu ý — nhưng
// không có đoạn dẫn khổ to như bài viết: đoạn đầu của một ô mô tả chỉ là đoạn
// đầu. Sinh thẻ <p> nên chỗ đặt nó phải là chỗ nhận được thẻ khối.
func ndKhoi(khoa string) template.HTML { return MarkdownGon(ND(khoa)) }

// ndx như ndd nhưng vá được chỗ trống giữa câu:
//
//	{{ndx "trangchu.gui.mail-co" "email" .MailToi}}
//
// đổi {email} trong chữ thành địa chỉ thật. Có nó thì cả câu nằm gọn trong
// MỘT ô nhập; cắt câu làm đôi quanh chỗ trống thì Kendy không đảo được trật
// tự, mà nhìn vào admin cũng không đoán ra hai nửa ghép lại ra câu gì.
//
// Thay TRƯỚC khi dựng markdown, để giá trị chèn vào đi qua html.EscapeString
// như mọi chữ khác.
func ndx(khoa string, cap ...any) template.HTML {
	s := ND(khoa)
	for i := 0; i+1 < len(cap); i += 2 {
		s = strings.ReplaceAll(s, "{"+fmt.Sprint(cap[i])+"}", fmt.Sprint(cap[i+1]))
	}
	return MarkdownDong(s)
}
