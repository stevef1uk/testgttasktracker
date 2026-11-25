import { test, expect, APIRequestContext, Page } from '@playwright/test';

// Selectors come from web/templates/index.html and web/static/app.js.

function uniqueName(prefix: string): string {
  return `${prefix}-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

async function findProjectId(request: APIRequestContext, name: string): Promise<string | null> {
  const res = await request.get('/api/projects');
  const projects = (await res.json()) || [];
  const p = projects.find((x: { name: string }) => x.name === name);
  return p ? p.id : null;
}

async function cleanupProject(request: APIRequestContext, name: string): Promise<void> {
  const id = await findProjectId(request, name);
  if (id) {
    await request.delete(`/api/projects/${encodeURIComponent(id)}`);
  }
}

async function apiTasks(request: APIRequestContext, name: string) {
  const pid = await findProjectId(request, name);
  expect(pid).not.toBeNull();
  const res = await request.get(`/api/projects/${encodeURIComponent(pid as string)}/tasks`);
  return (await res.json()) || [];
}

async function openProject(page: Page, name: string): Promise<void> {
  await page.goto('/');
  await expect(page.locator('#projects-panel')).toBeVisible();
  await page.locator('#project-name').fill(name);
  await page.locator('#project-create').click();
  const item = page.locator('#project-list li', { hasText: name });
  await expect(item).toBeVisible();
  await item.locator('.select-project').click();
}

async function createTask(page: Page, title: string, priority: string): Promise<void> {
  await page.locator('#task-title').fill(title);
  await page.locator('#task-priority').selectOption(priority);
  await page.locator('#task-create').click();
  await expect(page.locator('#list-todo li', { hasText: title })).toBeVisible();
}

function card(page: Page, title: string) {
  return page.locator('#kanban-board li', { hasText: title });
}

test.describe('taskboard smoke', () => {
  test('root page responds with HTML', async ({ request }) => {
    const res = await request.get('/');
    expect(res.ok()).toBeTruthy();
    expect(res.headers()['content-type']).toContain('text/html');
  });
});

test.describe('taskboard projects', () => {
  test('creating a project from the form lists it in the project list', async ({ page, request }) => {
    const name = uniqueName('e2e-project');
    try {
      await page.goto('/');
      await expect(page.locator('#projects-panel')).toBeVisible();
      await page.locator('#project-name').fill(name);
      await page.locator('#project-create').click();
      await expect(page.locator('#project-list')).toContainText(name);
    } finally {
      await cleanupProject(request, name);
    }
  });
});

test.describe('taskboard tasks', () => {
  test('creating a task through the form shows it in the To Do column', async ({ page, request }) => {
    const name = uniqueName('e2e-create');
    const title = uniqueName('task-create');
    await cleanupProject(request, name);
    try {
      await openProject(page, name);
      await createTask(page, title, 'medium');
      await expect(page.locator('#list-todo')).toContainText(title);
      await expect(page.locator('#list-doing')).not.toContainText(title);
      const tasks = await apiTasks(request, name);
      expect(tasks.find((t: { title: string }) => t.title === title)?.status).toBe('todo');
    } finally {
      await cleanupProject(request, name);
    }
  });

  test('move forward advances a task To Do -> Doing -> Done and hides the control on done', async ({ page, request }) => {
    const name = uniqueName('e2e-move');
    const title = uniqueName('task-move');
    await cleanupProject(request, name);
    try {
      await openProject(page, name);
      await createTask(page, title, 'high');

      await card(page, title).locator('.move-forward').click();
      await expect(page.locator('#list-doing')).toContainText(title);
      await expect(page.locator('#list-todo')).not.toContainText(title);
      let tasks = await apiTasks(request, name);
      expect(tasks.find((t: { title: string }) => t.title === title)?.status).toBe('doing');

      await card(page, title).locator('.move-forward').click();
      await expect(page.locator('#list-done')).toContainText(title);
      await expect(page.locator('#list-doing')).not.toContainText(title);
      await expect(card(page, title).locator('.move-forward')).toHaveCount(0);
      tasks = await apiTasks(request, name);
      expect(tasks.find((t: { title: string }) => t.title === title)?.status).toBe('done');
    } finally {
      await cleanupProject(request, name);
    }
  });

  test('deleting a task removes it from the board and the API list', async ({ page, request }) => {
    const name = uniqueName('e2e-delete');
    const title = uniqueName('task-delete');
    await cleanupProject(request, name);
    try {
      await openProject(page, name);
      await createTask(page, title, 'low');

      await card(page, title).locator('.delete-task').click();
      await expect(card(page, title)).toHaveCount(0);
      const tasks = await apiTasks(request, name);
      expect(tasks.find((t: { title: string }) => t.title === title)).toBeUndefined();
    } finally {
      await cleanupProject(request, name);
    }
  });

  test('priority filter shows only matching tasks and resets to all', async ({ page, request }) => {
    const name = uniqueName('e2e-filter');
    const highTitle = uniqueName('task-high');
    const lowTitle = uniqueName('task-low');
    await cleanupProject(request, name);
    try {
      await openProject(page, name);
      await createTask(page, highTitle, 'high');
      await createTask(page, lowTitle, 'low');

      await page.locator('#filter-priority').selectOption('high');
      await expect(card(page, highTitle)).toBeVisible();
      await expect(card(page, lowTitle)).toHaveCount(0);

      await page.locator('#filter-priority').selectOption('');
      await expect(card(page, highTitle)).toBeVisible();
      await expect(card(page, lowTitle)).toBeVisible();
    } finally {
      await cleanupProject(request, name);
    }
  });
});
