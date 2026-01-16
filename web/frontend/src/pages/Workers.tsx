import {
  Box,
  Typography,
  Button,
  Grid,
  Paper,
  Card,
  CardContent,
  CircularProgress,
  Chip,
  LinearProgress,
  Alert,
  useMediaQuery,
  useTheme,
} from '@mui/material';
import {
  RefreshOutlined,
  DnsOutlined,
  Speed,
  CheckCircle,
  Timer,
  FiberManualRecord,
  WorkOutline,
} from '@mui/icons-material';
import { useQuery } from '@tanstack/react-query';
import { workersApi } from '../api/workers';
import type { Worker } from '../api/types';

export default function Workers() {
  const theme = useTheme();
  const isMobile = useMediaQuery(theme.breakpoints.down('sm'));

  const { data, isLoading, error, refetch } = useQuery({
    queryKey: ['workers'],
    queryFn: workersApi.getAll,
    refetchInterval: 10000,
  });

  const workers = data?.data || [];

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'online': return { bg: '#D1FAE5', color: '#065F46', border: '#22C55E' };
      case 'offline': return { bg: '#FEE2E2', color: '#991B1B', border: '#EF4444' };
      case 'busy': return { bg: '#FEF3C7', color: '#92400E', border: '#F59E0B' };
      default: return { bg: '#E5E7EB', color: '#374151', border: '#9CA3AF' };
    }
  };

  const formatDuration = (seconds: number) => {
    if (seconds < 60) return `${seconds}s`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`;
    return `${Math.floor(seconds / 86400)}d`;
  };

  const formatTimeAgo = (dateStr: string) => {
    const seconds = Math.floor((Date.now() - new Date(dateStr).getTime()) / 1000);
    if (seconds < 0) return 'just now';
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
  };

  // Summary stats
  const onlineWorkers = workers.filter((w: Worker) => w.status === 'online' || w.status === 'busy').length;
  const busyWorkers = workers.filter((w: Worker) => w.status === 'busy').length;
  const totalJobsCompleted = workers.reduce((sum: number, w: Worker) => sum + (w.stats?.jobs_completed || 0), 0);
  const totalUptime = workers.reduce((sum: number, w: Worker) => sum + (w.stats?.uptime_seconds || 0), 0);

  if (isLoading && workers.length === 0) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress size={48} sx={{ color: '#000000' }} />
      </Box>
    );
  }

  if (error) {
    return (
      <Box sx={{ p: { xs: 2, md: 4 } }}>
        <Alert
          severity="error"
          action={<Button onClick={() => refetch()} sx={{ color: '#000' }}>Retry</Button>}
          sx={{ border: '2px solid #000000', borderRadius: 2 }}
        >
          Failed to fetch workers. Please try again.
        </Alert>
      </Box>
    );
  }

  return (
    <Box sx={{ p: { xs: 2, sm: 3, md: 4 }, bgcolor: '#F9FAFB', minHeight: '100%', width: '100%', boxSizing: 'border-box' }}>
      {/* Header */}
      <Box sx={{
        display: 'flex',
        flexDirection: { xs: 'column', sm: 'row' },
        justifyContent: 'space-between',
        alignItems: { xs: 'flex-start', sm: 'center' },
        mb: 4,
        gap: 2,
      }}>
        <Box>
          <Typography
            variant="h4"
            sx={{
              color: '#000000',
              fontWeight: 700,
              fontSize: { xs: '1.5rem', md: '1.75rem' },
              mb: 0.5,
            }}
          >
            Worker Management
          </Typography>
          <Typography variant="body2" sx={{ color: '#6B7280' }}>
            Monitor and manage scraping workers
          </Typography>
        </Box>
        <Button
          variant="contained"
          startIcon={<RefreshOutlined />}
          onClick={() => refetch()}
          disabled={isLoading}
          size={isMobile ? 'small' : 'medium'}
          sx={{
            bgcolor: '#FFD93D',
            color: '#000000',
            border: '2px solid #000000',
            fontWeight: 600,
            '&:hover': { bgcolor: '#FFC107' },
          }}
        >
          Refresh
        </Button>
      </Box>

      {/* Summary Cards */}
      <Grid container spacing={{ xs: 2, md: 3 }} sx={{ mb: 4 }}>
        <Grid size={{ xs: 6, md: 3 }}>
          <Paper sx={{ p: 3, border: '2px solid #000000', borderRadius: '16px' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Box sx={{
                p: 1.5,
                bgcolor: '#D1FAE5',
                borderRadius: '12px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}>
                <DnsOutlined sx={{ color: '#065F46', fontSize: 24 }} />
              </Box>
              <Box>
                <Typography variant="h5" sx={{ fontWeight: 700, color: '#000000' }}>
                  {onlineWorkers}/{workers.length}
                </Typography>
                <Typography variant="body2" sx={{ color: '#6B7280' }}>
                  Workers Online
                </Typography>
              </Box>
            </Box>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <Paper sx={{ p: 3, border: '2px solid #000000', borderRadius: '16px' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Box sx={{
                p: 1.5,
                bgcolor: '#FEF3C7',
                borderRadius: '12px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}>
                <Speed sx={{ color: '#92400E', fontSize: 24 }} />
              </Box>
              <Box>
                <Typography variant="h5" sx={{ fontWeight: 700, color: '#000000' }}>
                  {busyWorkers}
                </Typography>
                <Typography variant="body2" sx={{ color: '#6B7280' }}>
                  Currently Busy
                </Typography>
              </Box>
            </Box>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <Paper sx={{ p: 3, border: '2px solid #000000', borderRadius: '16px' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Box sx={{
                p: 1.5,
                bgcolor: '#DBEAFE',
                borderRadius: '12px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}>
                <CheckCircle sx={{ color: '#1D4ED8', fontSize: 24 }} />
              </Box>
              <Box>
                <Typography variant="h5" sx={{ fontWeight: 700, color: '#000000' }}>
                  {totalJobsCompleted}
                </Typography>
                <Typography variant="body2" sx={{ color: '#6B7280' }}>
                  Jobs Completed
                </Typography>
              </Box>
            </Box>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, md: 3 }}>
          <Paper sx={{ p: 3, border: '2px solid #000000', borderRadius: '16px' }}>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Box sx={{
                p: 1.5,
                bgcolor: '#D1FAE5',
                borderRadius: '12px',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
              }}>
                <Timer sx={{ color: '#065F46', fontSize: 24 }} />
              </Box>
              <Box>
                <Typography variant="h5" sx={{ fontWeight: 700, color: '#000000' }}>
                  {formatDuration(totalUptime)}
                </Typography>
                <Typography variant="body2" sx={{ color: '#6B7280' }}>
                  Total Uptime
                </Typography>
              </Box>
            </Box>
          </Paper>
        </Grid>
      </Grid>

      {/* Worker Cards */}
      <Grid container spacing={{ xs: 2, md: 3 }}>
        {workers.map((worker: Worker) => {
          const statusColor = getStatusColor(worker.status);
          const uptimeHours = (worker.stats?.uptime_seconds || 0) / 3600;

          return (
            <Grid size={{ xs: 12, md: 6, lg: 4 }} key={worker.id}>
              <Card
                sx={{
                  border: `2px solid ${statusColor.border}`,
                  borderRadius: '16px',
                  transition: 'transform 0.2s ease, box-shadow 0.2s ease',
                  '&:hover': {
                    transform: 'translateY(-4px)',
                    boxShadow: '0 8px 24px rgba(0,0,0,0.1)',
                  },
                }}
              >
                <CardContent sx={{ p: 3 }}>
                  {/* Header */}
                  <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', mb: 2 }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
                      <FiberManualRecord
                        sx={{
                          fontSize: 12,
                          color: worker.status === 'online' || worker.status === 'busy' ? '#22C55E' : '#EF4444',
                          animation: worker.status === 'online' || worker.status === 'busy' ? 'pulse 2s infinite' : 'none',
                          '@keyframes pulse': {
                            '0%': { opacity: 1 },
                            '50%': { opacity: 0.4 },
                            '100%': { opacity: 1 },
                          },
                        }}
                      />
                      <Typography variant="h6" sx={{ fontWeight: 700, color: '#000000' }}>
                        {worker.name}
                      </Typography>
                    </Box>
                    <Chip
                      label={worker.status.toUpperCase()}
                      size="small"
                      sx={{
                        bgcolor: statusColor.bg,
                        color: statusColor.color,
                        fontWeight: 700,
                        fontSize: '0.688rem',
                        border: 'none',
                      }}
                    />
                  </Box>

                  {/* Worker Info */}
                  <Box sx={{ mb: 3 }}>
                    <Typography variant="caption" sx={{ color: '#6B7280', display: 'block', mb: 0.5 }}>
                      ID: {worker.id.substring(0, 8)}...
                    </Typography>
                    {worker.current_job_id && (
                      <Chip
                        icon={<WorkOutline sx={{ fontSize: 14 }} />}
                        label={`Job #${worker.current_job_id}`}
                        size="small"
                        sx={{
                          bgcolor: '#F3F4F6',
                          color: '#374151',
                          fontWeight: 500,
                          fontSize: '0.75rem',
                          border: '1px solid #E5E7EB',
                          mt: 1,
                        }}
                      />
                    )}
                  </Box>

                  {/* Stats Grid */}
                  <Grid container spacing={2} sx={{ mb: 3 }}>
                    <Grid size={{ xs: 6 }}>
                      <Box sx={{ p: 2, bgcolor: '#F9FAFB', borderRadius: '12px', border: '1px solid #E5E7EB' }}>
                        <Typography variant="caption" sx={{ color: '#6B7280', display: 'block' }}>
                          Jobs Done
                        </Typography>
                        <Typography variant="h6" sx={{ fontWeight: 700, color: '#000000' }}>
                          {worker.stats?.jobs_completed || 0}
                        </Typography>
                      </Box>
                    </Grid>
                    <Grid size={{ xs: 6 }}>
                      <Box sx={{ p: 2, bgcolor: '#F9FAFB', borderRadius: '12px', border: '1px solid #E5E7EB' }}>
                        <Typography variant="caption" sx={{ color: '#6B7280', display: 'block' }}>
                          Uptime
                        </Typography>
                        <Typography variant="h6" sx={{ fontWeight: 700, color: '#000000' }}>
                          {formatDuration(worker.stats?.uptime_seconds || 0)}
                        </Typography>
                      </Box>
                    </Grid>
                  </Grid>

                  {/* Uptime Progress */}
                  <Box sx={{ mb: 2 }}>
                    <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
                      <Typography variant="caption" sx={{ color: '#6B7280', fontWeight: 600 }}>
                        Session Duration
                      </Typography>
                      <Typography variant="caption" sx={{ color: '#000000', fontWeight: 700 }}>
                        {uptimeHours.toFixed(1)}h
                      </Typography>
                    </Box>
                    <LinearProgress
                      variant="determinate"
                      value={Math.min(uptimeHours / 24 * 100, 100)}
                      sx={{
                        height: 8,
                        borderRadius: 4,
                        bgcolor: '#E5E7EB',
                        '& .MuiLinearProgress-bar': {
                          borderRadius: 4,
                          bgcolor: worker.status === 'online' ? '#22C55E' : worker.status === 'busy' ? '#F59E0B' : '#9CA3AF',
                        },
                      }}
                    />
                  </Box>

                  {/* Last Seen */}
                  <Box sx={{ display: 'flex', justifyContent: 'flex-end', alignItems: 'center' }}>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 0.5 }}>
                      <Timer sx={{ fontSize: 14, color: '#9CA3AF' }} />
                      <Typography variant="caption" sx={{ color: '#9CA3AF' }}>
                        {formatTimeAgo(worker.last_seen)}
                      </Typography>
                    </Box>
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          );
        })}

        {workers.length === 0 && (
          <Grid size={12}>
            <Paper sx={{ p: 6, textAlign: 'center', border: '2px solid #E5E7EB', borderRadius: '16px' }}>
              <DnsOutlined sx={{ fontSize: 48, color: '#9CA3AF', mb: 2 }} />
              <Typography variant="h6" sx={{ color: '#6B7280', mb: 1 }}>
                No workers connected
              </Typography>
              <Typography variant="body2" sx={{ color: '#9CA3AF' }}>
                Start a worker instance to begin processing jobs
              </Typography>
            </Paper>
          </Grid>
        )}
      </Grid>
    </Box>
  );
}
