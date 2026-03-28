const { spawn } = require('child_process');
const fs = require('fs');

function sleep(ms) { return new Promise(resolve => setTimeout(resolve, ms)); }

async function connectCDP() {
  const chrome = spawn('/opt/google/chrome/chrome', [
    '--headless=new',
    '--disable-gpu',
    '--no-sandbox',
    '--disable-setuid-sandbox',
    '--remote-debugging-port=9222',
    '--user-data-dir=/tmp/cdt-profile-bigops',
    'about:blank',
  ], { stdio: 'ignore' });

  let version;
  for (let i = 0; i < 30; i++) {
    try {
      const res = await fetch('http://127.0.0.1:9222/json/version');
      if (res.ok) {
        version = await res.json();
        break;
      }
    } catch {}
    await sleep(300);
  }
  if (!version?.webSocketDebuggerUrl) {
    chrome.kill('SIGTERM');
    throw new Error('Chrome DevTools remote debugging 未启动');
  }

  const ws = new WebSocket(version.webSocketDebuggerUrl);
  const pending = new Map();
  const events = [];
  let id = 0;

  ws.onmessage = event => {
    const msg = JSON.parse(event.data.toString());
    if (msg.id) {
      const resolver = pending.get(msg.id);
      if (!resolver) return;
      pending.delete(msg.id);
      if (msg.error) resolver.reject(new Error(JSON.stringify(msg.error)));
      else resolver.resolve(msg.result);
      return;
    }
    events.push(msg);
  };
  await new Promise((resolve, reject) => {
    ws.onopen = resolve;
    ws.onerror = reject;
  });

  function send(method, params = {}, sessionId) {
    const callId = ++id;
    const payload = { id: callId, method, params };
    if (sessionId) payload.sessionId = sessionId;
    return new Promise((resolve, reject) => {
      pending.set(callId, { resolve, reject });
      ws.send(JSON.stringify(payload));
    });
  }

  return { chrome, ws, events, send };
}

function drainEvents(events, sessionId) {
  const consoleErrors = [];
  const jsExceptions = [];
  const failedResponses = [];
  while (events.length) {
    const evt = events.shift();
    if (evt.sessionId && evt.sessionId !== sessionId) continue;
    if (evt.method === 'Runtime.consoleAPICalled' && evt.params?.type === 'error') {
      const text = (evt.params.args || []).map(arg => arg.value || arg.description || '').join(' ');
      consoleErrors.push(text);
    }
    if (evt.method === 'Runtime.exceptionThrown') {
      jsExceptions.push(evt.params?.exceptionDetails?.text || 'exception');
    }
    if (evt.method === 'Network.responseReceived') {
      const status = evt.params?.response?.status;
      const url = evt.params?.response?.url;
      if (status >= 400) failedResponses.push(`${status} ${url}`);
    }
  }
  return { consoleErrors, jsExceptions, failedResponses };
}

async function evalJS(send, sessionId, expression) {
  const result = await send('Runtime.evaluate', {
    expression,
    awaitPromise: true,
    returnByValue: true,
  }, sessionId);
  return result.result?.value;
}

async function waitFor(fn, check, timeoutMs = 15000, interval = 300) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    const value = await fn();
    if (check(value)) return value;
    await sleep(interval);
  }
  throw new Error('waitFor timeout');
}

(async () => {
  const { chrome, ws, events, send } = await connectCDP();
  const target = await send('Target.createTarget', { url: 'about:blank' });
  const attach = await send('Target.attachToTarget', { targetId: target.targetId, flatten: true });
  const sessionId = attach.sessionId;

  await send('Page.enable', {}, sessionId);
  await send('Runtime.enable', {}, sessionId);
  await send('Network.enable', {}, sessionId);
  await send('Page.navigate', { url: 'http://127.0.0.1:5173/login' }, sessionId);

  await waitFor(() => evalJS(send, sessionId, 'document.readyState'), value => value === 'complete', 10000, 200);
  await sleep(1200);

  const loginInfo = await evalJS(send, sessionId, `(() => ({
    href: location.href,
    placeholders: [...document.querySelectorAll('input')].map(input => input.getAttribute('placeholder')),
    buttons: [...document.querySelectorAll('button')].map(btn => btn.innerText.trim()).filter(Boolean),
  }))()`);

  const fillResult = await evalJS(send, sessionId, `(() => {
    const setValue = (selector, value) => {
      const el = document.querySelector(selector);
      if (!el) return false;
      el.focus();
      el.value = value;
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
      return true;
    };
    const okUser = setValue('input[placeholder="用户名"]', 'admin');
    const okPass = setValue('input[placeholder="密码"]', '8R{pd4Dwv');
    const btn = [...document.querySelectorAll('button')].find(b => b.innerText && b.innerText.includes('登录'));
    if (btn) btn.click();
    return { okUser, okPass, hasButton: !!btn };
  })()`);

  const afterLoginUrl = await waitFor(
    () => evalJS(send, sessionId, 'location.href'),
    value => value && !String(value).endsWith('/login')
  );

  const routeInfo = await evalJS(send, sessionId, `(() => {
    const app = document.querySelector('#app')?.__vue_app__;
    const router = app?.config?.globalProperties?.$router;
    if (!router) return { hasRouter: false };
    return {
      hasRouter: true,
      current: router.currentRoute.value.fullPath,
      routes: router.getRoutes().map(r => ({ name: r.name, path: r.path })).filter(r => String(r.path).includes('monitor') || String(r.name).includes('Monitor') || String(r.name).includes('Alert')),
    };
  })()`);

  await evalJS(send, sessionId, `(() => {
    const app = document.querySelector('#app')?.__vue_app__;
    app?.config?.globalProperties?.$router?.push('/monitor/dashboard');
    return true;
  })()`);
  await sleep(1800);
  const dashboardUrl = await evalJS(send, sessionId, 'location.href');
  const dashboardBody = await evalJS(send, sessionId, 'document.body.innerText.slice(0, 3000)');

  await evalJS(send, sessionId, `(() => {
    const app = document.querySelector('#app')?.__vue_app__;
    app?.config?.globalProperties?.$router?.push('/monitor/alert-rules');
    return true;
  })()`);
  await sleep(1800);
  const alertUrl = await evalJS(send, sessionId, 'location.href');
  const alertBody = await evalJS(send, sessionId, 'document.body.innerText.slice(0, 3000)');

  const capture = await send('Page.captureScreenshot', { format: 'png' }, sessionId);
  fs.writeFileSync('/tmp/chrome-devtools-smoke.png', Buffer.from(capture.data, 'base64'));

  const eventSummary = drainEvents(events, sessionId);
  const result = {
    loginInfo,
    fillResult,
    afterLoginUrl,
    routeInfo,
    dashboardUrl,
    dashboardHits: ['监控中心', 'Agent 实时列表', '最近告警', '热点 Top'].filter(text => dashboardBody.includes(text)),
    alertUrl,
    alertHits: ['告警规则', '告警事件', '立即巡检', '新增规则'].filter(text => alertBody.includes(text)),
    ...eventSummary,
  };

  fs.writeFileSync('/tmp/chrome-devtools-smoke.json', JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));

  ws.close();
  chrome.kill('SIGTERM');
})().catch(err => {
  console.error(err.stack);
  process.exit(1);
});
