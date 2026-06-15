#!/usr/bin/env node

const { chromium } = require('playwright');

function requiredEnv(name) {
  const value = process.env[name];
  if (!value) {
    throw new Error(`missing required env: ${name}`);
  }
  return value;
}

async function login(page, baseUrl, username, password) {
  await page.goto(`${baseUrl}/`, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1000);
  await page.locator('input[placeholder="用户名"]').fill(username);
  await page.locator('input[type="password"]').fill(password);
  await page.locator('button.ant-btn').first().click();
  await page.waitForTimeout(1500);
  await page.goto(`${baseUrl}/xui/inbounds`, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1500);
}

async function openAddModal(page) {
  await page.locator('button.ant-btn-primary').first().click();
  await page.waitForTimeout(500);
}

async function selectProtocol(page, name) {
  await page.locator('#inbound-modal .ant-select').first().click();
  await page.waitForTimeout(200);
  const options = page.locator('.ant-select-dropdown-menu-item');
  for (let i = 0; i < await options.count(); i++) {
    const text = ((await options.nth(i).textContent()) || '').trim().toLowerCase();
    if (text === name.toLowerCase()) {
      await options.nth(i).click();
      await page.waitForTimeout(500);
      return;
    }
  }
  throw new Error(`protocol option not found: ${name}`);
}

async function setSecurity(page, value) {
  const selects = page.locator('#inbound-modal .ant-select');
  await selects.nth(await selects.count() - 1).click();
  await page.waitForTimeout(200);
  const options = page.locator('.ant-select-dropdown-menu-item');
  for (let i = 0; i < await options.count(); i++) {
    const text = ((await options.nth(i).textContent()) || '').trim().toLowerCase();
    if (text === value.toLowerCase()) {
      await options.nth(i).click();
      await page.waitForTimeout(500);
      return;
    }
  }
  throw new Error(`security option not found: ${value}`);
}

async function closeModal(page) {
  await page.locator('#inbound-modal .ant-modal-close').click();
  await page.waitForTimeout(300);
}

async function main() {
  const baseUrl = requiredEnv('XUI_BASE_URL').replace(/\/+$/, '');
  const username = requiredEnv('XUI_USERNAME');
  const password = requiredEnv('XUI_PASSWORD');

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage();
  await login(page, baseUrl, username, password);

  const results = {};

  await openAddModal(page);
  await selectProtocol(page, 'vless');
  await setSecurity(page, 'tls');
  const tlsText = (await page.locator('#inbound-modal').textContent()) || '';
  results.tlsSelectorVisible = tlsText.includes('本地证书');
  results.tlsCertPathFilledBefore = tlsText.includes('/etc/x-ui/certs/');
  const certSelect = page.locator('#inbound-modal .ant-form-item').filter({ hasText: '本地证书' }).locator('.ant-select').first();
  if (await certSelect.count() > 0) {
    await certSelect.click();
    await page.waitForTimeout(200);
    await page.locator('.ant-select-dropdown-menu-item').first().click();
    await page.waitForTimeout(500);
  }
  const certFileValue = await page.locator('input').evaluateAll(inputs => {
    const match = inputs.find(input => input.value && input.value.includes('/etc/x-ui/certs/'));
    return match ? match.value : '';
  });
  results.tlsCertPathFilledAfter = certFileValue.includes('/etc/x-ui/certs/');
  await closeModal(page);

  await openAddModal(page);
  await selectProtocol(page, 'vless');
  await setSecurity(page, 'reality');
  const realityText = (await page.locator('#inbound-modal').textContent()) || '';
  results.realityHidesCertSelector = !realityText.includes('本地证书');
  await closeModal(page);

  await browser.close();
  console.log(JSON.stringify(results, null, 2));
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
