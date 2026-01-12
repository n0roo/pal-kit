# Tailwind CSS 가이드라인

> React 프로젝트에서 Tailwind CSS 사용 규칙

---

## 1. 개요

Tailwind CSS는 유틸리티 우선(Utility-First) CSS 프레임워크입니다.
일관된 스타일링을 위해 아래 가이드라인을 따릅니다.

---

## 2. 설치 및 설정

### 2.1 패키지 설치

```bash
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
```

### 2.2 tailwind.config.js

```javascript
/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './src/**/*.{js,ts,jsx,tsx,mdx}',
    './app/**/*.{js,ts,jsx,tsx,mdx}',
    './components/**/*.{js,ts,jsx,tsx,mdx}',
  ],
  theme: {
    extend: {
      colors: {
        primary: {
          50: '#eff6ff',
          100: '#dbeafe',
          500: '#3b82f6',
          600: '#2563eb',
          700: '#1d4ed8',
        },
        secondary: {
          500: '#8b5cf6',
          600: '#7c3aed',
        },
      },
      fontFamily: {
        sans: ['Pretendard', 'system-ui', 'sans-serif'],
      },
      spacing: {
        '18': '4.5rem',
        '88': '22rem',
      },
      borderRadius: {
        '4xl': '2rem',
      },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
    require('@tailwindcss/typography'),
  ],
  // MUI와 함께 사용 시
  important: '#__next', // 또는 true
  corePlugins: {
    preflight: false, // MUI CssBaseline 사용 시
  },
};
```

### 2.3 globals.css

```css
@tailwind base;
@tailwind components;
@tailwind utilities;

/* 커스텀 Base 스타일 */
@layer base {
  html {
    font-family: 'Pretendard', system-ui, sans-serif;
  }

  body {
    @apply bg-gray-50 text-gray-900;
  }
}

/* 커스텀 컴포넌트 */
@layer components {
  .btn-primary {
    @apply px-4 py-2 bg-primary-600 text-white rounded-lg
           hover:bg-primary-700 focus:ring-2 focus:ring-primary-500
           transition-colors duration-200;
  }

  .card {
    @apply bg-white rounded-xl shadow-sm border border-gray-100
           p-6 hover:shadow-md transition-shadow;
  }
}

/* 커스텀 유틸리티 */
@layer utilities {
  .text-balance {
    text-wrap: balance;
  }
}
```

---

## 3. 클래스 작성 규칙

### 3.1 클래스 순서

```typescript
// 권장 순서
<div
  className="
    // 1. 레이아웃 (display, position)
    flex items-center justify-between
    absolute top-0 left-0

    // 2. 박스 모델 (width, height, padding, margin)
    w-full h-16 p-4 mt-2

    // 3. 타이포그래피
    text-lg font-semibold text-gray-900

    // 4. 배경/테두리
    bg-white border border-gray-200 rounded-lg

    // 5. 효과 (shadow, opacity)
    shadow-sm opacity-90

    // 6. 상태 (hover, focus, active)
    hover:bg-gray-50 focus:ring-2

    // 7. 반응형 (sm:, md:, lg:)
    sm:flex-row md:w-1/2

    // 8. 다크 모드
    dark:bg-gray-800 dark:text-white
  "
/>
```

### 3.2 cn() 유틸리티 사용

```typescript
// lib/utils.ts
import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

// 사용
import { cn } from '@/lib/utils';

interface ButtonProps {
  variant?: 'primary' | 'secondary';
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}

function Button({ variant = 'primary', size = 'md', className }: ButtonProps) {
  return (
    <button
      className={cn(
        'rounded-lg font-medium transition-colors',
        {
          'bg-primary-600 text-white hover:bg-primary-700': variant === 'primary',
          'bg-gray-100 text-gray-900 hover:bg-gray-200': variant === 'secondary',
        },
        {
          'px-3 py-1.5 text-sm': size === 'sm',
          'px-4 py-2 text-base': size === 'md',
          'px-6 py-3 text-lg': size === 'lg',
        },
        className
      )}
    />
  );
}
```

---

## 4. 레이아웃 패턴

### 4.1 Flexbox

```typescript
// 수평 중앙 정렬
<div className="flex items-center justify-center">

// 수직 배치, 간격
<div className="flex flex-col gap-4">

// 양쪽 정렬
<div className="flex items-center justify-between">

// Wrap
<div className="flex flex-wrap gap-2">
```

### 4.2 Grid

```typescript
// 기본 그리드
<div className="grid grid-cols-3 gap-4">

// 반응형 그리드
<div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">

// 자동 맞춤
<div className="grid grid-cols-[repeat(auto-fill,minmax(250px,1fr))] gap-4">
```

### 4.3 Container

```typescript
// 중앙 정렬 컨테이너
<div className="container mx-auto px-4">

// 최대 너비 지정
<div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8">
```

---

## 5. 반응형 처리

### 5.1 Breakpoints

```typescript
// 기본 breakpoints
// sm: 640px
// md: 768px
// lg: 1024px
// xl: 1280px
// 2xl: 1536px

// 모바일 우선 (기본 → sm → md → lg)
<div className="
  w-full         // 모바일: 전체 너비
  sm:w-1/2       // sm 이상: 50%
  md:w-1/3       // md 이상: 33%
  lg:w-1/4       // lg 이상: 25%
">
```

### 5.2 반응형 패턴

```typescript
// 모바일: 세로, 데스크톱: 가로
<div className="flex flex-col md:flex-row gap-4">

// 모바일에서 숨김
<div className="hidden md:block">

// 모바일에서만 표시
<div className="block md:hidden">

// 반응형 텍스트
<h1 className="text-2xl sm:text-3xl md:text-4xl lg:text-5xl">
```

---

## 6. 다크 모드

### 6.1 설정

```javascript
// tailwind.config.js
module.exports = {
  darkMode: 'class', // 또는 'media'
};
```

### 6.2 사용

```typescript
// 다크 모드 클래스
<div className="bg-white dark:bg-gray-900 text-gray-900 dark:text-white">

// 다크 모드 토글
function ThemeToggle() {
  const [isDark, setIsDark] = useState(false);

  useEffect(() => {
    document.documentElement.classList.toggle('dark', isDark);
  }, [isDark]);

  return (
    <button onClick={() => setIsDark(!isDark)}>
      {isDark ? '라이트 모드' : '다크 모드'}
    </button>
  );
}
```

---

## 7. 애니메이션

### 7.1 Transition

```typescript
// 기본 transition
<button className="transition-colors duration-200 hover:bg-gray-100">

// 여러 속성
<div className="transition-all duration-300 ease-in-out hover:scale-105 hover:shadow-lg">
```

### 7.2 Animation

```typescript
// 내장 애니메이션
<div className="animate-spin">  // 회전
<div className="animate-ping">  // 펄스
<div className="animate-pulse"> // 페이드
<div className="animate-bounce"> // 바운스

// 커스텀 애니메이션
// tailwind.config.js
module.exports = {
  theme: {
    extend: {
      keyframes: {
        'fade-in': {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' },
        },
        'slide-up': {
          '0%': { transform: 'translateY(10px)', opacity: '0' },
          '100%': { transform: 'translateY(0)', opacity: '1' },
        },
      },
      animation: {
        'fade-in': 'fade-in 0.3s ease-out',
        'slide-up': 'slide-up 0.4s ease-out',
      },
    },
  },
};
```

---

## 8. 컴포넌트 패턴

### 8.1 Card

```typescript
function Card({ children, className }: CardProps) {
  return (
    <div
      className={cn(
        'bg-white rounded-xl shadow-sm border border-gray-100',
        'p-6 hover:shadow-md transition-shadow',
        'dark:bg-gray-800 dark:border-gray-700',
        className
      )}
    >
      {children}
    </div>
  );
}
```

### 8.2 Badge

```typescript
const badgeVariants = {
  default: 'bg-gray-100 text-gray-800',
  primary: 'bg-primary-100 text-primary-800',
  success: 'bg-green-100 text-green-800',
  warning: 'bg-yellow-100 text-yellow-800',
  error: 'bg-red-100 text-red-800',
};

function Badge({ variant = 'default', children }: BadgeProps) {
  return (
    <span
      className={cn(
        'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
        badgeVariants[variant]
      )}
    >
      {children}
    </span>
  );
}
```

### 8.3 Input

```typescript
function Input({ label, error, ...props }: InputProps) {
  return (
    <div className="space-y-1">
      {label && (
        <label className="block text-sm font-medium text-gray-700">
          {label}
        </label>
      )}
      <input
        className={cn(
          'block w-full rounded-lg border px-3 py-2',
          'focus:outline-none focus:ring-2 focus:ring-primary-500',
          'transition-colors duration-200',
          error
            ? 'border-red-300 focus:border-red-500 focus:ring-red-500'
            : 'border-gray-300 focus:border-primary-500'
        )}
        {...props}
      />
      {error && (
        <p className="text-sm text-red-600">{error}</p>
      )}
    </div>
  );
}
```

---

## 9. MUI와 혼용 시 주의사항

### 9.1 충돌 방지

```javascript
// tailwind.config.js
module.exports = {
  important: '#root', // 또는 '#__next'
  corePlugins: {
    preflight: false, // MUI CssBaseline 사용 시
  },
};
```

### 9.2 역할 분담

| Tailwind | MUI |
|----------|-----|
| 레이아웃 (flex, grid) | 테마 색상 |
| 간격 (p, m, gap) | 복잡한 컴포넌트 |
| 반응형 | Dialog, Snackbar |
| 유틸리티 | Form 컴포넌트 |

### 9.3 혼용 예시

```typescript
<Card className="space-y-4">
  <Typography variant="h6" className="font-bold">
    Title
  </Typography>
  <div className="grid grid-cols-2 gap-4">
    <TextField label="Name" fullWidth />
    <TextField label="Email" fullWidth />
  </div>
  <Button
    variant="contained"
    className="w-full sm:w-auto"
  >
    Submit
  </Button>
</Card>
```

---

## 10. Best Practices

### 10.1 권장 사항

- 유틸리티 클래스 우선 사용
- cn() 유틸리티로 조건부 클래스
- 반응형은 모바일 우선
- 커스텀 유틸리티 최소화

### 10.2 지양 사항

- @apply 과도한 사용 지양
- 인라인 스타일 혼용 지양
- !important 사용 지양
- 임의 값 남발 지양 (w-[137px])

---

<!-- pal:convention:ui:tailwind -->
