import { Button, Box, Typography } from "@mui/material"
import { JobForm } from "../../components/Job/JobForm"
import { ArrowBack } from "@mui/icons-material"
import { Link, useLocation } from "react-router-dom"
import type { Job } from "../../api/types"

interface LocationState {
  cloneFrom?: Job
  isRetry?: boolean
}

export default function JobCreate() {
  const location = useLocation()
  const state = location.state as LocationState | null
  const cloneFrom = state?.cloneFrom
  const isRetry = state?.isRetry

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, bgcolor: '#F9FAFB', minHeight: '100%' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 2, mb: 3 }}>
        <Button
          component={Link}
          to="/jobs"
          variant="text"
          sx={{ minWidth: 40, width: 40, height: 40, borderRadius: '50%', color: '#000000' }}
        >
          <ArrowBack />
        </Button>
        <Typography variant="h4" sx={{ fontWeight: 700, color: '#000000' }}>
          {isRetry ? 'Retry Failed Job' : cloneFrom ? 'Clone Job' : 'Create New Scraper Job'}
        </Typography>
      </Box>

      <Box sx={{ maxWidth: 800 }}>
        <JobForm cloneFrom={cloneFrom} isRetry={isRetry} />
      </Box>
    </Box>
  )
}
