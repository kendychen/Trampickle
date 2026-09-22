import fs from 'fs';
import path from 'path';
const ROOT = path.resolve('D:/Dự Án Sửa Chữa Pickleball');
const GT = path.join(ROOT, 'giao-trinh-sua-vot');
function walk(dir){ let r=[]; for(const e of fs.readdirSync(dir,{withFileTypes:true})){ const p=path.join(dir,e.name); if(e.isDirectory()) r.push(...walk(p)); else r.push(p);} return r; }
// Ensure all internal link URLs are lowercase (exclude external)
let fix=0;
for(const f of walk(GT)){
  if(!f.endsWith('.md')) continue;
  let t=fs.readFileSync(f,'utf8');
  const orig=t;
  t=t.replace(/\[([^\]]+)\]\(([^)]+)\)/g,(m,label,url)=>{
    if(url.startsWith('http://')||url.startsWith('https://')||url.startsWith('#')||url.startsWith('mailto:')||url.startsWith('tel:')) return m;
    // separate # ?
    let base=url, frag='', q='';
    if(base.includes('#')){ const i=base.indexOf('#'); frag=base.slice(i); base=base.slice(0,i); }
    if(base.includes('?')){ const i=base.indexOf('?'); q=base.slice(i); base=base.slice(0,i); }
    if(/[A-Z]/.test(base)){
      fix++;
      return `[${label}](${base.toLowerCase()}${q}${frag})`;
    }
    return m;
  });
  if(t!==orig){ fs.writeFileSync(f,t,'utf8'); console.log('lowered',path.relative(ROOT,f)); }
}
console.log('lower fix',fix);
// Add prev/next + related navigation to each teaching file
const order=[
  '01-nen-tang/1-1-cau-tao-vot.md',
  '01-nen-tang/1-2-vat-lieu-keo.md',
  '01-nen-tang/1-3-dung-cu-an-toan.md',
  '02-quy-trinh/2-1-nhan-vot.md',
  '02-quy-trinh/2-2-chan-doan.md',
  '02-quy-trinh/2-3-bao-gia.md',
  '02-quy-trinh/2-4-nghiem-thu.md',
  '03-ca-sua/readme.md',
  '03-ca-sua/3-1-nut-vo-vien-canh.md',
  '03-ca-sua/3-2-tach-lop.md',
  '03-ca-sua/3-3-diem-chet-loi-dap.md',
  '03-ca-sua/3-4-can-vot-gay-long.md',
  '03-ca-sua/3-5-thay-grip.md',
  '03-ca-sua/3-6-can-bang-vot.md',
  '03-ca-sua/3-7-nut-mat-vot.md',
  '03-ca-sua/3-8-ban-mat-mat-nham.md',
  '03-ca-sua/3-9-lao-xao-trong-vot.md',
  '04-lam-nghe/4-1-gia-bao-hanh.md',
  '04-lam-nghe/4-2-loi-tho-moi.md',
  '04-lam-nghe/4-3-khao-sat-thi-truong-can-gay.md',
];
function relPath(from,to){
  const fromDir=path.dirname(from);
  let rel=path.relative(fromDir,to).replace(/\\/g,'/');
  return rel.toLowerCase();
}
for(let i=0;i<order.length;i++){
  const cur=order[i];
  const fp=path.join(GT,cur);
  if(!fs.existsSync(fp)) continue;
  let t=fs.readFileSync(fp,'utf8');
  if(t.includes('<!-- seo-nav -->')) continue;
  const prev=i>0?order[i-1]:null;
  const next=i<order.length-1?order[i+1]:null;
  let nav='\n\n<!-- seo-nav -->\n---\n\n**Dieu huong:** ';
  if(prev) nav+=`[← Bai truoc](${relPath(cur,prev)}) | `;
  nav+=`[Muc luc](../readme.md) | [Nguon](../nguon.md)`;
  if(next) nav+=` | [Bai tiep →](${relPath(cur,next)})`;
  nav+='\n';
  // related
  const relatedMap={
    '03-ca-sua/3-2-tach-lop.md': ['03-ca-sua/3-3-diem-chet-loi-dap.md','03-ca-sua/3-7-nut-mat-vot.md','02-quy-trinh/2-2-chan-doan.md'],
    '03-ca-sua/3-3-diem-chet-loi-dap.md': ['03-ca-sua/3-2-tach-lop.md','03-ca-sua/3-9-lao-xao-trong-vot.md','02-quy-trinh/2-2-chan-doan.md'],
    '03-ca-sua/3-7-nut-mat-vot.md': ['03-ca-sua/3-2-tach-lop.md','03-ca-sua/3-1-nut-vo-vien-canh.md','01-nen-tang/1-2-vat-lieu-keo.md'],
    '03-ca-sua/3-4-can-vot-gay-long.md': ['04-lam-nghe/4-3-khao-sat-thi-truong-can-gay.md','03-ca-sua/3-5-thay-grip.md','03-ca-sua/3-6-can-bang-vot.md'],
    '03-ca-sua/3-8-ban-mat-mat-nham.md': ['02-quy-trinh/2-2-chan-doan.md','03-ca-sua/3-2-tach-lop.md','03-ca-sua/3-3-diem-chet-loi-dap.md'],
  };
  if(relatedMap[cur]){
    nav+='\n**Bai lien quan:** ' + relatedMap[cur].map(r=>`[${path.basename(r)}](${relPath(cur,r)})`).join(' · ') + '\n';
  }
  t=t.trimEnd()+nav+'\n';
  fs.writeFileSync(fp,t,'utf8');
  console.log('nav added',cur);
}
// Ensure README has internal link density
console.log('done');