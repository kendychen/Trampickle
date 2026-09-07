package core

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PWA — cho khách cài trạm lên màn hình chính điện thoại rồi mở như một app
// riêng, không thanh địa chỉ. Ba mảnh:
//
//	/manifest.webmanifest  nói cho máy biết cài cái gì, tên gì, icon nào
//	/sw.js                 service worker, giữ vỏ app cho lúc mạng chập chờn
//	/app                   màn hình đầu tiên sau khi cài: lưới ô dịch vụ
//
// CẢ BỘ CHỈ SỐNG TRÊN HTTPS. Trên http:// trình duyệt lặng lẽ bỏ service
// worker và không bao giờ mời cài — không log, không lỗi, không cảnh báo gì.
// Bản demo đang chạy IP trần nên đừng mất công tìm bug ở đây; có tên miền và
// chứng chỉ là nó tự chạy, không phải sửa dòng nào.
//
// Ngoại lệ duy nhất: localhost được coi là an toàn, nên thử ở máy nhà vẫn cài
// được thật.

// phienBanPWA đổi mỗi lần khởi động, và tên kho cache đi kèm nó. Deploy =
// restart = tên kho mới = máy khách bỏ hết bản cũ ngay lần vào sau. Nếu để một
// hằng số cố định thì Kendy đẩy bản mới lên mà điện thoại khách vẫn hiện bản
// cũ, và không ai đoán ra vì sao.
var phienBanPWA = fmt.Sprint(time.Now().Unix())

// Màu cho splash screen lúc mở app. Phải khớp --nen và --sang của cùng mã giao
// diện trong css.html; thêm giao diện mới thì thêm một dòng ở đây, thiếu thì
// rơi về cặp mặc định chứ không vỡ.
var mauTheme = map[string][2]string{
	"xuong":      {"#0E1116", "#D6F32F"},
	"xuong-sang": {"#F4F5EF", "#D6F32F"},
	"tpic":       {"#FFFFFF", "#D6F32F"},
	"tim":        {"#FBFAFF", "#6D4AE0"},
	"la":         {"#F6F5EE", "#35D68A"},
	"cobalt":     {"#FFFFFF", "#FEDD02"},
	"dien":       {"#FFFFFF", "#0080FF"},
	"bien":       {"#FFFFFF", "#3BAED4"},
	"kem":        {"#FFF9ED", "#FFDC69"},
	"do":         {"#FFFFFF", "#E1251B"},
	"cam":        {"#FFFFFF", "#FF6B00"},
}

func mauCuaTheme(ma string) (nen, nhan string) {
	if c, ok := mauTheme[ma]; ok {
		return c[0], c[1]
	}
	return "#FFFFFF", "#0E3B4E"
}

// mauNhanTheme cho template gọi thẳng: <meta name="theme-color">. Thanh trạng
// thái của điện thoại tô theo màu này khi app đang mở.
func mauNhanTheme(ma string) string {
	_, nhan := mauCuaTheme(ma)
	return nhan
}

// coCuaIcon trả cỡ để khai trong manifest. SVG thì "any" — vẽ lại được ở mọi
// cỡ. Ảnh bitmap phải khai đúng số thật: khai "any" cho một tấm 64px thì Chrome
// nhận manifest nhưng tới lúc cài mới từ chối, mà không nói vì sao.
func coCuaIcon() string {
	if iconMIME() == "image/svg+xml" {
		return "any"
	}
	ten, _ := tepLogo(loaiIcon)
	if ten == "" {
		return "any"
	}
	f, err := os.Open(filepath.Join(thuMucLogo(), ten))
	if err != nil {
		return "any"
	}
	defer f.Close()
	cf, _, err := image.DecodeConfig(f)
	if err != nil || cf.Width <= 0 {
		return "any"
	}
	return fmt.Sprintf("%dx%d", cf.Width, cf.Height)
}

func duCoIcon(co string) bool {
	var w, h int
	if _, err := fmt.Sscanf(co, "%dx%d", &w, &h); err != nil {
		return false
	}
	return w >= 192 && h >= 192
}

type biIcon struct {
	Src   string `json:"src"`
	Sizes string `json:"sizes"`
	Type  string `json:"type,omitempty"`
	MucDi string `json:"purpose,omitempty"`
}

func hManifest(w http.ResponseWriter, r *http.Request) {
	nen, nhan := mauCuaTheme(ThemeHienTai())
	ten := CFG.ThuongHieu.Ten
	if ten == "" {
		ten = "Trạm"
	}
	tenDai := ten
	if CFG.ThuongHieu.DongMoTa != "" {
		tenDai = ten + " — " + CFG.ThuongHieu.DongMoTa
	}

	co := coCuaIcon()
	icons := []biIcon{{Src: iconURL(), Sizes: co, Type: iconMIME(), MucDi: "any"}}
	// Icon Kendy tải lên có thể là ảnh nhỏ. Chrome đòi ít nhất một tấm từ
	// 192px trở lên mới cho cài, nên kèm bản vẽ SVG làm phao: nó vẽ lại được ở
	// mọi cỡ, nằm sẵn trong binary, không phụ thuộc Kendy đã tải icon lên chưa.
	if co != "any" && !duCoIcon(co) {
		icons = append(icons, biIcon{Src: "/favicon.svg", Sizes: "any", Type: "image/svg+xml", MucDi: "any"})
	}
	// maskable: Android cắt icon theo hình của máy (tròn, vuông bo, giọt
	// nước). Không khai thì nó dán icon vào một ô vuông trắng, nhìn lệch hẳn
	// giữa dàn icon khác.
	//
	// Ưu tiên icon Kendy tải lên, nhưng chỉ khi nó đủ 192px: máy cắt xong chỉ
	// còn đường tròn bán kính 40% cạnh, icon nhỏ mà bị phóng to rồi xén thì
	// vỡ. Điều kiện ngầm ở đây là ảnh tải lên có nền phủ kín khung và chừa lề
	// quanh hình — /qt/giao-dien nói rõ chỗ đó. Không đủ thì rơi về bản vẽ
	// trong binary, nó vốn vẽ vừa vùng an toàn.
	if duCoIcon(co) {
		icons = append(icons, biIcon{Src: iconURL(), Sizes: co, Type: iconMIME(), MucDi: "maskable"})
	} else {
		icons = append(icons, biIcon{Src: "/favicon.svg", Sizes: "any", Type: "image/svg+xml", MucDi: "maskable"})
	}

	m := map[string]any{
		"id":               "/app",
		"name":             tenDai,
		"short_name":       ten,
		"description":      moTaTrang["app"],
		"start_url":        "/app",
		"scope":            "/",
		"display":          "standalone",
		"orientation":      "portrait",
		"lang":             "vi",
		"dir":              "ltr",
		"background_color": nen,
		"theme_color":      nhan,
		"icons":            icons,
		"shortcuts": []map[string]any{
			{"name": ND("app.nut.tra-cuu"), "url": "/app/tra-cuu"},
			{"name": ND("app.nut.gui-anh"), "url": "/app/gui-anh"},
		},
	}
	w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(m)
}

// hServiceWorker phục vụ /sw.js ở ngay gốc. Đặt sâu hơn một cấp thì phạm vi
// của nó co lại đúng thư mục ấy và trang chủ nằm ngoài tầm.
func hServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Service-Worker-Allowed", "/")
	fmt.Fprintf(w, swJS, phienBanPWA)
}

// Mạng trước, kho sau, và chỉ với những gì khách xem. Trang quản trị, API,
// đăng nhập, tra cứu đơn thì đi thẳng ra mạng không đụng vào kho: cache một
// trang /qt là có ngày Kendy sửa đơn xong tải lại vẫn thấy số cũ, còn cache
// trang tra cứu là khách thấy trạng thái đơn của hôm kia.
const swJS = `// Sinh bởi Go, đừng sửa tay. Nguồn: core/pwa.go
const V = '%s';
const KHO = 'tram-' + V;

// Vỏ app: đủ để mở được app khi mất mạng hẳn.
const VO = ['/app', '/favicon.svg', '/manifest.webmanifest'];

self.addEventListener('install', function (e) {
  self.skipWaiting();
  e.waitUntil(caches.open(KHO).then(function (k) {
    // addAll hỏng một tệp là hỏng cả mẻ; thà thiếu một tệp còn hơn không cài
    // được service worker.
    return Promise.all(VO.map(function (u) { return k.add(u).catch(function () {}); }));
  }));
});

self.addEventListener('activate', function (e) {
  e.waitUntil(caches.keys().then(function (ts) {
    return Promise.all(ts.map(function (t) {
      if (t !== KHO) { return caches.delete(t); }
    }));
  }).then(function () { return self.clients.claim(); }));
});

function boQua(u) {
  return u.pathname.indexOf('/qt') === 0 ||
         u.pathname.indexOf('/api') === 0 ||
         u.pathname.indexOf('/noi-bo') === 0 ||
         u.pathname.indexOf('/don-anh') === 0 ||
         u.pathname === '/dang-nhap' ||
         u.pathname === '/dang-xuat' ||
         u.pathname === '/tra-cuu' ||
         u.pathname === '/app/tra-cuu';
}

self.addEventListener('fetch', function (e) {
  var rq = e.request;
  if (rq.method !== 'GET') { return; }
  var u = new URL(rq.url);
  if (u.origin !== self.location.origin) { return; }
  if (boQua(u)) { return; }

  // Trang: mạng trước. Khách phải thấy bảng dịch vụ mới nhất — chậm nửa giây
  // còn hơn đúng của nửa tháng trước.
  if (rq.mode === 'navigate') {
    e.respondWith(
      fetch(rq).then(function (rp) {
        if (rp.ok) {
          var ban = rp.clone();
          caches.open(KHO).then(function (k) { k.put(rq, ban); });
        }
        return rp;
      }).catch(function () {
        return caches.match(rq).then(function (c) { return c || caches.match('/app'); });
      })
    );
    return;
  }

  // Ảnh và phông: kho trước. Chúng đổi hiếm, mà tải lại mỗi lần thì tốn 3G
  // của khách.
  if (rq.destination === 'image' || rq.destination === 'font') {
    e.respondWith(
      caches.match(rq).then(function (c) {
        return c || fetch(rq).then(function (rp) {
          if (rp.ok) {
            var ban = rp.clone();
            caches.open(KHO).then(function (k) { k.put(rq, ban); });
          }
          return rp;
        });
      })
    );
  }
});
`

// hApp — màn hình app. Chỉ dịch vụ, không bài viết, không giới thiệu: người
// bấm icon trên màn hình chính là đang muốn làm một việc cụ thể với cây vợt,
// không phải ngồi đọc.
func hApp(w http.ResponseWriter, r *http.Request) {
	c := chung(r, "app")
	nen, nhan := mauCuaTheme(c.Theme)
	render(w, "app.html", struct {
		Chung
		DichVu  []DichVu
		MauNen  string
		MauNhan string
	}{c, DichVuDangBan(GiaiDoan), nen, nhan})
}

// laManHinhApp — request đang xin bản app hay bản web của cùng một trang.
// /tra-cuu và /app/tra-cuu chạy chung một handler, chung một truy vấn; khác
// nhau đúng cái vỏ. Nhân đôi handler thì hai bản sẽ lệch nhau sau vài lần sửa.
func laManHinhApp(r *http.Request) bool {
	return strings.HasPrefix(r.URL.Path, "/app/")
}

// mauTheoVo chọn mẫu theo đường vào.
func mauTheoVo(r *http.Request, web, app string) string {
	if laManHinhApp(r) {
		return app
	}
	return web
}

// layGuiAnh — vẫn là dữ liệu trang chủ (form gửi ảnh nằm trong trang chủ),
// nhưng đánh dấu đang đứng ở màn hình gửi ảnh để thanh đáy sáng đúng nút.
func layGuiAnh(r *http.Request) dlTrangChu {
	d := layTrangChu(r)
	if laManHinhApp(r) {
		d.Trang = "gui-anh"
	}
	return d
}

// hAppKiem — trang soi máy. Xem ui/app-kiem.html.
func hAppKiem(w http.ResponseWriter, r *http.Request) {
	render(w, "app-kiem.html", chung(r, "kiem"))
}

// hAppGuiAnh — màn hình gửi ảnh của app. Chỉ lo phần GET; bấm gửi thì rơi vào
// hGuiYeuCau như mọi đường khác, cùng một luật kiểm và cùng một chỗ lưu.
func hAppGuiAnh(w http.ResponseWriter, r *http.Request) {
	render(w, "app-guianh.html", layGuiAnh(r))
}

// soGoi bỏ khoảng trắng và dấu ngăn để nhét số vào href="tel:". Số hiện ra cho
// khách đọc thì giữ nguyên như Kendy gõ.
func soGoi(s string) string {
	return strings.NewReplacer(" ", "", ".", "", "-", "", "(", "", ")", "").Replace(s)
}
