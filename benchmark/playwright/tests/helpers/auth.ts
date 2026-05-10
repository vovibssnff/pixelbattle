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

  await ctx.request.post(`${baseURL}/api/register`, {
    data: { username: TEST_USER.username, password: TEST_USER.password, faculty: TEST_USER.faculty },
  }).catch(() => {});

  const resp = await ctx.request.post(`${baseURL}/api/login`, {
    data: { username: TEST_USER.username, password: TEST_USER.password },
  });

  if (!resp.ok()) {
    throw new Error(`Login failed: ${resp.status()} ${await resp.text()}`);
  }
}
