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
