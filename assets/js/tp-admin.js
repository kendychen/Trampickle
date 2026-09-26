(function(){"use strict";
var LS='tp_stats_v1',PK='tp_admin_ok',PASS='TramPickle2026';
function q(s){return document.querySelector(s);}
function esc(s){var d=document.createElement('div');d.textContent=s;return d.innerHTML;}
function needAuth(){try{return sessionStorage.getItem(PK)!=='1';}catch(e){return true;}}
function showLogin(){
 var el=q('#login');el.style.display='block';q('#panel').style.display='none';
 q('#btnLogin').onclick=function(){
  var v=q('#pass').value;
  if(v===PASS){try{sessionStorage.setItem(PK,'1');}catch(e){}
   el.style.display='none';q('#panel').style.display='block';render();
  } else q('#loginMsg').textContent='Sai mat khau.';
 };
 q('#pass').addEventListener('keydown',function(e){if(e.key==='Enter')q('#btnLogin').click();});
}
function load(){try{var a=JSON.parse(localStorage.getItem(LS)||'[]');return Array.isArray(a)?a:[];}catch(e){return [];}}
function fmtDate(d){return d.toISOString().slice(0,10);}
function fmt(n){return n.toLocaleString('vi-VN');}
function render(){
 var data=load(),days=parseInt(q('#range').value,10)||14;
 var cutoff=new Date();cutoff.setHours(0,0,0,0);cutoff.setDate(cutoff.getDate()-days+1);
 var cutoffMs=cutoff.getTime();
 var filtered=data.filter(function(e){return e.t>=cutoffMs;});
 var total=filtered.length,uniq=new Set(filtered.map(function(e){return e.sid;})).size;
 var todayStr=fmtDate(new Date());
 var today=filtered.filter(function(e){return (e.iso||'').slice(0,10)===todayStr;}).length;
 var withDwell=filtered.filter(function(e){return typeof e.dwell==='number';});
 var avgDwell=withDwell.length?Math.round(withDwell.reduce(function(s,e){return s+e.dwell;},0)/withDwell.length):0;
 q('#kTotal').textContent=fmt(total);
 q('#kUniq').textContent=fmt(uniq);
 q('#kToday').textContent=fmt(today);
 q('#kDwell').textContent=avgDwell?avgDwell+'s':'-';
 q('#kRange').textContent=days+' ngay';
 q('#meta').textContent='Tong luu: '+fmt(data.length)+' events (localStorage trinh duyet nay). Hybrid: tu POST /api/track neu backend ton tai.';
 var byDay={};for(var i=0;i<days;i++){var d=new Date(cutoff);d.setDate(cutoff.getDate()+i);byDay[fmtDate(d)]=0;}
 filtered.forEach(function(e){var k=(e.iso||'').slice(0,10);if(k in byDay)byDay[k]++;else if(k)byDay[k]=(byDay[k]||0)+1;});
 drawChart(byDay);
 var byPage={};filtered.forEach(function(e){var k=e.path||e.p||'/';byPage[k]=(byPage[k]||0)+1;});
 var topP=Object.entries(byPage).sort(function(a,b){return b[1]-a[1];}).slice(0,20);
 q('#topPages').innerHTML=topP.length?topP.map(function(r){return '<tr><td>'+esc(r[0])+'</td><td style="text-align:right">'+fmt(r[1])+'</td><td style="text-align:right">'+(total?Math.round(r[1]*100/total):0)+'%</td></tr>';}).join(''):'<tr><td colspan=3>chua co du lieu</td></tr>';
 var byRef={};filtered.forEach(function(e){var k=(e.ref||'').slice(0,80)||'(direct)';try{var u=new URL(e.ref);k=u.hostname+u.pathname.slice(0,40);}catch(_){}byRef[k]=(byRef[k]||0)+1;});
 var topR=Object.entries(byRef).sort(function(a,b){return b[1]-a[1];}).slice(0,15);
 q('#topRef').innerHTML=topR.map(function(r){return '<tr><td>'+esc(r[0])+'</td><td style="text-align:right">'+fmt(r[1])+'</td></tr>';}).join('');
 var recent=filtered.slice(-80).reverse();
 q('#recent').innerHTML=recent.length?recent.map(function(e){return '<tr><td>'+esc((e.iso||'').replace('T',' ').slice(0,19))+'</td><td>'+esc(e.path||e.p)+'</td><td>'+esc((e.ref||'').slice(0,40))+'</td><td>'+esc(e.vp||'')+'</td><td>'+(e.dwell?e.dwell+'s':'')+'</td></tr>';}).join(''):'<tr><td colspan=5>chua co du lieu</td></tr>';
}

function drawChart(byDay){
 var c=q('#chart'); if(!c) return;
 var ctx=c.getContext('2d');
 var labels=Object.keys(byDay), vals=Object.values(byDay);
 var W=c.width,H=c.height,pad=28;
 ctx.clearRect(0,0,W,H);
 var max=Math.max(1, Math.max.apply(null, vals));
 ctx.strokeStyle='#e5e7eb';ctx.lineWidth=1;
 for(var i=0;i<=4;i++){var y=pad+(H-pad*2)*i/4;ctx.beginPath();ctx.moveTo(pad,y);ctx.lineTo(W-pad,y);ctx.stroke();ctx.fillStyle='#9ca3af';ctx.font='10px system-ui';ctx.fillText(String(Math.round(max*(1-i/4))),2,y+3);}
 var bw=(W-pad*2)/labels.length*0.62,gap=(W-pad*2)/labels.length*0.38;
 vals.forEach(function(v,idx){var x=pad+idx*(bw+gap)+gap*0.5,h=(v/max)*(H-pad*2)*0.92,y=H-pad-h;ctx.fillStyle='#16a34a';ctx.fillRect(x,y,bw,h);if(v>0){ctx.fillStyle='#111827';ctx.font='10px system-ui';ctx.textAlign='center';ctx.fillText(String(v),x+bw/2,y-4);}ctx.fillStyle='#6b7280';ctx.font='9px system-ui';ctx.textAlign='center';ctx.fillText(labels[idx].slice(5),x+bw/2,H-6);});
}
document.addEventListener('DOMContentLoaded',function(){
 if(needAuth()) showLogin(); else {q('#login').style.display='none';q('#panel').style.display='block';render();}
 q('#range').addEventListener('change',render);
 q('#btnExport').onclick=function(){var d=load();var b=new Blob([JSON.stringify(d,null,2)],{type:'application/json'});var a=document.createElement('a');a.href=URL.createObjectURL(b);a.download='tp-stats-'+fmtDate(new Date())+'.json';a.click();};
 q('#btnCsv').onclick=function(){var d=load();var rows=[['iso','path','ref','sid','vp','w','h','dwell'].join(',')];d.forEach(function(e){rows.push([e.iso||'',e.path||'','\"'+(e.ref||'').replace(/\"/g,'\"\"')+'\"',e.sid||'',e.vp||'',e.w||'',e.h||'',e.dwell||''].join(','));});var b=new Blob([rows.join('\n')],{type:'text/csv'});var a=document.createElement('a');a.href=URL.createObjectURL(b);a.download='tp-stats.csv';a.click();};
 q('#btnClear').onclick=function(){if(confirm('Xoa toan bo thong ke tren trinh duyet nay?')){localStorage.removeItem(LS);Object.keys(localStorage).forEach(function(k){if(k.indexOf('tp_cnt_')===0)localStorage.removeItem(k);});render();}};
 q('#btnLogout').onclick=function(){try{sessionStorage.removeItem(PK);}catch(e){}location.reload();};
});
})();

