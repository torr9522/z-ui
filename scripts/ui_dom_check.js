#!/usr/bin/env node

const fs = require('fs');
const path = require('path');
const { chromium } = require('playwright');

function requiredEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`missing required env: ${name}`);
  }
  return value;
}

async function textVisible(page, selector, text) {
  const locator = page.locator(selector).filter({ hasText: text });
  return await locator.count() > 0;
}

async function switchChecked(locator) {
  const aria = await locator.getAttribute('aria-checked');
  if (aria === 'true') {
    return true;
  }
  if (aria === 'false') {
    return false;
  }
  const klass = (await locator.getAttribute('class')) || '';
  return klass.includes('ant-switch-checked');
}

async function anyModalText(page) {
  return (await page.locator('#inbound-modal').textContent()) || '';
}

async function openAddModal(page) {
  await page.locator('button.ant-btn-primary').first().click();
  await page.waitForTimeout(300);
}

async function closeModal(page) {
  await page.locator('#inbound-modal .ant-modal-close').click();
  await page.waitForTimeout(200);
}

async function chooseProtocol(page, label) {
  await page.locator('#inbound-modal .ant-select').first().click();
  await page.waitForTimeout(200);
  const options = page.locator('.ant-select-dropdown-menu-item');
  const count = await options.count();
  for (let i = 0; i < count; i++) {
    const text = ((await options.nth(i).textContent()) || '').trim().toLowerCase();
    if (text === label.trim().toLowerCase()) {
      await options.nth(i).click();
      await page.waitForTimeout(300);
      return;
    }
  }
  throw new Error(`protocol option not found: ${label}`);
}

async function setSocksPasswordAuth(page, enabled) {
  const switchButton = page.locator('#inbound-modal .ant-form-item').filter({ hasText: '密码认证' }).locator('button').first();
  const checked = await switchChecked(switchButton);
  if (checked !== enabled) {
    await switchButton.click();
    await page.waitForTimeout(200);
  }
}

async function setHttpPasswordAuth(page, enabled) {
  const switchButton = page.locator('#inbound-modal .ant-form-item').filter({ hasText: '密码认证' }).locator('button').first();
  const checked = await switchChecked(switchButton);
  if (checked !== enabled) {
    await switchButton.click();
    await page.waitForTimeout(200);
  }
}

async function inspectProtocol(page, label) {
  await openAddModal(page);
  await chooseProtocol(page, label);
  const text = await anyModalText(page);
  await closeModal(page);
  return text;
}

async function main() {
  const baseUrl = requiredEnv('XUI_BASE_URL').replace(/\/+$/, '');
  const username = requiredEnv('XUI_USERNAME');
  const password = requiredEnv('XUI_PASSWORD');
  const outDir = process.env.XUI_OUT_DIR || path.join(process.cwd(), 'tmp-ui-dom-check');
  fs.mkdirSync(outDir, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();

  await page.goto(`${baseUrl}/`, { waitUntil: 'networkidle' });
  await page.locator('input[placeholder="用户名"]').fill(username);
  await page.locator('input[type="password"]').fill(password);
  await page.locator('button.ant-btn').first().click();
  await page.waitForTimeout(1500);
  await page.goto(`${baseUrl}/xui/`, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1000);
  await page.goto(`${baseUrl}/xui/inbounds`, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1000);

  const result = {
    socks: {},
    http: {},
    portGuard: {},
    protocols: {},
  };

  await openAddModal(page);
  await chooseProtocol(page, 'socks');
  result.socks.defaultNoauth = !(await textVisible(page, '#inbound-modal', 'account1'));
  result.socks.defaultUdpOn = await switchChecked(page.locator('#inbound-modal .ant-form-item').filter({ hasText: '启用 udp' }).locator('button').first());
  result.socks.defaultAccountsHidden = !(await textVisible(page, '#inbound-modal', 'accounts'));
  await setSocksPasswordAuth(page, true);
  result.socks.passwordShowsAccounts = await textVisible(page, '#inbound-modal', 'accounts');
  await setSocksPasswordAuth(page, false);
  result.socks.switchBackHidesAccounts = !(await textVisible(page, '#inbound-modal', 'accounts'));
  await page.screenshot({ path: path.join(outDir, 'socks.png'), fullPage: true });
  await closeModal(page);

  await openAddModal(page);
  await chooseProtocol(page, 'http');
  result.http.defaultAuthOff = !(await switchChecked(page.locator('#inbound-modal .ant-form-item').filter({ hasText: '密码认证' }).locator('button').first()));
  result.http.defaultAccountsHidden = !(await textVisible(page, '#inbound-modal', 'accounts'));
  result.http.allowTransparentHidden = !(await textVisible(page, '#inbound-modal', 'allowTransparent'));
  await setHttpPasswordAuth(page, true);
  result.http.enableAuthShowsAccounts = await textVisible(page, '#inbound-modal', 'accounts');
  await page.screenshot({ path: path.join(outDir, 'http.png'), fullPage: true });
  await closeModal(page);

  await openAddModal(page);
  const portGuardText = await anyModalText(page);
  result.portGuard.hasSection = portGuardText.includes('端口保护');
  result.portGuard.noEnglishName = !portGuardText.includes('Port Guard');
  result.portGuard.hasWindowLabel = portGuardText.includes('窗口期 IP 限制');
  result.portGuard.hasIPv4LimitLabel = portGuardText.includes('窗口期唯一 IPv4 数');
  result.portGuard.hasBanSecondsLabel = portGuardText.includes('超限封禁秒数');
  result.portGuard.hasDescription = portGuardText.includes('统计指定时间窗口内访问该端口的唯一 IPv4 来源 IP');
  result.portGuard.hasNotRealtimeNotice = portGuardText.includes('不是实时在线 IP 数');
  result.portGuard.noForbiddenCopy = !/Online IP|Device Count|Client Count/i.test(portGuardText);
  await page.screenshot({ path: path.join(outDir, 'port-guard.png'), fullPage: true });
  await closeModal(page);

  const protocols = [
    ['VMess', 'vmess'],
    ['VLESS', 'vless'],
    ['Trojan', 'trojan'],
    ['Shadowsocks', 'shadowsocks'],
    ['Dokodemo-door', 'dokodemo-door'],
    ['SOCKS', 'socks'],
    ['HTTP', 'http'],
    ['mixed', 'mixed'],
    ['tunnel', 'tunnel'],
  ];
  for (const [name, option] of protocols) {
    const text = await inspectProtocol(page, option);
    result.protocols[name] = {
      hasSniffing: text.includes('sniffing') || text.includes('destOverride') || text.includes('routeOnly'),
      hasFallbacks: text.includes('fallbacks'),
    };
  }

  await page.screenshot({ path: path.join(outDir, 'inbounds-page.png'), fullPage: true });
  await browser.close();

  const outputPath = path.join(outDir, 'result.json');
  fs.writeFileSync(outputPath, JSON.stringify(result, null, 2));
  console.log(outputPath);
  console.log(JSON.stringify(result, null, 2));
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
