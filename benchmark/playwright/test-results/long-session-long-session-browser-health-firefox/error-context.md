# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: long-session.spec.ts >> long session browser health
- Location: tests/long-session.spec.ts:5:5

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
  5  | test('long session browser health', async ({ page }, testInfo) => {
> 6  |   await page.goto('/main', { waitUntil: 'domcontentloaded' });
     |              ^ Error: page.goto: NS_ERROR_CONNECTION_REFUSED
  7  |   await page.waitForSelector('#viewport-canvas', { timeout: 30_000 });
  8  | 
  9  |   const samples: Array<{ ts: number; heapMb: number; fpsApprox: number }> = [];
  10 |   const start = Date.now();
  11 |   while (Date.now() - start < 180_000) {
  12 |     const sample = await page.evaluate(() => {
  13 |       const perf = performance as any;
  14 |       const heap = perf.memory?.usedJSHeapSize ? perf.memory.usedJSHeapSize / (1024 * 1024) : 0;
  15 |       return { ts: Date.now(), heapMb: heap, fpsApprox: 60 };
  16 |     });
  17 |     samples.push(sample);
  18 |     await page.waitForTimeout(3000);
  19 |   }
  20 | 
  21 |   expect(samples.length).toBeGreaterThan(10);
  22 | 
  23 |   const outDir = process.env.PLAYWRIGHT_RESULTS_DIR || path.resolve(process.cwd(), '../../results/playwright');
  24 |   fs.mkdirSync(outDir, { recursive: true });
  25 |   const outPath = path.join(outDir, `long-session-${testInfo.project.name}.json`);
  26 |   fs.writeFileSync(outPath, JSON.stringify({ samples }, null, 2));
  27 | });
  28 | 
  29 | 
```