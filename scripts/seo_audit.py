# -*- coding: utf-8 -*-
import os, re, pathlib

ROOT = pathlib.Path(r"D:\Dự Án Sửa Chữa Pickleball")
GT = ROOT / "giao-trinh-sua-vot"

md_link_re = re.compile(r'\[([^\]]+)\]\(([^)]+)\)')

issues_upper = []
issues_broken = []
all_links = []

for p in GT.rglob("*.md"):
    text = p.read_text(encoding="utf-8", errors="ignore")
    for m in md_link_re.finditer(text):
        url = m.group(2).strip()
        # skip anchors only, mailto, tel
        if url.startswith("#") or url.startswith("mailto:") or url.startswith("tel:"):
            continue
        # check uppercase in URL path (internal only)
        is_external = url.startswith("http://") or url.startswith("https://")
        # for external, we still check path part after domain has uppercase (SEO bad)
        check_part = url
        if is_external:
            # strip scheme+domain
            try:
                after = url.split("//",1)[1]
                slash = after.find("/")
                check_part = after[slash+1:] if slash!=-1 else ""
            except: check_part = url
        else:
            # internal .md link
            check_part = url.split("#")[0].split("?")[0]
        if re.search(r"[A-Z]", check_part):
            issues_upper.append((str(p.relative_to(ROOT)), url, m.group(0)))
        # check broken internal file links (only .md and relative)
        if not is_external and ".md" in url:
            # remove anchor/query
            clean = url.split("#")[0].split("?")[0]
            # resolve relative
            target = (p.parent / clean).resolve()
            # try case-insensitive check
            exists = target.exists()
            if not exists:
                # try lowercase version
                try:
                    # check alternate case
                    parent = target.parent
                    if parent.exists():
                        lower_name = target.name.lower()
                        found = [f for f in parent.iterdir() if f.name.lower()==lower_name]
                        if found:
                            issues_broken.append((str(p.relative_to(ROOT)), url, f"exists as {found[0].name} (case mismatch)"))
                        else:
                            issues_broken.append((str(p.relative_to(ROOT)), url, "FILE NOT FOUND"))
                    else:
                        issues_broken.append((str(p.relative_to(ROOT)), url, "PARENT NOT FOUND"))
                except Exception as e:
                    issues_broken.append((str(p.relative_to(ROOT)), url, str(e)))
        all_links.append((str(p.relative_to(ROOT)), url))

print("=== UPPERCASE URL ISSUES ===")
if not issues_upper:
    print("OK - Khong co URL chua chu hoa trong duong dan")
else:
    for a,b,c in issues_upper: print(a, "->", b)

print("\n=== BROKEN INTERNAL LINKS ===")
if not issues_broken:
    print("OK - Khong co link noi bo gay")
else:
    for a,b,c in issues_broken: print(a, "->", b, "|", c)

print(f"\nTong so link quet: {len(all_links)}")

# Check interlink density per file
from collections import Counter
cnt = Counter([x[0] for x in all_links])
print("\n=== INTERNAL LINK DENSITY (so link / file) ===")
for f,c in cnt.most_common():
    print(f"{c:2d}  {f}")

# Check files with 0 or 1 links (orphan risk)
print("\n=== FILE IT CO LIEN KET (<2) - nguy co orphan ===")
for p in GT.rglob("*.md"):
    rel = str(p.relative_to(ROOT))
    if cnt.get(rel,0) < 2:
        print(rel, cnt.get(rel,0))

# Check for SEO meta: title H1, description
print("\n=== SEO META CHECK (title H1, mo ta dau bai) ===")
for p in GT.rglob("*.md"):
    t = p.read_text(encoding="utf-8", errors="ignore")
    has_h1 = bool(re.search(r"^#\s", t, re.M))
    # description = blockquote or first paragraph
    has_desc = "> " in t[:500]
    status = []
    if not has_h1: status.append("THIEU H1")
    print(f"{'OK' if has_h1 else '!!'} {p.relative_to(ROOT)} | H1:{has_h1} desc:{has_desc}")
