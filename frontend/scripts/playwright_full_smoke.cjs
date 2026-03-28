const { chromium } = require('/root/.npm/_npx/9833c18b2d85bc59/node_modules/playwright');
const fs = require('fs');

const BASE_URL = 'http://127.0.0.1:5173';
const USERNAME = 'admin';
const PASSWORD = '8R{pd4Dwv';

function flattenMenus(items, acc = []) {
  for (const item of items || []) {
    if (item.type !== 3 && item.visible !== 0 && item.path && item.component) {
      acc.push({
        name: item.name,
        title: item.title,
        path: item.path,
        component: item.component,
      });
    }
    if (item.children?.length) {
      flattenMenus(item.children, acc);
    }
  }
  return acc;
}

function sliceText(text) {
  return (text || '').replace(/\s+/g, ' ').trim().slice(0, 500);
}

(async () => {
  const browser = await chromium.launch({
    headless: true,
    executablePath: '/opt/google/chrome/chrome',
    chromiumSandbox: false,
  });
  const page = await browser.newPage({ viewport: { width: 1440, height: 960 } });

  const globalConsoleErrors = [];
  page.on('console', msg => {
    if (msg.type() === 'error') {
      globalConsoleErrors.push(msg.text());
    }
  });

  await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });
  await page.getByPlaceholder('用户名').fill(USERNAME);
  await page.getByPlaceholder('密码').fill(PASSWORD);
  await page.getByRole('button', { name: '登录' }).click();
  await page.waitForURL(url => !url.pathname.endsWith('/login'), { timeout: 15000 });
  await page.waitForTimeout(1200);

  const menuResponse = await page.evaluate(async () => {
    const token = localStorage.getItem('token');
    const response = await fetch('/api/v1/menus/user', {
      headers: { Authorization: `Bearer ${token}` },
    });
    return await response.json();
  });

  const routes = flattenMenus(menuResponse.data || []);
  const results = [];

  for (const route of routes) {
    const routeConsoleErrors = [];
    const routeFailedResponses = [];
    const consoleListener = msg => {
      if (msg.type() === 'error') routeConsoleErrors.push(msg.text());
    };
    const responseListener = res => {
      if (res.status() >= 400) routeFailedResponses.push(`${res.status()} ${res.url()}`);
    };
    page.on('console', consoleListener);
    page.on('response', responseListener);

    try {
      await page.goto(`${BASE_URL}${route.path}`, { waitUntil: 'networkidle', timeout: 20000 });
      await page.waitForTimeout(1000);
      const text = await page.locator('body').innerText();
      const title = await page.title();
      results.push({
        ...route,
        ok: !page.url().endsWith('/404'),
        finalUrl: page.url(),
        titleText: title,
        bodySample: sliceText(text),
        consoleErrors: routeConsoleErrors.slice(0, 10),
        failedResponses: routeFailedResponses.slice(0, 10),
      });
    } catch (error) {
      results.push({
        ...route,
        ok: false,
        finalUrl: page.url(),
        titleText: '',
        bodySample: '',
        consoleErrors: routeConsoleErrors.slice(0, 10),
        failedResponses: routeFailedResponses.slice(0, 10),
        error: error.message,
      });
    } finally {
      page.off('console', consoleListener);
      page.off('response', responseListener);
    }
  }

  await page.goto(`${BASE_URL}/monitor/dashboard`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1200);
  await page.screenshot({ path: '/tmp/playwright-full-smoke-monitor.png', fullPage: true });

  const summary = {
    totalRoutes: results.length,
    passedRoutes: results.filter(item => item.ok && item.consoleErrors.length === 0 && item.failedResponses.length === 0).length,
    failedRoutes: results.filter(item => !item.ok || item.consoleErrors.length || item.failedResponses.length).length,
    globalConsoleErrors: globalConsoleErrors.slice(0, 20),
  };

  fs.writeFileSync('/tmp/playwright-full-smoke.json', JSON.stringify({ summary, results }, null, 2));
  console.log(JSON.stringify({ summary, results }, null, 2));

  await browser.close();
})();
