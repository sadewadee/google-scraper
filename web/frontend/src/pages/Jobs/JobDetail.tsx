import { useParams, Link, useNavigate } from "react-router-dom"
import { toast } from "sonner"
import {
  Box,
  Typography,
  Button,
  Paper,
  Grid,
  Chip,
  CircularProgress,
  LinearProgress
} from "@mui/material"
import {
  ArrowBack,
  PlayArrow,
  Pause,
  Stop,
  DeleteOutline,
  ContentCopy,
  Replay,
  AccessTime,
  LocationOn,
  StorageOutlined
} from "@mui/icons-material"
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { jobsApi } from "../../api/jobs"
import { StatusChip } from "../../components/StatusChip"
import { StatCard } from "../../components/StatCard"
import { ResultsTable } from "../../components/Results/ResultsTable"

export default function JobDetail() {
  const { id } = useParams()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const { data: job, isLoading } = useQuery({
    queryKey: ["job", id],
    queryFn: () => jobsApi.getOne(id!),
    enabled: !!id,
    refetchInterval: (query) => {
      const jobData = query.state.data
      return jobData?.status === 'running' || jobData?.status === 'pending' ? 3000 : false
    }
  })

  const pauseMutation = useMutation({
    mutationFn: jobsApi.pause,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job", id] })
      toast.success("Job paused")
    },
    onError: () => {
      toast.error("Failed to pause job")
    }
  })

  const resumeMutation = useMutation({
    mutationFn: jobsApi.resume,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job", id] })
      toast.success("Job resumed")
    },
    onError: () => {
      toast.error("Failed to resume job")
    }
  })

  const cancelMutation = useMutation({
    mutationFn: jobsApi.cancel,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["job", id] })
      toast.success("Job cancelled")
    },
    onError: () => {
      toast.error("Failed to cancel job")
    }
  })

  const deleteMutation = useMutation({
    mutationFn: jobsApi.delete,
    onSuccess: () => {
      toast.success("Job deleted")
      navigate("/jobs")
    },
    onError: () => {
      toast.error("Failed to delete job")
    }
  })

  if (isLoading) {
    return (
      <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', height: '50vh' }}>
        <CircularProgress sx={{ color: '#000000' }} />
      </Box>
    )
  }

  if (!job) {
    return (
      <Box sx={{ p: 4, textAlign: 'center' }}>
        <Typography>Job not found.</Typography>
      </Box>
    )
  }

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, bgcolor: '#F9FAFB', minHeight: '100%' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', flexDirection: { xs: 'column', md: 'row' }, justifyContent: 'space-between', alignItems: { xs: 'flex-start', md: 'center' }, gap: 2, mb: 3 }}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
          <Button
            component={Link}
            to="/jobs"
            variant="text"
            sx={{ minWidth: 40, width: 40, height: 40, borderRadius: '50%', color: '#000000' }}
          >
            <ArrowBack />
          </Button>
          <Box>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, mb: 0.5 }}>
              <Typography variant="h5" sx={{ fontWeight: 700, color: '#000000' }}>
                Job #{id}
              </Typography>
              <StatusChip status={job.status} />
            </Box>
            <Typography variant="body2" sx={{ color: '#6B7280' }}>
              Created on {new Date(job.created_at).toLocaleString()}
            </Typography>
          </Box>
        </Box>

        <Box sx={{ display: 'flex', gap: 1, flexWrap: 'wrap' }}>
          {job.status === 'failed' && (
            <Button
              variant="outlined"
              onClick={() => navigate('/jobs/new', { state: { cloneFrom: job, isRetry: true } })}
              startIcon={<Replay />}
              sx={{ borderColor: '#000000', color: '#000000' }}
            >
              Retry
            </Button>
          )}
          <Button
            variant="outlined"
            onClick={() => navigate('/jobs/new', { state: { cloneFrom: job } })}
            startIcon={<ContentCopy />}
            sx={{ borderColor: '#000000', color: '#000000' }}
          >
            Clone
          </Button>
          {job.status === 'running' && (
            <Button
              variant="outlined"
              onClick={() => pauseMutation.mutate(id!)}
              disabled={pauseMutation.isPending}
              startIcon={<Pause />}
              sx={{ borderColor: '#F59E0B', color: '#F59E0B' }}
            >
              Pause
            </Button>
          )}
          {job.status === 'paused' && (
            <Button
              variant="outlined"
              onClick={() => resumeMutation.mutate(id!)}
              disabled={resumeMutation.isPending}
              startIcon={<PlayArrow />}
              sx={{ borderColor: '#22C55E', color: '#22C55E' }}
            >
              Resume
            </Button>
          )}
          {(job.status === 'pending' || job.status === 'running' || job.status === 'paused') && (
            <Button
              variant="outlined"
              onClick={() => {
                if (confirm('Are you sure you want to cancel this job?')) {
                  cancelMutation.mutate(id!)
                }
              }}
              disabled={cancelMutation.isPending}
              startIcon={<Stop />}
              sx={{ borderColor: '#EF4444', color: '#EF4444' }}
            >
              Cancel
            </Button>
          )}
          <Button
            variant="contained"
            onClick={() => {
              if (confirm('Are you sure you want to delete this job?')) {
                deleteMutation.mutate(id!)
              }
            }}
            disabled={deleteMutation.isPending}
            startIcon={<DeleteOutline />}
            sx={{ bgcolor: '#EF4444', color: '#FFFFFF', border: '2px solid #EF4444', '&:hover': { bgcolor: '#DC2626', borderColor: '#DC2626' } }}
          >
            Delete
          </Button>
        </Box>
      </Box>

      {/* Stats and Config */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 4 }}>
          <StatCard
            title="Total Results"
            value={`${job.progress.scraped_places} / ${job.progress.total_places || "-"}`}
            subtitle={job.progress.percentage > 0 ? `${Math.round(job.progress.percentage)}% completed` : "Pending"}
            icon={StorageOutlined}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 4 }}>
          <StatCard
            title="Duration"
            value="--"
            subtitle="Processing time"
            icon={AccessTime}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 4 }}>
          <Paper sx={{ p: 2, border: '2px solid #000000', borderRadius: '16px', height: '100%', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
            <Typography variant="h6" sx={{ fontWeight: 700, mb: 1.5, fontSize: '1rem' }}>Configuration</Typography>
            <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between' }}>
                <Typography variant="body2" sx={{ color: '#6B7280' }}>Name:</Typography>
                <Typography variant="body2" sx={{ fontWeight: 600 }}>{job.name}</Typography>
              </Box>
              <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
                <Typography variant="body2" sx={{ color: '#6B7280' }}>Keywords:</Typography>
                <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5 }}>
                  {job.config.keywords.map((k, i) => (
                    <Chip key={i} label={k} size="small" variant="outlined" sx={{ height: 20, fontSize: '0.7rem' }} />
                  ))}
                </Box>
              </Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', mt: 0.5 }}>
                <Typography variant="body2" sx={{ color: '#6B7280' }}>Location:</Typography>
                <Box sx={{ display: 'flex', alignItems: 'center' }}>
                  <LocationOn sx={{ fontSize: 14, mr: 0.5, color: '#6B7280' }} />
                  <Typography variant="body2" sx={{ fontWeight: 600 }}>
                    {job.config.geo_lat && job.config.geo_lon
                      ? `${job.config.geo_lat}, ${job.config.geo_lon}`
                      : "Auto"}
                  </Typography>
                </Box>
              </Box>
            </Box>
          </Paper>
        </Grid>
      </Grid>

      {/* Progress Bar */}
      <Paper sx={{ p: 3, mb: 3, border: '2px solid #000000', borderRadius: '16px' }}>
        <Box sx={{ display: 'flex', justifyContent: 'space-between', mb: 1 }}>
          <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>Progress</Typography>
          <Typography variant="subtitle2" sx={{ fontWeight: 600 }}>{Math.round(job.progress.percentage)}%</Typography>
        </Box>
        <LinearProgress
          variant="determinate"
          value={job.progress.percentage}
          sx={{
            height: 10,
            borderRadius: 5,
            bgcolor: '#E5E7EB',
            '& .MuiLinearProgress-bar': { bgcolor: '#000000' }
          }}
        />
      </Paper>

      {/* Results Table */}
      <Paper sx={{ border: '2px solid #000000', borderRadius: '16px', overflow: 'hidden' }}>
        <Box sx={{ p: 2, borderBottom: '1px solid #E5E7EB' }}>
          <Typography variant="h6" sx={{ fontWeight: 700 }}>Scraped Results</Typography>
        </Box>
        <Box sx={{ p: 2 }}>
          <ResultsTable jobId={id!} />
        </Box>
      </Paper>
    </Box>
  )
}
