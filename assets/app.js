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
let renderQueue=[];
let renderQueueActive=false;
let renderQueueGen=0;

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

function resetRenderQueue(){
  renderQueueGen++;
  renderQueue.length=0;
  renderQueueActive=false;
}
function scheduleSectionRender(fn){
  const gen = renderQueueGen;
  renderQueue.push({fn, gen});
  if(!renderQueueActive){
    renderQueueActive=true;
    requestAnimationFrame(processRenderQueue);
  }
}
function processRenderQueue(){
  if(!renderQueue.length){
    renderQueueActive=false;
    return;
  }
  const task = renderQueue.shift();
  if(task && task.gen === renderQueueGen){
    try{ task.fn(); }catch(err){ console.error('Render section failed', err); }
  }
  if(renderQueue.length){
    requestAnimationFrame(processRenderQueue);
  } else {
    renderQueueActive=false;
  }
}

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

  const gws = devices
    .filter(d=>d.gateway)
    .sort((a,b)=>a.online-b.online||a.name.localeCompare(b.name));
  const cps = devices
    .filter(d=>!d.gateway)
    .sort((a,b)=> (a.online - b.online) || a.name.localeCompare(b.name));
  const backbones = devices
    .filter(d=> (d.router || d.switch) && !d.gateway )
    .sort((a,b)=> (a.online - b.online) || a.name.localeCompare(b.name));

  resetRenderQueue();
  scheduleSectionRender(()=>renderGatewayGrid(gws, nowSec));
  scheduleSectionRender(()=>renderBackboneGrid(backbones, nowSec));
  scheduleSectionRender(()=>renderCpeGrid(cps));

  const footer=document.getElementById('footer');
  if(footer){
    const httpTxt = renderMeta.http ?? '--';
    const latTxt = renderMeta.api_latency ?? '--';
    const updatedTxt = renderMeta.updated ?? new Date().toLocaleTimeString();
    footer.innerText=HTTP , API latency , Updated ;
  }

  const total=devices.length;
  const online=devices.filter(d=>d.online).length;
  const health = total>0 ? Math.round((online/total)*100) : null;
  const offlineGw=gws.filter(d=>!d.online).length;
  const unacked=gws.filter(d=>!d.online && !(d.ack_until && d.ack_until>nowSec)).length;
  const latVals=gws.map(d=>d.latency).filter(v=>typeof v==='number' && isFinite(v));
  const avgLat = latVals.length ? Math.round(latVals.reduce((a,b)=>a+b,0)/latVals.length) : null;
  const highCpu=gws.filter(d=>typeof d.cpu==='number' && d.cpu>90).length;
  const highRam=gws.filter(d=>typeof d.ram==='number' && d.ram>90).length;

  const healthClass = health==null ? 'good' : (health>=95?'good':(health>=80?'warn':'bad'));
  const latClass = avgLat==null ? 'good' : (avgLat>500?'bad':(avgLat>100?'warn':'good'));
  const offlineClass = offlineGw>0 ? 'bad' : 'good';
  const unackedClass = unacked>0 ? 'bad' : 'good';
  const cpuClass = highCpu>0 ? 'bad' : 'good';
  const ramClass = highRam>0 ? 'bad' : 'good';

  const gwOnline = gws.filter(d=>d.online).length;
  const gwTotal = gws.length;
  const cpeOnline = cps.filter(d=>d.online).length;
  const cpeTotal = cps.length;
  const rbOnline = backbones.filter(d=>d.online).length;
  const rbTotal = backbones.length;

  const summaryHTML = [
    <span class="badge good">Gateways: /</span>,
    <span class="badge good">Routers/Switches: /</span>,
    <span class="badge good">CPEs: /</span>,
    <span class="badge ">Health: </span>,
    <span class="badge ">GW Offline: </span>,
    <span class="badge ">Unacked: </span>,
    <span class="badge ">Avg Latency: </span>,
    <span class="badge ">High CPU: </span>,
    <span class="badge ">High RAM: </span>
  ].join(' ');
  const overallEl=document.getElementById('overallSummary');
  if(overallEl) overallEl.innerHTML=summaryHTML;

  const shouldAlert = gws.some(d=>!d.online && !(d.ack_until && d.ack_until>nowSec));

  if(shouldAlert){
    if(!sirenShouldAlertPrev){
      clearTimeout(sirenTimeout); sirenTimeout=null;
      sirenNextDelayMs=30000;
    }
    if(!sirenTimeout){
      sirenTimeout=setTimeout(()=>{
        const stillAlert = devicesCache
          .filter(d=>d.gateway)
          .some(d=>!d.online && !(d.ack_until && d.ack_until>(Date.now()/1000)));
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

function renderGatewayGrid(gws, nowSec){
  const gateGrid=document.getElementById('gateGrid');
  if(!gateGrid) return;
  const html=gws.map(d=>{
    const badges=[badgeVal(d.cpu,'CPU','%'),badgeVal(d.ram,'RAM','%'),badgeVal(d.temp,'Temp','&deg;C'),badgeLatency(d.latency)].join(' ');
    const ackActive = d.ack_until && d.ack_until > nowSec;
    return `<div class="card ${d.online?'':'offline'} ${ackActive?'acked':''}">
      <div class="ack-badge">${badgeAck(d.ack_until)}
        <span class="badge good live-uptime" data-uptime="${d.uptime??''}"></span>
        <span class="badge bad live-outage" data-offline-since="${d.offline_since??''}"></span>
      </div>
      <h2>${d.name}</h2>
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

function renderBackboneGrid(backbones, nowSec){
  const routerGrid=document.getElementById('routerGrid');
  if(!routerGrid) return;
  const html=backbones.map(d=>{
    const badges=[badgeVal(d.cpu,'CPU','%'),badgeVal(d.ram,'RAM','%'),badgeVal(d.temp,'Temp','&deg;C'),badgeLatency(d.latency)].join(' ');
    const ackActive = d.ack_until && d.ack_until > nowSec;
    return `<div class="card ${d.online?'':'offline'} ${ackActive?'acked':''}">
      <div class="ack-badge">${badgeAck(d.ack_until)}
        <span class="badge good live-uptime" data-uptime="${d.uptime??''}"></span>
        <span class="badge bad live-outage" data-offline-since="${d.offline_since??''}"></span>
      </div>
      <h2>${d.name}</h2>
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

function renderCpeGrid(cps){
  const cpeGrid=document.getElementById('cpeGrid');
  if(!cpeGrid) return;
  const html=cps.map(d=>{
    const latBadge = (typeof d.cpe_latency==='number') ? badgeLatency(d.cpe_latency) : '';
    return `<div class="card ${d.online?'':'offline'}">
      <div class="cpe-badge">${latBadge}</div>
      <h2>${d.name}</h2>
      <div style="color:${d.online?'#b06cff':'#f55'}">${d.online?'ONLINE':'OFFLINE'}</div>
    </div>`;
  }).join('');
  cpeGrid.innerHTML = html;
}
