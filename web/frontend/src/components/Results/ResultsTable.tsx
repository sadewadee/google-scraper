import { useState, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { jobsApi } from "@/api/jobs"
import type { ResultEntry } from "@/api/types"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableRow,
    TableContainer,
    Paper,
    Button,
    TextField,
    Chip as Badge,
    Drawer,
    Box,
    InputAdornment,
    IconButton
} from "@mui/material"
import { BusinessDetail } from "./BusinessDetail"
import {
    Search,
    Download,
    ChevronLeft,
    ChevronRight,
    Settings,
    Star,
    OpenInNew as ExternalLink,
    Mail,
    Phone,
    Place as MapPin,
    Close as X,
    Check
} from "@mui/icons-material"

// Column definitions for results table
interface ColumnDef {
    key: keyof ResultEntry | string
    label: string
    defaultVisible: boolean
    render?: (entry: ResultEntry, onSelect?: (entry: ResultEntry) => void) => React.ReactNode
}

const AVAILABLE_COLUMNS: ColumnDef[] = [
    {
        key: "title",
        label: "Business Name",
        defaultVisible: true,
        render: (e, onSelect) => (
            <button
                type="button"
                style={{
                    background: 'none',
                    border: 'none',
                    padding: 0,
                    font: 'inherit',
                    cursor: 'pointer',
                    textAlign: 'left',
                    fontWeight: 500,
                    maxWidth: 200,
                    overflow: 'hidden',
                    textOverflow: 'ellipsis',
                    whiteSpace: 'nowrap'
                }}
                title={`View details for ${e.title}`}
                onClick={(event) => {
                    event.stopPropagation();
                    event.preventDefault();
                    onSelect?.(e);
                }}
            >
                <Box component="span" sx={{ '&:hover': { color: 'primary.main', textDecoration: 'underline' } }}>
                    {e.title}
                </Box>
            </button>
        ),
    },
    {
        key: "category",
        label: "Category",
        defaultVisible: true,
        render: (e) => (
            <Badge variant="outlined" size="small" label={e.category || "-"} sx={{ fontSize: '0.75rem' }} />
        ),
    },
    {
        key: "address",
        label: "Address",
        defaultVisible: true,
        render: (e) => (
            <div style={{ maxWidth: 200, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }} title={e.address}>
                <MapPin sx={{ fontSize: 12, mr: 0.5, verticalAlign: 'middle', color: 'text.secondary' }} />
                <span style={{ fontSize: '0.875rem', color: 'text.secondary' }}>{e.address || "-"}</span>
            </div>
        ),
    },
    {
        key: "phone",
        label: "Phone",
        defaultVisible: true,
        render: (e) => (
            <div style={{ fontSize: '0.875rem' }}>
                {e.phone ? (
                    <Box
                        component="a"
                        href={`tel:${e.phone}`}
                        sx={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 0.5,
                            color: 'inherit',
                            textDecoration: 'none',
                            '&:hover': { color: 'primary.main' }
                        }}
                    >
                        <Phone sx={{ fontSize: 12 }} />
                        {e.phone}
                    </Box>
                ) : (
                    <span style={{ color: 'text.secondary' }}>-</span>
                )}
            </div>
        ),
    },
    {
        key: "web_site",
        label: "Website",
        defaultVisible: true,
        render: (e) => (
            <div style={{ fontSize: '0.875rem', maxWidth: 150, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
                {e.web_site ? (
                    <Box
                        component="a"
                        href={e.web_site}
                        target="_blank"
                        rel="noopener noreferrer"
                        sx={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 0.5,
                            color: 'inherit',
                            textDecoration: 'none',
                            '&:hover': { color: 'primary.main' }
                        }}
                    >
                        <ExternalLink sx={{ fontSize: 12, flexShrink: 0 }} />
                        <span style={{ overflow: 'hidden', textOverflow: 'ellipsis' }}>{new URL(e.web_site).hostname}</span>
                    </Box>
                ) : (
                    <span style={{ color: 'text.secondary' }}>-</span>
                )}
            </div>
        ),
    },
    {
        key: "emails",
        label: "Email",
        defaultVisible: true,
        render: (e) => (
            <div style={{ fontSize: '0.875rem' }}>
                {e.emails && e.emails.length > 0 ? (
                    <Box
                        component="a"
                        href={`mailto:${e.emails[0]}`}
                        sx={{
                            display: 'flex',
                            alignItems: 'center',
                            gap: 0.5,
                            color: 'inherit',
                            textDecoration: 'none',
                            '&:hover': { color: 'primary.main' }
                        }}
                    >
                        <Mail sx={{ fontSize: 12 }} />
                        {e.emails[0]}
                        {e.emails.length > 1 && (
                            <Badge variant="outlined" size="small" label={`+${e.emails.length - 1}`} sx={{ ml: 0.5, fontSize: '0.65rem', height: 16 }} />
                        )}
                    </Box>
                ) : (
                    <span style={{ color: 'text.secondary' }}>-</span>
                )}
            </div>
        ),
    },
    {
        key: "review_rating",
        label: "Rating",
        defaultVisible: true,
        render: (e) => (
            <div style={{ display: 'flex', alignItems: 'center', gap: 4, fontSize: '0.875rem' }}>
                <Star sx={{ fontSize: 12, color: '#EAB308' }} />
                <span>{e.review_rating?.toFixed(1) || "-"}</span>
                <span style={{ color: 'text.secondary' }}>({e.review_count || 0})</span>
            </div>
        ),
    },
    {
        key: "status",
        label: "Status",
        defaultVisible: false,
        render: (e) => <span style={{ fontSize: '0.875rem' }}>{e.status || "-"}</span>,
    },
    {
        key: "latitude",
        label: "Latitude",
        defaultVisible: false,
        render: (e) => <span style={{ fontSize: '0.875rem', fontFamily: 'monospace' }}>{e.latitude?.toFixed(6) || "-"}</span>,
    },
    {
        key: "longitude",
        label: "Longitude",
        defaultVisible: false,
        render: (e) => <span style={{ fontSize: '0.875rem', fontFamily: 'monospace' }}>{e.longitude?.toFixed(6) || "-"}</span>,
    },
    {
        key: "place_id",
        label: "Place ID",
        defaultVisible: false,
        render: (e) => (
            <div style={{ maxWidth: 100, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap', fontSize: '0.75rem', fontFamily: 'monospace' }} title={e.place_id}>
                {e.place_id || "-"}
            </div>
        ),
    },
    {
        key: "price_range",
        label: "Price Range",
        defaultVisible: false,
        render: (e) => <span style={{ fontSize: '0.875rem' }}>{e.price_range || "-"}</span>,
    },
    {
        key: "link",
        label: "Google Maps",
        defaultVisible: false,
        render: (e) => (
            e.link ? (
                <Box
                    component="a"
                    href={e.link}
                    target="_blank"
                    rel="noopener noreferrer"
                    sx={{
                        fontSize: '0.875rem',
                        color: 'inherit',
                        '&:hover': { color: 'primary.main' }
                    }}
                >
                    <ExternalLink sx={{ fontSize: 16 }} />
                </Box>
            ) : <span style={{ color: 'text.secondary' }}>-</span>
        ),
    },
]

interface ResultsTableProps {
    jobId: string
}

export function ResultsTable({ jobId }: ResultsTableProps) {
    const [page, setPage] = useState(1)
    const [search, setSearch] = useState("")
    const [showColumnPicker, setShowColumnPicker] = useState(false)
    const [visibleColumns, setVisibleColumns] = useState<string[]>(
        AVAILABLE_COLUMNS.filter((c) => c.defaultVisible).map((c) => c.key)
    )
    const [selectedBusiness, setSelectedBusiness] = useState<ResultEntry | null>(null)
    const perPage = 25

    const { data, isLoading, error } = useQuery({
        queryKey: ["results", jobId, page, perPage],
        queryFn: () => jobsApi.getResults(jobId, page, perPage),
        enabled: !!jobId,
    })

    // Filter results based on search
    const dataToFilter = data?.data
    const filteredResults = useMemo(() => {
        if (!dataToFilter) return []
        if (!search.trim()) return dataToFilter

        const searchLower = search.toLowerCase()
        return dataToFilter.filter((entry) => {
            return (
                entry.title?.toLowerCase().includes(searchLower) ||
                entry.address?.toLowerCase().includes(searchLower) ||
                entry.phone?.toLowerCase().includes(searchLower) ||
                entry.category?.toLowerCase().includes(searchLower) ||
                entry.web_site?.toLowerCase().includes(searchLower) ||
                entry.emails?.some((e) => e.toLowerCase().includes(searchLower))
            )
        })
    }, [dataToFilter, search])

    const totalPages = data?.meta?.total_pages || 1
    const total = data?.meta?.total || 0

    const toggleColumn = (key: string) => {
        setVisibleColumns((prev) =>
            prev.includes(key) ? prev.filter((k) => k !== key) : [...prev, key]
        )
    }

    const handleExport = (format: 'csv' | 'json' | 'xlsx') => {
        const columnLabels = AVAILABLE_COLUMNS
            .filter(c => visibleColumns.includes(c.key))
            .map(c => c.label)
        const url = jobsApi.downloadResults(jobId, format, columnLabels)
        window.open(url, '_blank')
    }

    if (error) {
        return (
            <Box sx={{
                borderRadius: 1,
                border: 1,
                borderColor: 'error.light',
                bgcolor: 'error.light',
                p: 2,
                textAlign: 'center',
                color: 'error.contrastText',
                opacity: 0.9
            }}>
                Failed to load results. Please try again.
            </Box>
        )
    }

    return (
        <div className="space-y-4">
            {/* Toolbar */}
            <Box sx={{ display: 'flex', flexDirection: { xs: 'column', sm: 'row' }, gap: 1.5, justifyContent: 'space-between', mb: 2 }}>
                <div className="flex gap-2 flex-1" style={{ display: 'flex', gap: 8, flex: 1 }}>
                    {/* Search */}
                    <div className="relative flex-1 max-w-sm" style={{ flex: 1, maxWidth: 400 }}>
                        <TextField
                            placeholder="Search results..."
                            value={search}
                            onChange={(e: React.ChangeEvent<HTMLInputElement>) => setSearch(e.target.value)}
                            size="small"
                            fullWidth
                            slotProps={{
                                input: {
                                    startAdornment: (
                                        <InputAdornment position="start">
                                            <Search sx={{ fontSize: 16, color: 'text.secondary' }} />
                                        </InputAdornment>
                                    ),
                                    endAdornment: search && (
                                        <InputAdornment position="end">
                                            <IconButton size="small" onClick={() => setSearch("")}>
                                                <X sx={{ fontSize: 16 }} />
                                            </IconButton>
                                        </InputAdornment>
                                    )
                                }
                            }}
                        />
                    </div>


                    {/* Column Picker */}
                    <div className="relative" style={{ position: 'relative' }}>
                        <Button
                            variant="outlined"
                            style={{ minWidth: 40, padding: 8 }}
                            onClick={() => setShowColumnPicker(!showColumnPicker)}
                        >
                            <Settings sx={{ fontSize: 16 }} />
                        </Button>

                        {showColumnPicker && (
                            <div style={{
                                position: 'absolute',
                                top: '100%',
                                right: 0,
                                marginTop: 8,
                                zIndex: 50,
                                background: '#fff',
                                border: '1px solid rgba(0,0,0,0.12)',
                                borderRadius: 8,
                                boxShadow: '0px 5px 5px -3px rgba(0,0,0,0.2), 0px 8px 10px 1px rgba(0,0,0,0.14), 0px 3px 14px 2px rgba(0,0,0,0.12)',
                                padding: 12,
                                minWidth: 200
                            }}>
                                <div style={{ fontSize: '0.875rem', fontWeight: 500, marginBottom: 8 }}>Visible Columns</div>
                                <div style={{ maxHeight: 300, overflowY: 'auto' }}>
                                    {AVAILABLE_COLUMNS.map((col) => (
                                        <button
                                            key={col.key}
                                            onClick={() => toggleColumn(col.key)}
                                            style={{
                                                display: 'flex',
                                                alignItems: 'center',
                                                gap: 8,
                                                width: '100%',
                                                padding: '6px 8px',
                                                borderRadius: 4,
                                                border: 'none',
                                                background: 'transparent',
                                                cursor: 'pointer',
                                                textAlign: 'left',
                                                fontSize: '0.875rem'
                                            }}
                                            className="hover:bg-gray-100"
                                        >
                                            <div style={{
                                                width: 16,
                                                height: 16,
                                                borderRadius: 4,
                                                border: '1px solid #ccc',
                                                display: 'flex',
                                                alignItems: 'center',
                                                justifyContent: 'center',
                                                background: visibleColumns.includes(col.key) ? '#000' : 'transparent',
                                                borderColor: visibleColumns.includes(col.key) ? '#000' : '#ccc'
                                            }}>
                                                {visibleColumns.includes(col.key) && <Check sx={{ fontSize: 12, color: '#fff' }} />}
                                            </div>
                                            {col.label}
                                        </button>
                                    ))}
                                </div>
                            </div>
                        )}
                    </div>
                </div>

                {/* Export Buttons */}
                <div className="flex gap-2" style={{ display: 'flex', gap: 8 }}>
                    <Button variant="outlined" size="small" onClick={() => handleExport('csv')}>
                        <Download sx={{ mr: 1, fontSize: 16 }} />
                        CSV
                    </Button>
                    <Button variant="outlined" size="small" onClick={() => handleExport('xlsx')}>
                        <Download sx={{ mr: 1, fontSize: 16 }} />
                        XLSX
                    </Button>
                    <Button variant="outlined" size="small" onClick={() => handleExport('json')}>
                        <Download sx={{ mr: 1, fontSize: 16 }} />
                        JSON
                    </Button>
                </div>
            </div>

            {/* Results count */}
            <div style={{ fontSize: '0.875rem', color: 'text.secondary', marginBottom: 16 }}>
                {search ? (
                    <>Showing {filteredResults.length} of {total} results matching "{search}"</>
                ) : (
                    <>Showing {((page - 1) * perPage) + 1}-{Math.min(page * perPage, total)} of {total} results</>
                )}
            </div>

            {/* Table */}
            <TableContainer component={Paper} variant="outlined" sx={{ borderRadius: 2 }}>
                <Table>
                    <TableHead>
                        <TableRow>
                            {AVAILABLE_COLUMNS.filter((c) => visibleColumns.includes(c.key)).map((col) => (
                                <TableCell key={col.key} sx={{ fontWeight: 600 }}>{col.label}</TableCell>
                            ))}
                        </TableRow>
                    </TableHead>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={visibleColumns.length} sx={{ textAlign: 'center', py: 8 }}>
                                    <div style={{ display: 'flex', alignItems: 'center', justifyItems: 'center', gap: 8 }}>
                                        <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                                        Loading results...
                                    </div>
                                </TableCell>
                            </TableRow>
                        ) : filteredResults.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={visibleColumns.length} sx={{ textAlign: 'center', py: 8, color: 'text.secondary' }}>
                                    {search ? "No results match your search." : "No results yet."}
                                </TableCell>
                            </TableRow>
                        ) : (
                            filteredResults.map((entry, idx) => (
                                <TableRow key={entry.place_id || entry.cid || idx} hover>
                                    {AVAILABLE_COLUMNS.filter((c) => visibleColumns.includes(c.key)).map((col) => (
                                        <TableCell key={col.key}>
                                            {col.render ? col.render(entry, setSelectedBusiness) : String((entry as unknown as Record<string, unknown>)[col.key] || "-")}
                                        </TableCell>
                                    ))}
                                </TableRow>
                            ))
                        )}
                    </TableBody>
                </Table>
            </TableContainer>

            {/* Pagination */}
            {totalPages > 1 && !search && (
                <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginTop: 16 }}>
                    <div style={{ fontSize: '0.875rem', color: 'text.secondary' }}>
                        Page {page} of {totalPages}
                    </div>
                    <div style={{ display: 'flex', gap: 8 }}>
                        <Button
                            variant="outlined"
                            size="small"
                            onClick={() => setPage((p) => Math.max(1, p - 1))}
                            disabled={page === 1}
                        >
                            <ChevronLeft sx={{ fontSize: 16, mr: 0.5 }} />
                            Previous
                        </Button>
                        <Button
                            variant="outlined"
                            size="small"
                            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                            disabled={page === totalPages}
                        >
                            Next
                            <ChevronRight sx={{ fontSize: 16, ml: 0.5 }} />
                        </Button>
                    </div>
                </div>
            )}

            {/* Detail Drawer */}
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
                            <div style={{ fontWeight: 600, fontSize: '1.125rem' }}>Business Details</div>
                            <IconButton onClick={() => setSelectedBusiness(null)} size="small">
                                <X />
                            </IconButton>
                        </Box>
                        <Box sx={{ p: 3, overflowY: 'auto' }}>
                            <BusinessDetail data={selectedBusiness} />
                        </Box>
                    </>
                )}
            </Drawer>
        </div>
    )
}
