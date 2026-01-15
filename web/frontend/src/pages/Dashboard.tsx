import {
  Box,
  Typography,
  Button,
  CircularProgress,
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
