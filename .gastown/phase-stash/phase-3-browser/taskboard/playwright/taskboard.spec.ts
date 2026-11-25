import { test, expect } from '@playwright/test';

test.describe('taskboard smoke', () => {
  test('root page responds with HTML', async ({ request }) => {
    const res = await request.get('/');
    expect(res.ok()).toBeTruthy();
    expect(res.headers()['content-type']).toContain('text/html');
  });
});

test.describe('taskboard projects', () => {
  test('creating a project from the form lists it in the project list', async ({ page }) => {
    const name = `e2e-project-${Date.now()}`;

    await page.goto('/');
    await expect(page.locator('#projects-panel')).toBeVisible();

    await page.locator('#project-name').fill(name);
    await page.locator('#project-create').click();

    await expect(page.locator('#project-list')).toContainText(name);
  });
});