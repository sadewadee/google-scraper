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
  const total = jobs.length;

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
