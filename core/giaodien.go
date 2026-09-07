package core

// Chọn giao diện cho web khách.
//
// Mấy bộ giao diện dựng sẵn, khác nhau ở bảng màu VÀ ở dáng — cỡ chữ, nét
// chữ, bo góc, hình nút, cách xếp hero, cách chia ô, nhịp khối. Không khác ở
// nội dung, không khác ở HTML: toàn bộ khác biệt nằm trong css.html dưới dạng
// những khối chọn bằng một lớp trên thẻ <body>. Làm vậy để thêm hoặc bỏ một
// giao diện không phải đụng vào 30 template.
//
// Lưu ra data/giao-dien.yaml chứ không nhét vào config.yaml: config.yaml là
// thứ sửa bằng tay rồi commit, còn cái này đổi bằng nút bấm trên web và
// không nên nằm chung với cấu hình model, đường dẫn, ngưỡng.

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

type Theme struct {
	Ma   string
	Ten  string
	MoTa string
}

// ThemeCo — danh sách giao diện dựng sẵn. Thứ tự ở đây là thứ tự hiện trên
// trang chọn. Thêm giao diện mới: thêm một dòng ở đây, rồi thêm trong css.html
// BA khối cho nó, thiếu khối nào cũng bị trả lại:
//
//  1. màu   — body.theme-<ma> { --nen, --muc, --sang... }
//  2. dáng  — cỡ chữ, nét, bo góc, hình nút, nhịp khối
//  3. bố cục — CẢ MƯỜI KHỐI của trang chủ, không phải chỉ tấm hero: dải số
//     liệu, phiếu khám, hai hàng chữ-hình, bốn bước, ba cam kết, form, chân
//     trang, cộng bề ngang khung và số tầng đầu trang
//
// Đủ 1 mà thiếu 2 thì Kendy trả lại với câu "sao chỉ thay màu à". Đủ 1+2 mà
// thiếu 3 thì trả lại với câu "giao diện bố cục vẫn thế". Cả ba khối đều đổi
// được mà không đụng HTML, vì template dùng chung cho mọi giao diện.
//
// Riêng khối 3 đã bị trả lại hai lần vì làm nửa vời: lần đầu chỉ đổi tấm hero,
// lần sau đổi thêm lưới dịch vụ — vẫn là 2 trên 10 khối, mà 8 khối kia chiếm
// 80% chiều dài trang nên nhìn từ xa sáu bộ vẫn y hệt nhau. Cách kiểm duy nhất
// đáng tin: chụp nguyên trang cả sáu bộ rồi xếp cạnh nhau (scratchpad/tam.js
// và ghep.js). Nhìn từng bộ một thì bộ nào cũng thấy khác.
var ThemeCo = []Theme{
	{
		Ma:  "xuong",
		Ten: "Trạm",
		MoTa: "Nền than, màu nhấn là volt — đúng màu quả bóng pickleball. Trang chủ " +
			"và trang dịch vụ nền tối, bản vẽ vợt nét chạy dần, chữ tiêu đề to và " +
			"đậm. Bài viết giữ nền sáng để đọc lâu không mỏi.",
	},
	{
		Ma:  "xuong-sang",
		Ten: "Trạm sáng",
		MoTa: "Vẫn chất Trạm — volt, góc gắt, chữ tiêu đề bóp mạnh — nhưng bỏ " +
			"nền than: trang chủ và trang dịch vụ chạy nền sáng như các trang " +
			"còn lại. Chỉ tấm hero giữ nền tối để màu volt còn chỗ đứng.",
	},
	{
		Ma:  "tpic",
		Ten: "T-PIC",
		MoTa: "Đúng bốn màu của logo: volt quả bóng, ink xanh đen, steel cho chữ " +
			"mờ và đường kẻ. Nền trắng suốt cả site, kể cả trang chủ và trang " +
			"dịch vụ; chỉ tấm hero giữ nền ink để màu volt và logo còn chỗ đứng. " +
			"Ảnh dùng dè: một tấm cán vợt mờ sau hero và một nẹp ảnh bo quanh lưới " +
			"số liệu. Khối hiện ra bằng cách trôi lên, rê chuột thì thẻ nhấc 4px.",
	},
	{
		Ma:  "tim",
		Ten: "Tím",
		MoTa: "Hero gradient tím chuyển dần xuống nền sáng, chữ căn giữa, " +
			"nét chữ nhẹ hơn. Cả trang nền sáng, kể cả trang chủ.",
	},
	{
		Ma:  "la",
		Ten: "Sân cỏ",
		MoTa: "Xanh lá: hero là tấm xanh rừng, nền trang ngả kem chứ không " +
			"trắng tinh, bo góc to và mềm. Cả trang nền sáng, kể cả trang chủ.",
	},

	// Sáu bộ dưới đây dựng từ các trang mẫu của ngành sửa chữa. Màu và dáng
	// đo bằng máy chứ không nhìn ảnh đoán: màu đọc từ CSS của từng trang,
	// dáng đọc bằng getComputedStyle qua CDP (scratchpad/dodang.js), bố cục
	// đo bằng scratchpad/dobocuc.js. Ba mẫu đo được (cobalt, bien, kem); hai
	// mẫu chặn cả trình duyệt không giao diện lẫn curl nên bố cục của "dien"
	// và "do" là chọn theo thể loại chứ không sao từ mẫu — chỗ nào đoán đều
	// ghi rõ trong css.html. Chi tiết nguồn và số đo nằm ở đầu khối "SÁU GIAO
	// DIỆN DỰNG TỪ MẪU NGÀNH SỬA CHỮA" và trong khối DÁNG, khối BỐ CỤC của
	// từng bộ.
	{
		Ma:  "cobalt",
		Ten: "Cobalt & vàng",
		MoTa: "Ảnh phủ kín tấm hero, chữ đè lên nửa trái — bộ duy nhất " +
			"không chia hero thành hai cột. Dải số liệu là một băng mực đặc vắt " +
			"ngang, số nằm trên nhãn. Hàng chữ-hình chia đôi đều, ảnh vuông góc " +
			"chạm mép. Bốn bước bỏ hẳn nền thẻ, chỉ còn vạch vàng dựng đứng đầu " +
			"mỗi cột; ba cam kết cũng vậy. Xanh cobalt đặc, vàng chanh cho nút; " +
			"tiêu đề IN HOA nét mảnh giãn rộng, góc vuông tuyệt đối. Ảnh thật phủ " +
			"kín hero và một băng ảnh tối chạy dưới dải số liệu. Khối hiện ra bằng " +
			"cách trôi từ TRÁI, so le nhau 90ms; rê chuột thì thẻ trượt ngang chứ " +
			"không nhấc lên.",
	},
	{
		Ma:  "dien",
		Ten: "Xanh điện",
		MoTa: "Bộ căn giữa từ trên xuống dưới, và là trang DÀI NHẤT vì gần như không " +
			"khối nào chia cột. Hero một cột, chữ khổng lồ giữa màn. Lưới dịch vụ " +
			"hai cột, biểu tượng bên TRÁI chữ. Dải số liệu bỏ thẻ, còn ba con số " +
			"trần to gấp rưỡi ngăn nhau bằng kẻ dọc. Hai hàng chữ-hình xếp dọc, " +
			"ảnh nằm trên chữ. Ba cam kết kéo số 01-02-03 lên làm tiêu đề. Form " +
			"một cột giữa trang, hai thẻ phụ xuống dưới. Navy sâu, xanh điện chói. " +
			"Hero là ảnh mặt vợt nằm sau một dải tối vắt ngang; ba cam kết đứng " +
			"trên một tấm ảnh mờ và số 01-02-03 to gấp rưỡi. Khối hiện ra bằng " +
			"cách nở dần từ 94.5%, rê chuột thì nhấc 6px kèm quầng sáng.",
	},
	{
		Ma:  "bien",
		Ten: "Xanh biển",
		MoTa: "Bộ GỌN NHẤT — trang ngắn hơn các bộ khác cả nghìn pixel vì khối nào " +
			"cũng xếp được thành nhiều cột. Đầu trang HAI TẦNG: thương hiệu một " +
			"hàng, menu căn giữa hàng dưới. Hero đảo lại, hình trái chữ phải. " +
			"Lưới dịch vụ bốn ô một hàng. Dải số liệu để số BÊN TRÁI chữ chứ " +
			"không nằm trên. Bốn bước xếp 2×2, chân trang hai cột, cột phụ cạnh " +
			"form hẹp lại và đứng yên. Navy dịu, xanh biển nhạt, chữ thân 15px, " +
			"bóng tỏa đều — bộ khẽ nhất, đọc ra tin cậy hơn là mạnh mẽ. Ảnh chỉ " +
			"vào hai chỗ: nền hero và tường đồ nghề sau bản vẽ vợt; khung hình có " +
			"một tấm màu đè lấn lệch 22px sau lưng. Khối hiện ra CHỈ mờ dần, " +
			"không dịch chuyển, và rê chuột thì thẻ đứng yên — chỉ vạch trên đậm " +
			"lên. Bộ ít hiệu ứng nhất, đúng như mẫu gốc.",
	},
	{
		Ma:  "kem",
		Ten: "Kem & nắng",
		MoTa: "Khung rộng nhất trong tất cả — 1296px thay vì 1140px — và là bộ duy " +
			"nhất tách TIÊU ĐỀ KHỐI sang một cột trái riêng, nội dung chạy bên " +
			"phải, áp cho cả ba mục lớn giữa trang. Hero một cột căn trái, hình " +
			"thành băng rộng bên dưới. Dải số liệu và bốn bước đều là những mảng " +
			"màu rời nhau bo góc to, không viền không bóng, màu luân phiên; bốn " +
			"bước xếp 2×2. Form chạy hết khung, hai thẻ phụ thành hai mảng rộng " +
			"bên dưới. Ô biểu tượng dịch vụ là huy hiệu TRÒN có vành, chân trang " +
			"đứng trên một băng ảnh ấm, ảnh hero thở rất chậm. Khối hiện ra bằng " +
			"cách trôi từ PHẢI, chậm nhất trong bảy bộ (.7s). " +
			"Bộ duy nhất không nền trắng: nền kem ấm, hero chàm đêm, " +
			"nhấn vàng nắng.",
	},
	{
		Ma:  "do",
		Ten: "Đỏ thợ",
		MoTa: "Bộ XẾP DỌC: dịch vụ, số liệu, bốn bước, ba cam kết — không khối nào " +
			"chia cột, tất cả thành từng hàng ngang xếp chồng, đọc như bảng việc " +
			"dán tường trạm. Mỗi việc một hàng: biểu tượng trái, tên giữa, thời " +
			"gian và bảo hành dồn về mép phải. Số bước nằm trong ô vuông bên trái " +
			"hàng, số cam kết to mờ làm cột mốc. Tiêu đề khối tách sang cột trái. " +
			"Hai thẻ phụ nhảy LÊN TRÊN form. Khung hẹp lại còn 1040px cho hàng " +
			"không dài quá tầm mắt. Than và đỏ, tiêu đề IN HOA nét dày, góc vuông. " +
			"Khối quy trình là một DẢI ẢNH TỐI chạy hết bề ngang, chữ trắng, bốn " +
			"bước là bốn tấm trắng đặt lên trên — cả trang sáng rồi tối hẳn một " +
			"khoảng. Khối hiện ra bằng cách quét từ trái như tờ giấy chui ra khỏi " +
			"máy in, và mọi cử chỉ đều nhanh nhất (.1s).",
	},
	{
		Ma:  "cam",
		Ten: "Cam công trường",
		MoTa: "Mọi khối chia ô đều căn giữa như bảng hiệu treo tường, và khe giữa các " +
			"ô là vạch cảnh báo chéo chứ không phải đường kẻ — dịch vụ bốn ô một " +
			"hàng, dải số liệu, bốn bước, ba cam kết đều vậy. Nhãn bước là tem " +
			"cam dán chứ không phải chữ gạch chân. Hàng chữ-hình đảo chiều, khung " +
			"ảnh có vạch cam trên nóc. Thẻ phụ sang trái form, chân trang căn " +
			"giữa. Thép xám xanh và cam an toàn — tông nhà trạm rõ nhất. Hero có " +
			"một ĐƯỜNG CẮT CHÉO cứng chia đôi: nửa trái đặc màu cho chữ, nửa phải " +
			"là ảnh trạm gần như trần. Khối gửi yêu cầu in mờ ảnh đế giày. Khối " +
			"hiện ra bằng cách rơi từ trên xuống, có nảy nhẹ.",
	},
}

const themeMacDinh = "tpic"

type khoGiaoDien struct {
	Theme string `yaml:"theme"`
}

var (
	giaoDienMu    sync.RWMutex
	themeDangDung = themeMacDinh
)

func fileGiaoDien() string { return P("data/giao-dien.yaml") }

// NapGiaoDien đọc lựa chọn đã lưu. Thiếu file, file hỏng, hoặc ghi tên một
// giao diện không còn tồn tại — cả ba đều quay về mặc định thay vì làm sập
// server: đây là chuyện màu mè, không đáng để trang khách trắng xóa.
func NapGiaoDien() error {
	b, err := os.ReadFile(fileGiaoDien())
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var kho khoGiaoDien
	if err := yaml.Unmarshal(b, &kho); err != nil {
		return fmt.Errorf("data/giao-dien.yaml hỏng: %w", err)
	}
	if !CoTheme(kho.Theme) {
		return nil
	}
	giaoDienMu.Lock()
	themeDangDung = kho.Theme
	giaoDienMu.Unlock()
	return nil
}

func CoTheme(ma string) bool {
	for _, t := range ThemeCo {
		if t.Ma == ma {
			return true
		}
	}
	return false
}

func ThemeHienTai() string {
	giaoDienMu.RLock()
	defer giaoDienMu.RUnlock()
	return themeDangDung
}

func DatTheme(ma string) error {
	if !CoTheme(ma) {
		return fmt.Errorf("không có giao diện tên %q", ma)
	}
	giaoDienMu.Lock()
	themeDangDung = ma
	giaoDienMu.Unlock()

	b, err := yaml.Marshal(khoGiaoDien{Theme: ma})
	if err != nil {
		return err
	}
	dau := []byte("# Giao diện đang bật cho web khách. Đổi ở /qt/giao-dien.\n\n")
	return ghiAtomic(fileGiaoDien(), append(dau, b...))
}

// lopThan dựng lớp cho thẻ <body> của web khách.
//
// Nền tối giờ chỉ còn "trạm", và chỉ ở trang chủ với trang dịch vụ: hai trang
// đó người ta lướt, còn bài viết thì đọc — 1400 chữ trên nền đen mỏi mắt.
// Các giao diện còn lại sáng cả trang nên không có lớp này. "xuong-sang" chính
// là bản Trạm bỏ lớp tối, và T-PIC cũng đã chuyển sang nền trắng cả site
// (tấm hero vẫn tối, nhưng đó là màu của riêng khối hero chứ không phải lớp
// này) — đừng thêm hai cái đó vào đây.
func lopThan(trang, theme string) string {
	lop := "theme-" + theme
	if theme == "xuong" && (trang == "chu" || trang == "dich-vu") {
		lop += " toi"
	}
	return lop
}
