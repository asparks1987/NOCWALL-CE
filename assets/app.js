function openTab(id){
  document.querySelectorAll('.tabcontent').forEach(x=>x.style.display='none');
  document.querySelectorAll('.tablink').forEach(x=>x.classList.remove('active'));
  document.getElementById(id).style.display='block';
  event.target.classList.add('active');
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

function fetchDevices(){
 fetch('?ajax=devices').then(r=>{ if(r.status===401){ location.reload(); throw new Error('unauthorized'); } return r.json(); }).then(j=>{
  devicesCache=j.devices;
  const gws=j.devices.filter(d=>d.gateway).sort((a,b)=>a.online-b.online||a.name.localeCompare(b.name));
  // Sort CPEs: offline first, then by name
  const cps=j.devices
    .filter(d=>!d.gateway)
    .sort((a,b)=> (a.online - b.online) || a.name.localeCompare(b.name));
  // Backbone devices: routers and switches (excluding gateways already in gws)
  const backbones = j.devices
    .filter(d=> (d.router || d.switch) && !d.gateway )
    .sort((a,b)=> (a.online - b.online) || a.name.localeCompare(b.name));

  // Gateways
  const gwHTML=gws.map(d=>{
    const badges=[badgeVal(d.cpu,'CPU','%'),badgeVal(d.ram,'RAM','%'),badgeVal(d.temp,'Temp','°C'),badgeLatency(d.latency)].join(' ');
    const ackActive = d.ack_until && d.ack_until > (Date.now()/1000);
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
        
        <button onclick="showHistory('${d.id}','${d.name}')">History</button>
      </div>
    </div>`;
  }).join('');
  document.getElementById('gateGrid').innerHTML=gwHTML;

  // CPEs (show latency badge at top-right if recently pinged)
  const cpeHTML=cps.map(d=>{
    const latBadge = (typeof d.cpe_latency==='number') ? badgeLatency(d.cpe_latency) : '';
    return `<div class="card ${d.online?'':'offline'}">
      <div class="cpe-badge">${latBadge}</div>
      <h2>${d.name}</h2>
      <div style="color:${d.online?'#b06cff':'#f55'}">${d.online?'ONLINE':'OFFLINE'}</div>
    </div>`;
  }).join('');
  document.getElementById('cpeGrid').innerHTML=cpeHTML;

  // Routers & Switches grid
  const rbHTML = backbones.map(d=>{
    const badges=[badgeVal(d.cpu,'CPU','%'),badgeVal(d.ram,'RAM','%'),badgeVal(d.temp,'Temp','°C'),badgeLatency(d.latency)].join(' ');
    const ackActive = d.ack_until && d.ack_until > (Date.now()/1000);
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
        <button onclick="showHistory('${d.id}','${d.name}')">History</button>
      </div>
    </div>`;
  }).join('');
  const routerGrid = document.getElementById('routerGrid'); if(routerGrid) routerGrid.innerHTML = rbHTML;

  document.getElementById('footer').innerText=`HTTP ${j.http}, API latency ${j.api_latency} ms, Updated ${new Date().toLocaleTimeString()}`;

  // Overall header summary
  const nowSec=Math.floor(Date.now()/1000);
  const total=j.devices.length;
  const online=j.devices.filter(d=>d.online).length;
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
    `<span class="badge good">Gateways: ${gwOnline}/${gwTotal}</span>`,
    `<span class="badge good">Routers/Switches: ${rbOnline}/${rbTotal}</span>`,
    `<span class="badge good">CPEs: ${cpeOnline}/${cpeTotal}</span>`,
    `<span class="badge ${healthClass}">Health: ${health==null?'--':health+'%'}</span>`,
    `<span class="badge ${offlineClass}">GW Offline: ${offlineGw}</span>`,
    `<span class="badge ${unackedClass}">Unacked: ${unacked}</span>`,
    `<span class="badge ${latClass}">Avg Latency: ${avgLat==null?'--':avgLat+' ms'}</span>`,
    `<span class="badge ${cpuClass}">High CPU: ${highCpu}</span>`,
    `<span class="badge ${ramClass}">High RAM: ${highRam}</span>`
  ].join(' ');
  const overallEl=document.getElementById('overallSummary');
  if(overallEl) overallEl.innerHTML=summaryHTML;

  // Siren logic: only for UNACKED offline gateways (routers/switches do NOT trigger sound)
  const shouldAlert = gws.some(d=>!d.online && !(d.ack_until && d.ack_until>nowSec));

  if(shouldAlert){
    // Reset to initial 30s delay when entering alert state
    if(!sirenShouldAlertPrev){
      clearTimeout(sirenTimeout); sirenTimeout=null;
      sirenNextDelayMs=30000; // initial grace period
    }
    if(!sirenTimeout){
      sirenTimeout=setTimeout(()=>{
        // Re-evaluate at fire time using freshest cache
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
          // After first fire, repeat every 10 minutes while unacked
          sirenNextDelayMs=10*60*1000;
        }
        // Clear timer and allow next scheduling on next poll
        clearTimeout(sirenTimeout); sirenTimeout=null;
      }, sirenNextDelayMs);
    }
  } else {
    // No unacked offline => cancel and silence
    clearTimeout(sirenTimeout); sirenTimeout=null;
    sirenNextDelayMs=30000;
    const a=document.getElementById('siren');
    if(a){ a.pause(); a.currentTime=0; }
  }
  sirenShouldAlertPrev = shouldAlert;
 });
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
  // Optimistic UI
  let dev=devicesCache.find(x=>x.id===id);
  if(dev){dev.ack_until=Date.now()/1000+1800;}
  fetch(`?ajax=ack&id=${id}&dur=${dur}&t=${Date.now()}`).then(()=>fetchDevices());
}
function clearAck(id){
  let dev=devicesCache.find(x=>x.id===id);
  if(dev){dev.ack_until=null;}
  fetch(`?ajax=clear&id=${id}&t=${Date.now()}`).then(()=>fetchDevices());
}
function simulate(id){
  // Try to unlock audio on explicit user action
  unlockAudio();
  let dev=devicesCache.find(x=>x.id===id);
  if(dev){dev.simulate=true;}
  fetch(`?ajax=simulate&id=${id}&t=${Date.now()}`).then(()=>fetchDevices());
}
function clearSim(id){
  let dev=devicesCache.find(x=>x.id===id);
  if(dev){dev.simulate=false;}
  fetch(`?ajax=clearsim&id=${id}&t=${Date.now()}`).then(()=>fetchDevices());
}
function clearAll(){ fetch(`?ajax=clearall&t=${Date.now()}`).then(()=>fetchDevices()); }
function closeModal(){document.getElementById('historyModal').style.display='none';}
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
 fetch(`?ajax=history&id=${id}`).then(r=>r.json()).then(rows=>{
   document.getElementById('histTitle').innerText=`History for ${name}`;
   const labels=rows.map(r=>r.timestamp);
   const cpu=rows.map(r=>r.cpu);
   const ram=rows.map(r=>r.ram);
   const temp=rows.map(r=>r.temp);
   const lat=rows.map(r=>r.latency);

   new Chart(document.getElementById('cpuChart'),{type:'line',data:{labels,datasets:[{label:'CPU %',data:cpu,borderColor:'lime'}]}});
   new Chart(document.getElementById('ramChart'),{type:'line',data:{labels,datasets:[{label:'RAM %',data:ram,borderColor:'yellow'}]}});
   new Chart(document.getElementById('tempChart'),{type:'line',data:{labels,datasets:[{label:'Temp °C',data:temp,borderColor:'red'}]}});
   new Chart(document.getElementById('latChart'),{type:'line',data:{labels,datasets:[{label:'Latency ms',data:lat,borderColor:'cyan'}]}});
   const modal=document.getElementById('historyModal');
   modal.style.display='block';
 });
}
setInterval(fetchDevices,10000);
fetchDevices();

// Best-effort unlock on first user click anywhere
window.addEventListener('click',unlockAudio,{once:true});

// Close history modal when clicking the overlay or pressing Escape
(function(){
  const modal=document.getElementById('historyModal');
  if(modal){
    modal.addEventListener('click',(e)=>{ if(e.target===modal) closeModal(); });
  }
  window.addEventListener('keydown',(e)=>{ if(e.key==='Escape') closeModal(); });
})();

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




