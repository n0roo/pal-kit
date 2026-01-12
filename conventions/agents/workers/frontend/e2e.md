# E2E Worker 컨벤션

> E2E 테스트 전문 Worker

---

## 1. 역할 정의

E2E Worker는 **End-to-End 테스트**를 담당하는 전문 Worker입니다.

### 1.1 담당 영역

- 사용자 시나리오 테스트
- 크로스 브라우저 테스트
- 시각적 회귀 테스트
- 성능 테스트
- 접근성 테스트

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| E2E | Playwright, Cypress |
| 시각적 테스트 | Percy, Chromatic |
| 성능 | Lighthouse CI |

---

## 2. E2E 테스트 원칙

### 2.1 테스트 피라미드

```
         /\
        /  \  E2E (소수의 핵심 시나리오)
       /----\
      /      \ Integration (주요 기능)
     /--------\
    /          \ Unit (많은 단위 테스트)
   --------------
```

### 2.2 E2E 테스트 대상

| 대상 | 설명 | 예시 |
|------|------|------|
| Critical Path | 핵심 사용자 흐름 | 로그인 → 주문 → 결제 |
| Happy Path | 정상 시나리오 | 상품 검색 → 장바구니 |
| Edge Cases | 경계 케이스 | 재고 없음, 결제 실패 |

---

## 3. Playwright 테스트

### 3.1 기본 테스트 구조

```typescript
// tests/e2e/checkout.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Checkout Flow', () => {
  test.beforeEach(async ({ page }) => {
    // 로그인
    await page.goto('/login');
    await page.fill('[data-testid="email"]', 'test@example.com');
    await page.fill('[data-testid="password"]', 'password123');
    await page.click('[data-testid="login-button"]');
    await expect(page).toHaveURL('/dashboard');
  });

  test('should complete checkout successfully', async ({ page }) => {
    // 1. 상품 선택
    await page.goto('/products');
    await page.click('[data-testid="product-card"]:first-child');
    await page.click('[data-testid="add-to-cart"]');

    // 2. 장바구니 확인
    await page.goto('/cart');
    await expect(page.locator('[data-testid="cart-item"]')).toHaveCount(1);

    // 3. 체크아웃
    await page.click('[data-testid="checkout-button"]');

    // 4. 배송 정보 입력
    await page.fill('[data-testid="address"]', '서울시 강남구');
    await page.fill('[data-testid="phone"]', '010-1234-5678');

    // 5. 결제
    await page.click('[data-testid="payment-card"]');
    await page.click('[data-testid="complete-order"]');

    // 6. 완료 확인
    await expect(page).toHaveURL(/\/orders\/\d+/);
    await expect(page.locator('[data-testid="order-success"]')).toBeVisible();
  });

  test('should show error when cart is empty', async ({ page }) => {
    await page.goto('/cart');
    await expect(page.locator('[data-testid="empty-cart"]')).toBeVisible();
    await expect(page.locator('[data-testid="checkout-button"]')).toBeDisabled();
  });
});
```

### 3.2 Page Object Model

```typescript
// tests/e2e/pages/CheckoutPage.ts
import { Page, Locator, expect } from '@playwright/test';

export class CheckoutPage {
  readonly page: Page;
  readonly addressInput: Locator;
  readonly phoneInput: Locator;
  readonly paymentCards: Locator;
  readonly completeButton: Locator;
  readonly orderTotal: Locator;

  constructor(page: Page) {
    this.page = page;
    this.addressInput = page.locator('[data-testid="address"]');
    this.phoneInput = page.locator('[data-testid="phone"]');
    this.paymentCards = page.locator('[data-testid="payment-card"]');
    this.completeButton = page.locator('[data-testid="complete-order"]');
    this.orderTotal = page.locator('[data-testid="order-total"]');
  }

  async goto() {
    await this.page.goto('/checkout');
  }

  async fillShippingInfo(address: string, phone: string) {
    await this.addressInput.fill(address);
    await this.phoneInput.fill(phone);
  }

  async selectPaymentMethod(method: 'card' | 'bank' | 'kakao') {
    await this.paymentCards.filter({ hasText: method }).click();
  }

  async completeOrder() {
    await this.completeButton.click();
    await expect(this.page).toHaveURL(/\/orders\/\d+/);
  }

  async expectTotal(amount: string) {
    await expect(this.orderTotal).toHaveText(amount);
  }
}
```

### 3.3 Page Object 사용

```typescript
// tests/e2e/checkout-with-pom.spec.ts
import { test, expect } from '@playwright/test';
import { CheckoutPage } from './pages/CheckoutPage';
import { CartPage } from './pages/CartPage';

test.describe('Checkout with POM', () => {
  test('should complete checkout', async ({ page }) => {
    const cartPage = new CartPage(page);
    const checkoutPage = new CheckoutPage(page);

    await cartPage.goto();
    await cartPage.addItem('product-1');
    await cartPage.proceedToCheckout();

    await checkoutPage.fillShippingInfo('서울시 강남구', '010-1234-5678');
    await checkoutPage.selectPaymentMethod('card');
    await checkoutPage.expectTotal('₩50,000');
    await checkoutPage.completeOrder();
  });
});
```

---

## 4. 테스트 데이터 관리

### 4.1 Fixtures

```typescript
// tests/e2e/fixtures/auth.fixture.ts
import { test as base } from '@playwright/test';

type AuthFixtures = {
  authenticatedPage: Page;
};

export const test = base.extend<AuthFixtures>({
  authenticatedPage: async ({ page }, use) => {
    // 로그인
    await page.goto('/login');
    await page.fill('[data-testid="email"]', 'test@example.com');
    await page.fill('[data-testid="password"]', 'password123');
    await page.click('[data-testid="login-button"]');
    await page.waitForURL('/dashboard');

    await use(page);

    // 로그아웃
    await page.goto('/logout');
  },
});

export { expect } from '@playwright/test';
```

### 4.2 테스트 데이터

```typescript
// tests/e2e/data/testData.ts
export const testUsers = {
  standard: {
    email: 'test@example.com',
    password: 'password123',
  },
  premium: {
    email: 'premium@example.com',
    password: 'premium123',
  },
};

export const testProducts = {
  basic: {
    id: 'product-1',
    name: '테스트 상품',
    price: 10000,
  },
};

export const testAddresses = {
  seoul: {
    address: '서울시 강남구 역삼동 123',
    zipCode: '06234',
    phone: '010-1234-5678',
  },
};
```

---

## 5. 시각적 테스트

### 5.1 스크린샷 비교

```typescript
// tests/e2e/visual/homepage.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Visual Regression', () => {
  test('homepage should match snapshot', async ({ page }) => {
    await page.goto('/');
    await expect(page).toHaveScreenshot('homepage.png', {
      fullPage: true,
      animations: 'disabled',
    });
  });

  test('product card should match snapshot', async ({ page }) => {
    await page.goto('/products');
    const productCard = page.locator('[data-testid="product-card"]').first();
    await expect(productCard).toHaveScreenshot('product-card.png');
  });

  test('should match dark mode snapshot', async ({ page }) => {
    await page.goto('/');
    await page.click('[data-testid="theme-toggle"]');
    await expect(page).toHaveScreenshot('homepage-dark.png');
  });
});
```

---

## 6. 접근성 테스트

### 6.1 axe-core 통합

```typescript
// tests/e2e/a11y/accessibility.spec.ts
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';

test.describe('Accessibility', () => {
  test('homepage should have no a11y violations', async ({ page }) => {
    await page.goto('/');

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa'])
      .analyze();

    expect(results.violations).toEqual([]);
  });

  test('checkout form should be accessible', async ({ page }) => {
    await page.goto('/checkout');

    const results = await new AxeBuilder({ page })
      .include('[data-testid="checkout-form"]')
      .analyze();

    expect(results.violations).toEqual([]);
  });
});
```

---

## 7. 성능 테스트

### 7.1 Web Vitals 측정

```typescript
// tests/e2e/performance/vitals.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Performance', () => {
  test('should meet Core Web Vitals', async ({ page }) => {
    await page.goto('/');

    // Lighthouse 측정 (별도 설정 필요)
    const metrics = await page.evaluate(() =>
      JSON.stringify(performance.getEntriesByType('navigation'))
    );

    const navigationTiming = JSON.parse(metrics)[0];

    // First Contentful Paint
    expect(navigationTiming.domContentLoadedEventEnd).toBeLessThan(2000);

    // Time to Interactive
    expect(navigationTiming.loadEventEnd).toBeLessThan(5000);
  });

  test('should load images lazily', async ({ page }) => {
    await page.goto('/products');

    // 초기 로드된 이미지 수 확인
    const initialImages = await page.locator('img[loading="lazy"]').count();
    expect(initialImages).toBeGreaterThan(0);
  });
});
```

---

## 8. 테스트 설정

### 8.1 playwright.config.ts

```typescript
// playwright.config.ts
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: [
    ['html'],
    ['junit', { outputFile: 'results.xml' }],
  ],
  use: {
    baseURL: 'http://localhost:3000',
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    {
      name: 'firefox',
      use: { ...devices['Desktop Firefox'] },
    },
    {
      name: 'webkit',
      use: { ...devices['Desktop Safari'] },
    },
    {
      name: 'Mobile Chrome',
      use: { ...devices['Pixel 5'] },
    },
    {
      name: 'Mobile Safari',
      use: { ...devices['iPhone 12'] },
    },
  ],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:3000',
    reuseExistingServer: !process.env.CI,
  },
});
```

---

## 9. 파일 구조

```
tests/
└── e2e/
    ├── fixtures/
    │   └── auth.fixture.ts
    ├── pages/
    │   ├── LoginPage.ts
    │   ├── CartPage.ts
    │   └── CheckoutPage.ts
    ├── data/
    │   └── testData.ts
    ├── visual/
    │   └── homepage.spec.ts
    ├── a11y/
    │   └── accessibility.spec.ts
    ├── performance/
    │   └── vitals.spec.ts
    ├── checkout.spec.ts
    └── auth.spec.ts
```

---

## 10. 완료 체크리스트

- [ ] Critical Path 테스트 작성
- [ ] Page Object Model 구현
- [ ] 테스트 데이터/Fixtures 설정
- [ ] 시각적 회귀 테스트 설정
- [ ] 접근성 테스트 구현
- [ ] 성능 테스트 구현
- [ ] CI 파이프라인 연동

---

## 11. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| Flaky 테스트 | Engineer Worker | 테스트 안정화 |
| 시각적 차이 | UI Worker | UI 수정 확인 |
| 성능 저하 | Architect | 성능 최적화 검토 |
| 접근성 위반 | UI Worker | a11y 수정 |

---

<!-- pal:convention:workers:frontend:e2e -->
