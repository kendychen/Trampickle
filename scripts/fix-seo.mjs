import fs from 'fs';
import path from 'path';
const ROOT = path.resolve('D:/Dự Án Sửa Chữa Pickleball');
const GT = path.join(ROOT, 'giao-trinh-sua-vot');
console.log('ROOT', ROOT);
// 1. Rename 3 files to lowercase using fs
for (const name of ['00-KE-HOACH.md','NGUON.md','VIDEO.md']){
  const p = path.join(GT, name);
  const low = path.join(GT, name.toLowerCase());
  if (fs.existsSync(p)){
    const tmp = path.join(GT, path.parse(name).name + '_tmp' + path.parse(name).ext);
    try { fs.renameSync(p, tmp); fs.renameSync(tmp, low); console.log(`renamed ${name} -> ${name.toLowerCase()}`);} catch(e){ console.log(e.message)}
  } else {
    console.log('not found', p, 'check lower exists', fs.existsSync(low));
  }
}
// alias readme
function walk(dir){ let res=[]; for(const e of fs.readdirSync(dir,{withFileTypes:true})){ const fp=path.join(dir,e.name); if(e.isDirectory()) res.push(...walk(fp)); else res.push(fp);} return res;}
for (const f of walk(GT)){
  if (path.basename(f)==='README.md'){
    const alias = path.join(path.dirname(f),'readme.md');
    if (!fs.existsSync(alias)){ fs.copyFileSync(f, alias); console.log('alias', alias);}
  }
}
// 2 fix links
const mdRe = /\[([^\]]+)\]\(([^)]+)\)/g;
let fixes=0;
for (const f of walk(GT)){
  if (!f.endsWith('.md')) continue;
  let text = fs.readFileSync(f,'utf8');
  let orig=text;
  text = text.replace(mdRe, (m,label,url)=>{
    if (url.startsWith('http://')||url.startsWith('https://')||url.startsWith('#')||url.startsWith('mailto:')||url.startsWith('tel:')) return m;
    let base=url, frag='', q='';
    if (base.includes('#')){ const i=base.indexOf('#'); frag=base.slice(i); base=base.slice(0,i);}
    if (base.includes('?')){ const i=base.indexOf('?'); q=base.slice(i); base=base.slice(0,i);}
    if (/[A-Z]/.test(base)){ fixes++; return `[${label}](${base.toLowerCase()}${q}${frag})`; }
    return m;
  });
  if (text!==orig){ fs.writeFileSync(f,text,'utf8'); console.log('fixed', path.relative(ROOT,f));}
}
console.log('total fixes', fixes);
// 3 add Xem them
const related = {
  '01-nen-tang/1-3-dung-cu-an-toan.md': ['01-nen-tang/1-1-cau-tao-vot.md','01-nen-tang/1-2-vat-lieu-keo.md','02-quy-trinh/2-1-nhan-vot.md','02-quy-trinh/2-2-chan-doan.md'],
  '00-ke-hoach.md': ['readme.md','nguon.md','video.md'],
  'anh/readme.md': ['../01-nen-tang/1-1-cau-tao-vot.md'],
  '_nguon/readme.md': ['../nguon.md'],
};
for (const [rel,links] of Object.entries(related)){
  const p = path.join(GT, rel);
  if (fs.existsSync(p)){
    let t = fs.readFileSync(p,'utf8');
    if (!t.toLowerCase().includes('xem them')){
      const sec = '\n\n## Xem them\n\n' + links.map(l=>`- [${l}](${l})`).join('\n') + '\n';
      if (t.includes('## Cau hoi')) t=t.replace('## Cau hoi', sec+'## Cau hoi');
      else t=t.trimEnd()+sec;
      fs.writeFileSync(p,t,'utf8'); console.log('added Xem them', rel);
    }
  }
}
// 4 robots + sitemap
fs.writeFileSync(path.join(ROOT,'robots.txt'),'User-agent: *\nAllow: /\nSitemap: /sitemap.xml\n','utf8');
console.log('robots ok');
const base_url='https://example.com/';
let urls=[];
for (const f of walk(GT)) if (f.endsWith('.md')){ const rel=path.relative(ROOT,f).replace(/\\/g,'/').toLowerCase().replace('.md','.html'); urls.push(base_url+rel); }
let xml='<?xml version="1.0" encoding="UTF-8"?>\n<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">\n';
for (const u of urls.sort()) xml+=`  <url><loc>${u}</loc></url>\n`;
xml+='</urlset>\n';
fs.writeFileSync(path.join(ROOT,'sitemap.xml'), xml,'utf8');
console.log('sitemap', urls.length);
console.log('done');