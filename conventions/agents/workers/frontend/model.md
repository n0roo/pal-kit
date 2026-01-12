# Component Model Worker 컨벤션

> Logic 레이어 - 상태 관리 및 비즈니스 로직 전문 Worker

---

## 1. 역할 정의

Component Model Worker는 **Logic 레이어**에서 상태 관리, 비즈니스 로직, 데이터 변환을 담당하는 전문 Worker입니다.

### 1.1 담당 영역

- 전역/로컬 상태 관리
- Custom Hooks 구현
- 비즈니스 로직 처리
- 데이터 변환/정규화
- Form 상태 관리
- 유효성 검증 로직

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| 상태 관리 | Zustand, Jotai, Redux Toolkit |
| Form | React Hook Form, Zod |
| 유틸리티 | date-fns, lodash-es |

---

## 2. 상태 관리 규칙

### 2.1 Zustand Store

```typescript
// features/orders/store/orderStore.ts
import { create } from 'zustand';
import { devtools, persist } from 'zustand/middleware';
import { Order, OrderFilter } from '../types';

interface OrderState {
  orders: Order[];
  selectedOrder: Order | null;
  filter: OrderFilter;
  isLoading: boolean;
}

interface OrderActions {
  setOrders: (orders: Order[]) => void;
  selectOrder: (order: Order | null) => void;
  updateFilter: (filter: Partial<OrderFilter>) => void;
  clearFilter: () => void;
}

const initialFilter: OrderFilter = {
  status: 'all',
  dateRange: null,
  searchQuery: '',
};

export const useOrderStore = create<OrderState & OrderActions>()(
  devtools(
    persist(
      (set) => ({
        // State
        orders: [],
        selectedOrder: null,
        filter: initialFilter,
        isLoading: false,

        // Actions
        setOrders: (orders) => set({ orders }),
        selectOrder: (order) => set({ selectedOrder: order }),
        updateFilter: (filter) =>
          set((state) => ({
            filter: { ...state.filter, ...filter },
          })),
        clearFilter: () => set({ filter: initialFilter }),
      }),
      { name: 'order-store' }
    )
  )
);
```

### 2.2 Jotai Atoms

```typescript
// features/orders/atoms/orderAtoms.ts
import { atom } from 'jotai';
import { atomWithStorage } from 'jotai/utils';
import { Order, OrderFilter } from '../types';

// 기본 atom
export const ordersAtom = atom<Order[]>([]);
export const selectedOrderAtom = atom<Order | null>(null);

// 영속 저장 atom
export const orderFilterAtom = atomWithStorage<OrderFilter>('order-filter', {
  status: 'all',
  dateRange: null,
  searchQuery: '',
});

// 파생 atom (읽기 전용)
export const filteredOrdersAtom = atom((get) => {
  const orders = get(ordersAtom);
  const filter = get(orderFilterAtom);

  return orders.filter((order) => {
    if (filter.status !== 'all' && order.status !== filter.status) {
      return false;
    }
    if (filter.searchQuery) {
      return order.id.includes(filter.searchQuery);
    }
    return true;
  });
});

// 파생 atom (읽기/쓰기)
export const orderCountAtom = atom(
  (get) => get(ordersAtom).length,
  (get, set, newOrders: Order[]) => set(ordersAtom, newOrders)
);
```

---

## 3. Custom Hooks 규칙

### 3.1 데이터 페칭 Hook

```typescript
// features/orders/hooks/useOrders.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ordersApi } from '../api/orders';
import { useOrderStore } from '../store/orderStore';
import { Order, CreateOrderRequest } from '../types';

export function useOrders() {
  const setOrders = useOrderStore((state) => state.setOrders);

  return useQuery({
    queryKey: ['orders'],
    queryFn: ordersApi.getAll,
    onSuccess: (data) => {
      setOrders(data);
    },
  });
}

export function useCreateOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateOrderRequest) => ordersApi.create(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['orders'] });
    },
    onError: (error) => {
      console.error('주문 생성 실패:', error);
    },
  });
}
```

### 3.2 비즈니스 로직 Hook

```typescript
// features/orders/hooks/useOrderCalculation.ts
import { useMemo } from 'react';
import { Order, OrderItem } from '../types';

export function useOrderCalculation(items: OrderItem[]) {
  const subtotal = useMemo(() => {
    return items.reduce((sum, item) => sum + item.price * item.quantity, 0);
  }, [items]);

  const tax = useMemo(() => {
    return Math.floor(subtotal * 0.1);
  }, [subtotal]);

  const discount = useMemo(() => {
    // 10만원 이상 10% 할인
    if (subtotal >= 100000) {
      return Math.floor(subtotal * 0.1);
    }
    return 0;
  }, [subtotal]);

  const total = useMemo(() => {
    return subtotal + tax - discount;
  }, [subtotal, tax, discount]);

  return {
    subtotal,
    tax,
    discount,
    total,
    formattedTotal: formatCurrency(total),
  };
}

function formatCurrency(amount: number): string {
  return new Intl.NumberFormat('ko-KR', {
    style: 'currency',
    currency: 'KRW',
  }).format(amount);
}
```

### 3.3 UI 상태 Hook

```typescript
// features/orders/hooks/useOrderSelection.ts
import { useState, useCallback } from 'react';
import { Order } from '../types';

export function useOrderSelection(orders: Order[]) {
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const isSelected = useCallback(
    (id: string) => selectedIds.has(id),
    [selectedIds]
  );

  const toggle = useCallback((id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  }, []);

  const selectAll = useCallback(() => {
    setSelectedIds(new Set(orders.map((o) => o.id)));
  }, [orders]);

  const clearSelection = useCallback(() => {
    setSelectedIds(new Set());
  }, []);

  const selectedOrders = orders.filter((o) => selectedIds.has(o.id));

  return {
    selectedIds,
    selectedOrders,
    isSelected,
    toggle,
    selectAll,
    clearSelection,
    selectedCount: selectedIds.size,
    isAllSelected: selectedIds.size === orders.length,
  };
}
```

---

## 4. Form 상태 관리

### 4.1 React Hook Form + Zod

```typescript
// features/orders/hooks/useOrderForm.ts
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';

const orderSchema = z.object({
  productId: z.string().min(1, '상품을 선택해주세요'),
  quantity: z.number().min(1, '최소 1개 이상').max(100, '최대 100개까지'),
  shippingAddress: z.object({
    address: z.string().min(1, '주소를 입력해주세요'),
    city: z.string().min(1, '도시를 입력해주세요'),
    zipCode: z.string().regex(/^\d{5}$/, '올바른 우편번호를 입력해주세요'),
  }),
  paymentMethod: z.enum(['card', 'bank', 'kakao'], {
    errorMap: () => ({ message: '결제 수단을 선택해주세요' }),
  }),
});

type OrderFormData = z.infer<typeof orderSchema>;

export function useOrderForm(defaultValues?: Partial<OrderFormData>) {
  const form = useForm<OrderFormData>({
    resolver: zodResolver(orderSchema),
    defaultValues: {
      productId: '',
      quantity: 1,
      shippingAddress: {
        address: '',
        city: '',
        zipCode: '',
      },
      paymentMethod: undefined,
      ...defaultValues,
    },
  });

  const onSubmit = form.handleSubmit((data) => {
    console.log('Form submitted:', data);
    // API 호출
  });

  return {
    ...form,
    onSubmit,
  };
}
```

### 4.2 Form 유효성 검증

```typescript
// features/orders/validation/orderValidation.ts
import { z } from 'zod';

export const orderItemSchema = z.object({
  productId: z.string(),
  quantity: z.number().positive(),
  price: z.number().nonnegative(),
});

export const orderSchema = z.object({
  items: z.array(orderItemSchema).min(1, '최소 1개 상품이 필요합니다'),
  totalAmount: z.number().positive(),
});

export function validateOrder(data: unknown) {
  return orderSchema.safeParse(data);
}
```

---

## 5. 데이터 변환

### 5.1 DTO 변환

```typescript
// features/orders/utils/orderTransformers.ts
import { Order, OrderResponse, OrderViewModel } from '../types';

export function toOrder(response: OrderResponse): Order {
  return {
    id: response.id,
    status: response.status,
    items: response.items.map(toOrderItem),
    totalPrice: {
      amount: response.total_price.amount,
      currency: response.total_price.currency,
    },
    createdAt: new Date(response.created_at),
    updatedAt: new Date(response.updated_at),
  };
}

export function toOrderViewModel(order: Order): OrderViewModel {
  return {
    ...order,
    statusLabel: getStatusLabel(order.status),
    formattedPrice: formatPrice(order.totalPrice),
    formattedDate: formatDate(order.createdAt),
    canCancel: order.status === 'pending',
    canRefund: order.status === 'completed',
  };
}

function getStatusLabel(status: string): string {
  const labels: Record<string, string> = {
    pending: '대기중',
    processing: '처리중',
    completed: '완료',
    cancelled: '취소됨',
  };
  return labels[status] || status;
}
```

### 5.2 데이터 정규화

```typescript
// features/orders/utils/normalizeOrders.ts
import { normalize, schema } from 'normalizr';

const orderItem = new schema.Entity('orderItems');
const order = new schema.Entity('orders', {
  items: [orderItem],
});

export function normalizeOrders(orders: Order[]) {
  return normalize(orders, [order]);
}

// 결과:
// {
//   entities: {
//     orders: { 1: {...}, 2: {...} },
//     orderItems: { 101: {...}, 102: {...} }
//   },
//   result: [1, 2]
// }
```

---

## 6. 유틸리티 함수

### 6.1 날짜 포맷팅

```typescript
// shared/utils/dateUtils.ts
import { format, formatDistanceToNow, isToday, isYesterday } from 'date-fns';
import { ko } from 'date-fns/locale';

export function formatDate(date: Date): string {
  if (isToday(date)) {
    return `오늘 ${format(date, 'HH:mm')}`;
  }
  if (isYesterday(date)) {
    return `어제 ${format(date, 'HH:mm')}`;
  }
  return format(date, 'yyyy년 M월 d일 HH:mm', { locale: ko });
}

export function formatRelativeTime(date: Date): string {
  return formatDistanceToNow(date, { addSuffix: true, locale: ko });
}
```

### 6.2 가격 포맷팅

```typescript
// shared/utils/priceUtils.ts
import { Money } from '@/types';

export function formatPrice(money: Money): string {
  return new Intl.NumberFormat('ko-KR', {
    style: 'currency',
    currency: money.currency,
  }).format(money.amount);
}

export function parsePrice(value: string): number {
  return parseInt(value.replace(/[^0-9]/g, ''), 10) || 0;
}
```

---

## 7. 타입 정의

```typescript
// features/orders/types/index.ts
export interface Order {
  id: string;
  status: OrderStatus;
  items: OrderItem[];
  totalPrice: Money;
  createdAt: Date;
  updatedAt: Date;
}

export type OrderStatus = 'pending' | 'processing' | 'completed' | 'cancelled';

export interface OrderItem {
  id: string;
  productId: string;
  productName: string;
  quantity: number;
  price: Money;
}

export interface Money {
  amount: number;
  currency: string;
}

export interface OrderFilter {
  status: OrderStatus | 'all';
  dateRange: DateRange | null;
  searchQuery: string;
}

export interface OrderViewModel extends Order {
  statusLabel: string;
  formattedPrice: string;
  formattedDate: string;
  canCancel: boolean;
  canRefund: boolean;
}
```

---

## 8. 파일 구조

```
features/
└── orders/
    ├── store/
    │   └── orderStore.ts
    ├── atoms/
    │   └── orderAtoms.ts
    ├── hooks/
    │   ├── useOrders.ts
    │   ├── useOrderForm.ts
    │   └── useOrderCalculation.ts
    ├── utils/
    │   ├── orderTransformers.ts
    │   └── normalizeOrders.ts
    ├── validation/
    │   └── orderValidation.ts
    └── types/
        └── index.ts
```

---

## 9. 완료 체크리스트

- [ ] 상태 관리 Store/Atoms 구현
- [ ] Custom Hooks 구현
- [ ] Form 상태 관리 구현
- [ ] 유효성 검증 로직 구현
- [ ] 데이터 변환 유틸리티 구현
- [ ] 타입 정의
- [ ] 단위 테스트 작성

---

## 10. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 상태 구조 복잡 | Architect | 상태 리팩토링 |
| API 응답 변경 | Backend Worker | 변환 로직 수정 |
| 성능 이슈 | Architect | 메모이제이션 검토 |
| 비즈니스 로직 불명확 | User | 요구사항 명확화 |

---

<!-- pal:convention:workers:frontend:model -->
