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

// Màu splash screen lúc mở app, và màu thanh trạng thái của điện thoại. Phải
// khớp --md-surface và --md-primary-container trong css.html. Trước đây đây là
// một bảng tra theo mã giao diện; site còn đúng một bảng màu Material 3 nên nó
// thành hai hằng số. Đổi màu gốc thì sửa cả hai chỗ, không có cách nào bắt
// được lệch ngoài mắt người — CSS nằm trong template, Go không đọc được.
//
// mauNhanApp đi theo --green-dam chứ không theo vàng nhấn: vàng #FFC72C chỉ
// dùng trên nền tối. Thanh trạng thái là một mảng tô nằm ngay trên đầu trang
// màu kem, tức nền sáng, nên nó theo luật nền sáng. Thêm nữa vàng ở đó là một
// vệt chói ngay tầm mắt suốt phiên dùng.
const (
	mauNenApp  = "#CDC9DE"
	mauNhanApp = "#3A3276"
)

// Màu của app QUẢN LÝ. Thanh trạng thái tô theo dải đầu trang quản trị (nền
// tối --th-dam trong css.html), nền splash lấy màu vùng làm việc. Mở app ra
// là biết ngay đang đứng trong app nào, không phải đọc chữ.
const (
	mauNenQt  = "#F6F4FB"
	mauNhanQt = "#2E2A5E"
)

// mauNhanTheme cho template gọi thẳng: <meta name="theme-color">. Thanh trạng
// thái của điện thoại tô theo màu này khi app đang mở.
func mauNhanTheme() string { return mauNhanApp }

// mauNhanQtTheme — bản cho trang quản trị. Xem mauNhanTheme.
func mauNhanQtTheme() string { return mauNhanQt }

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
	nen, nhan := mauNenApp, mauNhanApp
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
		icons = append(icons, biIcon{Src: duongFavicon(), Sizes: "any", Type: "image/svg+xml", MucDi: "any"})
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
		icons = append(icons, biIcon{Src: duongFavicon(), Sizes: "any", Type: "image/svg+xml", MucDi: "maskable"})
	}
	// Bản PNG 512 đi kèm: một số máy Android dựng lối tắt từ danh sách icon
	// mà không vẽ SVG, và trình cài của Samsung đòi có ít nhất một bitmap.
	icons = append(icons, biIcon{Src: duongIconApp(), Sizes: "512x512", Type: "image/png", MucDi: "any"})

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

// hManifestQt — manifest thứ hai, cho app quản lý của Kendy và thợ.
//
// Phải khác "id" thì máy mới coi là hai app riêng. Trùng id là cài cái sau đè
// lên cái trước: bấm icon quản lý lại mở ra màn hình khách, mà không có lỗi
// nào để lần ra.
//
// "scope" để "/" chứ không co về "/qt/": hết phiên là trang tự nhảy sang
// /dang-nhap, đường đó nằm ngoài /qt. Scope hẹp thì đúng lúc đăng nhập app
// bung ra thanh địa chỉ xám của trình duyệt, trông như vừa bị văng ra ngoài.
// Hai app cùng scope "/" không sao — máy phân biệt nhau bằng id.
//
// Icon lấy bản vẽ trong binary chứ không lấy logo Kendy tải lên: logo ấy đã là
// icon của app khách rồi, dùng lại là hai icon giống hệt nhau nằm cạnh nhau.
func hManifestQt(w http.ResponseWriter, r *http.Request) {
	ten := CFG.ThuongHieu.Ten
	if ten == "" {
		ten = "Trạm"
	}
	icon := []biIcon{
		{Src: duongIconQt(), Sizes: "any", Type: "image/svg+xml", MucDi: "any"},
		{Src: duongIconQtPNG(), Sizes: "512x512", Type: "image/png", MucDi: "any"},
		{Src: duongIconQt(), Sizes: "any", Type: "image/svg+xml", MucDi: "maskable"},
	}
	m := map[string]any{
		"id":               "/qt/",
		"name":             ten + " — Quản lý",
		"short_name":       "Quản lý",
		"description":      "Đơn hàng, kho vật tư và sổ tiền của " + ten,
		"start_url":        "/qt/don-vot",
		"scope":            "/",
		"display":          "standalone",
		"orientation":      "portrait",
		"lang":             "vi",
		"dir":              "ltr",
		"background_color": mauNenQt,
		"theme_color":      mauNhanQt,
		"icons":            icon,
		"shortcuts": []map[string]any{
			{"name": "Đơn giày", "url": "/qt/don-giay"},
			{"name": "Tạo đơn", "url": "/qt/don-moi"},
			{"name": "Sổ tiền", "url": "/qt/tien"},
		},
	}
	w.Header().Set("Content-Type", "application/manifest+json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(m)
}

// hIconQt trả icon của app quản lý. Tệp tĩnh nhúng sẵn trong binary.
func hIconQt(w http.ResponseWriter, r *http.Request) {
	b, err := uiFS.ReadFile("ui/icon-qt.svg")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Write(b)
}

// hIconQtPNG — cùng hình ấy nhưng dạng PNG 512px. Safari KHÔNG đọc SVG trong
// apple-touch-icon: gặp SVG nó bỏ qua rồi tự chụp màn hình trang làm icon,
// ra một ô chữ li ti trên màn hình chính. Bản PNG này dựng sẵn từ
// ui/icon-qt.svg (Chrome headless), để cạnh nhau trong ui/ — sửa hình thì
// phải dựng lại tệp PNG, không tự sinh lúc chạy được.
func hIconQtPNG(w http.ResponseWriter, r *http.Request) {
	b, err := uiFS.ReadFile("ui/icon-qt.png")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Write(b)
}

// hIconApp — bản PNG 512px của icon app khách, đối xứng với hIconQtPNG. Cùng
// một lý do: Safari bỏ qua SVG trong apple-touch-icon rồi tự chụp màn hình
// trang làm icon. Dựng sẵn từ ui/favicon-4d.svg bằng Chrome headless — sửa
// hình thì phải dựng lại tệp PNG, không tự sinh lúc chạy được.
func hIconApp(w http.ResponseWriter, r *http.Request) {
	b, err := uiFS.ReadFile("ui/icon-app.png")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	w.Write(b)
}

// iconTouch — icon cho <link rel="apple-touch-icon">. Icon Kendy tải lên thì
// dùng thẳng (bao giờ cũng là ảnh bitmap), còn bản nhúng thì phải trả PNG chứ
// KHÔNG trả SVG: gặp SVG, Safari bỏ qua thẻ này.
func iconTouch() string {
	if iconMIME() == "image/svg+xml" {
		return duongIconApp()
	}
	return iconURL()
}

// hServiceWorker phục vụ /sw.js ở ngay gốc. Đặt sâu hơn một cấp thì phạm vi
// của nó co lại đúng thư mục ấy và trang chủ nằm ngoài tầm.
func hServiceWorker(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Service-Worker-Allowed", "/")
	fmt.Fprintf(w, swJS, phienBanPWA, duongFavicon(), duongIconQt(), cssURL(), duongIconQtPNG())
}

// Mạng trước, kho sau, và chỉ với những gì khách xem. Trang quản trị, API,
// đăng nhập, tra cứu đơn thì đi thẳng ra mạng không đụng vào kho: cache một
// trang /qt là có ngày Kendy sửa đơn xong tải lại vẫn thấy số cũ, còn cache
// trang tra cứu là khách thấy trạng thái đơn của hôm kia.
const swJS = `// Sinh bởi Go, đừng sửa tay. Nguồn: core/pwa.go
const V = '%s';
const KHO = 'tram-' + V;

// Vỏ app: đủ để mở được app khi mất mạng hẳn.
//
// /qt/mat-mang là ngoại lệ DUY NHẤT của luật "không cache trang quản lý": nó
// là một tấm chữ tĩnh, không có số liệu đơn nào trong đó, nên cache bao lâu
// cũng không nói dối được điều gì. Máy khách cũng tải nó về (vài KB) — thà
// thế còn hơn phải nuôi hai service worker.
//
// Tệp CSS đứng cuối: từ lúc style tách ra khỏi HTML, thiếu nó thì bản offline
// mở ra là một trang chữ trần không màu — trông như app hỏng chứ không như
// app đang mất mạng.
const VO = ['/app', '%s', '/manifest.webmanifest', '/qt/mat-mang', '%s', '%s'];

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
  if (boQua(u)) {
    // Trang quản lý vẫn không bao giờ vào kho — nhưng mất mạng thì đừng để
    // app hiện màn khủng long của Chrome, nó trông như app hỏng. Trả tấm
    // /qt/mat-mang tĩnh đã nằm sẵn trong kho từ lúc cài.
    if (rq.mode === 'navigate' && u.pathname.indexOf('/qt') === 0) {
      e.respondWith(fetch(rq).catch(function () {
        return caches.match('/qt/mat-mang').then(function (c) {
          return c || Response.error();
        });
      }));
    }
    return;
  }

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

  // Ảnh, phông và CSS: kho trước. Chúng đổi hiếm, mà tải lại mỗi lần thì tốn
  // 3G của khách. Riêng CSS thì tên tệp mang mã băm nội dung, nên bản trong
  // kho không bao giờ là bản cũ: sửa CSS là URL đổi theo.
  if (rq.destination === 'image' || rq.destination === 'font' || rq.destination === 'style') {
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

// --- Thông báo đơn hàng ---------------------------------------------------
// Service worker dùng chung cho cả hai app, nhưng chỉ app quản lý mới đăng ký
// nhận. Máy khách không có subscription nên không bao giờ chạy tới đoạn này.
self.addEventListener('push', function (e) {
  var d = {};
  try { d = e.data ? e.data.json() : {}; } catch (x) { d = {}; }
  // requireInteraction: tin đứng lại tới khi có người bấm. Ba việc được báo
  // đều là việc phải làm, không phải tin để ngắm — tự tắt sau vài giây thì
  // đúng lúc đang cầm tua vít là mất luôn.
  e.waitUntil(self.registration.showNotification(d.tieu_de || 'Trạm', {
    body: d.than || '',
    icon: '%s',
    tag: d.the || 'tram',
    renotify: true,
    requireInteraction: true,
    data: { duong: d.duong || '/qt' }
  }));
});

self.addEventListener('notificationclick', function (e) {
  e.notification.close();
  var duong = (e.notification.data && e.notification.data.duong) || '/qt';
  // Có cửa sổ app đang mở thì ĐIỀU HƯỚNG cửa sổ đó rồi kéo lên trước, đừng mở
  // thêm cái thứ hai: hai bản app cùng mở là hai phiên, sửa đơn ở bản này rồi
  // tải lại bản kia thì thấy số cũ.
  e.waitUntil(self.clients.matchAll({ type: 'window', includeUncontrolled: true }).then(function (ds) {
    for (var i = 0; i < ds.length; i++) {
      var c = ds[i];
      if (c.url.indexOf(self.location.origin) !== 0) { continue; }
      if ('navigate' in c) { return c.navigate(duong).then(function (x) { return x ? x.focus() : null; }); }
      if ('focus' in c) { return c.focus(); }
    }
    return self.clients.openWindow(duong);
  }));
});
`

// hApp — màn hình app. Chỉ dịch vụ, không bài viết, không giới thiệu: người
// bấm icon trên màn hình chính là đang muốn làm một việc cụ thể với cây vợt,
// không phải ngồi đọc.
func hApp(w http.ResponseWriter, r *http.Request) {
	c := chung(r, "app")
	nen, nhan := mauNenApp, mauNhanApp
	render(w, "app.html", struct {
		Chung
		DichVu  []DichVu
		Anh     []AnhHero
		MauNen  string
		MauNhan string
		// App: ảnh banner, đợt khuyến mãi, nhịp icon — sửa ở /qt/app.
		App AppCauHinh
	}{c, DichVuDangBan(GiaiDoan), DsAnhNoi(""), nen, nhan, AppCauHinhHienTai()})
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
