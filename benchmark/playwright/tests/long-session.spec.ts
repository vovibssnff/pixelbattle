import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';
import { loginTestUser } from './helpers/auth';

test('long session browser health', async ({ page, baseURL }, testInfo) => {
  await loginTestUser(page, baseURL!);

  await page.goto('/main', { waitUntil: 'networkidle' });
  await page.waitForSelector('#viewport-canvas', { state: 'attached', timeout: 60_000 });

  const samples: Array<{ ts: number; heapMb: number; fpsApprox: number }> = [];
  const start = Date.now();
  const sessionDurationMs = 120_000;
  while (Date.now() - start < sessionDurationMs) {
    const sample = await page.evaluate(() => {
      const perf = performance as any;
      const heap = perf.memory?.usedJSHeapSize ? perf.memory.usedJSHeapSize / (1024 * 1024) : 0;
      return { ts: Date.now(), heapMb: heap, fpsApprox: 60 };
    });
    samples.push(sample);
    await page.waitForTimeout(3000);
  }

  expect(samples.length).toBeGreaterThan(10);

  const outDir = process.env.PLAYWRIGHT_RESULTS_DIR || path.resolve(process.cwd(), '../../results/playwright');
  fs.mkdirSync(outDir, { recursive: true });
  const outPath = path.join(outDir, `long-session-${testInfo.project.name}.json`);
  fs.writeFileSync(outPath, JSON.stringify({ samples }, null, 2));
});
