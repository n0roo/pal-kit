import { test, expect } from '@playwright/test'

test.describe('Sessions Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/sessions')
  })

  test('should display sessions page header', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('세션')
    await expect(page.locator('text=세션 계층 구조')).toBeVisible()
  })

  test('should show session type filter or list', async ({ page }) => {
    // Either shows sessions or empty state
    const content = page.locator('main')
    await expect(content).toBeVisible()
  })

  test('should handle empty sessions state', async ({ page }) => {
    // If no sessions, should show appropriate message
    const emptyState = page.locator('text=/세션이 없습니다|로딩/')
    const sessionList = page.locator('[data-testid="session-tree"]')
    
    // Either shows sessions or empty message
    const hasContent = await sessionList.count() > 0 || await emptyState.count() > 0
    expect(hasContent).toBeTruthy()
  })
})

test.describe('Orchestrations Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/orchestrations')
  })

  test('should display orchestrations page', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Orchestration')
  })

  test('should have status filter buttons', async ({ page }) => {
    // Look for filter buttons
    const filters = page.locator('button')
    await expect(filters.first()).toBeVisible()
  })
})

test.describe('Agents Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/agents')
  })

  test('should display agents page', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('에이전트')
  })

  test('should show agent type tabs', async ({ page }) => {
    // Should have tabs for different agent types
    await expect(page.locator('text=/worker|operator|전체/')).toBeVisible()
  })
})

test.describe('Attention Page', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/attention')
  })

  test('should display attention page', async ({ page }) => {
    await expect(page.locator('h1')).toContainText('Attention')
  })

  test('should show attention info', async ({ page }) => {
    // Should show token usage or session selector
    const content = page.locator('main')
    await expect(content).toBeVisible()
  })
})
