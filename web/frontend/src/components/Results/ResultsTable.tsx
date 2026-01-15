import { useState, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { jobsApi } from "@/api/jobs"
import type { ResultEntry } from "@/api/types"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/UI/Table"
import { Button } from "@/components/UI/Button"
import { Input } from "@/components/UI/Input"
import { Badge } from "@/components/UI/Badge"
import { Drawer } from "@/components/UI/Drawer"
import { BusinessDetail } from "./BusinessDetail"
import {
    Search,
    Download,
    ChevronLeft,
    ChevronRight,
    Settings2,
    Star,
    ExternalLink,
    Mail,
    Phone,
    MapPin,
    X,
    Check,
} from "lucide-react"

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
                className="font-medium max-w-[200px] truncate hover:text-primary hover:underline text-left transition-colors cursor-pointer focus:outline-none focus:text-primary"
                title={`View details for ${e.title}`}
                onClick={(event) => {
                    event.stopPropagation();
                    event.preventDefault();
                    onSelect?.(e);
                }}
            >
                {e.title}
            </button>
        ),
    },
    {
        key: "category",
        label: "Category",
        defaultVisible: true,
        render: (e) => (
            <Badge variant="outline" className="text-xs">
                {e.category || "-"}
            </Badge>
        ),
    },
    {
        key: "address",
        label: "Address",
        defaultVisible: true,
        render: (e) => (
            <div className="max-w-[200px] truncate text-sm text-muted-foreground" title={e.address}>
                <MapPin className="inline h-3 w-3 mr-1" />
                {e.address || "-"}
            </div>
        ),
    },
    {
        key: "phone",
        label: "Phone",
        defaultVisible: true,
        render: (e) => (
            <div className="text-sm">
                {e.phone ? (
                    <a href={`tel:${e.phone}`} className="hover:text-primary flex items-center gap-1">
                        <Phone className="h-3 w-3" />
                        {e.phone}
                    </a>
                ) : (
                    <span className="text-muted-foreground">-</span>
                )}
            </div>
        ),
    },
    {
        key: "web_site",
        label: "Website",
        defaultVisible: true,
        render: (e) => (
            <div className="text-sm max-w-[150px] truncate">
                {e.web_site ? (
                    <a
                        href={e.web_site}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="hover:text-primary flex items-center gap-1"
                    >
                        <ExternalLink className="h-3 w-3 flex-shrink-0" />
                        <span className="truncate">{new URL(e.web_site).hostname}</span>
                    </a>
                ) : (
                    <span className="text-muted-foreground">-</span>
                )}
            </div>
        ),
    },
    {
        key: "emails",
        label: "Email",
        defaultVisible: true,
        render: (e) => (
            <div className="text-sm">
                {e.emails && e.emails.length > 0 ? (
                    <a href={`mailto:${e.emails[0]}`} className="hover:text-primary flex items-center gap-1">
                        <Mail className="h-3 w-3" />
                        {e.emails[0]}
                        {e.emails.length > 1 && (
                            <Badge variant="outline" className="ml-1 text-xs">
                                +{e.emails.length - 1}
                            </Badge>
                        )}
                    </a>
                ) : (
                    <span className="text-muted-foreground">-</span>
                )}
            </div>
        ),
    },
    {
        key: "review_rating",
        label: "Rating",
        defaultVisible: true,
        render: (e) => (
            <div className="flex items-center gap-1 text-sm">
                <Star className="h-3 w-3 fill-yellow-500 text-yellow-500" />
                <span>{e.review_rating?.toFixed(1) || "-"}</span>
                <span className="text-muted-foreground">({e.review_count || 0})</span>
            </div>
        ),
    },
    {
        key: "status",
        label: "Status",
        defaultVisible: false,
        render: (e) => <span className="text-sm">{e.status || "-"}</span>,
    },
    {
        key: "latitude",
        label: "Latitude",
        defaultVisible: false,
        render: (e) => <span className="text-sm font-mono">{e.latitude?.toFixed(6) || "-"}</span>,
    },
    {
        key: "longitude",
        label: "Longitude",
        defaultVisible: false,
        render: (e) => <span className="text-sm font-mono">{e.longitude?.toFixed(6) || "-"}</span>,
    },
    {
        key: "place_id",
        label: "Place ID",
        defaultVisible: false,
        render: (e) => (
            <div className="max-w-[100px] truncate text-xs font-mono" title={e.place_id}>
                {e.place_id || "-"}
            </div>
        ),
    },
    {
        key: "price_range",
        label: "Price Range",
        defaultVisible: false,
        render: (e) => <span className="text-sm">{e.price_range || "-"}</span>,
    },
    {
        key: "link",
        label: "Google Maps",
        defaultVisible: false,
        render: (e) => (
            e.link ? (
                <a
                    href={e.link}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-sm hover:text-primary"
                >
                    <ExternalLink className="h-4 w-4" />
                </a>
            ) : <span className="text-muted-foreground">-</span>
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
            <div className="rounded-md border border-destructive/50 bg-destructive/10 p-4 text-center text-destructive">
                Failed to load results. Please try again.
            </div>
        )
    }

    return (
        <div className="space-y-4">
            {/* Toolbar */}
            <div className="flex flex-col sm:flex-row gap-3 justify-between">
                <div className="flex gap-2 flex-1">
                    {/* Search */}
                    <div className="relative flex-1 max-w-sm">
                        <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                        <Input
                            placeholder="Search results..."
                            value={search}
                            onChange={(e) => setSearch(e.target.value)}
                            className="pl-9"
                        />
                        {search && (
                            <button
                                onClick={() => setSearch("")}
                                className="absolute right-3 top-1/2 transform -translate-y-1/2 text-muted-foreground hover:text-foreground"
                            >
                                <X className="h-4 w-4" />
                            </button>
                        )}
                    </div>

                    {/* Column Picker */}
                    <div className="relative">
                        <Button
                            variant="outline"
                            size="icon"
                            onClick={() => setShowColumnPicker(!showColumnPicker)}
                        >
                            <Settings2 className="h-4 w-4" />
                        </Button>

                        {showColumnPicker && (
                            <div className="absolute top-full right-0 mt-2 z-50 bg-neu-base border border-white/10 rounded-lg shadow-lg p-3 min-w-[200px]">
                                <div className="text-sm font-medium mb-2">Visible Columns</div>
                                <div className="space-y-1 max-h-[300px] overflow-y-auto">
                                    {AVAILABLE_COLUMNS.map((col) => (
                                        <button
                                            key={col.key}
                                            onClick={() => toggleColumn(col.key)}
                                            className="flex items-center gap-2 w-full px-2 py-1.5 rounded hover:bg-white/5 text-sm text-left"
                                        >
                                            <div className={`w-4 h-4 rounded border flex items-center justify-center ${visibleColumns.includes(col.key) ? 'bg-primary border-primary' : 'border-muted-foreground'}`}>
                                                {visibleColumns.includes(col.key) && <Check className="h-3 w-3 text-primary-foreground" />}
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
                <div className="flex gap-2">
                    <Button variant="outline" onClick={() => handleExport('csv')}>
                        <Download className="mr-2 h-4 w-4" />
                        CSV
                    </Button>
                    <Button variant="outline" onClick={() => handleExport('xlsx')}>
                        <Download className="mr-2 h-4 w-4" />
                        XLSX
                    </Button>
                    <Button variant="outline" onClick={() => handleExport('json')}>
                        <Download className="mr-2 h-4 w-4" />
                        JSON
                    </Button>
                </div>
            </div>

            {/* Results count */}
            <div className="text-sm text-muted-foreground">
                {search ? (
                    <>Showing {filteredResults.length} of {total} results matching "{search}"</>
                ) : (
                    <>Showing {((page - 1) * perPage) + 1}-{Math.min(page * perPage, total)} of {total} results</>
                )}
            </div>

            {/* Table */}
            <div className="rounded-md border border-white/10">
                <Table>
                    <TableHeader>
                        <TableRow>
                            {AVAILABLE_COLUMNS.filter((c) => visibleColumns.includes(c.key)).map((col) => (
                                <TableHead key={col.key}>{col.label}</TableHead>
                            ))}
                        </TableRow>
                    </TableHeader>
                    <TableBody>
                        {isLoading ? (
                            <TableRow>
                                <TableCell colSpan={visibleColumns.length} className="text-center py-8">
                                    <div className="flex items-center justify-center gap-2">
                                        <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                                        Loading results...
                                    </div>
                                </TableCell>
                            </TableRow>
                        ) : filteredResults.length === 0 ? (
                            <TableRow>
                                <TableCell colSpan={visibleColumns.length} className="text-center py-8 text-muted-foreground">
                                    {search ? "No results match your search." : "No results yet."}
                                </TableCell>
                            </TableRow>
                        ) : (
                            filteredResults.map((entry, idx) => (
                                <TableRow key={entry.place_id || entry.cid || idx}>
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
            </div>

            {/* Pagination */}
            {totalPages > 1 && !search && (
                <div className="flex items-center justify-between">
                    <div className="text-sm text-muted-foreground">
                        Page {page} of {totalPages}
                    </div>
                    <div className="flex gap-2">
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setPage((p) => Math.max(1, p - 1))}
                            disabled={page === 1}
                        >
                            <ChevronLeft className="h-4 w-4 mr-1" />
                            Previous
                        </Button>
                        <Button
                            variant="outline"
                            size="sm"
                            onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                            disabled={page === totalPages}
                        >
                            Next
                            <ChevronRight className="h-4 w-4 ml-1" />
                        </Button>
                    </div>
                </div>
            )}

            {/* Detail Drawer */}
            <Drawer
                isOpen={!!selectedBusiness}
                onClose={() => setSelectedBusiness(null)}
                title={selectedBusiness?.title || "Business Details"}
            >
                {selectedBusiness && <BusinessDetail data={selectedBusiness} />}
            </Drawer>
        </div>
    )
}
