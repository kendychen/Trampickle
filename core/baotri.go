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
:root{--forest:#2E2A5E;--lime:#FFC72C;--kem:#CDC9DE;--trang:#F6F4FB;--chu:#181528;--mo:#4E496E}
body{font-family:Roboto,system-ui,Arial,sans-serif;background:var(--trang);color:var(--chu);line-height:1.6;min-height:100vh;display:flex;flex-direction:column}
.top{height:4px;background:var(--lime)}
.hero{background:var(--forest);color:#EDEBF7;padding:28px 20px 36px;text-align:center;position:relative;overflow:hidden}
.hero::after{content:"";position:absolute;inset:auto -20% -60% -20%;height:220px;background:radial-gradient(ellipse at 50% 0%,rgba(255,199,44,.18),transparent 70%);pointer-events:none}
.badge{display:inline-flex;align-items:center;gap:8px;margin-top:18px;background:rgba(255,255,255,.10);border:1px solid rgba(255,255,255,.18);color:var(--lime);padding:6px 12px;border-radius:999px;font:700 11px/1 monospace;letter-spacing:.14em;text-transform:uppercase}
.badge i{width:8px;height:8px;border-radius:50%;background:var(--lime);box-shadow:0 0 0 6px rgba(255,199,44,.22);display:inline-block}
h1{margin:18px auto 10px;max-width:22ch;font-size:clamp(28px,4.2vw,40px);line-height:1.12;color:#fff}
.hero p{max-width:52ch;margin:0 auto;color:#B2ADCA;font-size:15px}
.wrap{flex:1;display:flex;align-items:flex-start;justify-content:center;padding:28px 16px 40px}
.card{width:min(640px,100%);background:#fff;border:1px solid #A5A0C0;border-radius:16px;box-shadow:0 8px 24px rgba(46,42,94,.10);overflow:hidden;margin-top:-28px;position:relative}
.head{padding:22px 22px 0;display:flex;gap:16px;align-items:flex-start}
.ico{flex:0 0 56px;width:56px;height:56px;border-radius:14px;background:var(--forest);display:grid;place-items:center}
.head h2{font-size:16px;line-height:1.3}
.head h2 span{color:var(--mo);font-weight:400;font-size:13px;display:block;margin-top:4px}
.div{height:1px;background:#DFDCEC;margin:16px 22px 0}
.body{padding:16px 22px 22px}
.body ul{margin:10px 0 0 18px;color:var(--mo);font-size:14px}
.hang{display:flex;flex-wrap:wrap;gap:10px;margin-top:18px}
.nut{display:inline-flex;align-items:center;justify-content:center;padding:11px 18px;border-radius:999px;font-weight:700;font-size:14px;text-decoration:none;border:1px solid transparent}
.nut-c{background:var(--forest);color:#fff}
.nut-p{background:#fff;color:var(--forest);border-color:#A5A0C0}
.foot{padding:12px 22px;background:#F6F4FB;border-top:1px solid #DFDCEC;display:flex;justify-content:space-between;gap:10px;color:var(--mo);font-size:12px}
.foot code{background:#fff;border:1px solid #DFDCEC;padding:2px 6px;border-radius:6px}
</style></head><body><div class="top"></div><header class="hero"><div style="letter-spacing:.14em;text-transform:uppercase;font:700 11px/1 monospace;color:#B2ADCA">TRẠM PICKLE · Sửa vợt Pickleball</div><div><span class="badge"><i></i> Đang bảo trì</span></div><h1>Trạm đang vá lại — quay lại sau ít phút</h1><p>Tạm đóng cửa để kiểm kê, cân lại vợt và đồng bộ dữ liệu. Đơn cũ vẫn an toàn, không mất gì.</p></header><main class="wrap"><div class="card"><div class="head"><div class="ico"><svg viewBox="0 0 24 24" width="28" height="28" fill="none"><circle cx="12" cy="12" r="9" stroke="#FFC72C" stroke-width="1.6"/><path d="M8 14l3-3 3 3" stroke="#fff" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"/></svg></div><div><h2>Vợt của bạn vẫn được giữ nguyên<span>Trạng thái đơn, ảnh và lịch sử sửa chữa không bị ảnh hưởng khi bảo trì.</span></h2></div></div><div class="div"></div><div class="body"><p style="font-size:14px;color:var(--mo)">Bạn vẫn có thể:</p><ul><li>Đăng nhập quản trị — <b>/qt</b> vẫn mở để kiểm tra đơn.</li><li>Quay lại trang chủ sau ít phút, hệ thống tự mở lại khi xong.</li></ul><div class="hang"><a class="nut nut-c" href="/">Thử tải lại trang</a><a class="nut nut-p" href="/dang-nhap">Đăng nhập quản trị</a></div></div><div class="foot"><span>Tự động thử lại sau 60 phút · Mã 503</span><span><code>trampickle.vn</code></span></div></div></main></body></html>`
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
