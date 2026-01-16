import { useState } from 'react';
import {
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  FormControl,
  FormLabel,
  RadioGroup,
  FormControlLabel,
  Radio,
  FormGroup,
  Checkbox,
  Typography,
  Box,
  Alert,
  Grid
} from '@mui/material';
import { DownloadOutlined } from '@mui/icons-material';

interface ExportDialogProps {
  open: boolean;
  onClose: () => void;
  onExport: (format: 'csv' | 'xlsx' | 'json', columns: string[]) => void;
  totalResults: number;
}

const AVAILABLE_COLUMNS = [
  { id: 'Title', label: 'Business Name', category: 'basic' },
  { id: 'Address', label: 'Address', category: 'basic' },
  { id: 'Category', label: 'Category', category: 'basic' },
  { id: 'Website', label: 'Website', category: 'contact' },
  { id: 'Phone', label: 'Phone', category: 'contact' },
  { id: 'Email', label: 'Email', category: 'contact' },
  { id: 'Rating', label: 'Rating', category: 'metrics' },
  { id: 'Reviews', label: 'Review Count', category: 'metrics' },
  { id: 'Google Maps URL', label: 'Google Maps Link', category: 'meta' },
  { id: 'Place ID', label: 'Place ID', category: 'meta' },
  { id: 'Latitude', label: 'Latitude', category: 'location' },
  { id: 'Longitude', label: 'Longitude', category: 'location' },
  { id: 'Timezone', label: 'Timezone', category: 'location' },
  { id: 'Opening Hours', label: 'Opening Hours', category: 'details' },
  { id: 'Price Range', label: 'Price Range', category: 'details' },
  { id: 'Status', label: 'Status', category: 'meta' },
];

const DEFAULT_COLUMNS = [
  'Title', 'Address', 'Phone', 'Website', 'Category', 'Rating', 'Reviews', 'Email'
];

export const ExportDialog = ({ open, onClose, onExport, totalResults }: ExportDialogProps) => {
  const [format, setFormat] = useState<'csv' | 'xlsx' | 'json'>('csv');
  const [selectedColumns, setSelectedColumns] = useState<string[]>(DEFAULT_COLUMNS);

  const handleColumnToggle = (columnId: string) => {
    setSelectedColumns(prev =>
      prev.includes(columnId)
        ? prev.filter(id => id !== columnId)
        : [...prev, columnId]
    );
  };

  const handleSelectAll = (checked: boolean) => {
    if (checked) {
      setSelectedColumns(AVAILABLE_COLUMNS.map(c => c.id));
    } else {
      setSelectedColumns([]);
    }
  };

  const handleExport = () => {
    onExport(format, selectedColumns);
    onClose();
  };

  // Group columns by category for better UI
  const columnsByCategory = AVAILABLE_COLUMNS.reduce((acc, col) => {
    if (!acc[col.category]) acc[col.category] = [];
    acc[col.category].push(col);
    return acc;
  }, {} as Record<string, typeof AVAILABLE_COLUMNS>);

  return (
    <Dialog open={open} onClose={onClose} maxWidth="md" fullWidth>
      <DialogTitle sx={{ fontWeight: 700 }}>Export Results</DialogTitle>
      <DialogContent dividers>
        <Box sx={{ mb: 3 }}>
          <Alert severity="info" sx={{ mb: 3 }}>
            You are about to export <strong>{totalResults.toLocaleString()}</strong> results.
          </Alert>

          <FormControl component="fieldset" sx={{ mb: 4, width: '100%' }}>
            <FormLabel component="legend" sx={{ fontWeight: 600, color: 'text.primary', mb: 1 }}>
              Export Format
            </FormLabel>
            <RadioGroup
              row
              value={format}
              onChange={(e) => setFormat(e.target.value as any)}
            >
              <FormControlLabel value="csv" control={<Radio />} label="CSV" />
              <FormControlLabel value="xlsx" control={<Radio />} label="Excel (XLSX)" />
              <FormControlLabel value="json" control={<Radio />} label="JSON" />
            </RadioGroup>
          </FormControl>

          {format !== 'json' && (
            <Box>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 2 }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                  Select Columns
                </Typography>
                <FormControlLabel
                  control={
                    <Checkbox
                      checked={selectedColumns.length === AVAILABLE_COLUMNS.length}
                      indeterminate={selectedColumns.length > 0 && selectedColumns.length < AVAILABLE_COLUMNS.length}
                      onChange={(e) => handleSelectAll(e.target.checked)}
                      size="small"
                    />
                  }
                  label="Select All"
                />
              </Box>

              <Grid container spacing={2}>
                {Object.entries(columnsByCategory).map(([category, columns]) => (
                  <Grid size={{ xs: 12, sm: 6, md: 4 }} key={category}>
                    <Box sx={{ mb: 2 }}>
                      <Typography variant="subtitle2" sx={{ textTransform: 'capitalize', color: 'text.secondary', mb: 1 }}>
                        {category}
                      </Typography>
                      <FormGroup>
                        {columns.map(col => (
                          <FormControlLabel
                            key={col.id}
                            control={
                              <Checkbox
                                size="small"
                                checked={selectedColumns.includes(col.id)}
                                onChange={() => handleColumnToggle(col.id)}
                              />
                            }
                            label={<Typography variant="body2">{col.label}</Typography>}
                          />
                        ))}
                      </FormGroup>
                    </Box>
                  </Grid>
                ))}
              </Grid>
            </Box>
          )}
        </Box>
      </DialogContent>
      <DialogActions sx={{ p: 2.5 }}>
        <Button onClick={onClose} color="inherit">
          Cancel
        </Button>
        <Button
          variant="contained"
          onClick={handleExport}
          startIcon={<DownloadOutlined />}
          disabled={format !== 'json' && selectedColumns.length === 0}
          sx={{
            bgcolor: '#000000',
            '&:hover': { bgcolor: '#333333' }
          }}
        >
          Download Export
        </Button>
      </DialogActions>
    </Dialog>
  );
};
