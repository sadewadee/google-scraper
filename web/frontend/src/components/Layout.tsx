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
