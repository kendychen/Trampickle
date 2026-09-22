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

func boBaoTri(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if DangBaoTri() {
			p := r.URL.Path
			allowed := strings.HasPrefix(p, "/qt") || strings.HasPrefix(p, "/dang-nhap") || p == "/bao-tri.html" || strings.HasPrefix(p, "/anh") || strings.HasPrefix(p, "/favicon")
			if !allowed {
				w.Header().Set("Retry-After", "3600")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte("<html><head><meta charset=utf-8><title>Bao tri</title></head><body style='font-family:sans-serif;max-width:600px;margin:60px auto;padding:24px'><h1>Dang bao tri</h1><p>Tram dang bao tri he thong, vui long quay lai sau it phut.</p><p><a href=/qt/dang-nhap>Dang nhap quan tri</a></p></body></html>"))
				return
			}
		}
		h.ServeHTTP(w, r)
	})
}
