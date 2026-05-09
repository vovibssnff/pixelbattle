# Instructions

- Following Playwright test failed.
- Explain why, be concise, respect Playwright best practices.
- Provide a snippet of code with the fix, if possible.

# Test info

- Name: long-session.spec.ts >> long session browser health
- Location: tests/long-session.spec.ts:5:5

# Error details

```
Error: page.goto: SSL_ERROR_UNKNOWN
Call log:
  - navigating to "https://192.168.122.71/main", waiting until "domcontentloaded"

```

# Page snapshot

```yaml
- generic [ref=e2]:
  - generic [ref=e3]:
    - heading "Secure Connection Failed" [level=1] [ref=e5]
    - paragraph [ref=e6]: An error occurred during a connection to 192.168.122.71. Peer reports it experienced an internal error.
    - paragraph [ref=e7]: "Error code: SSL_ERROR_INTERNAL_ERROR_ALERT"
    - list [ref=e9]:
      - listitem [ref=e10]: The page you are trying to view cannot be shown because the authenticity of the received data could not be verified.
      - listitem [ref=e11]: Please contact the website owners to inform them of this problem.
    - paragraph [ref=e12]:
      - link "Learn more…" [ref=e13] [cursor=pointer]:
        - /url: https://support.mozilla.org/1/firefox/148.0.2/Linux/en-US/connection-not-secure
  - button "Try Again" [active] [ref=e15]
```

# Test source

```ts
  1  | import { test, expect } from '@playwright/test';
  2  | import fs from 'fs';
  3  | import path from 'path';
  4  | 
  5  | test('long session browser health', async ({ page }, testInfo) => {
> 6  |   await page.goto('/main', { waitUntil: 'domcontentloaded' });
     |              ^ Error: page.goto: SSL_ERROR_UNKNOWN
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