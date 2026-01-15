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
