(function(){
  "use strict";
  var LS='tp_stats_v1',SS='tp_sid_v1';
  var EP='/api/track';
  function sid(){
    try{
      var s=sessionStorage.getItem(SS);
      if(!s){ s=Math.random().toString(36).slice(2,10)+Date.now().toString(36); sessionStorage.setItem(SS,s); }
      return s;
    }catch(e){ return 'nosess'; }
  }
  function isAdminPath(){ return location.pathname.indexOf('/admin')===0 || location.pathname.indexOf('/qt')===0; }
  function isBot(){
    var ua=(navigator.userAgent||'').toLowerCase();
    return /bot|crawl|spider|slurp|mediapartners|baidu|yandex|semrush|ahrefs|headless|puppeteer|playwright/.test(ua);
  }
  if(isAdminPath() || isBot() || navigator.webdriver) return;
  // respect DNT
  try{ if(navigator.doNotTrack==='1' || window.doNotTrack==='1') return; }catch(e){}
  var now=Date.now();
  var page=location.pathname + location.search;
  // debounce same page in 3s
  try{
    var last=sessionStorage.getItem('_tp_last');
    if(last){ var o=JSON.parse(last); if(o.p===page && now - o.t < 3000) return; }
    sessionStorage.setItem('_tp_last', JSON.stringify({p:page,t:now}));
  }catch(e){}
  var ev={
    t: now,
    iso: new Date(now).toISOString(),
    p: page,
    path: location.pathname,
    ref: document.referrer ? document.referrer.slice(0,512) : '',
    sid: sid(),
    w: screen.width, h: screen.height,
    lang: (navigator.language||'').slice(0,8),
    vp: (innerWidth+'x'+innerHeight)
  };
  // localStorage
  try{
    var a=JSON.parse(localStorage.getItem(LS)||'[]');
    if(!Array.isArray(a)) a=[];
    a.push(ev);
    // cap 8000 events ~ ~2MB, trim oldest
    if(a.length>8000) a=a.slice(a.length-8000);
    localStorage.setItem(LS, JSON.stringify(a));
    // also per-page counter for quick widget
    var ck='tp_cnt_'+location.pathname;
    var c=parseInt(localStorage.getItem(ck)||'0',10)+1;
    localStorage.setItem(ck, String(c));
  }catch(e){}
  // hybrid POST if backend exists
  var payload=JSON.stringify(ev);
  try{
    if(navigator.sendBeacon){
      // sendBeacon swallows errors; server may 404 - that's fine
      navigator.sendBeacon(EP, new Blob([payload],{type:'application/json'}));
    } else {
      fetch(EP,{method:'POST',headers:{'Content-Type':'application/json'},body:payload,keepalive:true}).catch(function(){});
    }
  }catch(e){}
  // time on page beacon
  var t0=now;
  function leave(){
    var d=Math.round((Date.now()-t0)/1000);
    if(d<3) return;
    try{
      var a2=JSON.parse(localStorage.getItem(LS)||'[]');
      // attach dwell to last event for same sid+page
      for(var i=a2.length-1;i>=0 && i>a2.length-6;i--){
        if(a2[i].sid===ev.sid && a2[i].p===ev.p && !a2[i].dwell){ a2[i].dwell=d; break; }
      }
      localStorage.setItem(LS, JSON.stringify(a2));
    }catch(e){}
  }
  window.addEventListener('pagehide', leave);
  window.addEventListener('visibilitychange', function(){ if(document.visibilityState==='hidden') leave(); });
})();
