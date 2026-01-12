# MUI (Material UI) 가이드라인

> React 프로젝트에서 MUI 사용 규칙

---

## 1. 개요

MUI는 Google의 Material Design을 구현한 React 컴포넌트 라이브러리입니다.
일관된 UI/UX를 위해 아래 가이드라인을 따릅니다.

---

## 2. 설치 및 설정

### 2.1 패키지 설치

```bash
npm install @mui/material @emotion/react @emotion/styled
npm install @mui/icons-material
npm install @mui/lab  # 추가 컴포넌트
```

### 2.2 Theme 설정

```typescript
// lib/theme.ts
import { createTheme } from '@mui/material/styles';

export const theme = createTheme({
  palette: {
    primary: {
      main: '#1976d2',
      light: '#42a5f5',
      dark: '#1565c0',
    },
    secondary: {
      main: '#9c27b0',
    },
    error: {
      main: '#d32f2f',
    },
    background: {
      default: '#fafafa',
      paper: '#ffffff',
    },
  },
  typography: {
    fontFamily: '"Pretendard", "Roboto", "Helvetica", "Arial", sans-serif',
    h1: {
      fontSize: '2.5rem',
      fontWeight: 700,
    },
    h2: {
      fontSize: '2rem',
      fontWeight: 600,
    },
    body1: {
      fontSize: '1rem',
      lineHeight: 1.6,
    },
  },
  shape: {
    borderRadius: 8,
  },
  spacing: 8, // 기본 8px
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none', // 대문자 변환 비활성화
        },
      },
    },
  },
});
```

### 2.3 ThemeProvider 적용

```typescript
// app/providers.tsx
import { ThemeProvider } from '@mui/material/styles';
import CssBaseline from '@mui/material/CssBaseline';
import { theme } from '@/lib/theme';

export function Providers({ children }: { children: React.ReactNode }) {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      {children}
    </ThemeProvider>
  );
}
```

---

## 3. 스타일링 규칙

### 3.1 sx prop vs styled()

| 방식 | 사용 시점 | 예시 |
|------|----------|------|
| **sx prop** | 일회성 스타일, 간단한 커스텀 | 특정 컴포넌트만의 스타일 |
| **styled()** | 재사용 컴포넌트, 복잡한 스타일 | 커스텀 Button, Card |
| **theme.components** | 전역 오버라이드 | 모든 Button 기본 스타일 |

### 3.2 sx prop 사용

```typescript
// 권장: 일회성 스타일
<Box
  sx={{
    display: 'flex',
    alignItems: 'center',
    gap: 2,              // theme.spacing(2) = 16px
    p: 3,                // padding: 24px
    mt: 2,               // marginTop: 16px
    bgcolor: 'primary.main',
    borderRadius: 1,     // theme.shape.borderRadius
    '&:hover': {
      bgcolor: 'primary.dark',
    },
  }}
>
  Content
</Box>

// 반응형
<Box
  sx={{
    width: { xs: '100%', sm: '50%', md: '33%' },
    display: { xs: 'none', md: 'block' },
  }}
/>
```

### 3.3 styled() 사용

```typescript
import { styled } from '@mui/material/styles';
import { Card } from '@mui/material';

// 재사용 컴포넌트
const StyledCard = styled(Card)(({ theme }) => ({
  padding: theme.spacing(3),
  borderRadius: theme.shape.borderRadius * 2,
  boxShadow: theme.shadows[2],
  transition: theme.transitions.create(['box-shadow', 'transform']),
  '&:hover': {
    boxShadow: theme.shadows[8],
    transform: 'translateY(-4px)',
  },
}));

// Props 기반 스타일
interface StatusChipProps {
  status: 'success' | 'error' | 'warning';
}

const StatusChip = styled(Chip)<StatusChipProps>(({ theme, status }) => ({
  backgroundColor: theme.palette[status].light,
  color: theme.palette[status].dark,
}));
```

---

## 4. 컴포넌트 사용 패턴

### 4.1 레이아웃 컴포넌트

```typescript
// Box: 기본 레이아웃
<Box component="section" sx={{ p: 2 }}>
  Content
</Box>

// Container: 중앙 정렬, 최대 너비
<Container maxWidth="lg">
  <Typography variant="h1">Title</Typography>
</Container>

// Stack: Flex 레이아웃
<Stack direction="row" spacing={2} alignItems="center">
  <Avatar />
  <Typography>Name</Typography>
</Stack>

// Grid: 그리드 레이아웃
<Grid container spacing={3}>
  <Grid item xs={12} md={6}>
    Left
  </Grid>
  <Grid item xs={12} md={6}>
    Right
  </Grid>
</Grid>
```

### 4.2 입력 컴포넌트

```typescript
// TextField with 검증
<TextField
  label="이메일"
  type="email"
  variant="outlined"
  fullWidth
  error={!!errors.email}
  helperText={errors.email?.message}
  {...register('email')}
/>

// Select
<FormControl fullWidth error={!!errors.status}>
  <InputLabel>상태</InputLabel>
  <Select label="상태" {...register('status')}>
    <MenuItem value="pending">대기</MenuItem>
    <MenuItem value="completed">완료</MenuItem>
  </Select>
  <FormHelperText>{errors.status?.message}</FormHelperText>
</FormControl>

// Autocomplete
<Autocomplete
  options={products}
  getOptionLabel={(option) => option.name}
  renderInput={(params) => (
    <TextField {...params} label="상품 선택" />
  )}
  onChange={(_, value) => setValue('product', value)}
/>
```

### 4.3 피드백 컴포넌트

```typescript
// Alert
<Alert severity="success" onClose={() => {}}>
  주문이 완료되었습니다.
</Alert>

// Snackbar
<Snackbar
  open={open}
  autoHideDuration={6000}
  onClose={handleClose}
  anchorOrigin={{ vertical: 'bottom', horizontal: 'center' }}
>
  <Alert severity="info">알림 메시지</Alert>
</Snackbar>

// Dialog
<Dialog open={open} onClose={handleClose}>
  <DialogTitle>주문 취소</DialogTitle>
  <DialogContent>
    <DialogContentText>
      정말 취소하시겠습니까?
    </DialogContentText>
  </DialogContent>
  <DialogActions>
    <Button onClick={handleClose}>아니오</Button>
    <Button onClick={handleConfirm} color="error">
      취소하기
    </Button>
  </DialogActions>
</Dialog>

// Skeleton
<Skeleton variant="rectangular" width="100%" height={200} />
<Skeleton variant="text" sx={{ fontSize: '1rem' }} />
```

---

## 5. 반응형 처리

### 5.1 useMediaQuery

```typescript
import { useMediaQuery, useTheme } from '@mui/material';

function ResponsiveComponent() {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));
  const isTablet = useMediaQuery(theme.breakpoints.between('sm', 'md'));
  const isDesktop = useMediaQuery(theme.breakpoints.up('md'));

  return (
    <Box>
      {isMobile && <MobileView />}
      {isTablet && <TabletView />}
      {isDesktop && <DesktopView />}
    </Box>
  );
}
```

### 5.2 Breakpoint 시스템

```typescript
// breakpoints 기본값
// xs: 0px
// sm: 600px
// md: 900px
// lg: 1200px
// xl: 1536px

// sx에서 반응형
<Typography
  sx={{
    fontSize: { xs: '1rem', sm: '1.25rem', md: '1.5rem' },
  }}
>
  Responsive Text
</Typography>
```

---

## 6. 아이콘 사용

```typescript
import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  Search as SearchIcon,
} from '@mui/icons-material';

// IconButton
<IconButton aria-label="삭제" onClick={handleDelete}>
  <DeleteIcon />
</IconButton>

// Button with Icon
<Button startIcon={<AddIcon />} variant="contained">
  추가
</Button>

// InputAdornment
<TextField
  InputProps={{
    startAdornment: (
      <InputAdornment position="start">
        <SearchIcon />
      </InputAdornment>
    ),
  }}
/>
```

---

## 7. 테마 확장

### 7.1 커스텀 컬러 추가

```typescript
// types/theme.d.ts
declare module '@mui/material/styles' {
  interface Palette {
    neutral: Palette['primary'];
  }
  interface PaletteOptions {
    neutral?: PaletteOptions['primary'];
  }
}

// lib/theme.ts
export const theme = createTheme({
  palette: {
    neutral: {
      main: '#64748b',
      light: '#94a3b8',
      dark: '#475569',
      contrastText: '#fff',
    },
  },
});

// 사용
<Button color="neutral">Neutral Button</Button>
```

### 7.2 커스텀 컴포넌트 variants

```typescript
declare module '@mui/material/Button' {
  interface ButtonPropsVariantOverrides {
    dashed: true;
  }
}

export const theme = createTheme({
  components: {
    MuiButton: {
      variants: [
        {
          props: { variant: 'dashed' },
          style: {
            border: '2px dashed',
          },
        },
      ],
    },
  },
});
```

---

## 8. Best Practices

### 8.1 권장 사항

- theme.spacing() 사용하여 일관된 간격
- theme.palette 사용하여 색상 일관성
- component prop으로 시맨틱 HTML
- aria-* 속성으로 접근성 확보

### 8.2 지양 사항

- 인라인 스타일 직접 사용 지양
- !important 사용 지양
- px 단위 직접 사용 지양 (spacing 활용)
- 커스텀 breakpoint 남발 지양

---

## 9. Tailwind와 혼용

```typescript
// MUI 컴포넌트에 Tailwind 클래스 추가
<Card className="hover:shadow-lg transition-shadow">
  <CardContent className="space-y-4">
    <Typography variant="h6">Title</Typography>
  </CardContent>
</Card>

// 주의: sx와 className 충돌 시 sx 우선
// 레이아웃: Tailwind, 테마 관련: MUI sx 권장
```

---

<!-- pal:convention:ui:mui -->
