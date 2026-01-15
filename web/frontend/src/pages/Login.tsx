import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Button,
  Alert,
  Avatar,
} from '@mui/material';
import { LockOutlined } from '@mui/icons-material';
import { setApiKey } from '../api/client';

export default function Login() {
  const [apiKey, setApiKeyValue] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const navigate = useNavigate();

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);

    if (!apiKey.trim()) {
      setError('API Key is required');
      setLoading(false);
      return;
    }

    try {
      setApiKey(apiKey.trim());
      navigate('/');
    } catch {
      setError('Invalid API Key');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Box
      sx={{
        minHeight: '100vh',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        bgcolor: '#F9FAFB',
        p: 2,
      }}
    >
      <Card sx={{ maxWidth: 400, width: '100%', border: '2px solid #000000' }}>
        <CardContent sx={{ p: 4 }}>
          <Box sx={{ display: 'flex', flexDirection: 'column', alignItems: 'center', mb: 3 }}>
            <Avatar
              sx={{
                width: 56,
                height: 56,
                bgcolor: '#FFD93D',
                color: '#000000',
                border: '2px solid #000000',
                mb: 2,
              }}
            >
              <LockOutlined />
            </Avatar>
            <Typography variant="h5" sx={{ fontWeight: 700, color: '#000000' }}>
              GMaps Scraper
            </Typography>
            <Typography variant="body2" sx={{ color: '#6B7280', mt: 0.5 }}>
              Enter your API key to continue
            </Typography>
          </Box>

          {error && (
            <Alert severity="error" sx={{ mb: 2, border: '1px solid #EF4444' }}>
              {error}
            </Alert>
          )}

          <form onSubmit={handleSubmit}>
            <TextField
              fullWidth
              label="API Key"
              type="password"
              value={apiKey}
              onChange={(e) => setApiKeyValue(e.target.value)}
              placeholder="Enter your API key"
              sx={{ mb: 3 }}
            />

            <Button
              fullWidth
              type="submit"
              variant="contained"
              disabled={loading}
              sx={{
                py: 1.5,
                bgcolor: '#FFD93D',
                color: '#000000',
                border: '2px solid #000000',
                fontWeight: 600,
                '&:hover': { bgcolor: '#FFC107' },
              }}
            >
              {loading ? 'Signing in...' : 'Sign In'}
            </Button>
          </form>
        </CardContent>
      </Card>
    </Box>
  );
}
