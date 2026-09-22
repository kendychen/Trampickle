import pathlib, re, shutil, os, sys
try:
    sys.stdout.reconfigure(encoding='utf-8')
except: pass
ROOT = pathlib.Path(__file__).resolve().parents[1]
GT = ROOT / "giao-trinh-sua-vot"
print("GT exists", GT.exists())
# 1 rename 3 files
for name in ["00-KE-HOACH.md","NGUON.md","VIDEO.md"]:
    p = GT / name
    if p.exists():
        low = GT / name.lower()
        tmp = GT / (p.stem + "_tmp_rename" + p.suffix)
        try:
            p.rename(tmp)
            tmp.rename(low)
            print(f"renamed {name} -> {low.name}")
        except Exception as e:
            print(e)
    else:
        print(f"not found {p} exists? {p.exists()}")
# alias readme lowercase
for p in GT.rglob("README.md"):
    alias = p.parent / "readme.md"
    if not alias.exists():
        shutil.copy2(p, alias)
        print(f"alias {alias.relative_to(ROOT)}")
# 2 fix links
md_re = re.compile(r"\[([^\]]+)\]\(([^)]+)\)")
fixes=0
for p in GT.rglob("*.md"):
    text = p.read_text(encoding="utf-8", errors="ignore")
    orig=text
    def repl(m):
        global fixes
        label, url = m.group(1), m.group(2)
        if url.startswith("http://") or url.startswith("https://") or url.startswith("#") or url.startswith("mailto:") or url.startswith("tel:"):
            return m.group(0)
        base=url
        frag=""
        q=""
        if "#" in base:
            base,frag = base.split("#",1)
            frag="#"+frag
        if "?" in base:
            base,q = base.split("?",1)
            q="?"+q
        if re.search(r"[A-Z]", base):
            new_base=base.lower()
            fixes+=1
            return f"[{label}]({new_base}{q}{frag})"
        return m.group(0)
    new_text = md_re.sub(repl, text)
    if new_text!=orig:
        p.write_text(new_text, encoding="utf-8")
        print(f"fixed {p.relative_to(ROOT)}")
print(f"total fixes {fixes}")
# 3 add Xem them for orphan files
related = {
    "01-nen-tang/1-3-dung-cu-an-toan.md": ["01-nen-tang/1-1-cau-tao-vot.md","01-nen-tang/1-2-vat-lieu-keo.md","02-quy-trinh/2-1-nhan-vot.md","02-quy-trinh/2-2-chan-doan.md"],
    "00-ke-hoach.md": ["readme.md","nguon.md","video.md"],
    "anh/readme.md": ["../01-nen-tang/1-1-cau-tao-vot.md"],
    "_nguon/readme.md": ["../nguon.md"],
}
for rel, links in related.items():
    p = GT / rel
    if p.exists():
        t=p.read_text(encoding="utf-8", errors="ignore")
        if "xem them" not in t.lower():
            sec="\n\n## Xem them\n\n" + "\n".join([f"- [{l}]({l})" for l in links]) + "\n"
            if "## Cau hoi" in t:
                t=t.replace("## Cau hoi", sec+"## Cau hoi")
            else:
                t=t.rstrip()+sec
            p.write_text(t, encoding="utf-8")
            print(f"added Xem them {rel}")
# 4 robots + sitemap
robots = ROOT / "robots.txt"
robots.write_text("User-agent: *\nAllow: /\nSitemap: /sitemap.xml\n", encoding="utf-8")
print("robots ok")
sitemap = ROOT / "sitemap.xml"
base_url="https://example.com/"
urls=[]
for p in GT.rglob("*.md"):
    rel=str(p.relative_to(ROOT)).replace("\\","/").lower().replace(".md",".html")
    urls.append(base_url+rel)
xml='<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n'
for u in sorted(urls):
    xml+=f"  <url><loc>{u}</loc></url>\n"
xml+="</urlset>\n"
sitemap.write_text(xml, encoding="utf-8")
print(f"sitemap {len(urls)} urls")
print("done")