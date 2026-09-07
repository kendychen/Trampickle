package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Bản kê khai phải mở được ra một app cài được: Chrome đọc nó, thiếu một khoá
// là im lặng không mời cài, không báo gì cả.
func TestManifestDuDeCaiDuoc(t *testing.T) {
	mux, ck := dungTrangThu(t, "chu")

	r := httptest.NewRequest("GET", "/manifest.webmanifest", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("manifest trả %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/manifest+json") {
		t.Errorf("kiểu nội dung %q, Chrome đòi application/manifest+json", ct)
	}

	var m struct {
		Name      string `json:"name"`
		ShortName string `json:"short_name"`
		StartURL  string `json:"start_url"`
		Scope     string `json:"scope"`
		Display   string `json:"display"`
		MauNen    string `json:"background_color"`
		MauNhan   string `json:"theme_color"`
		Icons     []struct {
			Src   string `json:"src"`
			Sizes string `json:"sizes"`
			MucDi string `json:"purpose"`
		} `json:"icons"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("manifest không phải JSON hợp lệ: %v", err)
	}
	if m.StartURL != "/app" {
		t.Errorf("start_url = %q, phải là /app — cài xong bấm icon là vào thẳng màn hình app", m.StartURL)
	}
	if m.Display != "standalone" {
		t.Errorf("display = %q, không standalone thì mở ra vẫn còn thanh địa chỉ", m.Display)
	}
	if m.Scope != "/" {
		t.Errorf("scope = %q, hẹp hơn / thì bấm sang trang dịch vụ là app nhảy ra trình duyệt", m.Scope)
	}
	if m.ShortName == "" || m.Name == "" {
		t.Error("thiếu tên, icon dưới màn hình chính sẽ không có nhãn")
	}
	if m.MauNen == "" || m.MauNhan == "" {
		t.Error("thiếu màu nền/màu nhấn, splash screen lúc mở app sẽ trắng trơn")
	}
	if len(m.Icons) == 0 {
		t.Fatal("không có icon nào thì Chrome từ chối cài")
	}
	// Phải có ít nhất một tấm vẽ được ở mọi cỡ, và một tấm cho Android cắt
	// theo hình máy. Thiếu tấm đầu thì có máy không cài được; thiếu tấm sau
	// thì icon nằm trong một ô vuông trắng giữa dàn icon bo tròn.
	var coAny, coMask bool
	for _, i := range m.Icons {
		if i.Sizes == "any" && i.MucDi != "maskable" {
			coAny = true
		}
		if i.MucDi == "maskable" {
			coMask = true
		}
	}
	if !coAny {
		t.Error("không có icon cỡ any — máy nào cũng phải cài được")
	}
	if !coMask {
		t.Error("không khai icon maskable, Android sẽ dán icon vào ô vuông trắng")
	}
}

// Service worker được phép giữ lại trang khách xem, nhưng đụng vào trang quản
// trị là có ngày Kendy sửa đơn xong tải lại vẫn thấy số cũ.
func TestServiceWorkerKhongCacheQuanTri(t *testing.T) {
	mux, ck := dungTrangThu(t, "chu")

	r := httptest.NewRequest("GET", "/sw.js", nil)
	r.AddCookie(ck)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("/sw.js trả %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); !strings.Contains(ct, "javascript") {
		t.Errorf("kiểu nội dung %q, trình duyệt không chịu chạy", ct)
	}
	js := w.Body.String()
	for _, can := range []string{"'/qt'", "'/api'", "'/tra-cuu'", "'/dang-nhap'"} {
		if !strings.Contains(js, can) {
			t.Errorf("service worker không loại trừ %s", can)
		}
	}
	if strings.Contains(js, "%!s") || strings.Contains(js, "%s") {
		t.Error("số hiệu bản chưa được thay vào, kho cache sẽ trùng tên giữa các lần deploy")
	}
	if !strings.Contains(js, "const V = '"+phienBanPWA+"'") {
		t.Error("thiếu số hiệu bản — deploy xong máy khách vẫn giữ bản cũ")
	}
}

// Màn hình app: chỉ việc làm được với cây vợt. Lọt cái nav bảy mục của web
// vào đây là hỏng đúng thứ khiến nó đáng gọi là app.
func TestManHinhApp(t *testing.T) {
	// Bảng giá thật, không bịa vài dịch vụ giả: lưới app hỏng vì một mã dịch
	// vụ không có biểu tượng thì bản bịa chẳng bao giờ bắt được.
	dungDichVuThat(t)
	mux, ck := dungTrangThu(t, "chu")
	if _, err := DatLienHe(LienHe{DienThoai: "090 123 45 67", Zalo: "0901234567"}); err != nil {
		t.Fatal(err)
	}

	than := moTrang(t, mux, ck, "/app")

	ds := DichVuDangBan(GiaiDoan)
	if len(ds) == 0 {
		t.Fatal("bảng dịch vụ thử rỗng, không kiểm được lưới")
	}
	for _, d := range ds {
		if !strings.Contains(than, d.Ten) {
			t.Errorf("lưới app thiếu %q", d.Ten)
		}
	}
	for _, can := range []string{
		`href="/manifest.webmanifest"`, // không có thì không cài được
		`href="tel:0901234567"`,        // số phải bỏ dấu cách mới bấm gọi được
		"zalo.me/0901234567",
		`href="/app/tra-cuu"`,
		"noindex", // Google index /app thì khách vào web bằng màn hình cụt
	} {
		if !strings.Contains(than, can) {
			t.Errorf("màn hình app thiếu %q", can)
		}
	}
	// Chân trang web bốn cột không được lọt vào.
	if strings.Contains(than, ND("chung.chan.cot-1")) {
		t.Error("màn hình app dính chân trang của web")
	}
	if strings.Contains(than, ND("chung.nav.bai-viet")) {
		t.Error("màn hình app dính thanh nav của web")
	}
}

// Lời mời cài phải bám cả web thường: khách gõ địa chỉ vào trang chủ trước
// chứ ít ai gõ thẳng /app.
func TestLoiMoiCaiCoOTrangWeb(t *testing.T) {
	mux, ck := dungTrangThu(t, "chu")

	than := moTrang(t, mux, ck, "/")
	for _, can := range []string{
		`rel="manifest"`,
		`name="theme-color"`,
		`id="moi-cai"`,
		"beforeinstallprompt",
		"serviceWorker",
		ND("caiapp.nut"),
	} {
		if !strings.Contains(than, can) {
			t.Errorf("trang chủ thiếu %q", can)
		}
	}
}

// Giao diện đổi ở /qt/giao-dien thì màu app đổi theo. Không thì mở app ra
// thấy một thanh trạng thái màu lạ không liên quan gì tới web.
func TestMauAppTheoGiaoDien(t *testing.T) {
	for _, ma := range []string{"tim", "la", "cam"} {
		nen, nhan := mauCuaTheme(ma)
		if nen == "" || nhan == "" || nen == nhan {
			t.Errorf("giao diện %q chưa khai đủ cặp màu", ma)
		}
	}
	for _, th := range ThemeCo {
		if _, co := mauTheme[th.Ma]; !co {
			t.Errorf("giao diện %q có trong ThemeCo mà thiếu màu trong mauTheme", th.Ma)
		}
	}
	if nen, nhan := mauCuaTheme("khong-co-that"); nen == "" || nhan == "" {
		t.Error("giao diện lạ phải rơi về cặp mặc định chứ không trả rỗng")
	}
}
