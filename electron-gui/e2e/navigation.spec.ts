import { test, expect } from '@playwright/test'

test.describe('Navigation', () => {
  test('should navigate to Sessions page', async ({ page }) => {
    await page.goto('/')

    // Click Sessions link (Korean label: 세션)
    await page.click('nav >> text=세션')

    // Should show sessions page (uses h2 for section header)
    await expect(page).toHaveURL('/sessions')
    await expect(page.locator('h2')).toContainText('세션')
  })

  test('should navigate to Orchestrations page', async ({ page }) => {
    await page.goto('/')

    await page.click('nav >> text=Orchestration')

    await expect(page).toHaveURL('/orchestrations')
    await expect(page.locator('h1')).toContainText('Orchestration')
  })

  test('should navigate to Agents page', async ({ page }) => {
    await page.goto('/')

    await page.click('nav >> text=에이전트')

    await expect(page).toHaveURL('/agents')
    await expect(page.locator('h1')).toContainText('에이전트')
  })

  test('should navigate to Attention page', async ({ page }) => {
    await page.goto('/')

    await page.click('nav >> text=Attention')

    // Attention page uses h2 for section header
    await expect(page).toHaveURL('/attention')
    await expect(page.locator('h2')).toContainText('Attention')
  })

  test('should highlight active nav item', async ({ page }) => {
    await page.goto('/sessions')

    // Sessions nav item should have active styling
    const activeNav = page.locator('nav a[href="/sessions"]')
    await expect(activeNav).toHaveClass(/bg-primary/)
  })

  test('should return to dashboard', async ({ page }) => {
    await page.goto('/sessions')

    // Click Dashboard (Korean label: 대시보드)
    await page.click('nav >> text=대시보드')

    await expect(page).toHaveURL('/')
  })
})
