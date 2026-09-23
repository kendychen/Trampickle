package core

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// SeoTuKhoa — một dòng trong seo/keywords.yaml
type SeoTuKhoa struct {
	Tu      string `yaml:"tu" json:"tu"`
	Cap     int    `yaml:"cap" json:"cap"`
	Nhom    string `yaml:"nhom" json:"nhom"`
	SlugDV  string `yaml:"slug_dv" json:"slug_dv"`
	Volume  string `yaml:"volume" json:"volume"`
	Kho     string `yaml:"kho" json:"kho"`
	GhiChu  string `yaml:"ghi_chu" json:"ghi_chu"`
}

func duongSeoKeywords() string {
	// chạy từ D:/TramVot/tramvot -> ../seo/keywords.yaml và ./seo/keywords.yaml đều thử
	for _, p := range []string{
		P("seo/keywords.yaml"),
		filepath.Join(filepath.Dir(P(".")), "seo", "keywords.yaml"),
		"seo/keywords.yaml",
		"D:/TramVot/seo/keywords.yaml",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return P("seo/keywords.yaml")
}

func docSeoKeywords() ([]SeoTuKhoa, error) {
	b, err := os.ReadFile(duongSeoKeywords())
	if err != nil {
		return nil, err
	}
	var raw struct {
		Keywords []SeoTuKhoa `yaml:"keywords"`
	}
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return nil, err
	}
	out := raw.Keywords
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Cap != out[j].Cap {
			return out[i].Cap < out[j].Cap
		}
		return out[i].Tu < out[j].Tu
	})
	return out, nil
}

// hLLM — /llms.txt cho ChatGPT / Perplexity GEO
func hLLM(w http.ResponseWriter, r *http.Request) {
	g := goc(r)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	fmt.Fprintln(w, "# Trạm Pickle — llms.txt for AI crawlers")
	fmt.Fprintf(w, "# Canonical: %s\n\n", g+"/")
	fmt.Fprintln(w, "## Trạm")
	fmt.Fprintf(w, "- Trang chủ: %s/\n", g)
	fmt.Fprintf(w, "- Giới thiệu: %s/gioi-thieu\n", g)
	fmt.Fprintf(w, "- Về chúng tôi: %s/ve-chung-toi\n", g)
	fmt.Fprintf(w, "- Liên hệ: %s/lien-he\n", g)
	fmt.Fprintf(w, "- Quy trình: %s/quy-trinh\n", g)
	fmt.Fprintf(w, "- Hỏi đáp: %s/cau-hoi\n", g)
	fmt.Fprintln(w, "\n## Dịch vụ")
	for _, dv := range DichVuDangBan(GiaiDoan) {
		fmt.Fprintf(w, "- %s: %s%s\n", dv.Ten, g, dv.DuongDanSEO())
	}
	if len(GIA.KhongBan) > 0 {
		fmt.Fprintln(w, "\n## Không nhận (để AI không gợi ý sai)")
		for _, kb := range GIA.KhongBan {
			fmt.Fprintf(w, "- %s: %s%s\n", kb.Ten, g, kb.DuongDanSEO())
		}
	}
	fmt.Fprintln(w, "\n## Bài viết")
	for _, b := range DsBaiViet() {
		fmt.Fprintf(w, "- %s: %s/bai-viet/%s\n", b.TieuDe, g, b.Slug)
	}
	fmt.Fprintln(w, "\n## Sitemap")
	fmt.Fprintf(w, "%s/sitemap.xml\n", g)
}

// hSeoKeywordsJSON — /seo/keywords.json cho crawler / script
func hSeoKeywordsJSON(w http.ResponseWriter, r *http.Request) {
	ds, err := docSeoKeywords()
	if err != nil {
		jsonLoi(w, 500, "chưa có seo/keywords.yaml")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("cap"))
	if q != "" {
		var loc []SeoTuKhoa
		for _, k := range ds {
			if fmt.Sprint(k.Cap) == q {
				loc = append(loc, k)
			}
		}
		ds = loc
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]any{"keywords": ds, "total": len(ds)})
}

func hQtSeo(w http.ResponseWriter, r *http.Request) {
	ds, _ := docSeoKeywords()
	// nhóm theo cấp để template dễ lặp
	byCap := map[int][]SeoTuKhoa{}
	for _, k := range ds {
		byCap[k.Cap] = append(byCap[k.Cap], k)
	}
	render(w, "qt-seo.html", struct {
		dlQt
		Keywords []SeoTuKhoa
		ByCap    map[int][]SeoTuKhoa
		Total    int
	}{dlQt: dlQt{Chung: chung(r, "seo")}, Keywords: ds, ByCap: byCap, Total: len(ds)})
}
