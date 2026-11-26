import { test, expect } from '@playwright/test';

test.describe('taskboard Playwright wiring', () => {
  test('runs on the configured chromium project', async ({ browserName }, testInfo) => {
    expect(testInfo.project.name).toBe('chromium');
    expect(browserName).toBe('chromium');
  });
});