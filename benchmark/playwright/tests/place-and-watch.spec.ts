import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { loginTestUser } from './helpers/auth';

test('place and watch latency', async ({ page, baseURL }, testInfo) => {
  await loginTestUser(page, baseURL!);

  await page.goto('/main', { waitUntil: 'networkidle' });
  await page.waitForSelector('#viewport-canvas', { state: 'attached', timeout: 60_000 });

  await page.evaluate(() => {
    (window as any).__pbMetrics = { wsToRender: [] as number[], clickToRender: [] as number[] };
  });

  const canvas = page.locator('#viewport-canvas');
  for (let i = 0; i < 10; i++) {
    const t0 = Date.now();
    await canvas.click({ position: { x: 50 + i * 3, y: 50 + i * 2 } });
    await page.waitForTimeout(250);
    await page.evaluate((startedAt) => {
      const now = Date.now();
      (window as any).__pbMetrics.clickToRender.push(now - startedAt);
    }, t0);
  }

  const metrics = await page.evaluate(() => (window as any).__pbMetrics);
  expect(Array.isArray(metrics.clickToRender)).toBeTruthy();

  const outDir = process.env.PLAYWRIGHT_RESULTS_DIR || path.resolve(process.cwd(), '../../results/playwright');
  fs.mkdirSync(outDir, { recursive: true });
  const outPath = path.join(outDir, `place-and-watch-${testInfo.project.name}.json`);
  fs.writeFileSync(outPath, JSON.stringify(metrics, null, 2));
});
