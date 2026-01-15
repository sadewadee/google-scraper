import { Chip } from '@mui/material';

type StatusType = 'pending' | 'queued' | 'running' | 'paused' | 'completed' | 'failed' | 'cancelled' | 'online' | 'offline' | 'busy';

interface StatusChipProps {
  status: StatusType | string;
  size?: 'small' | 'medium';
}

const getStatusColors = (status: string): { bg: string; color: string } => {
  switch (status.toLowerCase()) {
    case 'completed':
    case 'success':
    case 'online':
      return { bg: '#D1FAE5', color: '#065F46' };
    case 'failed':
    case 'error':
    case 'cancelled':
    case 'offline':
      return { bg: '#FEE2E2', color: '#991B1B' };
    case 'running':
    case 'busy':
    case 'processing':
      return { bg: '#DBEAFE', color: '#1E40AF' };
    case 'pending':
    case 'queued':
    case 'paused':
      return { bg: '#FEF3C7', color: '#92400E' };
    default:
      return { bg: '#E5E7EB', color: '#374151' };
  }
};

export const StatusChip = ({ status, size = 'small' }: StatusChipProps) => {
  const colors = getStatusColors(status);

  return (
    <Chip
      label={status}
      size={size}
      sx={{
        height: size === 'small' ? 22 : 28,
        fontSize: size === 'small' ? '0.688rem' : '0.813rem',
        fontWeight: 600,
        bgcolor: colors.bg,
        color: colors.color,
        border: 'none',
        textTransform: 'capitalize',
      }}
    />
  );
};
