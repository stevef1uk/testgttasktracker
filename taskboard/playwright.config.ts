import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './playwright',
  timeout: 10000,
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [['html', { open: 'never' }], ['list']],
  use: {
    baseURL: process.env.BASE_URL || 'http://127.0.0.1:8080',
    trace: 'on-first-retry',
  },
  webServer: process.env.BASE_URL
    ? undefined
    : {
        command: 'go run ./cmd/taskboard',
        cwd: __dirname,
        url: 'http://127.0.0.1:8080/',
        reuseExistingServer: true,
        timeout: 120000,
      },
  projects: [
    {
      name: 'chromium',
      use: {
        ...devices['Desktop Chrome'],
        launchOptions: {
          args: [
            '--disable-features=HttpsUpgrades,HttpsFirstModeV2,AutomaticHttpsDefault,AutomaticHttpsDefaultEnabled,HttpsOnlyMode',
            '--ignore-certificate-errors',
          ],
        },
      },
    },
  ],
});