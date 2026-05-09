import { defineConfig } from '@playwright/test';

const baseURL = process.env.PLAYWRIGHT_BASE_URL || 'http://localhost';
const outDir = process.env.PLAYWRIGHT_RESULTS_DIR || '../../results/playwright';

export default defineConfig({
  testDir: './tests',
  timeout: 180_000,
  retries: 0,
  reporter: [['json', { outputFile: `${outDir}/playwright-report.json` }], ['list']],
  use: {
    baseURL,
    ignoreHTTPSErrors: true,
    trace: 'off',
    screenshot: 'off',
    video: 'off',
  },
  projects: [
    { name: 'chromium', use: { browserName: 'chromium', viewport: { width: 1920, height: 1080 } } },
    { name: 'firefox', use: { browserName: 'firefox', viewport: { width: 1366, height: 768 } } },
  ],
});

