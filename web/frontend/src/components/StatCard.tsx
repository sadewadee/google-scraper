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
