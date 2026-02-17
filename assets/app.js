function openTab(id, ev){
  document.querySelectorAll('.tabcontent').forEach(x=>x.style.display='none');
  document.querySelectorAll('.tablink').forEach(x=>x.classList.remove('active'));
  document.getElementById(id).style.display='block';
  try{ if(ev && ev.target) ev.target.classList.add('active'); }catch(e){}
}
function badgeVal(v,label,suf){if(v==null)return'';let cls='good';if(v>90)cls='bad';else if(v>75)cls='warn';return `<span class="badge ${cls}">${label}: ${v}${suf}</span>`;}
function badgeLatency(v){if(v==null)return'';let cls='good';if(v>500)cls='bad';else if(v>100)cls='warn';return `<span class="badge ${cls}">Latency: ${v} ms</span>`;}

// Human-readable uptime badge from seconds
function fmtDuration(sec){
  if(typeof sec!=="number" || !isFinite(sec) || sec<0) return null;
  let s=Math.floor(sec);
  const d=Math.floor(s/86400); s%=86400;
  const h=Math.floor(s/3600); s%=3600;
  const m=Math.floor(s/60);
  const parts=[];
  if(d) parts.push(d+"d"); if(h) parts.push(h+"h"); if(m||parts.length===0) parts.push(m+"m");
  return parts.join(" ");
}
function badgeUptime(v){ const t=fmtDuration(v); return t?`<span class="badge good">Uptime: ${t}</span>`:''; }

// Ack badge with remaining time (h m s)
function fmtRemain(sec){
  if(typeof sec!=="number" || !isFinite(sec) || sec<=0) return null;
  let s=Math.floor(sec);
  const d=Math.floor(s/86400); s%=86400;
  const h=Math.floor(s/3600); s%=3600;
  const m=Math.floor(s/60);   s%=60;
  const parts=[];
  if(d) parts.push(d+"d"); if(h) parts.push(h+"h"); if(m) parts.push(m+"m"); parts.push(s+"s");
  return parts.join(" ");
}
function badgeAck(until){
  const nowSec=Math.floor(Date.now()/1000);
  if(!(until && until>nowSec)) return '';
  const rem=until-nowSec;
  const t=fmtRemain(rem);
  return t?`<span class="badge warn">ACK: ${t}</span>`:'';
}
// Outage badge showing how long offline
function badgeOutage(since, online){
  if(!since || online) return '';
  const nowSec=Math.floor(Date.now()/1000);
  const dur=nowSec - since;
  const t=fmtRemain(dur);
  return t?`<span class="badge bad">Outage: ${t}</span>`:'';
}

// Siren scheduling state
let sirenTimeout=null;          // Next scheduled siren timer
let sirenNextDelayMs=30000;     // 30s for first alert, 10m for repeats
let sirenShouldAlertPrev=false; // Track transitions to reset delay
let devicesCache=[];
let audioUnlocked=false;
const ACK_DURATION_MAP={'30m':1800,'1h':3600,'6h':21600,'8h':28800,'12h':43200};
let renderMeta={http:'--',api_latency:'--',updated:'--'};
let fetchRequestId=0;
let mutationVersion=0;
const pendingSimOverrides=new Map();
const SIM_OVERRIDE_TTL_MS = 60000;
let pollTimer=null;
let chartJsLoader=null;
let cpeHistoryChart=null;
let cpeHistoryReqId=0;
const AP_ALERT_GRACE_SEC = 900; // 15 minutes
const MIN_OFFLINE_TS = 1; // guard against missing/zero offline_since

const POLL_INTERVAL_NORMAL_MS = 5000;
const POLL_INTERVAL_FAST_MS = 2000;
const POLL_INTERVAL_ERROR_MS = 15000;

const DASH_SETTINGS_KEY = "nocwall.dashboard.settings.v1";
const DEFAULT_DASH_SETTINGS = {
  density: "normal",
  metrics: {
    cpu: true,
    ram: true,
    temp: true,
    latency: true,
    uptime: true,
    outage: true
  }
};

let dashSettings = loadDashSettings();

function loadDashSettings(){
  try{
    const raw = localStorage.getItem(DASH_SETTINGS_KEY);
    if(!raw){
      return JSON.parse(JSON.stringify(DEFAULT_DASH_SETTINGS));
    }
    const parsed = JSON.parse(raw);
    return normalizeDashSettings(parsed);
  }catch(_){
    return JSON.parse(JSON.stringify(DEFAULT_DASH_SETTINGS));
  }
}

function normalizeDashSettings(input){
  const normalized = JSON.parse(JSON.stringify(DEFAULT_DASH_SETTINGS));
  if(!input || typeof input !== "object"){
    return normalized;
  }
  if(input.density === "compact"){
    normalized.density = "compact";
  }
  const metricKeys = Object.keys(normalized.metrics);
  metricKeys.forEach(key=>{
    if(input.metrics && Object.prototype.hasOwnProperty.call(input.metrics,key)){
      normalized.metrics[key] = !!input.metrics[key];
    }
  });
  return normalized;
}

function saveDashSettings(){
  try{
    localStorage.setItem(DASH_SETTINGS_KEY, JSON.stringify(dashSettings));
  }catch(_){ }
}

function isMetricEnabled(metricName){
  return !!(dashSettings && dashSettings.metrics && dashSettings.metrics[metricName]);
}

function applyDisplayDensity(){
  if(!document.body) return;
  document.body.classList.toggle("density-compact", dashSettings.density === "compact");
}

function syncDisplayControls(){
  const density = document.getElementById("settingDensity");
  if(density){
    density.value = dashSettings.density;
  }
  const controlMap = {
    settingMetricCpu: "cpu",
    settingMetricRam: "ram",
    settingMetricTemp: "temp",
    settingMetricLatency: "latency",
    settingMetricUptime: "uptime",
    settingMetricOutage: "outage"
  };
  Object.keys(controlMap).forEach(id=>{
    const el = document.getElementById(id);
    if(el){
      el.checked = isMetricEnabled(controlMap[id]);
    }
  });
}

function applyDashboardSettings(){
  applyDisplayDensity();
  syncDisplayControls();
}

function bindDisplayControls(){
  const density = document.getElementById("settingDensity");
  if(density){
    density.addEventListener("change",()=>{
      dashSettings.density = density.value === "compact" ? "compact" : "normal";
      applyDashboardSettings();
      saveDashSettings();
      renderDevices();
    });
  }

  const metricMap = {
    settingMetricCpu: "cpu",
    settingMetricRam: "ram",
    settingMetricTemp: "temp",
    settingMetricLatency: "latency",
    settingMetricUptime: "uptime",
    settingMetricOutage: "outage"
  };

  Object.keys(metricMap).forEach(id=>{
    const el = document.getElementById(id);
    if(!el) return;
    el.addEventListener("change",()=>{
      dashSettings.metrics[metricMap[id]] = !!el.checked;
      saveDashSettings();
      renderDevices();
    });
  });

  const reset = document.getElementById("settingReset");
  if(reset){
    reset.addEventListener("click",()=>{
      dashSettings = JSON.parse(JSON.stringify(DEFAULT_DASH_SETTINGS));
      applyDashboardSettings();
      saveDashSettings();
      renderDevices();
    });
  }
}

function initDisplaySettings(){
  applyDashboardSettings();
  bindDisplayControls();
}

function schedulePoll(delayMs){
  if(pollTimer){
    clearTimeout(pollTimer);
  }
  const ms = (typeof delayMs === 'number' && isFinite(delayMs)) ? Math.max(0, delayMs) : POLL_INTERVAL_NORMAL_MS;
  pollTimer = setTimeout(()=>{ fetchDevices({ reason:'scheduled' }); }, ms);
}

async function fetchDevices(opts={}){
  const requestId = ++fetchRequestId;
  const startMutation = mutationVersion;
  const startedAt = Date.now();
  const metaBase = { updated: new Date().toLocaleTimeString() };
  const nextDelaySuccess = opts.nextDelaySuccess ?? POLL_INTERVAL_NORMAL_MS;
  const nextDelayError = opts.nextDelayError ?? POLL_INTERVAL_ERROR_MS;

  try{
    const resp = await fetch(`?ajax=devices&t=${Date.now()}`, { cache:'no-store' });
    if(requestId !== fetchRequestId) return;
    if(resp.status===401){ location.reload(); return; }

    let payload=null;
    let parseFailed=false;
    try{ payload = await resp.json(); }
    catch(_){ parseFailed=true; }

    const meta={
      http: resp.status,
      api_latency: (payload && typeof payload.api_latency==='number') ? `${payload.api_latency} ms` : `${Math.round(Date.now()-startedAt)} ms`,
      updated: metaBase.updated
    };

    if(parseFailed || !payload || !Array.isArray(payload.devices)){
      const metaErr = Object.assign({}, meta, {
        http: parseFailed ? 'ERR' : meta.http,
        api_latency: parseFailed ? '--' : meta.api_latency
      });
      renderDevices(metaErr);
      schedulePoll(nextDelayError);
      return;
    }

    if(!opts.force && startMutation !== mutationVersion){
      renderDevices(meta);
      schedulePoll(POLL_INTERVAL_FAST_MS);
      return;
    }

    devicesCache = payload.devices;
    renderDevices(meta, {fromServer:true});
    schedulePoll(nextDelaySuccess);
  }catch(_){
    if(requestId !== fetchRequestId) return;
    const meta={
      http:'ERR',
      api_latency:'--',
      updated: metaBase.updated
    };
    renderDevices(meta);
    schedulePoll(nextDelayError);
  }
}

function touchMutation(){
  mutationVersion++;
  if(mutationVersion > 1e9){
    mutationVersion = 1;
  }
}

function unlockAudio(){
  if(audioUnlocked) return;
  const a=document.getElementById('siren');
  if(!a) return;
  const prevMuted=a.muted;
  const prevVol=a.volume;
  a.muted=false;
  a.volume=0.05; // brief quiet blip
  const p=a.play();
  const onSuccess=()=>{
    setTimeout(()=>{ try{ a.pause(); a.currentTime=0; a.volume=prevVol; a.muted=prevMuted; }catch(_){}; audioUnlocked=true; const b=document.getElementById('enableSoundBtn'); if(b) b.style.display='none'; }, 120);
  };
  if(p && p.then){
    p.then(onSuccess).catch(()=>{ /* still blocked */ });
  } else {
    try{ onSuccess(); }catch(_){ }
  }
}

function enableSound(){
  unlockAudio();
}

const autoUnlockEvents=['click','pointerdown','touchstart','keydown'];
function handleAutoUnlock(){
  unlockAudio();
  if(audioUnlocked){
    autoUnlockEvents.forEach(evt=>window.removeEventListener(evt, handleAutoUnlock));
  }
}
autoUnlockEvents.forEach(evt=>window.addEventListener(evt, handleAutoUnlock, false));

function renderDevices(meta, opts){
  const fromServer = !!(opts && opts.fromServer);
  if(meta){
    renderMeta = Object.assign({}, renderMeta, meta);
  }
  const devices = Array.isArray(devicesCache) ? devicesCache : [];
  const nowMs = Date.now();
  const nowSec = Math.floor(nowMs/1000);

  devices.forEach(dev=>{
    if(!dev || !dev.id) return;
    const pending = pendingSimOverrides.get(dev.id);
    if(!pending) return;
    if(nowMs > pending.expires){
      pendingSimOverrides.delete(dev.id);
      return;
    }
    if(pending.mode === 'simulate'){
      if(fromServer && (dev.simulate || dev.online === false)){
        pendingSimOverrides.delete(dev.id);
        return;
      }
      dev.simulate = true;
      dev.online = false;
      if(!dev.offline_since){
        dev.offline_since = pending.since ?? nowSec;
      }
      pending.expires = nowMs + SIM_OVERRIDE_TTL_MS;
    } else if(pending.mode === 'clearSim'){
      if(fromServer && !dev.simulate){
        pendingSimOverrides.delete(dev.id);
        return;
      }
      dev.simulate = false;
      dev.online = true;
      if(Object.prototype.hasOwnProperty.call(dev,'offline_since')){
        delete dev.offline_since;
      }
      pending.expires = nowMs + SIM_OVERRIDE_TTL_MS;
    }
  });

  const gateways = devices.filter(d=>d.gateway).sort((a,b)=>a.online-b.online||a.name.localeCompare(b.name));
  const aps = devices
    .filter(d=>!d.gateway && d.ap)
    .sort((a,b)=> (a.online - b.online) || a.name.localeCompare(b.name));
  const routersSwitches = devices
    .filter(d=>!d.gateway && !d.ap && (d.router || d.switch))
    .sort((a,b)=> (a.online - b.online) || a.name.localeCompare(b.name));

  renderGatewayGrid(gateways, nowSec);
  requestAnimationFrame(()=> {
    renderApGrid(aps, nowSec);
    requestAnimationFrame(()=> renderRouterSwitchGrid(routersSwitches, nowSec));
  });

  const footer=document.getElementById('footer');
  if(footer){
    const httpTxt = renderMeta.http ?? '--';
    const latTxt = renderMeta.api_latency ?? '--';
    const updatedTxt = renderMeta.updated ?? new Date().toLocaleTimeString();
    footer.innerText=`HTTP ${httpTxt}, API latency ${latTxt}, Updated ${updatedTxt}`;
  }

  const total=devices.length;
  const online=devices.filter(d=>d.online).length;
  const health = total>0 ? Math.round((online/total)*100) : null;
  const offlineGw=gateways.filter(d=>!d.online).length;
  const unacked=gateways.filter(d=>!d.online && !(d.ack_until && d.ack_until>nowSec)).length;
  const latVals=gateways.map(d=>{
    if(typeof d.latency==='number' && isFinite(d.latency)) return d.latency;
    if(typeof d.cpe_latency==='number' && isFinite(d.cpe_latency)) return d.cpe_latency;
    return null;
  }).filter(v=>v!==null);
  const avgLat = latVals.length ? Math.round(latVals.reduce((a,b)=>a+b,0)/latVals.length) : null;
  const highCpu=gateways.filter(d=>typeof d.cpu==='number' && d.cpu>90).length;
  const highRam=gateways.filter(d=>typeof d.ram==='number' && d.ram>90).length;

  const healthClass = health==null ? 'good' : (health>=95?'good':(health>=80?'warn':'bad'));
  const latClass = avgLat==null ? 'good' : (avgLat>500?'bad':(avgLat>100?'warn':'good'));
  const offlineClass = offlineGw>0 ? 'bad' : 'good';
  const unackedClass = unacked>0 ? 'bad' : 'good';
  const cpuClass = highCpu>0 ? 'bad' : 'good';
  const ramClass = highRam>0 ? 'bad' : 'good';

  const gwOnline = gateways.filter(d=>d.online).length;
  const gwTotal = gateways.length;
  const apItems = devices.filter(d=>d.ap && !d.gateway);
  const apOnline = apItems.filter(d=>d.online).length;
  const routerItems = devices.filter(d=>d.router && !d.gateway);
  const routerOnline = routerItems.filter(d=>d.online).length;
  const switchItems = devices.filter(d=>d.switch && !d.gateway);
  const switchOnline = switchItems.filter(d=>d.online).length;

  const summaryHTML = [
    `<span class="badge good">Gateways: ${gwOnline}/${gwTotal}</span>`,
    `<span class="badge good">APs: ${apOnline}/${apItems.length}</span>`,
    `<span class="badge good">Routers: ${routerOnline}/${routerItems.length}</span>`,
    `<span class="badge good">Switches: ${switchOnline}/${switchItems.length}</span>`,
    `<span class="badge ${healthClass}">Health: ${health==null?'--':health+'%'}</span>`,
    `<span class="badge ${offlineClass}">Gateways Offline: ${offlineGw}</span>`,
    `<span class="badge ${unackedClass}">Unacked Gateways: ${unacked}</span>`,
    `<span class="badge ${latClass}">Avg Latency: ${avgLat==null?'--':avgLat+' ms'}</span>`,
    `<span class="badge ${cpuClass}">High CPU: ${highCpu}</span>`,
    `<span class="badge ${ramClass}">High RAM: ${highRam}</span>`
  ].join(' ');
  const overallEl=document.getElementById('overallSummary');
  if(overallEl) overallEl.innerHTML=summaryHTML;

  const isApAlertEligible = (dev, nowSeconds) => {
    if(!dev || dev.online || !dev.ap) return false;
    const offlineSince = typeof dev.offline_since === 'number' ? dev.offline_since : null;
    if(!offlineSince || offlineSince < MIN_OFFLINE_TS) return false; // ignore missing/zero timestamps
    return offlineSince <= (nowSeconds - AP_ALERT_GRACE_SEC);
  };

  const alertCandidates = gateways.concat(
    apItems.filter(d=> isApAlertEligible(d, nowSec))
  );
  const shouldAlert = alertCandidates.some(d=>!d.online && !(d.ack_until && d.ack_until>nowSec));

  if(shouldAlert){
    if(!sirenShouldAlertPrev){
      clearTimeout(sirenTimeout); sirenTimeout=null;
      sirenNextDelayMs=30000;
    }
    if(!sirenTimeout){
      sirenTimeout=setTimeout(()=>{
            const nowSeconds = Math.floor(Date.now()/1000);
            const stillAlert = devicesCache
              .filter(d=>d && (d.gateway || d.ap))
              .some(d=>{
                if(d.online) return false;
                if(d.ack_until && d.ack_until > nowSeconds) return false;
                if(d.ap){
                  return isApAlertEligible(d, nowSeconds);
                }
                return true; // gateways alert immediately
              });
        if(stillAlert){
          const a=document.getElementById('siren');
          if(a){
            try{ a.pause(); a.currentTime=0; a.muted=false; a.volume=1; }catch(_){ }
            const pr=a.play();
            if(pr && pr.catch){ pr.catch(()=>{ const b=document.getElementById('enableSoundBtn'); if(b) b.style.display=''; }); }
          }
          sirenNextDelayMs=10*60*1000;
        }
        clearTimeout(sirenTimeout); sirenTimeout=null;
      }, sirenNextDelayMs);
    }
  } else {
    clearTimeout(sirenTimeout); sirenTimeout=null;
    sirenNextDelayMs=30000;
    const a=document.getElementById('siren');
    if(a){ a.pause(); a.currentTime=0; }
  }
  sirenShouldAlertPrev = shouldAlert;
}
function toggleAckMenu(id){
  const el=document.getElementById('ack-'+id);
  if(!el) return;
  const showing = (el.style.display==='none' || !el.style.display);
  el.style.display = showing ? 'block' : 'none';
  const card = el.closest('.card');
  if(card){ card.style.zIndex = showing ? '10000' : ''; }
}
function ack(id,dur){
  touchMutation();
  const seconds = ACK_DURATION_MAP[dur] ?? 1800;
  const nowSec = Date.now()/1000;
  const dev = devicesCache.find(x=>x.id===id);
  if(dev){
    dev.ack_until = nowSec + seconds;
    renderDevices();
  }
  fetch(`?ajax=ack&id=${id}&dur=${dur}&t=${Date.now()}`).then(()=>fetchDevices());
}
function clearAck(id){
  touchMutation();
  const dev = devicesCache.find(x=>x.id===id);
  if(dev && dev.ack_until){
    dev.ack_until=null;
    renderDevices();
  }
  fetch(`?ajax=clear&id=${id}&t=${Date.now()}`).then(()=>fetchDevices());
}
function simulate(id){
  touchMutation();
  unlockAudio();
  const nowMs = Date.now();
  const nowSec = Math.floor(nowMs/1000);
  pendingSimOverrides.set(id,{
    mode:'simulate',
    since: nowSec,
    expires: nowMs + SIM_OVERRIDE_TTL_MS
  });
  const dev=devicesCache.find(x=>x.id===id);
  if(dev){
    dev.simulate=true;
    dev.online=false;
    if(!dev.offline_since){
      dev.offline_since=nowSec;
    }
  }
  renderDevices();
  fetch(`?ajax=simulate&id=${id}&t=${Date.now()}`).then(()=>fetchDevices());
}
function clearSim(id){
  touchMutation();
  const expires = Date.now() + SIM_OVERRIDE_TTL_MS;
  pendingSimOverrides.set(id,{
    mode:'clearSim',
    expires
  });
  const dev=devicesCache.find(x=>x.id===id);
  if(dev){
    dev.simulate=false;
    dev.online=true;
    if(Object.prototype.hasOwnProperty.call(dev,'offline_since')){
      delete dev.offline_since;
    }
  }
  renderDevices();
  fetch(`?ajax=clearsim&id=${id}&t=${Date.now()}`).then(()=>fetchDevices());
}
function clearAll(){
  let changed=false;
  devicesCache.forEach(dev=>{
    if(dev && dev.ack_until){
      dev.ack_until=null;
      changed=true;
    }
  });
  if(changed){
    touchMutation();
    renderDevices();
  }
  fetch(`?ajax=clearall&t=${Date.now()}`).then(()=>fetchDevices());
}
function openTLS(){
  const m=document.getElementById('tlsModal'); if(!m)return; m.style.display='block';
  const s=document.getElementById('tlsStatus'); if(s){ s.textContent='Fetching current Caddy config...'; }
  fetch('?ajax=caddy_cfg').then(async r=>{
    if(r.status===401){ location.reload(); return; }
    const txt = await r.text();
    if(s){ s.textContent = txt; }
  }).catch(()=>{ if(s) s.textContent='Unable to reach Caddy admin API. Ensure the Caddy container is running.'; });
}
function closeTLS(){ const m=document.getElementById('tlsModal'); if(m) m.style.display='none'; }
function submitTLS(){
  const domain=document.getElementById('tlsDomain').value.trim();
  const gotify=document.getElementById('tlsGotify').value.trim();
  const email=document.getElementById('tlsEmail').value.trim();
  const staging=document.getElementById('tlsStaging').checked;
  const s=document.getElementById('tlsStatus'); if(s) s.textContent='Sending config to Caddy...';
  const fd=new FormData(); fd.append('domain',domain); fd.append('gotify_domain',gotify); fd.append('email',email); if(staging) fd.append('staging','1');
  fetch('?ajax=provision_tls',{method:'POST',body:fd}).then(r=>r.json()).then(j=>{
    if(j.ok){ if(s) s.textContent='Caddy loaded config. Visit https://'+domain+'/ in a minute to verify certs.'; }
    else{ if(s) s.textContent='Failed: '+(j.error||'unknown')+' code='+(j.code||'')+' err='+(j.err||'')+' resp='+(j.resp||''); }
  }).catch(()=>{ if(s) s.textContent='Request failed.'; });
  return false;
}
function showHistory(id,name){
  const params=new URLSearchParams({view:'device',id});
  if(name){ params.set('name',name); }
  window.location.href='?'+params.toString();
}
initDisplaySettings();
schedulePoll(0);

// Live counters (update once per second)
function fmtDurationFull(sec){
  if(typeof sec!=="number" || !isFinite(sec) || sec<0) return null;
  let s=Math.floor(sec);
  const d=Math.floor(s/86400); s%=86400;
  const h=Math.floor(s/3600); s%=3600;
  const m=Math.floor(s/60);   s%=60;
  const parts=[];
  if(d) parts.push(d+"d"); if(h||d) parts.push(h+"h"); if(m||h||d) parts.push(m+"m"); parts.push(s+"s");
  return parts.join(" ");
}
function tickLiveCounters(){
  const nowSec=Math.floor(Date.now()/1000);
  document.querySelectorAll('.card .live-uptime').forEach(el=>{
    const u = parseInt(el.getAttribute('data-uptime'),10);
    if(!isNaN(u) && u>0){
      const t = fmtDurationFull(u + (nowSec % 1000000));
      if(t) el.textContent = 'Uptime: ' + t;
      el.style.display='';
    } else { el.style.display='none'; }
  });
  document.querySelectorAll('.card .live-outage').forEach(el=>{
    const s = parseInt(el.getAttribute('data-offline-since'),10);
    if(!isNaN(s) && s>0){
      const dur = nowSec - s;
      const t = fmtDurationFull(dur);
      if(t) el.textContent = 'Outage: ' + t;
      el.style.display='';
    } else { el.style.display='none'; }
  });
}
setInterval(tickLiveCounters, 1000);

// Account helpers
function changePassword(){
  const current = prompt('Enter current password');
  if(current===null) return;
  const next = prompt('Enter new password (min 6 chars)');
  if(next===null) return;
  const fd = new FormData();
  fd.append('current', current);
  fd.append('new', next);
  fetch('?ajax=changepw', {method:'POST', body: fd}).then(async r=>{
    if(r.status===401){ location.reload(); return; }
    const j = await r.json().catch(()=>({ok:0,error:'bad_json'}));
    if(j.ok){ alert('Password updated. You will be logged out.'); logout(); }
    else { alert('Failed to update password: '+(j.error||'unknown')); }
  }).catch(()=>{});
}
function logout(){
  window.location.href='?action=logout';
}

function buildMetricBadges(device, latencyVal){
  const badges=[];
  if(isMetricEnabled("cpu")) badges.push(badgeVal(device.cpu, "CPU", "%"));
  if(isMetricEnabled("ram")) badges.push(badgeVal(device.ram, "RAM", "%"));
  if(isMetricEnabled("temp")) badges.push(badgeVal(device.temp, "Temp", "&deg;C"));
  if(isMetricEnabled("latency")) badges.push(badgeLatency(latencyVal));
  return badges.join(" ");
}

function buildLiveStateBadges(device){
  let html = "";
  if(isMetricEnabled("uptime")){
    html += `<span class="badge good live-uptime" data-uptime="${device.uptime??''}"></span>`;
  }
  if(isMetricEnabled("outage")){
    html += `<span class="badge bad live-outage" data-offline-since="${device.offline_since??''}"></span>`;
  }
  return html;
}
function renderGatewayGrid(gws, nowSec){
  const gateGrid=document.getElementById('gateGrid');
  if(!gateGrid) return;
  const html=gws.map(d=>{
    const badges = buildMetricBadges(d, d.latency);
    const ackActive = d.ack_until && d.ack_until > nowSec;
    return `<div class="card ${d.online?'':'offline'} ${ackActive?'acked':''}">
      <div class="ack-badge">${badgeAck(d.ack_until)}${buildLiveStateBadges(d)}</div>
      <h2>${d.name}</h2>
      <div class="role-label">Gateway</div>
      <div class="status" style="color:${d.online?'#b06cff':'#f55'}">${d.online?'ONLINE':'OFFLINE'}</div>
      <div>${badges}</div>
      <div class="actions">
        ${!d.online ? `
          <div class="dropdown" style="${ackActive ? 'display:none' : ''}">
            <button onclick="toggleAckMenu('${d.id}')">Ack</button>
            <div id="ack-${d.id}" class="dropdown-content" style="display:none;background:#333;position:absolute;">
              <a href="#" onclick="ack('${d.id}','30m')">30m</a>
              <a href="#" onclick="ack('${d.id}','1h')">1h</a>
              <a href="#" onclick="ack('${d.id}','6h')">6h</a>
              <a href="#" onclick="ack('${d.id}','8h')">8h</a>
              <a href="#" onclick="ack('${d.id}','12h')">12h</a>
            </div>
          </div>
          ${ackActive ? `<button onclick="clearAck('${d.id}')">Clear Ack</button>`:''}
        `:``}
        ${d.simulate ? `<button onclick="clearSim('${d.id}')">End Test</button>` : (d.online ? `<button onclick="simulate('${d.id}')">Test Outage</button>` : '')}
        <button onclick="showHistory('${d.id}','${d.name}')">History</button>
      </div>
    </div>`;
  }).join('');
  gateGrid.innerHTML = html;
}

function renderApGrid(items, nowSec){
  const apGrid=document.getElementById('apGrid');
  if(!apGrid) return;
  const html=items.map(d=>{
    const latencyVal = (typeof d.latency==='number' && isFinite(d.latency)) ? d.latency : d.cpe_latency;
    const badges = buildMetricBadges(d, latencyVal);
    const ackActive = d.ack_until && d.ack_until > nowSec;
    const actions = `
        ${!d.online ? `
          <div class="dropdown" style="${ackActive ? 'display:none' : ''}">
            <button onclick="toggleAckMenu('${d.id}')">Ack</button>
            <div id="ack-${d.id}" class="dropdown-content" style="display:none;background:#333;position:absolute;">
              <a href="#" onclick="ack('${d.id}','30m')">30m</a>
              <a href="#" onclick="ack('${d.id}','1h')">1h</a>
              <a href="#" onclick="ack('${d.id}','6h')">6h</a>
              <a href="#" onclick="ack('${d.id}','8h')">8h</a>
              <a href="#" onclick="ack('${d.id}','12h')">12h</a>
            </div>
          </div>
          ${ackActive ? `<button onclick="clearAck('${d.id}')">Clear Ack</button>`:''}
        `:``}
        ${d.simulate ? `<button onclick="clearSim('${d.id}')">End Test</button>` : (d.online ? `<button onclick="simulate('${d.id}')">Test Outage</button>` : '')}
        <button onclick="showHistory('${d.id}','${d.name}')">History</button>
    `;
    return `<div class="card ${d.online?'':'offline'} ${ackActive?'acked':''}">
      <div class="ack-badge">${badgeAck(d.ack_until)}${buildLiveStateBadges(d)}</div>
      <h2>${d.name}</h2>
      <div class="role-label">Access Point</div>
      <div class="status" style="color:${d.online?'#b06cff':'#f55'}">${d.online?'ONLINE':'OFFLINE'}</div>
      <div>${badges}</div>
      <div class="actions">
        ${actions}
      </div>
    </div>`;
  }).join('');
  apGrid.innerHTML = html;
}

function renderRouterSwitchGrid(backbones, nowSec){
  const routerGrid=document.getElementById('routerGrid');
  if(!routerGrid) return;
  const html=backbones.map(d=>{
    const latencyVal = (typeof d.latency==='number' && isFinite(d.latency)) ? d.latency : d.cpe_latency;
    const badges = buildMetricBadges(d, latencyVal);
    const ackActive = d.ack_until && d.ack_until > nowSec;
    const roleLabel = d.router ? 'Router' : 'Switch';
    return `<div class="card ${d.online?'':'offline'} ${ackActive?'acked':''}">
      <div class="ack-badge">${badgeAck(d.ack_until)}${buildLiveStateBadges(d)}</div>
      <h2>${d.name}</h2>
      <div class="role-label">${roleLabel}</div>
      <div class="status" style="color:${d.online?'#b06cff':'#f55'}">${d.online?'ONLINE':'OFFLINE'}</div>
      <div>${badges}</div>
      <div class="actions">
        ${!d.online ? `
          <div class="dropdown" style="${ackActive ? 'display:none' : ''}">
            <button onclick="toggleAckMenu('${d.id}')">Ack</button>
            <div id="ack-${d.id}" class="dropdown-content" style="display:none;background:#333;position:absolute;">
              <a href="#" onclick="ack('${d.id}','30m')">30m</a>
              <a href="#" onclick="ack('${d.id}','1h')">1h</a>
              <a href="#" onclick="ack('${d.id}','6h')">6h</a>
              <a href="#" onclick="ack('${d.id}','8h')">8h</a>
              <a href="#" onclick="ack('${d.id}','12h')">12h</a>
            </div>
          </div>
          ${ackActive ? `<button onclick="clearAck('${d.id}')">Clear Ack</button>`:''}
        `:``}
        ${d.simulate ? `<button onclick="clearSim('${d.id}')">End Test</button>` : (d.online ? `<button onclick="simulate('${d.id}')">Test Outage</button>` : '')}
        <button onclick="showHistory('${d.id}','${d.name}')">History</button>
      </div>
    </div>`;
  }).join('');
  routerGrid.innerHTML = html;
}

function ensureChartJs(){
  if(window.Chart) return Promise.resolve();
  if(chartJsLoader) return chartJsLoader;
  chartJsLoader = new Promise((resolve,reject)=>{
    const script=document.createElement('script');
    script.src='https://cdn.jsdelivr.net/npm/chart.js';
    script.async=true;
    script.onload=()=>resolve();
    script.onerror=()=>reject(new Error('chartjs-load-failed'));
    document.head.appendChild(script);
  });
  return chartJsLoader;
}

function getCpeHistoryChart(){
  if(cpeHistoryChart) return cpeHistoryChart;
  if(typeof Chart==='undefined') return null;
  const canvas=document.getElementById('cpeHistoryChart');
  if(!canvas) return null;
  cpeHistoryChart=new Chart(canvas,{
    type:'line',
    data:{
      labels:[],
      datasets:[{
        label:'Latency (ms)',
        data:[],
        borderColor:'#7acbff',
        backgroundColor:'rgba(122,203,255,0.15)',
        tension:0.25,
        spanGaps:true,
        fill:true,
        pointRadius:0
      }]
    },
    options:{
      responsive:true,
      maintainAspectRatio:false,
      animation:false,
      plugins:{
        legend:{ display:false },
        tooltip:{
          mode:'index',
          intersect:false,
          callbacks:{
            label(ctx){
              const latency = typeof ctx.parsed.y === 'number' ? ctx.parsed.y : null;
              return latency!=null ? `${ctx.label}: ${latency} ms` : `${ctx.label}: no response`;
            }
          }
        }
      },
      scales:{
        x:{
          ticks:{ color:'#bbb' },
          grid:{ color:'rgba(255,255,255,0.04)' }
        },
        y:{
          ticks:{ color:'#bbb' },
          grid:{ color:'rgba(255,255,255,0.04)' }
        }
      }
    }
  });
  return cpeHistoryChart;
}

function setCpeHistoryStatus(text){
  const el=document.getElementById('cpeHistoryStatus');
  if(!el) return;
  if(text){
    el.textContent=text;
    el.style.display='';
  } else {
    el.textContent='';
    el.style.display='none';
  }
}

function setCpeHistoryEmpty(show,message){
  const el=document.getElementById('cpeHistoryEmpty');
  if(!el) return;
  if(show){
    el.textContent=message||'';
    el.style.display='';
  } else {
    el.textContent='';
    el.style.display='none';
  }
}

function setCpeHistoryStats(stats){
  const el=document.getElementById('cpeHistoryStats');
  if(!el) return;
  if(!stats){
    el.innerHTML='';
    el.style.display='none';
    return;
  }
  const parts=[];
  if(typeof stats.count==='number'){
    parts.push(`<span>${stats.count} sample${stats.count===1?'':'s'}</span>`);
  }
  if(typeof stats.avg==='number'){
    parts.push(`<span>Avg ${stats.avg} ms</span>`);
  }
  if(typeof stats.min==='number'){
    parts.push(`<span>Min ${stats.min} ms</span>`);
  }
  if(typeof stats.max==='number'){
    parts.push(`<span>Max ${stats.max} ms</span>`);
  }
  if(typeof stats.devices==='number' && stats.devices>0){
    parts.push(`<span>${stats.devices} unique station${stats.devices===1?'':'s'}</span>`);
  }
  el.innerHTML = parts.join('');
  el.style.display = parts.length ? 'flex' : 'none';
}

function formatCpeHistoryLabel(tsMs){
  if(typeof tsMs!=='number' || !isFinite(tsMs) || tsMs<=0) return 'Unknown';
  const d=new Date(tsMs);
  try{
    return d.toLocaleString([], { month:'short', day:'numeric', hour:'2-digit', minute:'2-digit' });
  }catch(_){
    return d.toISOString().replace('T',' ').slice(0,16);
  }
}

function openCpeHistory(id,name){
  const modal=document.getElementById('cpeHistoryModal');
  if(!modal) return;
  const title=document.getElementById('cpeHistoryTitle');
  if(title){
    if(id){
      title.textContent = name || id;
    } else {
      title.textContent = 'All Station Ping History';
    }
  }
  const subtitle=document.getElementById('cpeHistorySubtitle');
  if(subtitle){
    subtitle.textContent = id ? 'Last 7 days of recorded ping latency' : 'All sampled station pings for the last 7 days';
  }
  modal.style.display='block';
  setCpeHistoryStatus(id ? 'Loading ping history...' : 'Loading all station pings...');
  setCpeHistoryEmpty(false,'');
  setCpeHistoryStats(null);
  const loadId=++cpeHistoryReqId;
  Promise.all([ensureChartJs(), fetchCpeHistoryData(id)])
    .then(([,payload])=>{
      if(loadId !== cpeHistoryReqId) return;
      applyCpeHistoryPayload(payload);
    })
    .catch(()=>{
      if(loadId !== cpeHistoryReqId) return;
      setCpeHistoryStatus('Unable to load ping history.');
      setCpeHistoryEmpty(false,'');
      setCpeHistoryStats(null);
    });
}

function fetchCpeHistoryData(id){
  const params=new URLSearchParams({ ajax:'cpe_history', t:Date.now() });
  if(id){ params.set('id', id); }
  return fetch(`?${params.toString()}`, { cache:'no-store' })
    .then(resp=>{
      if(resp.status===401){ location.reload(); throw new Error('unauthorized'); }
      if(!resp.ok) throw new Error('http_'+resp.status);
      return resp.json();
    });
}

function applyCpeHistoryPayload(payload){
  const isSingleDevice = !!(payload && payload.device_id);
  const points = (payload && Array.isArray(payload.points)) ? payload.points : [];
  const labels=[];
  const values=[];
  const goodVals=[];
  const deviceSet=new Set();
  points.forEach(pt=>{
    const tsMs = typeof pt.ts_ms === 'number' ? pt.ts_ms : null;
    if(pt.device_id){
      deviceSet.add(pt.device_id);
    }
    labels.push(formatCpeHistoryLabel(tsMs));
    if(typeof pt.latency === 'number' && isFinite(pt.latency)){
      const val = Math.round(pt.latency*10)/10;
      values.push(val);
      goodVals.push(val);
    } else {
      values.push(null);
    }
  });
  const chart=getCpeHistoryChart();
  if(chart){
    chart.data.labels = labels;
    chart.data.datasets[0].data = values;
    chart.update('none');
  }
  const uniqueDeviceCount = deviceSet.size;
  if(points.length===0){
    setCpeHistoryStatus('');
    setCpeHistoryEmpty(true, isSingleDevice ? 'No ping samples recorded for this station in the last 7 days.' : 'No ping samples recorded for any station in the last 7 days.');
    setCpeHistoryStats(null);
  } else {
    const deviceSuffix = (!isSingleDevice && uniqueDeviceCount > 0) ? ` across ${uniqueDeviceCount} station${uniqueDeviceCount===1?'':'s'}` : '';
    setCpeHistoryStatus(`Loaded ${points.length} sample${points.length===1?'':'s'}${deviceSuffix}.`);
    setCpeHistoryEmpty(false,'');
    const includeDevicesStat = !isSingleDevice && uniqueDeviceCount>0;
    if(goodVals.length){
      const min = Math.min(...goodVals);
      const max = Math.max(...goodVals);
      const avg = goodVals.reduce((sum,val)=>sum+val,0)/goodVals.length;
      setCpeHistoryStats({
        count: points.length,
        avg: Math.round(avg*10)/10,
        min: Math.round(min*10)/10,
        max: Math.round(max*10)/10,
        devices: includeDevicesStat ? uniqueDeviceCount : undefined
      });
    } else {
      setCpeHistoryStats({
        count: points.length,
        devices: includeDevicesStat ? uniqueDeviceCount : undefined
      });
    }
  }
}

function closeCpeHistory(){
  const modal=document.getElementById('cpeHistoryModal');
  if(modal){
    modal.style.display='none';
  }
  cpeHistoryReqId++;
}

document.addEventListener('keydown',ev=>{
  if(ev.key==='Escape'){
    const modal=document.getElementById('cpeHistoryModal');
    if(modal && modal.style.display==='block'){
      closeCpeHistory();
    }
  }
});



