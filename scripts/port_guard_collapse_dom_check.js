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

async function login(page, baseUrl, username, password) {
  await page.goto(`${baseUrl}/`, { waitUntil: 'domcontentloaded' });
  await page.locator('input[placeholder="用户名"]').fill(username);
  await page.locator('input[type="password"]').fill(password);
  await page.locator('button.ant-btn').first().click();
  await page.waitForTimeout(1000);
  await page.goto(`${baseUrl}/xui/inbounds`, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1200);
}

async function openAddModal(page) {
  await page.locator('button.ant-btn-primary').first().click();
  await page.waitForTimeout(400);
}

async function modalText(page) {
  return (await page.locator('#inbound-modal').textContent()) || '';
}

async function portGuardHeader(page) {
  return page.locator('#inbound-modal button').filter({ hasText: '端口保护（高级功能）' }).first();
}

async function isPortGuardExpanded(page) {
  const alertVisible = await page.locator('#inbound-modal .ant-alert').filter({ hasText: '窗口期 IP 限制' }).first().isVisible();
  const switchVisible = await page.locator('#inbound-modal .ant-form-item').filter({ hasText: '启用端口保护' }).first().isVisible();
  const banVisible = await page.locator('#inbound-modal .ant-form-item').filter({ hasText: '超限封禁秒数' }).first().isVisible();
  return alertVisible && switchVisible && banVisible;
}

async function fillFirstFormInput(page, label, value) {
  const input = page.locator('#inbound-modal .ant-form-item').filter({ hasText: label }).locator('input').first();
  await input.fill(String(value));
}

async function selectProtocol(page, name) {
  await page.locator('#inbound-modal .ant-select').first().click();
  await page.waitForTimeout(200);
  const options = page.locator('.ant-select-dropdown-menu-item');
  for (let i = 0; i < await options.count(); i++) {
    const text = ((await options.nth(i).textContent()) || '').trim().toLowerCase();
    if (text === name.toLowerCase()) {
      await options.nth(i).click();
      await page.waitForTimeout(300);
      return;
    }
  }
  throw new Error(`protocol option not found: ${name}`);
}

async function setPortGuardSwitch(page, enabled) {
  const button = page.locator('#inbound-modal .ant-form-item').filter({ hasText: '启用端口保护' }).locator('button').first();
  const className = (await button.getAttribute('class')) || '';
  const checked = className.includes('ant-switch-checked') || (await button.getAttribute('aria-checked')) === 'true';
  if (checked !== enabled) {
    await button.click();
    await page.waitForTimeout(150);
  }
}

async function setNumberByLabel(page, label, value) {
  const input = page.locator('#inbound-modal .ant-form-item').filter({ hasText: label }).locator('input').first();
  await input.fill(String(value));
}

async function submitModal(page) {
  await page.locator('.ant-modal-footer .ant-btn-primary').first().click();
  await page.waitForTimeout(1800);
}

async function openEditByRemark(page, remark) {
  const row = page.locator('tr').filter({ hasText: remark }).first();
  await row.locator('a').filter({ hasText: '操作' }).click();
  await page.waitForTimeout(300);
  await page.locator('.ant-dropdown-menu-item').filter({ hasText: '编辑' }).click();
  await page.waitForTimeout(600);
}

async function deleteByRemark(page, remark) {
  const row = page.locator('tr').filter({ hasText: remark }).first();
  if (await row.count() === 0) {
    return;
  }
  await row.locator('a').filter({ hasText: '操作' }).click();
  await page.waitForTimeout(300);
  await page.locator('.ant-dropdown-menu-item').filter({ hasText: '删除' }).click();
  await page.waitForTimeout(300);
  await page.locator('.ant-modal-confirm-btns .ant-btn-primary').click();
  await page.waitForTimeout(1200);
}

async function runForViewport(browser, viewport, baseUrl, username, password, outDir, suffix) {
  const page = await browser.newPage({ viewport });
  await login(page, baseUrl, username, password);

  const remark = `pg-collapse-${Date.now()}-${suffix}`;
  const result = { suffix };

  await openAddModal(page);
  result.defaultCollapsed = !(await isPortGuardExpanded(page));
  result.defaultHeaderText = ((await (await portGuardHeader(page)).textContent()) || '').replace(/\s+/g, ' ').trim();
  await page.screenshot({ path: path.join(outDir, `port-guard-collapsed-${suffix}.png`), fullPage: true });

  await (await portGuardHeader(page)).click();
  await page.waitForTimeout(200);
  result.clickExpands = await isPortGuardExpanded(page);
  await page.screenshot({ path: path.join(outDir, `port-guard-expanded-${suffix}.png`), fullPage: true });

  await (await portGuardHeader(page)).click();
  await page.waitForTimeout(200);
  result.clickCollapses = !(await isPortGuardExpanded(page));

  await (await portGuardHeader(page)).click();
  await page.waitForTimeout(200);
  await fillFirstFormInput(page, '备注', remark);
  await selectProtocol(page, 'socks');
  await fillFirstFormInput(page, '端口', suffix === 'mobile' ? 31868 : 31867);
  await setPortGuardSwitch(page, true);
  await setNumberByLabel(page, '窗口期秒数', 300);
  await setNumberByLabel(page, '窗口期唯一 IPv4 数', 3);
  await setNumberByLabel(page, '超限封禁秒数', 300);
  await submitModal(page);

  result.savedRowVisible = await page.locator('tr').filter({ hasText: remark }).count() > 0;
  await openEditByRemark(page, remark);
  result.editEnabledAutoExpands = await isPortGuardExpanded(page);
  result.editHeaderText = ((await (await portGuardHeader(page)).textContent()) || '').replace(/\s+/g, ' ').trim();
  await page.screenshot({ path: path.join(outDir, `port-guard-edit-enabled-${suffix}.png`), fullPage: true });
  await page.locator('#inbound-modal .ant-modal-close').click();
  await page.waitForTimeout(300);

  await deleteByRemark(page, remark);
  result.cleanupRemoved = await page.locator('tr').filter({ hasText: remark }).count() === 0;
  await page.close();
  return result;
}

async function main() {
  const baseUrl = requiredEnv('XUI_BASE_URL').replace(/\/+$/, '');
  const username = requiredEnv('XUI_USERNAME');
  const password = requiredEnv('XUI_PASSWORD');
  const outDir = process.env.XUI_OUT_DIR || path.join(process.cwd(), 'tmp-port-guard-collapse-check');
  fs.mkdirSync(outDir, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const desktop = await runForViewport(browser, { width: 1366, height: 900 }, baseUrl, username, password, outDir, 'desktop');
  const mobile = await runForViewport(browser, { width: 390, height: 844 }, baseUrl, username, password, outDir, 'mobile');
  await browser.close();

  const result = { desktop, mobile };
  fs.writeFileSync(path.join(outDir, 'result.json'), JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
