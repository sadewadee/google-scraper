import { useState } from 'react';
import {
  Box,
  Typography,
  Paper,
  Button,
  TextField,
  IconButton,
  Grid,
  CircularProgress,
  Chip,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Tooltip,
  Tabs,
  Tab,
  TablePagination,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
} from '@mui/material';
import {
  RefreshOutlined,
  AddOutlined,
  DeleteOutlined,
  PublicOutlined,
  CheckCircleOutlined,
  WarningOutlined,
  CleaningServicesOutlined,
  ErrorOutlined,
  HourglassEmptyOutlined,
  BlockOutlined,
} from '@mui/icons-material';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { toast } from 'sonner';
import { proxyApi } from '../../api/proxy';
import type { ProxySource, Proxy } from '../../api/types';

export default function ProxyGate() {
  const queryClient = useQueryClient();
  const [addDialogOpen, setAddDialogOpen] = useState(false);
  const [newSourceUrl, setNewSourceUrl] = useState('');
  const [activeTab, setActiveTab] = useState(0);
  const [proxyPage, setProxyPage] = useState(1);
  const [proxyLimit] = useState(25);
  const [statusFilter, setStatusFilter] = useState<string>('');

  // Fetch stats
  const { data: statsData, isLoading: statsLoading } = useQuery({
    queryKey: ['proxygate-stats'],
    queryFn: proxyApi.getStats,
    refetchInterval: 30000,
  });

  // Fetch sources
  const { data: sourcesData, isLoading: sourcesLoading } = useQuery({
    queryKey: ['proxygate-sources'],
    queryFn: proxyApi.getSources,
    refetchInterval: 30000,
  });

  // Fetch proxy list
  const { data: proxiesData, isLoading: proxiesLoading } = useQuery({
    queryKey: ['proxygate-proxies', proxyPage, proxyLimit, statusFilter],
    queryFn: () => proxyApi.getProxies({
      page: proxyPage,
      limit: proxyLimit,
      status: statusFilter as 'pending' | 'healthy' | 'dead' | 'banned' | undefined,
    }),
    refetchInterval: 30000,
  });

  // Refresh mutation
  const refreshMutation = useMutation({
    mutationFn: proxyApi.refresh,
    onSuccess: () => {
      toast.success('Refresh triggered');
      queryClient.invalidateQueries({ queryKey: ['proxygate-stats'] });
    },
    onError: () => {
      toast.error('Failed to refresh proxies');
    },
  });

  // Add source mutation
  const addSourceMutation = useMutation({
    mutationFn: proxyApi.addSource,
    onSuccess: () => {
      toast.success('Source added');
      queryClient.invalidateQueries({ queryKey: ['proxygate-sources'] });
      setAddDialogOpen(false);
      setNewSourceUrl('');
    },
    onError: () => {
      toast.error('Failed to add source');
    },
  });

  // Delete source mutation
  const deleteSourceMutation = useMutation({
    mutationFn: proxyApi.deleteSource,
    onSuccess: () => {
      toast.success('Source deleted');
      queryClient.invalidateQueries({ queryKey: ['proxygate-sources'] });
    },
    onError: () => {
      toast.error('Failed to delete source');
    },
  });

  // Cleanup dead proxies mutation
  const cleanupMutation = useMutation({
    mutationFn: proxyApi.cleanupDeadProxies,
    onSuccess: (data) => {
      toast.success(`Cleaned up ${data.data.count} dead proxies`);
      queryClient.invalidateQueries({ queryKey: ['proxygate-proxies'] });
      queryClient.invalidateQueries({ queryKey: ['proxygate-stats'] });
    },
    onError: () => {
      toast.error('Failed to cleanup dead proxies');
    },
  });

  const stats = statsData?.data;
  const sources = sourcesData?.data || [];
  const proxies = proxiesData?.data || [];
  const proxyMeta = proxiesData?.meta || { total: 0, page: 1, limit: 25 };

  const formatLastUpdated = (dateStr: string) => {
    if (dateStr === 'never' || dateStr === 'not enabled') return dateStr;
    const date = new Date(dateStr);
    const seconds = Math.floor((Date.now() - date.getTime()) / 1000);
    if (seconds < 60) return `${seconds}s ago`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}m ago`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}h ago`;
    return `${Math.floor(seconds / 86400)}d ago`;
  };

  const handleAddSource = () => {
    if (!newSourceUrl.trim()) {
      toast.error('URL is required');
      return;
    }
    addSourceMutation.mutate(newSourceUrl.trim());
  };

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'healthy': return { bg: '#DCFCE7', color: '#166534' };
      case 'pending': return { bg: '#FEF3C7', color: '#92400E' };
      case 'dead': return { bg: '#FEE2E2', color: '#991B1B' };
      case 'banned': return { bg: '#FEE2E2', color: '#7C2D12' };
      default: return { bg: '#F3F4F6', color: '#374151' };
    }
  };

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'healthy': return <CheckCircleOutlined sx={{ fontSize: 14 }} />;
      case 'pending': return <HourglassEmptyOutlined sx={{ fontSize: 14 }} />;
      case 'dead': return <ErrorOutlined sx={{ fontSize: 14 }} />;
      case 'banned': return <BlockOutlined sx={{ fontSize: 14 }} />;
      default: return undefined;
    }
  };

  const formatTime = (ms?: number) => {
    if (!ms) return '-';
    if (ms < 1000) return `${ms}ms`;
    return `${(ms / 1000).toFixed(1)}s`;
  };

  if (statsLoading && sourcesLoading) {
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
            Proxy Gate
          </Typography>
          <Typography variant="body2" sx={{ color: '#6B7280' }}>
            Manage proxy sources and monitor pool health
          </Typography>
        </Box>
        <Box sx={{ display: 'flex', gap: 1 }}>
          <Button
            variant="contained"
            startIcon={<RefreshOutlined />}
            onClick={() => refreshMutation.mutate()}
            disabled={refreshMutation.isPending}
            sx={{ bgcolor: '#FFD93D', color: '#000000', border: '2px solid #000000' }}
          >
            {refreshMutation.isPending ? 'Refreshing...' : 'Refresh'}
          </Button>
          <Button
            variant="contained"
            startIcon={<AddOutlined />}
            onClick={() => setAddDialogOpen(true)}
            sx={{ bgcolor: '#22C55E', color: '#FFFFFF', border: '2px solid #000000' }}
          >
            Add Source
          </Button>
        </Box>
      </Box>

      {/* Stats Cards */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper
            sx={{
              p: 3,
              borderRadius: '16px',
              border: '2px solid #000000',
              bgcolor: '#F0FDF4',
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
              <PublicOutlined sx={{ color: '#22C55E' }} />
              <Typography variant="body2" sx={{ color: '#6B7280' }}>
                Total
              </Typography>
            </Box>
            <Typography variant="h3" sx={{ fontWeight: 700, color: '#000000' }}>
              {stats?.total_proxies ?? 0}
            </Typography>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper
            sx={{
              p: 3,
              borderRadius: '16px',
              border: '2px solid #000000',
              bgcolor: '#ECFDF5',
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
              <CheckCircleOutlined sx={{ color: '#10B981' }} />
              <Typography variant="body2" sx={{ color: '#6B7280' }}>
                Healthy
              </Typography>
            </Box>
            <Typography variant="h3" sx={{ fontWeight: 700, color: '#000000' }}>
              {stats?.healthy_proxies ?? 0}
            </Typography>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper
            sx={{
              p: 3,
              borderRadius: '16px',
              border: '2px solid #000000',
              bgcolor: '#FEF2F2',
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
              <ErrorOutlined sx={{ color: '#EF4444' }} />
              <Typography variant="body2" sx={{ color: '#6B7280' }}>
                Dead
              </Typography>
            </Box>
            <Typography variant="h3" sx={{ fontWeight: 700, color: '#000000' }}>
              {stats?.dead_proxies ?? 0}
            </Typography>
          </Paper>
        </Grid>
        <Grid size={{ xs: 6, sm: 3 }}>
          <Paper
            sx={{
              p: 3,
              borderRadius: '16px',
              border: '2px solid #000000',
              bgcolor: '#FFFBEB',
            }}
          >
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, mb: 1 }}>
              <WarningOutlined sx={{ color: '#F59E0B' }} />
              <Typography variant="body2" sx={{ color: '#6B7280' }}>
                Last Updated
              </Typography>
            </Box>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#000000' }}>
              {stats?.last_updated ? formatLastUpdated(stats.last_updated) : 'Never'}
            </Typography>
          </Paper>
        </Grid>
      </Grid>

      {/* Tabs */}
      <Paper sx={{ border: '2px solid #000000', borderRadius: '16px', overflow: 'hidden' }}>
        <Tabs
          value={activeTab}
          onChange={(_, v) => setActiveTab(v)}
          sx={{
            borderBottom: '2px solid #000000',
            bgcolor: '#F9FAFB',
            '& .MuiTab-root': { fontWeight: 600 },
          }}
        >
          <Tab label={`Sources (${sources.length})`} />
          <Tab label={`Proxies (${proxyMeta.total})`} />
        </Tabs>

        <Box sx={{ p: 3 }}>
          {activeTab === 0 && (
            <>
              {/* Sources Table */}
              {sources.length === 0 ? (
                <Box sx={{ textAlign: 'center', py: 4 }}>
                  <PublicOutlined sx={{ fontSize: 48, color: '#D1D5DB', mb: 2 }} />
                  <Typography sx={{ color: '#9CA3AF' }}>No proxy sources configured</Typography>
                  <Typography variant="body2" sx={{ color: '#9CA3AF', mb: 2 }}>
                    Add a source URL to start fetching proxies
                  </Typography>
                  <Button
                    variant="outlined"
                    startIcon={<AddOutlined />}
                    onClick={() => setAddDialogOpen(true)}
                    sx={{ borderColor: '#000000', color: '#000000' }}
                  >
                    Add First Source
                  </Button>
                </Box>
              ) : (
                <TableContainer>
                  <Table>
                    <TableHead>
                      <TableRow>
                        <TableCell sx={{ fontWeight: 600 }}>URL</TableCell>
                        <TableCell sx={{ fontWeight: 600 }}>Status</TableCell>
                        <TableCell sx={{ fontWeight: 600 }}>Added</TableCell>
                        <TableCell sx={{ fontWeight: 600, width: 80 }}>Actions</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {sources.map((source: ProxySource) => (
                        <TableRow key={source.id} hover>
                          <TableCell>
                            <Tooltip title={source.url}>
                              <Typography
                                variant="body2"
                                sx={{
                                  maxWidth: 400,
                                  overflow: 'hidden',
                                  textOverflow: 'ellipsis',
                                  whiteSpace: 'nowrap',
                                }}
                              >
                                {source.url}
                              </Typography>
                            </Tooltip>
                          </TableCell>
                          <TableCell>
                            <Chip
                              label={source.status === 'ok' ? 'Active' : 'Error'}
                              size="small"
                              sx={{
                                bgcolor: source.status === 'ok' ? '#DCFCE7' : '#FEE2E2',
                                color: source.status === 'ok' ? '#166534' : '#991B1B',
                                fontWeight: 600,
                              }}
                            />
                          </TableCell>
                          <TableCell>
                            <Typography variant="body2" sx={{ color: '#6B7280' }}>
                              {source.created_at
                                ? new Date(source.created_at).toLocaleDateString()
                                : '-'}
                            </Typography>
                          </TableCell>
                          <TableCell>
                            <IconButton
                              size="small"
                              onClick={() => deleteSourceMutation.mutate(source.id)}
                              disabled={deleteSourceMutation.isPending}
                              sx={{ color: '#EF4444' }}
                            >
                              <DeleteOutlined fontSize="small" />
                            </IconButton>
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </TableContainer>
              )}
            </>
          )}

          {activeTab === 1 && (
            <>
              {/* Proxies Filter & Actions */}
              <Box sx={{ display: 'flex', gap: 2, mb: 2, alignItems: 'center' }}>
                <FormControl size="small" sx={{ minWidth: 150 }}>
                  <InputLabel>Status</InputLabel>
                  <Select
                    value={statusFilter}
                    label="Status"
                    onChange={(e) => {
                      setStatusFilter(e.target.value);
                      setProxyPage(1);
                    }}
                  >
                    <MenuItem value="">All</MenuItem>
                    <MenuItem value="healthy">Healthy</MenuItem>
                    <MenuItem value="pending">Pending</MenuItem>
                    <MenuItem value="dead">Dead</MenuItem>
                    <MenuItem value="banned">Banned</MenuItem>
                  </Select>
                </FormControl>
                <Box sx={{ flex: 1 }} />
                <Button
                  variant="outlined"
                  size="small"
                  startIcon={<CleaningServicesOutlined />}
                  onClick={() => cleanupMutation.mutate()}
                  disabled={cleanupMutation.isPending}
                  sx={{ borderColor: '#EF4444', color: '#EF4444' }}
                >
                  {cleanupMutation.isPending ? 'Cleaning...' : 'Cleanup Dead'}
                </Button>
              </Box>

              {/* Proxies Table */}
              {proxiesLoading ? (
                <Box sx={{ display: 'flex', justifyContent: 'center', py: 4 }}>
                  <CircularProgress sx={{ color: '#000000' }} />
                </Box>
              ) : proxies.length === 0 ? (
                <Box sx={{ textAlign: 'center', py: 4 }}>
                  <PublicOutlined sx={{ fontSize: 48, color: '#D1D5DB', mb: 2 }} />
                  <Typography sx={{ color: '#9CA3AF' }}>No proxies found</Typography>
                  <Typography variant="body2" sx={{ color: '#9CA3AF' }}>
                    Refresh proxy sources to fetch new proxies
                  </Typography>
                </Box>
              ) : (
                <>
                  <TableContainer>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell sx={{ fontWeight: 600 }}>Address</TableCell>
                          <TableCell sx={{ fontWeight: 600 }}>Country</TableCell>
                          <TableCell sx={{ fontWeight: 600 }}>Status</TableCell>
                          <TableCell sx={{ fontWeight: 600 }}>Uptime</TableCell>
                          <TableCell sx={{ fontWeight: 600 }}>Response</TableCell>
                          <TableCell sx={{ fontWeight: 600 }}>Success/Fail</TableCell>
                          <TableCell sx={{ fontWeight: 600 }}>Last Used</TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {proxies.map((proxy: Proxy) => (
                          <TableRow key={proxy.id} hover>
                            <TableCell>
                              <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                                {proxy.ip}:{proxy.port}
                              </Typography>
                            </TableCell>
                            <TableCell>
                              <Typography variant="body2">
                                {proxy.country || '-'}
                              </Typography>
                            </TableCell>
                            <TableCell>
                              <Chip
                                icon={getStatusIcon(proxy.status)}
                                label={proxy.status}
                                size="small"
                                sx={{
                                  bgcolor: getStatusColor(proxy.status).bg,
                                  color: getStatusColor(proxy.status).color,
                                  fontWeight: 600,
                                  textTransform: 'capitalize',
                                }}
                              />
                            </TableCell>
                            <TableCell>
                              <Typography variant="body2">
                                {proxy.uptime ? `${proxy.uptime.toFixed(0)}%` : '-'}
                              </Typography>
                            </TableCell>
                            <TableCell>
                              <Typography variant="body2">
                                {formatTime(proxy.response_time)}
                              </Typography>
                            </TableCell>
                            <TableCell>
                              <Typography variant="body2">
                                <span style={{ color: '#22C55E' }}>{proxy.success_count}</span>
                                {' / '}
                                <span style={{ color: '#EF4444' }}>{proxy.fail_count}</span>
                              </Typography>
                            </TableCell>
                            <TableCell>
                              <Typography variant="body2" sx={{ color: '#6B7280' }}>
                                {proxy.last_used
                                  ? new Date(proxy.last_used).toLocaleString()
                                  : 'Never'}
                              </Typography>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  </TableContainer>
                  <TablePagination
                    component="div"
                    count={proxyMeta.total}
                    page={proxyPage - 1}
                    onPageChange={(_, page) => setProxyPage(page + 1)}
                    rowsPerPage={proxyLimit}
                    rowsPerPageOptions={[25]}
                    sx={{ borderTop: '1px solid #E5E7EB' }}
                  />
                </>
              )}
            </>
          )}
        </Box>
      </Paper>

      {/* Add Source Dialog */}
      <Dialog open={addDialogOpen} onClose={() => setAddDialogOpen(false)} maxWidth="sm" fullWidth>
        <DialogTitle sx={{ fontWeight: 700 }}>Add Proxy Source</DialogTitle>
        <DialogContent>
          <Typography variant="body2" sx={{ color: '#6B7280', mb: 2 }}>
            Enter a URL that returns a list of proxies (one IP:port per line)
          </Typography>
          <TextField
            autoFocus
            fullWidth
            label="Source URL"
            placeholder="https://raw.githubusercontent.com/..."
            value={newSourceUrl}
            onChange={(e) => setNewSourceUrl(e.target.value)}
            sx={{ mt: 1 }}
          />
          <Typography variant="caption" sx={{ color: '#9CA3AF', mt: 1, display: 'block' }}>
            Example: https://raw.githubusercontent.com/TheSpeedX/SOCKS-List/master/socks5.txt
          </Typography>
        </DialogContent>
        <DialogActions sx={{ p: 2 }}>
          <Button onClick={() => setAddDialogOpen(false)} sx={{ color: '#6B7280' }}>
            Cancel
          </Button>
          <Button
            onClick={handleAddSource}
            variant="contained"
            disabled={addSourceMutation.isPending}
            sx={{ bgcolor: '#22C55E', color: '#FFFFFF' }}
          >
            {addSourceMutation.isPending ? 'Adding...' : 'Add Source'}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  );
}
