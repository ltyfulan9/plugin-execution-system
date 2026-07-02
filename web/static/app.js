let token = localStorage.getItem('token') || '';
let plugins = [];

function headers(extra = {}) {
  const h = {'Content-Type': 'application/json', ...extra};
  if (token) h.Authorization = 'Bearer ' + token;
  return h;
}
async function api(path, opts = {}) {
  const res = await fetch(path, {...opts, headers: headers(opts.headers || {})});
  const body = await res.json();
  if (!res.ok) throw new Error(body.message || body.code);
  return body.data;
}
async function login(t) {
  token = t; localStorage.setItem('token', token);
  const me = await api('/api/auth/me');
  document.getElementById('user').textContent = `${me.username} / ${me.role}`;
  await loadPlugins(); await loadExecutions();
}
async function loadPlugins() {
  plugins = await api('/api/plugins');
  const html = plugins.map(p => `<div class="row"><label><input type="checkbox" value="${p.id}" ${p.status==='Enabled'?'':'disabled'}> ${p.name}@${p.version}</label><span class="tag ${p.status}">${p.status}</span><button onclick="enablePlugin('${p.id}')">Enable</button><button onclick="disablePlugin('${p.id}')">Disable</button></div>`).join('');
  document.getElementById('plugins').innerHTML = html || '暂无插件';
}
async function reloadPlugins() { await api('/api/plugins/reload', {method:'POST'}); await loadPlugins(); }
async function enablePlugin(id) { await api(`/api/plugins/${id}/enable`, {method:'POST'}); await loadPlugins(); }
async function disablePlugin(id) { await api(`/api/plugins/${id}/disable`, {method:'POST'}); await loadPlugins(); }
async function createExecution() {
  const ids = [...document.querySelectorAll('#plugins input:checked')].map(x=>x.value);
  const input = JSON.parse(document.getElementById('input').value);
  const idem = document.getElementById('idem').value.trim();
  const data = await api('/api/executions', {method:'POST', headers: idem ? {'Idempotency-Key': idem} : {}, body: JSON.stringify({plugin_ids: ids, input})});
  await loadExecutions(); pollExecution(data.id);
}
async function loadExecutions() {
  const items = await api('/api/executions');
  document.getElementById('executions').innerHTML = items.map(e => `<div class="row"><span>${e.id}</span><span class="tag ${e.status}">${e.status}</span><button onclick="showResults('${e.id}')">结果</button></div>`).join('') || '暂无任务';
}
async function showResults(id) {
  const summary = await api(`/api/executions/${id}/summary`);
  const results = await api(`/api/executions/${id}/results`);
  document.getElementById('results').textContent = JSON.stringify({summary, results}, null, 2);
}
async function pollExecution(id) {
  for (let i=0;i<20;i++) {
    const e = await api(`/api/executions/${id}`);
    if (['Success','PartialSuccess','Failed','Timeout','Canceled'].includes(e.status)) { await loadExecutions(); await showResults(id); return; }
    await new Promise(r => setTimeout(r, 500));
  }
}
if (token) login(token).catch(console.error);
