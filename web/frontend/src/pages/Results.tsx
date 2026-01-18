import { useState, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { resultsApi } from "../api/results"
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
  TextField,
  InputAdornment,
  Grid,
  Chip,
  IconButton,
  Link,
  CircularProgress,
  Drawer
} from "@mui/material"
import {
  Search as SearchIcon,
  DownloadOutlined,
  StorageOutlined,
  EmailOutlined,
  PhoneOutlined,
  LanguageOutlined,
  Star,
  Map,
  CloseOutlined
} from "@mui/icons-material"
import { StatCard } from "../components/StatCard"
import type { ResultEntry } from "../api/types"
import { ExportDialog } from "../components/Results/ExportDialog"
import { BusinessDetail } from "../components/Results/BusinessDetail"

export default function Results() {
  const [search, setSearch] = useState("")
  const [page, setPage] = useState(0)
  const [rowsPerPage, setRowsPerPage] = useState(50)
  const [exportDialogOpen, setExportDialogOpen] = useState(false)
  const [selectedBusiness, setSelectedBusiness] = useState<ResultEntry | null>(null)

  // Fetch all results globally
  const { data: resultsData, isLoading } = useQuery({
    queryKey: ["results", page + 1, rowsPerPage],
    queryFn: () => resultsApi.getAll(page + 1, rowsPerPage),
  })

  // Filter results based on search (client-side filtering of current page)
  const filteredResults = useMemo(() => {
    if (!resultsData?.data) return []
    if (!search.trim()) return resultsData.data as ResultEntry[]

    const searchLower = search.toLowerCase()
    return (resultsData.data as ResultEntry[]).filter((entry) => {
      return (
        entry.title?.toLowerCase().includes(searchLower) ||
        entry.address?.toLowerCase().includes(searchLower) ||
        entry.phone?.toLowerCase().includes(searchLower) ||
        entry.category?.toLowerCase().includes(searchLower) ||
        entry.web_site?.toLowerCase().includes(searchLower) ||
        entry.emails?.some((e) => e.toLowerCase().includes(searchLower))
      )
    })
  }, [resultsData, search])

  const total = resultsData?.meta?.total || 0

  const handleExport = (format: 'csv' | 'xlsx' | 'json', columns: string[]) => {
    const url = resultsApi.downloadUrl(format, columns)
    window.open(url, '_blank')
  }

  return (
    <Box sx={{ p: { xs: 2, md: 3 }, bgcolor: '#F9FAFB', minHeight: '100%' }}>
      {/* Header */}
      <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 3 }}>
        <Typography variant="h4" sx={{ fontWeight: 700, color: '#000000' }}>
          Results Database
        </Typography>
      </Box>

      {/* Stats */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid size={{ xs: 12, sm: 4 }}>
          <StatCard
            title="Total Results"
            value={total}
            subtitle="Scraped listings"
            icon={StorageOutlined}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 4 }}>
          <StatCard
            title="Current Page"
            value={`${page + 1}`}
            subtitle={`of ${Math.ceil(total / rowsPerPage)}`}
            icon={Map}
          />
        </Grid>
        <Grid size={{ xs: 12, sm: 4 }}>
          <StatCard
            title="Showing"
            value={filteredResults.length}
            subtitle="On this page"
            icon={SearchIcon}
          />
        </Grid>
      </Grid>

      {/* Search and Export */}
      <Paper sx={{ p: 2, mb: 3, border: '2px solid #000000', borderRadius: '16px' }}>
        <Box sx={{ display: 'flex', alignItems: 'center', mb: 2 }}>
          <StorageOutlined sx={{ mr: 1 }} />
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            All Scraped Results
          </Typography>
        </Box>

        <Box sx={{ display: 'flex', flexDirection: { xs: 'column', md: 'row' }, gap: 2 }}>
          <Box sx={{ flex: 1, position: 'relative' }}>
            <TextField
              fullWidth
              placeholder="Filter by name, address, phone..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              slotProps={{
                input: {
                  startAdornment: (
                    <InputAdornment position="start">
                      <SearchIcon sx={{ color: '#9CA3AF' }} />
                    </InputAdornment>
                  ),
                  endAdornment: search && (
                    <InputAdornment position="end">
                      <IconButton size="small" onClick={() => setSearch("")}>
                        <CloseOutlined fontSize="small" />
                      </IconButton>
                    </InputAdornment>
                  )
                }
              }}
              sx={{ bgcolor: '#FFFFFF' }}
            />
          </Box>
          <Box sx={{ display: 'flex', gap: 1 }}>
            <Button
              variant="contained"
              onClick={() => setExportDialogOpen(true)}
              startIcon={<DownloadOutlined />}
              sx={{
                bgcolor: '#000000',
                color: '#FFFFFF',
                '&:hover': { bgcolor: '#333333' }
              }}
            >
              Export
            </Button>
          </Box>
        </Box>
      </Paper>

      {/* Results Table */}
      <Paper sx={{ border: '2px solid #000000', borderRadius: '16px', overflow: 'hidden' }}>
        <Box sx={{ p: 2, borderBottom: '1px solid #E5E7EB' }}>
          <Typography variant="h6" sx={{ fontWeight: 700 }}>
            {search
              ? `Showing ${filteredResults.length} results matching "${search}"`
              : `Showing ${filteredResults.length} of ${total} results`
            }
          </Typography>
        </Box>

        <TableContainer>
          <Table>
            <TableHead>
              <TableRow sx={{ bgcolor: '#F9FAFB' }}>
                <TableCell sx={{ fontWeight: 700 }}>Business Name</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Category</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Address</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Phone</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Email</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Rating</TableCell>
                <TableCell sx={{ fontWeight: 700 }}>Website</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {isLoading ? (
                <TableRow>
                  <TableCell colSpan={7} sx={{ textAlign: 'center', py: 8 }}>
                    <Box sx={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 2 }}>
                      <CircularProgress size={20} sx={{ color: '#000000' }} />
                      <Typography>Loading results...</Typography>
                    </Box>
                  </TableCell>
                </TableRow>
              ) : filteredResults.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={7} sx={{ textAlign: 'center', py: 8 }}>
                    <Typography sx={{ color: '#9CA3AF' }}>
                      {search ? "No results match your search." : "No results in database yet."}
                    </Typography>
                  </TableCell>
                </TableRow>
              ) : (
                filteredResults.map((entry, idx) => (
                  <TableRow key={entry.place_id || entry.cid || idx} hover>
                    <TableCell>
                      <Link
                        component="button"
                        variant="body2"
                        onClick={() => setSelectedBusiness(entry)}
                        underline="hover"
                        sx={{
                          fontWeight: 600,
                          maxWidth: 200,
                          whiteSpace: 'nowrap',
                          overflow: 'hidden',
                          textOverflow: 'ellipsis',
                          display: 'block',
                          color: 'inherit',
                          textAlign: 'left',
                          cursor: 'pointer'
                        }}
                        title={`View details for ${entry.title}`}
                      >
                        {entry.title}
                      </Link>
                    </TableCell>
                    <TableCell>
                      <Chip
                        label={entry.category || "-"}
                        size="small"
                        variant="outlined"
                        sx={{ fontSize: '0.75rem', borderRadius: '4px' }}
                      />
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center', maxWidth: 200 }}>
                        <Map sx={{ fontSize: 14, mr: 0.5, color: '#9CA3AF' }} />
                        <Typography variant="body2" sx={{ whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }} title={entry.address}>
                          {entry.address || "-"}
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      {entry.phone ? (
                        <Link href={`tel:${entry.phone}`} underline="hover" sx={{ display: 'flex', alignItems: 'center', color: 'inherit' }}>
                          <PhoneOutlined sx={{ fontSize: 14, mr: 0.5 }} />
                          <Typography variant="body2">{entry.phone}</Typography>
                        </Link>
                      ) : (
                        <Typography variant="body2" color="text.secondary">-</Typography>
                      )}
                    </TableCell>
                    <TableCell>
                      {entry.emails && entry.emails.length > 0 ? (
                        <Link href={`mailto:${entry.emails[0]}`} underline="hover" sx={{ display: 'flex', alignItems: 'center', color: 'inherit' }}>
                          <EmailOutlined sx={{ fontSize: 14, mr: 0.5 }} />
                          <Typography variant="body2">{entry.emails[0]}</Typography>
                        </Link>
                      ) : (
                        <Typography variant="body2" color="text.secondary">-</Typography>
                      )}
                    </TableCell>
                    <TableCell>
                      <Box sx={{ display: 'flex', alignItems: 'center' }}>
                        <Star sx={{ fontSize: 16, color: '#EAB308', mr: 0.5 }} />
                        <Typography variant="body2" sx={{ fontWeight: 600 }}>
                          {entry.review_rating?.toFixed(1) || "-"}
                        </Typography>
                        <Typography variant="caption" sx={{ color: '#9CA3AF', ml: 0.5 }}>
                          ({entry.review_count || 0})
                        </Typography>
                      </Box>
                    </TableCell>
                    <TableCell>
                      {entry.web_site ? (
                        <Link
                          href={entry.web_site}
                          target="_blank"
                          rel="noopener noreferrer"
                          underline="hover"
                          sx={{ display: 'flex', alignItems: 'center', color: 'inherit' }}
                        >
                          <LanguageOutlined sx={{ fontSize: 14, mr: 0.5 }} />
                          <Typography variant="body2" sx={{ maxWidth: 150, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis' }}>
                            {new URL(entry.web_site).hostname}
                          </Typography>
                        </Link>
                      ) : (
                        <Typography variant="body2" color="text.secondary">-</Typography>
                      )}
                    </TableCell>
                  </TableRow>
                ))
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
            setRowsPerPage(parseInt(e.target.value, 10))
            setPage(0)
          }}
          rowsPerPageOptions={[25, 50, 100]}
        />
      </Paper>

      <ExportDialog
        open={exportDialogOpen}
        onClose={() => setExportDialogOpen(false)}
        onExport={handleExport}
        totalResults={total}
      />

      {/* Business Detail Drawer */}
      <Drawer
        anchor="right"
        open={!!selectedBusiness}
        onClose={() => setSelectedBusiness(null)}
        PaperProps={{
          sx: { width: { xs: '100%', sm: 600 }, p: 0 }
        }}
      >
        {selectedBusiness && (
          <>
            <Box sx={{ p: 2, borderBottom: '1px solid', borderColor: 'divider', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
              <Typography sx={{ fontWeight: 600, fontSize: '1.125rem' }}>Business Details</Typography>
              <IconButton onClick={() => setSelectedBusiness(null)} size="small">
                <CloseOutlined />
              </IconButton>
            </Box>
            <Box sx={{ p: 3, overflowY: 'auto' }}>
              <BusinessDetail data={selectedBusiness} />
            </Box>
          </>
        )}
      </Drawer>
    </Box>
  )
}
