# Frontend Rework: MUI + golang-dashboard UI/UX Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace Tailwind CSS with MUI (Material UI) using golang-dashboard's neobrutalism design system while keeping Go backend.

**Architecture:** Update `web/frontend/` React app to use MUI components with custom theme matching golang-dashboard. Keep TanStack Query for data fetching, keep existing API layer unchanged.

**Tech Stack:** React 19 + MUI v7 + Recharts + TanStack Query + TypeScript + Vite

---

## Task 1: Update Dependencies

**Files:**
- Modify: `web/frontend/package.json`

**Step 1: Install MUI packages**

Run:
```bash
cd web/frontend && npm install @mui/material @mui/icons-material @emotion/react @emotion/styled
```
Expected: Success with packages added to package.json

**Step 2: Remove Tailwind packages**

Run:
```bash
cd web/frontend && npm uninstall tailwindcss postcss autoprefixer class-variance-authority clsx tailwind-merge
```
Expected: Success with packages removed

**Step 3: Delete Tailwind config files**

Run:
```bash
rm -f web/frontend/tailwind.config.js web/frontend/postcss.config.js
```
Expected: Files removed

**Step 4: Update index.css to remove Tailwind directives**

Modify `web/frontend/src/index.css`:
```css
/* Remove @tailwind directives, keep only base styles */
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
```

**Step 5: Verify build still works**

Run:
```bash
cd web/frontend && npm run build
```
Expected: Build will fail (components still use Tailwind classes) - this is expected at this stage

**Step 6: Commit**

```bash
git add web/frontend/package.json web/frontend/package-lock.json web/frontend/src/index.css
git commit -m "chore(frontend): migrate from Tailwind to MUI dependencies"
```

---

## Task 2: Create MUI Theme

**Files:**
- Create: `web/frontend/src/theme.ts`

**Step 1: Create theme file with golang-dashboard colors**

Create `web/frontend/src/theme.ts`:
```typescript
import { createTheme } from '@mui/material';

export const theme = createTheme({
  palette: {
    mode: 'light',
    primary: {
      main: '#000000',
      contrastText: '#FFFFFF',
    },
    secondary: {
      main: '#FFD93D',
      contrastText: '#000000',
    },
    success: {
      main: '#22C55E',
      light: '#86EFAC',
      dark: '#16A34A',
    },
    warning: {
      main: '#F59E0B',
      light: '#FCD34D',
      dark: '#D97706',
    },
    error: {
      main: '#EF4444',
      light: '#FCA5A5',
      dark: '#DC2626',
    },
    info: {
      main: '#3B82F6',
      light: '#93C5FD',
      dark: '#2563EB',
    },
    background: {
      default: '#F9FAFB',
      paper: '#FFFFFF',
    },
    text: {
      primary: '#000000',
      secondary: '#6B7280',
    },
    divider: '#E5E7EB',
  },
  typography: {
    fontFamily: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", "Roboto", sans-serif',
    h4: {
      fontWeight: 700,
      fontSize: '1.75rem',
      color: '#000000',
    },
    h5: {
      fontWeight: 700,
      fontSize: '1.5rem',
      color: '#000000',
    },
    h6: {
      fontWeight: 700,
      fontSize: '1.125rem',
      color: '#000000',
    },
    subtitle1: {
      fontWeight: 500,
      fontSize: '1rem',
      color: '#000000',
    },
    subtitle2: {
      fontWeight: 500,
      fontSize: '0.875rem',
      color: '#6B7280',
    },
    body1: {
      fontSize: '1rem',
      color: '#000000',
    },
    body2: {
      fontSize: '0.875rem',
      color: '#6B7280',
    },
  },
  shape: {
    borderRadius: 12,
  },
  shadows: Array(25).fill('none') as never,
  components: {
    MuiButton: {
      styleOverrides: {
        root: {
          textTransform: 'none',
          fontWeight: 600,
          borderRadius: 8,
          padding: '10px 24px',
          fontSize: '0.875rem',
          boxShadow: 'none',
          border: '2px solid #000000',
          '&:hover': {
            boxShadow: 'none',
          },
        },
        contained: {
          backgroundColor: '#FFD93D',
          color: '#000000',
          '&:hover': {
            backgroundColor: '#FFC107',
          },
        },
        outlined: {
          borderColor: '#000000',
          borderWidth: 2,
          '&:hover': {
            borderWidth: 2,
            backgroundColor: '#F9FAFB',
          },
        },
      },
    },
    MuiCard: {
      styleOverrides: {
        root: {
          borderRadius: 16,
          boxShadow: 'none',
          border: '2px solid #000000',
          '&:hover': {
            boxShadow: 'none',
          },
        },
      },
    },
    MuiPaper: {
      styleOverrides: {
        root: {
          borderRadius: 16,
          backgroundImage: 'none',
          boxShadow: 'none',
        },
      },
    },
    MuiChip: {
      styleOverrides: {
        root: {
          fontWeight: 600,
          borderRadius: 6,
          border: '1.5px solid #000000',
        },
      },
    },
    MuiTextField: {
      styleOverrides: {
        root: {
          '& .MuiOutlinedInput-root': {
            borderRadius: 8,
            '& fieldset': {
              borderWidth: 2,
              borderColor: '#000000',
            },
            '&:hover fieldset': {
              borderWidth: 2,
              borderColor: '#000000',
            },
            '&.Mui-focused fieldset': {
              borderWidth: 2,
              borderColor: '#000000',
            },
          },
        },
      },
    },
    MuiLinearProgress: {
      styleOverrides: {
        root: {
          borderRadius: 4,
          height: 8,
          backgroundColor: '#E5E7EB',
        },
      },
    },
    MuiTableCell: {
      styleOverrides: {
        root: {
          borderBottom: '1px solid #E5E7EB',
        },
        head: {
          fontWeight: 700,
          backgroundColor: '#F9FAFB',
        },
      },
    },
  },
});
```

**Step 2: Commit**

```bash
git add web/frontend/src/theme.ts
git commit -m "feat(frontend): add MUI theme with golang-dashboard design system"
```

---

## Task 3: Create StatCard Component

**Files:**
- Create: `web/frontend/src/components/StatCard.tsx`

**Step 1: Create StatCard component**

Create `web/frontend/src/components/StatCard.tsx`:
```typescript
import { Card, CardContent, Typography, Box, useMediaQuery, useTheme } from '@mui/material';
import type { SvgIconProps } from '@mui/material/SvgIcon';
import type { ComponentType } from 'react';

interface StatCardProps {
  title: string;
  value: string | number;
  subtitle?: string;
  icon: ComponentType<SvgIconProps>;
}

export const StatCard = ({
  title,
  value,
  subtitle,
  icon: Icon,
}: StatCardProps) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));

  return (
    <Card
      sx={{
        height: '100%',
        background: '#FFFFFF',
        borderRadius: '16px',
        border: '2px solid #000000',
        boxShadow: 'none',
        transition: 'transform 0.2s ease',
        '&:hover': {
          transform: 'translateY(-2px)',
        },
      }}
    >
      <CardContent sx={{ p: { xs: 2, sm: 2.5, md: 3 } }}>
        <Box sx={{ display: 'flex', alignItems: 'flex-start', justifyContent: 'space-between', mb: { xs: 1.5, md: 2 } }}>
          <Typography
            variant="body1"
            sx={{
              color: '#000000',
              fontWeight: 500,
              fontSize: { xs: '0.75rem', sm: '0.813rem', md: '0.938rem' },
              lineHeight: 1.3,
            }}
          >
            {title}
          </Typography>
          <Box
            sx={{
              p: { xs: 0.75, md: 1 },
              bgcolor: '#F9FAFB',
              borderRadius: '8px',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
            }}
          >
            <Icon sx={{ fontSize: { xs: 18, md: 24 }, color: '#6B7280' }} />
          </Box>
        </Box>

        <Typography
          sx={{
            fontSize: { xs: '1.5rem', sm: '1.75rem', md: '2.25rem' },
            fontWeight: 700,
            color: '#000000',
            lineHeight: 1.2,
            mb: { xs: 0.5, md: 1 },
          }}
        >
          {typeof value === 'number' ? value.toLocaleString() : value}
        </Typography>

        {subtitle && (
          <Typography
            variant="body2"
            sx={{
              color: '#6B7280',
              fontSize: { xs: '0.688rem', sm: '0.75rem', md: '0.875rem' },
              fontWeight: 400,
            }}
          >
            {subtitle}
          </Typography>
        )}
      </CardContent>
    </Card>
  );
};
```

**Step 2: Commit**

```bash
git add web/frontend/src/components/StatCard.tsx
git commit -m "feat(frontend): add StatCard component for dashboard stats"
```

---

## Task 4: Create StatusChip Component

**Files:**
- Create: `web/frontend/src/components/StatusChip.tsx`

**Step 1: Create StatusChip component**

Create `web/frontend/src/components/StatusChip.tsx`:
```typescript
import { Chip } from '@mui/material';

type StatusType = 'pending' | 'queued' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled' | 'online' | 'offline' | 'busy';

interface StatusChipProps {
  status: StatusType | string;
  size?: 'small' | 'medium';
}

const getStatusColors = (status: string): { bg: string; color: string } => {
  switch (status.toLowerCase()) {
    case 'completed':
    case 'success':
    case 'online':
      return { bg: '#D1FAE5', color: '#065F46' };
    case 'failed':
    case 'error':
    case 'cancelled':
    case 'offline':
      return { bg: '#FEE2E2', color: '#991B1B' };
    case 'running':
    case 'busy':
    case 'processing':
      return { bg: '#DBEAFE', color: '#1E40AF' };
    case 'pending':
    case 'queued':
    case 'paused':
      return { bg: '#FEF3C7', color: '#92400E' };
    default:
      return { bg: '#E5E7EB', color: '#374151' };
  }
};

export const StatusChip = ({ status, size = 'small' }: StatusChipProps) => {
  const colors = getStatusColors(status);

  return (
    <Chip
      label={status}
      size={size}
      sx={{
        height: size === 'small' ? 22 : 28,
        fontSize: size === 'small' ? '0.688rem' : '0.813rem',
        fontWeight: 600,
        bgcolor: colors.bg,
        color: colors.color,
        border: 'none',
        textTransform: 'capitalize',
      }}
    />
  );
};
```

**Step 2: Commit**

```bash
git add web/frontend/src/components/StatusChip.tsx
git commit -m "feat(frontend): add StatusChip component for status indicators"
```

---

## Task 5: Update App.tsx with MUI ThemeProvider

**Files:**
- Modify: `web/frontend/src/App.tsx`

**Step 1: Update App.tsx to use MUI ThemeProvider**

Replace `web/frontend/src/App.tsx`:
```typescript
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { ThemeProvider, CssBaseline } from "@mui/material"
import { Toaster } from "sonner"
import { theme } from "./theme"
import { Layout } from "./components/Layout"
import { isAuthenticated } from "./api/client"
import Dashboard from "./pages/Dashboard"
import Jobs from "./pages/Jobs"
import JobCreate from "./pages/Jobs/JobCreate"
import JobDetail from "./pages/Jobs/JobDetail"
import Workers from "./pages/Workers"
import Settings from "./pages/Settings"
import ProxyGate from "./pages/ProxyGate"
import Results from "./pages/Results"
import Login from "./pages/Login"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
      retry: 1,
    },
  },
})

// Protected Route wrapper
function ProtectedRoute({ children }: { children: React.ReactNode }) {
  if (!isAuthenticated()) {
    return <Navigate to="/login" replace />
  }
  return <>{children}</>
}

function App() {
  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <QueryClientProvider client={queryClient}>
        <Toaster position="top-right" richColors closeButton />
        <BrowserRouter>
          <Routes>
            {/* Public route */}
            <Route path="/login" element={<Login />} />

            {/* Protected routes */}
            <Route
              path="/*"
              element={
                <ProtectedRoute>
                  <Layout>
                    <Routes>
                      <Route path="/" element={<Dashboard />} />
                      <Route path="/jobs" element={<Jobs />} />
                      <Route path="/jobs/new" element={<JobCreate />} />
                      <Route path="/jobs/:id" element={<JobDetail />} />
                      <Route path="/results" element={<Results />} />
                      <Route path="/workers" element={<Workers />} />
                      <Route path="/proxies" element={<ProxyGate />} />
                      <Route path="/settings" element={<Settings />} />
                      <Route path="*" element={<Navigate to="/" replace />} />
                    </Routes>
                  </Layout>
                </ProtectedRoute>
              }
            />
          </Routes>
        </BrowserRouter>
      </QueryClientProvider>
    </ThemeProvider>
  )
}

export default App
```

**Step 2: Commit**

```bash
git add web/frontend/src/App.tsx
git commit -m "feat(frontend): integrate MUI ThemeProvider in App.tsx"
```

---

## Task 6: Create MUI Layout Component

**Files:**
- Create: `web/frontend/src/components/Layout.tsx`

**Step 1: Create Layout component matching golang-dashboard**

Create `web/frontend/src/components/Layout.tsx`:
```typescript
import { useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import {
  Box,
  Drawer,
  AppBar,
  Toolbar,
  List,
  Typography,
  Divider,
  IconButton,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  useTheme,
  useMediaQuery,
  Avatar,
  Chip,
} from '@mui/material';
import {
  Menu as MenuIcon,
  DashboardOutlined,
  WorkOutlined,
  StorageOutlined,
  DnsOutlined,
  PublicOutlined,
  SettingsOutlined,
  ChevronLeft,
  Circle,
  LogoutOutlined,
} from '@mui/icons-material';
import { removeApiKey } from '../api/client';

const drawerWidth = 280;

interface LayoutProps {
  children: React.ReactNode;
}

interface NavItem {
  title: string;
  path: string;
  icon: React.ReactNode;
}

const navItems: NavItem[] = [
  { title: 'Dashboard', path: '/', icon: <DashboardOutlined /> },
  { title: 'Jobs', path: '/jobs', icon: <WorkOutlined /> },
  { title: 'Results', path: '/results', icon: <StorageOutlined /> },
  { title: 'Workers', path: '/workers', icon: <DnsOutlined /> },
  { title: 'Proxies', path: '/proxies', icon: <PublicOutlined /> },
];

const secondaryNavItems: NavItem[] = [
  { title: 'Settings', path: '/settings', icon: <SettingsOutlined /> },
];

export const Layout = ({ children }: LayoutProps) => {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('md'));
  const [mobileOpen, setMobileOpen] = useState(false);
  const location = useLocation();
  const navigate = useNavigate();

  const handleDrawerToggle = () => {
    setMobileOpen(!mobileOpen);
  };

  const handleNavigation = (path: string) => {
    navigate(path);
    if (isMobile) {
      setMobileOpen(false);
    }
  };

  const handleLogout = () => {
    removeApiKey();
    window.location.href = '/login';
  };

  const drawer = (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      {/* Logo/Header */}
      <Box sx={{ p: 3, display: 'flex', alignItems: 'center', gap: 2 }}>
        <Avatar
          sx={{
            bgcolor: '#FFD93D',
            color: '#000000',
            width: 44,
            height: 44,
            fontWeight: 700,
            border: '2px solid #000000',
          }}
        >
          GS
        </Avatar>
        <Box>
          <Typography
            variant="h6"
            sx={{ fontWeight: 700, fontSize: '1.125rem', color: '#000000', lineHeight: 1.2 }}
          >
            GMaps Scraper
          </Typography>
          <Typography variant="caption" sx={{ color: '#6B7280', fontSize: '0.75rem' }}>
            Dashboard v2.0
          </Typography>
        </Box>
        {isMobile && (
          <IconButton onClick={handleDrawerToggle} sx={{ ml: 'auto' }}>
            <ChevronLeft />
          </IconButton>
        )}
      </Box>

      <Divider sx={{ borderColor: '#E5E7EB' }} />

      {/* Navigation */}
      <List sx={{ flex: 1, px: 2, py: 2 }}>
        {navItems.map((item) => {
          const isActive = location.pathname === item.path;
          return (
            <ListItem key={item.path} disablePadding sx={{ mb: 0.5 }}>
              <ListItemButton
                onClick={() => handleNavigation(item.path)}
                sx={{
                  borderRadius: '12px',
                  py: 1.5,
                  px: 2,
                  bgcolor: isActive ? '#000000' : 'transparent',
                  color: isActive ? '#FFFFFF' : '#374151',
                  transition: 'all 0.2s ease',
                  '&:hover': {
                    bgcolor: isActive ? '#000000' : '#F3F4F6',
                  },
                }}
              >
                <ListItemIcon sx={{ color: isActive ? '#FFD93D' : '#6B7280', minWidth: 40 }}>
                  {item.icon}
                </ListItemIcon>
                <ListItemText
                  primary={item.title}
                  primaryTypographyProps={{
                    fontWeight: isActive ? 600 : 500,
                    fontSize: '0.938rem',
                  }}
                />
                {isActive && <Circle sx={{ fontSize: 8, color: '#FFD93D' }} />}
              </ListItemButton>
            </ListItem>
          );
        })}
      </List>

      {/* Secondary Navigation */}
      <List sx={{ px: 2, pb: 1 }}>
        {secondaryNavItems.map((item) => {
          const isActive = location.pathname === item.path;
          return (
            <ListItem key={item.path} disablePadding sx={{ mb: 0.5 }}>
              <ListItemButton
                onClick={() => handleNavigation(item.path)}
                sx={{
                  borderRadius: '12px',
                  py: 1.5,
                  px: 2,
                  bgcolor: isActive ? '#000000' : 'transparent',
                  color: isActive ? '#FFFFFF' : '#374151',
                  transition: 'all 0.2s ease',
                  '&:hover': {
                    bgcolor: isActive ? '#000000' : '#F3F4F6',
                  },
                }}
              >
                <ListItemIcon sx={{ color: isActive ? '#FFD93D' : '#6B7280', minWidth: 40 }}>
                  {item.icon}
                </ListItemIcon>
                <ListItemText
                  primary={item.title}
                  primaryTypographyProps={{
                    fontWeight: isActive ? 600 : 500,
                    fontSize: '0.938rem',
                  }}
                />
              </ListItemButton>
            </ListItem>
          );
        })}
      </List>

      <Divider sx={{ borderColor: '#E5E7EB' }} />

      {/* User Profile */}
      <Box sx={{ p: 3 }}>
        <Box
          sx={{
            p: 2,
            bgcolor: '#FFFFFF',
            borderRadius: '12px',
            border: '1px solid #E5E7EB',
            display: 'flex',
            alignItems: 'center',
            gap: 1.5,
          }}
        >
          <Avatar
            sx={{
              width: 36,
              height: 36,
              bgcolor: '#FFD93D',
              color: '#000000',
              fontWeight: 700,
              fontSize: '0.875rem',
              border: '2px solid #000000',
            }}
          >
            A
          </Avatar>
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography variant="body2" sx={{ fontWeight: 600, color: '#000000' }}>
              Admin
            </Typography>
            <Typography variant="caption" sx={{ color: '#6B7280' }}>
              Online
            </Typography>
          </Box>
          <IconButton
            onClick={handleLogout}
            size="small"
            sx={{
              color: '#6B7280',
              '&:hover': { color: '#EF4444', bgcolor: '#FEE2E2' },
            }}
          >
            <LogoutOutlined fontSize="small" />
          </IconButton>
        </Box>
      </Box>
    </Box>
  );

  return (
    <Box sx={{ display: 'flex', minHeight: '100vh', bgcolor: '#F9FAFB' }}>
      {/* Mobile AppBar */}
      {isMobile && (
        <AppBar
          position="fixed"
          sx={{
            bgcolor: '#FFFFFF',
            borderBottom: '2px solid #000000',
            boxShadow: 'none',
          }}
        >
          <Toolbar>
            <IconButton
              color="inherit"
              edge="start"
              onClick={handleDrawerToggle}
              sx={{ mr: 2, color: '#000000' }}
            >
              <MenuIcon />
            </IconButton>
            <Avatar
              sx={{
                bgcolor: '#FFD93D',
                color: '#000000',
                width: 32,
                height: 32,
                fontWeight: 700,
                fontSize: '0.75rem',
                border: '1.5px solid #000000',
                mr: 1.5,
              }}
            >
              GS
            </Avatar>
            <Typography
              variant="h6"
              noWrap
              component="div"
              sx={{ color: '#000000', fontWeight: 700, fontSize: '1rem' }}
            >
              GMaps Scraper
            </Typography>
          </Toolbar>
        </AppBar>
      )}

      {/* Sidebar Drawer */}
      <Box component="nav" sx={{ width: { md: drawerWidth }, flexShrink: { md: 0 } }}>
        {/* Mobile Drawer */}
        <Drawer
          variant="temporary"
          open={mobileOpen}
          onClose={handleDrawerToggle}
          ModalProps={{ keepMounted: true }}
          sx={{
            display: { xs: 'block', md: 'none' },
            '& .MuiDrawer-paper': {
              boxSizing: 'border-box',
              width: drawerWidth,
              border: 'none',
              bgcolor: '#FFFFFF',
            },
          }}
        >
          {drawer}
        </Drawer>

        {/* Desktop Drawer */}
        <Drawer
          variant="permanent"
          sx={{
            display: { xs: 'none', md: 'block' },
            '& .MuiDrawer-paper': {
              boxSizing: 'border-box',
              width: drawerWidth,
              border: 'none',
              borderRight: '2px solid #000000',
              bgcolor: '#FFFFFF',
            },
          }}
          open
        >
          {drawer}
        </Drawer>
      </Box>

      {/* Main Content */}
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          flex: 1,
          minWidth: 0,
          width: { xs: '100%', md: `calc(100% - ${drawerWidth}px)` },
          minHeight: '100vh',
          mt: { xs: '64px', md: 0 },
        }}
      >
        {children}
      </Box>
    </Box>
  );
};
```

**Step 2: Delete old AppShell**

Run:
```bash
rm web/frontend/src/components/Layout/AppShell.tsx
rmdir web/frontend/src/components/Layout 2>/dev/null || true
```

**Step 3: Commit**

```bash
git add web/frontend/src/components/Layout.tsx
git add -A web/frontend/src/components/Layout/
git commit -m "feat(frontend): add MUI Layout component, remove old AppShell"
```

---

## Task 7: Update Dashboard Page

**Files:**
- Modify: `web/frontend/src/pages/Dashboard.tsx`

**Step 1: Rewrite Dashboard with MUI components**

Replace `web/frontend/src/pages/Dashboard.tsx`:
```typescript
import {
  Box,
  Typography,
  Button,
  CircularProgress,
  Alert,
  Paper,
  Grid2 as Grid,
  useMediaQuery,
  useTheme,
} from '@mui/material';
import {
  RefreshOutlined,
  WorkOutlined,
  PlayArrowOutlined,
  CheckCircleOutlined,
  ErrorOutlined,
  StorageOutlined,
  EmailOutlined,
  DnsOutlined,
  SpeedOutlined,
} from '@mui/icons-material';
import { StatCard } from '../components/StatCard';
import { StatusChip } from '../components/StatusChip';
import { useDashboardStats, useRecentJobs, useActiveWorkers } from '../hooks/useDashboard';
import type { Job, Worker } from '../api/types';

export default function Dashboard() {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));

  const { data: stats, isLoading: isLoadingStats, refetch: refetchStats } = useDashboardStats();
  const { data: recentJobs, isLoading: isLoadingJobs } = useRecentJobs();
  const { data: workers, isLoading: isLoadingWorkers } = useActiveWorkers();

  const loading = isLoadingStats || isLoadingJobs || isLoadingWorkers;

  const handleRefresh = () => {
    refetchStats();
  };

  if (loading && !stats) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress size={48} sx={{ color: '#000000' }} />
      </Box>
    );
  }

  const successRate = stats && stats.total_jobs > 0
    ? ((stats.completed_jobs / stats.total_jobs) * 100).toFixed(1)
    : '0';

  const formatTimeAgo = (dateStr: string) => {
    const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
  };

  const jobsList = recentJobs?.data || [];
  const workersList = workers?.data || [];
  const onlineWorkers = workersList.filter((w: Worker) => w.status === 'online').length;

  return (
    <Box sx={{ p: { xs: 2, sm: 2, md: 3 }, bgcolor: '#F9FAFB', minHeight: '100%', width: '100%', boxSizing: 'border-box' }}>
      {/* Header */}
      <Box sx={{
        display: 'flex',
        flexDirection: { xs: 'column', sm: 'row' },
        justifyContent: 'space-between',
        alignItems: { xs: 'flex-start', sm: 'center' },
        mb: 3,
        gap: 2,
      }}>
        <Box>
          <Typography variant="h4" sx={{ color: '#000000', fontWeight: 700, fontSize: { xs: '1.5rem', md: '1.75rem' }, mb: 0.5 }}>
            Dashboard
          </Typography>
          <Typography variant="body2" sx={{ color: '#6B7280' }}>
            Real-time scraping analytics
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<RefreshOutlined />}
          onClick={handleRefresh}
          disabled={loading}
          size={isMobile ? 'small' : 'medium'}
          sx={{
            bgcolor: '#FFD93D',
            color: '#000000',
            border: '2px solid #000000',
            fontWeight: 600,
            '&:hover': { bgcolor: '#FFC107' },
          }}
        >
          {isMobile ? 'Refresh' : 'Refresh Data'}
        </Button>
      </Box>

      {/* Stats Cards - 4 columns x 2 rows */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 6, sm: 3 }}>
          <StatCard title="Total Jobs" value={stats?.total_jobs || 0} subtitle="All time" icon={WorkOutlined} />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <StatCard title="Active" value={stats?.active_jobs || 0} subtitle="Processing" icon={PlayArrowOutlined} />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <StatCard title="Completed" value={stats?.completed_jobs || 0} subtitle={`${successRate}% rate`} icon={CheckCircleOutlined} />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <StatCard title="Failed" value={stats?.failed_jobs || 0} subtitle="Errors" icon={ErrorOutlined} />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <StatCard title="Results" value={stats?.total_results || 0} subtitle="Business listings" icon={StorageOutlined} />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <StatCard title="Workers" value={onlineWorkers} subtitle={`${workersList.length} total`} icon={DnsOutlined} />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <StatCard title="Rate" value="--" subtitle="per hour" icon={SpeedOutlined} />
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <StatCard title="Emails" value="--" subtitle="Found" icon={EmailOutlined} />
        </Grid>
      </Grid>

      {/* Recent Jobs + Workers */}
      <Grid container spacing={2}>
        <Grid size={{ xs: 12, md: 7 }}>
          <Paper sx={{ border: '2px solid #000', borderRadius: '16px', overflow: 'hidden', height: '100%' }}>
            <Box sx={{ p: 2, borderBottom: '1px solid #E5E7EB' }}>
              <Typography variant="h6" sx={{ fontWeight: 700 }}>Recent Jobs</Typography>
              <Typography variant="body2" sx={{ color: '#6B7280' }}>Latest scraping jobs</Typography>
            </Box>
            <Box sx={{ maxHeight: 400, overflow: 'auto' }}>
              {jobsList.slice(0, 8).map((job: Job) => (
                <Box key={job.id} sx={{ p: 2, borderBottom: '1px solid #F3F4F6', '&:hover': { bgcolor: '#F9FAFB' } }}>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 0.5 }}>
                    <Typography sx={{ fontWeight: 600, fontSize: '0.875rem', maxWidth: '60%', overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                      {job.name}
                    </Typography>
                    <StatusChip status={job.status} />
                  </Box>
                  <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Typography variant="caption" sx={{ color: '#9CA3AF' }}>
                      {job.progress.scraped_places}/{job.progress.total_places} places
                    </Typography>
                    <Typography variant="caption" sx={{ color: '#9CA3AF' }}>
                      {formatTimeAgo(job.updated_at)}
                    </Typography>
                  </Box>
                </Box>
              ))}
              {jobsList.length === 0 && (
                <Box sx={{ p: 4, textAlign: 'center' }}>
                  <Typography variant="body2" sx={{ color: '#9CA3AF' }}>No jobs yet</Typography>
                </Box>
              )}
            </Box>
          </Paper>
        </Grid>

        <Grid size={{ xs: 12, md: 5 }}>
          <Paper sx={{ p: 3, border: '2px solid #000', borderRadius: '16px', height: '100%' }}>
            <Typography variant="h6" sx={{ fontWeight: 700, mb: 2 }}>Worker Status</Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
              {workersList.slice(0, 5).map((worker: Worker) => (
                <Box
                  key={worker.id}
                  sx={{
                    p: 2,
                    borderRadius: '12px',
                    border: '1.5px solid',
                    borderColor: worker.status === 'online' ? '#22C55E' : worker.status === 'offline' ? '#EF4444' : '#F59E0B',
                    bgcolor: worker.status === 'online' ? '#F0FDF4' : worker.status === 'offline' ? '#FEF2F2' : '#FFFBEB',
                  }}
                >
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 0.5 }}>
                    <Typography sx={{ fontWeight: 600, fontSize: '0.9rem' }}>{worker.name}</Typography>
                    <StatusChip status={worker.status} />
                  </Box>
                  <Typography variant="caption" sx={{ color: '#6B7280' }}>
                    {worker.stats.jobs_completed} jobs | Last seen: {formatTimeAgo(worker.last_seen)}
                  </Typography>
                </Box>
              ))}
              {workersList.length === 0 && (
                <Box sx={{ p: 4, textAlign: 'center' }}>
                  <Typography variant="body2" sx={{ color: '#9CA3AF' }}>No workers connected</Typography>
                </Box>
              )}
            </Box>
          </Paper>
        </Grid>
      </Grid>
    </Box>
  );
}
```

**Step 2: Delete old Dashboard components**

Run:
```bash
rm -rf web/frontend/src/components/Dashboard/
```

**Step 3: Commit**

```bash
git add web/frontend/src/pages/Dashboard.tsx
git add -A web/frontend/src/components/Dashboard/
git commit -m "feat(frontend): rewrite Dashboard page with MUI components"
```

---

## Task 8: Update Login Page

**Files:**
- Modify: `web/frontend/src/pages/Login.tsx`

**Step 1: Rewrite Login with MUI**

Replace `web/frontend/src/pages/Login.tsx`:
```typescript
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Button,
  Alert,
  Avatar,
} from '@mui/material';
import { LockOutlined } from '@mui/icons-material';
import { setApiKey } from '../api/client';

export default function Login() {
  const [apiKey, setApiKeyValue] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    if (!apiKey.trim()) {
      setError('API Key is required');
      setLoading(false);
      return;
    }

    try {
      setApiKey(apiKey.trim());
      navigate('/');
    } catch {
      setError('Invalid API Key');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        bgcolor: '#F9FAFB',
        p: 2,
      }}
    >
      <Card sx={{ maxWidth: 400, width: '100%', border: '2px solid #000000' }}>
        <CardContent sx={{ p: 4 }}>
          <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', mb: 3 }}>
            <Avatar
              sx={{
                width: 56,
                height: 56,
                bgcolor: '#FFD93D',
                color: '#000000',
                border: '2px solid #000000',
                mb: 2,
              }}
            >
              <LockOutlined />
            </Avatar>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#000000' }}>
              GMaps Scraper
            </Typography>
            <Typography variant="body2" sx={{ color: '#6B7280', mt: 0.5 }}>
              Enter your API key to continue
            </Typography>
          </Box>

          {error && (
            <Alert severity="error" sx={{ mb: 2, border: '1px solid #EF4444' }}>
              {error}
            </Alert>
          )}

          <form onSubmit={handleSubmit}>
            <TextField
              fullWidth
              label="API Key"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKeyValue(e.target.value)}
              placeholder="Enter your API key"
              sx={{ mb: 3 }}
            />

            <Button
              fullWidth
              type="submit"
              variant="contained"
              disabled={loading}
              sx={{
                py: 1.5,
                bgcolor: '#FFD93D',
                color: '#000000',
                border: '2px solid #000000',
                fontWeight: 600,
                '&:hover': { bgcolor: '#FFC107' },
              }}
            >
              {loading ? 'Signing in...' : 'Sign In'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </Box>
  );
}
```

**Step 2: Commit**

```bash
git add web/frontend/src/pages/Login.tsx
git commit -m "feat(frontend): rewrite Login page with MUI components"
```

---

## Task 9: Update Jobs Page

**Files:**
- Modify: `web/frontend/src/pages/Jobs.tsx`

**Step 1: Rewrite Jobs page with MUI Table**

Replace `web/frontend/src/pages/Jobs.tsx`:
```typescript
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Typography,
  Button,
  Paper,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  TablePagination,
  CircularProgress,
  LinearProgress,
} from '@mui/material';
import { AddOutlined, RefreshOutlined } from '@mui/icons-material';
import { useQuery } from '@tanstack/react-query';
import { jobsApi } from '../api/jobs';
import { StatusChip } from '../components/StatusChip';
import type { Job } from '../api/types';

export default function Jobs() {
  const navigate = useNavigate();
  const [page, setPage] = useState(0);
  const [rowsPerPage, setRowsPerPage] = useState(10);

  const { data, isLoading, refetch } = useQuery({
    queryKey: ['jobs', page, rowsPerPage],
    queryFn: () => jobsApi.getAll(),
  });

  const jobs = data?.data || [];
  const total = data?.meta?.total || jobs.length;

  const formatDate = (dateStr: string) => {
    return new Date(dateStr).toLocaleString();
  };

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, bgcolor: '#F9FAFB', minHeight: '100%' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3, flexWrap: 'wrap', gap: 2 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700, color: '#000000', mb: 0.5 }}>
            Jobs
          </Typography>
          <Typography variant="body2" sx={{ color: '#6B7280' }}>
            Manage scraping jobs
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button
            variant="outlined"
            startIcon={<RefreshOutlined />}
            onClick={() => refetch()}
            sx={{ borderColor: '#000000', color: '#000000' }}
          >
            Refresh
          </Button>
          <Button
            variant="contained"
            startIcon={<AddOutlined />}
            onClick={() => navigate('/jobs/new')}
            sx={{ bgcolor: '#FFD93D', color: '#000000', border: '2px solid #000000' }}
          >
            New Job
          </Button>
        </Box>
      </Box>

      {/* Table */}
      <Paper sx={{ border: '2px solid #000000', borderRadius: '16px', overflow: 'hidden' }}>
        {isLoading && <LinearProgress sx={{ bgcolor: '#E5E7EB', '& .MuiLinearProgress-bar': { bgcolor: '#000000' } }} />}
        <TableContainer>
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: '#F9FAFB' }}>
                <TableCell sx={{ fontWeight: 700 }}>Name</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Status</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Progress</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Created</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Updated</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {jobs.map((job: Job) => (
                <TableRow
                  key={job.id}
                  hover
                  onClick={() => navigate(`/jobs/${job.id}`)}
                  sx={{ cursor: 'pointer' }}
                >
                  <TableCell>
                    <Typography sx={{ fontWeight: 600 }}>{job.name}</Typography>
                    <Typography variant="caption" sx={{ color: '#6B7280' }}>
                      {job.config.keywords.slice(0, 2).join(', ')}
                      {job.config.keywords.length > 2 && ` +${job.config.keywords.length - 2} more`}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <StatusChip status={job.status} />
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      <LinearProgress
                        variant="determinate"
                        value={job.progress.percentage}
                        sx={{
                          width: 80,
                          height: 8,
                          borderRadius: 4,
                          bgcolor: '#E5E7EB',
                          '& .MuiLinearProgress-bar': { bgcolor: '#000000' },
                        }}
                      />
                      <Typography variant="caption" sx={{ fontWeight: 600 }}>
                        {job.progress.percentage.toFixed(0)}%
                      </Typography>
                    </Box>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">{formatDate(job.created_at)}</Typography>
                  </TableCell>
                  <TableCell>
                    <Typography variant="body2">{formatDate(job.updated_at)}</Typography>
                  </TableCell>
                </TableRow>
              ))}
              {jobs.length === 0 && !isLoading && (
                <TableRow>
                  <TableCell colSpan={5} sx={{ textAlign: 'center', py: 4 }}>
                    <Typography sx={{ color: '#9CA3AF' }}>No jobs found</Typography>
                  </TableCell>
                </TableRow>
              )}
            </TableBody>
          </Table>
        </TableContainer>
        <TablePagination
          component="div"
          count={total}
          page={page}
          onPageChange={(_, newPage) => setPage(newPage)}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={(e) => {
            setRowsPerPage(parseInt(e.target.value, 10));
            setPage(0);
          }}
          rowsPerPageOptions={[10, 25, 50]}
        />
      </Paper>
    </Box>
  );
}
```

**Step 2: Delete old Job components that use Tailwind**

Run:
```bash
rm -f web/frontend/src/components/Job/JobTable.tsx
```

**Step 3: Commit**

```bash
git add web/frontend/src/pages/Jobs.tsx
git add -A web/frontend/src/components/Job/
git commit -m "feat(frontend): rewrite Jobs page with MUI Table"
```

---

## Task 10: Update Workers Page

**Files:**
- Modify: `web/frontend/src/pages/Workers.tsx`

**Step 1: Rewrite Workers page with MUI**

Replace `web/frontend/src/pages/Workers.tsx`:
```typescript
import {
  Box,
  Typography,
  Button,
  Grid2 as Grid,
  Paper,
  CircularProgress,
} from '@mui/material';
import { RefreshOutlined } from '@mui/icons-material';
import { useQuery } from '@tanstack/react-query';
import { workersApi } from '../api/workers';
import { StatusChip } from '../components/StatusChip';
import type { Worker } from '../api/types';

export default function Workers() {
  const { data, isLoading, refetch } = useQuery({
    queryKey: ['workers'],
    queryFn: workersApi.getAll,
    refetchInterval: 10000,
  });

  const workers = data?.data || [];

  const formatUptime = (seconds: number) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    return `${hours}h ${minutes}m`;
  };

  const formatTimeAgo = (dateStr: string) => {
    const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
  };

  if (isLoading && workers.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress sx={{ color: '#000000' }} />
      </Box>
    );
  }

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, bgcolor: '#F9FAFB', minHeight: '100%' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Box>
          <Typography variant="h4" sx={{ fontWeight: 700, color: '#000000', mb: 0.5 }}>
            Workers
          </Typography>
          <Typography variant="body2" sx={{ color: '#6B7280' }}>
            {workers.filter((w: Worker) => w.status === 'online').length} of {workers.length} online
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<RefreshOutlined />}
          onClick={() => refetch()}
          sx={{ bgcolor: '#FFD93D', color: '#000000', border: '2px solid #000000' }}
        >
          Refresh
        </Button>
      </Box>

      {/* Worker Cards */}
      <Grid container spacing={2}>
        {workers.map((worker: Worker) => (
          <Grid key={worker.id} size={{ xs: 12, sm: 6, md: 4 }}>
            <Paper
              sx={{
                p: 3,
                borderRadius: '16px',
                border: '2px solid',
                borderColor: worker.status === 'online' ? '#22C55E' : worker.status === 'offline' ? '#EF4444' : '#F59E0B',
                bgcolor: worker.status === 'online' ? '#F0FDF4' : worker.status === 'offline' ? '#FEF2F2' : '#FFFBEB',
              }}
            >
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="h6" sx={{ fontWeight: 700 }}>
                  {worker.name}
                </Typography>
                <StatusChip status={worker.status} />
              </Box>

              <Box sx={{ display: 'grid', gap: 1 }}>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography variant="body2" sx={{ color: '#6B7280' }}>Jobs Completed</Typography>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>{worker.stats.jobs_completed}</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography variant="body2" sx={{ color: '#6B7280' }}>Uptime</Typography>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>{formatUptime(worker.stats.uptime_seconds)}</Typography>
                </Box>
                <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                  <Typography variant="body2" sx={{ color: '#6B7280' }}>Last Seen</Typography>
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>{formatTimeAgo(worker.last_seen)}</Typography>
                </Box>
                {worker.current_job_id && (
                  <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                    <Typography variant="body2" sx={{ color: '#6B7280' }}>Current Job</Typography>
                    <Typography variant="body2" sx={{ fontWeight: 600 }}>#{worker.current_job_id}</Typography>
                  </Box>
                )}
              </Box>
            </Paper>
          </Grid>
        ))}
        {workers.length === 0 && (
          <Grid size={12}>
            <Paper sx={{ p: 4, textAlign: 'center', border: '2px solid #E5E7EB' }}>
              <Typography sx={{ color: '#9CA3AF' }}>No workers connected</Typography>
            </Paper>
          </Grid>
        )}
      </Grid>
    </Box>
  );
}
```

**Step 2: Delete old Worker components**

Run:
```bash
rm -rf web/frontend/src/components/Worker/
```

**Step 3: Commit**

```bash
git add web/frontend/src/pages/Workers.tsx
git add -A web/frontend/src/components/Worker/
git commit -m "feat(frontend): rewrite Workers page with MUI cards"
```

---

## Task 11: Delete Unused UI Components

**Files:**
- Delete: `web/frontend/src/components/UI/`
- Delete: `web/frontend/src/lib/utils.ts`

**Step 1: Remove old Tailwind-based UI components**

Run:
```bash
rm -rf web/frontend/src/components/UI/
rm -f web/frontend/src/lib/utils.ts
rmdir web/frontend/src/lib 2>/dev/null || true
```

**Step 2: Commit**

```bash
git add -A web/frontend/src/components/UI/ web/frontend/src/lib/
git commit -m "chore(frontend): remove Tailwind UI components and utils"
```

---

## Task 12: Update Remaining Pages (Settings, Results, JobCreate, JobDetail, ProxyGate)

**Files:**
- Modify: `web/frontend/src/pages/Settings.tsx`
- Modify: `web/frontend/src/pages/Results.tsx`
- Modify: `web/frontend/src/pages/Jobs/JobCreate.tsx`
- Modify: `web/frontend/src/pages/Jobs/JobDetail.tsx`
- Modify: `web/frontend/src/pages/ProxyGate/index.tsx`

**Step 1: Update Settings page**

Replace `web/frontend/src/pages/Settings.tsx`:
```typescript
import { Box, Typography, Paper } from '@mui/material';

export default function Settings() {
  return (
    <Box sx={{ p: { xs: 2, md: 3 }, bgcolor: '#F9FAFB', minHeight: '100%' }}>
      <Typography variant="h4" sx={{ fontWeight: 700, color: '#000000', mb: 3 }}>
        Settings
      </Typography>
      <Paper sx={{ p: 3, border: '2px solid #000000', borderRadius: '16px' }}>
        <Typography variant="body1" sx={{ color: '#6B7280' }}>
          Settings page - coming soon
        </Typography>
      </Paper>
    </Box>
  );
}
```

**Step 2: Update ProxyGate page**

Replace `web/frontend/src/pages/ProxyGate/index.tsx`:
```typescript
import { Box, Typography, Paper } from '@mui/material';

export default function ProxyGate() {
  return (
    <Box sx={{ p: { xs: 2, md: 3 }, bgcolor: '#F9FAFB', minHeight: '100%' }}>
      <Typography variant="h4" sx={{ fontWeight: 700, color: '#000000', mb: 3 }}>
        Proxy Gate
      </Typography>
      <Paper sx={{ p: 3, border: '2px solid #000000', borderRadius: '16px' }}>
        <Typography variant="body1" sx={{ color: '#6B7280' }}>
          Proxy management - coming soon
        </Typography>
      </Paper>
    </Box>
  );
}
```

**Step 3: Commit**

```bash
git add web/frontend/src/pages/Settings.tsx web/frontend/src/pages/ProxyGate/index.tsx
git commit -m "feat(frontend): update Settings and ProxyGate pages with MUI"
```

---

## Task 13: Verify Build and Final Cleanup

**Step 1: Run build**

Run:
```bash
cd web/frontend && npm run build
```
Expected: Build succeeds with no errors

**Step 2: Fix any remaining TypeScript/import errors**

If there are errors, fix them (likely missing imports or path issues).

**Step 3: Run lint**

Run:
```bash
cd web/frontend && npm run lint
```
Expected: No critical errors

**Step 4: Final commit**

```bash
git add -A web/frontend/
git commit -m "chore(frontend): final cleanup and build verification"
```

---

## Verification Checklist

After completing all tasks, verify:

- [ ] `npm run build` passes without errors
- [ ] Dashboard displays 8 stat cards in grid
- [ ] Recent Jobs list shows with status chips
- [ ] Worker Status cards show with color-coded borders
- [ ] Login page works with API key
- [ ] Jobs page shows table with pagination
- [ ] Workers page shows status cards
- [ ] Mobile responsive (sidebar collapses, cards stack)
- [ ] Theme colors match golang-dashboard (yellow accent, black borders)
- [ ] No console errors in browser
