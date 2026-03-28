const { spawn } = require('child_process');
const fs = require('fs');

function sleep(ms) {
  return new Promise(resolve => setTimeout(resolve, ms));
}

async function connectCDP() {
  const chrome = spawn('/opt/google/chrome/chrome', [
    '--headless=new',
    '--disable-gpu',
    '--no-sandbox',
    '--disable-setuid-sandbox',
    '--remote-debugging-port=9236',
    '--user-data-dir=/tmp/cdt-profile-dashboard-check',
    'about:blank',
  ], { stdio: 'ignore' });

  let version;
  for (let i = 0; i < 40; i++) {
    try {
      const res = await fetch('http://127.0.0.1:9236/json/version');
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
    if (!msg.id) {
      events.push(msg);
      return;
    }
    const resolver = pending.get(msg.id);
    if (!resolver) return;
    pending.delete(msg.id);
    if (msg.error) resolver.reject(new Error(JSON.stringify(msg.error)));
    else resolver.resolve(msg.result);
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
  const failedResponses = [];
  while (events.length) {
    const evt = events.shift();
    if (evt.sessionId && evt.sessionId !== sessionId) continue;
    if (evt.method === 'Runtime.consoleAPICalled' && evt.params?.type === 'error') {
      consoleErrors.push((evt.params.args || []).map(arg => arg.value || arg.description || '').join(' '));
    }
    if (evt.method === 'Network.responseReceived') {
      const status = evt.params?.response?.status;
      const url = evt.params?.response?.url;
      if (status >= 400) failedResponses.push(`${status} ${url}`);
    }
  }
  return { consoleErrors, failedResponses };
}

async function evalJS(send, sessionId, expression) {
  const result = await send('Runtime.evaluate', {
    expression,
    awaitPromise: true,
    returnByValue: true,
  }, sessionId);
  return result.result?.value;
}

async function login(send, sessionId, username) {
  await send('Page.navigate', { url: 'http://127.0.0.1:5173/login' }, sessionId);
  await sleep(1200);
  await evalJS(send, sessionId, `(() => {
    localStorage.clear();
    sessionStorage.clear();
    const setValue = (selector, value) => {
      const el = document.querySelector(selector);
      if (!el) return false;
      const setter = Object.getOwnPropertyDescriptor(window.HTMLInputElement.prototype, 'value').set;
      setter.call(el, value);
      el.dispatchEvent(new Event('input', { bubbles: true }));
      el.dispatchEvent(new Event('change', { bubbles: true }));
      return true;
    };
    setValue('input[placeholder="用户名"]', ${JSON.stringify(username)});
    setValue('input[placeholder="密码"]', '8R{pd4Dwv');
    const btn = [...document.querySelectorAll('button')].find(b => b.innerText && b.innerText.includes('登录'));
    if (btn) btn.dispatchEvent(new MouseEvent('click', { bubbles: true }));
    return !!btn;
  })()`);
  await sleep(2800);
}

(async () => {
  const { chrome, ws, events, send } = await connectCDP();
  const target = await send('Target.createTarget', { url: 'about:blank' });
  const attach = await send('Target.attachToTarget', { targetId: target.targetId, flatten: true });
  const sessionId = attach.sessionId;

  await send('Page.enable', {}, sessionId);
  await send('Runtime.enable', {}, sessionId);
  await send('Network.enable', {}, sessionId);

  await login(send, sessionId, 'admin');
  const adminBody = await evalJS(send, sessionId, 'document.body.innerText');

  await login(send, sessionId, 'yunwei');
  const userBody = await evalJS(send, sessionId, 'document.body.innerText');

  const result = {
    admin: {
      hasWorkbench: adminBody.includes('个人工作台'),
      hasOverview: adminBody.includes('平台总览'),
      hasUsersCard: adminBody.includes('平台用户'),
      hasCloudCard: adminBody.includes('云账号'),
    },
    yunwei: {
      hasWorkbench: userBody.includes('个人工作台'),
      hasOverview: userBody.includes('平台总览'),
      hasUsersCard: userBody.includes('平台用户'),
      hasCloudCard: userBody.includes('云账号'),
      hasAssetsCard: userBody.includes('主机资产'),
      hasTicketLaunch: userBody.includes('发起工单'),
    },
    ...drainEvents(events, sessionId),
  };

  const capture = await send('Page.captureScreenshot', { format: 'png' }, sessionId);
  fs.writeFileSync('/tmp/chrome-devtools-dashboard-check.png', Buffer.from(capture.data, 'base64'));
  fs.writeFileSync('/tmp/chrome-devtools-dashboard-check.json', JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));
  ws.close();
  chrome.kill('SIGTERM');
})().catch(err => {
  console.error(err.stack);
  process.exit(1);
});
