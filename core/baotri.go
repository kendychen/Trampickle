package core

import (
	"net/http"
	"os"
	"strings"
	"sync"
)

var (
	btMu sync.RWMutex
	btOn bool
)

func fileBaoTri() string { return P("data/bao-tri.flag") }

func NapBaoTri() {
	_, err := os.Stat(fileBaoTri())
	btMu.Lock()
	btOn = err == nil
	btMu.Unlock()
}

func DangBaoTri() bool {
	btMu.RLock()
	defer btMu.RUnlock()
	return btOn
}

func DatBaoTri(bat bool) error {
	btMu.Lock()
	defer btMu.Unlock()
	if bat {
		if err := os.MkdirAll(P("data"), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(fileBaoTri(), []byte("1"), 0o644); err != nil {
			return err
		}
		btOn = true
	} else {
		_ = os.Remove(fileBaoTri())
		btOn = false
	}
	return nil
}

func htmlBaoTri() string {
	return `<!doctype html><html lang="vi"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><meta name="theme-color" content="#2E2A5E"><title>Đang bảo trì — Trạm Pickle</title><style>
*{box-sizing:border-box;margin:0;padding:0}
:root{--forest:#2E2A5E;--lime:#FFC72C;--trang:#F6F4FB;--chu:#181528;--mo:#6B6588;--line:#E8E6F2}
body{font-family:Roboto,system-ui,Arial,sans-serif;background:var(--trang);color:var(--chu);line-height:1.6;min-height:100vh;display:flex;flex-direction:column}
.top{height:4px;background:var(--lime)}
.hero{background:var(--forest);color:#fff;padding:32px 20px 64px;text-align:center;position:relative;overflow:hidden}
.hero::after{content:"";position:absolute;left:50%;bottom:-80px;transform:translateX(-50%);width:900px;height:280px;background:radial-gradient(ellipse at 50% 0%,rgba(255,199,44,.16),transparent 68%);pointer-events:none}
.hero-inner{position:relative;max-width:640px;margin:0 auto}
.eyebrow{letter-spacing:.14em;text-transform:uppercase;font:700 11px/1 monospace;color:#B2ADCA}
.badge{display:inline-flex;align-items:center;gap:8px;margin-top:16px;background:rgba(255,255,255,.10);border:1px solid rgba(255,255,255,.16);color:var(--lime);padding:6px 12px;border-radius:999px;font:700 11px/1 monospace;letter-spacing:.12em;text-transform:uppercase}
.badge i{width:8px;height:8px;border-radius:50%;background:var(--lime);box-shadow:0 0 0 5px rgba(255,199,44,.20);display:inline-block;flex:none}
h1{margin:20px auto 10px;max-width:18ch;font-size:clamp(26px,4vw,36px);line-height:1.18;color:#fff;text-wrap:balance}
.hero p{max-width:48ch;margin:0 auto;color:#C4BFD9;font-size:14.5px;line-height:1.6}
.wrap{flex:1;display:flex;align-items:flex-start;justify-content:center;padding:0 16px 40px}
.card{width:min(560px,100%);background:#fff;border:1px solid var(--line);border-radius:20px;box-shadow:0 16px 40px rgba(46,42,94,.12),0 2px 8px rgba(46,42,94,.06);overflow:hidden;margin-top:-28px;position:relative}
.head{padding:20px 22px 0;display:flex;gap:14px;align-items:flex-start}
.ico{flex:0 0 48px;width:48px;height:48px;border-radius:14px;background:var(--forest);display:grid;place-items:center;box-shadow:0 4px 12px rgba(46,42,94,.18)}
.head h2{font-size:15px;line-height:1.35;color:var(--chu)}
.head h2 span{color:var(--mo);font-weight:400;font-size:13px;display:block;margin-top:4px;line-height:1.5}
.div{height:1px;background:var(--line);margin:16px 22px 0}
.body{padding:16px 22px 20px}
.note{font-size:14px;color:var(--mo);line-height:1.6}
.hang{display:flex;flex-wrap:wrap;gap:10px;margin-top:16px}
.nut{display:inline-flex;align-items:center;justify-content:center;padding:11px 22px;border-radius:999px;font-weight:700;font-size:14px;text-decoration:none;border:1px solid transparent;transition:opacity .15s}
.nut-c{background:var(--forest);color:#fff}
.nut-c:hover{opacity:.92}
.foot{padding:11px 22px;background:#F9F8FD;border-top:1px solid var(--line);display:flex;justify-content:space-between;gap:10px;color:#8A86A8;font-size:12px}
.foot code{background:#fff;border:1px solid var(--line);padding:2px 6px;border-radius:6px;color:var(--mo)}
@media(max-width:480px){.hero{padding:28px 16px 56px}h1{font-size:26px}.card{border-radius:16px}.head{padding:16px 16px 0}.body{padding:14px 16px 16px}.foot{padding:10px 16px;flex-direction:column;align-items:center;text-align:center}}
</style></head><body><div class="top"></div><header class="hero"><div class="hero-inner"><div class="eyebrow">TRẠM PICKLE · Sửa vợt Pickleball</div><div><span class="badge"><i></i> Đang bảo trì</span></div><h1>Trạm đang vá lại — quay lại sau ít phút</h1><p>Tạm đóng cửa để kiểm kê, cân lại vợt và đồng bộ dữ liệu. Đơn cũ vẫn an toàn, không mất gì.</p></div></header><main class="wrap"><div class="card"><div class="head"><div class="ico"><svg viewBox="0 0 24 24" width="26" height="26" fill="none" aria-hidden="true"><circle cx="12" cy="12" r="8.5" stroke="#FFC72C" stroke-width="1.5"/><path d="M8.5 13.5L12 10l3.5 3.5" stroke="#fff" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg></div><div><h2>Vợt của bạn vẫn được giữ nguyên<span>Trạng thái đơn, ảnh và lịch sử sửa chữa không bị ảnh hưởng khi bảo trì.</span></h2></div></div><div class="div"></div><div class="body"><p class="note">Vui lòng quay lại sau ít phút — hệ thống tự mở lại khi xong.</p><div class="hang"><a class="nut nut-c" href="/">Thử tải lại trang</a></div></div><div class="foot"><span>Tự động thử lại sau 60 phút · Mã 503</span><span><code>trampickle.vn</code></span></div></div></main></body></html>`
}

func boBaoTri(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if p == "/bao-tri.html" {
			if DangBaoTri() {
				w.Header().Set("Retry-After", "3600")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(htmlBaoTri()))
				return
			}
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}
		if DangBaoTri() {
			allowed := strings.HasPrefix(p, "/qt") || strings.HasPrefix(p, "/dang-nhap") || strings.HasPrefix(p, "/anh") || strings.HasPrefix(p, "/favicon") || strings.HasPrefix(p, "/font") || p == "/sw.js" || p == "/manifest.webmanifest"
			if !allowed {
				w.Header().Set("Retry-After", "3600")
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.Header().Set("Cache-Control", "no-store")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(htmlBaoTri()))
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}
