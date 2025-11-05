(()=>{
  const detail = window.DEVICE_DETAIL || {};
  const deviceId = detail.id;
  const ackOptions = Array.isArray(detail.ackOptions) ? detail.ackOptions : [];

  const ackButtonsWrap = document.getElementById('ackButtons');
  const clearAckBtn     = document.getElementById('clearAckBtn');
  const statusBadge     = document.getElementById('detailStatusBadge');
  const ackBadge        = document.getElementById('detailAckBadge');
  const outageBadge     = document.getElementById('detailOutageBadge');
  const badgesEl        = document.getElementById('detailBadges');
  const updatedEl       = document.getElementById('detailUpdated');
  const subtitleEl      = document.getElementById('deviceSubtitle');
  const messageEl       = document.getElementById('detailMessage');
  const historyMsgEl    = document.getElementById('historyMessage');
  const titleEl         = document.getElementById('deviceTitle');
  const footerEl        = document.getElementById('detailFooter');

  if(!deviceId){
    if(messageEl) messageEl.textContent = 'Missing device identifier in request.';
    if(ackButtonsWrap) ackButtonsWrap.innerHTML = '';
    return;
  }

  const POLL_MS    = 1024;
  const HISTORY_MS = 1024;

  let deviceTimer  = null;
  let historyTimer = null;
  let deviceReqId  = 0;
  let historyReqId = 0;
  let currentDevice = null;
  let charts = null;
  let actionPending = false;
  let pendingNote = '';
  let detailMeta = { http: '--', api_latency: '--', updated: '--' };

  function updateFooter(){
    if(!footerEl) return;
    const httpTxt = detailMeta.http ?? '--';
    const latTxt  = detailMeta.api_latency ?? '--';
    const updTxt  = detailMeta.updated ?? '--';
    footerEl.textContent = `HTTP ${httpTxt}, API latency ${latTxt}, Updated ${updTxt}`;
  }
  function applyMeta(meta){
    if(!meta) return;
    detailMeta = {
      http: meta.http ?? detailMeta.http ?? '--',
      api_latency: meta.api_latency ?? detailMeta.api_latency ?? '--',
      updated: meta.updated ?? detailMeta.updated ?? '--'
    };
    updateFooter();
  }
  updateFooter();

  if(ackButtonsWrap){
    ackButtonsWrap.innerHTML = '';
    ackOptions.forEach(opt=>{
      const btn = document.createElement('button');
      btn.className = 'btn';
      btn.dataset.ack = opt;
      btn.textContent = `Ack ${opt}`;
      btn.addEventListener('click', ()=>handleAck(opt));
      ackButtonsWrap.appendChild(btn);
    });
  }
  if(clearAckBtn){
    clearAckBtn.addEventListener('click', handleClearAck);
  }

  function scheduleDevicePoll(delay=POLL_MS){
    clearTimeout(deviceTimer);
    deviceTimer = setTimeout(loadDevice, Math.max(delay, 0));
  }
  function scheduleHistoryPoll(delay=HISTORY_MS){
    clearTimeout(historyTimer);
    historyTimer = setTimeout(loadHistory, Math.max(delay, 0));
  }

  function badgeVal(v,label,suf){
    if(v==null || isNaN(v)) return '';
    let cls='good';
    if(v>90) cls='bad';
    else if(v>75) cls='warn';
    return `<span class="badge ${cls}">${label}: ${v}${suf}</span>`;
  }
  function badgeLatency(v){
    if(v==null || !isFinite(v)) return '';
    let cls='good';
    if(v>500) cls='bad';
    else if(v>100) cls='warn';
    return `<span class="badge ${cls}">Latency: ${v} ms</span>`;
  }
  function fmtRemain(sec){
    if(typeof sec!=='number' || !isFinite(sec) || sec<=0) return null;
    let s = Math.floor(sec);
    const d = Math.floor(s/86400); s%=86400;
    const h = Math.floor(s/3600); s%=3600;
    const m = Math.floor(s/60);   s%=60;
    const parts=[];
    if(d) parts.push(`${d}d`);
    if(h) parts.push(`${h}h`);
    if(m) parts.push(`${m}m`);
    parts.push(`${s}s`);
    return parts.join(' ');
  }
  function fmtDurationFull(sec){
    if(typeof sec!=='number' || !isFinite(sec) || sec<0) return null;
    let s = Math.floor(sec);
    const d = Math.floor(s/86400); s%=86400;
    const h = Math.floor(s/3600); s%=3600;
    const m = Math.floor(s/60);   s%=60;
    const parts=[];
    if(d) parts.push(`${d}d`);
    if(h||d) parts.push(`${h}h`);
    if(m||h||d) parts.push(`${m}m`);
    parts.push(`${s}s`);
    return parts.join(' ');
  }

  function setActionPending(flag, note=''){
    actionPending = !!flag;
    pendingNote = note || '';
    updateAckControls(currentDevice ? !currentDevice.online : false, !!(currentDevice && currentDevice.ack_until));
    if(messageEl && actionPending){
      messageEl.textContent = pendingNote;
    }
  }

  function renderDevice(dev, meta){
    if(meta) applyMeta(meta);
    currentDevice = dev;

    if(!dev){
      if(statusBadge){
        statusBadge.className = 'status-pill status-pill--loading';
        statusBadge.textContent = 'UNKNOWN';
      }
      if(ackBadge) ackBadge.style.display = 'none';
      if(outageBadge) outageBadge.style.display = 'none';
      if(badgesEl) badgesEl.innerHTML = '<span class="badge">No recent metrics</span>';
      if(updatedEl){
        const updatedLabel = detailMeta.updated ?? new Date().toLocaleTimeString();
        updatedEl.textContent = 'Last update: ' + updatedLabel;
      }
      updateAckControls(false,false);
      if(messageEl && !actionPending){
        messageEl.textContent = 'Device is not present in the latest poll.';
      }
      return;
    }

    const nowSec = Math.floor(Date.now()/1000);
    const offlineSince = dev.offline_since ? parseInt(dev.offline_since,10) : null;
    const ackUntil     = dev.ack_until ? parseInt(dev.ack_until,10) : null;
    const ackActive    = ackUntil && ackUntil > nowSec;
    const isOffline    = !dev.online;

    if(titleEl && dev.name && titleEl.dataset.locked!=='1'){
      titleEl.textContent = dev.name;
      document.title = `Device Detail - ${dev.name} | UISP NOC`;
    }

    if(subtitleEl){
      const bits=[];
      if(dev.role){ bits.push(dev.role.charAt(0).toUpperCase()+dev.role.slice(1)); }
      else if(dev.gateway) bits.push('Gateway');
      else if(dev.router) bits.push('Router');
      else if(dev.switch) bits.push('Switch');
      if(dev.id && dev.id!==dev.name) bits.push(`ID: ${dev.id}`);
      if(typeof dev.latency === 'number') bits.push(`Latency: ${dev.latency} ms`);
      subtitleEl.textContent = bits.join(' \u2022 ');
    }

    if(statusBadge){
      statusBadge.className = 'status-pill ' + (dev.online ? 'status-pill--online' : 'status-pill--offline');
      statusBadge.textContent = dev.online ? 'ONLINE' : 'OFFLINE';
    }
    if(ackBadge){
      if(ackActive){
        const remain = fmtRemain(ackUntil - nowSec);
        ackBadge.style.display = '';
        ackBadge.textContent = remain ? `ACK ${remain}` : 'ACK ACTIVE';
      } else {
        ackBadge.style.display = 'none';
      }
    }
    if(outageBadge){
      if(isOffline && offlineSince){
        const dur = fmtDurationFull(nowSec - offlineSince);
        outageBadge.style.display = '';
        outageBadge.textContent = dur ? `OUTAGE ${dur}` : 'OUTAGE';
      } else {
        outageBadge.style.display = 'none';
      }
    }

    if(badgesEl){
      const badges=[
        badgeVal(dev.cpu,'CPU','%'),
        badgeVal(dev.ram,'RAM','%'),
        badgeVal(dev.temp,'Temp','\u00B0C'),
        badgeLatency(dev.latency)
      ].filter(Boolean);
      badgesEl.innerHTML = badges.length ? badges.join(' ') : '<span class="badge">No recent metrics</span>';
    }

    if(updatedEl){
      const updatedLabel = detailMeta.updated ?? new Date().toLocaleTimeString();
      updatedEl.textContent = 'Last update: ' + updatedLabel;
    }

    updateAckControls(isOffline, ackActive);

    if(messageEl && !actionPending){
      if(ackActive && ackUntil){
        const until = new Date(ackUntil*1000).toLocaleTimeString();
        messageEl.textContent = `Ack active until ${until}.`;
      } else if(isOffline){
        messageEl.textContent = 'Device is offline. Acknowledge to silence sirens and notifications.';
      } else {
        messageEl.textContent = 'Device is online. Ack controls become available when the device is offline.';
      }
    }
  }

  function updateAckControls(isOffline, ackActive){
    if(ackButtonsWrap){
      const enableAck = isOffline && !ackActive && !actionPending;
      ackButtonsWrap.querySelectorAll('button[data-ack]').forEach(btn=>{
        btn.disabled = !enableAck;
      });
    }
    if(clearAckBtn){
      if(ackActive || actionPending){
        clearAckBtn.style.display = '';
        clearAckBtn.disabled = actionPending;
      } else {
        clearAckBtn.style.display = 'none';
      }
    }
  }

  function loadDevice(){
    clearTimeout(deviceTimer);
    const reqId = ++deviceReqId;
    return fetch(`?ajax=devices&t=${Date.now()}`)
      .then(r=>{
        if(!r.ok) throw new Error('HTTP '+r.status);
        return r.json();
      })
      .then(data=>{
        if(reqId !== deviceReqId) return;
        const meta = {
          http: data && data.http != null ? data.http : '--',
          api_latency: data && data.api_latency != null ? `${data.api_latency} ms` : '--',
          updated: new Date().toLocaleTimeString()
        };
        const devices = Array.isArray(data.devices) ? data.devices : [];
        const dev = devices.find(d=>d.id === deviceId);
        if(!dev){
          renderDevice(null, meta);
          return;
        }
        renderDevice(dev, meta);
      })
      .catch(err=>{
        if(reqId !== deviceReqId) return;
        console.error('Device detail fetch failed', err);
        applyMeta({http:'ERR', api_latency:'--', updated:new Date().toLocaleTimeString()});
        if(messageEl) messageEl.textContent = 'Unable to refresh device data.';
      })
      .finally(()=>{
        if(reqId === deviceReqId){
          scheduleDevicePoll(POLL_MS);
          if(actionPending){
            setActionPending(false);
          }
        }
      });
  }

  function ensureCharts(){
    if(charts) return charts;
    const baseOpts = {
      responsive: true,
      maintainAspectRatio: false,
      animation: false,
      scales: {
        x: {
          ticks: {
            color: '#e9e9ff',
            font: { size: 13, family: 'Segoe UI, sans-serif', weight: '600' }
          },
          grid: { color: 'rgba(255,255,255,0.08)' }
        },
        y: {
          ticks: {
            color: '#e9e9ff',
            font: { size: 13, family: 'Segoe UI, sans-serif', weight: '600' }
          },
          grid: { color: 'rgba(255,255,255,0.06)' }
        }
      },
      plugins: {
        legend: {
          labels: {
            color: '#fafaff',
            font: { size: 14, family: 'Segoe UI, sans-serif', weight: '600' }
          }
        },
        tooltip: {
          mode: 'index',
          intersect: false,
          padding: 12,
          backgroundColor: 'rgba(20,20,30,0.95)',
          titleColor: '#fafaff',
          bodyColor: '#d4d4ff',
          borderColor: 'rgba(120,120,220,0.6)',
          borderWidth: 1,
          titleFont: { size: 13, family: 'Segoe UI, sans-serif', weight: '600' },
          bodyFont: { size: 12, family: 'Segoe UI, sans-serif' }
        }
      }
    };
    charts = {
      cpu: new Chart(document.getElementById('cpuChart'),{
        type:'line',
        data:{ labels:[], datasets:[{label:'CPU %', data:[], borderColor:'#7dff96', backgroundColor:'rgba(125,255,150,0.1)', tension:0.25, spanGaps:true}]},
        options: baseOpts
      }),
      ram: new Chart(document.getElementById('ramChart'),{
        type:'line',
        data:{ labels:[], datasets:[{label:'RAM %', data:[], borderColor:'#ffe97a', backgroundColor:'rgba(255,233,122,0.1)', tension:0.25, spanGaps:true}]},
        options: baseOpts
      }),
      temp: new Chart(document.getElementById('tempChart'),{
        type:'line',
        data:{ labels:[], datasets:[{label:'Temp \u00B0C', data:[], borderColor:'#ff7a7a', backgroundColor:'rgba(255,122,122,0.1)', tension:0.25, spanGaps:true}]},
        options: baseOpts
      }),
      lat: new Chart(document.getElementById('latChart'),{
        type:'line',
        data:{ labels:[], datasets:[{label:'Latency ms', data:[], borderColor:'#7acbff', backgroundColor:'rgba(122,203,255,0.1)', tension:0.25, spanGaps:true}]},
        options: baseOpts
      })
    };
    return charts;
  }

  function loadHistory(){
    clearTimeout(historyTimer);
    const reqId = ++historyReqId;
    return fetch(`?ajax=history&id=${encodeURIComponent(deviceId)}&t=${Date.now()}`)
      .then(r=>{
        if(!r.ok) throw new Error('HTTP '+r.status);
        return r.json();
      })
      .then(rows=>{
        if(reqId !== historyReqId) return;
        const chartSet = ensureCharts();
        const labels = Array.isArray(rows) ? rows.map(r=>r.timestamp) : [];
        chartSet.cpu.data.labels  = labels;
        chartSet.ram.data.labels  = labels;
        chartSet.temp.data.labels = labels;
        chartSet.lat.data.labels  = labels;

        chartSet.cpu.data.datasets[0].data  = labels.map((_,i)=>rows[i]?.cpu ?? null);
        chartSet.ram.data.datasets[0].data  = labels.map((_,i)=>rows[i]?.ram ?? null);
        chartSet.temp.data.datasets[0].data = labels.map((_,i)=>rows[i]?.temp ?? null);
        chartSet.lat.data.datasets[0].data  = labels.map((_,i)=>rows[i]?.latency ?? null);

        chartSet.cpu.update('none');
        chartSet.ram.update('none');
        chartSet.temp.update('none');
        chartSet.lat.update('none');

        if(historyMsgEl){
          historyMsgEl.textContent = labels.length ? '' : 'No historical metrics recorded yet for this device.';
        }
      })
      .catch(err=>{
        if(reqId !== historyReqId) return;
        console.error('History fetch failed', err);
        if(historyMsgEl) historyMsgEl.textContent = 'Unable to load history data.';
      })
      .finally(()=>{
        if(reqId === historyReqId){
          scheduleHistoryPoll(HISTORY_MS);
        }
      });
  }

  function handleAck(dur){
    if(!dur || actionPending) return;
    setActionPending(true, `Sending ${dur} acknowledgement...`);
    fetch(`?ajax=ack&id=${encodeURIComponent(deviceId)}&dur=${encodeURIComponent(dur)}&t=${Date.now()}`)
      .then(()=>loadDevice())
      .catch(err=>{
        console.error('Ack failed', err);
        if(messageEl) messageEl.textContent = 'Failed to acknowledge outage.';
      })
      .finally(()=>{ setActionPending(false); });
  }

  function handleClearAck(){
    if(actionPending) return;
    setActionPending(true, 'Clearing acknowledgement...');
    fetch(`?ajax=clear&id=${encodeURIComponent(deviceId)}&t=${Date.now()}`)
      .then(()=>loadDevice())
      .catch(err=>{
        console.error('Clear ack failed', err);
        if(messageEl) messageEl.textContent = 'Failed to clear acknowledgement.';
      })
      .finally(()=>{ setActionPending(false); });
  }

  loadDevice();
  loadHistory();
})(); 
