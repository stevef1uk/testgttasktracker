import { test, expect, Page } from '@playwright/test';

// Selectors below are taken from web/static/app.js (DOM IDs, data-column lists,
// "Move forward" / "Delete" button labels). Each test performs real UI actions
// and asserts the resulting DOM state.

const uniq = (prefix: string) => `${prefix} ${Date.now()}-${Math.floor(Math.random() * 1e6)}`;

async function createProject(page: Page, name: string) {
  await page.locator('#project-name').fill(name);
  await page.locator('#project-form button[type="submit"]').click();
  await expect(page.locator('#project-select option', { hasText: name })).toHaveCount(1);
}

async function selectProject(page: Page, name: string) {
  await page.locator('#project-select').selectOption({ label: name });
  await expect(page.locator('#project-select')).toHaveValue(/.+/);
}

async function createTask(page: Page, title: string, priority: 'low' | 'medium' | 'high' = 'medium') {
  await page.locator('#task-title').fill(title);
  await page.locator('#task-priority').selectOption(priority);
  await page.locator('#task-form button[type="submit"]').click();
  await expect(page.locator('.card', { hasText: title })).toHaveCount(1);
}

function cardIn(page: Page, column: 'todo' | 'doing' | 'done', title: string) {
  return page.locator(`[data-column="${column}"] .card`, { hasText: title });
}

test.describe('taskboard browser flows', () => {
  test('initial shell renders three kanban columns and controls', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('[data-column="todo"]')).toBeVisible();
    await expect(page.locator('[data-column="doing"]')).toBeVisible();
    await expect(page.locator('[data-column="done"]')).toBeVisible();
    await expect(page.locator('#project-select')).toBeVisible();
    await expect(page.locator('#project-name')).toBeVisible();
    await expect(page.locator('#task-title')).toBeVisible();
    await expect(page.locator('#status')).toHaveText(/Ready|Unavailable/);
  });

  test('create project through the form and select it', async ({ page }) => {
    await page.goto('/');
    const name = uniq('Project');
    await createProject(page, name);
    await expect(page.locator('#project-name')).toHaveValue('');
    await selectProject(page, name);
    await expect(page.locator('#project-select')).toHaveValue(/.+/);
  });

  test('create task appears in the To Do column', async ({ page }) => {
    await page.goto('/');
    const name = uniq('Project');
    await createProject(page, name);
    await selectProject(page, name);
    const title = uniq('Task');
    await createTask(page, title, 'high');
    await expect(cardIn(page, 'todo', title)).toHaveCount(1);
    await expect(cardIn(page, 'todo', title).locator('.pill')).toHaveText('high');
  });

  test('Move forward advances a task To Do -> Doing -> Done', async ({ page }) => {
    await page.goto('/');
    const name = uniq('Project');
    await createProject(page, name);
    await selectProject(page, name);
    const title = uniq('Flow');
    await createTask(page, title);

    await cardIn(page, 'todo', title).getByRole('button', { name: 'Move forward' }).click();
    await expect(cardIn(page, 'doing', title)).toHaveCount(1);
    await expect(cardIn(page, 'todo', title)).toHaveCount(0);

    await cardIn(page, 'doing', title).getByRole('button', { name: 'Move forward' }).click();
    await expect(cardIn(page, 'done', title)).toHaveCount(1);
    await expect(cardIn(page, 'doing', title)).toHaveCount(0);

    await expect(cardIn(page, 'done', title).getByRole('button', { name: 'Move forward' })).toBeDisabled();
  });

  test('delete removes a task from the board', async ({ page }) => {
    await page.goto('/');
    const name = uniq('Project');
    await createProject(page, name);
    await selectProject(page, name);
    const title = uniq('Doomed');
    await createTask(page, title);
    await expect(cardIn(page, 'todo', title)).toHaveCount(1);

    await cardIn(page, 'todo', title).getByRole('button', { name: 'Delete' }).click();
    await expect(page.locator('.card', { hasText: title })).toHaveCount(0);
  });

  test('high-priority filter shows only high tasks and reset restores all', async ({ page }) => {
    await page.goto('/');
    const name = uniq('Project');
    await createProject(page, name);
    await selectProject(page, name);
    const highTitle = uniq('High');
    const lowTitle = uniq('Low');
    await createTask(page, highTitle, 'high');
    await createTask(page, lowTitle, 'low');
    await expect(cardIn(page, 'todo', lowTitle)).toHaveCount(1);

    await page.locator('#priority-filter').selectOption('high');
    await expect(cardIn(page, 'todo', highTitle)).toHaveCount(1);
    await expect(cardIn(page, 'todo', lowTitle)).toHaveCount(0);

    await page.locator('#priority-filter').selectOption('all');
    await expect(cardIn(page, 'todo', highTitle)).toHaveCount(1);
    await expect(cardIn(page, 'todo', lowTitle)).toHaveCount(1);
  });
});