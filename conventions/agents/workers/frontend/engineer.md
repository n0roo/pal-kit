# Frontend Engineer Worker 컨벤션

> Orchestration 레이어 - 프론트엔드 아키텍처 및 통합 전문 Worker

---

## 1. 역할 정의

Frontend Engineer Worker는 **Orchestration 레이어**에서 프론트엔드 아키텍처 설계, 페이지 구성, 라우팅을 담당하는 전문 Worker입니다.

### 1.1 담당 영역

- 페이지 컴포넌트 구성
- 라우팅 설정
- 레이아웃 시스템
- 상태 관리 구조
- API 통합
- 전역 에러 핸들링

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| 프레임워크 | React, Next.js, Vue |
| 라우팅 | React Router, Next.js App Router |
| 상태 관리 | Zustand, Jotai, Redux Toolkit |
| API | TanStack Query, SWR |

---

## 2. 프론트엔드 레이어 구조

### 2.1 PA-Layered 프론트엔드 매핑

```
Orchestration (Engineer Worker)
├── Pages/Routes
├── Layouts
└── API Integration

Logic (Model Worker)
├── State Management
├── Business Logic
└── Data Transformation

View (UI Worker)
├── UI Components
├── Design System
└── Styling
```

### 2.2 의존성 규칙

```
Orchestration은:
✅ Logic 레이어 참조 가능
✅ View 레이어 참조 가능
❌ 다른 Feature 직접 참조 지양 (공유 로직 통해서)
```

---

## 3. 페이지 구성 규칙

### 3.1 Next.js App Router 페이지

```typescript
// app/orders/page.tsx
import { Suspense } from 'react';
import { OrderList } from '@/features/orders/components/OrderList';
import { OrderListSkeleton } from '@/features/orders/components/OrderListSkeleton';
import { getOrders } from '@/features/orders/api/getOrders';

export default async function OrdersPage() {
  const orders = await getOrders();

  return (
    <main className="container mx-auto py-8">
      <h1 className="text-2xl font-bold mb-6">주문 내역</h1>
      <Suspense fallback={<OrderListSkeleton />}>
        <OrderList initialData={orders} />
      </Suspense>
    </main>
  );
}
```

### 3.2 React Router 페이지

```typescript
// pages/OrdersPage.tsx
import { useOrders } from '@/features/orders/hooks/useOrders';
import { OrderList } from '@/features/orders/components/OrderList';
import { PageLayout } from '@/shared/layouts/PageLayout';
import { LoadingSpinner } from '@/shared/components/LoadingSpinner';

export function OrdersPage() {
  const { data: orders, isLoading, error } = useOrders();

  if (isLoading) return <LoadingSpinner />;
  if (error) return <ErrorMessage error={error} />;

  return (
    <PageLayout title="주문 내역">
      <OrderList orders={orders} />
    </PageLayout>
  );
}
```

---

## 4. 라우팅 설정

### 4.1 Next.js App Router 구조

```
app/
├── layout.tsx           # Root Layout
├── page.tsx             # Home
├── (auth)/
│   ├── login/page.tsx
│   └── register/page.tsx
├── (main)/
│   ├── layout.tsx       # Main Layout
│   ├── orders/
│   │   ├── page.tsx     # Order List
│   │   └── [id]/page.tsx # Order Detail
│   └── products/
│       └── page.tsx
└── api/
    └── orders/route.ts  # API Route
```

### 4.2 라우트 가드

```typescript
// middleware.ts
import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

export function middleware(request: NextRequest) {
  const token = request.cookies.get('auth-token');
  const isAuthPage = request.nextUrl.pathname.startsWith('/login');
  const isProtectedPage = request.nextUrl.pathname.startsWith('/orders');

  if (isProtectedPage && !token) {
    return NextResponse.redirect(new URL('/login', request.url));
  }

  if (isAuthPage && token) {
    return NextResponse.redirect(new URL('/orders', request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/orders/:path*', '/login', '/register'],
};
```

---

## 5. 레이아웃 시스템

### 5.1 Root Layout

```typescript
// app/layout.tsx
import { Inter } from 'next/font/google';
import { Providers } from '@/app/providers';
import { Header } from '@/shared/components/Header';
import '@/styles/globals.css';

const inter = Inter({ subsets: ['latin'] });

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="ko">
      <body className={inter.className}>
        <Providers>
          <Header />
          {children}
        </Providers>
      </body>
    </html>
  );
}
```

### 5.2 Feature Layout

```typescript
// app/(main)/orders/layout.tsx
import { OrderSidebar } from '@/features/orders/components/OrderSidebar';

export default function OrdersLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex">
      <OrderSidebar />
      <main className="flex-1 p-6">{children}</main>
    </div>
  );
}
```

---

## 6. API 통합

### 6.1 API Client 설정

```typescript
// lib/api/client.ts
import axios from 'axios';

export const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request Interceptor
apiClient.interceptors.request.use((config) => {
  const token = getAuthToken();
  if (token) {
    config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

// Response Interceptor
apiClient.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // 토큰 만료 처리
      clearAuthToken();
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);
```

### 6.2 API 함수

```typescript
// features/orders/api/orders.ts
import { apiClient } from '@/lib/api/client';
import { Order, CreateOrderRequest } from '../types';

export const ordersApi = {
  getAll: () =>
    apiClient.get<Order[]>('/api/v1/orders').then((res) => res.data),

  getById: (id: string) =>
    apiClient.get<Order>(`/api/v1/orders/${id}`).then((res) => res.data),

  create: (data: CreateOrderRequest) =>
    apiClient.post<Order>('/api/v1/orders', data).then((res) => res.data),

  cancel: (id: string) =>
    apiClient.post(`/api/v1/orders/${id}/cancel`).then((res) => res.data),
};
```

### 6.3 TanStack Query 통합

```typescript
// features/orders/hooks/useOrders.ts
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { ordersApi } from '../api/orders';

export const orderKeys = {
  all: ['orders'] as const,
  detail: (id: string) => ['orders', id] as const,
};

export function useOrders() {
  return useQuery({
    queryKey: orderKeys.all,
    queryFn: ordersApi.getAll,
  });
}

export function useOrder(id: string) {
  return useQuery({
    queryKey: orderKeys.detail(id),
    queryFn: () => ordersApi.getById(id),
  });
}

export function useCreateOrder() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ordersApi.create,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: orderKeys.all });
    },
  });
}
```

---

## 7. 전역 에러 핸들링

### 7.1 Error Boundary

```typescript
// app/error.tsx
'use client';

import { useEffect } from 'react';
import { Button } from '@/shared/components/Button';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // 에러 로깅 서비스로 전송
    console.error(error);
  }, [error]);

  return (
    <div className="flex flex-col items-center justify-center min-h-screen">
      <h2 className="text-2xl font-bold mb-4">문제가 발생했습니다</h2>
      <p className="text-gray-600 mb-6">{error.message}</p>
      <Button onClick={reset}>다시 시도</Button>
    </div>
  );
}
```

### 7.2 Not Found 페이지

```typescript
// app/not-found.tsx
import Link from 'next/link';

export default function NotFound() {
  return (
    <div className="flex flex-col items-center justify-center min-h-screen">
      <h1 className="text-6xl font-bold text-gray-300">404</h1>
      <h2 className="text-2xl font-semibold mt-4">페이지를 찾을 수 없습니다</h2>
      <Link href="/" className="mt-6 text-blue-600 hover:underline">
        홈으로 돌아가기
      </Link>
    </div>
  );
}
```

---

## 8. Providers 구성

```typescript
// app/providers.tsx
'use client';

import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ThemeProvider } from 'next-themes';
import { useState } from 'react';

export function Providers({ children }: { children: React.ReactNode }) {
  const [queryClient] = useState(
    () =>
      new QueryClient({
        defaultOptions: {
          queries: {
            staleTime: 60 * 1000,
            retry: 1,
          },
        },
      })
  );

  return (
    <QueryClientProvider client={queryClient}>
      <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
        {children}
      </ThemeProvider>
    </QueryClientProvider>
  );
}
```

---

## 9. 파일 구조

```
src/
├── app/                    # Next.js App Router
│   ├── layout.tsx
│   ├── page.tsx
│   ├── providers.tsx
│   └── (main)/
│       └── orders/
├── features/               # Feature 모듈
│   └── orders/
│       ├── api/
│       ├── components/
│       ├── hooks/
│       └── types/
├── shared/                 # 공유 모듈
│   ├── components/
│   ├── hooks/
│   └── layouts/
└── lib/                    # 유틸리티
    ├── api/
    └── utils/
```

---

## 10. 완료 체크리스트

- [ ] 페이지 컴포넌트 구성
- [ ] 라우팅 설정
- [ ] 레이아웃 시스템 구현
- [ ] API 클라이언트 설정
- [ ] TanStack Query 통합
- [ ] 에러 핸들링 구현
- [ ] Providers 구성
- [ ] 단위 테스트 작성

---

## 11. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 라우팅 구조 변경 | Architect | 아키텍처 검토 |
| API 스펙 불일치 | Backend Worker | API 수정 요청 |
| 성능 이슈 | Architect | 최적화 전략 검토 |
| 상태 관리 복잡 | Architect | 상태 구조 재설계 |

---

<!-- pal:convention:workers:frontend:engineer -->
