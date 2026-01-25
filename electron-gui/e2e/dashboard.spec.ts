import { test, expect } from '@playwright/test'

test.describe('Dashboard', () => {
  test('should load dashboard page', async ({ page }) => {
    await page.goto('/')

    // Check title
    await expect(page.locator('h1')).toContainText('대시보드')

    // Check projects section header is present
    await expect(page.locator('h2:has-text("프로젝트")')).toBeVisible()
  })

  test('should show server status', async ({ page }) => {
    await page.goto('/')

    // Server status indicator should be visible
    const serverStatus = page.locator('text=/연결됨|연결 안됨/')
    await expect(serverStatus).toBeVisible()
  })

  test('should show recent events section', async ({ page }) => {
    await page.goto('/')

    // Events section header
    await expect(page.locator('text=최근 이벤트')).toBeVisible()
  })

  test('should show orchestrations section', async ({ page }) => {
    await page.goto('/')

    // Orchestrations section
    await expect(page.locator('text=실행 중인 Orchestration')).toBeVisible()
  })
})
