const { chromium } = require('/root/.npm/_npx/9833c18b2d85bc59/node_modules/playwright');
const fs = require('fs');

const BASE_URL = 'http://127.0.0.1:5173';
const USERNAME = 'admin';
const PASSWORD = '8R{pd4Dwv';

async function login(page) {
  await page.goto(`${BASE_URL}/login`, { waitUntil: 'networkidle' });
  await page.getByPlaceholder('用户名').fill(USERNAME);
  await page.getByPlaceholder('密码').fill(PASSWORD);
  await page.getByRole('button', { name: '登录' }).click();
  await page.waitForURL(url => !url.pathname.endsWith('/login'), { timeout: 15000 });
  await page.waitForTimeout(1200);
}

(async () => {
  const browser = await chromium.launch({
    headless: true,
    executablePath: '/opt/google/chrome/chrome',
    chromiumSandbox: false,
  });

  const page = await browser.newPage({ viewport: { width: 1440, height: 960 } });
  const result = {
    checks: {},
    consoleErrors: [],
    failedResponses: [],
  };

  page.on('console', msg => {
    if (msg.type() === 'error') {
      result.consoleErrors.push(msg.text());
    }
  });
  page.on('response', res => {
    if (res.status() >= 400) {
      result.failedResponses.push(`${res.status()} ${res.url()}`);
    }
  });

  await login(page);

  await page.goto(`${BASE_URL}/monitor/dashboard`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  let text = await page.locator('body').innerText();
  result.checks.dashboard = {
    ok: ['监控中心', 'Agent 实时列表', '最近告警'].every(item => text.includes(item)),
  };

  await page.goto(`${BASE_URL}/monitor/datasources`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  text = await page.locator('body').innerText();
  result.checks.datasources = {
    ok: text.includes('监控数据源'),
    hasMockProm: text.includes('mock-prom'),
  };

  await page.goto(`${BASE_URL}/monitor/query`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  await page.locator('textarea').first().fill('up');
  await page.getByRole('button', { name: '执行查询' }).click();
  await page.waitForTimeout(1200);
  text = await page.locator('body').innerText();
  result.checks.query = {
    ok: text.includes('PromQL 查询台'),
    hasResult: text.includes('demo:9100') || text.includes('node_load1'),
  };

  await page.goto(`${BASE_URL}/monitor/alerts`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  text = await page.locator('body').innerText();
  result.checks.alertEvents = {
    ok: text.includes('告警事件中心'),
    hasBatchAck: text.includes('批量确认'),
    hasRows: text.includes('disk-task-auto') || text.includes('disk-ticket-auto'),
  };

  await page.goto(`${BASE_URL}/monitor/agents/81fa4fbb-8a2b-4938-a368-e46ff57679c5`, { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);
  text = await page.locator('body').innerText();
  result.checks.agentDetail = {
    ok: text.includes('Agent 详情'),
    hasRecentAlerts: text.includes('最近告警'),
    hasRecentExecutions: text.includes('最近任务执行'),
  };

  await page.screenshot({ path: '/tmp/monitor-v3-playwright.png', fullPage: true });
  fs.writeFileSync('/tmp/monitor-v3-playwright.json', JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));

  await browser.close();
})().catch(err => {
  console.error(err.stack);
  process.exit(1);
});
