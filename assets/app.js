const KNOWN_TABS = ["gateways", "aps", "routers", "topology"];
function normalizeTabId(id){
  const raw = String(id || "").toLowerCase().trim();
  if(raw === "topology") return featureEnabled("topology") ? "topology" : "gateways";
  if(KNOWN_TABS.includes(raw)) return raw;
  return "gateways";
}
function setActiveTab(id, opts){
  const target = normalizeTabId(id);
  document.querySelectorAll('.tabcontent').forEach(x=>x.style.display='none');
  document.querySelectorAll('.tablink').forEach(x=>x.classList.remove('active'));
  const panel = document.getElementById(target);
  if(panel){
    panel.style.display = 'block';
  }
  let activated = false;
  try{
    if(opts && opts.event && opts.event.target){
      opts.event.target.classList.add('active');
      activated = true;
    }
  }catch(_){ }
  if(!activated){
    const btn = document.querySelector(`.tablink[data-tab="${target}"]`);
    if(btn) btn.classList.add('active');
  }
  if(target === "topology"){
    loadTopology(false);
  }
  if(opts && opts.persist && dashSettings && dashSettings.default_tab !== target){
    dashSettings.default_tab = target;
    saveDashSettings();
  }
  return target;
}
function openTab(id, ev){
  setActiveTab(id, { event: ev, persist: true });
}

let kioskMode = false;

function isTypingContext(target){
  const el = target || document.activeElement;
  if(!el || !el.tagName) return false;
  const tag = String(el.tagName).toLowerCase();
  if(tag === "input" || tag === "textarea" || tag === "select") return true;
  if(el.isContentEditable) return true;
  return false;
}

function setKioskMode(enabled){
  kioskMode = !!enabled;
  if(document.body){
    document.body.classList.toggle("kiosk-mode", kioskMode);
  }
}

function toggleKioskMode(){
  setKioskMode(!kioskMode);
}

function initKioskModeFromUrl(){
  try{
    const params = new URLSearchParams(window.location.search || "");
    const raw = String(params.get("kiosk") || "").trim().toLowerCase();
    if(raw === "1" || raw === "true" || raw === "yes" || raw === "on"){
      setKioskMode(true);
    }
  }catch(_){ }
}

function openShortcuts(){
  const modal = document.getElementById("shortcutsModal");
  if(!modal) return;
  modal.style.display = "block";
}

function closeShortcuts(){
  const modal = document.getElementById("shortcutsModal");
  if(!modal) return;
  modal.style.display = "none";
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
const POLL_INTERVAL_SLOW_MS = 10000;
const POLL_INTERVAL_ERROR_BASE_MS = 15000;
const POLL_INTERVAL_ERROR_MAX_MS = 120000;
const DATA_STALE_MIN_MS = 60000;
const DEVICE_CHANGE_HIGHLIGHT_MS = 120000;
const SOURCE_STATUS_REFRESH_MS = 60000;
const AP_SIREN_PREFS_KEY = "nocwall.ap.siren.v1";
const TAB_SIREN_PREFS_KEY = "nocwall.tab.siren.v1";
const CARD_ORDER_PREFS_KEY = "nocwall.card.order.v1";
const DEVICE_SNAPSHOT_KEY = "nocwall.devices.snapshot.v1";
let userPrefsLoaded=false;
let userPrefsSaveTimer=null;
let pollErrorCount=0;
let pollLastErrorMessage="";
let pollLastSuccessAtMs=0;
let pollLastFailureAtMs=0;
let pollNextRetryAtMs=0;
let pollBannerTickTimer=null;
const deviceStatusMemory = new Map();
let usingSnapshotFallback=false;
let snapshotFallbackSavedAtMs=0;
const DEFAULT_TAB_SIREN_PREFS = {
  gateways: true,
  aps: false,
  routers: false
};
const DEFAULT_CARD_ORDER_PREFS = {
  gateways: [],
  aps: [],
  routers: []
};
const INVENTORY_OVERVIEW_REFRESH_MS = 60000;
const FEATURE_FLAGS = (() => {
  const raw = (typeof window !== "undefined" && window.NOCWALL_FEATURES && typeof window.NOCWALL_FEATURES === "object")
    ? window.NOCWALL_FEATURES
    : {};
  const profile = String(raw.profile || "ce").toLowerCase();
  return {
    profile,
    strict_ce: !!raw.strict_ce,
    pro_features: !!raw.pro_features,
    advanced_metrics: !!raw.advanced_metrics,
    advanced_actions: !!raw.advanced_actions,
    display_controls: !!raw.display_controls,
    topology: !!raw.topology,
    inventory: !!raw.inventory,
    history: !!raw.history,
    ack: !!raw.ack,
    simulate: !!raw.simulate,
    cpe_history: !!raw.cpe_history
  };
})();
function featureEnabled(name){
  return !!FEATURE_FLAGS[name];
}

const DASH_SETTINGS_KEY = "nocwall.dashboard.settings.v1";
const DEFAULT_DASH_SETTINGS = {
  density: "normal",
  default_tab: "gateways",
  sort_mode: "manual",
  group_mode: "none",
  refresh_interval: "normal",
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
let apSirenPrefs = loadApSirenPrefs();
let tabSirenPrefs = loadTabSirenPrefs();
let cardOrderPrefs = loadCardOrderPrefs();
let dragState = null;
let inventoryOverviewByDevice = {};
let inventoryOverviewFetchedAt = 0;
let inventoryOverviewLoading = false;
let inventoryOverviewError = "";
let sourceStatusItems = [];
let sourceStatusSummary = null;
let sourceStatusLoading = false;
let sourceStatusLastLoadedAt = 0;
let topologyCache = { nodes: [], edges: [], health: null, fetched_at: "" };
let topologyLoading = false;
let topologyLastFetchMs = 0;
let topologyTrace = null;
const TOPOLOGY_REFRESH_MS = 30000;
let deviceSearchQuery = "";
let deviceQuickFilter = "all";

function normalizeApSirenPrefs(input){
  const out={};
  if(!input || typeof input!=="object" || Array.isArray(input)) return out;
  Object.keys(input).forEach(key=>{
    if(!key) return;
    out[key] = !!input[key];
  });
  return out;
}

function loadApSirenPrefs(){
  try{
    const raw = localStorage.getItem(AP_SIREN_PREFS_KEY);
    if(!raw) return {};
    const parsed = JSON.parse(raw);
    return normalizeApSirenPrefs(parsed);
  }catch(_){
    return {};
  }
}

function normalizeTabSirenPrefs(input){
  const normalized = Object.assign({}, DEFAULT_TAB_SIREN_PREFS);
  if(!input || typeof input!=="object" || Array.isArray(input)) return normalized;
  Object.keys(normalized).forEach(key=>{
    if(Object.prototype.hasOwnProperty.call(input,key)){
      normalized[key] = !!input[key];
    }
  });
  return normalized;
}

function loadTabSirenPrefs(){
  try{
    const raw = localStorage.getItem(TAB_SIREN_PREFS_KEY);
    if(!raw) return Object.assign({}, DEFAULT_TAB_SIREN_PREFS);
    const parsed = JSON.parse(raw);
    return normalizeTabSirenPrefs(parsed);
  }catch(_){
    return Object.assign({}, DEFAULT_TAB_SIREN_PREFS);
  }
}

function normalizeCardOrderPrefs(input){
  const normalized = {
    gateways: [],
    aps: [],
    routers: []
  };
  if(!input || typeof input!=="object" || Array.isArray(input)) return normalized;
  Object.keys(normalized).forEach(tabKey=>{
    const raw = input[tabKey];
    if(!Array.isArray(raw)) return;
    const seen = new Set();
    normalized[tabKey] = raw
      .map(v=>String(v || '').trim())
      .filter(v=>{
        if(!v || v.length > 180) return false;
        if(seen.has(v)) return false;
        seen.add(v);
        return true;
      })
      .slice(0,500);
  });
  return normalized;
}

function loadCardOrderPrefs(){
  try{
    const raw = localStorage.getItem(CARD_ORDER_PREFS_KEY);
    if(!raw) return normalizeCardOrderPrefs(DEFAULT_CARD_ORDER_PREFS);
    const parsed = JSON.parse(raw);
    return normalizeCardOrderPrefs(parsed);
  }catch(_){
    return normalizeCardOrderPrefs(DEFAULT_CARD_ORDER_PREFS);
  }
}

function saveDeviceSnapshot(devices){
  try{
    const list = Array.isArray(devices) ? devices : [];
    const payload = {
      saved_at: Date.now(),
      devices: list
    };
    localStorage.setItem(DEVICE_SNAPSHOT_KEY, JSON.stringify(payload));
  }catch(_){ }
}

function loadDeviceSnapshot(){
  try{
    const raw = localStorage.getItem(DEVICE_SNAPSHOT_KEY);
    if(!raw) return null;
    const parsed = JSON.parse(raw);
    if(!parsed || typeof parsed !== "object" || !Array.isArray(parsed.devices)) return null;
    const savedAt = Number(parsed.saved_at || 0);
    return {
      savedAtMs: Number.isFinite(savedAt) && savedAt > 0 ? savedAt : 0,
      devices: parsed.devices
    };
  }catch(_){
    return null;
  }
}

function saveApSirenPrefs(){
  try{
    localStorage.setItem(AP_SIREN_PREFS_KEY, JSON.stringify(apSirenPrefs));
  }catch(_){ }
  scheduleUserPrefsSave();
}

function saveTabSirenPrefs(){
  try{
    localStorage.setItem(TAB_SIREN_PREFS_KEY, JSON.stringify(tabSirenPrefs));
  }catch(_){ }
  scheduleUserPrefsSave();
}

function saveCardOrderPrefs(){
  try{
    localStorage.setItem(CARD_ORDER_PREFS_KEY, JSON.stringify(cardOrderPrefs));
  }catch(_){ }
  scheduleUserPrefsSave();
}

function isDeviceSirenEnabledById(id){
  if(!id) return true;
  return apSirenPrefs[id] !== false;
}

function toggleDeviceSiren(id){
  if(!id) return;
  if(isDeviceSirenEnabledById(id)){
    apSirenPrefs[id] = false;
  } else {
    delete apSirenPrefs[id];
  }
  saveApSirenPrefs();
  renderDevices();
}

// Backward-compatible alias (older handlers referenced AP naming).
function isApSirenEnabledById(id){
  return isDeviceSirenEnabledById(id);
}
function toggleApSiren(id){
  toggleDeviceSiren(id);
}

function isTabSirenEnabled(tabKey){
  return !!(tabSirenPrefs && tabSirenPrefs[tabKey] !== false);
}

function toggleTabSiren(tabKey){
  if(!Object.prototype.hasOwnProperty.call(DEFAULT_TAB_SIREN_PREFS, tabKey)) return;
  tabSirenPrefs[tabKey] = !isTabSirenEnabled(tabKey);
  saveTabSirenPrefs();
  syncTabSirenButtons();
  renderDevices();
}

function syncTabSirenButtons(){
  const map = [
    { id:'gatewaySirenToggleBtn', key:'gateways', label:'Gateways' },
    { id:'apTabSirenToggleBtn', key:'aps', label:'APs' },
    { id:'routerSirenToggleBtn', key:'routers', label:'Routers/Switches' }
  ];
  map.forEach(item=>{
    const btn = document.getElementById(item.id);
    if(!btn) return;
    btn.textContent = `${item.label} Siren: ${isTabSirenEnabled(item.key) ? 'On' : 'Off'}`;
  });
}

function bindTabSirenControls(){
  const map = [
    { id:'gatewaySirenToggleBtn', key:'gateways' },
    { id:'apTabSirenToggleBtn', key:'aps' },
    { id:'routerSirenToggleBtn', key:'routers' }
  ];
  map.forEach(item=>{
    const btn = document.getElementById(item.id);
    if(!btn) return;
    btn.addEventListener('click', ()=>toggleTabSiren(item.key));
  });
}

function applyCardOrder(tabKey, items){
  const list = Array.isArray(items) ? items.slice() : [];
  const prefs = (cardOrderPrefs && Array.isArray(cardOrderPrefs[tabKey])) ? cardOrderPrefs[tabKey] : [];
  if(!prefs.length || !list.length) return list;
  const byId = new Map();
  list.forEach(item=>{
    if(item && item.id) byId.set(String(item.id), item);
  });
  const used = new Set();
  const ordered = [];
  prefs.forEach(id=>{
    const key = String(id || '');
    if(!key || !byId.has(key) || used.has(key)) return;
    ordered.push(byId.get(key));
    used.add(key);
  });
  list.forEach(item=>{
    const key = item && item.id ? String(item.id) : '';
    if(!key || used.has(key)) return;
    ordered.push(item);
  });
  return ordered;
}

function getGridElementByTab(tabKey){
  if(tabKey === 'gateways') return document.getElementById('gateGrid');
  if(tabKey === 'aps') return document.getElementById('apGrid');
  if(tabKey === 'routers') return document.getElementById('routerGrid');
  return null;
}

function updateCardOrderFromGrid(tabKey){
  const grid = getGridElementByTab(tabKey);
  if(!grid) return;
  const ids = Array.from(grid.querySelectorAll('.card[data-card-id]'))
    .map(el=>String(el.getAttribute('data-card-id') || '').trim())
    .filter(Boolean);
  if(!ids.length) return;
  const next = normalizeCardOrderPrefs(cardOrderPrefs || DEFAULT_CARD_ORDER_PREFS);
  next[tabKey] = ids;
  cardOrderPrefs = next;
  saveCardOrderPrefs();
}

function clearDragTargets(){
  document.querySelectorAll('.card.drop-target').forEach(el=>el.classList.remove('drop-target'));
  document.querySelectorAll('.card.dragging').forEach(el=>el.classList.remove('dragging'));
}

function bindGridDragDrop(gridId, tabKey){
  const grid = document.getElementById(gridId);
  if(!grid || grid.dataset.dragBound === '1') return;
  grid.dataset.dragBound = '1';

  grid.addEventListener('dragstart', ev=>{
    const card = ev.target && ev.target.closest ? ev.target.closest('.card[data-card-id]') : null;
    if(!card) return;
    const id = String(card.getAttribute('data-card-id') || '').trim();
    if(!id) return;
    dragState = { tabKey, cardId: id };
    card.classList.add('dragging');
    if(ev.dataTransfer){
      ev.dataTransfer.effectAllowed = 'move';
      try{ ev.dataTransfer.setData('text/plain', id); }catch(_){ }
    }
  });

  grid.addEventListener('dragover', ev=>{
    if(!dragState || dragState.tabKey !== tabKey) return;
    const target = ev.target && ev.target.closest ? ev.target.closest('.card[data-card-id]') : null;
    if(!target) return;
    ev.preventDefault();
    clearDragTargets();
    target.classList.add('drop-target');
    if(ev.dataTransfer) ev.dataTransfer.dropEffect = 'move';
  });

  grid.addEventListener('drop', ev=>{
    if(!dragState || dragState.tabKey !== tabKey) return;
    ev.preventDefault();
    const target = ev.target && ev.target.closest ? ev.target.closest('.card[data-card-id]') : null;
    const sourceId = dragState.cardId;
    const targetId = target ? String(target.getAttribute('data-card-id') || '').trim() : '';
    if(!sourceId || !targetId || sourceId === targetId){
      dragState = null;
      clearDragTargets();
      return;
    }
    const ids = Array.from(grid.querySelectorAll('.card[data-card-id]'))
      .map(el=>String(el.getAttribute('data-card-id') || '').trim())
      .filter(Boolean);
    const from = ids.indexOf(sourceId);
    const to = ids.indexOf(targetId);
    if(from >= 0 && to >= 0){
      const [moved] = ids.splice(from, 1);
      ids.splice(to, 0, moved);
      const next = normalizeCardOrderPrefs(cardOrderPrefs || DEFAULT_CARD_ORDER_PREFS);
      next[tabKey] = ids;
      cardOrderPrefs = next;
      saveCardOrderPrefs();
      renderDevices();
    }
    dragState = null;
    clearDragTargets();
  });

  grid.addEventListener('dragend', ()=>{
    dragState = null;
    clearDragTargets();
  });
}

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
  if(typeof input.default_tab === "string"){
    normalized.default_tab = normalizeTabId(input.default_tab);
  }
  if(typeof input.sort_mode === "string"){
    normalized.sort_mode = normalizeSortMode(input.sort_mode);
  }
  if(typeof input.group_mode === "string"){
    normalized.group_mode = normalizeGroupMode(input.group_mode);
  }
  if(typeof input.refresh_interval === "string"){
    normalized.refresh_interval = normalizeRefreshInterval(input.refresh_interval);
  }
  const metricKeys = Object.keys(normalized.metrics);
  metricKeys.forEach(key=>{
    if(input.metrics && Object.prototype.hasOwnProperty.call(input.metrics,key)){
      normalized.metrics[key] = !!input.metrics[key];
    }
  });
  return normalized;
}

function normalizeSortMode(mode){
  const raw = String(mode || "").trim().toLowerCase();
  if(raw === "status_name" || raw === "name_asc" || raw === "last_seen_desc"){
    return raw;
  }
  return "manual";
}

function normalizeGroupMode(mode){
  const raw = String(mode || "").trim().toLowerCase();
  if(raw === "role" || raw === "site"){
    return raw;
  }
  return "none";
}

function normalizeRefreshInterval(mode){
  const raw = String(mode || "").trim().toLowerCase();
  if(raw === "fast" || raw === "slow"){
    return raw;
  }
  return "normal";
}

function normalizeQuickFilter(mode){
  const raw = String(mode || "").trim().toLowerCase();
  if(raw === "online" || raw === "offline"){
    return raw;
  }
  return "all";
}

function deviceSortLabel(mode){
  const m = normalizeSortMode(mode);
  if(m === "status_name") return "status + name";
  if(m === "name_asc") return "name";
  if(m === "last_seen_desc") return "last seen";
  return "manual";
}

function deviceGroupLabel(mode){
  const m = normalizeGroupMode(mode);
  if(m === "role") return "role";
  if(m === "site") return "site";
  return "none";
}

function refreshIntervalLabel(mode){
  const m = normalizeRefreshInterval(mode);
  if(m === "fast") return "2s";
  if(m === "slow") return "10s";
  return "5s";
}

function getSuccessPollIntervalMs(){
  const mode = normalizeRefreshInterval(dashSettings && dashSettings.refresh_interval);
  if(mode === "fast") return POLL_INTERVAL_FAST_MS;
  if(mode === "slow") return POLL_INTERVAL_SLOW_MS;
  return POLL_INTERVAL_NORMAL_MS;
}

function getErrorPollBackoffMs(){
  const exponent = Math.max(0, pollErrorCount - 1);
  const delay = POLL_INTERVAL_ERROR_BASE_MS * Math.pow(2, exponent);
  return Math.min(POLL_INTERVAL_ERROR_MAX_MS, delay);
}

function getStaleThresholdMs(){
  return Math.max(DATA_STALE_MIN_MS, getSuccessPollIntervalMs() * 3);
}

function formatRelativeAge(ms){
  if(typeof ms !== "number" || !isFinite(ms) || ms <= 0){
    return "just now";
  }
  const sec = Math.max(0, Math.floor(ms / 1000));
  if(sec < 60) return `${sec}s`;
  const min = Math.floor(sec / 60);
  if(min < 60) return `${min}m ${sec % 60}s`;
  const hr = Math.floor(min / 60);
  return `${hr}h ${min % 60}m`;
}

function compareDevicesBySortMode(a, b, mode){
  const m = normalizeSortMode(mode);
  const an = String((a && a.name) || "").toLowerCase();
  const bn = String((b && b.name) || "").toLowerCase();
  const aid = String((a && a.id) || "");
  const bid = String((b && b.id) || "");

  if(m === "status_name"){
    const ao = a && a.online ? 1 : 0;
    const bo = b && b.online ? 1 : 0;
    if(ao !== bo) return ao - bo;
  } else if(m === "name_asc"){
    // handled by name tie-break below
  } else if(m === "last_seen_desc"){
    const ats = Number((a && a.last_seen) || 0);
    const bts = Number((b && b.last_seen) || 0);
    if(ats !== bts) return bts - ats;
  } else {
    return 0;
  }

  const ncmp = an.localeCompare(bn);
  if(ncmp !== 0) return ncmp;
  return aid.localeCompare(bid);
}

function sortDevicesForDisplay(items){
  const list = Array.isArray(items) ? items.slice() : [];
  const mode = normalizeSortMode(dashSettings && dashSettings.sort_mode);
  if(mode === "manual") return list;
  list.sort((a,b)=>compareDevicesBySortMode(a,b,mode));
  return list;
}

function deviceMatchesQuickFilter(device){
  const mode = normalizeQuickFilter(deviceQuickFilter);
  if(mode === "online") return !!(device && device.online);
  if(mode === "offline") return !!(device && !device.online);
  return true;
}

function deviceMatchesSearch(device){
  const q = String(deviceSearchQuery || "").trim().toLowerCase();
  if(!q) return true;
  if(!device || typeof device !== "object") return false;
  const fields = [
    device.name,
    device.id,
    device.hostname,
    device.mac,
    device.serial,
    device.site,
    device.site_id,
    device.vendor,
    device.model,
    device.source_name,
    device.role
  ];
  const hay = fields.map(v=>String(v || "").toLowerCase()).join(" ");
  return hay.includes(q);
}

function applyViewQuery(devices){
  const list = Array.isArray(devices) ? devices : [];
  return list.filter(d=>deviceMatchesQuickFilter(d)).filter(d=>deviceMatchesSearch(d));
}

function syncViewControls(){
  const searchInput = document.getElementById("deviceSearchInput");
  if(searchInput){
    searchInput.value = deviceSearchQuery;
  }
  const sortEl = document.getElementById("sortModeSelect");
  if(sortEl){
    sortEl.value = normalizeSortMode(dashSettings && dashSettings.sort_mode);
  }
  const groupEl = document.getElementById("groupModeSelect");
  if(groupEl){
    groupEl.value = normalizeGroupMode(dashSettings && dashSettings.group_mode);
  }
  const mode = normalizeQuickFilter(deviceQuickFilter);
  document.querySelectorAll(".filter-btn[data-filter]").forEach(btn=>{
    const value = String(btn.getAttribute("data-filter") || "").trim().toLowerCase();
    btn.classList.toggle("active", value === mode);
  });
}

function bindViewControls(){
  const searchInput = document.getElementById("deviceSearchInput");
  if(searchInput){
    searchInput.addEventListener("input", ()=>{
      deviceSearchQuery = String(searchInput.value || "");
      renderDevices();
    });
  }
  const clearBtn = document.getElementById("deviceSearchClearBtn");
  if(clearBtn){
    clearBtn.addEventListener("click", ()=>{
      deviceSearchQuery = "";
      syncViewControls();
      renderDevices();
    });
  }
  const sortEl = document.getElementById("sortModeSelect");
  if(sortEl){
    sortEl.addEventListener("change", ()=>{
      dashSettings.sort_mode = normalizeSortMode(sortEl.value);
      saveDashSettings();
      renderDevices();
    });
  }
  const groupEl = document.getElementById("groupModeSelect");
  if(groupEl){
    groupEl.addEventListener("change", ()=>{
      dashSettings.group_mode = normalizeGroupMode(groupEl.value);
      saveDashSettings();
      renderDevices();
    });
  }
  document.querySelectorAll(".filter-btn[data-filter]").forEach(btn=>{
    btn.addEventListener("click", ()=>{
      const next = normalizeQuickFilter(btn.getAttribute("data-filter"));
      if(deviceQuickFilter === next) return;
      deviceQuickFilter = next;
      syncViewControls();
      renderDevices();
    });
  });
  syncViewControls();
}

function saveDashSettings(){
  try{
    localStorage.setItem(DASH_SETTINGS_KEY, JSON.stringify(dashSettings));
  }catch(_){ }
  scheduleUserPrefsSave();
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
  const refresh = document.getElementById("settingRefreshInterval");
  if(refresh){
    refresh.value = normalizeRefreshInterval(dashSettings && dashSettings.refresh_interval);
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
  syncViewControls();
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
  const refresh = document.getElementById("settingRefreshInterval");
  if(refresh){
    refresh.addEventListener("change",()=>{
      dashSettings.refresh_interval = normalizeRefreshInterval(refresh.value);
      saveDashSettings();
      schedulePoll(0, "refresh-change");
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
      schedulePoll(0, "settings-reset");
      renderDevices();
    });
  }
}

function initDisplaySettings(){
  if(featureEnabled("display_controls")){
    applyDashboardSettings();
    bindDisplayControls();
  } else {
    const controls = document.querySelector('.display-controls');
    if(controls) controls.style.display = 'none';
  }
  if(featureEnabled("advanced_actions")){
    bindTabSirenControls();
  }
  if(featureEnabled("advanced_actions")){
    bindGridDragDrop('gateGrid','gateways');
    bindGridDragDrop('apGrid','aps');
    bindGridDragDrop('routerGrid','routers');
  }
  if(featureEnabled("advanced_actions")){
    syncTabSirenButtons();
  }
  bindTopologyControls();
  bindSetupWizard();
  bindSourceStatusControls();
  bindViewControls();
  initKioskModeFromUrl();
  ensurePollBannerTicker();
  setActiveTab(dashSettings.default_tab, { persist: false });
  updateDataHealthBanner();
}

function scheduleUserPrefsSave(){
  if(!userPrefsLoaded) return;
  if(userPrefsSaveTimer){
    clearTimeout(userPrefsSaveTimer);
  }
  userPrefsSaveTimer = setTimeout(()=>{ saveUserPrefsToServer(); }, 250);
}

async function saveUserPrefsToServer(){
  if(!userPrefsLoaded) return;
  userPrefsSaveTimer = null;
  try{
    const fd = new FormData();
    fd.append('dashboard_settings', JSON.stringify(dashSettings));
    fd.append('ap_siren_prefs', JSON.stringify(apSirenPrefs));
    fd.append('tab_siren_prefs', JSON.stringify(tabSirenPrefs));
    fd.append('card_order_prefs', JSON.stringify(cardOrderPrefs));
    const resp = await fetch('?ajax=prefs_save', { method:'POST', body:fd });
    if(resp.status===401){ location.reload(); return; }
    const payload = await resp.json().catch(()=>null);
    if(payload && payload.ok && payload.preferences){
      dashSettings = normalizeDashSettings(payload.preferences.dashboard_settings);
      apSirenPrefs = normalizeApSirenPrefs(payload.preferences.ap_siren_prefs);
      tabSirenPrefs = normalizeTabSirenPrefs(payload.preferences.tab_siren_prefs);
      cardOrderPrefs = normalizeCardOrderPrefs(payload.preferences.card_order_prefs);
      try{
        localStorage.setItem(DASH_SETTINGS_KEY, JSON.stringify(dashSettings));
        localStorage.setItem(AP_SIREN_PREFS_KEY, JSON.stringify(apSirenPrefs));
        localStorage.setItem(TAB_SIREN_PREFS_KEY, JSON.stringify(tabSirenPrefs));
        localStorage.setItem(CARD_ORDER_PREFS_KEY, JSON.stringify(cardOrderPrefs));
      }catch(_){ }
      syncTabSirenButtons();
    }
  }catch(_){ }
}

async function loadUserPrefsFromServer(){
  try{
    const resp = await fetch('?ajax=prefs_get&t='+Date.now(), { cache:'no-store' });
    if(resp.status===401){ location.reload(); return; }
    const payload = await resp.json().catch(()=>null);
    if(payload && payload.ok && payload.preferences){
      dashSettings = normalizeDashSettings(payload.preferences.dashboard_settings);
      apSirenPrefs = normalizeApSirenPrefs(payload.preferences.ap_siren_prefs);
      tabSirenPrefs = normalizeTabSirenPrefs(payload.preferences.tab_siren_prefs);
      cardOrderPrefs = normalizeCardOrderPrefs(payload.preferences.card_order_prefs);
      try{
        localStorage.setItem(DASH_SETTINGS_KEY, JSON.stringify(dashSettings));
        localStorage.setItem(AP_SIREN_PREFS_KEY, JSON.stringify(apSirenPrefs));
        localStorage.setItem(TAB_SIREN_PREFS_KEY, JSON.stringify(tabSirenPrefs));
        localStorage.setItem(CARD_ORDER_PREFS_KEY, JSON.stringify(cardOrderPrefs));
      }catch(_){ }
      applyDashboardSettings();
      syncTabSirenButtons();
      syncViewControls();
      setActiveTab(dashSettings.default_tab, { persist: false });
      renderDevices();
    }
  }catch(_){ }
  userPrefsLoaded = true;
}

function setSourceStatusNotice(msg, kind){
  const el = document.getElementById("sourceStatusNotice");
  if(!el) return;
  el.textContent = msg || "";
  el.className = `source-status-notice ${kind || ""}`.trim();
}

function sourceHealthBadge(okValue){
  if(okValue === true) return `<span class="badge good">Healthy</span>`;
  if(okValue === false) return `<span class="badge bad">Failed</span>`;
  return `<span class="badge warn">Not Polled</span>`;
}

function formatPolledAt(value){
  const raw = String(value || "").trim();
  if(!raw) return "Never";
  const ts = Date.parse(raw);
  if(!Number.isFinite(ts)) return "Unknown";
  const ageSec = Math.max(0, Math.round((Date.now() - ts) / 1000));
  if(ageSec < 5) return "just now";
  if(ageSec < 60) return `${ageSec}s ago`;
  const ageMin = Math.round(ageSec / 60);
  if(ageMin < 60) return `${ageMin}m ago`;
  const ageHr = Math.round(ageMin / 60);
  if(ageHr < 48) return `${ageHr}h ago`;
  return new Date(ts).toLocaleString();
}

function renderSourceStatusStrip(){
  const summaryEl = document.getElementById("sourceStatusSummary");
  const listEl = document.getElementById("sourceStatusList");
  if(!summaryEl || !listEl) return;

  if(!Array.isArray(sourceStatusItems) || sourceStatusItems.length === 0){
    summaryEl.textContent = "No active UISP sources configured for this account.";
    listEl.innerHTML = `<div class="source-status-item"><div class="source-status-meta">Open Account Settings or use the setup wizard to add at least one source.</div></div>`;
    return;
  }

  const total = sourceStatusSummary && typeof sourceStatusSummary.total === "number" ? sourceStatusSummary.total : sourceStatusItems.length;
  const healthy = sourceStatusSummary && typeof sourceStatusSummary.healthy === "number" ? sourceStatusSummary.healthy : sourceStatusItems.filter(x=>x && x.ok === true).length;
  const failed = sourceStatusSummary && typeof sourceStatusSummary.failed === "number" ? sourceStatusSummary.failed : sourceStatusItems.filter(x=>x && x.ok === false).length;
  const never = sourceStatusSummary && typeof sourceStatusSummary.never_polled === "number" ? sourceStatusSummary.never_polled : sourceStatusItems.filter(x=>!x || !x.last_poll_at).length;
  const polledAt = sourceStatusSummary && sourceStatusSummary.last_poll_at ? formatPolledAt(sourceStatusSummary.last_poll_at) : "not yet";
  summaryEl.textContent = `Sources: ${healthy}/${total} healthy, ${failed} failed, ${never} not polled. Last poll: ${polledAt}.`;

  listEl.innerHTML = sourceStatusItems.map(item=>{
    const id = String(item.id || "");
    const name = escapeHtml(item.name || id || "Source");
    const url = escapeHtml(item.url || "--");
    const httpText = (typeof item.http === "number" && item.http > 0) ? `HTTP ${item.http}` : "HTTP --";
    const latencyText = (typeof item.latency_ms === "number" && item.latency_ms > 0) ? `${item.latency_ms}ms` : "--";
    const devicesText = (typeof item.device_count === "number") ? String(item.device_count) : "--";
    const errText = item.error ? `Err: ${escapeHtml(item.error)}` : "";
    return `<div class="source-status-item" data-source-id="${escapeHtml(id)}">
      <div class="source-status-title">
        <span class="source-status-name">${name}</span>
        ${sourceHealthBadge(item.ok)}
      </div>
      <div class="source-status-url">${url}</div>
      <div class="source-status-meta">${httpText} | Latency: ${latencyText} | Devices: ${devicesText}</div>
      <div class="source-status-meta">Last poll: ${escapeHtml(formatPolledAt(item.last_poll_at))}${errText ? ` | ${errText}` : ""}</div>
      <div class="source-status-actions">
        <button type="button" class="btn-outline source-poll-btn" data-source-id="${escapeHtml(id)}">Poll Now</button>
      </div>
    </div>`;
  }).join("");
}

function setSourcePollButtonsBusy(busy){
  const pollAllBtn = document.getElementById("pollAllSourcesBtn");
  if(pollAllBtn){
    pollAllBtn.disabled = !!busy;
    pollAllBtn.textContent = busy ? "Polling..." : "Poll All Sources";
  }
  document.querySelectorAll(".source-poll-btn").forEach(btn=>{
    btn.disabled = !!busy;
  });
}

async function loadSourceStatus(force){
  if(sourceStatusLoading) return;
  if(!force && (Date.now() - sourceStatusLastLoadedAt) < SOURCE_STATUS_REFRESH_MS) return;
  sourceStatusLoading = true;
  try{
    const resp = await fetch(`?ajax=sources_status&t=${Date.now()}`, { cache: "no-store" });
    if(resp.status===401){ location.reload(); return; }
    const payload = await resp.json().catch(()=>null);
    if(!payload || !payload.ok){
      setSourceStatusNotice("Unable to load source status.", "error");
      return;
    }
    sourceStatusItems = Array.isArray(payload.sources) ? payload.sources : [];
    sourceStatusSummary = payload.summary && typeof payload.summary === "object" ? payload.summary : null;
    sourceStatusLastLoadedAt = Date.now();
    setSourceStatusNotice("");
    renderSourceStatusStrip();
  }catch(_){
    setSourceStatusNotice("Source status request failed.", "error");
  } finally {
    sourceStatusLoading = false;
  }
}

async function pollSourceById(sourceId, silent, manageBusy, refreshAfter){
  const id = String(sourceId || "").trim();
  if(!id) return false;
  const ownsBusy = (manageBusy !== false);
  const shouldRefresh = (refreshAfter !== false);
  if(ownsBusy) setSourcePollButtonsBusy(true);
  if(!silent) setSourceStatusNotice(`Polling ${id}...`);
  try{
    const fd = new FormData();
    fd.append("id", id);
    const resp = await fetch("?ajax=sources_test", { method: "POST", body: fd });
    if(resp.status===401){ location.reload(); return false; }
    const payload = await resp.json().catch(()=>null);
    if(!payload){
      setSourceStatusNotice("Poll failed: invalid response.", "error");
      return false;
    }
    const ok = !!payload.ok;
    if(!silent){
      const latencyText = (typeof payload.latency_ms === "number") ? `${payload.latency_ms}ms` : "--";
      const deviceText = (typeof payload.device_count === "number") ? payload.device_count : "--";
      if(ok){
        setSourceStatusNotice(`Poll passed (${id}) - devices: ${deviceText}, latency: ${latencyText}.`, "ok");
      } else {
        const httpText = (typeof payload.http === "number") ? `HTTP ${payload.http}` : "unknown HTTP";
        setSourceStatusNotice(`Poll failed (${id}) - ${httpText}${payload.error ? `, ${payload.error}` : ""}.`, "error");
      }
    }
    if(shouldRefresh){
      await loadSourceStatus(true);
    }
    return ok;
  }catch(_){
    setSourceStatusNotice(`Poll failed (${id}) due to request error.`, "error");
    return false;
  } finally {
    if(ownsBusy) setSourcePollButtonsBusy(false);
  }
}

async function pollAllSources(){
  if(!Array.isArray(sourceStatusItems) || sourceStatusItems.length === 0){
    setSourceStatusNotice("No active sources to poll.", "error");
    return;
  }
  setSourcePollButtonsBusy(true);
  setSourceStatusNotice("Polling all sources...");
  let success = 0;
  let failed = 0;
  for(const item of sourceStatusItems){
    const id = String((item && item.id) || "").trim();
    if(!id) continue;
    const ok = await pollSourceById(id, true, false, false);
    if(ok) success++; else failed++;
  }
  await loadSourceStatus(true);
  setSourcePollButtonsBusy(false);
  setSourceStatusNotice(`Poll complete - ${success} passed, ${failed} failed.`, failed > 0 ? "error" : "ok");
}

function bindSourceStatusControls(){
  const pollAllBtn = document.getElementById("pollAllSourcesBtn");
  if(pollAllBtn){
    pollAllBtn.addEventListener("click", ()=>pollAllSources());
  }
  const listEl = document.getElementById("sourceStatusList");
  if(listEl){
    listEl.addEventListener("click", ev=>{
      const target = ev.target && ev.target.closest ? ev.target.closest(".source-poll-btn") : null;
      if(!target) return;
      const id = String(target.getAttribute("data-source-id") || "").trim();
      if(!id) return;
      pollSourceById(id, false);
    });
  }
}

function setSetupWizardStatus(msg, kind){
  const el = document.getElementById("setupWizardStatus");
  if(!el) return;
  el.textContent = msg || "";
  el.className = `wizard-status ${kind || ""}`.trim();
}

function setSetupWizardBusy(busy){
  const saveBtn = document.getElementById("wizardSaveTestBtn");
  const skipBtn = document.getElementById("wizardSkipBtn");
  const settingsBtn = document.getElementById("wizardOpenSettingsBtn");
  const nameEl = document.getElementById("wizardSourceName");
  const urlEl = document.getElementById("wizardSourceUrl");
  const tokenEl = document.getElementById("wizardSourceToken");
  [saveBtn, skipBtn, settingsBtn, nameEl, urlEl, tokenEl].forEach(el=>{
    if(el) el.disabled = !!busy;
  });
}

function showSetupWizard(){
  const modal = document.getElementById("setupWizardModal");
  if(!modal) return;
  modal.style.display = "block";
  modal.setAttribute("aria-hidden", "false");
}

function hideSetupWizard(){
  const modal = document.getElementById("setupWizardModal");
  if(!modal) return;
  modal.style.display = "none";
  modal.setAttribute("aria-hidden", "true");
}

async function saveAndTestSetupWizardSource(){
  const nameEl = document.getElementById("wizardSourceName");
  const urlEl = document.getElementById("wizardSourceUrl");
  const tokenEl = document.getElementById("wizardSourceToken");
  if(!urlEl || !tokenEl){
    return;
  }
  const name = (nameEl && nameEl.value ? String(nameEl.value) : "").trim();
  const url = String(urlEl.value || "").trim();
  const token = String(tokenEl.value || "").trim();
  if(!url){
    setSetupWizardStatus("UISP URL is required.", "error");
    return;
  }
  if(!token){
    setSetupWizardStatus("UISP API token is required.", "error");
    return;
  }

  setSetupWizardBusy(true);
  setSetupWizardStatus("Saving source...", "");
  try{
    const fd = new FormData();
    fd.append("name", name);
    fd.append("url", url);
    fd.append("token", token);
    fd.append("enabled", "1");
    const saveResp = await fetch("?ajax=sources_save", { method: "POST", body: fd });
    if(saveResp.status===401){ location.reload(); return; }
    const savePayload = await saveResp.json().catch(()=>null);
    if(!savePayload || !savePayload.ok){
      setSetupWizardStatus(`Save failed: ${((savePayload && (savePayload.message || savePayload.error)) || "unknown")}`, "error");
      return;
    }

    const sourceId = String(savePayload.id || "").trim();
    if(!sourceId){
      setSetupWizardStatus("Source saved, but test could not start (missing source id).", "error");
      return;
    }

    setSetupWizardStatus("Source saved. Testing connection...", "");
    const testFd = new FormData();
    testFd.append("id", sourceId);
    const testResp = await fetch("?ajax=sources_test", { method: "POST", body: testFd });
    if(testResp.status===401){ location.reload(); return; }
    const testPayload = await testResp.json().catch(()=>null);
    if(!testPayload){
      setSetupWizardStatus("Connection test failed: invalid response.", "error");
      return;
    }
    if(testPayload.ok){
      const latency = typeof testPayload.latency_ms === "number" ? `${testPayload.latency_ms}ms` : "--";
      const count = typeof testPayload.device_count === "number" ? testPayload.device_count : 0;
      setSetupWizardStatus(`Connected. Devices: ${count}, latency: ${latency}.`, "ok");
      setTimeout(()=>{
        hideSetupWizard();
        fetchDevices({ force: true, reason: "wizard-save-test" });
        loadSourceStatus(true);
      }, 500);
      return;
    }

    const httpCode = (typeof testPayload.http === "number") ? `HTTP ${testPayload.http}` : "connection failed";
    const err = testPayload.error ? `, ${testPayload.error}` : "";
    setSetupWizardStatus(`Source saved but test failed (${httpCode}${err}).`, "error");
  }catch(_){
    setSetupWizardStatus("Unable to save/test source right now.", "error");
  } finally {
    setSetupWizardBusy(false);
  }
}

function bindSetupWizard(){
  const saveBtn = document.getElementById("wizardSaveTestBtn");
  if(saveBtn){
    saveBtn.addEventListener("click", ()=>saveAndTestSetupWizardSource());
  }
  const skipBtn = document.getElementById("wizardSkipBtn");
  if(skipBtn){
    skipBtn.addEventListener("click", ()=>{
      setSetupWizardStatus("");
      hideSetupWizard();
    });
  }
  const settingsBtn = document.getElementById("wizardOpenSettingsBtn");
  if(settingsBtn){
    settingsBtn.addEventListener("click", ()=>manageUispSources());
  }
  const form = document.getElementById("setupWizardForm");
  if(form){
    form.addEventListener("submit", ev=>{
      ev.preventDefault();
      saveAndTestSetupWizardSource();
    });
  }
}

async function ensureFirstRunSourceWizard(){
  const modal = document.getElementById("setupWizardModal");
  if(!modal) return;
  try{
    const resp = await fetch(`?ajax=sources_list&t=${Date.now()}`, { cache: "no-store" });
    if(resp.status===401){ location.reload(); return; }
    const payload = await resp.json().catch(()=>null);
    const sources = (payload && payload.ok && Array.isArray(payload.sources)) ? payload.sources : null;
    if(sources && sources.length > 0){
      hideSetupWizard();
      return;
    }
    showSetupWizard();
    setSetupWizardStatus("No UISP source found for this account. Add one to begin.", "");
  }catch(_){
    showSetupWizard();
    setSetupWizardStatus("Unable to verify saved sources. You can still add one now.", "error");
  }
}

function escapeHtml(value){
  return String(value ?? "")
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

function formatLastSeen(value){
  const n = Number(value || 0);
  if(!isFinite(n) || n <= 0) return "--";
  const ms = n < 2000000000 ? n * 1000 : n;
  try{
    return new Date(ms).toLocaleString();
  }catch(_){
    return "--";
  }
}

function renderSiteAndLastSeen(device){
  const site = escapeHtml(device.site || device.site_id || "--");
  const lastSeen = escapeHtml(formatLastSeen(device.last_seen));
  return `<div class="card-meta">Site: ${site}</div><div class="card-meta">Last Seen: ${lastSeen}</div>`;
}

function getInventorySummary(device){
  if(!device || !device.id) return null;
  const row = inventoryOverviewByDevice[String(device.id)] || null;
  return row && typeof row === "object" ? row : null;
}

async function fetchInventoryOverview(force){
  if(!featureEnabled("inventory")) return;
  if(inventoryOverviewLoading) return;
  if(!force && (Date.now() - inventoryOverviewFetchedAt) < INVENTORY_OVERVIEW_REFRESH_MS) return;
  inventoryOverviewLoading = true;
  try{
    const resp = await fetch(`?ajax=inventory_overview&t=${Date.now()}`, { cache:"no-store" });
    if(resp.status===401){ location.reload(); return; }
    const payload = await resp.json().catch(()=>null);
    if(payload && payload.ok && payload.devices && typeof payload.devices === "object"){
      inventoryOverviewByDevice = payload.devices;
      inventoryOverviewError = "";
      inventoryOverviewFetchedAt = Date.now();
      renderDevices();
    } else {
      inventoryOverviewError = payload && payload.error ? String(payload.error) : "invalid_payload";
    }
  }catch(_){
    inventoryOverviewError = "request_failed";
  } finally {
    inventoryOverviewLoading = false;
  }
}

function getInventoryCardBadges(device){
  const summary = getInventorySummary(device);
  if(!summary) return "";
  const bits = [];
  const hasIdentity = !!summary.identity_id;
  bits.push(`<span class="badge ${hasIdentity ? "good" : "warn"}">Inv ${hasIdentity ? "Linked" : "Pending"}</span>`);
  if(summary.drift_changed){
    bits.push(`<span class="badge bad">Drift</span>`);
  }
  if(summary.lifecycle_level){
    const lvl = String(summary.lifecycle_level).toLowerCase();
    const cls = (lvl === "high") ? "bad" : ((lvl === "medium" || lvl === "moderate") ? "warn" : "good");
    bits.push(`<span class="badge ${cls}">Lifecycle: ${escapeHtml(summary.lifecycle_level)}</span>`);
  }
  return bits.join(" ");
}

function formatBitrate(v){
  if(typeof v !== "number" || !isFinite(v) || v <= 0) return "--";
  if(v >= 1_000_000_000) return `${(v/1_000_000_000).toFixed(2)} Gbps`;
  if(v >= 1_000_000) return `${(v/1_000_000).toFixed(1)} Mbps`;
  if(v >= 1_000) return `${(v/1_000).toFixed(1)} Kbps`;
  return `${Math.round(v)} bps`;
}

function lifecycleClass(level){
  const v = String(level || "").toLowerCase();
  if(v === "high") return "bad";
  if(v === "medium" || v === "moderate") return "warn";
  if(v === "low") return "good";
  return "warn";
}

function setInventoryModalStatus(msg, cls){
  const el = document.getElementById("inventoryStatus");
  if(!el) return;
  el.textContent = msg || "";
  el.className = `inventory-status ${cls || ""}`.trim();
}

function closeInventory(){
  const modal = document.getElementById("inventoryModal");
  if(modal) modal.style.display = "none";
}

function renderInventoryModal(payload, deviceName){
  const identityWrap = document.getElementById("inventoryIdentity");
  const ifaceWrap = document.getElementById("inventoryInterfaces");
  const neighWrap = document.getElementById("inventoryNeighbors");
  const driftWrap = document.getElementById("inventoryDrift");
  const titleEl = document.getElementById("inventoryTitle");
  if(titleEl){
    titleEl.textContent = deviceName ? `Inventory - ${deviceName}` : "Inventory";
  }
  if(!identityWrap || !ifaceWrap || !neighWrap || !driftWrap) return;

  const identity = payload && payload.identity ? payload.identity : null;
  if(!identity){
    identityWrap.innerHTML = `<div class="inventory-empty">${escapeHtml((payload && payload.message) || "No identity found yet.")}</div>`;
    ifaceWrap.innerHTML = "";
    neighWrap.innerHTML = "";
    driftWrap.innerHTML = "";
    return;
  }

  const lifecycle = payload.lifecycle || null;
  const ifaceSummary = payload.interface_summary || {};
  const ifaceRows = Array.isArray(payload.interfaces) ? payload.interfaces : [];
  const neighRows = Array.isArray(payload.neighbors) ? payload.neighbors : [];
  const driftRows = Array.isArray(payload.drift) ? payload.drift : [];

  identityWrap.innerHTML = `
    <div class="inventory-kv"><span>Identity ID</span><code>${escapeHtml(identity.identity_id || "--")}</code></div>
    <div class="inventory-kv"><span>Primary Device</span><code>${escapeHtml(identity.primary_device_id || "--")}</code></div>
    <div class="inventory-kv"><span>Name</span><strong>${escapeHtml(identity.name || "--")}</strong></div>
    <div class="inventory-kv"><span>Role/Site</span><span>${escapeHtml(identity.role || "--")} / ${escapeHtml(identity.site_id || "--")}</span></div>
    <div class="inventory-kv"><span>Vendor/Model</span><span>${escapeHtml(identity.vendor || "--")} / ${escapeHtml(identity.model || "--")}</span></div>
    <div class="inventory-kv"><span>Last Seen</span><span>${identity.last_seen ? new Date(identity.last_seen).toLocaleString() : "--"}</span></div>
    <div class="inventory-kv"><span>Lifecycle</span><span class="badge ${lifecycleClass(lifecycle && lifecycle.level)}">${escapeHtml((lifecycle && lifecycle.level) || "unknown")} (${escapeHtml((lifecycle && lifecycle.score) ?? "--")})</span></div>
    <div class="inventory-kv"><span>Interfaces</span><span>${escapeHtml(ifaceSummary.total ?? 0)} total, ${escapeHtml(ifaceSummary.oper_up ?? 0)} up, ${escapeHtml(ifaceSummary.oper_down ?? 0)} down</span></div>
  `;

  if(ifaceRows.length){
    ifaceWrap.innerHTML = `
      <table class="inventory-table">
        <thead>
          <tr><th>Name</th><th>Admin</th><th>Oper</th><th>Rx</th><th>Tx</th><th>Error</th></tr>
        </thead>
        <tbody>
          ${ifaceRows.slice(0,20).map(row=>`
            <tr>
              <td>${escapeHtml(row.name || "--")}</td>
              <td>${row.admin_up === true ? "up" : (row.admin_up === false ? "down" : "--")}</td>
              <td>${row.oper_up === true ? "up" : (row.oper_up === false ? "down" : "--")}</td>
              <td>${formatBitrate(row.rx_bps)}</td>
              <td>${formatBitrate(row.tx_bps)}</td>
              <td>${typeof row.error_rate === "number" ? row.error_rate.toFixed(4) : "--"}</td>
            </tr>
          `).join("")}
        </tbody>
      </table>
    `;
  } else {
    ifaceWrap.innerHTML = `<div class="inventory-empty">No interface facts ingested yet.</div>`;
  }

  if(neighRows.length){
    neighWrap.innerHTML = `
      <table class="inventory-table">
        <thead>
          <tr><th>Local Intf</th><th>Neighbor</th><th>Neighbor Intf</th><th>Protocol</th></tr>
        </thead>
        <tbody>
          ${neighRows.slice(0,20).map(row=>`
            <tr>
              <td>${escapeHtml(row.local_interface || "--")}</td>
              <td>${escapeHtml(row.neighbor_device_name || row.neighbor_identity_hint || "--")}</td>
              <td>${escapeHtml(row.neighbor_interface_hint || "--")}</td>
              <td>${escapeHtml(row.protocol || "--")}</td>
            </tr>
          `).join("")}
        </tbody>
      </table>
    `;
  } else {
    neighWrap.innerHTML = `<div class="inventory-empty">No neighbor links ingested yet.</div>`;
  }

  if(driftRows.length){
    driftWrap.innerHTML = `
      <table class="inventory-table">
        <thead>
          <tr><th>Observed</th><th>Changed</th><th>Fingerprint</th></tr>
        </thead>
        <tbody>
          ${driftRows.slice(0,10).map(row=>`
            <tr>
              <td>${row.observed_at ? new Date(row.observed_at).toLocaleString() : "--"}</td>
              <td><span class="badge ${row.changed ? "bad" : "good"}">${row.changed ? "yes" : "no"}</span></td>
              <td><code>${escapeHtml(String(row.fingerprint || "").slice(0,18) || "--")}</code></td>
            </tr>
          `).join("")}
        </tbody>
      </table>
    `;
  } else {
    driftWrap.innerHTML = `<div class="inventory-empty">No drift snapshots available.</div>`;
  }
}

async function openInventory(id, name){
  if(!featureEnabled("inventory")) return;
  const modal = document.getElementById("inventoryModal");
  if(!modal) return;
  modal.style.display = "block";
  setInventoryModalStatus("Loading inventory...", "");
  const identityWrap = document.getElementById("inventoryIdentity");
  const ifaceWrap = document.getElementById("inventoryInterfaces");
  const neighWrap = document.getElementById("inventoryNeighbors");
  const driftWrap = document.getElementById("inventoryDrift");
  if(identityWrap) identityWrap.innerHTML = "";
  if(ifaceWrap) ifaceWrap.innerHTML = "";
  if(neighWrap) neighWrap.innerHTML = "";
  if(driftWrap) driftWrap.innerHTML = "";

  try{
    const params = new URLSearchParams({ ajax:"inventory_device", id:String(id || "") });
    if(name) params.set("name", String(name));
    const resp = await fetch(`?${params.toString()}`, { cache:"no-store" });
    if(resp.status===401){ location.reload(); return; }
    const payload = await resp.json().catch(()=>null);
    if(!payload || !payload.ok){
      const msg = payload && payload.message ? payload.message : (payload && payload.error ? payload.error : "Unable to load inventory.");
      setInventoryModalStatus(msg, "error");
      return;
    }
    setInventoryModalStatus("Inventory loaded.", "ok");
    renderInventoryModal(payload, name || id);
  }catch(_){
    setInventoryModalStatus("Failed to load inventory.", "error");
  }
}

function setTopologyStatus(msg, isError){
  const el = document.getElementById("topologyStatus");
  if(!el) return;
  el.textContent = msg || "";
  el.classList.toggle("error", !!isError);
}

function topologyEdgeState(edge){
  if(!edge || edge.resolved !== true) return "unresolved";
  const updated = edge.updated_at ? Date.parse(edge.updated_at) : NaN;
  if(Number.isFinite(updated)){
    const ageMs = Date.now() - updated;
    if(ageMs > 6 * 60 * 60 * 1000){
      return "stale";
    }
  }
  return "resolved";
}

function topologyNodeLabel(node){
  if(!node) return "unknown";
  return node.label || node.identity_id || node.node_id || "unknown";
}

function buildTopologyLayout(nodes, width, height){
  const managed = nodes.filter(n=>n.kind === "managed");
  const external = nodes.filter(n=>n.kind !== "managed");
  const map = new Map();
  const centerX = width / 2;
  const centerY = height / 2;
  const managedRadius = Math.max(130, Math.min(width, height) * 0.28);
  const externalRadius = Math.max(managedRadius + 95, Math.min(width, height) * 0.44);

  const placeRing = (items, radius, phaseDeg) => {
    const total = Math.max(items.length, 1);
    items.forEach((node, idx)=>{
      const angle = ((idx / total) * 360 + phaseDeg) * Math.PI / 180;
      map.set(node.node_id, {
        x: centerX + radius * Math.cos(angle),
        y: centerY + radius * Math.sin(angle)
      });
    });
  };

  placeRing(managed, managedRadius, -90);
  placeRing(external, externalRadius, -75);
  return map;
}

function renderTopologyHealth(){
  const el = document.getElementById("topologyHealthSummary");
  if(!el) return;
  const health = topologyCache && topologyCache.health ? topologyCache.health : null;
  if(!health){
    el.innerHTML = "";
    return;
  }
  const parts = [
    `<span class="badge good">Nodes: ${health.node_count ?? 0}</span>`,
    `<span class="badge good">Managed: ${health.managed_node_count ?? 0}</span>`,
    `<span class="badge ${((health.unknown_neighbor_edges ?? 0) > 0) ? "warn" : "good"}">Unknown Links: ${health.unknown_neighbor_edges ?? 0}</span>`,
    `<span class="badge ${((health.isolated_managed_nodes ?? 0) > 0) ? "warn" : "good"}">Isolated: ${health.isolated_managed_nodes ?? 0}</span>`,
    `<span class="badge good">Components: ${health.connected_components ?? 0}</span>`
  ];
  el.innerHTML = parts.join(" ");
}

function populateTopologySelectors(){
  const sourceEl = document.getElementById("topologySourceSelect");
  const targetEl = document.getElementById("topologyTargetSelect");
  if(!sourceEl || !targetEl) return;
  const managed = (topologyCache.nodes || []).filter(n=>n.kind === "managed");
  const options = managed.map(n=>`<option value="${escapeHtml(n.node_id)}">${escapeHtml(topologyNodeLabel(n))}</option>`).join("");
  const prevSource = sourceEl.value;
  const prevTarget = targetEl.value;
  sourceEl.innerHTML = `<option value="">Select Source</option>${options}`;
  targetEl.innerHTML = `<option value="">Select Target</option>${options}`;
  if(prevSource && managed.some(n=>n.node_id === prevSource)) sourceEl.value = prevSource;
  if(prevTarget && managed.some(n=>n.node_id === prevTarget)) targetEl.value = prevTarget;
}

function renderTopologyGraph(){
  const svg = document.getElementById("topologySvg");
  if(!svg) return;
  const nodes = Array.isArray(topologyCache.nodes) ? topologyCache.nodes : [];
  const edges = Array.isArray(topologyCache.edges) ? topologyCache.edges : [];
  if(!nodes.length){
    svg.innerHTML = "";
    return;
  }

  const width = 1200;
  const height = 680;
  const pos = buildTopologyLayout(nodes, width, height);
  const highlightedNodeSet = new Set((topologyTrace && topologyTrace.node_ids) ? topologyTrace.node_ids : []);
  const highlightedEdgeSet = new Set((topologyTrace && topologyTrace.edge_ids) ? topologyTrace.edge_ids : []);
  const hasTrace = highlightedNodeSet.size > 0;

  let edgeSvg = "";
  edges.forEach(edge=>{
    const a = pos.get(edge.from_node_id);
    const b = pos.get(edge.to_node_id);
    if(!a || !b) return;
    const state = topologyEdgeState(edge);
    const edgeId = edge.edge_id || `${edge.from_node_id}->${edge.to_node_id}`;
    const isHighlight = highlightedEdgeSet.has(edgeId);
    const dim = hasTrace && !isHighlight;
    edgeSvg += `<line class="topo-edge ${state} ${dim ? "dimmed" : ""}" x1="${a.x.toFixed(1)}" y1="${a.y.toFixed(1)}" x2="${b.x.toFixed(1)}" y2="${b.y.toFixed(1)}"></line>`;
  });

  let nodeSvg = "";
  nodes.forEach(node=>{
    const p = pos.get(node.node_id);
    if(!p) return;
    const selected = highlightedNodeSet.has(node.node_id);
    const dim = hasTrace && !selected;
    const radius = (node.kind === "managed") ? 12 : 9;
    const label = escapeHtml(topologyNodeLabel(node));
    const sub = node.kind === "managed" ? escapeHtml(node.site_id || node.role || "") : "external";
    nodeSvg += `<g class="topo-node ${node.kind || "external"} ${selected ? "selected" : ""} ${dim ? "dimmed" : ""}" data-node-id="${escapeHtml(node.node_id)}">
      <circle cx="${p.x.toFixed(1)}" cy="${p.y.toFixed(1)}" r="${radius}"></circle>
      <text x="${(p.x + radius + 6).toFixed(1)}" y="${(p.y - 3).toFixed(1)}">${label}</text>
      <text x="${(p.x + radius + 6).toFixed(1)}" y="${(p.y + 11).toFixed(1)}" style="font-size:10px;fill:#a2a2a2;">${sub}</text>
    </g>`;
  });

  svg.innerHTML = `<g>${edgeSvg}${nodeSvg}</g>`;
}

function renderTopology(){
  renderTopologyHealth();
  populateTopologySelectors();
  renderTopologyGraph();
}

async function loadTopology(force){
  if(!featureEnabled("topology")) return;
  if(topologyLoading) return;
  if(!force && (Date.now() - topologyLastFetchMs) < TOPOLOGY_REFRESH_MS) return;
  topologyLoading = true;
  setTopologyStatus("Loading topology...", false);
  try{
    const resp = await fetch(`?ajax=topology_overview&t=${Date.now()}`, { cache:"no-store" });
    if(resp.status===401){ location.reload(); return; }
    const payload = await resp.json().catch(()=>null);
    if(!payload || !payload.ok){
      const msg = (payload && (payload.message || payload.error)) ? (payload.message || payload.error) : "Unable to load topology.";
      setTopologyStatus(msg, true);
      return;
    }
    topologyCache = {
      nodes: Array.isArray(payload.nodes) ? payload.nodes : [],
      edges: Array.isArray(payload.edges) ? payload.edges : [],
      health: (payload.health && typeof payload.health === "object") ? payload.health : null,
      fetched_at: payload.fetched_at || ""
    };
    topologyLastFetchMs = Date.now();
    setTopologyStatus(`Topology loaded: ${topologyCache.nodes.length} nodes, ${topologyCache.edges.length} edges.`, false);
    renderTopology();
  }catch(_){
    setTopologyStatus("Topology request failed.", true);
  } finally {
    topologyLoading = false;
  }
}

function clearTopologyTrace(){
  topologyTrace = null;
  renderTopology();
  setTopologyStatus("Trace cleared.", false);
}

async function traceTopologyPath(){
  if(!featureEnabled("topology")) return;
  const sourceEl = document.getElementById("topologySourceSelect");
  const targetEl = document.getElementById("topologyTargetSelect");
  if(!sourceEl || !targetEl) return;
  const sourceNodeID = String(sourceEl.value || "").trim();
  const targetNodeID = String(targetEl.value || "").trim();
  if(!sourceNodeID || !targetNodeID){
    setTopologyStatus("Select both source and target.", true);
    return;
  }
  if(sourceNodeID === targetNodeID){
    setTopologyStatus("Source and target must be different.", true);
    return;
  }
  setTopologyStatus("Tracing path...", false);
  try{
    const params = new URLSearchParams({ ajax:"topology_trace", source_node_id:sourceNodeID, target_node_id:targetNodeID, t:String(Date.now()) });
    const resp = await fetch(`?${params.toString()}`, { cache:"no-store" });
    if(resp.status===401){ location.reload(); return; }
    const payload = await resp.json().catch(()=>null);
    if(!payload || !payload.ok){
      setTopologyStatus((payload && (payload.message || payload.error)) || "Path trace failed.", true);
      return;
    }
    if(!payload.found){
      topologyTrace = null;
      renderTopology();
      setTopologyStatus(payload.message || "No path found.", true);
      return;
    }
    const nodeIds = Array.isArray(payload.nodes) ? payload.nodes.map(n=>n && n.node_id).filter(Boolean) : [];
    const edgeIds = Array.isArray(payload.edges) ? payload.edges.map(e=>e && e.edge_id).filter(Boolean) : [];
    topologyTrace = { node_ids: nodeIds, edge_ids: edgeIds };
    renderTopology();
    setTopologyStatus(`Path found: ${payload.hops ?? Math.max(0,nodeIds.length-1)} hops.`, false);
  }catch(_){
    setTopologyStatus("Path trace request failed.", true);
  }
}

function bindTopologyControls(){
  if(!featureEnabled("topology")) return;
  const refresh = document.getElementById("topologyRefreshBtn");
  const trace = document.getElementById("topologyTraceBtn");
  const clear = document.getElementById("topologyClearTraceBtn");
  if(refresh) refresh.addEventListener("click", ()=>loadTopology(true));
  if(trace) trace.addEventListener("click", ()=>traceTopologyPath());
  if(clear) clear.addEventListener("click", ()=>clearTopologyTrace());
}

function setDataHealthBanner(message, level){
  const banner = document.getElementById("dataHealthBanner");
  if(!banner) return;
  const msg = String(message || "").trim();
  if(!msg){
    banner.textContent = "";
    banner.className = "data-health-banner";
    banner.style.display = "none";
    return;
  }
  banner.textContent = msg;
  banner.className = `data-health-banner ${level || ""}`.trim();
  banner.style.display = "block";
}

function updateDataHealthBanner(){
  if(pollErrorCount > 0){
    const nextMs = Math.max(0, pollNextRetryAtMs - Date.now());
    const retryIn = formatRelativeAge(nextMs);
    const attempts = pollErrorCount;
    const errText = pollLastErrorMessage ? ` Last error: ${pollLastErrorMessage}.` : "";
    const fallbackAgeMs = snapshotFallbackSavedAtMs > 0 ? Math.max(0, Date.now() - snapshotFallbackSavedAtMs) : 0;
    const fallbackText = usingSnapshotFallback ? ` Showing cached snapshot (${formatRelativeAge(fallbackAgeMs)} old).` : "";
    setDataHealthBanner(`API degraded: retry #${attempts} in ${retryIn}.${errText}${fallbackText}`, "bad");
    return;
  }
  if(pollLastSuccessAtMs > 0){
    const ageMs = Date.now() - pollLastSuccessAtMs;
    if(ageMs >= getStaleThresholdMs()){
      setDataHealthBanner(`Data is stale: last successful update ${formatRelativeAge(ageMs)} ago.`, "warn");
      return;
    }
  }
  setDataHealthBanner("", "");
}

function ensurePollBannerTicker(){
  if(pollBannerTickTimer) return;
  pollBannerTickTimer = setInterval(()=>updateDataHealthBanner(), 1000);
}

function updateDeviceStatusMemory(devices, nowMs){
  const list = Array.isArray(devices) ? devices : [];
  const seen = new Set();
  list.forEach(device=>{
    const id = String((device && device.id) || "").trim();
    if(!id) return;
    seen.add(id);
    const online = !!(device && device.online);
    const record = deviceStatusMemory.get(id);
    if(!record){
      deviceStatusMemory.set(id, { online, changedAtMs: 0, seenAtMs: nowMs });
      return;
    }
    if(record.online !== online){
      record.online = online;
      record.changedAtMs = nowMs;
    }
    record.seenAtMs = nowMs;
  });
  deviceStatusMemory.forEach((record, id)=>{
    if(seen.has(id)) return;
    if(nowMs - (record.seenAtMs || 0) > 3600000){
      deviceStatusMemory.delete(id);
    }
  });
}

function getDeviceTransitionClass(device, nowMs){
  const id = String((device && device.id) || "").trim();
  if(!id) return "";
  const record = deviceStatusMemory.get(id);
  if(!record || !record.changedAtMs) return "";
  if(nowMs - record.changedAtMs > DEVICE_CHANGE_HIGHLIGHT_MS){
    return "";
  }
  return record.online ? "status-changed-online" : "status-changed-offline";
}

function schedulePoll(delayMs, reason){
  if(pollTimer){
    clearTimeout(pollTimer);
  }
  const fallback = (reason === "error") ? getErrorPollBackoffMs() : getSuccessPollIntervalMs();
  const ms = (typeof delayMs === "number" && isFinite(delayMs)) ? Math.max(0, delayMs) : fallback;
  pollNextRetryAtMs = Date.now() + ms;
  updateDataHealthBanner();
  pollTimer = setTimeout(()=>{ fetchDevices({ reason: reason || "scheduled" }); }, ms);
}

async function fetchDevices(opts={}){
  const requestId = ++fetchRequestId;
  const startMutation = mutationVersion;
  const startedAt = Date.now();
  const metaBase = { updated: new Date().toLocaleTimeString() };
  const nextDelaySuccess = opts.nextDelaySuccess ?? getSuccessPollIntervalMs();

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

    if(parseFailed || !payload || !Array.isArray(payload.devices) || resp.status < 200 || resp.status >= 300){
      pollErrorCount += 1;
      pollLastFailureAtMs = Date.now();
      pollLastErrorMessage = parseFailed ? "invalid JSON payload" : `HTTP ${resp.status}`;
      const metaErr = Object.assign({}, meta, {
        http: parseFailed ? 'ERR' : meta.http,
        api_latency: parseFailed ? '--' : meta.api_latency
      });
      const snapshot = loadDeviceSnapshot();
      if(snapshot && Array.isArray(snapshot.devices) && snapshot.devices.length){
        usingSnapshotFallback = true;
        snapshotFallbackSavedAtMs = snapshot.savedAtMs > 0 ? snapshot.savedAtMs : Date.now();
        devicesCache = snapshot.devices;
        const metaCache = Object.assign({}, metaErr, {
          updated: snapshot.savedAtMs > 0 ? new Date(snapshot.savedAtMs).toLocaleTimeString() : metaErr.updated
        });
        renderDevices(metaCache, { fromServer:false });
      } else {
        usingSnapshotFallback = false;
        snapshotFallbackSavedAtMs = 0;
        renderDevices(metaErr);
      }
      schedulePoll(undefined, "error");
      return;
    }
    pollErrorCount = 0;
    pollLastErrorMessage = "";
    pollLastSuccessAtMs = Date.now();
    usingSnapshotFallback = false;
    snapshotFallbackSavedAtMs = 0;

    if(!opts.force && startMutation !== mutationVersion){
      renderDevices(meta);
      schedulePoll(POLL_INTERVAL_FAST_MS);
      return;
    }

    devicesCache = payload.devices;
    saveDeviceSnapshot(devicesCache);
    renderDevices(meta, {fromServer:true});
    schedulePoll(nextDelaySuccess, "success");
  }catch(_){
    if(requestId !== fetchRequestId) return;
    pollErrorCount += 1;
    pollLastFailureAtMs = Date.now();
    pollLastErrorMessage = "request failed";
    const meta={
      http:'ERR',
      api_latency:'--',
      updated: metaBase.updated
    };
    const snapshot = loadDeviceSnapshot();
    if(snapshot && Array.isArray(snapshot.devices) && snapshot.devices.length){
      usingSnapshotFallback = true;
      snapshotFallbackSavedAtMs = snapshot.savedAtMs > 0 ? snapshot.savedAtMs : Date.now();
      devicesCache = snapshot.devices;
      const metaCache = Object.assign({}, meta, {
        updated: snapshot.savedAtMs > 0 ? new Date(snapshot.savedAtMs).toLocaleTimeString() : meta.updated
      });
      renderDevices(metaCache, { fromServer:false });
    } else {
      usingSnapshotFallback = false;
      snapshotFallbackSavedAtMs = 0;
      renderDevices(meta);
    }
    schedulePoll(undefined, "error");
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
  fetchInventoryOverview(false);
  const topoPanel = document.getElementById("topology");
  if(featureEnabled("topology") && topoPanel && topoPanel.style.display === "block"){
    loadTopology(false);
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
  updateDeviceStatusMemory(devices, nowMs);

  const allGateways = devices.filter(d=>d.gateway);
  const allAps = devices.filter(d=>!d.gateway && d.ap);
  const allRoutersSwitches = devices.filter(d=>!d.gateway && !d.ap && (d.router || d.switch));

  const visibleDevices = applyViewQuery(devices);
  const visibleGatewaysBase = visibleDevices.filter(d=>d.gateway);
  const visibleApsBase = visibleDevices.filter(d=>!d.gateway && d.ap);
  const visibleRoutersBase = visibleDevices.filter(d=>!d.gateway && !d.ap && (d.router || d.switch));
  const sortMode = normalizeSortMode(dashSettings && dashSettings.sort_mode);
  const sortedGateways = sortDevicesForDisplay(visibleGatewaysBase);
  const sortedAps = sortDevicesForDisplay(visibleApsBase);
  const sortedRouters = sortDevicesForDisplay(visibleRoutersBase);
  const gateways = (sortMode === "manual") ? applyCardOrder('gateways', sortedGateways) : sortedGateways;
  const aps = (sortMode === "manual") ? applyCardOrder('aps', sortedAps) : sortedAps;
  const routersSwitches = (sortMode === "manual") ? applyCardOrder('routers', sortedRouters) : sortedRouters;

  renderGatewayGrid(gateways, nowSec, nowMs);
  requestAnimationFrame(()=> {
    renderApGrid(aps, nowSec, nowMs);
    requestAnimationFrame(()=> renderRouterSwitchGrid(routersSwitches, nowSec, nowMs));
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
  const offlineGw=allGateways.filter(d=>!d.online).length;
  const unacked=allGateways.filter(d=>!d.online && !(d.ack_until && d.ack_until>nowSec)).length;
  const latVals=allGateways.map(d=>{
    if(typeof d.latency==='number' && isFinite(d.latency)) return d.latency;
    if(typeof d.cpe_latency==='number' && isFinite(d.cpe_latency)) return d.cpe_latency;
    return null;
  }).filter(v=>v!==null);
  const avgLat = latVals.length ? Math.round(latVals.reduce((a,b)=>a+b,0)/latVals.length) : null;
  const highCpu=allGateways.filter(d=>typeof d.cpu==='number' && d.cpu>90).length;
  const highRam=allGateways.filter(d=>typeof d.ram==='number' && d.ram>90).length;

  const healthClass = health==null ? 'good' : (health>=95?'good':(health>=80?'warn':'bad'));
  const latClass = avgLat==null ? 'good' : (avgLat>500?'bad':(avgLat>100?'warn':'good'));
  const offlineClass = offlineGw>0 ? 'bad' : 'good';
  const unackedClass = unacked>0 ? 'bad' : 'good';
  const cpuClass = highCpu>0 ? 'bad' : 'good';
  const ramClass = highRam>0 ? 'bad' : 'good';

  const gwOnline = allGateways.filter(d=>d.online).length;
  const gwTotal = allGateways.length;
  const apItems = allAps;
  const apOnline = apItems.filter(d=>d.online).length;
  const routerItems = allRoutersSwitches.filter(d=>d.router);
  const routerOnline = routerItems.filter(d=>d.online).length;
  const switchItems = allRoutersSwitches.filter(d=>d.switch);
  const switchOnline = switchItems.filter(d=>d.online).length;

  const summaryHTML = featureEnabled("advanced_metrics") && !featureEnabled("strict_ce")
    ? [
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
    ].join(' ')
    : [
      `<span class="badge good">Devices: ${online}/${total} online</span>`,
      `<span class="badge ${offlineClass}">Offline: ${total - online}</span>`,
      `<span class="badge ${healthClass}">Health: ${health==null?'--':health+'%'}</span>`
    ].join(' ');
  const overallEl=document.getElementById('overallSummary');
  if(overallEl) overallEl.innerHTML=summaryHTML;

  const viewSummaryEl = document.getElementById("viewControlsSummary");
  if(viewSummaryEl){
    const mode = normalizeQuickFilter(deviceQuickFilter);
    const modeLabel = (mode === "online") ? "online only" : ((mode === "offline") ? "offline only" : "all devices");
    const q = String(deviceSearchQuery || "").trim();
    const searchLabel = q ? `search: "${q}"` : "search: none";
    const groupMode = normalizeGroupMode(dashSettings && dashSettings.group_mode);
    const refreshMode = normalizeRefreshInterval(dashSettings && dashSettings.refresh_interval);
    viewSummaryEl.textContent = `Showing ${visibleDevices.length}/${devices.length} devices (${modeLabel}, ${searchLabel}, sort: ${deviceSortLabel(sortMode)}, group: ${deviceGroupLabel(groupMode)}, refresh: ${refreshIntervalLabel(refreshMode)}).`;
  }

  const isDeviceUnackedOffline = (dev, nowSeconds) => {
    if(!dev || dev.online) return false;
    if(!featureEnabled("ack")) return true;
    return !(dev.ack_until && dev.ack_until > nowSeconds);
  };
  const isGatewayAlertEligible = (dev, nowSeconds) => {
    if(!isTabSirenEnabled('gateways')) return false;
    if(!dev || !dev.gateway) return false;
    if(!isDeviceSirenEnabledById(dev.id)) return false;
    return isDeviceUnackedOffline(dev, nowSeconds);
  };
  const isApAlertEligible = (dev, nowSeconds) => {
    if(!isTabSirenEnabled('aps')) return false;
    if(!dev || !dev.ap) return false;
    if(!isDeviceUnackedOffline(dev, nowSeconds)) return false;
    if(!isDeviceSirenEnabledById(dev.id)) return false;
    const offlineSince = typeof dev.offline_since === 'number' ? dev.offline_since : null;
    if(!offlineSince || offlineSince < MIN_OFFLINE_TS) return false; // ignore missing/zero timestamps
    return offlineSince <= (nowSeconds - AP_ALERT_GRACE_SEC);
  };
  const isRouterSwitchAlertEligible = (dev, nowSeconds) => {
    if(!isTabSirenEnabled('routers')) return false;
    if(!dev || dev.gateway || dev.ap || !(dev.router || dev.switch)) return false;
    if(!isDeviceSirenEnabledById(dev.id)) return false;
    return isDeviceUnackedOffline(dev, nowSeconds);
  };

  const shouldAlert = devices.some(d =>
    isGatewayAlertEligible(d, nowSec) ||
    isApAlertEligible(d, nowSec) ||
    isRouterSwitchAlertEligible(d, nowSec)
  );

  if(shouldAlert){
    if(!sirenShouldAlertPrev){
      clearTimeout(sirenTimeout); sirenTimeout=null;
      sirenNextDelayMs=30000;
    }
    if(!sirenTimeout){
      sirenTimeout=setTimeout(()=>{
            const nowSeconds = Math.floor(Date.now()/1000);
            const stillAlert = devicesCache.some(d =>
              isGatewayAlertEligible(d, nowSeconds) ||
              isApAlertEligible(d, nowSeconds) ||
              isRouterSwitchAlertEligible(d, nowSeconds)
            );
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
  updateDataHealthBanner();
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
  if(!featureEnabled("ack")) return;
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
  if(!featureEnabled("ack")) return;
  touchMutation();
  const dev = devicesCache.find(x=>x.id===id);
  if(dev && dev.ack_until){
    dev.ack_until=null;
    renderDevices();
  }
  fetch(`?ajax=clear&id=${id}&t=${Date.now()}`).then(()=>fetchDevices());
}
function simulate(id){
  if(!featureEnabled("simulate")) return;
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
  if(!featureEnabled("simulate")) return;
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
  if(!featureEnabled("ack")) return;
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
  if(!featureEnabled("history")) return;
  const params=new URLSearchParams({view:'device',id});
  if(name){ params.set('name',name); }
  window.location.href='?'+params.toString();
}
initDisplaySettings();
loadUserPrefsFromServer().finally(()=>{ ensureFirstRunSourceWizard(); });
loadSourceStatus(true);
fetchInventoryOverview(true);
loadTopology(true);
schedulePoll(0);
setInterval(()=>{ loadSourceStatus(false); }, SOURCE_STATUS_REFRESH_MS);

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
  if(!featureEnabled("advanced_metrics") || featureEnabled("strict_ce")){
    return;
  }
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
  const next = prompt('Enter new password (min 8 chars)');
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
function manageUispSources(){
  window.location.href='?view=settings';
}
function manageUispToken(){
  // Backward compatibility for older button wiring.
  manageUispSources();
}
function logout(){
  window.location.href='?action=logout';
}

function buildMetricBadges(device, latencyVal){
  if(!featureEnabled("advanced_metrics") || featureEnabled("strict_ce")){
    return "";
  }
  const badges=[];
  if(isMetricEnabled("cpu")) badges.push(badgeVal(device.cpu, "CPU", "%"));
  if(isMetricEnabled("ram")) badges.push(badgeVal(device.ram, "RAM", "%"));
  if(isMetricEnabled("temp")) badges.push(badgeVal(device.temp, "Temp", "&deg;C"));
  if(isMetricEnabled("latency")) badges.push(badgeLatency(latencyVal));
  return badges.join(" ");
}

function buildLiveStateBadges(device){
  if(!featureEnabled("advanced_metrics") || featureEnabled("strict_ce")){
    return "";
  }
  let html = "";
  if(isMetricEnabled("uptime")){
    html += `<span class="badge good live-uptime" data-uptime="${device.uptime??''}"></span>`;
  }
  if(isMetricEnabled("outage")){
    html += `<span class="badge bad live-outage" data-offline-since="${device.offline_since??''}"></span>`;
  }
  return html;
}

function titleCaseLabel(value){
  const raw = String(value || "").trim();
  if(!raw) return "";
  return raw
    .replace(/[_-]+/g, " ")
    .replace(/\s+/g, " ")
    .split(" ")
    .map(part => part ? (part[0].toUpperCase() + part.slice(1).toLowerCase()) : "")
    .join(" ");
}

function getRoleGroupLabel(device, tabKey){
  if(tabKey === "gateways") return "Gateway";
  if(tabKey === "aps") return "Access Point";
  if(device && device.router) return "Router";
  if(device && device.switch) return "Switch";
  const fallback = titleCaseLabel(device && device.role);
  return fallback || "Other";
}

function getDeviceGroupLabel(device, tabKey, mode){
  const groupMode = normalizeGroupMode(mode);
  if(groupMode === "role"){
    return getRoleGroupLabel(device, tabKey);
  }
  if(groupMode === "site"){
    const site = String(
      (device && (device.site || device.site_name || device.site_id)) || "Unassigned Site"
    ).trim();
    return site || "Unassigned Site";
  }
  return "All Devices";
}

function renderGroupedCards(items, tabKey, renderCard){
  const list = Array.isArray(items) ? items : [];
  if(!list.length){
    return "";
  }
  const groupMode = normalizeGroupMode(dashSettings && dashSettings.group_mode);
  if(groupMode === "none"){
    return list.map(renderCard).join("");
  }
  const groups = [];
  const byKey = new Map();
  list.forEach(device=>{
    const label = getDeviceGroupLabel(device, tabKey, groupMode);
    const key = String(label || "").toLowerCase();
    if(!byKey.has(key)){
      const group = { label: label || "Other", items: [] };
      byKey.set(key, group);
      groups.push(group);
    }
    byKey.get(key).items.push(device);
  });
  groups.sort((a,b)=>String(a.label).localeCompare(String(b.label), undefined, { sensitivity: "base" }));
  return groups.map(group=>`<section class="device-group" data-group-mode="${groupMode}">
      <div class="device-group-header">${escapeHtml(group.label)}</div>
      <div class="device-group-grid">${group.items.map(renderCard).join("")}</div>
    </section>`).join("");
}

function renderGatewayGrid(gws, nowSec, nowMs){
  const gateGrid=document.getElementById('gateGrid');
  if(!gateGrid) return;
  const renderCard = d=>{
    const draggable = featureEnabled("advanced_actions") ? "true" : "false";
    const badges = buildMetricBadges(d, d.latency);
    const inventoryBadges = featureEnabled("inventory") ? getInventoryCardBadges(d) : "";
    const ackActive = d.ack_until && d.ack_until > nowSec;
    const deviceSirenEnabled = isDeviceSirenEnabledById(d.id);
    const minimalMode = featureEnabled("strict_ce");
    const topMeta = renderSiteAndLastSeen(d);
    const statusColor = d.online?'#b06cff':'#f55';
    const transitionClass = getDeviceTransitionClass(d, nowMs);
    if(minimalMode){
      return `<div class="card ${d.online?'':'offline'} ${transitionClass}" draggable="${draggable}" data-card-id="${d.id}" data-card-tab="gateways">
        <h2>${d.name}</h2>
        <div class="role-label">Gateway</div>
        <div class="status" style="color:${statusColor}">${d.online?'ONLINE':'OFFLINE'}</div>
        ${topMeta}
      </div>`;
    }
    return `<div class="card ${d.online?'':'offline'} ${ackActive?'acked':''} ${transitionClass}" draggable="${draggable}" data-card-id="${d.id}" data-card-tab="gateways">
      <div class="ack-badge">${badgeAck(d.ack_until)}${buildLiveStateBadges(d)}</div>
      <h2>${d.name}</h2>
      <div class="role-label">Gateway</div>
      <div class="status" style="color:${statusColor}">${d.online?'ONLINE':'OFFLINE'}</div>
      ${topMeta}
      <div>${badges} ${inventoryBadges}</div>
      <div class="actions">
        ${featureEnabled("ack") && !d.online ? `
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
        ${featureEnabled("simulate") ? (d.simulate ? `<button onclick="clearSim('${d.id}')">End Test</button>` : (d.online ? `<button onclick="simulate('${d.id}')">Test Outage</button>` : '')) : ''}
        ${featureEnabled("advanced_actions") ? `<button onclick="toggleDeviceSiren('${d.id}')">${deviceSirenEnabled ? 'Siren: On' : 'Siren: Off'}</button>` : ''}
        ${featureEnabled("inventory") ? `<button onclick="openInventory('${d.id}','${d.name}')">Inventory</button>` : ''}
        ${featureEnabled("history") ? `<button onclick="showHistory('${d.id}','${d.name}')">History</button>` : ''}
      </div>
    </div>`;
  };
  const html = renderGroupedCards(gws, "gateways", renderCard);
  gateGrid.innerHTML = html;
}

function renderApGrid(items, nowSec, nowMs){
  const apGrid=document.getElementById('apGrid');
  if(!apGrid) return;
  const renderCard = d=>{
    const draggable = featureEnabled("advanced_actions") ? "true" : "false";
    const latencyVal = (typeof d.latency==='number' && isFinite(d.latency)) ? d.latency : d.cpe_latency;
    const badges = buildMetricBadges(d, latencyVal);
    const inventoryBadges = featureEnabled("inventory") ? getInventoryCardBadges(d) : "";
    const ackActive = d.ack_until && d.ack_until > nowSec;
    const deviceSirenEnabled = isDeviceSirenEnabledById(d.id);
    const minimalMode = featureEnabled("strict_ce");
    const topMeta = renderSiteAndLastSeen(d);
    const statusColor = d.online?'#b06cff':'#f55';
    const transitionClass = getDeviceTransitionClass(d, nowMs);
    if(minimalMode){
      return `<div class="card ${d.online?'':'offline'} ${transitionClass}" draggable="${draggable}" data-card-id="${d.id}" data-card-tab="aps">
        <h2>${d.name}</h2>
        <div class="role-label">Access Point</div>
        <div class="status" style="color:${statusColor}">${d.online?'ONLINE':'OFFLINE'}</div>
        ${topMeta}
      </div>`;
    }
    const actions = `
        ${featureEnabled("ack") && !d.online ? `
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
        ${featureEnabled("simulate") ? (d.simulate ? `<button onclick="clearSim('${d.id}')">End Test</button>` : (d.online ? `<button onclick="simulate('${d.id}')">Test Outage</button>` : '')) : ''}
        ${featureEnabled("advanced_actions") ? `<button onclick="toggleDeviceSiren('${d.id}')">${deviceSirenEnabled ? 'Siren: On' : 'Siren: Off'}</button>` : ''}
        ${featureEnabled("inventory") ? `<button onclick="openInventory('${d.id}','${d.name}')">Inventory</button>` : ''}
        ${featureEnabled("history") ? `<button onclick="showHistory('${d.id}','${d.name}')">History</button>` : ''}
    `;
    return `<div class="card ${d.online?'':'offline'} ${ackActive?'acked':''} ${transitionClass}" draggable="${draggable}" data-card-id="${d.id}" data-card-tab="aps">
      <div class="ack-badge">${badgeAck(d.ack_until)}${buildLiveStateBadges(d)}</div>
      <h2>${d.name}</h2>
      <div class="role-label">Access Point</div>
      <div class="status" style="color:${statusColor}">${d.online?'ONLINE':'OFFLINE'}</div>
      ${topMeta}
      <div>${badges} ${inventoryBadges}</div>
      <div class="actions">
        ${actions}
      </div>
    </div>`;
  };
  const html = renderGroupedCards(items, "aps", renderCard);
  apGrid.innerHTML = html;
}

function renderRouterSwitchGrid(backbones, nowSec, nowMs){
  const routerGrid=document.getElementById('routerGrid');
  if(!routerGrid) return;
  const renderCard = d=>{
    const draggable = featureEnabled("advanced_actions") ? "true" : "false";
    const latencyVal = (typeof d.latency==='number' && isFinite(d.latency)) ? d.latency : d.cpe_latency;
    const badges = buildMetricBadges(d, latencyVal);
    const inventoryBadges = featureEnabled("inventory") ? getInventoryCardBadges(d) : "";
    const ackActive = d.ack_until && d.ack_until > nowSec;
    const deviceSirenEnabled = isDeviceSirenEnabledById(d.id);
    const roleLabel = d.router ? 'Router' : 'Switch';
    const minimalMode = featureEnabled("strict_ce");
    const topMeta = renderSiteAndLastSeen(d);
    const statusColor = d.online?'#b06cff':'#f55';
    const transitionClass = getDeviceTransitionClass(d, nowMs);
    if(minimalMode){
      return `<div class="card ${d.online?'':'offline'} ${transitionClass}" draggable="${draggable}" data-card-id="${d.id}" data-card-tab="routers">
        <h2>${d.name}</h2>
        <div class="role-label">${roleLabel}</div>
        <div class="status" style="color:${statusColor}">${d.online?'ONLINE':'OFFLINE'}</div>
        ${topMeta}
      </div>`;
    }
    return `<div class="card ${d.online?'':'offline'} ${ackActive?'acked':''} ${transitionClass}" draggable="${draggable}" data-card-id="${d.id}" data-card-tab="routers">
      <div class="ack-badge">${badgeAck(d.ack_until)}${buildLiveStateBadges(d)}</div>
      <h2>${d.name}</h2>
      <div class="role-label">${roleLabel}</div>
      <div class="status" style="color:${statusColor}">${d.online?'ONLINE':'OFFLINE'}</div>
      ${topMeta}
      <div>${badges} ${inventoryBadges}</div>
      <div class="actions">
        ${featureEnabled("ack") && !d.online ? `
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
        ${featureEnabled("simulate") ? (d.simulate ? `<button onclick="clearSim('${d.id}')">End Test</button>` : (d.online ? `<button onclick="simulate('${d.id}')">Test Outage</button>` : '')) : ''}
        ${featureEnabled("advanced_actions") ? `<button onclick="toggleDeviceSiren('${d.id}')">${deviceSirenEnabled ? 'Siren: On' : 'Siren: Off'}</button>` : ''}
        ${featureEnabled("inventory") ? `<button onclick="openInventory('${d.id}','${d.name}')">Inventory</button>` : ''}
        ${featureEnabled("history") ? `<button onclick="showHistory('${d.id}','${d.name}')">History</button>` : ''}
      </div>
    </div>`;
  };
  const html = renderGroupedCards(backbones, "routers", renderCard);
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
  if(!featureEnabled("cpe_history")) return;
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
  const key = String(ev.key || "").toLowerCase();
  if(key === 'escape'){
    const modal=document.getElementById('cpeHistoryModal');
    if(modal && modal.style.display==='block'){
      closeCpeHistory();
    }
    const invModal=document.getElementById('inventoryModal');
    if(invModal && invModal.style.display==='block'){
      closeInventory();
    }
    const wizardModal=document.getElementById('setupWizardModal');
    if(wizardModal && wizardModal.style.display==='block'){
      hideSetupWizard();
    }
    closeShortcuts();
    return;
  }
  if(isTypingContext(ev.target)){
    return;
  }
  if(key === '?' || (key === '/' && ev.shiftKey)){
    ev.preventDefault();
    openShortcuts();
    return;
  }
  if(key === 'k'){
    ev.preventDefault();
    toggleKioskMode();
    return;
  }
  if(key === '/'){
    ev.preventDefault();
    const search = document.getElementById("deviceSearchInput");
    if(search){
      search.focus();
      search.select();
    }
    return;
  }
  if(key === 'g'){
    setActiveTab("gateways", { persist: true });
    return;
  }
  if(key === 'a'){
    setActiveTab("aps", { persist: true });
    return;
  }
  if(key === 'r'){
    setActiveTab("routers", { persist: true });
    return;
  }
});



