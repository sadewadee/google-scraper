import { Box, Typography, Paper } from '@mui/material';

export default function ProxyGate() {
  return (
    <Box sx={{ p: { xs: 2, md: 3 }, bgcolor: '#F9FAFB', minHeight: '100%' }}>
      <Typography variant="h4" sx={{ fontWeight: 700, color: '#000000', mb: 3 }}>
        Proxy Gate
      </Typography>
      <Paper sx={{ p: 3, border: '2px solid #000000', borderRadius: '16px' }}>
        <Typography variant="body1" sx={{ color: '#6B7280' }}>
          Proxy management - coming soon
        </Typography>
      </Paper>
    </Box>
  );
}
