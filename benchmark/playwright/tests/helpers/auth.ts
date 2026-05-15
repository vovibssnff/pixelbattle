import { Page } from '@playwright/test';

const TEST_USER = {
  username: `bench_${process.pid}`,
  password: 'benchpass123',
  faculty: 'FTMI',
};

/**
 * Register (best-effort, may already exist) + login via password API.
 * Sets the session cookie so subsequent page.goto() calls are authenticated.
 */
export async function loginTestUser(page: Page, baseURL: string): Promise<void> {
  const ctx = page.context();
  const origin = new URL(baseURL).origin;

  await ctx.request
    .post(`${baseURL}/api/register`, {
      data: { username: TEST_USER.username, password: TEST_USER.password, faculty: TEST_USER.faculty },
    })
    .catch(() => {});

  const resp = await ctx.request.post(`${baseURL}/api/login`, {
    data: { username: TEST_USER.username, password: TEST_USER.password },
  });

  if (resp.status() === 401) {
    throw new Error(`Login failed: 401 ${await resp.text()}`);
  }

  // With maxRedirects: 0, Playwright may not apply Set-Cookie from 303 into the browser context, so
  // document fetch(/api/canvas.png) runs unauthenticated → redirect /login → no #viewport-canvas.
  // Following redirects applies cookies even if the final GET /main is 404 for the API client.
  const cookies = await page.context().cookies([origin]);
  if (!cookies.some((c) => c.name === 'user-session')) {
    throw new Error(`Login failed: no user-session cookie (last HTTP ${resp.status()})`);
  }
}
