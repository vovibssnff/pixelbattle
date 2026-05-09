# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: place-and-watch.spec.ts >> place and watch latency
- Location: tests/place-and-watch.spec.ts:5:5

# Error details

```
Error: page.goto: NS_ERROR_CONNECTION_REFUSED
Call log:
  - navigating to "https://192.168.122.71/main", waiting until "domcontentloaded"

```

# Page snapshot

```yaml
- generic [ref=e2]:
  - generic [ref=e3]:
    - heading "Unable to connect" [level=1] [ref=e5]
    - paragraph [ref=e6]: Firefox can’t establish a connection to the server at 192.168.122.71.
    - paragraph
    - list [ref=e8]:
      - listitem [ref=e9]: The site could be temporarily unavailable or too busy. Try again in a few moments.
      - listitem [ref=e10]: If you are unable to load any pages, check your computer’s network connection.
      - listitem [ref=e11]: If your computer or network is protected by a firewall or proxy, make sure that Nightly is permitted to access the web.
  - button "Try Again" [active] [ref=e13]
```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test';
  2  | import fs from 'fs';
  3  | import path from 'path';
  4  | 
  5  | test('place and watch latency', async ({ page }, testInfo) => {
> 6  |   await page.goto('/main', { waitUntil: 'domcontentloaded' });
     |              ^ Error: page.goto: NS_ERROR_CONNECTION_REFUSED
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