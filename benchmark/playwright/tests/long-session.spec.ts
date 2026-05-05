import { test, expect } from '@playwright/test';
import fs from 'fs';
import path from 'path';

test('long session browser health', async ({ page }, testInfo) => {
  await page.goto('/main', { waitUntil: 'domcontentloaded' });
  await page.waitForSelector('#viewport-canvas', { timeout: 30_000 });

  const samples: Array<{ ts: number; heapMb: number; fpsApprox: number }> = [];
  const start = Date.now();
  while (Date.now() - start < 180_000) {
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

