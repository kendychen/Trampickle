import fs from 'fs';
import path from 'path';
const ROOT = path.resolve('.');
const SRC = path.join(ROOT, 'giao-trinh-sua-vot');
const BAIVIET_SRC = path.join(ROOT, 'content/bai-viet');
const OUT = path.join(ROOT, 'dist');
// cây sitemap động của Go đã khai ở core/baiviet.go:hSitemap. Tập này chỉ
// build tài liệu tĩnh giao-trinh-sua-vot, nên sitemap tĩnh BỔ SUNG thêm các
// đường dẫn tiền phương (/ , /dich-vu ...) để khi nginx KHÔNG proxy về Go
// (deploy tĩnh) Google vẫn thấy đủ. Khi có proxy nginx (VPS Go đang chạy)
// thì file này bị qua mặt bởi proxy_pass — không xung đột.
// SEO/AI: build sinh thêm llms.txt / ai.txt cho AI Search (LLMs.txt spec)
const DOMAIN = 'https://trampickle.vn';
const SITE_NAME = 'Tram Pickle - Sua vot Pickleball chuyen nghiep';
const OG_IMAGE = DOMAIN + '/giao-trinh-sua-vot/anh/og-default.jpg';
const TRACK_JS = '/assets/js/tp-track.js';
// Service-area business (online): địa chỉ 58 Tố Hữu - Đại Mỗ vẫn HIỂN THỊ
// trên website (footer/lien-he) để khách gửi hàng, nhưng KHÔNG đưa
// streetAddress chi tiết vào JSON-LD schema. Schema chỉ khai ở mức
// addressLocality Hà Nội + areaServed Hà Nội để Google/AI hiểu là dịch vụ
// online nhận qua gửi hàng, hẹn trước khi mang tới — không phải storefront.
const GO_CORE_URLS = ["/","/dich-vu","/bai-viet","/quy-trinh","/cau-hoi","/gioi-thieu","/ve-chung-toi","/lien-he","/chinh-sach"];

function walk(dir){ let r=[]; for(const e of fs.readdirSync(dir,{withFileTypes:true})){ const p=path.join(dir,e.name); if(e.isDirectory()) r.push(...walk(p)); else r.push(p);} return r; }
function esc(s){ return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;'); }
function mdToHtml(md){
  // very simple: headings, links, tables, lists
  let html = md
    .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
    // code blocks ```...```
    // code blocks ```...``` - placeholder to avoid escaping inside
    .replace(/```([\s\S]*?)```/g,(m,c)=>`__CODEBLOCK_${Buffer.from(c).toString('base64').slice(0,120)}__`)
    // inline code
    .replace(/`([^`]+)`/g,'<code>$1</code>')
    // h1-3
    .replace(/^### (.+)$/gm,'<h3>$1</h3>')
    .replace(/^## (.+)$/gm,'<h2>$1</h2>')
    .replace(/^# (.+)$/gm,'<h1>$1</h1>')
    // bold
    .replace(/\*\*([^*]+)\*\*/g,'<strong>$1</strong>')
    // links [a](url) -> lower url path
    .replace(/\[([^\]]+)\]\(([^)]+)\)/g,(m,label,url)=>{
      if(url.startsWith('http')) return `<a href="${url}">${label}</a>`;
      const lower = url.toLowerCase().replace(/\.md$/,'.html');
      return `<a href="${lower}">${label}</a>`;
    })
    // images
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g,'<img alt="$1" src="$2" loading="lazy" decoding="async" style="max-width:100%;height:auto" />')
    // tables - keep as pre for simplicity (browser will render pipe)
    // paragraphs
  ;
  // convert markdown tables to HTML tables (loop over lines)
  {
    const lines = html.split('\n');
    let outLines=[];
    for(let i=0;i<lines.length;){
      const line=lines[i].trim();
      if(line.startsWith('|')){
        let block=[];
        while(i<lines.length && lines[i].trim().startsWith('|')){ block.push(lines[i].trim()); i++; }
        const isSep = (r)=> /^\|\s*[-:\s|]+\s*\|$/.test(r);
        if(block.some(isSep)){
          const dataRows=block.filter(r=>!isSep(r));
          if(dataRows.length>=1){
            const parseRow=(r)=> r.split('|').slice(1,-1).map(c=>c.trim());
            const ths=parseRow(dataRows[0]);
            let tbl='<table border="1" cellpadding="8" cellspacing="0" style="border-collapse:collapse;width:100%;margin:12px 0"><thead><tr>'+ths.map(c=>`<th style="background:#f6f8fa;text-align:left">${c}</th>`).join('')+'</tr></thead><tbody>';
            for(let k=1;k<dataRows.length;k++){ const tds=parseRow(dataRows[k]); tbl+='<tr>'+tds.map(c=>`<td>${c}</td>`).join('')+'</tr>'; }
            tbl+='</tbody></table>';
            outLines.push(tbl);
            continue;
          }
        }
        outLines.push(...block);
        continue;
      }
      outLines.push(lines[i]); i++;
    }
    html=outLines.join('\n');
  }
  // blockquote: lines starting with &gt; (escaped &gt;)
  html = html.replace(/^&gt;\s?(.*)$/gm,'<blockquote style="border-left:3px solid #0b57d0;padding-left:12px;color:#444;margin:12px 0">$1</blockquote>');
  // split lines to paragraphs (handle lists, headings etc)
  html = html.split(/\n{2,}/).map(block=>{
    const t=block.trim();
    if(!t) return '';
    if(t.startsWith('<h')||t.startsWith('<pre')||t.startsWith('<ul')||t.startsWith('<table')||t.startsWith('<img')||t.startsWith('<blockquote')||t.startsWith('__CODEBLOCK')) return t;
    // list
    if(/^[-*] /.test(t)) return '<ul>'+ t.split('\n').map(l=>l.replace(/^[-*] (.+)/,'<li>$1</li>')).join('') + '</ul>';
    if(/^\|/.test(t)) return `<pre>${t}</pre>`;
    if(/^---/.test(t)) return '<hr/>';
    return `<p>${t.replace(/\n/g,'<br/>')}</p>`;
  }).join('\n');
  // remove any remaining codeblock placeholders (FAQ json leaked) - strip them
  html = html.replace(/__CODEBLOCK_[A-Za-z0-9+/=]+__/g, '');
  return html;
}
function cleanBody(md){
  let out = md;
  // remove blockquote metadata lines: > Slug: `...` | URL: `...` etc and > Meta..., > OG...
  out = out.split('\n').filter(l=>{
    const t=l.trim();
    if(/^>\s*Slug:/i.test(t)) return false;
    if(/^>\s*Meta (title|desc)/i.test(t)) return false;
    if(/^>\s*OG image/i.test(t)) return false;
    if(/^>\s*Title:/i.test(t)) return false;
    if(/^>\s*Alt:/i.test(t)) return false;
    // generic > line that contains Slug/Meta/OG should be dropped if starts with >
    if(/^>/.test(t) && /(Slug:|Meta title|Meta desc|OG image)/i.test(t)) return false;
    return true;
  }).join('\n');
  // remove trailing FAQ/internal boilerplate from display: from "---" + "### FAQ" onwards and "Internal link:"
  // keep body up to CTA, drop everything after horizontal rule that contains FAQ schema
  const faqIdx = out.search(/---\s*\n### FAQ/i);
  if(faqIdx!==-1){ out = out.slice(0, faqIdx).trim(); }
  else {
    const altIdx = out.search(/```json[\s\S]*?```/);
    // if FAQ code block is at end, remove it and preceding heading
    if(altIdx!==-1){
      const before = out.slice(0, altIdx);
      const hIdx = before.lastIndexOf('### FAQ');
      if(hIdx!==-1) out = before.slice(0, hIdx).trim();
    }
  }
  // remove Internal link: line if remains
  out = out.split('\n').filter(l=>!/^Internal link:/i.test(l.trim())).join('\n');
  // remove horizontal rule alone at end
  out = out.replace(/\n---\s*$/,'').trim();
  return out;
}
function extractTitle(md){ const m=md.match(/^#\s+(.+)$/m); return m?m[1].trim():'Giao trinh sua vot Pickleball'; }
function stripMd(s){ return s.replace(/[*_`\[\]()]/g,'').replace(/\s+/g,' ').trim(); }
function extractDesc(md){
  const mDesc = md.match(/Meta desc[^:]*:\s*(.+)/i);
  if(mDesc){
    let d = stripMd(mDesc[1].trim()).replace(/^\s*[:\-–]+\s*/, '');
    d = d.replace(/\s+/g,' ').trim().slice(0,158);
    if(d.length>40) return d;
  }
  const lines=md.split('\n'); let after=false; let c=[];
  for(const l of lines){
    if(/^#\s/.test(l)){ after=true; continue; }
    if(!after) continue;
    if(/^\s*$/.test(l)||/^##/.test(l)||/^###/.test(l)||/^\|/.test(l)||/^[-*] /.test(l)||/^```/.test(l)) continue;
    let tt=l.trim(); if(!tt) continue;
    if(tt.startsWith('>')){
      if(/Slug:|Meta title|OG image/i.test(tt)) continue;
      tt=tt.replace(/^>\s*/,'').trim();
    }
    if(/Slug:|Meta title|OG image/i.test(tt)) continue;
    if(tt.length<20) continue;
    tt=stripMd(tt);
    if(tt.length>40) c.push(tt);
    if(c.length>=2) break;
  }
  let d=c[0]||''; d=d.slice(0,158);
  if(!d) return 'Tai lieu day nghe sua vot pickleball - Tram Pickle. Hoc tu nen tang den cac ca sua thuc te.';
  return d;
}
function extractFAQ(md){
  const i=md.search(/##\s*Cau hoi/i); if(i===-1) return [];
  const sec=md.slice(i,i+4000);
  return [...sec.matchAll(/^\s*\d+\.\s+(.+\?)\s*$/gm)].map(m=>m[1].trim()).slice(0,6);
}
function breadcrumbJsonLd(canonical, title){
  const parts = canonical.replace(DOMAIN,'').split('/').filter(Boolean);
  const items=[{"@type":"ListItem",position:1,name:"Trang chu",item:DOMAIN+"/"}];
  let acc='';
  parts.forEach((p,i)=>{ acc+='/'+p; const isLast=i===parts.length-1; items.push({"@type":"ListItem",position:i+2,name:isLast?title:decodeURIComponent(p).replace(/-/g,' '),item:DOMAIN+acc}); });
  return {"@context":"https://schema.org","@type":"BreadcrumbList",itemListElement:items};
}

fs.rmSync(OUT,{recursive:true,force:true});
fs.mkdirSync(OUT,{recursive:true});

const pages=[];
for(const f of walk(SRC)){
  if(!f.endsWith('.md')) continue;
  const rel = path.relative(ROOT,f).replace(/\\/g,'/').toLowerCase().replace(/\.md$/,'.html');
  const md = fs.readFileSync(f,'utf8');
  const title = extractTitle(md);
  const desc = extractDesc(md);
  const body = mdToHtml(md);
  const canonical = `${DOMAIN}/${rel}`;
  const faqs = extractFAQ(md);
  const breadcrumbLd = breadcrumbJsonLd(canonical, title);
  const articleLd = {"@context":"https://schema.org","@type":"TechArticle",headline:title,description:desc,url:canonical,image:OG_IMAGE,inLanguage:"vi-VN",author:{"@type":"Organization",name:"Tram Pickle",url:DOMAIN},publisher:{"@type":"Organization",name:"Tram Pickle",logo:{"@type":"ImageObject",url:OG_IMAGE}}};
  const faqLd = faqs.length?{"@context":"https://schema.org","@type":"FAQPage",mainEntity:faqs.map(q=>({"@type":"Question",name:q,acceptedAnswer:{"@type":"Answer",text:`Xem chi tiet trong bai ${title}.`}}))}:null;
  const jlds = [breadcrumbLd, articleLd]; if(faqLd) jlds.push(faqLd);
  const ldScripts = jlds.map(j=>`<script type="application/ld+json">${JSON.stringify(j)}</script>`).join('\n');
  const html = `<!doctype html><html lang="vi"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/><title>${esc(title)} | trampickle.vn</title><meta name="description" content="${esc(desc)}"/><link rel="canonical" href="${canonical}"/><meta property="og:site_name" content="${esc(SITE_NAME)}"/><meta property="og:title" content="${esc(title)}"/><meta property="og:description" content="${esc(desc)}"/><meta property="og:url" content="${canonical}"/><meta property="og:type" content="article"/><meta property="og:image" content="${OG_IMAGE}"/><meta property="og:locale" content="vi_VN"/><meta name="twitter:card" content="summary_large_image"/><meta name="twitter:title" content="${esc(title)}"/><meta name="twitter:description" content="${esc(desc)}"/><meta name="twitter:image" content="${OG_IMAGE}"/>${ldScripts}<script defer src="${TRACK_JS}"></script><style>body{max-width:820px;margin:0 auto;padding:24px;font-family:system-ui,Arial;line-height:1.6;color:#222} a{color:#0b57d0} pre{background:#f6f8fa;padding:12px;overflow:auto} code{background:#f0f0f0;padding:2px 4px;border-radius:4px} img{max-width:100%} h1,h2{border-bottom:1px solid #eee;padding-bottom:8px} nav a{margin-right:12px}</style></head><body><nav><a href="/"><strong>trampickle.vn</strong></a> <a href="/giao-trinh-sua-vot/readme.html">Giao trinh</a> <a href="/giao-trinh-sua-vot/nguon.html">Nguon</a> <a href="/sitemap.xml">Sitemap</a></nav>${body}<hr/><footer><p>© trampickle.vn - Giao trinh sua vot Pickleball. <a href="${DOMAIN}/">Trang chu</a> · <a href="/lien-he">Lien he 58 To Huu, Dai Mo, Ha Noi</a></p></footer></body></html>`;
  const outPath = path.join(OUT, rel);
  fs.mkdirSync(path.dirname(outPath),{recursive:true});
  fs.writeFileSync(outPath, html,'utf8');
  pages.push(rel);
  // also copy readme.html to index.html for directory index
  if(rel.endsWith('readme.html')){
    const idx = path.join(path.dirname(outPath),'index.html');
    if(!fs.existsSync(idx)) fs.copyFileSync(outPath, idx);
  }
}

// --- bai-viet: build 8 bai tu content/bai-viet/*.md -> dist/bai-viet/<slug>/index.html ---
const BAIVIET_OG_MAP = {
  "bang-gia-sua-vot-pickleball-ha-noi-2025": "/content/hinh-anh/01-bang-gia-2025.jpg",
  "nut-vien-vot-pickleball-sua-duoc-khong": "/content/hinh-anh/02-nut-vien.jpg",
  "vot-tach-lop-bong-mat-cach-xu-ly": "/content/hinh-anh/03-tach-lop.jpg",
  "diem-chet-loi-dap-vot-pickleball": "/content/hinh-anh/04-diem-chet.jpg",
  "khi-nao-thay-de-giay-pickleball": "/content/hinh-anh/05-de-giay-mon.jpg",
  "ve-sinh-giay-pickleball-dung-cach": "/content/hinh-anh/06-ve-sinh-giay.jpg",
  "thay-grip-can-bang-vot-khi-nao": "/content/hinh-anh/07-thay-grip.jpg",
  "bao-quan-vot-pickleball-ben-gap-doi": "/content/hinh-anh/08-bao-quan.jpg",
};
function extractSlug(md, fallback){
  const m = md.match(/Slug:\s*`([^`]+)`/);
  if(m) return m[1].trim();
  const m2 = md.match(/slug:\s*`([^`]+)`/i);
  if(m2) return m2[1].trim();
  return fallback.replace(/\.md$/,'').replace(/^\d+-/,'');
}
function extractMeta(md, key){
  const re = new RegExp(key + "\\s*[:\\-]\\s*([^\\n]+)", "i");
  const m2 = md.match(re);
  return m2 ? m2[1].replace(/^[\s`]+|[\s`]+$/g, "").trim() : "";
}
function extractFaqJson(md){
  const m = md.match(/```json([\s\S]*?)```/);
  if(!m) return null;
  try{ const j = JSON.parse(m[1]); if(j["@type"]==="FAQPage") return j; }catch(e){}
  return null;
}
if(fs.existsSync(BAIVIET_SRC)){
  for(const f of walk(BAIVIET_SRC)){
    if(!f.endsWith(".md")) continue;
    const md = fs.readFileSync(f,"utf8");
    const slug = extractSlug(md, path.basename(f));
    const title = extractTitle(md);
    let desc = extractMeta(md, "Meta desc");
    if(!desc) desc = extractDesc(md);
    const body = mdToHtml(cleanBody(md));
    const rel = "bai-viet/" + slug + "/index.html";
    const canonical = DOMAIN + "/bai-viet/" + slug;
    const ogImg = BAIVIET_OG_MAP[slug] ? DOMAIN + BAIVIET_OG_MAP[slug] : OG_IMAGE;
    const breadcrumbLd = breadcrumbJsonLd(canonical, title);
    const articleLd = {"@context":"https://schema.org","@type":"BlogPosting",headline:title,description:desc,url:canonical,image:ogImg,inLanguage:"vi-VN",author:{"@type":"Organization",name:"Tram Pickle",url:DOMAIN},publisher:{"@type":"Organization",name:"Tram Pickle",logo:{"@type":"ImageObject",url:OG_IMAGE}}};
    const faqJson = extractFaqJson(md);
    const jlds = [breadcrumbLd, articleLd];
    if(faqJson) jlds.push(faqJson);
    const ldScripts = jlds.map(j=>"<script type=\"application/ld+json\">"+JSON.stringify(j)+"</script>").join("\n");
    const html = "<!doctype html><html lang=\"vi\"><head><meta charset=\"utf-8\"/><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\"/><title>"+esc(title)+" | trampickle.vn</title><meta name=\"description\" content=\""+esc(desc)+"\"/><link rel=\"canonical\" href=\""+canonical+"\"/><meta property=\"og:site_name\" content=\""+esc(SITE_NAME)+"\"/><meta property=\"og:title\" content=\""+esc(title)+"\"/><meta property=\"og:description\" content=\""+esc(desc)+"\"/><meta property=\"og:url\" content=\""+canonical+"\"/><meta property=\"og:type\" content=\"article\"/><meta property=\"og:image\" content=\""+ogImg+"\"/><meta property=\"og:locale\" content=\"vi_VN\"/><meta name=\"twitter:card\" content=\"summary_large_image\"/>"+ldScripts+"<script defer src=\""+TRACK_JS+"\"></script><style>body{max-width:820px;margin:0 auto;padding:24px;font-family:system-ui,Arial;line-height:1.6;color:#222} a{color:#0b57d0} pre{background:#f6f8fa;padding:12px;overflow:auto} code{background:#f0f0f0;padding:2px 4px;border-radius:4px} img{max-width:100%} h1,h2{border-bottom:1px solid #eee;padding-bottom:8px} nav a{margin-right:12px}</style></head><body><nav><a href=\"/\"><strong>trampickle.vn</strong></a> <a href=\"/bai-viet\">Bai viet</a> <a href=\"/giao-trinh-sua-vot/readme.html\">Giao trinh</a> <a href=\"/sitemap.xml\">Sitemap</a></nav>"+body+"<hr/><footer><p>\u00a9 trampickle.vn - Sua chua vot Pickleball. <a href=\""+DOMAIN+"/\">Trang chu</a> \u00b7 <a href=\"/lien-he\">Lien he 58 To Huu, Dai Mo, Ha Noi</a></p></footer></body></html>";
    const outPath = path.join(OUT, rel);
    fs.mkdirSync(path.dirname(outPath),{recursive:true});
    fs.writeFileSync(outPath, html,"utf8");
    pages.push("bai-viet/"+slug+"/index.html");
    pages.push("bai-viet/"+slug);
  }
}

// root index
const rootIndex = `<!doctype html><html lang="vi"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/><title>trampickle.vn - Sua chua vot Pickleball</title><meta name="description" content="Sua chua vot Pickleball chuyen nghiep - Giao trinh sua vot, dich vu, lien he trampickle.vn"/><link rel="canonical" href="${DOMAIN}/"/><meta property="og:title" content="trampickle.vn"/><meta property="og:type" content="website"/><meta property="og:url" content="${DOMAIN}/"/><meta property="og:image" content="${OG_IMAGE}"/><script defer src="${TRACK_JS}"></script></head><body style="max-width:820px;margin:0 auto;padding:24px;font-family:system-ui"><h1>trampickle.vn</h1><p>Chuyen sua chua vot Pickleball.</p><ul>${pages.slice(0,30).map(p=>`<li><a href="/${p}">${p}</a></li>`).join('')}</ul><p><a href="/giao-trinh-sua-vot/readme.html">Vao giao trinh →</a></p></body></html>`;
fs.writeFileSync(path.join(OUT,'index.html'), rootIndex,'utf8');
// robots: copy từ gốc (đã có Disallow /qt ...)
if(fs.existsSync(path.join(ROOT,'robots.txt'))) fs.copyFileSync(path.join(ROOT,'robots.txt'), path.join(OUT,'robots.txt'));
// sitemap: hợp nhất GO_CORE_URLS + trang giao-trinh-sua-vot -> dist/sitemap.xml
// (để bản tĩnh không làm Google mù /, /dich-vu, /bai-viet)
{
  const allUrls = [...GO_CORE_URLS.map(p=>`${DOMAIN}${p}`), ...pages.map(p=>`${DOMAIN}/${p}`)];
  const uniq = [...new Set(allUrls)].sort();
  const sm = `<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n${uniq.map(u=>`  <url><loc>${u}</loc></url>`).join('\n')}\n</urlset>\n`;
  fs.writeFileSync(path.join(OUT,'sitemap.xml'), sm, 'utf8');
  // đồng bộ lại sitemap.xml gốc để lần sau không lệch
  fs.writeFileSync(path.join(ROOT,'sitemap.xml'), sm, 'utf8');
}
// copy assets if any
if(fs.existsSync(path.join(ROOT,'giao-trinh-sua-vot/anh'))) { fs.cpSync(path.join(ROOT,'giao-trinh-sua-vot/anh'), path.join(OUT,'giao-trinh-sua-vot/anh'), {recursive:true}); }
if(fs.existsSync(path.join(ROOT,'assets/js'))) { fs.cpSync(path.join(ROOT,'assets/js'), path.join(OUT,'assets/js'), {recursive:true}); }
if(fs.existsSync(path.join(ROOT,'content/hinh-anh'))) { fs.cpSync(path.join(ROOT,'content/hinh-anh'), path.join(OUT,'content/hinh-anh'), {recursive:true}); }
// admin panel rieng: /admin/thong-ke.html (noindex, ban ham analytics tu dong loai /admin)
{
  const adminHtml = fs.readFileSync(path.join(ROOT,'assets/admin-thong-ke.html'),'utf8');
  const outAdmin = path.join(OUT,'admin/thong-ke.html');
  fs.mkdirSync(path.dirname(outAdmin),{recursive:true});
  fs.writeFileSync(outAdmin, adminHtml,'utf8');
  const idxAdmin = path.join(OUT,'admin/index.html');
  if(!fs.existsSync(idxAdmin)) fs.copyFileSync(outAdmin, idxAdmin);
}
// llms.txt + ai.txt for AI Search (https://llmstxt.org) — static fallback for LLMs
{
  const lines = [
    `# ${SITE_NAME}`,
    `> ${extractDesc(fs.readFileSync(path.join(SRC,'README.md'),'utf8'))}`,
    ``,
    `URL: ${DOMAIN}/`,
    `Sitemap: ${DOMAIN}/sitemap.xml`,
    ``,
    `## Giao trinh sua vot Pickleball`,
    `Giao trinh day nghe sua vot Pickleball tu nen tang den cac ca sua thuc te.`,
    ...pages.slice(0,60).map(p=>`- [${p}](${DOMAIN}/${p})`),
    ``,
    `## Luu y AI`,
    `- Dia chi 58 To Huu, Dai Mo, Ha Noi hien thi tren site de gui hang; dang service-area (online, hen truoc).`,
    `- Sitemap dong cua Go (VPS) la canonical; sitemap tinh nay chi fallback.`,
  ];
  const llm = lines.join('\n')+'\n';
  fs.writeFileSync(path.join(OUT,'llms.txt'), llm,'utf8');
  fs.writeFileSync(path.join(OUT,'ai.txt'), `# AI access\nAllow: /\nSitemap: ${DOMAIN}/sitemap.xml\nLLMs-txt: ${DOMAIN}/llms.txt\n`,'utf8');
  // dong bo ra goc de preview
  fs.writeFileSync(path.join(ROOT,'llms.txt'), llm,'utf8');
}
console.log('built',pages.length,'pages to dist/');