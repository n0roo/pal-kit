import { test, expect } from '@playwright/test'

test.describe('Navigation', () => {
  test('should navigate to Sessions page', async ({ page }) => {
    await page.goto('/')
    
    // Click Sessions link
    await page.click('text=Sessions')
    
    // Should show sessions page
    await expect(page).toHaveURL('/sessions')
    await expect(page.locator('h1')).toContainText('세션')
  })

  test('should navigate to Orchestrations page', async ({ page }) => {
    await page.goto('/')
    
    await page.click('text=Orchestrations')
    
    await expect(page).toHaveURL('/orchestrations')
    await expect(page.locator('h1')).toContainText('Orchestration')
  })

  test('should navigate to Agents page', async ({ page }) => {
    await page.goto('/')
    
    await page.click('text=Agents')
    
    await expect(page).toHaveURL('/agents')
    await expect(page.locator('h1')).toContainText('에이전트')
  })

  test('should navigate to Attention page', async ({ page }) => {
    await page.goto('/')
    
    await page.click('text=Attention')
    
    await expect(page).toHaveURL('/attention')
    await expect(page.locator('h1')).toContainText('Attention')
  })

  test('should highlight active nav item', async ({ page }) => {
    await page.goto('/sessions')
    
    // Sessions nav item should have active styling
    const activeNav = page.locator('nav a[href="/sessions"]')
    await expect(activeNav).toHaveClass(/bg-dark-700|bg-primary/)
  })

  test('should return to dashboard', async ({ page }) => {
    await page.goto('/sessions')
    
    // Click Dashboard
    await page.click('text=Dashboard')
    
    await expect(page).toHaveURL('/')
  })
})
