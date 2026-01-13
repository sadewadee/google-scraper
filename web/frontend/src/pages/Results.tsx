import { useState, useMemo } from "react"
import { useQuery } from "@tanstack/react-query"
import { resultsApi } from "@/api/results"
import {
    Table,
    TableBody,
    TableCell,
    TableHead,
    TableHeader,
    TableRow,
} from "@/components/UI/Table"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/UI/Card"
import { Button } from "@/components/UI/Button"
import { Input } from "@/components/UI/Input"
import { Badge } from "@/components/UI/Badge"
import {
    Search,
    Download,
    ChevronLeft,
    ChevronRight,
    Star,
    ExternalLink,
    Mail,
    Phone,
    MapPin,
    X,
    Database,
} from "lucide-react"

interface ResultEntry {
    title?: string
    address?: string
    phone?: string
    web_site?: string
    category?: string
    review_rating?: number
    review_count?: number
    emails?: string[]
    place_id?: string
    cid?: string
    link?: string
}

export default function Results() {
    const [search, setSearch] = useState("")
    const [page, setPage] = useState(1)
    const perPage = 50

    // Fetch all results globally
    const { data: resultsData, isLoading } = useQuery({
        queryKey: ["results", page, perPage],
        queryFn: () => resultsApi.getAll(page, perPage),
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
    }, [resultsData?.data, search])

    const totalPages = resultsData?.meta?.total_pages || 1
    const total = resultsData?.meta?.total || 0

    return (
        <div className="space-y-6">
            <div className="flex items-center justify-between">
                <h2 className="text-3xl font-bold tracking-tight">Results Database</h2>
            </div>

            {/* Stats */}
            <div className="grid gap-4 md:grid-cols-3">
                <Card>
                    <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Total Results</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{total.toLocaleString()}</div>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Current Page</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">{page} / {totalPages}</div>
                    </CardContent>
                </Card>
                <Card>
                    <CardHeader className="pb-2">
                        <CardTitle className="text-sm font-medium">Showing</CardTitle>
                    </CardHeader>
                    <CardContent>
                        <div className="text-2xl font-bold">
                            {total > 0 ? `${((page - 1) * perPage) + 1}-${Math.min(page * perPage, total)}` : "0"}
                        </div>
                    </CardContent>
                </Card>
            </div>

            {/* Search and Export */}
            <Card>
                <CardHeader>
                    <CardTitle className="flex items-center gap-2">
                        <Database className="h-5 w-5" />
                        All Scraped Results
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="flex flex-col sm:flex-row gap-4">
                        <div className="flex-1">
                            <div className="relative">
                                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                                <Input
                                    placeholder="Filter by name, address, phone..."
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
                        </div>
                        <div className="flex gap-2">
                            <Button
                                variant="outline"
                                onClick={() => {
                                    const url = resultsApi.downloadUrl('csv')
                                    window.open(url, '_blank')
                                }}
                            >
                                <Download className="h-4 w-4 mr-2" />
                                CSV
                            </Button>
                            <Button
                                variant="outline"
                                onClick={() => {
                                    const url = resultsApi.downloadUrl('xlsx')
                                    window.open(url, '_blank')
                                }}
                            >
                                <Download className="h-4 w-4 mr-2" />
                                Excel
                            </Button>
                            <Button
                                variant="outline"
                                onClick={() => {
                                    const url = resultsApi.downloadUrl('json')
                                    window.open(url, '_blank')
                                }}
                            >
                                <Download className="h-4 w-4 mr-2" />
                                JSON
                            </Button>
                        </div>
                    </div>
                </CardContent>
            </Card>

            {/* Results Table */}
            <Card>
                <CardHeader className="flex flex-row items-center justify-between">
                    <CardTitle>
                        {search ? (
                            <>Showing {filteredResults.length} results matching "{search}"</>
                        ) : (
                            <>Showing {((page - 1) * perPage) + 1}-{Math.min(page * perPage, total)} of {total} results</>
                        )}
                    </CardTitle>
                </CardHeader>
                <CardContent>
                    <div className="rounded-md border border-white/10">
                        <Table>
                            <TableHeader>
                                <TableRow>
                                    <TableHead>Business Name</TableHead>
                                    <TableHead>Category</TableHead>
                                    <TableHead>Address</TableHead>
                                    <TableHead>Phone</TableHead>
                                    <TableHead>Email</TableHead>
                                    <TableHead>Rating</TableHead>
                                    <TableHead>Website</TableHead>
                                </TableRow>
                            </TableHeader>
                            <TableBody>
                                {isLoading ? (
                                    <TableRow>
                                        <TableCell colSpan={7} className="text-center py-8">
                                            <div className="flex items-center justify-center gap-2">
                                                <div className="h-4 w-4 animate-spin rounded-full border-2 border-primary border-t-transparent" />
                                                Loading results...
                                            </div>
                                        </TableCell>
                                    </TableRow>
                                ) : filteredResults.length === 0 ? (
                                    <TableRow>
                                        <TableCell colSpan={7} className="text-center py-8 text-muted-foreground">
                                            {search ? "No results match your search." : "No results in database yet."}
                                        </TableCell>
                                    </TableRow>
                                ) : (
                                    filteredResults.map((entry, idx) => (
                                        <TableRow key={entry.place_id || entry.cid || idx}>
                                            <TableCell>
                                                <div className="font-medium max-w-[200px] truncate" title={entry.title}>
                                                    {entry.title}
                                                </div>
                                            </TableCell>
                                            <TableCell>
                                                <Badge variant="outline" className="text-xs">
                                                    {entry.category || "-"}
                                                </Badge>
                                            </TableCell>
                                            <TableCell>
                                                <div className="max-w-[200px] truncate text-sm text-muted-foreground" title={entry.address}>
                                                    <MapPin className="inline h-3 w-3 mr-1" />
                                                    {entry.address || "-"}
                                                </div>
                                            </TableCell>
                                            <TableCell>
                                                <div className="text-sm">
                                                    {entry.phone ? (
                                                        <a href={`tel:${entry.phone}`} className="hover:text-primary flex items-center gap-1">
                                                            <Phone className="h-3 w-3" />
                                                            {entry.phone}
                                                        </a>
                                                    ) : (
                                                        <span className="text-muted-foreground">-</span>
                                                    )}
                                                </div>
                                            </TableCell>
                                            <TableCell>
                                                <div className="text-sm">
                                                    {entry.emails && entry.emails.length > 0 ? (
                                                        <a href={`mailto:${entry.emails[0]}`} className="hover:text-primary flex items-center gap-1">
                                                            <Mail className="h-3 w-3" />
                                                            {entry.emails[0]}
                                                        </a>
                                                    ) : (
                                                        <span className="text-muted-foreground">-</span>
                                                    )}
                                                </div>
                                            </TableCell>
                                            <TableCell>
                                                <div className="flex items-center gap-1 text-sm">
                                                    <Star className="h-3 w-3 fill-yellow-500 text-yellow-500" />
                                                    <span>{entry.review_rating?.toFixed(1) || "-"}</span>
                                                    <span className="text-muted-foreground">({entry.review_count || 0})</span>
                                                </div>
                                            </TableCell>
                                            <TableCell>
                                                <div className="text-sm max-w-[100px] truncate">
                                                    {entry.web_site ? (
                                                        <a
                                                            href={entry.web_site}
                                                            target="_blank"
                                                            rel="noopener noreferrer"
                                                            className="hover:text-primary flex items-center gap-1"
                                                        >
                                                            <ExternalLink className="h-3 w-3 flex-shrink-0" />
                                                            <span className="truncate">{new URL(entry.web_site).hostname}</span>
                                                        </a>
                                                    ) : (
                                                        <span className="text-muted-foreground">-</span>
                                                    )}
                                                </div>
                                            </TableCell>
                                        </TableRow>
                                    ))
                                )}
                            </TableBody>
                        </Table>
                    </div>

                    {/* Pagination */}
                    {totalPages > 1 && !search && (
                        <div className="flex items-center justify-between mt-4">
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
                </CardContent>
            </Card>
        </div>
    )
}
