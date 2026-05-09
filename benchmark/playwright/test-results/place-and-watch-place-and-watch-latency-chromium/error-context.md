# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: place-and-watch.spec.ts >> place and watch latency
- Location: tests/place-and-watch.spec.ts:5:5

# Error details

```
Error: page.goto: net::ERR_SSL_PROTOCOL_ERROR at https://192.168.122.71/main
Call log:
  - navigating to "https://192.168.122.71/main", waiting until "domcontentloaded"

```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test';
  2  | import fs from 'fs';
  3  | import path from 'path';
  4  | 
  5  | test('place and watch latency', async ({ page }, testInfo) => {
> 6  |   await page.goto('/main', { waitUntil: 'domcontentloaded' });
     |              ^ Error: page.goto: net::ERR_SSL_PROTOCOL_ERROR at https://192.168.122.71/main
  7  |   await page.waitForSelector('#viewport-canvas', { timeout: 30_000 });
  8  | 
  9  |   await page.evaluate(() => {
  10 |     (window as any).__pbMetrics = { wsToRender: [] as number[], clickToRender: [] as number[] };
  11 |   });
  12 | 
  13 |   const canvas = page.locator('#viewport-canvas');
  14 |   for (let i = 0; i < 10; i++) {
  15 |     const t0 = Date.now();
  16 |     await canvas.click({ position: { x: 50 + i * 3, y: 50 + i * 2 } });
  17 |     await page.waitForTimeout(250);
  18 |     await page.evaluate((startedAt) => {
  19 |       const now = Date.now();
  20 |       (window as any).__pbMetrics.clickToRender.push(now - startedAt);
  21 |     }, t0);
  22 |   }
  23 | 
  24 |   const metrics = await page.evaluate(() => (window as any).__pbMetrics);
  25 |   expect(Array.isArray(metrics.clickToRender)).toBeTruthy();
  26 | 
  27 |   const outDir = process.env.PLAYWRIGHT_RESULTS_DIR || path.resolve(process.cwd(), '../../results/playwright');
  28 |   fs.mkdirSync(outDir, { recursive: true });
  29 |   const outPath = path.join(outDir, `place-and-watch-${testInfo.project.name}.json`);
  30 |   fs.writeFileSync(outPath, JSON.stringify(metrics, null, 2));
  31 | });
  32 | 
  33 | 
```