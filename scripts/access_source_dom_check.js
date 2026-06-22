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

async function main() {
  const baseUrl = requiredEnv('XUI_BASE_URL').replace(/\/+$/, '');
  const username = requiredEnv('XUI_USERNAME');
  const password = requiredEnv('XUI_PASSWORD');
  const outDir = process.env.XUI_OUT_DIR || path.join(process.cwd(), 'tmp-access-source-check');
  fs.mkdirSync(outDir, { recursive: true });

  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1366, height: 900 } });
  await page.goto(`${baseUrl}/`, { waitUntil: 'domcontentloaded' });
  await page.locator('input[placeholder="用户名"]').fill(username);
  await page.locator('input[type="password"]').fill(password);
  await page.locator('button.ant-btn').first().click();
  await page.waitForTimeout(1000);
  await page.goto(`${baseUrl}/xui/access-source`, { waitUntil: 'domcontentloaded' });
  await page.waitForTimeout(1200);

  const text = (await page.locator('body').textContent()) || '';
  const result = {
    hasTitle: text.includes('端口共享检测'),
    hasReadOnlyNotice: text.includes('只读 access.log 分析'),
    saysPortOnlyStats: text.includes('每个入站端口出现过的去重来源 IP 数和命中次数'),
    hasIPProfileLayer: text.includes('IP画像') || text.includes('IP 画像'),
    hasGeoLiteNotice: text.includes('GeoLite2'),
    hasDeviceTypeLabel: text.includes('手机网络') || text.includes('云服务器') || text.includes('暂无 IP 画像数据'),
    saysNotOnlineUsers: text.includes('不代表真实在线用户'),
    saysNoRuntimeImpact: text.includes('不影响 Xray 或网络配置'),
    saysNoControlFeature: text.includes('不涉及任何封禁或限制功能'),
    hasUniqueIPCount: text.includes('去重IP数'),
    hasTotalHits: text.includes('命中次数'),
    hasPortStatsArea: text.includes('端口统计'),
    noPeriodStats: !text.includes('1小时') && !text.includes('24小时') && !text.includes('7天'),
    noTrendOrScore: !text.includes('趋势') && !text.includes('评分'),
    hasReadOnlyBoundary: text.includes('只读 access.log 分析') && text.includes('不影响 Xray 或网络配置'),
  };

  await page.screenshot({ path: path.join(outDir, 'access-source.png'), fullPage: true });
  await browser.close();

  fs.writeFileSync(path.join(outDir, 'result.json'), JSON.stringify(result, null, 2));
  console.log(JSON.stringify(result, null, 2));

  if (!Object.values(result).every(Boolean)) {
    process.exit(1);
  }
}

main().catch(err => {
  console.error(err);
  process.exit(1);
});
