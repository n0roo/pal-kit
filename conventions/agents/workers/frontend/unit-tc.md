# Unit TC Worker 컨벤션

> 프론트엔드 단위 테스트 전문 Worker

---

## 1. 역할 정의

Unit TC Worker는 **프론트엔드 단위 테스트**를 담당하는 전문 Worker입니다.

### 1.1 담당 영역

- 컴포넌트 단위 테스트
- Hook 테스트
- 유틸리티 함수 테스트
- Store/State 테스트
- 테스트 커버리지 관리

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| 테스트 러너 | Vitest, Jest |
| 테스팅 라이브러리 | React Testing Library |
| 모킹 | MSW, vi.mock |

---

## 2. 테스트 원칙

### 2.1 Testing Library 철학

```
"테스트는 소프트웨어가 사용되는 방식과 유사하게 작성되어야 한다"
```

| 하지 말 것 | 해야 할 것 |
|----------|----------|
| 구현 세부사항 테스트 | 사용자 행동 테스트 |
| 내부 상태 검증 | 렌더링 결과 검증 |
| className 쿼리 | role, label 쿼리 |

### 2.2 AAA 패턴

```typescript
test('should display error message on invalid input', () => {
  // Arrange (준비)
  render(<LoginForm />);

  // Act (실행)
  const input = screen.getByLabelText('이메일');
  await userEvent.type(input, 'invalid-email');
  await userEvent.click(screen.getByRole('button', { name: '로그인' }));

  // Assert (검증)
  expect(screen.getByText('올바른 이메일을 입력해주세요')).toBeInTheDocument();
});
```

---

## 3. 컴포넌트 테스트

### 3.1 기본 컴포넌트 테스트

```typescript
// features/orders/components/OrderCard.test.tsx
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { OrderCard } from './OrderCard';
import { createMockOrder } from '@/tests/mocks/orderMocks';

describe('OrderCard', () => {
  const mockOrder = createMockOrder({
    id: '123',
    status: 'pending',
    totalPrice: { amount: 50000, currency: 'KRW' },
  });

  it('should render order information', () => {
    render(<OrderCard order={mockOrder} />);

    expect(screen.getByText('주문 #123')).toBeInTheDocument();
    expect(screen.getByText('대기중')).toBeInTheDocument();
    expect(screen.getByText('₩50,000')).toBeInTheDocument();
  });

  it('should call onSelect when clicking detail button', async () => {
    const handleSelect = vi.fn();
    render(<OrderCard order={mockOrder} onSelect={handleSelect} />);

    await userEvent.click(screen.getByRole('button', { name: '상세보기' }));

    expect(handleSelect).toHaveBeenCalledTimes(1);
  });

  it('should show cancel button for pending orders', () => {
    render(<OrderCard order={mockOrder} />);

    expect(screen.getByRole('button', { name: '취소' })).toBeInTheDocument();
  });

  it('should hide cancel button for completed orders', () => {
    const completedOrder = createMockOrder({ status: 'completed' });
    render(<OrderCard order={completedOrder} />);

    expect(screen.queryByRole('button', { name: '취소' })).not.toBeInTheDocument();
  });
});
```

### 3.2 폼 컴포넌트 테스트

```typescript
// features/orders/components/OrderForm.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { OrderForm } from './OrderForm';

describe('OrderForm', () => {
  it('should submit form with valid data', async () => {
    const handleSubmit = vi.fn();
    render(<OrderForm onSubmit={handleSubmit} />);

    await userEvent.type(
      screen.getByLabelText('상품 ID'),
      'product-123'
    );
    await userEvent.type(
      screen.getByLabelText('수량'),
      '2'
    );
    await userEvent.click(screen.getByRole('button', { name: '주문하기' }));

    await waitFor(() => {
      expect(handleSubmit).toHaveBeenCalledWith({
        productId: 'product-123',
        quantity: 2,
      });
    });
  });

  it('should show validation error for empty product', async () => {
    render(<OrderForm onSubmit={vi.fn()} />);

    await userEvent.click(screen.getByRole('button', { name: '주문하기' }));

    expect(await screen.findByText('상품을 선택해주세요')).toBeInTheDocument();
  });

  it('should show validation error for invalid quantity', async () => {
    render(<OrderForm onSubmit={vi.fn()} />);

    await userEvent.type(screen.getByLabelText('수량'), '0');
    await userEvent.click(screen.getByRole('button', { name: '주문하기' }));

    expect(await screen.findByText('최소 1개 이상')).toBeInTheDocument();
  });
});
```

### 3.3 비동기 컴포넌트 테스트

```typescript
// features/orders/components/OrderList.test.tsx
import { render, screen, waitFor } from '@testing-library/react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { OrderList } from './OrderList';
import { server } from '@/tests/mocks/server';
import { http, HttpResponse } from 'msw';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: { retry: false },
  },
});

const wrapper = ({ children }) => (
  <QueryClientProvider client={queryClient}>
    {children}
  </QueryClientProvider>
);

describe('OrderList', () => {
  it('should show loading state initially', () => {
    render(<OrderList />, { wrapper });

    expect(screen.getByTestId('loading-skeleton')).toBeInTheDocument();
  });

  it('should render orders after loading', async () => {
    render(<OrderList />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText('주문 #1')).toBeInTheDocument();
      expect(screen.getByText('주문 #2')).toBeInTheDocument();
    });
  });

  it('should show empty state when no orders', async () => {
    server.use(
      http.get('/api/orders', () => {
        return HttpResponse.json([]);
      })
    );

    render(<OrderList />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText('주문 내역이 없습니다')).toBeInTheDocument();
    });
  });

  it('should show error message on fetch failure', async () => {
    server.use(
      http.get('/api/orders', () => {
        return HttpResponse.error();
      })
    );

    render(<OrderList />, { wrapper });

    await waitFor(() => {
      expect(screen.getByText('데이터를 불러오는데 실패했습니다')).toBeInTheDocument();
    });
  });
});
```

---

## 4. Hook 테스트

### 4.1 Custom Hook 테스트

```typescript
// features/orders/hooks/useOrderCalculation.test.ts
import { renderHook } from '@testing-library/react';
import { useOrderCalculation } from './useOrderCalculation';

describe('useOrderCalculation', () => {
  it('should calculate subtotal correctly', () => {
    const items = [
      { id: '1', price: 10000, quantity: 2 },
      { id: '2', price: 5000, quantity: 1 },
    ];

    const { result } = renderHook(() => useOrderCalculation(items));

    expect(result.current.subtotal).toBe(25000);
  });

  it('should calculate tax (10%)', () => {
    const items = [{ id: '1', price: 10000, quantity: 1 }];

    const { result } = renderHook(() => useOrderCalculation(items));

    expect(result.current.tax).toBe(1000);
  });

  it('should apply 10% discount for orders over 100,000', () => {
    const items = [{ id: '1', price: 50000, quantity: 3 }]; // 150,000원

    const { result } = renderHook(() => useOrderCalculation(items));

    expect(result.current.discount).toBe(15000);
    expect(result.current.total).toBe(150000); // 150000 + 15000(tax) - 15000(discount)
  });

  it('should not apply discount for orders under 100,000', () => {
    const items = [{ id: '1', price: 30000, quantity: 1 }];

    const { result } = renderHook(() => useOrderCalculation(items));

    expect(result.current.discount).toBe(0);
  });
});
```

### 4.2 상태 변경 Hook 테스트

```typescript
// features/orders/hooks/useOrderSelection.test.ts
import { renderHook, act } from '@testing-library/react';
import { useOrderSelection } from './useOrderSelection';

describe('useOrderSelection', () => {
  const mockOrders = [
    { id: '1', name: 'Order 1' },
    { id: '2', name: 'Order 2' },
    { id: '3', name: 'Order 3' },
  ];

  it('should start with empty selection', () => {
    const { result } = renderHook(() => useOrderSelection(mockOrders));

    expect(result.current.selectedCount).toBe(0);
    expect(result.current.isAllSelected).toBe(false);
  });

  it('should toggle selection', () => {
    const { result } = renderHook(() => useOrderSelection(mockOrders));

    act(() => {
      result.current.toggle('1');
    });

    expect(result.current.isSelected('1')).toBe(true);
    expect(result.current.selectedCount).toBe(1);

    act(() => {
      result.current.toggle('1');
    });

    expect(result.current.isSelected('1')).toBe(false);
  });

  it('should select all', () => {
    const { result } = renderHook(() => useOrderSelection(mockOrders));

    act(() => {
      result.current.selectAll();
    });

    expect(result.current.selectedCount).toBe(3);
    expect(result.current.isAllSelected).toBe(true);
  });

  it('should clear selection', () => {
    const { result } = renderHook(() => useOrderSelection(mockOrders));

    act(() => {
      result.current.selectAll();
      result.current.clearSelection();
    });

    expect(result.current.selectedCount).toBe(0);
  });
});
```

---

## 5. Store 테스트

### 5.1 Zustand Store 테스트

```typescript
// features/orders/store/orderStore.test.ts
import { useOrderStore } from './orderStore';

describe('orderStore', () => {
  beforeEach(() => {
    // Store 초기화
    useOrderStore.setState({
      orders: [],
      selectedOrder: null,
      filter: { status: 'all', searchQuery: '' },
    });
  });

  it('should set orders', () => {
    const orders = [{ id: '1' }, { id: '2' }];

    useOrderStore.getState().setOrders(orders);

    expect(useOrderStore.getState().orders).toEqual(orders);
  });

  it('should select order', () => {
    const order = { id: '1', status: 'pending' };

    useOrderStore.getState().selectOrder(order);

    expect(useOrderStore.getState().selectedOrder).toEqual(order);
  });

  it('should update filter', () => {
    useOrderStore.getState().updateFilter({ status: 'completed' });

    expect(useOrderStore.getState().filter.status).toBe('completed');
    expect(useOrderStore.getState().filter.searchQuery).toBe(''); // 기존 값 유지
  });

  it('should clear filter', () => {
    useOrderStore.getState().updateFilter({ status: 'completed', searchQuery: 'test' });
    useOrderStore.getState().clearFilter();

    expect(useOrderStore.getState().filter).toEqual({
      status: 'all',
      searchQuery: '',
    });
  });
});
```

---

## 6. 유틸리티 테스트

### 6.1 함수 테스트

```typescript
// shared/utils/priceUtils.test.ts
import { formatPrice, parsePrice } from './priceUtils';

describe('priceUtils', () => {
  describe('formatPrice', () => {
    it('should format KRW correctly', () => {
      expect(formatPrice({ amount: 10000, currency: 'KRW' })).toBe('₩10,000');
    });

    it('should format USD correctly', () => {
      expect(formatPrice({ amount: 99.99, currency: 'USD' })).toBe('$99.99');
    });

    it('should handle zero', () => {
      expect(formatPrice({ amount: 0, currency: 'KRW' })).toBe('₩0');
    });
  });

  describe('parsePrice', () => {
    it('should parse price string', () => {
      expect(parsePrice('₩10,000')).toBe(10000);
    });

    it('should return 0 for invalid input', () => {
      expect(parsePrice('invalid')).toBe(0);
    });
  });
});
```

---

## 7. MSW 모킹

### 7.1 Handler 설정

```typescript
// tests/mocks/handlers.ts
import { http, HttpResponse } from 'msw';

export const handlers = [
  http.get('/api/orders', () => {
    return HttpResponse.json([
      { id: '1', status: 'pending', totalPrice: { amount: 10000 } },
      { id: '2', status: 'completed', totalPrice: { amount: 20000 } },
    ]);
  }),

  http.post('/api/orders', async ({ request }) => {
    const body = await request.json();
    return HttpResponse.json(
      { id: '3', ...body, status: 'pending' },
      { status: 201 }
    );
  }),

  http.delete('/api/orders/:id', ({ params }) => {
    return HttpResponse.json({ success: true });
  }),
];
```

### 7.2 Server 설정

```typescript
// tests/mocks/server.ts
import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);
```

### 7.3 테스트 Setup

```typescript
// tests/setup.ts
import '@testing-library/jest-dom';
import { server } from './mocks/server';

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }));
afterEach(() => server.resetHandlers());
afterAll(() => server.close());
```

---

## 8. 테스트 Mock 팩토리

```typescript
// tests/mocks/orderMocks.ts
import { Order, OrderItem } from '@/features/orders/types';

export function createMockOrder(overrides?: Partial<Order>): Order {
  return {
    id: '1',
    status: 'pending',
    items: [createMockOrderItem()],
    totalPrice: { amount: 10000, currency: 'KRW' },
    createdAt: new Date('2024-01-01'),
    updatedAt: new Date('2024-01-01'),
    ...overrides,
  };
}

export function createMockOrderItem(overrides?: Partial<OrderItem>): OrderItem {
  return {
    id: '1',
    productId: 'product-1',
    productName: '테스트 상품',
    quantity: 1,
    price: { amount: 10000, currency: 'KRW' },
    ...overrides,
  };
}
```

---

## 9. 파일 구조

```
tests/
├── setup.ts
├── mocks/
│   ├── handlers.ts
│   ├── server.ts
│   └── orderMocks.ts
└── utils/
    └── renderWithProviders.tsx

features/
└── orders/
    ├── components/
    │   ├── OrderCard.tsx
    │   └── OrderCard.test.tsx  # 컴포넌트와 같은 위치
    ├── hooks/
    │   ├── useOrders.ts
    │   └── useOrders.test.ts
    └── store/
        ├── orderStore.ts
        └── orderStore.test.ts
```

---

## 10. 완료 체크리스트

- [ ] 컴포넌트 테스트 작성
- [ ] Hook 테스트 작성
- [ ] Store 테스트 작성
- [ ] 유틸리티 함수 테스트 작성
- [ ] MSW 모킹 설정
- [ ] Mock 팩토리 구현
- [ ] 커버리지 목표 달성 (80%+)

---

## 11. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 커버리지 미달 | Model/UI Worker | 테스트 추가 요청 |
| 테스트 불가 코드 | Architect | 리팩토링 검토 |
| Flaky 테스트 | Engineer Worker | 테스트 안정화 |
| 모킹 복잡 | Architect | 의존성 구조 검토 |

---

<!-- pal:convention:workers:frontend:unit-tc -->
