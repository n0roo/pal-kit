# Component UI Worker 컨벤션

> View 레이어 - UI 컴포넌트 전문 Worker

---

## 1. 역할 정의

Component UI Worker는 **View 레이어**에서 UI 컴포넌트, 디자인 시스템, 스타일링을 담당하는 전문 Worker입니다.

### 1.1 담당 영역

- 재사용 가능한 UI 컴포넌트
- 디자인 시스템 컴포넌트
- 스타일링 (CSS/Tailwind)
- 애니메이션/트랜지션
- 접근성(a11y) 구현

### 1.2 기술 스택

| 항목 | 기술 |
|------|------|
| 컴포넌트 | React, Radix UI |
| 스타일링 | Tailwind CSS, CSS Modules |
| 애니메이션 | Framer Motion |
| 아이콘 | Lucide Icons |

---

## 2. 컴포넌트 설계 원칙

### 2.1 컴포넌트 분류

```
Shared Components (공유)
├── Primitive: Button, Input, Select
├── Composite: Card, Modal, Toast
└── Layout: Container, Stack, Grid

Feature Components (기능별)
├── OrderCard
├── OrderList
└── OrderDetail
```

### 2.2 컴포넌트 설계 규칙

| 원칙 | 설명 |
|------|------|
| 단일 책임 | 하나의 역할만 수행 |
| 합성 가능 | 작은 컴포넌트 조합 |
| 제어/비제어 | 두 패턴 모두 지원 |
| 접근성 | ARIA 속성 필수 |

---

## 3. Shared 컴포넌트

### 3.1 Button

```typescript
// shared/components/Button/Button.tsx
import { forwardRef } from 'react';
import { cva, type VariantProps } from 'class-variance-authority';
import { cn } from '@/lib/utils';

const buttonVariants = cva(
  'inline-flex items-center justify-center rounded-md font-medium transition-colors focus-visible:outline-none focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default: 'bg-primary text-primary-foreground hover:bg-primary/90',
        secondary: 'bg-secondary text-secondary-foreground hover:bg-secondary/80',
        outline: 'border border-input bg-background hover:bg-accent',
        ghost: 'hover:bg-accent hover:text-accent-foreground',
        destructive: 'bg-destructive text-destructive-foreground hover:bg-destructive/90',
      },
      size: {
        sm: 'h-8 px-3 text-sm',
        md: 'h-10 px-4 text-sm',
        lg: 'h-12 px-6 text-base',
        icon: 'h-10 w-10',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'md',
    },
  }
);

interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  isLoading?: boolean;
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ className, variant, size, isLoading, children, disabled, ...props }, ref) => {
    return (
      <button
        ref={ref}
        className={cn(buttonVariants({ variant, size }), className)}
        disabled={disabled || isLoading}
        {...props}
      >
        {isLoading ? (
          <>
            <Spinner className="mr-2 h-4 w-4" />
            Loading...
          </>
        ) : (
          children
        )}
      </button>
    );
  }
);

Button.displayName = 'Button';
```

### 3.2 Input

```typescript
// shared/components/Input/Input.tsx
import { forwardRef } from 'react';
import { cn } from '@/lib/utils';

interface InputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  helperText?: string;
}

export const Input = forwardRef<HTMLInputElement, InputProps>(
  ({ className, label, error, helperText, id, ...props }, ref) => {
    const inputId = id || `input-${Math.random().toString(36).slice(2)}`;

    return (
      <div className="space-y-1">
        {label && (
          <label
            htmlFor={inputId}
            className="text-sm font-medium text-gray-700"
          >
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          className={cn(
            'flex h-10 w-full rounded-md border border-input bg-background px-3 py-2 text-sm',
            'placeholder:text-muted-foreground',
            'focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring',
            'disabled:cursor-not-allowed disabled:opacity-50',
            error && 'border-destructive focus-visible:ring-destructive',
            className
          )}
          aria-invalid={!!error}
          aria-describedby={error ? `${inputId}-error` : undefined}
          {...props}
        />
        {error && (
          <p id={`${inputId}-error`} className="text-sm text-destructive">
            {error}
          </p>
        )}
        {helperText && !error && (
          <p className="text-sm text-muted-foreground">{helperText}</p>
        )}
      </div>
    );
  }
);

Input.displayName = 'Input';
```

### 3.3 Modal

```typescript
// shared/components/Modal/Modal.tsx
import { Fragment } from 'react';
import { Dialog, Transition } from '@headlessui/react';
import { X } from 'lucide-react';
import { cn } from '@/lib/utils';

interface ModalProps {
  isOpen: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  children: React.ReactNode;
  size?: 'sm' | 'md' | 'lg' | 'xl';
}

const sizeClasses = {
  sm: 'max-w-sm',
  md: 'max-w-md',
  lg: 'max-w-lg',
  xl: 'max-w-xl',
};

export function Modal({
  isOpen,
  onClose,
  title,
  description,
  children,
  size = 'md',
}: ModalProps) {
  return (
    <Transition appear show={isOpen} as={Fragment}>
      <Dialog as="div" className="relative z-50" onClose={onClose}>
        {/* Backdrop */}
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-black/50" />
        </Transition.Child>

        {/* Modal */}
        <div className="fixed inset-0 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-4">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 scale-95"
              enterTo="opacity-100 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 scale-100"
              leaveTo="opacity-0 scale-95"
            >
              <Dialog.Panel
                className={cn(
                  'w-full rounded-lg bg-white p-6 shadow-xl',
                  sizeClasses[size]
                )}
              >
                {/* Header */}
                <div className="flex items-start justify-between">
                  {title && (
                    <Dialog.Title className="text-lg font-semibold">
                      {title}
                    </Dialog.Title>
                  )}
                  <button
                    onClick={onClose}
                    className="rounded-md p-1 hover:bg-gray-100"
                    aria-label="닫기"
                  >
                    <X className="h-5 w-5" />
                  </button>
                </div>

                {description && (
                  <Dialog.Description className="mt-2 text-sm text-gray-500">
                    {description}
                  </Dialog.Description>
                )}

                {/* Content */}
                <div className="mt-4">{children}</div>
              </Dialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </Dialog>
    </Transition>
  );
}
```

---

## 4. Feature 컴포넌트

### 4.1 OrderCard

```typescript
// features/orders/components/OrderCard/OrderCard.tsx
import { memo } from 'react';
import { Card, CardHeader, CardContent, CardFooter } from '@/shared/components/Card';
import { Badge } from '@/shared/components/Badge';
import { Button } from '@/shared/components/Button';
import { formatPrice, formatDate } from '@/shared/utils';
import { OrderViewModel } from '../../types';

interface OrderCardProps {
  order: OrderViewModel;
  onSelect?: () => void;
  onCancel?: () => void;
}

export const OrderCard = memo(function OrderCard({
  order,
  onSelect,
  onCancel,
}: OrderCardProps) {
  return (
    <Card className="hover:shadow-md transition-shadow">
      <CardHeader className="flex flex-row items-center justify-between">
        <div>
          <h3 className="font-semibold">주문 #{order.id}</h3>
          <p className="text-sm text-gray-500">{order.formattedDate}</p>
        </div>
        <Badge variant={getStatusVariant(order.status)}>
          {order.statusLabel}
        </Badge>
      </CardHeader>

      <CardContent>
        <ul className="space-y-2">
          {order.items.slice(0, 2).map((item) => (
            <li key={item.id} className="flex justify-between text-sm">
              <span>{item.productName}</span>
              <span className="text-gray-500">x{item.quantity}</span>
            </li>
          ))}
          {order.items.length > 2 && (
            <li className="text-sm text-gray-500">
              외 {order.items.length - 2}개 상품
            </li>
          )}
        </ul>
      </CardContent>

      <CardFooter className="flex justify-between items-center">
        <span className="font-semibold">{order.formattedPrice}</span>
        <div className="space-x-2">
          {order.canCancel && (
            <Button variant="outline" size="sm" onClick={onCancel}>
              취소
            </Button>
          )}
          <Button size="sm" onClick={onSelect}>
            상세보기
          </Button>
        </div>
      </CardFooter>
    </Card>
  );
});

function getStatusVariant(status: string) {
  const variants: Record<string, 'default' | 'success' | 'warning' | 'destructive'> = {
    pending: 'warning',
    processing: 'default',
    completed: 'success',
    cancelled: 'destructive',
  };
  return variants[status] || 'default';
}
```

### 4.2 OrderList

```typescript
// features/orders/components/OrderList/OrderList.tsx
import { OrderCard } from '../OrderCard';
import { OrderListSkeleton } from './OrderListSkeleton';
import { EmptyState } from '@/shared/components/EmptyState';
import { OrderViewModel } from '../../types';

interface OrderListProps {
  orders: OrderViewModel[];
  isLoading?: boolean;
  onSelectOrder: (order: OrderViewModel) => void;
  onCancelOrder: (orderId: string) => void;
}

export function OrderList({
  orders,
  isLoading,
  onSelectOrder,
  onCancelOrder,
}: OrderListProps) {
  if (isLoading) {
    return <OrderListSkeleton count={3} />;
  }

  if (orders.length === 0) {
    return (
      <EmptyState
        title="주문 내역이 없습니다"
        description="아직 주문한 상품이 없습니다. 쇼핑을 시작해보세요!"
        action={{ label: '쇼핑하러 가기', href: '/products' }}
      />
    );
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
      {orders.map((order) => (
        <OrderCard
          key={order.id}
          order={order}
          onSelect={() => onSelectOrder(order)}
          onCancel={() => onCancelOrder(order.id)}
        />
      ))}
    </div>
  );
}
```

---

## 5. 스타일링 규칙

### 5.1 Tailwind 클래스 순서

```typescript
// 권장 순서
<div
  className={cn(
    // 1. Layout
    'flex flex-col',
    // 2. Sizing
    'w-full h-auto min-h-[100px]',
    // 3. Spacing
    'p-4 m-2 gap-2',
    // 4. Typography
    'text-sm font-medium text-gray-900',
    // 5. Background
    'bg-white',
    // 6. Border
    'border border-gray-200 rounded-lg',
    // 7. Effects
    'shadow-sm',
    // 8. Transitions
    'transition-all duration-200',
    // 9. States
    'hover:shadow-md focus:ring-2',
    // 10. Responsive
    'md:flex-row lg:p-6'
  )}
/>
```

### 5.2 cn 유틸리티

```typescript
// lib/utils.ts
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
```

---

## 6. 애니메이션

### 6.1 Framer Motion

```typescript
// shared/components/AnimatedCard/AnimatedCard.tsx
import { motion } from 'framer-motion';

interface AnimatedCardProps {
  children: React.ReactNode;
  delay?: number;
}

export function AnimatedCard({ children, delay = 0 }: AnimatedCardProps) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 20 }}
      animate={{ opacity: 1, y: 0 }}
      exit={{ opacity: 0, y: -20 }}
      transition={{ duration: 0.3, delay }}
    >
      {children}
    </motion.div>
  );
}
```

### 6.2 애니메이션 프리셋

```typescript
// shared/animations/presets.ts
export const fadeIn = {
  initial: { opacity: 0 },
  animate: { opacity: 1 },
  exit: { opacity: 0 },
};

export const slideUp = {
  initial: { opacity: 0, y: 20 },
  animate: { opacity: 1, y: 0 },
  exit: { opacity: 0, y: -20 },
};

export const scaleIn = {
  initial: { opacity: 0, scale: 0.95 },
  animate: { opacity: 1, scale: 1 },
  exit: { opacity: 0, scale: 0.95 },
};

export const staggerChildren = {
  animate: { transition: { staggerChildren: 0.1 } },
};
```

---

## 7. 접근성(a11y)

### 7.1 접근성 체크리스트

- [ ] 모든 이미지에 alt 속성
- [ ] 폼 요소에 label 연결
- [ ] 키보드 네비게이션 지원
- [ ] 충분한 색상 대비
- [ ] ARIA 속성 적절히 사용
- [ ] 포커스 표시 명확

### 7.2 접근성 패턴

```typescript
// 키보드 네비게이션
function handleKeyDown(e: React.KeyboardEvent) {
  switch (e.key) {
    case 'Enter':
    case ' ':
      e.preventDefault();
      onSelect();
      break;
    case 'Escape':
      onClose();
      break;
  }
}

// ARIA 속성
<button
  aria-label="장바구니에 추가"
  aria-pressed={isAdded}
  aria-describedby="cart-tooltip"
>
  <CartIcon />
</button>
```

---

## 8. 파일 구조

```
shared/
├── components/
│   ├── Button/
│   │   ├── Button.tsx
│   │   ├── Button.test.tsx
│   │   └── index.ts
│   ├── Input/
│   ├── Modal/
│   └── Card/
├── animations/
│   └── presets.ts
└── styles/
    └── globals.css

features/
└── orders/
    └── components/
        ├── OrderCard/
        ├── OrderList/
        └── OrderDetail/
```

---

## 9. 완료 체크리스트

- [ ] Shared 컴포넌트 구현
- [ ] Feature 컴포넌트 구현
- [ ] 스타일링 적용
- [ ] 애니메이션 구현
- [ ] 접근성 검증
- [ ] 반응형 디자인
- [ ] Storybook 스토리 작성 (선택)
- [ ] 단위 테스트 작성

---

## 10. 에스컬레이션 기준

| 상황 | 대상 | 조치 |
|------|------|------|
| 디자인 변경 | Designer | 디자인 확인 |
| 접근성 이슈 | Architect | a11y 가이드 검토 |
| 성능 이슈 | Architect | 렌더링 최적화 |
| 애니메이션 복잡 | Architect | 애니메이션 전략 |

---

<!-- pal:convention:workers:frontend:ui -->
