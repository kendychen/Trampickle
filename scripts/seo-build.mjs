import { readFileSync, writeFileSync, mkdirSync } from 'fs';
import yaml from 'js-yaml';
const src='seo/keywords.yaml', dst='seo/keywords.json';
try{
  const doc=yaml.load(readFileSync(src,'utf8'));
  const keywords=doc.keywords||[];
  mkdirSync('seo',{recursive:true});
  writeFileSync(dst, JSON.stringify({keywords,total:keywords.length},null,2));
  console.log(`wrote ${keywords.length} keywords -> ${dst}`);
}catch(e){ console.error(e.message); process.exit(1); }
