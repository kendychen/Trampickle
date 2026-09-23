import fs from 'fs';
import path from 'path';
const ROOT = path.resolve('.');
const SRC = path.join(ROOT, 'giao-trinh-sua-vot');
const OUT = path.join(ROOT, 'dist');
// cây sitemap động của Go đã khai ở core/baiviet.go:hSitemap. Tập này chỉ
// build tài liệu tĩnh giao-trinh-sua-vot, nên sitemap tĩnh BỔ SUNG thêm các
// đường dẫn tiền phương (/ , /dich-vu ...) để khi nginx KHÔNG proxy về Go
// (deploy tĩnh) Google vẫn thấy đủ. Khi có proxy nginx (VPS Go đang chạy)
// thì file này bị qua mặt bởi proxy_pass — không xung đột.
const DOMAIN = 'https://trampickle.vn';
// Service-area business (online): địa chỉ 58 Tố Hữu - Đại Mỗ vẫn HIỂN THỊ
// trên website (footer/lien-he) để khách gửi hàng, nhưng KHÔNG đưa
// streetAddress chi tiết vào JSON-LD schema. Schema chỉ khai ở mức
// addressLocality Hà Nội + areaServed Hà Nội để Google/AI hiểu là dịch vụ
// online nhận qua gửi hàng, hẹn trước khi mang tới — không phải storefront.
const GO_CORE_URLS = ["/","/dich-vu","/bai-viet","/quy-trinh","/cau-hoi","/gioi-thieu","/ve-chung-toi","/lien-he","/chinh-sach"];

function walk(dir){ let r=[]; for(const e of fs.readdirSync(dir,{withFileTypes:true})){ const p=path.join(dir,e.name); if(e.isDirectory()) r.push(...walk(p)); else r.push(p);} return r; }
function mdToHtml(md){
  // very simple: headings, links, tables, lists
  let html = md
    .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
    // code blocks ```...```
    .replace(/```([\s\S]*?)```/g,(m,c)=>`<pre><code>${c}</code></pre>`)
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
    .replace(/!\[([^\]]*)\]\(([^)]+)\)/g,'<img alt="$1" src="$2" style="max-width:100%" />')
    // tables - keep as pre for simplicity (browser will render pipe)
    // paragraphs
  ;
  // split lines to paragraphs
  html = html.split(/\n{2,}/).map(block=>{
    const t=block.trim();
    if(!t) return '';
    if(t.startsWith('<h')||t.startsWith('<pre')||t.startsWith('<ul')||t.startsWith('<table')||t.startsWith('<img')) return t;
    // list
    if(/^[-*] /.test(t)) return '<ul>'+ t.split('\n').map(l=>l.replace(/^[-*] (.+)/,'<li>$1</li>')).join('') + '</ul>';
    if(/^\|/.test(t)) return `<pre>${t}</pre>`;
    return `<p>${t.replace(/\n/g,'<br/>')}</p>`;
  }).join('\n');
  return html;
}
function extractTitle(md){ const m=md.match(/^#\s+(.+)$/m); return m?m[1].trim():'Giao trinh sua vot Pickleball'; }
function extractDesc(md){ const m=md.match(/>\s*(.+)/); return m?m[1].slice(0,160):'Tai lieu day nghe sua vot pickleball - trampickle.vn'; }

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
  const html = `<!doctype html><html lang="vi"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/><title>${title} | trampickle.vn</title><meta name="description" content="${desc}"/><link rel="canonical" href="${canonical}"/><meta property="og:title" content="${title}"/><meta property="og:description" content="${desc}"/><meta property="og:url" content="${canonical}"/><meta property="og:type" content="article"/><style>body{max-width:820px;margin:0 auto;padding:24px;font-family:system-ui,Arial;line-height:1.6;color:#222} a{color:#0b57d0} pre{background:#f6f8fa;padding:12px;overflow:auto} code{background:#f0f0f0;padding:2px 4px;border-radius:4px} img{max-width:100%} h1,h2{border-bottom:1px solid #eee;padding-bottom:8px} nav a{margin-right:12px}</style></head><body><nav><a href="/"><strong>trampickle.vn</strong></a> <a href="/giao-trinh-sua-vot/readme.html">Giao trinh</a> <a href="/giao-trinh-sua-vot/nguon.html">Nguon</a> <a href="/sitemap.xml">Sitemap</a></nav>${body}<hr/><footer><p>© trampickle.vn - Giao trinh sua vot Pickleball. <a href="${DOMAIN}/">Trang chu</a></p></footer></body></html>`;
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
// root index
const rootIndex = `<!doctype html><html lang="vi"><head><meta charset="utf-8"/><meta name="viewport" content="width=device-width,initial-scale=1"/><title>trampickle.vn - Sua chua vot Pickleball</title><meta name="description" content="Sua chua vot Pickleball chuyen nghiep - Giao trinh sua vot, dich vu, lien he trampickle.vn"/><link rel="canonical" href="${DOMAIN}/"/></head><body style="max-width:820px;margin:0 auto;padding:24px;font-family:system-ui"><h1>trampickle.vn</h1><p>Chuyen sua chua vot Pickleball.</p><ul>${pages.slice(0,30).map(p=>`<li><a href="/${p}">${p}</a></li>`).join('')}</ul><p><a href="/giao-trinh-sua-vot/readme.html">Vao giao trinh →</a></p></body></html>`;
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
console.log('built',pages.length,'pages to dist/');